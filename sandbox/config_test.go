package sandbox

import (
	"os"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Image != "yaoapp/sandbox-claude:latest" {
		t.Errorf("expected default image 'yaoapp/sandbox-claude:latest', got '%s'", cfg.Image)
	}
	if cfg.MaxContainers != 100 {
		t.Errorf("expected MaxContainers 100, got %d", cfg.MaxContainers)
	}
	if cfg.IdleTimeout != 30*time.Minute {
		t.Errorf("expected IdleTimeout 30m, got %v", cfg.IdleTimeout)
	}
	if cfg.MaxMemory != "2g" {
		t.Errorf("expected MaxMemory '2g', got '%s'", cfg.MaxMemory)
	}
	if cfg.MaxCPU != 1.0 {
		t.Errorf("expected MaxCPU 1.0, got %f", cfg.MaxCPU)
	}
}

func TestConfigInit(t *testing.T) {
	// Clear any existing environment variables that might interfere
	os.Unsetenv("YAO_SANDBOX_WORKSPACE")
	os.Unsetenv("YAO_SANDBOX_IPC")

	// Test with dataRoot
	cfg := &Config{}
	cfg.Init("/tmp/yao-test")

	if cfg.WorkspaceRoot != "/tmp/yao-test/sandbox/workspace" {
		t.Errorf("expected WorkspaceRoot '/tmp/yao-test/sandbox/workspace', got '%s'", cfg.WorkspaceRoot)
	}
	if cfg.IPCDir != "/tmp/yao-test/sandbox/ipc" {
		t.Errorf("expected IPCDir '/tmp/yao-test/sandbox/ipc', got '%s'", cfg.IPCDir)
	}
}

func TestConfigInitWithEnv(t *testing.T) {
	// Set environment variables
	os.Setenv("YAO_SANDBOX_IMAGE", "test/image:v1")
	os.Setenv("YAO_SANDBOX_MAX", "50")
	os.Setenv("YAO_SANDBOX_IDLE_TIMEOUT", "15m")
	os.Setenv("YAO_SANDBOX_MEMORY", "4g")
	os.Setenv("YAO_SANDBOX_CPU", "2.5")
	defer func() {
		os.Unsetenv("YAO_SANDBOX_IMAGE")
		os.Unsetenv("YAO_SANDBOX_MAX")
		os.Unsetenv("YAO_SANDBOX_IDLE_TIMEOUT")
		os.Unsetenv("YAO_SANDBOX_MEMORY")
		os.Unsetenv("YAO_SANDBOX_CPU")
	}()

	cfg := &Config{}
	cfg.Init("/tmp/yao-test")

	if cfg.Image != "test/image:v1" {
		t.Errorf("expected Image 'test/image:v1', got '%s'", cfg.Image)
	}
	if cfg.MaxContainers != 50 {
		t.Errorf("expected MaxContainers 50, got %d", cfg.MaxContainers)
	}
	if cfg.IdleTimeout != 15*time.Minute {
		t.Errorf("expected IdleTimeout 15m, got %v", cfg.IdleTimeout)
	}
	if cfg.MaxMemory != "4g" {
		t.Errorf("expected MaxMemory '4g', got '%s'", cfg.MaxMemory)
	}
	if cfg.MaxCPU != 2.5 {
		t.Errorf("expected MaxCPU 2.5, got %f", cfg.MaxCPU)
	}
}

func TestConfigInitWithWorkspaceEnv(t *testing.T) {
	// Test YAO_SANDBOX_WORKSPACE env var
	os.Setenv("YAO_SANDBOX_WORKSPACE", "/custom/workspace")
	os.Setenv("YAO_SANDBOX_IPC", "/custom/ipc")
	defer func() {
		os.Unsetenv("YAO_SANDBOX_WORKSPACE")
		os.Unsetenv("YAO_SANDBOX_IPC")
	}()

	cfg := &Config{}
	cfg.Init("/tmp/yao-test")

	if cfg.WorkspaceRoot != "/custom/workspace" {
		t.Errorf("expected WorkspaceRoot '/custom/workspace', got '%s'", cfg.WorkspaceRoot)
	}
	if cfg.IPCDir != "/custom/ipc" {
		t.Errorf("expected IPCDir '/custom/ipc', got '%s'", cfg.IPCDir)
	}
}

func TestConfigInitWithPresetValues(t *testing.T) {
	// Test that preset values are not overwritten by defaults
	cfg := &Config{
		Image:         "preset/image:v2",
		MaxContainers: 200,
		IdleTimeout:   1 * time.Hour,
		MaxMemory:     "8g",
		MaxCPU:        4.0,
		WorkspaceRoot: "/preset/workspace",
		IPCDir:        "/preset/ipc",
	}
	cfg.Init("/tmp/yao-test")

	if cfg.Image != "preset/image:v2" {
		t.Errorf("expected Image 'preset/image:v2', got '%s'", cfg.Image)
	}
	if cfg.MaxContainers != 200 {
		t.Errorf("expected MaxContainers 200, got %d", cfg.MaxContainers)
	}
	if cfg.IdleTimeout != 1*time.Hour {
		t.Errorf("expected IdleTimeout 1h, got %v", cfg.IdleTimeout)
	}
	if cfg.MaxMemory != "8g" {
		t.Errorf("expected MaxMemory '8g', got '%s'", cfg.MaxMemory)
	}
	if cfg.MaxCPU != 4.0 {
		t.Errorf("expected MaxCPU 4.0, got %f", cfg.MaxCPU)
	}
	if cfg.WorkspaceRoot != "/preset/workspace" {
		t.Errorf("expected WorkspaceRoot '/preset/workspace', got '%s'", cfg.WorkspaceRoot)
	}
	if cfg.IPCDir != "/preset/ipc" {
		t.Errorf("expected IPCDir '/preset/ipc', got '%s'", cfg.IPCDir)
	}
}

func TestConfigInitInvalidEnvValues(t *testing.T) {
	// Test with invalid environment values
	os.Setenv("YAO_SANDBOX_MAX", "invalid")
	os.Setenv("YAO_SANDBOX_IDLE_TIMEOUT", "not-a-duration")
	os.Setenv("YAO_SANDBOX_CPU", "not-a-float")
	defer func() {
		os.Unsetenv("YAO_SANDBOX_MAX")
		os.Unsetenv("YAO_SANDBOX_IDLE_TIMEOUT")
		os.Unsetenv("YAO_SANDBOX_CPU")
	}()

	cfg := &Config{}
	cfg.Init("/tmp/yao-test")

	// Invalid env values should fall back to defaults
	if cfg.MaxContainers != 100 {
		t.Errorf("expected MaxContainers 100 (default), got %d", cfg.MaxContainers)
	}
	if cfg.IdleTimeout != 30*time.Minute {
		t.Errorf("expected IdleTimeout 30m (default), got %v", cfg.IdleTimeout)
	}
	if cfg.MaxCPU != 1.0 {
		t.Errorf("expected MaxCPU 1.0 (default), got %f", cfg.MaxCPU)
	}
}

func TestConfigInitNegativeValues(t *testing.T) {
	// Test with negative/zero values in env
	os.Setenv("YAO_SANDBOX_MAX", "-5")
	os.Setenv("YAO_SANDBOX_CPU", "-1.0")
	defer func() {
		os.Unsetenv("YAO_SANDBOX_MAX")
		os.Unsetenv("YAO_SANDBOX_CPU")
	}()

	cfg := &Config{}
	cfg.Init("/tmp/yao-test")

	// Negative values should fall back to defaults
	if cfg.MaxContainers != 100 {
		t.Errorf("expected MaxContainers 100 (default), got %d", cfg.MaxContainers)
	}
	if cfg.MaxCPU != 1.0 {
		t.Errorf("expected MaxCPU 1.0 (default), got %f", cfg.MaxCPU)
	}
}

func TestConfigInitZeroMax(t *testing.T) {
	// Test with zero max containers
	os.Setenv("YAO_SANDBOX_MAX", "0")
	defer os.Unsetenv("YAO_SANDBOX_MAX")

	cfg := &Config{}
	cfg.Init("/tmp/yao-test")

	// Zero is rejected by v > 0 check, should use default
	if cfg.MaxContainers != 100 {
		t.Errorf("expected MaxContainers 100 (default), got %d", cfg.MaxContainers)
	}
}

func TestContainerName(t *testing.T) {
	tests := []struct {
		userID   string
		chatID   string
		expected string
	}{
		{"user1", "chat1", "yao-sandbox-user1-chat1"},
		{"u123", "c456", "yao-sandbox-u123-c456"},
		{"test-user", "test-chat", "yao-sandbox-test-user-test-chat"},
		{"", "", "yao-sandbox--"},
		{"user_with_underscore", "chat-with-dash", "yao-sandbox-user_with_underscore-chat-with-dash"},
		{"UPPERCASE", "lowercase", "yao-sandbox-UPPERCASE-lowercase"},
	}

	for _, tt := range tests {
		result := containerName(tt.userID, tt.chatID)
		if result != tt.expected {
			t.Errorf("containerName(%s, %s) = %s, want %s", tt.userID, tt.chatID, result, tt.expected)
		}
	}
}

func TestConfigEnvPriority(t *testing.T) {
	// Env vars should override preset config values
	os.Setenv("YAO_SANDBOX_IMAGE", "env/override:latest")
	defer os.Unsetenv("YAO_SANDBOX_IMAGE")

	cfg := &Config{
		Image: "preset/image:v1",
	}
	cfg.Init("/tmp/yao-test")

	// Env should win
	if cfg.Image != "env/override:latest" {
		t.Errorf("expected Image 'env/override:latest' (from env), got '%s'", cfg.Image)
	}
}
