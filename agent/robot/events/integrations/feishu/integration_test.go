//go:build integration

package feishu_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/robot/events/integrations/feishu"
	robottypes "github.com/yaoapp/yao/agent/robot/types"
	fsapi "github.com/yaoapp/yao/integrations/feishu"
)

func TestFeishuIntegration_Apply(t *testing.T) {
	appID := os.Getenv("FEISHU_TEST_APP_ID")
	if appID == "" {
		t.Fatal("FEISHU_TEST_APP_ID is required for this test")
	}
	appSecret := os.Getenv("FEISHU_TEST_APP_SECRET")
	if appSecret == "" {
		t.Fatal("FEISHU_TEST_APP_SECRET is required for this test")
	}

	a := feishu.NewTestAdapter()
	defer a.Close()

	robot := &robottypes.Robot{
		MemberID: "robot_intg_feishu_adapter",
		TeamID:   "team_intg_fs",
		Config: &robottypes.Config{
			Integrations: &robottypes.Integrations{
				Feishu: &robottypes.FeishuConfig{
					Enabled:   true,
					AppID:     appID,
					AppSecret: appSecret,
				},
			},
		},
	}

	a.Apply(context.Background(), robot)

	bot := a.GetBot("robot_intg_feishu_adapter")
	require.NotNil(t, bot, "bot should be registered")
	t.Logf("OK  Apply: feishu bot registered robot=%s app=%s", robot.MemberID, appID)
}

func TestFeishuIntegration_Apply_Update(t *testing.T) {
	appID := os.Getenv("FEISHU_TEST_APP_ID")
	if appID == "" {
		t.Fatal("FEISHU_TEST_APP_ID is required for this test")
	}
	appSecret := os.Getenv("FEISHU_TEST_APP_SECRET")
	if appSecret == "" {
		t.Fatal("FEISHU_TEST_APP_SECRET is required for this test")
	}

	a := feishu.NewTestAdapter()
	defer a.Close()

	robot := &robottypes.Robot{
		MemberID: "robot_intg_feishu_update",
		TeamID:   "team_intg_fs",
		Config: &robottypes.Config{
			Integrations: &robottypes.Integrations{
				Feishu: &robottypes.FeishuConfig{
					Enabled:   true,
					AppID:     appID,
					AppSecret: appSecret,
				},
			},
		},
	}

	a.Apply(context.Background(), robot)
	require.NotNil(t, a.GetBot("robot_intg_feishu_update"))

	a.Apply(context.Background(), robot)
	assert.Equal(t, 1, a.BotCount())

	a.Remove(context.Background(), "robot_intg_feishu_update")
	assert.Nil(t, a.GetBot("robot_intg_feishu_update"), "bot should be removed")
	t.Log("OK  Apply/Remove lifecycle verified")
}

func TestFeishuIntegration_Dedup(t *testing.T) {
	a := feishu.NewTestAdapter()
	defer a.Close()

	key := "fs:test-robot:msg-12345"
	assert.True(t, a.MarkSeen(key), "first time should return true")
	assert.False(t, a.MarkSeen(key), "second time should return false (dedup)")
	t.Log("OK  dedup working correctly")
}

func TestFeishuIntegration_HandleMessages(t *testing.T) {
	appID := os.Getenv("FEISHU_TEST_APP_ID")
	if appID == "" {
		t.Fatal("FEISHU_TEST_APP_ID is required for this test")
	}
	appSecret := os.Getenv("FEISHU_TEST_APP_SECRET")
	if appSecret == "" {
		t.Fatal("FEISHU_TEST_APP_SECRET is required for this test")
	}

	a := feishu.NewTestAdapterWithBot("robot_intg_feishu_handle", appID, appSecret)
	defer a.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cms := []*fsapi.ConvertedMessage{
		{
			MessageID: "test_msg_1",
			ChatID:    "test_chat_1",
			ChatType:  "p2p",
			SenderID:  "test_sender_1",
			Text:      "Hello from integration test",
		},
	}

	a.HandleMessagesForTest(ctx, "robot_intg_feishu_handle", cms)

	assert.False(t, a.MarkSeen("fs:robot_intg_feishu_handle:test_msg_1"),
		"message should be marked as seen after handleMessages")
	t.Log("OK  handleMessages processed 1 message")
}

func TestFeishuIntegration_ApplyDisabled(t *testing.T) {
	a := feishu.NewTestAdapter()
	defer a.Close()

	robot := &robottypes.Robot{
		MemberID: "robot_intg_feishu_disabled",
		TeamID:   "team_intg_fs",
		Config: &robottypes.Config{
			Integrations: &robottypes.Integrations{
				Feishu: &robottypes.FeishuConfig{
					Enabled:   false,
					AppID:     "some_app",
					AppSecret: "some_secret",
				},
			},
		},
	}

	a.Apply(context.Background(), robot)
	assert.Nil(t, a.GetBot("robot_intg_feishu_disabled"), "disabled bot should not be registered")
	t.Log("OK  disabled config not registered")
}
