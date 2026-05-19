//go:build integration

package assistant_test

import (
	stdContext "context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/assistant"
	agentContext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

func TestBuildRequestBasic(t *testing.T) {
	testprepare.PrepareSandbox(t)

	ast, err := assistant.Get("tests.fullfields")
	require.NoError(t, err)
	require.NotNil(t, ast)

	ctx := newTestContext("test-chat-build-basic", "tests.fullfields")

	messages := []agentContext.Message{
		{Role: agentContext.RoleUser, Content: "Hello"},
	}

	finalMessages, options, err := ast.BuildRequest(ctx, messages, nil)
	require.NoError(t, err)
	require.NotEmpty(t, finalMessages, "returned messages should not be empty")
	require.NotNil(t, options, "returned options should not be nil")
	assert.Equal(t, agentContext.MessageRole("system"), finalMessages[0].Role, "first message should be system role")
}

func TestBuildSystemPrompts(t *testing.T) {
	testprepare.PrepareSandbox(t)

	ast, err := assistant.Get("tests.fullfields")
	require.NoError(t, err)

	ctx := newTestContext("test-chat-build-prompts", "tests.fullfields")
	ctx.Locale = "en-us"
	ctx.Theme = "dark"

	promptMessages := assistant.ExportBuildSystemPrompts(ast, ctx, nil)
	require.NotEmpty(t, promptMessages, "system prompts should not be empty")

	found := false
	for _, msg := range promptMessages {
		if msg.Role == agentContext.RoleSystem {
			content, ok := msg.Content.(string)
			if !ok {
				continue
			}
			assert.NotContains(t, content, "$CTX.LOCALE", "LOCALE variable should be replaced")
			assert.NotContains(t, content, "$CTX.ASSISTANT_NAME", "ASSISTANT_NAME variable should be replaced")
			if containsString(msg.Content, "en-us") {
				found = true
			}
		}
	}
	assert.True(t, found, "should find system prompt with locale value replaced")
}

func TestPromptPresetSelection(t *testing.T) {
	testprepare.PrepareSandbox(t)

	ast, err := assistant.Get("tests.hook-prompt-preset")
	require.NoError(t, err)
	require.NotNil(t, ast)

	ctx := newTestContext("test-chat-preset", "tests.hook-prompt-preset")

	t.Run("DefaultPrompts", func(t *testing.T) {
		prompts := assistant.ExportGetAssistantPrompts(ast, ctx, nil)
		require.NotEmpty(t, prompts, "default prompts should not be empty")
		assert.Contains(t, prompts[0].Content, "default", "should return default prompts")
	})

	t.Run("PresetFromCreateResponse", func(t *testing.T) {
		createResponse := &agentContext.HookCreateResponse{
			PromptPreset: "chat.prompts",
		}
		prompts := assistant.ExportGetAssistantPrompts(ast, ctx, createResponse)
		require.NotEmpty(t, prompts, "chat preset prompts should not be empty")
		assert.Contains(t, prompts[0].Content, "chat mode", "should return chat preset prompts")
	})

	t.Run("PresetFromMetadata", func(t *testing.T) {
		metaCtx := agentContext.New(stdContext.Background(), ctx.Authorized, "test-chat-preset-meta")
		metaCtx.AssistantID = "tests.hook-prompt-preset"
		metaCtx.Locale = "en-us"
		metaCtx.Theme = "light"
		metaCtx.Metadata = map[string]interface{}{
			"__prompt_preset": "task.prompts",
		}

		prompts := assistant.ExportGetAssistantPrompts(ast, metaCtx, nil)
		require.NotEmpty(t, prompts, "task preset prompts should not be empty")
		assert.Contains(t, prompts[0].Content, "task mode", "should return task preset prompts")
	})
}

func TestDisableGlobalPrompts(t *testing.T) {
	testprepare.PrepareSandbox(t)

	ctx := newTestContext("test-chat-disable-global", "tests.hook-disable-global-prompts")

	t.Run("AssistantDisablesGlobal", func(t *testing.T) {
		ast, err := assistant.Get("tests.hook-disable-global-prompts")
		require.NoError(t, err)
		result := assistant.ExportShouldDisableGlobalPrompts(ast, ctx, nil)
		assert.True(t, result, "assistant with disable_global_prompts=true should return true")
	})

	t.Run("AssistantDoesNotDisableGlobal", func(t *testing.T) {
		ast, err := assistant.Get("tests.fullfields")
		require.NoError(t, err)
		ctxFF := newTestContext("test-chat-no-disable", "tests.fullfields")
		result := assistant.ExportShouldDisableGlobalPrompts(ast, ctxFF, nil)
		assert.False(t, result, "assistant without disable_global_prompts should return false")
	})

	t.Run("OverrideByCreateResponse", func(t *testing.T) {
		ast, err := assistant.Get("tests.fullfields")
		require.NoError(t, err)
		ctxFF := newTestContext("test-chat-override-cr", "tests.fullfields")

		disableTrue := true
		createResponse := &agentContext.HookCreateResponse{
			DisableGlobalPrompts: &disableTrue,
		}
		result := assistant.ExportShouldDisableGlobalPrompts(ast, ctxFF, createResponse)
		assert.True(t, result, "createResponse.DisableGlobalPrompts=true should override assistant setting")
	})

	t.Run("OverrideByMetadata", func(t *testing.T) {
		ast, err := assistant.Get("tests.fullfields")
		require.NoError(t, err)
		metaCtx := newTestContext("test-chat-override-meta", "tests.fullfields")
		metaCtx.Metadata["__disable_global_prompts"] = true

		result := assistant.ExportShouldDisableGlobalPrompts(ast, metaCtx, nil)
		assert.True(t, result, "ctx.Metadata __disable_global_prompts=true should override assistant setting")
	})
}

func TestBuildContextVariables(t *testing.T) {
	testprepare.PrepareSandbox(t)

	ast, err := assistant.Get("tests.fullfields")
	require.NoError(t, err)

	ctx := newTestContext("test-chat-ctx-vars", "tests.fullfields")
	ctx.Locale = "zh-cn"
	ctx.Theme = "dark"

	vars := assistant.ExportBuildContextVariables(ast, ctx)
	require.NotNil(t, vars, "context variables map should not be nil")

	assert.Equal(t, "tests.fullfields", vars["ASSISTANT_ID"])
	assert.NotEmpty(t, vars["ASSISTANT_NAME"], "ASSISTANT_NAME should not be empty")
	assert.Equal(t, "zh-cn", vars["LOCALE"])
	assert.Equal(t, "dark", vars["THEME"])
	assert.Equal(t, "test-user-123", vars["USER_ID"])
	assert.Equal(t, "test-team-456", vars["TEAM_ID"])
	assert.Equal(t, "test-tenant-789", vars["TENANT_ID"])
}
