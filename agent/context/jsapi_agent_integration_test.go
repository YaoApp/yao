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

	_ "github.com/yaoapp/yao/agent/assistant"
)

func TestAgent_Call_V8(t *testing.T) {
	testprepare.PrepareSandbox(t)

	authorized := &types.AuthorizedInfo{
		Subject:  "test-user",
		UserID:   "test-123",
		TenantID: "test-tenant",
	}

	ctx := agentctx.New(stdContext.Background(), authorized, "test-chat-v8-call")
	ctx.AssistantID = "tests.simple-greeting"
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
					has_response: result.response !== undefined && result.response !== null,
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
	hasResponse, _ := result["has_response"].(bool)
	assert.True(t, hasResponse, "Should have response object")
	errorStr, _ := result["error"].(string)
	assert.Empty(t, errorStr, "Should not have error")
}

func TestAgent_Call_WithOptions_V8(t *testing.T) {
	testprepare.PrepareSandbox(t)

	authorized := &types.AuthorizedInfo{
		Subject:  "test-user",
		UserID:   "test-123",
		TenantID: "test-tenant",
	}

	ctx := agentctx.New(stdContext.Background(), authorized, "test-chat-v8-options")
	ctx.AssistantID = "tests.simple-greeting"
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
	result, ok := res.(map[string]interface{})
	require.True(t, ok, "Result should be a map")

	success, _ := result["success"].(bool)
	if !success {
		t.Fatalf("Test failed: %v", result["error"])
	}

	assert.Equal(t, "tests.simple-greeting", result["agent_id"])
}

func TestAgent_All_V8(t *testing.T) {
	testprepare.PrepareSandbox(t)

	authorized := &types.AuthorizedInfo{
		Subject:  "test-user",
		UserID:   "test-123",
		TenantID: "test-tenant",
	}

	ctx := agentctx.New(stdContext.Background(), authorized, "test-chat-v8-all")
	ctx.AssistantID = "tests.simple-greeting"
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
	result, ok := res.(map[string]interface{})
	require.True(t, ok, "Result should be a map")

	success, _ := result["success"].(bool)
	if !success {
		t.Fatalf("Test failed: %v", result["error"])
	}

	assert.Equal(t, float64(2), result["count"])
	assert.Equal(t, "tests.simple-greeting", result["first_agent"])
	assert.Equal(t, "tests.simple-greeting", result["second_agent"])
}

func TestAgent_Any_V8(t *testing.T) {
	testprepare.PrepareSandbox(t)

	authorized := &types.AuthorizedInfo{
		Subject:  "test-user",
		UserID:   "test-123",
		TenantID: "test-tenant",
	}

	ctx := agentctx.New(stdContext.Background(), authorized, "test-chat-v8-any")
	ctx.AssistantID = "tests.simple-greeting"
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
	result, ok := res.(map[string]interface{})
	require.True(t, ok, "Result should be a map")

	success, _ := result["success"].(bool)
	if !success {
		t.Fatalf("Test failed: %v", result["error"])
	}

	assert.Equal(t, float64(2), result["count"])
}

func TestAgent_Race_V8(t *testing.T) {
	testprepare.PrepareSandbox(t)

	authorized := &types.AuthorizedInfo{
		Subject:  "test-user",
		UserID:   "test-123",
		TenantID: "test-tenant",
	}

	ctx := agentctx.New(stdContext.Background(), authorized, "test-chat-v8-race")
	ctx.AssistantID = "tests.simple-greeting"
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
	result, ok := res.(map[string]interface{})
	require.True(t, ok, "Result should be a map")

	success, _ := result["success"].(bool)
	if !success {
		t.Fatalf("Test failed: %v", result["error"])
	}

	assert.Equal(t, float64(2), result["count"])
}

func TestAgent_ErrorHandling_V8(t *testing.T) {
	testprepare.PrepareSandbox(t)

	authorized := &types.AuthorizedInfo{
		Subject:  "test-user",
		UserID:   "test-123",
		TenantID: "test-tenant",
	}

	ctx := agentctx.New(stdContext.Background(), authorized, "test-chat-v8-error")
	ctx.AssistantID = "tests.simple-greeting"
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
	result, ok := res.(map[string]interface{})
	require.True(t, ok, "Result should be a map")

	success, _ := result["success"].(bool)
	assert.True(t, success, "JS execution should succeed")
	hasError, _ := result["has_error"].(bool)
	assert.True(t, hasError, "Result should have error for non-existent agent")
	errorMsg, _ := result["error_message"].(string)
	assert.Contains(t, errorMsg, "agent", "Error should mention agent")
}

func TestAgent_EmptyRequests_V8(t *testing.T) {
	testprepare.PrepareSandbox(t)

	authorized := &types.AuthorizedInfo{
		Subject:  "test-user",
		UserID:   "test-123",
		TenantID: "test-tenant",
	}

	ctx := agentctx.New(stdContext.Background(), authorized, "test-chat-v8-empty")
	ctx.AssistantID = "tests.simple-greeting"
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
	result, ok := res.(map[string]interface{})
	require.True(t, ok, "Result should be a map")

	success, _ := result["success"].(bool)
	assert.True(t, success)
	assert.Equal(t, float64(0), result["count"])
}

func TestAgent_InvalidArguments_V8(t *testing.T) {
	testprepare.PrepareSandbox(t)

	authorized := &types.AuthorizedInfo{
		Subject:  "test-user",
		UserID:   "test-123",
		TenantID: "test-tenant",
	}

	ctx := agentctx.New(stdContext.Background(), authorized, "test-chat-v8-invalid")
	ctx.AssistantID = "tests.simple-greeting"
	defer ctx.Release()

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				ctx.agent.Call();
				return { success: false, error: "Should have thrown" };
			} catch (error) {
				return { success: true, error: error.message };
			}
		}`, ctx)

	require.NoError(t, err)
	result, ok := res.(map[string]interface{})
	require.True(t, ok, "Result should be a map")
	success, _ := result["success"].(bool)
	assert.True(t, success, "Should catch the error")
	errorStr, _ := result["error"].(string)
	assert.NotEmpty(t, errorStr, "Error message should not be empty")
}

func TestAgent_Call_WithCallback_V8(t *testing.T) {
	testprepare.PrepareSandbox(t)

	authorized := &types.AuthorizedInfo{
		Subject:  "test-user",
		UserID:   "test-123",
		TenantID: "test-tenant",
	}

	ctx := agentctx.New(stdContext.Background(), authorized, "test-chat-v8-callback")
	ctx.AssistantID = "tests.simple-greeting"
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
							messageCount++;
							messages.push({
								type: msg.type,
								has_props: msg.props !== undefined
							});
							return 0;
						}
					}
				);
				
				return { 
					success: true,
					agent_id: result.agent_id,
					has_content: result.content && result.content.length > 0,
					message_count: messageCount,
					received_messages: messages.slice(0, 5),
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

	messageCount, _ := result["message_count"].(float64)
	t.Logf("Received %v messages via callback", messageCount)
	assert.Greater(t, messageCount, float64(0), "Should have received messages via callback")

	receivedMsgs, _ := result["received_messages"].([]interface{})
	if len(receivedMsgs) > 0 {
		firstMsg := receivedMsgs[0].(map[string]interface{})
		t.Logf("First message type: %v", firstMsg["type"])
		assert.NotEmpty(t, firstMsg["type"], "Message should have type")
	}
}

func TestAgent_Call_WithCallback_Stop_V8(t *testing.T) {
	testprepare.PrepareSandbox(t)

	authorized := &types.AuthorizedInfo{
		Subject:  "test-user",
		UserID:   "test-123",
		TenantID: "test-tenant",
	}

	ctx := agentctx.New(stdContext.Background(), authorized, "test-chat-v8-callback-stop")
	ctx.AssistantID = "tests.simple-greeting"
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
							if (messageCount >= 3) {
								return 1;
							}
							return 0;
						}
					}
				);
				
				return { 
					success: true,
					message_count: messageCount,
					stopped_early: messageCount <= 5
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
}

func TestAgent_All_WithGlobalCallback_V8(t *testing.T) {
	testprepare.PrepareSandbox(t)

	authorized := &types.AuthorizedInfo{
		Subject:  "test-user",
		UserID:   "test-123",
		TenantID: "test-tenant",
	}

	ctx := agentctx.New(stdContext.Background(), authorized, "test-chat-v8-all-callback")
	ctx.AssistantID = "tests.simple-greeting"
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

	messagesByAgent, _ := result["messages_by_agent"].(map[string]interface{})
	t.Logf("Messages by agent: %v", messagesByAgent)

	assert.Greater(t, len(messagesByAgent), 0, "Should have received messages from agents")
}
