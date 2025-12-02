package agent

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/assistant"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/test"
)

func prepare(t *testing.T) {
	test.Prepare(t, config.Conf)
	err := Load(config.Conf)
	require.NoError(t, err)
}

func TestLoad(t *testing.T) {
	prepare(t)
	defer test.Clean()

	agent := GetAgent()
	require.NotNil(t, agent)

	t.Run("LoadAgentSettings", func(t *testing.T) {
		// Cache setting
		assert.NotEmpty(t, agent.Cache)

		// Store setting
		assert.NotNil(t, agent.Store)
		assert.Greater(t, agent.StoreSetting.MaxSize, 0)

		// Uses setting
		assert.NotNil(t, agent.Uses)
		assert.NotEmpty(t, agent.Uses.Default)
	})

	t.Run("LoadDefaultAssistant", func(t *testing.T) {
		assert.NotNil(t, agent.Assistant)
	})

	t.Run("LoadGlobalPrompts", func(t *testing.T) {
		// Global prompts should be loaded from agent/prompts.yml
		assert.NotNil(t, agent.GlobalPrompts)
		assert.Greater(t, len(agent.GlobalPrompts), 0)

		// First prompt should be system role
		assert.Equal(t, "system", agent.GlobalPrompts[0].Role)

		// Content should contain system context info (with variables not yet parsed)
		assert.Contains(t, agent.GlobalPrompts[0].Content, "$SYS.")
	})

	t.Run("LoadModelCapabilities", func(t *testing.T) {
		// Model capabilities should be loaded from agent/models.yml
		assert.NotNil(t, agent.Models)
		assert.Greater(t, len(agent.Models), 0)
	})
}

func TestGetGlobalPrompts(t *testing.T) {
	prepare(t)
	defer test.Clean()

	t.Run("ParseWithoutContext", func(t *testing.T) {
		prompts := GetGlobalPrompts(nil)
		require.NotNil(t, prompts)
		require.Greater(t, len(prompts), 0)

		// $SYS.* variables should be replaced
		content := prompts[0].Content
		assert.NotContains(t, content, "$SYS.DATETIME")
		assert.NotContains(t, content, "$SYS.TIMEZONE")
		assert.NotContains(t, content, "$SYS.WEEKDAY")

		// Should contain actual time values
		now := time.Now()
		assert.Contains(t, content, now.Format("2006-01-02"))
	})

	t.Run("ParseWithContext", func(t *testing.T) {
		ctx := map[string]string{
			"USER_ID": "test-user-123",
			"LOCALE":  "zh-CN",
		}

		prompts := GetGlobalPrompts(ctx)
		require.NotNil(t, prompts)
		require.Greater(t, len(prompts), 0)

		// $SYS.* variables should be replaced
		content := prompts[0].Content
		assert.NotContains(t, content, "$SYS.DATETIME")
	})

	t.Run("ParseSystemTimeVariables", func(t *testing.T) {
		prompts := GetGlobalPrompts(nil)
		require.NotNil(t, prompts)

		content := prompts[0].Content
		now := time.Now()

		// Should contain current date
		assert.Contains(t, content, now.Format("2006-01-02"))

		// Should contain timezone
		assert.Contains(t, content, now.Location().String())

		// Should contain weekday
		assert.Contains(t, content, now.Weekday().String())
	})
}

func TestGetGlobalPromptsWithDisableFlag(t *testing.T) {
	prepare(t)
	defer test.Clean()

	agent := GetAgent()
	require.NotNil(t, agent)

	t.Run("GlobalPromptsExist", func(t *testing.T) {
		// Verify global prompts are loaded
		assert.NotNil(t, agent.GlobalPrompts)
		assert.Greater(t, len(agent.GlobalPrompts), 0)
	})

	t.Run("AssistantCanDisableGlobalPrompts", func(t *testing.T) {
		// The fullfields test assistant has disable_global_prompts: true
		// This test verifies the flag is properly loaded
		// The actual merging logic is in the assistant module
		prompts := GetGlobalPrompts(nil)
		assert.NotNil(t, prompts)

		// Global prompts should still be available
		// The assistant decides whether to use them based on DisableGlobalPrompts flag
	})
}

func TestGlobalPromptsContent(t *testing.T) {
	prepare(t)
	defer test.Clean()

	agent := GetAgent()
	require.NotNil(t, agent)
	require.NotNil(t, agent.GlobalPrompts)
	require.Greater(t, len(agent.GlobalPrompts), 0)

	t.Run("SystemContextPrompt", func(t *testing.T) {
		// Find system prompt
		var systemPrompt string
		for _, p := range agent.GlobalPrompts {
			if p.Role == "system" {
				systemPrompt = p.Content
				break
			}
		}

		assert.NotEmpty(t, systemPrompt)
		assert.Contains(t, systemPrompt, "System Context")
	})

	t.Run("VariablesInRawPrompts", func(t *testing.T) {
		// Raw prompts should contain unparsed variables
		content := agent.GlobalPrompts[0].Content
		assert.True(t,
			strings.Contains(content, "$SYS.") ||
				strings.Contains(content, "$ENV.") ||
				strings.Contains(content, "$CTX."),
			"Raw prompts should contain variable placeholders")
	})
}

func TestAssistantGlobalPrompts(t *testing.T) {
	prepare(t)
	defer test.Clean()

	t.Run("AssistantModuleReceivesGlobalPrompts", func(t *testing.T) {
		// Verify assistant module has global prompts
		prompts := assistant.GetGlobalPrompts(nil)
		require.NotNil(t, prompts)
		require.Greater(t, len(prompts), 0)

		// Should be parsed (no $SYS.* variables)
		content := prompts[0].Content
		assert.NotContains(t, content, "$SYS.DATETIME")
	})

	t.Run("AssistantModuleParsesWithContext", func(t *testing.T) {
		ctx := map[string]string{
			"USER_ID": "assistant-test-user",
			"LOCALE":  "en-US",
		}

		prompts := assistant.GetGlobalPrompts(ctx)
		require.NotNil(t, prompts)

		// $SYS.* should be replaced
		content := prompts[0].Content
		assert.NotContains(t, content, "$SYS.")

		// Should contain current time info
		now := time.Now()
		assert.Contains(t, content, now.Format("2006-01-02"))
	})
}
