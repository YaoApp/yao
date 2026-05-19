//go:build e2e

package assistant_test

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/assistant"
	agentContext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/output/message"
	oauthTypes "github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

type e2eMockResponseWriter struct {
	headers http.Header
	buffer  *bytes.Buffer
	status  int
}

func newE2EMockResponseWriter() *e2eMockResponseWriter {
	return &e2eMockResponseWriter{
		headers: make(http.Header),
		buffer:  &bytes.Buffer{},
		status:  http.StatusOK,
	}
}

func (m *e2eMockResponseWriter) Header() http.Header         { return m.headers }
func (m *e2eMockResponseWriter) Write(b []byte) (int, error) { return m.buffer.Write(b) }
func (m *e2eMockResponseWriter) WriteHeader(statusCode int)  { m.status = statusCode }

func newE2ETestContext(t *testing.T, identity *testprepare.TestIdentity, chatID, assistantID string) *agentContext.Context {
	t.Helper()

	authorized := &oauthTypes.AuthorizedInfo{
		Subject:   "e2e-test-user",
		ClientID:  "e2e-test-client",
		Scope:     "openid profile email",
		SessionID: "e2e-session",
		UserID:    identity.AlphaOwnerUserID,
		TeamID:    identity.AlphaTeamID,
		TenantID:  "e2e-tenant",
	}

	ctx := agentContext.New(nil, authorized, chatID)
	ctx.AssistantID = assistantID
	ctx.Locale = "en-us"
	ctx.Theme = "light"
	ctx.Client = agentContext.Client{
		Type:      "web",
		UserAgent: "E2ETestAgent/1.0",
		IP:        "127.0.0.1",
	}
	ctx.Referer = agentContext.RefererAPI
	ctx.Accept = agentContext.AcceptWebCUI
	ctx.IDGenerator = message.NewIDGenerator()
	ctx.Metadata = make(map[string]interface{})
	ctx.Writer = newE2EMockResponseWriter()
	return ctx
}

func TestE2ELLMStream(t *testing.T) {
	identity := testprepare.PrepareE2E(t)
	require.NotNil(t, identity)

	ast, err := assistant.Get("tests.simple-greeting")
	require.NoError(t, err)
	require.NotNil(t, ast)

	ctx := newE2ETestContext(t, identity, "chat-e2e-llm-stream", "tests.simple-greeting")
	messages := []agentContext.Message{
		{Role: agentContext.RoleUser, Content: "Hello, how are you?"},
	}

	response, err := ast.Stream(ctx, messages)
	require.NoError(t, err, "Stream should not return an error in E2E")
	require.NotNil(t, response, "response should not be nil")

	assert.NotNil(t, response.Completion, "response.Completion should not be nil")
	assert.NotEmpty(t, response.AssistantID, "response.AssistantID should not be empty")
	assert.NotEmpty(t, response.ChatID, "response.ChatID should not be empty")
	assert.NotEmpty(t, response.ContextID, "response.ContextID should not be empty")
	assert.NotEmpty(t, response.TraceID, "response.TraceID should not be empty")
}

func TestE2EHookLLMStream(t *testing.T) {
	identity := testprepare.PrepareE2E(t)
	require.NotNil(t, identity)

	ast, err := assistant.Get("tests.hook-echo")
	require.NoError(t, err)
	require.NotNil(t, ast)
	require.NotNil(t, ast.HookScript, "hook-echo must have a HookScript")

	ctx := newE2ETestContext(t, identity, "chat-e2e-hook-llm", "tests.hook-echo")
	messages := []agentContext.Message{
		{Role: agentContext.RoleUser, Content: "trigger create hook"},
	}

	response, err := ast.Stream(ctx, messages)
	require.NoError(t, err, "Stream should not return an error in E2E with hook")
	require.NotNil(t, response, "response should not be nil")

	assert.NotEmpty(t, response.AssistantID)
	assert.NotEmpty(t, response.ChatID)
	assert.NotEmpty(t, response.ContextID)
	assert.NotEmpty(t, response.TraceID)
}

func TestE2ESandboxStream(t *testing.T) {
	identity := testprepare.PrepareE2E(t)
	require.NotNil(t, identity)

	ast, err := assistant.Get("tests.sandbox-v2.oneshot-cli")
	if err != nil {
		t.Fatalf("failed to load tests.sandbox-v2.oneshot-cli: %v (sandbox assistant must exist for E2E)", err)
	}
	require.NotNil(t, ast)

	ctx := newE2ETestContext(t, identity, "chat-e2e-sandbox", "tests.sandbox-v2.oneshot-cli")
	messages := []agentContext.Message{
		{Role: agentContext.RoleUser, Content: "echo hello world"},
	}

	response, err := ast.Stream(ctx, messages)
	require.NoError(t, err, "Sandbox Stream should not return an error in E2E")
	require.NotNil(t, response, "response should not be nil")

	assert.NotEmpty(t, response.AssistantID)
	assert.NotEmpty(t, response.ChatID)
	assert.NotEmpty(t, response.ContextID)
}
