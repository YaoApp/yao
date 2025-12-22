package context_test

import (
	stdContext "context"
	"testing"

	"github.com/yaoapp/gou/mcp/types"
	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/test"
)

// newTestMCPContext creates a test context
func newTestMCPContext() *context.Context {
	ctx := context.New(stdContext.Background(), nil, "test-chat")
	ctx.AssistantID = "test-assistant"
	ctx.Locale = "en"
	ctx.Referer = context.RefererAPI

	// Initialize stack and trace
	stack, traceID, _ := context.EnterStack(ctx, "test-assistant", &context.Options{})
	ctx.Stack = stack
	_ = traceID // traceID is set in stack

	return ctx
}

// TestListResources tests the ListResources function
func TestListResources(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	ctx := newTestMCPContext()

	result, err := ctx.ListResources("echo", "")
	if err != nil {
		t.Fatalf("ListResources failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	if len(result.Resources) == 0 {
		t.Error("Expected resources, got empty list")
	}

	t.Logf("✓ ListResources returned %d resources", len(result.Resources))

	// Check if specific resources exist
	resourceNames := make(map[string]bool)
	for _, resource := range result.Resources {
		resourceNames[resource.Name] = true
		t.Logf("  - Resource: %s (URI: %s)", resource.Name, resource.URI)
	}

	if !resourceNames["info"] {
		t.Error("Expected 'info' resource not found")
	}
	if !resourceNames["health"] {
		t.Error("Expected 'health' resource not found")
	}
}

// TestReadResource tests the ReadResource function
func TestReadResource(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	ctx := newTestMCPContext()

	t.Run("ReadServerInfo", func(t *testing.T) {
		result, err := ctx.ReadResource("echo", "echo://info")
		if err != nil {
			t.Fatalf("ReadResource failed: %v", err)
		}

		if result == nil {
			t.Fatal("Expected result, got nil")
		}

		if len(result.Contents) == 0 {
			t.Error("Expected contents, got empty list")
		}

		t.Logf("✓ ReadResource returned %d contents", len(result.Contents))
	})

	t.Run("ReadHealthCheck", func(t *testing.T) {
		result, err := ctx.ReadResource("echo", "echo://health?check=all")
		if err != nil {
			t.Fatalf("ReadResource failed: %v", err)
		}

		if result == nil {
			t.Fatal("Expected result, got nil")
		}

		if len(result.Contents) == 0 {
			t.Error("Expected contents, got empty list")
		}

		t.Logf("✓ ReadResource for health check returned %d contents", len(result.Contents))
	})
}

// TestListTools tests the ListTools function
func TestListTools(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	ctx := newTestMCPContext()

	result, err := ctx.ListTools("echo", "")
	if err != nil {
		t.Fatalf("ListTools failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	if len(result.Tools) == 0 {
		t.Error("Expected tools, got empty list")
	}

	t.Logf("✓ ListTools returned %d tools", len(result.Tools))

	// Check if specific tools exist
	toolNames := make(map[string]bool)
	for _, tool := range result.Tools {
		toolNames[tool.Name] = true
	}

	if !toolNames["ping"] {
		t.Error("Expected 'ping' tool not found")
	}
	if !toolNames["status"] {
		t.Error("Expected 'status' tool not found")
	}
	if !toolNames["echo"] {
		t.Error("Expected 'echo' tool not found")
	}
}

// TestCallTool tests the CallTool function
func TestCallTool(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	ctx := newTestMCPContext()

	t.Run("CallPing", func(t *testing.T) {
		result, err := ctx.CallTool("echo", "ping", map[string]interface{}{
			"count":   3,
			"message": "test",
		})
		if err != nil {
			t.Fatalf("CallTool failed: %v", err)
		}

		if result == nil {
			t.Fatal("Expected result, got nil")
		}

		if len(result.Content) == 0 {
			t.Error("Expected content, got empty list")
		}

		t.Logf("✓ CallTool (ping) returned %d contents", len(result.Content))
	})

	t.Run("CallStatus", func(t *testing.T) {
		result, err := ctx.CallTool("echo", "status", map[string]interface{}{
			"verbose": true,
		})
		if err != nil {
			t.Fatalf("CallTool failed: %v", err)
		}

		if result == nil {
			t.Fatal("Expected result, got nil")
		}

		if len(result.Content) == 0 {
			t.Error("Expected content, got empty list")
		}

		t.Logf("✓ CallTool (status) returned %d contents", len(result.Content))
	})

	t.Run("CallEcho", func(t *testing.T) {
		result, err := ctx.CallTool("echo", "echo", map[string]interface{}{
			"message":   "Hello World",
			"uppercase": true,
		})
		if err != nil {
			t.Fatalf("CallTool failed: %v", err)
		}

		if result == nil {
			t.Fatal("Expected result, got nil")
		}

		if len(result.Content) == 0 {
			t.Error("Expected content, got empty list")
		}

		t.Logf("✓ CallTool (echo) returned %d contents", len(result.Content))
	})
}

// TestCallTools tests the CallTools function (sequential)
func TestCallTools(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

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
	if err != nil {
		t.Fatalf("CallTools failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	if len(result.Results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(result.Results))
	}

	t.Logf("✓ CallTools returned %d results", len(result.Results))
}

// TestCallToolsParallel tests the CallToolsParallel function
func TestCallToolsParallel(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

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
	if err != nil {
		t.Fatalf("CallToolsParallel failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	if len(result.Results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(result.Results))
	}

	t.Logf("✓ CallToolsParallel returned %d results", len(result.Results))
}

// TestListPrompts tests the ListPrompts function
func TestListPrompts(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	ctx := newTestMCPContext()

	result, err := ctx.ListPrompts("echo", "")
	if err != nil {
		t.Fatalf("ListPrompts failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	if len(result.Prompts) == 0 {
		t.Error("Expected prompts, got empty list")
	}

	t.Logf("✓ ListPrompts returned %d prompts", len(result.Prompts))

	// Check if specific prompts exist
	promptNames := make(map[string]bool)
	for _, prompt := range result.Prompts {
		promptNames[prompt.Name] = true
	}

	if !promptNames["test_connection"] {
		t.Error("Expected 'test_connection' prompt not found")
	}
	if !promptNames["test_echo"] {
		t.Error("Expected 'test_echo' prompt not found")
	}
}

// TestGetPrompt tests the GetPrompt function
func TestGetPrompt(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	ctx := newTestMCPContext()

	t.Run("GetTestConnectionPrompt", func(t *testing.T) {
		result, err := ctx.GetPrompt("echo", "test_connection", map[string]interface{}{
			"detailed": "true",
		})
		if err != nil {
			t.Fatalf("GetPrompt failed: %v", err)
		}

		if result == nil {
			t.Fatal("Expected result, got nil")
		}

		if len(result.Messages) == 0 {
			t.Error("Expected messages, got empty list")
		}

		t.Logf("✓ GetPrompt returned %d messages", len(result.Messages))
	})

	t.Run("GetTestEchoPrompt", func(t *testing.T) {
		result, err := ctx.GetPrompt("echo", "test_echo", map[string]interface{}{
			"message": "Hello",
			"format":  "uppercase",
		})
		if err != nil {
			t.Fatalf("GetPrompt failed: %v", err)
		}

		if result == nil {
			t.Fatal("Expected result, got nil")
		}

		if len(result.Messages) == 0 {
			t.Error("Expected messages, got empty list")
		}

		t.Logf("✓ GetPrompt returned %d messages", len(result.Messages))
	})
}

// TestListSamples tests the ListSamples function
func TestListSamples(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	ctx := newTestMCPContext()

	t.Run("ListToolSamples", func(t *testing.T) {
		result, err := ctx.ListSamples("echo", types.SampleTool, "ping")
		if err != nil {
			t.Fatalf("ListSamples failed: %v", err)
		}

		if result == nil {
			t.Fatal("Expected result, got nil")
		}

		if len(result.Samples) == 0 {
			t.Error("Expected samples, got empty list")
		}

		t.Logf("✓ ListSamples for tool 'ping' returned %d samples", len(result.Samples))
	})

	t.Run("ListResourceSamples", func(t *testing.T) {
		result, err := ctx.ListSamples("echo", types.SampleResource, "info")
		if err != nil {
			t.Fatalf("ListSamples failed: %v", err)
		}

		if result == nil {
			t.Fatal("Expected result, got nil")
		}

		if len(result.Samples) == 0 {
			t.Error("Expected samples, got empty list")
		}

		t.Logf("✓ ListSamples for resource 'info' returned %d samples", len(result.Samples))
	})
}

// TestGetSample tests the GetSample function
func TestGetSample(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	ctx := newTestMCPContext()

	t.Run("GetToolSample", func(t *testing.T) {
		result, err := ctx.GetSample("echo", types.SampleTool, "ping", 0)
		if err != nil {
			t.Fatalf("GetSample failed: %v", err)
		}

		if result == nil {
			t.Fatal("Expected result, got nil")
		}

		if result.Name == "" {
			t.Error("Expected sample name, got empty string")
		}

		t.Logf("✓ GetSample for tool 'ping' returned sample '%s'", result.Name)
	})

	t.Run("GetResourceSample", func(t *testing.T) {
		result, err := ctx.GetSample("echo", types.SampleResource, "info", 0)
		if err != nil {
			t.Fatalf("GetSample failed: %v", err)
		}

		if result == nil {
			t.Fatal("Expected result, got nil")
		}

		if result.Name == "" {
			t.Error("Expected sample name, got empty string")
		}

		t.Logf("✓ GetSample for resource 'info' returned sample '%s'", result.Name)
	})
}

// TestMCPWithTrace tests MCP operations with trace
func TestMCPWithTrace(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	ctx := newTestMCPContext()

	// Initialize trace
	trace, err := ctx.Trace()
	if err != nil {
		t.Fatalf("Failed to initialize trace: %v", err)
	}

	if trace == nil {
		t.Fatal("Expected trace, got nil")
	}

	// Call tool with trace
	result, err := ctx.CallTool("echo", "ping", map[string]interface{}{
		"count": 5,
	})
	if err != nil {
		t.Fatalf("CallTool with trace failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	// Get trace nodes to verify trace was created
	nodes, err := trace.GetAllNodes()
	if err != nil {
		t.Fatalf("Failed to get trace nodes: %v", err)
	}

	if len(nodes) == 0 {
		t.Error("Expected trace nodes, got empty list")
	}

	t.Logf("✓ MCP operation created %d trace nodes", len(nodes))
}
