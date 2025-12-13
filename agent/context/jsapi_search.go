package context

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
