package llmprovider

import (
	"github.com/yaoapp/gou/connector"
	goullm "github.com/yaoapp/gou/llm"
)

// pi-ai wire protocol identifiers.
const (
	APIOpenAICompletions = "openai-completions"
	APIAnthropicMessages = "anthropic-messages"
)

// ConnectorProfile holds normalized model parameters extracted from a connector.
// Field names and value domains align with pi-ai (@earendil-works/pi-ai) configuration,
// so downstream consumers (DSH, Claude, OpenCode runners) work in pi-ai vocabulary.
// One ConnectorProfile corresponds to one Connector+Model pair (gou's current 1:1 design).
// When gou refactors the connector model, it can output this format directly.
type ConnectorProfile struct {
	// Connection
	APIKey  string
	BaseURL string
	API     string // pi-ai protocol: "openai-completions" | "anthropic-messages"

	// Model
	Model string
	Input []string // pi-ai input modalities: ["text"] | ["text", "image"]

	// Context / token limits
	ContextWindow int // caps.MaxInputTokens; 0 = unknown
	MaxTokens     int // caps.MaxOutputTokens; 0 = unknown

	// Reasoning (pi-ai vocabulary)
	Reasoning        string            // route-level default: "off"|"low"|"medium"|"high"|"xhigh"|"max"|""
	ReasoningEfforts map[string]string // model-level level→wire map (e.g. {off:none, low:low}); nil = unknown
	DisableReasoning bool              // true → model has no reasoning (render reasoningEfforts: false)
	BudgetTokens     int               // > 0 → thinkingBudgets.high

	// Wire compatibility
	ThinkingFormat  string   // pi-ai compat.thinkingFormat: "deepseek"|"openai"|""; cleared for anthropic-messages
	NoDeveloperRole bool     // true → compat.supportsDeveloperRole: false
	Protocols       []string // setting["protocols"] raw list for dual-protocol gateways
}

// ExtractProfile reads connector configuration and translates it into pi-ai vocabulary.
// All connector→pi-ai translation happens here; callers only consume pi-ai terms.
func ExtractProfile(c connector.Connector) ConnectorProfile {
	if c == nil {
		return ConnectorProfile{Input: []string{"text"}}
	}

	p := ConnectorProfile{Input: []string{"text"}}

	// --- Connection: key, url, model ---
	if lc, ok := c.(goullm.LLMConnector); ok {
		p.APIKey = lc.GetKey()
		p.BaseURL = lc.GetURL()
		if m := lc.GetModel(); m != "" {
			p.Model = m
		}

		// --- Capabilities ---
		if caps := lc.GetCapabilities(); caps != nil {
			p.ContextWindow = caps.MaxInputTokens
			p.MaxTokens = caps.MaxOutputTokens
			if caps.HasVision() {
				p.Input = []string{"text", "image"}
			}
		}
	} else {
		p.APIKey = connectorSettingStr(c, "key")
		if p.APIKey == "" {
			p.APIKey = connectorSettingStr(c, "api_key")
		}
		p.BaseURL = connectorSettingStr(c, "host")
		if p.BaseURL == "" {
			p.BaseURL = connectorSettingStr(c, "base_url")
		}
		p.Model = connectorSettingStr(c, "model")
	}

	// --- API protocol ---
	p.API = extractAPI(c)
	p.Protocols = extractProtocols(c)

	// --- Reasoning: 3-level priority chain ---
	extractReasoning(c, &p)

	// --- Thinking format ---
	p.ThinkingFormat = extractThinkingFormat(c)
	if p.API == APIAnthropicMessages {
		p.ThinkingFormat = "" // thinkingFormat is an openai-completions compat field
	}

	// --- Developer role ---
	p.NoDeveloperRole = p.API != APIAnthropicMessages

	// --- ReasoningEfforts from metadata ---
	extractReasoningEffortsMap(c, &p)

	// debugDumpProfile(c, &p)
	return p
}

// debugDumpProfile prints connector and profile state to stderr for diagnostics.
// Commented out in normal operation; uncomment the call in ExtractProfile to enable.
//
// func debugDumpProfile(c connector.Connector, p *ConnectorProfile) {
// 	fmt.Fprintf(os.Stderr, "\n[ExtractProfile] id=%s model=%q\n", c.ID(), p.Model)
// 	fmt.Fprintf(os.Stderr, "  GetMetadata()=%v\n", c.GetMetadata())
// 	fmt.Fprintf(os.Stderr, "  Setting()=%v\n", c.Setting())
// 	if lc, ok := c.(goullm.LLMConnector); ok {
// 		fmt.Fprintf(os.Stderr, "  GetCapabilities()=%+v\n", lc.GetCapabilities())
// 	}
// 	fmt.Fprintf(os.Stderr, "  => api=%q Reasoning=%q Efforts=%v Disable=%v Ctx=%d Max=%d ThinkFmt=%q NoDev=%v\n",
// 		p.API, p.Reasoning, p.ReasoningEfforts, p.DisableReasoning, p.ContextWindow, p.MaxTokens, p.ThinkingFormat, p.NoDeveloperRole)
// }

// extractAPI determines the pi-ai wire protocol from connector type and settings.
func extractAPI(c connector.Connector) string {
	// setting["protocols"] overrides connector type
	if settings := c.Setting(); settings != nil {
		if protocols, ok := settings["protocols"]; ok {
			if list, ok := protocols.([]interface{}); ok {
				for _, v := range list {
					if s, ok := v.(string); ok && s == "anthropic" {
						return APIAnthropicMessages
					}
				}
			}
		}
	}
	if c.Is(connector.ANTHROPIC) {
		return APIAnthropicMessages
	}
	return APIOpenAICompletions
}

// extractProtocols reads the raw protocols list from setting.
func extractProtocols(c connector.Connector) []string {
	settings := c.Setting()
	if settings == nil {
		return nil
	}
	raw, ok := settings["protocols"]
	if !ok {
		return nil
	}
	list, ok := raw.([]interface{})
	if !ok {
		return nil
	}
	var out []string
	for _, v := range list {
		if s, ok := v.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

// extractReasoning populates Reasoning and BudgetTokens using the 3-level priority chain:
//  1. Setting()["reasoning_effort"] — from ExtraBody promotion
//  2. GetMetadata()["reasoning_effort"] — metadata backup
//  3. Setting()["thinking"] — legacy .conn.yao compatibility
//
// When priorities 1/2 yield no value, fallback uses thinkingType and hasReasoning.
func extractReasoning(c connector.Connector, p *ConnectorProfile) {
	settings := c.Setting()

	// Always read budgetTokens and thinkingType from Setting()["thinking"]
	var thinkingType string
	if settings != nil {
		if thinking, ok := settings["thinking"].(map[string]interface{}); ok {
			if t, ok := thinking["type"].(string); ok {
				thinkingType = t
			}
			if bt, ok := thinking["budget_tokens"].(float64); ok {
				p.BudgetTokens = int(bt)
			} else if bt, ok := thinking["budget_tokens"].(int); ok {
				p.BudgetTokens = bt
			}
		}
	}

	// hasReasoning from capabilities
	var hasReasoning bool
	if lc, ok := c.(goullm.LLMConnector); ok {
		if caps := lc.GetCapabilities(); caps != nil {
			hasReasoning = caps.Reasoning
		}
	}

	// Priority 1: Setting()["reasoning_effort"]
	if settings != nil {
		if v, ok := settings["reasoning_effort"].(string); ok && v != "" {
			p.Reasoning = connectorEffortToPiAi(v)
			return
		}
	}

	// Priority 2: GetMetadata()["reasoning_effort"]
	if meta := c.GetMetadata(); meta != nil {
		if v, ok := meta["reasoning_effort"].(string); ok && v != "" {
			p.Reasoning = connectorEffortToPiAi(v)
			return
		}
	}

	// Priority 3: fallback from thinkingType / hasReasoning
	switch thinkingType {
	case "disabled":
		p.Reasoning = "off"
		return
	case "enabled", "adaptive":
		p.Reasoning = "high"
		return
	}
	if hasReasoning {
		p.Reasoning = "high"
		return
	}
	// No preference — leave empty, let pi-ai decide from model catalog.
}

// connectorEffortToPiAi translates a connector reasoning_effort value to pi-ai vocabulary.
func connectorEffortToPiAi(v string) string {
	switch v {
	case "none":
		return "off"
	case "thinking":
		return "high"
	case "off":
		return "off"
	case "low", "medium", "high", "xhigh", "max":
		return v
	default:
		return "high"
	}
}

// extractThinkingFormat reads the thinking wire format from a connector.
// Returns a format string (e.g. "deepseek") for pi-ai's compat.thinkingFormat,
// or "" to let pi-ai detect from the endpoint URL.
func extractThinkingFormat(c connector.Connector) string {
	if meta := c.GetMetadata(); meta != nil {
		if v, ok := meta["thinking_format"].(string); ok && v != "" {
			return v
		}
	}
	if settings := c.Setting(); settings != nil {
		if thinking, ok := settings["thinking"].(map[string]interface{}); ok {
			if _, hasType := thinking["type"]; hasType {
				return "deepseek"
			}
		}
	}
	return ""
}

// extractReasoningEffortsMap populates ReasoningEfforts and DisableReasoning.
// Uses metadata["reasoning_efforts"] when available, otherwise infers from connector state.
// The result is filtered to the current variant's effort level only.
func extractReasoningEffortsMap(c connector.Connector, p *ConnectorProfile) {
	meta := c.GetMetadata()

	// Try explicit reasoning_efforts list from metadata first.
	if meta != nil {
		if raw, ok := meta["reasoning_efforts"]; ok {
			if fullMap := parseEffortsList(raw); fullMap != nil {
				if filtered := filterEffortsForVariant(fullMap, p.Reasoning, c); filtered != nil {
					p.ReasoningEfforts = filtered
					return
				}
			}
		}
	}

	// Fallback: infer from reasoning_effort (singular) + capabilities.
	inferReasoningEfforts(c, p)
}

// filterEffortsForVariant reduces the full family efforts map to only the
// current variant's level. Disabled variants get off + one non-off level
// (PI requires at least one non-off). The off wire value is set by dialect:
// deepseek → "" (YAML null, PI sends thinking:{type:disabled}),
// standard → "none" (PI sends reasoning_effort:none).
func filterEffortsForVariant(fullMap map[string]string, reasoning string, c connector.Connector) map[string]string {
	if reasoning == "" {
		return nil
	}

	if reasoning == "off" {
		result := make(map[string]string, 2)
		if isDeepseekDialect(c) {
			result["off"] = "" // YAML null → PI uses deepseek format to disable
		} else {
			result["off"] = "none" // PI sends reasoning_effort=none
		}
		// PI validation requires at least one non-off level.
		if wire, ok := fullMap["high"]; ok {
			result["high"] = wire
		} else {
			for k, v := range fullMap {
				if k != "off" {
					result[k] = v
					break
				}
			}
		}
		return result
	}

	// Active reasoning variant: only the current level.
	result := make(map[string]string, 1)
	if wire, ok := fullMap[reasoning]; ok {
		result[reasoning] = wire
	} else {
		result[reasoning] = reasoning
	}
	return result
}

// parseEffortsList converts a reasoning_efforts value ([]interface{} or []string)
// into the pi-ai level->wire map. Returns nil when the value is absent or empty.
func parseEffortsList(raw interface{}) map[string]string {
	var list []interface{}
	switch v := raw.(type) {
	case []interface{}:
		list = v
	case []string:
		list = make([]interface{}, len(v))
		for i, s := range v {
			list[i] = s
		}
	default:
		return nil
	}
	if len(list) == 0 {
		return nil
	}

	m := make(map[string]string, len(list))
	for _, item := range list {
		s, ok := item.(string)
		if !ok {
			continue
		}
		switch s {
		case "none":
			m["off"] = "none"
		case "thinking":
			m["high"] = "high"
		case "off":
			m["off"] = "off"
		case "low", "medium", "high", "xhigh", "max":
			m[s] = s
		}
	}
	if len(m) == 0 {
		return nil
	}
	return m
}

// inferReasoningEfforts derives ReasoningEfforts from the connector's
// reasoning_effort (singular) and capabilities when no explicit efforts list exists.
// Distinguishes three cases:
//   - pure non-reasoning (no reasoning hints) → DisableReasoning=true
//   - disabled variant of a reasoning family → off + high, wire by dialect
//   - active reasoning variant → single-level map
func inferReasoningEfforts(c connector.Connector, p *ConnectorProfile) {
	effort := p.Reasoning // already resolved by extractReasoning

	if effort == "" {
		p.DisableReasoning = true
		p.Reasoning = "off"
		return
	}

	if effort == "off" {
		if isDeepseekDialect(c) {
			// deepseek dialect: off wire=null (empty string → YAML null)
			// PI sends thinking:{type:disabled}
			p.ReasoningEfforts = map[string]string{"off": "", "high": "high"}
		} else if hasReasoningHints(c) {
			// standard dialect: off wire="none"
			// PI sends reasoning_effort=none
			p.ReasoningEfforts = map[string]string{"off": "none", "high": "high"}
		} else {
			// pure non-reasoning model
			p.DisableReasoning = true
		}
		return
	}

	// Active reasoning variant: single-level map.
	if effort == "thinking" {
		p.ReasoningEfforts = map[string]string{"high": "high"}
	} else {
		p.ReasoningEfforts = map[string]string{effort: effort}
	}
}

// isDeepseekDialect reports whether the connector uses the deepseek thinking
// wire format (thinking:{type:enabled/disabled}). PI needs thinkingFormat:"deepseek"
// and off:null for this dialect.
func isDeepseekDialect(c connector.Connector) bool {
	if s := c.Setting(); s != nil {
		if _, ok := s["thinking"]; ok {
			return true
		}
	}
	return false
}

// hasReasoningHints reports whether the connector belongs to a reasoning model
// family, even when the current variant is disabled. Disabled variants carry
// reasoning-related parameters (thinking, reasoning_effort, enable_thinking)
// that distinguish them from pure non-reasoning models.
func hasReasoningHints(c connector.Connector) bool {
	if s := c.Setting(); s != nil {
		if _, ok := s["thinking"]; ok {
			return true
		}
		if v, ok := s["reasoning_effort"].(string); ok && v != "" {
			return true
		}
		if _, ok := s["enable_thinking"]; ok {
			return true
		}
	}
	if m := c.GetMetadata(); m != nil {
		if v, ok := m["reasoning_effort"].(string); ok && v != "" {
			return true
		}
	}
	return false
}

// connectorSettingStr reads a string value from a connector's Setting() map.
func connectorSettingStr(c connector.Connector, key string) string {
	if c == nil {
		return ""
	}
	settings := c.Setting()
	if settings == nil {
		return ""
	}
	if v, ok := settings[key].(string); ok {
		return v
	}
	return ""
}
