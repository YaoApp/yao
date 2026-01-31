package claude

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	agentContext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/output/message"
	"github.com/yaoapp/yao/config"
	infraSandbox "github.com/yaoapp/yao/sandbox"
	"github.com/yaoapp/yao/test"
)

// TestRealClaudeCLIExecution tests real Claude CLI execution with streaming
// This test requires:
// 1. Docker running with yaoapp/sandbox-claude:latest image
// 2. Environment variables: DEEPSEEK_API_KEY, DEEPSEEK_API_PROXY, DEEPSEEK_MODELS_V3
func TestRealClaudeCLIExecution(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping real E2E test in short mode")
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
		ChatID:        fmt.Sprintf("test-real-e2e-%d", time.Now().UnixNano()),
		ConnectorHost: apiProxy,
		ConnectorKey:  apiKey,
		Model:         model,
		SystemPrompt:  "You are a helpful assistant. Reply concisely.",
		Timeout:       3 * time.Minute,
	}

	t.Logf("Creating executor with options:")
	t.Logf("  ConnectorHost: %s", opts.ConnectorHost)
	t.Logf("  Model: %s", opts.Model)
	t.Logf("  SystemPrompt: %s", opts.SystemPrompt)

	exec, err := NewExecutor(manager, opts)
	if err != nil {
		t.Skipf("Skipping test: Failed to create executor: %v", err)
	}
	defer exec.Close()

	// Verify shouldSkipClaudeCLI returns false
	if exec.shouldSkipClaudeCLI() {
		t.Fatal("shouldSkipClaudeCLI should return false when SystemPrompt is set")
	}

	// Test 1: First, manually test claude-proxy
	t.Log("=== Test 1: Verify claude-proxy is working ===")
	stdCtx := context.Background()

	// Prepare environment (this starts claude-proxy)
	err = exec.prepareEnvironment(stdCtx)
	require.NoError(t, err, "prepareEnvironment should succeed")

	// Check proxy is running
	result, err := exec.manager.Exec(stdCtx, exec.containerName, []string{"pgrep", "-f", "claude-proxy"}, nil)
	if err != nil || result.ExitCode != 0 {
		t.Log("claude-proxy not running, checking why...")
		// Check proxy log
		logContent, _ := exec.ReadFile(stdCtx, "proxy.log")
		t.Logf("Proxy log: %s", string(logContent))

		// Check config
		configContent, _ := exec.ReadFile(stdCtx, ".claude-proxy.json")
		t.Logf("Proxy config: %s", string(configContent))
	} else {
		t.Logf("claude-proxy is running with PID: %s", strings.TrimSpace(result.Stdout))
	}

	// Test 2: Test simple command execution
	t.Log("=== Test 2: Simple command execution ===")
	ctx := agentContext.New(stdCtx, nil, opts.ChatID)
	messages := []agentContext.Message{
		{Role: "user", Content: "Reply with exactly: HELLO_TEST_SUCCESS"},
	}

	// Collect streaming output
	var streamedChunks []string
	var streamedContent strings.Builder
	streamHandler := func(chunkType message.StreamChunkType, data []byte) int {
		chunk := string(data)
		streamedChunks = append(streamedChunks, chunk)
		streamedContent.Write(data)
		t.Logf("Stream chunk [%s]: %q", chunkType, chunk)
		return 0 // continue streaming
	}

	t.Log("Executing Claude CLI...")
	startTime := time.Now()
	response, err := exec.Stream(ctx, messages, streamHandler)
	duration := time.Since(startTime)
	t.Logf("Execution took: %v", duration)

	if err != nil {
		t.Logf("Stream error: %v", err)

		// Debug: check what's in the container
		t.Log("=== Debug info ===")

		// Check proxy log
		logContent, _ := exec.ReadFile(stdCtx, "proxy.log")
		t.Logf("Proxy log:\n%s", string(logContent))

		// List workspace
		output, _ := exec.Exec(stdCtx, []string{"ls", "-la", "/workspace"})
		t.Logf("Workspace contents:\n%s", output)

		// Check environment
		output, _ = exec.Exec(stdCtx, []string{"env"})
		t.Logf("Environment:\n%s", output)

		t.Fatalf("Stream failed: %v", err)
	}

	require.NotNil(t, response, "Response should not be nil")

	// Log results
	t.Logf("=== Results ===")
	t.Logf("Response ID: %s", response.ID)
	t.Logf("Response Model: %s", response.Model)
	t.Logf("Response Content: %v", response.Content)
	t.Logf("Streamed chunks count: %d", len(streamedChunks))
	t.Logf("Total streamed content: %s", streamedContent.String())

	// Verify we got some response
	var fullResponse string
	if content, ok := response.Content.(string); ok {
		fullResponse = content
	}
	if fullResponse == "" {
		fullResponse = streamedContent.String()
	}

	if fullResponse == "" {
		// Check proxy log for errors
		logContent, _ := exec.ReadFile(stdCtx, "proxy.log")
		t.Logf("Proxy log (for debugging):\n%s", string(logContent))
		t.Fatal("Got empty response from Claude CLI")
	}

	t.Logf("✓ Successfully got response: %s", fullResponse)

	// Check if streaming worked
	if len(streamedChunks) > 0 {
		t.Logf("✓ Streaming worked with %d chunks", len(streamedChunks))
	} else {
		t.Log("⚠ No streaming chunks received (might be buffered)")
	}
}

// TestClaudeCLIDirectExecution tests running claude directly in the container
func TestClaudeCLIDirectExecution(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping real E2E test in short mode")
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

	dataRoot := os.Getenv("YAO_ROOT")
	if dataRoot == "" {
		t.Skip("Skipping test: YAO_ROOT not set")
	}

	cfg := infraSandbox.DefaultConfig()
	cfg.Init(dataRoot)

	manager, err := infraSandbox.NewManager(cfg)
	if err != nil {
		t.Skipf("Skipping test: Docker not available: %v", err)
	}
	defer manager.Close()

	opts := &Options{
		Command:       "claude",
		Image:         "yaoapp/sandbox-claude:latest",
		UserID:        "test-user",
		ChatID:        fmt.Sprintf("test-direct-%d", time.Now().UnixNano()),
		ConnectorHost: apiProxy,
		ConnectorKey:  apiKey,
		Model:         model,
		Timeout:       3 * time.Minute,
	}

	exec, err := NewExecutor(manager, opts)
	if err != nil {
		t.Skipf("Skipping test: Failed to create executor: %v", err)
	}
	defer exec.Close()

	stdCtx := context.Background()

	// Step 1: Write proxy config and start proxy
	t.Log("=== Step 1: Start claude-proxy ===")
	err = exec.prepareEnvironment(stdCtx)
	require.NoError(t, err)

	// Wait for proxy to start
	time.Sleep(2 * time.Second)

	// Check proxy status
	result, err := exec.manager.Exec(stdCtx, exec.containerName, []string{"pgrep", "-f", "claude-proxy"}, nil)
	if err == nil && result.ExitCode == 0 {
		t.Logf("✓ claude-proxy running, PID: %s", strings.TrimSpace(result.Stdout))
	} else {
		t.Log("⚠ claude-proxy might not be running")
	}

	// Step 2: Run claude CLI directly with simple prompt
	t.Log("=== Step 2: Run claude CLI directly ===")

	// Build a simple command - pass env vars explicitly
	directCmd := []string{
		"bash", "-c",
		`echo '{"type":"user","message":{"role":"user","content":"say hello"}}' | claude -p --dangerously-skip-permissions --permission-mode bypassPermissions --input-format stream-json --output-format stream-json --verbose 2>&1`,
	}

	reader, err := exec.manager.Stream(stdCtx, exec.containerName, directCmd, &infraSandbox.ExecOptions{
		WorkDir: exec.workDir,
		Timeout: 2 * time.Minute,
		Env: map[string]string{
			"ANTHROPIC_BASE_URL": "http://127.0.0.1:3456",
			"ANTHROPIC_API_KEY":  "dummy",
		},
	})
	if err != nil {
		t.Fatalf("Failed to execute: %v", err)
	}
	defer reader.Close()

	// Read output
	buf := make([]byte, 64*1024)
	var output strings.Builder
	for {
		n, err := reader.Read(buf)
		if n > 0 {
			chunk := string(buf[:n])
			output.WriteString(chunk)
			t.Logf("Output chunk: %q", chunk)
		}
		if err != nil {
			break
		}
	}

	t.Logf("=== Full output ===\n%s", output.String())

	if output.Len() == 0 {
		// Check logs
		logContent, _ := exec.ReadFile(stdCtx, "proxy.log")
		t.Logf("Proxy log:\n%s", string(logContent))
		t.Fatal("Got no output from claude CLI")
	}

	// Check for success indicators
	outputStr := output.String()
	if strings.Contains(outputStr, "error") || strings.Contains(outputStr, "Error") {
		t.Logf("⚠ Output contains error")
	}
	if strings.Contains(outputStr, "content_block") || strings.Contains(outputStr, "message_start") {
		t.Log("✓ Got streaming JSON output from Claude CLI")
	}
}
