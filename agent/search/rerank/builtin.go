package rerank

import (
	"sort"

	"github.com/yaoapp/yao/agent/search/types"
)

// BuiltinReranker implements simple score-based reranking
// For production use cases requiring semantic understanding, use Agent or MCP mode.
type BuiltinReranker struct{}

// NewBuiltinReranker creates a new builtin reranker
func NewBuiltinReranker() *BuiltinReranker {
	return &BuiltinReranker{}
}

// Rerank sorts items by weighted score (score * weight) and returns top N
// This is a simple implementation without semantic understanding.
func (r *BuiltinReranker) Rerank(query string, items []*types.ResultItem, opts *types.RerankOptions) ([]*types.ResultItem, error) {
	if len(items) == 0 {
		return items, nil
	}

	// Calculate weighted scores
	type scoredItem struct {
		item          *types.ResultItem
		weightedScore float64
	}

	scored := make([]scoredItem, len(items))
	for i, item := range items {
		// Weighted score = base score * source weight
		// Higher weight sources (user=1.0) get priority over lower (auto=0.6)
		weight := item.Weight
		if weight == 0 {
			weight = 0.6 // Default weight for items without weight
		}
		scored[i] = scoredItem{
			item:          item,
			weightedScore: item.Score * weight,
		}
	}

	// Sort by weighted score descending
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].weightedScore > scored[j].weightedScore
	})

	// Get top N
	topN := opts.TopN
	if topN <= 0 || topN > len(scored) {
		topN = len(scored)
	}

	result := make([]*types.ResultItem, topN)
	for i := 0; i < topN; i++ {
		result[i] = scored[i].item
	}

	return result, nil
}
