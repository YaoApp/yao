package llm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListPresetProviders(t *testing.T) {
	providers := ListPresetProviders()
	assert.GreaterOrEqual(t, len(providers), 40, "expected at least 40 providers from PiG catalog")

	has := func(id string) bool {
		for _, p := range providers {
			if p == id {
				return true
			}
		}
		return false
	}
	assert.True(t, has("openai"), "openai must be in catalog")
	assert.True(t, has("anthropic"), "anthropic must be in catalog")
	assert.True(t, has("deepseek"), "deepseek must be in catalog")
	assert.True(t, has("google"), "google must be in catalog")
	assert.True(t, has("openrouter"), "openrouter must be in catalog")
}

func TestListPresets(t *testing.T) {
	models := ListPresets("openai")
	assert.Greater(t, len(models), 30, "openai should have 30+ models")

	models = ListPresets("deepseek")
	assert.Greater(t, len(models), 0, "deepseek should have models")

	models = ListPresets("nonexistent")
	assert.Nil(t, models, "unknown provider returns nil")
}

func TestListPresetsOpenRouterHasAllTypes(t *testing.T) {
	models := ListPresets("openrouter")
	assert.Greater(t, len(models), 400, "openrouter should have 400+ total models")
}

func TestLookupPreset(t *testing.T) {
	t.Run("found chat model", func(t *testing.T) {
		m := LookupPreset("openai", "gpt-4.1")
		require.NotNil(t, m)
		assert.Equal(t, "gpt-4.1", m.ModelID)
		assert.Equal(t, "GPT-4.1", m.Name)
		assert.Greater(t, m.ContextWindow, 0)
		assert.Greater(t, m.MaxOutputTokens, 0)
		assert.Greater(t, m.InputCostPer1M, 0.0)
	})

	t.Run("found reasoning model with thinking map", func(t *testing.T) {
		m := LookupPreset("deepseek", "deepseek-flash")
		require.NotNil(t, m)
		assert.True(t, m.Reasoning)
		assert.NotNil(t, m.ThinkingLevelMap, "reasoning model should have thinking_level_map")
	})

	t.Run("found model with compat", func(t *testing.T) {
		m := LookupPreset("deepseek", "deepseek-flash")
		require.NotNil(t, m)
		assert.NotNil(t, m.Compat, "deepseek should have compat config")
	})

	t.Run("not found returns nil", func(t *testing.T) {
		m := LookupPreset("openai", "nonexistent-model")
		assert.Nil(t, m)
	})

	t.Run("returns copy not reference", func(t *testing.T) {
		m1 := LookupPreset("openai", "gpt-4.1")
		m2 := LookupPreset("openai", "gpt-4.1")
		require.NotNil(t, m1)
		require.NotNil(t, m2)
		m1.Name = "modified"
		assert.NotEqual(t, m1.Name, m2.Name, "should return independent copies")
	})
}

func TestApplyPreset(t *testing.T) {
	t.Run("fills empty fields from preset", func(t *testing.T) {
		m := &Model{ModelID: "gpt-4.1"}
		ApplyPreset(m, "openai")
		assert.Equal(t, "GPT-4.1", m.Name)
		assert.Greater(t, m.ContextWindow, 0)
		assert.Greater(t, m.MaxOutputTokens, 0)
		assert.Greater(t, m.InputCostPer1M, 0.0)
	})

	t.Run("user values take priority", func(t *testing.T) {
		m := &Model{
			ModelID:       "gpt-4.1",
			Name:          "My Custom Name",
			ContextWindow: 999,
		}
		ApplyPreset(m, "openai")
		assert.Equal(t, "My Custom Name", m.Name, "user name should not be overwritten")
		assert.Equal(t, 999, m.ContextWindow, "user context_window should not be overwritten")
		assert.Greater(t, m.MaxOutputTokens, 0, "max_output_tokens should be filled from preset")
	})

	t.Run("fills thinking map for reasoning model", func(t *testing.T) {
		m := &Model{ModelID: "deepseek-flash"}
		ApplyPreset(m, "deepseek")
		assert.True(t, m.Reasoning)
		assert.NotNil(t, m.ThinkingLevelMap)
		assert.NotEmpty(t, m.MaxThinking, "should infer max_thinking from map")
	})

	t.Run("fills capabilities-derived booleans", func(t *testing.T) {
		pm := LookupPreset("openai", "gpt-4.1")
		require.NotNil(t, pm)

		m := &Model{ModelID: "gpt-4.1"}
		ApplyPreset(m, "openai")
		assert.True(t, m.SupportsImages, "should derive supports_images from capabilities")
	})

	t.Run("noop for unknown model", func(t *testing.T) {
		m := &Model{ModelID: "nonexistent"}
		ApplyPreset(m, "openai")
		assert.Empty(t, m.Name, "no changes for unknown model")
	})

	t.Run("fills input limits", func(t *testing.T) {
		m := &Model{ModelID: "gpt-4.1"}
		ApplyPreset(m, "openai")
		assert.NotNil(t, m.InputLimits, "gpt-4.1 should have input limits")
	})
}

func TestInferMaxThinking(t *testing.T) {
	assert.Equal(t, ThinkingMax, inferMaxThinking(map[string]*string{
		"max":    nil,
		"high":   ptrStr("high"),
		"medium": ptrStr("medium"),
	}))

	assert.Equal(t, ThinkingHigh, inferMaxThinking(map[string]*string{
		"high":   ptrStr("high"),
		"medium": ptrStr("medium"),
	}))

	assert.Equal(t, ThinkingLevel(""), inferMaxThinking(map[string]*string{}))
}

func ptrStr(s string) *string { return &s }
