package assistant_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent"
	"github.com/yaoapp/yao/agent/assistant"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/test"
)

// TestLoadSandboxBasicAssistant tests loading the basic sandbox test assistant
func TestLoadSandboxBasicAssistant(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	ast, err := assistant.LoadPath("/assistants/tests/sandbox/basic")
	require.NoError(t, err)
	require.NotNil(t, ast)

	// Verify basic fields
	assert.Equal(t, "tests.sandbox.basic", ast.ID)
	assert.Equal(t, "Sandbox Basic Test", ast.Name)
	assert.Equal(t, "deepseek.v3", ast.Connector)

	// Verify sandbox configuration
	require.NotNil(t, ast.Sandbox, "Sandbox should be configured")
	assert.Equal(t, "claude", ast.Sandbox.Command)
	assert.Equal(t, "5m", ast.Sandbox.Timeout)

	// Verify HasSandbox returns true
	assert.True(t, ast.HasSandbox(), "HasSandbox should return true")
}

// TestLoadSandboxHooksAssistant tests loading the hooks sandbox test assistant
func TestLoadSandboxHooksAssistant(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	ast, err := assistant.LoadPath("/assistants/tests/sandbox/hooks")
	require.NoError(t, err)
	require.NotNil(t, ast)

	// Verify basic fields
	assert.Equal(t, "tests.sandbox.hooks", ast.ID)
	assert.Equal(t, "Sandbox Hooks Test", ast.Name)
	assert.Equal(t, "deepseek.v3", ast.Connector)

	// Verify sandbox configuration
	require.NotNil(t, ast.Sandbox, "Sandbox should be configured")
	assert.Equal(t, "claude", ast.Sandbox.Command)

	// Verify hooks are loaded
	assert.NotNil(t, ast.HookScript, "HookScript should be loaded")
}

// TestLoadSandboxFullAssistant tests loading the full sandbox test assistant with MCPs and Skills
func TestLoadSandboxFullAssistant(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Load agent to ensure MCPs are available
	err := agent.Load(config.Conf)
	require.NoError(t, err, "agent.Load should succeed")

	ast, err := assistant.LoadPath("/assistants/tests/sandbox/full")
	require.NoError(t, err)
	require.NotNil(t, ast)

	// Verify basic fields
	assert.Equal(t, "tests.sandbox.full", ast.ID)
	assert.Equal(t, "Sandbox Full Test", ast.Name)
	assert.Equal(t, "deepseek.v3", ast.Connector)

	// Verify sandbox configuration
	require.NotNil(t, ast.Sandbox, "Sandbox should be configured")
	assert.Equal(t, "claude", ast.Sandbox.Command)
	assert.Equal(t, "5m", ast.Sandbox.Timeout)

	// Verify sandbox arguments (command-specific options)
	require.NotNil(t, ast.Sandbox.Arguments, "Sandbox arguments should be configured")
	assert.Equal(t, float64(10), ast.Sandbox.Arguments["max_turns"])
	assert.Equal(t, "acceptEdits", ast.Sandbox.Arguments["permission_mode"])

	// Verify MCP configuration
	require.NotNil(t, ast.MCP, "MCP should be configured")
	require.NotNil(t, ast.MCP.Servers, "MCP.Servers should be configured")
	assert.Len(t, ast.MCP.Servers, 1, "Should have 1 MCP server configured")
	assert.Equal(t, "echo", ast.MCP.Servers[0].ServerID, "MCP server ID should be 'echo'")
	assert.Contains(t, ast.MCP.Servers[0].Tools, "ping", "MCP tools should contain 'ping'")
	assert.Contains(t, ast.MCP.Servers[0].Tools, "echo", "MCP tools should contain 'echo'")

	// Verify hooks are loaded
	assert.NotNil(t, ast.HookScript, "HookScript should be loaded")
}

// TestSandboxConfigValidation tests sandbox configuration validation
func TestSandboxConfigValidation(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	tests := []struct {
		name     string
		path     string
		hasError bool
	}{
		{
			name:     "Basic sandbox config",
			path:     "/assistants/tests/sandbox/basic",
			hasError: false,
		},
		{
			name:     "Hooks sandbox config",
			path:     "/assistants/tests/sandbox/hooks",
			hasError: false,
		},
		{
			name:     "Full sandbox config with MCPs",
			path:     "/assistants/tests/sandbox/full",
			hasError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ast, err := assistant.LoadPath(tt.path)
			if tt.hasError {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, ast)
			require.NotNil(t, ast.Sandbox)
			assert.NotEmpty(t, ast.Sandbox.Command)
		})
	}
}

// TestSkillsDirectoryResolution tests that skills directory exists and has correct structure
// Note: Skills are auto-discovered from skills/ directory, not stored in AssistantModel
func TestSkillsDirectoryResolution(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	ast, err := assistant.LoadPath("/assistants/tests/sandbox/full")
	require.NoError(t, err)
	require.NotNil(t, ast)

	// Get app root from environment
	appRoot := os.Getenv("YAO_ROOT")
	require.NotEmpty(t, appRoot, "YAO_ROOT should be set")

	// Verify assistant path is set
	assert.NotEmpty(t, ast.Path, "Assistant path should be set")

	// Build expected skills directory path
	// ast.Path is like "/assistants/tests/sandbox/full"
	expectedSkillsDir := filepath.Join(appRoot, ast.Path, "skills")

	// Verify skills directory exists
	info, err := os.Stat(expectedSkillsDir)
	require.NoError(t, err, "Skills directory should exist: %s", expectedSkillsDir)
	assert.True(t, info.IsDir(), "Skills path should be a directory")

	// Verify skills directory structure
	entries, err := os.ReadDir(expectedSkillsDir)
	require.NoError(t, err, "Should be able to read skills directory")

	// Find echo-test skill
	var foundEchoTest bool
	for _, entry := range entries {
		if entry.IsDir() && entry.Name() == "echo-test" {
			foundEchoTest = true

			// Verify SKILL.md exists (required)
			skillMdPath := filepath.Join(expectedSkillsDir, "echo-test", "SKILL.md")
			_, err := os.Stat(skillMdPath)
			assert.NoError(t, err, "SKILL.md should exist")

			// Verify scripts directory exists (optional but we created it)
			scriptsDir := filepath.Join(expectedSkillsDir, "echo-test", "scripts")
			_, err = os.Stat(scriptsDir)
			assert.NoError(t, err, "scripts directory should exist")

			// Verify echo.sh exists
			echoShPath := filepath.Join(scriptsDir, "echo.sh")
			_, err = os.Stat(echoShPath)
			assert.NoError(t, err, "echo.sh should exist")

			break
		}
	}
	assert.True(t, foundEchoTest, "echo-test skill should exist in skills directory")
}

// TestMCPConfiguration tests that MCP is correctly loaded for sandbox assistant
func TestMCPConfiguration(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Load agent to ensure MCPs are available
	err := agent.Load(config.Conf)
	require.NoError(t, err, "agent.Load should succeed")

	ast, err := assistant.LoadPath("/assistants/tests/sandbox/full")
	require.NoError(t, err)
	require.NotNil(t, ast)

	// Verify MCP configuration structure
	require.NotNil(t, ast.MCP, "MCP should not be nil")
	require.NotNil(t, ast.MCP.Servers, "MCP.Servers should not be nil")
	assert.Len(t, ast.MCP.Servers, 1, "Should have 1 MCP server configured")

	// Verify echo server configuration
	echoServer := ast.MCP.Servers[0]
	assert.Equal(t, "echo", echoServer.ServerID, "Server ID should be 'echo'")
	assert.Len(t, echoServer.Tools, 3, "Should have 3 tools configured")
	assert.Contains(t, echoServer.Tools, "ping")
	assert.Contains(t, echoServer.Tools, "echo")
	assert.Contains(t, echoServer.Tools, "status")
}
