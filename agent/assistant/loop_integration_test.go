//go:build integration

package assistant_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/assistant"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

func TestToolLoopConfig(t *testing.T) {
	testprepare.PrepareSandbox(t)

	ast, err := assistant.Get("tests.tool-loop")
	require.NoError(t, err)
	require.NotNil(t, ast)

	require.NotNil(t, ast.MCP, "tool-loop should have MCP configuration")
	require.NotNil(t, ast.MCP.Options, "tool-loop should have MCP options")

	disabled := assistant.ExportIsToolLoopDisabled(ast)
	assert.False(t, disabled, "tool_loop=true should not be disabled")

	maxTurns := assistant.ExportGetMaxToolLoopTurns(ast)
	assert.Equal(t, 3, maxTurns, "max_turn should be 3 as configured")
}

func TestToolLoopDisabledConfig(t *testing.T) {
	testprepare.PrepareSandbox(t)

	ast, err := assistant.Get("tests.tool-loop")
	require.NoError(t, err)
	require.NotNil(t, ast)

	ast.MCP.Options["tool_loop"] = false

	disabled := assistant.ExportIsToolLoopDisabled(ast)
	assert.True(t, disabled, "tool_loop=false should be disabled")
}
