package search

import (
	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/search/types"
)

// JSAPI implements context.SearchAPI interface
// Provides ctx.search.Web(), ctx.search.KB(), ctx.search.DB(), ctx.search.Parallel()
type JSAPI struct {
	ctx    *context.Context
	config *types.Config
	uses   *Uses
}

// NewJSAPI creates a new search JSAPI instance
func NewJSAPI(ctx *context.Context, config *types.Config, uses *Uses) *JSAPI {
	return &JSAPI{
		ctx:    ctx,
		config: config,
		uses:   uses,
	}
}

// Web executes web search
// Options:
//   - limit: int - max results (default: 10)
//   - sites: []string - restrict to specific sites
//   - time_range: string - "day", "week", "month", "year"
//   - rerank: map[string]interface{} - rerank options
func (api *JSAPI) Web(query string, opts map[string]interface{}) interface{} {
	// TODO: Implement web search
	// 1. Build Request from query and opts
	// 2. Call web handler
	// 3. Return Result or error
	return &types.Result{
		Type:  types.SearchTypeWeb,
		Query: query,
		Error: "not implemented",
	}
}

// KB executes knowledge base search
// Options:
//   - collections: []string - collection IDs
//   - threshold: float64 - similarity threshold (0-1)
//   - limit: int - max results
//   - graph: bool - enable graph association
//   - rerank: map[string]interface{} - rerank options
func (api *JSAPI) KB(query string, opts map[string]interface{}) interface{} {
	// TODO: Implement KB search
	// 1. Build Request from query and opts
	// 2. Call KB handler
	// 3. Return Result or error
	return &types.Result{
		Type:  types.SearchTypeKB,
		Query: query,
		Error: "not implemented",
	}
}

// DB executes database search
// Options:
//   - models: []string - model IDs
//   - wheres: []map[string]interface{} - pre-defined filters (GOU QueryDSL Where format)
//   - orders: []map[string]interface{} - sort orders (GOU QueryDSL Order format)
//   - select: []string - fields to return
//   - limit: int - max results
//   - rerank: map[string]interface{} - rerank options
func (api *JSAPI) DB(query string, opts map[string]interface{}) interface{} {
	// TODO: Implement DB search
	// 1. Build Request from query and opts
	// 2. Call DB handler
	// 3. Return Result or error
	return &types.Result{
		Type:  types.SearchTypeDB,
		Query: query,
		Error: "not implemented",
	}
}

// Parallel executes multiple searches in parallel
// Each request should have:
//   - type: string - "web", "kb", or "db"
//   - query: string - search query
//   - ... other type-specific options
func (api *JSAPI) Parallel(requests []interface{}) []interface{} {
	// TODO: Implement parallel search
	// 1. Parse requests into []Request
	// 2. Call SearchMultiple
	// 3. Return []Result
	results := make([]interface{}, len(requests))
	for i := range requests {
		results[i] = &types.Result{
			Error: "not implemented",
		}
	}
	return results
}

// init registers the JSAPI factory with context package
func init() {
	// Note: The actual factory is set by assistant package during initialization
	// This avoids circular dependency: context -> search -> context
	// See: assistant/assistant.go init()
}

// SetJSAPIFactory sets the factory function for creating SearchAPI instances
// Called by assistant package during initialization
func SetJSAPIFactory() {
	context.SearchAPIFactory = func(ctx *context.Context) context.SearchAPI {
		// Get config and uses from context or use defaults
		// TODO: Get actual config from assistant
		return NewJSAPI(ctx, nil, nil)
	}
}
