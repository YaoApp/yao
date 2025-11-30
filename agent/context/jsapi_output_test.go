package context

import (
	"bytes"
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	v8 "github.com/yaoapp/gou/runtime/v8"
	"github.com/yaoapp/yao/agent/output/message"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/test"
)

// mockResponseWriter is a mock implementation of http.ResponseWriter for testing
type mockResponseWriter struct {
	headers http.Header
	buffer  *bytes.Buffer
	status  int
}

func newMockResponseWriter() *mockResponseWriter {
	return &mockResponseWriter{
		headers: make(http.Header),
		buffer:  &bytes.Buffer{},
		status:  http.StatusOK,
	}
}

func (m *mockResponseWriter) Header() http.Header {
	return m.headers
}

func (m *mockResponseWriter) Write(b []byte) (int, error) {
	return m.buffer.Write(b)
}

func (m *mockResponseWriter) WriteHeader(statusCode int) {
	m.status = statusCode
}

// TestJsValueSend test the Send method on Context
func TestJsValueSend(t *testing.T) {

	test.Prepare(t, config.Conf)
	defer test.Clean()

	cxt := &Context{
		ChatID:      "test-chat-id",
		AssistantID: "test-assistant-id",
		Context:     context.Background(),
		Accept:      "standard",
		Locale:      "en",
		Writer:      newMockResponseWriter(),
		IDGenerator: message.NewIDGenerator(),
	}

	// Test sending string shorthand
	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				// Send simple string
				ctx.Send("Hello World");
				return { success: true };
			} catch (error) {
				return { success: false, error: error.message };
			}
		}`, cxt)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	result, ok := res.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", res)
	}
	assert.Equal(t, true, result["success"], "Send string should succeed")

	// Test sending message object
	res, err = v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				// Send message object
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
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	result, ok = res.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", res)
	}
	assert.Equal(t, true, result["success"], "Send message object should succeed")
}

// TestJsValueSendGroup test the SendGroup method on Context
// TestJsValueSendDeltaUpdates test delta updates in Send
func TestJsValueSendDeltaUpdates(t *testing.T) {

	test.Prepare(t, config.Conf)
	defer test.Clean()

	cxt := &Context{
		ChatID:      "test-chat-id",
		AssistantID: "test-assistant-id",
		Context:     context.Background(),
		Accept:      "standard",
		Locale:      "en",
		Writer:      newMockResponseWriter(),
		IDGenerator: message.NewIDGenerator(),
	}

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				// Send initial message
				ctx.Send({
					type: "text",
					props: { content: "Hello" },
					id: "msg_1",
					delta: false
				});
				
				// Send delta update (append)
				ctx.Send({
					type: "text",
					props: { content: " World" },
					id: "msg_1",
					delta: true,
					delta_path: "content",
					delta_action: "append"
				});
				
				// Send completion (no done field needed)
				
				return { success: true };
			} catch (error) {
				return { success: false, error: error.message };
			}
		}`, cxt)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	result, ok := res.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", res)
	}
	assert.Equal(t, true, result["success"], "Delta updates should succeed")
}

// TestJsValueSendMultipleTypes test sending different message types
func TestJsValueSendMultipleTypes(t *testing.T) {

	test.Prepare(t, config.Conf)
	defer test.Clean()

	cxt := &Context{
		ChatID:      "test-chat-id",
		AssistantID: "test-assistant-id",
		Context:     context.Background(),
		Accept:      "standard",
		Locale:      "en",
		Writer:      newMockResponseWriter(),
		IDGenerator: message.NewIDGenerator(),
	}

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				// Text message
				ctx.Send({
					type: "text",
					props: { content: "Hello" }
				});
				
				// Thinking message
				ctx.Send({
					type: "thinking",
					props: { content: "Let me think..." }
				});
				
				// Loading message
				ctx.Send({
					type: "loading",
					props: { message: "Processing..." }
				});
				
				// Tool call message
				ctx.Send({
					type: "tool_call",
					props: {
						id: "call_123",
						name: "get_weather",
						arguments: '{"location": "San Francisco"}'
					}
				});
				
				// Error message
				ctx.Send({
					type: "error",
					props: {
						message: "Something went wrong",
						code: "ERR_500"
					}
				});
				
				// Image message
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
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	result, ok := res.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", res)
	}
	assert.Equal(t, true, result["success"], "Multiple message types should succeed")
}

// TestJsValueSendErrorHandling test error handling in Send
func TestJsValueSendErrorHandling(t *testing.T) {

	test.Prepare(t, config.Conf)
	defer test.Clean()

	cxt := &Context{
		ChatID:      "test-chat-id",
		AssistantID: "test-assistant-id",
		Context:     context.Background(),
		Accept:      "standard",
		Locale:      "en",
		Writer:      newMockResponseWriter(),
		IDGenerator: message.NewIDGenerator(),
	}

	// Test invalid argument - no arguments
	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				ctx.Send();
				return { success: true };
			} catch (error) {
				return { success: false, error: error.message };
			}
		}`, cxt)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	result, ok := res.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", res)
	}
	assert.Equal(t, false, result["success"], "Send without arguments should fail")
	assert.Contains(t, result["error"], "Send requires a message argument", "Error should mention missing message")
}

// TestJsValueSendGroupErrorHandling test error handling in SendGroup
// TestJsValueSendWithCUIAccept test Send with CUI accept types
func TestJsValueSendWithCUIAccept(t *testing.T) {

	test.Prepare(t, config.Conf)
	defer test.Clean()

	acceptTypes := []string{"cui-web", "cui-native", "cui-desktop"}

	for _, acceptType := range acceptTypes {
		t.Run(acceptType, func(t *testing.T) {
			cxt := &Context{
				ChatID:      "test-chat-id",
				AssistantID: "test-assistant-id",
				Context:     context.Background(),
				Accept:      Accept(acceptType),
				Locale:      "en",
				Writer:      newMockResponseWriter(),
			}

			res, err := v8.Call(v8.CallOptions{}, `
				function test(ctx) {
					try {
						ctx.Send({
							type: "text",
							props: { content: "Hello CUI" }
						});
						return { success: true };
					} catch (error) {
						return { success: false, error: error.message };
					}
				}`, cxt)
			if err != nil {
				t.Fatalf("Call failed: %v", err)
			}

			result, ok := res.(map[string]interface{})
			if !ok {
				t.Fatalf("Expected map result, got %T", res)
			}
			assert.Equal(t, true, result["success"], "Send with "+acceptType+" should succeed")
		})
	}
}

// TestJsValueSendGroupWithMetadata test SendGroup with various metadata
// TestJsValueSendChainedCalls test chained Send calls
func TestJsValueSendChainedCalls(t *testing.T) {

	test.Prepare(t, config.Conf)
	defer test.Clean()

	cxt := &Context{
		ChatID:      "test-chat-id",
		AssistantID: "test-assistant-id",
		Context:     context.Background(),
		Accept:      "standard",
		Locale:      "en",
		Writer:      newMockResponseWriter(),
		IDGenerator: message.NewIDGenerator(),
	}

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				// Multiple sequential sends (each auto-flushes)
				ctx.Send("Step 1");
				ctx.Send("Step 2");
				ctx.Send("Step 3");
				ctx.Send("Step 4");
				
				return { success: true };
			} catch (error) {
				return { success: false, error: error.message };
			}
		}`, cxt)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	result, ok := res.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", res)
	}
	assert.Equal(t, true, result["success"], "Chained Send calls should succeed")
}

// TestJsValueIDGenerators test ID generator methods
func TestJsValueIDGenerators(t *testing.T) {

	test.Prepare(t, config.Conf)
	defer test.Clean()

	cxt := &Context{
		ChatID:      "test-chat-id",
		AssistantID: "test-assistant-id",
		Context:     context.Background(),
		Accept:      "standard",
		Locale:      "en",
		Writer:      newMockResponseWriter(),
		IDGenerator: message.NewIDGenerator(),
	}

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				// Test MessageID generator
				const msgId1 = ctx.MessageID();
				const msgId2 = ctx.MessageID();
				
				// Test BlockID generator
				const blockId1 = ctx.BlockID();
				const blockId2 = ctx.BlockID();
				
				// Test ThreadID generator
				const threadId1 = ctx.ThreadID();
				const threadId2 = ctx.ThreadID();
				
				// Verify IDs are strings and sequential
				if (typeof msgId1 !== 'string' || typeof msgId2 !== 'string') {
					throw new Error('MessageID should return string');
				}
				if (typeof blockId1 !== 'string' || typeof blockId2 !== 'string') {
					throw new Error('BlockID should return string');
				}
				if (typeof threadId1 !== 'string' || typeof threadId2 !== 'string') {
					throw new Error('ThreadID should return string');
				}
				
				// Verify they follow the pattern (M1, M2, B1, B2, T1, T2)
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
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	result, ok := res.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", res)
	}
	assert.Equal(t, true, result["success"], "ID generators should succeed")
}

// TestJsValueSendWithBlockID test Send with block_id parameter
func TestJsValueSendWithBlockID(t *testing.T) {

	test.Prepare(t, config.Conf)
	defer test.Clean()

	cxt := &Context{
		ChatID:      "test-chat-id",
		AssistantID: "test-assistant-id",
		Context:     context.Background(),
		Accept:      "standard",
		Locale:      "en",
		Writer:      newMockResponseWriter(),
		IDGenerator: message.NewIDGenerator(),
	}

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				// Generate block ID manually
				const blockId = ctx.BlockID();
				
				// Send multiple messages with same block ID
				const msg1 = ctx.Send("Message 1", blockId);
				const msg2 = ctx.Send("Message 2", blockId);
				const msg3 = ctx.Send("Message 3", blockId);
				
				// Send message with block_id in object (higher priority)
				const msg4 = ctx.Send({
					type: "text",
					props: { content: "Message 4" },
					block_id: "B_custom"
				}, blockId);  // blockId parameter should be ignored
				
				return { 
					success: true,
					msg1: msg1,
					msg2: msg2,
					msg3: msg3,
					msg4: msg4,
					blockId: blockId
				};
			} catch (error) {
				return { success: false, error: error.message };
			}
		}`, cxt)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	result, ok := res.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", res)
	}
	assert.Equal(t, true, result["success"], "Send with blockId should succeed")
}

// TestJsValueReplace test ctx.Replace method
func TestJsValueReplace(t *testing.T) {

	test.Prepare(t, config.Conf)
	defer test.Clean()

	cxt := &Context{
		ChatID:      "test-chat-id",
		AssistantID: "test-assistant-id",
		Context:     context.Background(),
		Accept:      "standard",
		Locale:      "en",
		Writer:      newMockResponseWriter(),
		IDGenerator: message.NewIDGenerator(),
	}

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				// Send initial message
				const msgId = ctx.Send("Initial content");
				
				// Replace with new content
				ctx.Replace(msgId, "Updated content");
				
				// Replace with object
				ctx.Replace(msgId, {
					type: "text",
					props: { content: "Final content" }
				});
				
				return { success: true, msgId: msgId };
			} catch (error) {
				return { success: false, error: error.message };
			}
		}`, cxt)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	result, ok := res.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", res)
	}
	assert.Equal(t, true, result["success"], "Replace should succeed")
}

// TestJsValueAppend test ctx.Append method
func TestJsValueAppend(t *testing.T) {

	test.Prepare(t, config.Conf)
	defer test.Clean()

	cxt := &Context{
		ChatID:      "test-chat-id",
		AssistantID: "test-assistant-id",
		Context:     context.Background(),
		Accept:      "standard",
		Locale:      "en",
		Writer:      newMockResponseWriter(),
		IDGenerator: message.NewIDGenerator(),
	}

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				// Send initial message
				const msgId = ctx.Send("Hello");
				
				// Append to default path
				ctx.Append(msgId, " World");
				ctx.Append(msgId, "!");
				
				// Append to specific path
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
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	result, ok := res.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", res)
	}
	assert.Equal(t, true, result["success"], "Append should succeed")
}

// TestJsValueMerge test ctx.Merge method
func TestJsValueMerge(t *testing.T) {

	test.Prepare(t, config.Conf)
	defer test.Clean()

	cxt := &Context{
		ChatID:      "test-chat-id",
		AssistantID: "test-assistant-id",
		Context:     context.Background(),
		Accept:      "standard",
		Locale:      "en",
		Writer:      newMockResponseWriter(),
		IDGenerator: message.NewIDGenerator(),
	}

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				// Send initial message with object
				const msgId = ctx.Send({
					type: "status",
					props: {
						status: "running",
						progress: 0,
						started: true
					}
				});
				
				// Merge updates (keeps other fields)
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
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	result, ok := res.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", res)
	}
	if !result["success"].(bool) {
		t.Logf("Error: %v", result["error"])
	}
	assert.Equal(t, true, result["success"], "Merge should succeed")
}

// TestJsValueSet test ctx.Set method
func TestJsValueSet(t *testing.T) {

	test.Prepare(t, config.Conf)
	defer test.Clean()

	cxt := &Context{
		ChatID:      "test-chat-id",
		AssistantID: "test-assistant-id",
		Context:     context.Background(),
		Accept:      "standard",
		Locale:      "en",
		Writer:      newMockResponseWriter(),
		IDGenerator: message.NewIDGenerator(),
	}

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				// Send initial message
				const msgId = ctx.Send({
					type: "result",
					props: { content: "Initial" }
				});
				
				// Set new fields
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
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	result, ok := res.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", res)
	}
	if !result["success"].(bool) {
		t.Logf("Error: %v", result["error"])
	}
	assert.Equal(t, true, result["success"], "Set should succeed")
}

// TestJsValueBlockIDInheritance test that delta operations inherit block_id
func TestJsValueBlockIDInheritance(t *testing.T) {

	test.Prepare(t, config.Conf)
	defer test.Clean()

	cxt := &Context{
		ChatID:      "test-chat-id",
		AssistantID: "test-assistant-id",
		Context:     context.Background(),
		Accept:      "standard",
		Locale:      "en",
		Writer:      newMockResponseWriter(),
		IDGenerator: message.NewIDGenerator(),
	}

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				// Send message with block_id
				const blockId = ctx.BlockID();
				const msgId = ctx.Send("Initial message", blockId);
				
				// Delta operations should inherit block_id automatically
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
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	result, ok := res.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", res)
	}
	if !result["success"].(bool) {
		t.Logf("Error: %v", result["error"])
	}
	assert.Equal(t, true, result["success"], "Delta operations should inherit block_id")
}

// TestJsValueEndBlock tests the EndBlock method on Context
func TestJsValueEndBlock(t *testing.T) {

	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Setup mock writer
	mockWriter := newMockResponseWriter()

	// Use New() to properly initialize messageMetadata
	cxt := New(context.Background(), nil, "test-chat-id")
	cxt.AssistantID = "test-assistant-id"
	cxt.Accept = AcceptWebCUI
	cxt.Locale = "en"
	cxt.Writer = mockWriter

	// Test EndBlock method
	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				// Create a block and send messages
				const block_id = ctx.BlockID(); // "B1"
				
				ctx.Send("Message 1", block_id);
				ctx.Send("Message 2", block_id);
				ctx.Send("Message 3", block_id);
				
				// End the block manually
				ctx.EndBlock(block_id);
				
				return { success: true, block_id: block_id };
			} catch (error) {
				return { success: false, error: error.message };
			}
		}`, cxt)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	result, ok := res.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", res)
	}
	if !result["success"].(bool) {
		t.Logf("Error: %v", result["error"])
	}
	assert.Equal(t, true, result["success"], "EndBlock should work correctly")

	// Verify that block_end event was sent
	output := mockWriter.buffer.String()
	assert.Contains(t, output, "block_end", "Output should contain block_end event")
}
