package dingtalk

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	robottypes "github.com/yaoapp/yao/agent/robot/types"
	dtapi "github.com/yaoapp/yao/integrations/dingtalk"
)

var (
	dtClientID     string
	dtClientSecret string
)

func TestMain(m *testing.M) {
	dtClientID = os.Getenv("DINGTALK_TEST_CLIENT_ID")
	dtClientSecret = os.Getenv("DINGTALK_TEST_CLIENT_SECRET")
	os.Exit(m.Run())
}

func skipIfNoCreds(t *testing.T) {
	t.Helper()
	if dtClientID == "" || dtClientSecret == "" {
		t.Skip("DINGTALK_TEST_CLIENT_ID or DINGTALK_TEST_CLIENT_SECRET not set")
	}
}

// TestE2E_Adapter_Apply verifies that Apply correctly registers a bot.
func TestE2E_Adapter_Apply(t *testing.T) {
	skipIfNoCreds(t)

	a := &Adapter{
		bots:   make(map[string]*botEntry),
		appIdx: make(map[string]string),
		dedup:  newDedupStore(),
		stopCh: make(chan struct{}),
	}
	defer close(a.stopCh)

	robot := &robottypes.Robot{
		MemberID: "robot_e2e_dt_adapter",
		TeamID:   "team_e2e_dt",
		Config: &robottypes.Config{
			Integrations: &robottypes.Integrations{
				DingTalk: &robottypes.DingTalkConfig{
					Enabled:      true,
					ClientID:     dtClientID,
					ClientSecret: dtClientSecret,
				},
			},
		},
	}

	a.Apply(context.Background(), robot)

	a.mu.RLock()
	entry, ok := a.bots["robot_e2e_dt_adapter"]
	a.mu.RUnlock()

	require.True(t, ok, "bot should be registered")
	assert.Equal(t, dtClientID, entry.clientID)
	assert.NotNil(t, entry.bot)

	t.Logf("OK  Apply: dingtalk bot registered robot=%s client=%s", robot.MemberID, entry.clientID)
}

// TestE2E_Adapter_Apply_Update verifies re-Apply with same clientID is a no-op.
func TestE2E_Adapter_Apply_Update(t *testing.T) {
	skipIfNoCreds(t)

	a := &Adapter{
		bots:   make(map[string]*botEntry),
		appIdx: make(map[string]string),
		dedup:  newDedupStore(),
		stopCh: make(chan struct{}),
	}
	defer close(a.stopCh)

	robot := &robottypes.Robot{
		MemberID: "robot_e2e_dt_update",
		TeamID:   "team_e2e_dt",
		Config: &robottypes.Config{
			Integrations: &robottypes.Integrations{
				DingTalk: &robottypes.DingTalkConfig{
					Enabled:      true,
					ClientID:     dtClientID,
					ClientSecret: dtClientSecret,
				},
			},
		},
	}

	a.Apply(context.Background(), robot)
	a.mu.RLock()
	_, ok := a.bots["robot_e2e_dt_update"]
	a.mu.RUnlock()
	require.True(t, ok)

	a.Apply(context.Background(), robot)
	a.mu.RLock()
	assert.Len(t, a.bots, 1)
	a.mu.RUnlock()

	a.Remove(context.Background(), "robot_e2e_dt_update")
	a.mu.RLock()
	_, ok = a.bots["robot_e2e_dt_update"]
	a.mu.RUnlock()
	assert.False(t, ok, "bot should be removed")
	t.Log("OK  Apply/Remove lifecycle verified")
}

// TestE2E_Adapter_Dedup verifies deduplication works.
func TestE2E_Adapter_Dedup(t *testing.T) {
	a := &Adapter{
		bots:   make(map[string]*botEntry),
		appIdx: make(map[string]string),
		dedup:  newDedupStore(),
		stopCh: make(chan struct{}),
	}
	defer close(a.stopCh)

	key := "dt:test-robot:msg-12345"
	assert.True(t, a.dedup.markSeen(key), "first time should return true")
	assert.False(t, a.dedup.markSeen(key), "second time should return false (dedup)")
	t.Log("OK  dedup working correctly")
}

// TestE2E_Adapter_HandleMessages verifies message handling through the adapter.
func TestE2E_Adapter_HandleMessages(t *testing.T) {
	skipIfNoCreds(t)

	a := &Adapter{
		bots:   make(map[string]*botEntry),
		appIdx: make(map[string]string),
		dedup:  newDedupStore(),
		stopCh: make(chan struct{}),
	}
	defer close(a.stopCh)

	entry := &botEntry{
		robotID:  "robot_e2e_dt_handle",
		clientID: dtClientID,
		bot:      dtapi.NewBot(dtClientID, dtClientSecret),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cms := []*dtapi.ConvertedMessage{
		{
			MessageID:        "test_msg_1",
			ConversationID:   "test_conv_1",
			ConversationType: "1",
			SenderID:         "test_sender_1",
			SenderNick:       "Test User",
			Text:             "Hello from E2E test",
			SessionWebhook:   "https://oapi.dingtalk.com/robot/sendBySession/xxx",
		},
	}

	a.handleMessages(ctx, entry, cms)

	assert.False(t, a.dedup.markSeen("dt:robot_e2e_dt_handle:test_msg_1"),
		"message should be marked as seen after handleMessages")
	t.Log("OK  handleMessages processed 1 message")
}

// TestE2E_Adapter_ApplyDisabled verifies Apply removes bot when disabled.
func TestE2E_Adapter_ApplyDisabled(t *testing.T) {
	a := &Adapter{
		bots:   make(map[string]*botEntry),
		appIdx: make(map[string]string),
		dedup:  newDedupStore(),
		stopCh: make(chan struct{}),
	}
	defer close(a.stopCh)

	robot := &robottypes.Robot{
		MemberID: "robot_e2e_dt_disabled",
		TeamID:   "team_e2e_dt",
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
	a.mu.RLock()
	_, ok := a.bots["robot_e2e_dt_disabled"]
	a.mu.RUnlock()
	assert.False(t, ok, "disabled bot should not be registered")
	t.Log("OK  disabled config not registered")
}

// TestE2E_Adapter_GetAccessToken verifies real DingTalk credentials work.
func TestE2E_Adapter_GetAccessToken(t *testing.T) {
	skipIfNoCreds(t)

	b := dtapi.NewBot(dtClientID, dtClientSecret)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	token, err := b.GetAccessToken(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, token)
	t.Logf("OK  DingTalk access token obtained, len=%d", len(token))
}
