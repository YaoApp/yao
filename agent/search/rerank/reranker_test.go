package rerank

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/agent/search/types"
)

func TestReranker_BuiltinMode(t *testing.T) {
	reranker := NewReranker("builtin", &types.RerankConfig{TopN: 5})

	items := []*types.ResultItem{
		{CitationID: "ref_001", Score: 0.9, Weight: 1.0},
		{CitationID: "ref_002", Score: 0.8, Weight: 1.0},
		{CitationID: "ref_003", Score: 0.7, Weight: 1.0},
	}

	result, err := reranker.Rerank(nil, "test query", items, nil)

	assert.NoError(t, err)
	assert.Len(t, result, 3)
	assert.Equal(t, "ref_001", result[0].CitationID)
}

func TestReranker_EmptyUsesRerank(t *testing.T) {
	// Empty usesRerank should use builtin
	reranker := NewReranker("", &types.RerankConfig{TopN: 5})

	items := []*types.ResultItem{
		{CitationID: "ref_001", Score: 0.9, Weight: 1.0},
	}

	result, err := reranker.Rerank(nil, "test query", items, nil)

	assert.NoError(t, err)
	assert.Len(t, result, 1)
}

func TestReranker_MergeOptions(t *testing.T) {
	// Config sets TopN = 5
	reranker := NewReranker("builtin", &types.RerankConfig{TopN: 5})

	items := []*types.ResultItem{
		{CitationID: "ref_001", Score: 0.9, Weight: 1.0},
		{CitationID: "ref_002", Score: 0.8, Weight: 1.0},
		{CitationID: "ref_003", Score: 0.7, Weight: 1.0},
		{CitationID: "ref_004", Score: 0.6, Weight: 1.0},
		{CitationID: "ref_005", Score: 0.5, Weight: 1.0},
		{CitationID: "ref_006", Score: 0.4, Weight: 1.0},
	}

	// Runtime opts override config
	result, err := reranker.Rerank(nil, "test query", items, &types.RerankOptions{TopN: 3})

	assert.NoError(t, err)
	assert.Len(t, result, 3)
}

func TestReranker_ConfigTopN(t *testing.T) {
	// Config sets TopN = 3
	reranker := NewReranker("builtin", &types.RerankConfig{TopN: 3})

	items := []*types.ResultItem{
		{CitationID: "ref_001", Score: 0.9, Weight: 1.0},
		{CitationID: "ref_002", Score: 0.8, Weight: 1.0},
		{CitationID: "ref_003", Score: 0.7, Weight: 1.0},
		{CitationID: "ref_004", Score: 0.6, Weight: 1.0},
		{CitationID: "ref_005", Score: 0.5, Weight: 1.0},
	}

	// No runtime opts, should use config TopN
	result, err := reranker.Rerank(nil, "test query", items, nil)

	assert.NoError(t, err)
	assert.Len(t, result, 3)
}

func TestReranker_NilConfig(t *testing.T) {
	reranker := NewReranker("builtin", nil)

	items := []*types.ResultItem{
		{CitationID: "ref_001", Score: 0.9, Weight: 1.0},
	}

	result, err := reranker.Rerank(nil, "test query", items, nil)

	assert.NoError(t, err)
	assert.Len(t, result, 1)
}

func TestReranker_EmptyItems(t *testing.T) {
	reranker := NewReranker("builtin", &types.RerankConfig{TopN: 5})

	result, err := reranker.Rerank(nil, "test query", []*types.ResultItem{}, nil)

	assert.NoError(t, err)
	assert.Empty(t, result)
}
