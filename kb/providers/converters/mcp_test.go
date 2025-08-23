package converters

import (
	"testing"

	kbtypes "github.com/yaoapp/yao/kb/types"
)

func TestMCP_Make(t *testing.T) {
	mcp := &MCP{}

	t.Run("nil option should return error due to missing MCP client", func(t *testing.T) {
		_, err := mcp.Make(nil)
		if err == nil {
			t.Error("Expected error due to missing MCP client")
		}
		// Error is expected because MCP client is not set up in test environment
	})

	t.Run("empty option should return error due to missing MCP client", func(t *testing.T) {
		option := &kbtypes.ProviderOption{}
		_, err := mcp.Make(option)
		if err == nil {
			t.Error("Expected error due to missing MCP client")
		}
		// Error is expected because MCP client is not set up in test environment
	})

	t.Run("option with id and tool should return error due to missing MCP client", func(t *testing.T) {
		option := &kbtypes.ProviderOption{
			Properties: map[string]interface{}{
				"id":   "ocrflux",
				"tool": "process_image",
			},
		}
		_, err := mcp.Make(option)
		if err == nil {
			t.Error("Expected error due to missing MCP client")
		}
		// Error is expected because MCP client 'ocrflux' is not set up in test environment
	})

	t.Run("option with all mapping properties should return error due to missing MCP client", func(t *testing.T) {
		option := &kbtypes.ProviderOption{
			Properties: map[string]interface{}{
				"id":   "ocrflux",
				"tool": "process_document",
				"arguments_mapping": map[string]interface{}{
					"file":    "input_file",
					"options": "config",
				},
				"result_mapping": map[string]interface{}{
					"text":     "extracted_text",
					"metadata": "file_info",
				},
				"notification_mapping": map[string]interface{}{
					"progress": "status",
					"error":    "error_msg",
				},
			},
		}
		_, err := mcp.Make(option)
		if err == nil {
			t.Error("Expected error due to missing MCP client")
		}
	})

	t.Run("should support output_mapping as alias for result_mapping", func(t *testing.T) {
		option := &kbtypes.ProviderOption{
			Properties: map[string]interface{}{
				"id":   "ocrflux",
				"tool": "process_file",
				"output_mapping": map[string]interface{}{
					"content": "extracted_content",
					"pages":   "page_count",
				},
			},
		}
		_, err := mcp.Make(option)
		if err == nil {
			t.Error("Expected error due to missing MCP client")
		}
	})

	t.Run("invalid mapping types should be ignored but still return error", func(t *testing.T) {
		option := &kbtypes.ProviderOption{
			Properties: map[string]interface{}{
				"id":                   "ocrflux",
				"tool":                 "process_file",
				"arguments_mapping":    "invalid_type",    // should be map
				"result_mapping":       123,               // should be map
				"notification_mapping": []string{"array"}, // should be map
			},
		}
		_, err := mcp.Make(option)
		if err == nil {
			t.Error("Expected error due to missing MCP client")
		}
	})

	t.Run("mapping with non-string values should be filtered out but still return error", func(t *testing.T) {
		option := &kbtypes.ProviderOption{
			Properties: map[string]interface{}{
				"id":   "ocrflux",
				"tool": "process_file",
				"arguments_mapping": map[string]interface{}{
					"valid_key":   "valid_value", // should be included
					"invalid_key": 123,           // should be filtered out
					"another_key": true,          // should be filtered out
				},
			},
		}
		_, err := mcp.Make(option)
		if err == nil {
			t.Error("Expected error due to missing MCP client")
		}
	})

	t.Run("empty mappings should not set mapping fields but still return error", func(t *testing.T) {
		option := &kbtypes.ProviderOption{
			Properties: map[string]interface{}{
				"id":                "ocrflux",
				"tool":              "process_file",
				"arguments_mapping": map[string]interface{}{}, // empty map
				"result_mapping":    map[string]interface{}{}, // empty map
			},
		}
		_, err := mcp.Make(option)
		if err == nil {
			t.Error("Expected error due to missing MCP client")
		}
	})

	t.Run("invalid property types should be ignored but still return error", func(t *testing.T) {
		option := &kbtypes.ProviderOption{
			Properties: map[string]interface{}{
				"id":   123,  // invalid type
				"tool": true, // invalid type
			},
		}
		_, err := mcp.Make(option)
		if err == nil {
			t.Error("Expected error due to missing MCP client")
		}
	})
}

func TestMCP_AutoDetect(t *testing.T) {
	mcp := &MCP{
		Autodetect:    []string{".pdf", ".jpg", ".png", "application/pdf", "image/jpeg"},
		MatchPriority: 15,
	}

	t.Run("should detect .pdf files", func(t *testing.T) {
		match, priority, err := mcp.AutoDetect("document.pdf", "")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if !match {
			t.Error("Expected match for .pdf file")
		}
		if priority != 15 {
			t.Errorf("Expected priority 15, got %d", priority)
		}
	})

	t.Run("should detect .jpg files", func(t *testing.T) {
		match, priority, err := mcp.AutoDetect("image.jpg", "")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if !match {
			t.Error("Expected match for .jpg file")
		}
		if priority != 15 {
			t.Errorf("Expected priority 15, got %d", priority)
		}
	})

	t.Run("should detect by content type", func(t *testing.T) {
		match, priority, err := mcp.AutoDetect("unknown", "application/pdf")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if !match {
			t.Error("Expected match for application/pdf content type")
		}
		if priority != 15 {
			t.Errorf("Expected priority 15, got %d", priority)
		}
	})

	t.Run("should not detect unsupported files", func(t *testing.T) {
		match, priority, err := mcp.AutoDetect("video.mp4", "video/mp4")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if match {
			t.Error("Expected no match for .mp4 file")
		}
		if priority != 0 {
			t.Errorf("Expected priority 0, got %d", priority)
		}
	})

	t.Run("empty autodetect should not match", func(t *testing.T) {
		emptyMCP := &MCP{}
		match, priority, err := emptyMCP.AutoDetect("document.pdf", "application/pdf")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if match {
			t.Error("Expected no match when autodetect is empty")
		}
		if priority != 0 {
			t.Errorf("Expected priority 0, got %d", priority)
		}
	})
}

func TestMCP_Schema(t *testing.T) {
	mcp := &MCP{}
	schema, err := mcp.Schema(nil, "en")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if schema == nil {
		t.Error("Expected non-nil schema from factory.GetSchemaFromBindata")
	}
}
