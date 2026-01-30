package claude

import (
	"context"
	"os"
	"testing"

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
		ChatID:        "test-chat-claude-1",
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
		ChatID:  "test-chat-file-ops",
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
		ChatID:  "test-chat-exec",
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
