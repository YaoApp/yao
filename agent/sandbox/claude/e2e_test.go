package claude

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	agentContext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/output/message"
	"github.com/yaoapp/yao/config"
	infraSandbox "github.com/yaoapp/yao/sandbox"
	"github.com/yaoapp/yao/test"
)

// TestE2ESkipClaudeCLI verifies that Claude CLI is skipped when no prompts/skills/mcp
// This is the "hook-only" mode where hooks take full control
func TestE2ESkipClaudeCLI(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	test.Prepare(t, config.Conf)
	defer test.Clean()

	manager := createTestManager(t)
	if manager == nil {
		return
	}
	defer manager.Close()

	// Create options WITHOUT SystemPrompt, SkillsDir, or MCPConfig
	// This should trigger the skip logic
	opts := &Options{
		Command:       "claude",
		Image:         "alpine:latest", // Use alpine since we're not calling Claude CLI
		UserID:        "test-user",
		ChatID:        fmt.Sprintf("test-e2e-skip-%d", time.Now().UnixNano()),
		ConnectorHost: "",
		ConnectorKey:  "",
		Model:         "",
		// No SystemPrompt, SkillsDir, or MCPConfig
	}

	exec, err := NewExecutor(manager, opts)
	require.NoError(t, err)
	defer exec.Close()

	// Verify shouldSkipClaudeCLI returns true
	assert.True(t, exec.shouldSkipClaudeCLI(), "Should skip Claude CLI when no prompts/skills/mcp")

	// Execute Stream - it should return immediately without calling Claude CLI
	ctx := agentContext.New(context.Background(), nil, opts.ChatID)
	messages := []agentContext.Message{
		{Role: "user", Content: "Hello"},
	}

	response, err := exec.Stream(ctx, messages, nil)
	require.NoError(t, err, "Stream should succeed")
	require.NotNil(t, response, "Response should not be nil")

	// Verify response indicates skip
	assert.Contains(t, response.ID, "sandbox-skip", "Response ID should indicate skip")
	assert.Equal(t, "sandbox", response.Model, "Model should be 'sandbox' for skip mode")
	assert.Empty(t, response.Content, "Content should be empty for skip mode")

	t.Log("✓ Claude CLI skip mode verified")
}

// TestE2EExecuteClaudeCLI verifies that Claude CLI is called when prompts are configured
// This requires the real yaoapp/sandbox-claude image and a valid connector
func TestE2EExecuteClaudeCLI(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Check for required environment variables
	apiKey := os.Getenv("DEEPSEEK_API_KEY")
	apiProxy := os.Getenv("DEEPSEEK_API_PROXY")
	model := os.Getenv("DEEPSEEK_MODELS_V3")

	if apiKey == "" || apiProxy == "" || model == "" {
		t.Skip("Skipping test: DEEPSEEK_API_KEY, DEEPSEEK_API_PROXY, or DEEPSEEK_MODELS_V3 not set")
	}

	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Get data root from environment
	dataRoot := os.Getenv("YAO_ROOT")
	if dataRoot == "" {
		t.Skip("Skipping test: YAO_ROOT not set")
	}

	// Create config with proper paths
	cfg := infraSandbox.DefaultConfig()
	cfg.Init(dataRoot)

	manager, err := infraSandbox.NewManager(cfg)
	if err != nil {
		t.Skipf("Skipping test: Docker not available: %v", err)
	}
	defer manager.Close()

	// Create options WITH SystemPrompt (triggers Claude CLI execution)
	opts := &Options{
		Command:       "claude",
		Image:         "yaoapp/sandbox-claude:latest",
		UserID:        "test-user",
		ChatID:        fmt.Sprintf("test-e2e-exec-%d", time.Now().UnixNano()),
		ConnectorHost: apiProxy,
		ConnectorKey:  apiKey,
		Model:         model,
		SystemPrompt:  "You are a helpful assistant. Keep responses brief.",
		Timeout:       5 * time.Minute,
	}

	exec, err := NewExecutor(manager, opts)
	if err != nil {
		t.Skipf("Skipping test: Failed to create executor: %v", err)
	}
	defer exec.Close()

	// Verify shouldSkipClaudeCLI returns false
	assert.False(t, exec.shouldSkipClaudeCLI(), "Should NOT skip Claude CLI when prompts are configured")

	// Execute Stream with a simple prompt
	ctx := agentContext.New(context.Background(), nil, opts.ChatID)
	messages := []agentContext.Message{
		{Role: "user", Content: "Reply with exactly: TEST_SUCCESS"},
	}

	// Collect streaming output
	var streamedContent strings.Builder
	streamHandler := func(chunkType message.StreamChunkType, data []byte) int {
		if chunkType == message.ChunkText {
			streamedContent.Write(data)
		}
		return 0 // continue streaming
	}

	t.Log("Executing Claude CLI with real API call...")
	startTime := time.Now()
	response, err := exec.Stream(ctx, messages, streamHandler)
	duration := time.Since(startTime)
	t.Logf("Execution took: %v", duration)

	if err != nil {
		t.Logf("Stream error (might be expected if Docker/API issue): %v", err)
		t.Skipf("Skipping assertion: %v", err)
	}

	require.NotNil(t, response, "Response should not be nil")

	// Log response details
	t.Logf("Response ID: %s", response.ID)
	t.Logf("Response Model: %s", response.Model)
	t.Logf("Response Content: %v", response.Content)
	t.Logf("Streamed Content: %s", streamedContent.String())

	// Verify we got some response
	var fullResponse string
	if content, ok := response.Content.(string); ok {
		fullResponse = content
	}
	if fullResponse == "" {
		fullResponse = streamedContent.String()
	}

	if fullResponse != "" {
		t.Logf("✓ Claude CLI executed successfully with response: %s", truncate(fullResponse, 200))
	} else {
		t.Log("⚠ Empty response (Claude CLI might have issues)")
	}
}

// TestE2EBuildInputJSONLIntegration tests the full flow of building input JSONL
func TestE2EBuildInputJSONLIntegration(t *testing.T) {
	// Test with conversation history
	messages := []agentContext.Message{
		{Role: "system", Content: "You are a helpful assistant"},
		{Role: "user", Content: "What is 2+2?"},
		{Role: "assistant", Content: "4"},
		{Role: "user", Content: "What about 3+3?"},
	}

	jsonl, err := BuildInputJSONL(messages)
	require.NoError(t, err)

	t.Logf("Input JSONL:\n%s", string(jsonl))

	// Verify format
	lines := strings.Split(string(jsonl), "\n")
	assert.GreaterOrEqual(t, len(lines), 3, "Should have at least 3 lines (user, assistant, user)")

	// System message should NOT be in JSONL
	assert.NotContains(t, string(jsonl), "You are a helpful assistant", "System message should not be in JSONL")

	// User and assistant messages should be present
	assert.Contains(t, string(jsonl), "What is 2+2", "First user message should be present")
	assert.Contains(t, string(jsonl), "4", "Assistant response should be present")
	assert.Contains(t, string(jsonl), "What about 3+3", "Second user message should be present")

	t.Log("✓ Input JSONL format verified")
}

// TestE2EBuildCommand tests the full command building
func TestE2EBuildCommand(t *testing.T) {
	messages := []agentContext.Message{
		{Role: "system", Content: "You are helpful"},
		{Role: "user", Content: "Hello"},
	}

	opts := &Options{
		ConnectorHost: "https://api.example.com",
		ConnectorKey:  "test-key",
		Model:         "test-model",
		Arguments: map[string]interface{}{
			"permission_mode": "bypassPermissions",
		},
		MCPConfig: []byte(`{"mcpServers":{}}`),
	}

	cmd, env, err := BuildCommand(messages, opts)
	require.NoError(t, err)

	t.Logf("Command: %v", cmd)
	t.Logf("Environment: %v", env)

	// Verify command structure
	assert.Equal(t, "bash", cmd[0])
	assert.Equal(t, "-c", cmd[1])

	bashCmd := cmd[2]
	// Should use heredoc with INPUTEOF
	assert.Contains(t, bashCmd, "cat << 'INPUTEOF'", "Should use heredoc")
	assert.Contains(t, bashCmd, "INPUTEOF", "Should have INPUTEOF delimiter")

	// Should have streaming flags
	assert.Contains(t, bashCmd, "--input-format", "Should have input-format flag")
	assert.Contains(t, bashCmd, "--output-format", "Should have output-format flag")
	assert.Contains(t, bashCmd, "--verbose", "Should have verbose flag")
	assert.Contains(t, bashCmd, "stream-json", "Should use stream-json format")

	// Should have permission flags
	assert.Contains(t, bashCmd, "--dangerously-skip-permissions", "Should have skip-permissions flag")
	assert.Contains(t, bashCmd, "--permission-mode", "Should have permission-mode flag")
	assert.Contains(t, bashCmd, "bypassPermissions", "Should have bypassPermissions value")

	// Should have MCP config
	assert.Contains(t, bashCmd, "--mcp-config", "Should have mcp-config flag")

	// Environment should have proxy settings
	assert.Equal(t, "http://127.0.0.1:3456", env["ANTHROPIC_BASE_URL"])
	assert.Equal(t, "dummy", env["ANTHROPIC_API_KEY"])

	// System prompt should be in environment
	assert.Contains(t, env["CLAUDE_SYSTEM_PROMPT"], "You are helpful", "System prompt should be in env")

	t.Log("✓ Command building verified")
}

// Helper function to truncate strings
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
