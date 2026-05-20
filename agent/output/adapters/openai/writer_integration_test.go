//go:build integration

package openai_test

import (
	"bytes"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/output/adapters/openai"
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

func TestWriter_BasicWrite(t *testing.T) {
	testprepare.PrepareSandbox(t)

	w := newMockResponseWriter()
	writer, err := openai.NewWriter(message.Options{
		Accept: "standard",
		Writer: w,
	})
	require.NoError(t, err)
	require.NotNil(t, writer)

	msg := &message.Message{
		Type: message.TypeText,
		Props: map[string]interface{}{
			"content": "Hello from OpenAI writer",
		},
	}

	err = writer.Write(msg)
	require.NoError(t, err)

	result := w.buffer.String()
	assert.True(t, strings.Contains(result, "data: "), "expected SSE data prefix")
	assert.True(t, strings.Contains(result, "Hello from OpenAI writer"), "expected content in output")
	assert.True(t, strings.Contains(result, "chat.completion.chunk"), "expected OpenAI chunk format")
}

func TestWriter_Close(t *testing.T) {
	testprepare.PrepareSandbox(t)

	w := newMockResponseWriter()
	writer, err := openai.NewWriter(message.Options{
		Accept: "standard",
		Writer: w,
	})
	require.NoError(t, err)

	err = writer.Close()
	require.NoError(t, err)

	result := w.buffer.String()
	assert.True(t, strings.Contains(result, "[DONE]"), "expected [DONE] in output")
}

func TestWriter_WriteThinking(t *testing.T) {
	testprepare.PrepareSandbox(t)

	w := newMockResponseWriter()
	writer, err := openai.NewWriter(message.Options{
		Accept: "standard",
		Writer: w,
	})
	require.NoError(t, err)

	msg := &message.Message{
		Type: message.TypeThinking,
		Props: map[string]interface{}{
			"content": "Reasoning step 1",
		},
	}

	err = writer.Write(msg)
	require.NoError(t, err)

	result := w.buffer.String()
	assert.True(t, strings.Contains(result, "Reasoning step 1"))
	assert.True(t, strings.Contains(result, "reasoning_content"))
}

func TestWriter_WriteMultipleMessages(t *testing.T) {
	testprepare.PrepareSandbox(t)

	w := newMockResponseWriter()
	writer, err := openai.NewWriter(message.Options{
		Accept: "standard",
		Writer: w,
	})
	require.NoError(t, err)

	msgs := []*message.Message{
		{Type: message.TypeText, Props: map[string]interface{}{"content": "first"}},
		{Type: message.TypeText, Props: map[string]interface{}{"content": "second"}},
	}

	for _, msg := range msgs {
		err = writer.Write(msg)
		require.NoError(t, err)
	}

	err = writer.Close()
	require.NoError(t, err)

	result := w.buffer.String()
	assert.True(t, strings.Contains(result, "first"))
	assert.True(t, strings.Contains(result, "second"))
	assert.True(t, strings.Contains(result, "[DONE]"))
}

func TestWriter_WriteGroup(t *testing.T) {
	testprepare.PrepareSandbox(t)

	w := newMockResponseWriter()
	writer, err := openai.NewWriter(message.Options{
		Accept: "standard",
		Writer: w,
	})
	require.NoError(t, err)

	group := &message.Group{
		ID: "group_1",
		Messages: []*message.Message{
			{Type: message.TypeText, Props: map[string]interface{}{"content": "msg in group"}},
		},
	}

	err = writer.WriteGroup(group)
	require.NoError(t, err)

	result := w.buffer.String()
	assert.True(t, strings.Contains(result, "msg in group"))
}
