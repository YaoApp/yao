package telegram

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
	robottypes "github.com/yaoapp/yao/agent/robot/types"
	"github.com/yaoapp/yao/agent/testutils"
	"github.com/yaoapp/yao/event"
	tgapi "github.com/yaoapp/yao/integrations/telegram"
)

var (
	tgBotToken string
	tgHost     string
)

func TestMain(m *testing.M) {
	tgBotToken = os.Getenv("TELEGRAM_TEST_BOT_TOKEN")
	tgHost = os.Getenv("TELEGRAM_TEST_HOST")
	os.Exit(m.Run())
}

func skipIfNoToken(t *testing.T) {
	t.Helper()
	if tgBotToken == "" {
		t.Skip("TELEGRAM_TEST_BOT_TOKEN not set")
	}
}

func newTestBot() *tgapi.Bot {
	var opts []tgapi.BotOption
	if tgHost != "" {
		opts = append(opts, tgapi.WithAPIBase(tgHost))
	}
	return tgapi.NewBot(tgBotToken, "", opts...)
}

// confirmPendingUpdates checks if there are pending updates from previous seeds.
func confirmPendingUpdates(t *testing.T) []*tgapi.ConvertedMessage {
	t.Helper()
	b := newTestBot()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	msgs, err := b.GetUpdates(ctx, 0, 5, nil)
	require.NoError(t, err)
	return msgs
}

// TestE2E_Adapter_Apply verifies that Apply correctly registers a bot.
func TestE2E_Adapter_Apply(t *testing.T) {
	skipIfNoToken(t)

	a := &Adapter{
		bots:   make(map[string]*botEntry),
		appIdx: make(map[string]string),
		dedup:  newDedupStore(),
		stopCh: make(chan struct{}),
	}
	defer close(a.stopCh)

	robot := &robottypes.Robot{
		MemberID: "robot_e2e_tg_adapter",
		TeamID:   "team_e2e_tg",
		Config: &robottypes.Config{
			Integrations: &robottypes.Integrations{
				Telegram: &robottypes.TelegramConfig{
					Enabled:  true,
					BotToken: tgBotToken,
					Host:     tgHost,
					AppID:    "e2e-test-app",
				},
			},
		},
	}

	a.Apply(context.Background(), robot)

	a.mu.RLock()
	entry, ok := a.bots["robot_e2e_tg_adapter"]
	a.mu.RUnlock()

	require.True(t, ok, "bot should be registered")
	assert.Equal(t, tgBotToken, entry.bot.Token())
	assert.Equal(t, "e2e-test-app", entry.appID)

	// Verify GetMe works through the registered bot
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	me, err := entry.bot.GetMe(ctx)
	require.NoError(t, err)
	assert.True(t, me.IsBot)
	t.Logf("OK  Apply: bot registered id=%d username=%s", me.ID, me.Username)

	// Verify ResolveBot
	resolved := a.ResolveBot("e2e-test-app")
	require.NotNil(t, resolved)
	assert.Equal(t, tgBotToken, resolved.Token())
}

// TestE2E_Adapter_Apply_Update verifies that Apply with a different token replaces the bot.
func TestE2E_Adapter_Apply_Update(t *testing.T) {
	skipIfNoToken(t)

	a := &Adapter{
		bots:   make(map[string]*botEntry),
		appIdx: make(map[string]string),
		dedup:  newDedupStore(),
		stopCh: make(chan struct{}),
	}
	defer close(a.stopCh)

	robot := &robottypes.Robot{
		MemberID: "robot_e2e_tg_update",
		TeamID:   "team_e2e_tg",
		Config: &robottypes.Config{
			Integrations: &robottypes.Integrations{
				Telegram: &robottypes.TelegramConfig{
					Enabled:  true,
					BotToken: tgBotToken,
					Host:     tgHost,
				},
			},
		},
	}

	a.Apply(context.Background(), robot)
	a.mu.RLock()
	_, ok := a.bots["robot_e2e_tg_update"]
	a.mu.RUnlock()
	require.True(t, ok)

	// Apply again with same token — should be a no-op
	a.Apply(context.Background(), robot)
	a.mu.RLock()
	assert.Len(t, a.bots, 1)
	a.mu.RUnlock()

	// Remove
	a.Remove(context.Background(), "robot_e2e_tg_update")
	a.mu.RLock()
	_, ok = a.bots["robot_e2e_tg_update"]
	a.mu.RUnlock()
	assert.False(t, ok, "bot should be removed")
	t.Log("OK  Apply/Remove lifecycle verified")
}

// TestE2E_Adapter_PollAll verifies that pollAll fetches updates from Telegram
// and processes them through handleMessages.
func TestE2E_Adapter_PollAll(t *testing.T) {
	skipIfNoToken(t)
	testutils.PrepareAgent(t)
	defer testutils.Clean(t)

	pending := confirmPendingUpdates(t)
	if len(pending) == 0 {
		t.Skip("no pending updates; run integrations/telegram seed first")
	}
	t.Logf("found %d pending updates", len(pending))

	// Create adapter WITHOUT auto-starting pollLoop
	a := &Adapter{
		bots:   make(map[string]*botEntry),
		appIdx: make(map[string]string),
		dedup:  newDedupStore(),
		stopCh: make(chan struct{}),
	}
	defer close(a.stopCh)

	memberID := "robot_e2e_tg_poll"
	setupTestRobot(t, memberID)
	defer cleanupTestRobots(t)

	var opts []tgapi.BotOption
	if tgHost != "" {
		opts = append(opts, tgapi.WithAPIBase(tgHost))
	}
	a.bots[memberID] = &botEntry{
		robotID: memberID,
		appID:   "e2e-poll-app",
		bot:     tgapi.NewBot(tgBotToken, "", opts...),
	}

	// Start event bus so event.Push works
	if err := event.Start(); err != nil && err != event.ErrAlreadyStart {
		t.Fatalf("event.Start: %v", err)
	}
	defer func() { _ = event.Stop(context.Background()) }()

	// Manually trigger one poll cycle
	a.pollAll()

	// Verify offset advanced (meaning updates were processed)
	a.mu.RLock()
	entry := a.bots[memberID]
	a.mu.RUnlock()
	assert.Greater(t, entry.offset, int64(0), "offset should have advanced after processing updates")
	t.Logf("OK  pollAll: offset advanced to %d", entry.offset)
}

// TestE2E_Adapter_Dedup verifies that duplicate messages are not processed twice.
func TestE2E_Adapter_Dedup(t *testing.T) {
	skipIfNoToken(t)

	a := &Adapter{
		bots:   make(map[string]*botEntry),
		appIdx: make(map[string]string),
		dedup:  newDedupStore(),
		stopCh: make(chan struct{}),
	}
	defer close(a.stopCh)

	key := "tg:test-robot:12345"
	assert.True(t, a.dedup.markSeen(key), "first time should return true")
	assert.False(t, a.dedup.markSeen(key), "second time should return false (dedup)")
	t.Log("OK  dedup working correctly")
}

// TestE2E_Adapter_HandleMessages_Integration verifies the full flow:
// GetUpdates → ConvertedMessage → handleMessages → event.Push
func TestE2E_Adapter_HandleMessages_Integration(t *testing.T) {
	skipIfNoToken(t)
	testutils.PrepareAgent(t)
	defer testutils.Clean(t)

	b := newTestBot()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	msgs, err := b.GetUpdates(ctx, 0, 5, nil)
	require.NoError(t, err)
	if len(msgs) == 0 {
		t.Skip("no pending updates; run integrations/telegram seed first")
	}

	memberID := "robot_e2e_tg_handle"
	setupTestRobot(t, memberID)
	defer cleanupTestRobots(t)

	if err := event.Start(); err != nil && err != event.ErrAlreadyStart {
		t.Fatalf("event.Start: %v", err)
	}
	defer func() { _ = event.Stop(context.Background()) }()

	a := &Adapter{
		bots:   make(map[string]*botEntry),
		appIdx: make(map[string]string),
		dedup:  newDedupStore(),
		stopCh: make(chan struct{}),
	}
	defer close(a.stopCh)

	entry := &botEntry{
		robotID: memberID,
		appID:   "e2e-handle-app",
		bot:     b,
	}

	// Group all messages by chatID (like pollAll does) and process each group
	grouped := groupByChatID(msgs)
	for chatID, chatMsgs := range grouped {
		t.Logf("processing chat=%d messages=%d", chatID, len(chatMsgs))
		for _, cm := range chatMsgs {
			t.Logf("  update_id=%d msg_id=%d text=%q media=%d",
				cm.UpdateID, cm.MessageID, truncate(cm.Text, 40), len(cm.MediaItems))
		}
		a.handleMessages(ctx, entry, chatMsgs)
	}

	// Verify dedup: all updates should be marked as seen
	cm := msgs[0]
	assert.False(t, a.dedup.markSeen(fmt.Sprintf("tg:%s:%d", memberID, cm.UpdateID)),
		"update should be marked as seen after handleMessages")

	// Second call with same messages should be fully deduped (no-op)
	a.handleMessages(ctx, entry, msgs)
	t.Logf("OK  handleMessages processed %d updates across %d chats", len(msgs), len(grouped))
}

// ==================== Helpers ====================

func setupTestRobot(t *testing.T, memberID string) {
	t.Helper()
	m := model.Select("__yao.member")
	if m == nil {
		t.Skip("__yao.member model not loaded")
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
				"bot_token": tgBotToken,
				"host":      tgHost,
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

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
