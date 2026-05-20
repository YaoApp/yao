//go:build integration

package context_test

import (
	stdContext "context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/gou/mcp/types"
	agentctx "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

func newTestMCPContext() *agentctx.Context {
	ctx := agentctx.New(stdContext.Background(), nil, "test-chat")
	ctx.AssistantID = "test-assistant"
	ctx.Locale = "en"
	ctx.Referer = agentctx.RefererAPI

	stack, traceID, _ := agentctx.EnterStack(ctx, "test-assistant", &agentctx.Options{})
	ctx.Stack = stack
	_ = traceID
	return ctx
}

func TestListResources(t *testing.T) {
	testprepare.PrepareSandbox(t)

	ctx := newTestMCPContext()

	result, err := ctx.ListResources("echo", "")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.NotEmpty(t, result.Resources)

	resourceNames := make(map[string]bool)
	for _, resource := range result.Resources {
		resourceNames[resource.Name] = true
	}

	assert.True(t, resourceNames["info"], "Expected 'info' resource")
	assert.True(t, resourceNames["health"], "Expected 'health' resource")
}

func TestReadResource(t *testing.T) {
	testprepare.PrepareSandbox(t)

	ctx := newTestMCPContext()

	t.Run("ReadServerInfo", func(t *testing.T) {
		result, err := ctx.ReadResource("echo", "echo://info")
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.NotEmpty(t, result.Contents)
	})

	t.Run("ReadHealthCheck", func(t *testing.T) {
		result, err := ctx.ReadResource("echo", "echo://health?check=all")
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.NotEmpty(t, result.Contents)
	})
}

func TestListTools(t *testing.T) {
	testprepare.PrepareSandbox(t)

	ctx := newTestMCPContext()

	result, err := ctx.ListTools("echo", "")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.NotEmpty(t, result.Tools)

	toolNames := make(map[string]bool)
	for _, tool := range result.Tools {
		toolNames[tool.Name] = true
	}

	assert.True(t, toolNames["ping"], "Expected 'ping' tool")
	assert.True(t, toolNames["status"], "Expected 'status' tool")
	assert.True(t, toolNames["echo"], "Expected 'echo' tool")
}

func TestCallTool(t *testing.T) {
	testprepare.PrepareSandbox(t)

	ctx := newTestMCPContext()

	t.Run("CallPing", func(t *testing.T) {
		result, err := ctx.CallTool("echo", "ping", map[string]interface{}{
			"count":   3,
			"message": "test",
		})
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.NotEmpty(t, result.Content)
	})

	t.Run("CallStatus", func(t *testing.T) {
		result, err := ctx.CallTool("echo", "status", map[string]interface{}{
			"verbose": true,
		})
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.NotEmpty(t, result.Content)
	})

	t.Run("CallEcho", func(t *testing.T) {
		result, err := ctx.CallTool("echo", "echo", map[string]interface{}{
			"message":   "Hello World",
			"uppercase": true,
		})
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.NotEmpty(t, result.Content)
	})
}

func TestCallTools(t *testing.T) {
	testprepare.PrepareSandbox(t)

	ctx := newTestMCPContext()

	tools := []types.ToolCall{
		{
			Name: "ping",
			Arguments: map[string]interface{}{
				"count": 1,
			},
		},
		{
			Name: "status",
			Arguments: map[string]interface{}{
				"verbose": false,
			},
		},
		{
			Name: "echo",
			Arguments: map[string]interface{}{
				"message": "test",
			},
		},
	}

	result, err := ctx.CallTools("echo", tools)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Len(t, result.Results, 3)
}

func TestCallToolsParallel(t *testing.T) {
	testprepare.PrepareSandbox(t)

	ctx := newTestMCPContext()

	tools := []types.ToolCall{
		{
			Name: "ping",
			Arguments: map[string]interface{}{
				"count": 1,
			},
		},
		{
			Name: "status",
			Arguments: map[string]interface{}{
				"verbose": true,
			},
		},
	}

	result, err := ctx.CallToolsParallel("echo", tools)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Len(t, result.Results, 2)
}

func TestListPrompts(t *testing.T) {
	testprepare.PrepareSandbox(t)

	ctx := newTestMCPContext()

	result, err := ctx.ListPrompts("echo", "")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.NotEmpty(t, result.Prompts)

	promptNames := make(map[string]bool)
	for _, prompt := range result.Prompts {
		promptNames[prompt.Name] = true
	}

	assert.True(t, promptNames["test_connection"], "Expected 'test_connection' prompt")
	assert.True(t, promptNames["test_echo"], "Expected 'test_echo' prompt")
}

func TestGetPrompt(t *testing.T) {
	testprepare.PrepareSandbox(t)

	ctx := newTestMCPContext()

	t.Run("GetTestConnectionPrompt", func(t *testing.T) {
		result, err := ctx.GetPrompt("echo", "test_connection", map[string]interface{}{
			"detailed": "true",
		})
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.NotEmpty(t, result.Messages)
	})

	t.Run("GetTestEchoPrompt", func(t *testing.T) {
		result, err := ctx.GetPrompt("echo", "test_echo", map[string]interface{}{
			"message": "Hello",
			"format":  "uppercase",
		})
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.NotEmpty(t, result.Messages)
	})
}

func TestListSamples(t *testing.T) {
	testprepare.PrepareSandbox(t)

	ctx := newTestMCPContext()

	t.Run("ListToolSamples", func(t *testing.T) {
		result, err := ctx.ListSamples("echo", types.SampleTool, "ping")
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.NotEmpty(t, result.Samples)
	})

	t.Run("ListResourceSamples", func(t *testing.T) {
		result, err := ctx.ListSamples("echo", types.SampleResource, "info")
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.NotEmpty(t, result.Samples)
	})
}

func TestGetSample(t *testing.T) {
	testprepare.PrepareSandbox(t)

	ctx := newTestMCPContext()

	t.Run("GetToolSample", func(t *testing.T) {
		result, err := ctx.GetSample("echo", types.SampleTool, "ping", 0)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.NotEmpty(t, result.Name)
	})

	t.Run("GetResourceSample", func(t *testing.T) {
		result, err := ctx.GetSample("echo", types.SampleResource, "info", 0)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.NotEmpty(t, result.Name)
	})
}

func TestMCPWithTrace(t *testing.T) {
	testprepare.PrepareSandbox(t)

	ctx := newTestMCPContext()

	trace, err := ctx.Trace()
	require.NoError(t, err)
	require.NotNil(t, trace)

	result, err := ctx.CallTool("echo", "ping", map[string]interface{}{
		"count": 5,
	})
	require.NoError(t, err)
	require.NotNil(t, result)

	nodes, err := trace.GetAllNodes()
	require.NoError(t, err)
	assert.NotEmpty(t, nodes)
}
