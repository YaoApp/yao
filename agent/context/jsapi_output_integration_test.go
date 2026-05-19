//go:build integration

package context_test

import (
	"bytes"
	stdContext "context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v8 "github.com/yaoapp/gou/runtime/v8"
	agentctx "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

type testMockResponseWriter struct {
	headers http.Header
	buffer  *bytes.Buffer
	status  int
}

func newTestMockResponseWriter() *testMockResponseWriter {
	return &testMockResponseWriter{
		headers: make(http.Header),
		buffer:  &bytes.Buffer{},
		status:  http.StatusOK,
	}
}

func (m *testMockResponseWriter) Header() http.Header {
	return m.headers
}

func (m *testMockResponseWriter) Write(b []byte) (int, error) {
	return m.buffer.Write(b)
}

func (m *testMockResponseWriter) WriteHeader(statusCode int) {
	m.status = statusCode
}

func TestJsValueSend(t *testing.T) {
	testprepare.PrepareSandbox(t)

	cxt := agentctx.New(stdContext.Background(), nil, "test-chat-id")
	cxt.AssistantID = "test-assistant-id"
	cxt.Accept = agentctx.AcceptStandard
	cxt.Locale = "en"
	cxt.Writer = newTestMockResponseWriter()

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				ctx.Send("Hello World");
				return { success: true };
			} catch (error) {
				return { success: false, error: error.message };
			}
		}`, cxt)
	require.NoError(t, err, "Call failed")

	result, ok := res.(map[string]interface{})
	require.True(t, ok, "Expected map result, got %T", res)
	assert.Equal(t, true, result["success"], "Send string should succeed")

	res, err = v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				ctx.Send({
					type: "text",
					props: {
						content: "Hello from JavaScript"
					},
					id: "msg_123",
					metadata: {
						timestamp: Date.now(),
						sequence: 1
					}
				});
				return { success: true };
			} catch (error) {
				return { success: false, error: error.message };
			}
		}`, cxt)
	require.NoError(t, err, "Call failed")

	result, ok = res.(map[string]interface{})
	require.True(t, ok, "Expected map result, got %T", res)
	assert.Equal(t, true, result["success"], "Send message object should succeed")
}

func TestJsValueSendDeltaUpdates(t *testing.T) {
	testprepare.PrepareSandbox(t)

	cxt := agentctx.New(stdContext.Background(), nil, "test-chat-id")
	cxt.AssistantID = "test-assistant-id"
	cxt.Accept = agentctx.AcceptStandard
	cxt.Locale = "en"
	cxt.Writer = newTestMockResponseWriter()

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				ctx.Send({
					type: "text",
					props: { content: "Hello" },
					id: "msg_1",
					delta: false
				});
				
				ctx.Send({
					type: "text",
					props: { content: " World" },
					id: "msg_1",
					delta: true,
					delta_path: "content",
					delta_action: "append"
				});
				
				return { success: true };
			} catch (error) {
				return { success: false, error: error.message };
			}
		}`, cxt)
	require.NoError(t, err, "Call failed")

	result, ok := res.(map[string]interface{})
	require.True(t, ok, "Expected map result, got %T", res)
	assert.Equal(t, true, result["success"], "Delta updates should succeed")
}

func TestJsValueSendMultipleTypes(t *testing.T) {
	testprepare.PrepareSandbox(t)

	cxt := agentctx.New(stdContext.Background(), nil, "test-chat-id")
	cxt.AssistantID = "test-assistant-id"
	cxt.Accept = agentctx.AcceptStandard
	cxt.Locale = "en"
	cxt.Writer = newTestMockResponseWriter()

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				ctx.Send({ type: "text", props: { content: "Hello" } });
				ctx.Send({ type: "thinking", props: { content: "Let me think..." } });
				ctx.Send({ type: "loading", props: { message: "Processing..." } });
				ctx.Send({
					type: "tool_call",
					props: {
						id: "call_123",
						name: "get_weather",
						arguments: '{"location": "San Francisco"}'
					}
				});
				ctx.Send({
					type: "error",
					props: { message: "Something went wrong", code: "ERR_500" }
				});
				ctx.Send({
					type: "image",
					props: {
						url: "https://example.com/image.jpg",
						alt: "Example image",
						width: 800,
						height: 600
					}
				});
				
				return { success: true };
			} catch (error) {
				return { success: false, error: error.message };
			}
		}`, cxt)
	require.NoError(t, err, "Call failed")

	result, ok := res.(map[string]interface{})
	require.True(t, ok, "Expected map result, got %T", res)
	assert.Equal(t, true, result["success"], "Multiple message types should succeed")
}

func TestJsValueSendErrorHandling(t *testing.T) {
	testprepare.PrepareSandbox(t)

	cxt := agentctx.New(stdContext.Background(), nil, "test-chat-id")
	cxt.AssistantID = "test-assistant-id"
	cxt.Accept = agentctx.AcceptStandard
	cxt.Locale = "en"
	cxt.Writer = newTestMockResponseWriter()

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				ctx.Send();
				return { success: true };
			} catch (error) {
				return { success: false, error: error.message };
			}
		}`, cxt)
	require.NoError(t, err, "Call failed")

	result, ok := res.(map[string]interface{})
	require.True(t, ok, "Expected map result, got %T", res)
	assert.Equal(t, false, result["success"], "Send without arguments should fail")
	assert.Contains(t, result["error"], "Send requires a message argument", "Error should mention missing message")
}

func TestJsValueSendWithCUIAccept(t *testing.T) {
	testprepare.PrepareSandbox(t)

	acceptTypes := []agentctx.Accept{agentctx.AcceptWebCUI, agentctx.AccepNativeCUI, agentctx.AcceptDesktopCUI}

	for _, acceptType := range acceptTypes {
		t.Run(string(acceptType), func(t *testing.T) {
			cxt := agentctx.New(stdContext.Background(), nil, "test-chat-id")
			cxt.AssistantID = "test-assistant-id"
			cxt.Accept = acceptType
			cxt.Locale = "en"
			cxt.Writer = newTestMockResponseWriter()

			res, err := v8.Call(v8.CallOptions{}, `
				function test(ctx) {
					try {
						ctx.Send({ type: "text", props: { content: "Hello CUI" } });
						return { success: true };
					} catch (error) {
						return { success: false, error: error.message };
					}
				}`, cxt)
			require.NoError(t, err, "Call failed")

			result, ok := res.(map[string]interface{})
			require.True(t, ok, "Expected map result, got %T", res)
			assert.Equal(t, true, result["success"], "Send with "+string(acceptType)+" should succeed")
		})
	}
}

func TestJsValueSendChainedCalls(t *testing.T) {
	testprepare.PrepareSandbox(t)

	cxt := agentctx.New(stdContext.Background(), nil, "test-chat-id")
	cxt.AssistantID = "test-assistant-id"
	cxt.Accept = agentctx.AcceptStandard
	cxt.Locale = "en"
	cxt.Writer = newTestMockResponseWriter()

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				ctx.Send("Step 1");
				ctx.Send("Step 2");
				ctx.Send("Step 3");
				ctx.Send("Step 4");
				
				return { success: true };
			} catch (error) {
				return { success: false, error: error.message };
			}
		}`, cxt)
	require.NoError(t, err, "Call failed")

	result, ok := res.(map[string]interface{})
	require.True(t, ok, "Expected map result, got %T", res)
	assert.Equal(t, true, result["success"], "Chained Send calls should succeed")
}

func TestJsValueIDGenerators(t *testing.T) {
	testprepare.PrepareSandbox(t)

	cxt := agentctx.New(stdContext.Background(), nil, "test-chat-id")
	cxt.AssistantID = "test-assistant-id"
	cxt.Accept = agentctx.AcceptStandard
	cxt.Locale = "en"
	cxt.Writer = newTestMockResponseWriter()

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				const msgId1 = ctx.MessageID();
				const msgId2 = ctx.MessageID();
				const blockId1 = ctx.BlockID();
				const blockId2 = ctx.BlockID();
				const threadId1 = ctx.ThreadID();
				const threadId2 = ctx.ThreadID();
				
				if (typeof msgId1 !== 'string' || typeof msgId2 !== 'string') {
					throw new Error('MessageID should return string');
				}
				if (typeof blockId1 !== 'string' || typeof blockId2 !== 'string') {
					throw new Error('BlockID should return string');
				}
				if (typeof threadId1 !== 'string' || typeof threadId2 !== 'string') {
					throw new Error('ThreadID should return string');
				}
				
				if (!msgId1.startsWith('M') || !msgId2.startsWith('M')) {
					throw new Error('MessageID should start with M');
				}
				if (!blockId1.startsWith('B') || !blockId2.startsWith('B')) {
					throw new Error('BlockID should start with B');
				}
				if (!threadId1.startsWith('T') || !threadId2.startsWith('T')) {
					throw new Error('ThreadID should start with T');
				}
				
				return { 
					success: true,
					msgId1: msgId1,
					msgId2: msgId2,
					blockId1: blockId1,
					blockId2: blockId2,
					threadId1: threadId1,
					threadId2: threadId2
				};
			} catch (error) {
				return { success: false, error: error.message };
			}
		}`, cxt)
	require.NoError(t, err, "Call failed")

	result, ok := res.(map[string]interface{})
	require.True(t, ok, "Expected map result, got %T", res)
	assert.Equal(t, true, result["success"], "ID generators should succeed")
}

func TestJsValueReplace(t *testing.T) {
	testprepare.PrepareSandbox(t)

	cxt := agentctx.New(stdContext.Background(), nil, "test-chat-id")
	cxt.AssistantID = "test-assistant-id"
	cxt.Accept = agentctx.AcceptStandard
	cxt.Locale = "en"
	cxt.Writer = newTestMockResponseWriter()

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				const msgId = ctx.Send("Initial content");
				ctx.Replace(msgId, "Updated content");
				ctx.Replace(msgId, {
					type: "text",
					props: { content: "Final content" }
				});
				
				return { success: true, msgId: msgId };
			} catch (error) {
				return { success: false, error: error.message };
			}
		}`, cxt)
	require.NoError(t, err, "Call failed")

	result, ok := res.(map[string]interface{})
	require.True(t, ok, "Expected map result, got %T", res)
	assert.Equal(t, true, result["success"], "Replace should succeed")
}

func TestJsValueAppend(t *testing.T) {
	testprepare.PrepareSandbox(t)

	cxt := agentctx.New(stdContext.Background(), nil, "test-chat-id")
	cxt.AssistantID = "test-assistant-id"
	cxt.Accept = agentctx.AcceptStandard
	cxt.Locale = "en"
	cxt.Writer = newTestMockResponseWriter()

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				const msgId = ctx.Send("Hello");
				ctx.Append(msgId, " World");
				ctx.Append(msgId, "!");
				
				const msgId2 = ctx.Send({
					type: "data",
					props: { content: "Line 1\n" }
				});
				ctx.Append(msgId2, "Line 2\n", "props.content");
				ctx.Append(msgId2, "Line 3\n", "props.content");
				
				return { success: true, msgId: msgId, msgId2: msgId2 };
			} catch (error) {
				return { success: false, error: error.message };
			}
		}`, cxt)
	require.NoError(t, err, "Call failed")

	result, ok := res.(map[string]interface{})
	require.True(t, ok, "Expected map result, got %T", res)
	assert.Equal(t, true, result["success"], "Append should succeed")
}

func TestJsValueMerge(t *testing.T) {
	testprepare.PrepareSandbox(t)

	cxt := agentctx.New(stdContext.Background(), nil, "test-chat-id")
	cxt.AssistantID = "test-assistant-id"
	cxt.Accept = agentctx.AcceptStandard
	cxt.Locale = "en"
	cxt.Writer = newTestMockResponseWriter()

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				const msgId = ctx.Send({
					type: "status",
					props: { status: "running", progress: 0, started: true }
				});
				
				ctx.Merge(msgId, {
					type: "status",
					props: { progress: 50 }
				}, "props");
				ctx.Merge(msgId, {
					type: "status",
					props: { progress: 100, status: "completed" }
				}, "props");
				
				return { success: true, msgId: msgId };
			} catch (error) {
				return { success: false, error: error.message };
			}
		}`, cxt)
	require.NoError(t, err, "Call failed")

	result, ok := res.(map[string]interface{})
	require.True(t, ok, "Expected map result, got %T", res)
	if !result["success"].(bool) {
		t.Logf("Error: %v", result["error"])
	}
	assert.Equal(t, true, result["success"], "Merge should succeed")
}

func TestJsValueSet(t *testing.T) {
	testprepare.PrepareSandbox(t)

	cxt := agentctx.New(stdContext.Background(), nil, "test-chat-id")
	cxt.AssistantID = "test-assistant-id"
	cxt.Accept = agentctx.AcceptStandard
	cxt.Locale = "en"
	cxt.Writer = newTestMockResponseWriter()

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				const msgId = ctx.Send({
					type: "result",
					props: { content: "Initial" }
				});
				
				ctx.Set(msgId, {
					type: "result",
					props: { status: "success" }
				}, "props.status");
				ctx.Set(msgId, {
					type: "result",
					props: { timestamp: Date.now() }
				}, "props.timestamp");
				ctx.Set(msgId, {
					type: "result",
					props: { metadata: { duration: 1500 } }
				}, "props.metadata");
				
				return { success: true, msgId: msgId };
			} catch (error) {
				return { success: false, error: error.message };
			}
		}`, cxt)
	require.NoError(t, err, "Call failed")

	result, ok := res.(map[string]interface{})
	require.True(t, ok, "Expected map result, got %T", res)
	if !result["success"].(bool) {
		t.Logf("Error: %v", result["error"])
	}
	assert.Equal(t, true, result["success"], "Set should succeed")
}

func TestJsValueBlockIDInheritance(t *testing.T) {
	testprepare.PrepareSandbox(t)

	cxt := agentctx.New(stdContext.Background(), nil, "test-chat-id")
	cxt.AssistantID = "test-assistant-id"
	cxt.Accept = agentctx.AcceptStandard
	cxt.Locale = "en"
	cxt.Writer = newTestMockResponseWriter()

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				const blockId = ctx.BlockID();
				const msgId = ctx.Send("Initial message", blockId);
				
				ctx.Append(msgId, " appended");
				ctx.Replace(msgId, "Replaced message");
				ctx.Merge(msgId, {
					type: "text",
					props: { status: "done" }
				}, "props");
				ctx.Set(msgId, {
					type: "text",
					props: { state: "final" }
				}, "props.state");
				
				return { success: true, msgId: msgId, blockId: blockId };
			} catch (error) {
				return { success: false, error: error.message };
			}
		}`, cxt)
	require.NoError(t, err, "Call failed")

	result, ok := res.(map[string]interface{})
	require.True(t, ok, "Expected map result, got %T", res)
	if !result["success"].(bool) {
		t.Logf("Error: %v", result["error"])
	}
	assert.Equal(t, true, result["success"], "Delta operations should inherit block_id")
}

func TestJsValueEndBlock(t *testing.T) {
	testprepare.PrepareSandbox(t)

	mockWriter := newTestMockResponseWriter()

	cxt := agentctx.New(stdContext.Background(), nil, "test-chat-id")
	cxt.AssistantID = "test-assistant-id"
	cxt.Accept = agentctx.AcceptWebCUI
	cxt.Locale = "en"
	cxt.Writer = mockWriter

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				const block_id = ctx.BlockID();
				
				ctx.Send("Message 1", block_id);
				ctx.Send("Message 2", block_id);
				ctx.Send("Message 3", block_id);
				
				ctx.EndBlock(block_id);
				
				return { success: true, block_id: block_id };
			} catch (error) {
				return { success: false, error: error.message };
			}
		}`, cxt)
	require.NoError(t, err, "Call failed")

	result, ok := res.(map[string]interface{})
	require.True(t, ok, "Expected map result, got %T", res)
	if !result["success"].(bool) {
		t.Logf("Error: %v", result["error"])
	}
	assert.Equal(t, true, result["success"], "EndBlock should work correctly")

	cxt.CloseSafeWriter()

	output := mockWriter.buffer.String()
	assert.Contains(t, output, "block_end", "Output should contain block_end event")
}

func TestJsValueSendStream(t *testing.T) {
	testprepare.PrepareSandbox(t)

	mockWriter := newTestMockResponseWriter()

	cxt := agentctx.New(stdContext.Background(), nil, "test-chat-id")
	cxt.AssistantID = "test-assistant-id"
	cxt.Accept = agentctx.AcceptWebCUI
	cxt.Locale = "en"
	cxt.Writer = mockWriter

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				const msgId = ctx.SendStream({
					type: "text",
					props: { content: "Initial content" }
				});
				
				if (typeof msgId !== 'string' || msgId === '') {
					throw new Error('SendStream should return a message ID');
				}
				
				return { success: true, msgId: msgId };
			} catch (error) {
				return { success: false, error: error.message };
			}
		}`, cxt)
	require.NoError(t, err, "Call failed")

	result, ok := res.(map[string]interface{})
	require.True(t, ok, "Expected map result, got %T", res)
	if !result["success"].(bool) {
		t.Logf("Error: %v", result["error"])
	}
	assert.Equal(t, true, result["success"], "SendStream should work correctly")

	cxt.CloseSafeWriter()

	output := mockWriter.buffer.String()
	assert.Contains(t, output, "message_start", "Output should contain message_start event")
	assert.NotContains(t, output, "message_end", "Output should NOT contain message_end event (streaming)")
}

func TestJsValueSendStreamWithBlockID(t *testing.T) {
	testprepare.PrepareSandbox(t)

	mockWriter := newTestMockResponseWriter()

	cxt := agentctx.New(stdContext.Background(), nil, "test-chat-id")
	cxt.AssistantID = "test-assistant-id"
	cxt.Accept = agentctx.AcceptWebCUI
	cxt.Locale = "en"
	cxt.Writer = mockWriter

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				const blockId = ctx.BlockID();
				const msgId = ctx.SendStream({
					type: "text",
					props: { content: "Streaming with block" },
					block_id: blockId
				});
				
				return { success: true, msgId: msgId, blockId: blockId };
			} catch (error) {
				return { success: false, error: error.message };
			}
		}`, cxt)
	require.NoError(t, err, "Call failed")

	result, ok := res.(map[string]interface{})
	require.True(t, ok, "Expected map result, got %T", res)
	assert.Equal(t, true, result["success"], "SendStream with blockId should succeed")

	cxt.CloseSafeWriter()

	output := mockWriter.buffer.String()
	assert.Contains(t, output, "block_start", "Output should contain block_start event")
}

func TestJsValueEnd(t *testing.T) {
	testprepare.PrepareSandbox(t)

	mockWriter := newTestMockResponseWriter()

	cxt := agentctx.New(stdContext.Background(), nil, "test-chat-id")
	cxt.AssistantID = "test-assistant-id"
	cxt.Accept = agentctx.AcceptWebCUI
	cxt.Locale = "en"
	cxt.Writer = mockWriter

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				const msgId = ctx.SendStream({
					type: "text",
					props: { content: "Hello" }
				});
				ctx.End(msgId);
				
				return { success: true, msgId: msgId };
			} catch (error) {
				return { success: false, error: error.message };
			}
		}`, cxt)
	require.NoError(t, err, "Call failed")

	result, ok := res.(map[string]interface{})
	require.True(t, ok, "Expected map result, got %T", res)
	if !result["success"].(bool) {
		t.Logf("Error: %v", result["error"])
	}
	assert.Equal(t, true, result["success"], "End should work correctly")

	cxt.CloseSafeWriter()

	output := mockWriter.buffer.String()
	assert.Contains(t, output, "message_end", "Output should contain message_end event after End()")
}

func TestJsValueEndWithFinalContent(t *testing.T) {
	testprepare.PrepareSandbox(t)

	mockWriter := newTestMockResponseWriter()

	cxt := agentctx.New(stdContext.Background(), nil, "test-chat-id")
	cxt.AssistantID = "test-assistant-id"
	cxt.Accept = agentctx.AcceptWebCUI
	cxt.Locale = "en"
	cxt.Writer = mockWriter

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				const msgId = ctx.SendStream({
					type: "text",
					props: { content: "Start" }
				});
				ctx.End(msgId, " End");
				
				return { success: true, msgId: msgId };
			} catch (error) {
				return { success: false, error: error.message };
			}
		}`, cxt)
	require.NoError(t, err, "Call failed")

	result, ok := res.(map[string]interface{})
	require.True(t, ok, "Expected map result, got %T", res)
	if !result["success"].(bool) {
		t.Logf("Error: %v", result["error"])
	}
	assert.Equal(t, true, result["success"], "End with final content should work correctly")

	cxt.CloseSafeWriter()

	output := mockWriter.buffer.String()
	assert.Contains(t, output, "message_end", "Output should contain message_end event")
}

func TestJsValueStreamingWorkflow(t *testing.T) {
	testprepare.PrepareSandbox(t)

	mockWriter := newTestMockResponseWriter()

	cxt := agentctx.New(stdContext.Background(), nil, "test-chat-id")
	cxt.AssistantID = "test-assistant-id"
	cxt.Accept = agentctx.AcceptWebCUI
	cxt.Locale = "en"
	cxt.Writer = mockWriter

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				const msgId = ctx.SendStream({
					type: "text",
					props: { content: "# Title\n\n" }
				});
				
				ctx.Append(msgId, "First paragraph. ");
				ctx.Append(msgId, "Second sentence. ");
				ctx.Append(msgId, "Third sentence.\n\n");
				ctx.Append(msgId, "Second paragraph.");
				
				ctx.End(msgId);
				
				return { success: true, msgId: msgId };
			} catch (error) {
				return { success: false, error: error.message };
			}
		}`, cxt)
	require.NoError(t, err, "Call failed")

	result, ok := res.(map[string]interface{})
	require.True(t, ok, "Expected map result, got %T", res)
	if !result["success"].(bool) {
		t.Logf("Error: %v", result["error"])
	}
	assert.Equal(t, true, result["success"], "Streaming workflow should work correctly")

	cxt.CloseSafeWriter()

	output := mockWriter.buffer.String()
	assert.Contains(t, output, "message_start", "Output should contain message_start")
	assert.Contains(t, output, "message_end", "Output should contain message_end")
	assert.Contains(t, output, "# Title", "Output should contain initial content")
	assert.Contains(t, output, "First paragraph", "Output should contain appended content")
}

func TestJsValueSendStreamStringShorthand(t *testing.T) {
	testprepare.PrepareSandbox(t)

	mockWriter := newTestMockResponseWriter()

	cxt := agentctx.New(stdContext.Background(), nil, "test-chat-id")
	cxt.AssistantID = "test-assistant-id"
	cxt.Accept = agentctx.AcceptWebCUI
	cxt.Locale = "en"
	cxt.Writer = mockWriter

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				const msgId = ctx.SendStream("Hello streaming");
				
				if (typeof msgId !== 'string' || msgId === '') {
					throw new Error('SendStream should return a message ID');
				}
				
				ctx.End(msgId);
				
				return { success: true, msgId: msgId };
			} catch (error) {
				return { success: false, error: error.message };
			}
		}`, cxt)
	require.NoError(t, err, "Call failed")

	result, ok := res.(map[string]interface{})
	require.True(t, ok, "Expected map result, got %T", res)
	assert.Equal(t, true, result["success"], "SendStream with string shorthand should succeed")
}

func TestJsValueEndErrorHandling(t *testing.T) {
	testprepare.PrepareSandbox(t)

	mockWriter := newTestMockResponseWriter()

	cxt := agentctx.New(stdContext.Background(), nil, "test-chat-id")
	cxt.AssistantID = "test-assistant-id"
	cxt.Accept = agentctx.AcceptWebCUI
	cxt.Locale = "en"
	cxt.Writer = mockWriter

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				ctx.End();
				return { success: true };
			} catch (error) {
				return { success: false, error: error.message };
			}
		}`, cxt)
	require.NoError(t, err, "Call failed")

	result, ok := res.(map[string]interface{})
	require.True(t, ok, "Expected map result, got %T", res)
	assert.Equal(t, false, result["success"], "End without arguments should fail")
	assert.Contains(t, result["error"], "messageId", "Error should mention missing messageId")
}

func TestJsValueEndWithInvalidMessageID(t *testing.T) {
	testprepare.PrepareSandbox(t)

	mockWriter := newTestMockResponseWriter()

	cxt := agentctx.New(stdContext.Background(), nil, "test-chat-id")
	cxt.AssistantID = "test-assistant-id"
	cxt.Accept = agentctx.AcceptWebCUI
	cxt.Locale = "en"
	cxt.Writer = mockWriter

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				ctx.End(123);
				return { success: true };
			} catch (error) {
				return { success: false, error: error.message };
			}
		}`, cxt)
	require.NoError(t, err, "Call failed")

	result, ok := res.(map[string]interface{})
	require.True(t, ok, "Expected map result, got %T", res)
	assert.Equal(t, false, result["success"], "End with non-string messageId should fail")
	assert.Contains(t, result["error"], "string", "Error should mention messageId must be string")
}

func TestJsValueSendStreamErrorHandling(t *testing.T) {
	testprepare.PrepareSandbox(t)

	mockWriter := newTestMockResponseWriter()

	cxt := agentctx.New(stdContext.Background(), nil, "test-chat-id")
	cxt.AssistantID = "test-assistant-id"
	cxt.Accept = agentctx.AcceptWebCUI
	cxt.Locale = "en"
	cxt.Writer = mockWriter

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				ctx.SendStream();
				return { success: true };
			} catch (error) {
				return { success: false, error: error.message };
			}
		}`, cxt)
	require.NoError(t, err, "Call failed")

	result, ok := res.(map[string]interface{})
	require.True(t, ok, "Expected map result, got %T", res)
	assert.Equal(t, false, result["success"], "SendStream without arguments should fail")
	assert.Contains(t, result["error"], "SendStream requires a message argument", "Error should mention missing message")
}

func TestJsValueMultipleStreams(t *testing.T) {
	testprepare.PrepareSandbox(t)

	mockWriter := newTestMockResponseWriter()

	cxt := agentctx.New(stdContext.Background(), nil, "test-chat-id")
	cxt.AssistantID = "test-assistant-id"
	cxt.Accept = agentctx.AcceptWebCUI
	cxt.Locale = "en"
	cxt.Writer = mockWriter

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				const msg1 = ctx.SendStream({ type: "text", props: { content: "Stream 1: " } });
				const msg2 = ctx.SendStream({ type: "text", props: { content: "Stream 2: " } });
				
				ctx.Append(msg1, "A");
				ctx.Append(msg2, "X");
				ctx.Append(msg1, "B");
				ctx.Append(msg2, "Y");
				ctx.Append(msg1, "C");
				ctx.Append(msg2, "Z");
				
				ctx.End(msg1);
				ctx.End(msg2);
				
				return { success: true, msg1: msg1, msg2: msg2 };
			} catch (error) {
				return { success: false, error: error.message };
			}
		}`, cxt)
	require.NoError(t, err, "Call failed")

	result, ok := res.(map[string]interface{})
	require.True(t, ok, "Expected map result, got %T", res)
	if !result["success"].(bool) {
		t.Logf("Error: %v", result["error"])
	}
	assert.Equal(t, true, result["success"], "Multiple streams should work correctly")
	assert.NotEqual(t, result["msg1"], result["msg2"], "Message IDs should be different")
}

func TestJsValueSendVsSendStream(t *testing.T) {
	testprepare.PrepareSandbox(t)

	t.Run("Send auto-ends", func(t *testing.T) {
		mockWriter := newTestMockResponseWriter()
		cxt := agentctx.New(stdContext.Background(), nil, "test-chat-id")
		cxt.AssistantID = "test-assistant-id"
		cxt.Accept = agentctx.AcceptWebCUI
		cxt.Locale = "en"
		cxt.Writer = mockWriter

		_, err := v8.Call(v8.CallOptions{}, `
			function test(ctx) {
				ctx.Send("Complete message");
				return true;
			}`, cxt)
		require.NoError(t, err, "Call failed")

		cxt.CloseSafeWriter()

		output := mockWriter.buffer.String()
		assert.Contains(t, output, "message_start", "Send should emit message_start")
		assert.Contains(t, output, "message_end", "Send should auto-emit message_end")
	})

	t.Run("SendStream requires explicit End", func(t *testing.T) {
		mockWriter := newTestMockResponseWriter()
		cxt := agentctx.New(stdContext.Background(), nil, "test-chat-id")
		cxt.AssistantID = "test-assistant-id"
		cxt.Accept = agentctx.AcceptWebCUI
		cxt.Locale = "en"
		cxt.Writer = mockWriter

		_, err := v8.Call(v8.CallOptions{}, `
			function test(ctx) {
				const msgId = ctx.SendStream("Streaming message");
				return msgId;
			}`, cxt)
		require.NoError(t, err, "Call failed")

		cxt.CloseSafeWriter()

		output := mockWriter.buffer.String()
		assert.Contains(t, output, "message_start", "SendStream should emit message_start")
		assert.NotContains(t, output, "message_end", "SendStream should NOT auto-emit message_end")
	})
}
