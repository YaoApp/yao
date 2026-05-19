//go:build integration

package hook_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/assistant"
	agentContext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/output/message"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

func TestNext(t *testing.T) {
	testprepare.PrepareSandbox(t)

	agent, err := assistant.Get("tests.hook-next")
	require.NoError(t, err, "failed to get tests.hook-next assistant")
	require.NotNil(t, agent.HookScript, "tests.hook-next has no hook script")

	ctx := newTestContext("chat-test-next-hook", "tests.hook-next")

	t.Run("ReturnNull", func(t *testing.T) {
		payload := &agentContext.NextHookPayload{
			Messages:   []agentContext.Message{{Role: agentContext.RoleUser, Content: "return_null"}},
			Completion: &agentContext.CompletionResponse{Content: "Test completion"},
		}
		res, _, err := agent.HookScript.Next(ctx, payload)
		require.NoError(t, err)
		assert.Nil(t, res)
	})

	t.Run("ReturnUndefined", func(t *testing.T) {
		payload := &agentContext.NextHookPayload{
			Messages:   []agentContext.Message{{Role: agentContext.RoleUser, Content: "return_undefined"}},
			Completion: &agentContext.CompletionResponse{Content: "Test completion"},
		}
		res, _, err := agent.HookScript.Next(ctx, payload)
		require.NoError(t, err)
		assert.Nil(t, res)
	})

	t.Run("ReturnEmpty", func(t *testing.T) {
		payload := &agentContext.NextHookPayload{
			Messages:   []agentContext.Message{{Role: agentContext.RoleUser, Content: "return_empty"}},
			Completion: &agentContext.CompletionResponse{Content: "Test completion"},
		}
		res, _, err := agent.HookScript.Next(ctx, payload)
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Nil(t, res.Delegate)
		assert.Nil(t, res.Data)
	})

	t.Run("ReturnCustomData", func(t *testing.T) {
		payload := &agentContext.NextHookPayload{
			Messages:   []agentContext.Message{{Role: agentContext.RoleUser, Content: "return_custom_data"}},
			Completion: &agentContext.CompletionResponse{Content: "Test completion"},
		}
		res, _, err := agent.HookScript.Next(ctx, payload)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.NotNil(t, res.Data)

		dataMap, ok := res.Data.(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "Custom response from Next Hook", dataMap["message"])
		assert.Equal(t, true, dataMap["test"])
		assert.Contains(t, dataMap, "timestamp")
		assert.Nil(t, res.Delegate)
	})

	t.Run("ReturnDataWithMetadata", func(t *testing.T) {
		payload := &agentContext.NextHookPayload{
			Messages:   []agentContext.Message{{Role: agentContext.RoleUser, Content: "return_data_with_metadata"}},
			Completion: &agentContext.CompletionResponse{Content: "Test completion"},
		}
		res, _, err := agent.HookScript.Next(ctx, payload)
		require.NoError(t, err)
		require.NotNil(t, res)

		dataMap, ok := res.Data.(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "success", dataMap["result"])

		require.NotNil(t, res.Metadata)
		assert.Equal(t, "next", res.Metadata["hook"])
		assert.Equal(t, true, res.Metadata["processed"])
	})

	t.Run("ReturnDelegate", func(t *testing.T) {
		payload := &agentContext.NextHookPayload{
			Messages:   []agentContext.Message{{Role: agentContext.RoleUser, Content: "return_delegate"}},
			Completion: &agentContext.CompletionResponse{Content: "Test completion"},
		}
		res, _, err := agent.HookScript.Next(ctx, payload)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.NotNil(t, res.Delegate)
		assert.Equal(t, "tests.hook-echo", res.Delegate.AgentID)
		require.Len(t, res.Delegate.Messages, 1)
		content, ok := res.Delegate.Messages[0].Content.(string)
		require.True(t, ok)
		assert.Equal(t, "Hello from delegated agent", content)
	})

	t.Run("VerifyPayload", func(t *testing.T) {
		payload := &agentContext.NextHookPayload{
			Messages: []agentContext.Message{
				{Role: agentContext.RoleSystem, Content: "System message"},
				{Role: agentContext.RoleUser, Content: "verify_payload"},
			},
			Completion: &agentContext.CompletionResponse{
				Content: "Test completion content",
				Usage:   &message.UsageInfo{PromptTokens: 10, CompletionTokens: 20, TotalTokens: 30},
			},
			Tools: []agentContext.ToolCallResponse{
				{ToolCallID: "call_123", Server: "test-server", Tool: "test-tool", Result: map[string]interface{}{"success": true}},
			},
		}
		res, _, err := agent.HookScript.Next(ctx, payload)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.NotNil(t, res.Data)

		dataMap, ok := res.Data.(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "success", dataMap["validation"])
	})

	t.Run("VerifyTools", func(t *testing.T) {
		payload := &agentContext.NextHookPayload{
			Messages:   []agentContext.Message{{Role: agentContext.RoleUser, Content: "verify_tools"}},
			Completion: &agentContext.CompletionResponse{Content: "Test"},
			Tools: []agentContext.ToolCallResponse{
				{ToolCallID: "call_1", Server: "server1", Tool: "tool1", Result: map[string]interface{}{"value": 42}},
				{ToolCallID: "call_2", Server: "server2", Tool: "tool2", Error: "Tool execution failed"},
			},
		}
		res, _, err := agent.HookScript.Next(ctx, payload)
		require.NoError(t, err)
		require.NotNil(t, res)

		dataMap, ok := res.Data.(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, float64(2), dataMap["total_tools"])
		assert.Equal(t, float64(1), dataMap["successful"])
		assert.Equal(t, float64(1), dataMap["failed"])
	})

	t.Run("ConditionalDelegateNoMatch", func(t *testing.T) {
		payload := &agentContext.NextHookPayload{
			Messages:   []agentContext.Message{{Role: agentContext.RoleUser, Content: "conditional_delegate"}},
			Completion: &agentContext.CompletionResponse{Content: "normal response"},
		}
		res, _, err := agent.HookScript.Next(ctx, payload)
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Nil(t, res.Delegate)

		dataMap, ok := res.Data.(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "No delegation needed", dataMap["message"])
	})

	t.Run("ConditionalDelegateMatch", func(t *testing.T) {
		payload := &agentContext.NextHookPayload{
			Messages:   []agentContext.Message{{Role: agentContext.RoleUser, Content: "conditional_delegate"}},
			Completion: &agentContext.CompletionResponse{Content: "Please delegate this to another agent"},
		}
		res, _, err := agent.HookScript.Next(ctx, payload)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.NotNil(t, res.Delegate)
		assert.Equal(t, "tests.hook-echo", res.Delegate.AgentID)
	})

	t.Run("HandleError", func(t *testing.T) {
		payload := &agentContext.NextHookPayload{
			Messages:   []agentContext.Message{{Role: agentContext.RoleUser, Content: "handle_error"}},
			Completion: &agentContext.CompletionResponse{Content: "Test"},
			Error:      "Tool execution failed: timeout",
		}
		res, _, err := agent.HookScript.Next(ctx, payload)
		require.NoError(t, err)
		require.NotNil(t, res)

		dataMap, ok := res.Data.(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "Tool execution failed: timeout", dataMap["error"])
		assert.Equal(t, true, dataMap["recovered"])
	})

	t.Run("DefaultScenario", func(t *testing.T) {
		payload := &agentContext.NextHookPayload{
			Messages:   []agentContext.Message{{Role: agentContext.RoleUser, Content: "unknown_scenario_xyz"}},
			Completion: &agentContext.CompletionResponse{Content: "Some completion"},
		}
		res, _, err := agent.HookScript.Next(ctx, payload)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.NotNil(t, res.Data)

		dataMap, ok := res.Data.(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "Next Hook executed successfully", dataMap["message"])
	})
}

func TestNextNilScript(t *testing.T) {
	testprepare.PrepareSandbox(t)
	noHookAgent, err := assistant.Get("tests.simple-greeting")
	require.NoError(t, err)
	require.Nil(t, noHookAgent.HookScript)
}
