package llmprovider

import (
	"encoding/json"
	"testing"
)

func TestExtractOptionsFromSetting_ReasoningEffort(t *testing.T) {
	setting := map[string]interface{}{
		"host":             "https://api.deepseek.com",
		"key":              "sk-xxx",
		"model":            "deepseek-v4-flash",
		"reasoning_effort": "high",
	}
	opts := extractOptionsFromSetting(setting)
	if opts == nil {
		t.Fatal("expected non-nil opts")
	}
	if opts["reasoning_effort"] != "high" {
		t.Errorf("reasoning_effort = %v, want 'high'", opts["reasoning_effort"])
	}
}

func TestExtractOptionsFromSetting_Thinking(t *testing.T) {
	setting := map[string]interface{}{
		"thinking": map[string]interface{}{"type": "enabled", "budget_tokens": 8192},
	}
	opts := extractOptionsFromSetting(setting)
	if opts == nil {
		t.Fatal("expected non-nil opts")
	}
	thinkMap, ok := opts["thinking"].(map[string]interface{})
	if !ok {
		t.Fatalf("thinking type = %T, want map", opts["thinking"])
	}
	if thinkMap["type"] != "enabled" {
		t.Errorf("thinking.type = %v, want 'enabled'", thinkMap["type"])
	}
}

func TestExtractOptionsFromSetting_NoRelevantKeys(t *testing.T) {
	setting := map[string]interface{}{
		"host":  "https://api.openai.com",
		"key":   "sk-xxx",
		"model": "gpt-4o",
	}
	opts := extractOptionsFromSetting(setting)
	if opts != nil {
		t.Errorf("expected nil for setting without reasoning keys, got %v", opts)
	}
}

func TestExtractOptionsFromSetting_NilSetting(t *testing.T) {
	opts := extractOptionsFromSetting(nil)
	if opts != nil {
		t.Error("expected nil for nil setting")
	}
}

func TestExtractOptionsFromSetting_ReasoningEffortNonString(t *testing.T) {
	setting := map[string]interface{}{
		"reasoning_effort": 42,
	}
	opts := extractOptionsFromSetting(setting)
	if opts != nil {
		t.Errorf("expected nil when reasoning_effort is non-string, got %v", opts)
	}
}

func TestMarshalModelDSL_IncludesMetadata(t *testing.T) {
	p := &Provider{
		Type:   "openai",
		APIURL: "https://api.deepseek.com",
		APIKey: "sk-test",
	}
	meta := map[string]interface{}{
		"model_name":        "DeepSeek V4 Flash",
		"model_family":      "deepseek-v4-flash",
		"reasoning_efforts": []string{"none", "high"},
		"reasoning_effort":  "high",
	}
	m := &ModelInfo{
		ID:           "deepseek-v4-flash-thinking",
		Name:         "DeepSeek V4 Flash Thinking",
		Capabilities: []string{"streaming", "tool_calls", "reasoning"},
		Options: map[string]interface{}{
			"thinking": map[string]interface{}{"type": "enabled"},
		},
		Metadata: meta,
	}
	data, err := marshalModelDSL(p, m)
	if err != nil {
		t.Fatal(err)
	}

	var dsl map[string]interface{}
	if err := json.Unmarshal(data, &dsl); err != nil {
		t.Fatal(err)
	}

	got, ok := dsl["metadata"].(map[string]interface{})
	if !ok {
		t.Fatal("metadata missing or wrong type in DSL output")
	}
	if got["model_name"] != "DeepSeek V4 Flash" {
		t.Errorf("model_name = %v", got["model_name"])
	}
	if got["model_family"] != "deepseek-v4-flash" {
		t.Errorf("model_family = %v", got["model_family"])
	}
	if got["reasoning_effort"] != "high" {
		t.Errorf("reasoning_effort = %v", got["reasoning_effort"])
	}
}

func TestMarshalModelDSL_NoMetadata(t *testing.T) {
	p := &Provider{
		Type:   "openai",
		APIURL: "https://api.openai.com",
		APIKey: "sk-test",
	}
	m := &ModelInfo{
		ID:           "gpt-4o",
		Name:         "GPT-4o",
		Capabilities: []string{"streaming"},
	}
	data, err := marshalModelDSL(p, m)
	if err != nil {
		t.Fatal(err)
	}

	var dsl map[string]interface{}
	if err := json.Unmarshal(data, &dsl); err != nil {
		t.Fatal(err)
	}

	if _, exists := dsl["metadata"]; exists {
		t.Error("metadata should be absent when ModelInfo.Metadata is nil")
	}
}
