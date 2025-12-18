package assistant_test

import (
	stdContext "context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/assistant"
	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/output/message"
	"github.com/yaoapp/yao/agent/testutils"
	"github.com/yaoapp/yao/openapi/oauth/types"
)

// newSearchAutoHookDisableTestContext creates a test context
func newSearchAutoHookDisableTestContext(chatID, assistantID string) *context.Context {
	authorized := &types.AuthorizedInfo{
		Subject:  "test-user",
		UserID:   "test-123",
		TenantID: "test-tenant",
	}

	ctx := context.New(stdContext.Background(), authorized, chatID)
	ctx.ID = chatID
	ctx.AssistantID = assistantID
	ctx.Locale = "en-us"
	ctx.Client = context.Client{
		Type: "web",
		IP:   "127.0.0.1",
	}
	ctx.Referer = context.RefererAPI
	ctx.Accept = context.AcceptWebCUI
	ctx.IDGenerator = message.NewIDGenerator()
	ctx.Metadata = make(map[string]interface{})
	return ctx
}

func TestSearchAutoHookDisable(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	ast, err := assistant.LoadPath("/assistants/tests/search-auto-hook-disable")
	require.NoError(t, err)
	require.NotNil(t, ast)

	t.Run("ShouldHaveSearchConfigEnabled", func(t *testing.T) {
		// Search config is enabled in package.yao
		assert.NotNil(t, ast.Search, "search config should be set")
		assert.NotNil(t, ast.Uses, "uses config should be set")
		assert.Equal(t, "builtin", ast.Uses.Search, "uses.search should be builtin in config")
	})

	t.Run("ShouldHaveHookScript", func(t *testing.T) {
		// Hook script should be loaded
		assert.NotNil(t, ast.HookScript, "hook script should be loaded")
	})

	t.Run("HookShouldDisableSearch", func(t *testing.T) {
		// Create context
		ctx := newSearchAutoHookDisableTestContext("test-chat-id", "tests.search-auto-hook-disable")

		// Create messages
		messages := []context.Message{
			{
				Role:    "user",
				Content: "Test message",
			},
		}

		// Call Create hook directly
		opts := &context.Options{}
		response, _, err := ast.HookScript.Create(ctx, messages, opts)
		require.NoError(t, err)
		require.NotNil(t, response)

		// Verify hook returns uses.search = "disabled"
		assert.NotNil(t, response.Uses, "hook should return uses")
		assert.Equal(t, "disabled", response.Uses.Search, "hook should disable search")
	})

	t.Run("StreamShouldRespectHookDisable", func(t *testing.T) {
		// Get agent via assistant.Get (required for Stream)
		agent, err := assistant.Get("tests.search-auto-hook-disable")
		require.NoError(t, err)
		require.NotNil(t, agent)

		// Create context
		ctx := newSearchAutoHookDisableTestContext("test-search-hook-disable", "tests.search-auto-hook-disable")

		// Create messages
		messages := []context.Message{
			{
				Role:    "user",
				Content: "What is AI?",
			},
		}

		// Execute stream - hook will disable search
		response, err := agent.Stream(ctx, messages)
		require.NoError(t, err)
		require.NotNil(t, response)

		assert.NotNil(t, response.Completion, "should have completion")
		t.Logf("✓ Stream executed with hook disabling search")
	})
}
