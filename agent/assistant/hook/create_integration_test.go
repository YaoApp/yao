//go:build integration

package hook_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/assistant"
	agentContext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

func TestCreate(t *testing.T) {
	testprepare.PrepareSandbox(t)

	agent, err := assistant.Get("tests.hook-echo")
	require.NoError(t, err, "failed to get tests.hook-echo assistant")
	require.NotNil(t, agent.HookScript, "tests.hook-echo has no hook script")

	ctx := newTestContext("chat-test-create-hook", "tests.hook-echo")

	t.Run("ReturnNull", func(t *testing.T) {
		res, _, err := agent.HookScript.Create(ctx, []agentContext.Message{{Role: "user", Content: "return_null"}})
		require.NoError(t, err)
		assert.Nil(t, res)
	})

	t.Run("ReturnUndefined", func(t *testing.T) {
		res, _, err := agent.HookScript.Create(ctx, []agentContext.Message{{Role: "user", Content: "return_undefined"}})
		require.NoError(t, err)
		assert.Nil(t, res)
	})

	t.Run("ReturnEmpty", func(t *testing.T) {
		res, _, err := agent.HookScript.Create(ctx, []agentContext.Message{{Role: "user", Content: "return_empty"}})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Empty(t, res.Messages)
	})

	t.Run("ReturnFull", func(t *testing.T) {
		res, _, err := agent.HookScript.Create(ctx, []agentContext.Message{{Role: "user", Content: "return_full"}})
		require.NoError(t, err)
		require.NotNil(t, res)

		require.Len(t, res.Messages, 2)
		assert.Equal(t, agentContext.RoleSystem, res.Messages[0].Role)
		assert.Equal(t, agentContext.RoleUser, res.Messages[1].Role)

		require.NotNil(t, res.Audio)
		assert.Equal(t, "alloy", res.Audio.Voice)
		assert.Equal(t, "mp3", res.Audio.Format)

		require.NotNil(t, res.Temperature)
		assert.InDelta(t, 0.7, *res.Temperature, 0.01)

		require.NotNil(t, res.MaxTokens)
		assert.Equal(t, 2000, *res.MaxTokens)

		require.NotNil(t, res.MaxCompletionTokens)
		assert.Equal(t, 1500, *res.MaxCompletionTokens)
	})

	t.Run("ReturnPartial", func(t *testing.T) {
		res, _, err := agent.HookScript.Create(ctx, []agentContext.Message{{Role: "user", Content: "return_partial"}})
		require.NoError(t, err)
		require.NotNil(t, res)

		require.Len(t, res.Messages, 1)
		require.NotNil(t, res.Temperature)
		assert.InDelta(t, 0.5, *res.Temperature, 0.01)
		assert.Nil(t, res.Audio)
		assert.Nil(t, res.MaxTokens)
	})

	t.Run("ReturnDefault", func(t *testing.T) {
		testContent := "Hello, how are you?"
		res, _, err := agent.HookScript.Create(ctx, []agentContext.Message{{Role: "user", Content: testContent}})
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Len(t, res.Messages, 1)
		assert.Equal(t, agentContext.RoleUser, res.Messages[0].Role)
		content, ok := res.Messages[0].Content.(string)
		require.True(t, ok)
		assert.Equal(t, testContent, content)
	})

	t.Run("VerifyContext", func(t *testing.T) {
		res, _, err := agent.HookScript.Create(ctx, []agentContext.Message{{Role: "user", Content: "verify_context"}})
		require.NoError(t, err)
		require.NotNil(t, res)
		require.NotEmpty(t, res.Messages)

		content, ok := res.Messages[0].Content.(string)
		require.True(t, ok)
		assert.Equal(t, "success:all_fields_validated", content, "context validation failed; details in second message")
	})

	t.Run("AdjustContext", func(t *testing.T) {
		adjustCtx := newTestContext("chat-test-adjust", "tests.hook-echo")
		res, _, err := agent.HookScript.Create(adjustCtx, []agentContext.Message{{Role: "user", Content: "adjust_context"}})
		require.NoError(t, err)
		require.NotNil(t, res)

		assert.Equal(t, "adjusted-connector", res.Connector)
		assert.Equal(t, "zh-cn", res.Locale)
		assert.Equal(t, "dark", res.Theme)
		assert.Equal(t, "/adjusted/route", res.Route)

		assert.Equal(t, "zh-cn", adjustCtx.Locale)
		assert.Equal(t, "dark", adjustCtx.Theme)
		assert.Equal(t, "/adjusted/route", adjustCtx.Route)
		assert.Equal(t, true, adjustCtx.Metadata["adjusted"])
	})

	t.Run("AdjustUses", func(t *testing.T) {
		usesCtx := newTestContext("chat-test-uses", "tests.hook-echo")
		res, _, err := agent.HookScript.Create(usesCtx, []agentContext.Message{{Role: "user", Content: "adjust_uses"}})
		require.NoError(t, err)
		require.NotNil(t, res)
		require.NotNil(t, res.Uses)
		assert.Equal(t, "mcp:vision-server", res.Uses.Vision)
		assert.Equal(t, "mcp:audio-server", res.Uses.Audio)
		assert.Equal(t, "agent", res.Uses.Search)
		assert.Equal(t, "mcp:fetch-server", res.Uses.Fetch)
	})

	t.Run("AdjustUsesForce", func(t *testing.T) {
		usesCtx := newTestContext("chat-test-uses-force", "tests.hook-echo")
		res, _, err := agent.HookScript.Create(usesCtx, []agentContext.Message{{Role: "user", Content: "adjust_uses_force"}})
		require.NoError(t, err)
		require.NotNil(t, res)
		require.NotNil(t, res.Uses)
		assert.Equal(t, "tests.vision-test", res.Uses.Vision)
		require.NotNil(t, res.ForceUses)
		assert.True(t, *res.ForceUses)
	})

	t.Run("NilScriptSafety", func(t *testing.T) {
		noHookAgent, err := assistant.Get("tests.simple-greeting")
		require.NoError(t, err)
		require.Nil(t, noHookAgent.HookScript)
	})
}
