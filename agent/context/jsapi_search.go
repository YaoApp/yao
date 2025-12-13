package context

import (
	"github.com/yaoapp/gou/runtime/v8/bridge"
	"rogchap.com/v8go"
)

// SearchAPI defines the search JSAPI interface for ctx.search.*
// This interface is defined here to avoid circular dependency between context and search packages.
// The actual implementation is in agent/search/jsapi.go
type SearchAPI interface {
	// Web executes web search
	// Returns *types.Result or error information
	Web(query string, opts map[string]interface{}) interface{}

	// KB executes knowledge base search
	// Returns *types.Result or error information
	KB(query string, opts map[string]interface{}) interface{}

	// DB executes database search
	// Returns *types.Result or error information
	DB(query string, opts map[string]interface{}) interface{}

	// Parallel search methods - inspired by JavaScript Promise
	// All waits for all searches to complete (like Promise.all)
	All(requests []interface{}) []interface{}
	// Any returns when any search succeeds with results (like Promise.any)
	Any(requests []interface{}) []interface{}
	// Race returns when any search completes (like Promise.race)
	Race(requests []interface{}) []interface{}
}

// SearchAPIFactory is a function type that creates a SearchAPI for a context
// This is set by the search package during initialization
var SearchAPIFactory func(ctx *Context) SearchAPI

// Search returns the search API for this context
// Returns nil if SearchAPIFactory is not set
func (ctx *Context) Search() SearchAPI {
	if SearchAPIFactory == nil {
		return nil
	}
	return SearchAPIFactory(ctx)
}

// newSearchObject creates a new search object with all search methods
// This is called from jsapi.go NewObject() to mount ctx.search
func (ctx *Context) newSearchObject(iso *v8go.Isolate) *v8go.ObjectTemplate {
	searchObj := v8go.NewObjectTemplate(iso)

	// Single search methods
	searchObj.Set("Web", ctx.searchWebMethod(iso))
	searchObj.Set("KB", ctx.searchKBMethod(iso))
	searchObj.Set("DB", ctx.searchDBMethod(iso))

	// Parallel search methods - inspired by JavaScript Promise
	searchObj.Set("All", ctx.searchAllMethod(iso))
	searchObj.Set("Any", ctx.searchAnyMethod(iso))
	searchObj.Set("Race", ctx.searchRaceMethod(iso))

	return searchObj
}

// searchWebMethod implements ctx.search.Web(query, options?)
// Options:
//   - limit: number - max results (default: 10)
//   - sites: string[] - restrict to specific sites
//   - time_range: string - "day", "week", "month", "year"
//   - rerank: { top_n: number } - rerank options
func (ctx *Context) searchWebMethod(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		v8ctx := info.Context()
		args := info.Args()

		// Validate arguments
		if len(args) < 1 {
			return bridge.JsException(v8ctx, "Web requires query parameter")
		}

		// Get query string
		if !args[0].IsString() {
			return bridge.JsException(v8ctx, "query must be a string")
		}
		query := args[0].String()

		// Parse options (optional)
		var opts map[string]interface{}
		if len(args) >= 2 && !args[1].IsUndefined() && !args[1].IsNull() {
			goVal, err := bridge.GoValue(args[1], v8ctx)
			if err != nil {
				return bridge.JsException(v8ctx, "invalid options: "+err.Error())
			}
			if optsMap, ok := goVal.(map[string]interface{}); ok {
				opts = optsMap
			}
		}

		// Get search API
		searchAPI := ctx.Search()
		if searchAPI == nil {
			return bridge.JsException(v8ctx, "search API not available")
		}

		// Execute search
		result := searchAPI.Web(query, opts)

		// Convert result to JS value
		jsVal, err := bridge.JsValue(v8ctx, result)
		if err != nil {
			return bridge.JsException(v8ctx, "failed to convert result: "+err.Error())
		}

		return jsVal
	})
}

// searchKBMethod implements ctx.search.KB(query, options?)
// Options:
//   - collections: string[] - collection IDs
//   - threshold: number - similarity threshold (0-1)
//   - limit: number - max results
//   - graph: boolean - enable graph association
//   - rerank: { top_n: number } - rerank options
func (ctx *Context) searchKBMethod(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		v8ctx := info.Context()
		args := info.Args()

		// Validate arguments
		if len(args) < 1 {
			return bridge.JsException(v8ctx, "KB requires query parameter")
		}

		// Get query string
		if !args[0].IsString() {
			return bridge.JsException(v8ctx, "query must be a string")
		}
		query := args[0].String()

		// Parse options (optional)
		var opts map[string]interface{}
		if len(args) >= 2 && !args[1].IsUndefined() && !args[1].IsNull() {
			goVal, err := bridge.GoValue(args[1], v8ctx)
			if err != nil {
				return bridge.JsException(v8ctx, "invalid options: "+err.Error())
			}
			if optsMap, ok := goVal.(map[string]interface{}); ok {
				opts = optsMap
			}
		}

		// Get search API
		searchAPI := ctx.Search()
		if searchAPI == nil {
			return bridge.JsException(v8ctx, "search API not available")
		}

		// Execute search
		result := searchAPI.KB(query, opts)

		// Convert result to JS value
		jsVal, err := bridge.JsValue(v8ctx, result)
		if err != nil {
			return bridge.JsException(v8ctx, "failed to convert result: "+err.Error())
		}

		return jsVal
	})
}

// searchDBMethod implements ctx.search.DB(query, options?)
// Options:
//   - models: string[] - model IDs
//   - wheres: Where[] - pre-defined filters (GOU QueryDSL Where format)
//   - orders: Order[] - sort orders (GOU QueryDSL Order format)
//   - select: string[] - fields to return
//   - limit: number - max results
//   - rerank: { top_n: number } - rerank options
func (ctx *Context) searchDBMethod(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		v8ctx := info.Context()
		args := info.Args()

		// Validate arguments
		if len(args) < 1 {
			return bridge.JsException(v8ctx, "DB requires query parameter")
		}

		// Get query string
		if !args[0].IsString() {
			return bridge.JsException(v8ctx, "query must be a string")
		}
		query := args[0].String()

		// Parse options (optional)
		var opts map[string]interface{}
		if len(args) >= 2 && !args[1].IsUndefined() && !args[1].IsNull() {
			goVal, err := bridge.GoValue(args[1], v8ctx)
			if err != nil {
				return bridge.JsException(v8ctx, "invalid options: "+err.Error())
			}
			if optsMap, ok := goVal.(map[string]interface{}); ok {
				opts = optsMap
			}
		}

		// Get search API
		searchAPI := ctx.Search()
		if searchAPI == nil {
			return bridge.JsException(v8ctx, "search API not available")
		}

		// Execute search
		result := searchAPI.DB(query, opts)

		// Convert result to JS value
		jsVal, err := bridge.JsValue(v8ctx, result)
		if err != nil {
			return bridge.JsException(v8ctx, "failed to convert result: "+err.Error())
		}

		return jsVal
	})
}

// searchAllMethod implements ctx.search.All(requests)
// Waits for all searches to complete (like Promise.all)
// Each request should have:
//   - type: string - "web", "kb", or "db"
//   - query: string - search query
//   - ... other type-specific options
func (ctx *Context) searchAllMethod(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		v8ctx := info.Context()
		args := info.Args()

		// Validate arguments
		if len(args) < 1 {
			return bridge.JsException(v8ctx, "All requires requests parameter")
		}

		// Parse requests array
		goVal, err := bridge.GoValue(args[0], v8ctx)
		if err != nil {
			return bridge.JsException(v8ctx, "invalid requests: "+err.Error())
		}

		requestsArray, ok := goVal.([]interface{})
		if !ok {
			return bridge.JsException(v8ctx, "requests must be an array")
		}

		// Get search API
		searchAPI := ctx.Search()
		if searchAPI == nil {
			return bridge.JsException(v8ctx, "search API not available")
		}

		// Execute parallel search
		results := searchAPI.All(requestsArray)

		// Convert results to JS value
		jsVal, err := bridge.JsValue(v8ctx, results)
		if err != nil {
			return bridge.JsException(v8ctx, "failed to convert results: "+err.Error())
		}

		return jsVal
	})
}

// searchAnyMethod implements ctx.search.Any(requests)
// Returns when any search succeeds with results (like Promise.any)
// Each request should have:
//   - type: string - "web", "kb", or "db"
//   - query: string - search query
//   - ... other type-specific options
func (ctx *Context) searchAnyMethod(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		v8ctx := info.Context()
		args := info.Args()

		// Validate arguments
		if len(args) < 1 {
			return bridge.JsException(v8ctx, "Any requires requests parameter")
		}

		// Parse requests array
		goVal, err := bridge.GoValue(args[0], v8ctx)
		if err != nil {
			return bridge.JsException(v8ctx, "invalid requests: "+err.Error())
		}

		requestsArray, ok := goVal.([]interface{})
		if !ok {
			return bridge.JsException(v8ctx, "requests must be an array")
		}

		// Get search API
		searchAPI := ctx.Search()
		if searchAPI == nil {
			return bridge.JsException(v8ctx, "search API not available")
		}

		// Execute parallel search
		results := searchAPI.Any(requestsArray)

		// Convert results to JS value
		jsVal, err := bridge.JsValue(v8ctx, results)
		if err != nil {
			return bridge.JsException(v8ctx, "failed to convert results: "+err.Error())
		}

		return jsVal
	})
}

// searchRaceMethod implements ctx.search.Race(requests)
// Returns when any search completes (like Promise.race)
// Each request should have:
//   - type: string - "web", "kb", or "db"
//   - query: string - search query
//   - ... other type-specific options
func (ctx *Context) searchRaceMethod(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		v8ctx := info.Context()
		args := info.Args()

		// Validate arguments
		if len(args) < 1 {
			return bridge.JsException(v8ctx, "Race requires requests parameter")
		}

		// Parse requests array
		goVal, err := bridge.GoValue(args[0], v8ctx)
		if err != nil {
			return bridge.JsException(v8ctx, "invalid requests: "+err.Error())
		}

		requestsArray, ok := goVal.([]interface{})
		if !ok {
			return bridge.JsException(v8ctx, "requests must be an array")
		}

		// Get search API
		searchAPI := ctx.Search()
		if searchAPI == nil {
			return bridge.JsException(v8ctx, "search API not available")
		}

		// Execute parallel search
		results := searchAPI.Race(requestsArray)

		// Convert results to JS value
		jsVal, err := bridge.JsValue(v8ctx, results)
		if err != nil {
			return bridge.JsException(v8ctx, "failed to convert results: "+err.Error())
		}

		return jsVal
	})
}
