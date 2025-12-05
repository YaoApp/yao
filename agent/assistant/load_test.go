package assistant

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	store "github.com/yaoapp/yao/agent/store/types"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/test"
)

func prepare(t *testing.T) {
	test.Prepare(t, config.Conf)
}

// TestLoadPath tests loading assistant from path
func TestLoadPath(t *testing.T) {
	prepare(t)
	defer test.Clean()

	t.Run("LoadFullFieldsAssistant", func(t *testing.T) {
		assistant, err := LoadPath("/assistants/tests/fullfields")
		require.NoError(t, err)
		require.NotNil(t, assistant)

		// Basic fields
		assert.Equal(t, "tests.fullfields", assistant.ID)
		assert.Equal(t, "Full Fields Test Assistant", assistant.Name)
		assert.Equal(t, "assistant", assistant.Type)
		assert.Equal(t, "/api/__yao/app/icons/app.png", assistant.Avatar)
		assert.Equal(t, "gpt-4o", assistant.Connector)
		assert.Equal(t, "/assistants/tests/fullfields", assistant.Path)
		assert.Equal(t, "Test assistant with all available fields for unit testing", assistant.Description)

		// Boolean fields
		assert.True(t, assistant.Public)
		assert.True(t, assistant.Readonly)
		assert.True(t, assistant.Mentionable)
		assert.False(t, assistant.Automated)
		assert.True(t, assistant.DisableGlobalPrompts)

		// Share field
		assert.Equal(t, "team", assistant.Share)

		// Sort field
		assert.Equal(t, 100, assistant.Sort)

		// Tags
		assert.NotNil(t, assistant.Tags)
		assert.Contains(t, assistant.Tags, "Test")
		assert.Contains(t, assistant.Tags, "Development")
		assert.Contains(t, assistant.Tags, "FullFields")

		// Options
		assert.NotNil(t, assistant.Options)
		assert.Equal(t, 0.7, assistant.Options["temperature"])
		assert.Equal(t, float64(2000), assistant.Options["max_tokens"])

		// Prompts (default prompts from prompts.yml)
		assert.NotNil(t, assistant.Prompts)
		assert.GreaterOrEqual(t, len(assistant.Prompts), 1)
		assert.Equal(t, "system", assistant.Prompts[0].Role)

		// Script (from src/index.ts)
		assert.NotNil(t, assistant.HookScript)
	})

	t.Run("LoadConnectorOptions", func(t *testing.T) {
		assistant, err := LoadPath("/assistants/tests/fullfields")
		require.NoError(t, err)
		require.NotNil(t, assistant)

		// ConnectorOptions
		assert.NotNil(t, assistant.ConnectorOptions)
		assert.NotNil(t, assistant.ConnectorOptions.Optional)
		assert.True(t, *assistant.ConnectorOptions.Optional)
		assert.NotNil(t, assistant.ConnectorOptions.Connectors)
		assert.Contains(t, assistant.ConnectorOptions.Connectors, "gpt-4o")
		assert.Contains(t, assistant.ConnectorOptions.Connectors, "gpt-4o-mini")
		assert.Contains(t, assistant.ConnectorOptions.Connectors, "deepseek")
		assert.NotNil(t, assistant.ConnectorOptions.Filters)
		assert.Len(t, assistant.ConnectorOptions.Filters, 2)
	})

	t.Run("LoadPromptPresets", func(t *testing.T) {
		assistant, err := LoadPath("/assistants/tests/fullfields")
		require.NoError(t, err)
		require.NotNil(t, assistant)

		// PromptPresets (from prompts directory)
		assert.NotNil(t, assistant.PromptPresets)

		// Top-level presets: chat.yml -> "chat", task.yml -> "task"
		chatPreset, hasChat := assistant.PromptPresets["chat"]
		assert.True(t, hasChat, "Should have 'chat' preset")
		assert.NotEmpty(t, chatPreset)

		taskPreset, hasTask := assistant.PromptPresets["task"]
		assert.True(t, hasTask, "Should have 'task' preset")
		assert.NotEmpty(t, taskPreset)

		// Nested presets: chat/friendly.yml -> "chat.friendly"
		friendlyPreset, hasFriendly := assistant.PromptPresets["chat.friendly"]
		assert.True(t, hasFriendly, "Should have 'chat.friendly' preset")
		assert.NotEmpty(t, friendlyPreset)

		professionalPreset, hasProfessional := assistant.PromptPresets["chat.professional"]
		assert.True(t, hasProfessional, "Should have 'chat.professional' preset")
		assert.NotEmpty(t, professionalPreset)

		// task/analysis.yml -> "task.analysis"
		analysisPreset, hasAnalysis := assistant.PromptPresets["task.analysis"]
		assert.True(t, hasAnalysis, "Should have 'task.analysis' preset")
		assert.NotEmpty(t, analysisPreset)
	})

	t.Run("LoadKnowledgeBase", func(t *testing.T) {
		assistant, err := LoadPath("/assistants/tests/fullfields")
		require.NoError(t, err)
		require.NotNil(t, assistant)

		// KB
		assert.NotNil(t, assistant.KB)
		assert.NotNil(t, assistant.KB.Collections)
		assert.Contains(t, assistant.KB.Collections, "test-collection")
		assert.NotNil(t, assistant.KB.Options)
		assert.Equal(t, float64(5), assistant.KB.Options["top_k"])
	})

	t.Run("LoadMCPServers", func(t *testing.T) {
		assistant, err := LoadPath("/assistants/tests/fullfields")
		require.NoError(t, err)
		require.NotNil(t, assistant)

		// MCP
		assert.NotNil(t, assistant.MCP)
		assert.NotNil(t, assistant.MCP.Servers)
		assert.Len(t, assistant.MCP.Servers, 1)
		assert.Equal(t, "echo", assistant.MCP.Servers[0].ServerID)
		assert.Contains(t, assistant.MCP.Servers[0].Tools, "ping")
		assert.Contains(t, assistant.MCP.Servers[0].Tools, "echo")
	})

	t.Run("LoadWorkflow", func(t *testing.T) {
		assistant, err := LoadPath("/assistants/tests/fullfields")
		require.NoError(t, err)
		require.NotNil(t, assistant)

		// Workflow
		assert.NotNil(t, assistant.Workflow)
		assert.NotNil(t, assistant.Workflow.Workflows)
		assert.Contains(t, assistant.Workflow.Workflows, "test-workflow")
		assert.NotNil(t, assistant.Workflow.Options)
		assert.Equal(t, float64(10), assistant.Workflow.Options["max_steps"])
	})

	t.Run("LoadPlaceholder", func(t *testing.T) {
		assistant, err := LoadPath("/assistants/tests/fullfields")
		require.NoError(t, err)
		require.NotNil(t, assistant)

		// Placeholder
		assert.NotNil(t, assistant.Placeholder)
		assert.Equal(t, "Full Fields Test", assistant.Placeholder.Title)
		assert.Equal(t, "Test assistant with complete field coverage", assistant.Placeholder.Description)
		assert.NotNil(t, assistant.Placeholder.Prompts)
		assert.Len(t, assistant.Placeholder.Prompts, 3)
	})

	t.Run("LoadLocales", func(t *testing.T) {
		assistant, err := LoadPath("/assistants/tests/fullfields")
		require.NoError(t, err)
		require.NotNil(t, assistant)

		// Locales
		assert.NotNil(t, assistant.Locales)

		enLocale, hasEn := assistant.Locales["en-us"]
		assert.True(t, hasEn, "Should have en-us locale")
		assert.NotNil(t, enLocale)

		zhLocale, hasZh := assistant.Locales["zh-cn"]
		assert.True(t, hasZh, "Should have zh-cn locale")
		assert.NotNil(t, zhLocale)
	})

	t.Run("LoadNonExistentAssistant", func(t *testing.T) {
		_, err := LoadPath("/assistants/non-existent")
		assert.Error(t, err)
	})
}

// TestLoadPathMCPTest tests loading the MCP test assistant
func TestLoadPathMCPTest(t *testing.T) {
	prepare(t)
	defer test.Clean()

	assistant, err := LoadPath("/assistants/tests/mcptest")
	require.NoError(t, err)
	require.NotNil(t, assistant)

	assert.Equal(t, "tests.mcptest", assistant.ID)
	assert.Equal(t, "MCP Test Assistant", assistant.Name)
	assert.Equal(t, "gpt-4o", assistant.Connector)

	// MCP configuration
	assert.NotNil(t, assistant.MCP)
	assert.Len(t, assistant.MCP.Servers, 1)
	assert.Equal(t, "echo", assistant.MCP.Servers[0].ServerID)

	// Locales
	assert.NotNil(t, assistant.Locales)
	assert.Contains(t, assistant.Locales, "en-us")
	assert.Contains(t, assistant.Locales, "zh-cn")
}

// TestLoadPathBuildRequest tests loading the build request test assistant
func TestLoadPathBuildRequest(t *testing.T) {
	prepare(t)
	defer test.Clean()

	assistant, err := LoadPath("/assistants/tests/buildrequest")
	require.NoError(t, err)
	require.NotNil(t, assistant)

	assert.Equal(t, "tests.buildrequest", assistant.ID)
	assert.Equal(t, "Build Request Test", assistant.Name)

	// HookScript should be loaded
	assert.NotNil(t, assistant.HookScript)

	// Options
	assert.NotNil(t, assistant.Options)
	assert.Equal(t, 0.5, assistant.Options["temperature"])
}

// TestCache tests the assistant cache functionality
func TestCache(t *testing.T) {
	// Clear any existing cache
	ClearCache()

	// Set small cache for testing
	SetCache(3)
	assert.NotNil(t, loaded)

	// Create test assistants
	ast1 := &Assistant{AssistantModel: store.AssistantModel{ID: "id1", Name: "Assistant 1"}}
	ast2 := &Assistant{AssistantModel: store.AssistantModel{ID: "id2", Name: "Assistant 2"}}
	ast3 := &Assistant{AssistantModel: store.AssistantModel{ID: "id3", Name: "Assistant 3"}}
	ast4 := &Assistant{AssistantModel: store.AssistantModel{ID: "id4", Name: "Assistant 4"}}

	t.Run("PutAndGet", func(t *testing.T) {
		loaded.Put(ast1)
		assert.Equal(t, 1, loaded.Len())

		cached, exists := loaded.Get("id1")
		assert.True(t, exists)
		assert.Equal(t, ast1, cached)
	})

	t.Run("CacheEviction", func(t *testing.T) {
		loaded.Put(ast2)
		loaded.Put(ast3)
		assert.Equal(t, 3, loaded.Len())

		// Access ast1 to make it recently used
		loaded.Get("id1")

		// Add ast4, should evict ast2 (least recently used)
		loaded.Put(ast4)
		assert.Equal(t, 3, loaded.Len())

		_, exists := loaded.Get("id2")
		assert.False(t, exists, "ast2 should be evicted")

		_, exists = loaded.Get("id1")
		assert.True(t, exists, "ast1 should still exist")

		_, exists = loaded.Get("id4")
		assert.True(t, exists, "ast4 should exist")
	})

	t.Run("ClearCache", func(t *testing.T) {
		ClearCache()
		assert.Nil(t, loaded)
	})

	t.Run("SetCacheAfterClear", func(t *testing.T) {
		SetCache(100)
		assert.NotNil(t, loaded)
	})
}

// TestClone tests the assistant Clone method
func TestClone(t *testing.T) {
	prepare(t)
	defer test.Clean()

	t.Run("CloneFullFieldsAssistant", func(t *testing.T) {
		original, err := LoadPath("/assistants/tests/fullfields")
		require.NoError(t, err)

		clone := original.Clone()
		require.NotNil(t, clone)

		// Basic fields should be equal
		assert.Equal(t, original.ID, clone.ID)
		assert.Equal(t, original.Name, clone.Name)
		assert.Equal(t, original.Type, clone.Type)
		assert.Equal(t, original.Connector, clone.Connector)
		assert.Equal(t, original.Description, clone.Description)

		// Verify deep copy - modifying original should not affect clone
		if len(original.Tags) > 0 {
			originalTag := original.Tags[0]
			original.Tags[0] = "modified"
			assert.NotEqual(t, original.Tags[0], clone.Tags[0])
			original.Tags[0] = originalTag // restore
		}

		if original.Options != nil {
			original.Options["test_key"] = "test_value"
			_, exists := clone.Options["test_key"]
			assert.False(t, exists, "Clone should not have modified key")
			delete(original.Options, "test_key") // cleanup
		}
	})

	t.Run("CloneNil", func(t *testing.T) {
		var nilAssistant *Assistant
		assert.Nil(t, nilAssistant.Clone())
	})
}

// TestUpdate tests the assistant Update method
func TestUpdate(t *testing.T) {
	prepare(t)
	defer test.Clean()

	t.Run("UpdateBasicFields", func(t *testing.T) {
		assistant, err := LoadPath("/assistants/tests/fullfields")
		require.NoError(t, err)

		updates := map[string]interface{}{
			"name":        "Updated Name",
			"description": "Updated description",
			"tags":        []string{"updated", "tags"},
		}

		err = assistant.Update(updates)
		require.NoError(t, err)

		assert.Equal(t, "Updated Name", assistant.Name)
		assert.Equal(t, "Updated description", assistant.Description)
		assert.Equal(t, []string{"updated", "tags"}, assistant.Tags)
	})

	t.Run("UpdateConnectorOptions", func(t *testing.T) {
		assistant, err := LoadPath("/assistants/tests/fullfields")
		require.NoError(t, err)

		updates := map[string]interface{}{
			"connector_options": map[string]interface{}{
				"optional":   false,
				"connectors": []string{"new-connector"},
			},
		}

		err = assistant.Update(updates)
		require.NoError(t, err)

		assert.NotNil(t, assistant.ConnectorOptions)
		assert.NotNil(t, assistant.ConnectorOptions.Optional)
		assert.False(t, *assistant.ConnectorOptions.Optional)
		assert.Contains(t, assistant.ConnectorOptions.Connectors, "new-connector")
	})

	t.Run("UpdatePromptPresets", func(t *testing.T) {
		assistant, err := LoadPath("/assistants/tests/fullfields")
		require.NoError(t, err)

		updates := map[string]interface{}{
			"prompt_presets": map[string]interface{}{
				"custom": []map[string]interface{}{
					{"role": "system", "content": "Custom preset"},
				},
			},
		}

		err = assistant.Update(updates)
		require.NoError(t, err)

		assert.NotNil(t, assistant.PromptPresets)
		customPreset, exists := assistant.PromptPresets["custom"]
		assert.True(t, exists)
		assert.Len(t, customPreset, 1)
	})

	t.Run("UpdateSource", func(t *testing.T) {
		assistant, err := LoadPath("/assistants/tests/fullfields")
		require.NoError(t, err)

		updates := map[string]interface{}{
			"source": "function Create(ctx, messages) { return { messages: messages }; }",
		}

		err = assistant.Update(updates)
		require.NoError(t, err)

		assert.Equal(t, "function Create(ctx, messages) { return { messages: messages }; }", assistant.Source)
	})

	t.Run("UpdateNilAssistant", func(t *testing.T) {
		var nilAssistant *Assistant
		err := nilAssistant.Update(map[string]interface{}{"name": "test"})
		assert.Error(t, err)
	})
}

// TestMap tests the assistant Map method
func TestMap(t *testing.T) {
	prepare(t)
	defer test.Clean()

	assistant, err := LoadPath("/assistants/tests/fullfields")
	require.NoError(t, err)

	m := assistant.Map()
	require.NotNil(t, m)

	// Check all fields are present
	assert.Equal(t, assistant.ID, m["assistant_id"])
	assert.Equal(t, assistant.Name, m["name"])
	assert.Equal(t, assistant.Type, m["type"])
	assert.Equal(t, assistant.Connector, m["connector"])
	assert.Equal(t, assistant.Description, m["description"])
	assert.Equal(t, assistant.Path, m["path"])
	assert.Equal(t, assistant.Tags, m["tags"])
	assert.Equal(t, assistant.Options, m["options"])
	assert.Equal(t, assistant.Prompts, m["prompts"])
	assert.Equal(t, assistant.KB, m["kb"])
	assert.Equal(t, assistant.MCP, m["mcp"])
	assert.Equal(t, assistant.Workflow, m["workflow"])
	assert.Equal(t, assistant.Placeholder, m["placeholder"])
	assert.Equal(t, assistant.Locales, m["locales"])

	// New fields
	assert.Equal(t, assistant.ConnectorOptions, m["connector_options"])
	assert.Equal(t, assistant.PromptPresets, m["prompt_presets"])
	assert.Equal(t, assistant.Source, m["source"])
}

// TestValidate tests the assistant Validate method
func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		ast     *Assistant
		wantErr bool
	}{
		{
			name: "ValidAssistant",
			ast: &Assistant{
				AssistantModel: store.AssistantModel{
					ID:        "test-id",
					Name:      "Test Assistant",
					Connector: "gpt-4o",
				},
			},
			wantErr: false,
		},
		{
			name: "MissingID",
			ast: &Assistant{
				AssistantModel: store.AssistantModel{
					Name:      "Test Assistant",
					Connector: "gpt-4o",
				},
			},
			wantErr: true,
		},
		{
			name: "MissingName",
			ast: &Assistant{
				AssistantModel: store.AssistantModel{
					ID:        "test-id",
					Connector: "gpt-4o",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.ast.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
