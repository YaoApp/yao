//go:build unit

package assistant_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/assistant"
	agentContext "github.com/yaoapp/yao/agent/context"
	storetypes "github.com/yaoapp/yao/agent/store/types"
)

func TestBuildToolLoopMessages(t *testing.T) {
	previousMsgs := []agentContext.Message{
		{Role: agentContext.RoleSystem, Content: "You are a helper."},
		{Role: agentContext.RoleUser, Content: "Use tool X to fetch data."},
	}

	completion := &agentContext.CompletionResponse{
		Content: "I'll use tool X",
		ToolCalls: []agentContext.ToolCall{
			{
				ID:   "tc1",
				Type: agentContext.ToolTypeFunction,
				Function: agentContext.Function{
					Name:      "echo__ping",
					Arguments: `{"message":"hello"}`,
				},
			},
		},
	}

	toolResponses := []agentContext.ToolCallResponse{
		{ToolCallID: "tc1", Server: "echo", Tool: "ping", Result: "pong"},
	}

	result := assistant.ExportBuildToolLoopMessages(previousMsgs, completion, toolResponses)

	expectedLen := len(previousMsgs) + 1 + len(toolResponses)
	require.Len(t, result, expectedLen, "should have previous + 1 assistant + tool response messages")

	assert.Equal(t, agentContext.RoleSystem, result[0].Role)
	assert.Equal(t, agentContext.RoleUser, result[1].Role)
	assert.Equal(t, agentContext.RoleAssistant, result[2].Role)
	assert.Equal(t, "I'll use tool X", result[2].Content)
	assert.Len(t, result[2].ToolCalls, 1)
	assert.Equal(t, agentContext.RoleTool, result[3].Role)
	assert.NotNil(t, result[3].ToolCallID, "tool message should have ToolCallID")
	assert.Equal(t, "tc1", *result[3].ToolCallID)
}

func TestMessageText_String(t *testing.T) {
	msg := agentContext.Message{Role: agentContext.RoleUser, Content: "hello world"}
	result := assistant.ExportMessageText(msg)
	assert.Equal(t, "hello world", result)
}

func TestMessageText_Parts(t *testing.T) {
	msg := agentContext.Message{
		Role: agentContext.RoleUser,
		Content: []interface{}{
			map[string]interface{}{"type": "text", "text": "part one"},
			map[string]interface{}{"type": "image_url", "url": "http://img.png"},
			map[string]interface{}{"type": "text", "text": "part two"},
		},
	}
	result := assistant.ExportMessageText(msg)
	assert.Contains(t, result, "part one")
	assert.Contains(t, result, "part two")
	assert.NotContains(t, result, "http://img.png")
}

func TestMessageText_Nil(t *testing.T) {
	msg := agentContext.Message{Role: agentContext.RoleUser, Content: nil}
	result := assistant.ExportMessageText(msg)
	assert.Equal(t, "", result)
}

func TestBuildLoopFallbackMarkdown(t *testing.T) {
	fullMessages := []agentContext.Message{
		{Role: agentContext.RoleSystem, Content: "System prompt here"},
		{Role: agentContext.RoleUser, Content: "What is the weather?"},
		{Role: agentContext.RoleAssistant, Content: "Let me check."},
	}

	toolResults := []agentContext.ToolCallResponse{
		{ToolCallID: "tc1", Server: "weather", Tool: "get_forecast", Result: map[string]interface{}{"temp": 25}},
		{ToolCallID: "tc2", Server: "", Tool: "fallback_tool", Error: "service unavailable"},
	}

	md := assistant.ExportBuildLoopFallbackMarkdown(fullMessages, toolResults)

	assert.Contains(t, md, "## Assistant Context")
	assert.Contains(t, md, "System prompt here")
	assert.Contains(t, md, "## Conversation")
	assert.Contains(t, md, "**User**: What is the weather?")
	assert.Contains(t, md, "**Assistant**: Let me check.")
	assert.Contains(t, md, "## Tool Results")
	assert.Contains(t, md, "### weather.get_forecast")
	assert.Contains(t, md, "### fallback_tool")
	assert.Contains(t, md, "service unavailable")
}

func TestIsToolLoopDisabled(t *testing.T) {
	tests := []struct {
		name     string
		mcp      *storetypes.MCPServers
		expected bool
	}{
		{
			name:     "nil MCP — default enabled",
			mcp:      nil,
			expected: false,
		},
		{
			name:     "MCP with no options — default enabled",
			mcp:      &storetypes.MCPServers{},
			expected: false,
		},
		{
			name: "tool_loop=true — enabled",
			mcp: &storetypes.MCPServers{
				Options: map[string]interface{}{"tool_loop": true},
			},
			expected: false,
		},
		{
			name: "tool_loop=false — disabled",
			mcp: &storetypes.MCPServers{
				Options: map[string]interface{}{"tool_loop": false},
			},
			expected: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ast := &assistant.Assistant{}
			ast.MCP = tc.mcp
			result := assistant.ExportIsToolLoopDisabled(ast)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestGetMaxToolLoopTurns(t *testing.T) {
	tests := []struct {
		name     string
		mcp      *storetypes.MCPServers
		expected int
	}{
		{
			name:     "nil MCP — default 5",
			mcp:      nil,
			expected: 5,
		},
		{
			name: "custom max_turn=3",
			mcp: &storetypes.MCPServers{
				Options: map[string]interface{}{"max_turn": 3},
			},
			expected: 3,
		},
		{
			name: "float64 max_turn=7.0",
			mcp: &storetypes.MCPServers{
				Options: map[string]interface{}{"max_turn": float64(7)},
			},
			expected: 7,
		},
		{
			name:     "nil options — default 5",
			mcp:      &storetypes.MCPServers{Options: nil},
			expected: 5,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ast := &assistant.Assistant{}
			ast.MCP = tc.mcp
			result := assistant.ExportGetMaxToolLoopTurns(ast)
			assert.Equal(t, tc.expected, result)
		})
	}
}
