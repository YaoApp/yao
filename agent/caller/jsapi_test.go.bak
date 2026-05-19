package caller_test

import (
	stdContext "context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/caller"
	"github.com/yaoapp/yao/agent/context"
)

func TestNewJSAPI(t *testing.T) {
	ctx := context.New(stdContext.Background(), nil, "test-chat")
	api := caller.NewJSAPI(ctx)
	require.NotNil(t, api)
}

func TestJSAPI_Call_NoAgentGetter(t *testing.T) {
	// Reset AgentGetterFunc
	originalGetter := caller.AgentGetterFunc
	caller.AgentGetterFunc = nil
	defer func() { caller.AgentGetterFunc = originalGetter }()

	ctx := context.New(stdContext.Background(), nil, "test-chat")
	api := caller.NewJSAPI(ctx)

	messages := []interface{}{
		map[string]interface{}{
			"role":    "user",
			"content": "Hello",
		},
	}

	result := api.Call("test-agent", messages, nil)
	require.NotNil(t, result)

	r, ok := result.(*caller.Result)
	require.True(t, ok)
	assert.Equal(t, "test-agent", r.AgentID)
	assert.Contains(t, r.Error, "agent getter not initialized")
}

func TestJSAPI_All_Empty(t *testing.T) {
	ctx := context.New(stdContext.Background(), nil, "test-chat")
	api := caller.NewJSAPI(ctx)
	results := api.All([]interface{}{})
	assert.Len(t, results, 0)
}

func TestJSAPI_Any_Empty(t *testing.T) {
	ctx := context.New(stdContext.Background(), nil, "test-chat")
	api := caller.NewJSAPI(ctx)
	results := api.Any([]interface{}{})
	assert.Len(t, results, 0)
}

func TestJSAPI_Race_Empty(t *testing.T) {
	ctx := context.New(stdContext.Background(), nil, "test-chat")
	api := caller.NewJSAPI(ctx)
	results := api.Race([]interface{}{})
	assert.Len(t, results, 0)
}

func TestJSAPI_All_InvalidRequests(t *testing.T) {
	ctx := context.New(stdContext.Background(), nil, "test-chat")
	api := caller.NewJSAPI(ctx)

	// Mix of invalid and valid requests
	requests := []interface{}{
		"invalid", // Not a map
		map[string]interface{}{
			"messages": []interface{}{}, // Missing agent
		},
		map[string]interface{}{
			"agent": "test-agent", // Missing messages
		},
	}

	results := api.All(requests)
	// None should produce a result (all invalid)
	assert.Len(t, results, 0)
}

func TestJSAPI_Call_WithOptions(t *testing.T) {
	// Reset AgentGetterFunc
	originalGetter := caller.AgentGetterFunc
	caller.AgentGetterFunc = nil
	defer func() { caller.AgentGetterFunc = originalGetter }()

	ctx := context.New(stdContext.Background(), nil, "test-chat")
	api := caller.NewJSAPI(ctx)

	messages := []interface{}{
		map[string]interface{}{
			"role":    "user",
			"content": "Hello",
		},
	}

	opts := map[string]interface{}{
		"connector": "gpt4",
		"mode":      "chat",
		"metadata": map[string]interface{}{
			"key": "value",
		},
		"skip": map[string]interface{}{
			"history": true,
			"trace":   true,
		},
	}

	result := api.Call("test-agent", messages, opts)
	require.NotNil(t, result)

	r, ok := result.(*caller.Result)
	require.True(t, ok)
	assert.Equal(t, "test-agent", r.AgentID)
	// Still errors because AgentGetterFunc is nil
	assert.Contains(t, r.Error, "agent getter not initialized")
}

func TestSetJSAPIFactory(t *testing.T) {
	// Reset factory
	context.AgentAPIFactory = nil

	// Set factory
	caller.SetJSAPIFactory()

	// Verify factory is set
	require.NotNil(t, context.AgentAPIFactory)

	// Create a mock context
	ctx := context.New(stdContext.Background(), nil, "test-chat")

	// Get agent API
	agentAPI := context.AgentAPIFactory(ctx)
	require.NotNil(t, agentAPI)
}

func TestJSAPI_ImplementsAgentAPI(t *testing.T) {
	// Verify JSAPI implements context.AgentAPI interface
	ctx := context.New(stdContext.Background(), nil, "test-chat")
	var _ context.AgentAPI = caller.NewJSAPI(ctx)
}
