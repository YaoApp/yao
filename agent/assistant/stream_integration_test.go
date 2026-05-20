//go:build integration

package assistant_test

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/assistant"
	agentContext "github.com/yaoapp/yao/agent/context"
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
		buffer:  &bytes.Buffer{},
		status:  http.StatusOK,
	}
}

func (m *mockResponseWriter) Header() http.Header         { return m.headers }
func (m *mockResponseWriter) Write(b []byte) (int, error) { return m.buffer.Write(b) }
func (m *mockResponseWriter) WriteHeader(statusCode int)  { m.status = statusCode }

func newStreamTestContext(t *testing.T, chatID, assistantID string) *agentContext.Context {
	t.Helper()
	ctx := newTestContext(chatID, assistantID)
	ctx.Writer = newMockResponseWriter()
	return ctx
}

func TestStreamBasic(t *testing.T) {
	testprepare.PrepareSandbox(t)

	ast, err := assistant.Get("tests.simple-greeting")
	require.NoError(t, err)
	require.NotNil(t, ast)

	ctx := newStreamTestContext(t, "chat-stream-basic", "tests.simple-greeting")
	messages := []agentContext.Message{
		{Role: agentContext.RoleUser, Content: "Hello"},
	}

	response, err := ast.Stream(ctx, messages)
	require.NoError(t, err)
	require.NotNil(t, response, "response should not be nil")
	assert.NotNil(t, response.Completion, "response.Completion should not be nil with mock LLM")
	assert.NotEmpty(t, response.AssistantID, "response.AssistantID should not be empty")
	assert.NotEmpty(t, response.ChatID, "response.ChatID should not be empty")
}

func TestStreamNoPrompt(t *testing.T) {
	testprepare.PrepareSandbox(t)

	ast, err := assistant.Get("tests.no-prompt")
	require.NoError(t, err)
	require.NotNil(t, ast)

	ctx := newStreamTestContext(t, "chat-stream-no-prompt", "tests.no-prompt")
	messages := []agentContext.Message{
		{Role: agentContext.RoleUser, Content: "Hello"},
	}

	response, err := ast.Stream(ctx, messages)
	require.NoError(t, err)
	require.NotNil(t, response, "response should not be nil even without prompts")
	// no-prompt has no Prompts and no MCP, so LLM call is skipped
	assert.Nil(t, response.Completion, "response.Completion should be nil without prompts or MCP")
}

func TestStreamWithHookNext(t *testing.T) {
	testprepare.PrepareSandbox(t)

	ast, err := assistant.Get("tests.hook-next")
	require.NoError(t, err)
	require.NotNil(t, ast)
	require.NotNil(t, ast.HookScript)

	ctx := newStreamTestContext(t, "chat-stream-hook-next", "tests.hook-next")
	messages := []agentContext.Message{
		{Role: agentContext.RoleUser, Content: "return_custom_data"},
	}

	response, err := ast.Stream(ctx, messages)
	require.NoError(t, err)
	require.NotNil(t, response, "response should not be nil")
	assert.NotNil(t, response.Next, "response.Next should not be nil for custom data return")
}

func TestStreamPermissionDenied(t *testing.T) {
	testprepare.PrepareSandbox(t)

	ast, err := assistant.Get("tests.simple-greeting")
	require.NoError(t, err)
	require.NotNil(t, ast)

	// Create context without Authorized
	ctx := agentContext.New(nil, nil, "chat-stream-permission")
	ctx.AssistantID = "tests.simple-greeting"
	ctx.Writer = newMockResponseWriter()

	messages := []agentContext.Message{
		{Role: agentContext.RoleUser, Content: "Hello"},
	}

	_, err = ast.Stream(ctx, messages)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "authorized information not found")
}
