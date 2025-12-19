package web_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/assistant"
	agentContext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/search/handlers/web"
	"github.com/yaoapp/yao/agent/search/types"
	"github.com/yaoapp/yao/agent/testutils"
	oauthTypes "github.com/yaoapp/yao/openapi/oauth/types"
)

// TestAgentProviderWithAssistantConfig tests AgentProvider using web-agent-caller assistant config
func TestAgentProviderWithAssistantConfig(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Load the web-agent-caller test assistant to get its config
	ast, err := assistant.LoadPath("/assistants/tests/web-agent-caller")
	require.NoError(t, err)
	require.NotNil(t, ast)
	require.NotNil(t, ast.Uses)

	// Verify assistant config
	assert.Equal(t, "tests.web-agent-caller", ast.ID)
	assert.Equal(t, "tests.web-agent", ast.Uses.Web)

	// Create AgentProvider from uses.web
	provider := web.NewAgentProvider(ast.Uses.Web)
	require.NotNil(t, provider)

	// Create a mock context
	ctx := createTestContext(t)

	// Execute search
	req := &types.Request{
		Query:  "Yao App Engine",
		Type:   types.SearchTypeWeb,
		Source: types.SourceAuto,
		Limit:  5,
	}

	result, err := provider.Search(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Verify result structure
	assert.Equal(t, types.SearchTypeWeb, result.Type)
	assert.Equal(t, "Yao App Engine", result.Query)
	assert.Equal(t, types.SourceAuto, result.Source)

	// Agent should return mock results from Next hook
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

		t.Logf("Agent search returned %d results in %dms", result.Total, result.Duration)
	} else {
		t.Logf("Agent search returned error: %s", result.Error)
	}
}

// TestAgentProviderWithSiteRestriction tests AgentProvider with domain restriction
func TestAgentProviderWithSiteRestriction(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Create AgentProvider
	provider := web.NewAgentProvider("tests.web-agent")

	// Create a mock context
	ctx := createTestContext(t)

	// Execute search with site restriction
	req := &types.Request{
		Query:  "documentation",
		Type:   types.SearchTypeWeb,
		Source: types.SourceHook,
		Sites:  []string{"github.com"},
		Limit:  3,
	}

	result, err := provider.Search(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, types.SearchTypeWeb, result.Type)
	assert.Equal(t, types.SourceHook, result.Source)

	if result.Error == "" {
		// All results should be from github.com (mock data respects sites)
		for _, item := range result.Items {
			assert.Contains(t, item.URL, "github.com", "Result URL should be from github.com")
		}
		t.Logf("Site-restricted agent search returned %d results", result.Total)
	} else {
		t.Logf("Agent search returned error: %s", result.Error)
	}
}

// TestAgentProviderWithTimeRange tests AgentProvider with time range filter
func TestAgentProviderWithTimeRange(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Create AgentProvider
	provider := web.NewAgentProvider("tests.web-agent")

	// Create a mock context
	ctx := createTestContext(t)

	// Execute search with time range
	req := &types.Request{
		Query:     "artificial intelligence news",
		Type:      types.SearchTypeWeb,
		Source:    types.SourceAuto,
		TimeRange: "week",
		Limit:     5,
	}

	result, err := provider.Search(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, result)

	if result.Error == "" {
		t.Logf("Time-ranged agent search (last week) returned %d results in %dms", result.Total, result.Duration)
	} else {
		t.Logf("Agent search returned error: %s", result.Error)
	}
}

// TestAgentProviderNotFound tests AgentProvider when agent is not found
func TestAgentProviderNotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Create AgentProvider with non-existent agent
	provider := web.NewAgentProvider("nonexistent.agent")

	// Create a mock context
	ctx := createTestContext(t)

	req := &types.Request{
		Query:  "test query",
		Type:   types.SearchTypeWeb,
		Source: types.SourceAuto,
	}

	result, err := provider.Search(ctx, req)

	// Should not return error, but result should have error message
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.NotEmpty(t, result.Error)
	assert.Contains(t, result.Error, "not found")
}

// TestAgentProviderWithoutContext tests AgentProvider without context
func TestAgentProviderWithoutContext(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Create AgentProvider
	provider := web.NewAgentProvider("tests.web-agent")

	req := &types.Request{
		Query:  "test query",
		Type:   types.SearchTypeWeb,
		Source: types.SourceAuto,
	}

	// Call without context (nil)
	result, err := provider.Search(nil, req)

	// Should still work - agent provider handles nil context
	require.NoError(t, err)
	require.NotNil(t, result)
	// May have error if context is required for agent call
	t.Logf("Agent search without context: error=%s, total=%d", result.Error, result.Total)
}

// TestWebHandlerAgentMode tests the web handler in agent mode
func TestWebHandlerAgentMode(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Create handler with agent mode
	handler := web.NewHandler("tests.web-agent", nil)
	require.NotNil(t, handler)

	// Verify type
	assert.Equal(t, types.SearchTypeWeb, handler.Type())

	// Create a mock context
	ctx := createTestContext(t)

	// Execute search with context
	req := &types.Request{
		Query:  "Yao framework",
		Type:   types.SearchTypeWeb,
		Source: types.SourceAuto,
		Limit:  5,
	}

	result, err := handler.SearchWithContext(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, types.SearchTypeWeb, result.Type)
	assert.Equal(t, "Yao framework", result.Query)

	if result.Error == "" {
		t.Logf("Handler agent mode returned %d results", result.Total)
	} else {
		t.Logf("Handler agent mode returned error: %s", result.Error)
	}
}

// TestWebHandlerAgentModeWithoutContext tests the web handler in agent mode without context
func TestWebHandlerAgentModeWithoutContext(t *testing.T) {
	// Create handler with agent mode
	handler := web.NewHandler("tests.web-agent", nil)
	require.NotNil(t, handler)

	req := &types.Request{
		Query:  "test",
		Type:   types.SearchTypeWeb,
		Source: types.SourceAuto,
	}

	// Call Search() without context (uses SearchWithContext with nil)
	result, err := handler.Search(req)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.NotEmpty(t, result.Error)
	assert.Contains(t, result.Error, "requires context")
}

// createTestContext creates a test context for agent calls
func createTestContext(t *testing.T) *agentContext.Context {
	authorized := &oauthTypes.AuthorizedInfo{
		UserID:   "test-user",
		TenantID: "test-tenant",
	}
	ctx := agentContext.New(context.Background(), authorized, "test-chat-id")
	ctx.AssistantID = "tests.web-agent-caller"
	return ctx
}
