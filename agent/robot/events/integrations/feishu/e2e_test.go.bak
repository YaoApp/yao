package feishu

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	robottypes "github.com/yaoapp/yao/agent/robot/types"
	fsapi "github.com/yaoapp/yao/integrations/feishu"
)

var (
	fsAppID     string
	fsAppSecret string
)

func TestMain(m *testing.M) {
	fsAppID = os.Getenv("FEISHU_TEST_APP_ID")
	fsAppSecret = os.Getenv("FEISHU_TEST_APP_SECRET")
	os.Exit(m.Run())
}

func skipIfNoCreds(t *testing.T) {
	t.Helper()
	if fsAppID == "" || fsAppSecret == "" {
		t.Skip("FEISHU_TEST_APP_ID or FEISHU_TEST_APP_SECRET not set")
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
		MemberID: "robot_e2e_feishu_adapter",
		TeamID:   "team_e2e_fs",
		Config: &robottypes.Config{
			Integrations: &robottypes.Integrations{
				Feishu: &robottypes.FeishuConfig{
					Enabled:   true,
					AppID:     fsAppID,
					AppSecret: fsAppSecret,
				},
			},
		},
	}

	a.Apply(context.Background(), robot)

	a.mu.RLock()
	entry, ok := a.bots["robot_e2e_feishu_adapter"]
	a.mu.RUnlock()

	require.True(t, ok, "bot should be registered")
	assert.Equal(t, fsAppID, entry.appID)
	assert.NotNil(t, entry.bot)

	t.Logf("OK  Apply: feishu bot registered robot=%s app=%s", robot.MemberID, entry.appID)
}

// TestE2E_Adapter_Apply_Update verifies re-Apply with same appID is a no-op.
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
		MemberID: "robot_e2e_feishu_update",
		TeamID:   "team_e2e_fs",
		Config: &robottypes.Config{
			Integrations: &robottypes.Integrations{
				Feishu: &robottypes.FeishuConfig{
					Enabled:   true,
					AppID:     fsAppID,
					AppSecret: fsAppSecret,
				},
			},
		},
	}

	a.Apply(context.Background(), robot)
	a.mu.RLock()
	_, ok := a.bots["robot_e2e_feishu_update"]
	a.mu.RUnlock()
	require.True(t, ok)

	// Apply again — should be no-op
	a.Apply(context.Background(), robot)
	a.mu.RLock()
	assert.Len(t, a.bots, 1)
	a.mu.RUnlock()

	// Remove
	a.Remove(context.Background(), "robot_e2e_feishu_update")
	a.mu.RLock()
	_, ok = a.bots["robot_e2e_feishu_update"]
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

	key := "fs:test-robot:msg-12345"
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
		robotID: "robot_e2e_feishu_handle",
		appID:   fsAppID,
		bot:     fsapi.NewBot(fsAppID, fsAppSecret),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cms := []*fsapi.ConvertedMessage{
		{
			MessageID: "test_msg_1",
			ChatID:    "test_chat_1",
			ChatType:  "p2p",
			SenderID:  "test_sender_1",
			Text:      "Hello from E2E test",
		},
	}

	// This should not panic even without event bus running
	a.handleMessages(ctx, entry, cms)

	// Verify dedup: should be marked as seen
	assert.False(t, a.dedup.markSeen("fs:robot_e2e_feishu_handle:test_msg_1"),
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
		MemberID: "robot_e2e_feishu_disabled",
		TeamID:   "team_e2e_fs",
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
	a.mu.RLock()
	_, ok := a.bots["robot_e2e_feishu_disabled"]
	a.mu.RUnlock()
	assert.False(t, ok, "disabled bot should not be registered")
	t.Log("OK  disabled config not registered")
}
