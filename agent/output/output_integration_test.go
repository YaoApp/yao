//go:build integration

package output_test

import (
	"bytes"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/output"
	"github.com/yaoapp/yao/agent/output/message"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

type mockWriter struct {
	headers http.Header
	buffer  *bytes.Buffer
	status  int
}

func newMockWriter() *mockWriter {
	return &mockWriter{
		headers: make(http.Header),
		buffer:  new(bytes.Buffer),
		status:  200,
	}
}

func (m *mockWriter) Header() http.Header         { return m.headers }
func (m *mockWriter) Write(b []byte) (int, error) { return m.buffer.Write(b) }
func (m *mockWriter) WriteHeader(s int)           { m.status = s }
func (m *mockWriter) Flush()                      {}

func TestNewOutput_Standard(t *testing.T) {
	testprepare.PrepareSandbox(t)

	w := newMockWriter()
	o, err := output.NewOutput(message.Options{
		Accept: output.AcceptStandard,
		Writer: w,
	})
	require.NoError(t, err)
	require.NotNil(t, o)
	assert.NotNil(t, o.Writer)
}

func TestNewOutput_CUI(t *testing.T) {
	testprepare.PrepareSandbox(t)

	w := newMockWriter()
	o, err := output.NewOutput(message.Options{
		Accept: output.AcceptWebCUI,
		Writer: w,
	})
	require.NoError(t, err)
	require.NotNil(t, o)
	assert.NotNil(t, o.Writer)
}

func TestNewOutput_Default(t *testing.T) {
	testprepare.PrepareSandbox(t)

	w := newMockWriter()
	o, err := output.NewOutput(message.Options{
		Accept: "",
		Writer: w,
	})
	require.NoError(t, err)
	require.NotNil(t, o)
	assert.NotNil(t, o.Writer)
}

func TestOutput_SendAndClose(t *testing.T) {
	testprepare.PrepareSandbox(t)

	w := newMockWriter()
	o, err := output.NewOutput(message.Options{
		Accept: output.AcceptStandard,
		Writer: w,
	})
	require.NoError(t, err)

	msg := output.NewTextMessage("Hello, world!")
	err = o.Send(msg)
	require.NoError(t, err)

	err = o.Close()
	require.NoError(t, err)

	result := w.buffer.String()
	assert.True(t, strings.Contains(result, "data: "), "expected SSE data prefix")
	assert.True(t, strings.Contains(result, "Hello, world!"), "expected message content")
	assert.True(t, strings.Contains(result, "[DONE]"), "expected [DONE] marker")
}

func TestOutput_SendMulti(t *testing.T) {
	testprepare.PrepareSandbox(t)

	w := newMockWriter()
	o, err := output.NewOutput(message.Options{
		Accept: output.AcceptStandard,
		Writer: w,
	})
	require.NoError(t, err)

	msg1 := output.NewTextMessage("first")
	msg2 := output.NewTextMessage("second")
	err = o.SendMulti(msg1, msg2)
	require.NoError(t, err)

	err = o.Close()
	require.NoError(t, err)

	result := w.buffer.String()
	assert.True(t, strings.Contains(result, "first"))
	assert.True(t, strings.Contains(result, "second"))
}

func TestOutput_CUI_SendAndClose(t *testing.T) {
	testprepare.PrepareSandbox(t)

	w := newMockWriter()
	o, err := output.NewOutput(message.Options{
		Accept: output.AcceptWebCUI,
		Writer: w,
	})
	require.NoError(t, err)

	msg := output.NewTextMessage("CUI message")
	err = o.Send(msg)
	require.NoError(t, err)

	err = o.Close()
	require.NoError(t, err)

	result := w.buffer.String()
	assert.True(t, strings.Contains(result, "data: "), "expected SSE data prefix")
	assert.True(t, strings.Contains(result, "CUI message"), "expected message content in output")
}
