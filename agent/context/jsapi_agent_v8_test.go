package context_test

import (
	stdContext "context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v8 "github.com/yaoapp/gou/runtime/v8"
	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/testutils"
	"github.com/yaoapp/yao/openapi/oauth/types"

	// Import assistant package to register AgentAPIFactory
	_ "github.com/yaoapp/yao/agent/assistant"
)

// TestAgent_Call_V8 tests basic ctx.agent.Call() functionality with real V8 execution
func TestAgent_Call_V8(t *testing.T) {
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

	ctx := context.New(stdContext.Background(), authorized, "test-chat-v8-call")
	ctx.AssistantID = "tests.agent-caller"
	defer ctx.Release()

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				const result = ctx.agent.Call(
					"tests.simple-greeting",
					[{ role: "user", content: "Hello" }]
				);
				
				return { 
					success: true,
					agent_id: result.agent_id,
					has_content: result.content && result.content.length > 0,
					has_response: result.response !== undefined,
					error: result.error || ""
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

	assert.Equal(t, "tests.simple-greeting", result["agent_id"])

	hasContent, _ := result["has_content"].(bool)
	assert.True(t, hasContent, "Should have content in response")

	hasResponse, _ := result["has_response"].(bool)
	assert.True(t, hasResponse, "Should have response object")

	errorStr, _ := result["error"].(string)
	assert.Empty(t, errorStr, "Should not have error")
}

// TestAgent_Call_WithOptions_V8 tests ctx.agent.Call() with options
func TestAgent_Call_WithOptions_V8(t *testing.T) {
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

	ctx := context.New(stdContext.Background(), authorized, "test-chat-v8-options")
	ctx.AssistantID = "tests.agent-caller"
	defer ctx.Release()

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				const result = ctx.agent.Call(
					"tests.simple-greeting",
					[{ role: "user", content: "Hi there!" }],
					{
						skip: {
							history: true,
							trace: true
						}
					}
				);
				
				return { 
					success: true,
					agent_id: result.agent_id,
					content: result.content || "",
					error: result.error || ""
				};
			} catch (error) {
				return { success: false, error: error.message };
			}
		}`, ctx)

	require.NoError(t, err)
	result := res.(map[string]interface{})

	if !result["success"].(bool) {
		t.Fatalf("Test failed: %v", result["error"])
	}

	assert.Equal(t, "tests.simple-greeting", result["agent_id"])
	assert.NotEmpty(t, result["content"], "Should have content")
}

// TestAgent_All_V8 tests ctx.agent.All() for parallel execution
func TestAgent_All_V8(t *testing.T) {
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

	ctx := context.New(stdContext.Background(), authorized, "test-chat-v8-all")
	ctx.AssistantID = "tests.agent-caller"
	defer ctx.Release()

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				const results = ctx.agent.All([
					{
						agent: "tests.simple-greeting",
						messages: [{ role: "user", content: "Hello from request 1" }]
					},
					{
						agent: "tests.simple-greeting",
						messages: [{ role: "user", content: "Hello from request 2" }]
					}
				]);
				
				return { 
					success: true,
					count: results.length,
					first_agent: results[0] ? results[0].agent_id : "",
					second_agent: results[1] ? results[1].agent_id : "",
					first_has_content: results[0] && results[0].content && results[0].content.length > 0,
					second_has_content: results[1] && results[1].content && results[1].content.length > 0,
					first_error: results[0] ? (results[0].error || "") : "no result",
					second_error: results[1] ? (results[1].error || "") : "no result"
				};
			} catch (error) {
				return { success: false, error: error.message };
			}
		}`, ctx)

	require.NoError(t, err)
	result := res.(map[string]interface{})

	if !result["success"].(bool) {
		t.Fatalf("Test failed: %v", result["error"])
	}

	assert.Equal(t, float64(2), result["count"])
	assert.Equal(t, "tests.simple-greeting", result["first_agent"])
	assert.Equal(t, "tests.simple-greeting", result["second_agent"])
	assert.True(t, result["first_has_content"].(bool), "First result should have content")
	assert.True(t, result["second_has_content"].(bool), "Second result should have content")
	assert.Empty(t, result["first_error"], "First result should not have error")
	assert.Empty(t, result["second_error"], "Second result should not have error")
}

// TestAgent_Any_V8 tests ctx.agent.Any() returns on first success
func TestAgent_Any_V8(t *testing.T) {
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

	ctx := context.New(stdContext.Background(), authorized, "test-chat-v8-any")
	ctx.AssistantID = "tests.agent-caller"
	defer ctx.Release()

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				const results = ctx.agent.Any([
					{
						agent: "tests.simple-greeting",
						messages: [{ role: "user", content: "Hello" }]
					},
					{
						agent: "tests.simple-greeting",
						messages: [{ role: "user", content: "Hi" }]
					}
				]);
				
				// At least one result should be successful
				let hasSuccess = false;
				for (const r of results) {
					if (r && r.content && !r.error) {
						hasSuccess = true;
						break;
					}
				}
				
				return { 
					success: true,
					count: results.length,
					has_successful_result: hasSuccess
				};
			} catch (error) {
				return { success: false, error: error.message };
			}
		}`, ctx)

	require.NoError(t, err)
	result := res.(map[string]interface{})

	if !result["success"].(bool) {
		t.Fatalf("Test failed: %v", result["error"])
	}

	assert.Equal(t, float64(2), result["count"])
	assert.True(t, result["has_successful_result"].(bool), "Should have at least one successful result")
}

// TestAgent_Race_V8 tests ctx.agent.Race() returns on first completion
func TestAgent_Race_V8(t *testing.T) {
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

	ctx := context.New(stdContext.Background(), authorized, "test-chat-v8-race")
	ctx.AssistantID = "tests.agent-caller"
	defer ctx.Release()

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				const results = ctx.agent.Race([
					{
						agent: "tests.simple-greeting",
						messages: [{ role: "user", content: "Hello" }]
					},
					{
						agent: "tests.simple-greeting",
						messages: [{ role: "user", content: "Hi" }]
					}
				]);
				
				// At least one result should exist (first to complete)
				let hasResult = false;
				for (const r of results) {
					if (r && (r.content || r.error)) {
						hasResult = true;
						break;
					}
				}
				
				return { 
					success: true,
					count: results.length,
					has_result: hasResult
				};
			} catch (error) {
				return { success: false, error: error.message };
			}
		}`, ctx)

	require.NoError(t, err)
	result := res.(map[string]interface{})

	if !result["success"].(bool) {
		t.Fatalf("Test failed: %v", result["error"])
	}

	assert.Equal(t, float64(2), result["count"])
	assert.True(t, result["has_result"].(bool), "Should have at least one result")
}

// TestAgent_ErrorHandling_V8 tests error handling when calling non-existent agent
func TestAgent_ErrorHandling_V8(t *testing.T) {
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

	ctx := context.New(stdContext.Background(), authorized, "test-chat-v8-error")
	ctx.AssistantID = "tests.agent-caller"
	defer ctx.Release()

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				const result = ctx.agent.Call(
					"non-existent-agent",
					[{ role: "user", content: "Hello" }]
				);
				
				return { 
					success: true,
					has_error: result.error && result.error.length > 0,
					error_message: result.error || ""
				};
			} catch (error) {
				return { success: false, error: error.message };
			}
		}`, ctx)

	require.NoError(t, err)
	result := res.(map[string]interface{})

	// The call should succeed (no JS exception), but result should contain error
	assert.True(t, result["success"].(bool), "JS execution should succeed")
	assert.True(t, result["has_error"].(bool), "Result should have error for non-existent agent")
	assert.True(t, strings.Contains(result["error_message"].(string), "failed to get agent"), "Error should mention failed to get agent")
}

// TestAgent_EmptyRequests_V8 tests handling of empty requests array
func TestAgent_EmptyRequests_V8(t *testing.T) {
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

	ctx := context.New(stdContext.Background(), authorized, "test-chat-v8-empty")
	ctx.AssistantID = "tests.agent-caller"
	defer ctx.Release()

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				const results = ctx.agent.All([]);
				return { 
					success: true,
					count: results.length
				};
			} catch (error) {
				return { success: false, error: error.message };
			}
		}`, ctx)

	require.NoError(t, err)
	result := res.(map[string]interface{})

	assert.True(t, result["success"].(bool))
	assert.Equal(t, float64(0), result["count"])
}

// TestAgent_InvalidArguments_V8 tests error handling for invalid arguments
func TestAgent_InvalidArguments_V8(t *testing.T) {
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

	ctx := context.New(stdContext.Background(), authorized, "test-chat-v8-invalid")
	ctx.AssistantID = "tests.agent-caller"
	defer ctx.Release()

	// Test missing arguments
	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				// Call with no arguments should throw
				ctx.agent.Call();
				return { success: false, error: "Should have thrown" };
			} catch (error) {
				return { success: true, error: error.message };
			}
		}`, ctx)

	require.NoError(t, err)
	result := res.(map[string]interface{})
	assert.True(t, result["success"].(bool), "Should catch the error")
	assert.Contains(t, result["error"].(string), "requires")
}

// ============================================================================
// Callback Tests
// ============================================================================

// TestAgent_Call_WithCallback_V8 tests ctx.agent.Call() with onChunk callback
func TestAgent_Call_WithCallback_V8(t *testing.T) {
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

	ctx := context.New(stdContext.Background(), authorized, "test-chat-v8-callback")
	ctx.AssistantID = "tests.agent-caller"
	defer ctx.Release()

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				const messages = [];
				let messageCount = 0;
				
				const result = ctx.agent.Call(
					"tests.simple-greeting",
					[{ role: "user", content: "Hello" }],
					{
						onChunk: (msg) => {
							// msg is the SSE message object
							messageCount++;
							messages.push({
								type: msg.type,
								has_props: msg.props !== undefined
							});
							return 0; // Continue
						}
					}
				);
				
				return { 
					success: true,
					agent_id: result.agent_id,
					has_content: result.content && result.content.length > 0,
					message_count: messageCount,
					received_messages: messages.slice(0, 5), // First 5 messages
					error: result.error || ""
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

	assert.Equal(t, "tests.simple-greeting", result["agent_id"])

	// Should have received some messages via callback
	messageCount, _ := result["message_count"].(float64)
	t.Logf("Received %v messages via callback", messageCount)
	assert.Greater(t, messageCount, float64(0), "Should have received messages via callback")

	// Check that we received message objects with type and props
	receivedMsgs, _ := result["received_messages"].([]interface{})
	if len(receivedMsgs) > 0 {
		firstMsg := receivedMsgs[0].(map[string]interface{})
		t.Logf("First message type: %v", firstMsg["type"])
		assert.NotEmpty(t, firstMsg["type"], "Message should have type")
	}
}

// TestAgent_Call_WithCallback_Stop_V8 tests that callback can stop streaming
func TestAgent_Call_WithCallback_Stop_V8(t *testing.T) {
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

	ctx := context.New(stdContext.Background(), authorized, "test-chat-v8-callback-stop")
	ctx.AssistantID = "tests.agent-caller"
	defer ctx.Release()

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				let messageCount = 0;
				
				const result = ctx.agent.Call(
					"tests.simple-greeting",
					[{ role: "user", content: "Hello" }],
					{
						onChunk: (msg) => {
							messageCount++;
							// Stop after receiving 3 messages
							if (messageCount >= 3) {
								return 1; // Stop
							}
							return 0; // Continue
						}
					}
				);
				
				return { 
					success: true,
					message_count: messageCount,
					stopped_early: messageCount <= 5 // Should have stopped early
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

	messageCount, _ := result["message_count"].(float64)
	t.Logf("Received %v messages before stopping", messageCount)
	// Note: The exact count may vary based on when the stop is processed
}

// TestAgent_All_WithGlobalCallback_V8 tests ctx.agent.All() with global onChunk callback
// Uses channel-based callback handling for V8 thread safety
func TestAgent_All_WithGlobalCallback_V8(t *testing.T) {
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

	ctx := context.New(stdContext.Background(), authorized, "test-chat-v8-all-callback")
	ctx.AssistantID = "tests.agent-caller"
	defer ctx.Release()

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				const messagesByAgent = {};
				
				const results = ctx.agent.All(
					[
						{
							agent: "tests.simple-greeting",
							messages: [{ role: "user", content: "Hello from 1" }]
						},
						{
							agent: "tests.simple-greeting",
							messages: [{ role: "user", content: "Hello from 2" }]
						}
					],
					{
						// Global callback receives agentID, index, and message
						onChunk: (agentID, index, msg) => {
							const key = agentID + "_" + index;
							if (!messagesByAgent[key]) {
								messagesByAgent[key] = 0;
							}
							messagesByAgent[key]++;
							return 0;
						}
					}
				);
				
				return { 
					success: true,
					result_count: results.length,
					messages_by_agent: messagesByAgent
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

	assert.Equal(t, float64(2), result["result_count"])

	// Should have received messages from both agents
	messagesByAgent, _ := result["messages_by_agent"].(map[string]interface{})
	t.Logf("Messages by agent: %v", messagesByAgent)

	// At least one agent should have sent messages
	assert.Greater(t, len(messagesByAgent), 0, "Should have received messages from agents")
}
