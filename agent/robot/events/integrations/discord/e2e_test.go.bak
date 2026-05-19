package discord

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	robottypes "github.com/yaoapp/yao/agent/robot/types"
	dcapi "github.com/yaoapp/yao/integrations/discord"
)

var (
	dcBotToken string
	dcAppID    string
)

func TestMain(m *testing.M) {
	dcBotToken = os.Getenv("DISCORD_TEST_BOT_TOKEN")
	dcAppID = os.Getenv("DISCORD_TEST_APP_ID")
	os.Exit(m.Run())
}

func skipIfNoToken(t *testing.T) {
	t.Helper()
	if dcBotToken == "" {
		t.Skip("DISCORD_TEST_BOT_TOKEN not set")
	}
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
		MemberID: "robot_e2e_dc_adapter",
		TeamID:   "team_e2e_dc",
		Config: &robottypes.Config{
			Integrations: &robottypes.Integrations{
				Discord: &robottypes.DiscordConfig{
					Enabled:  true,
					BotToken: dcBotToken,
					AppID:    dcAppID,
				},
			},
		},
	}

	a.Apply(context.Background(), robot)

	a.mu.RLock()
	entry, ok := a.bots["robot_e2e_dc_adapter"]
	a.mu.RUnlock()

	require.True(t, ok, "bot should be registered")
	assert.Equal(t, dcBotToken, entry.bot.Token())
	assert.Equal(t, dcAppID, entry.appID)

	t.Logf("OK  Apply: discord bot registered robot=%s app=%s", robot.MemberID, entry.appID)
}

// TestE2E_Adapter_Apply_Update verifies re-Apply with same token is a no-op.
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
		MemberID: "robot_e2e_dc_update",
		TeamID:   "team_e2e_dc",
		Config: &robottypes.Config{
			Integrations: &robottypes.Integrations{
				Discord: &robottypes.DiscordConfig{
					Enabled:  true,
					BotToken: dcBotToken,
					AppID:    dcAppID,
				},
			},
		},
	}

	a.Apply(context.Background(), robot)
	a.mu.RLock()
	_, ok := a.bots["robot_e2e_dc_update"]
	a.mu.RUnlock()
	require.True(t, ok)

	a.Apply(context.Background(), robot)
	a.mu.RLock()
	assert.Len(t, a.bots, 1)
	a.mu.RUnlock()

	a.Remove(context.Background(), "robot_e2e_dc_update")
	a.mu.RLock()
	_, ok = a.bots["robot_e2e_dc_update"]
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

	key := "dc:test-robot:msg-12345"
	assert.True(t, a.dedup.markSeen(key), "first time should return true")
	assert.False(t, a.dedup.markSeen(key), "second time should return false (dedup)")
	t.Log("OK  dedup working correctly")
}

// TestE2E_Adapter_HandleMessages verifies message handling.
func TestE2E_Adapter_HandleMessages(t *testing.T) {
	skipIfNoToken(t)

	bot, err := dcapi.NewBot(dcBotToken, dcAppID)
	require.NoError(t, err)

	a := &Adapter{
		bots:   make(map[string]*botEntry),
		appIdx: make(map[string]string),
		dedup:  newDedupStore(),
		stopCh: make(chan struct{}),
	}
	defer close(a.stopCh)

	entry := &botEntry{
		robotID: "robot_e2e_dc_handle",
		appID:   dcAppID,
		bot:     bot,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cms := []*dcapi.ConvertedMessage{
		{
			MessageID:  "test_msg_1",
			ChannelID:  "test_ch_1",
			AuthorID:   "test_user_1",
			AuthorName: "TestUser",
			Text:       "Hello from E2E test",
		},
	}

	a.handleMessages(ctx, entry, cms)

	assert.False(t, a.dedup.markSeen("dc:robot_e2e_dc_handle:test_msg_1"),
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
		MemberID: "robot_e2e_dc_disabled",
		TeamID:   "team_e2e_dc",
		Config: &robottypes.Config{
			Integrations: &robottypes.Integrations{
				Discord: &robottypes.DiscordConfig{
					Enabled:  false,
					BotToken: "some_token",
				},
			},
		},
	}

	a.Apply(context.Background(), robot)
	a.mu.RLock()
	_, ok := a.bots["robot_e2e_dc_disabled"]
	a.mu.RUnlock()
	assert.False(t, ok, "disabled bot should not be registered")
	t.Log("OK  disabled config not registered")
}

// TestE2E_BotUser verifies real Discord credentials.
func TestE2E_BotUser(t *testing.T) {
	skipIfNoToken(t)

	bot, err := dcapi.NewBot(dcBotToken, dcAppID)
	require.NoError(t, err)

	user, err := bot.BotUser()
	require.NoError(t, err)
	assert.NotEmpty(t, user.ID)
	assert.NotEmpty(t, user.Username)
	assert.True(t, user.Bot)
	t.Logf("OK  Discord bot verified: id=%s username=%s", user.ID, user.Username)
}
