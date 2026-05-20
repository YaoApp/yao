//go:build integration

package assistant_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/assistant"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

func TestLoadPathFullFields(t *testing.T) {
	testprepare.PrepareSandbox(t)

	ast, err := assistant.LoadPath("/assistants/tests/fullfields")
	require.NoError(t, err)
	require.NotNil(t, ast)

	assert.Equal(t, "tests.fullfields", ast.ID)
	assert.Equal(t, "Full Fields Test", ast.Name)
	assert.Equal(t, "assistant", ast.Type)
	assert.Equal(t, "openai.mock", ast.Connector)
	assert.NotEmpty(t, ast.Description)

	// Options
	require.NotNil(t, ast.Options)
	assert.Equal(t, 0.7, ast.Options["temperature"])

	// Tags
	require.NotNil(t, ast.Tags)
	assert.Contains(t, ast.Tags, "Test")
	assert.Contains(t, ast.Tags, "FullFields")

	// Sort, Public, Mentionable
	assert.Equal(t, 100, ast.Sort)
	assert.True(t, ast.Public)
	assert.True(t, ast.Mentionable)

	// ConnectorOptions
	require.NotNil(t, ast.ConnectorOptions)
	require.NotNil(t, ast.ConnectorOptions.Optional)
	assert.True(t, *ast.ConnectorOptions.Optional)

	// KB
	require.NotNil(t, ast.KB)
	require.NotNil(t, ast.KB.Collections)
	assert.Contains(t, ast.KB.Collections, "test-collection")

	// MCP
	require.NotNil(t, ast.MCP)
	assert.Greater(t, len(ast.MCP.Servers), 0)

	// Search
	require.NotNil(t, ast.Search)
	require.NotNil(t, ast.Search.Web)

	// Uses
	require.NotNil(t, ast.Uses)
	assert.Equal(t, "builtin", ast.Uses.Search)

	// Prompts
	require.NotNil(t, ast.Prompts)
	assert.Greater(t, len(ast.Prompts), 0)

	// PromptPresets
	require.NotNil(t, ast.PromptPresets)
	assert.Contains(t, ast.PromptPresets, "chat.prompts")
	assert.Contains(t, ast.PromptPresets, "task.prompts")

	// Placeholder
	require.NotNil(t, ast.Placeholder)

	// Modes
	require.NotNil(t, ast.Modes)
	assert.Contains(t, ast.Modes, "chat")
	assert.Contains(t, ast.Modes, "task")
	assert.Equal(t, "chat", ast.DefaultMode)

	// HookScript
	assert.NotNil(t, ast.HookScript)
}

func TestLoadPathSimple(t *testing.T) {
	testprepare.PrepareSandbox(t)

	ast, err := assistant.LoadPath("/assistants/tests/simple-greeting")
	require.NoError(t, err)
	require.NotNil(t, ast)

	assert.Equal(t, "tests.simple-greeting", ast.ID)
	assert.NotEmpty(t, ast.Name)
}

func TestLoadPathNoPrompt(t *testing.T) {
	testprepare.PrepareSandbox(t)

	ast, err := assistant.LoadPath("/assistants/tests/no-prompt")
	require.NoError(t, err)
	require.NotNil(t, ast)

	assert.True(t, ast.Prompts == nil || len(ast.Prompts) == 0)
	assert.Nil(t, ast.MCP)
	assert.Nil(t, ast.HookScript)
}

func TestLoadStoreBasic(t *testing.T) {
	testprepare.PrepareSandbox(t)

	ast, err := assistant.Get("tests.fullfields")
	require.NoError(t, err)
	require.NotNil(t, ast)

	// Second call should return the cached version
	ast2, err := assistant.Get("tests.fullfields")
	require.NoError(t, err)
	require.NotNil(t, ast2)
	assert.Equal(t, ast.ID, ast2.ID)
}

func TestLoadStoreNotFound(t *testing.T) {
	testprepare.PrepareSandbox(t)

	_, err := assistant.Get("nonexistent.assistant.id")
	assert.Error(t, err)
}

func TestLoadBuiltIn(t *testing.T) {
	testprepare.PrepareSandbox(t)

	err := assistant.LoadBuiltIn()
	require.NoError(t, err)

	cache := assistant.GetCache()
	require.NotNil(t, cache)
	assert.Greater(t, cache.Len(), 0)
}

func TestLoadPathDisableGlobalPrompts(t *testing.T) {
	testprepare.PrepareSandbox(t)

	ast, err := assistant.LoadPath("/assistants/tests/hook-disable-global-prompts")
	require.NoError(t, err)
	require.NotNil(t, ast)

	assert.True(t, ast.DisableGlobalPrompts)
}
