package providers

import (
	"testing"

	"github.com/yaoapp/gou/graphrag/types"
	kbtypes "github.com/yaoapp/yao/kb/types"
)

func TestStructured_Options(t *testing.T) {
	s := &Structured{}

	t.Run("nil option should return default values", func(t *testing.T) {
		options, err := s.Options(nil)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if options == nil {
			t.Fatal("Expected options, got nil")
		}

		// Check default values
		expected := &types.ChunkingOptions{
			Size:           300,
			Overlap:        20,
			MaxDepth:       3,
			SizeMultiplier: 3,
			MaxConcurrent:  1,
			Separator:      "",
			EnableDebug:    false,
		}

		if options.Size != expected.Size {
			t.Errorf("Expected Size %d, got %d", expected.Size, options.Size)
		}
		if options.Overlap != expected.Overlap {
			t.Errorf("Expected Overlap %d, got %d", expected.Overlap, options.Overlap)
		}
		if options.MaxDepth != expected.MaxDepth {
			t.Errorf("Expected MaxDepth %d, got %d", expected.MaxDepth, options.MaxDepth)
		}
		if options.SizeMultiplier != expected.SizeMultiplier {
			t.Errorf("Expected SizeMultiplier %d, got %d", expected.SizeMultiplier, options.SizeMultiplier)
		}
		if options.MaxConcurrent != expected.MaxConcurrent {
			t.Errorf("Expected MaxConcurrent %d, got %d", expected.MaxConcurrent, options.MaxConcurrent)
		}
	})

	t.Run("empty properties should return default values", func(t *testing.T) {
		option := &kbtypes.ProviderOption{
			Label:       "test",
			Value:       "test",
			Description: "test",
			Properties:  map[string]interface{}{},
		}

		options, err := s.Options(option)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		// Should still have default values
		if options.Size != 300 {
			t.Errorf("Expected Size 300, got %d", options.Size)
		}
	})

	t.Run("custom properties with int values", func(t *testing.T) {
		option := &kbtypes.ProviderOption{
			Properties: map[string]interface{}{
				"size":            500,
				"overlap":         30,
				"max_depth":       5,
				"size_multiplier": 4,
				"max_concurrent":  15,
			},
		}

		options, err := s.Options(option)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if options.Size != 500 {
			t.Errorf("Expected Size 500, got %d", options.Size)
		}
		if options.Overlap != 30 {
			t.Errorf("Expected Overlap 30, got %d", options.Overlap)
		}
		if options.MaxDepth != 5 {
			t.Errorf("Expected MaxDepth 5, got %d", options.MaxDepth)
		}
		if options.SizeMultiplier != 4 {
			t.Errorf("Expected SizeMultiplier 4, got %d", options.SizeMultiplier)
		}
		if options.MaxConcurrent != 15 {
			t.Errorf("Expected MaxConcurrent 15, got %d", options.MaxConcurrent)
		}
	})

	t.Run("custom properties with float64 values", func(t *testing.T) {
		option := &kbtypes.ProviderOption{
			Properties: map[string]interface{}{
				"size":            500.0,
				"overlap":         30.0,
				"max_depth":       5.0,
				"size_multiplier": 4.0,
				"max_concurrent":  15.0,
			},
		}

		options, err := s.Options(option)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if options.Size != 500 {
			t.Errorf("Expected Size 500, got %d", options.Size)
		}
		if options.Overlap != 30 {
			t.Errorf("Expected Overlap 30, got %d", options.Overlap)
		}
		if options.MaxDepth != 5 {
			t.Errorf("Expected MaxDepth 5, got %d", options.MaxDepth)
		}
		if options.SizeMultiplier != 4 {
			t.Errorf("Expected SizeMultiplier 4, got %d", options.SizeMultiplier)
		}
		if options.MaxConcurrent != 15 {
			t.Errorf("Expected MaxConcurrent 15, got %d", options.MaxConcurrent)
		}
	})

	t.Run("partial properties should use defaults for missing values", func(t *testing.T) {
		option := &kbtypes.ProviderOption{
			Properties: map[string]interface{}{
				"size":    800,
				"overlap": 100,
				// max_depth, size_multiplier, max_concurrent not provided
			},
		}

		options, err := s.Options(option)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if options.Size != 800 {
			t.Errorf("Expected Size 800, got %d", options.Size)
		}
		if options.Overlap != 100 {
			t.Errorf("Expected Overlap 100, got %d", options.Overlap)
		}
		// Should use defaults for missing values
		if options.MaxDepth != 3 {
			t.Errorf("Expected MaxDepth 3 (default), got %d", options.MaxDepth)
		}
		if options.SizeMultiplier != 3 {
			t.Errorf("Expected SizeMultiplier 3 (default), got %d", options.SizeMultiplier)
		}
		if options.MaxConcurrent != 1 {
			t.Errorf("Expected MaxConcurrent 1 (default), got %d", options.MaxConcurrent)
		}
	})

	t.Run("invalid type values should be ignored", func(t *testing.T) {
		option := &kbtypes.ProviderOption{
			Properties: map[string]interface{}{
				"size":    "invalid", // string instead of int/float
				"overlap": true,      // bool instead of int/float
			},
		}

		options, err := s.Options(option)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		// Should use defaults when invalid types are provided
		if options.Size != 300 {
			t.Errorf("Expected Size 300 (default), got %d", options.Size)
		}
		if options.Overlap != 20 {
			t.Errorf("Expected Overlap 20 (default), got %d", options.Overlap)
		}
	})
}

func TestSemantic_Options(t *testing.T) {
	s := &Semantic{}

	t.Run("nil option should return default values", func(t *testing.T) {
		options, err := s.Options(nil)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if options == nil {
			t.Fatal("Expected options, got nil")
		}

		// Check default values
		if options.Size != 300 {
			t.Errorf("Expected Size 300, got %d", options.Size)
		}
		if options.Overlap != 50 {
			t.Errorf("Expected Overlap 50, got %d", options.Overlap)
		}
		if options.MaxDepth != 3 {
			t.Errorf("Expected MaxDepth 3, got %d", options.MaxDepth)
		}
		if options.SizeMultiplier != 3 {
			t.Errorf("Expected SizeMultiplier 3, got %d", options.SizeMultiplier)
		}
		if options.MaxConcurrent != 1 {
			t.Errorf("Expected MaxConcurrent 1, got %d", options.MaxConcurrent)
		}

		// Check semantic options
		if options.SemanticOptions == nil {
			t.Fatal("Expected SemanticOptions, got nil")
		}
		if options.SemanticOptions.ContextSize != 1800 {
			t.Errorf("Expected ContextSize 1800, got %d", options.SemanticOptions.ContextSize)
		}
		if options.SemanticOptions.MaxRetry != 3 {
			t.Errorf("Expected MaxRetry 3, got %d", options.SemanticOptions.MaxRetry)
		}
		if options.SemanticOptions.MaxConcurrent != 1 {
			t.Errorf("Expected MaxConcurrent 1, got %d", options.SemanticOptions.MaxConcurrent)
		}
		if options.SemanticOptions.Toolcall != true {
			t.Errorf("Expected Toolcall true, got %v", options.SemanticOptions.Toolcall)
		}
		if options.SemanticOptions.Connector != "openai.gpt-4o-mini" {
			t.Errorf("Expected Connector 'openai.gpt-4o-mini', got '%s'", options.SemanticOptions.Connector)
		}
	})

	t.Run("basic chunking properties", func(t *testing.T) {
		option := &kbtypes.ProviderOption{
			Properties: map[string]interface{}{
				"size":            600,
				"overlap":         100,
				"max_depth":       4,
				"size_multiplier": 5,
				"max_concurrent":  20,
			},
		}

		options, err := s.Options(option)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if options.Size != 600 {
			t.Errorf("Expected Size 600, got %d", options.Size)
		}
		if options.Overlap != 100 {
			t.Errorf("Expected Overlap 100, got %d", options.Overlap)
		}
		// Context size should be updated based on size
		if options.SemanticOptions.ContextSize != 3600 { // 600 * 6
			t.Errorf("Expected ContextSize 3600, got %d", options.SemanticOptions.ContextSize)
		}
	})

	t.Run("semantic specific properties", func(t *testing.T) {
		option := &kbtypes.ProviderOption{
			Properties: map[string]interface{}{
				"semantic": map[string]interface{}{
					"connector":               "openai.gpt-4o-mini",
					"toolcall":                true,
					"context_size":            2400,
					"options":                 `{"temperature": 0.7}`,
					"prompt":                  "Custom system prompt",
					"max_retry":               5,
					"semantic_max_concurrent": 15,
				},
			},
		}

		options, err := s.Options(option)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if options.SemanticOptions.Connector != "openai.gpt-4o-mini" {
			t.Errorf("Expected Connector 'openai.gpt-4o-mini', got '%s'", options.SemanticOptions.Connector)
		}
		if options.SemanticOptions.Toolcall != true {
			t.Errorf("Expected Toolcall true, got %v", options.SemanticOptions.Toolcall)
		}
		if options.SemanticOptions.ContextSize != 2400 {
			t.Errorf("Expected ContextSize 2400, got %d", options.SemanticOptions.ContextSize)
		}
		if options.SemanticOptions.Options != `{"temperature": 0.7}` {
			t.Errorf("Expected Options '{\"temperature\": 0.7}', got '%s'", options.SemanticOptions.Options)
		}
		if options.SemanticOptions.Prompt != "Custom system prompt" {
			t.Errorf("Expected Prompt 'Custom system prompt', got '%s'", options.SemanticOptions.Prompt)
		}
		if options.SemanticOptions.MaxRetry != 5 {
			t.Errorf("Expected MaxRetry 5, got %d", options.SemanticOptions.MaxRetry)
		}
		if options.SemanticOptions.MaxConcurrent != 15 {
			t.Errorf("Expected MaxConcurrent 15, got %d", options.SemanticOptions.MaxConcurrent)
		}
	})

	t.Run("context size auto-calculation", func(t *testing.T) {
		option := &kbtypes.ProviderOption{
			Properties: map[string]interface{}{
				"size": 400, // Should result in context_size = 400 * 6 = 2400
			},
		}

		options, err := s.Options(option)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if options.Size != 400 {
			t.Errorf("Expected Size 400, got %d", options.Size)
		}
		if options.SemanticOptions.ContextSize != 2400 {
			t.Errorf("Expected ContextSize 2400 (auto-calculated), got %d", options.SemanticOptions.ContextSize)
		}
	})

	t.Run("explicit context size overrides auto-calculation", func(t *testing.T) {
		option := &kbtypes.ProviderOption{
			Properties: map[string]interface{}{
				"size": 400, // Would auto-calculate to 2400
				"semantic": map[string]interface{}{
					"context_size": 3000, // Explicit override
				},
			},
		}

		options, err := s.Options(option)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if options.SemanticOptions.ContextSize != 3000 {
			t.Errorf("Expected ContextSize 3000 (explicit), got %d", options.SemanticOptions.ContextSize)
		}
	})

	t.Run("float64 values for semantic properties", func(t *testing.T) {
		option := &kbtypes.ProviderOption{
			Properties: map[string]interface{}{
				"size": 500.0,
				"semantic": map[string]interface{}{
					"context_size":            3000.0,
					"max_retry":               4.0,
					"semantic_max_concurrent": 12.0,
				},
			},
		}

		options, err := s.Options(option)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if options.Size != 500 {
			t.Errorf("Expected Size 500, got %d", options.Size)
		}
		if options.SemanticOptions.ContextSize != 3000 {
			t.Errorf("Expected ContextSize 3000, got %d", options.SemanticOptions.ContextSize)
		}
		if options.SemanticOptions.MaxRetry != 4 {
			t.Errorf("Expected MaxRetry 4, got %d", options.SemanticOptions.MaxRetry)
		}
		if options.SemanticOptions.MaxConcurrent != 12 {
			t.Errorf("Expected MaxConcurrent 12, got %d", options.SemanticOptions.MaxConcurrent)
		}
	})

	t.Run("mixed valid and invalid properties", func(t *testing.T) {
		option := &kbtypes.ProviderOption{
			Properties: map[string]interface{}{
				"size": 500, // valid
				"semantic": map[string]interface{}{
					"connector": 123,       // invalid type for string
					"toolcall":  "invalid", // invalid type for bool
					"max_retry": 3,         // valid
				},
			},
		}

		options, err := s.Options(option)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		// Valid properties should be set
		if options.Size != 500 {
			t.Errorf("Expected Size 500, got %d", options.Size)
		}
		if options.SemanticOptions.MaxRetry != 3 {
			t.Errorf("Expected MaxRetry 3, got %d", options.SemanticOptions.MaxRetry)
		}

		// Invalid properties should use defaults
		if options.SemanticOptions.Connector != "openai.gpt-4o-mini" {
			t.Errorf("Expected Connector 'openai.gpt-4o-mini' (default), got '%s'", options.SemanticOptions.Connector)
		}
		if options.SemanticOptions.Toolcall != true {
			t.Errorf("Expected Toolcall true (default), got %v", options.SemanticOptions.Toolcall)
		}
	})
}

func TestStructured_Schema(t *testing.T) {
	s := &Structured{}
	schema, err := s.Schema(nil, "en")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if schema == nil {
		t.Error("Expected non-nil schema from factory.GetSchemaFromBindata")
	}
}

func TestSemantic_Schema(t *testing.T) {
	s := &Semantic{}
	schema, err := s.Schema(nil, "en")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if schema == nil {
		t.Error("Expected non-nil schema from factory.GetSchemaFromBindata")
	}
}
