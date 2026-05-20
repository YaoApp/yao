//go:build unit

package types_test

import (
	"testing"
	"time"

	types "github.com/yaoapp/yao/agent/store/types"
)

func TestToDatabase(t *testing.T) {
	t.Run("NilInput", func(t *testing.T) {
		result, err := types.ToDatabase(nil)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if result != nil {
			t.Errorf("Expected nil result, got: %v", result)
		}
	})

	t.Run("DatabasePointer", func(t *testing.T) {
		db := &types.Database{Models: []string{"model1", "model2"}}
		result, err := types.ToDatabase(db)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if result != db {
			t.Errorf("Expected same pointer")
		}
	})

	t.Run("DatabaseValue", func(t *testing.T) {
		db := types.Database{Models: []string{"model1", "model2"}}
		result, err := types.ToDatabase(db)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if len(result.Models) != 2 {
			t.Errorf("Expected 2 models, got %d", len(result.Models))
		}
	})

	t.Run("StringSlice", func(t *testing.T) {
		models := []string{"model1", "model2", "model3"}
		result, err := types.ToDatabase(models)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if len(result.Models) != 3 {
			t.Errorf("Expected 3 models, got %d", len(result.Models))
		}
		if result.Models[0] != "model1" {
			t.Errorf("Expected 'model1', got '%s'", result.Models[0])
		}
	})

	t.Run("InterfaceSlice", func(t *testing.T) {
		models := []interface{}{"model1", "model2", 123}
		result, err := types.ToDatabase(models)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if len(result.Models) != 3 {
			t.Errorf("Expected 3 models, got %d", len(result.Models))
		}
		if result.Models[2] != "123" {
			t.Errorf("Expected '123', got '%s'", result.Models[2])
		}
	})

	t.Run("MapInput", func(t *testing.T) {
		data := map[string]interface{}{
			"models": []string{"model1", "model2"},
		}
		result, err := types.ToDatabase(data)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if len(result.Models) != 2 {
			t.Errorf("Expected 2 models, got %d", len(result.Models))
		}
	})

	t.Run("InvalidInput", func(t *testing.T) {
		invalidData := make(chan int)
		_, err := types.ToDatabase(invalidData)
		if err == nil {
			t.Error("Expected error for invalid input")
		}
	})

	t.Run("InvalidJSONUnmarshal", func(t *testing.T) {
		data := map[string]interface{}{
			"invalid_field": "should cause unmarshal to fail gracefully",
		}
		result, err := types.ToDatabase(data)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if result == nil {
			t.Error("Expected non-nil result")
		}
	})
}

func TestToKnowledgeBase(t *testing.T) {
	t.Run("NilInput", func(t *testing.T) {
		result, err := types.ToKnowledgeBase(nil)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if result != nil {
			t.Errorf("Expected nil result, got: %v", result)
		}
	})

	t.Run("KnowledgeBasePointer", func(t *testing.T) {
		kb := &types.KnowledgeBase{Collections: []string{"col1", "col2"}}
		result, err := types.ToKnowledgeBase(kb)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if result != kb {
			t.Errorf("Expected same pointer")
		}
	})

	t.Run("KnowledgeBaseValue", func(t *testing.T) {
		kb := types.KnowledgeBase{Collections: []string{"col1", "col2"}}
		result, err := types.ToKnowledgeBase(kb)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if len(result.Collections) != 2 {
			t.Errorf("Expected 2 collections, got %d", len(result.Collections))
		}
	})

	t.Run("StringSlice", func(t *testing.T) {
		collections := []string{"col1", "col2", "col3"}
		result, err := types.ToKnowledgeBase(collections)
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
		result, err := types.ToKnowledgeBase(collections)
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
		result, err := types.ToKnowledgeBase(data)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if len(result.Collections) != 2 {
			t.Errorf("Expected 2 collections, got %d", len(result.Collections))
		}
	})

	t.Run("InvalidInput", func(t *testing.T) {
		invalidData := make(chan int)
		_, err := types.ToKnowledgeBase(invalidData)
		if err == nil {
			t.Error("Expected error for invalid input")
		}
	})

	t.Run("InvalidJSONUnmarshal", func(t *testing.T) {
		data := map[string]interface{}{
			"invalid_field": "should cause unmarshal to fail gracefully",
		}
		result, err := types.ToKnowledgeBase(data)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if result == nil {
			t.Error("Expected non-nil result")
		}
	})
}

func TestToMCPServers(t *testing.T) {
	t.Run("NilInput", func(t *testing.T) {
		result, err := types.ToMCPServers(nil)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if result != nil {
			t.Errorf("Expected nil result, got: %v", result)
		}
	})

	t.Run("MCPServersPointer", func(t *testing.T) {
		mcp := &types.MCPServers{Servers: []types.MCPServerConfig{{ServerID: "server1"}, {ServerID: "server2"}}}
		result, err := types.ToMCPServers(mcp)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if result != mcp {
			t.Errorf("Expected same pointer")
		}
	})

	t.Run("MCPServersValue", func(t *testing.T) {
		mcp := types.MCPServers{Servers: []types.MCPServerConfig{{ServerID: "server1"}, {ServerID: "server2"}}}
		result, err := types.ToMCPServers(mcp)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if len(result.Servers) != 2 {
			t.Errorf("Expected 2 servers, got %d", len(result.Servers))
		}
	})

	t.Run("MapInput", func(t *testing.T) {
		data := map[string]interface{}{
			"servers": []interface{}{"server1", "server2"},
		}
		result, err := types.ToMCPServers(data)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if len(result.Servers) != 2 {
			t.Errorf("Expected 2 servers, got %d", len(result.Servers))
		}
		if result.Servers[0].ServerID != "server1" {
			t.Errorf("Expected 'server1', got '%s'", result.Servers[0].ServerID)
		}
	})

	t.Run("InvalidInput", func(t *testing.T) {
		invalidData := make(chan int)
		_, err := types.ToMCPServers(invalidData)
		if err == nil {
			t.Error("Expected error for invalid input")
		}
	})

	t.Run("InvalidJSONUnmarshal", func(t *testing.T) {
		data := map[string]interface{}{
			"invalid_field": "should cause unmarshal to fail gracefully",
		}
		result, err := types.ToMCPServers(data)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if result == nil {
			t.Error("Expected non-nil result")
		}
	})
}

func TestToWorkflow(t *testing.T) {
	t.Run("NilInput", func(t *testing.T) {
		result, err := types.ToWorkflow(nil)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if result != nil {
			t.Errorf("Expected nil result, got: %v", result)
		}
	})

	t.Run("WorkflowPointer", func(t *testing.T) {
		wf := &types.Workflow{Workflows: []string{"wf1", "wf2"}}
		result, err := types.ToWorkflow(wf)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if result != wf {
			t.Errorf("Expected same pointer")
		}
	})

	t.Run("WorkflowValue", func(t *testing.T) {
		wf := types.Workflow{Workflows: []string{"wf1", "wf2"}}
		result, err := types.ToWorkflow(wf)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if len(result.Workflows) != 2 {
			t.Errorf("Expected 2 workflows, got %d", len(result.Workflows))
		}
	})

	t.Run("StringSlice", func(t *testing.T) {
		workflows := []string{"wf1", "wf2", "wf3"}
		result, err := types.ToWorkflow(workflows)
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
		result, err := types.ToWorkflow(workflows)
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
		result, err := types.ToWorkflow(data)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if len(result.Workflows) != 2 {
			t.Errorf("Expected 2 workflows, got %d", len(result.Workflows))
		}
	})

	t.Run("InvalidInput", func(t *testing.T) {
		invalidData := make(chan int)
		_, err := types.ToWorkflow(invalidData)
		if err == nil {
			t.Error("Expected error for invalid input")
		}
	})

	t.Run("InvalidJSONUnmarshal", func(t *testing.T) {
		data := map[string]interface{}{
			"invalid_field": "should cause unmarshal to fail gracefully",
		}
		result, err := types.ToWorkflow(data)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if result == nil {
			t.Error("Expected non-nil result")
		}
	})
}

func TestToMySQLTime(t *testing.T) {
	t.Run("Int64Zero", func(t *testing.T) {
		result := types.ToMySQLTime(int64(0))
		if result != "0000-00-00 00:00:00" {
			t.Errorf("Expected '0000-00-00 00:00:00', got '%s'", result)
		}
	})

	t.Run("Int64Timestamp", func(t *testing.T) {
		timestamp := int64(1609459200000000000)
		result := types.ToMySQLTime(timestamp)
		if len(result) != 19 {
			t.Errorf("Expected 19 character timestamp, got %d: '%s'", len(result), result)
		}
	})

	t.Run("IntZero", func(t *testing.T) {
		result := types.ToMySQLTime(int(0))
		if result != "0000-00-00 00:00:00" {
			t.Errorf("Expected '0000-00-00 00:00:00', got '%s'", result)
		}
	})

	t.Run("IntTimestamp", func(t *testing.T) {
		timestamp := int(1609459200000000000)
		result := types.ToMySQLTime(timestamp)
		if len(result) != 19 {
			t.Errorf("Expected 19 character timestamp, got %d: '%s'", len(result), result)
		}
	})

	t.Run("StringMySQLFormat", func(t *testing.T) {
		mysqlTime := "2021-01-01 12:30:45"
		result := types.ToMySQLTime(mysqlTime)
		if result != mysqlTime {
			t.Errorf("Expected '%s', got '%s'", mysqlTime, result)
		}
	})

	t.Run("StringRFC3339", func(t *testing.T) {
		rfc3339Time := "2021-01-01T12:30:45Z"
		result := types.ToMySQLTime(rfc3339Time)
		expected := "2021-01-01 12:30:45"
		if result != expected {
			t.Errorf("Expected '%s', got '%s'", expected, result)
		}
	})

	t.Run("StringUnixTimestamp", func(t *testing.T) {
		result := types.ToMySQLTime("1609459200000000000")
		if len(result) != 19 {
			t.Errorf("Expected 19 character timestamp, got %d: '%s'", len(result), result)
		}
	})

	t.Run("StringInvalidFormat", func(t *testing.T) {
		invalidTime := "not-a-valid-time"
		result := types.ToMySQLTime(invalidTime)
		if result != invalidTime {
			t.Errorf("Expected '%s', got '%s'", invalidTime, result)
		}
	})

	t.Run("TimeZero", func(t *testing.T) {
		zeroTime := time.Time{}
		result := types.ToMySQLTime(zeroTime)
		if result != "0000-00-00 00:00:00" {
			t.Errorf("Expected '0000-00-00 00:00:00', got '%s'", result)
		}
	})

	t.Run("TimeNormal", func(t *testing.T) {
		normalTime := time.Date(2021, 1, 1, 12, 30, 45, 0, time.UTC)
		result := types.ToMySQLTime(normalTime)
		expected := "2021-01-01 12:30:45"
		if result != expected {
			t.Errorf("Expected '%s', got '%s'", expected, result)
		}
	})

	t.Run("NilInput", func(t *testing.T) {
		result := types.ToMySQLTime(nil)
		if result != "0000-00-00 00:00:00" {
			t.Errorf("Expected '0000-00-00 00:00:00', got '%s'", result)
		}
	})

	t.Run("UnknownType", func(t *testing.T) {
		result := types.ToMySQLTime(struct{}{})
		if result != "0000-00-00 00:00:00" {
			t.Errorf("Expected '0000-00-00 00:00:00', got '%s'", result)
		}
	})
}

func TestToAssistantModel(t *testing.T) {
	t.Run("NilInput", func(t *testing.T) {
		result, err := types.ToAssistantModel(nil)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if result != nil {
			t.Errorf("Expected nil result, got: %v", result)
		}
	})

	t.Run("AssistantModelPointer", func(t *testing.T) {
		model := &types.AssistantModel{
			ID:   "test-id",
			Name: "Test Assistant",
		}
		result, err := types.ToAssistantModel(model)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if result != model {
			t.Errorf("Expected same pointer")
		}
	})

	t.Run("AssistantModelValue", func(t *testing.T) {
		model := types.AssistantModel{
			ID:   "test-id",
			Name: "Test Assistant",
		}
		result, err := types.ToAssistantModel(model)
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
			"connector_options": map[string]interface{}{
				"optional":   true,
				"connectors": []string{"openai", "anthropic"},
				"filters":    []string{"vision", "tool_calls"},
			},
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
			"modes":        []string{"chat", "task"},
			"default_mode": "chat",
			"options": map[string]interface{}{
				"temperature": 0.7,
			},
			"prompts": []map[string]interface{}{
				{"role": "system", "content": "You are helpful"},
			},
			"prompt_presets": map[string]interface{}{
				"chat": []map[string]interface{}{
					{"role": "system", "content": "You are a chat assistant"},
				},
				"task": []map[string]interface{}{
					{"role": "system", "content": "You are a task assistant"},
				},
			},
			"disable_global_prompts": true,
			"source":                 "function hook() { return 'test'; }",
			"kb": map[string]interface{}{
				"collections": []string{"col1"},
			},
			"db": map[string]interface{}{
				"models": []string{"model1"},
			},
			"mcp": map[string]interface{}{
				"servers": []string{"server1"},
			},
			"workflow": map[string]interface{}{
				"workflows": []string{"wf1"},
			},
			"placeholder": map[string]interface{}{
				"title": "Enter message",
			},
			"locales": map[string]interface{}{
				"en": map[string]interface{}{
					"name": "English Name",
				},
			},
			"dependencies": map[string]interface{}{
				"echo":     "^1.0.0",
				"customer": ">=2.0.0",
			},
		}

		result, err := types.ToAssistantModel(data)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

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
		if result.ConnectorOptions == nil {
			t.Error("Expected ConnectorOptions to be set")
		} else {
			if result.ConnectorOptions.Optional == nil || !*result.ConnectorOptions.Optional {
				t.Error("Expected ConnectorOptions.Optional to be true")
			}
			if len(result.ConnectorOptions.Connectors) != 2 {
				t.Errorf("Expected 2 connectors in options, got %d", len(result.ConnectorOptions.Connectors))
			}
			if len(result.ConnectorOptions.Filters) != 2 {
				t.Errorf("Expected 2 filters, got %d", len(result.ConnectorOptions.Filters))
			}
		}
		if result.Path != "/path/to/assistant" {
			t.Errorf("Expected Path, got '%s'", result.Path)
		}
		if result.Source != "function hook() { return 'test'; }" {
			t.Errorf("Expected Source, got '%s'", result.Source)
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
		if len(result.Modes) != 2 {
			t.Errorf("Expected 2 modes, got %d", len(result.Modes))
		}
		if result.Modes[0] != "chat" {
			t.Errorf("Expected first mode 'chat', got '%s'", result.Modes[0])
		}
		if result.DefaultMode != "chat" {
			t.Errorf("Expected default_mode 'chat', got '%s'", result.DefaultMode)
		}
		if result.Options == nil {
			t.Error("Expected Options to be set")
		}
		if len(result.Prompts) != 1 {
			t.Errorf("Expected 1 prompt, got %d", len(result.Prompts))
		}
		if result.PromptPresets == nil {
			t.Error("Expected PromptPresets to be set")
		} else {
			if len(result.PromptPresets) != 2 {
				t.Errorf("Expected 2 prompt presets, got %d", len(result.PromptPresets))
			}
			if chatPrompts, ok := result.PromptPresets["chat"]; !ok {
				t.Error("Expected 'chat' prompt preset")
			} else if len(chatPrompts) != 1 {
				t.Errorf("Expected 1 chat prompt, got %d", len(chatPrompts))
			}
			if taskPrompts, ok := result.PromptPresets["task"]; !ok {
				t.Error("Expected 'task' prompt preset")
			} else if len(taskPrompts) != 1 {
				t.Errorf("Expected 1 task prompt, got %d", len(taskPrompts))
			}
		}
		if !result.DisableGlobalPrompts {
			t.Error("Expected DisableGlobalPrompts to be true")
		}
		if result.KB == nil {
			t.Error("Expected KB to be set")
		}
		if result.DB == nil {
			t.Error("Expected DB to be set")
		}
		if result.MCP == nil {
			t.Error("Expected MCP to be set")
		}
		if result.Workflow == nil {
			t.Error("Expected Workflow to be set")
		}
		if result.Placeholder == nil {
			t.Error("Expected Placeholder to be set")
		}
		if result.Locales == nil {
			t.Error("Expected Locales to be set")
		}
		if result.Dependencies == nil {
			t.Error("Expected Dependencies to be set")
		} else {
			if len(result.Dependencies) != 2 {
				t.Errorf("Expected 2 dependencies, got %d", len(result.Dependencies))
			}
			if result.Dependencies["echo"] != "^1.0.0" {
				t.Errorf("Expected echo dependency '^1.0.0', got '%s'", result.Dependencies["echo"])
			}
			if result.Dependencies["customer"] != ">=2.0.0" {
				t.Errorf("Expected customer dependency '>=2.0.0', got '%s'", result.Dependencies["customer"])
			}
		}
	})

	t.Run("MapWithFloatNumbers", func(t *testing.T) {
		data := map[string]interface{}{
			"sort":       float64(150),
			"created_at": float64(1609459200),
			"updated_at": float64(1609459300),
		}

		result, err := types.ToAssistantModel(data)
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
			"modes":        nil,
			"default_mode": "",
			"options":      nil,
			"prompts":      nil,
			"kb":           nil,
			"db":           nil,
			"mcp":          nil,
			"workflow":     nil,
			"placeholder":  nil,
			"locales":      nil,
			"dependencies": nil,
		}

		result, err := types.ToAssistantModel(data)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if result.ID != "test-id" {
			t.Errorf("Expected ID 'test-id', got '%s'", result.ID)
		}
		if result.Tags != nil {
			t.Error("Expected Tags to be nil")
		}
		if result.Dependencies != nil {
			t.Error("Expected Dependencies to be nil")
		}
		if result.Modes != nil {
			t.Error("Expected Modes to be nil")
		}
		if result.DefaultMode != "" {
			t.Error("Expected DefaultMode to be empty")
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

		result, err := types.ToAssistantModel(input)
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
		invalidData := make(chan int)
		_, err := types.ToAssistantModel(invalidData)
		if err == nil {
			t.Error("Expected error for invalid input")
		}
	})

	t.Run("EmptyMap", func(t *testing.T) {
		data := map[string]interface{}{}
		result, err := types.ToAssistantModel(data)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if result == nil {
			t.Error("Expected non-nil result")
		}
		if result.ID != "" {
			t.Errorf("Expected empty ID, got '%s'", result.ID)
		}
	})
}

func TestToAssistantModelNewFields(t *testing.T) {
	t.Run("ConnectorOptions", func(t *testing.T) {
		data := map[string]interface{}{
			"connector_options": map[string]interface{}{
				"optional":   true,
				"connectors": []string{"openai", "anthropic", "azure"},
				"filters":    []string{"vision", "tool_calls", "audio"},
			},
		}

		result, err := types.ToAssistantModel(data)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if result.ConnectorOptions == nil {
			t.Fatal("Expected ConnectorOptions to be set")
		}

		if result.ConnectorOptions.Optional == nil || !*result.ConnectorOptions.Optional {
			t.Error("Expected Optional to be true")
		}

		if len(result.ConnectorOptions.Connectors) != 3 {
			t.Errorf("Expected 3 connectors, got %d", len(result.ConnectorOptions.Connectors))
		}

		if len(result.ConnectorOptions.Filters) != 3 {
			t.Errorf("Expected 3 filters, got %d", len(result.ConnectorOptions.Filters))
		}
	})

	t.Run("PromptPresets", func(t *testing.T) {
		data := map[string]interface{}{
			"prompt_presets": map[string]interface{}{
				"chat": []map[string]interface{}{
					{"role": "system", "content": "You are a helpful chat assistant"},
					{"role": "user", "content": "Example question"},
				},
				"task": []map[string]interface{}{
					{"role": "system", "content": "You are a task completion assistant"},
				},
				"analyze": []map[string]interface{}{
					{"role": "system", "content": "You are a data analysis assistant"},
				},
			},
		}

		result, err := types.ToAssistantModel(data)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if result.PromptPresets == nil {
			t.Fatal("Expected PromptPresets to be set")
		}

		if len(result.PromptPresets) != 3 {
			t.Errorf("Expected 3 prompt preset modes, got %d", len(result.PromptPresets))
		}

		if chatPrompts, ok := result.PromptPresets["chat"]; !ok {
			t.Error("Expected 'chat' mode in prompt presets")
		} else if len(chatPrompts) != 2 {
			t.Errorf("Expected 2 prompts in chat mode, got %d", len(chatPrompts))
		}

		if taskPrompts, ok := result.PromptPresets["task"]; !ok {
			t.Error("Expected 'task' mode in prompt presets")
		} else if len(taskPrompts) != 1 {
			t.Errorf("Expected 1 prompt in task mode, got %d", len(taskPrompts))
		}

		if analyzePrompts, ok := result.PromptPresets["analyze"]; !ok {
			t.Error("Expected 'analyze' mode in prompt presets")
		} else if len(analyzePrompts) != 1 {
			t.Errorf("Expected 1 prompt in analyze mode, got %d", len(analyzePrompts))
		}
	})

	t.Run("Source", func(t *testing.T) {
		hookScript := `
function beforeChat(context) {
  console.log('Hook called');
  return context;
}
`
		data := map[string]interface{}{
			"source": hookScript,
		}

		result, err := types.ToAssistantModel(data)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if result.Source != hookScript {
			t.Errorf("Expected Source to match, got '%s'", result.Source)
		}
	})

	t.Run("AllNewFields", func(t *testing.T) {
		data := map[string]interface{}{
			"connector_options": map[string]interface{}{
				"optional":   true,
				"connectors": []string{"openai"},
				"filters":    []string{"vision"},
			},
			"prompt_presets": map[string]interface{}{
				"chat": []map[string]interface{}{
					{"role": "system", "content": "Chat mode"},
				},
			},
			"disable_global_prompts": true,
			"source":                 "function test() {}",
		}

		result, err := types.ToAssistantModel(data)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if result.ConnectorOptions == nil {
			t.Error("Expected ConnectorOptions to be set")
		}
		if result.PromptPresets == nil {
			t.Error("Expected PromptPresets to be set")
		}
		if !result.DisableGlobalPrompts {
			t.Error("Expected DisableGlobalPrompts to be true")
		}
		if result.Source == "" {
			t.Error("Expected Source to be set")
		}
	})

	t.Run("NilNewFields", func(t *testing.T) {
		data := map[string]interface{}{
			"connector_options":      nil,
			"prompt_presets":         nil,
			"disable_global_prompts": nil,
			"source":                 nil,
		}

		result, err := types.ToAssistantModel(data)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if result.ConnectorOptions != nil {
			t.Error("Expected ConnectorOptions to be nil")
		}
		if result.PromptPresets != nil {
			t.Error("Expected PromptPresets to be nil")
		}
		if result.DisableGlobalPrompts {
			t.Error("Expected DisableGlobalPrompts to be false")
		}
		if result.Source != "" {
			t.Error("Expected Source to be empty")
		}
	})

	t.Run("DisableGlobalPrompts", func(t *testing.T) {
		data := map[string]interface{}{
			"disable_global_prompts": true,
		}
		result, err := types.ToAssistantModel(data)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if !result.DisableGlobalPrompts {
			t.Error("Expected DisableGlobalPrompts to be true")
		}

		data = map[string]interface{}{
			"disable_global_prompts": false,
		}
		result, err = types.ToAssistantModel(data)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if result.DisableGlobalPrompts {
			t.Error("Expected DisableGlobalPrompts to be false")
		}

		data = map[string]interface{}{
			"disable_global_prompts": 1,
		}
		result, err = types.ToAssistantModel(data)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if !result.DisableGlobalPrompts {
			t.Error("Expected DisableGlobalPrompts to be true for int 1")
		}

		data = map[string]interface{}{
			"disable_global_prompts": "true",
		}
		result, err = types.ToAssistantModel(data)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if !result.DisableGlobalPrompts {
			t.Error("Expected DisableGlobalPrompts to be true for string 'true'")
		}
	})
}

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

		result, err := types.ToAssistantModel(data)
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

		result, err := types.ToAssistantModel(data)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if len(result.Prompts) != 2 {
			t.Errorf("Expected 2 prompts, got %d", len(result.Prompts))
		}
	})
}

func TestGetBoolValue(t *testing.T) {
	t.Run("BoolTrue", func(t *testing.T) {
		data := map[string]interface{}{"key": true}
		result := types.GetBoolValueForTest(data, "key")
		if !result {
			t.Error("Expected true")
		}
	})

	t.Run("BoolFalse", func(t *testing.T) {
		data := map[string]interface{}{"key": false}
		result := types.GetBoolValueForTest(data, "key")
		if result {
			t.Error("Expected false")
		}
	})

	t.Run("IntNonZero", func(t *testing.T) {
		data := map[string]interface{}{"key": 1}
		result := types.GetBoolValueForTest(data, "key")
		if !result {
			t.Error("Expected true for non-zero int")
		}
	})

	t.Run("IntZero", func(t *testing.T) {
		data := map[string]interface{}{"key": 0}
		result := types.GetBoolValueForTest(data, "key")
		if result {
			t.Error("Expected false for zero int")
		}
	})

	t.Run("Int64NonZero", func(t *testing.T) {
		data := map[string]interface{}{"key": int64(1)}
		result := types.GetBoolValueForTest(data, "key")
		if !result {
			t.Error("Expected true for non-zero int64")
		}
	})

	t.Run("Int64Zero", func(t *testing.T) {
		data := map[string]interface{}{"key": int64(0)}
		result := types.GetBoolValueForTest(data, "key")
		if result {
			t.Error("Expected false for zero int64")
		}
	})

	t.Run("Float64NonZero", func(t *testing.T) {
		data := map[string]interface{}{"key": float64(1.5)}
		result := types.GetBoolValueForTest(data, "key")
		if !result {
			t.Error("Expected true for non-zero float64")
		}
	})

	t.Run("Float64Zero", func(t *testing.T) {
		data := map[string]interface{}{"key": float64(0)}
		result := types.GetBoolValueForTest(data, "key")
		if result {
			t.Error("Expected false for zero float64")
		}
	})

	t.Run("StringTrue", func(t *testing.T) {
		data := map[string]interface{}{"key": "true"}
		result := types.GetBoolValueForTest(data, "key")
		if !result {
			t.Error("Expected true for string 'true'")
		}
	})

	t.Run("StringOne", func(t *testing.T) {
		data := map[string]interface{}{"key": "1"}
		result := types.GetBoolValueForTest(data, "key")
		if !result {
			t.Error("Expected true for string '1'")
		}
	})

	t.Run("StringFalse", func(t *testing.T) {
		data := map[string]interface{}{"key": "false"}
		result := types.GetBoolValueForTest(data, "key")
		if result {
			t.Error("Expected false for string 'false'")
		}
	})

	t.Run("StringOther", func(t *testing.T) {
		data := map[string]interface{}{"key": "other"}
		result := types.GetBoolValueForTest(data, "key")
		if result {
			t.Error("Expected false for other string values")
		}
	})

	t.Run("NilValue", func(t *testing.T) {
		data := map[string]interface{}{"key": nil}
		result := types.GetBoolValueForTest(data, "key")
		if result {
			t.Error("Expected false for nil value")
		}
	})

	t.Run("MissingKey", func(t *testing.T) {
		data := map[string]interface{}{}
		result := types.GetBoolValueForTest(data, "missing")
		if result {
			t.Error("Expected false for missing key")
		}
	})

	t.Run("UnsupportedType", func(t *testing.T) {
		data := map[string]interface{}{"key": struct{}{}}
		result := types.GetBoolValueForTest(data, "key")
		if result {
			t.Error("Expected false for unsupported type")
		}
	})
}

func TestModelID(t *testing.T) {
	t.Run("WithCustomModel", func(t *testing.T) {
		assistant := types.AssistantModel{
			ID:        "test123",
			Name:      "Test Assistant",
			Connector: "openai",
			Options: map[string]interface{}{
				"model": "gpt-4o",
			},
		}
		result := assistant.ModelID()
		expected := "test-assistant-gpt-4o-yao_test123"
		if result != expected {
			t.Errorf("Expected '%s', got '%s'", expected, result)
		}
	})

	t.Run("WithModelInOptions", func(t *testing.T) {
		assistant := types.AssistantModel{
			ID:        "abc456",
			Name:      "My Bot",
			Connector: "openai",
			Options: map[string]interface{}{
				"model": "gpt-3.5-turbo",
			},
		}
		result := assistant.ModelID()
		expected := "my-bot-gpt-3.5-turbo-yao_abc456"
		if result != expected {
			t.Errorf("Expected '%s', got '%s'", expected, result)
		}
	})

	t.Run("WithoutCustomModel", func(t *testing.T) {
		assistant := types.AssistantModel{
			ID:        "xyz789",
			Name:      "Default Assistant",
			Connector: "openai",
		}
		result := assistant.ModelID()
		expected := "default-assistant-unknown-yao_xyz789"
		if result != expected {
			t.Errorf("Expected '%s', got '%s'", expected, result)
		}
	})

	t.Run("WithoutConnector", func(t *testing.T) {
		assistant := types.AssistantModel{
			ID:   "noconn",
			Name: "No Connector",
		}
		result := assistant.ModelID()
		expected := "no-connector-unknown-yao_noconn"
		if result != expected {
			t.Errorf("Expected '%s', got '%s'", expected, result)
		}
	})

	t.Run("WithSpacesInName", func(t *testing.T) {
		assistant := types.AssistantModel{
			ID:        "spaces",
			Name:      "Test Bot With Spaces",
			Connector: "anthropic",
			Options: map[string]interface{}{
				"model": "claude-3",
			},
		}
		result := assistant.ModelID()
		expected := "test-bot-with-spaces-claude-3-yao_spaces"
		if result != expected {
			t.Errorf("Expected '%s', got '%s'", expected, result)
		}
	})

	t.Run("WithUpperCaseName", func(t *testing.T) {
		assistant := types.AssistantModel{
			ID:        "upper",
			Name:      "UPPERCASE-NAME",
			Connector: "openai",
			Options: map[string]interface{}{
				"model": "GPT-4",
			},
		}
		result := assistant.ModelID()
		expected := "uppercase-name-GPT-4-yao_upper"
		if result != expected {
			t.Errorf("Expected '%s', got '%s'", expected, result)
		}
	})

	t.Run("WithEmptyOptions", func(t *testing.T) {
		assistant := types.AssistantModel{
			ID:        "empty",
			Name:      "Empty Options",
			Connector: "openai",
			Options:   map[string]interface{}{},
		}
		result := assistant.ModelID()
		expected := "empty-options-unknown-yao_empty"
		if result != expected {
			t.Errorf("Expected '%s', got '%s'", expected, result)
		}
	})
}

func TestToConnectorOptions(t *testing.T) {
	t.Run("NilInput", func(t *testing.T) {
		result, err := types.ToConnectorOptions(nil)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if result != nil {
			t.Errorf("Expected nil result, got: %v", result)
		}
	})

	t.Run("ConnectorOptionsPointer", func(t *testing.T) {
		optionalTrue := true
		opts := &types.ConnectorOptions{
			Optional:   &optionalTrue,
			Connectors: []string{"openai", "anthropic"},
			Filters:    []types.ModelCapability{types.CapVision, types.CapToolCalls},
		}
		result, err := types.ToConnectorOptions(opts)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if result != opts {
			t.Errorf("Expected same pointer")
		}
	})

	t.Run("ConnectorOptionsValue", func(t *testing.T) {
		optionalTrue := true
		opts := types.ConnectorOptions{
			Optional:   &optionalTrue,
			Connectors: []string{"openai", "anthropic"},
			Filters:    []types.ModelCapability{types.CapVision, types.CapToolCalls},
		}
		result, err := types.ToConnectorOptions(opts)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if result.Optional == nil || !*result.Optional {
			t.Error("Expected Optional to be true")
		}
		if len(result.Connectors) != 2 {
			t.Errorf("Expected 2 connectors, got %d", len(result.Connectors))
		}
		if len(result.Filters) != 2 {
			t.Errorf("Expected 2 filters, got %d", len(result.Filters))
		}
	})

	t.Run("MapInput", func(t *testing.T) {
		data := map[string]interface{}{
			"optional":   true,
			"connectors": []string{"openai", "anthropic", "azure"},
			"filters":    []string{"vision", "tool_calls", "audio"},
		}
		result, err := types.ToConnectorOptions(data)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if result.Optional == nil || !*result.Optional {
			t.Error("Expected Optional to be true")
		}
		if len(result.Connectors) != 3 {
			t.Errorf("Expected 3 connectors, got %d", len(result.Connectors))
		}
		if len(result.Filters) != 3 {
			t.Errorf("Expected 3 filters, got %d", len(result.Filters))
		}
	})

	t.Run("MapInputOptionalOnly", func(t *testing.T) {
		data := map[string]interface{}{
			"optional": true,
		}
		result, err := types.ToConnectorOptions(data)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if result.Optional == nil || !*result.Optional {
			t.Error("Expected Optional to be true")
		}
		if result.Connectors != nil {
			t.Error("Expected Connectors to be nil")
		}
		if result.Filters != nil {
			t.Error("Expected Filters to be nil")
		}
	})

	t.Run("MapInputOptionalFalse", func(t *testing.T) {
		data := map[string]interface{}{
			"optional":   false,
			"connectors": []string{"openai"},
			"filters":    []string{"vision"},
		}
		result, err := types.ToConnectorOptions(data)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if result.Optional == nil {
			t.Error("Expected Optional to be set")
		} else if *result.Optional {
			t.Error("Expected Optional to be false")
		}
		if len(result.Connectors) != 1 {
			t.Errorf("Expected 1 connector, got %d", len(result.Connectors))
		}
	})

	t.Run("MapInputOptionalNil", func(t *testing.T) {
		data := map[string]interface{}{
			"connectors": []string{"openai"},
		}
		result, err := types.ToConnectorOptions(data)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if result.Optional != nil {
			t.Errorf("Expected Optional to be nil (not set), got: %v", *result.Optional)
		}
		if len(result.Connectors) != 1 {
			t.Errorf("Expected 1 connector, got %d", len(result.Connectors))
		}
	})

	t.Run("InvalidInput", func(t *testing.T) {
		invalidData := make(chan int)
		_, err := types.ToConnectorOptions(invalidData)
		if err == nil {
			t.Error("Expected error for invalid input")
		}
	})

	t.Run("InvalidJSONUnmarshal", func(t *testing.T) {
		data := map[string]interface{}{
			"invalid_field": "should cause unmarshal to fail gracefully",
		}
		result, err := types.ToConnectorOptions(data)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if result == nil {
			t.Error("Expected non-nil result")
		}
	})
}

func TestToModes(t *testing.T) {
	t.Run("NilInput", func(t *testing.T) {
		result, err := types.ToModes(nil)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if result != nil {
			t.Errorf("Expected nil result, got: %v", result)
		}
	})

	t.Run("StringSlice", func(t *testing.T) {
		modes := []string{"chat", "task", "analyze"}
		result, err := types.ToModes(modes)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if len(result) != 3 {
			t.Errorf("Expected 3 modes, got %d", len(result))
		}
		if result[0] != "chat" {
			t.Errorf("Expected 'chat', got '%s'", result[0])
		}
		if result[1] != "task" {
			t.Errorf("Expected 'task', got '%s'", result[1])
		}
		if result[2] != "analyze" {
			t.Errorf("Expected 'analyze', got '%s'", result[2])
		}
	})

	t.Run("InterfaceSlice", func(t *testing.T) {
		modes := []interface{}{"chat", "task", 123}
		result, err := types.ToModes(modes)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if len(result) != 3 {
			t.Errorf("Expected 3 modes, got %d", len(result))
		}
		if result[0] != "chat" {
			t.Errorf("Expected 'chat', got '%s'", result[0])
		}
		if result[2] != "123" {
			t.Errorf("Expected '123', got '%s'", result[2])
		}
	})

	t.Run("SingleString", func(t *testing.T) {
		mode := "chat"
		result, err := types.ToModes(mode)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if len(result) != 1 {
			t.Errorf("Expected 1 mode, got %d", len(result))
		}
		if result[0] != "chat" {
			t.Errorf("Expected 'chat', got '%s'", result[0])
		}
	})

	t.Run("EmptySlice", func(t *testing.T) {
		modes := []string{}
		result, err := types.ToModes(modes)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if len(result) != 0 {
			t.Errorf("Expected 0 modes, got %d", len(result))
		}
	})

	t.Run("InvalidInput", func(t *testing.T) {
		invalidData := make(chan int)
		_, err := types.ToModes(invalidData)
		if err == nil {
			t.Error("Expected error for invalid input")
		}
	})

	t.Run("InvalidJSONUnmarshal", func(t *testing.T) {
		data := map[string]interface{}{
			"invalid": "structure",
		}
		_, err := types.ToModes(data)
		if err == nil {
			t.Error("Expected error for invalid unmarshal")
		}
	})

	t.Run("MixedTypes", func(t *testing.T) {
		modes := []interface{}{"chat", 456, "task", true}
		result, err := types.ToModes(modes)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if len(result) != 4 {
			t.Errorf("Expected 4 modes, got %d", len(result))
		}
		if result[0] != "chat" {
			t.Errorf("Expected 'chat', got '%s'", result[0])
		}
		if result[1] != "456" {
			t.Errorf("Expected '456', got '%s'", result[1])
		}
		if result[2] != "task" {
			t.Errorf("Expected 'task', got '%s'", result[2])
		}
		if result[3] != "true" {
			t.Errorf("Expected 'true', got '%s'", result[3])
		}
	})
}

func TestToPromptPresets(t *testing.T) {
	t.Run("NilInput", func(t *testing.T) {
		result, err := types.ToPromptPresets(nil)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if result != nil {
			t.Errorf("Expected nil result, got: %v", result)
		}
	})

	t.Run("MapStringPromptSlice", func(t *testing.T) {
		presets := map[string][]types.Prompt{
			"chat": {
				{Role: "system", Content: "You are a chat assistant"},
			},
			"task": {
				{Role: "system", Content: "You are a task assistant"},
			},
		}
		result, err := types.ToPromptPresets(presets)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if len(result) != 2 {
			t.Errorf("Expected 2 presets, got %d", len(result))
		}
		if len(result["chat"]) != 1 {
			t.Errorf("Expected 1 chat prompt, got %d", len(result["chat"]))
		}
		if len(result["task"]) != 1 {
			t.Errorf("Expected 1 task prompt, got %d", len(result["task"]))
		}
	})

	t.Run("MapInput", func(t *testing.T) {
		data := map[string]interface{}{
			"chat": []interface{}{
				map[string]interface{}{"role": "system", "content": "Chat mode system prompt"},
				map[string]interface{}{"role": "user", "content": "Example user message"},
			},
			"task": []interface{}{
				map[string]interface{}{"role": "system", "content": "Task mode system prompt"},
			},
			"analyze": []interface{}{
				map[string]interface{}{"role": "system", "content": "Analyze mode system prompt"},
			},
		}
		result, err := types.ToPromptPresets(data)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if len(result) != 3 {
			t.Errorf("Expected 3 presets, got %d", len(result))
		}
		if len(result["chat"]) != 2 {
			t.Errorf("Expected 2 chat prompts, got %d", len(result["chat"]))
		}
		if len(result["task"]) != 1 {
			t.Errorf("Expected 1 task prompt, got %d", len(result["task"]))
		}
		if len(result["analyze"]) != 1 {
			t.Errorf("Expected 1 analyze prompt, got %d", len(result["analyze"]))
		}
	})

	t.Run("EmptyMap", func(t *testing.T) {
		data := map[string]interface{}{}
		result, err := types.ToPromptPresets(data)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if result == nil {
			t.Error("Expected non-nil result")
		}
		if len(result) != 0 {
			t.Errorf("Expected empty map, got %d entries", len(result))
		}
	})

	t.Run("SinglePreset", func(t *testing.T) {
		data := map[string]interface{}{
			"default": []interface{}{
				map[string]interface{}{"role": "system", "content": "Default prompt"},
			},
		}
		result, err := types.ToPromptPresets(data)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if len(result) != 1 {
			t.Errorf("Expected 1 preset, got %d", len(result))
		}
		if _, ok := result["default"]; !ok {
			t.Error("Expected 'default' key in result")
		}
	})

	t.Run("InvalidInput", func(t *testing.T) {
		invalidData := make(chan int)
		_, err := types.ToPromptPresets(invalidData)
		if err == nil {
			t.Error("Expected error for invalid input")
		}
	})

	t.Run("InvalidJSONUnmarshal", func(t *testing.T) {
		data := "not a map"
		_, err := types.ToPromptPresets(data)
		if err == nil {
			t.Error("Expected error for invalid JSON unmarshal")
		}
	})

	t.Run("PromptWithAllFields", func(t *testing.T) {
		data := map[string]interface{}{
			"advanced": []interface{}{
				map[string]interface{}{
					"role":    "system",
					"content": "Advanced system prompt",
					"name":    "system-prompt",
				},
				map[string]interface{}{
					"role":    "user",
					"content": "User example",
					"name":    "user-example",
				},
				map[string]interface{}{
					"role":    "assistant",
					"content": "Assistant response",
					"name":    "assistant-response",
				},
			},
		}
		result, err := types.ToPromptPresets(data)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if len(result["advanced"]) != 3 {
			t.Errorf("Expected 3 prompts in advanced, got %d", len(result["advanced"]))
		}
		if result["advanced"][0].Role != "system" {
			t.Errorf("Expected role 'system', got '%s'", result["advanced"][0].Role)
		}
		if result["advanced"][0].Content != "Advanced system prompt" {
			t.Errorf("Expected content 'Advanced system prompt', got '%s'", result["advanced"][0].Content)
		}
	})
}

func TestParseModelID(t *testing.T) {
	t.Run("ValidModelID", func(t *testing.T) {
		modelID := "test-assistant-gpt-4o-yao_test123"
		result := types.ParseModelID(modelID)
		expected := "test123"
		if result != expected {
			t.Errorf("Expected '%s', got '%s'", expected, result)
		}
	})

	t.Run("ValidModelIDWithMultipleDashes", func(t *testing.T) {
		modelID := "my-test-bot-gpt-3.5-turbo-yao_abc456"
		result := types.ParseModelID(modelID)
		expected := "abc456"
		if result != expected {
			t.Errorf("Expected '%s', got '%s'", expected, result)
		}
	})

	t.Run("ValidModelIDWithHyphenInID", func(t *testing.T) {
		modelID := "assistant-name-model-yao_id-with-dash"
		result := types.ParseModelID(modelID)
		expected := "id-with-dash"
		if result != expected {
			t.Errorf("Expected '%s', got '%s'", expected, result)
		}
	})

	t.Run("InvalidModelIDNoYaoPrefix", func(t *testing.T) {
		modelID := "test-assistant-gpt-4o-test123"
		result := types.ParseModelID(modelID)
		if result != "" {
			t.Errorf("Expected empty string, got '%s'", result)
		}
	})

	t.Run("InvalidModelIDEmpty", func(t *testing.T) {
		modelID := ""
		result := types.ParseModelID(modelID)
		if result != "" {
			t.Errorf("Expected empty string, got '%s'", result)
		}
	})

	t.Run("InvalidModelIDOnlyYaoPrefix", func(t *testing.T) {
		modelID := "yao_"
		result := types.ParseModelID(modelID)
		if result != "" {
			t.Errorf("Expected empty string, got '%s'", result)
		}
	})

	t.Run("RoundTrip", func(t *testing.T) {
		assistant := types.AssistantModel{
			ID:        "roundtrip123",
			Name:      "Round Trip Test",
			Connector: "openai",
			Options: map[string]interface{}{
				"model": "gpt-4",
			},
		}
		modelID := assistant.ModelID()
		extractedID := types.ParseModelID(modelID)
		if extractedID != assistant.ID {
			t.Errorf("Round trip failed: expected '%s', got '%s'", assistant.ID, extractedID)
		}
	})
}
