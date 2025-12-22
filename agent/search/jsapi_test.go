package search_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/search"
	"github.com/yaoapp/yao/agent/search/types"
	"github.com/yaoapp/yao/agent/testutils"
)

func TestNewJSAPI(t *testing.T) {
	api := search.NewJSAPI(nil, nil, nil)
	require.NotNil(t, api)
}

func TestJSAPI_Web(t *testing.T) {
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

func TestJSAPI_KB(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	api := search.NewJSAPI(nil, &types.Config{
		KB: &types.KBConfig{Collections: []string{"docs"}},
	}, nil)

	result := api.KB("test query", nil)
	require.NotNil(t, result)

	r, ok := result.(*types.Result)
	require.True(t, ok)
	assert.Equal(t, types.SearchTypeKB, r.Type)
	assert.Equal(t, "test query", r.Query)
	assert.Equal(t, types.SourceHook, r.Source)
}

func TestJSAPI_KB_WithOptions(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	api := search.NewJSAPI(nil, &types.Config{
		KB: &types.KBConfig{Collections: []string{"docs"}},
	}, nil)

	opts := map[string]interface{}{
		"collections": []interface{}{"docs", "faq"},
		"threshold":   0.8,
		"limit":       float64(10),
		"graph":       true,
	}

	result := api.KB("knowledge base query", opts)
	require.NotNil(t, result)

	r, ok := result.(*types.Result)
	require.True(t, ok)
	assert.Equal(t, types.SearchTypeKB, r.Type)
	assert.Equal(t, "knowledge base query", r.Query)
}

func TestJSAPI_DB(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	api := search.NewJSAPI(nil, &types.Config{
		DB: &types.DBConfig{Models: []string{"product"}},
	}, &search.Uses{QueryDSL: "builtin"})

	result := api.DB("test query", nil)
	require.NotNil(t, result)

	r, ok := result.(*types.Result)
	require.True(t, ok)
	assert.Equal(t, types.SearchTypeDB, r.Type)
	assert.Equal(t, "test query", r.Query)
	assert.Equal(t, types.SourceHook, r.Source)
}

func TestJSAPI_DB_WithOptions(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	api := search.NewJSAPI(nil, &types.Config{
		DB: &types.DBConfig{Models: []string{"product"}},
	}, &search.Uses{QueryDSL: "builtin"})

	opts := map[string]interface{}{
		"models": []interface{}{"product", "order"},
		"select": []interface{}{"id", "name", "price"},
		"limit":  float64(20),
	}

	result := api.DB("database query", opts)
	require.NotNil(t, result)

	r, ok := result.(*types.Result)
	require.True(t, ok)
	assert.Equal(t, types.SearchTypeDB, r.Type)
	assert.Equal(t, "database query", r.Query)
}

func TestJSAPI_All(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	api := search.NewJSAPI(nil, &types.Config{
		KB: &types.KBConfig{Collections: []string{"docs"}},
		DB: &types.DBConfig{Models: []string{"product"}},
	}, nil)

	requests := []interface{}{
		map[string]interface{}{
			"type":  "kb",
			"query": "KB query",
		},
		map[string]interface{}{
			"type":  "db",
			"query": "DB query",
		},
	}

	results := api.All(requests)
	require.Len(t, results, 2)

	// First result (KB)
	r0, ok := results[0].(*types.Result)
	require.True(t, ok)
	assert.Equal(t, types.SearchTypeKB, r0.Type)
	assert.Equal(t, "KB query", r0.Query)

	// Second result (DB)
	r1, ok := results[1].(*types.Result)
	require.True(t, ok)
	assert.Equal(t, types.SearchTypeDB, r1.Type)
	assert.Equal(t, "DB query", r1.Query)
}

func TestJSAPI_Any(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	api := search.NewJSAPI(nil, &types.Config{
		KB: &types.KBConfig{Collections: []string{"docs"}},
		DB: &types.DBConfig{Models: []string{"product"}},
	}, nil)

	requests := []interface{}{
		map[string]interface{}{
			"type":  "kb",
			"query": "KB query",
		},
		map[string]interface{}{
			"type":  "db",
			"query": "DB query",
		},
	}

	results := api.Any(requests)
	require.Len(t, results, 2)

	// At least one result should be present
	hasResult := false
	for _, r := range results {
		if r != nil {
			hasResult = true
			break
		}
	}
	assert.True(t, hasResult)
}

func TestJSAPI_Race(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	api := search.NewJSAPI(nil, &types.Config{
		KB: &types.KBConfig{Collections: []string{"docs"}},
		DB: &types.DBConfig{Models: []string{"product"}},
	}, nil)

	requests := []interface{}{
		map[string]interface{}{
			"type":  "kb",
			"query": "KB query",
		},
		map[string]interface{}{
			"type":  "db",
			"query": "DB query",
		},
	}

	results := api.Race(requests)
	require.Len(t, results, 2)

	// At least one result should be present
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
	api := search.NewJSAPI(nil, nil, nil)
	results := api.All([]interface{}{})
	assert.Len(t, results, 0)
}

func TestJSAPI_Any_Empty(t *testing.T) {
	api := search.NewJSAPI(nil, nil, nil)
	results := api.Any([]interface{}{})
	assert.Len(t, results, 0)
}

func TestJSAPI_Race_Empty(t *testing.T) {
	api := search.NewJSAPI(nil, nil, nil)
	results := api.Race([]interface{}{})
	assert.Len(t, results, 0)
}

func TestJSAPI_Web_WithRerank(t *testing.T) {
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

func TestJSAPI_All_InvalidRequests(t *testing.T) {
	api := search.NewJSAPI(nil, &types.Config{
		Web: &types.WebConfig{Provider: "tavily"},
	}, &search.Uses{Web: "builtin"})

	// Mix of invalid and valid requests
	requests := []interface{}{
		"invalid", // Not a map
		map[string]interface{}{
			"query": "no type", // Missing type
		},
		map[string]interface{}{
			"type": "web", // Missing query
		},
		map[string]interface{}{
			"type":  "web",
			"query": "valid query",
		},
	}

	results := api.All(requests)
	// Only the valid request should produce a result
	assert.Len(t, results, 1)
}

func TestSetJSAPIFactory(t *testing.T) {
	// Reset factory
	context.SearchAPIFactory = nil

	// Set factory with nil getter (uses defaults)
	search.SetJSAPIFactory(nil)

	// Verify factory is set
	require.NotNil(t, context.SearchAPIFactory)

	// Create a mock context
	ctx := context.New(nil, nil, "test-chat")

	// Get search API
	searchAPI := context.SearchAPIFactory(ctx)
	require.NotNil(t, searchAPI)
}

func TestSetJSAPIFactory_WithGetter(t *testing.T) {
	// Reset factory
	context.SearchAPIFactory = nil

	// Set factory with custom getter
	search.SetJSAPIFactory(func(assistantID string) (*types.Config, *search.Uses) {
		if assistantID == "test-assistant" {
			return &types.Config{
				Web: &types.WebConfig{Provider: "tavily"},
			}, &search.Uses{Web: "builtin"}
		}
		return nil, nil
	})

	// Verify factory is set
	require.NotNil(t, context.SearchAPIFactory)

	// Create a context with assistant ID
	ctx := context.New(nil, nil, "test-chat")
	ctx.AssistantID = "test-assistant"

	// Get search API
	searchAPI := context.SearchAPIFactory(ctx)
	require.NotNil(t, searchAPI)
}

func TestJSAPI_ImplementsSearchAPI(t *testing.T) {
	// Verify JSAPI implements context.SearchAPI interface
	var _ context.SearchAPI = search.NewJSAPI(nil, nil, nil)
}
