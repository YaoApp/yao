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

func TestNewOrchestrator(t *testing.T) {
	ctx := agentContext.New(context.Background(), nil, "test-orch")
	defer ctx.Release()

	orch := caller.NewOrchestrator(ctx)
	assert.NotNil(t, orch)
}

func TestOrchestrator_All_Empty(t *testing.T) {
	ctx := agentContext.New(context.Background(), nil, "test-all-empty")
	defer ctx.Release()

	orch := caller.NewOrchestrator(ctx)
	results := orch.All([]*caller.Request{})
	assert.NotNil(t, results)
	assert.Empty(t, results)
}

func TestOrchestrator_Any_Empty(t *testing.T) {
	ctx := agentContext.New(context.Background(), nil, "test-any-empty")
	defer ctx.Release()

	orch := caller.NewOrchestrator(ctx)
	results := orch.Any([]*caller.Request{})
	assert.NotNil(t, results)
	assert.Empty(t, results)
}

func TestOrchestrator_Race_Empty(t *testing.T) {
	ctx := agentContext.New(context.Background(), nil, "test-race-empty")
	defer ctx.Release()

	orch := caller.NewOrchestrator(ctx)
	results := orch.Race([]*caller.Request{})
	assert.NotNil(t, results)
	assert.Empty(t, results)
}

func TestOrchestrator_All_NoGetter(t *testing.T) {
	original := caller.AgentGetterFunc
	defer func() { caller.AgentGetterFunc = original }()
	caller.AgentGetterFunc = nil

	ctx := agentContext.New(context.Background(), nil, "test-all-nogetter")
	defer ctx.Release()

	orch := caller.NewOrchestrator(ctx)
	results := orch.All([]*caller.Request{
		{
			AgentID: "nonexistent-agent",
			Messages: []agentContext.Message{
				{Role: agentContext.RoleUser, Content: "hello"},
			},
		},
	})

	require.Len(t, results, 1)
	assert.Contains(t, results[0].Error, "agent getter not initialized")
}

func TestOrchestrator_Any_NoGetter(t *testing.T) {
	original := caller.AgentGetterFunc
	defer func() { caller.AgentGetterFunc = original }()
	caller.AgentGetterFunc = nil

	ctx := agentContext.New(context.Background(), nil, "test-any-nogetter")
	defer ctx.Release()

	orch := caller.NewOrchestrator(ctx)
	results := orch.Any([]*caller.Request{
		{
			AgentID: "nonexistent-agent",
			Messages: []agentContext.Message{
				{Role: agentContext.RoleUser, Content: "hello"},
			},
		},
	})

	require.Len(t, results, 1)
	assert.Contains(t, results[0].Error, "agent getter not initialized")
}

func TestOrchestrator_Race_NoGetter(t *testing.T) {
	original := caller.AgentGetterFunc
	defer func() { caller.AgentGetterFunc = original }()
	caller.AgentGetterFunc = nil

	ctx := agentContext.New(context.Background(), nil, "test-race-nogetter")
	defer ctx.Release()

	orch := caller.NewOrchestrator(ctx)
	results := orch.Race([]*caller.Request{
		{
			AgentID: "nonexistent-agent",
			Messages: []agentContext.Message{
				{Role: agentContext.RoleUser, Content: "hello"},
			},
		},
	})

	require.Len(t, results, 1)
	assert.Contains(t, results[0].Error, "agent getter not initialized")
}
