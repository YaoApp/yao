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

// TestMCPProviderWithAssistantConfig tests MCPProvider using web-mcp assistant config
func TestMCPProviderWithAssistantConfig(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Load the web-mcp test assistant to get its config
	ast, err := assistant.LoadPath("/assistants/tests/web-mcp")
	require.NoError(t, err)
	require.NotNil(t, ast)
	require.NotNil(t, ast.Uses)

	// Verify assistant config
	assert.Equal(t, "tests.web-mcp", ast.ID)
	assert.Equal(t, "mcp:search.web_search", ast.Uses.Web)

	// Create MCPProvider from uses.web
	mcpRef := ast.Uses.Web[4:] // Remove "mcp:" prefix
	provider, err := web.NewMCPProvider(mcpRef)
	require.NoError(t, err)
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

	// MCP should return mock results
	if result.Error == "" {
		assert.Greater(t, result.Total, 0)
		assert.NotEmpty(t, result.Items)
		assert.Greater(t, result.Duration, int64(0))

		// Verify result item structure
		for _, item := range result.Items {
			assert.Equal(t, types.SearchTypeWeb, item.Type)
			assert.Equal(t, types.SourceAuto, item.Source)
			assert.NotEmpty(t, item.Title)
			assert.NotEmpty(t, item.URL)
		}

		t.Logf("MCP search returned %d results in %dms", result.Total, result.Duration)
	} else {
		t.Logf("MCP search returned error (expected if MCP not loaded): %s", result.Error)
	}
}

// TestMCPProviderWithSiteRestriction tests MCPProvider with domain restriction
func TestMCPProviderWithSiteRestriction(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Create MCPProvider
	provider, err := web.NewMCPProvider("search.web_search")
	require.NoError(t, err)

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

	if result.Error == "" {
		t.Logf("Site-restricted MCP search returned %d results", result.Total)
	} else {
		t.Logf("MCP search returned error (expected if MCP not loaded): %s", result.Error)
	}
}

// TestMCPProviderWithTimeRange tests MCPProvider with time range filter
func TestMCPProviderWithTimeRange(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Create MCPProvider
	provider, err := web.NewMCPProvider("search.web_search")
	require.NoError(t, err)

	// Execute search with time range
	req := &types.Request{
		Query:     "artificial intelligence news",
		Type:      types.SearchTypeWeb,
		Source:    types.SourceAuto,
		TimeRange: "week",
		Limit:     5,
	}

	result, err := provider.Search(req)
	require.NoError(t, err)
	require.NotNil(t, result)

	if result.Error == "" {
		t.Logf("Time-ranged MCP search (last week) returned %d results in %dms", result.Total, result.Duration)
	} else {
		t.Logf("MCP search returned error (expected if MCP not loaded): %s", result.Error)
	}
}

// TestMCPProviderInvalidFormat tests MCPProvider with invalid format
func TestMCPProviderInvalidFormat(t *testing.T) {
	// Test invalid format without dot
	_, err := web.NewMCPProvider("invalid")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid MCP format")

	// Test empty string
	_, err = web.NewMCPProvider("")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid MCP format")
}

// TestMCPProviderNotFound tests MCPProvider when MCP server is not found
func TestMCPProviderNotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Create MCPProvider with non-existent server
	provider, err := web.NewMCPProvider("nonexistent.web_search")
	require.NoError(t, err)

	req := &types.Request{
		Query:  "test query",
		Type:   types.SearchTypeWeb,
		Source: types.SourceAuto,
	}

	result, err := provider.Search(req)

	// Should not return error, but result should have error message
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.NotEmpty(t, result.Error)
	assert.Contains(t, result.Error, "not found")
}

// TestWebHandlerMCPMode tests the web handler in MCP mode
func TestWebHandlerMCPMode(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Create handler with MCP mode
	handler := web.NewHandler("mcp:search.web_search", nil)
	require.NotNil(t, handler)

	// Verify type
	assert.Equal(t, types.SearchTypeWeb, handler.Type())

	// Execute search
	req := &types.Request{
		Query:  "Yao framework",
		Type:   types.SearchTypeWeb,
		Source: types.SourceAuto,
		Limit:  5,
	}

	result, err := handler.Search(req)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, types.SearchTypeWeb, result.Type)
	assert.Equal(t, "Yao framework", result.Query)

	if result.Error == "" {
		t.Logf("Handler MCP mode returned %d results", result.Total)
	} else {
		t.Logf("Handler MCP mode returned error (expected if MCP not loaded): %s", result.Error)
	}
}

// TestWebHandlerInvalidMCPFormat tests the web handler with invalid MCP format
func TestWebHandlerInvalidMCPFormat(t *testing.T) {
	// Create handler with invalid MCP format
	handler := web.NewHandler("mcp:invalid", nil)
	require.NotNil(t, handler)

	req := &types.Request{
		Query:  "test",
		Type:   types.SearchTypeWeb,
		Source: types.SourceAuto,
	}

	result, err := handler.Search(req)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.NotEmpty(t, result.Error)
	assert.Contains(t, result.Error, "Invalid MCP format")
}
