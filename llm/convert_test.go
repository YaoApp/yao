package llm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToConnectorProfile(t *testing.T) {
	high := "high"
	max := "max"

	p := &Provider{
		ID:         1,
		ProviderID: ProviderDeepSeek,
		API:        APIOpenAICompletions,
		BaseURL:    "https://api.deepseek.com",
		APIKey:     "sk-test",
		Compat: JSONMap{
			"thinkingFormat":        "deepseek",
			"supportsDeveloperRole": false,
		},
	}
	m := &Model{
		ModelID:         "deepseek-flash",
		ContextWindow:   1000000,
		MaxOutputTokens: 384000,
		Reasoning:       true,
		Input:           []string{"text", "image"},
		ThinkingLevelMap: ThinkingLevelMap{
			ThinkingHigh: &high,
			ThinkingMax:  &max,
			ThinkingOff:  nil,
		},
	}

	cp := ToConnectorProfile(p, m)

	assert.Equal(t, "sk-test", cp.APIKey)
	assert.Equal(t, "https://api.deepseek.com", cp.BaseURL)
	assert.Equal(t, "openai-completions", cp.API)
	assert.Equal(t, "deepseek-flash", cp.Model)
	assert.Equal(t, []string{"text", "image"}, cp.Input)
	assert.Equal(t, 1000000, cp.ContextWindow)
	assert.Equal(t, 384000, cp.MaxTokens)
	assert.Equal(t, "high", cp.Reasoning)
	assert.Equal(t, "deepseek", cp.ThinkingFormat)
	assert.True(t, cp.NoDeveloperRole)
	assert.False(t, cp.DisableReasoning)
	assert.NotNil(t, cp.ReasoningEfforts)
	assert.Equal(t, "high", cp.ReasoningEfforts["high"])
	assert.Equal(t, "max", cp.ReasoningEfforts["max"])
}

func TestToConnectorProfile_NonReasoning(t *testing.T) {
	p := &Provider{API: APIOpenAICompletions, BaseURL: "https://api.openai.com/v1"}
	m := &Model{ModelID: "gpt-4o", Reasoning: false, Input: []string{"text"}}

	cp := ToConnectorProfile(p, m)
	assert.True(t, cp.DisableReasoning)
	assert.Empty(t, cp.Reasoning)
	assert.Nil(t, cp.ReasoningEfforts)
}

func TestToPiAiRoute(t *testing.T) {
	high := "high"
	max := "max"

	p := &Provider{
		API:     APIOpenAICompletions,
		BaseURL: "https://api.deepseek.com",
		Compat: JSONMap{
			"thinkingFormat":        "deepseek",
			"supportsDeveloperRole": false,
		},
	}
	m := &Model{
		ModelID:         "deepseek-flash",
		ContextWindow:   1000000,
		MaxOutputTokens: 384000,
		Reasoning:       true,
		Input:           []string{"text", "image"},
		ThinkingLevelMap: ThinkingLevelMap{
			ThinkingHigh: &high,
			ThinkingMax:  &max,
			ThinkingOff:  nil,
		},
	}

	route := ToPiAiRoute(p, m, "yao-primary", "DSH_KEY_PRIMARY")

	assert.Equal(t, "yao-primary", route.Name)
	assert.Equal(t, "openai-completions", route.API)
	assert.Equal(t, "https://api.deepseek.com", route.BaseURL)
	assert.Equal(t, "DSH_KEY_PRIMARY", route.APIKeyEnv)
	assert.Equal(t, "high", route.Reasoning)
	assert.Equal(t, "deepseek", route.ThinkingFormat)
	assert.True(t, route.NoDeveloperRole)

	assert.Len(t, route.Models, 1)
	mc := route.Models[0]
	assert.Equal(t, "deepseek-flash", mc.ID)
	assert.Equal(t, []string{"text", "image"}, mc.Input)
	assert.Equal(t, 1000000, mc.ContextWindow)
	assert.Equal(t, 384000, mc.MaxTokens)
	assert.NotNil(t, mc.ReasoningEfforts)
	assert.False(t, mc.DisableReasoning)
}

func TestToPiAiRoute_ModelAPI(t *testing.T) {
	p := &Provider{API: APIAnthropicMessages, BaseURL: "https://api.deepseek.com/anthropic"}
	m := &Model{
		ModelID:  "deepseek-flash-thinking-low",
		ModelAPI: "deepseek-flash",
		Input:    []string{"text", "image"},
	}
	route := ToPiAiRoute(p, m, "r", "K")
	require.Len(t, route.Models, 1)
	assert.Equal(t, "deepseek-flash", route.Models[0].ID, "ModelAPI should take priority over ModelID")
}

func TestToConnectorProfile_ModelAPI(t *testing.T) {
	p := &Provider{API: APIAnthropicMessages, BaseURL: "https://api.deepseek.com/anthropic"}
	m := &Model{
		ModelID:  "deepseek-flash-thinking-low",
		ModelAPI: "deepseek-flash",
		Input:    []string{"text"},
	}
	cp := ToConnectorProfile(p, m)
	assert.Equal(t, "deepseek-flash", cp.Model, "ModelAPI should take priority over ModelID")
}

func TestToPiAiRoute_NonReasoning(t *testing.T) {
	p := &Provider{API: APIOpenAICompletions, BaseURL: "https://api.openai.com/v1"}
	m := &Model{ModelID: "gpt-4o", Input: []string{"text"}}

	route := ToPiAiRoute(p, m, "yao-primary", "KEY")
	assert.Empty(t, route.Reasoning)
	assert.Len(t, route.Models, 1)
	assert.True(t, route.Models[0].DisableReasoning)
}

func TestMergeCompat(t *testing.T) {
	t.Run("both nil", func(t *testing.T) {
		assert.Nil(t, mergeCompat(nil, nil))
	})

	t.Run("provider only", func(t *testing.T) {
		m := mergeCompat(JSONMap{"a": 1}, nil)
		assert.Equal(t, 1, m["a"])
	})

	t.Run("model overrides provider", func(t *testing.T) {
		m := mergeCompat(JSONMap{"a": 1, "b": 2}, JSONMap{"a": 10, "c": 3})
		assert.Equal(t, 10, m["a"])
		assert.Equal(t, 2, m["b"])
		assert.Equal(t, 3, m["c"])
	})
}

func TestInferDefaultReasoning(t *testing.T) {
	high := "high"
	max := "max"

	assert.Equal(t, "high", inferDefaultReasoning(ThinkingLevelMap{
		ThinkingHigh: &high,
		ThinkingMax:  &max,
	}))

	assert.Equal(t, "max", inferDefaultReasoning(ThinkingLevelMap{
		ThinkingMax: &max,
	}))

	assert.Equal(t, "", inferDefaultReasoning(nil))

	assert.Equal(t, "", inferDefaultReasoning(ThinkingLevelMap{
		ThinkingOff: nil,
	}), "only 'off' entry should return empty")
}

func TestFlattenThinkingMap(t *testing.T) {
	high := "high"
	m := flattenThinkingMap(ThinkingLevelMap{
		ThinkingHigh: &high,
		ThinkingOff:  nil,
	})
	assert.Equal(t, "high", m["high"])
	assert.Equal(t, "off", m["off"], "nil value should use level name as default")

	assert.Nil(t, flattenThinkingMap(nil))
}

// ---------------------------------------------------------------------------
// Edge cases
// ---------------------------------------------------------------------------

func TestToConnectorProfile_NilCompat(t *testing.T) {
	p := &Provider{API: APIOpenAICompletions, BaseURL: "https://example.com"}
	m := &Model{ModelID: "m1", Input: []string{"text"}}

	cp := ToConnectorProfile(p, m)
	assert.Empty(t, cp.ThinkingFormat)
	assert.False(t, cp.NoDeveloperRole)
	assert.Equal(t, 0, cp.BudgetTokens)
}

func TestToConnectorProfile_BudgetTokens(t *testing.T) {
	p := &Provider{
		API: APIOpenAICompletions,
		Compat: JSONMap{
			"budgetTokens": 10000,
		},
	}
	m := &Model{ModelID: "m1", Input: []string{"text"}}

	cp := ToConnectorProfile(p, m)
	assert.Equal(t, 10000, cp.BudgetTokens)
}

func TestToConnectorProfile_ModelCompatOverridesProvider(t *testing.T) {
	p := &Provider{
		API:    APIOpenAICompletions,
		Compat: JSONMap{"thinkingFormat": "provider-format"},
	}
	m := &Model{
		ModelID: "m1",
		Input:   []string{"text"},
		Compat:  JSONMap{"thinkingFormat": "model-format"},
	}

	cp := ToConnectorProfile(p, m)
	assert.Equal(t, "model-format", cp.ThinkingFormat, "model compat should override provider")
}

func TestToConnectorProfile_Protocols(t *testing.T) {
	p := &Provider{
		API:       APIOpenAICompletions,
		Protocols: []API{APIOpenAICompletions, APIAnthropicMessages},
	}
	m := &Model{ModelID: "m1", Input: []string{"text"}}

	cp := ToConnectorProfile(p, m)
	assert.Equal(t, []string{"openai-completions", "anthropic-messages"}, cp.Protocols)
}

func TestToConnectorProfile_NilProtocols(t *testing.T) {
	p := &Provider{API: APIOpenAICompletions}
	m := &Model{ModelID: "m1", Input: []string{"text"}}

	cp := ToConnectorProfile(p, m)
	assert.Nil(t, cp.Protocols)
}

func TestToPiAiRoute_BudgetTokens(t *testing.T) {
	p := &Provider{
		API: APIAnthropicMessages,
		Compat: JSONMap{
			"budgetTokens": 50000,
		},
	}
	m := &Model{
		ModelID:   "claude-sonnet",
		Input:     []string{"text"},
		Reasoning: true,
		ThinkingLevelMap: ThinkingLevelMap{
			ThinkingHigh: ptrStr("high"),
		},
	}

	route := ToPiAiRoute(p, m, "route", "KEY")
	assert.Equal(t, 50000, route.BudgetTokens)
}

func TestToPiAiRoute_NilCompat(t *testing.T) {
	p := &Provider{API: APIOpenAICompletions}
	m := &Model{ModelID: "m1", Input: []string{"text"}}

	route := ToPiAiRoute(p, m, "r", "K")
	assert.Equal(t, 0, route.BudgetTokens)
	assert.Empty(t, route.ThinkingFormat)
	assert.False(t, route.NoDeveloperRole)
}

func TestToStringSliceFromAPIs(t *testing.T) {
	assert.Nil(t, toStringSliceFromAPIs(nil))
	assert.Equal(t, []string{"openai-completions"}, toStringSliceFromAPIs([]API{APIOpenAICompletions}))
}

func TestGetCompatHelpers(t *testing.T) {
	t.Run("getCompatString nil map", func(t *testing.T) {
		assert.Equal(t, "", getCompatString(nil, "key"))
	})
	t.Run("getCompatBool nil map", func(t *testing.T) {
		assert.Nil(t, getCompatBool(nil, "key"))
	})
	t.Run("getCompatBool missing key", func(t *testing.T) {
		assert.Nil(t, getCompatBool(JSONMap{}, "missing"))
	})
	t.Run("getCompatBool non-bool value", func(t *testing.T) {
		assert.Nil(t, getCompatBool(JSONMap{"k": "not-bool"}, "k"))
	})
	t.Run("getCompatInt nil map", func(t *testing.T) {
		assert.Equal(t, 0, getCompatInt(nil, "key"))
	})
}
