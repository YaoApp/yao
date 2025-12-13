package interfaces

import (
	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/search/types"
)

// Reranker reorders search results by relevance
type Reranker interface {
	// Rerank reorders results based on query relevance
	// ctx is required for Agent and MCP modes, can be nil for builtin mode
	Rerank(ctx *context.Context, query string, items []*types.ResultItem, opts *types.RerankOptions) ([]*types.ResultItem, error)
}
