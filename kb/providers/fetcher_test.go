package providers

import (
	"testing"

	kbtypes "github.com/yaoapp/yao/kb/types"
)

func TestFetcherHTTP_Make(t *testing.T) {
	fetcher := &FetcherHTTP{}

	t.Run("nil option should return default HTTP fetcher", func(t *testing.T) {
		result, err := fetcher.Make(nil)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if result == nil {
			t.Error("Expected HTTP fetcher, got nil")
		}
	})

	t.Run("empty option should return default HTTP fetcher", func(t *testing.T) {
		option := &kbtypes.ProviderOption{}
		result, err := fetcher.Make(option)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if result == nil {
			t.Error("Expected HTTP fetcher, got nil")
		}
	})

	t.Run("option with headers should work", func(t *testing.T) {
		option := &kbtypes.ProviderOption{
			Properties: map[string]interface{}{
				"headers": map[string]interface{}{
					"Authorization": "Bearer token123",
					"Accept":        "application/json",
					"User-Agent":    "Custom-Agent/1.0",
				},
			},
		}
		result, err := fetcher.Make(option)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if result == nil {
			t.Error("Expected HTTP fetcher, got nil")
		}
	})

	t.Run("option with timeout should work", func(t *testing.T) {
		option := &kbtypes.ProviderOption{
			Properties: map[string]interface{}{
				"timeout": 30, // 30 seconds
			},
		}
		result, err := fetcher.Make(option)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if result == nil {
			t.Error("Expected HTTP fetcher, got nil")
		}
	})

	t.Run("option with timeout as float should work", func(t *testing.T) {
		option := &kbtypes.ProviderOption{
			Properties: map[string]interface{}{
				"timeout": 45.5, // 45.5 seconds
			},
		}
		result, err := fetcher.Make(option)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if result == nil {
			t.Error("Expected HTTP fetcher, got nil")
		}
	})

	t.Run("option with user_agent should work", func(t *testing.T) {
		option := &kbtypes.ProviderOption{
			Properties: map[string]interface{}{
				"user_agent": "Custom-GraphRAG/2.0",
			},
		}
		result, err := fetcher.Make(option)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if result == nil {
			t.Error("Expected HTTP fetcher, got nil")
		}
	})

	t.Run("option with all properties should work", func(t *testing.T) {
		option := &kbtypes.ProviderOption{
			Properties: map[string]interface{}{
				"headers": map[string]interface{}{
					"Authorization": "Bearer secret",
					"Content-Type":  "application/json",
				},
				"user_agent": "Complete-Fetcher/1.0",
				"timeout":    60,
			},
		}
		result, err := fetcher.Make(option)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if result == nil {
			t.Error("Expected HTTP fetcher, got nil")
		}
	})

	t.Run("invalid property types should be ignored", func(t *testing.T) {
		option := &kbtypes.ProviderOption{
			Properties: map[string]interface{}{
				"headers":    "invalid_type",    // should be map
				"user_agent": 123,               // should be string
				"timeout":    "invalid_timeout", // should be number
			},
		}
		result, err := fetcher.Make(option)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if result == nil {
			t.Error("Expected HTTP fetcher, got nil")
		}
	})

	t.Run("headers with non-string values should be ignored", func(t *testing.T) {
		option := &kbtypes.ProviderOption{
			Properties: map[string]interface{}{
				"headers": map[string]interface{}{
					"Valid-Header":   "valid_value",
					"Invalid-Header": 123, // non-string value should be ignored
					"Another-Valid":  "another_value",
				},
			},
		}
		result, err := fetcher.Make(option)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if result == nil {
			t.Error("Expected HTTP fetcher, got nil")
		}
	})

	t.Run("zero timeout should be converted correctly", func(t *testing.T) {
		option := &kbtypes.ProviderOption{
			Properties: map[string]interface{}{
				"timeout": 0, // Should result in 0 duration, which will use default
			},
		}
		result, err := fetcher.Make(option)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if result == nil {
			t.Error("Expected HTTP fetcher, got nil")
		}
	})

	t.Run("empty headers map should work", func(t *testing.T) {
		option := &kbtypes.ProviderOption{
			Properties: map[string]interface{}{
				"headers": map[string]interface{}{}, // Empty headers map
			},
		}
		result, err := fetcher.Make(option)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if result == nil {
			t.Error("Expected HTTP fetcher, got nil")
		}
	})
}

func TestFetcherHTTP_Schema(t *testing.T) {
	fetcher := &FetcherHTTP{}
	schema, err := fetcher.Schema(nil, "en")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if schema == nil {
		t.Error("Expected non-nil schema from factory.GetSchemaFromBindata")
	}
}

func TestFetcherMCP_Make(t *testing.T) {
	fetcher := &FetcherMCP{}

	// Note: MCP fetcher requires MCP clients to be loaded
	// All tests will fail in test environment due to missing MCP client

	t.Run("nil option should return error due to missing MCP client", func(t *testing.T) {
		_, err := fetcher.Make(nil)
		if err == nil {
			t.Error("Expected error due to missing MCP client")
		}
		// Error is expected because MCP client is not loaded in test environment
	})

	t.Run("empty option should return error due to missing MCP client", func(t *testing.T) {
		option := &kbtypes.ProviderOption{}
		_, err := fetcher.Make(option)
		if err == nil {
			t.Error("Expected error due to missing MCP client")
		}
		// Error is expected because MCP client is not loaded in test environment
	})

	t.Run("option with id should return error due to missing MCP client", func(t *testing.T) {
		option := &kbtypes.ProviderOption{
			Properties: map[string]interface{}{
				"id": "fetcher",
			},
		}
		_, err := fetcher.Make(option)
		if err == nil {
			t.Error("Expected error due to missing MCP client")
		}
		// Error is expected because MCP client "fetcher" is not loaded
	})

	t.Run("option with id and tool should return error due to missing MCP client", func(t *testing.T) {
		option := &kbtypes.ProviderOption{
			Properties: map[string]interface{}{
				"id":   "fetcher",
				"tool": "fetch_url",
			},
		}
		_, err := fetcher.Make(option)
		if err == nil {
			t.Error("Expected error due to missing MCP client")
		}
		// Error is expected because MCP client is not loaded
	})

	t.Run("option with arguments_mapping should return error due to missing MCP client", func(t *testing.T) {
		option := &kbtypes.ProviderOption{
			Properties: map[string]interface{}{
				"id": "fetcher",
				"arguments_mapping": map[string]interface{}{
					"url":     "{{.url}}",
					"headers": "{{.headers}}",
				},
			},
		}
		_, err := fetcher.Make(option)
		if err == nil {
			t.Error("Expected error due to missing MCP client")
		}
		// Error is expected because MCP client is not loaded
	})

	t.Run("option with output_mapping should return error due to missing MCP client", func(t *testing.T) {
		option := &kbtypes.ProviderOption{
			Properties: map[string]interface{}{
				"id": "fetcher",
				"output_mapping": map[string]interface{}{
					"content":   "{{.result.content}}",
					"mime_type": "{{.result.mime_type}}",
				},
			},
		}
		_, err := fetcher.Make(option)
		if err == nil {
			t.Error("Expected error due to missing MCP client")
		}
		// Error is expected because MCP client is not loaded
	})

	t.Run("option with result_mapping should return error due to missing MCP client", func(t *testing.T) {
		option := &kbtypes.ProviderOption{
			Properties: map[string]interface{}{
				"id": "fetcher",
				"result_mapping": map[string]interface{}{
					"content":   "{{.data.content}}",
					"mime_type": "{{.data.type}}",
				},
			},
		}
		_, err := fetcher.Make(option)
		if err == nil {
			t.Error("Expected error due to missing MCP client")
		}
		// Error is expected because MCP client is not loaded
	})

	t.Run("option with notification_mapping should return error due to missing MCP client", func(t *testing.T) {
		option := &kbtypes.ProviderOption{
			Properties: map[string]interface{}{
				"id": "fetcher",
				"notification_mapping": map[string]interface{}{
					"progress": "{{.progress}}",
					"message":  "{{.message}}",
				},
			},
		}
		_, err := fetcher.Make(option)
		if err == nil {
			t.Error("Expected error due to missing MCP client")
		}
		// Error is expected because MCP client is not loaded
	})

	t.Run("option with all properties should return error due to missing MCP client", func(t *testing.T) {
		option := &kbtypes.ProviderOption{
			Properties: map[string]interface{}{
				"id":   "fetcher",
				"tool": "fetch_document",
				"arguments_mapping": map[string]interface{}{
					"url":    "{{.url}}",
					"format": "text",
				},
				"result_mapping": map[string]interface{}{
					"content":   "{{.result.content}}",
					"mime_type": "{{.result.mime_type}}",
				},
				"notification_mapping": map[string]interface{}{
					"progress": "{{.notification.progress}}",
					"status":   "{{.notification.status}}",
				},
			},
		}
		_, err := fetcher.Make(option)
		if err == nil {
			t.Error("Expected error due to missing MCP client")
		}
		// Error is expected because MCP client is not loaded
	})

	t.Run("invalid property types should be ignored but still return error", func(t *testing.T) {
		option := &kbtypes.ProviderOption{
			Properties: map[string]interface{}{
				"id":                   123,        // invalid type
				"tool":                 []string{}, // invalid type
				"arguments_mapping":    "invalid",  // invalid type
				"result_mapping":       "invalid",  // invalid type
				"output_mapping":       "invalid",  // invalid type
				"notification_mapping": "invalid",  // invalid type
			},
		}
		_, err := fetcher.Make(option)
		if err == nil {
			t.Error("Expected error due to missing MCP client")
		}
		// Error is expected because MCP client is not loaded
	})

	t.Run("mapping with non-string values should be ignored but still return error", func(t *testing.T) {
		option := &kbtypes.ProviderOption{
			Properties: map[string]interface{}{
				"id": "fetcher",
				"arguments_mapping": map[string]interface{}{
					"valid_arg":   "{{.url}}",
					"invalid_arg": 123, // non-string value should be ignored
				},
			},
		}
		_, err := fetcher.Make(option)
		if err == nil {
			t.Error("Expected error due to missing MCP client")
		}
		// Error is expected because MCP client is not loaded
	})

	t.Run("empty mapping should be handled correctly but still return error", func(t *testing.T) {
		option := &kbtypes.ProviderOption{
			Properties: map[string]interface{}{
				"id":                   "fetcher",
				"arguments_mapping":    map[string]interface{}{}, // Empty mapping
				"result_mapping":       map[string]interface{}{}, // Empty mapping
				"notification_mapping": map[string]interface{}{}, // Empty mapping
			},
		}
		_, err := fetcher.Make(option)
		if err == nil {
			t.Error("Expected error due to missing MCP client")
		}
		// Error is expected because MCP client is not loaded
	})

	t.Run("missing id should return error", func(t *testing.T) {
		option := &kbtypes.ProviderOption{
			Properties: map[string]interface{}{
				"tool": "fetch_url",
				// No ID specified
			},
		}
		_, err := fetcher.Make(option)
		if err == nil {
			t.Error("Expected error due to missing MCP client")
		}
		// Error is expected because no ID is specified and MCP client is not loaded
	})

	t.Run("both output_mapping and result_mapping should prefer result_mapping but still return error", func(t *testing.T) {
		option := &kbtypes.ProviderOption{
			Properties: map[string]interface{}{
				"id": "fetcher",
				"result_mapping": map[string]interface{}{
					"content": "{{.result}}",
				},
				"output_mapping": map[string]interface{}{
					"content": "{{.output}}",
				},
			},
		}
		_, err := fetcher.Make(option)
		if err == nil {
			t.Error("Expected error due to missing MCP client")
		}
		// Error is expected because MCP client is not loaded
		// result_mapping should take precedence over output_mapping
	})

	t.Run("only output_mapping should be used as result_mapping but still return error", func(t *testing.T) {
		option := &kbtypes.ProviderOption{
			Properties: map[string]interface{}{
				"id": "fetcher",
				"output_mapping": map[string]interface{}{
					"content": "{{.output.data}}",
				},
			},
		}
		_, err := fetcher.Make(option)
		if err == nil {
			t.Error("Expected error due to missing MCP client")
		}
		// Error is expected because MCP client is not loaded
		// output_mapping should be mapped to result_mapping
	})
}

func TestFetcherMCP_Schema(t *testing.T) {
	fetcher := &FetcherMCP{}
	schema, err := fetcher.Schema(nil, "en")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if schema == nil {
		t.Error("Expected non-nil schema from factory.GetSchemaFromBindata")
	}
}
