package keyword_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/assistant"
	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/search/nlp/keyword"
	searchTypes "github.com/yaoapp/yao/agent/search/types"
	"github.com/yaoapp/yao/agent/testutils"
	oauthTypes "github.com/yaoapp/yao/openapi/oauth/types"
)

func TestAgentProviderWithAssistantConfig(t *testing.T) {
	// Skip if running short tests
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Initialize test environment
	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Load the keyword-agent assistant that will provide keywords
	ast, err := assistant.Get("tests.keyword-agent")
	require.NoError(t, err)
	require.NotNil(t, ast)

	// Create test context
	ctx := newTestContext(t)

	// Create extractor with agent mode
	extractor := keyword.NewExtractor("tests.keyword-agent", &searchTypes.KeywordConfig{
		MaxKeywords: 5,
		Language:    "auto",
	})

	// Test extraction
	content := "Machine learning and deep learning are subfields of artificial intelligence"
	keywords, err := extractor.Extract(ctx, content, nil)
	require.NoError(t, err)
	assert.NotEmpty(t, keywords, "Agent should return keywords")
	assert.LessOrEqual(t, len(keywords), 5, "Should respect max_keywords")

	// Verify keywords are relevant
	t.Logf("Extracted keywords: %v", keywords)
}

func TestAgentProviderWithCustomOptions(t *testing.T) {
	// Skip if running short tests
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Initialize test environment
	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Create test context
	ctx := newTestContext(t)

	// Create extractor with agent mode
	extractor := keyword.NewExtractor("tests.keyword-agent", &searchTypes.KeywordConfig{
		MaxKeywords: 10,
	})

	// Test with runtime options override
	content := "Python programming language for data science and web development"
	keywords, err := extractor.Extract(ctx, content, &searchTypes.KeywordOptions{
		MaxKeywords: 3, // Override to 3
	})
	require.NoError(t, err)
	assert.NotEmpty(t, keywords)
	assert.LessOrEqual(t, len(keywords), 3, "Should respect runtime max_keywords override")

	t.Logf("Extracted keywords (max 3): %v", keywords)
}

func TestAgentProviderWithoutContext(t *testing.T) {
	// Test that agent mode requires context
	extractor := keyword.NewExtractor("tests.keyword-agent", nil)

	_, err := extractor.Extract(nil, "test content", nil)
	assert.Error(t, err, "Agent mode should require context")
	assert.Contains(t, err.Error(), "context is required")
}

func TestAgentProviderAgentNotFound(t *testing.T) {
	// Skip if running short tests
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Initialize test environment
	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Create test context
	ctx := newTestContext(t)

	// Create extractor with non-existent agent
	extractor := keyword.NewExtractor("non-existent-agent", nil)

	_, err := extractor.Extract(ctx, "test content", nil)
	assert.Error(t, err, "Should error for non-existent agent")
	assert.Contains(t, err.Error(), "failed to get agent")
}

// newTestContext creates a test context with required fields
func newTestContext(t *testing.T) *context.Context {
	t.Helper()
	authorized := &oauthTypes.AuthorizedInfo{
		UserID: "test-user",
	}
	chatID := "test-chat-keyword"
	ctx := context.New(t.Context(), authorized, chatID)
	return ctx
}
