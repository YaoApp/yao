package context_test

import (
	"bytes"
	stdContext "context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/output/message"
)

func TestMessageLifecycleEvents(t *testing.T) {
	// Create a mock response writer
	var buf bytes.Buffer
	mockWriter := &mockResponseWriter{
		buffer:  &buf,
		headers: make(http.Header),
	}

	// Create context using New() to ensure proper initialization
	ctx := context.New(stdContext.Background(), nil, "test-chat")
	ctx.Accept = context.AcceptWebCUI
	ctx.Writer = mockWriter
	ctx.AssistantID = "test-assistant"
	ctx.Locale = "en"

	// Send a simple text message
	err := ctx.Send(&message.Message{
		Type: message.TypeText,
		Props: map[string]interface{}{
			"content": "Hello World",
		},
	})
	assert.NoError(t, err)

	// Flush to ensure all messages are written
	ctx.Flush()

	// Parse output to find events
	output := buf.String()
	t.Logf("Output:\n%s", output)

	lines := bytes.Split([]byte(output), []byte("\n"))

	var messages []map[string]interface{}
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		// CUI format: data: {...}
		if bytes.HasPrefix(line, []byte("data: ")) {
			line = bytes.TrimPrefix(line, []byte("data: "))
		}

		var msg map[string]interface{}
		if err := json.Unmarshal(line, &msg); err == nil {
			messages = append(messages, msg)
			t.Logf("Message: type=%s", msg["type"])
		}
	}

	// Check for events
	hasMessageStart := false
	hasMessageEnd := false
	hasTextMessage := false

	for _, msg := range messages {
		msgType, _ := msg["type"].(string)

		if msgType == "event" {
			if props, ok := msg["props"].(map[string]interface{}); ok {
				if eventType, ok := props["event"].(string); ok {
					t.Logf("Event type: %s", eventType)
					if eventType == "message_start" {
						hasMessageStart = true
						t.Logf("✓ Found message_start event")
					}
					if eventType == "message_end" {
						hasMessageEnd = true
						t.Logf("✓ Found message_end event: %+v", props)
					}
				}
			}
		} else if msgType == "text" {
			hasTextMessage = true
			t.Logf("✓ Found text message")
		}
	}

	t.Logf("Summary: start=%v, text=%v, end=%v", hasMessageStart, hasTextMessage, hasMessageEnd)

	assert.True(t, hasMessageStart, "Should have message_start event")
	assert.True(t, hasTextMessage, "Should have text message")
	assert.True(t, hasMessageEnd, "Should have message_end event")
}

// mockResponseWriter implements http.ResponseWriter for testing
type mockResponseWriter struct {
	buffer     *bytes.Buffer
	statusCode int
	headers    http.Header
}

func (m *mockResponseWriter) Header() http.Header {
	return m.headers
}

func (m *mockResponseWriter) Write(data []byte) (int, error) {
	return m.buffer.Write(data)
}

func (m *mockResponseWriter) WriteHeader(statusCode int) {
	m.statusCode = statusCode
}
