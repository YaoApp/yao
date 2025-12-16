package assistant_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent"
	"github.com/yaoapp/yao/agent/assistant"
	store "github.com/yaoapp/yao/agent/store/types"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/test"
)

func prepare(t *testing.T) {
	test.Prepare(t, config.Conf)
}

func prepareAgent(t *testing.T) {
	test.Prepare(t, config.Conf)
	err := agent.Load(config.Conf)
	require.NoError(t, err, "agent.Load should succeed")
}

// TestLoadPath tests loading assistant from path
func TestLoadPath(t *testing.T) {
	prepare(t)
	defer test.Clean()

	t.Run("LoadFullFieldsAssistant", func(t *testing.T) {
		assistant, err := assistant.LoadPath("/assistants/tests/fullfields")
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
		assistant, err := assistant.LoadPath("/assistants/tests/fullfields")
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
		assistant, err := assistant.LoadPath("/assistants/tests/fullfields")
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
		assistant, err := assistant.LoadPath("/assistants/tests/fullfields")
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
		assistant, err := assistant.LoadPath("/assistants/tests/fullfields")
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
		assistant, err := assistant.LoadPath("/assistants/tests/fullfields")
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
		assistant, err := assistant.LoadPath("/assistants/tests/fullfields")
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
		assistant, err := assistant.LoadPath("/assistants/tests/fullfields")
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
		_, err := assistant.LoadPath("/assistants/non-existent")
		assert.Error(t, err)
	})
}

// TestLoadPathMCPTest tests loading the MCP test assistant
func TestLoadPathMCPTest(t *testing.T) {
	prepare(t)
	defer test.Clean()

	assistant, err := assistant.LoadPath("/assistants/tests/mcptest")
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

	assistant, err := assistant.LoadPath("/assistants/tests/buildrequest")
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
	assistant.ClearCache()

	// Set small cache for testing
	assistant.SetCache(3)
	assert.NotNil(t, assistant.GetCache())

	// Create test assistants
	ast1 := &assistant.Assistant{AssistantModel: store.AssistantModel{ID: "id1", Name: "Assistant 1"}}
	ast2 := &assistant.Assistant{AssistantModel: store.AssistantModel{ID: "id2", Name: "Assistant 2"}}
	ast3 := &assistant.Assistant{AssistantModel: store.AssistantModel{ID: "id3", Name: "Assistant 3"}}
	ast4 := &assistant.Assistant{AssistantModel: store.AssistantModel{ID: "id4", Name: "Assistant 4"}}

	t.Run("PutAndGet", func(t *testing.T) {
		assistant.GetCache().Put(ast1)
		assert.Equal(t, 1, assistant.GetCache().Len())

		cached, exists := assistant.GetCache().Get("id1")
		assert.True(t, exists)
		assert.Equal(t, ast1, cached)
	})

	t.Run("CacheEviction", func(t *testing.T) {
		assistant.GetCache().Put(ast2)
		assistant.GetCache().Put(ast3)
		assert.Equal(t, 3, assistant.GetCache().Len())

		// Access ast1 to make it recently used
		assistant.GetCache().Get("id1")

		// Add ast4, should evict ast2 (least recently used)
		assistant.GetCache().Put(ast4)
		assert.Equal(t, 3, assistant.GetCache().Len())

		_, exists := assistant.GetCache().Get("id2")
		assert.False(t, exists, "ast2 should be evicted")

		_, exists = assistant.GetCache().Get("id1")
		assert.True(t, exists, "ast1 should still exist")

		_, exists = assistant.GetCache().Get("id4")
		assert.True(t, exists, "ast4 should exist")
	})

	t.Run("ClearCache", func(t *testing.T) {
		assistant.ClearCache()
		assert.Nil(t, assistant.GetCache())
	})

	t.Run("SetCacheAfterClear", func(t *testing.T) {
		assistant.SetCache(100)
		assert.NotNil(t, assistant.GetCache())
	})
}

// TestClone tests the assistant Clone method
func TestClone(t *testing.T) {
	prepare(t)
	defer test.Clean()

	t.Run("CloneFullFieldsAssistant", func(t *testing.T) {
		original, err := assistant.LoadPath("/assistants/tests/fullfields")
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
		var nilAssistant *assistant.Assistant
		assert.Nil(t, nilAssistant.Clone())
	})
}

// TestUpdate tests the assistant Update method
func TestUpdate(t *testing.T) {
	prepare(t)
	defer test.Clean()

	t.Run("UpdateBasicFields", func(t *testing.T) {
		assistant, err := assistant.LoadPath("/assistants/tests/fullfields")
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
		assistant, err := assistant.LoadPath("/assistants/tests/fullfields")
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
		assistant, err := assistant.LoadPath("/assistants/tests/fullfields")
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
		assistant, err := assistant.LoadPath("/assistants/tests/fullfields")
		require.NoError(t, err)

		updates := map[string]interface{}{
			"source": "function Create(ctx, messages) { return { messages: messages }; }",
		}

		err = assistant.Update(updates)
		require.NoError(t, err)

		assert.Equal(t, "function Create(ctx, messages) { return { messages: messages }; }", assistant.Source)
	})

	t.Run("UpdateNilAssistant", func(t *testing.T) {
		var nilAssistant *assistant.Assistant
		err := nilAssistant.Update(map[string]interface{}{"name": "test"})
		assert.Error(t, err)
	})
}

// TestMap tests the assistant Map method
func TestMap(t *testing.T) {
	prepare(t)
	defer test.Clean()

	assistant, err := assistant.LoadPath("/assistants/tests/fullfields")
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

// TestLoadSystemAgents tests loading system agents from bindata
func TestLoadSystemAgents(t *testing.T) {
	prepareAgent(t)
	defer test.Clean()

	// Clear cache first
	assistant.ClearCache()
	assistant.SetCache(200)

	t.Run("LoadSystemAgents", func(t *testing.T) {
		err := assistant.LoadSystemAgents()
		require.NoError(t, err)

		// Check __yao.keyword
		keywordAst, keywordExists := assistant.GetCache().Get("__yao.keyword")
		require.True(t, keywordExists, "__yao.keyword should be loaded")
		assert.Equal(t, "__yao.keyword", keywordAst.ID)
		assert.Equal(t, "Keyword Extractor", keywordAst.Name)
		assert.True(t, keywordAst.Readonly)
		assert.True(t, keywordAst.BuiltIn)
		assert.Contains(t, keywordAst.Tags, "system")
		assert.NotNil(t, keywordAst.Prompts)
		assert.Greater(t, len(keywordAst.Prompts), 0)

		// Check __yao.querydsl
		querydslAst, querydslExists := assistant.GetCache().Get("__yao.querydsl")
		require.True(t, querydslExists, "__yao.querydsl should be loaded")
		assert.Equal(t, "__yao.querydsl", querydslAst.ID)
		assert.Equal(t, "Query Builder", querydslAst.Name)
		assert.True(t, querydslAst.Readonly)
		assert.True(t, querydslAst.BuiltIn)
		assert.Contains(t, querydslAst.Tags, "system")
		assert.NotNil(t, querydslAst.Prompts)
		assert.Greater(t, len(querydslAst.Prompts), 0)

		// Check __yao.title
		titleAst, titleExists := assistant.GetCache().Get("__yao.title")
		require.True(t, titleExists, "__yao.title should be loaded")
		assert.Equal(t, "__yao.title", titleAst.ID)
		assert.Equal(t, "Title Generator", titleAst.Name)
		assert.True(t, titleAst.Readonly)
		assert.True(t, titleAst.BuiltIn)

		// Check __yao.prompt
		promptAst, promptExists := assistant.GetCache().Get("__yao.prompt")
		require.True(t, promptExists, "__yao.prompt should be loaded")
		assert.Equal(t, "__yao.prompt", promptAst.ID)
		assert.Equal(t, "Prompt Optimizer", promptAst.Name)
		assert.True(t, promptAst.Readonly)
		assert.True(t, promptAst.BuiltIn)

		// Check __yao.needsearch
		needsearchAst, needsearchExists := assistant.GetCache().Get("__yao.needsearch")
		require.True(t, needsearchExists, "__yao.needsearch should be loaded")
		assert.Equal(t, "__yao.needsearch", needsearchAst.ID)
		assert.Equal(t, "Reference Checker", needsearchAst.Name)
		assert.True(t, needsearchAst.Readonly)
		assert.True(t, needsearchAst.BuiltIn)
	})

	t.Run("SystemAgentsSavedToStorage", func(t *testing.T) {
		// System agents should be saved to storage
		require.NotNil(t, assistant.GetStore(), "storage should be initialized")

		// Check __yao.keyword in storage
		builtIn := true
		tags := []string{"system"}
		res, err := assistant.GetStore().GetAssistants(store.AssistantFilter{
			BuiltIn: &builtIn,
			Tags:    tags,
			Select:  []string{"assistant_id", "name"},
		})
		require.NoError(t, err)
		require.Greater(t, len(res.Data), 0, "System agents should be in storage")

		// Verify at least one system agent exists
		found := false
		for _, ast := range res.Data {
			if ast.ID == "__yao.keyword" || ast.ID == "__yao.querydsl" {
				found = true
				break
			}
		}
		assert.True(t, found, "System agents should be found in storage")
	})

	t.Run("SystemAgentsGetFromStorage", func(t *testing.T) {
		// Clear cache to force loading from storage
		assistant.GetCache().Clear()

		// Test Get for each system agent
		systemAgents := []string{
			"__yao.keyword",
			"__yao.querydsl",
			"__yao.title",
			"__yao.prompt",
			"__yao.needsearch",
			"__yao.entity",
		}

		for _, agentID := range systemAgents {
			ast, err := assistant.Get(agentID)
			require.NoError(t, err, "Get(%s) should succeed", agentID)
			require.NotNil(t, ast, "Get(%s) should return assistant", agentID)
			assert.Equal(t, agentID, ast.ID)
			assert.True(t, ast.BuiltIn, "%s should be built-in", agentID)
			assert.True(t, ast.Readonly, "%s should be readonly", agentID)
			assert.Contains(t, ast.Tags, "system", "%s should have system tag", agentID)
			assert.Equal(t, "worker", ast.Type, "%s should be worker type", agentID)
			assert.NotNil(t, ast.Prompts, "%s should have prompts", agentID)
			assert.Greater(t, len(ast.Prompts), 0, "%s should have at least one prompt", agentID)
		}
	})
}

// TestValidate tests the assistant Validate method
func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		ast     *assistant.Assistant
		wantErr bool
	}{
		{
			name: "ValidAssistant",
			ast: &assistant.Assistant{
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
			ast: &assistant.Assistant{
				AssistantModel: store.AssistantModel{
					Name:      "Test Assistant",
					Connector: "gpt-4o",
				},
			},
			wantErr: true,
		},
		{
			name: "MissingName",
			ast: &assistant.Assistant{
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
