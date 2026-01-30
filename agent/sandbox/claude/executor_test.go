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
	"github.com/yaoapp/yao/config"
	infraSandbox "github.com/yaoapp/yao/sandbox"
	"github.com/yaoapp/yao/test"
)

// createTestManager creates a sandbox manager for testing with proper configuration
func createTestManager(t *testing.T) *infraSandbox.Manager {
	// Get data root from environment or use temp directory
	dataRoot := os.Getenv("YAO_ROOT")
	if dataRoot == "" {
		dataRoot = t.TempDir()
	}

	// Create config with proper paths
	cfg := infraSandbox.DefaultConfig()
	cfg.Init(dataRoot)

	manager, err := infraSandbox.NewManager(cfg)
	if err != nil {
		t.Skipf("Skipping test: Docker not available: %v", err)
		return nil
	}

	return manager
}

func TestNewClaudeExecutor(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	manager := createTestManager(t)
	if manager == nil {
		return
	}
	defer manager.Close()

	opts := &Options{
		Command:       "claude",
		Image:         "yaoapp/sandbox-claude:latest",
		UserID:        "test-user",
		ChatID:        fmt.Sprintf("test-chat-claude-%d", time.Now().UnixNano()),
		ConnectorHost: "https://api.example.com",
		ConnectorKey:  "key123",
		Model:         "test-model",
	}

	exec, err := NewExecutor(manager, opts)
	require.NoError(t, err)
	require.NotNil(t, exec)

	// Verify executor was created
	assert.Equal(t, "/workspace", exec.GetWorkDir())
	assert.NoError(t, exec.Close())
}

func TestClaudeExecutorMissingRequiredFields(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	manager := createTestManager(t)
	if manager == nil {
		return
	}
	defer manager.Close()

	// Missing UserID
	_, err := NewExecutor(manager, &Options{
		Command: "claude",
		ChatID:  "test-chat",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "UserID is required")

	// Missing ChatID
	_, err = NewExecutor(manager, &Options{
		Command: "claude",
		UserID:  "test-user",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ChatID is required")
}

func TestClaudeExecutorFileOperations(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	manager := createTestManager(t)
	if manager == nil {
		return
	}
	defer manager.Close()

	opts := &Options{
		Command: "claude",
		Image:   "alpine:latest", // Use alpine for simpler testing
		UserID:  "test-user",
		ChatID:  fmt.Sprintf("test-chat-file-ops-%d", time.Now().UnixNano()),
	}

	exec, err := NewExecutor(manager, opts)
	require.NoError(t, err)
	defer exec.Close()

	ctx := context.Background()

	// Test WriteFile
	content := []byte("Hello, World!")
	err = exec.WriteFile(ctx, "test-file.txt", content)
	require.NoError(t, err)

	// Test ReadFile
	readContent, err := exec.ReadFile(ctx, "test-file.txt")
	require.NoError(t, err)
	assert.Equal(t, content, readContent)

	// Test ListDir
	files, err := exec.ListDir(ctx, ".")
	require.NoError(t, err)
	assert.True(t, len(files) > 0, "Expected at least one file in directory")

	// Find our test file
	var found bool
	for _, f := range files {
		if f.Name == "test-file.txt" {
			found = true
			break
		}
	}
	assert.True(t, found, "Expected to find test-file.txt in directory listing")
}

func TestClaudeExecutorExec(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	manager := createTestManager(t)
	if manager == nil {
		return
	}
	defer manager.Close()

	opts := &Options{
		Command: "claude",
		Image:   "alpine:latest", // Use alpine for simpler testing
		UserID:  "test-user",
		ChatID:  fmt.Sprintf("test-chat-exec-%d", time.Now().UnixNano()),
	}

	exec, err := NewExecutor(manager, opts)
	require.NoError(t, err)
	defer exec.Close()

	ctx := context.Background()

	// Test simple echo command
	output, err := exec.Exec(ctx, []string{"echo", "hello-world"})
	require.NoError(t, err)
	assert.Contains(t, output, "hello-world")
}

// TestClaudeExecutorMCPConfigWrite tests that MCP config is correctly written to container
func TestClaudeExecutorMCPConfigWrite(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	manager := createTestManager(t)
	if manager == nil {
		return
	}
	defer manager.Close()

	// Create MCP config JSON
	mcpConfig := []byte(`{"mcpServers":{"echo":{"command":"yao-mcp-proxy","args":["echo"],"tools":["ping","echo"]}}}`)

	opts := &Options{
		Command:   "claude",
		Image:     "alpine:latest",
		UserID:    "test-user",
		ChatID:    fmt.Sprintf("test-chat-mcp-write-%d", time.Now().UnixNano()),
		MCPConfig: mcpConfig,
	}

	exec, err := NewExecutor(manager, opts)
	require.NoError(t, err)
	defer exec.Close()

	ctx := context.Background()

	// Call prepareEnvironment to write configs
	err = exec.prepareEnvironment(ctx)
	require.NoError(t, err, "prepareEnvironment should succeed")

	// Verify MCP config was written by reading it back
	readContent, err := exec.ReadFile(ctx, ".mcp.json")
	require.NoError(t, err, "Should be able to read .mcp.json")
	require.NotEmpty(t, readContent, "MCP config should not be empty")

	t.Logf("MCP config in container: %s", string(readContent))

	// Verify content matches
	assert.JSONEq(t, string(mcpConfig), string(readContent), "MCP config content should match")

	t.Log("✓ MCP config verified in container")
}

// TestClaudeExecutorSkillsCopy tests that skills directory is correctly copied to container
// Uses real test application skills directory
func TestClaudeExecutorSkillsCopy(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	manager := createTestManager(t)
	if manager == nil {
		return
	}
	defer manager.Close()

	// Use real skills directory from test application
	appRoot := os.Getenv("YAO_ROOT")
	require.NotEmpty(t, appRoot, "YAO_ROOT should be set")

	skillsDir := appRoot + "/assistants/tests/sandbox/full/skills"

	// Verify skills directory exists on host
	info, err := os.Stat(skillsDir)
	require.NoError(t, err, "Skills directory should exist: %s", skillsDir)
	require.True(t, info.IsDir(), "Skills path should be a directory")
	t.Logf("Using real skills directory: %s", skillsDir)

	opts := &Options{
		Command:   "claude",
		Image:     "alpine:latest",
		UserID:    "test-user",
		ChatID:    fmt.Sprintf("test-chat-skills-%d", time.Now().UnixNano()),
		SkillsDir: skillsDir,
	}

	exec, err := NewExecutor(manager, opts)
	require.NoError(t, err)
	defer exec.Close()

	ctx := context.Background()

	// Call prepareEnvironment to copy skills
	err = exec.prepareEnvironment(ctx)
	require.NoError(t, err, "prepareEnvironment should succeed")

	// Verify .claude directory was created
	output, err := exec.Exec(ctx, []string{"ls", "-la", ".claude"})
	require.NoError(t, err, ".claude directory should exist")
	t.Logf(".claude directory contents:\n%s", output)

	// Verify skills directory exists in container
	output, err = exec.Exec(ctx, []string{"ls", "-la", ".claude/skills"})
	require.NoError(t, err, "skills directory should exist in container")
	t.Logf("skills directory contents:\n%s", output)
	assert.Contains(t, output, "echo-test", "echo-test skill should exist")

	// Verify echo-test skill was copied correctly
	output, err = exec.Exec(ctx, []string{"ls", "-la", ".claude/skills/echo-test"})
	require.NoError(t, err, "echo-test skill directory should exist")
	assert.Contains(t, output, "SKILL.md", "SKILL.md should exist in echo-test")
	assert.Contains(t, output, "scripts", "scripts directory should exist in echo-test")
	t.Logf("echo-test skill contents:\n%s", output)

	// Read SKILL.md content to verify
	readContent, err := exec.ReadFile(ctx, ".claude/skills/echo-test/SKILL.md")
	require.NoError(t, err, "Should be able to read SKILL.md from container")
	require.NotEmpty(t, readContent, "SKILL.md content should not be empty")
	// Verify content contains expected strings from the real SKILL.md
	assert.Contains(t, string(readContent), "name: echo-test", "SKILL.md should contain skill name")
	assert.Contains(t, string(readContent), "# Echo Test", "SKILL.md should contain the title")
	t.Logf("✓ SKILL.md content verified (%d bytes)", len(readContent))

	t.Log("✓ Skills directory verified in container with real test data")
}

// TestClaudeExecutorPrepareEnvironmentIntegration tests full environment preparation
// Uses real test application data
func TestClaudeExecutorPrepareEnvironmentIntegration(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	manager := createTestManager(t)
	if manager == nil {
		return
	}
	defer manager.Close()

	// Use real skills directory from test application
	appRoot := os.Getenv("YAO_ROOT")
	require.NotEmpty(t, appRoot, "YAO_ROOT should be set")
	skillsDir := appRoot + "/assistants/tests/sandbox/full/skills"

	// Verify skills directory exists
	_, err := os.Stat(skillsDir)
	require.NoError(t, err, "Skills directory should exist")

	// Create MCP config (simulating what buildMCPConfigForSandbox produces)
	mcpConfig := []byte(`{"mcpServers":{"echo":{"command":"yao-mcp-proxy","args":["echo"],"tools":["ping","echo","status"]}}}`)

	opts := &Options{
		Command:       "claude",
		Image:         "alpine:latest",
		UserID:        "test-user",
		ChatID:        fmt.Sprintf("test-chat-full-env-%d", time.Now().UnixNano()),
		ConnectorHost: "https://api.test.com",
		ConnectorKey:  "test-key",
		Model:         "test-model",
		MCPConfig:     mcpConfig,
		SkillsDir:     skillsDir,
	}

	exec, err := NewExecutor(manager, opts)
	require.NoError(t, err)
	defer exec.Close()

	ctx := context.Background()

	// Call prepareEnvironment
	err = exec.prepareEnvironment(ctx)
	require.NoError(t, err, "prepareEnvironment should succeed")

	// Verify all files exist
	// 1. Check CCR config
	ccrContent, err := exec.Exec(ctx, []string{"cat", "/home/sandbox/.claude-code-router/config.json"})
	require.NoError(t, err, "CCR config should exist")
	assert.Contains(t, ccrContent, "api_base_url", "CCR config should contain api_base_url")
	t.Logf("✓ CCR config verified: %d bytes", len(ccrContent))

	// 2. Check MCP config
	mcpContent, err := exec.ReadFile(ctx, ".mcp.json")
	require.NoError(t, err, "MCP config should exist in container")
	assert.JSONEq(t, string(mcpConfig), string(mcpContent), "MCP config content should match")
	t.Logf("✓ MCP config verified: %s", string(mcpContent))

	// 3. Check Skills directory structure
	output, err := exec.Exec(ctx, []string{"ls", "-la", ".claude/skills"})
	require.NoError(t, err, "Skills directory should exist in container")
	assert.Contains(t, output, "echo-test", "echo-test skill should exist")
	t.Logf("✓ Skills directory contents:\n%s", output)

	// 4. Check skill content
	skillContent, err := exec.ReadFile(ctx, ".claude/skills/echo-test/SKILL.md")
	require.NoError(t, err, "SKILL.md should exist in container")
	require.NotEmpty(t, skillContent, "SKILL.md should not be empty")
	assert.Contains(t, string(skillContent), "name: echo-test", "SKILL.md should contain skill name")
	assert.Contains(t, string(skillContent), "# Echo Test", "SKILL.md should contain the title")
	t.Logf("✓ SKILL.md verified: %d bytes", len(skillContent))

	t.Log("✓ Full environment preparation verified with real test data")
}

// TestClaudeExecutorIPCSocketMount verifies that IPC socket is bind mounted to container
func TestClaudeExecutorIPCSocketMount(t *testing.T) {
	manager := createTestManager(t)

	opts := &Options{
		Command:       "claude",
		Image:         "alpine:latest",
		UserID:        "test-user",
		ChatID:        "test-ipc-socket-" + fmt.Sprintf("%d", time.Now().UnixNano()),
		ConnectorHost: "https://api.test.com",
		ConnectorKey:  "test-key",
		Model:         "test-model",
	}

	exec, err := NewExecutor(manager, opts)
	require.NoError(t, err)
	defer exec.Close()

	ctx := context.Background()

	// Check if IPC socket exists in container
	output, err := exec.Exec(ctx, []string{"ls", "-la", "/tmp/yao.sock"})
	require.NoError(t, err, "IPC socket should exist in container")
	assert.Contains(t, output, "yao.sock", "Should find yao.sock file")
	t.Logf("✓ IPC socket mounted: %s", strings.TrimSpace(output))

	// Verify it's a socket file (starts with 's' in ls output)
	assert.Contains(t, output, "srw", "Should be a socket file (starts with 's')")
	t.Log("✓ IPC socket is correctly bind mounted to container")
}
