//go:build integration

package search_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/search"
	"github.com/yaoapp/yao/agent/search/types"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

func TestNewJSAPI(t *testing.T) {
	testprepare.PrepareSandbox(t)

	api := search.NewJSAPI(nil, nil, nil)
	require.NotNil(t, api)
}

func TestJSAPI_Web(t *testing.T) {
	testprepare.PrepareSandbox(t)

	api := search.NewJSAPI(nil, &types.Config{
		Web: &types.WebConfig{Provider: "tavily"},
	}, &search.Uses{Web: "builtin"})

	result := api.Web("test query", nil)
	require.NotNil(t, result)

	r, ok := result.(*types.Result)
	require.True(t, ok)
	assert.Equal(t, types.SearchTypeWeb, r.Type)
	assert.Equal(t, "test query", r.Query)
	assert.Equal(t, types.SourceHook, r.Source)
}

func TestJSAPI_Web_WithOptions(t *testing.T) {
	testprepare.PrepareSandbox(t)

	api := search.NewJSAPI(nil, &types.Config{
		Web: &types.WebConfig{Provider: "tavily"},
	}, &search.Uses{Web: "builtin"})

	opts := map[string]interface{}{
		"limit":      float64(5),
		"sites":      []interface{}{"github.com", "stackoverflow.com"},
		"time_range": "week",
	}

	result := api.Web("golang concurrency", opts)
	require.NotNil(t, result)

	r, ok := result.(*types.Result)
	require.True(t, ok)
	assert.Equal(t, types.SearchTypeWeb, r.Type)
	assert.Equal(t, "golang concurrency", r.Query)
}

func TestJSAPI_Web_WithRerank(t *testing.T) {
	testprepare.PrepareSandbox(t)

	api := search.NewJSAPI(nil, &types.Config{
		Web: &types.WebConfig{Provider: "tavily"},
	}, &search.Uses{Web: "builtin"})

	opts := map[string]interface{}{
		"limit": float64(10),
		"rerank": map[string]interface{}{
			"top_n": float64(5),
		},
	}

	result := api.Web("test query", opts)
	require.NotNil(t, result)

	r, ok := result.(*types.Result)
	require.True(t, ok)
	assert.Equal(t, types.SearchTypeWeb, r.Type)
}

func TestJSAPI_All_WebOnly(t *testing.T) {
	testprepare.PrepareSandbox(t)

	api := search.NewJSAPI(nil, &types.Config{
		Web: &types.WebConfig{Provider: "tavily"},
	}, &search.Uses{Web: "builtin"})

	requests := []interface{}{
		map[string]interface{}{
			"type":  "web",
			"query": "query 1",
		},
		map[string]interface{}{
			"type":  "web",
			"query": "query 2",
		},
	}

	results := api.All(requests)
	require.Len(t, results, 2)

	r0, ok := results[0].(*types.Result)
	require.True(t, ok)
	assert.Equal(t, types.SearchTypeWeb, r0.Type)
	assert.Equal(t, "query 1", r0.Query)

	r1, ok := results[1].(*types.Result)
	require.True(t, ok)
	assert.Equal(t, types.SearchTypeWeb, r1.Type)
	assert.Equal(t, "query 2", r1.Query)
}

func TestJSAPI_Any_WebOnly(t *testing.T) {
	testprepare.PrepareSandbox(t)

	api := search.NewJSAPI(nil, &types.Config{
		Web: &types.WebConfig{Provider: "tavily"},
	}, &search.Uses{Web: "builtin"})

	requests := []interface{}{
		map[string]interface{}{
			"type":  "web",
			"query": "query 1",
		},
		map[string]interface{}{
			"type":  "web",
			"query": "query 2",
		},
	}

	results := api.Any(requests)
	require.Len(t, results, 2)

	hasResult := false
	for _, r := range results {
		if r != nil {
			hasResult = true
			break
		}
	}
	assert.True(t, hasResult)
}

func TestJSAPI_Race_WebOnly(t *testing.T) {
	testprepare.PrepareSandbox(t)

	api := search.NewJSAPI(nil, &types.Config{
		Web: &types.WebConfig{Provider: "tavily"},
	}, &search.Uses{Web: "builtin"})

	requests := []interface{}{
		map[string]interface{}{
			"type":  "web",
			"query": "query 1",
		},
		map[string]interface{}{
			"type":  "web",
			"query": "query 2",
		},
	}

	results := api.Race(requests)
	require.Len(t, results, 2)

	hasResult := false
	for _, r := range results {
		if r != nil {
			hasResult = true
			break
		}
	}
	assert.True(t, hasResult)
}

func TestJSAPI_All_Empty(t *testing.T) {
	testprepare.PrepareSandbox(t)

	api := search.NewJSAPI(nil, nil, nil)
	results := api.All([]interface{}{})
	assert.Len(t, results, 0)
}

func TestJSAPI_All_InvalidRequests(t *testing.T) {
	testprepare.PrepareSandbox(t)

	api := search.NewJSAPI(nil, &types.Config{
		Web: &types.WebConfig{Provider: "tavily"},
	}, &search.Uses{Web: "builtin"})

	requests := []interface{}{
		"invalid",
		map[string]interface{}{"query": "no type"},
		map[string]interface{}{"type": "web"},
		map[string]interface{}{"type": "web", "query": "valid query"},
	}

	results := api.All(requests)
	assert.Len(t, results, 1)
}

func TestSetJSAPIFactory(t *testing.T) {
	testprepare.PrepareSandbox(t)

	context.SearchAPIFactory = nil
	search.SetJSAPIFactory(nil)
	require.NotNil(t, context.SearchAPIFactory)

	ctx := context.New(nil, nil, "test-chat")
	searchAPI := context.SearchAPIFactory(ctx)
	require.NotNil(t, searchAPI)
}

func TestSetJSAPIFactory_WithGetter(t *testing.T) {
	testprepare.PrepareSandbox(t)

	context.SearchAPIFactory = nil
	search.SetJSAPIFactory(func(assistantID string) (*types.Config, *search.Uses) {
		if assistantID == "test-assistant" {
			return &types.Config{
				Web: &types.WebConfig{Provider: "tavily"},
			}, &search.Uses{Web: "builtin"}
		}
		return nil, nil
	})

	require.NotNil(t, context.SearchAPIFactory)
	ctx := context.New(nil, nil, "test-chat")
	ctx.AssistantID = "test-assistant"

	searchAPI := context.SearchAPIFactory(ctx)
	require.NotNil(t, searchAPI)
}

func TestJSAPI_ImplementsSearchAPI(t *testing.T) {
	testprepare.PrepareSandbox(t)
	var _ context.SearchAPI = search.NewJSAPI(nil, nil, nil)
}
