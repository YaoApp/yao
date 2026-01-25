package context_test

import (
	stdContext "context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/context"
)

func TestContext_Agent_NilFactory(t *testing.T) {
	// Reset factory
	context.AgentAPIFactory = nil

	ctx := context.New(stdContext.Background(), nil, "test-chat")
	agentAPI := ctx.Agent()
	assert.Nil(t, agentAPI)
}

func TestContext_Agent_WithFactory(t *testing.T) {
	// Set up a mock factory
	var capturedCtx *context.Context
	context.AgentAPIFactory = func(ctx *context.Context) context.AgentAPI {
		capturedCtx = ctx
		return &mockAgentAPI{}
	}
	defer func() { context.AgentAPIFactory = nil }()

	ctx := context.New(stdContext.Background(), nil, "test-chat")
	agentAPI := ctx.Agent()

	require.NotNil(t, agentAPI)
	assert.Equal(t, ctx, capturedCtx)
}

// mockAgentAPI implements context.AgentAPI for testing
type mockAgentAPI struct{}

func (m *mockAgentAPI) Call(agentID string, messages []interface{}, opts map[string]interface{}) interface{} {
	return map[string]interface{}{
		"agent_id": agentID,
		"content":  "mock response",
	}
}

func (m *mockAgentAPI) All(requests []interface{}) []interface{} {
	return []interface{}{}
}

func (m *mockAgentAPI) Any(requests []interface{}) []interface{} {
	return []interface{}{}
}

func (m *mockAgentAPI) Race(requests []interface{}) []interface{} {
	return []interface{}{}
}
