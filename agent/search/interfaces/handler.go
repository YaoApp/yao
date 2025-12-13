package interfaces

import (
	"github.com/yaoapp/yao/agent/search/types"
)

// Handler defines the interface for search implementations
type Handler interface {
	// Type returns the search type this handler supports
	Type() types.SearchType

	// Search executes the search and returns results
	Search(req *types.Request) (*types.Result, error)
}
