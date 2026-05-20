//go:build e2e

package discord_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/robot/events/integrations/discord"
	robottypes "github.com/yaoapp/yao/agent/robot/types"
	dcapi "github.com/yaoapp/yao/integrations/discord"
)

func TestDiscordAdapter(t *testing.T) {
	botToken := os.Getenv("DISCORD_TEST_BOT_TOKEN")
	if botToken == "" {
		t.Fatal("DISCORD_TEST_BOT_TOKEN is required for this test")
	}
	appID := os.Getenv("DISCORD_TEST_APP_ID")
	if appID == "" {
		t.Fatal("DISCORD_TEST_APP_ID is required for this test")
	}

	t.Run("Apply", func(t *testing.T) {
		a := discord.NewTestAdapter()
		defer a.Close()

		robot := &robottypes.Robot{
			MemberID: "robot_e2e_dc_adapter",
			TeamID:   "team_e2e_dc",
			Config: &robottypes.Config{
				Integrations: &robottypes.Integrations{
					Discord: &robottypes.DiscordConfig{
						Enabled:  true,
						BotToken: botToken,
						AppID:    appID,
					},
				},
			},
		}

		a.Apply(context.Background(), robot)

		bot := a.GetBot("robot_e2e_dc_adapter")
		require.NotNil(t, bot, "bot should be registered")
		t.Logf("OK  Apply: discord bot registered robot=%s app=%s", robot.MemberID, appID)
	})

	t.Run("Apply_Update", func(t *testing.T) {
		a := discord.NewTestAdapter()
		defer a.Close()

		robot := &robottypes.Robot{
			MemberID: "robot_e2e_dc_update",
			TeamID:   "team_e2e_dc",
			Config: &robottypes.Config{
				Integrations: &robottypes.Integrations{
					Discord: &robottypes.DiscordConfig{
						Enabled:  true,
						BotToken: botToken,
						AppID:    appID,
					},
				},
			},
		}

		a.Apply(context.Background(), robot)
		require.NotNil(t, a.GetBot("robot_e2e_dc_update"))

		a.Apply(context.Background(), robot)
		assert.Equal(t, 1, a.BotCount())

		a.Remove(context.Background(), "robot_e2e_dc_update")
		assert.Nil(t, a.GetBot("robot_e2e_dc_update"), "bot should be removed")
		t.Log("OK  Apply/Remove lifecycle verified")
	})

	t.Run("Dedup", func(t *testing.T) {
		a := discord.NewTestAdapter()
		defer a.Close()

		key := "dc:test-robot:msg-12345"
		assert.True(t, a.MarkSeen(key), "first time should return true")
		assert.False(t, a.MarkSeen(key), "second time should return false (dedup)")
		t.Log("OK  dedup working correctly")
	})

	t.Run("HandleMessages", func(t *testing.T) {
		bot, err := dcapi.NewBot(botToken, appID)
		require.NoError(t, err)
		_ = bot

		a := discord.NewTestAdapterWithBot("robot_e2e_dc_handle", botToken, appID)
		defer a.Close()

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

		a.HandleMessagesForTest(ctx, "robot_e2e_dc_handle", cms)

		assert.False(t, a.MarkSeen("dc:robot_e2e_dc_handle:test_msg_1"),
			"message should be marked as seen after handleMessages")
		t.Log("OK  handleMessages processed 1 message")
	})

	t.Run("ApplyDisabled", func(t *testing.T) {
		a := discord.NewTestAdapter()
		defer a.Close()

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
		assert.Nil(t, a.GetBot("robot_e2e_dc_disabled"), "disabled bot should not be registered")
		t.Log("OK  disabled config not registered")
	})

	t.Run("BotUser", func(t *testing.T) {
		bot, err := dcapi.NewBot(botToken, appID)
		require.NoError(t, err)

		user, err := bot.BotUser()
		require.NoError(t, err)
		assert.NotEmpty(t, user.ID)
		assert.NotEmpty(t, user.Username)
		assert.True(t, user.Bot)
		t.Logf("OK  Discord bot verified: id=%s username=%s", user.ID, user.Username)
	})
}
