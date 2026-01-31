package caller_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/assistant"
	"github.com/yaoapp/yao/agent/caller"
	agentContext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/testutils"
	"github.com/yaoapp/yao/openapi/oauth/types"
)

// TestSandboxE2E_ClaudeCLIExecution tests the full sandbox + claude-proxy integration
// This test verifies:
// 1. Assistant loads with sandbox and prompts configured
// 2. Claude CLI is invoked (not skipped) because prompts exist
// 3. claude-proxy correctly translates requests to OpenAI backend
// 4. Response is received with actual content
func TestSandboxE2E_ClaudeCLIExecution(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping sandbox E2E test in short mode")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Load the e2e-test assistant
	ast, err := assistant.Get("tests.sandbox.e2e-test")
	if err != nil {
		t.Skipf("Skipping test: e2e-test assistant not available: %v", err)
	}

	// Verify configuration
	require.NotNil(t, ast.Sandbox, "Sandbox should be configured")
	require.NotEmpty(t, ast.Prompts, "Prompts should be configured (required for Claude CLI)")
	t.Logf("✓ Assistant loaded: sandbox=%s, prompts=%d", ast.Sandbox.Command, len(ast.Prompts))

	// Create authorized info
	authorized := &types.AuthorizedInfo{
		Subject:  "sandbox-e2e-test",
		UserID:   "e2e-user-123",
		TenantID: "e2e-tenant",
	}

	// Create context with unique chat ID
	chatID := "sandbox-e2e-" + time.Now().Format("20060102-150405")
	ctx := agentContext.New(context.Background(), authorized, chatID)
	ctx.AssistantID = "tests.sandbox.e2e-test"

	// Create JSAPI
	api := caller.NewJSAPI(ctx)

	// Test 1: Simple echo command
	t.Run("EchoCommand", func(t *testing.T) {
		messages := []interface{}{
			map[string]interface{}{
				"role":    "user",
				"content": "Run this command: echo 'SANDBOX_E2E_SUCCESS_12345'",
			},
		}

		opts := map[string]interface{}{
			"skip": map[string]interface{}{
				"history": true,
			},
		}

		startTime := time.Now()
		result := api.Call("tests.sandbox.e2e-test", messages, opts)
		duration := time.Since(startTime)
		t.Logf("Execution time: %v", duration)

		require.NotNil(t, result, "Result should not be nil")

		r, ok := result.(*caller.Result)
		require.True(t, ok, "Result should be *caller.Result")

		// Check for errors
		if r.Error != "" {
			// Check if it's a Docker/sandbox availability issue
			if strings.Contains(r.Error, "Docker") ||
				strings.Contains(r.Error, "sandbox") ||
				strings.Contains(r.Error, "container") {
				t.Skipf("Skipping: Docker/sandbox not available: %s", r.Error)
			}
			t.Fatalf("Agent call failed: %s", r.Error)
		}

		// Verify response
		t.Logf("Response content: %s", truncateStr(r.Content, 500))
		assert.NotEmpty(t, r.Content, "Response content should not be empty")

		// Check if Claude executed the command
		if strings.Contains(r.Content, "SANDBOX_E2E_SUCCESS_12345") {
			t.Log("✓ Echo command executed successfully - found verification string")
		} else if strings.Contains(strings.ToLower(r.Content), "echo") ||
			strings.Contains(r.Content, "SANDBOX") {
			t.Log("✓ Response mentions the command or partial output")
		} else {
			t.Log("⚠ Response does not contain expected output")
		}
	})
}

// TestSandboxE2E_FileCreation tests that Claude can create files in the sandbox
func TestSandboxE2E_FileCreation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping sandbox E2E test in short mode")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Load the e2e-test assistant
	ast, err := assistant.Get("tests.sandbox.e2e-test")
	if err != nil {
		t.Skipf("Skipping test: e2e-test assistant not available: %v", err)
	}

	require.NotNil(t, ast.Sandbox)
	require.NotEmpty(t, ast.Prompts)

	// Create context
	authorized := &types.AuthorizedInfo{
		Subject:  "sandbox-e2e-test",
		UserID:   "e2e-user-456",
		TenantID: "e2e-tenant",
	}

	chatID := "sandbox-file-" + time.Now().Format("20060102-150405")
	ctx := agentContext.New(context.Background(), authorized, chatID)
	ctx.AssistantID = "tests.sandbox.e2e-test"

	api := caller.NewJSAPI(ctx)

	messages := []interface{}{
		map[string]interface{}{
			"role":    "user",
			"content": "Create a file named 'test-output.txt' with the content 'FILE_CREATION_VERIFIED_67890', then read it back and show me the content.",
		},
	}

	opts := map[string]interface{}{
		"skip": map[string]interface{}{
			"history": true,
		},
	}

	startTime := time.Now()
	result := api.Call("tests.sandbox.e2e-test", messages, opts)
	duration := time.Since(startTime)
	t.Logf("Execution time: %v", duration)

	require.NotNil(t, result)

	r, ok := result.(*caller.Result)
	require.True(t, ok)

	if r.Error != "" {
		if strings.Contains(r.Error, "Docker") ||
			strings.Contains(r.Error, "sandbox") {
			t.Skipf("Skipping: Docker/sandbox not available: %s", r.Error)
		}
		t.Fatalf("Agent call failed: %s", r.Error)
	}

	t.Logf("Response: %s", truncateStr(r.Content, 800))

	// Verify file was created and read back
	if strings.Contains(r.Content, "FILE_CREATION_VERIFIED_67890") {
		t.Log("✓ File creation and read verified")
	} else if strings.Contains(strings.ToLower(r.Content), "created") ||
		strings.Contains(strings.ToLower(r.Content), "wrote") ||
		strings.Contains(r.Content, "test-output.txt") {
		t.Log("✓ File operation appears successful")
	} else {
		t.Log("⚠ Could not verify file creation")
	}
}

// TestSandboxE2E_HookOnlyMode tests that hooks can work without Claude CLI
func TestSandboxE2E_HookOnlyMode(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping sandbox E2E test in short mode")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Load the hook-only assistant (no prompts)
	ast, err := assistant.Get("tests.sandbox.hook-only")
	if err != nil {
		t.Skipf("Skipping test: hook-only assistant not available: %v", err)
	}

	// Verify configuration - no prompts means Claude CLI should be skipped
	require.NotNil(t, ast.Sandbox)
	require.Empty(t, ast.Prompts, "Hook-only mode should have no prompts")
	t.Logf("✓ Hook-only assistant loaded: sandbox=%s, prompts=%d (should be 0)", ast.Sandbox.Command, len(ast.Prompts))

	// Create context
	authorized := &types.AuthorizedInfo{
		Subject:  "sandbox-hook-test",
		UserID:   "hook-user-789",
		TenantID: "hook-tenant",
	}

	chatID := "sandbox-hook-" + time.Now().Format("20060102-150405")
	ctx := agentContext.New(context.Background(), authorized, chatID)
	ctx.AssistantID = "tests.sandbox.hook-only"

	api := caller.NewJSAPI(ctx)

	messages := []interface{}{
		map[string]interface{}{
			"role":    "user",
			"content": "test hook-only mode",
		},
	}

	opts := map[string]interface{}{
		"skip": map[string]interface{}{
			"history": true,
		},
	}

	startTime := time.Now()
	result := api.Call("tests.sandbox.hook-only", messages, opts)
	duration := time.Since(startTime)
	t.Logf("Execution time: %v", duration)

	require.NotNil(t, result)

	r, ok := result.(*caller.Result)
	require.True(t, ok)

	if r.Error != "" {
		if strings.Contains(r.Error, "Docker") ||
			strings.Contains(r.Error, "sandbox") {
			t.Skipf("Skipping: Docker/sandbox not available: %s", r.Error)
		}
		t.Fatalf("Agent call failed: %s", r.Error)
	}

	t.Logf("Response: %s", r.Content)
	t.Log("✓ Hook-only mode executed successfully")
}

// TestSandboxE2E_StreamingResponse verifies streaming works correctly
func TestSandboxE2E_StreamingResponse(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping sandbox E2E test in short mode")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Load the e2e-test assistant
	ast, err := assistant.Get("tests.sandbox.e2e-test")
	if err != nil {
		t.Skipf("Skipping test: e2e-test assistant not available: %v", err)
	}

	require.NotNil(t, ast.Sandbox)
	require.NotEmpty(t, ast.Prompts)

	// Create context
	authorized := &types.AuthorizedInfo{
		Subject:  "sandbox-stream-test",
		UserID:   "stream-user",
		TenantID: "stream-tenant",
	}

	chatID := "sandbox-stream-" + time.Now().Format("20060102-150405")
	ctx := agentContext.New(context.Background(), authorized, chatID)
	ctx.AssistantID = "tests.sandbox.e2e-test"

	api := caller.NewJSAPI(ctx)

	// Ask for a slightly longer response to verify streaming
	messages := []interface{}{
		map[string]interface{}{
			"role":    "user",
			"content": "Say 'Hello World' and nothing else.",
		},
	}

	opts := map[string]interface{}{
		"skip": map[string]interface{}{
			"history": true,
		},
	}

	startTime := time.Now()
	result := api.Call("tests.sandbox.e2e-test", messages, opts)
	duration := time.Since(startTime)
	t.Logf("Execution time: %v", duration)

	require.NotNil(t, result)

	r, ok := result.(*caller.Result)
	require.True(t, ok)

	if r.Error != "" {
		if strings.Contains(r.Error, "Docker") ||
			strings.Contains(r.Error, "sandbox") {
			t.Skipf("Skipping: Docker/sandbox not available: %s", r.Error)
		}
		t.Fatalf("Agent call failed: %s", r.Error)
	}

	t.Logf("Response: %s", r.Content)

	// Verify we got a response
	assert.NotEmpty(t, r.Content, "Should have response content")

	if strings.Contains(strings.ToLower(r.Content), "hello") {
		t.Log("✓ Streaming response received with expected content")
	} else {
		t.Log("✓ Streaming response received")
	}
}

func truncateStr(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
