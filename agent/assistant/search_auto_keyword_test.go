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

// newKeywordTestContext creates a test context for keyword extraction tests
func newKeywordTestContext(chatID, assistantID string) *context.Context {
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

func TestSearchAutoKeyword(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	ast, err := assistant.LoadPath("/assistants/tests/search-auto-keyword")
	require.NoError(t, err)
	require.NotNil(t, ast)

	t.Run("ShouldHaveKeywordConfig", func(t *testing.T) {
		assert.NotNil(t, ast.Search, "search config should be set")
		assert.NotNil(t, ast.Search.Keyword, "keyword config should be set")
		assert.Equal(t, 5, ast.Search.Keyword.MaxKeywords)
		assert.Equal(t, "auto", ast.Search.Keyword.Language)
	})

	t.Run("ShouldHaveKeywordInUses", func(t *testing.T) {
		assert.NotNil(t, ast.Uses, "uses config should be set")
		assert.Equal(t, "builtin", ast.Uses.Keyword)
	})

	t.Run("StreamWithKeywordExtraction", func(t *testing.T) {
		// Get agent via assistant.Get (required for Stream)
		agent, err := assistant.Get("tests.search-auto-keyword")
		require.NoError(t, err)
		require.NotNil(t, agent)

		// Create context
		ctx := newKeywordTestContext("test-search-keyword", "tests.search-auto-keyword")

		// Create messages with a verbose query that should benefit from keyword extraction
		messages := []context.Message{
			{
				Role:    "user",
				Content: "I want to find the best wireless headphones under 100 dollars for programming and music",
			},
		}

		// Execute stream without Skip.Keyword (keyword extraction should happen)
		response, err := agent.Stream(ctx, messages)

		// Assert no error (if API key is configured)
		if err != nil {
			if strings.Contains(err.Error(), "API key") || strings.Contains(err.Error(), "api_key") {
				t.Logf("Expected error without API key: %v", err)
				return
			}
			require.NoError(t, err)
		}

		require.NotNil(t, response)
		assert.NotNil(t, response.Completion, "should have completion")
		t.Logf("✓ Stream with keyword extraction executed successfully")
	})

	t.Run("StreamWithSkipKeyword", func(t *testing.T) {
		// Get agent via assistant.Get (required for Stream)
		agent, err := assistant.Get("tests.search-auto-keyword")
		require.NoError(t, err)
		require.NotNil(t, agent)

		// Create context
		ctx := newKeywordTestContext("test-search-skip-keyword", "tests.search-auto-keyword")

		// Create messages
		messages := []context.Message{
			{
				Role:    "user",
				Content: "I want to find the best wireless headphones under 100 dollars",
			},
		}

		// Execute stream with Skip.Keyword = true (keyword extraction should be skipped)
		opts := &context.Options{
			Skip: &context.Skip{
				Keyword: true,
			},
		}
		response, err := agent.Stream(ctx, messages, opts)

		// Assert no error (if API key is configured)
		if err != nil {
			if strings.Contains(err.Error(), "API key") || strings.Contains(err.Error(), "api_key") {
				t.Logf("Expected error without API key: %v", err)
				return
			}
			require.NoError(t, err)
		}

		require.NotNil(t, response)
		assert.NotNil(t, response.Completion, "should have completion")
		t.Logf("✓ Stream with Skip.Keyword executed successfully")
	})
}

func TestSearchAutoKeywordNotConfigured(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Use the search-auto-web assistant which does NOT have uses.keyword configured
	ast, err := assistant.LoadPath("/assistants/tests/search-auto-web")
	require.NoError(t, err)
	require.NotNil(t, ast)

	t.Run("ShouldNotHaveKeywordInUses", func(t *testing.T) {
		// uses.keyword should be empty (not configured)
		if ast.Uses != nil {
			assert.Empty(t, ast.Uses.Keyword, "uses.keyword should be empty")
		}
	})

	t.Run("StreamShouldSkipKeywordExtraction", func(t *testing.T) {
		// Get agent via assistant.Get (required for Stream)
		agent, err := assistant.Get("tests.search-auto-web")
		require.NoError(t, err)
		require.NotNil(t, agent)

		// Create context
		ctx := newKeywordTestContext("test-no-keyword", "tests.search-auto-web")

		// Create messages
		messages := []context.Message{
			{
				Role:    "user",
				Content: "What is the latest news about AI?",
			},
		}

		// Execute stream - keyword extraction should NOT happen because uses.keyword is not set
		response, err := agent.Stream(ctx, messages)

		// Assert no error (if API key is configured)
		if err != nil {
			if strings.Contains(err.Error(), "API key") || strings.Contains(err.Error(), "api_key") {
				t.Logf("Expected error without API key: %v", err)
				return
			}
			require.NoError(t, err)
		}

		require.NotNil(t, response)
		assert.NotNil(t, response.Completion, "should have completion")
		t.Logf("✓ Stream without keyword config executed successfully")
	})
}
