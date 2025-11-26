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
