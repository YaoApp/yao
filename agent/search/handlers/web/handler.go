package web

import (
	"github.com/yaoapp/yao/agent/search/types"
)

// Handler implements web search
type Handler struct {
	usesWeb string           // "builtin", "<assistant-id>", "mcp:<server>.<tool>"
	config  *types.WebConfig // Web search configuration
}

// NewHandler creates a new web search handler
func NewHandler(usesWeb string, cfg *types.WebConfig) *Handler {
	return &Handler{usesWeb: usesWeb, config: cfg}
}

// Type returns the search type this handler supports
func (h *Handler) Type() types.SearchType {
	return types.SearchTypeWeb
}

// Search executes web search based on uses.web mode
// TODO: Implement actual search logic
func (h *Handler) Search(req *types.Request) (*types.Result, error) {
	// Skeleton implementation - returns empty result
	return &types.Result{
		Type:   types.SearchTypeWeb,
		Query:  req.Query,
		Source: req.Source,
		Items:  []*types.ResultItem{},
		Total:  0,
	}, nil
}
