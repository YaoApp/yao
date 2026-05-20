//go:build integration

package context_test

import (
	stdContext "context"
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/assistant"
	agentctx "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/search/types"
	oauthTypes "github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

func newSearchTestContext(chatID, assistantID string) *agentctx.Context {
	authorized := &oauthTypes.AuthorizedInfo{
		Subject:   "test-user",
		ClientID:  "test-client-id",
		Scope:     "openid profile email",
		SessionID: "test-session-id",
		UserID:    "test-user-123",
	}

	ctx := agentctx.New(stdContext.Background(), authorized, chatID)
	ctx.AssistantID = assistantID
	ctx.Locale = "en-us"
	ctx.Referer = agentctx.RefererAPI
	ctx.Accept = agentctx.AcceptWebCUI
	ctx.Metadata = make(map[string]interface{})
	return ctx
}

func getResponseContent(res *agentctx.HookCreateResponse) string {
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

func requireSerpAPIKey(t *testing.T) {
	t.Helper()
	key := os.Getenv("SERPAPI_API_KEY")
	require.NotEmpty(t, key, "SERPAPI_API_KEY must be set in agent-test.env")
}

func TestSearchJSAPI_Web(t *testing.T) {
	testprepare.PrepareSandbox(t)
	requireSerpAPIKey(t)

	agent, err := assistant.Get("tests.search-jsapi")
	require.NoError(t, err, "Failed to get tests.search-jsapi assistant")
	require.NotNil(t, agent.HookScript, "The tests.search-jsapi assistant has no script")

	ctx := newSearchTestContext("chat-search-web", "tests.search-jsapi")

	res, _, err := agent.HookScript.Create(ctx, []agentctx.Message{{Role: "user", Content: "test:web Yao App Engine"}})
	require.NoError(t, err, "Create hook failed")
	require.NotNil(t, res, "Expected non-nil response")

	content := getResponseContent(res)
	require.NotEmpty(t, content, "Expected response content")

	var result types.Result
	err = json.Unmarshal([]byte(content), &result)
	require.NoError(t, err, "Response should be valid JSON: %s", content)

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

func TestSearchJSAPI_WebWithSites(t *testing.T) {
	testprepare.PrepareSandbox(t)
	requireSerpAPIKey(t)

	agent, err := assistant.Get("tests.search-jsapi")
	require.NoError(t, err)
	require.NotNil(t, agent.HookScript)

	ctx := newSearchTestContext("chat-search-web-sites", "tests.search-jsapi")

	res, _, err := agent.HookScript.Create(ctx, []agentctx.Message{{Role: "user", Content: "test:web_sites Yao App Engine"}})
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

	t.Logf("Site-restricted search returned %d items (site filtering depends on provider support)", len(result.Items))
}

func TestSearchJSAPI_All(t *testing.T) {
	testprepare.PrepareSandbox(t)
	requireSerpAPIKey(t)

	agent, err := assistant.Get("tests.search-jsapi")
	require.NoError(t, err)
	require.NotNil(t, agent.HookScript)

	ctx := newSearchTestContext("chat-search-all", "tests.search-jsapi")

	res, _, err := agent.HookScript.Create(ctx, []agentctx.Message{{Role: "user", Content: "test:all"}})
	require.NoError(t, err)
	require.NotNil(t, res)

	content := getResponseContent(res)
	require.NotEmpty(t, content, "Expected response content")

	var results []*types.Result
	err = json.Unmarshal([]byte(content), &results)
	require.NoError(t, err, "Response should be valid JSON array: %s", content)

	assert.Len(t, results, 2, "should have 2 results")

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

func TestSearchJSAPI_Any(t *testing.T) {
	testprepare.PrepareSandbox(t)
	requireSerpAPIKey(t)

	agent, err := assistant.Get("tests.search-jsapi")
	require.NoError(t, err)
	require.NotNil(t, agent.HookScript)

	ctx := newSearchTestContext("chat-search-any", "tests.search-jsapi")

	res, _, err := agent.HookScript.Create(ctx, []agentctx.Message{{Role: "user", Content: "test:any"}})
	require.NoError(t, err)
	require.NotNil(t, res)

	content := getResponseContent(res)
	require.NotEmpty(t, content, "Expected response content")

	var results []*types.Result
	err = json.Unmarshal([]byte(content), &results)
	require.NoError(t, err, "Response should be valid JSON array: %s", content)

	assert.Len(t, results, 2, "should have 2 result slots")

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

func TestSearchJSAPI_Race(t *testing.T) {
	testprepare.PrepareSandbox(t)
	requireSerpAPIKey(t)

	agent, err := assistant.Get("tests.search-jsapi")
	require.NoError(t, err)
	require.NotNil(t, agent.HookScript)

	ctx := newSearchTestContext("chat-search-race", "tests.search-jsapi")

	res, _, err := agent.HookScript.Create(ctx, []agentctx.Message{{Role: "user", Content: "test:race"}})
	require.NoError(t, err)
	require.NotNil(t, res)

	content := getResponseContent(res)
	require.NotEmpty(t, content, "Expected response content")

	var results []*types.Result
	err = json.Unmarshal([]byte(content), &results)
	require.NoError(t, err, "Response should be valid JSON array: %s", content)

	assert.Len(t, results, 2, "should have 2 result slots")

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

func TestSearchJSAPI_InvalidCommand(t *testing.T) {
	testprepare.PrepareSandbox(t)

	agent, err := assistant.Get("tests.search-jsapi")
	require.NoError(t, err)
	require.NotNil(t, agent.HookScript)

	ctx := newSearchTestContext("chat-search-invalid", "tests.search-jsapi")

	res, _, err := agent.HookScript.Create(ctx, []agentctx.Message{{Role: "user", Content: "invalid command"}})
	require.NoError(t, err)
	require.NotNil(t, res)

	content := getResponseContent(res)
	assert.Contains(t, content, "Invalid test command", "should return error message")
}

func TestSearchJSAPI_UnknownMethod(t *testing.T) {
	testprepare.PrepareSandbox(t)

	agent, err := assistant.Get("tests.search-jsapi")
	require.NoError(t, err)
	require.NotNil(t, agent.HookScript)

	ctx := newSearchTestContext("chat-search-unknown", "tests.search-jsapi")

	res, _, err := agent.HookScript.Create(ctx, []agentctx.Message{{Role: "user", Content: "test:unknown"}})
	require.NoError(t, err)
	require.NotNil(t, res)

	content := getResponseContent(res)
	assert.Contains(t, content, "Unknown test method", "should return error message")
}
