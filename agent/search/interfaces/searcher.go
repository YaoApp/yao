package interfaces

import (
	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/search/types"
)

// Searcher is the main interface exposed to external callers
type Searcher interface {
	// Search executes a single search request
	Search(ctx *context.Context, req *types.Request) (*types.Result, error)

	// Parallel search methods - inspired by JavaScript Promise
	// All waits for all searches to complete (like Promise.all)
	All(ctx *context.Context, reqs []*types.Request) ([]*types.Result, error)
	// Any returns when any search succeeds with results (like Promise.any)
	Any(ctx *context.Context, reqs []*types.Request) ([]*types.Result, error)
	// Race returns when any search completes (like Promise.race)
	Race(ctx *context.Context, reqs []*types.Request) ([]*types.Result, error)

	// BuildReferences converts search results to unified Reference format for LLM
	BuildReferences(results []*types.Result) []*types.Reference
}
