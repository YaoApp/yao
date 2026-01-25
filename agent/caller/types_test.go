package caller_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/caller"
	"github.com/yaoapp/yao/agent/context"
)

func TestCallOptions_ToContextOptions_Nil(t *testing.T) {
	var opts *caller.CallOptions
	ctxOpts := opts.ToContextOptions()
	assert.Nil(t, ctxOpts)
}

func TestCallOptions_ToContextOptions_Empty(t *testing.T) {
	opts := &caller.CallOptions{}
	ctxOpts := opts.ToContextOptions()
	require.NotNil(t, ctxOpts)
	assert.Empty(t, ctxOpts.Connector)
	assert.Empty(t, ctxOpts.Mode)
	assert.Nil(t, ctxOpts.Metadata)
	assert.Nil(t, ctxOpts.Skip)
}

func TestCallOptions_ToContextOptions_Full(t *testing.T) {
	opts := &caller.CallOptions{
		Connector: "gpt4",
		Mode:      "chat",
		Metadata: map[string]interface{}{
			"key": "value",
		},
		Skip: &context.Skip{
			History: true,
			Trace:   true,
			Output:  false,
		},
	}

	ctxOpts := opts.ToContextOptions()
	require.NotNil(t, ctxOpts)
	assert.Equal(t, "gpt4", ctxOpts.Connector)
	assert.Equal(t, "chat", ctxOpts.Mode)
	assert.Equal(t, "value", ctxOpts.Metadata["key"])
	require.NotNil(t, ctxOpts.Skip)
	assert.True(t, ctxOpts.Skip.History)
	assert.True(t, ctxOpts.Skip.Trace)
	assert.False(t, ctxOpts.Skip.Output)
}

func TestRequest_Basic(t *testing.T) {
	req := &caller.Request{
		AgentID: "test-agent",
		Messages: []context.Message{
			{Role: "user", Content: "Hello"},
		},
	}

	assert.Equal(t, "test-agent", req.AgentID)
	assert.Len(t, req.Messages, 1)
	assert.Equal(t, context.MessageRole("user"), req.Messages[0].Role)
}

func TestResult_Basic(t *testing.T) {
	result := &caller.Result{
		AgentID: "test-agent",
		Content: "Hello response",
	}

	assert.Equal(t, "test-agent", result.AgentID)
	assert.Equal(t, "Hello response", result.Content)
	assert.Empty(t, result.Error)
}

func TestResult_WithError(t *testing.T) {
	result := &caller.Result{
		AgentID: "test-agent",
		Error:   "something went wrong",
	}

	assert.Equal(t, "test-agent", result.AgentID)
	assert.Equal(t, "something went wrong", result.Error)
	assert.Empty(t, result.Content)
}
