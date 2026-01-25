package caller_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/assistant"
	"github.com/yaoapp/yao/agent/caller"
	agentContext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/testutils"
	"github.com/yaoapp/yao/openapi/oauth/types"
)

func TestIntegration_Call_RealAgent(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Load the simple-greeting agent
	ast, err := assistant.Get("tests.simple-greeting")
	require.NoError(t, err)
	require.NotNil(t, ast)

	// Create authorized info for the context
	authorized := &types.AuthorizedInfo{
		Subject:  "test-user",
		UserID:   "test-123",
		TenantID: "test-tenant",
	}

	// Create a context with authorization
	ctx := agentContext.New(context.Background(), authorized, "test-chat-integration")
	ctx.AssistantID = "tests.agent-caller"

	// Create JSAPI
	api := caller.NewJSAPI(ctx)

	// Call the simple-greeting agent
	messages := []interface{}{
		map[string]interface{}{
			"role":    "user",
			"content": "Hello!",
		},
	}

	opts := map[string]interface{}{
		"skip": map[string]interface{}{
			"history": true,
		},
	}

	result := api.Call("tests.simple-greeting", messages, opts)
	require.NotNil(t, result)

	r, ok := result.(*caller.Result)
	require.True(t, ok)
	assert.Equal(t, "tests.simple-greeting", r.AgentID)

	// Should either have content or error
	if r.Error != "" {
		t.Logf("Agent call error: %s", r.Error)
	} else {
		t.Logf("Agent response content: %s", r.Content)
		assert.NotEmpty(t, r.Content)
	}
}

func TestIntegration_All_RealAgents(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Create authorized info for the context
	authorized := &types.AuthorizedInfo{
		Subject:  "test-user",
		UserID:   "test-123",
		TenantID: "test-tenant",
	}

	// Create a context with authorization
	ctx := agentContext.New(context.Background(), authorized, "test-chat-all")
	ctx.AssistantID = "tests.agent-caller"

	// Create JSAPI
	api := caller.NewJSAPI(ctx)

	// Call multiple agents in parallel
	requests := []interface{}{
		map[string]interface{}{
			"agent": "tests.simple-greeting",
			"messages": []interface{}{
				map[string]interface{}{
					"role":    "user",
					"content": "Hello from test 1!",
				},
			},
			"options": map[string]interface{}{
				"skip": map[string]interface{}{
					"history": true,
				},
			},
		},
		map[string]interface{}{
			"agent": "tests.simple-greeting",
			"messages": []interface{}{
				map[string]interface{}{
					"role":    "user",
					"content": "Hello from test 2!",
				},
			},
			"options": map[string]interface{}{
				"skip": map[string]interface{}{
					"history": true,
				},
			},
		},
	}

	results := api.All(requests)
	require.Len(t, results, 2)

	for i, result := range results {
		r, ok := result.(*caller.Result)
		require.True(t, ok, "result %d should be *caller.Result", i)
		assert.Equal(t, "tests.simple-greeting", r.AgentID)
		t.Logf("Result[%d]: content=%s, error=%s", i, r.Content, r.Error)
	}
}

func TestIntegration_Any_RealAgents(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Create authorized info for the context
	authorized := &types.AuthorizedInfo{
		Subject:  "test-user",
		UserID:   "test-123",
		TenantID: "test-tenant",
	}

	// Create a context with authorization
	ctx := agentContext.New(context.Background(), authorized, "test-chat-any")
	ctx.AssistantID = "tests.agent-caller"

	// Create JSAPI
	api := caller.NewJSAPI(ctx)

	// Call multiple agents - return when any succeeds
	requests := []interface{}{
		map[string]interface{}{
			"agent": "tests.simple-greeting",
			"messages": []interface{}{
				map[string]interface{}{
					"role":    "user",
					"content": "Hello from any test 1!",
				},
			},
			"options": map[string]interface{}{
				"skip": map[string]interface{}{
					"history": true,
				},
			},
		},
		map[string]interface{}{
			"agent": "tests.simple-greeting",
			"messages": []interface{}{
				map[string]interface{}{
					"role":    "user",
					"content": "Hello from any test 2!",
				},
			},
			"options": map[string]interface{}{
				"skip": map[string]interface{}{
					"history": true,
				},
			},
		},
	}

	results := api.Any(requests)
	require.Len(t, results, 2)

	// At least one should have a result
	hasResult := false
	for i, result := range results {
		if result != nil {
			r, ok := result.(*caller.Result)
			if ok && r != nil && r.Error == "" {
				hasResult = true
				t.Logf("Any Result[%d]: content=%s", i, r.Content)
			}
		}
	}
	assert.True(t, hasResult, "At least one result should succeed")
}

func TestIntegration_Race_RealAgents(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Create authorized info for the context
	authorized := &types.AuthorizedInfo{
		Subject:  "test-user",
		UserID:   "test-123",
		TenantID: "test-tenant",
	}

	// Create a context with authorization
	ctx := agentContext.New(context.Background(), authorized, "test-chat-race")
	ctx.AssistantID = "tests.agent-caller"

	// Create JSAPI
	api := caller.NewJSAPI(ctx)

	// Call multiple agents - return when any completes
	requests := []interface{}{
		map[string]interface{}{
			"agent": "tests.simple-greeting",
			"messages": []interface{}{
				map[string]interface{}{
					"role":    "user",
					"content": "Hello from race test 1!",
				},
			},
			"options": map[string]interface{}{
				"skip": map[string]interface{}{
					"history": true,
				},
			},
		},
		map[string]interface{}{
			"agent": "tests.simple-greeting",
			"messages": []interface{}{
				map[string]interface{}{
					"role":    "user",
					"content": "Hello from race test 2!",
				},
			},
			"options": map[string]interface{}{
				"skip": map[string]interface{}{
					"history": true,
				},
			},
		},
	}

	results := api.Race(requests)
	require.Len(t, results, 2)

	// At least one should have completed
	hasResult := false
	for i, result := range results {
		if result != nil {
			r, ok := result.(*caller.Result)
			if ok && r != nil {
				hasResult = true
				t.Logf("Race Result[%d]: content=%s, error=%s", i, r.Content, r.Error)
			}
		}
	}
	assert.True(t, hasResult, "At least one result should complete")
}
