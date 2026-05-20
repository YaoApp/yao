//go:build integration

package assistant_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/assistant"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

func TestLoadScriptsFullFields(t *testing.T) {
	testprepare.PrepareSandbox(t)

	ast, err := assistant.Get("tests.fullfields")
	require.NoError(t, err)
	require.NotNil(t, ast)

	assert.NotNil(t, ast.HookScript, "fullfields has src/index.ts, HookScript should not be nil")
}

func TestLoadScriptsNoScript(t *testing.T) {
	testprepare.PrepareSandbox(t)

	ast, err := assistant.Get("tests.no-prompt")
	require.NoError(t, err)
	require.NotNil(t, ast)

	assert.Nil(t, ast.HookScript, "no-prompt has no src directory, HookScript should be nil")
	assert.True(t, ast.Scripts == nil || len(ast.Scripts) == 0,
		"no-prompt has no src directory, Scripts should be nil or empty")
}

func TestLoadScriptsHookEcho(t *testing.T) {
	testprepare.PrepareSandbox(t)

	ast, err := assistant.Get("tests.hook-echo")
	require.NoError(t, err)
	require.NotNil(t, ast)

	assert.NotNil(t, ast.HookScript, "hook-echo has src/index.ts, HookScript should not be nil")
}
