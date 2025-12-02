package assistant_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/assistant"
	"github.com/yaoapp/yao/agent/context"
	store "github.com/yaoapp/yao/agent/store/types"
	"github.com/yaoapp/yao/agent/testutils"
	"github.com/yaoapp/yao/openapi/oauth/types"
)

func TestBuildSystemPromptsIntegration(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	t.Run("AssistantWithLocale", func(t *testing.T) {
		// Load an assistant with locales
		ast, err := assistant.Get("tests.fullfields")
		require.NoError(t, err)

		ctx := &context.Context{
			Locale: "zh-cn",
			Authorized: &types.AuthorizedInfo{
				UserID: "test-user-123",
				TeamID: "test-team-456",
			},
			Metadata: map[string]interface{}{
				"CUSTOM_VAR": "custom-value",
				"INT_VAR":    42,
				"BOOL_VAR":   true,
			},
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

		ctx := &context.Context{
			Locale: "en-us",
		}

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

		ctx := &context.Context{
			Metadata: map[string]interface{}{
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
			},
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

		ctx := &context.Context{
			Authorized: &types.AuthorizedInfo{
				UserID:   "user-123",
				Subject:  "user@example.com", // PII - should not be exposed
				TeamID:   "team-456",
				TenantID: "tenant-789",
			},
			Client: context.Client{
				Type: "web",
				IP:   "192.168.1.1", // Should not be exposed
			},
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

		ctx := &context.Context{
			Authorized: &types.AuthorizedInfo{
				UserID: "user-abc",
				TeamID: "team-xyz",
			},
			Metadata: map[string]interface{}{
				"MY_VAR": "my-value",
			},
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

		ctx := &context.Context{}

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

		ctx := &context.Context{}

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

		ctx := &context.Context{
			Authorized: &types.AuthorizedInfo{
				UserID: "all-vars-user",
			},
			Metadata: map[string]interface{}{
				"CUSTOM_KEY": "custom-value-123",
			},
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
}
