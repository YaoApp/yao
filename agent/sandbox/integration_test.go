package sandbox

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	agentContext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/config"
	infraSandbox "github.com/yaoapp/yao/sandbox"
	"github.com/yaoapp/yao/test"
)

// createIntegrationTestManager creates a sandbox manager for integration testing
func createIntegrationTestManager(t *testing.T) *infraSandbox.Manager {
	dataRoot := os.Getenv("YAO_ROOT")
	if dataRoot == "" {
		dataRoot = t.TempDir()
	}

	cfg := infraSandbox.DefaultConfig()
	cfg.Init(dataRoot)

	manager, err := infraSandbox.NewManager(cfg)
	if err != nil {
		t.Skipf("Skipping test: Docker not available: %v", err)
		return nil
	}

	return manager
}

// TestExecutorInterfaceCompatibility verifies that the executor implements both interfaces correctly
func TestExecutorInterfaceCompatibility(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	manager := createIntegrationTestManager(t)
	if manager == nil {
		return
	}
	defer manager.Close()

	// Create executor via factory function
	opts := &Options{
		Command: "claude",
		Image:   "alpine:latest",
		UserID:  "test-user",
		ChatID:  "test-compat",
	}

	executor, err := New(manager, opts)
	require.NoError(t, err)
	require.NotNil(t, executor)
	defer executor.Close()

	// Verify executor implements agent/sandbox.Executor interface
	var _ Executor = executor

	// Verify executor can be cast to context.SandboxExecutor
	ctxExecutor, ok := executor.(agentContext.SandboxExecutor)
	require.True(t, ok, "executor should implement context.SandboxExecutor")
	require.NotNil(t, ctxExecutor)

	// Test SandboxExecutor methods work
	ctx := context.Background()

	// WriteFile
	err = ctxExecutor.WriteFile(ctx, "compat-test.txt", []byte("compatibility test"))
	require.NoError(t, err)

	// ReadFile
	content, err := ctxExecutor.ReadFile(ctx, "compat-test.txt")
	require.NoError(t, err)
	assert.Equal(t, "compatibility test", string(content))

	// ListDir
	files, err := ctxExecutor.ListDir(ctx, ".")
	require.NoError(t, err)
	assert.True(t, len(files) > 0)

	// Exec
	output, err := ctxExecutor.Exec(ctx, []string{"echo", "compat"})
	require.NoError(t, err)
	assert.Contains(t, output, "compat")

	// GetWorkDir
	workDir := ctxExecutor.GetWorkDir()
	assert.NotEmpty(t, workDir)
}

// TestExecutorRoundTrip tests the full round-trip of creating executor and performing operations
func TestExecutorRoundTrip(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	manager := createIntegrationTestManager(t)
	if manager == nil {
		return
	}
	defer manager.Close()

	opts := &Options{
		Command:       "claude",
		Image:         "alpine:latest",
		UserID:        "test-user",
		ChatID:        "test-roundtrip",
		ConnectorHost: "https://api.example.com",
		ConnectorKey:  "test-key",
		Model:         "test-model",
	}

	// Create executor
	executor, err := New(manager, opts)
	require.NoError(t, err)
	require.NotNil(t, executor)
	defer executor.Close()

	ctx := context.Background()

	// 1. Write a file
	testContent := "Hello, integration test!"
	err = executor.WriteFile(ctx, "integration.txt", []byte(testContent))
	require.NoError(t, err, "WriteFile should succeed")

	// 2. Read the file back
	readContent, err := executor.ReadFile(ctx, "integration.txt")
	require.NoError(t, err, "ReadFile should succeed")
	assert.Equal(t, testContent, string(readContent), "Content should match")

	// 3. List directory
	files, err := executor.ListDir(ctx, ".")
	require.NoError(t, err, "ListDir should succeed")

	var found bool
	for _, f := range files {
		if f.Name == "integration.txt" {
			found = true
			assert.False(t, f.IsDir, "Should not be a directory")
			assert.Equal(t, int64(len(testContent)), f.Size, "Size should match")
			break
		}
	}
	assert.True(t, found, "Should find integration.txt in listing")

	// 4. Execute command
	output, err := executor.Exec(ctx, []string{"cat", "/workspace/integration.txt"})
	require.NoError(t, err, "Exec should succeed")
	assert.Contains(t, output, testContent, "cat output should contain file content")

	// 5. Verify workdir
	assert.Equal(t, "/workspace", executor.GetWorkDir(), "WorkDir should be /workspace")
}

// TestMultipleExecutorsIsolation verifies that multiple executors have isolated workspaces
func TestMultipleExecutorsIsolation(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	manager := createIntegrationTestManager(t)
	if manager == nil {
		return
	}
	defer manager.Close()

	// Create two executors with different chat IDs
	opts1 := &Options{
		Command: "claude",
		Image:   "alpine:latest",
		UserID:  "test-user",
		ChatID:  "test-isolation-1",
	}
	opts2 := &Options{
		Command: "claude",
		Image:   "alpine:latest",
		UserID:  "test-user",
		ChatID:  "test-isolation-2",
	}

	exec1, err := New(manager, opts1)
	require.NoError(t, err)
	defer exec1.Close()

	exec2, err := New(manager, opts2)
	require.NoError(t, err)
	defer exec2.Close()

	ctx := context.Background()

	// Write different content to each executor
	err = exec1.WriteFile(ctx, "test.txt", []byte("executor 1"))
	require.NoError(t, err)

	err = exec2.WriteFile(ctx, "test.txt", []byte("executor 2"))
	require.NoError(t, err)

	// Read back and verify isolation
	content1, err := exec1.ReadFile(ctx, "test.txt")
	require.NoError(t, err)
	assert.Equal(t, "executor 1", string(content1), "Executor 1 should have its own content")

	content2, err := exec2.ReadFile(ctx, "test.txt")
	require.NoError(t, err)
	assert.Equal(t, "executor 2", string(content2), "Executor 2 should have its own content")
}
