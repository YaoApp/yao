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

// TestSerpAPIProviderWithAssistantConfig tests SerpAPIProvider using web-serpapi assistant config
func TestSerpAPIProviderWithAssistantConfig(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Load the web-serpapi test assistant to get its config
	ast, err := assistant.LoadPath("/assistants/tests/web-serpapi")
	require.NoError(t, err)
	require.NotNil(t, ast)
	require.NotNil(t, ast.Search)
	require.NotNil(t, ast.Search.Web)

	// Verify assistant config
	assert.Equal(t, "tests.web-serpapi", ast.ID)
	assert.Equal(t, "serpapi", ast.Search.Web.Provider)
	assert.Equal(t, "$ENV.SERPAPI_API_KEY", ast.Search.Web.APIKeyEnv)
	assert.Equal(t, 10, ast.Search.Web.MaxResults)

	// Create SerpAPIProvider with assistant's web config
	provider := web.NewSerpAPIProvider(ast.Search.Web)
	require.NotNil(t, provider)

	// Execute search
	req := &types.Request{
		Query:  "golang programming language",
		Type:   types.SearchTypeWeb,
		Source: types.SourceAuto,
		Limit:  5,
	}

	result, err := provider.Search(req)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Verify result structure
	assert.Equal(t, types.SearchTypeWeb, result.Type)
	assert.Equal(t, "golang programming language", result.Query)
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
		assert.Greater(t, item.Score, 0.0)
	}

	t.Logf("Search returned %d results in %dms", result.Total, result.Duration)
}

// TestSerpAPIProviderWithSiteRestriction tests SerpAPIProvider with domain restriction
func TestSerpAPIProviderWithSiteRestriction(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Load the web-serpapi test assistant
	ast, err := assistant.LoadPath("/assistants/tests/web-serpapi")
	require.NoError(t, err)
	require.NotNil(t, ast.Search)
	require.NotNil(t, ast.Search.Web)

	// Create SerpAPIProvider
	provider := web.NewSerpAPIProvider(ast.Search.Web)

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

// TestSerpAPIProviderWithMultipleSites tests SerpAPIProvider with multiple domain restrictions
func TestSerpAPIProviderWithMultipleSites(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Load the web-serpapi test assistant
	ast, err := assistant.LoadPath("/assistants/tests/web-serpapi")
	require.NoError(t, err)
	require.NotNil(t, ast.Search)
	require.NotNil(t, ast.Search.Web)

	// Create SerpAPIProvider
	provider := web.NewSerpAPIProvider(ast.Search.Web)

	// Execute search with multiple site restrictions
	req := &types.Request{
		Query:  "golang tutorial",
		Type:   types.SearchTypeWeb,
		Source: types.SourceAuto,
		Sites:  []string{"github.com", "golang.org"},
		Limit:  5,
	}

	result, err := provider.Search(req)
	require.NoError(t, err)
	require.NotNil(t, result)

	// API key must be valid - search should succeed
	require.Empty(t, result.Error, "Search should succeed with valid API key, got error: %s", result.Error)
	require.NotEmpty(t, result.Items, "Search should return results")

	// Results should be from either github.com or golang.org
	for _, item := range result.Items {
		isValidSite := false
		for _, site := range req.Sites {
			if containsSite(item.URL, site) {
				isValidSite = true
				break
			}
		}
		assert.True(t, isValidSite, "Result URL should be from github.com or golang.org: %s", item.URL)
	}
	t.Logf("Multi-site search returned %d results", result.Total)
}

// TestSerpAPIProviderWithTimeRange tests SerpAPIProvider with time range filter
func TestSerpAPIProviderWithTimeRange(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Load the web-serpapi test assistant
	ast, err := assistant.LoadPath("/assistants/tests/web-serpapi")
	require.NoError(t, err)
	require.NotNil(t, ast.Search)
	require.NotNil(t, ast.Search.Web)

	// Create SerpAPIProvider
	provider := web.NewSerpAPIProvider(ast.Search.Web)

	// Execute search with time range
	req := &types.Request{
		Query:     "artificial intelligence news",
		Type:      types.SearchTypeWeb,
		Source:    types.SourceAuto,
		TimeRange: "week", // Last week
		Limit:     5,
	}

	result, err := provider.Search(req)
	require.NoError(t, err)
	require.NotNil(t, result)

	// API key must be valid - search should succeed
	require.Empty(t, result.Error, "Search should succeed with valid API key, got error: %s", result.Error)
	t.Logf("Time-ranged search (last week) returned %d results in %dms", result.Total, result.Duration)
}

// TestSerpAPIProviderWithoutAPIKey tests graceful degradation when API key is missing
func TestSerpAPIProviderWithoutAPIKey(t *testing.T) {
	// Create provider with nil config (no API key)
	provider := web.NewSerpAPIProvider(nil)
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

// TestSerpAPIProviderWithEmptyConfig tests provider with empty config
func TestSerpAPIProviderWithEmptyConfig(t *testing.T) {
	// Create provider with empty config
	cfg := &types.WebConfig{}
	provider := web.NewSerpAPIProvider(cfg)
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

// TestSerpAPIProviderMaxResults tests that max_results from config is respected
func TestSerpAPIProviderMaxResults(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Load the web-serpapi test assistant
	ast, err := assistant.LoadPath("/assistants/tests/web-serpapi")
	require.NoError(t, err)
	require.NotNil(t, ast.Search)
	require.NotNil(t, ast.Search.Web)

	// Create SerpAPIProvider
	provider := web.NewSerpAPIProvider(ast.Search.Web)

	// Execute search without limit (should use config's max_results)
	req := &types.Request{
		Query:  "machine learning",
		Type:   types.SearchTypeWeb,
		Source: types.SourceAuto,
		// No Limit set, should use config's max_results (10)
	}

	result, err := provider.Search(req)
	require.NoError(t, err)
	require.NotNil(t, result)

	// API key must be valid - search should succeed
	require.Empty(t, result.Error, "Search should succeed with valid API key, got error: %s", result.Error)

	// Should respect max_results from config (+1 for possible answer box)
	assert.LessOrEqual(t, result.Total, ast.Search.Web.MaxResults+1)
	t.Logf("Search without limit returned %d results (max: %d)", result.Total, ast.Search.Web.MaxResults)

	// Execute search with explicit limit
	req2 := &types.Request{
		Query:  "machine learning",
		Type:   types.SearchTypeWeb,
		Source: types.SourceAuto,
		Limit:  3, // Override config's max_results
	}

	result2, err := provider.Search(req2)
	require.NoError(t, err)
	require.NotNil(t, result2)

	// API key must be valid - search should succeed
	require.Empty(t, result2.Error, "Search should succeed with valid API key, got error: %s", result2.Error)

	// Should respect request's limit (+1 for possible answer box)
	assert.LessOrEqual(t, result2.Total, 4)
	t.Logf("Search with limit=3 returned %d results", result2.Total)
}

// TestSerpAPIProviderWithBingEngine tests SerpAPIProvider with Bing search engine
func TestSerpAPIProviderWithBingEngine(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Load the web-serpapi test assistant to get base config
	ast, err := assistant.LoadPath("/assistants/tests/web-serpapi")
	require.NoError(t, err)
	require.NotNil(t, ast.Search)
	require.NotNil(t, ast.Search.Web)

	// Create config with Bing engine
	bingConfig := &types.WebConfig{
		Provider:   "serpapi",
		APIKeyEnv:  ast.Search.Web.APIKeyEnv,
		MaxResults: 5,
		Engine:     "bing",
	}

	// Create SerpAPIProvider with Bing engine
	provider := web.NewSerpAPIProvider(bingConfig)
	require.NotNil(t, provider)

	// Execute search
	req := &types.Request{
		Query:  "Golang programming",
		Type:   types.SearchTypeWeb,
		Source: types.SourceAuto,
		Limit:  5,
	}

	result, err := provider.Search(req)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Verify result structure
	assert.Equal(t, types.SearchTypeWeb, result.Type)
	assert.Equal(t, "Golang programming", result.Query)

	// API key must be valid - search should succeed
	require.Empty(t, result.Error, "Bing search should succeed with valid API key, got error: %s", result.Error)

	// Verify we got results
	assert.Greater(t, result.Total, 0)
	assert.NotEmpty(t, result.Items)

	t.Logf("Bing search returned %d results in %dms", result.Total, result.Duration)
}

// TestSerpAPIProviderEngineDefault tests that default engine is Google
func TestSerpAPIProviderEngineDefault(t *testing.T) {
	// Create provider with config that has no engine specified
	cfg := &types.WebConfig{
		Provider:   "serpapi",
		APIKeyEnv:  "SERPAPI_API_KEY",
		MaxResults: 10,
		// Engine not set - should default to "google"
	}

	provider := web.NewSerpAPIProvider(cfg)
	require.NotNil(t, provider)

	// We can't directly check the engine field since it's private,
	// but we verify the provider is created successfully
	// The actual engine usage is tested in integration tests
}

// containsSite checks if url contains the site domain
func containsSite(url, site string) bool {
	return len(url) >= len(site) && containsHelper(url, site)
}

// containsHelper is a helper function for string containment check
func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
