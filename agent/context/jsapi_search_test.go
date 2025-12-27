package context_test

import (
	stdContext "context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/assistant"
	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/search/types"
	"github.com/yaoapp/yao/agent/testutils"
	oauthTypes "github.com/yaoapp/yao/openapi/oauth/types"
)

// Note: SearchAPIFactory is set by assistant.init() with proper config getter
// We import assistant package to ensure init() runs before tests

// newSearchTestContext creates a Context for search JSAPI testing
func newSearchTestContext(chatID, assistantID string) *context.Context {
	authorized := &oauthTypes.AuthorizedInfo{
		Subject:   "test-user",
		ClientID:  "test-client-id",
		Scope:     "openid profile email",
		SessionID: "test-session-id",
		UserID:    "test-user-123",
	}

	ctx := context.New(stdContext.Background(), authorized, chatID)
	ctx.AssistantID = assistantID
	ctx.Locale = "en-us"
	ctx.Referer = context.RefererAPI
	ctx.Accept = context.AcceptWebCUI
	ctx.Metadata = make(map[string]interface{})
	return ctx
}

// getResponseContent extracts the content from the first assistant message
func getResponseContent(res *context.HookCreateResponse) string {
	if res == nil || len(res.Messages) == 0 {
		return ""
	}
	for _, msg := range res.Messages {
		if msg.Role == "assistant" {
			if content, ok := msg.Content.(string); ok {
				return content
			}
		}
	}
	return ""
}

// TestSearchJSAPI_Web tests ctx.search.Web() via Create Hook
// Skip: requires external API key (Tavily/Serper)
func TestSearchJSAPI_Web(t *testing.T) {
	t.Skip("Skipping: requires external API key (Tavily/Serper)")
	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Load the search-jsapi test assistant
	agent, err := assistant.Get("tests.search-jsapi")
	require.NoError(t, err, "Failed to get tests.search-jsapi assistant")
	require.NotNil(t, agent.HookScript, "The tests.search-jsapi assistant has no script")

	ctx := newSearchTestContext("chat-search-web", "tests.search-jsapi")

	// Call Create hook with test:web command
	res, _, err := agent.HookScript.Create(ctx, []context.Message{{Role: "user", Content: "test:web Yao App Engine"}})
	require.NoError(t, err, "Create hook failed")
	require.NotNil(t, res, "Expected non-nil response")

	// Get response content from messages
	content := getResponseContent(res)
	require.NotEmpty(t, content, "Expected response content")

	// Parse the JSON response
	var result types.Result
	err = json.Unmarshal([]byte(content), &result)
	require.NoError(t, err, "Response should be valid JSON: %s", content)

	// Verify result
	assert.Equal(t, types.SearchTypeWeb, result.Type, "type should be web")
	assert.Equal(t, "Yao App Engine", result.Query, "query should match")
	assert.Empty(t, result.Error, "should not have error: %s", result.Error)
	assert.Greater(t, len(result.Items), 0, "should have items")

	t.Logf("Web search returned %d items", len(result.Items))
	for i, item := range result.Items {
		if i < 3 {
			t.Logf("  [%s] %s - %s", item.CitationID, item.Title, item.URL)
		}
	}
}

// TestSearchJSAPI_WebWithSites tests ctx.search.Web() with site restriction
// Skip: requires external API key (Tavily/Serper)
func TestSearchJSAPI_WebWithSites(t *testing.T) {
	t.Skip("Skipping: requires external API key (Tavily/Serper)")
	testutils.Prepare(t)
	defer testutils.Clean(t)

	agent, err := assistant.Get("tests.search-jsapi")
	require.NoError(t, err)
	require.NotNil(t, agent.HookScript)

	ctx := newSearchTestContext("chat-search-web-sites", "tests.search-jsapi")

	res, _, err := agent.HookScript.Create(ctx, []context.Message{{Role: "user", Content: "test:web_sites Yao App Engine"}})
	require.NoError(t, err)
	require.NotNil(t, res)

	content := getResponseContent(res)
	require.NotEmpty(t, content, "Expected response content")

	var result types.Result
	err = json.Unmarshal([]byte(content), &result)
	require.NoError(t, err, "Response should be valid JSON: %s", content)

	assert.Equal(t, types.SearchTypeWeb, result.Type)
	assert.Empty(t, result.Error, "should not have error: %s", result.Error)
	assert.Greater(t, len(result.Items), 0, "should have items")

	// Verify all results are from allowed sites
	allowedSites := []string{"github.com", "yaoapps.com"}
	for _, item := range result.Items {
		isAllowed := false
		for _, site := range allowedSites {
			if strings.Contains(item.URL, site) {
				isAllowed = true
				break
			}
		}
		assert.True(t, isAllowed, "URL %s should be from allowed sites", item.URL)
	}

	t.Logf("Site-restricted search returned %d items", len(result.Items))
}

// TestSearchJSAPI_KB tests ctx.search.KB() via Create Hook (skeleton)
func TestSearchJSAPI_KB(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	agent, err := assistant.Get("tests.search-jsapi")
	require.NoError(t, err)
	require.NotNil(t, agent.HookScript)

	ctx := newSearchTestContext("chat-search-kb", "tests.search-jsapi")

	res, _, err := agent.HookScript.Create(ctx, []context.Message{{Role: "user", Content: "test:kb test query"}})
	require.NoError(t, err)
	require.NotNil(t, res)

	content := getResponseContent(res)
	require.NotEmpty(t, content, "Expected response content")

	var result types.Result
	err = json.Unmarshal([]byte(content), &result)
	require.NoError(t, err, "Response should be valid JSON: %s", content)

	assert.Equal(t, types.SearchTypeKB, result.Type, "type should be kb")
	assert.Equal(t, "test query", result.Query, "query should match")
	assert.Equal(t, types.SourceHook, result.Source, "source should be hook")
}

// TestSearchJSAPI_DB tests ctx.search.DB() via Create Hook (skeleton)
func TestSearchJSAPI_DB(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	agent, err := assistant.Get("tests.search-jsapi")
	require.NoError(t, err)
	require.NotNil(t, agent.HookScript)

	ctx := newSearchTestContext("chat-search-db", "tests.search-jsapi")

	res, _, err := agent.HookScript.Create(ctx, []context.Message{{Role: "user", Content: "test:db test query"}})
	require.NoError(t, err)
	require.NotNil(t, res)

	content := getResponseContent(res)
	require.NotEmpty(t, content, "Expected response content")

	var result types.Result
	err = json.Unmarshal([]byte(content), &result)
	require.NoError(t, err, "Response should be valid JSON: %s", content)

	assert.Equal(t, types.SearchTypeDB, result.Type, "type should be db")
	assert.Equal(t, "test query", result.Query, "query should match")
	assert.Equal(t, types.SourceHook, result.Source, "source should be hook")
}

// TestSearchJSAPI_All tests ctx.search.All() via Create Hook
// Skip: requires external API key (Tavily/Serper)
func TestSearchJSAPI_All(t *testing.T) {
	t.Skip("Skipping: requires external API key (Tavily/Serper)")
	testutils.Prepare(t)
	defer testutils.Clean(t)

	agent, err := assistant.Get("tests.search-jsapi")
	require.NoError(t, err)
	require.NotNil(t, agent.HookScript)

	ctx := newSearchTestContext("chat-search-all", "tests.search-jsapi")

	res, _, err := agent.HookScript.Create(ctx, []context.Message{{Role: "user", Content: "test:all"}})
	require.NoError(t, err)
	require.NotNil(t, res)

	content := getResponseContent(res)
	require.NotEmpty(t, content, "Expected response content")

	// Parse as array of results
	var results []*types.Result
	err = json.Unmarshal([]byte(content), &results)
	require.NoError(t, err, "Response should be valid JSON array: %s", content)

	assert.Len(t, results, 2, "should have 2 results")

	// Both should succeed
	successCount := 0
	totalItems := 0
	for _, r := range results {
		if r != nil && r.Error == "" {
			successCount++
			totalItems += len(r.Items)
		}
	}

	assert.Equal(t, 2, successCount, "both searches should succeed")
	assert.Greater(t, totalItems, 0, "should have items")

	t.Logf("All search: %d results, %d total items", len(results), totalItems)
}

// TestSearchJSAPI_Any tests ctx.search.Any() via Create Hook
// Skip: requires external API key (Tavily/Serper)
func TestSearchJSAPI_Any(t *testing.T) {
	t.Skip("Skipping: requires external API key (Tavily/Serper)")
	testutils.Prepare(t)
	defer testutils.Clean(t)

	agent, err := assistant.Get("tests.search-jsapi")
	require.NoError(t, err)
	require.NotNil(t, agent.HookScript)

	ctx := newSearchTestContext("chat-search-any", "tests.search-jsapi")

	res, _, err := agent.HookScript.Create(ctx, []context.Message{{Role: "user", Content: "test:any"}})
	require.NoError(t, err)
	require.NotNil(t, res)

	content := getResponseContent(res)
	require.NotEmpty(t, content, "Expected response content")

	var results []*types.Result
	err = json.Unmarshal([]byte(content), &results)
	require.NoError(t, err, "Response should be valid JSON array: %s", content)

	assert.Len(t, results, 2, "should have 2 result slots")

	// At least one should have results
	hasSuccess := false
	for _, r := range results {
		if r != nil && len(r.Items) > 0 && r.Error == "" {
			hasSuccess = true
			break
		}
	}
	assert.True(t, hasSuccess, "at least one search should succeed")

	t.Logf("Any search completed")
}

// TestSearchJSAPI_Race tests ctx.search.Race() via Create Hook
// Skip: requires external API key (Tavily/Serper)
func TestSearchJSAPI_Race(t *testing.T) {
	t.Skip("Skipping: requires external API key (Tavily/Serper)")
	testutils.Prepare(t)
	defer testutils.Clean(t)

	agent, err := assistant.Get("tests.search-jsapi")
	require.NoError(t, err)
	require.NotNil(t, agent.HookScript)

	ctx := newSearchTestContext("chat-search-race", "tests.search-jsapi")

	res, _, err := agent.HookScript.Create(ctx, []context.Message{{Role: "user", Content: "test:race"}})
	require.NoError(t, err)
	require.NotNil(t, res)

	content := getResponseContent(res)
	require.NotEmpty(t, content, "Expected response content")

	var results []*types.Result
	err = json.Unmarshal([]byte(content), &results)
	require.NoError(t, err, "Response should be valid JSON array: %s", content)

	assert.Len(t, results, 2, "should have 2 result slots")

	// At least one should have completed
	hasResult := false
	for _, r := range results {
		if r != nil {
			hasResult = true
			break
		}
	}
	assert.True(t, hasResult, "at least one search should complete")

	t.Logf("Race search completed")
}

// TestSearchJSAPI_InvalidCommand tests invalid test command
func TestSearchJSAPI_InvalidCommand(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	agent, err := assistant.Get("tests.search-jsapi")
	require.NoError(t, err)
	require.NotNil(t, agent.HookScript)

	ctx := newSearchTestContext("chat-search-invalid", "tests.search-jsapi")

	res, _, err := agent.HookScript.Create(ctx, []context.Message{{Role: "user", Content: "invalid command"}})
	require.NoError(t, err)
	require.NotNil(t, res)

	content := getResponseContent(res)
	assert.Contains(t, content, "Invalid test command", "should return error message")
}

// TestSearchJSAPI_UnknownMethod tests unknown test method
func TestSearchJSAPI_UnknownMethod(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	agent, err := assistant.Get("tests.search-jsapi")
	require.NoError(t, err)
	require.NotNil(t, agent.HookScript)

	ctx := newSearchTestContext("chat-search-unknown", "tests.search-jsapi")

	res, _, err := agent.HookScript.Create(ctx, []context.Message{{Role: "user", Content: "test:unknown"}})
	require.NoError(t, err)
	require.NotNil(t, res)

	content := getResponseContent(res)
	assert.Contains(t, content, "Unknown test method", "should return error message")
}
