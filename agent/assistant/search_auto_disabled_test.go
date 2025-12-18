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

// newSearchAutoDisabledTestContext creates a test context
func newSearchAutoDisabledTestContext(chatID, assistantID string) *context.Context {
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

func TestSearchAutoDisabled(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	ast, err := assistant.LoadPath("/assistants/tests/search-auto-disabled")
	require.NoError(t, err)
	require.NotNil(t, ast)

	t.Run("ShouldHaveSearchConfig", func(t *testing.T) {
		// Search config is set but uses.search is disabled
		assert.NotNil(t, ast.Search, "search config should be set")
		assert.NotNil(t, ast.Search.Web, "web search config should be set")
	})

	t.Run("ShouldHaveDisabledUses", func(t *testing.T) {
		assert.NotNil(t, ast.Uses, "uses config should be set")
		assert.Equal(t, "disabled", ast.Uses.Search, "uses.search should be disabled")
	})

	t.Run("StreamShouldNotExecuteSearch", func(t *testing.T) {
		// Get agent via assistant.Get (required for Stream)
		agent, err := assistant.Get("tests.search-auto-disabled")
		require.NoError(t, err)
		require.NotNil(t, agent)

		// Create context
		ctx := newSearchAutoDisabledTestContext("test-search-auto-disabled", "tests.search-auto-disabled")

		// Create messages
		messages := []context.Message{
			{
				Role:    "user",
				Content: "Hello, how are you?",
			},
		}

		// Execute stream - should NOT trigger search because uses.search is "disabled"
		response, err := agent.Stream(ctx, messages)
		require.NoError(t, err)
		require.NotNil(t, response)

		assert.NotNil(t, response.Completion, "should have completion")
		t.Logf("✓ Stream executed without search (disabled)")
	})
}
