//go:build integration

package assistant_test

import (
	stdContext "context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/assistant"
	"github.com/yaoapp/yao/agent/context"
	searchTypes "github.com/yaoapp/yao/agent/search/types"
	store "github.com/yaoapp/yao/agent/store/types"
	"github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

// ---------------------------------------------------------------------------
// load_test.go.bak — TestCache
// ---------------------------------------------------------------------------

func TestCache(t *testing.T) {
	assistant.ClearCache()
	assistant.SetCache(3)
	assert.NotNil(t, assistant.GetCache())

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

		assistant.GetCache().Get("id1")

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

// ---------------------------------------------------------------------------
// load_test.go.bak — TestClone
// ---------------------------------------------------------------------------

func TestClone(t *testing.T) {
	testprepare.PrepareSandbox(t)

	t.Run("CloneFullFieldsAssistant", func(t *testing.T) {
		original, err := assistant.LoadPath("/assistants/tests/fullfields")
		require.NoError(t, err)

		clone := original.Clone()
		require.NotNil(t, clone)

		assert.Equal(t, original.ID, clone.ID)
		assert.Equal(t, original.Name, clone.Name)
		assert.Equal(t, original.Type, clone.Type)
		assert.Equal(t, original.Connector, clone.Connector)
		assert.Equal(t, original.Description, clone.Description)

		if len(original.Tags) > 0 {
			originalTag := original.Tags[0]
			original.Tags[0] = "modified"
			assert.NotEqual(t, original.Tags[0], clone.Tags[0])
			original.Tags[0] = originalTag
		}

		if original.Options != nil {
			original.Options["test_key"] = "test_value"
			_, exists := clone.Options["test_key"]
			assert.False(t, exists, "Clone should not have modified key")
			delete(original.Options, "test_key")
		}

		if original.Dependencies != nil {
			original.Dependencies["test_dep"] = "^9.9.9"
			_, exists := clone.Dependencies["test_dep"]
			assert.False(t, exists, "Clone dependencies should not have modified key")
			delete(original.Dependencies, "test_dep")
		}
	})

	t.Run("CloneNil", func(t *testing.T) {
		var nilAssistant *assistant.Assistant
		assert.Nil(t, nilAssistant.Clone())
	})
}

// ---------------------------------------------------------------------------
// load_test.go.bak — TestUpdate
// ---------------------------------------------------------------------------

func TestUpdate(t *testing.T) {
	testprepare.PrepareSandbox(t)

	t.Run("UpdateBasicFields", func(t *testing.T) {
		ast, err := assistant.LoadPath("/assistants/tests/fullfields")
		require.NoError(t, err)

		updates := map[string]interface{}{
			"name":        "Updated Name",
			"description": "Updated description",
			"tags":        []string{"updated", "tags"},
		}

		err = ast.Update(updates)
		require.NoError(t, err)

		assert.Equal(t, "Updated Name", ast.Name)
		assert.Equal(t, "Updated description", ast.Description)
		assert.Equal(t, []string{"updated", "tags"}, ast.Tags)
	})

	t.Run("UpdateConnectorOptions", func(t *testing.T) {
		ast, err := assistant.LoadPath("/assistants/tests/fullfields")
		require.NoError(t, err)

		updates := map[string]interface{}{
			"connector_options": map[string]interface{}{
				"optional":   false,
				"connectors": []string{"new-connector"},
			},
		}

		err = ast.Update(updates)
		require.NoError(t, err)

		assert.NotNil(t, ast.ConnectorOptions)
		assert.NotNil(t, ast.ConnectorOptions.Optional)
		assert.False(t, *ast.ConnectorOptions.Optional)
		assert.Contains(t, ast.ConnectorOptions.Connectors, "new-connector")
	})

	t.Run("UpdatePromptPresets", func(t *testing.T) {
		ast, err := assistant.LoadPath("/assistants/tests/fullfields")
		require.NoError(t, err)

		updates := map[string]interface{}{
			"prompt_presets": map[string]interface{}{
				"custom": []map[string]interface{}{
					{"role": "system", "content": "Custom preset"},
				},
			},
		}

		err = ast.Update(updates)
		require.NoError(t, err)

		assert.NotNil(t, ast.PromptPresets)
		customPreset, exists := ast.PromptPresets["custom"]
		assert.True(t, exists)
		assert.Len(t, customPreset, 1)
	})

	t.Run("UpdateSource", func(t *testing.T) {
		ast, err := assistant.LoadPath("/assistants/tests/fullfields")
		require.NoError(t, err)

		updates := map[string]interface{}{
			"source": "function Create(ctx, messages) { return { messages: messages }; }",
		}

		err = ast.Update(updates)
		require.NoError(t, err)

		assert.Equal(t, "function Create(ctx, messages) { return { messages: messages }; }", ast.Source)
	})

	t.Run("UpdateNilAssistant", func(t *testing.T) {
		var nilAssistant *assistant.Assistant
		err := nilAssistant.Update(map[string]interface{}{"name": "test"})
		assert.Error(t, err)
	})
}

// ---------------------------------------------------------------------------
// load_test.go.bak — TestMap
// ---------------------------------------------------------------------------

func TestMap(t *testing.T) {
	testprepare.PrepareSandbox(t)

	ast, err := assistant.LoadPath("/assistants/tests/fullfields")
	require.NoError(t, err)

	m := ast.Map()
	require.NotNil(t, m)

	assert.Equal(t, ast.ID, m["assistant_id"])
	assert.Equal(t, ast.Name, m["name"])
	assert.Equal(t, ast.Type, m["type"])
	assert.Equal(t, ast.Connector, m["connector"])
	assert.Equal(t, ast.Description, m["description"])
	assert.Equal(t, ast.Path, m["path"])
	assert.Equal(t, ast.Tags, m["tags"])
	assert.Equal(t, ast.Options, m["options"])
	assert.Equal(t, ast.Prompts, m["prompts"])
	assert.Equal(t, ast.KB, m["kb"])
	assert.Equal(t, ast.MCP, m["mcp"])
	assert.Equal(t, ast.Workflow, m["workflow"])
	assert.Equal(t, ast.Placeholder, m["placeholder"])
	assert.Equal(t, ast.Locales, m["locales"])

	assert.Equal(t, ast.ConnectorOptions, m["connector_options"])
	assert.Equal(t, ast.PromptPresets, m["prompt_presets"])
	assert.Equal(t, ast.Source, m["source"])
	assert.Equal(t, ast.Dependencies, m["dependencies"])
}

// ---------------------------------------------------------------------------
// load_test.go.bak — TestValidate
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// load_test.go.bak — TestLoadSystemAgents
// ---------------------------------------------------------------------------

func TestLoadSystemAgents(t *testing.T) {
	testprepare.PrepareSandbox(t)

	assistant.ClearCache()
	assistant.SetCache(200)

	t.Run("LoadSystemAgents", func(t *testing.T) {
		err := assistant.LoadSystemAgents()
		require.NoError(t, err)

		keywordAst, keywordExists := assistant.GetCache().Get("__yao.keyword")
		require.True(t, keywordExists, "__yao.keyword should be loaded")
		assert.Equal(t, "__yao.keyword", keywordAst.ID)
		assert.Equal(t, "Keyword Extractor", keywordAst.Name)
		assert.True(t, keywordAst.Readonly)
		assert.True(t, keywordAst.BuiltIn)
		assert.Contains(t, keywordAst.Tags, "system")
		assert.NotNil(t, keywordAst.Prompts)
		assert.Greater(t, len(keywordAst.Prompts), 0)

		querydslAst, querydslExists := assistant.GetCache().Get("__yao.querydsl")
		require.True(t, querydslExists, "__yao.querydsl should be loaded")
		assert.Equal(t, "__yao.querydsl", querydslAst.ID)
		assert.Equal(t, "Query Builder", querydslAst.Name)
		assert.True(t, querydslAst.Readonly)
		assert.True(t, querydslAst.BuiltIn)
		assert.Contains(t, querydslAst.Tags, "system")
		assert.NotNil(t, querydslAst.Prompts)
		assert.Greater(t, len(querydslAst.Prompts), 0)

		titleAst, titleExists := assistant.GetCache().Get("__yao.title")
		require.True(t, titleExists, "__yao.title should be loaded")
		assert.Equal(t, "__yao.title", titleAst.ID)
		assert.Equal(t, "Title Generator", titleAst.Name)
		assert.True(t, titleAst.Readonly)
		assert.True(t, titleAst.BuiltIn)

		promptAst, promptExists := assistant.GetCache().Get("__yao.prompt")
		require.True(t, promptExists, "__yao.prompt should be loaded")
		assert.Equal(t, "__yao.prompt", promptAst.ID)
		assert.Equal(t, "Prompt Optimizer", promptAst.Name)
		assert.True(t, promptAst.Readonly)
		assert.True(t, promptAst.BuiltIn)

		needsearchAst, needsearchExists := assistant.GetCache().Get("__yao.needsearch")
		require.True(t, needsearchExists, "__yao.needsearch should be loaded")
		assert.Equal(t, "__yao.needsearch", needsearchAst.ID)
		assert.Equal(t, "Reference Checker", needsearchAst.Name)
		assert.True(t, needsearchAst.Readonly)
		assert.True(t, needsearchAst.BuiltIn)
	})

	t.Run("SystemAgentsSavedToStorage", func(t *testing.T) {
		require.NotNil(t, assistant.GetStore(), "storage should be initialized")

		builtIn := true
		tags := []string{"system"}
		res, err := assistant.GetStore().GetAssistants(store.AssistantFilter{
			BuiltIn: &builtIn,
			Tags:    tags,
			Select:  []string{"assistant_id", "name"},
		})
		require.NoError(t, err)
		require.Greater(t, len(res.Data), 0, "System agents should be in storage")

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
		assistant.GetCache().Clear()

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

// ---------------------------------------------------------------------------
// load_test.go.bak — TestLoadPathSandboxV2
// ---------------------------------------------------------------------------

func TestLoadPathSandboxV2(t *testing.T) {
	testprepare.PrepareSandbox(t)

	t.Run("OneshotCLI", func(t *testing.T) {
		ast, err := assistant.LoadPath("/assistants/tests/sandbox-v2/oneshot-cli")
		require.NoError(t, err)
		require.NotNil(t, ast)

		assert.Equal(t, "Sandbox V2 Oneshot CLI", ast.Name)
		assert.Contains(t, ast.Tags, "SandboxV2")

		require.NotNil(t, ast.SandboxV2, "SandboxV2 should be loaded")
		assert.Equal(t, "2.0", ast.SandboxV2.Version)
		assert.Equal(t, "yaoapp/tai-sandbox-claude:latest", ast.SandboxV2.Computer.Image)
		assert.Equal(t, "2GB", ast.SandboxV2.Computer.Memory)
		assert.Equal(t, float64(2), ast.SandboxV2.Computer.CPUs)
		assert.Equal(t, "/workspace", ast.SandboxV2.Computer.WorkDir)
		assert.Equal(t, "claude", ast.SandboxV2.Runner.Name)
		assert.Equal(t, "cli", ast.SandboxV2.Runner.Mode)
		assert.Equal(t, "oneshot", ast.SandboxV2.Lifecycle)

		assert.NotNil(t, ast.SandboxV2.Runner.Options)
		assert.Equal(t, float64(5), ast.SandboxV2.Runner.Options["max_turns"])

		assert.Nil(t, ast.Sandbox, "V1 Sandbox should be nil when V2 is present")
		assert.NotEmpty(t, ast.ConfigHash, "ConfigHash should be computed for V2 sandbox")
		assert.True(t, ast.HasSandboxV2())
	})

	t.Run("ConfigHashDeterministic", func(t *testing.T) {
		ast1, err := assistant.LoadPath("/assistants/tests/sandbox-v2/oneshot-cli")
		require.NoError(t, err)
		ast2, err := assistant.LoadPath("/assistants/tests/sandbox-v2/oneshot-cli")
		require.NoError(t, err)
		assert.Equal(t, ast1.ConfigHash, ast2.ConfigHash, "same config should produce same hash")
	})

}

// ---------------------------------------------------------------------------
// load_store_test.go.bak — helper
// ---------------------------------------------------------------------------

func newStoreTestContext(chatID, assistantID string) *context.Context {
	authorized := &types.AuthorizedInfo{
		Subject:   "test-user",
		ClientID:  "test-client-id",
		Scope:     "openid profile email",
		SessionID: "test-session-id",
		UserID:    "test-user-123",
		TeamID:    "test-team-456",
		TenantID:  "test-tenant-789",
	}

	ctx := context.New(stdContext.Background(), authorized, chatID)
	ctx.AssistantID = assistantID
	ctx.Locale = "en-us"
	ctx.Theme = "light"
	ctx.Client = context.Client{
		Type:      "web",
		UserAgent: "TestAgent/1.0",
		IP:        "127.0.0.1",
	}
	ctx.Referer = context.RefererAPI
	ctx.Accept = context.AcceptWebCUI
	ctx.Route = ""
	ctx.Metadata = make(map[string]interface{})
	return ctx
}

// ---------------------------------------------------------------------------
// load_store_test.go.bak — TestLoadStoreWithSource
// ---------------------------------------------------------------------------

func TestLoadStoreWithSource(t *testing.T) {
	testprepare.PrepareSandbox(t)

	assistantID := "test.store-with-source"
	now := time.Now().UnixNano()

	ast := &assistant.Assistant{
		AssistantModel: store.AssistantModel{
			ID:          assistantID,
			Name:        "Test Assistant With Source",
			Type:        "assistant",
			Connector:   "gpt-4o",
			Description: "Test assistant loaded from store with source code",
			Prompts: []store.Prompt{
				{Role: "system", Content: "You are a helpful assistant."},
			},
			Options: map[string]interface{}{
				"temperature": 0.7,
			},
			Tags: []string{"Test", "Source"},
			Source: `
// @ts-nocheck
function Create(ctx, messages) {
	return null;
}
`,
			CreatedAt: now,
			UpdatedAt: now,
		},
	}

	err := ast.Save()
	require.NoError(t, err)

	defer func() {
		storage := assistant.GetStorage()
		if storage != nil {
			storage.DeleteAssistant(assistantID)
		}
		assistant.GetCache().Clear()
	}()

	assistant.GetCache().Clear()

	loaded, err := assistant.Get(assistantID)
	require.NoError(t, err)
	require.NotNil(t, loaded)

	assert.Equal(t, assistantID, loaded.ID)
	assert.Equal(t, "Test Assistant With Source", loaded.Name)
	assert.Equal(t, "assistant", loaded.Type)
	assert.Equal(t, "Test assistant loaded from store with source code", loaded.Description)

	require.NotNil(t, loaded.Prompts)
	assert.Len(t, loaded.Prompts, 1)
	assert.Equal(t, "system", loaded.Prompts[0].Role)
	assert.Equal(t, "You are a helpful assistant.", loaded.Prompts[0].Content)

	assert.NotNil(t, loaded.Options)
	assert.Equal(t, 0.7, loaded.Options["temperature"])

	assert.NotNil(t, loaded.Tags)
	assert.Contains(t, loaded.Tags, "Test")
	assert.Contains(t, loaded.Tags, "Source")

	assert.NotNil(t, loaded.HookScript, "HookScript should be compiled from Source field")
	assert.NotEmpty(t, loaded.Source)
}

// ---------------------------------------------------------------------------
// load_store_test.go.bak — TestLoadStoreWithoutSource
// ---------------------------------------------------------------------------

func TestLoadStoreWithoutSource(t *testing.T) {
	testprepare.PrepareSandbox(t)

	assistantID := "test.store-without-source"
	now := time.Now().UnixNano()

	ast := &assistant.Assistant{
		AssistantModel: store.AssistantModel{
			ID:          assistantID,
			Name:        "Test Assistant Without Source",
			Type:        "assistant",
			Connector:   "gpt-4o",
			Description: "Test assistant loaded from store without source code",
			Prompts: []store.Prompt{
				{Role: "system", Content: "You are a helpful assistant without hooks."},
			},
			Options: map[string]interface{}{
				"temperature": 0.5,
				"max_tokens":  1000,
			},
			Tags:      []string{"Test", "NoSource"},
			CreatedAt: now,
			UpdatedAt: now,
		},
	}

	err := ast.Save()
	require.NoError(t, err)

	defer func() {
		storage := assistant.GetStorage()
		if storage != nil {
			storage.DeleteAssistant(assistantID)
		}
		assistant.GetCache().Clear()
	}()

	assistant.GetCache().Clear()

	loaded, err := assistant.Get(assistantID)
	require.NoError(t, err)
	require.NotNil(t, loaded)

	assert.Equal(t, assistantID, loaded.ID)
	assert.Equal(t, "Test Assistant Without Source", loaded.Name)
	assert.Equal(t, "assistant", loaded.Type)
	assert.Equal(t, "Test assistant loaded from store without source code", loaded.Description)

	require.NotNil(t, loaded.Prompts)
	assert.Len(t, loaded.Prompts, 1)
	assert.Equal(t, "system", loaded.Prompts[0].Role)

	assert.NotNil(t, loaded.Options)
	assert.Equal(t, 0.5, loaded.Options["temperature"])
	assert.Equal(t, float64(1000), loaded.Options["max_tokens"])

	assert.NotNil(t, loaded.Tags)
	assert.Contains(t, loaded.Tags, "Test")
	assert.Contains(t, loaded.Tags, "NoSource")

	assert.Nil(t, loaded.HookScript, "HookScript should be nil when no Source field")
	assert.Empty(t, loaded.Source)
}

// ---------------------------------------------------------------------------
// load_store_test.go.bak — TestLoadStoreWithSourceExecuteHook
// ---------------------------------------------------------------------------

func TestLoadStoreWithSourceExecuteHook(t *testing.T) {
	testprepare.PrepareSandbox(t)

	assistantID := "test.store-source-hook"
	now := time.Now().UnixNano()

	ast := &assistant.Assistant{
		AssistantModel: store.AssistantModel{
			ID:        assistantID,
			Name:      "Test Source Hook",
			Type:      "assistant",
			Connector: "gpt-4o",
			Prompts: []store.Prompt{
				{Role: "system", Content: "Default prompt"},
			},
			Source: `
// @ts-nocheck
function Create(ctx: any, messages: any[]): any {
	return {
		temperature: 0.9,
		metadata: {
			hook_executed: true,
			chat_id: ctx.chat_id
		}
	};
}
`,
			CreatedAt: now,
			UpdatedAt: now,
		},
	}

	err := ast.Save()
	require.NoError(t, err)

	defer func() {
		storage := assistant.GetStorage()
		if storage != nil {
			storage.DeleteAssistant(assistantID)
		}
		assistant.GetCache().Clear()
	}()

	assistant.GetCache().Clear()

	loaded, err := assistant.Get(assistantID)
	require.NoError(t, err)
	require.NotNil(t, loaded)
	require.NotNil(t, loaded.HookScript, "HookScript should be compiled from Source")

	assert.NotNil(t, loaded.HookScript.Script)

	ctx := newStoreTestContext("test-chat-id", assistantID)
	messages := []context.Message{{Role: "user", Content: "Hello"}}

	res, _, err := loaded.HookScript.Create(ctx, messages, &context.Options{})
	require.NoError(t, err, "Create hook should execute without error")
	require.NotNil(t, res, "Create hook should return a response")

	require.NotNil(t, res.Temperature, "Temperature should be set")
	assert.Equal(t, 0.9, *res.Temperature, "Temperature should be 0.9")

	require.NotNil(t, res.Metadata, "Metadata should be set")
	assert.Equal(t, true, res.Metadata["hook_executed"], "hook_executed should be true")
	assert.Equal(t, "test-chat-id", res.Metadata["chat_id"], "chat_id should match context")
}

// ---------------------------------------------------------------------------
// load_store_test.go.bak — TestLoadStoreWithPromptPresets
// ---------------------------------------------------------------------------

func TestLoadStoreWithPromptPresets(t *testing.T) {
	testprepare.PrepareSandbox(t)

	assistantID := "test.store-with-presets"
	now := time.Now().UnixNano()

	ast := &assistant.Assistant{
		AssistantModel: store.AssistantModel{
			ID:        assistantID,
			Name:      "Test With Presets",
			Type:      "assistant",
			Connector: "gpt-4o",
			Prompts: []store.Prompt{
				{Role: "system", Content: "Default prompt"},
			},
			PromptPresets: map[string][]store.Prompt{
				"friendly": {
					{Role: "system", Content: "You are a friendly assistant."},
				},
				"professional": {
					{Role: "system", Content: "You are a professional assistant."},
				},
				"mode.casual": {
					{Role: "system", Content: "You are a casual assistant."},
				},
			},
			CreatedAt: now,
			UpdatedAt: now,
		},
	}

	err := ast.Save()
	require.NoError(t, err)

	defer func() {
		storage := assistant.GetStorage()
		if storage != nil {
			storage.DeleteAssistant(assistantID)
		}
		assistant.GetCache().Clear()
	}()

	assistant.GetCache().Clear()

	loaded, err := assistant.Get(assistantID)
	require.NoError(t, err)
	require.NotNil(t, loaded)

	require.NotNil(t, loaded.PromptPresets)
	assert.Len(t, loaded.PromptPresets, 3)

	friendlyPreset, ok := loaded.PromptPresets["friendly"]
	assert.True(t, ok)
	assert.Len(t, friendlyPreset, 1)
	assert.Equal(t, "You are a friendly assistant.", friendlyPreset[0].Content)

	professionalPreset, ok := loaded.PromptPresets["professional"]
	assert.True(t, ok)
	assert.Len(t, professionalPreset, 1)
	assert.Equal(t, "You are a professional assistant.", professionalPreset[0].Content)

	casualPreset, ok := loaded.PromptPresets["mode.casual"]
	assert.True(t, ok)
	assert.Len(t, casualPreset, 1)
	assert.Equal(t, "You are a casual assistant.", casualPreset[0].Content)
}

// ---------------------------------------------------------------------------
// load_store_test.go.bak — TestLoadStoreWithDisableGlobalPrompts
// ---------------------------------------------------------------------------

func TestLoadStoreWithDisableGlobalPrompts(t *testing.T) {
	testprepare.PrepareSandbox(t)

	assistantID := "test.store-disable-global"
	now := time.Now().UnixNano()

	ast := &assistant.Assistant{
		AssistantModel: store.AssistantModel{
			ID:                   assistantID,
			Name:                 "Test Disable Global Prompts",
			Type:                 "assistant",
			Connector:            "gpt-4o",
			DisableGlobalPrompts: true,
			Prompts: []store.Prompt{
				{Role: "system", Content: "Only this prompt should be used."},
			},
			CreatedAt: now,
			UpdatedAt: now,
		},
	}

	err := ast.Save()
	require.NoError(t, err)

	defer func() {
		storage := assistant.GetStorage()
		if storage != nil {
			storage.DeleteAssistant(assistantID)
		}
		assistant.GetCache().Clear()
	}()

	assistant.GetCache().Clear()

	loaded, err := assistant.Get(assistantID)
	require.NoError(t, err)
	require.NotNil(t, loaded)

	assert.True(t, loaded.DisableGlobalPrompts)
}

// ---------------------------------------------------------------------------
// load_store_test.go.bak — TestLoadStoreCaching
// ---------------------------------------------------------------------------

func TestLoadStoreCaching(t *testing.T) {
	testprepare.PrepareSandbox(t)

	assistantID := "test.store-caching"
	now := time.Now().UnixNano()

	ast := &assistant.Assistant{
		AssistantModel: store.AssistantModel{
			ID:        assistantID,
			Name:      "Test Caching",
			Type:      "assistant",
			Connector: "gpt-4o",
			CreatedAt: now,
			UpdatedAt: now,
		},
	}

	err := ast.Save()
	require.NoError(t, err)

	defer func() {
		storage := assistant.GetStorage()
		if storage != nil {
			storage.DeleteAssistant(assistantID)
		}
		assistant.GetCache().Clear()
	}()

	assistant.GetCache().Clear()

	ast1, err := assistant.Get(assistantID)
	require.NoError(t, err)
	require.NotNil(t, ast1)

	ast2, err := assistant.Get(assistantID)
	require.NoError(t, err)
	require.NotNil(t, ast2)

	assert.Same(t, ast1, ast2)
}

// ---------------------------------------------------------------------------
// load_store_test.go.bak — TestLoadStoreWithAllFields
// ---------------------------------------------------------------------------

func TestLoadStoreWithAllFields(t *testing.T) {
	testprepare.PrepareSandbox(t)

	assistantID := "test.store-all-fields"
	now := time.Now().UnixNano()

	ast := &assistant.Assistant{
		AssistantModel: store.AssistantModel{
			ID:          assistantID,
			Name:        "Test All Fields",
			Type:        "assistant",
			Avatar:      "/api/icons/test.png",
			Connector:   "gpt-4o",
			Description: "Test assistant with all fields",
			Tags:        []string{"Test", "AllFields", "Complete"},
			Readonly:    true,
			Public:      true,
			Share:       "team",
			Mentionable: true,
			Automated:   false,
			Sort:        100,
			Options: map[string]interface{}{
				"temperature": 0.8,
				"max_tokens":  2000,
			},
			Prompts: []store.Prompt{
				{Role: "system", Content: "You are a test assistant."},
				{Role: "system", Content: "Follow all instructions carefully."},
			},
			PromptPresets: map[string][]store.Prompt{
				"default": {
					{Role: "system", Content: "Default mode prompt."},
				},
			},
			DisableGlobalPrompts: true,
			Placeholder: &store.Placeholder{
				Title:       "Test Placeholder",
				Description: "This is a test placeholder",
				Prompts:     []string{"Test prompt 1", "Test prompt 2"},
			},
			Dependencies: map[string]string{
				"echo":     "^1.0.0",
				"customer": ">=2.0.0",
			},
			Source: `
// @ts-nocheck
function Create(ctx: any, messages: any[]): any {
	return { 
		temperature: 0.5,
		metadata: {
			assistant_name: "Test All Fields",
			executed: true
		}
	};
}
`,
			CreatedAt: now,
			UpdatedAt: now,
		},
	}

	err := ast.Save()
	require.NoError(t, err)

	defer func() {
		storage := assistant.GetStorage()
		if storage != nil {
			storage.DeleteAssistant(assistantID)
		}
		assistant.GetCache().Clear()
	}()

	assistant.GetCache().Clear()

	loaded, err := assistant.Get(assistantID)
	require.NoError(t, err)
	require.NotNil(t, loaded)

	assert.Equal(t, assistantID, loaded.ID)
	assert.Equal(t, "Test All Fields", loaded.Name)
	assert.Equal(t, "assistant", loaded.Type)
	assert.Equal(t, "/api/icons/test.png", loaded.Avatar)
	assert.Equal(t, "Test assistant with all fields", loaded.Description)

	assert.True(t, loaded.Readonly)
	assert.True(t, loaded.Public)
	assert.Equal(t, "team", loaded.Share)
	assert.True(t, loaded.Mentionable)
	assert.False(t, loaded.Automated)
	assert.True(t, loaded.DisableGlobalPrompts)
	assert.Equal(t, 100, loaded.Sort)

	assert.Len(t, loaded.Tags, 3)
	assert.Contains(t, loaded.Tags, "Test")
	assert.Contains(t, loaded.Tags, "AllFields")
	assert.Contains(t, loaded.Tags, "Complete")

	assert.Equal(t, 0.8, loaded.Options["temperature"])
	assert.Equal(t, float64(2000), loaded.Options["max_tokens"])

	assert.Len(t, loaded.Prompts, 2)

	assert.NotNil(t, loaded.PromptPresets)
	assert.Contains(t, loaded.PromptPresets, "default")

	assert.NotNil(t, loaded.Placeholder)
	assert.Equal(t, "Test Placeholder", loaded.Placeholder.Title)
	assert.Equal(t, "This is a test placeholder", loaded.Placeholder.Description)
	assert.Len(t, loaded.Placeholder.Prompts, 2)

	assert.NotNil(t, loaded.HookScript)
	assert.NotEmpty(t, loaded.Source)

	require.NotNil(t, loaded.Dependencies)
	assert.Len(t, loaded.Dependencies, 2)
	assert.Equal(t, "^1.0.0", loaded.Dependencies["echo"])
	assert.Equal(t, ">=2.0.0", loaded.Dependencies["customer"])

	ctx := newStoreTestContext("test-chat-all-fields", assistantID)
	messages := []context.Message{{Role: "user", Content: "Test message"}}

	res, _, err := loaded.HookScript.Create(ctx, messages, &context.Options{})
	require.NoError(t, err, "Create hook should execute without error")
	require.NotNil(t, res, "Create hook should return a response")

	require.NotNil(t, res.Temperature, "Temperature should be set")
	assert.Equal(t, 0.5, *res.Temperature, "Temperature should be 0.5")

	require.NotNil(t, res.Metadata, "Metadata should be set")
	assert.Equal(t, "Test All Fields", res.Metadata["assistant_name"], "assistant_name should match")
	assert.Equal(t, true, res.Metadata["executed"], "executed should be true")
}

// ---------------------------------------------------------------------------
// load_store_test.go.bak — TestLoadStoreHookWithTypeScript
// ---------------------------------------------------------------------------

func TestLoadStoreHookWithTypeScript(t *testing.T) {
	testprepare.PrepareSandbox(t)

	assistantID := "test.store-typescript-hook"
	now := time.Now().UnixNano()

	ast := &assistant.Assistant{
		AssistantModel: store.AssistantModel{
			ID:        assistantID,
			Name:      "Test TypeScript Hook",
			Type:      "assistant",
			Connector: "gpt-4o",
			Prompts: []store.Prompt{
				{Role: "system", Content: "Default prompt"},
			},
			Source: `
interface CreateContext {
	chat_id: string;
	assistant_id: string;
	locale: string;
	authorized?: {
		user_id: string;
		team_id: string;
	};
}

interface Message {
	role: string;
	content: string | object;
}

interface CreateResponse {
	temperature?: number;
	messages?: Message[];
	metadata?: Record<string, any>;
}

function Create(ctx: CreateContext, messages: Message[]): CreateResponse | null {
	const chatId: string = ctx.chat_id || "unknown";
	const locale: string = ctx.locale || "en-us";
	const userId: string = ctx.authorized?.user_id || "anonymous";
	
	const userMessages: Message[] = messages.filter((m: Message) => m.role === "user");
	const messageCount: number = userMessages.length;
	
	return {
		temperature: 0.7,
		messages: [
			{
				role: "system",
				content: "TypeScript hook executed successfully"
			}
		],
		metadata: {
			chat_id: chatId,
			locale: locale,
			user_id: userId,
			message_count: messageCount,
			typescript_features: true
		}
	};
}
`,
			CreatedAt: now,
			UpdatedAt: now,
		},
	}

	err := ast.Save()
	require.NoError(t, err)

	defer func() {
		storage := assistant.GetStorage()
		if storage != nil {
			storage.DeleteAssistant(assistantID)
		}
		assistant.GetCache().Clear()
	}()

	assistant.GetCache().Clear()

	loaded, err := assistant.Get(assistantID)
	require.NoError(t, err)
	require.NotNil(t, loaded)
	require.NotNil(t, loaded.HookScript, "HookScript should be compiled from TypeScript Source")

	ctx := newStoreTestContext("ts-test-chat", assistantID)
	messages := []context.Message{
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Hi there"},
		{Role: "user", Content: "How are you?"},
	}

	res, _, err := loaded.HookScript.Create(ctx, messages, &context.Options{})
	require.NoError(t, err, "TypeScript Create hook should execute without error")
	require.NotNil(t, res, "Create hook should return a response")

	require.NotNil(t, res.Temperature)
	assert.Equal(t, 0.7, *res.Temperature)

	require.Len(t, res.Messages, 1)
	assert.Equal(t, context.RoleSystem, res.Messages[0].Role)
	assert.Equal(t, "TypeScript hook executed successfully", res.Messages[0].Content)

	require.NotNil(t, res.Metadata)
	assert.Equal(t, "ts-test-chat", res.Metadata["chat_id"])
	assert.Equal(t, "en-us", res.Metadata["locale"])
	assert.Equal(t, "test-user-123", res.Metadata["user_id"])
	assert.Equal(t, float64(2), res.Metadata["message_count"])
	assert.Equal(t, true, res.Metadata["typescript_features"])
}

// ---------------------------------------------------------------------------
// load_store_test.go.bak — TestLoadStoreHookReturnNull
// ---------------------------------------------------------------------------

func TestLoadStoreHookReturnNull(t *testing.T) {
	testprepare.PrepareSandbox(t)

	assistantID := "test.store-hook-null"
	now := time.Now().UnixNano()

	ast := &assistant.Assistant{
		AssistantModel: store.AssistantModel{
			ID:        assistantID,
			Name:      "Test Hook Return Null",
			Type:      "assistant",
			Connector: "gpt-4o",
			Source: `
function Create(ctx: any, messages: any[]): any {
	return null;
}
`,
			CreatedAt: now,
			UpdatedAt: now,
		},
	}

	err := ast.Save()
	require.NoError(t, err)

	defer func() {
		storage := assistant.GetStorage()
		if storage != nil {
			storage.DeleteAssistant(assistantID)
		}
		assistant.GetCache().Clear()
	}()

	assistant.GetCache().Clear()

	loaded, err := assistant.Get(assistantID)
	require.NoError(t, err)
	require.NotNil(t, loaded)
	require.NotNil(t, loaded.HookScript)

	ctx := newStoreTestContext("null-test-chat", assistantID)
	messages := []context.Message{{Role: "user", Content: "Hello"}}

	res, _, err := loaded.HookScript.Create(ctx, messages, &context.Options{})
	require.NoError(t, err, "Hook returning null should not error")
	assert.Nil(t, res, "Hook returning null should return nil response")
}

// ---------------------------------------------------------------------------
// load_store_test.go.bak — TestLoadStoreHookWithPromptPreset
// ---------------------------------------------------------------------------

func TestLoadStoreHookWithPromptPreset(t *testing.T) {
	testprepare.PrepareSandbox(t)

	assistantID := "test.store-hook-preset"
	now := time.Now().UnixNano()

	ast := &assistant.Assistant{
		AssistantModel: store.AssistantModel{
			ID:        assistantID,
			Name:      "Test Hook Prompt Preset",
			Type:      "assistant",
			Connector: "gpt-4o",
			Prompts: []store.Prompt{
				{Role: "system", Content: "Default prompt"},
			},
			PromptPresets: map[string][]store.Prompt{
				"friendly": {
					{Role: "system", Content: "You are a friendly assistant."},
				},
				"professional": {
					{Role: "system", Content: "You are a professional assistant."},
				},
			},
			Source: `
function Create(ctx: any, messages: any[]): any {
	const firstMsg = messages[0];
	if (firstMsg && typeof firstMsg.content === "string") {
		if (firstMsg.content.includes("friendly")) {
			return { prompt_preset: "friendly" };
		}
		if (firstMsg.content.includes("professional")) {
			return { prompt_preset: "professional" };
		}
	}
	return null;
}
`,
			CreatedAt: now,
			UpdatedAt: now,
		},
	}

	err := ast.Save()
	require.NoError(t, err)

	defer func() {
		storage := assistant.GetStorage()
		if storage != nil {
			storage.DeleteAssistant(assistantID)
		}
		assistant.GetCache().Clear()
	}()

	assistant.GetCache().Clear()

	loaded, err := assistant.Get(assistantID)
	require.NoError(t, err)
	require.NotNil(t, loaded)
	require.NotNil(t, loaded.HookScript)

	t.Run("SelectFriendlyPreset", func(t *testing.T) {
		ctx := newStoreTestContext("preset-test-1", assistantID)
		messages := []context.Message{{Role: "user", Content: "Be friendly please"}}

		res, _, err := loaded.HookScript.Create(ctx, messages, &context.Options{})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Equal(t, "friendly", res.PromptPreset)
	})

	t.Run("SelectProfessionalPreset", func(t *testing.T) {
		ctx := newStoreTestContext("preset-test-2", assistantID)
		messages := []context.Message{{Role: "user", Content: "Be professional"}}

		res, _, err := loaded.HookScript.Create(ctx, messages, &context.Options{})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Equal(t, "professional", res.PromptPreset)
	})

	t.Run("NoPreset", func(t *testing.T) {
		ctx := newStoreTestContext("preset-test-3", assistantID)
		messages := []context.Message{{Role: "user", Content: "Hello"}}

		res, _, err := loaded.HookScript.Create(ctx, messages, &context.Options{})
		require.NoError(t, err)
		assert.Nil(t, res)
	})
}

// ---------------------------------------------------------------------------
// load_store_test.go.bak — TestLoadStoreHookDisableGlobalPrompts
// ---------------------------------------------------------------------------

func TestLoadStoreHookDisableGlobalPrompts(t *testing.T) {
	testprepare.PrepareSandbox(t)

	assistantID := "test.store-hook-disable-global"
	now := time.Now().UnixNano()

	ast := &assistant.Assistant{
		AssistantModel: store.AssistantModel{
			ID:        assistantID,
			Name:      "Test Hook Disable Global",
			Type:      "assistant",
			Connector: "gpt-4o",
			Source: `
function Create(ctx: any, messages: any[]): any {
	const firstMsg = messages[0];
	if (firstMsg && typeof firstMsg.content === "string") {
		if (firstMsg.content.includes("disable_global")) {
			return { disable_global_prompts: true };
		}
		if (firstMsg.content.includes("enable_global")) {
			return { disable_global_prompts: false };
		}
	}
	return null;
}
`,
			CreatedAt: now,
			UpdatedAt: now,
		},
	}

	err := ast.Save()
	require.NoError(t, err)

	defer func() {
		storage := assistant.GetStorage()
		if storage != nil {
			storage.DeleteAssistant(assistantID)
		}
		assistant.GetCache().Clear()
	}()

	assistant.GetCache().Clear()

	loaded, err := assistant.Get(assistantID)
	require.NoError(t, err)
	require.NotNil(t, loaded)
	require.NotNil(t, loaded.HookScript)

	t.Run("DisableGlobalPrompts", func(t *testing.T) {
		ctx := newStoreTestContext("disable-test-1", assistantID)
		messages := []context.Message{{Role: "user", Content: "disable_global prompts"}}

		res, _, err := loaded.HookScript.Create(ctx, messages, &context.Options{})
		require.NoError(t, err)
		require.NotNil(t, res)
		require.NotNil(t, res.DisableGlobalPrompts)
		assert.True(t, *res.DisableGlobalPrompts)
	})

	t.Run("EnableGlobalPrompts", func(t *testing.T) {
		ctx := newStoreTestContext("disable-test-2", assistantID)
		messages := []context.Message{{Role: "user", Content: "enable_global prompts"}}

		res, _, err := loaded.HookScript.Create(ctx, messages, &context.Options{})
		require.NoError(t, err)
		require.NotNil(t, res)
		require.NotNil(t, res.DisableGlobalPrompts)
		assert.False(t, *res.DisableGlobalPrompts)
	})
}

// ---------------------------------------------------------------------------
// load_store_test.go.bak — TestLoadStoreWithSearchConfig
// ---------------------------------------------------------------------------

func TestLoadStoreWithSearchConfig(t *testing.T) {
	testprepare.PrepareSandbox(t)

	assistantID := "test.store-with-search"
	now := time.Now().UnixNano()

	ast := &assistant.Assistant{
		AssistantModel: store.AssistantModel{
			ID:        assistantID,
			Name:      "Test With Search Config",
			Type:      "assistant",
			Connector: "gpt-4o",
			Uses: &context.Uses{
				Vision:   "agent",
				Audio:    "mcp:audio-server",
				Fetch:    "agent",
				Web:      "builtin",
				Keyword:  "builtin",
				QueryDSL: "builtin",
				Rerank:   "builtin",
			},
			Search: &searchTypes.Config{
				Web: &searchTypes.WebConfig{
					Provider:   "tavily",
					MaxResults: 15,
				},
				KB: &searchTypes.KBConfig{
					Collections: []string{"docs", "faq"},
					Threshold:   0.8,
					Graph:       true,
				},
				DB: &searchTypes.DBConfig{
					Models:     []string{"user", "product"},
					MaxResults: 50,
				},
				Keyword: &searchTypes.KeywordConfig{
					MaxKeywords: 8,
					Language:    "auto",
				},
				QueryDSL: &searchTypes.QueryDSLConfig{
					Strict: true,
				},
				Rerank: &searchTypes.RerankConfig{
					TopN: 5,
				},
				Citation: &searchTypes.CitationConfig{
					Format:           "#cite:{id}",
					AutoInjectPrompt: false,
					CustomPrompt:     "Please cite sources.",
				},
				Weights: &searchTypes.WeightsConfig{
					User: 1.0,
					Hook: 0.85,
					Auto: 0.65,
				},
				Options: &searchTypes.OptionsConfig{
					SkipThreshold: 3,
				},
			},
			CreatedAt: now,
			UpdatedAt: now,
		},
	}

	err := ast.Save()
	require.NoError(t, err)

	defer func() {
		storage := assistant.GetStorage()
		if storage != nil {
			storage.DeleteAssistant(assistantID)
		}
		assistant.GetCache().Clear()
	}()

	assistant.GetCache().Clear()

	loaded, err := assistant.Get(assistantID)
	require.NoError(t, err)
	require.NotNil(t, loaded)

	require.NotNil(t, loaded.Uses)
	assert.Equal(t, "agent", loaded.Uses.Vision)
	assert.Equal(t, "mcp:audio-server", loaded.Uses.Audio)
	assert.Equal(t, "agent", loaded.Uses.Fetch)
	assert.Equal(t, "builtin", loaded.Uses.Web)
	assert.Equal(t, "builtin", loaded.Uses.Keyword)
	assert.Equal(t, "builtin", loaded.Uses.QueryDSL)
	assert.Equal(t, "builtin", loaded.Uses.Rerank)

	require.NotNil(t, loaded.Search)

	require.NotNil(t, loaded.Search.Web)
	assert.Equal(t, "tavily", loaded.Search.Web.Provider)
	assert.Equal(t, 15, loaded.Search.Web.MaxResults)

	require.NotNil(t, loaded.Search.KB)
	assert.Equal(t, []string{"docs", "faq"}, loaded.Search.KB.Collections)
	assert.Equal(t, 0.8, loaded.Search.KB.Threshold)
	assert.True(t, loaded.Search.KB.Graph)

	require.NotNil(t, loaded.Search.DB)
	assert.Equal(t, []string{"user", "product"}, loaded.Search.DB.Models)
	assert.Equal(t, 50, loaded.Search.DB.MaxResults)

	require.NotNil(t, loaded.Search.Keyword)
	assert.Equal(t, 8, loaded.Search.Keyword.MaxKeywords)
	assert.Equal(t, "auto", loaded.Search.Keyword.Language)

	require.NotNil(t, loaded.Search.QueryDSL)
	assert.True(t, loaded.Search.QueryDSL.Strict)

	require.NotNil(t, loaded.Search.Rerank)
	assert.Equal(t, 5, loaded.Search.Rerank.TopN)

	require.NotNil(t, loaded.Search.Citation)
	assert.Equal(t, "#cite:{id}", loaded.Search.Citation.Format)
	assert.False(t, loaded.Search.Citation.AutoInjectPrompt)
	assert.Equal(t, "Please cite sources.", loaded.Search.Citation.CustomPrompt)

	require.NotNil(t, loaded.Search.Weights)
	assert.Equal(t, 1.0, loaded.Search.Weights.User)
	assert.Equal(t, 0.85, loaded.Search.Weights.Hook)
	assert.Equal(t, 0.65, loaded.Search.Weights.Auto)

	require.NotNil(t, loaded.Search.Options)
	assert.Equal(t, 3, loaded.Search.Options.SkipThreshold)
}

// ---------------------------------------------------------------------------
// load_store_test.go.bak — TestLoadStoreWithPartialSearchConfig
// ---------------------------------------------------------------------------

func TestLoadStoreWithPartialSearchConfig(t *testing.T) {
	testprepare.PrepareSandbox(t)

	assistantID := "test.store-partial-search"
	now := time.Now().UnixNano()

	ast := &assistant.Assistant{
		AssistantModel: store.AssistantModel{
			ID:        assistantID,
			Name:      "Test Partial Search Config",
			Type:      "assistant",
			Connector: "gpt-4o",
			Search: &searchTypes.Config{
				Web: &searchTypes.WebConfig{
					Provider: "serper",
				},
			},
			CreatedAt: now,
			UpdatedAt: now,
		},
	}

	err := ast.Save()
	require.NoError(t, err)

	defer func() {
		storage := assistant.GetStorage()
		if storage != nil {
			storage.DeleteAssistant(assistantID)
		}
		assistant.GetCache().Clear()
	}()

	assistant.GetCache().Clear()

	loaded, err := assistant.Get(assistantID)
	require.NoError(t, err)
	require.NotNil(t, loaded)

	require.NotNil(t, loaded.Search)

	require.NotNil(t, loaded.Search.Web)
	assert.Equal(t, "serper", loaded.Search.Web.Provider)

	assert.Nil(t, loaded.Search.KB)
	assert.Nil(t, loaded.Search.DB)
	assert.Nil(t, loaded.Search.Keyword)
	assert.Nil(t, loaded.Search.QueryDSL)
	assert.Nil(t, loaded.Search.Rerank)
	assert.Nil(t, loaded.Search.Citation)
	assert.Nil(t, loaded.Search.Weights)
	assert.Nil(t, loaded.Search.Options)
}

// ---------------------------------------------------------------------------
// load_store_test.go.bak — TestLoadStoreWithoutSearchConfig
// ---------------------------------------------------------------------------

func TestLoadStoreWithoutSearchConfig(t *testing.T) {
	testprepare.PrepareSandbox(t)

	assistantID := "test.store-no-search"
	now := time.Now().UnixNano()

	ast := &assistant.Assistant{
		AssistantModel: store.AssistantModel{
			ID:        assistantID,
			Name:      "Test No Search Config",
			Type:      "assistant",
			Connector: "gpt-4o",
			CreatedAt: now,
			UpdatedAt: now,
		},
	}

	err := ast.Save()
	require.NoError(t, err)

	defer func() {
		storage := assistant.GetStorage()
		if storage != nil {
			storage.DeleteAssistant(assistantID)
		}
		assistant.GetCache().Clear()
	}()

	assistant.GetCache().Clear()

	loaded, err := assistant.Get(assistantID)
	require.NoError(t, err)
	require.NotNil(t, loaded)

	assert.Nil(t, loaded.Search)
}
