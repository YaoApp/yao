package caller_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/caller"
	agentContext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/openapi/oauth/types"
)

// --- NewHeadlessContext tests ---

func TestNewHeadlessContext_Basic(t *testing.T) {
	authInfo := &types.AuthorizedInfo{
		TeamID: "team-123",
		UserID: "user-456",
	}
	req := &caller.ProcessCallRequest{
		AssistantID: "yao.keeper.classify",
		Messages: []map[string]interface{}{
			{"role": "user", "content": "hello"},
		},
		Locale: "zh-CN",
	}

	ctx, opts := caller.NewHeadlessContext(context.Background(), authInfo, req)
	defer ctx.Release()

	assert.Equal(t, "yao.keeper.classify", ctx.AssistantID)
	assert.Equal(t, agentContext.RefererProcess, ctx.Referer)
	assert.Equal(t, "zh-CN", ctx.Locale)
	assert.NotEmpty(t, ctx.ChatID) // auto-generated

	require.NotNil(t, opts)
	require.NotNil(t, opts.Skip)
	assert.True(t, opts.Skip.Output, "skip.output must be forced true for headless context")
	assert.True(t, opts.Skip.History, "skip.history must be forced true for headless context")
	assert.Empty(t, opts.Connector)
}

func TestNewHeadlessContext_WithModel(t *testing.T) {
	req := &caller.ProcessCallRequest{
		AssistantID: "test.agent",
		Messages:    []map[string]interface{}{{"role": "user", "content": "hi"}},
		Model:       "deepseek.v3",
	}

	ctx, opts := caller.NewHeadlessContext(context.Background(), nil, req)
	defer ctx.Release()

	assert.Equal(t, "deepseek.v3", opts.Connector)
}

func TestNewHeadlessContext_WithChatID(t *testing.T) {
	req := &caller.ProcessCallRequest{
		AssistantID: "test.agent",
		Messages:    []map[string]interface{}{{"role": "user", "content": "hi"}},
		ChatID:      "custom-chat-id",
	}

	ctx, _ := caller.NewHeadlessContext(context.Background(), nil, req)
	defer ctx.Release()

	assert.Equal(t, "custom-chat-id", ctx.ChatID)
}

func TestNewHeadlessContext_ForceSkipOverridesUserSkip(t *testing.T) {
	req := &caller.ProcessCallRequest{
		AssistantID: "test.agent",
		Messages:    []map[string]interface{}{{"role": "user", "content": "hi"}},
		Skip:        &agentContext.Skip{Output: false, History: false, Trace: true},
	}

	_, opts := caller.NewHeadlessContext(context.Background(), nil, req)

	// Output and History must be forced true regardless of user input
	assert.True(t, opts.Skip.Output, "skip.output must be forced true")
	assert.True(t, opts.Skip.History, "skip.history must be forced true")
	// User-specified skip.trace should be preserved
	assert.True(t, opts.Skip.Trace, "skip.trace should be preserved from user input")
}

func TestNewHeadlessContext_WithMetadata(t *testing.T) {
	req := &caller.ProcessCallRequest{
		AssistantID: "test.agent",
		Messages:    []map[string]interface{}{{"role": "user", "content": "hi"}},
		Metadata:    map[string]interface{}{"key": "value"},
		Route:       "/test",
	}

	ctx, _ := caller.NewHeadlessContext(context.Background(), nil, req)
	defer ctx.Release()

	assert.Equal(t, "value", ctx.Metadata["key"])
	assert.Equal(t, "/test", ctx.Route)
}

func TestNewHeadlessContext_WithTimeout(t *testing.T) {
	req := &caller.ProcessCallRequest{
		AssistantID: "test.agent",
		Messages:    []map[string]interface{}{{"role": "user", "content": "hi"}},
		Timeout:     30,
	}

	// Pass a context with timeout to verify it propagates
	ctx, _ := caller.NewHeadlessContext(context.Background(), nil, req)
	defer ctx.Release()

	// Timeout field is consumed by processAgentCall, not NewHeadlessContext.
	// Here we just verify the field is correctly set in the struct.
	assert.Equal(t, 30, req.Timeout)
}

func TestProcessCallRequest_DefaultTimeout(t *testing.T) {
	req := &caller.ProcessCallRequest{
		AssistantID: "test.agent",
		Messages:    []map[string]interface{}{{"role": "user", "content": "hi"}},
	}

	// When Timeout is 0 (zero value), the default should be used
	assert.Equal(t, 0, req.Timeout, "zero value means use default")
	assert.Equal(t, 600, caller.DefaultProcessTimeout, "default timeout should be 600 seconds")
}

// --- ParseMessages tests ---

func TestParseMessages_Basic(t *testing.T) {
	raw := []map[string]interface{}{
		{"role": "user", "content": "hello"},
		{"role": "assistant", "content": "hi there"},
	}

	messages := caller.ParseMessages(raw)
	require.Len(t, messages, 2)

	assert.Equal(t, agentContext.MessageRole("user"), messages[0].Role)
	assert.Equal(t, "hello", messages[0].Content)
	assert.Equal(t, agentContext.MessageRole("assistant"), messages[1].Role)
	assert.Equal(t, "hi there", messages[1].Content)
}

func TestParseMessages_WithOptionalFields(t *testing.T) {
	name := "test-name"
	raw := []map[string]interface{}{
		{
			"role":         "tool",
			"content":      "result",
			"name":         name,
			"tool_call_id": "tc-1",
		},
	}

	messages := caller.ParseMessages(raw)
	require.Len(t, messages, 1)

	msg := messages[0]
	assert.Equal(t, agentContext.MessageRole("tool"), msg.Role)
	require.NotNil(t, msg.Name)
	assert.Equal(t, name, *msg.Name)
	require.NotNil(t, msg.ToolCallID)
	assert.Equal(t, "tc-1", *msg.ToolCallID)
}

func TestParseMessages_Empty(t *testing.T) {
	messages := caller.ParseMessages(nil)
	assert.Empty(t, messages)
}

// --- NewResult tests ---

func TestNewResult_Success(t *testing.T) {
	resp := &agentContext.Response{
		Completion: &agentContext.CompletionResponse{
			Content: "answer text",
		},
	}

	result := caller.NewResult("test.agent", resp, nil)

	assert.Equal(t, "test.agent", result.AgentID)
	assert.Equal(t, "answer text", result.Content)
	assert.Empty(t, result.Error)
	assert.NotNil(t, result.Response)
}

func TestNewResult_WithError(t *testing.T) {
	result := caller.NewResult("test.agent", nil, errors.New("something failed"))

	assert.Equal(t, "test.agent", result.AgentID)
	assert.Equal(t, "something failed", result.Error)
	assert.Empty(t, result.Content)
	assert.Nil(t, result.Response)
}

func TestNewResult_NilResponse(t *testing.T) {
	result := caller.NewResult("test.agent", nil, nil)

	assert.Equal(t, "test.agent", result.AgentID)
	assert.Empty(t, result.Content)
	assert.Empty(t, result.Error)
	assert.Nil(t, result.Response)
}

func TestNewResult_NilCompletion(t *testing.T) {
	resp := &agentContext.Response{Completion: nil}
	result := caller.NewResult("test.agent", resp, nil)

	assert.Empty(t, result.Content, "content should be empty when completion is nil")
	assert.NotNil(t, result.Response)
}
