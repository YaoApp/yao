//go:build unit

package context_test

import (
	"bytes"
	stdContext "context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	agentctx "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/output/message"
)

func TestMessageLifecycleEvents(t *testing.T) {
	var buf bytes.Buffer
	mockWriter := &lifecycleMockWriter{
		buffer:  &buf,
		headers: make(http.Header),
	}

	ctx := agentctx.New(stdContext.Background(), nil, "test-chat")
	ctx.Accept = agentctx.AcceptWebCUI
	ctx.Writer = mockWriter
	ctx.AssistantID = "test-assistant"
	ctx.Locale = "en"

	err := ctx.Send(&message.Message{
		Type: message.TypeText,
		Props: map[string]interface{}{
			"content": "Hello World",
		},
	})
	require.NoError(t, err)

	ctx.Flush()
	ctx.CloseSafeWriter()

	output := buf.String()
	t.Logf("Output:\n%s", output)

	lines := bytes.Split([]byte(output), []byte("\n"))

	var messages []map[string]interface{}
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		if bytes.HasPrefix(line, []byte("data: ")) {
			line = bytes.TrimPrefix(line, []byte("data: "))
		}

		var msg map[string]interface{}
		if err := json.Unmarshal(line, &msg); err == nil {
			messages = append(messages, msg)
		}
	}

	hasMessageStart := false
	hasMessageEnd := false
	hasTextMessage := false

	for _, msg := range messages {
		msgType, _ := msg["type"].(string)
		if msgType == "event" {
			if props, ok := msg["props"].(map[string]interface{}); ok {
				if eventType, ok := props["event"].(string); ok {
					if eventType == "message_start" {
						hasMessageStart = true
					}
					if eventType == "message_end" {
						hasMessageEnd = true
					}
				}
			}
		} else if msgType == "text" {
			hasTextMessage = true
		}
	}

	assert.True(t, hasMessageStart, "Should have message_start event")
	assert.True(t, hasTextMessage, "Should have text message")
	assert.True(t, hasMessageEnd, "Should have message_end event")
}

type lifecycleMockWriter struct {
	buffer     *bytes.Buffer
	statusCode int
	headers    http.Header
}

func (m *lifecycleMockWriter) Header() http.Header {
	return m.headers
}

func (m *lifecycleMockWriter) Write(data []byte) (int, error) {
	return m.buffer.Write(data)
}

func (m *lifecycleMockWriter) WriteHeader(statusCode int) {
	m.statusCode = statusCode
}
