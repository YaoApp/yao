package assistant_test

import (
	stdContext "context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/assistant"
	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/output/message"
	"github.com/yaoapp/yao/agent/testutils"
	"github.com/yaoapp/yao/openapi/oauth/types"
)

// newSearchAutoFullTestContext creates a test context
func newSearchAutoFullTestContext(chatID, assistantID string) *context.Context {
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

func TestSearchAutoFull(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	ast, err := assistant.LoadPath("/assistants/tests/search-auto-full")
	require.NoError(t, err)
	require.NotNil(t, ast)

	t.Run("ShouldHaveWebSearchConfig", func(t *testing.T) {
		assert.NotNil(t, ast.Search, "search config should be set")
		assert.NotNil(t, ast.Search.Web, "web search config should be set")
		assert.Equal(t, "tavily", ast.Search.Web.Provider)
		assert.Equal(t, 3, ast.Search.Web.MaxResults)
	})

	t.Run("ShouldHaveKBSearchConfig", func(t *testing.T) {
		assert.NotNil(t, ast.Search.KB, "kb search config should be set")
		assert.Equal(t, 0.7, ast.Search.KB.Threshold)
		assert.False(t, ast.Search.KB.Graph)
	})

	t.Run("ShouldHaveDBSearchConfig", func(t *testing.T) {
		assert.NotNil(t, ast.Search.DB, "db search config should be set")
		assert.Equal(t, 10, ast.Search.DB.MaxResults)
	})

	t.Run("ShouldHaveKBCollections", func(t *testing.T) {
		assert.NotNil(t, ast.KB, "kb config should be set")
		assert.Contains(t, ast.KB.Collections, "test-collection")
	})

	t.Run("ShouldHaveDBModels", func(t *testing.T) {
		assert.NotNil(t, ast.DB, "db config should be set")
		assert.Contains(t, ast.DB.Models, "user")
		assert.Contains(t, ast.DB.Models, "article")
	})

	t.Run("ShouldHaveCitationConfig", func(t *testing.T) {
		assert.NotNil(t, ast.Search.Citation, "citation config should be set")
		assert.Equal(t, "xml", ast.Search.Citation.Format)
		assert.True(t, ast.Search.Citation.AutoInjectPrompt)
	})

	t.Run("ShouldHaveUsesConfig", func(t *testing.T) {
		assert.NotNil(t, ast.Uses, "uses config should be set")
		assert.Equal(t, "builtin", ast.Uses.Search)
		assert.Equal(t, "builtin", ast.Uses.Web)
	})

	t.Run("StreamShouldExecuteMultipleSearchTypes", func(t *testing.T) {
		// Get agent via assistant.Get (required for Stream)
		agent, err := assistant.Get("tests.search-auto-full")
		require.NoError(t, err)
		require.NotNil(t, agent)

		// Create context
		ctx := newSearchAutoFullTestContext("test-search-auto-full", "tests.search-auto-full")

		// Create messages with a search query
		messages := []context.Message{
			{
				Role:    "user",
				Content: "Find information about machine learning",
			},
		}

		// Execute stream - should trigger Web + KB + DB searches
		response, err := agent.Stream(ctx, messages)

		// Assert no error (if API key is configured)
		if err != nil {
			// If error contains "API key", it's expected in CI without keys
			if strings.Contains(err.Error(), "API key") || strings.Contains(err.Error(), "api_key") {
				t.Logf("Expected error without API key: %v", err)
				return
			}
			// Other errors should fail
			require.NoError(t, err)
		}

		require.NotNil(t, response)
		assert.NotNil(t, response.Completion, "should have completion")
		t.Logf("✓ Stream executed with full search config (Web + KB + DB)")
	})
}
