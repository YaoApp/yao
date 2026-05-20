//go:build e2e

package hook_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/assistant"
	agentContext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

func TestRealWorldNextStandard(t *testing.T) {
	testprepare.PrepareE2E(t)

	agent, err := assistant.Get("tests.realworld-next")
	require.NoError(t, err, "failed to get tests.realworld-next assistant")
	require.NotNil(t, agent.HookScript, "tests.realworld-next has no hook script")

	ctx := newTestContext("test-next-standard", "tests.realworld-next")

	messages := []agentContext.Message{
		{Role: agentContext.RoleUser, Content: "scenario: standard"},
		{Role: agentContext.RoleAssistant, Content: "I'll process your request using standard response."},
	}

	payload := &agentContext.NextHookPayload{
		Messages: messages,
		Completion: &agentContext.CompletionResponse{
			Content: "Processing complete. Standard response will be used.",
		},
	}

	response, _, err := agent.HookScript.Next(ctx, payload)
	require.NoError(t, err)
	assert.Nil(t, response, "Standard scenario should return nil")
}

func TestRealWorldNextCustomData(t *testing.T) {
	testprepare.PrepareE2E(t)

	agent, err := assistant.Get("tests.realworld-next")
	require.NoError(t, err)
	require.NotNil(t, agent.HookScript)

	ctx := newTestContext("test-next-custom", "tests.realworld-next")

	payload := &agentContext.NextHookPayload{
		Messages: []agentContext.Message{
			{Role: agentContext.RoleUser, Content: "scenario: custom_data"},
			{Role: agentContext.RoleAssistant, Content: "Here's some information for you."},
		},
		Completion: &agentContext.CompletionResponse{
			Content: "This is the LLM completion that will be summarized.",
		},
	}

	response, _, err := agent.HookScript.Next(ctx, payload)
	require.NoError(t, err)
	require.NotNil(t, response, "Custom data scenario should return response")
	require.NotNil(t, response.Data, "Response should have Data")

	dataMap, ok := response.Data.(map[string]interface{})
	require.True(t, ok, "Data should be a map")
	assert.Equal(t, "custom_response", dataMap["type"])
	assert.Contains(t, dataMap, "timestamp")
}

func TestRealWorldNextDelegate(t *testing.T) {
	testprepare.PrepareE2E(t)

	agent, err := assistant.Get("tests.realworld-next")
	require.NoError(t, err)
	require.NotNil(t, agent.HookScript)

	ctx := newTestContext("test-next-delegate", "tests.realworld-next")

	payload := &agentContext.NextHookPayload{
		Messages: []agentContext.Message{
			{Role: agentContext.RoleUser, Content: "scenario: delegate"},
		},
		Completion: &agentContext.CompletionResponse{
			Content: "I should delegate this request to another agent.",
		},
	}

	response, _, err := agent.HookScript.Next(ctx, payload)
	require.NoError(t, err)
	require.NotNil(t, response, "Delegate scenario should return response")
	require.NotNil(t, response.Delegate, "Response should have Delegate")
	assert.Equal(t, "tests.create", response.Delegate.AgentID)
	assert.NotEmpty(t, response.Delegate.Messages)
}

func TestRealWorldNextProcessTools(t *testing.T) {
	testprepare.PrepareE2E(t)

	agent, err := assistant.Get("tests.realworld-next")
	require.NoError(t, err)
	require.NotNil(t, agent.HookScript)

	ctx := newTestContext("test-next-tools", "tests.realworld-next")

	tools := []agentContext.ToolCallResponse{
		{
			ToolCallID: "call_1",
			Server:     "test-server",
			Tool:       "test-tool-1",
			Result:     map[string]interface{}{"status": "success"},
			Error:      "",
		},
		{
			ToolCallID: "call_2",
			Server:     "test-server",
			Tool:       "test-tool-2",
			Result:     nil,
			Error:      "Tool execution failed",
		},
	}

	payload := &agentContext.NextHookPayload{
		Messages: []agentContext.Message{
			{Role: agentContext.RoleUser, Content: "scenario: process_tools"},
		},
		Completion: &agentContext.CompletionResponse{
			Content: "Tool calls have been executed.",
		},
		Tools: tools,
	}

	response, _, err := agent.HookScript.Next(ctx, payload)
	require.NoError(t, err)
	require.NotNil(t, response, "Process tools scenario should return response")
	require.NotNil(t, response.Data, "Response should have Data")

	dataMap, ok := response.Data.(map[string]interface{})
	require.True(t, ok, "Data should be a map")
	assert.Equal(t, "Tool execution summary", dataMap["message"])

	summary, ok := dataMap["summary"].(map[string]interface{})
	require.True(t, ok, "Should have summary")
	assert.Equal(t, float64(2), summary["total"])
	assert.Equal(t, float64(1), summary["successful"])
	assert.Equal(t, float64(1), summary["failed"])
}

func TestRealWorldNextErrorRecovery(t *testing.T) {
	testprepare.PrepareE2E(t)

	agent, err := assistant.Get("tests.realworld-next")
	require.NoError(t, err)
	require.NotNil(t, agent.HookScript)

	ctx := newTestContext("test-next-error", "tests.realworld-next")

	payload := &agentContext.NextHookPayload{
		Messages: []agentContext.Message{
			{Role: agentContext.RoleUser, Content: "scenario: error_recovery"},
		},
		Completion: &agentContext.CompletionResponse{
			Content: "An error occurred during processing.",
		},
		Error: "System error: Database connection timeout",
	}

	response, _, err := agent.HookScript.Next(ctx, payload)
	require.NoError(t, err)
	require.NotNil(t, response, "Error recovery scenario should return response")
	require.NotNil(t, response.Data, "Response should have Data")

	dataMap, ok := response.Data.(map[string]interface{})
	require.True(t, ok, "Data should be a map")
	assert.Equal(t, "Error was handled by Next Hook", dataMap["message"])
	assert.Contains(t, dataMap, "error")
	assert.Contains(t, dataMap, "recovery_action")
}

func TestRealWorldNextConditional(t *testing.T) {
	testprepare.PrepareE2E(t)

	agent, err := assistant.Get("tests.realworld-next")
	require.NoError(t, err)
	require.NotNil(t, agent.HookScript)

	ctx := newTestContext("test-next-conditional", "tests.realworld-next")

	t.Run("ConditionalSuccess", func(t *testing.T) {
		payload := &agentContext.NextHookPayload{
			Messages: []agentContext.Message{
				{Role: agentContext.RoleUser, Content: "scenario: conditional"},
			},
			Completion: &agentContext.CompletionResponse{
				Content: "The operation completed successfully. All tasks are done.",
			},
		}

		response, _, err := agent.HookScript.Next(ctx, payload)
		require.NoError(t, err)
		require.NotNil(t, response, "Conditional scenario should return response")
		require.NotNil(t, response.Data, "Response should have Data")

		dataMap, ok := response.Data.(map[string]interface{})
		require.True(t, ok, "Data should be a map")
		assert.Equal(t, "Conditional analysis complete", dataMap["message"])
		assert.Contains(t, dataMap, "action")
		assert.Contains(t, dataMap, "conditions")
	})

	t.Run("ConditionalDelegate", func(t *testing.T) {
		payload := &agentContext.NextHookPayload{
			Messages: []agentContext.Message{
				{Role: agentContext.RoleUser, Content: "scenario: conditional"},
			},
			Completion: &agentContext.CompletionResponse{
				Content: "I should delegate this request to another service for better handling.",
			},
		}

		response, _, err := agent.HookScript.Next(ctx, payload)
		require.NoError(t, err)
		require.NotNil(t, response, "Conditional delegate should return response")
		require.NotNil(t, response.Delegate, "Should delegate based on condition")
		assert.Equal(t, "tests.create", response.Delegate.AgentID)
	})
}

func TestRealWorldNextDefault(t *testing.T) {
	testprepare.PrepareE2E(t)

	agent, err := assistant.Get("tests.realworld-next")
	require.NoError(t, err)
	require.NotNil(t, agent.HookScript)

	ctx := newTestContext("test-next-default", "tests.realworld-next")

	payload := &agentContext.NextHookPayload{
		Messages: []agentContext.Message{
			{Role: agentContext.RoleUser, Content: "Just a normal request"},
		},
		Completion: &agentContext.CompletionResponse{
			Content: "Here's the response to your request.",
		},
	}

	response, _, err := agent.HookScript.Next(ctx, payload)
	require.NoError(t, err)
	assert.Nil(t, response, "Default scenario should return nil for standard response")
}
