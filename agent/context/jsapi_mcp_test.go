package context_test

import (
	stdContext "context"
	"testing"

	"github.com/stretchr/testify/assert"
	v8 "github.com/yaoapp/gou/runtime/v8"
	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/test"
)

// newMCPTestContext creates a test context for MCP testing
func newMCPTestContext() *context.Context {
	ctx := context.New(stdContext.Background(), nil, "test-chat-id")
	ctx.AssistantID = "test-assistant-id"
	ctx.Locale = "en"
	ctx.Referer = context.RefererAPI
	stack, _, _ := context.EnterStack(ctx, "test-assistant", &context.Options{})
	ctx.Stack = stack
	return ctx
}

// TestMCPListResources tests MCP.ListResources from JavaScript
func TestMCPListResources(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	ctx := newMCPTestContext()

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			// List resources from echo MCP
			const result = ctx.mcp.ListResources("echo", "")
			
			if (!result || !result.resources) {
				throw new Error("Expected resources")
			}
			
			return {
				count: result.resources.length,
				has_info: result.resources.some(r => r.name === "info"),
				has_health: result.resources.some(r => r.name === "health")
			}
		}`, ctx)

	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	result, ok := res.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", res)
	}

	assert.Equal(t, float64(2), result["count"], "should have 2 resources")
	assert.Equal(t, true, result["has_info"], "should have info resource")
	assert.Equal(t, true, result["has_health"], "should have health resource")
}

// TestMCPReadResource tests MCP.ReadResource from JavaScript
func TestMCPReadResource(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	ctx := newMCPTestContext()

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			// Read info resource
			const result = ctx.mcp.ReadResource("echo", "echo://info")
			
			if (!result || !result.contents) {
				throw new Error("Expected contents")
			}
			
			return {
				count: result.contents.length,
				has_content: result.contents.length > 0
			}
		}`, ctx)

	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	result, ok := res.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", res)
	}

	assert.Equal(t, float64(1), result["count"], "should have 1 content")
	assert.Equal(t, true, result["has_content"], "should have content")
}

// TestMCPListTools tests MCP.ListTools from JavaScript
func TestMCPListTools(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	ctx := newMCPTestContext()

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			// List tools from echo MCP
			const result = ctx.mcp.ListTools("echo", "")
			
			if (!result || !result.tools) {
				throw new Error("Expected tools")
			}
			
			return {
				count: result.tools.length,
				has_ping: result.tools.some(t => t.name === "ping"),
				has_status: result.tools.some(t => t.name === "status"),
				has_echo: result.tools.some(t => t.name === "echo")
			}
		}`, ctx)

	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	result, ok := res.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", res)
	}

	assert.Equal(t, float64(3), result["count"], "should have 3 tools")
	assert.Equal(t, true, result["has_ping"], "should have ping tool")
	assert.Equal(t, true, result["has_status"], "should have status tool")
	assert.Equal(t, true, result["has_echo"], "should have echo tool")
}

// TestMCPCallTool tests MCP.CallTool from JavaScript
func TestMCPCallTool(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	ctx := newMCPTestContext()

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			// Call ping tool
			const result = ctx.mcp.CallTool("echo", "ping", { count: 3, message: "test" })
			
			if (!result || !result.content) {
				throw new Error("Expected content")
			}
			
			return {
				has_content: result.content.length > 0,
				is_error: result.isError || false
			}
		}`, ctx)

	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	result, ok := res.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", res)
	}

	assert.Equal(t, true, result["has_content"], "should have content")
	assert.Equal(t, false, result["is_error"], "should not be error")
}

// TestMCPCallTools tests MCP.CallTools from JavaScript
func TestMCPCallTools(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	ctx := newMCPTestContext()

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			// Call multiple tools sequentially
			const tools = [
				{ name: "ping", arguments: { count: 1 } },
				{ name: "status", arguments: { verbose: false } }
			]
			
			const result = ctx.mcp.CallTools("echo", tools)
			
			if (!result || !result.results) {
				throw new Error("Expected results")
			}
			
			return {
				count: result.results.length,
				all_success: result.results.every(r => !r.isError)
			}
		}`, ctx)

	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	result, ok := res.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", res)
	}

	assert.Equal(t, float64(2), result["count"], "should have 2 results")
	assert.Equal(t, true, result["all_success"], "all calls should succeed")
}

// TestMCPCallToolsParallel tests MCP.CallToolsParallel from JavaScript
func TestMCPCallToolsParallel(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	ctx := newMCPTestContext()

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			// Call multiple tools in parallel
			const tools = [
				{ name: "ping", arguments: { count: 1 } },
				{ name: "status", arguments: { verbose: true } }
			]
			
			const result = ctx.mcp.CallToolsParallel("echo", tools)
			
			if (!result || !result.results) {
				throw new Error("Expected results")
			}
			
			return {
				count: result.results.length,
				all_success: result.results.every(r => !r.isError)
			}
		}`, ctx)

	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	result, ok := res.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", res)
	}

	assert.Equal(t, float64(2), result["count"], "should have 2 results")
	assert.Equal(t, true, result["all_success"], "all calls should succeed")
}

// TestMCPListPrompts tests MCP.ListPrompts from JavaScript
func TestMCPListPrompts(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	ctx := newMCPTestContext()

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			// List prompts from echo MCP
			const result = ctx.mcp.ListPrompts("echo", "")
			
			if (!result || !result.prompts) {
				throw new Error("Expected prompts")
			}
			
			return {
				count: result.prompts.length,
				has_test_connection: result.prompts.some(p => p.name === "test_connection"),
				has_test_echo: result.prompts.some(p => p.name === "test_echo")
			}
		}`, ctx)

	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	result, ok := res.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", res)
	}

	assert.Equal(t, float64(2), result["count"], "should have 2 prompts")
	assert.Equal(t, true, result["has_test_connection"], "should have test_connection prompt")
	assert.Equal(t, true, result["has_test_echo"], "should have test_echo prompt")
}

// TestMCPGetPrompt tests MCP.GetPrompt from JavaScript
func TestMCPGetPrompt(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	ctx := newMCPTestContext()

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			// Get test_connection prompt
			const result = ctx.mcp.GetPrompt("echo", "test_connection", { detailed: "true" })
			
			if (!result || !result.messages) {
				throw new Error("Expected messages")
			}
			
			return {
				count: result.messages.length,
				has_messages: result.messages.length > 0
			}
		}`, ctx)

	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	result, ok := res.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", res)
	}

	assert.Equal(t, float64(1), result["count"], "should have 1 message")
	assert.Equal(t, true, result["has_messages"], "should have messages")
}

// TestMCPListSamples tests MCP.ListSamples from JavaScript
func TestMCPListSamples(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	ctx := newMCPTestContext()

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			// List samples for ping tool
			const result = ctx.mcp.ListSamples("echo", "tool", "ping")
			
			if (!result || !result.samples) {
				throw new Error("Expected samples")
			}
			
			return {
				count: result.samples.length,
				has_samples: result.samples.length > 0
			}
		}`, ctx)

	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	result, ok := res.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", res)
	}

	assert.Equal(t, float64(3), result["count"], "should have 3 samples")
	assert.Equal(t, true, result["has_samples"], "should have samples")
}

// TestMCPGetSample tests MCP.GetSample from JavaScript
func TestMCPGetSample(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	ctx := newMCPTestContext()

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			// Get first sample for ping tool
			const result = ctx.mcp.GetSample("echo", "tool", "ping", 0)
			
			if (!result) {
				throw new Error("Expected sample")
			}
			
			return {
				has_name: !!result.name,
				has_input: !!result.input,
				name: result.name
			}
		}`, ctx)

	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	result, ok := res.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", res)
	}

	assert.Equal(t, true, result["has_name"], "should have name")
	assert.Equal(t, true, result["has_input"], "should have input")
	assert.Equal(t, "single_ping", result["name"], "name should be single_ping")
}

// TestMCPJsApiWithTrace tests MCP operations with trace from JavaScript
func TestMCPJsApiWithTrace(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	ctx := newMCPTestContext()

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			// Get trace (property, not method call)
			const trace = ctx.trace
			
			// Call MCP tool - should create trace node
			const result = ctx.mcp.CallTool("echo", "ping", { count: 5 })
			
			// Verify trace and result exist
			return {
				has_trace: !!trace,
				has_result: !!result,
				has_content: result.content && result.content.length > 0
			}
		}`, ctx)

	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	result, ok := res.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", res)
	}

	assert.Equal(t, true, result["has_trace"], "should have trace")
	assert.Equal(t, true, result["has_result"], "should have result")
	assert.Equal(t, true, result["has_content"], "should have content")
}
