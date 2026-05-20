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

func TestLlm_Stream_V8(t *testing.T) {
	testprepare.PrepareSandbox(t)

	authorized := &types.AuthorizedInfo{
		Subject:  "test-user",
		UserID:   "test-123",
		TenantID: "test-tenant",
	}

	ctx := agentctx.New(stdContext.Background(), authorized, "test-chat-llm-stream")
	ctx.AssistantID = "tests.simple-greeting"
	defer ctx.Release()

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
			const result = ctx.llm.Stream("openai.mock", [
				{ role: "user", content: "Say hello in one word" }
			], {
				temperature: 0.1,
				max_tokens: 10
			});
				
				return {
					success: true,
					connector: result.connector,
					has_content: result.content && result.content.length > 0,
					has_response: result.response !== undefined,
					error: result.error || ""
				};
			} catch (error) {
				return { success: false, error: error.message };
			}
		}`, ctx)

	require.NoError(t, err)
	require.NotNil(t, res)

	result, ok := res.(map[string]interface{})
	require.True(t, ok, "result should be a map")

	success, _ := result["success"].(bool)
	if !success {
		t.Logf("Test result: %v", result)
	}
	require.True(t, success, "Test should succeed, error: %v", result["error"])

	assert.Equal(t, "openai.mock", result["connector"])

	hasContent, _ := result["has_content"].(bool)
	assert.True(t, hasContent, "Should have content in response")

	hasResponse, _ := result["has_response"].(bool)
	assert.True(t, hasResponse, "Should have response object")
}

func TestLlm_Stream_WithCallback_V8(t *testing.T) {
	testprepare.PrepareSandbox(t)

	authorized := &types.AuthorizedInfo{
		Subject:  "test-user",
		UserID:   "test-123",
		TenantID: "test-tenant",
	}

	ctx := agentctx.New(stdContext.Background(), authorized, "test-chat-llm-callback")
	ctx.AssistantID = "tests.simple-greeting"
	defer ctx.Release()

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				let callbackCount = 0;
				let receivedTypes = [];
				
			const result = ctx.llm.Stream("openai.mock", [
				{ role: "user", content: "Say hi" }
			], {
				temperature: 0.1,
				max_tokens: 10,
				onChunk: function(msg) {
						callbackCount++;
						if (msg && msg.type) {
							receivedTypes.push(msg.type);
						}
						return 0;
					}
				});
				
				return {
					success: true,
					connector: result.connector,
					callbackCount: callbackCount,
					receivedTypes: receivedTypes,
					has_content: result.content && result.content.length > 0,
					error: result.error || ""
				};
			} catch (error) {
				return { success: false, error: error.message };
			}
		}`, ctx)

	require.NoError(t, err)
	require.NotNil(t, res)

	result, ok := res.(map[string]interface{})
	require.True(t, ok, "result should be a map")

	success, _ := result["success"].(bool)
	if !success {
		t.Logf("Test result: %v", result)
	}
	require.True(t, success, "Test should succeed, error: %v", result["error"])

	callbackCount, _ := result["callbackCount"].(float64)
	assert.Greater(t, callbackCount, float64(0), "Callback should be called at least once")
}

func TestLlm_All_V8(t *testing.T) {
	testprepare.PrepareSandbox(t)

	authorized := &types.AuthorizedInfo{
		Subject:  "test-user",
		UserID:   "test-123",
		TenantID: "test-tenant",
	}

	ctx := agentctx.New(stdContext.Background(), authorized, "test-chat-llm-all")
	ctx.AssistantID = "tests.simple-greeting"
	defer ctx.Release()

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				const results = ctx.llm.All([
				{
					connector: "openai.mock",
					messages: [{ role: "user", content: "Say 'one'" }],
					options: { temperature: 0.1, max_tokens: 5 }
				},
				{
					connector: "openai.mock",
					messages: [{ role: "user", content: "Say 'two'" }],
					options: { temperature: 0.1, max_tokens: 5 }
				}
			]);
				
				return {
					success: true,
					count: results.length,
					results: results.map(r => ({
						connector: r.connector,
						has_content: r.content && r.content.length > 0,
						error: r.error || ""
					}))
				};
			} catch (error) {
				return { success: false, error: error.message };
			}
		}`, ctx)

	require.NoError(t, err)
	require.NotNil(t, res)

	result, ok := res.(map[string]interface{})
	require.True(t, ok, "result should be a map")

	success, _ := result["success"].(bool)
	if !success {
		t.Logf("Test result: %v", result)
	}
	require.True(t, success, "Test should succeed, error: %v", result["error"])

	count, _ := result["count"].(float64)
	assert.Equal(t, float64(2), count, "Should have 2 results")

	results, _ := result["results"].([]interface{})
	require.Len(t, results, 2)

	for i, r := range results {
		rMap, _ := r.(map[string]interface{})
		hasContent, _ := rMap["has_content"].(bool)
		assert.True(t, hasContent, "Result %d should have content", i)
		errorStr, _ := rMap["error"].(string)
		assert.Empty(t, errorStr, "Result %d should not have error", i)
	}
}

func TestLlm_All_WithCallback_V8(t *testing.T) {
	testprepare.PrepareSandbox(t)

	authorized := &types.AuthorizedInfo{
		Subject:  "test-user",
		UserID:   "test-123",
		TenantID: "test-tenant",
	}

	ctx := agentctx.New(stdContext.Background(), authorized, "test-chat-llm-all-callback")
	ctx.AssistantID = "tests.simple-greeting"
	defer ctx.Release()

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				let callbackCount = 0;
				let indexesSeen = new Set();
				
			const results = ctx.llm.All([
				{
					connector: "openai.mock",
					messages: [{ role: "user", content: "Say 'A'" }],
					options: { temperature: 0.1, max_tokens: 5 }
				},
				{
					connector: "openai.mock",
					messages: [{ role: "user", content: "Say 'B'" }],
					options: { temperature: 0.1, max_tokens: 5 }
				}
			], {
					onChunk: function(connectorID, index, msg) {
						callbackCount++;
						indexesSeen.add(index);
						return 0;
					}
				});
				
				return {
					success: true,
					count: results.length,
					callbackCount: callbackCount,
					indexesSeen: indexesSeen.size
				};
			} catch (error) {
				return { success: false, error: error.message };
			}
		}`, ctx)

	require.NoError(t, err)
	require.NotNil(t, res)

	result, ok := res.(map[string]interface{})
	require.True(t, ok, "result should be a map")

	success, _ := result["success"].(bool)
	if !success {
		t.Logf("Test result: %v", result)
	}
	require.True(t, success, "Test should succeed, error: %v", result["error"])

	callbackCount, _ := result["callbackCount"].(float64)
	assert.Greater(t, callbackCount, float64(0), "Callback should be called")

	indexesSeen, _ := result["indexesSeen"].(float64)
	assert.GreaterOrEqual(t, indexesSeen, float64(1), "Should have seen callbacks from at least one request")
}

func TestLlm_Any_V8(t *testing.T) {
	testprepare.PrepareSandbox(t)

	authorized := &types.AuthorizedInfo{
		Subject:  "test-user",
		UserID:   "test-123",
		TenantID: "test-tenant",
	}

	ctx := agentctx.New(stdContext.Background(), authorized, "test-chat-llm-any")
	ctx.AssistantID = "tests.simple-greeting"
	defer ctx.Release()

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
			const results = ctx.llm.Any([
				{
					connector: "openai.mock",
					messages: [{ role: "user", content: "Say 'hello'" }],
					options: { temperature: 0.1, max_tokens: 5 }
				},
				{
					connector: "openai.mock",
					messages: [{ role: "user", content: "Say 'world'" }],
					options: { temperature: 0.1, max_tokens: 5 }
				}
			]);
				
				return {
					success: true,
					count: results.length,
					first_has_content: results[0] && results[0].content && results[0].content.length > 0,
					first_error: results[0] ? (results[0].error || "") : "no result"
				};
			} catch (error) {
				return { success: false, error: error.message };
			}
		}`, ctx)

	require.NoError(t, err)
	require.NotNil(t, res)

	result, ok := res.(map[string]interface{})
	require.True(t, ok, "result should be a map")

	success, _ := result["success"].(bool)
	if !success {
		t.Logf("Test result: %v", result)
	}
	require.True(t, success, "Test should succeed, error: %v", result["error"])

	count, _ := result["count"].(float64)
	assert.Equal(t, float64(1), count, "Should have 1 result (first success)")

	firstHasContent, _ := result["first_has_content"].(bool)
	assert.True(t, firstHasContent, "First result should have content")

	firstError, _ := result["first_error"].(string)
	assert.Empty(t, firstError, "First result should not have error")
}

func TestLlm_Race_V8(t *testing.T) {
	testprepare.PrepareSandbox(t)

	authorized := &types.AuthorizedInfo{
		Subject:  "test-user",
		UserID:   "test-123",
		TenantID: "test-tenant",
	}

	ctx := agentctx.New(stdContext.Background(), authorized, "test-chat-llm-race")
	ctx.AssistantID = "tests.simple-greeting"
	defer ctx.Release()

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
			const results = ctx.llm.Race([
				{
					connector: "openai.mock",
					messages: [{ role: "user", content: "Say 'fast'" }],
					options: { temperature: 0.1, max_tokens: 5 }
				},
				{
					connector: "openai.mock",
					messages: [{ role: "user", content: "Say 'slow'" }],
					options: { temperature: 0.1, max_tokens: 5 }
				}
			]);
				
				return {
					success: true,
					count: results.length,
					has_result: results[0] !== undefined,
					first_connector: results[0] ? results[0].connector : "",
					first_has_content: results[0] && results[0].content && results[0].content.length > 0
				};
			} catch (error) {
				return { success: false, error: error.message };
			}
		}`, ctx)

	require.NoError(t, err)
	require.NotNil(t, res)

	result, ok := res.(map[string]interface{})
	require.True(t, ok, "result should be a map")

	success, _ := result["success"].(bool)
	if !success {
		t.Logf("Test result: %v", result)
	}
	require.True(t, success, "Test should succeed, error: %v", result["error"])

	count, _ := result["count"].(float64)
	assert.Equal(t, float64(1), count, "Should have 1 result (first to complete)")

	hasResult, _ := result["has_result"].(bool)
	assert.True(t, hasResult, "Should have a result")

	firstConnector, _ := result["first_connector"].(string)
	assert.Equal(t, "openai.mock", firstConnector, "First result should have connector")
}

func TestLlm_Stream_InvalidConnector_V8(t *testing.T) {
	testprepare.PrepareSandbox(t)

	authorized := &types.AuthorizedInfo{
		Subject:  "test-user",
		UserID:   "test-123",
		TenantID: "test-tenant",
	}

	ctx := agentctx.New(stdContext.Background(), authorized, "test-chat-llm-invalid")
	ctx.AssistantID = "tests.simple-greeting"
	defer ctx.Release()

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				const result = ctx.llm.Stream("invalid-connector-that-does-not-exist", [
					{ role: "user", content: "Hello" }
				]);
				
				return {
					has_error: result.error && result.error.length > 0,
					error: result.error || ""
				};
			} catch (error) {
				return { has_error: true, error: error.message };
			}
		}`, ctx)

	require.NoError(t, err)
	require.NotNil(t, res)

	result, ok := res.(map[string]interface{})
	require.True(t, ok, "result should be a map")

	hasError, _ := result["has_error"].(bool)
	assert.True(t, hasError, "Should have error for invalid connector")
}
