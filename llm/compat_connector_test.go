package llm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/llmprovider"
)

// ---------------------------------------------------------------------------
// inferProviderID
// ---------------------------------------------------------------------------

func TestInferProviderID(t *testing.T) {
	tests := []struct {
		name   string
		old    llmprovider.Provider
		expect ProviderID
	}{
		// URL-based inference takes priority over connector type
		{"openai url", llmprovider.Provider{Name: "OpenAI", Type: "openai", APIURL: "https://api.openai.com/v1"}, ProviderOpenAI},
		{"anthropic url", llmprovider.Provider{Name: "Anthropic", Type: "anthropic", APIURL: "https://api.anthropic.com"}, ProviderAnthropic},
		{"moapi deepseek url", llmprovider.Provider{Name: "DS", Type: "moapi", APIURL: "https://api.deepseek.com/v1"}, ProviderDeepSeek},
		{"moapi moonshot url", llmprovider.Provider{Name: "Moon", Type: "moapi", APIURL: "https://api.moonshot.cn"}, ProviderMoonshot},
		{"custom groq url", llmprovider.Provider{Name: "Groq", Type: "custom", APIURL: "https://api.groq.com"}, ProviderGroq},
		{"custom together url", llmprovider.Provider{Name: "Together", Type: "custom", APIURL: "https://api.together.xyz"}, ProviderTogether},
		{"custom openrouter url", llmprovider.Provider{Name: "OpenRouter", Type: "custom", APIURL: "https://openrouter.ai/api/v1"}, ProviderOpenRouter},
		{"custom google url", llmprovider.Provider{Name: "Google", Type: "custom", APIURL: "https://generativelanguage.googleapis.com"}, ProviderGoogle},
		{"custom mistral url", llmprovider.Provider{Name: "Mistral", Type: "custom", APIURL: "https://api.mistral.ai"}, ProviderMistral},
		{"custom azure url", llmprovider.Provider{Name: "Azure", Type: "custom", APIURL: "https://myresource.azure.openai.com"}, ProviderAzure},
		{"openai type deepseek url", llmprovider.Provider{Name: "DS", Type: "openai", APIURL: "https://api.deepseek.com/v1"}, ProviderDeepSeek},
		// URL doesn't match any known vendor → fall back to name slug
		{"unknown url name slug", llmprovider.Provider{Name: "Tao Service", Type: "openai", APIURL: "https://api.yaoagents.com"}, ProviderID("tao-service")},
		{"volcengine url name slug", llmprovider.Provider{Name: "DeepSeek R1", Type: "openai", APIURL: "https://ark.cn-beijing.volces.com/api/v3/"}, ProviderID("deepseek-r1")},
		{"no url name slug", llmprovider.Provider{Name: "My Local LLM", Type: "openai"}, ProviderID("my-local-llm")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := inferProviderID(&tt.old)
			assert.Equal(t, tt.expect, got)
		})
	}
}

// ---------------------------------------------------------------------------
// inferAPI
// ---------------------------------------------------------------------------

func TestInferAPI(t *testing.T) {
	assert.Equal(t, APIAnthropicMessages, inferAPI(&llmprovider.Provider{Type: "anthropic"}))
	assert.Equal(t, APIOpenAICompletions, inferAPI(&llmprovider.Provider{Type: "openai"}))
	assert.Equal(t, APIOpenAICompletions, inferAPI(&llmprovider.Provider{Type: "moapi"}))
	assert.Equal(t, APIOpenAICompletions, inferAPI(&llmprovider.Provider{Type: "custom"}))
}

// ---------------------------------------------------------------------------
// translateScope
// ---------------------------------------------------------------------------

func TestTranslateScope(t *testing.T) {
	scope, teamID, userID := translateScope(llmprovider.ProviderOwner{Type: "system"})
	assert.Equal(t, ScopeSystem, scope)
	assert.Empty(t, teamID)
	assert.Empty(t, userID)

	scope, teamID, userID = translateScope(llmprovider.ProviderOwner{Type: "team", TeamID: "t1"})
	assert.Equal(t, ScopeTeam, scope)
	assert.Equal(t, "t1", teamID)
	assert.Empty(t, userID)

	scope, teamID, userID = translateScope(llmprovider.ProviderOwner{Type: "user", UserID: "u1"})
	assert.Equal(t, ScopeUser, scope)
	assert.Empty(t, teamID)
	assert.Equal(t, "u1", userID)

	scope, _, _ = translateScope(llmprovider.ProviderOwner{Type: ""})
	assert.Equal(t, ScopeSystem, scope)
}

// ---------------------------------------------------------------------------
// Translate (end-to-end: old Provider → new Provider + []Model)
// ---------------------------------------------------------------------------

func TestTranslate(t *testing.T) {
	old := llmprovider.Provider{
		Key:    "openai-1",
		Name:   "My OpenAI",
		Type:   "openai",
		APIURL: "https://api.openai.com/v1",
		APIKey: "sk-test",
		Models: []llmprovider.ModelInfo{
			{
				ID:              "gpt-4o",
				Name:            "GPT-4o",
				Capabilities:    []string{"vision", "tool_calls", "streaming", "json"},
				Enabled:         true,
				MaxInputTokens:  128000,
				MaxOutputTokens: 16384,
				Metadata: map[string]interface{}{
					"reasoning_efforts": []interface{}{"none", "low", "medium", "high"},
					"model_family":      "openai-gpt-4o",
					"model_name":        "GPT-4o",
				},
			},
			{
				ID:           "gpt-5.4-reasoning-low",
				Model:        "gpt-5.4",
				Name:         "GPT-5.4 Reasoning (Low)",
				Capabilities: []string{"streaming", "reasoning"},
				Enabled:      false,
				Metadata: map[string]interface{}{
					"model_family": "openai-gpt-5.4",
				},
			},
		},
		Enabled: true,
		Status:  "configured",
		Owner:   llmprovider.ProviderOwner{Type: "system"},
	}

	p, models := Translate(&old)

	assert.Equal(t, "My OpenAI", p.Name)
	assert.Equal(t, ProviderOpenAI, p.ProviderID)
	assert.Equal(t, APIOpenAICompletions, p.API)
	assert.Equal(t, "https://api.openai.com/v1", p.BaseURL)
	assert.Equal(t, "sk-test", p.APIKey)
	assert.True(t, p.Enabled)
	assert.Equal(t, "configured", p.Status)
	assert.Equal(t, ScopeSystem, p.Scope)

	require.Len(t, models, 2)

	m0 := models[0]
	assert.Equal(t, "gpt-4o", m0.ModelID)
	assert.Empty(t, m0.ModelAPI, "base model has no Model override")
	assert.Equal(t, "GPT-4o", m0.Name)
	assert.Equal(t, "openai-gpt-4o", m0.Family)
	assert.True(t, m0.Enabled)
	assert.Equal(t, 128000, m0.ContextWindow)
	assert.Equal(t, 16384, m0.MaxOutputTokens)
	assert.True(t, m0.SupportsImages)
	assert.True(t, m0.SupportsToolUse)
	assert.True(t, m0.Streaming)
	assert.True(t, m0.JSON)
	assert.False(t, m0.Reasoning)
	assert.Equal(t, []string{"text", "image"}, m0.Input)
	require.NotNil(t, m0.ThinkingLevelMap)
	assert.Contains(t, m0.ThinkingLevelMap, ThinkingOff)
	assert.Contains(t, m0.ThinkingLevelMap, ThinkingHigh)
	// Metadata should carry over model_name but not extracted keys
	require.NotNil(t, m0.Metadata)
	assert.Equal(t, "GPT-4o", m0.Metadata["model_name"])
	_, hasFamily := m0.Metadata["model_family"]
	assert.False(t, hasFamily, "model_family should not be in Metadata (extracted to Family)")

	m1 := models[1]
	assert.Equal(t, "gpt-5.4-reasoning-low", m1.ModelID)
	assert.Equal(t, "gpt-5.4", m1.ModelAPI, "variant should carry wire model name")
	assert.Equal(t, "openai-gpt-5.4", m1.Family)
	assert.False(t, m1.Enabled)
	assert.Equal(t, []string{"text"}, m1.Input)
}

func TestTranslateEmptyStatus(t *testing.T) {
	old := llmprovider.Provider{Name: "X", Type: "openai"}
	p, _ := Translate(&old)
	assert.Equal(t, "unconfigured", p.Status)
}

func TestTranslateDeepseekThinking(t *testing.T) {
	old := llmprovider.Provider{
		Name:   "DeepSeek",
		Type:   "moapi",
		APIURL: "https://api.deepseek.com",
		Models: []llmprovider.ModelInfo{
			{
				ID:   "deepseek-r1",
				Name: "DeepSeek R1",
				Options: map[string]interface{}{
					"thinking": map[string]interface{}{
						"type":          "enabled",
						"budget_tokens": float64(8192),
					},
				},
				Capabilities: []string{"reasoning"},
			},
		},
	}

	p, models := Translate(&old)
	assert.Equal(t, ProviderDeepSeek, p.ProviderID)

	require.Len(t, models, 1)
	m := models[0]
	assert.True(t, m.Reasoning)
	require.NotNil(t, m.ThinkingLevelMap)
	assert.Contains(t, m.ThinkingLevelMap, ThinkingHigh)

	require.NotNil(t, m.Compat)
	assert.Equal(t, "deepseek", m.Compat["thinkingFormat"])
	assert.Equal(t, 8192, m.Compat["budgetTokens"])
}

func TestTranslateAnthropicThinkingNoThinkingFormat(t *testing.T) {
	// Simulates deepseek preset (type: anthropic) with thinking models.
	// thinkingFormat must NOT be set for anthropic-messages protocol.
	old := llmprovider.Provider{
		Name:   "DeepSeek Anthropic",
		Type:   "anthropic",
		APIURL: "https://api.deepseek.com/anthropic",
		Models: []llmprovider.ModelInfo{
			{
				ID:    "deepseek-flash-thinking-low",
				Model: "deepseek-flash",
				Name:  "DeepSeek V4.1 Flash Thinking Low",
				Options: map[string]interface{}{
					"thinking": map[string]interface{}{
						"type": "enabled",
					},
					"reasoning_effort": "low",
				},
				Metadata: map[string]interface{}{
					"model_family":      "deepseek-v4.1-flash",
					"reasoning_efforts": []interface{}{"none", "low", "high", "max"},
				},
				Capabilities: []string{"vision", "tool_calls", "streaming", "json", "reasoning"},
				Enabled:      true,
			},
		},
	}

	p, models := Translate(&old)
	assert.Equal(t, APIAnthropicMessages, p.API)

	require.Len(t, models, 1)
	m := models[0]
	assert.Equal(t, "deepseek-flash-thinking-low", m.ModelID)
	assert.Equal(t, "deepseek-flash", m.ModelAPI)
	assert.Equal(t, "deepseek-v4.1-flash", m.Family)
	assert.True(t, m.Reasoning)

	// thinkingFormat must be absent for anthropic-messages
	if m.Compat != nil {
		_, hasTF := m.Compat["thinkingFormat"]
		assert.False(t, hasTF, "anthropic-messages must not have thinkingFormat")
		_, hasSDR := m.Compat["supportsDeveloperRole"]
		assert.False(t, hasSDR, "anthropic-messages must not set supportsDeveloperRole=false")
	}
}

func TestTranslateAnthropicNoDeveloperRole(t *testing.T) {
	old := llmprovider.Provider{
		Name: "Anthropic",
		Type: "anthropic",
		Models: []llmprovider.ModelInfo{
			{ID: "claude-3-5-sonnet", Name: "Claude 3.5 Sonnet"},
		},
	}

	_, models := Translate(&old)
	require.Len(t, models, 1)

	compat := models[0].Compat
	// Anthropic providers should NOT set supportsDeveloperRole=false
	if compat != nil {
		_, hasSupportsDev := compat["supportsDeveloperRole"]
		assert.False(t, hasSupportsDev, "anthropic should not set supportsDeveloperRole")
	}
}

func TestTranslateNonAnthropicNoDeveloperRole(t *testing.T) {
	old := llmprovider.Provider{
		Name: "OpenAI",
		Type: "openai",
		Models: []llmprovider.ModelInfo{
			{ID: "gpt-4o", Name: "GPT-4o"},
		},
	}

	_, models := Translate(&old)
	require.Len(t, models, 1)

	require.NotNil(t, models[0].Compat)
	assert.Equal(t, false, models[0].Compat["supportsDeveloperRole"])
}

// ---------------------------------------------------------------------------
// translateThinkingMap
// ---------------------------------------------------------------------------

func TestTranslateThinkingMap(t *testing.T) {
	// Priority 1: metadata reasoning_efforts
	mi := llmprovider.ModelInfo{
		Metadata: map[string]interface{}{
			"reasoning_efforts": []interface{}{"low", "medium", "high"},
		},
		Options: map[string]interface{}{
			"reasoning_effort": "medium",
		},
	}
	tlm := translateThinkingMap(mi)
	require.NotNil(t, tlm)
	assert.Contains(t, tlm, ThinkingLow)
	assert.Contains(t, tlm, ThinkingMedium)
	assert.Contains(t, tlm, ThinkingHigh)

	// Priority 2: Options["reasoning_effort"]
	mi2 := llmprovider.ModelInfo{
		Options: map[string]interface{}{
			"reasoning_effort": "high",
		},
	}
	tlm2 := translateThinkingMap(mi2)
	require.NotNil(t, tlm2)
	require.Len(t, tlm2, 1)
	assert.Contains(t, tlm2, ThinkingHigh)

	// Priority 3: Options["thinking"]
	mi3 := llmprovider.ModelInfo{
		Options: map[string]interface{}{
			"thinking": map[string]interface{}{
				"type": "enabled",
			},
		},
	}
	tlm3 := translateThinkingMap(mi3)
	require.NotNil(t, tlm3)
	assert.Contains(t, tlm3, ThinkingHigh)

	// Options["thinking"]["type"]="disabled" → ThinkingOff
	mi4 := llmprovider.ModelInfo{
		Options: map[string]interface{}{
			"thinking": map[string]interface{}{
				"type": "disabled",
			},
		},
	}
	tlm4 := translateThinkingMap(mi4)
	require.NotNil(t, tlm4)
	assert.Contains(t, tlm4, ThinkingOff)

	// None of the above → nil
	mi5 := llmprovider.ModelInfo{}
	assert.Nil(t, translateThinkingMap(mi5))
}

// ---------------------------------------------------------------------------
// parseOldEffortsList
// ---------------------------------------------------------------------------

func TestParseOldEffortsList(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
		keys  []ThinkingLevel
	}{
		{
			"[]interface{}",
			[]interface{}{"none", "low", "medium", "high"},
			[]ThinkingLevel{ThinkingOff, ThinkingLow, ThinkingMedium, ThinkingHigh},
		},
		{
			"[]string",
			[]string{"low", "high"},
			[]ThinkingLevel{ThinkingLow, ThinkingHigh},
		},
		{
			"thinking keyword",
			[]interface{}{"thinking"},
			[]ThinkingLevel{ThinkingHigh},
		},
		{
			"off keyword",
			[]interface{}{"off"},
			[]ThinkingLevel{ThinkingOff},
		},
		{
			"xhigh and max",
			[]interface{}{"xhigh", "max"},
			[]ThinkingLevel{ThinkingXHigh, ThinkingMax},
		},
		{"nil input", nil, nil},
		{"int input", 42, nil},
		{"empty list", []interface{}{}, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseOldEffortsList(tt.input)
			if tt.keys == nil {
				assert.Nil(t, got)
				return
			}
			require.NotNil(t, got)
			for _, k := range tt.keys {
				assert.Contains(t, got, k)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// singleEffortToMap
// ---------------------------------------------------------------------------

func TestSingleEffortToMap(t *testing.T) {
	tests := []struct {
		input  string
		expect ThinkingLevel
	}{
		{"none", ThinkingOff},
		{"off", ThinkingOff},
		{"low", ThinkingLow},
		{"medium", ThinkingMedium},
		{"high", ThinkingHigh},
		{"xhigh", ThinkingXHigh},
		{"max", ThinkingMax},
		{"thinking", ThinkingHigh},
		{"unknown-value", ThinkingHigh},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := singleEffortToMap(tt.input)
			require.Len(t, got, 1)
			assert.Contains(t, got, tt.expect)
		})
	}
}

// ---------------------------------------------------------------------------
// extractModelCompat
// ---------------------------------------------------------------------------

func TestExtractModelCompat(t *testing.T) {
	// With metadata thinking_format (openai-completions)
	mi := llmprovider.ModelInfo{
		Metadata: map[string]interface{}{
			"thinking_format": "deepseek",
		},
	}
	compat := extractModelCompat(mi, APIOpenAICompletions)
	require.NotNil(t, compat)
	assert.Equal(t, "deepseek", compat["thinkingFormat"])
	assert.Equal(t, false, compat["supportsDeveloperRole"]) // non-anthropic

	// With Options["thinking"] → inferred deepseek format (openai-completions)
	mi2 := llmprovider.ModelInfo{
		Options: map[string]interface{}{
			"thinking": map[string]interface{}{
				"type":          "enabled",
				"budget_tokens": float64(4096),
			},
		},
	}
	compat2 := extractModelCompat(mi2, APIOpenAICompletions)
	require.NotNil(t, compat2)
	assert.Equal(t, "deepseek", compat2["thinkingFormat"])
	assert.Equal(t, 4096, compat2["budgetTokens"])

	// Anthropic: no supportsDeveloperRole, no thinkingFormat
	mi3 := llmprovider.ModelInfo{}
	compat3 := extractModelCompat(mi3, APIAnthropicMessages)
	assert.Nil(t, compat3, "anthropic with no extras should yield nil compat")

	// Budget tokens as int (openai-completions)
	mi4 := llmprovider.ModelInfo{
		Options: map[string]interface{}{
			"thinking": map[string]interface{}{
				"type":          "enabled",
				"budget_tokens": 2048,
			},
		},
	}
	compat4 := extractModelCompat(mi4, APIOpenAICompletions)
	assert.Equal(t, 2048, compat4["budgetTokens"])

	// Anthropic with Options["thinking"] — must NOT produce thinkingFormat
	mi5 := llmprovider.ModelInfo{
		Options: map[string]interface{}{
			"thinking": map[string]interface{}{
				"type":          "enabled",
				"budget_tokens": float64(8192),
			},
		},
	}
	compat5 := extractModelCompat(mi5, APIAnthropicMessages)
	require.NotNil(t, compat5, "anthropic with budget_tokens should produce compat")
	assert.Equal(t, 8192, compat5["budgetTokens"])
	_, hasTF := compat5["thinkingFormat"]
	assert.False(t, hasTF, "anthropic must not have thinkingFormat")
	_, hasSDR := compat5["supportsDeveloperRole"]
	assert.False(t, hasSDR, "anthropic must not set supportsDeveloperRole=false")
}

// ---------------------------------------------------------------------------
// parseOldRoleTarget
// ---------------------------------------------------------------------------

func TestParseOldRoleTarget(t *testing.T) {
	rt := parseOldRoleTarget(map[string]interface{}{
		"provider": "openai-1",
		"model":    "gpt-4o",
	})
	assert.Equal(t, "openai-1", rt.Provider)
	assert.Equal(t, "gpt-4o", rt.Model)

	rt2 := parseOldRoleTarget("not a map")
	assert.Empty(t, rt2.Provider)
	assert.Empty(t, rt2.Model)

	rt3 := parseOldRoleTarget(nil)
	assert.Empty(t, rt3.Provider)
}

// ---------------------------------------------------------------------------
// resolveOldRoleRef
// ---------------------------------------------------------------------------

func TestResolveOldRoleRef(t *testing.T) {
	keyToRef := map[string]int{
		"openai-1": 10,
		"deepseek": 20,
	}

	// Direct key match
	ref, modelID := resolveOldRoleRef(oldRoleTarget{Provider: "openai-1", Model: "gpt-4o"}, keyToRef)
	assert.Equal(t, 10, ref)
	assert.Equal(t, "gpt-4o", modelID)

	// Key with model suffix: "deepseek:deepseek-r1"
	ref, modelID = resolveOldRoleRef(oldRoleTarget{Provider: "deepseek:deepseek-r1"}, keyToRef)
	assert.Equal(t, 20, ref)
	assert.Equal(t, "deepseek-r1", modelID)

	// No match
	ref, modelID = resolveOldRoleRef(oldRoleTarget{Provider: "unknown"}, keyToRef)
	assert.Equal(t, 0, ref)
	assert.Empty(t, modelID)

	// Direct match has model in target
	ref, modelID = resolveOldRoleRef(oldRoleTarget{Provider: "openai-1", Model: "o1"}, keyToRef)
	assert.Equal(t, 10, ref)
	assert.Equal(t, "o1", modelID)

	// Direct match with no model, no suffix → empty model
	ref, modelID = resolveOldRoleRef(oldRoleTarget{Provider: "openai-1"}, keyToRef)
	assert.Equal(t, 10, ref)
	assert.Empty(t, modelID)
}

// ---------------------------------------------------------------------------
// buildCapSet
// ---------------------------------------------------------------------------

func TestBuildCapSet(t *testing.T) {
	caps := buildCapSet([]string{"vision", "tool_calls", "streaming"})
	assert.True(t, caps["vision"])
	assert.True(t, caps["tool_calls"])
	assert.True(t, caps["streaming"])
	assert.False(t, caps["audio"])

	empty := buildCapSet(nil)
	assert.Empty(t, empty)
}

// ---------------------------------------------------------------------------
// Translate edge cases
// ---------------------------------------------------------------------------

func TestTranslateImageEditingCapability(t *testing.T) {
	old := llmprovider.Provider{
		Type: "openai",
		Models: []llmprovider.ModelInfo{
			{
				ID:           "dall-e-3",
				Name:         "DALL-E 3",
				Capabilities: []string{"image_generation", "image_editing"},
			},
		},
	}

	_, models := Translate(&old)
	require.Len(t, models, 1)
	assert.Equal(t, "multipart", models[0].ImageEditing)
	assert.True(t, models[0].ImageGeneration)
}

func TestTranslateTeamScope(t *testing.T) {
	old := llmprovider.Provider{
		Type:  "openai",
		Name:  "Team AI",
		Owner: llmprovider.ProviderOwner{Type: "team", TeamID: "team-42"},
	}
	p, _ := Translate(&old)
	assert.Equal(t, ScopeTeam, p.Scope)
	assert.Equal(t, "team-42", p.TeamID)
}

// ---------------------------------------------------------------------------
// inferModelType
// ---------------------------------------------------------------------------

func TestInferModelType(t *testing.T) {
	tests := []struct {
		name   string
		caps   []string
		expect ModelType
	}{
		{"decision", []string{"decision"}, ModelTypeDecision},
		{"image_generation", []string{"image_generation", "image_editing"}, ModelTypeImage},
		{"chat default", []string{"vision", "tool_calls", "streaming", "json"}, ModelTypeChat},
		{"embedding", []string{"embedding"}, ModelTypeChat},
		{"audio", []string{"audio"}, ModelTypeChat},
		{"stt", []string{"stt"}, ModelTypeChat},
		{"decision + image prefers decision", []string{"decision", "image_generation"}, ModelTypeDecision},
		{"empty caps", nil, ModelTypeChat},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := inferModelType(buildCapSet(tt.caps))
			assert.Equal(t, tt.expect, got)
		})
	}
}

// ---------------------------------------------------------------------------
// wireModelName
// ---------------------------------------------------------------------------

func TestWireModelName(t *testing.T) {
	// Variant entry with explicit Model field
	mi := llmprovider.ModelInfo{ID: "deepseek-flash-thinking-low", Model: "deepseek-flash"}
	assert.Equal(t, "deepseek-flash", wireModelName(mi))

	// Base entry with no Model field — wire name equals ID
	mi2 := llmprovider.ModelInfo{ID: "deepseek-flash"}
	assert.Equal(t, "deepseek-flash", wireModelName(mi2))
}

// ---------------------------------------------------------------------------
// groupProviders
// ---------------------------------------------------------------------------

func TestGroupProviders(t *testing.T) {
	old := []llmprovider.Provider{
		{
			Key:    "ds-1",
			Name:   "DeepSeek V4",
			Type:   "openai",
			APIURL: "https://api.deepseek.com/v1",
			APIKey: "key-1",
			Owner:  llmprovider.ProviderOwner{Type: "system"},
			Models: []llmprovider.ModelInfo{
				{ID: "deepseek-flash", Name: "Flash"},
			},
			Enabled: true,
		},
		{
			Key:    "ds-2",
			Name:   "DeepSeek V4 Thinking",
			Type:   "openai",
			APIURL: "https://api.deepseek.com/v1",
			APIKey: "key-1",
			Owner:  llmprovider.ProviderOwner{Type: "system"},
			Models: []llmprovider.ModelInfo{
				{ID: "deepseek-flash-thinking", Model: "deepseek-flash", Name: "Flash Thinking"},
			},
			Enabled: false,
		},
		{
			Key:    "openai-1",
			Name:   "OpenAI Official",
			Type:   "openai",
			APIURL: "https://api.openai.com/v1",
			APIKey: "sk-openai",
			Owner:  llmprovider.ProviderOwner{Type: "system"},
			Models: []llmprovider.ModelInfo{
				{ID: "gpt-4o", Name: "GPT-4o"},
			},
			Enabled: true,
		},
	}

	groups := groupProviders(old)

	// ds-1 and ds-2 share (deepseek, same URL, system) → one group
	// openai-1 has different URL → separate group
	require.Len(t, groups, 2)

	// Find the deepseek group
	var dsGroup, oaiGroup *providerGroup
	for i := range groups {
		if groups[i].key.providerID == ProviderDeepSeek {
			dsGroup = &groups[i]
		} else {
			oaiGroup = &groups[i]
		}
	}

	require.NotNil(t, dsGroup)
	assert.Equal(t, ProviderDeepSeek, dsGroup.key.providerID)
	assert.Len(t, dsGroup.oldKeys, 2)
	assert.Contains(t, dsGroup.oldKeys, "ds-1")
	assert.Contains(t, dsGroup.oldKeys, "ds-2")
	assert.Len(t, dsGroup.models, 2)
	assert.True(t, dsGroup.enabled, "should be true because ds-1 is enabled")

	require.NotNil(t, oaiGroup)
	assert.Equal(t, ProviderOpenAI, oaiGroup.key.providerID)
	assert.Len(t, oaiGroup.oldKeys, 1)
}

func TestGroupProviders_DedupSameProviderIDDiffURL(t *testing.T) {
	// Two openai-type providers with different URLs in same scope
	old := []llmprovider.Provider{
		{
			Key:    "openai-official",
			Name:   "OpenAI",
			Type:   "openai",
			APIURL: "https://api.openai.com/v1",
			Owner:  llmprovider.ProviderOwner{Type: "system"},
			Models: []llmprovider.ModelInfo{{ID: "gpt-4o"}},
		},
		{
			Key:    "tao-service",
			Name:   "Tao Service",
			Type:   "openai",
			APIURL: "https://tao.example.com/v1",
			Owner:  llmprovider.ProviderOwner{Type: "system"},
			Models: []llmprovider.ModelInfo{{ID: "tao-model"}},
		},
	}

	groups := groupProviders(old)
	require.Len(t, groups, 2)

	// First keeps standard ID; second uses name-derived slug
	assert.Equal(t, ProviderOpenAI, groups[0].key.providerID)
	assert.Equal(t, ProviderID("tao-service"), groups[1].key.providerID)
}

// ---------------------------------------------------------------------------
// slugifyName
// ---------------------------------------------------------------------------

func TestSlugifyName(t *testing.T) {
	tests := []struct {
		name   string
		expect ProviderID
	}{
		{"MiniMax M2.1 Highspeed", "minimax-m2.1-highspeed"},
		{"GPT-4.1 Mini", "gpt-4.1-mini"},
		{"Tao Service", "tao-service"},
		{"Qwen3.8-27B NVFP4 (No Thinking)", "qwen3.8-27b-nvfp4-no-thinking"},
		{"Siliconflow DeepSeek OCR", "siliconflow-deepseek-ocr"},
		{"  Spaces  Everywhere  ", "spaces-everywhere"},
		{"", "provider"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expect, slugifyName(tt.name))
		})
	}
}

// ---------------------------------------------------------------------------
// mergeModelVariants
// ---------------------------------------------------------------------------

func TestMergeModelVariants(t *testing.T) {
	// Simulate deepseek-flash family: 4 variants → 1 merged model
	models := []Model{
		{
			ModelID: "deepseek-flash", ModelAPI: "",
			Name: "DeepSeek Flash", ContextWindow: 100000, MaxOutputTokens: 50000,
			Enabled: true, Reasoning: false,
			ThinkingLevelMap: ThinkingLevelMap{ThinkingOff: nil},
			Capabilities:     []string{"vision", "streaming"},
		},
		{
			ModelID: "deepseek-flash-thinking", ModelAPI: "deepseek-flash",
			Name: "Flash Thinking High", ContextWindow: 1048576, MaxOutputTokens: 384000,
			Enabled: true, Reasoning: true,
			ThinkingLevelMap: ThinkingLevelMap{ThinkingHigh: strPtr("high")},
		},
		{
			ModelID: "deepseek-flash-thinking-max", ModelAPI: "deepseek-flash",
			Name: "Flash Thinking Max", ContextWindow: 1048576, MaxOutputTokens: 384000,
			Enabled: false, Reasoning: true,
			ThinkingLevelMap: ThinkingLevelMap{ThinkingMax: strPtr("max")},
		},
		{
			ModelID: "deepseek-flash-thinking-low", ModelAPI: "deepseek-flash",
			Name: "Flash Thinking Low", ContextWindow: 1048576, MaxOutputTokens: 384000,
			Enabled: false, Reasoning: true,
			ThinkingLevelMap: ThinkingLevelMap{ThinkingLow: strPtr("low")},
		},
	}

	variantMap := make(map[string]string)
	merged := mergeModelVariants(models, variantMap)

	require.Len(t, merged, 1, "4 variants should merge to 1 model")
	m := merged[0]
	assert.Equal(t, "deepseek-flash", m.ModelID)
	assert.Equal(t, "deepseek-flash", m.ModelAPI)
	assert.True(t, m.Enabled, "at least one variant was enabled")
	assert.True(t, m.Reasoning, "at least one variant had reasoning")
	assert.Equal(t, 1048576, m.ContextWindow, "should take max")
	assert.Equal(t, 384000, m.MaxOutputTokens, "should take max")

	// ThinkingLevelMap should contain all levels
	require.NotNil(t, m.ThinkingLevelMap)
	assert.Contains(t, m.ThinkingLevelMap, ThinkingOff)
	assert.Contains(t, m.ThinkingLevelMap, ThinkingHigh)
	assert.Contains(t, m.ThinkingLevelMap, ThinkingMax)
	assert.Contains(t, m.ThinkingLevelMap, ThinkingLow)

	// Variant aliases
	require.NotNil(t, m.Metadata)
	aliases, ok := m.Metadata["variant_aliases"].([]string)
	require.True(t, ok)
	assert.Len(t, aliases, 3)
	assert.Contains(t, aliases, "deepseek-flash-thinking")
	assert.Contains(t, aliases, "deepseek-flash-thinking-max")
	assert.Contains(t, aliases, "deepseek-flash-thinking-low")

	// VariantMap
	assert.Equal(t, "deepseek-flash", variantMap["deepseek-flash-thinking"])
	assert.Equal(t, "deepseek-flash", variantMap["deepseek-flash-thinking-max"])
	assert.Equal(t, "deepseek-flash", variantMap["deepseek-flash-thinking-low"])
}

func TestMergeModelVariants_SingleModel(t *testing.T) {
	models := []Model{
		{ModelID: "gpt-4o", Name: "GPT-4o"},
	}
	variantMap := make(map[string]string)
	merged := mergeModelVariants(models, variantMap)

	require.Len(t, merged, 1)
	assert.Equal(t, "gpt-4o", merged[0].ModelID)
	assert.Empty(t, variantMap, "no variants to remap")
}

func TestMergeModelVariants_DifferentModels(t *testing.T) {
	models := []Model{
		{ModelID: "gpt-4o", Name: "GPT-4o"},
		{ModelID: "gpt-5", Name: "GPT-5"},
		{ModelID: "dall-e-3", Name: "DALL-E 3"},
	}
	variantMap := make(map[string]string)
	merged := mergeModelVariants(models, variantMap)

	assert.Len(t, merged, 3, "different models should not merge")
	assert.Empty(t, variantMap)
}

// ---------------------------------------------------------------------------
// extractScopeID
// ---------------------------------------------------------------------------

func TestExtractScopeID(t *testing.T) {
	assert.Equal(t, "123", extractScopeID("setting:t123:llm.roles", "setting:t", ":llm.roles"))
	assert.Equal(t, "u42", extractScopeID("setting:uu42:llm.roles", "setting:u", ":llm.roles"))
	assert.Equal(t, "", extractScopeID("wrong-format", "setting:t", ":llm.roles"))
	assert.Equal(t, "", extractScopeID("setting:t:llm.roles", "setting:t", ":llm.roles"), "empty ID")
}

// ---------------------------------------------------------------------------
// translateModel Type field
// ---------------------------------------------------------------------------

func TestTranslateModelType(t *testing.T) {
	// Decision model
	mi := llmprovider.ModelInfo{
		ID:           "jev-latest",
		Capabilities: []string{"decision"},
	}
	m := translateModel(mi, APIOpenAICompletions)
	assert.Equal(t, ModelTypeDecision, m.Type)

	// Image model
	mi2 := llmprovider.ModelInfo{
		ID:           "dall-e-3",
		Capabilities: []string{"image_generation", "image_editing"},
	}
	m2 := translateModel(mi2, APIOpenAICompletions)
	assert.Equal(t, ModelTypeImage, m2.Type)

	// Embedding model (type stays chat, Embedding bool is set)
	mi3 := llmprovider.ModelInfo{
		ID:           "text-embedding-3-small",
		Capabilities: []string{"embedding"},
	}
	m3 := translateModel(mi3, APIOpenAICompletions)
	assert.Equal(t, ModelTypeChat, m3.Type)
	assert.True(t, m3.Embedding)

	// Audio model (type stays chat, Audio bool is set)
	mi4 := llmprovider.ModelInfo{
		ID:           "whisper-1",
		Capabilities: []string{"audio"},
	}
	m4 := translateModel(mi4, APIOpenAICompletions)
	assert.Equal(t, ModelTypeChat, m4.Type)
	assert.True(t, m4.Audio)
}

func strPtr(s string) *string {
	return &s
}
