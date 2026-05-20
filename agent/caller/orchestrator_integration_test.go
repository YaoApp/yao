//go:build integration

package caller_test

import (
	"bytes"
	stdContext "context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/caller"
	agentContext "github.com/yaoapp/yao/agent/context"
	oauthTypes "github.com/yaoapp/yao/openapi/oauth/types"
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

func newIntegrationTestContext(chatID, assistantID string) *agentContext.Context {
	authorized := &oauthTypes.AuthorizedInfo{
		Subject:    "test-user",
		ClientID:   "test-client-id",
		Scope:      "openid profile email",
		SessionID:  "test-session-id",
		UserID:     "test-user-123",
		TeamID:     "test-team-456",
		TenantID:   "test-tenant-789",
		RememberMe: true,
	}

	ctx := agentContext.New(stdContext.Background(), authorized, chatID)
	ctx.AssistantID = assistantID
	ctx.Writer = newMockResponseWriter()
	return ctx
}

func TestIntegration_Call_RealAgent(t *testing.T) {
	testprepare.PrepareSandbox(t)

	ctx := newIntegrationTestContext("test-orch-call", "tests.caller-orchestrator")
	defer ctx.Release()

	orch := caller.NewOrchestrator(ctx)
	results := orch.All([]*caller.Request{
		{
			AgentID: "tests.caller-target",
			Messages: []agentContext.Message{
				{Role: agentContext.RoleUser, Content: "hello"},
			},
		},
	})

	require.Len(t, results, 1)
	assert.Empty(t, results[0].Error)
	assert.NotNil(t, results[0].Response)
}

func TestIntegration_Any_RealAgent(t *testing.T) {
	testprepare.PrepareSandbox(t)

	ctx := newIntegrationTestContext("test-orch-any", "tests.caller-orchestrator")
	defer ctx.Release()

	orch := caller.NewOrchestrator(ctx)
	results := orch.Any([]*caller.Request{
		{
			AgentID: "tests.caller-target",
			Messages: []agentContext.Message{
				{Role: agentContext.RoleUser, Content: "hello from any"},
			},
		},
	})

	require.Len(t, results, 1)
	assert.Empty(t, results[0].Error)
	assert.NotNil(t, results[0].Response)
}

func TestIntegration_Race_RealAgent(t *testing.T) {
	testprepare.PrepareSandbox(t)

	ctx := newIntegrationTestContext("test-orch-race", "tests.caller-orchestrator")
	defer ctx.Release()

	orch := caller.NewOrchestrator(ctx)
	results := orch.Race([]*caller.Request{
		{
			AgentID: "tests.caller-target",
			Messages: []agentContext.Message{
				{Role: agentContext.RoleUser, Content: "hello from race"},
			},
		},
	})

	require.Len(t, results, 1)
	assert.Empty(t, results[0].Error)
	assert.NotNil(t, results[0].Response)
}

func TestIntegration_All_MultipleAgents(t *testing.T) {
	testprepare.PrepareSandbox(t)

	ctx := newIntegrationTestContext("test-orch-multi", "tests.caller-orchestrator")
	defer ctx.Release()

	orch := caller.NewOrchestrator(ctx)
	results := orch.All([]*caller.Request{
		{
			AgentID: "tests.caller-target",
			Messages: []agentContext.Message{
				{Role: agentContext.RoleUser, Content: "first call"},
			},
		},
		{
			AgentID: "tests.caller-target",
			Messages: []agentContext.Message{
				{Role: agentContext.RoleUser, Content: "second call"},
			},
		},
	})

	require.Len(t, results, 2)
	assert.Empty(t, results[0].Error)
	assert.Empty(t, results[1].Error)
	assert.NotNil(t, results[0].Response)
	assert.NotNil(t, results[1].Response)
}

func TestIntegration_All_NonexistentAgent(t *testing.T) {
	testprepare.PrepareSandbox(t)

	ctx := newIntegrationTestContext("test-orch-notfound", "tests.caller-orchestrator")
	defer ctx.Release()

	orch := caller.NewOrchestrator(ctx)
	results := orch.All([]*caller.Request{
		{
			AgentID: "nonexistent.agent.xyz",
			Messages: []agentContext.Message{
				{Role: agentContext.RoleUser, Content: "hello"},
			},
		},
	})

	require.Len(t, results, 1)
	assert.NotEmpty(t, results[0].Error, "should have error for nonexistent agent")
}
