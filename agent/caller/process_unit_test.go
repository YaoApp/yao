//go:build unit

package caller_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/caller"
	agentContext "github.com/yaoapp/yao/agent/context"
)

// ---------------------------------------------------------------------------
// NewHeadlessContext tests
// ---------------------------------------------------------------------------

func TestNewHeadlessContext_Basic(t *testing.T) {
	req := &caller.ProcessCallRequest{
		AssistantID: "tests.caller-target",
		Messages: []map[string]interface{}{
			{"role": "user", "content": "hello"},
		},
	}

	ctx, opts := caller.NewHeadlessContext(context.Background(), nil, req)
	defer ctx.Release()

	assert.Equal(t, "tests.caller-target", ctx.AssistantID)
	assert.Equal(t, agentContext.RefererProcess, ctx.Referer)
	assert.NotEmpty(t, ctx.ChatID, "ChatID should be auto-generated")

	require.NotNil(t, opts)
	require.NotNil(t, opts.Skip)
	assert.True(t, opts.Skip.Output, "Skip.Output must be forced true for headless")
	assert.True(t, opts.Skip.History, "Skip.History must be forced true for headless")
}

func TestNewHeadlessContext_WithModel(t *testing.T) {
	req := &caller.ProcessCallRequest{
		AssistantID: "tests.caller-target",
		Model:       "gpt-4o",
		Messages: []map[string]interface{}{
			{"role": "user", "content": "test"},
		},
	}

	ctx, opts := caller.NewHeadlessContext(context.Background(), nil, req)
	defer ctx.Release()

	assert.Equal(t, "gpt-4o", opts.Connector)
}

func TestNewHeadlessContext_WithChatID(t *testing.T) {
	req := &caller.ProcessCallRequest{
		AssistantID: "tests.caller-target",
		ChatID:      "custom-chat-id-123",
		Messages: []map[string]interface{}{
			{"role": "user", "content": "test"},
		},
	}

	ctx, opts := caller.NewHeadlessContext(context.Background(), nil, req)
	defer ctx.Release()
	_ = opts

	assert.Equal(t, "custom-chat-id-123", ctx.ChatID)
}

func TestNewHeadlessContext_ForceSkipOverrides(t *testing.T) {
	req := &caller.ProcessCallRequest{
		AssistantID: "tests.caller-target",
		Skip: &agentContext.Skip{
			Output:  false,
			History: false,
			Trace:   true,
		},
		Messages: []map[string]interface{}{
			{"role": "user", "content": "test"},
		},
	}

	ctx, opts := caller.NewHeadlessContext(context.Background(), nil, req)
	defer ctx.Release()

	require.NotNil(t, opts.Skip)
	assert.True(t, opts.Skip.Output, "Skip.Output must be forced true regardless of request")
	assert.True(t, opts.Skip.History, "Skip.History must be forced true regardless of request")
	assert.True(t, opts.Skip.Trace, "Skip.Trace should preserve caller's setting")
}

func TestNewHeadlessContext_WithMetadata(t *testing.T) {
	req := &caller.ProcessCallRequest{
		AssistantID: "tests.caller-target",
		Locale:      "zh-cn",
		Route:       "/custom/route",
		Metadata: map[string]interface{}{
			"source": "unit-test",
			"key":    42,
		},
		Messages: []map[string]interface{}{
			{"role": "user", "content": "test"},
		},
	}

	ctx, opts := caller.NewHeadlessContext(context.Background(), nil, req)
	defer ctx.Release()
	_ = opts

	assert.Equal(t, "zh-cn", ctx.Locale)
	assert.Equal(t, "/custom/route", ctx.Route)
	require.NotNil(t, ctx.Metadata)
	assert.Equal(t, "unit-test", ctx.Metadata["source"])
	assert.Equal(t, 42, ctx.Metadata["key"])
}

// ---------------------------------------------------------------------------
// ParseMessages tests
// ---------------------------------------------------------------------------

func TestParseMessages_Basic(t *testing.T) {
	raw := []map[string]interface{}{
		{"role": "user", "content": "hello world"},
		{"role": "assistant", "content": "hi there"},
	}

	messages := caller.ParseMessages(raw)
	require.Len(t, messages, 2)
	assert.Equal(t, agentContext.RoleUser, messages[0].Role)
	assert.Equal(t, "hello world", messages[0].Content)
	assert.Equal(t, agentContext.RoleAssistant, messages[1].Role)
	assert.Equal(t, "hi there", messages[1].Content)
}

func TestParseMessages_WithOptionalFields(t *testing.T) {
	name := "test-func"
	raw := []map[string]interface{}{
		{
			"role":         "tool",
			"content":      "result data",
			"name":         name,
			"tool_call_id": "call_abc123",
		},
		{
			"role":    "assistant",
			"content": "",
			"tool_calls": []interface{}{
				map[string]interface{}{
					"id":   "call_abc123",
					"type": "function",
					"function": map[string]interface{}{
						"name":      "get_weather",
						"arguments": `{"location":"Tokyo"}`,
					},
				},
			},
		},
	}

	messages := caller.ParseMessages(raw)
	require.Len(t, messages, 2)

	// First message: tool result
	assert.Equal(t, agentContext.MessageRole("tool"), messages[0].Role)
	require.NotNil(t, messages[0].Name)
	assert.Equal(t, "test-func", *messages[0].Name)
	require.NotNil(t, messages[0].ToolCallID)
	assert.Equal(t, "call_abc123", *messages[0].ToolCallID)

	// Second message: assistant with tool_calls
	assert.Equal(t, agentContext.RoleAssistant, messages[1].Role)
	require.Len(t, messages[1].ToolCalls, 1)
	assert.Equal(t, "call_abc123", messages[1].ToolCalls[0].ID)
	assert.Equal(t, agentContext.ToolCallType("function"), messages[1].ToolCalls[0].Type)
	assert.Equal(t, "get_weather", messages[1].ToolCalls[0].Function.Name)
	assert.Equal(t, `{"location":"Tokyo"}`, messages[1].ToolCalls[0].Function.Arguments)
}

func TestParseMessages_Empty(t *testing.T) {
	messages := caller.ParseMessages([]map[string]interface{}{})
	assert.Empty(t, messages)
	assert.NotNil(t, messages, "should return empty slice, not nil")
}

// ---------------------------------------------------------------------------
// NewResult tests
// ---------------------------------------------------------------------------

func TestNewResult_Success(t *testing.T) {
	resp := &agentContext.Response{
		Completion: &agentContext.CompletionResponse{
			Content: "hello from agent",
		},
	}

	result := caller.NewResult("agent-1", resp, nil)
	assert.Equal(t, "agent-1", result.AgentID)
	assert.Equal(t, "hello from agent", result.Content)
	assert.Empty(t, result.Error)
	assert.NotNil(t, result.Response)
}

func TestNewResult_WithError(t *testing.T) {
	err := assert.AnError
	result := caller.NewResult("agent-err", nil, err)
	assert.Equal(t, "agent-err", result.AgentID)
	assert.Equal(t, err.Error(), result.Error)
	assert.Empty(t, result.Content)
	assert.Nil(t, result.Response)
}

func TestNewResult_NilResponse(t *testing.T) {
	result := caller.NewResult("agent-nil", nil, nil)
	assert.Equal(t, "agent-nil", result.AgentID)
	assert.Empty(t, result.Content)
	assert.Empty(t, result.Error)
	assert.Nil(t, result.Response)
}

func TestNewResult_NilCompletion(t *testing.T) {
	resp := &agentContext.Response{}
	result := caller.NewResult("agent-nocomp", resp, nil)
	assert.Equal(t, "agent-nocomp", result.AgentID)
	assert.Empty(t, result.Content)
	assert.Empty(t, result.Error)
	assert.NotNil(t, result.Response)
}

// ---------------------------------------------------------------------------
// ProcessCallRequest defaults
// ---------------------------------------------------------------------------

func TestProcessCallRequest_DefaultTimeout(t *testing.T) {
	req := &caller.ProcessCallRequest{
		AssistantID: "test-agent",
	}
	assert.Equal(t, 0, req.Timeout, "Timeout zero value triggers DefaultProcessTimeout in processAgentCall")
	assert.Equal(t, 600, caller.DefaultProcessTimeout)
}

func TestNewHeadlessContext_WithTimeout(t *testing.T) {
	req := &caller.ProcessCallRequest{
		AssistantID: "test.agent",
		Messages:    []map[string]interface{}{{"role": "user", "content": "hi"}},
		Timeout:     30,
	}

	ctx, _ := caller.NewHeadlessContext(context.Background(), nil, req)
	defer ctx.Release()

	assert.Equal(t, 30, req.Timeout)
}
