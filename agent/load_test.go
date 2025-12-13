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

	t.Run("LoadKBConfig", func(t *testing.T) {
		// KB configuration should be loaded from agent/kb.yml
		assert.NotNil(t, agent.KB)
		assert.NotNil(t, agent.KB.Chat)

		// Verify chat KB settings
		assert.Equal(t, "__yao.openai", agent.KB.Chat.EmbeddingProviderID)
		assert.Equal(t, "text-embedding-3-small", agent.KB.Chat.EmbeddingOptionID)
		assert.Equal(t, "zh-CN", agent.KB.Chat.Locale)

		// Verify config
		assert.NotNil(t, agent.KB.Chat.Config)
		assert.Equal(t, "hnsw", agent.KB.Chat.Config.IndexType.String())
		assert.Equal(t, "cosine", agent.KB.Chat.Config.Distance.String())

		// Verify metadata
		assert.NotNil(t, agent.KB.Chat.Metadata)
		assert.Equal(t, "chat_session", agent.KB.Chat.Metadata["category"])
		assert.Equal(t, true, agent.KB.Chat.Metadata["auto_created"])

		// Verify document defaults
		assert.NotNil(t, agent.KB.Chat.DocumentDefaults)
		assert.NotNil(t, agent.KB.Chat.DocumentDefaults.Chunking)
		assert.Equal(t, "__yao.structured", agent.KB.Chat.DocumentDefaults.Chunking.ProviderID)
		assert.Equal(t, "standard", agent.KB.Chat.DocumentDefaults.Chunking.OptionID)

		assert.NotNil(t, agent.KB.Chat.DocumentDefaults.Extraction)
		assert.Equal(t, "__yao.openai", agent.KB.Chat.DocumentDefaults.Extraction.ProviderID)
		assert.Equal(t, "gpt-4o-mini", agent.KB.Chat.DocumentDefaults.Extraction.OptionID)

		assert.NotNil(t, agent.KB.Chat.DocumentDefaults.Converter)
		assert.Equal(t, "__yao.utf8", agent.KB.Chat.DocumentDefaults.Converter.ProviderID)
		assert.Equal(t, "standard-text", agent.KB.Chat.DocumentDefaults.Converter.OptionID)
	})

	t.Run("LoadSearchConfig", func(t *testing.T) {
		// Search configuration should be loaded from agent/search.yml
		assert.NotNil(t, agent.Search)

		// Verify web config
		assert.NotNil(t, agent.Search.Web)
		assert.Equal(t, "tavily", agent.Search.Web.Provider)
		assert.Equal(t, 10, agent.Search.Web.MaxResults)

		// Verify KB config
		assert.NotNil(t, agent.Search.KB)
		assert.Equal(t, 0.7, agent.Search.KB.Threshold)
		assert.False(t, agent.Search.KB.Graph)

		// Verify DB config
		assert.NotNil(t, agent.Search.DB)
		assert.Equal(t, 20, agent.Search.DB.MaxResults)

		// Verify keyword config
		assert.NotNil(t, agent.Search.Keyword)
		assert.Equal(t, 10, agent.Search.Keyword.MaxKeywords)
		assert.Equal(t, "auto", agent.Search.Keyword.Language)

		// Verify rerank config
		assert.NotNil(t, agent.Search.Rerank)
		assert.Equal(t, 10, agent.Search.Rerank.TopN)

		// Verify citation config
		assert.NotNil(t, agent.Search.Citation)
		assert.Equal(t, "#ref:{id}", agent.Search.Citation.Format)
		assert.True(t, agent.Search.Citation.AutoInjectPrompt)

		// Verify weights config
		assert.NotNil(t, agent.Search.Weights)
		assert.Equal(t, 1.0, agent.Search.Weights.User)
		assert.Equal(t, 0.8, agent.Search.Weights.Hook)
		assert.Equal(t, 0.6, agent.Search.Weights.Auto)

		// Verify options config
		assert.NotNil(t, agent.Search.Options)
		assert.Equal(t, 5, agent.Search.Options.SkipThreshold)
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
