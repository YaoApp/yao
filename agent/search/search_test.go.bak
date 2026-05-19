package search

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/agent/search/types"
)

func TestNew(t *testing.T) {
	t.Run("with nil config and uses", func(t *testing.T) {
		s := New(nil, nil)
		assert.NotNil(t, s)
		assert.NotNil(t, s.config)
		assert.NotNil(t, s.handlers)
		assert.NotNil(t, s.citation)
		assert.Equal(t, 3, len(s.handlers)) // web, kb, db
	})

	t.Run("with config", func(t *testing.T) {
		cfg := &types.Config{
			Web: &types.WebConfig{
				Provider:   "tavily",
				MaxResults: 10,
			},
			KB: &types.KBConfig{
				Collections: []string{"docs"},
				Threshold:   0.8,
			},
			DB: &types.DBConfig{
				Models:     []string{"product"},
				MaxResults: 20,
			},
		}
		s := New(cfg, nil)
		assert.NotNil(t, s)
		assert.Equal(t, cfg, s.config)
	})

	t.Run("with uses", func(t *testing.T) {
		uses := &Uses{
			Search:   "builtin",
			Web:      "builtin",
			Keyword:  "builtin",
			QueryDSL: "builtin",
			Rerank:   "builtin",
		}
		s := New(nil, uses)
		assert.NotNil(t, s)
	})
}

func TestSearcher_Search_UnsupportedType(t *testing.T) {
	s := New(nil, nil)

	req := &types.Request{
		Type:  "unsupported",
		Query: "test",
	}

	result, err := s.Search(nil, req)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "unsupported search type", result.Error)
}

func TestSearcher_Search_Web(t *testing.T) {
	// Note: This test uses skeleton handlers that return empty results
	// Real tests with actual API calls are in handlers/web/*_test.go
	cfg := &types.Config{
		Web: &types.WebConfig{
			Provider: "tavily",
		},
	}
	s := New(cfg, &Uses{Web: "builtin"})

	req := &types.Request{
		Type:   types.SearchTypeWeb,
		Query:  "test query",
		Source: types.SourceAuto,
	}

	result, err := s.Search(nil, req)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, types.SearchTypeWeb, result.Type)
	assert.Equal(t, "test query", result.Query)
	// Note: actual result depends on API key availability
}

func TestSearcher_Search_KB(t *testing.T) {
	cfg := &types.Config{
		KB: &types.KBConfig{
			Collections: []string{"docs"},
			Threshold:   0.7,
		},
	}
	s := New(cfg, nil)

	req := &types.Request{
		Type:        types.SearchTypeKB,
		Query:       "test query",
		Source:      types.SourceHook,
		Collections: []string{"docs"},
	}

	result, err := s.Search(nil, req)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, types.SearchTypeKB, result.Type)
	assert.Equal(t, "test query", result.Query)
	assert.Equal(t, types.SourceHook, result.Source)
	// Skeleton returns empty items
	assert.Equal(t, 0, len(result.Items))
}

func TestSearcher_Search_DB(t *testing.T) {
	cfg := &types.Config{
		DB: &types.DBConfig{
			Models:     []string{"product"},
			MaxResults: 20,
		},
	}
	s := New(cfg, &Uses{QueryDSL: "builtin"})

	req := &types.Request{
		Type:   types.SearchTypeDB,
		Query:  "find products under $100",
		Source: types.SourceUser,
		Models: []string{"product"},
	}

	result, err := s.Search(nil, req)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, types.SearchTypeDB, result.Type)
	assert.Equal(t, "find products under $100", result.Query)
	assert.Equal(t, types.SourceUser, result.Source)
	// Skeleton returns empty items
	assert.Equal(t, 0, len(result.Items))
}

func TestSearcher_Search_WeightAssignment(t *testing.T) {
	cfg := &types.Config{
		KB: &types.KBConfig{
			Collections: []string{"docs"},
		},
		Weights: &types.WeightsConfig{
			User: 1.0,
			Hook: 0.8,
			Auto: 0.6,
		},
	}
	s := New(cfg, nil)

	// Test with different sources
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
			Type:        types.SearchTypeKB,
			Query:       "test",
			Source:      tc.source,
			Collections: []string{"docs"},
		}
		result, err := s.Search(nil, req)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		// Items are empty in skeleton, so weight assignment can't be verified here
		// This test ensures the code path works without error
	}
}

func TestSearcher_All(t *testing.T) {
	cfg := &types.Config{
		KB: &types.KBConfig{
			Collections: []string{"docs"},
		},
		DB: &types.DBConfig{
			Models: []string{"product"},
		},
	}
	s := New(cfg, nil)

	reqs := []*types.Request{
		{
			Type:        types.SearchTypeKB,
			Query:       "KB query",
			Source:      types.SourceAuto,
			Collections: []string{"docs"},
		},
		{
			Type:   types.SearchTypeDB,
			Query:  "DB query",
			Source: types.SourceAuto,
			Models: []string{"product"},
		},
	}

	// Test All() - waits for all searches to complete (like Promise.all)
	results, err := s.All(nil, reqs)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(results))

	// Verify each result corresponds to its request
	assert.Equal(t, types.SearchTypeKB, results[0].Type)
	assert.Equal(t, "KB query", results[0].Query)

	assert.Equal(t, types.SearchTypeDB, results[1].Type)
	assert.Equal(t, "DB query", results[1].Query)
}

func TestSearcher_Any(t *testing.T) {
	cfg := &types.Config{
		KB: &types.KBConfig{
			Collections: []string{"docs"},
		},
		DB: &types.DBConfig{
			Models: []string{"product"},
		},
	}
	s := New(cfg, nil)

	reqs := []*types.Request{
		{
			Type:        types.SearchTypeKB,
			Query:       "KB query",
			Source:      types.SourceAuto,
			Collections: []string{"docs"},
		},
		{
			Type:   types.SearchTypeDB,
			Query:  "DB query",
			Source: types.SourceAuto,
			Models: []string{"product"},
		},
	}

	// Test Any() - returns when first search has results (like Promise.any)
	// Note: With skeleton handlers returning empty results, this will wait for all
	results, err := s.Any(nil, reqs)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(results))
}

func TestSearcher_Race(t *testing.T) {
	cfg := &types.Config{
		KB: &types.KBConfig{
			Collections: []string{"docs"},
		},
		DB: &types.DBConfig{
			Models: []string{"product"},
		},
	}
	s := New(cfg, nil)

	reqs := []*types.Request{
		{
			Type:        types.SearchTypeKB,
			Query:       "KB query",
			Source:      types.SourceAuto,
			Collections: []string{"docs"},
		},
		{
			Type:   types.SearchTypeDB,
			Query:  "DB query",
			Source: types.SourceAuto,
			Models: []string{"product"},
		},
	}

	// Test Race() - returns when first search completes (like Promise.race)
	results, err := s.Race(nil, reqs)
	assert.NoError(t, err)
	// At least one result should be set
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
	s := New(nil, nil)

	results, err := s.All(nil, []*types.Request{})
	assert.NoError(t, err)
	assert.Equal(t, 0, len(results))
}

func TestSearcher_Any_Empty(t *testing.T) {
	s := New(nil, nil)

	results, err := s.Any(nil, []*types.Request{})
	assert.NoError(t, err)
	assert.Equal(t, 0, len(results))
}

func TestSearcher_Race_Empty(t *testing.T) {
	s := New(nil, nil)

	results, err := s.Race(nil, []*types.Request{})
	assert.NoError(t, err)
	assert.Equal(t, 0, len(results))
}

func TestSearcher_All_ManyRequests(t *testing.T) {
	cfg := &types.Config{
		KB: &types.KBConfig{
			Collections: []string{"docs"},
		},
	}
	s := New(cfg, nil)

	// Create multiple requests to test parallel execution
	reqs := make([]*types.Request, 10)
	for i := 0; i < 10; i++ {
		reqs[i] = &types.Request{
			Type:        types.SearchTypeKB,
			Query:       "test query",
			Source:      types.SourceAuto,
			Collections: []string{"docs"},
		}
	}

	results, err := s.All(nil, reqs)
	assert.NoError(t, err)
	assert.Equal(t, 10, len(results))

	// All results should be valid
	for _, result := range results {
		assert.NotNil(t, result)
		assert.Equal(t, types.SearchTypeKB, result.Type)
	}
}

func TestSearcher_BuildReferences(t *testing.T) {
	s := New(nil, nil)

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
		{
			Type: types.SearchTypeKB,
			Items: []*types.ResultItem{
				{
					CitationID: "2",
					Type:       types.SearchTypeKB,
					Source:     types.SourceHook,
					Weight:     0.8,
					Title:      "KB Result",
					Content:    "KB content",
				},
			},
		},
	}

	refs := s.BuildReferences(results)
	assert.Equal(t, 2, len(refs))
	assert.Equal(t, "1", refs[0].ID)
	assert.Equal(t, "2", refs[1].ID)
}

func TestSearcher_CitationGeneration(t *testing.T) {
	s := New(nil, nil)

	// Reset citation generator for predictable IDs
	s.citation.Reset()

	// Note: This test would need actual results with items to verify citation generation
	// The skeleton handlers return empty items, so we test the citation generator directly

	id1 := s.citation.Next()
	id2 := s.citation.Next()
	id3 := s.citation.Next()

	// Citation IDs are now simple integers
	assert.Equal(t, "1", id1)
	assert.Equal(t, "2", id2)
	assert.Equal(t, "3", id3)
}
