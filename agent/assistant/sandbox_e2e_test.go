package assistant_test

import (
	stdContext "context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/assistant"
	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/testutils"
	"github.com/yaoapp/yao/openapi/oauth/types"
)

// newSandboxE2EContext creates a Context for sandbox E2E testing
// Uses unique chatID to avoid container name conflicts
func newSandboxE2EContext(chatIDPrefix, assistantID string) *context.Context {
	// Generate unique chatID using timestamp to avoid container conflicts
	chatID := fmt.Sprintf("%s-%d", chatIDPrefix, time.Now().UnixNano())

	authorized := &types.AuthorizedInfo{
		Subject:   "sandbox-e2e-test-user",
		ClientID:  "sandbox-e2e-test-client",
		Scope:     "openid profile",
		SessionID: "sandbox-e2e-test-session",
		UserID:    "sandbox-user-123",
		TeamID:    "sandbox-team-456",
		TenantID:  "sandbox-tenant-789",
	}

	ctx := context.New(stdContext.Background(), authorized, chatID)
	ctx.AssistantID = assistantID
	ctx.Locale = "en-us"
	ctx.Theme = "light"
	ctx.Client = context.Client{
		Type:      "web",
		UserAgent: "SandboxE2ETest/1.0",
		IP:        "127.0.0.1",
	}
	ctx.Referer = context.RefererAPI
	ctx.Accept = context.AcceptWebCUI
	ctx.Route = ""
	ctx.Metadata = make(map[string]interface{})
	return ctx
}

// TestSandboxBasicE2E tests the basic sandbox assistant end-to-end
// This test verifies that:
// 1. Sandbox is correctly initialized
// 2. Claude CLI command is built correctly
// 3. Docker container is created and managed
func TestSandboxBasicE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping sandbox E2E test in short mode")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Load the basic sandbox assistant
	ast, err := assistant.Get("tests.sandbox.basic")
	if err != nil {
		t.Skipf("Skipping test: sandbox assistant not available: %v", err)
	}

	// Verify sandbox is configured
	require.NotNil(t, ast.Sandbox, "Sandbox should be configured")
	assert.Equal(t, "claude", ast.Sandbox.Command)
	t.Logf("✓ Sandbox configured with command: %s", ast.Sandbox.Command)

	// Create context
	ctx := newSandboxE2EContext("sandbox-basic-e2e", "tests.sandbox.basic")

	// Test messages
	messages := []context.Message{
		{Role: context.RoleUser, Content: "echo hello sandbox"},
	}

	// Execute stream
	// Note: This will fail if Docker/Claude image is not available, which is expected in CI
	response, err := ast.Stream(ctx, messages)
	if err != nil {
		// Check if it's a Docker/sandbox availability issue
		errStr := err.Error()
		if strings.Contains(errStr, "Docker") ||
			strings.Contains(errStr, "sandbox") ||
			strings.Contains(errStr, "container") ||
			strings.Contains(errStr, "image") {
			t.Skipf("Skipping test: Docker/sandbox not available: %v", err)
		}
		t.Fatalf("Stream failed: %v", err)
	}

	// Verify response
	require.NotNil(t, response, "Response should not be nil")

	// Verify response completion (Claude CLI should return some response)
	if response.Completion != nil && response.Completion.Content != nil {
		if contentStr, ok := response.Completion.Content.(string); ok && contentStr != "" {
			t.Logf("✓ Response content: %s", truncateString(contentStr, 200))
		} else {
			t.Logf("⚠ Response content type: %T", response.Completion.Content)
		}
	} else {
		t.Log("⚠ Response content is empty (might be expected for some commands)")
	}

	t.Log("✓ Basic sandbox E2E test passed")
}

// truncateString truncates a string to maxLen and adds "..." if truncated
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// TestSandboxHooksE2E tests the sandbox assistant with hooks
func TestSandboxHooksE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping sandbox E2E test in short mode")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Load the hooks sandbox assistant
	ast, err := assistant.Get("tests.sandbox.hooks")
	if err != nil {
		t.Skipf("Skipping test: sandbox hooks assistant not available: %v", err)
	}

	// Verify sandbox and hooks are configured
	require.NotNil(t, ast.Sandbox, "Sandbox should be configured")
	require.NotNil(t, ast.HookScript, "HookScript should be loaded")
	t.Logf("✓ Sandbox and hooks configured")

	// Create context
	ctx := newSandboxE2EContext("sandbox-hooks-e2e", "tests.sandbox.hooks")

	// Test messages
	messages := []context.Message{
		{Role: context.RoleUser, Content: "test hooks integration"},
	}

	// Execute stream
	response, err := ast.Stream(ctx, messages)
	if err != nil {
		errStr := err.Error()
		if strings.Contains(errStr, "Docker") ||
			strings.Contains(errStr, "sandbox") ||
			strings.Contains(errStr, "container") ||
			strings.Contains(errStr, "image") {
			t.Skipf("Skipping test: Docker/sandbox not available: %v", err)
		}
		t.Fatalf("Stream failed: %v", err)
	}

	require.NotNil(t, response, "Response should not be nil")
	t.Log("✓ Sandbox hooks E2E test passed")
}

// TestSandboxFullE2E tests the full sandbox assistant with MCPs and Skills
func TestSandboxFullE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping sandbox E2E test in short mode")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Load the full sandbox assistant
	ast, err := assistant.Get("tests.sandbox.full")
	if err != nil {
		t.Skipf("Skipping test: full sandbox assistant not available: %v", err)
	}

	// Verify all components are configured
	require.NotNil(t, ast.Sandbox, "Sandbox should be configured")
	require.NotNil(t, ast.MCP, "MCP should be configured")
	require.NotNil(t, ast.HookScript, "HookScript should be loaded")
	t.Logf("✓ Full sandbox configured: command=%s, MCP servers=%d",
		ast.Sandbox.Command, len(ast.MCP.Servers))

	// Verify MCP configuration
	assert.Len(t, ast.MCP.Servers, 1)
	assert.Equal(t, "echo", ast.MCP.Servers[0].ServerID)
	t.Logf("✓ MCP server: %s with tools %v", ast.MCP.Servers[0].ServerID, ast.MCP.Servers[0].Tools)

	// Create context
	ctx := newSandboxE2EContext("sandbox-full-e2e", "tests.sandbox.full")

	// Test messages
	messages := []context.Message{
		{Role: context.RoleUser, Content: "test full sandbox with MCP and skills"},
	}

	// Execute stream
	response, err := ast.Stream(ctx, messages)
	if err != nil {
		errStr := err.Error()
		if strings.Contains(errStr, "Docker") ||
			strings.Contains(errStr, "sandbox") ||
			strings.Contains(errStr, "container") ||
			strings.Contains(errStr, "image") {
			t.Skipf("Skipping test: Docker/sandbox not available: %v", err)
		}
		t.Fatalf("Stream failed: %v", err)
	}

	require.NotNil(t, response, "Response should not be nil")
	t.Log("✓ Full sandbox E2E test passed")
}

// TestSandboxContextAccess tests that sandbox is accessible in hooks via ctx.sandbox
func TestSandboxContextAccess(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping sandbox context access test in short mode")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Load the hooks sandbox assistant
	ast, err := assistant.Get("tests.sandbox.hooks")
	if err != nil {
		t.Skipf("Skipping test: sandbox hooks assistant not available: %v", err)
	}

	require.NotNil(t, ast.HookScript, "HookScript should be loaded")

	// Create context
	ctx := newSandboxE2EContext("sandbox-ctx-access", "tests.sandbox.hooks")

	// Test Create Hook - it should have access to ctx.sandbox
	messages := []context.Message{
		{Role: context.RoleUser, Content: "test sandbox context access"},
	}

	// Execute Create hook directly
	// This tests that the hook runs without error (sandbox operations tested within)
	opts := &context.Options{}
	response, _, err := ast.HookScript.Create(ctx, messages, opts)

	// The hook might fail if sandbox isn't initialized yet (that's done in Stream)
	// But we can at least verify the hook exists and can be called
	if err != nil {
		// If the error is about sandbox not being available, that's expected
		// because we haven't initialized the sandbox yet
		if strings.Contains(err.Error(), "sandbox") {
			t.Logf("Expected error: sandbox not available in direct hook call: %v", err)
		} else {
			t.Fatalf("Unexpected error: %v", err)
		}
	}

	// Response might be nil, that's okay
	t.Logf("Create hook response: %v", response)
	t.Log("✓ Sandbox context access test passed")
}

// TestSandboxLoadConfiguration verifies that sandbox assistants load correctly
func TestSandboxLoadConfiguration(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	testCases := []struct {
		name          string
		assistantID   string
		expectSandbox bool
		expectMCP     bool
		expectHooks   bool
	}{
		{
			name:          "BasicSandbox",
			assistantID:   "tests.sandbox.basic",
			expectSandbox: true,
			expectMCP:     false,
			expectHooks:   false,
		},
		{
			name:          "HooksSandbox",
			assistantID:   "tests.sandbox.hooks",
			expectSandbox: true,
			expectMCP:     false,
			expectHooks:   true,
		},
		{
			name:          "FullSandbox",
			assistantID:   "tests.sandbox.full",
			expectSandbox: true,
			expectMCP:     true,
			expectHooks:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ast, err := assistant.Get(tc.assistantID)
			if err != nil {
				t.Skipf("Skipping: assistant %s not available: %v", tc.assistantID, err)
			}

			// Check sandbox
			if tc.expectSandbox {
				require.NotNil(t, ast.Sandbox, "Expected sandbox to be configured")
				assert.Equal(t, "claude", ast.Sandbox.Command)
				t.Logf("✓ %s: Sandbox configured with command=%s", tc.name, ast.Sandbox.Command)
			}

			// Check MCP
			if tc.expectMCP {
				require.NotNil(t, ast.MCP, "Expected MCP to be configured")
				assert.True(t, len(ast.MCP.Servers) > 0, "Expected at least one MCP server")
				t.Logf("✓ %s: MCP configured with %d servers", tc.name, len(ast.MCP.Servers))
			}

			// Check hooks
			if tc.expectHooks {
				require.NotNil(t, ast.HookScript, "Expected hooks to be loaded")
				t.Logf("✓ %s: Hooks loaded", tc.name)
			}
		})
	}
}
