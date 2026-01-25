package caller_test

import (
	stdContext "context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/caller"
	"github.com/yaoapp/yao/agent/context"
)

func TestNewOrchestrator(t *testing.T) {
	ctx := context.New(stdContext.Background(), nil, "test-chat")
	orch := caller.NewOrchestrator(ctx)
	require.NotNil(t, orch)
}

func TestOrchestrator_All_Empty(t *testing.T) {
	ctx := context.New(stdContext.Background(), nil, "test-chat")
	orch := caller.NewOrchestrator(ctx)

	results := orch.All([]*caller.Request{})
	assert.Len(t, results, 0)
}

func TestOrchestrator_Any_Empty(t *testing.T) {
	ctx := context.New(stdContext.Background(), nil, "test-chat")
	orch := caller.NewOrchestrator(ctx)

	results := orch.Any([]*caller.Request{})
	assert.Len(t, results, 0)
}

func TestOrchestrator_Race_Empty(t *testing.T) {
	ctx := context.New(stdContext.Background(), nil, "test-chat")
	orch := caller.NewOrchestrator(ctx)

	results := orch.Race([]*caller.Request{})
	assert.Len(t, results, 0)
}

func TestOrchestrator_All_NoGetter(t *testing.T) {
	// Reset AgentGetterFunc
	originalGetter := caller.AgentGetterFunc
	caller.AgentGetterFunc = nil
	defer func() { caller.AgentGetterFunc = originalGetter }()

	ctx := context.New(stdContext.Background(), nil, "test-chat")
	orch := caller.NewOrchestrator(ctx)

	reqs := []*caller.Request{
		{
			AgentID:  "agent1",
			Messages: []context.Message{{Role: "user", Content: "Hello"}},
		},
		{
			AgentID:  "agent2",
			Messages: []context.Message{{Role: "user", Content: "World"}},
		},
	}

	results := orch.All(reqs)
	require.Len(t, results, 2)

	// All should have errors because no getter
	for i, r := range results {
		require.NotNil(t, r, "result %d should not be nil", i)
		assert.Contains(t, r.Error, "agent getter not initialized")
	}
}

func TestOrchestrator_Any_NoGetter(t *testing.T) {
	// Reset AgentGetterFunc
	originalGetter := caller.AgentGetterFunc
	caller.AgentGetterFunc = nil
	defer func() { caller.AgentGetterFunc = originalGetter }()

	ctx := context.New(stdContext.Background(), nil, "test-chat")
	orch := caller.NewOrchestrator(ctx)

	reqs := []*caller.Request{
		{
			AgentID:  "agent1",
			Messages: []context.Message{{Role: "user", Content: "Hello"}},
		},
		{
			AgentID:  "agent2",
			Messages: []context.Message{{Role: "user", Content: "World"}},
		},
	}

	results := orch.Any(reqs)
	require.Len(t, results, 2)

	// At least one result should exist
	hasResult := false
	for _, r := range results {
		if r != nil {
			hasResult = true
			assert.Contains(t, r.Error, "agent getter not initialized")
		}
	}
	assert.True(t, hasResult)
}

func TestOrchestrator_Race_NoGetter(t *testing.T) {
	// Reset AgentGetterFunc
	originalGetter := caller.AgentGetterFunc
	caller.AgentGetterFunc = nil
	defer func() { caller.AgentGetterFunc = originalGetter }()

	ctx := context.New(stdContext.Background(), nil, "test-chat")
	orch := caller.NewOrchestrator(ctx)

	reqs := []*caller.Request{
		{
			AgentID:  "agent1",
			Messages: []context.Message{{Role: "user", Content: "Hello"}},
		},
		{
			AgentID:  "agent2",
			Messages: []context.Message{{Role: "user", Content: "World"}},
		},
	}

	results := orch.Race(reqs)
	require.Len(t, results, 2)

	// At least one result should exist (first to complete)
	hasResult := false
	for _, r := range results {
		if r != nil {
			hasResult = true
		}
	}
	assert.True(t, hasResult)
}

func TestOrchestrator_All_NilRequest(t *testing.T) {
	ctx := context.New(stdContext.Background(), nil, "test-chat")
	orch := caller.NewOrchestrator(ctx)

	reqs := []*caller.Request{
		nil,
		{
			AgentID:  "agent1",
			Messages: []context.Message{{Role: "user", Content: "Hello"}},
		},
	}

	// Reset AgentGetterFunc
	originalGetter := caller.AgentGetterFunc
	caller.AgentGetterFunc = nil
	defer func() { caller.AgentGetterFunc = originalGetter }()

	results := orch.All(reqs)
	require.Len(t, results, 2)

	// First result should have "nil request" error
	assert.Contains(t, results[0].Error, "nil request")
}
