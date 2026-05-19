//go:build e2e

package telegram_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/xun/capsule"
	"github.com/yaoapp/yao/agent/robot/events/integrations/telegram"
	robottypes "github.com/yaoapp/yao/agent/robot/types"
	"github.com/yaoapp/yao/event"
	tgapi "github.com/yaoapp/yao/integrations/telegram"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

func TestTelegramAdapter(t *testing.T) {
	token := os.Getenv("TELEGRAM_TEST_BOT_TOKEN")
	if token == "" {
		t.Fatal("TELEGRAM_TEST_BOT_TOKEN is required for this test")
	}
	host := os.Getenv("TELEGRAM_TEST_HOST")
	if host == "" {
		t.Fatal("TELEGRAM_TEST_HOST is required for this test")
	}

	t.Run("Apply", func(t *testing.T) {
		a := telegram.NewTestAdapter()
		defer a.Close()

		robot := &robottypes.Robot{
			MemberID: "robot_e2e_tg_adapter",
			TeamID:   "team_e2e_tg",
			Config: &robottypes.Config{
				Integrations: &robottypes.Integrations{
					Telegram: &robottypes.TelegramConfig{
						Enabled:  true,
						BotToken: token,
						Host:     host,
						AppID:    "e2e-test-app",
					},
				},
			},
		}

		a.Apply(context.Background(), robot)

		entry := a.GetBot("robot_e2e_tg_adapter")
		require.NotNil(t, entry, "bot should be registered")

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		me, err := entry.GetMe(ctx)
		require.NoError(t, err)
		assert.True(t, me.IsBot)
		t.Logf("OK  Apply: bot registered id=%d username=%s", me.ID, me.Username)
	})

	t.Run("Apply_Update", func(t *testing.T) {
		a := telegram.NewTestAdapter()
		defer a.Close()

		robot := &robottypes.Robot{
			MemberID: "robot_e2e_tg_update",
			TeamID:   "team_e2e_tg",
			Config: &robottypes.Config{
				Integrations: &robottypes.Integrations{
					Telegram: &robottypes.TelegramConfig{
						Enabled:  true,
						BotToken: token,
						Host:     host,
					},
				},
			},
		}

		a.Apply(context.Background(), robot)
		require.NotNil(t, a.GetBot("robot_e2e_tg_update"))

		a.Apply(context.Background(), robot)
		assert.Equal(t, 1, a.BotCount())

		a.Remove(context.Background(), "robot_e2e_tg_update")
		assert.Nil(t, a.GetBot("robot_e2e_tg_update"), "bot should be removed")
		t.Log("OK  Apply/Remove lifecycle verified")
	})

	t.Run("PollAll", func(t *testing.T) {
		testprepare.PrepareE2E(t)

		pending := confirmPendingUpdates(t, token, host)
		if len(pending) == 0 {
			t.Fatal("no pending updates; run integrations/telegram seed first")
		}
		t.Logf("found %d pending updates", len(pending))

		memberID := "robot_e2e_tg_poll"
		setupTestRobot(t, memberID, token, host)
		defer cleanupTestRobots(t)

		if err := event.Start(); err != nil && err != event.ErrAlreadyStart {
			t.Fatalf("event.Start: %v", err)
		}
		defer func() { _ = event.Stop(context.Background()) }()

		a := telegram.NewTestAdapterWithBot(memberID, "e2e-poll-app", token, host)
		defer a.Close()

		a.PollAll()

		offset := a.GetOffset(memberID)
		assert.Greater(t, offset, int64(0), "offset should have advanced after processing updates")
		t.Logf("OK  pollAll: offset advanced to %d", offset)
	})

	t.Run("Dedup", func(t *testing.T) {
		a := telegram.NewTestAdapter()
		defer a.Close()

		key := "tg:test-robot:12345"
		assert.True(t, a.MarkSeen(key), "first time should return true")
		assert.False(t, a.MarkSeen(key), "second time should return false (dedup)")
		t.Log("OK  dedup working correctly")
	})

	t.Run("HandleMessages_Integration", func(t *testing.T) {
		testprepare.PrepareE2E(t)

		var opts []tgapi.BotOption
		if host != "" {
			opts = append(opts, tgapi.WithAPIBase(host))
		}
		b := tgapi.NewBot(token, "", opts...)
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		msgs, err := b.GetUpdates(ctx, 0, 5, nil)
		require.NoError(t, err)
		if len(msgs) == 0 {
			t.Fatal("no pending updates; run integrations/telegram seed first")
		}

		memberID := "robot_e2e_tg_handle"
		setupTestRobot(t, memberID, token, host)
		defer cleanupTestRobots(t)

		if err := event.Start(); err != nil && err != event.ErrAlreadyStart {
			t.Fatalf("event.Start: %v", err)
		}
		defer func() { _ = event.Stop(context.Background()) }()

		a := telegram.NewTestAdapterWithBot(memberID, "e2e-handle-app", token, host)
		defer a.Close()

		a.HandleMessagesForTest(ctx, memberID, msgs)

		cm := msgs[0]
		assert.False(t, a.MarkSeen(fmt.Sprintf("tg:%s:%d", memberID, cm.UpdateID)),
			"update should be marked as seen after handleMessages")
		t.Logf("OK  handleMessages processed %d updates", len(msgs))
	})
}

func confirmPendingUpdates(t *testing.T, token, host string) []*tgapi.ConvertedMessage {
	t.Helper()
	var opts []tgapi.BotOption
	if host != "" {
		opts = append(opts, tgapi.WithAPIBase(host))
	}
	b := tgapi.NewBot(token, "", opts...)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	msgs, err := b.GetUpdates(ctx, 0, 5, nil)
	require.NoError(t, err)
	return msgs
}

func setupTestRobot(t *testing.T, memberID, token, host string) {
	t.Helper()
	m := model.Select("__yao.member")
	if m == nil {
		t.Fatal("__yao.member model not loaded")
	}
	qb := capsule.Query()

	robotConfig := map[string]interface{}{
		"identity": map[string]interface{}{
			"role":   "Telegram E2E Test Robot",
			"duties": []string{"Process Telegram messages"},
		},
		"integrations": map[string]interface{}{
			"telegram": map[string]interface{}{
				"enabled":   true,
				"bot_token": token,
				"host":      host,
				"app_id":    "e2e-tg-app-" + memberID,
			},
		},
		"resources": map[string]interface{}{
			"phases": map[string]interface{}{
				"host": "robot.host",
			},
		},
	}
	configJSON, _ := json.Marshal(robotConfig)

	err := qb.Table(m.MetaData.Table.Name).Insert([]map[string]interface{}{
		{
			"member_id":       memberID,
			"team_id":         "team_e2e_tg",
			"member_type":     "robot",
			"display_name":    "E2E TG Adapter Test " + memberID,
			"system_prompt":   "You are a test robot for Telegram adapter E2E testing.",
			"status":          "active",
			"role_id":         "member",
			"autonomous_mode": false,
			"robot_status":    "idle",
			"robot_config":    string(configJSON),
		},
	})
	if err != nil {
		t.Fatalf("setup robot %s: %v", memberID, err)
	}
}

func cleanupTestRobots(t *testing.T) {
	t.Helper()
	m := model.Select("__yao.member")
	if m == nil {
		return
	}
	qb := capsule.Query()
	_, _ = qb.Table(m.MetaData.Table.Name).Where("member_id", "like", "robot_e2e_tg%").Delete()
}
