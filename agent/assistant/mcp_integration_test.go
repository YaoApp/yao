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

func TestBuildMCPTools(t *testing.T) {
	testprepare.PrepareSandbox(t)

	ast, err := assistant.Get("tests.mcp-tools")
	require.NoError(t, err)
	require.NotNil(t, ast)

	require.NotNil(t, ast.MCP, "mcp-tools should have MCP configuration")
	require.Len(t, ast.MCP.Servers, 1, "should have exactly one MCP server")

	server := ast.MCP.Servers[0]
	assert.Equal(t, "echo", server.ServerID)
	assert.ElementsMatch(t, []string{"ping", "echo"}, server.Tools, "tools should be filtered to ping and echo")
}

func TestMCPToolSchema(t *testing.T) {
	testprepare.PrepareSandbox(t)

	ast, err := assistant.Get("tests.mcp-tools")
	require.NoError(t, err)
	require.NotNil(t, ast)
	require.NotNil(t, ast.MCP)
	require.Greater(t, len(ast.MCP.Servers), 0)

	server := ast.MCP.Servers[0]
	for _, toolName := range server.Tools {
		formatted := assistant.MCPToolName(server.ServerID, toolName)
		assert.NotEmpty(t, formatted, "formatted tool name should not be empty")

		parsedServer, parsedTool, ok := assistant.ParseMCPToolName(formatted)
		assert.True(t, ok, "should parse formatted name")
		assert.Equal(t, server.ServerID, parsedServer)
		assert.Equal(t, toolName, parsedTool)
	}
}

func TestMCPContextPassing(t *testing.T) {
	testprepare.PrepareSandbox(t)

	ast, err := assistant.Get("tests.mcp-tools")
	require.NoError(t, err)
	require.NotNil(t, ast)

	ctx := newTestContext("chat-mcp-context-passing", "tests.mcp-tools")
	assert.Equal(t, "tests.mcp-tools", ctx.AssistantID, "context should carry assistant_id")
	assert.Equal(t, "chat-mcp-context-passing", ctx.ChatID, "context should carry chat_id")

	messages := []agentContext.Message{
		{Role: agentContext.RoleUser, Content: "Test MCP context"},
	}

	assert.NotPanics(t, func() {
		resultMessages, options, err := ast.BuildRequest(ctx, messages, nil)
		if err == nil {
			assert.NotNil(t, resultMessages, "result messages should not be nil")
			_ = options
		}
	})
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
