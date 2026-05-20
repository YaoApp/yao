//go:build unit

package context_test

import (
	stdContext "context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/context"
)

// =============================================================================
// AgentAPIFactory Tests
// =============================================================================

func TestAgentNilFactory(t *testing.T) {
	context.AgentAPIFactory = nil

	ctx := context.New(stdContext.Background(), nil, "test-chat")
	agentAPI := ctx.Agent()
	assert.Nil(t, agentAPI)
}

func TestAgentWithFactory(t *testing.T) {
	var capturedCtx *context.Context
	context.AgentAPIFactory = func(ctx *context.Context) context.AgentAPI {
		capturedCtx = ctx
		return &unitMockAgentAPI{}
	}
	defer func() { context.AgentAPIFactory = nil }()

	ctx := context.New(stdContext.Background(), nil, "test-chat")
	agentAPI := ctx.Agent()

	require.NotNil(t, agentAPI)
	assert.Equal(t, ctx, capturedCtx)
}

func TestAgentFactoryCalledEachTime(t *testing.T) {
	callCount := 0
	context.AgentAPIFactory = func(ctx *context.Context) context.AgentAPI {
		callCount++
		return &unitMockAgentAPI{}
	}
	defer func() { context.AgentAPIFactory = nil }()

	ctx := context.New(stdContext.Background(), nil, "test-chat")

	ctx.Agent()
	ctx.Agent()
	ctx.Agent()

	assert.Equal(t, 3, callCount)
}

func TestAgentFactoryResetToNil(t *testing.T) {
	context.AgentAPIFactory = func(ctx *context.Context) context.AgentAPI {
		return &unitMockAgentAPI{}
	}

	ctx := context.New(stdContext.Background(), nil, "test-chat")
	agentAPI := ctx.Agent()
	require.NotNil(t, agentAPI)

	context.AgentAPIFactory = nil
	agentAPI = ctx.Agent()
	assert.Nil(t, agentAPI)
}

func TestAgentFactoryReturnsNil(t *testing.T) {
	context.AgentAPIFactory = func(ctx *context.Context) context.AgentAPI {
		return nil
	}
	defer func() { context.AgentAPIFactory = nil }()

	ctx := context.New(stdContext.Background(), nil, "test-chat")
	agentAPI := ctx.Agent()
	assert.Nil(t, agentAPI)
}

// =============================================================================
// Mock AgentAPI
// =============================================================================

type unitMockAgentAPI struct{}

func (m *unitMockAgentAPI) Call(agentID string, messages []interface{}, opts map[string]interface{}) interface{} {
	return map[string]interface{}{
		"agent_id": agentID,
		"content":  "mock response",
	}
}

func (m *unitMockAgentAPI) All(requests []interface{}) []interface{} {
	return []interface{}{}
}

func (m *unitMockAgentAPI) Any(requests []interface{}) []interface{} {
	return []interface{}{}
}

func (m *unitMockAgentAPI) Race(requests []interface{}) []interface{} {
	return []interface{}{}
}
