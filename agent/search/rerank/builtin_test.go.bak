package rerank

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/agent/search/types"
)

func TestBuiltinReranker_EmptyItems(t *testing.T) {
	reranker := NewBuiltinReranker()
	result, err := reranker.Rerank("test query", []*types.ResultItem{}, &types.RerankOptions{TopN: 5})

	assert.NoError(t, err)
	assert.Empty(t, result)
}

func TestBuiltinReranker_SortByWeightedScore(t *testing.T) {
	reranker := NewBuiltinReranker()

	items := []*types.ResultItem{
		{CitationID: "ref_001", Score: 0.8, Weight: 0.6}, // weighted: 0.48
		{CitationID: "ref_002", Score: 0.6, Weight: 1.0}, // weighted: 0.60
		{CitationID: "ref_003", Score: 0.9, Weight: 0.8}, // weighted: 0.72
		{CitationID: "ref_004", Score: 0.5, Weight: 1.0}, // weighted: 0.50
	}

	result, err := reranker.Rerank("test query", items, &types.RerankOptions{TopN: 10})

	assert.NoError(t, err)
	assert.Len(t, result, 4)

	// Should be sorted by weighted score: ref_003 (0.72) > ref_002 (0.60) > ref_004 (0.50) > ref_001 (0.48)
	assert.Equal(t, "ref_003", result[0].CitationID)
	assert.Equal(t, "ref_002", result[1].CitationID)
	assert.Equal(t, "ref_004", result[2].CitationID)
	assert.Equal(t, "ref_001", result[3].CitationID)
}

func TestBuiltinReranker_TopN(t *testing.T) {
	reranker := NewBuiltinReranker()

	items := []*types.ResultItem{
		{CitationID: "ref_001", Score: 0.9, Weight: 1.0},
		{CitationID: "ref_002", Score: 0.8, Weight: 1.0},
		{CitationID: "ref_003", Score: 0.7, Weight: 1.0},
		{CitationID: "ref_004", Score: 0.6, Weight: 1.0},
		{CitationID: "ref_005", Score: 0.5, Weight: 1.0},
	}

	result, err := reranker.Rerank("test query", items, &types.RerankOptions{TopN: 3})

	assert.NoError(t, err)
	assert.Len(t, result, 3)
	assert.Equal(t, "ref_001", result[0].CitationID)
	assert.Equal(t, "ref_002", result[1].CitationID)
	assert.Equal(t, "ref_003", result[2].CitationID)
}

func TestBuiltinReranker_DefaultWeight(t *testing.T) {
	reranker := NewBuiltinReranker()

	// Items without weight should use default 0.6
	items := []*types.ResultItem{
		{CitationID: "ref_001", Score: 0.9, Weight: 0},   // weighted: 0.9 * 0.6 = 0.54
		{CitationID: "ref_002", Score: 0.5, Weight: 1.0}, // weighted: 0.5 * 1.0 = 0.50
	}

	result, err := reranker.Rerank("test query", items, &types.RerankOptions{TopN: 10})

	assert.NoError(t, err)
	assert.Len(t, result, 2)
	// ref_001 (0.54) > ref_002 (0.50)
	assert.Equal(t, "ref_001", result[0].CitationID)
	assert.Equal(t, "ref_002", result[1].CitationID)
}

func TestBuiltinReranker_TopNLargerThanItems(t *testing.T) {
	reranker := NewBuiltinReranker()

	items := []*types.ResultItem{
		{CitationID: "ref_001", Score: 0.9, Weight: 1.0},
		{CitationID: "ref_002", Score: 0.8, Weight: 1.0},
	}

	// TopN > len(items) should return all items
	result, err := reranker.Rerank("test query", items, &types.RerankOptions{TopN: 10})

	assert.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestBuiltinReranker_ZeroTopN(t *testing.T) {
	reranker := NewBuiltinReranker()

	items := []*types.ResultItem{
		{CitationID: "ref_001", Score: 0.9, Weight: 1.0},
		{CitationID: "ref_002", Score: 0.8, Weight: 1.0},
	}

	// TopN = 0 should return all items
	result, err := reranker.Rerank("test query", items, &types.RerankOptions{TopN: 0})

	assert.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestBuiltinReranker_SameWeightedScore(t *testing.T) {
	reranker := NewBuiltinReranker()

	// Items with same weighted score - order should be stable
	items := []*types.ResultItem{
		{CitationID: "ref_001", Score: 0.8, Weight: 1.0}, // weighted: 0.80
		{CitationID: "ref_002", Score: 0.8, Weight: 1.0}, // weighted: 0.80
		{CitationID: "ref_003", Score: 0.4, Weight: 1.0}, // weighted: 0.40
	}

	result, err := reranker.Rerank("test query", items, &types.RerankOptions{TopN: 10})

	assert.NoError(t, err)
	assert.Len(t, result, 3)
	// ref_003 should be last
	assert.Equal(t, "ref_003", result[2].CitationID)
}
