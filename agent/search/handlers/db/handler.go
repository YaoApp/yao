package db

import (
	"time"

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
// TODO: Implement actual QueryDSL generation and model query logic
func (h *Handler) Search(req *types.Request) (*types.Result, error) {
	start := time.Now()

	// Validate request
	if req.Query == "" {
		return &types.Result{
			Type:     types.SearchTypeDB,
			Query:    req.Query,
			Source:   req.Source,
			Items:    []*types.ResultItem{},
			Total:    0,
			Duration: time.Since(start).Milliseconds(),
			Error:    "query is required",
		}, nil
	}

	// Get models from request or config
	models := req.Models
	if len(models) == 0 && h.config != nil {
		models = h.config.Models
	}

	// If no models specified, return empty result
	if len(models) == 0 {
		return &types.Result{
			Type:     types.SearchTypeDB,
			Query:    req.Query,
			Source:   req.Source,
			Items:    []*types.ResultItem{},
			Total:    0,
			Duration: time.Since(start).Milliseconds(),
		}, nil
	}

	// Get max results
	maxResults := req.Limit
	if maxResults == 0 && h.config != nil && h.config.MaxResults > 0 {
		maxResults = h.config.MaxResults
	}
	if maxResults == 0 {
		maxResults = 20 // default
	}

	// TODO: Implement actual DB search
	// 1. Get model schemas for specified models
	// 2. Generate QueryDSL from natural language query using uses.querydsl mode:
	//    - "builtin": template-based generation
	//    - "<assistant-id>": delegate to LLM assistant
	//    - "mcp:<server>.<tool>": call external MCP tool
	// 3. Execute QueryDSL on each model
	// 4. Format results and return

	// For now, return empty result (skeleton)
	result := &types.Result{
		Type:     types.SearchTypeDB,
		Query:    req.Query,
		Source:   req.Source,
		Items:    []*types.ResultItem{},
		Total:    0,
		Duration: time.Since(start).Milliseconds(),
	}

	// Store maxResults for later use
	_ = maxResults

	return result, nil
}
