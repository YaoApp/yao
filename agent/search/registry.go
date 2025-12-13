package search

import (
	"github.com/yaoapp/yao/agent/search/interfaces"
	"github.com/yaoapp/yao/agent/search/types"
)

// Registry manages search handlers
type Registry struct {
	handlers map[types.SearchType]interfaces.Handler
}

// NewRegistry creates a new handler registry
func NewRegistry() *Registry {
	return &Registry{
		handlers: make(map[types.SearchType]interfaces.Handler),
	}
}

// Register registers a handler for a search type
func (r *Registry) Register(handler interfaces.Handler) {
	r.handlers[handler.Type()] = handler
}

// Get returns the handler for a search type
func (r *Registry) Get(t types.SearchType) (interfaces.Handler, bool) {
	h, ok := r.handlers[t]
	return h, ok
}
