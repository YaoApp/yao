package interfaces

import (
	"github.com/yaoapp/yao/agent/search/types"
)

// Reranker reorders search results by relevance
type Reranker interface {
	// Rerank reorders results based on query relevance
	Rerank(query string, items []*types.ResultItem, opts *types.RerankOptions) ([]*types.ResultItem, error)
}
