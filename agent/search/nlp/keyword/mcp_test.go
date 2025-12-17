package keyword_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/search/nlp/keyword"
	searchTypes "github.com/yaoapp/yao/agent/search/types"
	"github.com/yaoapp/yao/agent/testutils"
	oauthTypes "github.com/yaoapp/yao/openapi/oauth/types"
)

func TestMCPProviderWithAssistantConfig(t *testing.T) {
	// Skip if running short tests
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Initialize test environment
	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Create test context
	ctx := newMCPTestContext(t)

	// Create extractor with MCP mode
	extractor := keyword.NewExtractor("mcp:search.extract_keywords", &searchTypes.KeywordConfig{
		MaxKeywords: 5,
		Language:    "auto",
	})

	// Test extraction
	content := "Machine learning and deep learning are subfields of artificial intelligence"
	keywords, err := extractor.Extract(ctx, content, nil)
	require.NoError(t, err)
	assert.NotEmpty(t, keywords, "MCP should return keywords")
	assert.LessOrEqual(t, len(keywords), 5, "Should respect max_keywords")

	t.Logf("Extracted keywords via MCP: %v", keywords)
}

func TestMCPProviderWithCustomOptions(t *testing.T) {
	// Skip if running short tests
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Initialize test environment
	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Create test context
	ctx := newMCPTestContext(t)

	// Create extractor with MCP mode
	extractor := keyword.NewExtractor("mcp:search.extract_keywords", &searchTypes.KeywordConfig{
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

	t.Logf("Extracted keywords via MCP (max 3): %v", keywords)
}

func TestMCPProviderInvalidFormat(t *testing.T) {
	// Test invalid MCP format fallback to system agent (requires context)
	extractor := keyword.NewExtractor("mcp:invalid", nil)

	// Should fallback to system agent which requires context
	_, err := extractor.Extract(nil, "test content for keyword extraction", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context is required")
}

func TestMCPProviderServerNotFound(t *testing.T) {
	// Skip if running short tests
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Initialize test environment
	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Create test context
	ctx := newMCPTestContext(t)

	// Create extractor with non-existent MCP server
	extractor := keyword.NewExtractor("mcp:nonexistent.extract_keywords", &searchTypes.KeywordConfig{})

	_, err := extractor.Extract(ctx, "test content", nil)
	assert.Error(t, err, "Should error for non-existent MCP server")
	assert.Contains(t, err.Error(), "not found")
}

func TestMCPProviderToolNotFound(t *testing.T) {
	// Skip if running short tests
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Initialize test environment
	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Create test context
	ctx := newMCPTestContext(t)

	// Create extractor with non-existent tool
	extractor := keyword.NewExtractor("mcp:search.nonexistent_tool", &searchTypes.KeywordConfig{})

	_, err := extractor.Extract(ctx, "test content", nil)
	assert.Error(t, err, "Should error for non-existent MCP tool")
}

func TestMCPProviderEmptyContent(t *testing.T) {
	// Skip if running short tests
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Initialize test environment
	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Create test context
	ctx := newMCPTestContext(t)

	// Create extractor with MCP mode
	extractor := keyword.NewExtractor("mcp:search.extract_keywords", nil)

	// Test with empty content - MCP tool should return error
	_, err := extractor.Extract(ctx, "", nil)
	assert.Error(t, err, "Should error for empty content")
}

// newMCPTestContext creates a test context for MCP tests
func newMCPTestContext(t *testing.T) *context.Context {
	t.Helper()
	authorized := &oauthTypes.AuthorizedInfo{
		UserID: "test-user",
	}
	chatID := "test-chat-mcp-keyword"
	ctx := context.New(t.Context(), authorized, chatID)
	return ctx
}
