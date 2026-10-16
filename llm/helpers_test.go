package llm

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/kun/maps"
)

// ---------------------------------------------------------------------------
// getString
// ---------------------------------------------------------------------------

func TestGetString(t *testing.T) {
	row := maps.MapStr{"name": "hello", "count": 42, "nil_val": nil}

	assert.Equal(t, "hello", getString(row, "name"))
	assert.Equal(t, "42", getString(row, "count"), "non-string falls back to Sprintf")
	assert.Equal(t, "", getString(row, "nil_val"))
	assert.Equal(t, "", getString(row, "missing"))
}

// ---------------------------------------------------------------------------
// getInt
// ---------------------------------------------------------------------------

func TestGetInt(t *testing.T) {
	row := maps.MapStr{
		"int":     42,
		"int64":   int64(100),
		"float64": float64(3.9),
		"jnum":    json.Number("77"),
		"nil_val": nil,
	}

	assert.Equal(t, 42, getInt(row, "int"))
	assert.Equal(t, 100, getInt(row, "int64"))
	assert.Equal(t, 3, getInt(row, "float64"), "truncates float")
	assert.Equal(t, 77, getInt(row, "jnum"))
	assert.Equal(t, 0, getInt(row, "nil_val"))
	assert.Equal(t, 0, getInt(row, "missing"))
}

// ---------------------------------------------------------------------------
// getFloat
// ---------------------------------------------------------------------------

func TestGetFloat(t *testing.T) {
	row := maps.MapStr{
		"f64":     2.5,
		"f32":     float32(1.5),
		"int":     10,
		"int64":   int64(20),
		"jnum":    json.Number("3.14"),
		"nil_val": nil,
		"str":     "not a number",
	}

	assert.InDelta(t, 2.5, getFloat(row, "f64"), 0.001)
	assert.InDelta(t, 1.5, getFloat(row, "f32"), 0.001)
	assert.InDelta(t, 10.0, getFloat(row, "int"), 0.001)
	assert.InDelta(t, 20.0, getFloat(row, "int64"), 0.001)
	assert.InDelta(t, 3.14, getFloat(row, "jnum"), 0.001)
	assert.InDelta(t, 0.0, getFloat(row, "nil_val"), 0.001)
	assert.InDelta(t, 0.0, getFloat(row, "missing"), 0.001)
	assert.InDelta(t, 0.0, getFloat(row, "str"), 0.001)
}

// ---------------------------------------------------------------------------
// getBool
// ---------------------------------------------------------------------------

func TestGetBool(t *testing.T) {
	row := maps.MapStr{
		"bool_t":  true,
		"bool_f":  false,
		"int_1":   1,
		"int_0":   0,
		"int64_1": int64(1),
		"float_1": float64(1.0),
		"float_0": float64(0),
		"nil_val": nil,
	}

	assert.True(t, getBool(row, "bool_t"))
	assert.False(t, getBool(row, "bool_f"))
	assert.True(t, getBool(row, "int_1"))
	assert.False(t, getBool(row, "int_0"))
	assert.True(t, getBool(row, "int64_1"))
	assert.True(t, getBool(row, "float_1"))
	assert.False(t, getBool(row, "float_0"))
	assert.False(t, getBool(row, "nil_val"))
	assert.False(t, getBool(row, "missing"))
}

// ---------------------------------------------------------------------------
// getTime
// ---------------------------------------------------------------------------

func TestGetTime(t *testing.T) {
	now := time.Date(2026, 10, 16, 14, 0, 0, 0, time.UTC)

	t.Run("time.Time value", func(t *testing.T) {
		row := maps.MapStr{"ts": now}
		assert.Equal(t, now, getTime(row, "ts"))
	})

	t.Run("RFC3339 string", func(t *testing.T) {
		row := maps.MapStr{"ts": "2026-10-16T14:00:00Z"}
		got := getTime(row, "ts")
		assert.Equal(t, now, got)
	})

	t.Run("datetime string", func(t *testing.T) {
		row := maps.MapStr{"ts": "2026-10-16 14:00:00"}
		got := getTime(row, "ts")
		assert.Equal(t, now, got)
	})

	t.Run("nil value", func(t *testing.T) {
		row := maps.MapStr{"ts": nil}
		assert.True(t, getTime(row, "ts").IsZero())
	})

	t.Run("missing key", func(t *testing.T) {
		row := maps.MapStr{}
		assert.True(t, getTime(row, "ts").IsZero())
	})

	t.Run("unparseable string", func(t *testing.T) {
		row := maps.MapStr{"ts": "not-a-date"}
		assert.True(t, getTime(row, "ts").IsZero())
	})
}

// ---------------------------------------------------------------------------
// toIntFromAny
// ---------------------------------------------------------------------------

func TestToIntFromAny(t *testing.T) {
	assert.Equal(t, 42, toIntFromAny(42))
	assert.Equal(t, 100, toIntFromAny(int64(100)))
	assert.Equal(t, 3, toIntFromAny(float64(3.7)))
	assert.Equal(t, 77, toIntFromAny(json.Number("77")))
	assert.Equal(t, 0, toIntFromAny(nil))
	assert.Equal(t, 0, toIntFromAny("hello"))
}

// ---------------------------------------------------------------------------
// toFloatFromAny
// ---------------------------------------------------------------------------

func TestToFloatFromAny(t *testing.T) {
	assert.InDelta(t, 2.5, toFloatFromAny(float64(2.5)), 0.001)
	assert.InDelta(t, 1.5, toFloatFromAny(float32(1.5)), 0.001)
	assert.InDelta(t, 10.0, toFloatFromAny(10), 0.001)
	assert.InDelta(t, 20.0, toFloatFromAny(int64(20)), 0.001)
	assert.InDelta(t, 3.14, toFloatFromAny(json.Number("3.14")), 0.001)
	assert.InDelta(t, 0.0, toFloatFromAny(nil), 0.001)
	assert.InDelta(t, 0.0, toFloatFromAny("nope"), 0.001)
}

// ---------------------------------------------------------------------------
// toJSONMap
// ---------------------------------------------------------------------------

func TestToJSONMap(t *testing.T) {
	t.Run("from map[string]interface{}", func(t *testing.T) {
		m := map[string]interface{}{"key": "val"}
		assert.Equal(t, JSONMap{"key": "val"}, toJSONMap(m))
	})

	t.Run("from JSON string", func(t *testing.T) {
		m := toJSONMap(`{"key":"val"}`)
		assert.Equal(t, "val", m["key"])
	})

	t.Run("from invalid JSON string", func(t *testing.T) {
		assert.Nil(t, toJSONMap("not json"))
	})

	t.Run("from nil", func(t *testing.T) {
		assert.Nil(t, toJSONMap(nil))
	})

	t.Run("from unsupported type", func(t *testing.T) {
		assert.Nil(t, toJSONMap(42))
	})
}

// ---------------------------------------------------------------------------
// toAPISlice
// ---------------------------------------------------------------------------

func TestToAPISlice(t *testing.T) {
	t.Run("from []interface{}", func(t *testing.T) {
		s := []interface{}{"openai-completions", "anthropic-messages"}
		got := toAPISlice(s)
		assert.Equal(t, []API{"openai-completions", "anthropic-messages"}, got)
	})

	t.Run("from []string", func(t *testing.T) {
		s := []string{"openai-completions"}
		got := toAPISlice(s)
		assert.Equal(t, []API{"openai-completions"}, got)
	})

	t.Run("from JSON string", func(t *testing.T) {
		got := toAPISlice(`["openai-completions"]`)
		assert.Equal(t, []API{"openai-completions"}, got)
	})

	t.Run("from nil", func(t *testing.T) {
		assert.Nil(t, toAPISlice(nil))
	})
}

// ---------------------------------------------------------------------------
// toStringSliceFromAny
// ---------------------------------------------------------------------------

func TestToStringSliceFromAny(t *testing.T) {
	t.Run("from []interface{}", func(t *testing.T) {
		s := []interface{}{"text", "image"}
		assert.Equal(t, []string{"text", "image"}, toStringSliceFromAny(s))
	})

	t.Run("from []string", func(t *testing.T) {
		s := []string{"text"}
		assert.Equal(t, []string{"text"}, toStringSliceFromAny(s))
	})

	t.Run("from JSON string", func(t *testing.T) {
		got := toStringSliceFromAny(`["text","image"]`)
		assert.Equal(t, []string{"text", "image"}, got)
	})

	t.Run("from nil", func(t *testing.T) {
		assert.Nil(t, toStringSliceFromAny(nil))
	})
}

// ---------------------------------------------------------------------------
// toMapStringAny
// ---------------------------------------------------------------------------

func TestToMapStringAny(t *testing.T) {
	t.Run("from map[string]interface{}", func(t *testing.T) {
		m := map[string]interface{}{"temp": 0.7}
		assert.Equal(t, map[string]any{"temp": 0.7}, toMapStringAny(m))
	})

	t.Run("from JSON string", func(t *testing.T) {
		got := toMapStringAny(`{"temp":0.7}`)
		assert.InDelta(t, 0.7, got["temp"], 0.001)
	})

	t.Run("from nil", func(t *testing.T) {
		assert.Nil(t, toMapStringAny(nil))
	})
}

// ---------------------------------------------------------------------------
// toThinkingLevelMap
// ---------------------------------------------------------------------------

func TestToThinkingLevelMap(t *testing.T) {
	t.Run("from map with string values", func(t *testing.T) {
		m := map[string]interface{}{
			"high": "high",
			"max":  nil,
		}
		got := toThinkingLevelMap(m)
		assert.NotNil(t, got[ThinkingHigh])
		assert.Equal(t, "high", *got[ThinkingHigh])
		assert.Nil(t, got[ThinkingMax])
	})

	t.Run("from JSON string", func(t *testing.T) {
		got := toThinkingLevelMap(`{"medium":"medium","max":null}`)
		assert.NotNil(t, got[ThinkingMedium])
		assert.Equal(t, "medium", *got[ThinkingMedium])
		assert.Nil(t, got[ThinkingMax])
	})

	t.Run("from nil", func(t *testing.T) {
		assert.Nil(t, toThinkingLevelMap(nil))
	})
}

// ---------------------------------------------------------------------------
// toSamplingParamsByLevel
// ---------------------------------------------------------------------------

func TestToSamplingParamsByLevel(t *testing.T) {
	t.Run("from map", func(t *testing.T) {
		m := map[string]interface{}{
			"high": map[string]interface{}{"temperature": 0.5},
		}
		got := toSamplingParamsByLevel(m)
		assert.InDelta(t, 0.5, got[ThinkingHigh]["temperature"], 0.001)
	})

	t.Run("skips non-map values", func(t *testing.T) {
		m := map[string]interface{}{
			"high":   map[string]interface{}{"t": 0.5},
			"medium": "invalid",
		}
		got := toSamplingParamsByLevel(m)
		assert.Len(t, got, 1)
		assert.Contains(t, got, ThinkingHigh)
	})

	t.Run("from nil", func(t *testing.T) {
		assert.Nil(t, toSamplingParamsByLevel(nil))
	})
}

// ---------------------------------------------------------------------------
// toCostTiers
// ---------------------------------------------------------------------------

func TestToCostTiers(t *testing.T) {
	t.Run("from []interface{}", func(t *testing.T) {
		arr := []interface{}{
			map[string]interface{}{
				"inputTokensAbove": float64(200000),
				"input":            1.5,
				"output":           6.0,
				"cacheRead":        0.15,
				"cacheWrite":       1.875,
			},
		}
		got := toCostTiers(arr)
		assert.Len(t, got, 1)
		assert.Equal(t, 200000, got[0].InputTokensAbove)
		assert.InDelta(t, 1.5, got[0].InputCostPer1M, 0.001)
		assert.InDelta(t, 6.0, got[0].OutputCostPer1M, 0.001)
		assert.InDelta(t, 0.15, got[0].CacheReadCostPer1M, 0.001)
		assert.InDelta(t, 1.875, got[0].CacheWriteCostPer1M, 0.001)
	})

	t.Run("from JSON string", func(t *testing.T) {
		j := `[{"inputTokensAbove":100,"input":2.0,"output":8.0,"cacheRead":0.2,"cacheWrite":2.5}]`
		got := toCostTiers(j)
		assert.Len(t, got, 1)
		assert.Equal(t, 100, got[0].InputTokensAbove)
	})

	t.Run("from nil", func(t *testing.T) {
		assert.Nil(t, toCostTiers(nil))
	})
}

// ---------------------------------------------------------------------------
// toInputLimits
// ---------------------------------------------------------------------------

func TestToInputLimits(t *testing.T) {
	t.Run("from map", func(t *testing.T) {
		m := map[string]interface{}{
			"maxRequestBytes": float64(4718592),
			"images": map[string]interface{}{
				"maxPerMessage": float64(20),
				"resize": map[string]interface{}{
					"maxWidth": float64(2000),
				},
			},
		}
		got := toInputLimits(m)
		assert.NotNil(t, got)
		assert.Equal(t, 4718592, got.MaxRequestBytes)
		assert.NotNil(t, got.Images)
		assert.Equal(t, 20, got.Images.MaxPerMessage)
		assert.NotNil(t, got.Images.Resize)
		assert.Equal(t, 2000, got.Images.Resize.MaxWidth)
	})

	t.Run("from JSON string", func(t *testing.T) {
		j := `{"maxRequestBytes":1000}`
		got := toInputLimits(j)
		assert.NotNil(t, got)
		assert.Equal(t, 1000, got.MaxRequestBytes)
	})

	t.Run("from nil", func(t *testing.T) {
		assert.Nil(t, toInputLimits(nil))
	})
}

// ---------------------------------------------------------------------------
// toPromptCache
// ---------------------------------------------------------------------------

func TestToPromptCache(t *testing.T) {
	t.Run("from map", func(t *testing.T) {
		m := map[string]interface{}{
			"live":   float64(300),
			"stored": float64(3600),
		}
		got := toPromptCache(m)
		assert.Equal(t, 300, got["live"])
		assert.Equal(t, 3600, got["stored"])
	})

	t.Run("from nil", func(t *testing.T) {
		assert.Nil(t, toPromptCache(nil))
	})
}

// ---------------------------------------------------------------------------
// Row conversion round-trips (struct → row → struct)
// ---------------------------------------------------------------------------

func TestProviderRowRoundtrip(t *testing.T) {
	p := &Provider{
		ID:         5,
		Name:       "My DeepSeek",
		ProviderID: ProviderDeepSeek,
		API:        APIOpenAICompletions,
		Protocols:  []API{APIOpenAICompletions, APIOpenAIResponses},
		BaseURL:    "https://api.deepseek.com",
		APIKey:     "sk-test",
		Headers:    JSONMap{"Authorization": "Bearer extra"},
		Compat:     JSONMap{"thinkingFormat": "deepseek"},
		Enabled:    true,
		Status:     "connected",
		PresetKey:  "deepseek-v3",
		Scope:      ScopeSystem,
		Priority:   1,
		Metadata:   JSONMap{"note": "testing"},
	}

	row := providerToRow(p)

	assert.Equal(t, 5, row["id"])
	assert.Equal(t, "deepseek", row["provider_id"])
	assert.Equal(t, "openai-completions", row["api"])
	assert.Equal(t, true, row["enabled"])
	assert.Equal(t, "deepseek-v3", row["preset_key"])
	assert.NotNil(t, row["protocols"])
	assert.NotNil(t, row["headers"])
	assert.NotNil(t, row["compat"])
	assert.NotNil(t, row["metadata"])

	// Direct in-memory roundtrip (toAPISlice handles []API natively)
	direct := rowToProvider(maps.MapStr(row))
	assert.Equal(t, p.Protocols, direct.Protocols, "native []API roundtrip")

	// JSON simulation — mirrors actual DB path where gou model deserializes
	// JSON columns as []interface{} / map[string]interface{}
	data, err := json.Marshal(row)
	assert.NoError(t, err)
	var dbRow map[string]interface{}
	assert.NoError(t, json.Unmarshal(data, &dbRow))

	restored := rowToProvider(maps.MapStr(dbRow))
	assert.Equal(t, p.ID, restored.ID)
	assert.Equal(t, p.Name, restored.Name)
	assert.Equal(t, p.ProviderID, restored.ProviderID)
	assert.Equal(t, p.API, restored.API)
	assert.Equal(t, p.BaseURL, restored.BaseURL)
	assert.Equal(t, p.APIKey, restored.APIKey)
	assert.Equal(t, p.Enabled, restored.Enabled)
	assert.Equal(t, p.Status, restored.Status)
	assert.Equal(t, p.PresetKey, restored.PresetKey)
	assert.Equal(t, p.Scope, restored.Scope)
	assert.Equal(t, p.Priority, restored.Priority)
	assert.Len(t, restored.Protocols, 2)
	assert.NotNil(t, restored.Headers)
	assert.NotNil(t, restored.Compat)
	assert.NotNil(t, restored.Metadata)
}

func TestProviderRowRoundtrip_MinimalFields(t *testing.T) {
	p := &Provider{
		Name:       "Bare",
		ProviderID: ProviderOpenAI,
		API:        APIOpenAICompletions,
		Scope:      ScopeSystem,
	}

	row := providerToRow(p)
	assert.Nil(t, row["protocols"], "nil slice should not be in row")
	assert.Nil(t, row["headers"])
	assert.Nil(t, row["compat"])
	assert.Nil(t, row["metadata"])
	assert.Nil(t, row["id"], "zero ID should not be in row")

	restored := rowToProvider(maps.MapStr(row))
	assert.Equal(t, 0, restored.ID)
	assert.Equal(t, "Bare", restored.Name)
	assert.Nil(t, restored.Protocols)
	assert.Nil(t, restored.Headers)
}

func TestModelRowRoundtrip(t *testing.T) {
	high := "high"
	m := &Model{
		ID:              7,
		ProviderRef:     1,
		ModelID:         "gpt-4o",
		Name:            "GPT-4o",
		Type:            ModelTypeChat,
		ModelAPI:        "openai-completions",
		Family:          "gpt-4o",
		Enabled:         true,
		ContextWindow:   128000,
		MaxOutputTokens: 16384,
		SupportsImages:  true,
		SupportsToolUse: true,
		Reasoning:       false,
		Streaming:       true,
		Input:           []string{"text", "image"},
		Output:          []string{"text"},
		Capabilities:    []string{"vision", "tool_calls"},
		MaxThinking:     ThinkingHigh,
		ThinkingLevelMap: ThinkingLevelMap{
			ThinkingHigh: &high,
		},
		SamplingParams: map[string]any{"temperature": 0.7},
		SamplingParamsByThinkingLevel: SamplingParamsByThinkingLevel{
			ThinkingHigh: map[string]any{"temperature": 0.3},
		},
		InputCostPer1M:      2.5,
		OutputCostPer1M:     10.0,
		CacheReadCostPer1M:  0.25,
		CacheWriteCostPer1M: 3.0,
		CostTiers: []CostTier{{
			InputTokensAbove: 200000,
			InputCostPer1M:   1.5,
			OutputCostPer1M:  6.0,
		}},
		InputLimits: &ModelInputLimits{
			MaxRequestBytes: 4718592,
			Images:          &ModelImageInputLimits{MaxPerMessage: 20},
		},
		PromptCache: ModelPromptCache{"live": 300},
		Compat:      JSONMap{"thinkingFormat": "openai"},
		Lab:         "experimental",
		Priority:    0,
		Metadata:    JSONMap{"source": "preset"},
	}

	row := modelToRow(m)
	assert.Equal(t, 7, row["id"])
	assert.Equal(t, "gpt-4o", row["model_id"])
	assert.Equal(t, true, row["supports_images"])
	assert.Equal(t, true, row["streaming"])
	assert.Equal(t, "openai-completions", row["model_api"])
	assert.NotNil(t, row["input"])
	assert.NotNil(t, row["output"])
	assert.NotNil(t, row["capabilities"])
	assert.NotNil(t, row["thinking_level_map"])
	assert.NotNil(t, row["sampling_params"])
	assert.NotNil(t, row["sampling_params_by_level"])
	assert.NotNil(t, row["cost_tiers"])
	assert.NotNil(t, row["input_limits"])
	assert.NotNil(t, row["prompt_cache"])
	assert.NotNil(t, row["compat"])
	assert.NotNil(t, row["metadata"])
	assert.Equal(t, "high", row["max_thinking"])
	assert.Equal(t, "experimental", row["lab"])

	// Simulate DB roundtrip: JSON marshal/unmarshal converts Go structs
	// ([]CostTier, *ModelInputLimits, etc.) to the []interface{} / map[string]interface{}
	// forms that gou model returns from the database.
	data, err := json.Marshal(row)
	assert.NoError(t, err)
	var dbRow map[string]interface{}
	assert.NoError(t, json.Unmarshal(data, &dbRow))

	restored := rowToModel(maps.MapStr(dbRow))
	assert.Equal(t, m.ID, restored.ID)
	assert.Equal(t, m.ModelID, restored.ModelID)
	assert.Equal(t, m.Name, restored.Name)
	assert.Equal(t, m.Type, restored.Type)
	assert.Equal(t, m.ModelAPI, restored.ModelAPI)
	assert.Equal(t, m.ContextWindow, restored.ContextWindow)
	assert.Equal(t, m.MaxOutputTokens, restored.MaxOutputTokens)
	assert.Equal(t, m.SupportsImages, restored.SupportsImages)
	assert.Equal(t, m.Streaming, restored.Streaming)
	assert.Equal(t, m.MaxThinking, restored.MaxThinking)
	assert.InDelta(t, m.InputCostPer1M, restored.InputCostPer1M, 0.001)
	assert.InDelta(t, m.OutputCostPer1M, restored.OutputCostPer1M, 0.001)
	assert.InDelta(t, m.CacheReadCostPer1M, restored.CacheReadCostPer1M, 0.001)
	assert.InDelta(t, m.CacheWriteCostPer1M, restored.CacheWriteCostPer1M, 0.001)
	assert.Equal(t, m.Lab, restored.Lab)
	assert.Equal(t, m.Input, restored.Input)
	assert.Equal(t, m.Output, restored.Output)
	assert.Equal(t, m.Capabilities, restored.Capabilities)
	assert.NotNil(t, restored.ThinkingLevelMap)
	assert.NotNil(t, restored.SamplingParams)
	assert.NotNil(t, restored.SamplingParamsByThinkingLevel)
	assert.NotNil(t, restored.CostTiers)
	assert.Len(t, restored.CostTiers, 1)
	assert.Equal(t, 200000, restored.CostTiers[0].InputTokensAbove)
	assert.NotNil(t, restored.InputLimits)
	assert.Equal(t, 4718592, restored.InputLimits.MaxRequestBytes)
	assert.NotNil(t, restored.PromptCache)
	assert.Equal(t, 300, restored.PromptCache["live"])
	assert.NotNil(t, restored.Compat)
	assert.NotNil(t, restored.Metadata)
}

func TestModelRowRoundtrip_MinimalFields(t *testing.T) {
	m := &Model{
		ProviderRef: 1,
		ModelID:     "simple",
		Name:        "Simple",
		Type:        ModelTypeChat,
	}

	row := modelToRow(m)
	assert.Nil(t, row["id"])
	assert.Nil(t, row["input"])
	assert.Nil(t, row["thinking_level_map"])
	assert.Nil(t, row["compat"])
	assert.Nil(t, row["cost_tiers"])

	restored := rowToModel(maps.MapStr(row))
	assert.Equal(t, "simple", restored.ModelID)
	assert.Equal(t, ModelTypeChat, restored.Type)
	assert.Nil(t, restored.Input)
	assert.Nil(t, restored.ThinkingLevelMap)
}

func TestRoleBindingRowRoundtrip(t *testing.T) {
	rb := &RoleBinding{
		ID:          3,
		Role:        RoleDefault,
		ProviderRef: 1,
		ModelID:     "gpt-4o",
		Scope:       ScopeUser,
		TeamID:      "",
		UserID:      "user-123",
	}

	row := roleBindingToRow(rb)
	assert.Equal(t, 3, row["id"])
	assert.Equal(t, "default", row["role"])
	assert.Equal(t, "user", row["scope"])
	assert.Equal(t, "user-123", row["user_id"])

	restored := rowToRoleBinding(maps.MapStr(row))
	assert.Equal(t, rb.ID, restored.ID)
	assert.Equal(t, rb.Role, restored.Role)
	assert.Equal(t, rb.ProviderRef, restored.ProviderRef)
	assert.Equal(t, rb.ModelID, restored.ModelID)
	assert.Equal(t, rb.Scope, restored.Scope)
	assert.Equal(t, rb.UserID, restored.UserID)
	assert.Empty(t, restored.TeamID)
}

// ---------------------------------------------------------------------------
// buildScopeWheres
// ---------------------------------------------------------------------------

func TestBuildScopeWheres(t *testing.T) {
	t.Run("all empty returns no wheres", func(t *testing.T) {
		wheres := buildScopeWheres("", "", "")
		assert.Empty(t, wheres)
	})

	t.Run("scope only", func(t *testing.T) {
		wheres := buildScopeWheres(ScopeSystem, "", "")
		assert.Len(t, wheres, 1)
		assert.Equal(t, "scope", wheres[0].Column)
		assert.Equal(t, "system", wheres[0].Value)
	})

	t.Run("scope + teamID", func(t *testing.T) {
		wheres := buildScopeWheres(ScopeTeam, "team-1", "")
		assert.Len(t, wheres, 2)
		assert.Equal(t, "scope", wheres[0].Column)
		assert.Equal(t, "team_id", wheres[1].Column)
		assert.Equal(t, "team-1", wheres[1].Value)
	})

	t.Run("scope + userID", func(t *testing.T) {
		wheres := buildScopeWheres(ScopeUser, "", "user-1")
		assert.Len(t, wheres, 2)
		assert.Equal(t, "scope", wheres[0].Column)
		assert.Equal(t, "user_id", wheres[1].Column)
	})

	t.Run("all populated", func(t *testing.T) {
		wheres := buildScopeWheres(ScopeUser, "team-1", "user-1")
		assert.Len(t, wheres, 3)
	})
}
