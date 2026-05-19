//go:build integration

package assistant_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/assistant"
	agentContext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

func TestMCPToolNameFormat(t *testing.T) {
	testprepare.PrepareSandbox(t)

	ast, err := assistant.Get("tests.fullfields")
	require.NoError(t, err)
	require.NotNil(t, ast)

	require.NotNil(t, ast.MCP, "fullfields should have MCP configuration")
	require.Greater(t, len(ast.MCP.Servers), 0, "fullfields should have at least one MCP server")

	firstServer := ast.MCP.Servers[0]
	assert.NotEmpty(t, firstServer.ServerID, "first MCP server should have a ServerID")
}

func TestBuildRequestWithMCP(t *testing.T) {
	testprepare.PrepareSandbox(t)

	ast, err := assistant.Get("tests.fullfields")
	require.NoError(t, err)
	require.NotNil(t, ast)

	ctx := newTestContext("chat-mcp-build-request", "tests.fullfields")

	messages := []agentContext.Message{
		{Role: agentContext.RoleUser, Content: "Hello"},
	}

	// BuildRequest should not panic even if MCP server is unavailable in test env
	assert.NotPanics(t, func() {
		resultMessages, options, err := ast.BuildRequest(ctx, messages, nil)
		// BuildRequest may return an error if connector is not available,
		// but it should never panic
		if err == nil {
			assert.NotNil(t, resultMessages, "result messages should not be nil")
			_ = options // options may or may not have tools depending on MCP availability
		}
	})
}
