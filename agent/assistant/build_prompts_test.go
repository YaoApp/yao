package assistant_test

import (
	stdContext "context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/assistant"
	"github.com/yaoapp/yao/agent/context"
	store "github.com/yaoapp/yao/agent/store/types"
	"github.com/yaoapp/yao/agent/testutils"
	"github.com/yaoapp/yao/openapi/oauth/types"
)

// containsString is a helper to check if a content (string or interface{}) contains a substring
func containsString(content interface{}, substr string) bool {
	switch v := content.(type) {
	case string:
		return strings.Contains(v, substr)
	default:
		return false
	}
}

// newPromptTestContext creates a context suitable for prompt testing with Create Hook
func newPromptTestContext(chatID, assistantID string) *context.Context {
	authorized := &types.AuthorizedInfo{
		Subject:   "test-user",
		ClientID:  "test-client-id",
		UserID:    "test-user-123",
		TeamID:    "test-team-456",
		TenantID:  "test-tenant-789",
		SessionID: "test-session-id",
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
	ctx.Metadata = make(map[string]interface{})
	return ctx
}

// newMinimalTestContext creates a minimal context for testing
// Use this when you only need specific fields set
func newMinimalTestContext() *context.Context {
	return context.New(stdContext.Background(), nil, "test-chat")
}

func TestBuildSystemPromptsIntegration(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	t.Run("AssistantWithLocale", func(t *testing.T) {
		// Load an assistant with locales
		ast, err := assistant.Get("tests.fullfields")
		require.NoError(t, err)

		ctx := newMinimalTestContext()
		ctx.Locale = "zh-cn"
		ctx.Authorized = &types.AuthorizedInfo{
			UserID: "test-user-123",
			TeamID: "test-team-456",
		}
		ctx.Metadata = map[string]interface{}{
			"CUSTOM_VAR": "custom-value",
			"INT_VAR":    42,
			"BOOL_VAR":   true,
		}

		// Build request to test the full flow
		messages := []context.Message{
			{Role: context.RoleUser, Content: "Hello"},
		}

		finalMessages, options, err := ast.BuildRequest(ctx, messages, nil)
		require.NoError(t, err)
		require.NotNil(t, options)

		// Should have system prompts prepended
		assert.Greater(t, len(finalMessages), 1)

		// First messages should be system prompts
		hasSystemPrompt := false
		for _, msg := range finalMessages {
			if msg.Role == context.RoleSystem {
				hasSystemPrompt = true
				break
			}
		}
		assert.True(t, hasSystemPrompt, "Should have system prompts")
	})

	t.Run("DisableGlobalPrompts", func(t *testing.T) {
		// Load fullfields assistant which has disable_global_prompts: true
		ast, err := assistant.Get("tests.fullfields")
		require.NoError(t, err)
		require.True(t, ast.DisableGlobalPrompts)

		ctx := newMinimalTestContext()
		ctx.Locale = "en-us"

		messages := []context.Message{
			{Role: context.RoleUser, Content: "Hello"},
		}

		finalMessages, _, err := ast.BuildRequest(ctx, messages, nil)
		require.NoError(t, err)

		// Should still have assistant prompts
		hasSystemPrompt := false
		for _, msg := range finalMessages {
			if msg.Role == context.RoleSystem {
				hasSystemPrompt = true
				break
			}
		}
		assert.True(t, hasSystemPrompt, "Should have assistant prompts even with global disabled")
	})

	t.Run("MetadataTypeConversion", func(t *testing.T) {
		ast, err := assistant.Get("yaobots")
		require.NoError(t, err)

		ctx := newMinimalTestContext()
		ctx.Metadata = map[string]interface{}{
			"STRING_VAL": "hello",
			"INT_VAL":    123,
			"INT64_VAL":  int64(456),
			"FLOAT_VAL":  3.14,
			"BOOL_TRUE":  true,
			"BOOL_FALSE": false,
			"UINT_VAL":   uint(789),
			"NIL_VAL":    nil,
			"EMPTY_VAL":  "",
			"ZERO_INT":   0,
			"ZERO_FLOAT": 0.0,
		}

		messages := []context.Message{
			{Role: context.RoleUser, Content: "Test metadata"},
		}

		// This should not panic
		_, _, err = ast.BuildRequest(ctx, messages, nil)
		require.NoError(t, err)
	})

	t.Run("AuthorizedInfoPrivacy", func(t *testing.T) {
		ast, err := assistant.Get("yaobots")
		require.NoError(t, err)

		ctx := newMinimalTestContext()
		ctx.Authorized = &types.AuthorizedInfo{
			UserID:   "user-123",
			Subject:  "user@example.com", // PII - should not be exposed
			TeamID:   "team-456",
			TenantID: "tenant-789",
		}
		ctx.Client = context.Client{
			Type: "web",
			IP:   "192.168.1.1", // Should not be exposed
		}

		messages := []context.Message{
			{Role: context.RoleUser, Content: "Test privacy"},
		}

		finalMessages, _, err := ast.BuildRequest(ctx, messages, nil)
		require.NoError(t, err)

		// Check that sensitive info is not in any system prompts
		for _, msg := range finalMessages {
			if msg.Role == context.RoleSystem {
				assert.NotContains(t, msg.Content, "user@example.com", "Subject should not be in prompts")
				assert.NotContains(t, msg.Content, "192.168.1.1", "IP should not be in prompts")
			}
		}
	})

	t.Run("ContextVariablesInPrompts", func(t *testing.T) {
		// Set up global prompts with variables
		assistant.SetGlobalPrompts([]store.Prompt{
			{Role: "system", Content: "User ID: $CTX.USER_ID, Team: $CTX.TEAM_ID, Custom: $CTX.MY_VAR"},
		})
		defer assistant.SetGlobalPrompts(nil)

		ast, err := assistant.Get("yaobots")
		require.NoError(t, err)

		ctx := newMinimalTestContext()
		ctx.Authorized = &types.AuthorizedInfo{
			UserID: "user-abc",
			TeamID: "team-xyz",
		}
		ctx.Metadata = map[string]interface{}{
			"MY_VAR": "my-value",
		}

		messages := []context.Message{
			{Role: context.RoleUser, Content: "Test variables"},
		}

		finalMessages, _, err := ast.BuildRequest(ctx, messages, nil)
		require.NoError(t, err)

		// Find the global prompt and verify variables are replaced
		found := false
		for _, msg := range finalMessages {
			if msg.Role == context.RoleSystem && !found {
				if assert.Contains(t, msg.Content, "User ID: user-abc") {
					found = true
					assert.Contains(t, msg.Content, "Team: team-xyz")
					assert.Contains(t, msg.Content, "Custom: my-value")
				}
			}
		}
		assert.True(t, found, "Should find global prompt with replaced variables")
	})

	t.Run("SystemVariablesReplacement", func(t *testing.T) {
		// Set up global prompts with $SYS.* variables
		assistant.SetGlobalPrompts([]store.Prompt{
			{Role: "system", Content: "Time: $SYS.TIME, Date: $SYS.DATE, Datetime: $SYS.DATETIME, Weekday: $SYS.WEEKDAY"},
		})
		defer assistant.SetGlobalPrompts(nil)

		ast, err := assistant.Get("yaobots")
		require.NoError(t, err)

		ctx := newMinimalTestContext()

		messages := []context.Message{
			{Role: context.RoleUser, Content: "Test system variables"},
		}

		finalMessages, _, err := ast.BuildRequest(ctx, messages, nil)
		require.NoError(t, err)

		// Find the global prompt and verify $SYS.* variables are replaced
		found := false
		for _, msg := range finalMessages {
			if msg.Role == context.RoleSystem {
				// Should NOT contain $SYS. prefix (variables should be replaced)
				if !assert.NotContains(t, msg.Content, "$SYS.TIME") {
					continue
				}
				if !assert.NotContains(t, msg.Content, "$SYS.DATE") {
					continue
				}
				if !assert.NotContains(t, msg.Content, "$SYS.DATETIME") {
					continue
				}
				if !assert.NotContains(t, msg.Content, "$SYS.WEEKDAY") {
					continue
				}

				// Should contain "Time:", "Date:", etc. with actual values
				assert.Contains(t, msg.Content, "Time:")
				assert.Contains(t, msg.Content, "Date:")
				assert.Contains(t, msg.Content, "Datetime:")
				assert.Contains(t, msg.Content, "Weekday:")
				found = true
				break
			}
		}
		assert.True(t, found, "Should find global prompt with replaced $SYS.* variables")
	})

	t.Run("EnvVariablesReplacement", func(t *testing.T) {
		// Set test environment variable
		t.Setenv("TEST_PROMPT_VAR", "env-test-value")

		// Set up global prompts with $ENV.* variables
		assistant.SetGlobalPrompts([]store.Prompt{
			{Role: "system", Content: "Env Value: $ENV.TEST_PROMPT_VAR, Not Exist: $ENV.NOT_EXIST_VAR_XYZ"},
		})
		defer assistant.SetGlobalPrompts(nil)

		ast, err := assistant.Get("yaobots")
		require.NoError(t, err)

		ctx := newMinimalTestContext()

		messages := []context.Message{
			{Role: context.RoleUser, Content: "Test env variables"},
		}

		finalMessages, _, err := ast.BuildRequest(ctx, messages, nil)
		require.NoError(t, err)

		// Find the global prompt and verify $ENV.* variables are replaced
		found := false
		for _, msg := range finalMessages {
			if msg.Role == context.RoleSystem {
				// Should NOT contain $ENV. prefix for existing vars
				if !assert.NotContains(t, msg.Content, "$ENV.TEST_PROMPT_VAR") {
					continue
				}
				// Should contain the actual env value
				assert.Contains(t, msg.Content, "Env Value: env-test-value")
				// Non-existent env var should be replaced with empty string
				assert.Contains(t, msg.Content, "Not Exist: ")
				assert.NotContains(t, msg.Content, "$ENV.NOT_EXIST_VAR_XYZ")
				found = true
				break
			}
		}
		assert.True(t, found, "Should find global prompt with replaced $ENV.* variables")
	})

	t.Run("AllVariableTypesReplacement", func(t *testing.T) {
		// Set test environment variable
		t.Setenv("TEST_APP_NAME", "MyTestApp")

		// Set up global prompts with all variable types
		assistant.SetGlobalPrompts([]store.Prompt{
			{Role: "system", Content: `System Info:
- Time: $SYS.TIME
- Date: $SYS.DATE
- App: $ENV.TEST_APP_NAME
- User: $CTX.USER_ID
- Custom: $CTX.CUSTOM_KEY
- Assistant: $CTX.ASSISTANT_NAME`},
		})
		defer assistant.SetGlobalPrompts(nil)

		ast, err := assistant.Get("yaobots")
		require.NoError(t, err)

		ctx := newMinimalTestContext()
		ctx.Authorized = &types.AuthorizedInfo{
			UserID: "all-vars-user",
		}
		ctx.Metadata = map[string]interface{}{
			"CUSTOM_KEY": "custom-value-123",
		}

		messages := []context.Message{
			{Role: context.RoleUser, Content: "Test all variables"},
		}

		finalMessages, _, err := ast.BuildRequest(ctx, messages, nil)
		require.NoError(t, err)

		// Find the global prompt and verify ALL variable types are replaced
		found := false
		for _, msg := range finalMessages {
			if msg.Role == context.RoleSystem && !found {
				content := msg.Content

				// Check $SYS.* replaced
				if assert.NotContains(t, content, "$SYS.TIME") &&
					assert.NotContains(t, content, "$SYS.DATE") {

					// Check $ENV.* replaced
					assert.NotContains(t, content, "$ENV.TEST_APP_NAME")
					assert.Contains(t, content, "App: MyTestApp")

					// Check $CTX.* replaced
					assert.NotContains(t, content, "$CTX.USER_ID")
					assert.Contains(t, content, "User: all-vars-user")

					assert.NotContains(t, content, "$CTX.CUSTOM_KEY")
					assert.Contains(t, content, "Custom: custom-value-123")

					// Check assistant name from $CTX.ASSISTANT_NAME
					assert.NotContains(t, content, "$CTX.ASSISTANT_NAME")

					found = true
				}
			}
		}
		assert.True(t, found, "Should find global prompt with all variable types replaced")
	})

	t.Run("PromptPresetFromHook", func(t *testing.T) {
		// Load fullfields assistant which has prompt_presets
		ast, err := assistant.Get("tests.fullfields")
		require.NoError(t, err)
		require.NotNil(t, ast.PromptPresets)
		require.Contains(t, ast.PromptPresets, "chat.friendly")

		ctx := newMinimalTestContext()

		messages := []context.Message{
			{Role: context.RoleUser, Content: "Test preset from hook"},
		}

		// Hook returns prompt_preset
		createResponse := &context.HookCreateResponse{
			PromptPreset: "chat.friendly",
		}

		finalMessages, _, err := ast.BuildRequest(ctx, messages, createResponse)
		require.NoError(t, err)

		// Should have system prompts from the preset
		hasSystemPrompt := false
		for _, msg := range finalMessages {
			if msg.Role == context.RoleSystem {
				hasSystemPrompt = true
				// Verify it's from the friendly preset (check content)
				assert.Contains(t, msg.Content, "friendly", "Should use friendly preset prompts")
				break
			}
		}
		assert.True(t, hasSystemPrompt, "Should have system prompts from preset")
	})

	t.Run("PromptPresetFromMetadata", func(t *testing.T) {
		// Load fullfields assistant which has prompt_presets
		ast, err := assistant.Get("tests.fullfields")
		require.NoError(t, err)

		ctx := newMinimalTestContext()
		ctx.Metadata = map[string]interface{}{
			"__prompt_preset": "chat.professional",
		}

		messages := []context.Message{
			{Role: context.RoleUser, Content: "Test preset from metadata"},
		}

		finalMessages, _, err := ast.BuildRequest(ctx, messages, nil)
		require.NoError(t, err)

		// Should have system prompts from the preset
		hasSystemPrompt := false
		for _, msg := range finalMessages {
			if msg.Role == context.RoleSystem {
				hasSystemPrompt = true
				// Verify it's from the professional preset
				assert.Contains(t, msg.Content, "professional", "Should use professional preset prompts")
				break
			}
		}
		assert.True(t, hasSystemPrompt, "Should have system prompts from preset")
	})

	t.Run("PromptPresetHookOverridesMetadata", func(t *testing.T) {
		// Load fullfields assistant
		ast, err := assistant.Get("tests.fullfields")
		require.NoError(t, err)

		ctx := newMinimalTestContext()
		ctx.Metadata = map[string]interface{}{
			"__prompt_preset": "chat.professional", // Lower priority
		}

		messages := []context.Message{
			{Role: context.RoleUser, Content: "Test hook overrides metadata"},
		}

		// Hook returns different preset (higher priority)
		createResponse := &context.HookCreateResponse{
			PromptPreset: "chat.friendly",
		}

		finalMessages, _, err := ast.BuildRequest(ctx, messages, createResponse)
		require.NoError(t, err)

		// Should use hook's preset, not metadata's
		for _, msg := range finalMessages {
			if msg.Role == context.RoleSystem {
				assert.Contains(t, msg.Content, "friendly", "Hook preset should override metadata preset")
				break
			}
		}
	})

	t.Run("PromptPresetNotFound", func(t *testing.T) {
		// Load fullfields assistant
		ast, err := assistant.Get("tests.fullfields")
		require.NoError(t, err)

		ctx := newMinimalTestContext()
		ctx.Metadata = map[string]interface{}{
			"__prompt_preset": "non.existent.preset",
		}

		messages := []context.Message{
			{Role: context.RoleUser, Content: "Test non-existent preset"},
		}

		finalMessages, _, err := ast.BuildRequest(ctx, messages, nil)
		require.NoError(t, err)

		// Should fallback to default prompts (not crash)
		hasSystemPrompt := false
		for _, msg := range finalMessages {
			if msg.Role == context.RoleSystem {
				hasSystemPrompt = true
				break
			}
		}
		assert.True(t, hasSystemPrompt, "Should fallback to default prompts when preset not found")
	})

	t.Run("DisableGlobalPromptsFromHook", func(t *testing.T) {
		// Set global prompts
		assistant.SetGlobalPrompts([]store.Prompt{
			{Role: "system", Content: "GLOBAL_PROMPT_MARKER"},
		})
		defer assistant.SetGlobalPrompts(nil)

		// Load an assistant that does NOT disable global prompts
		ast, err := assistant.Get("yaobots")
		require.NoError(t, err)
		require.False(t, ast.DisableGlobalPrompts)

		ctx := newMinimalTestContext()

		messages := []context.Message{
			{Role: context.RoleUser, Content: "Test disable from hook"},
		}

		// Hook disables global prompts
		disableTrue := true
		createResponse := &context.HookCreateResponse{
			DisableGlobalPrompts: &disableTrue,
		}

		finalMessages, _, err := ast.BuildRequest(ctx, messages, createResponse)
		require.NoError(t, err)

		// Should NOT have global prompt
		for _, msg := range finalMessages {
			if msg.Role == context.RoleSystem {
				assert.NotContains(t, msg.Content, "GLOBAL_PROMPT_MARKER", "Global prompts should be disabled by hook")
			}
		}
	})

	t.Run("DisableGlobalPromptsFromMetadata", func(t *testing.T) {
		// Set global prompts
		assistant.SetGlobalPrompts([]store.Prompt{
			{Role: "system", Content: "GLOBAL_PROMPT_MARKER_2"},
		})
		defer assistant.SetGlobalPrompts(nil)

		// Load an assistant that does NOT disable global prompts
		ast, err := assistant.Get("yaobots")
		require.NoError(t, err)

		ctx := newMinimalTestContext()
		ctx.Metadata = map[string]interface{}{
			"__disable_global_prompts": true,
		}

		messages := []context.Message{
			{Role: context.RoleUser, Content: "Test disable from metadata"},
		}

		finalMessages, _, err := ast.BuildRequest(ctx, messages, nil)
		require.NoError(t, err)

		// Should NOT have global prompt
		for _, msg := range finalMessages {
			if msg.Role == context.RoleSystem {
				assert.NotContains(t, msg.Content, "GLOBAL_PROMPT_MARKER_2", "Global prompts should be disabled by metadata")
			}
		}
	})

	t.Run("EnableGlobalPromptsOverrideAssistant", func(t *testing.T) {
		// Set global prompts
		assistant.SetGlobalPrompts([]store.Prompt{
			{Role: "system", Content: "GLOBAL_ENABLED_MARKER"},
		})
		defer assistant.SetGlobalPrompts(nil)

		// Load fullfields assistant which has disable_global_prompts: true
		ast, err := assistant.Get("tests.fullfields")
		require.NoError(t, err)
		require.True(t, ast.DisableGlobalPrompts)

		ctx := newMinimalTestContext()

		messages := []context.Message{
			{Role: context.RoleUser, Content: "Test enable override"},
		}

		// Hook enables global prompts (overrides assistant's disable)
		disableFalse := false
		createResponse := &context.HookCreateResponse{
			DisableGlobalPrompts: &disableFalse,
		}

		finalMessages, _, err := ast.BuildRequest(ctx, messages, createResponse)
		require.NoError(t, err)

		// Should have global prompt (hook enabled it)
		found := false
		for _, msg := range finalMessages {
			if msg.Role == context.RoleSystem && msg.Content == "GLOBAL_ENABLED_MARKER" {
				found = true
				break
			}
		}
		assert.True(t, found, "Global prompts should be enabled by hook override")
	})
}

// TestPromptPresetAssistant tests the tests.promptpreset assistant with Create Hook
func TestPromptPresetAssistant(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	t.Run("LoadPromptPresetAssistant", func(t *testing.T) {
		ast, err := assistant.Get("tests.promptpreset")
		require.NoError(t, err)
		require.NotNil(t, ast)

		assert.Equal(t, "tests.promptpreset", ast.ID)
		assert.Equal(t, "Prompt Preset Test", ast.Name)
		assert.False(t, ast.DisableGlobalPrompts)

		// Should have prompt presets loaded
		require.NotNil(t, ast.PromptPresets)
		assert.Contains(t, ast.PromptPresets, "mode.friendly")
		assert.Contains(t, ast.PromptPresets, "mode.professional")

		// Should have script
		assert.NotNil(t, ast.HookScript)
	})

	t.Run("CreateHookSelectsFriendlyPreset", func(t *testing.T) {
		ast, err := assistant.Get("tests.promptpreset")
		require.NoError(t, err)

		ctx := newPromptTestContext("chat-friendly-test", "tests.promptpreset")

		messages := []context.Message{
			{Role: context.RoleUser, Content: "use friendly mode please"},
		}

		// Call Create hook
		createResponse, _, err := ast.HookScript.Create(ctx, messages, &context.Options{})
		require.NoError(t, err)
		require.NotNil(t, createResponse)
		assert.Equal(t, "mode.friendly", createResponse.PromptPreset)

		// Build request
		finalMessages, _, err := ast.BuildRequest(ctx, messages, createResponse)
		require.NoError(t, err)

		// Should have friendly preset marker in one of the system messages
		found := false
		for _, msg := range finalMessages {
			if msg.Role == context.RoleSystem && containsString(msg.Content, "FRIENDLY_PRESET_MARKER") {
				found = true
				break
			}
		}
		assert.True(t, found, "Should use friendly preset from Create Hook")
	})

	t.Run("CreateHookSelectsProfessionalPreset", func(t *testing.T) {
		ast, err := assistant.Get("tests.promptpreset")
		require.NoError(t, err)

		ctx := newPromptTestContext("chat-professional-test", "tests.promptpreset")

		messages := []context.Message{
			{Role: context.RoleUser, Content: "use professional tone"},
		}

		// Call Create hook
		createResponse, _, err := ast.HookScript.Create(ctx, messages, &context.Options{})
		require.NoError(t, err)
		require.NotNil(t, createResponse)
		assert.Equal(t, "mode.professional", createResponse.PromptPreset)

		// Build request
		finalMessages, _, err := ast.BuildRequest(ctx, messages, createResponse)
		require.NoError(t, err)

		// Should have professional preset marker in one of the system messages
		found := false
		for _, msg := range finalMessages {
			if msg.Role == context.RoleSystem && containsString(msg.Content, "PROFESSIONAL_PRESET_MARKER") {
				found = true
				break
			}
		}
		assert.True(t, found, "Should use professional preset from Create Hook")
	})

	t.Run("CreateHookDisablesGlobalPrompts", func(t *testing.T) {
		// Set global prompts
		assistant.SetGlobalPrompts([]store.Prompt{
			{Role: "system", Content: "GLOBAL_MARKER_FOR_DISABLE_TEST"},
		})
		defer assistant.SetGlobalPrompts(nil)

		ast, err := assistant.Get("tests.promptpreset")
		require.NoError(t, err)

		ctx := newPromptTestContext("chat-disable-global-test", "tests.promptpreset")

		messages := []context.Message{
			{Role: context.RoleUser, Content: "disable global prompts"},
		}

		// Call Create hook
		createResponse, _, err := ast.HookScript.Create(ctx, messages, &context.Options{})
		require.NoError(t, err)
		require.NotNil(t, createResponse)
		require.NotNil(t, createResponse.DisableGlobalPrompts)
		assert.True(t, *createResponse.DisableGlobalPrompts)

		// Build request
		finalMessages, _, err := ast.BuildRequest(ctx, messages, createResponse)
		require.NoError(t, err)

		// Should NOT have global prompt
		for _, msg := range finalMessages {
			if msg.Role == context.RoleSystem {
				assert.NotContains(t, msg.Content, "GLOBAL_MARKER_FOR_DISABLE_TEST")
			}
		}
	})

	t.Run("CreateHookPresetAndDisableGlobal", func(t *testing.T) {
		// Set global prompts
		assistant.SetGlobalPrompts([]store.Prompt{
			{Role: "system", Content: "GLOBAL_MARKER_COMBINED_TEST"},
		})
		defer assistant.SetGlobalPrompts(nil)

		ast, err := assistant.Get("tests.promptpreset")
		require.NoError(t, err)

		ctx := newPromptTestContext("chat-combined-test", "tests.promptpreset")

		messages := []context.Message{
			{Role: context.RoleUser, Content: "friendly no global"},
		}

		// Call Create hook
		createResponse, _, err := ast.HookScript.Create(ctx, messages, &context.Options{})
		require.NoError(t, err)
		require.NotNil(t, createResponse)
		assert.Equal(t, "mode.friendly", createResponse.PromptPreset)
		require.NotNil(t, createResponse.DisableGlobalPrompts)
		assert.True(t, *createResponse.DisableGlobalPrompts)

		// Build request
		finalMessages, _, err := ast.BuildRequest(ctx, messages, createResponse)
		require.NoError(t, err)

		// Should have friendly preset but NOT global
		hasFriendly := false
		for _, msg := range finalMessages {
			if msg.Role == context.RoleSystem {
				assert.NotContains(t, msg.Content, "GLOBAL_MARKER_COMBINED_TEST")
				if containsString(msg.Content, "FRIENDLY_PRESET_MARKER") {
					hasFriendly = true
				}
			}
		}
		assert.True(t, hasFriendly, "Should have friendly preset")
	})

	t.Run("CreateHookUnknownPresetFallback", func(t *testing.T) {
		ast, err := assistant.Get("tests.promptpreset")
		require.NoError(t, err)

		ctx := newPromptTestContext("chat-unknown-preset-test", "tests.promptpreset")

		messages := []context.Message{
			{Role: context.RoleUser, Content: "unknown preset test"},
		}

		// Call Create hook
		createResponse, _, err := ast.HookScript.Create(ctx, messages, &context.Options{})
		require.NoError(t, err)
		require.NotNil(t, createResponse)
		assert.Equal(t, "non.existent.preset", createResponse.PromptPreset)

		// Build request - should not error, fallback to default
		finalMessages, _, err := ast.BuildRequest(ctx, messages, createResponse)
		require.NoError(t, err)

		// Should fallback to default prompts
		found := false
		for _, msg := range finalMessages {
			if msg.Role == context.RoleSystem && containsString(msg.Content, "DEFAULT_PROMPT_MARKER") {
				found = true
				break
			}
		}
		assert.True(t, found, "Should fallback to default prompts when preset not found")
	})

	t.Run("CreateHookReturnsNull", func(t *testing.T) {
		ast, err := assistant.Get("tests.promptpreset")
		require.NoError(t, err)

		ctx := newPromptTestContext("chat-null-test", "tests.promptpreset")

		messages := []context.Message{
			{Role: context.RoleUser, Content: "just a normal message"},
		}

		// Call Create hook - should return nil
		createResponse, _, err := ast.HookScript.Create(ctx, messages, &context.Options{})
		require.NoError(t, err)
		assert.Nil(t, createResponse)

		// Build request with nil createResponse
		finalMessages, _, err := ast.BuildRequest(ctx, messages, nil)
		require.NoError(t, err)

		// Should use default prompts
		found := false
		for _, msg := range finalMessages {
			if msg.Role == context.RoleSystem && containsString(msg.Content, "DEFAULT_PROMPT_MARKER") {
				found = true
				break
			}
		}
		assert.True(t, found, "Should use default prompts when hook returns null")
	})
}
