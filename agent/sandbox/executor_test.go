package sandbox

import (
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

func TestNewExecutorWithInvalidOptions(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	manager := createTestManager(t)
	if manager == nil {
		return
	}
	defer manager.Close()

	// Test with nil options
	_, err := New(manager, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "options is required")

	// Test with invalid command
	_, err = New(manager, &Options{
		Command: "invalid",
		UserID:  "user1",
		ChatID:  "chat1",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported command type")
}

func TestNewExecutorWithValidOptions(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	manager := createTestManager(t)
	if manager == nil {
		return
	}
	defer manager.Close()

	// Test with valid claude options
	opts := &Options{
		Command:       "claude",
		UserID:        "test-user",
		ChatID:        "test-chat",
		ConnectorHost: "https://api.example.com",
		ConnectorKey:  "key123",
		Model:         "test-model",
	}

	exec, err := New(manager, opts)
	require.NoError(t, err)
	require.NotNil(t, exec)

	// Verify executor was created
	assert.NotEmpty(t, exec.GetWorkDir())
	assert.NoError(t, exec.Close())
}

func TestDefaultImageIsSetWhenEmpty(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	manager := createTestManager(t)
	if manager == nil {
		return
	}
	defer manager.Close()

	opts := &Options{
		Command: "claude",
		Image:   "", // Empty, should be set to default
		UserID:  "test-user",
		ChatID:  "test-chat-2",
	}

	exec, err := New(manager, opts)
	require.NoError(t, err)
	defer exec.Close() // Ensure cleanup

	// The image should have been set to default
	assert.Equal(t, "yaoapp/sandbox-claude:latest", opts.Image)
}

func TestCursorExecutorNotImplemented(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	manager := createTestManager(t)
	if manager == nil {
		return
	}
	defer manager.Close()

	opts := &Options{
		Command: "cursor",
		UserID:  "test-user",
		ChatID:  "test-chat-3",
	}

	_, err := New(manager, opts)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not implemented")
}
