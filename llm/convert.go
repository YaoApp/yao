package llm

import (
	"github.com/yaoapp/yao/llmprovider"
)

// ToConnectorProfile converts a Provider+Model pair to the legacy
// ConnectorProfile format for consumers that have not yet migrated.
// This is a bridge until all consumers switch to yao/llm directly.
func ToConnectorProfile(p *Provider, m *Model) llmprovider.ConnectorProfile {
	cp := llmprovider.ConnectorProfile{
		APIKey:    p.APIKey,
		BaseURL:   p.BaseURL,
		API:       string(p.API),
		Model:     modelAPIName(m),
		Input:     m.Input,
		Protocols: toStringSliceFromAPIs(p.Protocols),
	}

	if m.ContextWindow > 0 {
		cp.ContextWindow = m.ContextWindow
	}
	if m.MaxOutputTokens > 0 {
		cp.MaxTokens = m.MaxOutputTokens
	}

	// Reasoning fields
	if m.Reasoning && m.ThinkingLevelMap != nil {
		cp.Reasoning = inferDefaultReasoning(m.ThinkingLevelMap)
		cp.ReasoningEfforts = flattenThinkingMap(m.ThinkingLevelMap)
	}
	if !m.Reasoning {
		cp.DisableReasoning = true
	}

	// Compat extraction — model values override provider values
	merged := mergeCompat(p.Compat, m.Compat)
	if tf := getCompatString(merged, "thinkingFormat"); tf != "" {
		cp.ThinkingFormat = tf
	}
	if ndr := getCompatBool(merged, "supportsDeveloperRole"); ndr != nil && !*ndr {
		cp.NoDeveloperRole = true
	}
	if bt := getCompatInt(merged, "budgetTokens"); bt > 0 {
		cp.BudgetTokens = bt
	}

	return cp
}

// ToPiAiRoute constructs a DSH PiAiRoute from a Provider+Model pair.
// This is the zero-translation path for dsh runner configuration.
func ToPiAiRoute(p *Provider, m *Model, routeName, apiKeyEnv string) PiAiRoute {
	route := PiAiRoute{
		Name:      routeName,
		API:       string(p.API),
		BaseURL:   p.BaseURL,
		APIKeyEnv: apiKeyEnv,
	}

	// Reasoning
	if m.Reasoning && m.ThinkingLevelMap != nil {
		route.Reasoning = inferDefaultReasoning(m.ThinkingLevelMap)
	}

	// Budget tokens from compat
	merged := mergeCompat(p.Compat, m.Compat)
	if bt := getCompatInt(merged, "budgetTokens"); bt > 0 {
		route.BudgetTokens = bt
	}

	// ThinkingFormat
	if tf := getCompatString(merged, "thinkingFormat"); tf != "" {
		route.ThinkingFormat = tf
	}

	// NoDeveloperRole
	if ndr := getCompatBool(merged, "supportsDeveloperRole"); ndr != nil && !*ndr {
		route.NoDeveloperRole = true
	}

	// Model config
	mc := PiAiModelConfig{
		ID:    modelAPIName(m),
		Input: m.Input,
	}
	if m.ContextWindow > 0 {
		mc.ContextWindow = m.ContextWindow
	}
	if m.MaxOutputTokens > 0 {
		mc.MaxTokens = m.MaxOutputTokens
	}
	if m.Reasoning && m.ThinkingLevelMap != nil {
		mc.ReasoningEfforts = flattenThinkingMap(m.ThinkingLevelMap)
	}
	if !m.Reasoning {
		mc.DisableReasoning = true
	}
	route.Models = []PiAiModelConfig{mc}

	return route
}

// ---------------------------------------------------------------------------
// Compat merge and extraction
// ---------------------------------------------------------------------------

// mergeCompat merges provider-level and model-level compat maps.
// Model values override provider values.
func mergeCompat(providerCompat, modelCompat JSONMap) JSONMap {
	if providerCompat == nil && modelCompat == nil {
		return nil
	}
	merged := make(JSONMap)
	for k, v := range providerCompat {
		merged[k] = v
	}
	for k, v := range modelCompat {
		merged[k] = v
	}
	return merged
}

func getCompatString(m JSONMap, key string) string {
	if m == nil {
		return ""
	}
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getCompatBool(m JSONMap, key string) *bool {
	if m == nil {
		return nil
	}
	v, ok := m[key]
	if !ok {
		return nil
	}
	if b, ok := v.(bool); ok {
		return &b
	}
	return nil
}

func getCompatInt(m JSONMap, key string) int {
	if m == nil {
		return 0
	}
	return toIntFromAny(m[key])
}

// ---------------------------------------------------------------------------
// ThinkingLevelMap helpers
// ---------------------------------------------------------------------------

// inferDefaultReasoning picks a sensible default reasoning level from the map.
// Prefers "high" > "medium" > first available level.
func inferDefaultReasoning(m ThinkingLevelMap) string {
	if m == nil {
		return ""
	}
	preferred := []ThinkingLevel{ThinkingHigh, ThinkingMedium, ThinkingLow}
	for _, lv := range preferred {
		if _, ok := m[lv]; ok {
			return string(lv)
		}
	}
	for lv := range m {
		if lv != ThinkingOff {
			return string(lv)
		}
	}
	return ""
}

// flattenThinkingMap converts ThinkingLevelMap to the map[string]string
// format that PiAiModelConfig.ReasoningEfforts expects.
// nil *string values are rendered as the level name itself (provider default).
func flattenThinkingMap(m ThinkingLevelMap) map[string]string {
	if m == nil {
		return nil
	}
	out := make(map[string]string, len(m))
	for lv, v := range m {
		if v != nil {
			out[string(lv)] = *v
		} else {
			out[string(lv)] = string(lv)
		}
	}
	return out
}

// modelAPIName returns the wire model name for API requests.
// Uses ModelAPI (the upstream model identifier) when set, otherwise ModelID.
func modelAPIName(m *Model) string {
	if m.ModelAPI != "" {
		return m.ModelAPI
	}
	return m.ModelID
}

func toStringSliceFromAPIs(apis []API) []string {
	if apis == nil {
		return nil
	}
	out := make([]string, len(apis))
	for i, a := range apis {
		out[i] = string(a)
	}
	return out
}
