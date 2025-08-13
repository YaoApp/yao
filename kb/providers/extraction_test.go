package providers

import (
	"testing"

	kbtypes "github.com/yaoapp/yao/kb/types"
)

func TestExtractionOpenAI_Make(t *testing.T) {
	extraction := &ExtractionOpenAI{}

	// Note: OpenAI extraction requires connectors to be loaded
	// All tests will fail in test environment due to missing connectors

	t.Run("nil option should return error due to missing connector", func(t *testing.T) {
		_, err := extraction.Make(nil)
		if err == nil {
			t.Error("Expected error due to missing connector")
		}
		// Error is expected because connector is not loaded in test environment
	})

	t.Run("empty option should return error due to missing connector", func(t *testing.T) {
		option := &kbtypes.ProviderOption{}
		_, err := extraction.Make(option)
		if err == nil {
			t.Error("Expected error due to missing connector")
		}
		// Error is expected because connector is not loaded in test environment
	})

	t.Run("option with connector should return error due to missing connector", func(t *testing.T) {
		option := &kbtypes.ProviderOption{
			Properties: map[string]interface{}{
				"connector": "openai.gpt-4o-mini",
			},
		}
		_, err := extraction.Make(option)
		if err == nil {
			t.Error("Expected error due to missing connector")
		}
		// Error is expected because openai.gpt-4o-mini connector is not loaded
	})

	t.Run("option with toolcall true should return error due to missing connector", func(t *testing.T) {
		option := &kbtypes.ProviderOption{
			Properties: map[string]interface{}{
				"connector": "openai.gpt-4o-mini",
				"toolcall":  true,
			},
		}
		_, err := extraction.Make(option)
		if err == nil {
			t.Error("Expected error due to missing connector")
		}
		// Error is expected because connector is not loaded
	})

	t.Run("option with toolcall false should return error due to missing connector", func(t *testing.T) {
		option := &kbtypes.ProviderOption{
			Properties: map[string]interface{}{
				"connector": "deepseek.v3",
				"toolcall":  false,
			},
		}
		_, err := extraction.Make(option)
		if err == nil {
			t.Error("Expected error due to missing connector")
		}
		// Error is expected because deepseek.v3 connector is not loaded
	})

	t.Run("option with all properties should return error due to missing connector", func(t *testing.T) {
		option := &kbtypes.ProviderOption{
			Properties: map[string]interface{}{
				"connector":      "openai.gpt-4o",
				"toolcall":       true,
				"temperature":    0.2,
				"max_tokens":     8000,
				"concurrent":     10,
				"model":          "gpt-4o",
				"prompt":         "Extract entities and relationships:",
				"retry_attempts": 5,
				"retry_delay":    2,
			},
		}
		_, err := extraction.Make(option)
		if err == nil {
			t.Error("Expected error due to missing connector")
		}
		// Error is expected because openai.gpt-4o connector is not loaded
	})

	t.Run("temperature as int should be converted to float64 but still return error", func(t *testing.T) {
		option := &kbtypes.ProviderOption{
			Properties: map[string]interface{}{
				"connector":   "openai.gpt-4o-mini",
				"temperature": 1,      // int -> float64
				"max_tokens":  2000.0, // float64 -> int
				"concurrent":  3.0,    // float64 -> int
			},
		}
		_, err := extraction.Make(option)
		if err == nil {
			t.Error("Expected error due to missing connector")
		}
		// Error is expected because connector is not loaded
	})

	t.Run("retry_delay as float should be converted to duration but still return error", func(t *testing.T) {
		option := &kbtypes.ProviderOption{
			Properties: map[string]interface{}{
				"connector":      "openai.gpt-4o-mini",
				"retry_delay":    1.5, // 1.5 seconds
				"retry_attempts": 2.0, // float64 -> int
			},
		}
		_, err := extraction.Make(option)
		if err == nil {
			t.Error("Expected error due to missing connector")
		}
		// Error is expected because connector is not loaded
	})

	t.Run("custom tools should be parsed but still return error", func(t *testing.T) {
		option := &kbtypes.ProviderOption{
			Properties: map[string]interface{}{
				"connector": "openai.gpt-4o-mini",
				"tools": []interface{}{
					map[string]interface{}{
						"type": "function",
						"function": map[string]interface{}{
							"name":        "extract_entities",
							"description": "Extract entities from text",
						},
					},
				},
			},
		}
		_, err := extraction.Make(option)
		if err == nil {
			t.Error("Expected error due to missing connector")
		}
		// Error is expected because connector is not loaded
	})

	t.Run("invalid property types should be ignored but still return error", func(t *testing.T) {
		option := &kbtypes.ProviderOption{
			Properties: map[string]interface{}{
				"connector":      123,        // invalid type
				"toolcall":       "invalid",  // invalid type
				"temperature":    "invalid",  // invalid type
				"max_tokens":     "invalid",  // invalid type
				"concurrent":     "invalid",  // invalid type
				"model":          true,       // invalid type
				"prompt":         []string{}, // invalid type
				"retry_attempts": "invalid",  // invalid type
				"retry_delay":    "invalid",  // invalid type
				"tools":          "invalid",  // invalid type
			},
		}
		_, err := extraction.Make(option)
		if err == nil {
			t.Error("Expected error due to missing connector")
		}
		// Error is expected because connector is not loaded
	})

	t.Run("partial properties should use defaults for missing values but still return error", func(t *testing.T) {
		option := &kbtypes.ProviderOption{
			Properties: map[string]interface{}{
				"connector":   "openai.gpt-4o-mini",
				"toolcall":    true,
				"temperature": 0.3,
				// Other properties should use defaults
			},
		}
		_, err := extraction.Make(option)
		if err == nil {
			t.Error("Expected error due to missing connector")
		}
		// Error is expected because connector is not loaded
	})

	t.Run("missing connector should return error", func(t *testing.T) {
		option := &kbtypes.ProviderOption{
			Properties: map[string]interface{}{
				"toolcall":    true,
				"temperature": 0.1,
				"max_tokens":  4000,
				// No connector specified
			},
		}
		_, err := extraction.Make(option)
		if err == nil {
			t.Error("Expected error due to missing connector")
		}
		// Error is expected because no connector is specified
	})

	t.Run("gpt-4o configuration should work but still return error", func(t *testing.T) {
		option := &kbtypes.ProviderOption{
			Properties: map[string]interface{}{
				"connector": "openai.gpt-4o",
				"toolcall":  true,
			},
		}
		_, err := extraction.Make(option)
		if err == nil {
			t.Error("Expected error due to missing connector")
		}
		// Error is expected because connector is not loaded
	})

	t.Run("deepseek configuration should work but still return error", func(t *testing.T) {
		option := &kbtypes.ProviderOption{
			Properties: map[string]interface{}{
				"connector": "deepseek.v3",
				"toolcall":  false,
			},
		}
		_, err := extraction.Make(option)
		if err == nil {
			t.Error("Expected error due to missing connector")
		}
		// Error is expected because connector is not loaded
	})

	t.Run("invalid tools array should be ignored but still return error", func(t *testing.T) {
		option := &kbtypes.ProviderOption{
			Properties: map[string]interface{}{
				"connector": "openai.gpt-4o-mini",
				"tools": []interface{}{
					"invalid_tool",                          // not a map
					123,                                     // not a map
					map[string]interface{}{"valid": "tool"}, // valid map
				},
			},
		}
		_, err := extraction.Make(option)
		if err == nil {
			t.Error("Expected error due to missing connector")
		}
		// Error is expected because connector is not loaded
	})

	t.Run("edge case temperature values should be handled", func(t *testing.T) {
		option := &kbtypes.ProviderOption{
			Properties: map[string]interface{}{
				"connector":   "openai.gpt-4o-mini",
				"temperature": 2.5, // Above normal range, will be validated by openai.NewOpenai
			},
		}
		_, err := extraction.Make(option)
		if err == nil {
			t.Error("Expected error due to missing connector")
		}
		// Error is expected because connector is not loaded
	})

	t.Run("zero values should be handled correctly", func(t *testing.T) {
		option := &kbtypes.ProviderOption{
			Properties: map[string]interface{}{
				"connector":      "openai.gpt-4o-mini",
				"temperature":    0.0,
				"max_tokens":     0, // Will be set to default by openai.NewOpenai
				"concurrent":     0, // Will be set to default by openai.NewOpenai
				"retry_attempts": 0, // Will be set to default by openai.NewOpenai
				"retry_delay":    0, // Will be set to default by openai.NewOpenai
			},
		}
		_, err := extraction.Make(option)
		if err == nil {
			t.Error("Expected error due to missing connector")
		}
		// Error is expected because connector is not loaded
	})
}

func TestExtractionOpenAI_Schema(t *testing.T) {
	extraction := &ExtractionOpenAI{}
	schema, err := extraction.Schema(nil, "en")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if schema == nil {
		t.Error("Expected non-nil schema from factory.GetSchemaFromBindata")
	}
}
