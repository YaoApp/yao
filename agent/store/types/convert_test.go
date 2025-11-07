package types

import (
	"testing"
	"time"
)

// TestToKnowledgeBase tests the ToKnowledgeBase conversion function
func TestToKnowledgeBase(t *testing.T) {
	t.Run("NilInput", func(t *testing.T) {
		result, err := ToKnowledgeBase(nil)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if result != nil {
			t.Errorf("Expected nil result, got: %v", result)
		}
	})

	t.Run("KnowledgeBasePointer", func(t *testing.T) {
		kb := &KnowledgeBase{Collections: []string{"col1", "col2"}}
		result, err := ToKnowledgeBase(kb)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if result != kb {
			t.Errorf("Expected same pointer")
		}
	})

	t.Run("KnowledgeBaseValue", func(t *testing.T) {
		kb := KnowledgeBase{Collections: []string{"col1", "col2"}}
		result, err := ToKnowledgeBase(kb)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if len(result.Collections) != 2 {
			t.Errorf("Expected 2 collections, got %d", len(result.Collections))
		}
	})

	t.Run("StringSlice", func(t *testing.T) {
		collections := []string{"col1", "col2", "col3"}
		result, err := ToKnowledgeBase(collections)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if len(result.Collections) != 3 {
			t.Errorf("Expected 3 collections, got %d", len(result.Collections))
		}
		if result.Collections[0] != "col1" {
			t.Errorf("Expected 'col1', got '%s'", result.Collections[0])
		}
	})

	t.Run("InterfaceSlice", func(t *testing.T) {
		collections := []interface{}{"col1", "col2", 123}
		result, err := ToKnowledgeBase(collections)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if len(result.Collections) != 3 {
			t.Errorf("Expected 3 collections, got %d", len(result.Collections))
		}
		if result.Collections[2] != "123" {
			t.Errorf("Expected '123', got '%s'", result.Collections[2])
		}
	})

	t.Run("MapInput", func(t *testing.T) {
		data := map[string]interface{}{
			"collections": []string{"col1", "col2"},
		}
		result, err := ToKnowledgeBase(data)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if len(result.Collections) != 2 {
			t.Errorf("Expected 2 collections, got %d", len(result.Collections))
		}
	})

	t.Run("InvalidInput", func(t *testing.T) {
		// Test with data that can't be marshaled
		invalidData := make(chan int)
		_, err := ToKnowledgeBase(invalidData)
		if err == nil {
			t.Error("Expected error for invalid input")
		}
	})

	t.Run("InvalidJSONUnmarshal", func(t *testing.T) {
		// Test with data that marshals but can't unmarshal to KnowledgeBase
		data := map[string]interface{}{
			"invalid_field": "should cause unmarshal to fail gracefully",
		}
		result, err := ToKnowledgeBase(data)
		// Should not error, just return empty KnowledgeBase
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if result == nil {
			t.Error("Expected non-nil result")
		}
	})
}

// TestToMCPServers tests the ToMCPServers conversion function
func TestToMCPServers(t *testing.T) {
	t.Run("NilInput", func(t *testing.T) {
		result, err := ToMCPServers(nil)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if result != nil {
			t.Errorf("Expected nil result, got: %v", result)
		}
	})

	t.Run("MCPServersPointer", func(t *testing.T) {
		mcp := &MCPServers{Servers: []string{"server1", "server2"}}
		result, err := ToMCPServers(mcp)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if result != mcp {
			t.Errorf("Expected same pointer")
		}
	})

	t.Run("MCPServersValue", func(t *testing.T) {
		mcp := MCPServers{Servers: []string{"server1", "server2"}}
		result, err := ToMCPServers(mcp)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if len(result.Servers) != 2 {
			t.Errorf("Expected 2 servers, got %d", len(result.Servers))
		}
	})

	t.Run("StringSlice", func(t *testing.T) {
		servers := []string{"server1", "server2", "server3"}
		result, err := ToMCPServers(servers)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if len(result.Servers) != 3 {
			t.Errorf("Expected 3 servers, got %d", len(result.Servers))
		}
		if result.Servers[0] != "server1" {
			t.Errorf("Expected 'server1', got '%s'", result.Servers[0])
		}
	})

	t.Run("InterfaceSlice", func(t *testing.T) {
		servers := []interface{}{"server1", "server2", 456}
		result, err := ToMCPServers(servers)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if len(result.Servers) != 3 {
			t.Errorf("Expected 3 servers, got %d", len(result.Servers))
		}
		if result.Servers[2] != "456" {
			t.Errorf("Expected '456', got '%s'", result.Servers[2])
		}
	})

	t.Run("MapInput", func(t *testing.T) {
		data := map[string]interface{}{
			"servers": []string{"server1", "server2"},
		}
		result, err := ToMCPServers(data)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if len(result.Servers) != 2 {
			t.Errorf("Expected 2 servers, got %d", len(result.Servers))
		}
	})

	t.Run("InvalidInput", func(t *testing.T) {
		// Test with data that can't be marshaled
		invalidData := make(chan int)
		_, err := ToMCPServers(invalidData)
		if err == nil {
			t.Error("Expected error for invalid input")
		}
	})

	t.Run("InvalidJSONUnmarshal", func(t *testing.T) {
		// Test with data that marshals but can't unmarshal to MCPServers
		data := map[string]interface{}{
			"invalid_field": "should cause unmarshal to fail gracefully",
		}
		result, err := ToMCPServers(data)
		// Should not error, just return empty MCPServers
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if result == nil {
			t.Error("Expected non-nil result")
		}
	})
}

// TestToWorkflow tests the ToWorkflow conversion function
func TestToWorkflow(t *testing.T) {
	t.Run("NilInput", func(t *testing.T) {
		result, err := ToWorkflow(nil)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if result != nil {
			t.Errorf("Expected nil result, got: %v", result)
		}
	})

	t.Run("WorkflowPointer", func(t *testing.T) {
		wf := &Workflow{Workflows: []string{"wf1", "wf2"}}
		result, err := ToWorkflow(wf)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if result != wf {
			t.Errorf("Expected same pointer")
		}
	})

	t.Run("WorkflowValue", func(t *testing.T) {
		wf := Workflow{Workflows: []string{"wf1", "wf2"}}
		result, err := ToWorkflow(wf)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if len(result.Workflows) != 2 {
			t.Errorf("Expected 2 workflows, got %d", len(result.Workflows))
		}
	})

	t.Run("StringSlice", func(t *testing.T) {
		workflows := []string{"wf1", "wf2", "wf3"}
		result, err := ToWorkflow(workflows)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if len(result.Workflows) != 3 {
			t.Errorf("Expected 3 workflows, got %d", len(result.Workflows))
		}
		if result.Workflows[0] != "wf1" {
			t.Errorf("Expected 'wf1', got '%s'", result.Workflows[0])
		}
	})

	t.Run("InterfaceSlice", func(t *testing.T) {
		workflows := []interface{}{"wf1", "wf2", 789}
		result, err := ToWorkflow(workflows)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if len(result.Workflows) != 3 {
			t.Errorf("Expected 3 workflows, got %d", len(result.Workflows))
		}
		if result.Workflows[2] != "789" {
			t.Errorf("Expected '789', got '%s'", result.Workflows[2])
		}
	})

	t.Run("MapInput", func(t *testing.T) {
		data := map[string]interface{}{
			"workflows": []string{"wf1", "wf2"},
		}
		result, err := ToWorkflow(data)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if len(result.Workflows) != 2 {
			t.Errorf("Expected 2 workflows, got %d", len(result.Workflows))
		}
	})

	t.Run("InvalidInput", func(t *testing.T) {
		// Test with data that can't be marshaled
		invalidData := make(chan int)
		_, err := ToWorkflow(invalidData)
		if err == nil {
			t.Error("Expected error for invalid input")
		}
	})

	t.Run("InvalidJSONUnmarshal", func(t *testing.T) {
		// Test with data that marshals but can't unmarshal to Workflow
		data := map[string]interface{}{
			"invalid_field": "should cause unmarshal to fail gracefully",
		}
		result, err := ToWorkflow(data)
		// Should not error, just return empty Workflow
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if result == nil {
			t.Error("Expected non-nil result")
		}
	})
}

// TestToMySQLTime tests the ToMySQLTime conversion function
func TestToMySQLTime(t *testing.T) {
	t.Run("Int64Zero", func(t *testing.T) {
		result := ToMySQLTime(int64(0))
		if result != "0000-00-00 00:00:00" {
			t.Errorf("Expected '0000-00-00 00:00:00', got '%s'", result)
		}
	})

	t.Run("Int64Timestamp", func(t *testing.T) {
		// Unix timestamp in nanoseconds: 1609459200000000000 = 2021-01-01 00:00:00 UTC
		timestamp := int64(1609459200000000000)
		result := ToMySQLTime(timestamp)
		// Should be in format "2021-01-01 00:00:00" or similar depending on timezone
		if len(result) != 19 {
			t.Errorf("Expected 19 character timestamp, got %d: '%s'", len(result), result)
		}
	})

	t.Run("IntZero", func(t *testing.T) {
		result := ToMySQLTime(int(0))
		if result != "0000-00-00 00:00:00" {
			t.Errorf("Expected '0000-00-00 00:00:00', got '%s'", result)
		}
	})

	t.Run("IntTimestamp", func(t *testing.T) {
		timestamp := int(1609459200000000000)
		result := ToMySQLTime(timestamp)
		if len(result) != 19 {
			t.Errorf("Expected 19 character timestamp, got %d: '%s'", len(result), result)
		}
	})

	t.Run("StringMySQLFormat", func(t *testing.T) {
		mysqlTime := "2021-01-01 12:30:45"
		result := ToMySQLTime(mysqlTime)
		if result != mysqlTime {
			t.Errorf("Expected '%s', got '%s'", mysqlTime, result)
		}
	})

	t.Run("StringRFC3339", func(t *testing.T) {
		rfc3339Time := "2021-01-01T12:30:45Z"
		result := ToMySQLTime(rfc3339Time)
		expected := "2021-01-01 12:30:45"
		if result != expected {
			t.Errorf("Expected '%s', got '%s'", expected, result)
		}
	})

	t.Run("StringUnixTimestamp", func(t *testing.T) {
		// Unix timestamp in seconds as string
		result := ToMySQLTime("1609459200000000000")
		if len(result) != 19 {
			t.Errorf("Expected 19 character timestamp, got %d: '%s'", len(result), result)
		}
	})

	t.Run("StringInvalidFormat", func(t *testing.T) {
		invalidTime := "not-a-valid-time"
		result := ToMySQLTime(invalidTime)
		// Should return the original string when it can't be parsed
		if result != invalidTime {
			t.Errorf("Expected '%s', got '%s'", invalidTime, result)
		}
	})

	t.Run("TimeZero", func(t *testing.T) {
		zeroTime := time.Time{}
		result := ToMySQLTime(zeroTime)
		if result != "0000-00-00 00:00:00" {
			t.Errorf("Expected '0000-00-00 00:00:00', got '%s'", result)
		}
	})

	t.Run("TimeNormal", func(t *testing.T) {
		normalTime := time.Date(2021, 1, 1, 12, 30, 45, 0, time.UTC)
		result := ToMySQLTime(normalTime)
		expected := "2021-01-01 12:30:45"
		if result != expected {
			t.Errorf("Expected '%s', got '%s'", expected, result)
		}
	})

	t.Run("NilInput", func(t *testing.T) {
		result := ToMySQLTime(nil)
		if result != "0000-00-00 00:00:00" {
			t.Errorf("Expected '0000-00-00 00:00:00', got '%s'", result)
		}
	})

	t.Run("UnknownType", func(t *testing.T) {
		// Test with unsupported type
		result := ToMySQLTime(struct{}{})
		if result != "0000-00-00 00:00:00" {
			t.Errorf("Expected '0000-00-00 00:00:00', got '%s'", result)
		}
	})
}

// TestToAssistantModel tests the ToAssistantModel conversion function
func TestToAssistantModel(t *testing.T) {
	t.Run("NilInput", func(t *testing.T) {
		result, err := ToAssistantModel(nil)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if result != nil {
			t.Errorf("Expected nil result, got: %v", result)
		}
	})

	t.Run("AssistantModelPointer", func(t *testing.T) {
		model := &AssistantModel{
			ID:   "test-id",
			Name: "Test Assistant",
		}
		result, err := ToAssistantModel(model)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if result != model {
			t.Errorf("Expected same pointer")
		}
	})

	t.Run("AssistantModelValue", func(t *testing.T) {
		model := AssistantModel{
			ID:   "test-id",
			Name: "Test Assistant",
		}
		result, err := ToAssistantModel(model)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if result.ID != "test-id" {
			t.Errorf("Expected 'test-id', got '%s'", result.ID)
		}
	})

	t.Run("MapWithAllFields", func(t *testing.T) {
		data := map[string]interface{}{
			"assistant_id": "test-id",
			"type":         "assistant",
			"name":         "Test Assistant",
			"avatar":       "https://example.com/avatar.png",
			"connector":    "openai",
			"path":         "/path/to/assistant",
			"description":  "Test description",
			"share":        "team",
			"built_in":     true,
			"readonly":     false,
			"public":       true,
			"mentionable":  true,
			"automated":    false,
			"sort":         100,
			"created_at":   int64(1609459200),
			"updated_at":   int64(1609459300),
			"tags":         []string{"tag1", "tag2"},
			"options": map[string]interface{}{
				"temperature": 0.7,
			},
			"prompts": []map[string]interface{}{
				{"role": "system", "content": "You are helpful"},
			},
			"kb": map[string]interface{}{
				"collections": []string{"col1"},
			},
			"mcp": map[string]interface{}{
				"servers": []string{"server1"},
			},
			"workflow": map[string]interface{}{
				"workflows": []string{"wf1"},
			},
			"tools": map[string]interface{}{
				"calls": []string{"tool1"},
			},
			"placeholder": map[string]interface{}{
				"title": "Enter message",
			},
			"locales": map[string]interface{}{
				"en": map[string]interface{}{
					"name": "English Name",
				},
			},
		}

		result, err := ToAssistantModel(data)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		// Verify all fields
		if result.ID != "test-id" {
			t.Errorf("Expected ID 'test-id', got '%s'", result.ID)
		}
		if result.Type != "assistant" {
			t.Errorf("Expected Type 'assistant', got '%s'", result.Type)
		}
		if result.Name != "Test Assistant" {
			t.Errorf("Expected Name 'Test Assistant', got '%s'", result.Name)
		}
		if result.Avatar != "https://example.com/avatar.png" {
			t.Errorf("Expected Avatar URL, got '%s'", result.Avatar)
		}
		if result.Connector != "openai" {
			t.Errorf("Expected Connector 'openai', got '%s'", result.Connector)
		}
		if result.Path != "/path/to/assistant" {
			t.Errorf("Expected Path, got '%s'", result.Path)
		}
		if result.Description != "Test description" {
			t.Errorf("Expected Description, got '%s'", result.Description)
		}
		if result.Share != "team" {
			t.Errorf("Expected Share 'team', got '%s'", result.Share)
		}
		if !result.BuiltIn {
			t.Error("Expected BuiltIn to be true")
		}
		if result.Readonly {
			t.Error("Expected Readonly to be false")
		}
		if !result.Public {
			t.Error("Expected Public to be true")
		}
		if !result.Mentionable {
			t.Error("Expected Mentionable to be true")
		}
		if result.Automated {
			t.Error("Expected Automated to be false")
		}
		if result.Sort != 100 {
			t.Errorf("Expected Sort 100, got %d", result.Sort)
		}
		if result.CreatedAt != 1609459200 {
			t.Errorf("Expected CreatedAt 1609459200, got %d", result.CreatedAt)
		}
		if result.UpdatedAt != 1609459300 {
			t.Errorf("Expected UpdatedAt 1609459300, got %d", result.UpdatedAt)
		}
		if len(result.Tags) != 2 {
			t.Errorf("Expected 2 tags, got %d", len(result.Tags))
		}
		if result.Options == nil {
			t.Error("Expected Options to be set")
		}
		if len(result.Prompts) != 1 {
			t.Errorf("Expected 1 prompt, got %d", len(result.Prompts))
		}
		if result.KB == nil {
			t.Error("Expected KB to be set")
		}
		if result.MCP == nil {
			t.Error("Expected MCP to be set")
		}
		if result.Workflow == nil {
			t.Error("Expected Workflow to be set")
		}
		if result.Tools == nil {
			t.Error("Expected Tools to be set")
		}
		if result.Placeholder == nil {
			t.Error("Expected Placeholder to be set")
		}
		if result.Locales == nil {
			t.Error("Expected Locales to be set")
		}
	})

	t.Run("MapWithFloatNumbers", func(t *testing.T) {
		data := map[string]interface{}{
			"sort":       float64(150),
			"created_at": float64(1609459200),
			"updated_at": float64(1609459300),
		}

		result, err := ToAssistantModel(data)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if result.Sort != 150 {
			t.Errorf("Expected Sort 150, got %d", result.Sort)
		}
		if result.CreatedAt != 1609459200 {
			t.Errorf("Expected CreatedAt 1609459200, got %d", result.CreatedAt)
		}
		if result.UpdatedAt != 1609459300 {
			t.Errorf("Expected UpdatedAt 1609459300, got %d", result.UpdatedAt)
		}
	})

	t.Run("MapWithNilFields", func(t *testing.T) {
		data := map[string]interface{}{
			"assistant_id": "test-id",
			"tags":         nil,
			"options":      nil,
			"prompts":      nil,
			"kb":           nil,
			"mcp":          nil,
			"workflow":     nil,
			"tools":        nil,
			"placeholder":  nil,
			"locales":      nil,
		}

		result, err := ToAssistantModel(data)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if result.ID != "test-id" {
			t.Errorf("Expected ID 'test-id', got '%s'", result.ID)
		}
		// All nil fields should remain nil
		if result.Tags != nil {
			t.Error("Expected Tags to be nil")
		}
		if result.Options != nil {
			t.Error("Expected Options to be nil")
		}
	})

	t.Run("StructInput", func(t *testing.T) {
		type CustomStruct struct {
			AssistantID string `json:"assistant_id"`
			Name        string `json:"name"`
			Type        string `json:"type"`
		}

		input := CustomStruct{
			AssistantID: "custom-id",
			Name:        "Custom Assistant",
			Type:        "bot",
		}

		result, err := ToAssistantModel(input)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if result.ID != "custom-id" {
			t.Errorf("Expected ID 'custom-id', got '%s'", result.ID)
		}
		if result.Name != "Custom Assistant" {
			t.Errorf("Expected Name 'Custom Assistant', got '%s'", result.Name)
		}
		if result.Type != "bot" {
			t.Errorf("Expected Type 'bot', got '%s'", result.Type)
		}
	})

	t.Run("InvalidInput", func(t *testing.T) {
		// Test with data that can't be marshaled
		invalidData := make(chan int)
		_, err := ToAssistantModel(invalidData)
		if err == nil {
			t.Error("Expected error for invalid input")
		}
	})

	t.Run("EmptyMap", func(t *testing.T) {
		data := map[string]interface{}{}
		result, err := ToAssistantModel(data)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if result == nil {
			t.Error("Expected non-nil result")
		}
		// All fields should have default values
		if result.ID != "" {
			t.Errorf("Expected empty ID, got '%s'", result.ID)
		}
	})
}

// TestToAssistantModelComplexTypes tests complex type conversions in ToAssistantModel
func TestToAssistantModelComplexTypes(t *testing.T) {
	t.Run("CompleteLocales", func(t *testing.T) {
		data := map[string]interface{}{
			"locales": map[string]interface{}{
				"en": map[string]interface{}{
					"locale": "en",
					"messages": map[string]interface{}{
						"name":        "English Name",
						"description": "English Description",
					},
				},
				"zh": map[string]interface{}{
					"locale": "zh",
					"messages": map[string]interface{}{
						"name":        "中文名称",
						"description": "中文描述",
					},
				},
			},
		}

		result, err := ToAssistantModel(data)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if result.Locales == nil {
			t.Fatal("Expected Locales to be set")
		}

		if len(result.Locales) != 2 {
			t.Errorf("Expected 2 locales, got %d", len(result.Locales))
		}
	})

	t.Run("ComplexPrompts", func(t *testing.T) {
		data := map[string]interface{}{
			"prompts": []interface{}{
				map[string]interface{}{
					"role":    "system",
					"content": "You are a helpful assistant",
				},
				map[string]interface{}{
					"role":    "user",
					"content": "Hello",
				},
			},
		}

		result, err := ToAssistantModel(data)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if len(result.Prompts) != 2 {
			t.Errorf("Expected 2 prompts, got %d", len(result.Prompts))
		}
	})
}

// TestGetBoolValue tests the getBoolValue helper function
func TestGetBoolValue(t *testing.T) {
	t.Run("BoolTrue", func(t *testing.T) {
		data := map[string]interface{}{"key": true}
		result := getBoolValue(data, "key")
		if !result {
			t.Error("Expected true")
		}
	})

	t.Run("BoolFalse", func(t *testing.T) {
		data := map[string]interface{}{"key": false}
		result := getBoolValue(data, "key")
		if result {
			t.Error("Expected false")
		}
	})

	t.Run("IntNonZero", func(t *testing.T) {
		data := map[string]interface{}{"key": 1}
		result := getBoolValue(data, "key")
		if !result {
			t.Error("Expected true for non-zero int")
		}
	})

	t.Run("IntZero", func(t *testing.T) {
		data := map[string]interface{}{"key": 0}
		result := getBoolValue(data, "key")
		if result {
			t.Error("Expected false for zero int")
		}
	})

	t.Run("Int64NonZero", func(t *testing.T) {
		data := map[string]interface{}{"key": int64(1)}
		result := getBoolValue(data, "key")
		if !result {
			t.Error("Expected true for non-zero int64")
		}
	})

	t.Run("Int64Zero", func(t *testing.T) {
		data := map[string]interface{}{"key": int64(0)}
		result := getBoolValue(data, "key")
		if result {
			t.Error("Expected false for zero int64")
		}
	})

	t.Run("Float64NonZero", func(t *testing.T) {
		data := map[string]interface{}{"key": float64(1.5)}
		result := getBoolValue(data, "key")
		if !result {
			t.Error("Expected true for non-zero float64")
		}
	})

	t.Run("Float64Zero", func(t *testing.T) {
		data := map[string]interface{}{"key": float64(0)}
		result := getBoolValue(data, "key")
		if result {
			t.Error("Expected false for zero float64")
		}
	})

	t.Run("StringTrue", func(t *testing.T) {
		data := map[string]interface{}{"key": "true"}
		result := getBoolValue(data, "key")
		if !result {
			t.Error("Expected true for string 'true'")
		}
	})

	t.Run("StringOne", func(t *testing.T) {
		data := map[string]interface{}{"key": "1"}
		result := getBoolValue(data, "key")
		if !result {
			t.Error("Expected true for string '1'")
		}
	})

	t.Run("StringFalse", func(t *testing.T) {
		data := map[string]interface{}{"key": "false"}
		result := getBoolValue(data, "key")
		if result {
			t.Error("Expected false for string 'false'")
		}
	})

	t.Run("StringOther", func(t *testing.T) {
		data := map[string]interface{}{"key": "other"}
		result := getBoolValue(data, "key")
		if result {
			t.Error("Expected false for other string values")
		}
	})

	t.Run("NilValue", func(t *testing.T) {
		data := map[string]interface{}{"key": nil}
		result := getBoolValue(data, "key")
		if result {
			t.Error("Expected false for nil value")
		}
	})

	t.Run("MissingKey", func(t *testing.T) {
		data := map[string]interface{}{}
		result := getBoolValue(data, "missing")
		if result {
			t.Error("Expected false for missing key")
		}
	})

	t.Run("UnsupportedType", func(t *testing.T) {
		data := map[string]interface{}{"key": struct{}{}}
		result := getBoolValue(data, "key")
		if result {
			t.Error("Expected false for unsupported type")
		}
	})
}
