package llm

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// API constants — values must match PiG ai.API exactly
// ---------------------------------------------------------------------------

func TestAPIConstants(t *testing.T) {
	cases := []struct {
		got  API
		want string
	}{
		{APIOpenAICompletions, "openai-completions"},
		{APIOpenAIResponses, "openai-responses"},
		{APIAzureResponses, "azure-openai-responses"},
		{APICodexResponses, "openai-codex-responses"},
		{APIAnthropicMessages, "anthropic-messages"},
		{APIGoogleGenerativeAI, "google-generative-ai"},
		{APIGoogleVertex, "google-vertex"},
		{APIBedrockConverse, "bedrock-converse-stream"},
		{APIMistral, "mistral-conversations"},
		{APIPiMessages, "pi-messages"},
	}
	assert.Len(t, cases, 10, "expected 10 API constants")
	for _, tc := range cases {
		assert.Equal(t, tc.want, string(tc.got))
	}
}

func TestImageAPIConstants(t *testing.T) {
	assert.Equal(t, "openrouter-images", string(ImageAPIOpenRouter))
}

func TestDecisionAPIConstants(t *testing.T) {
	assert.Equal(t, "typesafe-system-one", string(DecisionAPITypesafeSystemOne))
	assert.Equal(t, "cloudflare-workers-ai-system-one", string(DecisionAPICFWorkersAISystemOne))
	assert.Equal(t, "llama-cpp-classify", string(DecisionAPILlamaCppClassify))
}

// ---------------------------------------------------------------------------
// ModelType
// ---------------------------------------------------------------------------

func TestModelTypeConstants(t *testing.T) {
	assert.Equal(t, "chat", string(ModelTypeChat))
	assert.Equal(t, "image", string(ModelTypeImage))
	assert.Equal(t, "decision", string(ModelTypeDecision))
}

// ---------------------------------------------------------------------------
// ProviderID — values must match PiG ProviderMetadata.ProviderID
// ---------------------------------------------------------------------------

func TestProviderIDConstants(t *testing.T) {
	ids := []ProviderID{
		ProviderOpenAI, ProviderAnthropic, ProviderGoogle,
		ProviderAzure, ProviderBedrock, ProviderDeepSeek,
		ProviderMoonshot, ProviderMistral, ProviderOpenRouter,
		ProviderTogether, ProviderGroq,
	}
	assert.Len(t, ids, 11, "expected 11 ProviderID constants")

	assert.Equal(t, "openai", string(ProviderOpenAI))
	assert.Equal(t, "anthropic", string(ProviderAnthropic))
	assert.Equal(t, "deepseek", string(ProviderDeepSeek))
	assert.Equal(t, "moonshotai", string(ProviderMoonshot))
	assert.Equal(t, "openrouter", string(ProviderOpenRouter))
}

// ---------------------------------------------------------------------------
// ThinkingLevel — values must match PiG ai.ThinkingLevel
// ---------------------------------------------------------------------------

func TestThinkingLevelConstants(t *testing.T) {
	assert.Equal(t, "off", string(ThinkingOff))
	assert.Equal(t, ThinkingOff, ThinkingNone, "ThinkingNone is an alias for ThinkingOff")
	assert.Equal(t, "minimal", string(ThinkingMinimal))
	assert.Equal(t, "low", string(ThinkingLow))
	assert.Equal(t, "medium", string(ThinkingMedium))
	assert.Equal(t, "high", string(ThinkingHigh))
	assert.Equal(t, "xhigh", string(ThinkingXHigh))
	assert.Equal(t, "max", string(ThinkingMax))
}

// ---------------------------------------------------------------------------
// Scope
// ---------------------------------------------------------------------------

func TestScopeConstants(t *testing.T) {
	assert.Equal(t, "system", string(ScopeSystem))
	assert.Equal(t, "team", string(ScopeTeam))
	assert.Equal(t, "user", string(ScopeUser))
}

// ---------------------------------------------------------------------------
// Role constants
// ---------------------------------------------------------------------------

func TestRoleConstants(t *testing.T) {
	roles := []string{
		RoleDefault, RoleHeavy, RoleLight,
		RoleVision, RoleAudio, RoleEmbedding, RoleDecision,
	}
	assert.Len(t, roles, 7, "expected 7 role constants")
	assert.Equal(t, "default", RoleDefault)
	assert.Equal(t, "heavy", RoleHeavy)
	assert.Equal(t, "light", RoleLight)
	assert.Equal(t, "decision", RoleDecision)
}

// ---------------------------------------------------------------------------
// Provider.HeadersMap
// ---------------------------------------------------------------------------

func TestProviderHeadersMap(t *testing.T) {
	t.Run("nil headers returns nil", func(t *testing.T) {
		p := &Provider{}
		assert.Nil(t, p.HeadersMap())
	})

	t.Run("empty headers returns nil", func(t *testing.T) {
		p := &Provider{Headers: JSONMap{}}
		assert.Nil(t, p.HeadersMap())
	})

	t.Run("string values are preserved", func(t *testing.T) {
		p := &Provider{Headers: JSONMap{
			"Authorization": "Bearer sk-xxx",
			"X-Custom":      "value",
		}}
		got := p.HeadersMap()
		assert.Equal(t, map[string]string{
			"Authorization": "Bearer sk-xxx",
			"X-Custom":      "value",
		}, got)
	})

	t.Run("non-string values are skipped", func(t *testing.T) {
		p := &Provider{Headers: JSONMap{
			"Good":    "kept",
			"BadInt":  42,
			"BadBool": true,
		}}
		got := p.HeadersMap()
		assert.Equal(t, map[string]string{"Good": "kept"}, got)
	})
}

// ---------------------------------------------------------------------------
// ThinkingLevelMap operations
// ---------------------------------------------------------------------------

func TestThinkingLevelMap(t *testing.T) {
	medium := "medium"
	high := "high"
	m := ThinkingLevelMap{
		ThinkingMedium: &medium,
		ThinkingHigh:   &high,
		ThinkingMax:    nil, // nil = provider-specific disable
	}

	assert.Equal(t, "medium", *m[ThinkingMedium])
	assert.Equal(t, "high", *m[ThinkingHigh])
	assert.Nil(t, m[ThinkingMax], "nil pointer means provider-specific disable")
	_, exists := m[ThinkingLow]
	assert.False(t, exists, "unmapped level should not exist")
}

// ---------------------------------------------------------------------------
// CostTier JSON tag alignment with PiG ai.CostTier
// ---------------------------------------------------------------------------

func TestCostTierJSONTags(t *testing.T) {
	tier := CostTier{
		InputTokensAbove:    200000,
		InputCostPer1M:      1.5,
		OutputCostPer1M:     6.0,
		CacheReadCostPer1M:  0.15,
		CacheWriteCostPer1M: 1.875,
	}
	data, err := json.Marshal(tier)
	require.NoError(t, err)

	var raw map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &raw))

	assert.Contains(t, raw, "inputTokensAbove")
	assert.Contains(t, raw, "input")
	assert.Contains(t, raw, "output")
	assert.Contains(t, raw, "cacheRead")
	assert.Contains(t, raw, "cacheWrite")
	assert.InDelta(t, 200000.0, raw["inputTokensAbove"], 0.01)
	assert.InDelta(t, 1.5, raw["input"], 0.001)
}

func TestCostTierJSONRoundtrip(t *testing.T) {
	original := CostTier{
		InputTokensAbove:    100000,
		InputCostPer1M:      2.0,
		OutputCostPer1M:     8.0,
		CacheReadCostPer1M:  0.2,
		CacheWriteCostPer1M: 2.5,
	}
	data, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded CostTier
	require.NoError(t, json.Unmarshal(data, &decoded))
	assert.Equal(t, original, decoded)
}

// ---------------------------------------------------------------------------
// ModelInputLimits JSON tag alignment with PiG ai.ModelInputLimits
// ---------------------------------------------------------------------------

func TestModelInputLimitsJSONTags(t *testing.T) {
	limits := &ModelInputLimits{
		MaxRequestBytes: 4718592,
		Images: &ModelImageInputLimits{
			Resize: &ModelImageResizeOptions{
				MaxWidth:    2000,
				MaxHeight:   2000,
				MaxBytes:    4718592,
				JPEGQuality: 80,
			},
			MaxPerMessage: 20,
			MaxPerRequest: 100,
		},
	}

	data, err := json.Marshal(limits)
	require.NoError(t, err)

	var raw map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &raw))

	assert.Contains(t, raw, "maxRequestBytes")
	assert.Contains(t, raw, "images")

	images := raw["images"].(map[string]interface{})
	assert.Contains(t, images, "resize")
	assert.Contains(t, images, "maxPerMessage")
	assert.Contains(t, images, "maxPerRequest")

	resize := images["resize"].(map[string]interface{})
	assert.Contains(t, resize, "maxWidth")
	assert.Contains(t, resize, "maxHeight")
	assert.Contains(t, resize, "maxBytes")
	assert.Contains(t, resize, "jpegQuality")
}

func TestModelInputLimitsJSONRoundtrip(t *testing.T) {
	original := &ModelInputLimits{
		MaxRequestBytes: 4718592,
		Images: &ModelImageInputLimits{
			Resize: &ModelImageResizeOptions{
				MaxWidth:    2000,
				MaxHeight:   2000,
				MaxBytes:    4718592,
				JPEGQuality: 80,
			},
			MaxPerMessage: 20,
		},
	}
	data, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded ModelInputLimits
	require.NoError(t, json.Unmarshal(data, &decoded))
	assert.Equal(t, *original, decoded)
}

func TestModelInputLimitsOmitsZeroFields(t *testing.T) {
	limits := &ModelInputLimits{}
	data, err := json.Marshal(limits)
	require.NoError(t, err)
	assert.Equal(t, `{}`, string(data))
}

// ---------------------------------------------------------------------------
// Provider JSON serialization
// ---------------------------------------------------------------------------

func TestProviderJSONRoundtrip(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	original := Provider{
		ID:         1,
		Name:       "My OpenAI",
		ProviderID: ProviderOpenAI,
		API:        APIOpenAICompletions,
		BaseURL:    "https://api.openai.com/v1",
		Enabled:    true,
		Status:     "connected",
		Scope:      ScopeSystem,
		Priority:   0,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded Provider
	require.NoError(t, json.Unmarshal(data, &decoded))

	assert.Equal(t, original.Name, decoded.Name)
	assert.Equal(t, original.ProviderID, decoded.ProviderID)
	assert.Equal(t, original.API, decoded.API)
	assert.Equal(t, original.Scope, decoded.Scope)
	assert.Equal(t, original.Status, decoded.Status)
}

// ---------------------------------------------------------------------------
// Model JSON serialization
// ---------------------------------------------------------------------------

func TestModelJSONRoundtrip(t *testing.T) {
	medium := "medium"
	high := "high"

	original := Model{
		ID:              1,
		ProviderRef:     1,
		ModelID:         "o3",
		Name:            "o3",
		Type:            ModelTypeChat,
		Enabled:         true,
		ContextWindow:   200000,
		MaxOutputTokens: 100000,
		SupportsImages:  true,
		SupportsToolUse: true,
		Reasoning:       true,
		Streaming:       true,
		Input:           []string{"text", "image"},
		MaxThinking:     ThinkingMax,
		ThinkingLevelMap: ThinkingLevelMap{
			ThinkingMedium: &medium,
			ThinkingHigh:   &high,
			ThinkingMax:    nil,
		},
		InputCostPer1M:  2.0,
		OutputCostPer1M: 8.0,
		Capabilities:    []string{"vision", "tool_calls", "reasoning"},
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded Model
	require.NoError(t, json.Unmarshal(data, &decoded))

	assert.Equal(t, original.ModelID, decoded.ModelID)
	assert.Equal(t, original.Type, decoded.Type)
	assert.Equal(t, original.ContextWindow, decoded.ContextWindow)
	assert.Equal(t, original.SupportsImages, decoded.SupportsImages)
	assert.Equal(t, original.Reasoning, decoded.Reasoning)
	assert.Equal(t, original.Input, decoded.Input)
	assert.Equal(t, original.MaxThinking, decoded.MaxThinking)
	assert.InDelta(t, original.InputCostPer1M, decoded.InputCostPer1M, 0.001)
	assert.Equal(t, original.Capabilities, decoded.Capabilities)

	require.NotNil(t, decoded.ThinkingLevelMap[ThinkingMedium])
	assert.Equal(t, "medium", *decoded.ThinkingLevelMap[ThinkingMedium])
	require.NotNil(t, decoded.ThinkingLevelMap[ThinkingHigh])
	assert.Equal(t, "high", *decoded.ThinkingLevelMap[ThinkingHigh])
	assert.Nil(t, decoded.ThinkingLevelMap[ThinkingMax])
}

// ---------------------------------------------------------------------------
// RoleBinding JSON serialization
// ---------------------------------------------------------------------------

func TestRoleBindingJSONRoundtrip(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	original := RoleBinding{
		ID:          1,
		Role:        RoleHeavy,
		ProviderRef: 1,
		ModelID:     "o3",
		Scope:       ScopeUser,
		UserID:      "user-123",
		UpdatedAt:   now,
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded RoleBinding
	require.NoError(t, json.Unmarshal(data, &decoded))

	assert.Equal(t, original.Role, decoded.Role)
	assert.Equal(t, original.Scope, decoded.Scope)
	assert.Equal(t, original.UserID, decoded.UserID)
	assert.Empty(t, decoded.TeamID, "team_id should be empty for user-scoped binding")
}

// ---------------------------------------------------------------------------
// Model with ImageModel fields (Type=image, Output set)
// ---------------------------------------------------------------------------

func TestModelImageModelFields(t *testing.T) {
	m := Model{
		ModelID:         "dall-e-3",
		Name:            "DALL-E 3",
		Type:            ModelTypeImage,
		ModelAPI:        string(ImageAPIOpenRouter),
		Input:           []string{"text", "image"},
		Output:          []string{"image"},
		ImageGeneration: true,
		InputCostPer1M:  5.0,
	}

	data, err := json.Marshal(m)
	require.NoError(t, err)

	var raw map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &raw))

	assert.Equal(t, "image", raw["type"])
	assert.Equal(t, "openrouter-images", raw["model_api"])
	assert.True(t, raw["image_generation"].(bool))
}

// ---------------------------------------------------------------------------
// Model with DecisionModel fields (Type=decision)
// ---------------------------------------------------------------------------

func TestModelDecisionModelFields(t *testing.T) {
	m := Model{
		ModelID:       "typesafe-classifier-v1",
		Name:          "TypeSafe Classifier",
		Type:          ModelTypeDecision,
		ModelAPI:      string(DecisionAPITypesafeSystemOne),
		Input:         []string{"text"},
		ContextWindow: 32000,
	}

	data, err := json.Marshal(m)
	require.NoError(t, err)

	var raw map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &raw))

	assert.Equal(t, "decision", raw["type"])
	assert.Equal(t, "typesafe-system-one", raw["model_api"])
}

// ---------------------------------------------------------------------------
// .mod.yao validation
// ---------------------------------------------------------------------------

func TestModYaoFilesAreValidJSON(t *testing.T) {
	files := []string{
		"../yao/models/llm/provider.mod.yao",
		"../yao/models/llm/model.mod.yao",
		"../yao/models/llm/role_binding.mod.yao",
	}

	for _, f := range files {
		t.Run(f, func(t *testing.T) {
			data, err := readTestFile(t, f)
			require.NoError(t, err, "file should be readable")

			var parsed map[string]interface{}
			require.NoError(t, json.Unmarshal(data, &parsed), "file should be valid JSON")

			assert.Contains(t, parsed, "name")
			assert.Contains(t, parsed, "table")
			assert.Contains(t, parsed, "columns")
			assert.Contains(t, parsed, "indexes")
			assert.Contains(t, parsed, "option")

			assert.True(t, parsed["builtin"].(bool), "system model must be builtin")
			assert.True(t, parsed["readonly"].(bool), "system model must be readonly")
		})
	}
}

func TestProviderModYaoColumns(t *testing.T) {
	data, err := readTestFile(t, "../yao/models/llm/provider.mod.yao")
	require.NoError(t, err)

	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &parsed))

	table := parsed["table"].(map[string]interface{})
	assert.Equal(t, "llm_provider", table["name"])

	columns := parsed["columns"].([]interface{})
	colNames := extractColumnNames(columns)

	required := []string{
		"id", "name", "provider_id", "api", "protocols", "base_url",
		"api_key", "headers", "compat", "enabled", "status",
		"preset_key", "scope", "team_id", "user_id", "priority", "metadata",
	}
	for _, name := range required {
		assert.Contains(t, colNames, name, "provider model missing column: %s", name)
	}

	apiKeyCol := findColumn(columns, "api_key")
	require.NotNil(t, apiKeyCol, "api_key column must exist")
	assert.Equal(t, "AES", apiKeyCol["crypt"], "api_key must use AES encryption")
}

func TestModelModYaoColumns(t *testing.T) {
	data, err := readTestFile(t, "../yao/models/llm/model.mod.yao")
	require.NoError(t, err)

	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &parsed))

	table := parsed["table"].(map[string]interface{})
	assert.Equal(t, "llm_model", table["name"])

	columns := parsed["columns"].([]interface{})
	colNames := extractColumnNames(columns)

	required := []string{
		"id", "provider_ref", "model_id", "name", "type", "model_api",
		"family", "enabled", "context_window", "max_output_tokens",
		"supports_images", "supports_tool_use", "reasoning",
		"audio", "stt", "streaming", "json_mode", "embedding",
		"image_generation", "image_editing", "ocr", "temperature_adjustable",
		"max_thinking", "input", "output",
		"thinking_level_map", "sampling_params", "sampling_params_by_level",
		"capabilities", "compat",
		"input_cost_per_1m", "output_cost_per_1m",
		"cache_read_cost_per_1m", "cache_write_cost_per_1m",
		"cost_tiers", "input_limits", "prompt_cache",
		"lab", "priority", "metadata",
	}
	for _, name := range required {
		assert.Contains(t, colNames, name, "model mod missing column: %s", name)
	}
}

func TestRoleBindingModYaoColumns(t *testing.T) {
	data, err := readTestFile(t, "../yao/models/llm/role_binding.mod.yao")
	require.NoError(t, err)

	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &parsed))

	table := parsed["table"].(map[string]interface{})
	assert.Equal(t, "llm_role_binding", table["name"])

	columns := parsed["columns"].([]interface{})
	colNames := extractColumnNames(columns)

	required := []string{"id", "role", "provider_ref", "model_id", "scope", "team_id", "user_id"}
	for _, name := range required {
		assert.Contains(t, colNames, name, "role_binding mod missing column: %s", name)
	}
}

func TestModYaoUniqueIndexes(t *testing.T) {
	cases := []struct {
		file      string
		indexName string
		columns   []string
	}{
		{"../yao/models/llm/provider.mod.yao", "uk_providers_scope", []string{"provider_id", "scope", "team_id", "user_id"}},
		{"../yao/models/llm/model.mod.yao", "uk_models_provider_model", []string{"provider_ref", "model_id"}},
		{"../yao/models/llm/role_binding.mod.yao", "uk_role_scope", []string{"role", "scope", "team_id", "user_id"}},
	}

	for _, tc := range cases {
		t.Run(tc.indexName, func(t *testing.T) {
			data, err := readTestFile(t, tc.file)
			require.NoError(t, err)

			var parsed map[string]interface{}
			require.NoError(t, json.Unmarshal(data, &parsed))

			indexes := parsed["indexes"].([]interface{})
			idx := findIndex(indexes, tc.indexName)
			require.NotNil(t, idx, "index %s must exist", tc.indexName)

			assert.Equal(t, "unique", idx["type"])

			idxCols := toStringSlice(idx["columns"].([]interface{}))
			assert.Equal(t, tc.columns, idxCols)
		})
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func readTestFile(t *testing.T, path string) ([]byte, error) {
	t.Helper()
	return os.ReadFile(path)
}

func extractColumnNames(columns []interface{}) []string {
	names := make([]string, 0, len(columns))
	for _, c := range columns {
		col := c.(map[string]interface{})
		names = append(names, col["name"].(string))
	}
	return names
}

func findColumn(columns []interface{}, name string) map[string]interface{} {
	for _, c := range columns {
		col := c.(map[string]interface{})
		if col["name"].(string) == name {
			return col
		}
	}
	return nil
}

func findIndex(indexes []interface{}, name string) map[string]interface{} {
	for _, idx := range indexes {
		m := idx.(map[string]interface{})
		if m["name"].(string) == name {
			return m
		}
	}
	return nil
}

func toStringSlice(arr []interface{}) []string {
	out := make([]string, len(arr))
	for i, v := range arr {
		out[i] = v.(string)
	}
	return out
}
