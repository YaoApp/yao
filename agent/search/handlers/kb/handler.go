package kb

import (
	"github.com/yaoapp/yao/agent/search/types"
)

// Handler implements KB search
type Handler struct {
	config *types.KBConfig // KB search configuration
}

// NewHandler creates a new KB search handler
func NewHandler(cfg *types.KBConfig) *Handler {
	return &Handler{config: cfg}
}

// Type returns the search type this handler supports
func (h *Handler) Type() types.SearchType {
	return types.SearchTypeKB
}

// Search executes vector search and optional graph association
// TODO: Implement actual search logic
func (h *Handler) Search(req *types.Request) (*types.Result, error) {
	// Skeleton implementation - returns empty result
	return &types.Result{
		Type:   types.SearchTypeKB,
		Query:  req.Query,
		Source: req.Source,
		Items:  []*types.ResultItem{},
		Total:  0,
	}, nil
}
