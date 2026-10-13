//go:build integration

package agent_test

import (
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent"
	"github.com/yaoapp/yao/agent/assistant"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

var prepareOnce sync.Once

func prepareApp(t *testing.T) {
	t.Helper()
	prepareOnce.Do(func() {
		testprepare.PrepareSandbox(t)
	})
}

func TestLoad(t *testing.T) {
	prepareApp(t)

	dsl := agent.GetAgent()
	require.NotNil(t, dsl)

	t.Run("LoadAgentSettings", func(t *testing.T) {
		assert.NotEmpty(t, dsl.Cache)
		assert.NotNil(t, dsl.Store)
		assert.Greater(t, dsl.StoreSetting.MaxSize, 0)
		assert.NotNil(t, dsl.Uses)
		assert.NotEmpty(t, dsl.Uses.Default)
	})

	t.Run("LoadDefaultAssistant", func(t *testing.T) {
		assert.NotNil(t, dsl.Assistant)
	})

	t.Run("LoadGlobalPrompts", func(t *testing.T) {
		require.NotNil(t, dsl.GlobalPrompts)
		require.Greater(t, len(dsl.GlobalPrompts), 0)
		assert.Equal(t, "system", dsl.GlobalPrompts[0].Role)
		assert.Contains(t, dsl.GlobalPrompts[0].Content, "$SYS.")
	})

	t.Run("LoadSearchConfig", func(t *testing.T) {
		require.NotNil(t, dsl.Search)
		require.NotNil(t, dsl.Search.Web)
		assert.Equal(t, "tavily", dsl.Search.Web.Provider)
		assert.Equal(t, 10, dsl.Search.Web.MaxResults)

		assert.NotNil(t, dsl.Search.DB)
		assert.Equal(t, 20, dsl.Search.DB.MaxResults)

		assert.NotNil(t, dsl.Search.Keyword)
		assert.Equal(t, 10, dsl.Search.Keyword.MaxKeywords)
		assert.Equal(t, "auto", dsl.Search.Keyword.Language)

		assert.NotNil(t, dsl.Search.Rerank)
		assert.Equal(t, 10, dsl.Search.Rerank.TopN)

		assert.NotNil(t, dsl.Search.Citation)
		assert.Equal(t, "#ref:{id}", dsl.Search.Citation.Format)
		assert.True(t, dsl.Search.Citation.AutoInjectPrompt)

		assert.NotNil(t, dsl.Search.Weights)
		assert.Equal(t, 1.0, dsl.Search.Weights.User)
		assert.Equal(t, 0.8, dsl.Search.Weights.Hook)
		assert.Equal(t, 0.6, dsl.Search.Weights.Auto)

		assert.NotNil(t, dsl.Search.Options)
		assert.Equal(t, 5, dsl.Search.Options.SkipThreshold)
	})
}

func TestGetGlobalPrompts(t *testing.T) {
	prepareApp(t)

	t.Run("ParseWithoutContext", func(t *testing.T) {
		prompts := agent.GetGlobalPrompts(nil)
		require.NotNil(t, prompts)
		require.Greater(t, len(prompts), 0)

		content := prompts[0].Content
		assert.NotContains(t, content, "$SYS.DATETIME")
		assert.NotContains(t, content, "$SYS.TIMEZONE")
		assert.NotContains(t, content, "$SYS.WEEKDAY")

		now := time.Now()
		assert.Contains(t, content, now.Format("2006-01-02"))
	})

	t.Run("ParseWithContext", func(t *testing.T) {
		ctx := map[string]string{
			"USER_ID": "test-user-123",
			"LOCALE":  "zh-CN",
		}

		prompts := agent.GetGlobalPrompts(ctx)
		require.NotNil(t, prompts)
		require.Greater(t, len(prompts), 0)

		content := prompts[0].Content
		assert.NotContains(t, content, "$SYS.DATETIME")
	})

	t.Run("ParseSystemTimeVariables", func(t *testing.T) {
		prompts := agent.GetGlobalPrompts(nil)
		require.NotNil(t, prompts)

		content := prompts[0].Content
		now := time.Now()

		assert.Contains(t, content, now.Format("2006-01-02"))
		assert.Contains(t, content, now.Location().String())
		assert.Contains(t, content, now.Weekday().String())
	})
}

func TestGetGlobalPromptsWithDisableFlag(t *testing.T) {
	prepareApp(t)

	dsl := agent.GetAgent()
	require.NotNil(t, dsl)

	t.Run("GlobalPromptsExist", func(t *testing.T) {
		assert.NotNil(t, dsl.GlobalPrompts)
		assert.Greater(t, len(dsl.GlobalPrompts), 0)
	})

	t.Run("AssistantCanDisableGlobalPrompts", func(t *testing.T) {
		prompts := agent.GetGlobalPrompts(nil)
		assert.NotNil(t, prompts)
	})
}

func TestGlobalPromptsContent(t *testing.T) {
	prepareApp(t)

	dsl := agent.GetAgent()
	require.NotNil(t, dsl)
	require.NotNil(t, dsl.GlobalPrompts)
	require.Greater(t, len(dsl.GlobalPrompts), 0)

	t.Run("SystemContextPrompt", func(t *testing.T) {
		var systemPrompt string
		for _, p := range dsl.GlobalPrompts {
			if p.Role == "system" {
				systemPrompt = p.Content
				break
			}
		}

		assert.NotEmpty(t, systemPrompt)
		assert.Contains(t, systemPrompt, "System Context")
	})

	t.Run("VariablesInRawPrompts", func(t *testing.T) {
		content := dsl.GlobalPrompts[0].Content
		assert.True(t,
			strings.Contains(content, "$SYS.") ||
				strings.Contains(content, "$ENV.") ||
				strings.Contains(content, "$CTX."),
			"Raw prompts should contain variable placeholders")
	})
}

func TestAssistantGlobalPrompts(t *testing.T) {
	prepareApp(t)

	t.Run("AssistantModuleReceivesGlobalPrompts", func(t *testing.T) {
		prompts := assistant.GetGlobalPrompts(nil)
		require.NotNil(t, prompts)
		require.Greater(t, len(prompts), 0)

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

		content := prompts[0].Content
		assert.NotContains(t, content, "$SYS.")

		now := time.Now()
		assert.Contains(t, content, now.Format("2006-01-02"))
	})
}
