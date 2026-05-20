//go:build integration

package dingtalk_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/robot/events/integrations/dingtalk"
	robottypes "github.com/yaoapp/yao/agent/robot/types"
	dtapi "github.com/yaoapp/yao/integrations/dingtalk"
)

func TestDingTalkIntegration_Apply(t *testing.T) {
	clientID := os.Getenv("DINGTALK_TEST_CLIENT_ID")
	if clientID == "" {
		t.Fatal("DINGTALK_TEST_CLIENT_ID is required for this test")
	}
	clientSecret := os.Getenv("DINGTALK_TEST_CLIENT_SECRET")
	if clientSecret == "" {
		t.Fatal("DINGTALK_TEST_CLIENT_SECRET is required for this test")
	}

	a := dingtalk.NewTestAdapter()
	defer a.Close()

	robot := &robottypes.Robot{
		MemberID: "robot_intg_dt_adapter",
		TeamID:   "team_intg_dt",
		Config: &robottypes.Config{
			Integrations: &robottypes.Integrations{
				DingTalk: &robottypes.DingTalkConfig{
					Enabled:      true,
					ClientID:     clientID,
					ClientSecret: clientSecret,
				},
			},
		},
	}

	a.Apply(context.Background(), robot)

	bot := a.GetBot("robot_intg_dt_adapter")
	require.NotNil(t, bot, "bot should be registered")
	t.Logf("OK  Apply: dingtalk bot registered robot=%s client=%s", robot.MemberID, clientID)
}

func TestDingTalkIntegration_Apply_Update(t *testing.T) {
	clientID := os.Getenv("DINGTALK_TEST_CLIENT_ID")
	if clientID == "" {
		t.Fatal("DINGTALK_TEST_CLIENT_ID is required for this test")
	}
	clientSecret := os.Getenv("DINGTALK_TEST_CLIENT_SECRET")
	if clientSecret == "" {
		t.Fatal("DINGTALK_TEST_CLIENT_SECRET is required for this test")
	}

	a := dingtalk.NewTestAdapter()
	defer a.Close()

	robot := &robottypes.Robot{
		MemberID: "robot_intg_dt_update",
		TeamID:   "team_intg_dt",
		Config: &robottypes.Config{
			Integrations: &robottypes.Integrations{
				DingTalk: &robottypes.DingTalkConfig{
					Enabled:      true,
					ClientID:     clientID,
					ClientSecret: clientSecret,
				},
			},
		},
	}

	a.Apply(context.Background(), robot)
	require.NotNil(t, a.GetBot("robot_intg_dt_update"))

	a.Apply(context.Background(), robot)
	assert.Equal(t, 1, a.BotCount())

	a.Remove(context.Background(), "robot_intg_dt_update")
	assert.Nil(t, a.GetBot("robot_intg_dt_update"), "bot should be removed")
	t.Log("OK  Apply/Remove lifecycle verified")
}

func TestDingTalkIntegration_Dedup(t *testing.T) {
	a := dingtalk.NewTestAdapter()
	defer a.Close()

	key := "dt:test-robot:msg-12345"
	assert.True(t, a.MarkSeen(key), "first time should return true")
	assert.False(t, a.MarkSeen(key), "second time should return false (dedup)")
	t.Log("OK  dedup working correctly")
}

func TestDingTalkIntegration_HandleMessages(t *testing.T) {
	clientID := os.Getenv("DINGTALK_TEST_CLIENT_ID")
	if clientID == "" {
		t.Fatal("DINGTALK_TEST_CLIENT_ID is required for this test")
	}
	clientSecret := os.Getenv("DINGTALK_TEST_CLIENT_SECRET")
	if clientSecret == "" {
		t.Fatal("DINGTALK_TEST_CLIENT_SECRET is required for this test")
	}

	a := dingtalk.NewTestAdapterWithBot("robot_intg_dt_handle", clientID, clientSecret)
	defer a.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cms := []*dtapi.ConvertedMessage{
		{
			MessageID:        "test_msg_1",
			ConversationID:   "test_conv_1",
			ConversationType: "1",
			SenderID:         "test_sender_1",
			SenderNick:       "Test User",
			Text:             "Hello from integration test",
			SessionWebhook:   "https://oapi.dingtalk.com/robot/sendBySession/xxx",
		},
	}

	a.HandleMessagesForTest(ctx, "robot_intg_dt_handle", cms)

	assert.False(t, a.MarkSeen("dt:robot_intg_dt_handle:test_msg_1"),
		"message should be marked as seen after handleMessages")
	t.Log("OK  handleMessages processed 1 message")
}

func TestDingTalkIntegration_ApplyDisabled(t *testing.T) {
	a := dingtalk.NewTestAdapter()
	defer a.Close()

	robot := &robottypes.Robot{
		MemberID: "robot_intg_dt_disabled",
		TeamID:   "team_intg_dt",
		Config: &robottypes.Config{
			Integrations: &robottypes.Integrations{
				DingTalk: &robottypes.DingTalkConfig{
					Enabled:      false,
					ClientID:     "some_id",
					ClientSecret: "some_secret",
				},
			},
		},
	}

	a.Apply(context.Background(), robot)
	assert.Nil(t, a.GetBot("robot_intg_dt_disabled"), "disabled bot should not be registered")
	t.Log("OK  disabled config not registered")
}

func TestDingTalkIntegration_GetAccessToken(t *testing.T) {
	clientID := os.Getenv("DINGTALK_TEST_CLIENT_ID")
	if clientID == "" {
		t.Fatal("DINGTALK_TEST_CLIENT_ID is required for this test")
	}
	clientSecret := os.Getenv("DINGTALK_TEST_CLIENT_SECRET")
	if clientSecret == "" {
		t.Fatal("DINGTALK_TEST_CLIENT_SECRET is required for this test")
	}

	b := dtapi.NewBot(clientID, clientSecret)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	token, err := b.GetAccessToken(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, token)
	t.Logf("OK  DingTalk access token obtained, len=%d", len(token))
}
