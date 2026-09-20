package setting

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/yaoapp/yao/llmprovider"
)

// ---------------------------------------------------------------------------
// classifySwitchValues
// ---------------------------------------------------------------------------

func TestClassifySwitchValues_ObjectValues(t *testing.T) {
	sw := &reasoningSwitch{
		Param: "thinking",
		Values: []interface{}{
			map[string]interface{}{"type": "enabled"},
			map[string]interface{}{"type": "disabled"},
		},
	}
	disabled, enabled := classifySwitchValues(sw)
	if disabled == nil {
		t.Fatal("expected disabled value")
	}
	if enabled == nil {
		t.Fatal("expected enabled value")
	}
	dm, _ := disabled.(map[string]interface{})
	if dm["type"] != "disabled" {
		t.Fatalf("expected disabled type, got %v", dm["type"])
	}
	em, _ := enabled.(map[string]interface{})
	if em["type"] != "enabled" {
		t.Fatalf("expected enabled type, got %v", em["type"])
	}
}

func TestClassifySwitchValues_BoolValues(t *testing.T) {
	sw := &reasoningSwitch{
		Param:  "enable_thinking",
		Values: []interface{}{true, false},
	}
	disabled, enabled := classifySwitchValues(sw)
	if disabled != false {
		t.Fatalf("expected disabled=false, got %v", disabled)
	}
	if enabled != true {
		t.Fatalf("expected enabled=true, got %v", enabled)
	}
}

func TestClassifySwitchValues_OnlyEnabled(t *testing.T) {
	sw := &reasoningSwitch{
		Param: "thinking",
		Values: []interface{}{
			map[string]interface{}{"type": "enabled", "keep": "all"},
		},
	}
	disabled, enabled := classifySwitchValues(sw)
	if disabled != nil {
		t.Fatalf("expected nil disabled, got %v", disabled)
	}
	if enabled == nil {
		t.Fatal("expected non-nil enabled")
	}
}

func TestClassifySwitchValues_Empty(t *testing.T) {
	sw := &reasoningSwitch{Param: "thinking", Values: []interface{}{}}
	disabled, enabled := classifySwitchValues(sw)
	if disabled != nil || enabled != nil {
		t.Fatalf("expected nil/nil for empty values, got %v/%v", disabled, enabled)
	}
}

// ---------------------------------------------------------------------------
// isSwitchDisabled
// ---------------------------------------------------------------------------

func TestIsSwitchDisabled(t *testing.T) {
	tests := []struct {
		name   string
		value  interface{}
		expect bool
	}{
		{"bool false", false, true},
		{"bool true", true, false},
		{"object disabled", map[string]interface{}{"type": "disabled"}, true},
		{"object enabled", map[string]interface{}{"type": "enabled"}, false},
		{"object enabled with keep", map[string]interface{}{"type": "enabled", "keep": "all"}, false},
		{"string unknown", "something", false},
		{"nil", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isSwitchDisabled(tt.value)
			if got != tt.expect {
				t.Fatalf("isSwitchDisabled(%v) = %v, want %v", tt.value, got, tt.expect)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// expandReasoningVariants
// ---------------------------------------------------------------------------

func deepseekFlashBase() map[string]interface{} {
	return map[string]interface{}{
		"id":                "deepseek-flash",
		"name":              "DeepSeek Flash",
		"capabilities":      []interface{}{"chat", "streaming", "json", "tool_calls", "reasoning", "vision"},
		"max_input_tokens":  1000000,
		"max_output_tokens": 384000,
		"reasoning": map[string]interface{}{
			"switch": map[string]interface{}{
				"param":  "thinking",
				"values": []interface{}{map[string]interface{}{"type": "enabled"}, map[string]interface{}{"type": "disabled"}},
			},
			"effort": map[string]interface{}{
				"param":  "reasoning_effort",
				"values": []interface{}{"low", "high", "max"},
			},
			"default": map[string]interface{}{
				"thinking":         map[string]interface{}{"type": "enabled"},
				"reasoning_effort": "high",
			},
			"can_disable": true,
			"disable_with": []interface{}{
				map[string]interface{}{"thinking": map[string]interface{}{"type": "disabled"}},
			},
		},
	}
}

func TestExpandReasoningVariants_DeepseekFlash(t *testing.T) {
	invalidateRemoteModelCache()
	remoteReasoningCache = make(map[string]interface{})

	variants := expandReasoningVariants(deepseekFlashBase())
	if len(variants) != 4 {
		t.Fatalf("expected 4 variants for deepseek-flash, got %d", len(variants))
	}

	// Base: disabled
	assertVariant(t, variants[0], "deepseek-flash", "", true, false)
	// thinking-low
	assertVariant(t, variants[1], "deepseek-flash-thinking-low", "deepseek-flash", false, true)
	// thinking-high
	assertVariant(t, variants[2], "deepseek-flash-thinking-high", "deepseek-flash", false, true)
	// thinking-max
	assertVariant(t, variants[3], "deepseek-flash-thinking-max", "deepseek-flash", false, true)

	// Verify variant names include thinking level
	if variants[0].Name != "DeepSeek Flash" {
		t.Fatalf("base variant name = %q, want 'DeepSeek Flash'", variants[0].Name)
	}
	if variants[1].Name != "DeepSeek Flash (Thinking: Low)" {
		t.Fatalf("thinking-low name = %q, want 'DeepSeek Flash (Thinking: Low)'", variants[1].Name)
	}
	if variants[2].Name != "DeepSeek Flash (Thinking: High)" {
		t.Fatalf("thinking-high name = %q, want 'DeepSeek Flash (Thinking: High)'", variants[2].Name)
	}
	if variants[3].Name != "DeepSeek Flash (Thinking: Max)" {
		t.Fatalf("thinking-max name = %q, want 'DeepSeek Flash (Thinking: Max)'", variants[3].Name)
	}

	// Verify disabled variant Options equals disable_with[0]
	dOpts := variants[0].Options
	if dOpts == nil {
		t.Fatal("expected options on disabled variant")
	}
	dThinking, ok := dOpts["thinking"].(map[string]interface{})
	if !ok {
		t.Fatal("expected thinking map in disabled variant options")
	}
	if dThinking["type"] != "disabled" {
		t.Fatalf("disabled variant thinking type = %v, want disabled", dThinking["type"])
	}

	// Verify enabled variant Options structure (nested object preserved)
	opts := variants[1].Options
	if opts == nil {
		t.Fatal("expected options on thinking-low")
	}
	if _, ok := opts["thinking"]; !ok {
		t.Fatal("expected thinking key in options")
	}
	if _, ok := opts["reasoning_effort"]; !ok {
		t.Fatal("expected reasoning_effort key in options")
	}
}

func TestExpandReasoningVariants_KimiK3(t *testing.T) {
	invalidateRemoteModelCache()
	remoteReasoningCache = make(map[string]interface{})

	base := map[string]interface{}{
		"id":           "kimi-k3",
		"name":         "Kimi K3",
		"capabilities": []interface{}{"chat", "streaming", "json", "tool_calls", "reasoning", "vision"},
		"reasoning": map[string]interface{}{
			"switch": map[string]interface{}{
				"param":  "thinking",
				"values": []interface{}{map[string]interface{}{"type": "enabled"}, map[string]interface{}{"type": "disabled"}},
			},
			"effort": map[string]interface{}{
				"param":  "reasoning_effort",
				"values": []interface{}{"none", "low", "high", "max"},
			},
			"default": map[string]interface{}{
				"reasoning_effort": "max",
			},
			"can_disable": true,
			"disable_with": []interface{}{
				map[string]interface{}{"thinking": map[string]interface{}{"type": "disabled"}},
				map[string]interface{}{"reasoning_effort": "none"},
			},
		},
	}

	variants := expandReasoningVariants(base)
	// 1 disabled + 3 effort (none skipped) = 4
	if len(variants) != 4 {
		t.Fatalf("expected 4 variants for kimi-k3, got %d", len(variants))
	}

	// Base: disabled variant
	assertVariant(t, variants[0], "kimi-k3", "", true, false)
	dOpts := variants[0].Options
	if dOpts == nil {
		t.Fatal("expected options on disabled variant")
	}
	dThinking, ok := dOpts["thinking"].(map[string]interface{})
	if !ok {
		t.Fatal("expected thinking map in kimi-k3 disabled variant options")
	}
	if dThinking["type"] != "disabled" {
		t.Fatalf("kimi-k3 disabled variant thinking type = %v, want disabled", dThinking["type"])
	}

	// Thinking variants
	assertVariant(t, variants[1], "kimi-k3-thinking-low", "kimi-k3", false, true)
	assertVariant(t, variants[2], "kimi-k3-thinking-high", "kimi-k3", false, true)
	assertVariant(t, variants[3], "kimi-k3-thinking-max", "kimi-k3", false, true)

	// "none" should NOT produce a variant
	for _, v := range variants {
		if v.ID == "kimi-k3-thinking-none" {
			t.Fatal("reasoning_effort=none should be skipped")
		}
	}

	// Verify variant names
	if variants[0].Name != "Kimi K3" {
		t.Fatalf("disabled variant name = %q, want 'Kimi K3'", variants[0].Name)
	}
	if variants[1].Name != "Kimi K3 (Thinking: Low)" {
		t.Fatalf("thinking-low name = %q, want 'Kimi K3 (Thinking: Low)'", variants[1].Name)
	}
}

func TestExpandReasoningVariants_KimiK27Code(t *testing.T) {
	invalidateRemoteModelCache()
	remoteReasoningCache = make(map[string]interface{})

	base := map[string]interface{}{
		"id":           "kimi-k2.7-code",
		"name":         "Kimi K2.7 Code",
		"capabilities": []interface{}{"chat", "streaming", "json", "tool_calls", "vision"},
		"reasoning": map[string]interface{}{
			"switch": map[string]interface{}{
				"param":  "thinking",
				"values": []interface{}{map[string]interface{}{"type": "enabled", "keep": "all"}},
			},
			"default": map[string]interface{}{
				"thinking": map[string]interface{}{"type": "enabled", "keep": "all"},
			},
			"can_disable": false,
		},
	}

	variants := expandReasoningVariants(base)
	if len(variants) != 1 {
		t.Fatalf("expected 1 variant for kimi-k2.7-code, got %d", len(variants))
	}

	v := variants[0]
	if v.ID != "kimi-k2.7-code" {
		t.Fatalf("expected ID=kimi-k2.7-code, got %s", v.ID)
	}
	if !v.Enabled {
		t.Fatal("expected enabled=true")
	}
	if !hasCap(v.Capabilities, "reasoning") {
		t.Fatal("expected reasoning capability")
	}
}

func TestExpandReasoningVariants_Qwen38Flash(t *testing.T) {
	invalidateRemoteModelCache()
	remoteReasoningCache = make(map[string]interface{})

	base := map[string]interface{}{
		"id":           "qwen3.8-flash",
		"name":         "Qwen 3.8 Flash",
		"capabilities": []interface{}{"chat", "streaming", "json", "reasoning", "vision"},
		"reasoning": map[string]interface{}{
			"switch": map[string]interface{}{
				"param":  "enable_thinking",
				"values": []interface{}{true, false},
			},
			"effort": map[string]interface{}{
				"param":  "reasoning_effort",
				"values": []interface{}{"none", "minimal", "low", "medium", "high", "xhigh", "max"},
			},
			"default": map[string]interface{}{
				"enable_thinking":  true,
				"reasoning_effort": "high",
			},
			"can_disable": true,
			"disable_with": []interface{}{
				map[string]interface{}{"enable_thinking": false},
				map[string]interface{}{"reasoning_effort": "none"},
			},
		},
	}

	variants := expandReasoningVariants(base)
	// 1 disabled + 6 effort (none skipped) = 7
	if len(variants) != 7 {
		t.Fatalf("expected 7 variants for qwen3.8-flash (none skipped), got %d", len(variants))
	}

	// Base: disabled — Options should equal disable_with[0]
	assertVariant(t, variants[0], "qwen3.8-flash", "", true, false)
	if hasCap(variants[0].Capabilities, "reasoning") {
		t.Fatal("disabled variant should not have reasoning cap")
	}
	dOpts := variants[0].Options
	if dOpts == nil {
		t.Fatal("expected options on qwen disabled variant")
	}
	if dOpts["enable_thinking"] != false {
		t.Fatalf("qwen disabled variant enable_thinking = %v, want false", dOpts["enable_thinking"])
	}

	// "none" should NOT produce a variant (redundant with disabled base)
	for _, v := range variants {
		if v.ID == "qwen3.8-flash-thinking-none" {
			t.Fatal("reasoning_effort=none should be skipped; it is equivalent to thinking disabled")
		}
	}

	// All thinking variants should have reasoning cap
	for i := 1; i < len(variants); i++ {
		if !hasCap(variants[i].Capabilities, "reasoning") {
			t.Fatalf("variant %s should have reasoning cap", variants[i].ID)
		}
	}
}

func TestExpandReasoningVariants_GPT4o(t *testing.T) {
	invalidateRemoteModelCache()
	remoteReasoningCache = make(map[string]interface{})

	base := map[string]interface{}{
		"id":           "gpt-4o",
		"name":         "GPT-4o",
		"capabilities": []interface{}{"chat", "streaming", "json", "vision"},
	}

	variants := expandReasoningVariants(base)
	if len(variants) != 1 {
		t.Fatalf("expected 1 variant for gpt-4o, got %d", len(variants))
	}
	if variants[0].ID != "gpt-4o" {
		t.Fatalf("expected ID=gpt-4o, got %s", variants[0].ID)
	}
	if !variants[0].Enabled {
		t.Fatal("expected enabled=true")
	}
	if hasCap(variants[0].Capabilities, "reasoning") {
		t.Fatal("gpt-4o should not have reasoning cap")
	}
}

// ---------------------------------------------------------------------------
// resolveConnectorID
// ---------------------------------------------------------------------------

func deepseekFlashReasoning() interface{} {
	return map[string]interface{}{
		"switch": map[string]interface{}{
			"param":  "thinking",
			"values": []interface{}{map[string]interface{}{"type": "enabled"}, map[string]interface{}{"type": "disabled"}},
		},
		"effort": map[string]interface{}{
			"param":  "reasoning_effort",
			"values": []interface{}{"low", "high", "max"},
		},
		"default": map[string]interface{}{
			"thinking":         map[string]interface{}{"type": "enabled"},
			"reasoning_effort": "high",
		},
		"can_disable": true,
		"disable_with": []interface{}{
			map[string]interface{}{"thinking": map[string]interface{}{"type": "disabled"}},
		},
	}
}

func TestResolveConnectorID(t *testing.T) {
	reasoning := deepseekFlashReasoning()

	tests := []struct {
		name     string
		modelID  string
		params   map[string]interface{}
		reason   interface{}
		expected string
	}{
		{
			name:    "full params: enabled + high",
			modelID: "deepseek-flash",
			params: map[string]interface{}{
				"thinking":         map[string]interface{}{"type": "enabled"},
				"reasoning_effort": "high",
			},
			reason:   reasoning,
			expected: "deepseek-flash-thinking-high",
		},
		{
			name:    "partial params: effort only, switch补全",
			modelID: "deepseek-flash",
			params: map[string]interface{}{
				"reasoning_effort": "max",
			},
			reason:   reasoning,
			expected: "deepseek-flash-thinking-max",
		},
		{
			name:    "switch disabled",
			modelID: "deepseek-flash",
			params: map[string]interface{}{
				"thinking": map[string]interface{}{"type": "disabled"},
			},
			reason:   reasoning,
			expected: "deepseek-flash",
		},
		{
			name:     "empty params: use default",
			modelID:  "deepseek-flash",
			params:   map[string]interface{}{},
			reason:   reasoning,
			expected: "deepseek-flash-thinking-high",
		},
		{
			name:     "no reasoning spec",
			modelID:  "whisper-1",
			params:   map[string]interface{}{},
			reason:   nil,
			expected: "whisper-1",
		},
		{
			name:    "qwen dynamic param names",
			modelID: "qwen3.8-flash",
			params: map[string]interface{}{
				"enable_thinking":  true,
				"reasoning_effort": "high",
			},
			reason: map[string]interface{}{
				"switch": map[string]interface{}{
					"param":  "enable_thinking",
					"values": []interface{}{true, false},
				},
				"effort": map[string]interface{}{
					"param":  "reasoning_effort",
					"values": []interface{}{"none", "low", "high", "max"},
				},
				"default": map[string]interface{}{
					"enable_thinking":  true,
					"reasoning_effort": "high",
				},
			},
			expected: "qwen3.8-flash-thinking-high",
		},
		{
			name:    "kimi-k3 default effort returns base",
			modelID: "kimi-k3",
			params: map[string]interface{}{
				"reasoning_effort": "max",
			},
			reason: map[string]interface{}{
				"effort": map[string]interface{}{
					"param":  "reasoning_effort",
					"values": []interface{}{"low", "high", "max"},
				},
				"default": map[string]interface{}{
					"reasoning_effort": "max",
				},
			},
			expected: "kimi-k3",
		},
		{
			name:    "kimi-k3 non-default effort",
			modelID: "kimi-k3",
			params: map[string]interface{}{
				"reasoning_effort": "low",
			},
			reason: map[string]interface{}{
				"effort": map[string]interface{}{
					"param":  "reasoning_effort",
					"values": []interface{}{"low", "high", "max"},
				},
				"default": map[string]interface{}{
					"reasoning_effort": "max",
				},
			},
			expected: "kimi-k3-effort-low",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveConnectorID(tt.modelID, tt.params, tt.reason)
			if got != tt.expected {
				t.Fatalf("resolveConnectorID(%s, %v) = %s, want %s", tt.modelID, tt.params, got, tt.expected)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// taoFetchServices
// ---------------------------------------------------------------------------

func TestTaoFetchServices_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/services" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"object": "list",
			"data": []map[string]interface{}{
				{"endpoint": "chat_completions", "service": "chat", "unit": "TOK"},
				{"endpoint": "embeddings", "service": "embedding", "unit": "TOK"},
				{"endpoint": "search", "service": "search", "unit": "REQ"},
				{"endpoint": "fetch", "service": "fetch", "unit": "REQ"},
				{"endpoint": "ocr", "service": "ocr", "unit": "IMG"},
				{"endpoint": "generations", "service": "image", "unit": "IMG"},
				{"endpoint": "transcriptions", "service": "audio", "unit": "10S"},
			},
		})
	}))
	defer srv.Close()

	svc := taoFetchServices(srv.URL)
	if !svc.LLM {
		t.Fatal("expected llm=true")
	}
	if !svc.Search {
		t.Fatal("expected search=true")
	}
	if !svc.Scrape {
		t.Fatal("expected scrape=true (from fetch)")
	}
	if !svc.OCR {
		t.Fatal("expected ocr=true")
	}
	if !svc.Embedding {
		t.Fatal("expected embedding=true")
	}
	if !svc.Audio {
		t.Fatal("expected audio=true")
	}
	if !svc.Image {
		t.Fatal("expected image=true")
	}
}

func TestTaoFetchServices_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	svc := taoFetchServices(srv.URL)
	if svc.LLM || svc.Search || svc.Scrape || svc.OCR {
		t.Fatal("expected all false on server error")
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func assertVariant(t *testing.T, v llmprovider.ModelInfo, id, model string, enabled, hasReasoning bool) {
	t.Helper()
	if v.ID != id {
		t.Fatalf("expected ID=%s, got %s", id, v.ID)
	}
	if v.Model != model {
		t.Fatalf("expected Model=%s, got %s", model, v.Model)
	}
	if v.Enabled != enabled {
		t.Fatalf("expected Enabled=%v for %s, got %v", enabled, id, v.Enabled)
	}
	if hasReasoning && !hasCap(v.Capabilities, "reasoning") {
		t.Fatalf("expected reasoning cap for %s", id)
	}
	if !hasReasoning && hasCap(v.Capabilities, "reasoning") {
		t.Fatalf("unexpected reasoning cap for %s", id)
	}
}

func assertVariantExists(t *testing.T, variants []llmprovider.ModelInfo, id, model string) {
	t.Helper()
	for _, v := range variants {
		if v.ID == id {
			if v.Model != model {
				t.Fatalf("variant %s: expected Model=%s, got %s", id, model, v.Model)
			}
			return
		}
	}
	t.Fatalf("variant %s not found", id)
}

func hasCap(caps []string, cap string) bool {
	for _, c := range caps {
		if c == cap {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// mapRemoteModel
// ---------------------------------------------------------------------------

func TestMapRemoteModel_Basic(t *testing.T) {
	item := map[string]interface{}{
		"id":                "test-model",
		"name":              "Test Model",
		"capabilities":      []interface{}{"chat", "vision"},
		"description":       "A test model",
		"max_input_tokens":  128000.0,
		"max_output_tokens": 16384.0,
	}
	m := mapRemoteModel(item)
	if m == nil {
		t.Fatal("expected non-nil result")
	}
	if m["id"] != "test-model" {
		t.Fatalf("id = %v, want test-model", m["id"])
	}
	if m["name"] != "Test Model" {
		t.Fatalf("name = %v, want Test Model", m["name"])
	}
	caps := m["capabilities"].([]string)
	if len(caps) != 2 || caps[0] != "chat" || caps[1] != "vision" {
		t.Fatalf("capabilities = %v", caps)
	}
	if m["description"] != "A test model" {
		t.Fatalf("description = %v", m["description"])
	}
	if m["max_input_tokens"] != 128000 {
		t.Fatalf("max_input_tokens = %v", m["max_input_tokens"])
	}
	if m["max_output_tokens"] != 16384 {
		t.Fatalf("max_output_tokens = %v", m["max_output_tokens"])
	}
}

func TestMapRemoteModel_EmptyID(t *testing.T) {
	m := mapRemoteModel(map[string]interface{}{"id": ""})
	if m != nil {
		t.Fatal("expected nil for empty id")
	}
}

func TestMapRemoteModel_NameFallback(t *testing.T) {
	m := mapRemoteModel(map[string]interface{}{"id": "model-x"})
	if m["name"] != "model-x" {
		t.Fatalf("name should fallback to id, got %v", m["name"])
	}
}

func TestMapRemoteModel_WithReasoning(t *testing.T) {
	reasoning := map[string]interface{}{"switch": map[string]interface{}{"param": "thinking"}}
	m := mapRemoteModel(map[string]interface{}{
		"id":        "model-r",
		"reasoning": reasoning,
	})
	if m["reasoning"] == nil {
		t.Fatal("expected reasoning to be preserved")
	}
}

func TestMapRemoteModel_NoCaps(t *testing.T) {
	m := mapRemoteModel(map[string]interface{}{"id": "model-nc"})
	caps := m["capabilities"].([]string)
	if len(caps) != 0 {
		t.Fatalf("expected empty caps, got %v", caps)
	}
}

// ---------------------------------------------------------------------------
// parseReasoningSpec
// ---------------------------------------------------------------------------

func TestParseReasoningSpec_Nil(t *testing.T) {
	if parseReasoningSpec(nil) != nil {
		t.Fatal("expected nil for nil input")
	}
}

func TestParseReasoningSpec_NonMap(t *testing.T) {
	if parseReasoningSpec("not a map") != nil {
		t.Fatal("expected nil for non-map input")
	}
}

func TestParseReasoningSpec_EmptyMap(t *testing.T) {
	spec := parseReasoningSpec(map[string]interface{}{})
	if spec == nil {
		t.Fatal("expected non-nil spec for empty map")
	}
	if spec.Switch != nil || spec.Effort != nil || spec.Default != nil {
		t.Fatal("expected all nil fields for empty map")
	}
}

func TestParseReasoningSpec_SwitchOnly(t *testing.T) {
	spec := parseReasoningSpec(map[string]interface{}{
		"switch": map[string]interface{}{
			"param":  "thinking",
			"values": []interface{}{true, false},
		},
	})
	if spec.Switch == nil {
		t.Fatal("expected switch")
	}
	if spec.Switch.Param != "thinking" {
		t.Fatalf("switch param = %s, want thinking", spec.Switch.Param)
	}
	if len(spec.Switch.Values) != 2 {
		t.Fatalf("switch values = %d, want 2", len(spec.Switch.Values))
	}
}

func TestParseReasoningSpec_EffortOnly(t *testing.T) {
	spec := parseReasoningSpec(map[string]interface{}{
		"effort": map[string]interface{}{
			"param":  "reasoning_effort",
			"values": []interface{}{"low", "high"},
		},
	})
	if spec.Effort == nil {
		t.Fatal("expected effort")
	}
	if spec.Effort.Param != "reasoning_effort" {
		t.Fatalf("effort param = %s, want reasoning_effort", spec.Effort.Param)
	}
	if len(spec.Effort.Values) != 2 {
		t.Fatalf("effort values = %d, want 2", len(spec.Effort.Values))
	}
}

func TestParseReasoningSpec_CanDisable(t *testing.T) {
	spec := parseReasoningSpec(map[string]interface{}{
		"switch": map[string]interface{}{
			"param":  "thinking",
			"values": []interface{}{map[string]interface{}{"type": "enabled"}, map[string]interface{}{"type": "disabled"}},
		},
		"can_disable": true,
		"disable_with": []interface{}{
			map[string]interface{}{"thinking": map[string]interface{}{"type": "disabled"}},
			map[string]interface{}{"reasoning_effort": "none"},
		},
	})
	if spec.CanDisable == nil || !*spec.CanDisable {
		t.Fatal("expected CanDisable=true")
	}
	if len(spec.DisableWith) != 2 {
		t.Fatalf("expected 2 DisableWith entries, got %d", len(spec.DisableWith))
	}
	if spec.DisableWith[0]["thinking"] == nil {
		t.Fatal("expected thinking key in DisableWith[0]")
	}
	if spec.DisableWith[1]["reasoning_effort"] != "none" {
		t.Fatalf("DisableWith[1] reasoning_effort = %v, want none", spec.DisableWith[1]["reasoning_effort"])
	}
}

func TestParseReasoningSpec_CanDisableFalse(t *testing.T) {
	spec := parseReasoningSpec(map[string]interface{}{
		"can_disable": false,
	})
	if spec.CanDisable == nil || *spec.CanDisable {
		t.Fatal("expected CanDisable=false")
	}
	if len(spec.DisableWith) != 0 {
		t.Fatalf("expected 0 DisableWith entries, got %d", len(spec.DisableWith))
	}
}

func TestParseReasoningSpec_EmptyParam(t *testing.T) {
	spec := parseReasoningSpec(map[string]interface{}{
		"switch": map[string]interface{}{"param": "", "values": []interface{}{true}},
		"effort": map[string]interface{}{"param": "", "values": []interface{}{"low"}},
	})
	if spec.Switch != nil {
		t.Fatal("expected nil switch for empty param")
	}
	if spec.Effort != nil {
		t.Fatal("expected nil effort for empty param")
	}
}

// ---------------------------------------------------------------------------
// Utility functions: toStringSlice, removeCap, ensureCap, copyMap
// ---------------------------------------------------------------------------

func TestToStringSlice_StringSlice(t *testing.T) {
	result := toStringSlice([]string{"a", "b"})
	if len(result) != 2 || result[0] != "a" || result[1] != "b" {
		t.Fatalf("got %v", result)
	}
}

func TestToStringSlice_InterfaceSlice(t *testing.T) {
	result := toStringSlice([]interface{}{"x", "y", 42})
	if len(result) != 2 || result[0] != "x" || result[1] != "y" {
		t.Fatalf("got %v (non-strings should be skipped)", result)
	}
}

func TestToStringSlice_Nil(t *testing.T) {
	if toStringSlice(nil) != nil {
		t.Fatal("expected nil")
	}
}

func TestRemoveCap(t *testing.T) {
	result := removeCap([]string{"chat", "reasoning", "vision"}, "reasoning")
	if len(result) != 2 || result[0] != "chat" || result[1] != "vision" {
		t.Fatalf("got %v", result)
	}
}

func TestRemoveCap_NotPresent(t *testing.T) {
	result := removeCap([]string{"chat", "vision"}, "reasoning")
	if len(result) != 2 {
		t.Fatalf("should preserve all when not present, got %v", result)
	}
}

func TestEnsureCap_Missing(t *testing.T) {
	result := ensureCap([]string{"chat", "vision"}, "reasoning")
	if len(result) != 3 || result[2] != "reasoning" {
		t.Fatalf("got %v", result)
	}
}

func TestEnsureCap_AlreadyPresent(t *testing.T) {
	result := ensureCap([]string{"chat", "reasoning"}, "reasoning")
	if len(result) != 2 {
		t.Fatalf("should not duplicate, got %v", result)
	}
}

func TestCopyMap(t *testing.T) {
	orig := map[string]interface{}{"a": 1, "b": "two"}
	cp := copyMap(orig)
	cp["a"] = 99
	if orig["a"] != 1 {
		t.Fatal("copyMap should not affect original")
	}
}

func TestCopyMap_Empty(t *testing.T) {
	cp := copyMap(map[string]interface{}{})
	if len(cp) != 0 {
		t.Fatal("expected empty map")
	}
}

// ---------------------------------------------------------------------------
// taoFetchServices edge cases
// ---------------------------------------------------------------------------

func TestTaoFetchServices_EmptyData(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{"data": []interface{}{}})
	}))
	defer srv.Close()

	svc := taoFetchServices(srv.URL)
	if svc.LLM || svc.Search || svc.OCR {
		t.Fatal("expected all false for empty data")
	}
}

func TestTaoFetchServices_DuplicateService(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{
				{"service": "image", "endpoint": "generations"},
				{"service": "image", "endpoint": "edits"},
				{"service": "audio", "endpoint": "transcriptions"},
				{"service": "audio", "endpoint": "speech"},
			},
		})
	}))
	defer srv.Close()

	svc := taoFetchServices(srv.URL)
	if !svc.Image {
		t.Fatal("expected image=true")
	}
	if !svc.Audio {
		t.Fatal("expected audio=true")
	}
	if svc.LLM || svc.Search {
		t.Fatal("expected LLM/Search=false")
	}
}

func TestTaoFetchServices_UnknownService(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{
				{"service": "chat"},
				{"service": "unknown_new_service"},
			},
		})
	}))
	defer srv.Close()

	svc := taoFetchServices(srv.URL)
	if !svc.LLM {
		t.Fatal("expected chat→LLM=true")
	}
}

func TestTaoFetchServices_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("not json"))
	}))
	defer srv.Close()

	svc := taoFetchServices(srv.URL)
	if svc.LLM || svc.Search {
		t.Fatal("expected all false for invalid JSON")
	}
}

// ---------------------------------------------------------------------------
// buildModelInfo
// ---------------------------------------------------------------------------

func TestBuildModelInfo_Basic(t *testing.T) {
	base := map[string]interface{}{
		"name":              "Test",
		"max_input_tokens":  1000.0,
		"max_output_tokens": 500.0,
	}
	mi := buildModelInfo("conn-1", "base-model", []string{"chat"}, map[string]interface{}{"key": "val"}, base, true, "")
	if mi.ID != "conn-1" {
		t.Fatalf("ID = %s", mi.ID)
	}
	if mi.Model != "base-model" {
		t.Fatalf("Model = %s", mi.Model)
	}
	if !mi.Enabled {
		t.Fatal("expected enabled")
	}
	if mi.MaxInputTokens != 1000 {
		t.Fatalf("MaxInputTokens = %d", mi.MaxInputTokens)
	}
	if mi.MaxOutputTokens != 500 {
		t.Fatalf("MaxOutputTokens = %d", mi.MaxOutputTokens)
	}
}

func TestBuildModelInfo_WithSuffix(t *testing.T) {
	base := map[string]interface{}{"name": "Model X"}
	mi := buildModelInfo("x-thinking-high", "x", nil, nil, base, false, "(Thinking: High)")
	if mi.Name != "Model X (Thinking: High)" {
		t.Fatalf("name = %q, want 'Model X (Thinking: High)'", mi.Name)
	}
}

func TestBuildModelInfo_EmptyModel(t *testing.T) {
	mi := buildModelInfo("conn-2", "", []string{"chat"}, nil, map[string]interface{}{"name": "X"}, false, "")
	if mi.Model != "" {
		t.Fatalf("expected empty Model, got %s", mi.Model)
	}
}

func TestBuildModelInfo_NameFallback(t *testing.T) {
	mi := buildModelInfo("conn-3", "", nil, nil, map[string]interface{}{}, true, "")
	if mi.Name != "conn-3" {
		t.Fatalf("name should fallback to connID, got %s", mi.Name)
	}
}

// ---------------------------------------------------------------------------
// mapRemoteModel — service filtering
// ---------------------------------------------------------------------------

func TestMapRemoteModel_SkipsOCR(t *testing.T) {
	m := mapRemoteModel(map[string]interface{}{
		"id": "general_basic", "name": "通用文字识别",
		"service": "ocr", "capabilities": []interface{}{"ocr", "pdf"},
	})
	if m != nil {
		t.Fatal("expected nil for OCR service model")
	}
}

func TestMapRemoteModel_SkipsFetch(t *testing.T) {
	m := mapRemoteModel(map[string]interface{}{
		"id": "fetch-html", "name": "网页抓取 (HTML)",
		"service": "fetch", "capabilities": []interface{}{"fetch"},
	})
	if m != nil {
		t.Fatal("expected nil for fetch service model")
	}
}

func TestMapRemoteModel_KeepsChat(t *testing.T) {
	m := mapRemoteModel(map[string]interface{}{
		"id": "deepseek-flash", "name": "DeepSeek Flash",
		"service": "chat", "capabilities": []interface{}{"chat", "streaming"},
	})
	if m == nil {
		t.Fatal("chat model should not be filtered")
	}
}

func TestMapRemoteModel_KeepsAudio(t *testing.T) {
	m := mapRemoteModel(map[string]interface{}{
		"id": "whisper-1", "name": "Whisper",
		"service": "audio", "capabilities": []interface{}{"audio_transcribe"},
	})
	if m == nil {
		t.Fatal("audio model should not be filtered")
	}
	caps := toStringSlice(m["capabilities"])
	if !containsCap(caps, "audio_transcribe") {
		t.Fatal("expected audio_transcribe preserved")
	}
	if !containsCap(caps, "audio") {
		t.Fatal("expected audio alias added for audio_transcribe")
	}
}

func TestMapRemoteModel_KeepsImage(t *testing.T) {
	m := mapRemoteModel(map[string]interface{}{
		"id": "image2.5", "name": "OpenAI Image 2.5",
		"service": "image", "capabilities": []interface{}{"image_generate", "image_edit"},
	})
	if m == nil {
		t.Fatal("image model should not be filtered")
	}
	caps := toStringSlice(m["capabilities"])
	if !containsCap(caps, "image_generate") {
		t.Fatal("expected image_generate preserved")
	}
	if !containsCap(caps, "image_generation") {
		t.Fatal("expected image_generation alias added for image_generate")
	}
}

// ---------------------------------------------------------------------------
// normalizeCapabilities
// ---------------------------------------------------------------------------

func TestNormalizeCapabilities_Empty(t *testing.T) {
	out := normalizeCapabilities(nil)
	if len(out) != 0 {
		t.Fatalf("expected empty, got %v", out)
	}
}

func TestNormalizeCapabilities_ImageGenerate(t *testing.T) {
	out := normalizeCapabilities([]string{"image_generate", "image_edit"})
	if !containsCap(out, "image_generate") || !containsCap(out, "image_generation") {
		t.Fatalf("expected both image_generate and image_generation, got %v", out)
	}
	if !containsCap(out, "image_edit") || !containsCap(out, "image_editing") {
		t.Fatalf("expected both image_edit and image_editing, got %v", out)
	}
}

func TestNormalizeCapabilities_ImageEdit(t *testing.T) {
	out := normalizeCapabilities([]string{"image_edit"})
	if !containsCap(out, "image_edit") || !containsCap(out, "image_editing") {
		t.Fatalf("expected both image_edit and image_editing, got %v", out)
	}
}

func TestNormalizeCapabilities_AudioTranscribe(t *testing.T) {
	out := normalizeCapabilities([]string{"audio_transcribe"})
	if !containsCap(out, "audio_transcribe") || !containsCap(out, "audio") {
		t.Fatalf("expected audio_transcribe + audio, got %v", out)
	}
}

func TestNormalizeCapabilities_AudioSpeech(t *testing.T) {
	out := normalizeCapabilities([]string{"audio_speech"})
	if !containsCap(out, "audio_speech") {
		t.Fatalf("expected audio_speech preserved, got %v", out)
	}
	if containsCap(out, "audio") {
		t.Fatalf("audio_speech should NOT produce audio alias (TTS ≠ STT), got %v", out)
	}
}

func TestNormalizeCapabilities_BothAudioNoDuplicate(t *testing.T) {
	out := normalizeCapabilities([]string{"audio_transcribe", "audio_speech"})
	count := 0
	for _, c := range out {
		if c == "audio" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected exactly 1 audio alias (from audio_transcribe only), got %d in %v", count, out)
	}
	if !containsCap(out, "audio_speech") {
		t.Fatalf("expected audio_speech preserved, got %v", out)
	}
}

func TestNormalizeCapabilities_ChatPassthrough(t *testing.T) {
	out := normalizeCapabilities([]string{"chat", "streaming", "tool_calls"})
	if len(out) != 3 {
		t.Fatalf("expected 3 caps unchanged, got %v", out)
	}
}

func containsCap(caps []string, target string) bool {
	for _, c := range caps {
		if c == target {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// can_disable / disable_with edge cases
// ---------------------------------------------------------------------------

func TestExpandReasoningVariants_CanDisableFalseOverridesSwitch(t *testing.T) {
	invalidateRemoteModelCache()
	remoteReasoningCache = make(map[string]interface{})

	// switch has both enabled+disabled, but can_disable=false → no disabled variant
	base := map[string]interface{}{
		"id":           "model-no-disable",
		"name":         "No Disable Model",
		"capabilities": []interface{}{"chat", "reasoning"},
		"reasoning": map[string]interface{}{
			"switch": map[string]interface{}{
				"param":  "thinking",
				"values": []interface{}{map[string]interface{}{"type": "enabled"}, map[string]interface{}{"type": "disabled"}},
			},
			"can_disable": false,
		},
	}

	variants := expandReasoningVariants(base)
	// can_disable=false → Case 4: only enabled variant
	if len(variants) != 1 {
		t.Fatalf("expected 1 variant (can_disable=false overrides switch), got %d", len(variants))
	}
	if !hasCap(variants[0].Capabilities, "reasoning") {
		t.Fatal("expected reasoning cap on the sole variant")
	}
}

func TestExpandReasoningVariants_CanDisableTrueEmptyDisableWith(t *testing.T) {
	invalidateRemoteModelCache()
	remoteReasoningCache = make(map[string]interface{})

	// can_disable=true but disable_with is empty → fallback to classifySwitchValues
	base := map[string]interface{}{
		"id":           "model-fallback",
		"name":         "Fallback Model",
		"capabilities": []interface{}{"chat", "reasoning"},
		"reasoning": map[string]interface{}{
			"switch": map[string]interface{}{
				"param":  "thinking",
				"values": []interface{}{map[string]interface{}{"type": "enabled"}, map[string]interface{}{"type": "disabled"}},
			},
			"can_disable":  true,
			"disable_with": []interface{}{},
		},
	}

	variants := expandReasoningVariants(base)
	// Case 3 (switch only, no effort): disabled + enabled = 2
	if len(variants) != 2 {
		t.Fatalf("expected 2 variants (fallback to switch inference), got %d", len(variants))
	}
	assertVariant(t, variants[0], "model-fallback", "", true, false)
	assertVariant(t, variants[1], "model-fallback-thinking", "model-fallback", false, true)

	// Verify disabled variant Options uses inferred switch value
	dOpts := variants[0].Options
	if dOpts == nil {
		t.Fatal("expected options on disabled variant")
	}
	dThinking, ok := dOpts["thinking"].(map[string]interface{})
	if !ok {
		t.Fatal("expected thinking map in fallback disabled options")
	}
	if dThinking["type"] != "disabled" {
		t.Fatalf("fallback disabled thinking type = %v, want disabled", dThinking["type"])
	}
}
