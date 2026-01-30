package context_test

import (
	stdContext "context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v8 "github.com/yaoapp/gou/runtime/v8"
	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/config"
	infraSandbox "github.com/yaoapp/yao/sandbox"
	"github.com/yaoapp/yao/test"
)

// createTestSandboxManager creates a real sandbox manager for testing
func createTestSandboxManager(t *testing.T) *infraSandbox.Manager {
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

// createTestContainer creates a container and returns a cleanup function
func createTestContainer(t *testing.T, manager *infraSandbox.Manager, userID, chatID string) (*infraSandbox.Container, func()) {
	container, err := manager.GetOrCreate(stdContext.Background(), userID, chatID)
	require.NoError(t, err)
	require.NotNil(t, container)

	// Return cleanup function that removes the container
	cleanup := func() {
		err := manager.Remove(stdContext.Background(), container.Name)
		if err != nil {
			t.Logf("Warning: failed to cleanup container %s: %v", container.Name, err)
		}
	}

	return container, cleanup
}

// realSandboxExecutor wraps infraSandbox.Manager to implement context.SandboxExecutor
type realSandboxExecutor struct {
	manager       *infraSandbox.Manager
	containerName string
	workDir       string
}

func (e *realSandboxExecutor) ReadFile(ctx stdContext.Context, path string) ([]byte, error) {
	fullPath := e.workDir + "/" + path
	return e.manager.ReadFile(ctx, e.containerName, fullPath)
}

func (e *realSandboxExecutor) WriteFile(ctx stdContext.Context, path string, content []byte) error {
	fullPath := e.workDir + "/" + path
	return e.manager.WriteFile(ctx, e.containerName, fullPath, content)
}

func (e *realSandboxExecutor) ListDir(ctx stdContext.Context, path string) ([]infraSandbox.FileInfo, error) {
	fullPath := e.workDir + "/" + path
	return e.manager.ListDir(ctx, e.containerName, fullPath)
}

func (e *realSandboxExecutor) Exec(ctx stdContext.Context, cmd []string) (string, error) {
	result, err := e.manager.Exec(ctx, e.containerName, cmd, &infraSandbox.ExecOptions{
		WorkDir: e.workDir,
	})
	if err != nil {
		return "", err
	}
	return result.Stdout, nil
}

func (e *realSandboxExecutor) GetWorkDir() string {
	return e.workDir
}

// TestJsSandboxNotAvailable tests ctx.sandbox when not configured
func TestJsSandboxNotAvailable(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	ctx := context.New(stdContext.Background(), nil, "test-chat-no-sandbox")
	ctx.AssistantID = "test-assistant"

	// Test that ctx.sandbox is undefined when not configured
	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				if (ctx.sandbox === undefined || ctx.sandbox === null) {
					return { success: true, hasSandbox: false };
				}
				return { success: true, hasSandbox: true };
			} catch (error) {
				return { success: false, error: error.message };
			}
		}`, ctx)

	require.NoError(t, err)
	result, ok := res.(map[string]interface{})
	require.True(t, ok, "Expected map result")
	assert.Equal(t, true, result["success"])
	assert.Equal(t, false, result["hasSandbox"], "ctx.sandbox should not be available when not configured")
}

// TestJsSandboxWriteFile tests ctx.sandbox.WriteFile via JavaScript
func TestJsSandboxWriteFile(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	manager := createTestSandboxManager(t)
	if manager == nil {
		return
	}
	defer manager.Close()

	// Create container with auto-cleanup
	container, cleanup := createTestContainer(t, manager, "test-user", "test-js-writefile")
	defer cleanup()

	executor := &realSandboxExecutor{
		manager:       manager,
		containerName: container.Name,
		workDir:       "/workspace",
	}

	// Create context with sandbox
	ctx := context.New(stdContext.Background(), nil, "test-chat-writefile")
	ctx.AssistantID = "test-assistant"
	ctx.SetSandboxExecutor(executor)

	// Test WriteFile via JavaScript
	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				if (!ctx.sandbox) {
					return { success: false, error: "sandbox not available" };
				}
				
				// Write a file
				ctx.sandbox.WriteFile("js-test.txt", "Hello from JavaScript!");
				
				return { success: true };
			} catch (error) {
				return { success: false, error: error.message };
			}
		}`, ctx)

	require.NoError(t, err)
	result, ok := res.(map[string]interface{})
	require.True(t, ok, "Expected map result")
	assert.Equal(t, true, result["success"], "WriteFile should succeed: %v", result["error"])

	// Verify file was written by reading it back directly
	content, err := executor.ReadFile(stdContext.Background(), "js-test.txt")
	require.NoError(t, err)
	assert.Equal(t, "Hello from JavaScript!", string(content))
}

// TestJsSandboxReadFile tests ctx.sandbox.ReadFile via JavaScript
func TestJsSandboxReadFile(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	manager := createTestSandboxManager(t)
	if manager == nil {
		return
	}
	defer manager.Close()

	// Create container with auto-cleanup
	container, cleanup := createTestContainer(t, manager, "test-user", "test-js-readfile")
	defer cleanup()

	executor := &realSandboxExecutor{
		manager:       manager,
		containerName: container.Name,
		workDir:       "/workspace",
	}

	// Write a file first
	err := executor.WriteFile(stdContext.Background(), "read-test.txt", []byte("Content to read"))
	require.NoError(t, err)

	// Create context with sandbox
	ctx := context.New(stdContext.Background(), nil, "test-chat-readfile")
	ctx.AssistantID = "test-assistant"
	ctx.SetSandboxExecutor(executor)

	// Test ReadFile via JavaScript
	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				if (!ctx.sandbox) {
					return { success: false, error: "sandbox not available" };
				}
				
				// Read the file
				const content = ctx.sandbox.ReadFile("read-test.txt");
				
				return { success: true, content: content };
			} catch (error) {
				return { success: false, error: error.message };
			}
		}`, ctx)

	require.NoError(t, err)
	result, ok := res.(map[string]interface{})
	require.True(t, ok, "Expected map result")
	assert.Equal(t, true, result["success"], "ReadFile should succeed: %v", result["error"])
	assert.Equal(t, "Content to read", result["content"])
}

// TestJsSandboxListDir tests ctx.sandbox.ListDir via JavaScript
func TestJsSandboxListDir(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	manager := createTestSandboxManager(t)
	if manager == nil {
		return
	}
	defer manager.Close()

	// Create container with auto-cleanup
	container, cleanup := createTestContainer(t, manager, "test-user", "test-js-listdir")
	defer cleanup()

	executor := &realSandboxExecutor{
		manager:       manager,
		containerName: container.Name,
		workDir:       "/workspace",
	}

	// Write some files first
	err := executor.WriteFile(stdContext.Background(), "file1.txt", []byte("content1"))
	require.NoError(t, err)
	err = executor.WriteFile(stdContext.Background(), "file2.txt", []byte("content2"))
	require.NoError(t, err)

	// Create context with sandbox
	ctx := context.New(stdContext.Background(), nil, "test-chat-listdir")
	ctx.AssistantID = "test-assistant"
	ctx.SetSandboxExecutor(executor)

	// Test ListDir via JavaScript
	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				if (!ctx.sandbox) {
					return { success: false, error: "sandbox not available" };
				}
				
				// List directory
				const files = ctx.sandbox.ListDir(".");
				
				// Find our test files
				const fileNames = files.map(f => f.name);
				const hasFile1 = fileNames.includes("file1.txt");
				const hasFile2 = fileNames.includes("file2.txt");
				
				return { 
					success: true, 
					fileCount: files.length,
					hasFile1: hasFile1,
					hasFile2: hasFile2,
					files: fileNames
				};
			} catch (error) {
				return { success: false, error: error.message };
			}
		}`, ctx)

	require.NoError(t, err)
	result, ok := res.(map[string]interface{})
	require.True(t, ok, "Expected map result")
	assert.Equal(t, true, result["success"], "ListDir should succeed: %v", result["error"])
	assert.Equal(t, true, result["hasFile1"], "Should find file1.txt")
	assert.Equal(t, true, result["hasFile2"], "Should find file2.txt")
}

// TestJsSandboxExec tests ctx.sandbox.Exec via JavaScript
func TestJsSandboxExec(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	manager := createTestSandboxManager(t)
	if manager == nil {
		return
	}
	defer manager.Close()

	// Create container with auto-cleanup
	container, cleanup := createTestContainer(t, manager, "test-user", "test-js-exec")
	defer cleanup()

	executor := &realSandboxExecutor{
		manager:       manager,
		containerName: container.Name,
		workDir:       "/workspace",
	}

	// Create context with sandbox
	ctx := context.New(stdContext.Background(), nil, "test-chat-exec")
	ctx.AssistantID = "test-assistant"
	ctx.SetSandboxExecutor(executor)

	// Test Exec via JavaScript
	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				if (!ctx.sandbox) {
					return { success: false, error: "sandbox not available" };
				}
				
				// Execute echo command
				const output = ctx.sandbox.Exec(["echo", "hello-from-js"]);
				
				return { success: true, output: output.trim() };
			} catch (error) {
				return { success: false, error: error.message };
			}
		}`, ctx)

	require.NoError(t, err)
	result, ok := res.(map[string]interface{})
	require.True(t, ok, "Expected map result")
	assert.Equal(t, true, result["success"], "Exec should succeed: %v", result["error"])
	// Output may contain Docker stream header bytes, so use Contains
	output, _ := result["output"].(string)
	assert.Contains(t, output, "hello-from-js", "Exec output should contain expected text")
}

// TestJsSandboxWorkdir tests ctx.sandbox.workdir property via JavaScript
func TestJsSandboxWorkdir(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	manager := createTestSandboxManager(t)
	if manager == nil {
		return
	}
	defer manager.Close()

	// Create container with auto-cleanup
	container, cleanup := createTestContainer(t, manager, "test-user", "test-js-workdir")
	defer cleanup()

	executor := &realSandboxExecutor{
		manager:       manager,
		containerName: container.Name,
		workDir:       "/workspace",
	}

	// Create context with sandbox
	ctx := context.New(stdContext.Background(), nil, "test-chat-workdir")
	ctx.AssistantID = "test-assistant"
	ctx.SetSandboxExecutor(executor)

	// Test workdir property via JavaScript
	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				if (!ctx.sandbox) {
					return { success: false, error: "sandbox not available" };
				}
				
				// Get workdir property
				const workdir = ctx.sandbox.workdir;
				
				return { success: true, workdir: workdir };
			} catch (error) {
				return { success: false, error: error.message };
			}
		}`, ctx)

	require.NoError(t, err)
	result, ok := res.(map[string]interface{})
	require.True(t, ok, "Expected map result")
	assert.Equal(t, true, result["success"], "workdir access should succeed: %v", result["error"])
	assert.Equal(t, "/workspace", result["workdir"])
}

// TestJsSandboxCompleteWorkflow tests a complete workflow via JavaScript
func TestJsSandboxCompleteWorkflow(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	manager := createTestSandboxManager(t)
	if manager == nil {
		return
	}
	defer manager.Close()

	// Create container with auto-cleanup
	container, cleanup := createTestContainer(t, manager, "test-user", "test-js-workflow")
	defer cleanup()

	executor := &realSandboxExecutor{
		manager:       manager,
		containerName: container.Name,
		workDir:       "/workspace",
	}

	// Create context with sandbox
	ctx := context.New(stdContext.Background(), nil, "test-chat-workflow")
	ctx.AssistantID = "test-assistant"
	ctx.SetSandboxExecutor(executor)

	// Test complete workflow: write file, exec cat, verify content
	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			try {
				if (!ctx.sandbox) {
					return { success: false, error: "sandbox not available" };
				}
				
				// 1. Check workdir
				const workdir = ctx.sandbox.workdir;
				if (workdir !== "/workspace") {
					return { success: false, error: "unexpected workdir: " + workdir };
				}
				
				// 2. Write a file
				const testContent = "Test workflow content: " + Date.now();
				ctx.sandbox.WriteFile("workflow-test.txt", testContent);
				
				// 3. Read it back
				const readContent = ctx.sandbox.ReadFile("workflow-test.txt");
				if (readContent !== testContent) {
					return { success: false, error: "content mismatch after read" };
				}
				
				// 4. List directory and verify file exists
				const files = ctx.sandbox.ListDir(".");
				const fileNames = files.map(f => f.name);
				if (!fileNames.includes("workflow-test.txt")) {
					return { success: false, error: "file not found in listing" };
				}
				
				// 5. Execute cat command
				const catOutput = ctx.sandbox.Exec(["cat", workdir + "/workflow-test.txt"]);
				if (!catOutput.includes("Test workflow content")) {
					return { success: false, error: "cat output mismatch" };
				}
				
				// 6. Execute pwd command
				const pwdOutput = ctx.sandbox.Exec(["pwd"]);
				if (!pwdOutput.includes("/workspace")) {
					return { success: false, error: "pwd output mismatch: " + pwdOutput };
				}
				
				return { 
					success: true, 
					workdir: workdir,
					content: readContent,
					fileCount: files.length
				};
			} catch (error) {
				return { success: false, error: error.message, stack: error.stack };
			}
		}`, ctx)

	require.NoError(t, err)
	result, ok := res.(map[string]interface{})
	require.True(t, ok, "Expected map result")
	assert.Equal(t, true, result["success"], "Complete workflow should succeed: %v", result["error"])
	assert.Equal(t, "/workspace", result["workdir"])
}
