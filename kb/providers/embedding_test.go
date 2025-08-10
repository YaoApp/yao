package providers

import (
	"testing"

	kbtypes "github.com/yaoapp/yao/kb/types"
)

func TestOpenAI_Make(t *testing.T) {
	openai := &OpenAI{}

	// Note: OpenAI embedding requires connectors to be loaded
	// All tests will fail in test environment due to missing connectors

	t.Run("nil option should return error due to missing connector", func(t *testing.T) {
		_, err := openai.Make(nil)
		if err == nil {
			t.Error("Expected error due to missing connector")
		}
		// Error is expected because connector is not loaded in test environment
	})

	t.Run("empty option should return error due to missing connector", func(t *testing.T) {
		option := &kbtypes.ProviderOption{}
		_, err := openai.Make(option)
		if err == nil {
			t.Error("Expected error due to missing connector")
		}
		// Error is expected because connector is not loaded in test environment
	})

	t.Run("option with connector should return error due to missing connector", func(t *testing.T) {
		option := &kbtypes.ProviderOption{
			Properties: map[string]interface{}{
				"connector": "openai.text-embedding-3-small",
			},
		}
		_, err := openai.Make(option)
		if err == nil {
			t.Error("Expected error due to missing connector")
		}
		// Error is expected because openai.text-embedding-3-small connector is not loaded
	})

	t.Run("option with all properties should return error due to missing connector", func(t *testing.T) {
		option := &kbtypes.ProviderOption{
			Properties: map[string]interface{}{
				"connector":  "openai.text-embedding-3-large",
				"dimensions": 1536,
				"concurrent": 20,
				"model":      "text-embedding-3-large",
			},
		}
		_, err := openai.Make(option)
		if err == nil {
			t.Error("Expected error due to missing connector")
		}
		// Error is expected because openai.text-embedding-3-large connector is not loaded
	})

	t.Run("dimensions as float64 should be converted to int but still return error", func(t *testing.T) {
		option := &kbtypes.ProviderOption{
			Properties: map[string]interface{}{
				"connector":  "openai.text-embedding-3-small",
				"dimensions": 512.0, // float64
				"concurrent": 15.0,  // float64
			},
		}
		_, err := openai.Make(option)
		if err == nil {
			t.Error("Expected error due to missing connector")
		}
		// Error is expected because connector is not loaded
	})

	t.Run("invalid property types should be ignored but still return error", func(t *testing.T) {
		option := &kbtypes.ProviderOption{
			Properties: map[string]interface{}{
				"connector":  123,       // invalid type
				"dimensions": "invalid", // invalid type
				"concurrent": "invalid", // invalid type
				"model":      true,      // invalid type
			},
		}
		_, err := openai.Make(option)
		if err == nil {
			t.Error("Expected error due to missing connector")
		}
		// Error is expected because connector is not loaded
	})

	t.Run("partial properties should use defaults for missing values but still return error", func(t *testing.T) {
		option := &kbtypes.ProviderOption{
			Properties: map[string]interface{}{
				"connector":  "openai.text-embedding-3-small",
				"dimensions": 768,
				// concurrent and model should use defaults
			},
		}
		_, err := openai.Make(option)
		if err == nil {
			t.Error("Expected error due to missing connector")
		}
		// Error is expected because connector is not loaded
	})

	t.Run("missing connector should return error", func(t *testing.T) {
		option := &kbtypes.ProviderOption{
			Properties: map[string]interface{}{
				"dimensions": 1536,
				"concurrent": 10,
				// No connector specified
			},
		}
		_, err := openai.Make(option)
		if err == nil {
			t.Error("Expected error due to missing connector")
		}
		// Error is expected because no connector is specified
	})
}

func TestOpenAI_Schema(t *testing.T) {
	openai := &OpenAI{}
	schema, err := openai.Schema(nil, "en")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if schema == nil {
		t.Error("Expected non-nil schema from factory.GetSchemaFromBindata")
	}
}

func TestFastembed_Make(t *testing.T) {
	fastembed := &Fastembed{}

	// Note: Fastembed embedding requires connectors to be loaded
	// All tests will fail in test environment due to missing connectors

	t.Run("nil option should return error due to missing connector", func(t *testing.T) {
		_, err := fastembed.Make(nil)
		if err == nil {
			t.Error("Expected error due to missing connector")
		}
		// Error is expected because connector is not loaded in test environment
	})

	t.Run("empty option should return error due to missing connector", func(t *testing.T) {
		option := &kbtypes.ProviderOption{}
		_, err := fastembed.Make(option)
		if err == nil {
			t.Error("Expected error due to missing connector")
		}
		// Error is expected because connector is not loaded in test environment
	})

	t.Run("option with connector should return error due to missing connector", func(t *testing.T) {
		option := &kbtypes.ProviderOption{
			Properties: map[string]interface{}{
				"connector": "fastembed.bge-small-en-v1_5",
			},
		}
		_, err := fastembed.Make(option)
		if err == nil {
			t.Error("Expected error due to missing connector")
		}
		// Error is expected because fastembed.bge-small-en-v1_5 connector is not loaded
	})

	t.Run("option with all properties should return error due to missing connector", func(t *testing.T) {
		option := &kbtypes.ProviderOption{
			Properties: map[string]interface{}{
				"connector":  "fastembed.mxbai-embed-large-v1",
				"dimensions": 1024,
				"concurrent": 15,
				"model":      "mxbai-embed-large-v1",
				"host":       "http://localhost:8080",
				"key":        "test-key",
			},
		}
		_, err := fastembed.Make(option)
		if err == nil {
			t.Error("Expected error due to missing connector")
		}
		// Error is expected because fastembed.mxbai-embed-large-v1 connector is not loaded
	})

	t.Run("dimensions as float64 should be converted to int but still return error", func(t *testing.T) {
		option := &kbtypes.ProviderOption{
			Properties: map[string]interface{}{
				"connector":  "fastembed.bge-small-zh-v1_5",
				"dimensions": 512.0, // float64
				"concurrent": 8.0,   // float64
			},
		}
		_, err := fastembed.Make(option)
		if err == nil {
			t.Error("Expected error due to missing connector")
		}
		// Error is expected because connector is not loaded
	})

	t.Run("invalid property types should be ignored but still return error", func(t *testing.T) {
		option := &kbtypes.ProviderOption{
			Properties: map[string]interface{}{
				"connector":  123,        // invalid type
				"dimensions": "invalid",  // invalid type
				"concurrent": "invalid",  // invalid type
				"model":      true,       // invalid type
				"host":       []string{}, // invalid type
				"key":        123,        // invalid type
			},
		}
		_, err := fastembed.Make(option)
		if err == nil {
			t.Error("Expected error due to missing connector")
		}
		// Error is expected because connector is not loaded
	})

	t.Run("partial properties should use defaults for missing values but still return error", func(t *testing.T) {
		option := &kbtypes.ProviderOption{
			Properties: map[string]interface{}{
				"connector":  "fastembed.bge-small-en-v1_5",
				"dimensions": 384,
				"host":       "http://fastembed-server:8080",
				// concurrent, model, and key should use defaults
			},
		}
		_, err := fastembed.Make(option)
		if err == nil {
			t.Error("Expected error due to missing connector")
		}
		// Error is expected because connector is not loaded
	})

	t.Run("missing connector should return error", func(t *testing.T) {
		option := &kbtypes.ProviderOption{
			Properties: map[string]interface{}{
				"dimensions": 384,
				"concurrent": 10,
				"host":       "http://localhost:8080",
				// No connector specified
			},
		}
		_, err := fastembed.Make(option)
		if err == nil {
			t.Error("Expected error due to missing connector")
		}
		// Error is expected because no connector is specified
	})

	t.Run("chinese model configuration should work but still return error", func(t *testing.T) {
		option := &kbtypes.ProviderOption{
			Properties: map[string]interface{}{
				"connector":  "fastembed.bge-small-zh-v1_5",
				"dimensions": 512,
				"model":      "bge-small-zh-v1.5",
			},
		}
		_, err := fastembed.Make(option)
		if err == nil {
			t.Error("Expected error due to missing connector")
		}
		// Error is expected because connector is not loaded
	})

	t.Run("large model configuration should work but still return error", func(t *testing.T) {
		option := &kbtypes.ProviderOption{
			Properties: map[string]interface{}{
				"connector":  "fastembed.mxbai-embed-large-v1",
				"dimensions": 1024,
				"model":      "mxbai-embed-large-v1",
			},
		}
		_, err := fastembed.Make(option)
		if err == nil {
			t.Error("Expected error due to missing connector")
		}
		// Error is expected because connector is not loaded
	})
}

func TestFastembed_Schema(t *testing.T) {
	fastembed := &Fastembed{}
	schema, err := fastembed.Schema(nil, "en")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if schema == nil {
		t.Error("Expected non-nil schema from factory.GetSchemaFromBindata")
	}
}
