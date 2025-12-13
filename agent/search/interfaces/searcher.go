package interfaces

import (
	"github.com/yaoapp/yao/agent/search/types"
)

// Searcher is the main interface exposed to external callers
type Searcher interface {
	// Search executes a single search request
	Search(req *types.Request) (*types.Result, error)

	// SearchMultiple executes multiple searches (potentially in parallel)
	SearchMultiple(reqs []*types.Request) ([]*types.Result, error)

	// BuildReferences converts search results to unified Reference format for LLM
	BuildReferences(results []*types.Result) []*types.Reference
}
