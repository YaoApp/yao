package assistant_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent"
	"github.com/yaoapp/yao/agent/assistant"
	agentContext "github.com/yaoapp/yao/agent/context"
	agentsandbox "github.com/yaoapp/yao/agent/sandbox"
	"github.com/yaoapp/yao/agent/sandbox/claude"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/test"
)

// TestSandboxOptionsBuilding tests that sandbox options are correctly built from assistant config
func TestSandboxOptionsBuilding(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Load agent to ensure connectors are available
	err := agent.Load(config.Conf)
	require.NoError(t, err, "agent.Load should succeed")

	// Load the full test assistant
	ast, err := assistant.LoadPath("/assistants/tests/sandbox/full")
	require.NoError(t, err)
	require.NotNil(t, ast)

	// Verify sandbox is configured
	require.NotNil(t, ast.Sandbox)
	assert.Equal(t, "claude", ast.Sandbox.Command)
	assert.Equal(t, "5m", ast.Sandbox.Timeout)

	// Verify arguments are set
	require.NotNil(t, ast.Sandbox.Arguments)
	assert.Equal(t, float64(10), ast.Sandbox.Arguments["max_turns"])
	assert.Equal(t, "acceptEdits", ast.Sandbox.Arguments["permission_mode"])

	// Verify MCP configuration
	require.NotNil(t, ast.MCP)
	assert.Len(t, ast.MCP.Servers, 1)
	assert.Equal(t, "echo", ast.MCP.Servers[0].ServerID)

	t.Logf("Sandbox config: command=%s, timeout=%s", ast.Sandbox.Command, ast.Sandbox.Timeout)
	t.Logf("Sandbox arguments: %v", ast.Sandbox.Arguments)
	t.Logf("MCP servers: %v", ast.MCP.Servers)
}

// TestClaudeCommandBuilding tests that Claude CLI commands are correctly built
func TestClaudeCommandBuilding(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Create test messages
	messages := []agentContext.Message{
		{Role: "system", Content: "You are a helpful coding assistant."},
		{Role: "user", Content: "Hello, how are you?"},
	}

	// Create options similar to what buildSandboxOptions would produce
	opts := &claude.Options{
		Command:       "claude",
		UserID:        "test-user",
		ChatID:        "test-chat",
		ConnectorHost: "https://ark.cn-beijing.volces.com/api/v3",
		ConnectorKey:  "test-api-key",
		Model:         "ep-xxxxx",
		Arguments: map[string]interface{}{
			"max_turns":       10,
			"permission_mode": "acceptEdits",
		},
	}

	// Build the command
	cmd, env, err := claude.BuildCommand(messages, opts)
	require.NoError(t, err)

	// Verify command structure
	// Command is now: ["bash", "-c", "nohup ccr start ... && ccr code ..."]
	assert.NotEmpty(t, cmd)
	assert.Equal(t, "bash", cmd[0], "Command should start with bash")
	assert.Equal(t, "-c", cmd[1], "Second arg should be -c")
	assert.Contains(t, cmd[2], "ccr code", "Bash command should contain ccr code")
	assert.Contains(t, cmd[2], "--permission-mode", "Should include permission mode")
	t.Logf("Built command: %v", cmd)

	// Verify environment variables
	assert.NotEmpty(t, env)
	assert.Equal(t, "https://ark.cn-beijing.volces.com/api/v3", env["CCR_API_BASE"])
	assert.Equal(t, "test-api-key", env["CCR_API_KEY"])
	assert.Equal(t, "ep-xxxxx", env["CCR_MODEL"])
	assert.Equal(t, "10", env["CLAUDE_MAX_TURNS"])
	assert.Equal(t, "acceptEdits", env["CLAUDE_PERMISSION_MODE"])
	assert.Equal(t, "stream-json", env["CLAUDE_OUTPUT_FORMAT"])
	assert.Contains(t, env["CLAUDE_SYSTEM_PROMPT"], "You are a helpful coding assistant")
	t.Logf("Built environment: %v", env)
}

// TestClaudeCCRConfigBuilding tests that CCR config is correctly built
func TestClaudeCCRConfigBuilding(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	opts := &claude.Options{
		ConnectorHost: "https://ark.cn-beijing.volces.com/api/v3",
		ConnectorKey:  "test-api-key",
		Model:         "ep-xxxxx",
	}

	configJSON, err := claude.BuildCCRConfig(opts)
	require.NoError(t, err)
	require.NotEmpty(t, configJSON)

	t.Logf("CCR config: %s", string(configJSON))

	// Verify the JSON contains expected fields (CCR uses snake_case)
	assert.Contains(t, string(configJSON), "api_base_url")
	assert.Contains(t, string(configJSON), "api_key")
	assert.Contains(t, string(configJSON), "models")
	// Verify CCR format fields
	assert.Contains(t, string(configJSON), "Providers")
	assert.Contains(t, string(configJSON), "Router")
	assert.Contains(t, string(configJSON), "volcengine")
}

// TestDefaultImageSelection tests that default images are correctly selected
func TestDefaultImageSelection(t *testing.T) {
	tests := []struct {
		command       string
		expectedImage string
	}{
		{"claude", "yaoapp/sandbox-claude:latest"},
		{"cursor", "yaoapp/sandbox-cursor:latest"},
		{"unknown", ""},
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			image := agentsandbox.DefaultImage(tt.command)
			assert.Equal(t, tt.expectedImage, image)
		})
	}
}

// TestSandboxCommandValidation tests that command validation works correctly
func TestSandboxCommandValidation(t *testing.T) {
	tests := []struct {
		command string
		valid   bool
	}{
		{"claude", true},
		{"cursor", true},
		{"invalid", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			result := agentsandbox.IsValidCommand(tt.command)
			assert.Equal(t, tt.valid, result)
		})
	}
}

// TestHasSandboxMethod tests the HasSandbox method on Assistant
func TestHasSandboxMethod(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Test assistant with sandbox
	astWithSandbox, err := assistant.LoadPath("/assistants/tests/sandbox/basic")
	require.NoError(t, err)
	assert.True(t, astWithSandbox.HasSandbox(), "Assistant with sandbox config should return true")

	// Test assistant without sandbox (fullfields doesn't have sandbox)
	astWithoutSandbox, err := assistant.LoadPath("/assistants/tests/fullfields")
	require.NoError(t, err)
	assert.False(t, astWithoutSandbox.HasSandbox(), "Assistant without sandbox config should return false")
}
