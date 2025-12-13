package search

import (
	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/search/types"
)

// JSAPI implements context.SearchAPI interface
// Provides ctx.search.Web(), ctx.search.KB(), ctx.search.DB(), ctx.search.All(), ctx.search.Any(), ctx.search.Race()
type JSAPI struct {
	ctx      *context.Context
	searcher *Searcher
}

// NewJSAPI creates a new search JSAPI instance
func NewJSAPI(ctx *context.Context, config *types.Config, uses *Uses) *JSAPI {
	return &JSAPI{
		ctx:      ctx,
		searcher: New(config, uses),
	}
}

// Web executes web search
// Options:
//   - limit: int - max results (default: 10)
//   - sites: []string - restrict to specific sites
//   - time_range: string - "day", "week", "month", "year"
//   - rerank: map[string]interface{} - rerank options
func (api *JSAPI) Web(query string, opts map[string]interface{}) interface{} {
	req := api.buildRequest(types.SearchTypeWeb, query, opts)
	result, _ := api.searcher.Search(api.ctx, req)
	return result
}

// KB executes knowledge base search
// Options:
//   - collections: []string - collection IDs
//   - threshold: float64 - similarity threshold (0-1)
//   - limit: int - max results
//   - graph: bool - enable graph association
//   - rerank: map[string]interface{} - rerank options
func (api *JSAPI) KB(query string, opts map[string]interface{}) interface{} {
	req := api.buildRequest(types.SearchTypeKB, query, opts)
	result, _ := api.searcher.Search(api.ctx, req)
	return result
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
	req := api.buildRequest(types.SearchTypeDB, query, opts)
	result, _ := api.searcher.Search(api.ctx, req)
	return result
}

// All executes all searches and waits for all to complete (like Promise.all)
// Each request should have:
//   - type: string - "web", "kb", or "db"
//   - query: string - search query
//   - ... other type-specific options
func (api *JSAPI) All(requests []interface{}) []interface{} {
	reqs := api.parseRequests(requests)
	results, _ := api.searcher.All(api.ctx, reqs)
	return api.convertResults(results)
}

// Any returns as soon as any search succeeds with results (like Promise.any)
// Each request should have:
//   - type: string - "web", "kb", or "db"
//   - query: string - search query
//   - ... other type-specific options
func (api *JSAPI) Any(requests []interface{}) []interface{} {
	reqs := api.parseRequests(requests)
	results, _ := api.searcher.Any(api.ctx, reqs)
	return api.convertResults(results)
}

// Race returns as soon as any search completes (like Promise.race)
// Each request should have:
//   - type: string - "web", "kb", or "db"
//   - query: string - search query
//   - ... other type-specific options
func (api *JSAPI) Race(requests []interface{}) []interface{} {
	reqs := api.parseRequests(requests)
	results, _ := api.searcher.Race(api.ctx, reqs)
	return api.convertResults(results)
}

// buildRequest builds a Request from query and options
func (api *JSAPI) buildRequest(searchType types.SearchType, query string, opts map[string]interface{}) *types.Request {
	req := &types.Request{
		Type:   searchType,
		Query:  query,
		Source: types.SourceHook, // JSAPI calls are from hooks
	}

	if opts == nil {
		return req
	}

	// Common options
	if limit, ok := opts["limit"].(float64); ok {
		req.Limit = int(limit)
	} else if limit, ok := opts["limit"].(int); ok {
		req.Limit = limit
	}

	// Web-specific options
	if searchType == types.SearchTypeWeb {
		if sites, ok := opts["sites"].([]interface{}); ok {
			req.Sites = toStringSlice(sites)
		}
		if timeRange, ok := opts["time_range"].(string); ok {
			req.TimeRange = timeRange
		}
	}

	// KB-specific options
	if searchType == types.SearchTypeKB {
		if collections, ok := opts["collections"].([]interface{}); ok {
			req.Collections = toStringSlice(collections)
		}
		if threshold, ok := opts["threshold"].(float64); ok {
			req.Threshold = threshold
		}
		if graph, ok := opts["graph"].(bool); ok {
			req.Graph = graph
		}
	}

	// DB-specific options
	if searchType == types.SearchTypeDB {
		if models, ok := opts["models"].([]interface{}); ok {
			req.Models = toStringSlice(models)
		}
		if selectFields, ok := opts["select"].([]interface{}); ok {
			req.Select = toStringSlice(selectFields)
		}
		// Note: wheres and orders are more complex, handled by QueryDSL generator
	}

	// Rerank options
	if rerankOpts, ok := opts["rerank"].(map[string]interface{}); ok {
		req.Rerank = &types.RerankOptions{}
		if topN, ok := rerankOpts["top_n"].(float64); ok {
			req.Rerank.TopN = int(topN)
		} else if topN, ok := rerankOpts["top_n"].(int); ok {
			req.Rerank.TopN = topN
		}
	}

	return req
}

// parseRequests parses an array of request objects into typed Requests
func (api *JSAPI) parseRequests(requests []interface{}) []*types.Request {
	reqs := make([]*types.Request, 0, len(requests))
	for _, r := range requests {
		reqMap, ok := r.(map[string]interface{})
		if !ok {
			continue
		}

		// Get type
		typeStr, ok := reqMap["type"].(string)
		if !ok {
			continue
		}
		searchType := types.SearchType(typeStr)

		// Get query
		query, ok := reqMap["query"].(string)
		if !ok {
			continue
		}

		// Build request with remaining options
		req := api.buildRequest(searchType, query, reqMap)
		reqs = append(reqs, req)
	}
	return reqs
}

// convertResults converts typed Results to interface slice for JS
func (api *JSAPI) convertResults(results []*types.Result) []interface{} {
	out := make([]interface{}, len(results))
	for i, r := range results {
		out[i] = r
	}
	return out
}

// toStringSlice converts []interface{} to []string
func toStringSlice(arr []interface{}) []string {
	result := make([]string, 0, len(arr))
	for _, v := range arr {
		if s, ok := v.(string); ok {
			result = append(result, s)
		}
	}
	return result
}

// ConfigGetter is a function type that retrieves search config and uses for an assistant
type ConfigGetter func(assistantID string) (*types.Config, *Uses)

// configGetter is set by assistant package during initialization
var configGetter ConfigGetter

// SetJSAPIFactory sets the factory function for creating SearchAPI instances
// Called by assistant package during initialization
// getter: function to get search config and uses from assistant ID
func SetJSAPIFactory(getter ConfigGetter) {
	configGetter = getter
	context.SearchAPIFactory = func(ctx *context.Context) context.SearchAPI {
		var config *types.Config
		var uses *Uses
		if configGetter != nil && ctx.AssistantID != "" {
			config, uses = configGetter(ctx.AssistantID)
		}
		return NewJSAPI(ctx, config, uses)
	}
}
