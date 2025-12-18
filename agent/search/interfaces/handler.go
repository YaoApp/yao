package interfaces

import (
	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/search/types"
)

// Handler defines the interface for search implementations
type Handler interface {
	// Type returns the search type this handler supports
	Type() types.SearchType

	// Search executes the search and returns results
	Search(req *types.Request) (*types.Result, error)
}

// ContextHandler extends Handler with context support
// Handlers that need context (e.g., DB handler for QueryDSL generation) should implement this
type ContextHandler interface {
	Handler

	// SearchWithContext executes the search with context and returns results
	SearchWithContext(ctx *context.Context, req *types.Request) (*types.Result, error)
}
