//go:build integration

package context_test

import (
	stdContext "context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v8 "github.com/yaoapp/gou/runtime/v8"
	agentctx "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

func newMCPTestContext() *agentctx.Context {
	ctx := agentctx.New(stdContext.Background(), nil, "test-chat-id")
	ctx.AssistantID = "test-assistant-id"
	ctx.Locale = "en"
	ctx.Referer = agentctx.RefererAPI
	stack, _, _ := agentctx.EnterStack(ctx, "test-assistant", &agentctx.Options{})
	ctx.Stack = stack
	return ctx
}

func TestMCPListResources(t *testing.T) {
	testprepare.PrepareSandbox(t)

	ctx := newMCPTestContext()

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
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
	require.NoError(t, err, "Call failed")

	result, ok := res.(map[string]interface{})
	require.True(t, ok, "Expected map result, got %T", res)

	assert.Equal(t, float64(2), result["count"], "should have 2 resources")
	assert.Equal(t, true, result["has_info"], "should have info resource")
	assert.Equal(t, true, result["has_health"], "should have health resource")
}

func TestMCPReadResource(t *testing.T) {
	testprepare.PrepareSandbox(t)

	ctx := newMCPTestContext()

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			const result = ctx.mcp.ReadResource("echo", "echo://info")
			
			if (!result || !result.contents) {
				throw new Error("Expected contents")
			}
			
			return {
				count: result.contents.length,
				has_content: result.contents.length > 0
			}
		}`, ctx)
	require.NoError(t, err, "Call failed")

	result, ok := res.(map[string]interface{})
	require.True(t, ok, "Expected map result, got %T", res)

	assert.Equal(t, float64(1), result["count"], "should have 1 content")
	assert.Equal(t, true, result["has_content"], "should have content")
}

func TestMCPListTools(t *testing.T) {
	testprepare.PrepareSandbox(t)

	ctx := newMCPTestContext()

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
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
	require.NoError(t, err, "Call failed")

	result, ok := res.(map[string]interface{})
	require.True(t, ok, "Expected map result, got %T", res)

	assert.Equal(t, float64(3), result["count"], "should have 3 tools")
	assert.Equal(t, true, result["has_ping"], "should have ping tool")
	assert.Equal(t, true, result["has_status"], "should have status tool")
	assert.Equal(t, true, result["has_echo"], "should have echo tool")
}

func TestMCPCallTool(t *testing.T) {
	testprepare.PrepareSandbox(t)

	ctx := newMCPTestContext()

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			const result = ctx.mcp.CallTool("echo", "ping", { count: 3, message: "test" })
			
			if (result === undefined || result === null) {
				throw new Error("Expected result")
			}
			
			return {
				has_result: true,
				message: result.message
			}
		}`, ctx)
	require.NoError(t, err, "Call failed")

	result, ok := res.(map[string]interface{})
	require.True(t, ok, "Expected map result, got %T", res)

	assert.Equal(t, true, result["has_result"], "should have result")
	assert.Equal(t, "test", result["message"], "should have message")
}

func TestMCPCallTools(t *testing.T) {
	testprepare.PrepareSandbox(t)

	ctx := newMCPTestContext()

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			const tools = [
				{ name: "ping", arguments: { count: 1 } },
				{ name: "status", arguments: { verbose: false } }
			]
			
			const results = ctx.mcp.CallTools("echo", tools)
			
			if (!Array.isArray(results)) {
				throw new Error("Expected array of results")
			}
			
			return {
				count: results.length,
				ping_message: results[0]?.message,
				status_online: results[1]?.status === "online"
			}
		}`, ctx)
	require.NoError(t, err, "Call failed")

	result, ok := res.(map[string]interface{})
	require.True(t, ok, "Expected map result, got %T", res)

	assert.Equal(t, float64(2), result["count"], "should have 2 results")
	assert.Equal(t, "pong", result["ping_message"], "ping should return pong")
	assert.Equal(t, true, result["status_online"], "status should be online")
}

func TestMCPCallToolsParallel(t *testing.T) {
	testprepare.PrepareSandbox(t)

	ctx := newMCPTestContext()

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			const tools = [
				{ name: "ping", arguments: { count: 1 } },
				{ name: "status", arguments: { verbose: true } }
			]
			
			const results = ctx.mcp.CallToolsParallel("echo", tools)
			
			if (!Array.isArray(results)) {
				throw new Error("Expected array of results")
			}
			
			return {
				count: results.length,
				ping_message: results[0]?.message,
				status_online: results[1]?.status === "online"
			}
		}`, ctx)
	require.NoError(t, err, "Call failed")

	result, ok := res.(map[string]interface{})
	require.True(t, ok, "Expected map result, got %T", res)

	assert.Equal(t, float64(2), result["count"], "should have 2 results")
	assert.Equal(t, "pong", result["ping_message"], "ping should return pong")
	assert.Equal(t, true, result["status_online"], "status should be online")
}

func TestMCPListPrompts(t *testing.T) {
	testprepare.PrepareSandbox(t)

	ctx := newMCPTestContext()

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
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
	require.NoError(t, err, "Call failed")

	result, ok := res.(map[string]interface{})
	require.True(t, ok, "Expected map result, got %T", res)

	assert.Equal(t, float64(2), result["count"], "should have 2 prompts")
	assert.Equal(t, true, result["has_test_connection"], "should have test_connection prompt")
	assert.Equal(t, true, result["has_test_echo"], "should have test_echo prompt")
}

func TestMCPGetPrompt(t *testing.T) {
	testprepare.PrepareSandbox(t)

	ctx := newMCPTestContext()

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			const result = ctx.mcp.GetPrompt("echo", "test_connection", { detailed: "true" })
			
			if (!result || !result.messages) {
				throw new Error("Expected messages")
			}
			
			return {
				count: result.messages.length,
				has_messages: result.messages.length > 0
			}
		}`, ctx)
	require.NoError(t, err, "Call failed")

	result, ok := res.(map[string]interface{})
	require.True(t, ok, "Expected map result, got %T", res)

	assert.Equal(t, float64(1), result["count"], "should have 1 message")
	assert.Equal(t, true, result["has_messages"], "should have messages")
}

func TestMCPJsApiWithTrace(t *testing.T) {
	testprepare.PrepareSandbox(t)

	ctx := newMCPTestContext()

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			const trace = ctx.trace
			
			const result = ctx.mcp.CallTool("echo", "ping", { count: 5 })
			
			return {
				has_trace: !!trace,
				has_result: result !== undefined && result !== null,
				ping_message: result?.message
			}
		}`, ctx)
	require.NoError(t, err, "Call failed")

	result, ok := res.(map[string]interface{})
	require.True(t, ok, "Expected map result, got %T", res)

	assert.Equal(t, true, result["has_trace"], "should have trace")
	assert.Equal(t, true, result["has_result"], "should have result")
	assert.Equal(t, "pong", result["ping_message"], "should have ping response")
}

func TestMCP_All_V8(t *testing.T) {
	testprepare.PrepareSandbox(t)

	authorized := &types.AuthorizedInfo{
		Subject:  "test-user",
		UserID:   "test-123",
		TenantID: "test-tenant",
	}

	ctx := agentctx.New(stdContext.Background(), authorized, "test-chat-mcp-all")
	ctx.AssistantID = "tests.simple-greeting"
	defer ctx.Release()

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				const results = ctx.mcp.All([
					{ mcp: "echo", tool: "ping", arguments: { count: 1 } },
					{ mcp: "echo", tool: "status", arguments: { verbose: false } },
					{ mcp: "echo", tool: "echo", arguments: { message: "hello" } }
				]);
				
				return {
					success: true,
					count: results.length,
					results: results.map(r => ({
						mcp: r.mcp,
						tool: r.tool,
						has_result: r.result !== undefined && r.result !== null,
						error: r.error || ""
					}))
				};
			} catch (error) {
				return { success: false, error: error.message };
			}
		}`, ctx)

	require.NoError(t, err)
	result, ok := res.(map[string]interface{})
	require.True(t, ok, "Result should be a map")

	success, _ := result["success"].(bool)
	if !success {
		t.Fatalf("Test failed: %v", result["error"])
	}

	var count int
	switch v := result["count"].(type) {
	case int:
		count = v
	case int32:
		count = int(v)
	case int64:
		count = int(v)
	case float64:
		count = int(v)
	default:
		t.Logf("Unexpected count type: %T, value: %v", result["count"], result["count"])
	}
	assert.Equal(t, 3, count, "Should have 3 results")

	results, ok := result["results"].([]interface{})
	require.True(t, ok, "Results should be an array")
	require.Len(t, results, 3)

	for i, r := range results {
		resMap, ok := r.(map[string]interface{})
		require.True(t, ok, "Result %d should be a map", i)

		hasResult, _ := resMap["has_result"].(bool)
		assert.True(t, hasResult, "Result %d should have parsed result", i)

		errorStr, _ := resMap["error"].(string)
		assert.Empty(t, errorStr, "Result %d should not have error", i)
	}
}

func TestMCP_All_WithError_V8(t *testing.T) {
	testprepare.PrepareSandbox(t)

	authorized := &types.AuthorizedInfo{
		Subject:  "test-user",
		UserID:   "test-123",
		TenantID: "test-tenant",
	}

	ctx := agentctx.New(stdContext.Background(), authorized, "test-chat-mcp-all-error")
	ctx.AssistantID = "tests.simple-greeting"
	defer ctx.Release()

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				const results = ctx.mcp.All([
					{ mcp: "echo", tool: "ping", arguments: { count: 1 } },
					{ mcp: "nonexistent-mcp", tool: "some-tool", arguments: {} },
					{ mcp: "echo", tool: "status", arguments: {} }
				]);
				
				return {
					success: true,
					count: results.length,
					results: results.map(r => ({
						mcp: r.mcp,
						tool: r.tool,
						has_result: r.result !== undefined && r.result !== null,
						has_error: r.error !== undefined && r.error !== "" && r.error !== null
					}))
				};
			} catch (error) {
				return { success: false, error: error.message };
			}
		}`, ctx)

	require.NoError(t, err)
	result, ok := res.(map[string]interface{})
	require.True(t, ok, "Result should be a map")

	success, _ := result["success"].(bool)
	if !success {
		t.Fatalf("Test failed: %v", result["error"])
	}

	var count int
	switch v := result["count"].(type) {
	case int:
		count = v
	case int32:
		count = int(v)
	case int64:
		count = int(v)
	case float64:
		count = int(v)
	}
	assert.Equal(t, 3, count, "Should have 3 results")

	results, ok := result["results"].([]interface{})
	require.True(t, ok, "Results should be an array")

	r0, _ := results[0].(map[string]interface{})
	assert.True(t, r0["has_result"].(bool), "Ping should have result")
	assert.False(t, r0["has_error"].(bool), "Ping should not have error")

	r1, _ := results[1].(map[string]interface{})
	assert.False(t, r1["has_result"].(bool), "Nonexistent should not have result")
	assert.True(t, r1["has_error"].(bool), "Nonexistent should have error")

	r2, _ := results[2].(map[string]interface{})
	assert.True(t, r2["has_result"].(bool), "Status should have result")
	assert.False(t, r2["has_error"].(bool), "Status should not have error")
}

func TestMCP_Any_V8(t *testing.T) {
	testprepare.PrepareSandbox(t)

	authorized := &types.AuthorizedInfo{
		Subject:  "test-user",
		UserID:   "test-123",
		TenantID: "test-tenant",
	}

	ctx := agentctx.New(stdContext.Background(), authorized, "test-chat-mcp-any")
	ctx.AssistantID = "tests.simple-greeting"
	defer ctx.Release()

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				const results = ctx.mcp.Any([
					{ mcp: "echo", tool: "ping", arguments: { count: 1 } },
					{ mcp: "echo", tool: "status", arguments: {} }
				]);
				
				const successResults = results.filter(r => r && r.result && !r.error);
				
				return {
					success: true,
					total_count: results.length,
					success_count: successResults.length,
					has_at_least_one_success: successResults.length >= 1
				};
			} catch (error) {
				return { success: false, error: error.message };
			}
		}`, ctx)

	require.NoError(t, err)
	result, ok := res.(map[string]interface{})
	require.True(t, ok, "Result should be a map")

	success, _ := result["success"].(bool)
	if !success {
		t.Fatalf("Test failed: %v", result["error"])
	}

	hasAtLeastOne, _ := result["has_at_least_one_success"].(bool)
	assert.True(t, hasAtLeastOne, "Should have at least one successful result")
}

func TestMCP_Any_AllFail_V8(t *testing.T) {
	testprepare.PrepareSandbox(t)

	authorized := &types.AuthorizedInfo{
		Subject:  "test-user",
		UserID:   "test-123",
		TenantID: "test-tenant",
	}

	ctx := agentctx.New(stdContext.Background(), authorized, "test-chat-mcp-any-fail")
	ctx.AssistantID = "tests.simple-greeting"
	defer ctx.Release()

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				const results = ctx.mcp.Any([
					{ mcp: "nonexistent-1", tool: "tool1", arguments: {} },
					{ mcp: "nonexistent-2", tool: "tool2", arguments: {} }
				]);
				
				const failedResults = results.filter(r => r && r.error);
				
				return {
					success: true,
					total_count: results.length,
					failed_count: failedResults.length,
					all_failed: failedResults.length === results.length
				};
			} catch (error) {
				return { success: false, error: error.message };
			}
		}`, ctx)

	require.NoError(t, err)
	result, ok := res.(map[string]interface{})
	require.True(t, ok, "Result should be a map")

	success, _ := result["success"].(bool)
	if !success {
		t.Fatalf("Test failed: %v", result["error"])
	}

	allFailed, _ := result["all_failed"].(bool)
	assert.True(t, allFailed, "All requests should fail")
}

func TestMCP_Race_V8(t *testing.T) {
	testprepare.PrepareSandbox(t)

	authorized := &types.AuthorizedInfo{
		Subject:  "test-user",
		UserID:   "test-123",
		TenantID: "test-tenant",
	}

	ctx := agentctx.New(stdContext.Background(), authorized, "test-chat-mcp-race")
	ctx.AssistantID = "tests.simple-greeting"
	defer ctx.Release()

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				const results = ctx.mcp.Race([
					{ mcp: "echo", tool: "ping", arguments: { count: 1 } },
					{ mcp: "echo", tool: "status", arguments: {} }
				]);
				
				const completedResults = results.filter(r => r !== undefined && r !== null);
				
				return {
					success: true,
					total_count: results.length,
					completed_count: completedResults.length,
					has_at_least_one_completed: completedResults.length >= 1
				};
			} catch (error) {
				return { success: false, error: error.message };
			}
		}`, ctx)

	require.NoError(t, err)
	result, ok := res.(map[string]interface{})
	require.True(t, ok, "Result should be a map")

	success, _ := result["success"].(bool)
	if !success {
		t.Fatalf("Test failed: %v", result["error"])
	}

	hasAtLeastOne, _ := result["has_at_least_one_completed"].(bool)
	assert.True(t, hasAtLeastOne, "Should have at least one completed result")
}

func TestMCP_All_ResultContent_V8(t *testing.T) {
	testprepare.PrepareSandbox(t)

	authorized := &types.AuthorizedInfo{
		Subject:  "test-user",
		UserID:   "test-123",
		TenantID: "test-tenant",
	}

	ctx := agentctx.New(stdContext.Background(), authorized, "test-chat-mcp-content")
	ctx.AssistantID = "tests.simple-greeting"
	defer ctx.Release()

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				const results = ctx.mcp.All([
					{ mcp: "echo", tool: "echo", arguments: { message: "hello world", uppercase: true } }
				]);
				
				if (results.length !== 1) {
					return { success: false, error: "Expected 1 result" };
				}
				
				const r = results[0];
				if (r.error) {
					return { success: false, error: "Tool call failed: " + r.error };
				}
				
				const data = r.result;
				if (!data) {
					return { success: false, error: "Result should have parsed data" };
				}
				
				return {
					success: true,
					echo_message: data.echo,
					uppercase_flag: data.uppercase,
					original_length: data.length
				};
			} catch (error) {
				return { success: false, error: error.message };
			}
		}`, ctx)

	require.NoError(t, err)
	result, ok := res.(map[string]interface{})
	require.True(t, ok, "Result should be a map")

	success, _ := result["success"].(bool)
	if !success {
		t.Fatalf("Test failed: %v", result["error"])
	}

	echoMessage, _ := result["echo_message"].(string)
	assert.Equal(t, "HELLO WORLD", echoMessage, "Echo message should be uppercase")

	uppercaseFlag, _ := result["uppercase_flag"].(bool)
	assert.True(t, uppercaseFlag, "Uppercase flag should be true")

	var originalLength int
	switch v := result["original_length"].(type) {
	case int:
		originalLength = v
	case int32:
		originalLength = int(v)
	case int64:
		originalLength = int(v)
	case float64:
		originalLength = int(v)
	}
	assert.Equal(t, 11, originalLength, "Original message length should be 11")
}

func TestMCP_All_MultipleTools_V8(t *testing.T) {
	testprepare.PrepareSandbox(t)

	authorized := &types.AuthorizedInfo{
		Subject:  "test-user",
		UserID:   "test-123",
		TenantID: "test-tenant",
	}

	ctx := agentctx.New(stdContext.Background(), authorized, "test-chat-mcp-multi")
	ctx.AssistantID = "tests.simple-greeting"
	defer ctx.Release()

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				const results = ctx.mcp.All([
					{ mcp: "echo", tool: "ping", arguments: { count: 5 } },
					{ mcp: "echo", tool: "echo", arguments: { message: "test", uppercase: false } }
				]);
				
				if (results.length !== 2) {
					return { success: false, error: "Expected 2 results" };
				}
				
				const ping = results[0];
				const echo = results[1];
				
				return {
					success: true,
					ping_message: ping.result?.message,
					echo_message: echo.result?.echo
				};
			} catch (error) {
				return { success: false, error: error.message };
			}
		}`, ctx)

	require.NoError(t, err)
	result, ok := res.(map[string]interface{})
	require.True(t, ok, "Result should be a map")

	success, _ := result["success"].(bool)
	if !success {
		t.Fatalf("Test failed: %v", result["error"])
	}

	pingMessage, _ := result["ping_message"].(string)
	assert.Equal(t, "pong", pingMessage, "Ping should return pong")

	echoMessage, _ := result["echo_message"].(string)
	assert.Equal(t, "test", echoMessage, "Echo should return the message")
}

func TestMCP_CallTool_ParsedResult_V8(t *testing.T) {
	testprepare.PrepareSandbox(t)

	authorized := &types.AuthorizedInfo{
		Subject:  "test-user",
		UserID:   "test-123",
		TenantID: "test-tenant",
	}

	ctx := agentctx.New(stdContext.Background(), authorized, "test-chat-mcp-calltool-parsed")
	ctx.AssistantID = "tests.simple-greeting"
	defer ctx.Release()

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				const result = ctx.mcp.CallTool("echo", "echo", { 
					message: "test message", 
					uppercase: true 
				});
				
				if (result === undefined || result === null) {
					return { success: false, error: "Result should not be null" };
				}
				
				return {
					success: true,
					echo_message: result.echo,
					original_length: result.length
				};
			} catch (error) {
				return { success: false, error: error.message };
			}
		}`, ctx)

	require.NoError(t, err)
	result, ok := res.(map[string]interface{})
	require.True(t, ok, "Result should be a map")

	success, _ := result["success"].(bool)
	if !success {
		t.Fatalf("Test failed: %v", result["error"])
	}

	echoMessage, _ := result["echo_message"].(string)
	assert.Equal(t, "TEST MESSAGE", echoMessage, "Echo message should be uppercase")

	var length int
	switch v := result["original_length"].(type) {
	case int:
		length = v
	case int32:
		length = int(v)
	case int64:
		length = int(v)
	case float64:
		length = int(v)
	}
	assert.Equal(t, 12, length, "Original message length should be 12")
}

func TestMCP_CallToolsParallel_ParsedResult_V8(t *testing.T) {
	testprepare.PrepareSandbox(t)

	authorized := &types.AuthorizedInfo{
		Subject:  "test-user",
		UserID:   "test-123",
		TenantID: "test-tenant",
	}

	ctx := agentctx.New(stdContext.Background(), authorized, "test-chat-mcp-calltools-parsed")
	ctx.AssistantID = "tests.simple-greeting"
	defer ctx.Release()

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				const results = ctx.mcp.CallToolsParallel("echo", [
					{ name: "ping", arguments: { count: 2 } },
					{ name: "echo", arguments: { message: "hello", uppercase: false } }
				]);
				
				if (!Array.isArray(results)) {
					return { success: false, error: "Results should be an array" };
				}
				if (results.length !== 2) {
					return { success: false, error: "Expected 2 results, got " + results.length };
				}
				
				const pingResult = results[0];
				const echoResult = results[1];
				
				return {
					success: true,
					ping_message: pingResult?.message,
					echo_message: echoResult?.echo
				};
			} catch (error) {
				return { success: false, error: error.message };
			}
		}`, ctx)

	require.NoError(t, err)
	result, ok := res.(map[string]interface{})
	require.True(t, ok, "Result should be a map")

	success, _ := result["success"].(bool)
	if !success {
		t.Fatalf("Test failed: %v", result["error"])
	}

	pingMessage, _ := result["ping_message"].(string)
	assert.Equal(t, "pong", pingMessage, "Ping result should have message='pong'")

	echoMessage, _ := result["echo_message"].(string)
	assert.Equal(t, "hello", echoMessage, "Echo message should be preserved (no uppercase)")
}
