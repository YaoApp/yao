package search_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/assistant"
	"github.com/yaoapp/yao/agent/search"
	"github.com/yaoapp/yao/agent/search/types"
	"github.com/yaoapp/yao/agent/testutils"
)

// =============================================================================
// Web Search Integration Tests - Single Search
// =============================================================================

// TestWebSearch_Tavily tests web search using Tavily provider via assistant config
// Skip: requires external API key (Tavily)
func TestWebSearch_Tavily(t *testing.T) {
	t.Skip("Skipping: requires external API key (Tavily)")
	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Load the web-tavily test assistant
	ast, err := assistant.LoadPath("/assistants/tests/web-tavily")
	require.NoError(t, err)
	require.NotNil(t, ast.Search)
	require.NotNil(t, ast.Search.Web)

	// Verify assistant config
	assert.Equal(t, "tavily", ast.Search.Web.Provider)

	// Create Searcher with assistant's config
	uses := &search.Uses{Web: "builtin"}
	s := search.New(ast.Search, uses)

	// Execute search
	req := &types.Request{
		Type:   types.SearchTypeWeb,
		Query:  "Yao App Engine low-code platform",
		Source: types.SourceAuto,
		Limit:  5,
	}

	result, err := s.Search(nil, req)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Empty(t, result.Error, "Search should succeed, got error: %s", result.Error)

	// Verify results
	assert.NotEmpty(t, result.Items, "Should have search results")
	for _, item := range result.Items {
		assert.NotEmpty(t, item.CitationID, "Each item should have citation ID")
		assert.NotEmpty(t, item.Content, "Each item should have content")
		t.Logf("  [%s] %s - %s", item.CitationID, item.Title, item.URL)
	}
	t.Logf("Tavily search returned %d results", len(result.Items))
}

// TestWebSearch_Serper tests web search using Serper provider via assistant config
// Skip: requires external API key (Serper)
func TestWebSearch_Serper(t *testing.T) {
	t.Skip("Skipping: requires external API key (Serper)")
	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Load the web-serper test assistant
	ast, err := assistant.LoadPath("/assistants/tests/web-serper")
	require.NoError(t, err)
	require.NotNil(t, ast.Search)
	require.NotNil(t, ast.Search.Web)

	// Verify assistant config
	assert.Equal(t, "serper", ast.Search.Web.Provider)

	// Create Searcher with assistant's config
	uses := &search.Uses{Web: "builtin"}
	s := search.New(ast.Search, uses)

	// Execute search
	req := &types.Request{
		Type:   types.SearchTypeWeb,
		Query:  "Go programming language concurrency",
		Source: types.SourceAuto,
		Limit:  5,
	}

	result, err := s.Search(nil, req)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Empty(t, result.Error, "Search should succeed, got error: %s", result.Error)

	// Verify results
	assert.NotEmpty(t, result.Items, "Should have search results")
	for _, item := range result.Items {
		assert.NotEmpty(t, item.CitationID, "Each item should have citation ID")
		t.Logf("  [%s] %s - %s", item.CitationID, item.Title, item.URL)
	}
	t.Logf("Serper search returned %d results", len(result.Items))
}

// TestWebSearch_SerpAPI tests web search using SerpAPI provider via assistant config
// Skip: requires external API key (SerpAPI)
func TestWebSearch_SerpAPI(t *testing.T) {
	t.Skip("Skipping: requires external API key (SerpAPI)")
	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Load the web-serpapi test assistant
	ast, err := assistant.LoadPath("/assistants/tests/web-serpapi")
	require.NoError(t, err)
	require.NotNil(t, ast.Search)
	require.NotNil(t, ast.Search.Web)

	// Verify assistant config
	assert.Equal(t, "serpapi", ast.Search.Web.Provider)

	// Create Searcher with assistant's config
	uses := &search.Uses{Web: "builtin"}
	s := search.New(ast.Search, uses)

	// Execute search
	req := &types.Request{
		Type:   types.SearchTypeWeb,
		Query:  "Kubernetes container orchestration",
		Source: types.SourceAuto,
		Limit:  5,
	}

	result, err := s.Search(nil, req)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Empty(t, result.Error, "Search should succeed, got error: %s", result.Error)

	// Verify results
	assert.NotEmpty(t, result.Items, "Should have search results")
	for _, item := range result.Items {
		assert.NotEmpty(t, item.CitationID, "Each item should have citation ID")
		t.Logf("  [%s] %s - %s", item.CitationID, item.Title, item.URL)
	}
	t.Logf("SerpAPI search returned %d results", len(result.Items))
}

// =============================================================================
// Web Search Integration Tests - Parallel Search
// =============================================================================

// TestWebSearch_All tests parallel web search with All() - like Promise.all
// Skip: requires external API key (Serper)
func TestWebSearch_All(t *testing.T) {
	t.Skip("Skipping: requires external API key (Serper)")
	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Load the web-serper test assistant
	ast, err := assistant.LoadPath("/assistants/tests/web-serper")
	require.NoError(t, err)
	require.NotNil(t, ast.Search)

	// Create Searcher
	uses := &search.Uses{Web: "builtin"}
	s := search.New(ast.Search, uses)

	// Multiple queries
	reqs := []*types.Request{
		{Type: types.SearchTypeWeb, Query: "artificial intelligence", Source: types.SourceAuto, Limit: 3},
		{Type: types.SearchTypeWeb, Query: "machine learning", Source: types.SourceAuto, Limit: 3},
		{Type: types.SearchTypeWeb, Query: "deep learning", Source: types.SourceAuto, Limit: 3},
	}

	// Execute parallel search with All() - waits for all searches to complete
	results, err := s.All(nil, reqs)
	require.NoError(t, err)
	require.Len(t, results, 3, "Should have 3 results")

	// Verify all results
	for i, result := range results {
		require.NotNil(t, result, "Result %d should not be nil", i)
		if result.Error == "" {
			assert.NotEmpty(t, result.Items, "Result %d should have items", i)
			t.Logf("Query '%s': %d results", reqs[i].Query, len(result.Items))
		} else {
			t.Logf("Query '%s': error - %s", reqs[i].Query, result.Error)
		}
	}
}

// TestWebSearch_Any tests parallel web search with Any() - like Promise.any
// Skip: requires external API key (Serper)
func TestWebSearch_Any(t *testing.T) {
	t.Skip("Skipping: requires external API key (Serper)")
	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Load the web-serper test assistant
	ast, err := assistant.LoadPath("/assistants/tests/web-serper")
	require.NoError(t, err)
	require.NotNil(t, ast.Search)

	// Create Searcher
	uses := &search.Uses{Web: "builtin"}
	s := search.New(ast.Search, uses)

	// Multiple queries
	reqs := []*types.Request{
		{Type: types.SearchTypeWeb, Query: "golang channels", Source: types.SourceAuto, Limit: 3},
		{Type: types.SearchTypeWeb, Query: "rust ownership", Source: types.SourceAuto, Limit: 3},
		{Type: types.SearchTypeWeb, Query: "python asyncio", Source: types.SourceAuto, Limit: 3},
	}

	// Execute parallel search with Any() - returns when first search succeeds
	results, err := s.Any(nil, reqs)
	require.NoError(t, err)

	// Any() returns as soon as any search succeeds
	hasSuccess := false
	for _, result := range results {
		if result != nil && len(result.Items) > 0 && result.Error == "" {
			hasSuccess = true
			t.Logf("First success: '%s' with %d results", result.Query, len(result.Items))
			break
		}
	}
	assert.True(t, hasSuccess, "At least one search should succeed")
}

// TestWebSearch_Race tests parallel web search with Race() - like Promise.race
// Skip: requires external API key (Serper)
func TestWebSearch_Race(t *testing.T) {
	t.Skip("Skipping: requires external API key (Serper)")
	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Load the web-serper test assistant
	ast, err := assistant.LoadPath("/assistants/tests/web-serper")
	require.NoError(t, err)
	require.NotNil(t, ast.Search)

	// Create Searcher
	uses := &search.Uses{Web: "builtin"}
	s := search.New(ast.Search, uses)

	// Multiple queries
	reqs := []*types.Request{
		{Type: types.SearchTypeWeb, Query: "docker containers", Source: types.SourceAuto, Limit: 3},
		{Type: types.SearchTypeWeb, Query: "kubernetes pods", Source: types.SourceAuto, Limit: 3},
	}

	// Execute parallel search with Race() - returns when first search completes
	results, err := s.Race(nil, reqs)
	require.NoError(t, err)

	// Race() returns immediately when first result arrives
	hasResult := false
	for _, result := range results {
		if result != nil {
			hasResult = true
			t.Logf("First to complete: '%s'", result.Query)
			break
		}
	}
	assert.True(t, hasResult, "Should have at least one result")
}

// =============================================================================
// Web Search - Citation and Reference Tests
// =============================================================================

// TestWebSearch_BuildReferences tests building references from web search results
// Skip: requires external API key (Serper)
func TestWebSearch_BuildReferences(t *testing.T) {
	t.Skip("Skipping: requires external API key (Serper)")
	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Load the web-serper test assistant
	ast, err := assistant.LoadPath("/assistants/tests/web-serper")
	require.NoError(t, err)
	require.NotNil(t, ast.Search)

	// Create Searcher with weights config
	uses := &search.Uses{Web: "builtin"}
	s := search.New(ast.Search, uses)

	// Execute search
	req := &types.Request{
		Type:   types.SearchTypeWeb,
		Query:  "OpenAI GPT-4",
		Source: types.SourceAuto,
		Limit:  5,
	}

	result, err := s.Search(nil, req)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Empty(t, result.Error, "Search should succeed")
	require.NotEmpty(t, result.Items, "Should have results")

	// Build references
	refs := s.BuildReferences([]*types.Result{result})
	assert.NotEmpty(t, refs, "Should have references")

	for _, ref := range refs {
		assert.NotEmpty(t, ref.ID, "Reference should have ID")
		assert.Equal(t, types.SearchTypeWeb, ref.Type, "Reference type should be web")
		assert.Equal(t, types.SourceAuto, ref.Source, "Reference source should be auto")
		t.Logf("  Ref: %s - %s (weight: %.2f)", ref.ID, ref.Title, ref.Weight)
	}
}

// =============================================================================
// Web Search - Error Handling Tests
// =============================================================================

// TestWebSearch_SiteRestriction tests web search with site restriction
// Skip: requires external API key (Serper)
func TestWebSearch_SiteRestriction(t *testing.T) {
	t.Skip("Skipping: requires external API key (Serper)")
	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Load the web-serper test assistant
	ast, err := assistant.LoadPath("/assistants/tests/web-serper")
	require.NoError(t, err)
	require.NotNil(t, ast.Search)

	// Create Searcher
	uses := &search.Uses{Web: "builtin"}
	s := search.New(ast.Search, uses)

	// Execute search with site restriction
	req := &types.Request{
		Type:   types.SearchTypeWeb,
		Query:  "yao-app-engine",
		Source: types.SourceAuto,
		Sites:  []string{"github.com"},
		Limit:  5,
	}

	result, err := s.Search(nil, req)
	require.NoError(t, err)
	require.NotNil(t, result)

	if result.Error == "" && len(result.Items) > 0 {
		// Log results
		for _, item := range result.Items {
			t.Logf("  %s - %s", item.Title, item.URL)
		}
	}
}
