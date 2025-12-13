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
	"github.com/yaoapp/yao/agent/testutils"
	"github.com/yaoapp/yao/openapi/oauth/types"
)

// TestLoadStoreWithSource tests loading assistant from database with Source field
func TestLoadStoreWithSource(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Create assistant with Source
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
			// Simple Create hook that returns null
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

	// Save to database
	err := ast.Save()
	require.NoError(t, err)

	// Cleanup after test
	defer func() {
		storage := assistant.GetStorage()
		if storage != nil {
			storage.DeleteAssistant(assistantID)
		}
		assistant.GetCache().Clear()
	}()

	// Clear cache to ensure fresh load from database
	assistant.GetCache().Clear()

	// Load from store
	loaded, err := assistant.Get(assistantID)
	require.NoError(t, err)
	require.NotNil(t, loaded)

	// Verify basic fields
	assert.Equal(t, assistantID, loaded.ID)
	assert.Equal(t, "Test Assistant With Source", loaded.Name)
	assert.Equal(t, "assistant", loaded.Type)
	assert.Equal(t, "Test assistant loaded from store with source code", loaded.Description)

	// Verify prompts
	require.NotNil(t, loaded.Prompts)
	assert.Len(t, loaded.Prompts, 1)
	assert.Equal(t, "system", loaded.Prompts[0].Role)
	assert.Equal(t, "You are a helpful assistant.", loaded.Prompts[0].Content)

	// Verify options
	assert.NotNil(t, loaded.Options)
	assert.Equal(t, 0.7, loaded.Options["temperature"])

	// Verify tags
	assert.NotNil(t, loaded.Tags)
	assert.Contains(t, loaded.Tags, "Test")
	assert.Contains(t, loaded.Tags, "Source")

	// Verify script was compiled from source
	assert.NotNil(t, loaded.HookScript, "HookScript should be compiled from Source field")

	// Verify source is stored
	assert.NotEmpty(t, loaded.Source)
}

// TestLoadStoreWithoutSource tests loading assistant from database without Source field
func TestLoadStoreWithoutSource(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Create assistant without Source
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

	// Save to database
	err := ast.Save()
	require.NoError(t, err)

	// Cleanup after test
	defer func() {
		storage := assistant.GetStorage()
		if storage != nil {
			storage.DeleteAssistant(assistantID)
		}
		assistant.GetCache().Clear()
	}()

	// Clear cache to ensure fresh load from database
	assistant.GetCache().Clear()

	// Load from store
	loaded, err := assistant.Get(assistantID)
	require.NoError(t, err)
	require.NotNil(t, loaded)

	// Verify basic fields
	assert.Equal(t, assistantID, loaded.ID)
	assert.Equal(t, "Test Assistant Without Source", loaded.Name)
	assert.Equal(t, "assistant", loaded.Type)
	assert.Equal(t, "Test assistant loaded from store without source code", loaded.Description)

	// Verify prompts
	require.NotNil(t, loaded.Prompts)
	assert.Len(t, loaded.Prompts, 1)
	assert.Equal(t, "system", loaded.Prompts[0].Role)

	// Verify options
	assert.NotNil(t, loaded.Options)
	assert.Equal(t, 0.5, loaded.Options["temperature"])
	assert.Equal(t, float64(1000), loaded.Options["max_tokens"])

	// Verify tags
	assert.NotNil(t, loaded.Tags)
	assert.Contains(t, loaded.Tags, "Test")
	assert.Contains(t, loaded.Tags, "NoSource")

	// Verify script is nil (no source)
	assert.Nil(t, loaded.HookScript, "HookScript should be nil when no Source field")
	assert.Empty(t, loaded.Source)
}

// newStoreTestContext creates a Context for testing with commonly used fields pre-populated.
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

// TestLoadStoreWithSourceExecuteHook tests that Source-based script is properly compiled and can execute
func TestLoadStoreWithSourceExecuteHook(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Create assistant with a working Create hook
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
			// Create hook that modifies temperature and adds metadata
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

	// Save to database
	err := ast.Save()
	require.NoError(t, err)

	// Cleanup after test
	defer func() {
		storage := assistant.GetStorage()
		if storage != nil {
			storage.DeleteAssistant(assistantID)
		}
		assistant.GetCache().Clear()
	}()

	// Clear cache
	assistant.GetCache().Clear()

	// Load from store
	loaded, err := assistant.Get(assistantID)
	require.NoError(t, err)
	require.NotNil(t, loaded)
	require.NotNil(t, loaded.HookScript, "HookScript should be compiled from Source")

	// Verify the script object exists and is usable
	assert.NotNil(t, loaded.HookScript.Script)

	// Execute the Create hook
	ctx := newStoreTestContext("test-chat-id", assistantID)
	messages := []context.Message{{Role: "user", Content: "Hello"}}

	res, _, err := loaded.HookScript.Create(ctx, messages, &context.Options{})
	require.NoError(t, err, "Create hook should execute without error")
	require.NotNil(t, res, "Create hook should return a response")

	// Verify temperature was set
	require.NotNil(t, res.Temperature, "Temperature should be set")
	assert.Equal(t, 0.9, *res.Temperature, "Temperature should be 0.9")

	// Verify metadata was set
	require.NotNil(t, res.Metadata, "Metadata should be set")
	assert.Equal(t, true, res.Metadata["hook_executed"], "hook_executed should be true")
	assert.Equal(t, "test-chat-id", res.Metadata["chat_id"], "chat_id should match context")
}

// TestLoadStoreWithPromptPresets tests loading assistant with prompt presets from database
func TestLoadStoreWithPromptPresets(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

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

	// Verify prompt presets
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

// TestLoadStoreWithDisableGlobalPrompts tests loading assistant with disable_global_prompts flag
func TestLoadStoreWithDisableGlobalPrompts(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

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

// TestLoadStoreCaching tests that loaded assistants are cached
func TestLoadStoreCaching(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

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

	// First load
	ast1, err := assistant.Get(assistantID)
	require.NoError(t, err)
	require.NotNil(t, ast1)

	// Second load - should be from cache
	ast2, err := assistant.Get(assistantID)
	require.NoError(t, err)
	require.NotNil(t, ast2)

	// Should be the same instance (from cache)
	assert.Same(t, ast1, ast2)
}

// TestLoadStoreNotFound tests loading non-existent assistant
func TestLoadStoreNotFound(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	assistant.GetCache().Clear()

	_, err := assistant.Get("non-existent-assistant-id-12345")
	assert.Error(t, err)
}

// TestLoadStoreWithAllFields tests loading assistant with comprehensive fields
func TestLoadStoreWithAllFields(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

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

	// Verify all fields
	assert.Equal(t, assistantID, loaded.ID)
	assert.Equal(t, "Test All Fields", loaded.Name)
	assert.Equal(t, "assistant", loaded.Type)
	assert.Equal(t, "/api/icons/test.png", loaded.Avatar)
	assert.Equal(t, "Test assistant with all fields", loaded.Description)

	// Boolean fields
	assert.True(t, loaded.Readonly)
	assert.True(t, loaded.Public)
	assert.Equal(t, "team", loaded.Share)
	assert.True(t, loaded.Mentionable)
	assert.False(t, loaded.Automated)
	assert.True(t, loaded.DisableGlobalPrompts)
	assert.Equal(t, 100, loaded.Sort)

	// Tags
	assert.Len(t, loaded.Tags, 3)
	assert.Contains(t, loaded.Tags, "Test")
	assert.Contains(t, loaded.Tags, "AllFields")
	assert.Contains(t, loaded.Tags, "Complete")

	// Options
	assert.Equal(t, 0.8, loaded.Options["temperature"])
	assert.Equal(t, float64(2000), loaded.Options["max_tokens"])

	// Prompts
	assert.Len(t, loaded.Prompts, 2)

	// Prompt presets
	assert.NotNil(t, loaded.PromptPresets)
	assert.Contains(t, loaded.PromptPresets, "default")

	// Placeholder
	assert.NotNil(t, loaded.Placeholder)
	assert.Equal(t, "Test Placeholder", loaded.Placeholder.Title)
	assert.Equal(t, "This is a test placeholder", loaded.Placeholder.Description)
	assert.Len(t, loaded.Placeholder.Prompts, 2)

	// Script from source
	assert.NotNil(t, loaded.HookScript)
	assert.NotEmpty(t, loaded.Source)

	// Execute the Create hook to verify it works
	ctx := newStoreTestContext("test-chat-all-fields", assistantID)
	messages := []context.Message{{Role: "user", Content: "Test message"}}

	res, _, err := loaded.HookScript.Create(ctx, messages, &context.Options{})
	require.NoError(t, err, "Create hook should execute without error")
	require.NotNil(t, res, "Create hook should return a response")

	// Verify hook returned expected values
	require.NotNil(t, res.Temperature, "Temperature should be set")
	assert.Equal(t, 0.5, *res.Temperature, "Temperature should be 0.5")

	require.NotNil(t, res.Metadata, "Metadata should be set")
	assert.Equal(t, "Test All Fields", res.Metadata["assistant_name"], "assistant_name should match")
	assert.Equal(t, true, res.Metadata["executed"], "executed should be true")
}

// TestLoadStoreHookWithTypeScript tests that TypeScript features work in Source field
func TestLoadStoreHookWithTypeScript(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

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
			// TypeScript code with type annotations and interfaces
			Source: `
// TypeScript interfaces
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

// Create hook with full TypeScript syntax
function Create(ctx: CreateContext, messages: Message[]): CreateResponse | null {
	// Type-safe access to context
	const chatId: string = ctx.chat_id || "unknown";
	const locale: string = ctx.locale || "en-us";
	const userId: string = ctx.authorized?.user_id || "anonymous";
	
	// Process messages
	const userMessages: Message[] = messages.filter((m: Message) => m.role === "user");
	const messageCount: number = userMessages.length;
	
	// Return typed response
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

	// Execute the Create hook
	ctx := newStoreTestContext("ts-test-chat", assistantID)
	messages := []context.Message{
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Hi there"},
		{Role: "user", Content: "How are you?"},
	}

	res, _, err := loaded.HookScript.Create(ctx, messages, &context.Options{})
	require.NoError(t, err, "TypeScript Create hook should execute without error")
	require.NotNil(t, res, "Create hook should return a response")

	// Verify temperature
	require.NotNil(t, res.Temperature)
	assert.Equal(t, 0.7, *res.Temperature)

	// Verify messages
	require.Len(t, res.Messages, 1)
	assert.Equal(t, context.RoleSystem, res.Messages[0].Role)
	assert.Equal(t, "TypeScript hook executed successfully", res.Messages[0].Content)

	// Verify metadata
	require.NotNil(t, res.Metadata)
	assert.Equal(t, "ts-test-chat", res.Metadata["chat_id"])
	assert.Equal(t, "en-us", res.Metadata["locale"])
	assert.Equal(t, "test-user-123", res.Metadata["user_id"])
	assert.Equal(t, float64(2), res.Metadata["message_count"]) // 2 user messages
	assert.Equal(t, true, res.Metadata["typescript_features"])
}

// TestLoadStoreHookReturnNull tests that hook returning null works correctly
func TestLoadStoreHookReturnNull(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

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
	// Return null to indicate no modifications
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

// TestLoadStoreHookWithPromptPreset tests that hook can return prompt_preset
func TestLoadStoreHookWithPromptPreset(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

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
	// Check first message to determine preset
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

	// Test friendly preset selection
	t.Run("SelectFriendlyPreset", func(t *testing.T) {
		ctx := newStoreTestContext("preset-test-1", assistantID)
		messages := []context.Message{{Role: "user", Content: "Be friendly please"}}

		res, _, err := loaded.HookScript.Create(ctx, messages, &context.Options{})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Equal(t, "friendly", res.PromptPreset)
	})

	// Test professional preset selection
	t.Run("SelectProfessionalPreset", func(t *testing.T) {
		ctx := newStoreTestContext("preset-test-2", assistantID)
		messages := []context.Message{{Role: "user", Content: "Be professional"}}

		res, _, err := loaded.HookScript.Create(ctx, messages, &context.Options{})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Equal(t, "professional", res.PromptPreset)
	})

	// Test no preset (returns null)
	t.Run("NoPreset", func(t *testing.T) {
		ctx := newStoreTestContext("preset-test-3", assistantID)
		messages := []context.Message{{Role: "user", Content: "Hello"}}

		res, _, err := loaded.HookScript.Create(ctx, messages, &context.Options{})
		require.NoError(t, err)
		assert.Nil(t, res)
	})
}

// TestLoadStoreHookDisableGlobalPrompts tests that hook can disable global prompts
func TestLoadStoreHookDisableGlobalPrompts(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

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

	// Test disable global prompts
	t.Run("DisableGlobalPrompts", func(t *testing.T) {
		ctx := newStoreTestContext("disable-test-1", assistantID)
		messages := []context.Message{{Role: "user", Content: "disable_global prompts"}}

		res, _, err := loaded.HookScript.Create(ctx, messages, &context.Options{})
		require.NoError(t, err)
		require.NotNil(t, res)
		require.NotNil(t, res.DisableGlobalPrompts)
		assert.True(t, *res.DisableGlobalPrompts)
	})

	// Test enable global prompts
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

// TestLoadStoreWithSearchConfig tests loading assistant with search configuration from database
func TestLoadStoreWithSearchConfig(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

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

	// Verify Uses
	require.NotNil(t, loaded.Uses)
	assert.Equal(t, "agent", loaded.Uses.Vision)
	assert.Equal(t, "mcp:audio-server", loaded.Uses.Audio)
	assert.Equal(t, "agent", loaded.Uses.Fetch)
	assert.Equal(t, "builtin", loaded.Uses.Web)
	assert.Equal(t, "builtin", loaded.Uses.Keyword)
	assert.Equal(t, "builtin", loaded.Uses.QueryDSL)
	assert.Equal(t, "builtin", loaded.Uses.Rerank)

	// Verify Search config
	require.NotNil(t, loaded.Search)

	// Web config
	require.NotNil(t, loaded.Search.Web)
	assert.Equal(t, "tavily", loaded.Search.Web.Provider)
	assert.Equal(t, 15, loaded.Search.Web.MaxResults)

	// KB config
	require.NotNil(t, loaded.Search.KB)
	assert.Equal(t, []string{"docs", "faq"}, loaded.Search.KB.Collections)
	assert.Equal(t, 0.8, loaded.Search.KB.Threshold)
	assert.True(t, loaded.Search.KB.Graph)

	// DB config
	require.NotNil(t, loaded.Search.DB)
	assert.Equal(t, []string{"user", "product"}, loaded.Search.DB.Models)
	assert.Equal(t, 50, loaded.Search.DB.MaxResults)

	// Keyword config
	require.NotNil(t, loaded.Search.Keyword)
	assert.Equal(t, 8, loaded.Search.Keyword.MaxKeywords)
	assert.Equal(t, "auto", loaded.Search.Keyword.Language)

	// QueryDSL config
	require.NotNil(t, loaded.Search.QueryDSL)
	assert.True(t, loaded.Search.QueryDSL.Strict)

	// Rerank config
	require.NotNil(t, loaded.Search.Rerank)
	assert.Equal(t, 5, loaded.Search.Rerank.TopN)

	// Citation config
	require.NotNil(t, loaded.Search.Citation)
	assert.Equal(t, "#cite:{id}", loaded.Search.Citation.Format)
	assert.False(t, loaded.Search.Citation.AutoInjectPrompt)
	assert.Equal(t, "Please cite sources.", loaded.Search.Citation.CustomPrompt)

	// Weights config
	require.NotNil(t, loaded.Search.Weights)
	assert.Equal(t, 1.0, loaded.Search.Weights.User)
	assert.Equal(t, 0.85, loaded.Search.Weights.Hook)
	assert.Equal(t, 0.65, loaded.Search.Weights.Auto)

	// Options config
	require.NotNil(t, loaded.Search.Options)
	assert.Equal(t, 3, loaded.Search.Options.SkipThreshold)
}

// TestLoadStoreWithPartialSearchConfig tests loading assistant with partial search configuration
func TestLoadStoreWithPartialSearchConfig(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

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
				// Only web config, others are nil
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

	// Verify Search config
	require.NotNil(t, loaded.Search)

	// Web config should be set
	require.NotNil(t, loaded.Search.Web)
	assert.Equal(t, "serper", loaded.Search.Web.Provider)

	// Other configs should be nil
	assert.Nil(t, loaded.Search.KB)
	assert.Nil(t, loaded.Search.DB)
	assert.Nil(t, loaded.Search.Keyword)
	assert.Nil(t, loaded.Search.QueryDSL)
	assert.Nil(t, loaded.Search.Rerank)
	assert.Nil(t, loaded.Search.Citation)
	assert.Nil(t, loaded.Search.Weights)
	assert.Nil(t, loaded.Search.Options)
}

// TestLoadStoreWithoutSearchConfig tests loading assistant without search configuration
func TestLoadStoreWithoutSearchConfig(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	assistantID := "test.store-no-search"
	now := time.Now().UnixNano()

	ast := &assistant.Assistant{
		AssistantModel: store.AssistantModel{
			ID:        assistantID,
			Name:      "Test No Search Config",
			Type:      "assistant",
			Connector: "gpt-4o",
			// No Search config
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

	// Search config should be nil
	assert.Nil(t, loaded.Search)
}
