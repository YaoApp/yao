package web_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/assistant"
	"github.com/yaoapp/yao/agent/search/handlers/web"
	"github.com/yaoapp/yao/agent/search/types"
	"github.com/yaoapp/yao/agent/testutils"
)

// TestTavilyProviderWithAssistantConfig tests TavilyProvider using web-tavily assistant config
func TestTavilyProviderWithAssistantConfig(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Load the web-tavily test assistant to get its config
	ast, err := assistant.LoadPath("/assistants/tests/web-tavily")
	require.NoError(t, err)
	require.NotNil(t, ast)
	require.NotNil(t, ast.Search)
	require.NotNil(t, ast.Search.Web)

	// Verify assistant config
	assert.Equal(t, "tests.web-tavily", ast.ID)
	assert.Equal(t, "tavily", ast.Search.Web.Provider)
	assert.Equal(t, "$ENV.TAVILY_API_KEY", ast.Search.Web.APIKeyEnv)
	assert.Equal(t, 10, ast.Search.Web.MaxResults)

	// Create TavilyProvider with assistant's web config
	provider := web.NewTavilyProvider(ast.Search.Web)
	require.NotNil(t, provider)

	// Execute search
	req := &types.Request{
		Query:  "Yao App Engine",
		Type:   types.SearchTypeWeb,
		Source: types.SourceAuto,
		Limit:  5,
	}

	result, err := provider.Search(req)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Verify result structure
	assert.Equal(t, types.SearchTypeWeb, result.Type)
	assert.Equal(t, "Yao App Engine", result.Query)
	assert.Equal(t, types.SourceAuto, result.Source)

	// API key must be valid - search should succeed
	require.Empty(t, result.Error, "Search should succeed with valid API key, got error: %s", result.Error)

	// Verify we got results
	assert.Greater(t, result.Total, 0)
	assert.NotEmpty(t, result.Items)
	assert.Greater(t, result.Duration, int64(0))

	// Verify result item structure
	for _, item := range result.Items {
		assert.Equal(t, types.SearchTypeWeb, item.Type)
		assert.Equal(t, types.SourceAuto, item.Source)
		assert.NotEmpty(t, item.Title)
		assert.NotEmpty(t, item.URL)
		// Content may be empty for some results
	}

	t.Logf("Search returned %d results in %dms", result.Total, result.Duration)
}

// TestTavilyProviderWithSiteRestriction tests TavilyProvider with domain restriction
func TestTavilyProviderWithSiteRestriction(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Load the web-tavily test assistant
	ast, err := assistant.LoadPath("/assistants/tests/web-tavily")
	require.NoError(t, err)
	require.NotNil(t, ast.Search)
	require.NotNil(t, ast.Search.Web)

	// Create TavilyProvider
	provider := web.NewTavilyProvider(ast.Search.Web)

	// Execute search with site restriction
	req := &types.Request{
		Query:  "documentation",
		Type:   types.SearchTypeWeb,
		Source: types.SourceHook,
		Sites:  []string{"github.com"},
		Limit:  3,
	}

	result, err := provider.Search(req)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, types.SearchTypeWeb, result.Type)
	assert.Equal(t, types.SourceHook, result.Source)

	// API key must be valid - search should succeed
	require.Empty(t, result.Error, "Search should succeed with valid API key, got error: %s", result.Error)
	require.NotEmpty(t, result.Items, "Search should return results")

	// All results should be from github.com
	for _, item := range result.Items {
		assert.Contains(t, item.URL, "github.com", "Result URL should be from github.com")
	}
	t.Logf("Site-restricted search returned %d results from github.com", result.Total)
}

// TestTavilyProviderWithoutAPIKey tests graceful degradation when API key is missing
func TestTavilyProviderWithoutAPIKey(t *testing.T) {
	// Create provider with nil config (no API key)
	provider := web.NewTavilyProvider(nil)
	require.NotNil(t, provider)

	req := &types.Request{
		Query:  "test query",
		Type:   types.SearchTypeWeb,
		Source: types.SourceAuto,
	}

	result, err := provider.Search(req)

	// Should not return error, but result should have error message
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, types.SearchTypeWeb, result.Type)
	assert.Equal(t, "test query", result.Query)
	assert.NotEmpty(t, result.Error)
	assert.Contains(t, result.Error, "API key")
	assert.Empty(t, result.Items)
	assert.Equal(t, 0, result.Total)
}

// TestTavilyProviderWithEmptyConfig tests provider with empty config
func TestTavilyProviderWithEmptyConfig(t *testing.T) {
	// Create provider with empty config
	cfg := &types.WebConfig{}
	provider := web.NewTavilyProvider(cfg)
	require.NotNil(t, provider)

	req := &types.Request{
		Query:  "test query",
		Type:   types.SearchTypeWeb,
		Source: types.SourceUser,
	}

	result, err := provider.Search(req)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.NotEmpty(t, result.Error)
	assert.Contains(t, result.Error, "API key")
}

// TestTavilyProviderMaxResults tests that max_results from config is respected
func TestTavilyProviderMaxResults(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Load the web-tavily test assistant
	ast, err := assistant.LoadPath("/assistants/tests/web-tavily")
	require.NoError(t, err)
	require.NotNil(t, ast.Search)
	require.NotNil(t, ast.Search.Web)

	// Create TavilyProvider
	provider := web.NewTavilyProvider(ast.Search.Web)

	// Execute search without limit (should use config's max_results)
	req := &types.Request{
		Query:  "artificial intelligence",
		Type:   types.SearchTypeWeb,
		Source: types.SourceAuto,
		// No Limit set, should use config's max_results (10)
	}

	result, err := provider.Search(req)
	require.NoError(t, err)
	require.NotNil(t, result)

	// API key must be valid - search should succeed
	require.Empty(t, result.Error, "Search should succeed with valid API key, got error: %s", result.Error)

	// Should respect max_results from config
	assert.LessOrEqual(t, result.Total, ast.Search.Web.MaxResults)
	t.Logf("Search without limit returned %d results (max: %d)", result.Total, ast.Search.Web.MaxResults)

	// Execute search with explicit limit
	req2 := &types.Request{
		Query:  "artificial intelligence",
		Type:   types.SearchTypeWeb,
		Source: types.SourceAuto,
		Limit:  3, // Override config's max_results
	}

	result2, err := provider.Search(req2)
	require.NoError(t, err)
	require.NotNil(t, result2)

	// API key must be valid - search should succeed
	require.Empty(t, result2.Error, "Search should succeed with valid API key, got error: %s", result2.Error)

	// Should respect request's limit
	assert.LessOrEqual(t, result2.Total, 3)
	t.Logf("Search with limit=3 returned %d results", result2.Total)
}
