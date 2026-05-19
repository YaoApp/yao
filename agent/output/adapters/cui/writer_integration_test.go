//go:build integration

package cui_test

import (
	"bytes"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/output/adapters/cui"
	"github.com/yaoapp/yao/agent/output/message"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

type mockResponseWriter struct {
	headers http.Header
	buffer  *bytes.Buffer
	status  int
}

func newMockResponseWriter() *mockResponseWriter {
	return &mockResponseWriter{
		headers: make(http.Header),
		buffer:  new(bytes.Buffer),
		status:  200,
	}
}

func (m *mockResponseWriter) Header() http.Header         { return m.headers }
func (m *mockResponseWriter) Write(b []byte) (int, error) { return m.buffer.Write(b) }
func (m *mockResponseWriter) WriteHeader(s int)           { m.status = s }
func (m *mockResponseWriter) Flush()                      {}

func TestCUIWriter_BasicWrite(t *testing.T) {
	testprepare.PrepareSandbox(t)

	w := newMockResponseWriter()
	writer, err := cui.NewWriter(message.Options{
		Accept: "cui-web",
		Writer: w,
	})
	require.NoError(t, err)
	require.NotNil(t, writer)

	msg := &message.Message{
		Type: message.TypeText,
		Props: map[string]interface{}{
			"content": "Hello from CUI writer",
		},
	}

	err = writer.Write(msg)
	require.NoError(t, err)

	result := w.buffer.String()
	assert.True(t, strings.Contains(result, "data: "), "expected SSE data prefix")
	assert.True(t, strings.Contains(result, "Hello from CUI writer"), "expected content in output")
	assert.True(t, strings.Contains(result, "text"), "expected message type in output")
}

func TestCUIWriter_Close(t *testing.T) {
	testprepare.PrepareSandbox(t)

	w := newMockResponseWriter()
	writer, err := cui.NewWriter(message.Options{
		Accept: "cui-web",
		Writer: w,
	})
	require.NoError(t, err)

	err = writer.Close()
	require.NoError(t, err)

	// CUI writer does not send [DONE] on close (unlike openai)
	result := w.buffer.String()
	assert.Equal(t, "", result)
}

func TestCUIWriter_WriteMultipleMessages(t *testing.T) {
	testprepare.PrepareSandbox(t)

	w := newMockResponseWriter()
	writer, err := cui.NewWriter(message.Options{
		Accept: "cui-web",
		Writer: w,
	})
	require.NoError(t, err)

	msgs := []*message.Message{
		{Type: message.TypeText, Props: map[string]interface{}{"content": "first message"}},
		{Type: message.TypeThinking, Props: map[string]interface{}{"content": "thinking..."}},
		{Type: message.TypeLoading, Props: map[string]interface{}{"message": "loading"}},
	}

	for _, msg := range msgs {
		err = writer.Write(msg)
		require.NoError(t, err)
	}

	result := w.buffer.String()
	assert.True(t, strings.Contains(result, "first message"))
	assert.True(t, strings.Contains(result, "thinking..."))
	assert.True(t, strings.Contains(result, "loading"))
}

func TestCUIWriter_WriteGroup(t *testing.T) {
	testprepare.PrepareSandbox(t)

	w := newMockResponseWriter()
	writer, err := cui.NewWriter(message.Options{
		Accept: "cui-web",
		Writer: w,
	})
	require.NoError(t, err)

	group := &message.Group{
		ID: "grp_001",
		Messages: []*message.Message{
			{Type: message.TypeText, Props: map[string]interface{}{"content": "group msg 1"}},
			{Type: message.TypeText, Props: map[string]interface{}{"content": "group msg 2"}},
		},
	}

	err = writer.WriteGroup(group)
	require.NoError(t, err)

	result := w.buffer.String()
	assert.True(t, strings.Contains(result, "data: "), "expected SSE format")
	assert.True(t, strings.Contains(result, "grp_001"), "expected group ID in output")
}

func TestCUIWriter_EventMessage(t *testing.T) {
	testprepare.PrepareSandbox(t)

	w := newMockResponseWriter()
	writer, err := cui.NewWriter(message.Options{
		Accept: "cui-web",
		Writer: w,
	})
	require.NoError(t, err)

	msg := &message.Message{
		Type: message.TypeEvent,
		Props: map[string]interface{}{
			"event":   "stream_start",
			"message": "Stream started",
		},
	}

	err = writer.Write(msg)
	require.NoError(t, err)

	result := w.buffer.String()
	assert.True(t, strings.Contains(result, "stream_start"))
	assert.True(t, strings.Contains(result, "Stream started"))
}
