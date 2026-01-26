package context_test

import (
	stdContext "context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v8 "github.com/yaoapp/gou/runtime/v8"
	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/testutils"
	"github.com/yaoapp/yao/openapi/oauth/types"
)

// TestMCP_All_V8 tests ctx.mcp.All() with real V8 execution
func TestMCP_All_V8(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	authorized := &types.AuthorizedInfo{
		Subject:  "test-user",
		UserID:   "test-123",
		TenantID: "test-tenant",
	}

	ctx := context.New(stdContext.Background(), authorized, "test-chat-mcp-all")
	ctx.AssistantID = "tests.agent-caller"
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
					// Each result has mcp, tool, result (parsed), error
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

	// Should have 3 results - handle different integer types
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

	// Check each result
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

// TestMCP_All_WithError_V8 tests ctx.mcp.All() with some failing requests
func TestMCP_All_WithError_V8(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	authorized := &types.AuthorizedInfo{
		Subject:  "test-user",
		UserID:   "test-123",
		TenantID: "test-tenant",
	}

	ctx := context.New(stdContext.Background(), authorized, "test-chat-mcp-all-error")
	ctx.AssistantID = "tests.agent-caller"
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

	// Should have 3 results - handle different integer types
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

	// Check results
	results, ok := result["results"].([]interface{})
	require.True(t, ok, "Results should be an array")

	// First result (ping) should succeed
	r0, _ := results[0].(map[string]interface{})
	assert.True(t, r0["has_result"].(bool), "Ping should have result")
	assert.False(t, r0["has_error"].(bool), "Ping should not have error")

	// Second result (nonexistent) should fail
	r1, _ := results[1].(map[string]interface{})
	assert.False(t, r1["has_result"].(bool), "Nonexistent should not have result")
	assert.True(t, r1["has_error"].(bool), "Nonexistent should have error")

	// Third result (status) should succeed
	r2, _ := results[2].(map[string]interface{})
	assert.True(t, r2["has_result"].(bool), "Status should have result")
	assert.False(t, r2["has_error"].(bool), "Status should not have error")
}

// TestMCP_Any_V8 tests ctx.mcp.Any() with real V8 execution
func TestMCP_Any_V8(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	authorized := &types.AuthorizedInfo{
		Subject:  "test-user",
		UserID:   "test-123",
		TenantID: "test-tenant",
	}

	ctx := context.New(stdContext.Background(), authorized, "test-chat-mcp-any")
	ctx.AssistantID = "tests.agent-caller"
	defer ctx.Release()

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				const results = ctx.mcp.Any([
					{ mcp: "echo", tool: "ping", arguments: { count: 1 } },
					{ mcp: "echo", tool: "status", arguments: {} }
				]);
				
				// Find success results (has result field, no error)
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

// TestMCP_Any_AllFail_V8 tests ctx.mcp.Any() when all requests fail
func TestMCP_Any_AllFail_V8(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	authorized := &types.AuthorizedInfo{
		Subject:  "test-user",
		UserID:   "test-123",
		TenantID: "test-tenant",
	}

	ctx := context.New(stdContext.Background(), authorized, "test-chat-mcp-any-fail")
	ctx.AssistantID = "tests.agent-caller"
	defer ctx.Release()

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				const results = ctx.mcp.Any([
					{ mcp: "nonexistent-1", tool: "tool1", arguments: {} },
					{ mcp: "nonexistent-2", tool: "tool2", arguments: {} }
				]);
				
				// All should fail
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

// TestMCP_Race_V8 tests ctx.mcp.Race() with real V8 execution
func TestMCP_Race_V8(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	authorized := &types.AuthorizedInfo{
		Subject:  "test-user",
		UserID:   "test-123",
		TenantID: "test-tenant",
	}

	ctx := context.New(stdContext.Background(), authorized, "test-chat-mcp-race")
	ctx.AssistantID = "tests.agent-caller"
	defer ctx.Release()

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				const results = ctx.mcp.Race([
					{ mcp: "echo", tool: "ping", arguments: { count: 1 } },
					{ mcp: "echo", tool: "status", arguments: {} }
				]);
				
				// Find completed results (could be success or error)
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

// TestMCP_All_ResultContent_V8 tests that the result contains parsed content directly
func TestMCP_All_ResultContent_V8(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	authorized := &types.AuthorizedInfo{
		Subject:  "test-user",
		UserID:   "test-123",
		TenantID: "test-tenant",
	}

	ctx := context.New(stdContext.Background(), authorized, "test-chat-mcp-content")
	ctx.AssistantID = "tests.agent-caller"
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
				
				// Result should contain parsed data directly
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

	// Handle different integer types from V8
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

// TestMCP_All_MultipleTools_V8 tests All with multiple tools and verifies parsed results
func TestMCP_All_MultipleTools_V8(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	authorized := &types.AuthorizedInfo{
		Subject:  "test-user",
		UserID:   "test-123",
		TenantID: "test-tenant",
	}

	ctx := context.New(stdContext.Background(), authorized, "test-chat-mcp-multi")
	ctx.AssistantID = "tests.agent-caller"
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
				
				// Access parsed results directly
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

// TestMCP_CallTool_ParsedResult_V8 tests that ctx.mcp.CallTool() returns parsed result directly
func TestMCP_CallTool_ParsedResult_V8(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	authorized := &types.AuthorizedInfo{
		Subject:  "test-user",
		UserID:   "test-123",
		TenantID: "test-tenant",
	}

	ctx := context.New(stdContext.Background(), authorized, "test-chat-mcp-calltool-parsed")
	ctx.AssistantID = "tests.agent-caller"
	defer ctx.Release()

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				// Test CallTool returns parsed result directly
				const result = ctx.mcp.CallTool("echo", "echo", { 
					message: "test message", 
					uppercase: true 
				});
				
				// Result should be the parsed data directly
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

	// Handle different integer types from V8
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

// TestMCP_CallToolsParallel_ParsedResult_V8 tests that ctx.mcp.CallToolsParallel() returns parsed results directly
func TestMCP_CallToolsParallel_ParsedResult_V8(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	authorized := &types.AuthorizedInfo{
		Subject:  "test-user",
		UserID:   "test-123",
		TenantID: "test-tenant",
	}

	ctx := context.New(stdContext.Background(), authorized, "test-chat-mcp-calltools-parsed")
	ctx.AssistantID = "tests.agent-caller"
	defer ctx.Release()

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				// Test CallToolsParallel returns parsed results directly as array
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
				
				// Each result is the parsed data directly
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
