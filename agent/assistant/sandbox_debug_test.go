package assistant_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/assistant"
	"github.com/yaoapp/yao/agent/testutils"
)

// TestSandboxDebugHasSandbox tests the HasSandbox method directly
func TestSandboxDebugHasSandbox(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	testCases := []struct {
		name        string
		assistantID string
		expectTrue  bool
	}{
		{"BasicSandbox", "tests.sandbox.basic", true},
		{"HooksSandbox", "tests.sandbox.hooks", true},
		{"FullSandbox", "tests.sandbox.full", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ast, err := assistant.Get(tc.assistantID)
			require.NoError(t, err, "Failed to get assistant %s", tc.assistantID)

			// Check Sandbox struct
			t.Logf("Assistant ID: %s", ast.ID)
			t.Logf("Sandbox: %+v", ast.Sandbox)

			if ast.Sandbox != nil {
				t.Logf("Sandbox.Command: %q", ast.Sandbox.Command)
				t.Logf("Sandbox.Timeout: %s", ast.Sandbox.Timeout)
				t.Logf("Sandbox.Image: %s", ast.Sandbox.Image)
				t.Logf("Sandbox.Arguments: %v", ast.Sandbox.Arguments)
			}

			// Check HasSandbox
			hasSandbox := ast.HasSandbox()
			t.Logf("HasSandbox() = %v", hasSandbox)

			if tc.expectTrue {
				assert.True(t, hasSandbox, "Expected HasSandbox() to be true for %s", tc.assistantID)
			} else {
				assert.False(t, hasSandbox, "Expected HasSandbox() to be false for %s", tc.assistantID)
			}
		})
	}
}

// TestSandboxDebugPrompts tests if Prompts is set (affects execution path)
func TestSandboxDebugPrompts(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	ast, err := assistant.Get("tests.sandbox.basic")
	require.NoError(t, err)

	t.Logf("Assistant ID: %s", ast.ID)
	t.Logf("Prompts: %v", ast.Prompts)
	t.Logf("MCP: %v", ast.MCP)
	t.Logf("HasSandbox: %v", ast.HasSandbox())

	// The condition in agent.go is:
	// if ast.Prompts != nil || ast.MCP != nil {
	//   // ... execute LLM
	//   if ast.HasSandbox() {
	//     // sandbox path
	//   } else {
	//     // direct LLM path
	//   }
	// }
	// So we need Prompts or MCP to be non-nil
	if ast.Prompts == nil && ast.MCP == nil {
		t.Log("WARNING: Neither Prompts nor MCP is set, LLM phase will be skipped entirely!")
	}
}
