//go:build integration

package search_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/search"
	"github.com/yaoapp/yao/agent/search/types"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

func TestNew(t *testing.T) {
	testprepare.PrepareSandbox(t)

	t.Run("nil config and uses", func(t *testing.T) {
		s := search.New(nil, nil)
		require.NotNil(t, s)
	})

	t.Run("with web config", func(t *testing.T) {
		cfg := &types.Config{
			Web: &types.WebConfig{
				Provider:   "tavily",
				MaxResults: 10,
			},
		}
		s := search.New(cfg, nil)
		require.NotNil(t, s)
	})

	t.Run("with uses", func(t *testing.T) {
		uses := &search.Uses{
			Search:   "builtin",
			Web:      "builtin",
			Keyword:  "builtin",
			QueryDSL: "builtin",
			Rerank:   "builtin",
		}
		s := search.New(nil, uses)
		require.NotNil(t, s)
	})
}

func TestSearcher_Search_UnsupportedType(t *testing.T) {
	testprepare.PrepareSandbox(t)

	s := search.New(nil, nil)
	req := &types.Request{
		Type:  "unsupported",
		Query: "test",
	}

	result, err := s.Search(nil, req)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "unsupported search type", result.Error)
}

func TestSearcher_Search_Web(t *testing.T) {
	testprepare.PrepareSandbox(t)

	cfg := &types.Config{
		Web: &types.WebConfig{
			Provider: "tavily",
		},
	}
	s := search.New(cfg, &search.Uses{Web: "builtin"})

	req := &types.Request{
		Type:   types.SearchTypeWeb,
		Query:  "test query",
		Source: types.SourceAuto,
	}

	result, err := s.Search(nil, req)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, types.SearchTypeWeb, result.Type)
	assert.Equal(t, "test query", result.Query)
}

func TestSearcher_Search_WeightAssignment(t *testing.T) {
	testprepare.PrepareSandbox(t)

	cfg := &types.Config{
		Weights: &types.WeightsConfig{
			User: 1.0,
			Hook: 0.8,
			Auto: 0.6,
		},
	}
	s := search.New(cfg, nil)

	sources := []struct {
		source types.SourceType
		weight float64
	}{
		{types.SourceUser, 1.0},
		{types.SourceHook, 0.8},
		{types.SourceAuto, 0.6},
	}

	for _, tc := range sources {
		req := &types.Request{
			Type:   types.SearchTypeWeb,
			Query:  "test",
			Source: tc.source,
		}
		result, err := s.Search(nil, req)
		require.NoError(t, err)
		require.NotNil(t, result)
	}
}

func TestSearcher_All(t *testing.T) {
	testprepare.PrepareSandbox(t)
	s := search.New(nil, nil)

	reqs := []*types.Request{
		{Type: types.SearchTypeWeb, Query: "web query", Source: types.SourceAuto},
		{Type: types.SearchTypeWeb, Query: "web query 2", Source: types.SourceAuto},
	}

	results, err := s.All(nil, reqs)
	require.NoError(t, err)
	assert.Equal(t, 2, len(results))
	assert.Equal(t, "web query", results[0].Query)
	assert.Equal(t, "web query 2", results[1].Query)
}

func TestSearcher_Any(t *testing.T) {
	testprepare.PrepareSandbox(t)
	s := search.New(nil, nil)

	reqs := []*types.Request{
		{Type: types.SearchTypeWeb, Query: "web query", Source: types.SourceAuto},
		{Type: types.SearchTypeWeb, Query: "web query 2", Source: types.SourceAuto},
	}

	results, err := s.Any(nil, reqs)
	require.NoError(t, err)
	assert.Equal(t, 2, len(results))
}

func TestSearcher_Race(t *testing.T) {
	testprepare.PrepareSandbox(t)
	s := search.New(nil, nil)

	reqs := []*types.Request{
		{Type: types.SearchTypeWeb, Query: "web query", Source: types.SourceAuto},
		{Type: types.SearchTypeWeb, Query: "web query 2", Source: types.SourceAuto},
	}

	results, err := s.Race(nil, reqs)
	require.NoError(t, err)

	hasResult := false
	for _, r := range results {
		if r != nil {
			hasResult = true
			break
		}
	}
	assert.True(t, hasResult)
}

func TestSearcher_All_Empty(t *testing.T) {
	testprepare.PrepareSandbox(t)
	s := search.New(nil, nil)

	results, err := s.All(nil, []*types.Request{})
	require.NoError(t, err)
	assert.Equal(t, 0, len(results))
}

func TestSearcher_Any_Empty(t *testing.T) {
	testprepare.PrepareSandbox(t)
	s := search.New(nil, nil)

	results, err := s.Any(nil, []*types.Request{})
	require.NoError(t, err)
	assert.Equal(t, 0, len(results))
}

func TestSearcher_Race_Empty(t *testing.T) {
	testprepare.PrepareSandbox(t)
	s := search.New(nil, nil)

	results, err := s.Race(nil, []*types.Request{})
	require.NoError(t, err)
	assert.Equal(t, 0, len(results))
}

func TestSearcher_All_ManyRequests(t *testing.T) {
	testprepare.PrepareSandbox(t)
	s := search.New(nil, nil)

	reqs := make([]*types.Request, 10)
	for i := 0; i < 10; i++ {
		reqs[i] = &types.Request{
			Type:   types.SearchTypeWeb,
			Query:  "test query",
			Source: types.SourceAuto,
		}
	}

	results, err := s.All(nil, reqs)
	require.NoError(t, err)
	assert.Equal(t, 10, len(results))

	for _, result := range results {
		require.NotNil(t, result)
		assert.Equal(t, types.SearchTypeWeb, result.Type)
	}
}

func TestSearcher_BuildReferences(t *testing.T) {
	testprepare.PrepareSandbox(t)
	s := search.New(nil, nil)

	results := []*types.Result{
		{
			Type: types.SearchTypeWeb,
			Items: []*types.ResultItem{
				{
					CitationID: "1",
					Type:       types.SearchTypeWeb,
					Source:     types.SourceAuto,
					Weight:     0.6,
					Title:      "Web Result",
					Content:    "Web content",
					URL:        "https://example.com",
				},
			},
		},
	}

	refs := s.BuildReferences(results)
	assert.Equal(t, 1, len(refs))
	assert.Equal(t, "1", refs[0].ID)
}
