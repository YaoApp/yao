package db

import (
	"github.com/yaoapp/yao/agent/search/types"
)

// Handler implements DB search
type Handler struct {
	usesQueryDSL string          // "builtin", "<assistant-id>", "mcp:<server>.<tool>"
	config       *types.DBConfig // DB search configuration
}

// NewHandler creates a new DB search handler
func NewHandler(usesQueryDSL string, cfg *types.DBConfig) *Handler {
	return &Handler{usesQueryDSL: usesQueryDSL, config: cfg}
}

// Type returns the search type this handler supports
func (h *Handler) Type() types.SearchType {
	return types.SearchTypeDB
}

// Search converts NL to QueryDSL and executes
// TODO: Implement actual search logic
func (h *Handler) Search(req *types.Request) (*types.Result, error) {
	// Skeleton implementation - returns empty result
	return &types.Result{
		Type:   types.SearchTypeDB,
		Query:  req.Query,
		Source: req.Source,
		Items:  []*types.ResultItem{},
		Total:  0,
	}, nil
}
