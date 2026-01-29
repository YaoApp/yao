package sandbox

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/test"
)

// getTestDirs returns workspace and IPC directories for testing.
// Uses environment variables if set (for CI), otherwise creates temp directories.
// Returns workspaceRoot, ipcDir, tmpDir (empty if using env vars), error
func getTestDirs(prefix string) (string, string, string, error) {
	workspaceRoot := os.Getenv("YAO_SANDBOX_WORKSPACE")
	ipcDir := os.Getenv("YAO_SANDBOX_IPC")

	var tmpDir string
	var err error

	if workspaceRoot == "" || ipcDir == "" {
		// Create temporary directories for test
		tmpDir, err = os.MkdirTemp("", prefix)
		if err != nil {
			return "", "", "", err
		}
		if workspaceRoot == "" {
			workspaceRoot = filepath.Join(tmpDir, "workspace")
		}
		if ipcDir == "" {
			ipcDir = filepath.Join(tmpDir, "ipc")
		}
	}

	return workspaceRoot, ipcDir, tmpDir, nil
}

// skipIfNoDocker skips the test if Docker is not available
func skipIfNoDocker(t *testing.T) *Manager {
	t.Helper()

	workspaceRoot, ipcDir, tmpDir, err := getTestDirs("sandbox-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	cfg := &Config{
		Image:         "yaoapp/sandbox-claude:latest",
		WorkspaceRoot: workspaceRoot,
		IPCDir:        ipcDir,
		MaxContainers: 5,
		IdleTimeout:   1 * time.Minute,
		MaxMemory:     "512m",
		MaxCPU:        0.5,
	}

	m, err := NewManager(cfg)
	if err != nil {
		// Clean up temp dir
		if tmpDir != "" {
			os.RemoveAll(tmpDir)
		}
		if strings.Contains(err.Error(), "Docker not available") ||
			strings.Contains(err.Error(), "Cannot connect to the Docker daemon") {
			t.Skipf("Skipping test: %v", err)
		}
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Store tmpDir in test cleanup
	t.Cleanup(func() {
		m.Close()
		if tmpDir != "" {
			os.RemoveAll(tmpDir)
		}
	})

	return m
}

// TestNewManager tests manager creation
func TestNewManager(t *testing.T) {
	m := skipIfNoDocker(t)

	if m.dockerClient == nil {
		t.Error("Docker client should not be nil")
	}

	if m.ipcManager == nil {
		t.Error("IPC manager should not be nil")
	}

	if m.config == nil {
		t.Error("Config should not be nil")
	}
}

// TestNewManagerWithNilConfig tests manager creation with nil config
func TestNewManagerWithNilConfig(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sandbox-test-nil-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Set environment variables for default paths
	os.Setenv("YAO_SANDBOX_WORKSPACE", filepath.Join(tmpDir, "workspace"))
	os.Setenv("YAO_SANDBOX_IPC", filepath.Join(tmpDir, "ipc"))
	defer os.Unsetenv("YAO_SANDBOX_WORKSPACE")
	defer os.Unsetenv("YAO_SANDBOX_IPC")

	cfg := DefaultConfig()
	cfg.Init(tmpDir)

	m, err := NewManager(cfg)
	if err != nil {
		if strings.Contains(err.Error(), "Docker not available") {
			t.Skip("Docker not available")
		}
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer m.Close()

	if m.config.MaxContainers != 100 {
		t.Errorf("Expected MaxContainers 100, got %d", m.config.MaxContainers)
	}
}

// TestGetOrCreate tests container creation
func TestGetOrCreate(t *testing.T) {
	m := skipIfNoDocker(t)

	ctx := context.Background()
	userID := "test-user"
	chatID := "test-chat-" + time.Now().Format("20060102150405")

	// Create container
	container, err := m.GetOrCreate(ctx, userID, chatID)
	if err != nil {
		t.Fatalf("GetOrCreate failed: %v", err)
	}

	// Verify container properties
	if container.UserID != userID {
		t.Errorf("Expected UserID %s, got %s", userID, container.UserID)
	}

	if container.ChatID != chatID {
		t.Errorf("Expected ChatID %s, got %s", chatID, container.ChatID)
	}

	expectedName := containerName(userID, chatID)
	if container.Name != expectedName {
		t.Errorf("Expected Name %s, got %s", expectedName, container.Name)
	}

	if container.Status != StatusCreated {
		t.Errorf("Expected Status %s, got %s", StatusCreated, container.Status)
	}

	// Get same container again (should return existing)
	container2, err := m.GetOrCreate(ctx, userID, chatID)
	if err != nil {
		t.Fatalf("GetOrCreate (second call) failed: %v", err)
	}

	if container.ID != container2.ID {
		t.Error("Expected same container on second GetOrCreate call")
	}

	// Cleanup
	if err := m.Remove(ctx, container.Name); err != nil {
		t.Logf("Warning: failed to remove container: %v", err)
	}
}

// TestContainerStartStopRemove tests container lifecycle
func TestContainerStartStopRemove(t *testing.T) {
	m := skipIfNoDocker(t)

	ctx := context.Background()
	userID := "lifecycle-user"
	chatID := "lifecycle-chat-" + time.Now().Format("20060102150405")

	// Create container
	container, err := m.GetOrCreate(ctx, userID, chatID)
	if err != nil {
		t.Fatalf("GetOrCreate failed: %v", err)
	}

	// Start container
	if err := m.Start(ctx, container.Name); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Verify status is running
	c, ok := m.containers.Load(container.Name)
	if !ok {
		t.Fatal("Container not found in map")
	}
	if c.(*Container).Status != StatusRunning {
		t.Errorf("Expected status %s, got %s", StatusRunning, c.(*Container).Status)
	}

	// Stop container
	if err := m.Stop(ctx, container.Name); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	// Verify status is stopped
	c, ok = m.containers.Load(container.Name)
	if !ok {
		t.Fatal("Container not found in map after stop")
	}
	if c.(*Container).Status != StatusStopped {
		t.Errorf("Expected status %s, got %s", StatusStopped, c.(*Container).Status)
	}

	// Remove container
	if err := m.Remove(ctx, container.Name); err != nil {
		t.Fatalf("Remove failed: %v", err)
	}

	// Verify container is removed from map
	if _, ok := m.containers.Load(container.Name); ok {
		t.Error("Container should be removed from map")
	}
}

// TestExec tests command execution
func TestExec(t *testing.T) {
	m := skipIfNoDocker(t)

	ctx := context.Background()
	userID := "exec-user"
	chatID := "exec-chat-" + time.Now().Format("20060102150405")

	// Create and start container
	container, err := m.GetOrCreate(ctx, userID, chatID)
	if err != nil {
		t.Fatalf("GetOrCreate failed: %v", err)
	}
	defer m.Remove(ctx, container.Name)

	// Execute simple command
	result, err := m.Exec(ctx, container.Name, []string{"echo", "hello world"}, nil)
	if err != nil {
		t.Fatalf("Exec failed: %v", err)
	}

	// Note: Docker multiplexed stream includes header bytes
	if !strings.Contains(result.Stdout, "hello world") {
		t.Errorf("Expected stdout to contain 'hello world', got: %s", result.Stdout)
	}
}

// TestExecWithEnv tests command execution with environment variables
func TestExecWithEnv(t *testing.T) {
	m := skipIfNoDocker(t)

	ctx := context.Background()
	userID := "exec-env-user"
	chatID := "exec-env-chat-" + time.Now().Format("20060102150405")

	container, err := m.GetOrCreate(ctx, userID, chatID)
	if err != nil {
		t.Fatalf("GetOrCreate failed: %v", err)
	}
	defer m.Remove(ctx, container.Name)

	result, err := m.Exec(ctx, container.Name, []string{"sh", "-c", "echo $TEST_VAR"}, &ExecOptions{
		Env: map[string]string{
			"TEST_VAR": "test_value_123",
		},
	})
	if err != nil {
		t.Fatalf("Exec with env failed: %v", err)
	}

	if !strings.Contains(result.Stdout, "test_value_123") {
		t.Errorf("Expected stdout to contain 'test_value_123', got: %s", result.Stdout)
	}
}

// TestExecWithTimeout tests command execution timeout
func TestExecWithTimeout(t *testing.T) {
	m := skipIfNoDocker(t)

	ctx := context.Background()
	userID := "exec-timeout-user"
	chatID := "exec-timeout-chat-" + time.Now().Format("20060102150405")

	container, err := m.GetOrCreate(ctx, userID, chatID)
	if err != nil {
		t.Fatalf("GetOrCreate failed: %v", err)
	}
	defer m.Remove(ctx, container.Name)

	// Execute command with short timeout (sleep 10s but timeout after 500ms)
	start := time.Now()
	_, err = m.Exec(ctx, container.Name, []string{"sleep", "10"}, &ExecOptions{
		Timeout: 500 * time.Millisecond,
	})
	elapsed := time.Since(start)

	// Should timeout with context deadline exceeded
	if err == nil {
		t.Error("Expected timeout error, but command completed without error")
	} else if err != context.DeadlineExceeded {
		t.Logf("Got error (expected context.DeadlineExceeded): %v", err)
	}

	// Verify it didn't wait the full 10 seconds
	if elapsed > 5*time.Second {
		t.Errorf("Timeout took too long: %v (expected < 5s)", elapsed)
	}

	t.Logf("Timeout completed in %v", elapsed)
}

// TestFileOperations tests filesystem operations
func TestFileOperations(t *testing.T) {
	m := skipIfNoDocker(t)

	ctx := context.Background()
	userID := "file-user"
	chatID := "file-chat-" + time.Now().Format("20060102150405")

	container, err := m.GetOrCreate(ctx, userID, chatID)
	if err != nil {
		t.Fatalf("GetOrCreate failed: %v", err)
	}
	defer m.Remove(ctx, container.Name)

	// Start container first
	if err := m.Start(ctx, container.Name); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Test MkDir
	testDir := "/workspace/testdir"
	if err := m.MkDir(ctx, container.Name, testDir); err != nil {
		t.Fatalf("MkDir failed: %v", err)
	}

	// Test WriteFile
	testFile := "/workspace/testdir/test.txt"
	testContent := []byte("Hello, Sandbox!")
	if err := m.WriteFile(ctx, container.Name, testFile, testContent); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// Test ReadFile
	content, err := m.ReadFile(ctx, container.Name, testFile)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	if string(content) != string(testContent) {
		t.Errorf("Expected content '%s', got '%s'", testContent, content)
	}

	// Test Stat
	info, err := m.Stat(ctx, container.Name, testFile)
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}

	if info == nil {
		t.Fatal("Stat returned nil")
	}

	if info.Name != "test.txt" {
		t.Errorf("Expected name 'test.txt', got '%s'", info.Name)
	}

	if info.Size != int64(len(testContent)) {
		t.Errorf("Expected size %d, got %d", len(testContent), info.Size)
	}

	// Test ListDir
	files, err := m.ListDir(ctx, container.Name, "/workspace/testdir")
	if err != nil {
		t.Fatalf("ListDir failed: %v", err)
	}

	found := false
	for _, f := range files {
		if f.Name == "test.txt" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected to find test.txt in directory listing")
	}

	// Test RemoveFile
	if err := m.RemoveFile(ctx, container.Name, testFile); err != nil {
		t.Fatalf("RemoveFile failed: %v", err)
	}

	// Verify file is removed - check via ls instead of stat
	// (stat command may still succeed with different output)
	files2, err := m.ListDir(ctx, container.Name, "/workspace/testdir")
	if err != nil {
		t.Fatalf("ListDir after removal failed: %v", err)
	}

	found = false
	for _, f := range files2 {
		if f.Name == "test.txt" {
			found = true
			break
		}
	}
	if found {
		t.Error("File test.txt should be removed")
	}
}

// TestCopyOperations tests copy to/from container
func TestCopyOperations(t *testing.T) {
	m := skipIfNoDocker(t)

	ctx := context.Background()
	userID := "copy-user"
	chatID := "copy-chat-" + time.Now().Format("20060102150405")

	container, err := m.GetOrCreate(ctx, userID, chatID)
	if err != nil {
		t.Fatalf("GetOrCreate failed: %v", err)
	}
	defer m.Remove(ctx, container.Name)

	// Start container
	if err := m.Start(ctx, container.Name); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Create temp file on host
	tmpDir, err := os.MkdirTemp("", "sandbox-copy-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	hostFile := filepath.Join(tmpDir, "source.txt")
	if err := os.WriteFile(hostFile, []byte("copy test content"), 0644); err != nil {
		t.Fatalf("Failed to write host file: %v", err)
	}

	// Copy to container
	if err := m.CopyToContainer(ctx, container.Name, hostFile, "/workspace/"); err != nil {
		t.Fatalf("CopyToContainer failed: %v", err)
	}

	// Verify file exists in container
	content, err := m.ReadFile(ctx, container.Name, "/workspace/source.txt")
	if err != nil {
		t.Fatalf("ReadFile after copy failed: %v", err)
	}

	if string(content) != "copy test content" {
		t.Errorf("Expected 'copy test content', got '%s'", content)
	}

	// Copy from container
	extractDir := filepath.Join(tmpDir, "extracted")
	os.MkdirAll(extractDir, 0755)

	if err := m.CopyFromContainer(ctx, container.Name, "/workspace/source.txt", extractDir); err != nil {
		t.Fatalf("CopyFromContainer failed: %v", err)
	}

	// Verify extracted file
	extractedContent, err := os.ReadFile(filepath.Join(extractDir, "source.txt"))
	if err != nil {
		t.Fatalf("Failed to read extracted file: %v", err)
	}

	if string(extractedContent) != "copy test content" {
		t.Errorf("Expected 'copy test content', got '%s'", extractedContent)
	}
}

// TestListContainers tests listing containers for a user
func TestListContainers(t *testing.T) {
	m := skipIfNoDocker(t)

	ctx := context.Background()
	userID := "list-user"
	chatIDs := []string{
		"list-chat-1-" + time.Now().Format("20060102150405"),
		"list-chat-2-" + time.Now().Format("20060102150405"),
	}

	// Create multiple containers for same user
	for _, chatID := range chatIDs {
		container, err := m.GetOrCreate(ctx, userID, chatID)
		if err != nil {
			t.Fatalf("GetOrCreate failed for %s: %v", chatID, err)
		}
		defer m.Remove(ctx, container.Name)
	}

	// List containers for user
	containers, err := m.List(ctx, userID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(containers) != 2 {
		t.Errorf("Expected 2 containers, got %d", len(containers))
	}

	// Verify all containers belong to user
	for _, c := range containers {
		if c.UserID != userID {
			t.Errorf("Expected UserID %s, got %s", userID, c.UserID)
		}
	}
}

// TestConcurrencyLimit tests the max containers limit
func TestConcurrencyLimit(t *testing.T) {
	workspaceRoot, ipcDir, tmpDir, err := getTestDirs("sandbox-concurrency-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	if tmpDir != "" {
		defer os.RemoveAll(tmpDir)
	}

	cfg := &Config{
		Image:         "yaoapp/sandbox-claude:latest",
		WorkspaceRoot: workspaceRoot,
		IPCDir:        ipcDir,
		MaxContainers: 2, // Low limit for testing
		IdleTimeout:   1 * time.Minute,
		MaxMemory:     "256m",
		MaxCPU:        0.25,
	}

	m, err := NewManager(cfg)
	if err != nil {
		if strings.Contains(err.Error(), "Docker not available") {
			t.Skip("Docker not available")
		}
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer m.Close()

	ctx := context.Background()

	// Create containers up to limit
	containers := make([]*Container, 0)
	for i := 0; i < cfg.MaxContainers; i++ {
		c, err := m.GetOrCreate(ctx, "limit-user", "limit-chat-"+string(rune('a'+i)))
		if err != nil {
			t.Fatalf("GetOrCreate failed for container %d: %v", i, err)
		}
		containers = append(containers, c)
	}

	// Try to create one more - should fail
	_, err = m.GetOrCreate(ctx, "limit-user", "limit-chat-extra")
	if err != ErrTooManyContainers {
		t.Errorf("Expected ErrTooManyContainers, got: %v", err)
	}

	// Cleanup
	for _, c := range containers {
		m.Remove(ctx, c.Name)
	}
}

// TestConcurrentAccess tests concurrent container access
func TestConcurrentAccess(t *testing.T) {
	m := skipIfNoDocker(t)

	ctx := context.Background()
	userID := "concurrent-user"
	chatID := "concurrent-chat-" + time.Now().Format("20060102150405")

	// Create container
	container, err := m.GetOrCreate(ctx, userID, chatID)
	if err != nil {
		t.Fatalf("GetOrCreate failed: %v", err)
	}
	defer m.Remove(ctx, container.Name)

	// Concurrent GetOrCreate calls should return same container
	var wg sync.WaitGroup
	results := make(chan *Container, 10)
	errors := make(chan error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c, err := m.GetOrCreate(ctx, userID, chatID)
			if err != nil {
				errors <- err
				return
			}
			results <- c
		}()
	}

	wg.Wait()
	close(results)
	close(errors)

	// Check for errors
	for err := range errors {
		t.Errorf("Concurrent GetOrCreate error: %v", err)
	}

	// All results should have same ID
	var firstID string
	for c := range results {
		if firstID == "" {
			firstID = c.ID
		} else if c.ID != firstID {
			t.Errorf("Expected same container ID, got different: %s vs %s", firstID, c.ID)
		}
	}
}

// TestContainerNotFound tests operations on non-existent container
func TestContainerNotFound(t *testing.T) {
	m := skipIfNoDocker(t)

	ctx := context.Background()
	fakeName := "yao-sandbox-fake-user-fake-chat"

	// Test Stop on non-existent (should not error)
	if err := m.Stop(ctx, fakeName); err != nil {
		t.Errorf("Stop on non-existent container should not error: %v", err)
	}

	// Test Remove on non-existent (should not error)
	if err := m.Remove(ctx, fakeName); err != nil {
		t.Errorf("Remove on non-existent container should not error: %v", err)
	}

	// Test ensureRunning on non-existent (should error)
	if err := m.ensureRunning(ctx, fakeName); err != ErrContainerNotFound {
		t.Errorf("Expected ErrContainerNotFound, got: %v", err)
	}
}

// TestCleanup tests the cleanup function
func TestCleanup(t *testing.T) {
	workspaceRoot, ipcDir, tmpDir, err := getTestDirs("sandbox-cleanup-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	if tmpDir != "" {
		defer os.RemoveAll(tmpDir)
	}

	cfg := &Config{
		Image:         "yaoapp/sandbox-claude:latest",
		WorkspaceRoot: workspaceRoot,
		IPCDir:        ipcDir,
		MaxContainers: 10,
		IdleTimeout:   100 * time.Millisecond, // Very short for testing
		MaxMemory:     "256m",
		MaxCPU:        0.25,
	}

	m, err := NewManager(cfg)
	if err != nil {
		if strings.Contains(err.Error(), "Docker not available") {
			t.Skip("Docker not available")
		}
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer m.Close()

	ctx := context.Background()

	// Create and start container
	container, err := m.GetOrCreate(ctx, "cleanup-user", "cleanup-chat")
	if err != nil {
		t.Fatalf("GetOrCreate failed: %v", err)
	}
	defer m.Remove(ctx, container.Name)

	if err := m.Start(ctx, container.Name); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Verify running
	c, _ := m.containers.Load(container.Name)
	if c.(*Container).Status != StatusRunning {
		t.Fatalf("Container should be running")
	}

	// Set LastUsedAt to past
	c.(*Container).LastUsedAt = time.Now().Add(-1 * time.Hour)

	// Run cleanup
	if err := m.Cleanup(ctx); err != nil {
		t.Fatalf("Cleanup failed: %v", err)
	}

	// Verify stopped
	c, _ = m.containers.Load(container.Name)
	if c.(*Container).Status != StatusStopped {
		t.Errorf("Container should be stopped after cleanup, got: %s", c.(*Container).Status)
	}
}

// TestManagerWithYaoApp tests sandbox with Yao application loaded
// This is the full integration test that loads the Yao application environment
func TestManagerWithYaoApp(t *testing.T) {
	// Check if YAO_TEST_APPLICATION is set
	if os.Getenv("YAO_TEST_APPLICATION") == "" {
		t.Skip("Skipping: YAO_TEST_APPLICATION not set")
	}

	// Prepare Yao test environment
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Now test with the Yao environment loaded
	workspaceRoot, ipcDir, tmpDir, err := getTestDirs("sandbox-yao-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	if tmpDir != "" {
		defer os.RemoveAll(tmpDir)
	}

	cfg := &Config{
		Image:         "yaoapp/sandbox-claude:latest",
		WorkspaceRoot: workspaceRoot,
		IPCDir:        ipcDir,
		MaxContainers: 5,
		IdleTimeout:   5 * time.Minute,
		MaxMemory:     "1g",
		MaxCPU:        1.0,
	}

	m, err := NewManager(cfg)
	if err != nil {
		if strings.Contains(err.Error(), "Docker not available") {
			t.Skip("Docker not available")
		}
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer m.Close()

	ctx := context.Background()

	// Create container
	container, err := m.GetOrCreate(ctx, "yao-user", "yao-chat")
	if err != nil {
		t.Fatalf("GetOrCreate failed: %v", err)
	}
	defer m.Remove(ctx, container.Name)

	// Start container
	if err := m.Start(ctx, container.Name); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Execute a command to verify container is working
	result, err := m.Exec(ctx, container.Name, []string{"node", "--version"}, nil)
	if err != nil {
		t.Fatalf("Exec node --version failed: %v", err)
	}

	if !strings.Contains(result.Stdout, "v") {
		t.Errorf("Expected node version output, got: %s", result.Stdout)
	}

	// Execute Python version check
	result, err = m.Exec(ctx, container.Name, []string{"python3", "--version"}, nil)
	if err != nil {
		t.Fatalf("Exec python3 --version failed: %v", err)
	}

	if !strings.Contains(result.Stdout, "Python") {
		t.Errorf("Expected Python version output, got: %s", result.Stdout)
	}

	t.Log("Sandbox integration with Yao app successful")
}

// TestGetAccessors tests getter methods
func TestGetAccessors(t *testing.T) {
	m := skipIfNoDocker(t)

	// Test GetIPCManager
	ipcMgr := m.GetIPCManager()
	if ipcMgr == nil {
		t.Error("GetIPCManager should not return nil")
	}

	// Test GetConfig
	cfg := m.GetConfig()
	if cfg == nil {
		t.Error("GetConfig should not return nil")
	}

	if cfg.MaxContainers != 5 {
		t.Errorf("Expected MaxContainers 5, got %d", cfg.MaxContainers)
	}
}

// TestEnsureImageAutoPull tests that missing images are automatically pulled
func TestEnsureImageAutoPull(t *testing.T) {
	workspaceRoot, ipcDir, tmpDir, err := getTestDirs("sandbox-autopull-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	if tmpDir != "" {
		defer os.RemoveAll(tmpDir)
	}

	// Use a small known image for testing
	cfg := &Config{
		Image:         "alpine:latest",
		WorkspaceRoot: workspaceRoot,
		IPCDir:        ipcDir,
		MaxContainers: 2,
		IdleTimeout:   1 * time.Minute,
		MaxMemory:     "128m",
		MaxCPU:        0.25,
	}

	m, err := NewManager(cfg)
	if err != nil {
		if strings.Contains(err.Error(), "Docker not available") {
			t.Skip("Docker not available")
		}
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer m.Close()

	ctx := context.Background()

	// Create container - should auto-pull alpine if not present
	container, err := m.GetOrCreate(ctx, "autopull-user", "autopull-chat")
	if err != nil {
		t.Fatalf("GetOrCreate failed (should auto-pull image): %v", err)
	}
	defer m.Remove(ctx, container.Name)

	// Verify container was created
	if container.Status != StatusCreated {
		t.Errorf("Expected status %s, got %s", StatusCreated, container.Status)
	}

	t.Log("Image auto-pull successful")
}
