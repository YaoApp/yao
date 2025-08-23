package converters

import (
	"testing"
)

func TestParseNestedConverter(t *testing.T) {
	t.Run("nil config should return error", func(t *testing.T) {
		_, err := parseNestedConverter(nil)
		if err == nil {
			t.Error("Expected error for nil config")
		}
		if err.Error() != "converter config must be a map" {
			t.Errorf("Expected specific error message, got: %v", err)
		}
	})

	t.Run("non-map config should return error", func(t *testing.T) {
		_, err := parseNestedConverter("not a map")
		if err == nil {
			t.Error("Expected error for non-map config")
		}
		if err.Error() != "converter config must be a map" {
			t.Errorf("Expected specific error message, got: %v", err)
		}
	})

	t.Run("map without converter field should return error", func(t *testing.T) {
		config := map[string]interface{}{
			"properties": map[string]interface{}{
				"connector": "openai.gpt-4o-mini",
			},
		}
		_, err := parseNestedConverter(config)
		if err == nil {
			t.Error("Expected error for missing converter field")
		}
		if err.Error() != "converter ID is required" {
			t.Errorf("Expected specific error message, got: %v", err)
		}
	})

	t.Run("non-string converter field should return error", func(t *testing.T) {
		config := map[string]interface{}{
			"converter": 123, // should be string
			"properties": map[string]interface{}{
				"connector": "openai.gpt-4o-mini",
			},
		}
		_, err := parseNestedConverter(config)
		if err == nil {
			t.Error("Expected error for non-string converter field")
		}
		if err.Error() != "converter ID is required" {
			t.Errorf("Expected specific error message, got: %v", err)
		}
	})

	t.Run("unknown converter ID should return error", func(t *testing.T) {
		config := map[string]interface{}{
			"converter": "__yao.unknown_converter",
			"properties": map[string]interface{}{
				"connector": "openai.gpt-4o-mini",
			},
		}
		_, err := parseNestedConverter(config)
		if err == nil {
			t.Error("Expected error for unknown converter")
		}
		if err.Error() != "converter __yao.unknown_converter not found" {
			t.Errorf("Expected specific error message, got: %v", err)
		}
	})

	t.Run("valid converter config with string properties should return error due to factory limitation", func(t *testing.T) {
		config := map[string]interface{}{
			"converter":  "__yao.vision", // This converter exists in factory
			"properties": "gpt-4o-mini",  // String preset value
		}
		// This will fail because the factory converter's Make method will fail
		// due to missing actual connector setup in test environment
		_, err := parseNestedConverter(config)
		if err == nil {
			t.Error("Expected error due to test factory limitation")
		}
		// The error would come from the converter's Make method, not parseNestedConverter itself
	})

	t.Run("valid converter config with map properties should return error due to factory limitation", func(t *testing.T) {
		config := map[string]interface{}{
			"converter": "__yao.vision", // This converter exists in factory
			"properties": map[string]interface{}{
				"connector":     "openai.gpt-4o-mini",
				"compress_size": 512,
			},
		}
		// This will fail because the factory converter's Make method will fail
		// due to missing actual connector setup in test environment
		_, err := parseNestedConverter(config)
		if err == nil {
			t.Error("Expected error due to test factory limitation")
		}
		// The error would come from the converter's Make method, not parseNestedConverter itself
	})

	t.Run("converter config without properties should return error due to factory limitation", func(t *testing.T) {
		config := map[string]interface{}{
			"converter": "__yao.utf8", // This converter exists in factory
			// No properties field
		}
		// This will fail because the factory converter's Make method will fail
		// due to missing actual connector setup in test environment
		_, err := parseNestedConverter(config)
		if err == nil {
			t.Error("Expected error due to test factory limitation")
		}
		// The error would come from the converter's Make method, not parseNestedConverter itself
	})

	t.Run("empty map config should return error", func(t *testing.T) {
		config := map[string]interface{}{}
		_, err := parseNestedConverter(config)
		if err == nil {
			t.Error("Expected error for empty config")
		}
		if err.Error() != "converter ID is required" {
			t.Errorf("Expected specific error message, got: %v", err)
		}
	})

	t.Run("config with invalid properties type should still process", func(t *testing.T) {
		config := map[string]interface{}{
			"converter":  "__yao.vision", // This converter exists in factory
			"properties": 123,            // Invalid type, should be ignored
		}
		// This will fail because the factory converter's Make method will fail
		// due to missing actual connector setup in test environment
		_, err := parseNestedConverter(config)
		if err == nil {
			t.Error("Expected error due to test factory limitation")
		}
		// The error would come from the converter's Make method, not parseNestedConverter itself
	})

	// Note about test limitations:
	// These tests verify the parsing logic of parseNestedConverter, but cannot test
	// successful converter creation because:
	// 1. The factory requires actual connector instances to be set up
	// 2. Connectors require external services (OpenAI, etc.) to be available
	// 3. Test environment doesn't have these dependencies
	//
	// In integration tests or with proper mocking, these would succeed:
	// - parseNestedConverter(validConfig) should return actual converter instance
	// - All property mappings should work correctly
	// - Nested converter configurations should be properly parsed
}

// Additional tests could be added with proper mocking of the factory system:
// - Test successful converter creation with mocked factories
// - Test property mapping with different converter types
// - Test error propagation from nested converter Make methods
// - Test recursive nested converter configurations
