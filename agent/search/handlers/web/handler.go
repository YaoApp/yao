package web

import (
	"fmt"
	"strings"
	"time"

	agentContext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/search/types"
	"github.com/yaoapp/yao/tools/websearch"
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
// ctx is optional and only required for agent mode
func (h *Handler) Search(req *types.Request) (*types.Result, error) {
	return h.SearchWithContext(nil, req)
}

// SearchWithContext executes web search with optional agent context
// ctx is required for agent mode, optional for builtin and MCP modes
func (h *Handler) SearchWithContext(ctx *agentContext.Context, req *types.Request) (*types.Result, error) {
	switch {
	case h.usesWeb == "builtin" || h.usesWeb == "":
		return h.builtinSearch(ctx, req)
	case strings.HasPrefix(h.usesWeb, "mcp:"):
		return h.mcpSearch(req)
	default:
		// Agent mode: delegate to assistant for AI-powered search
		if ctx == nil {
			return &types.Result{
				Type:   types.SearchTypeWeb,
				Query:  req.Query,
				Source: req.Source,
				Items:  []*types.ResultItem{},
				Total:  0,
				Error:  "Agent mode requires context",
			}, nil
		}
		return h.agentSearch(ctx, req)
	}
}

// builtinSearch delegates to tools/websearch which reads Settings → ENV config.
func (h *Handler) builtinSearch(ctx *agentContext.Context, req *types.Request) (*types.Result, error) {
	startTime := time.Now()

	limit := req.Limit
	if limit <= 0 {
		limit = 10
	}

	var userID, teamID string
	if ctx != nil && ctx.Authorized != nil {
		userID = ctx.Authorized.UserID
		teamID = ctx.Authorized.TeamID
	}

	results := websearch.Search(req.Query, limit, userID, teamID)

	items := make([]*types.ResultItem, 0, len(results))
	for _, r := range results {
		items = append(items, &types.ResultItem{
			Type:    types.SearchTypeWeb,
			Title:   r.Title,
			Content: r.Content,
			URL:     r.URL,
			Score:   r.Score,
			Source:  req.Source,
		})
	}

	return &types.Result{
		Type:     types.SearchTypeWeb,
		Query:    req.Query,
		Source:   req.Source,
		Items:    items,
		Total:    len(items),
		Duration: time.Since(startTime).Milliseconds(),
	}, nil
}

// agentSearch delegates to an assistant for AI-powered search
func (h *Handler) agentSearch(ctx *agentContext.Context, req *types.Request) (*types.Result, error) {
	provider := NewAgentProvider(h.usesWeb)
	return provider.Search(ctx, req)
}

// mcpSearch calls external MCP tool
func (h *Handler) mcpSearch(req *types.Request) (*types.Result, error) {
	// Parse "mcp:server.tool"
	mcpRef := strings.TrimPrefix(h.usesWeb, "mcp:")

	provider, err := NewMCPProvider(mcpRef)
	if err != nil {
		return &types.Result{
			Type:   types.SearchTypeWeb,
			Query:  req.Query,
			Source: req.Source,
			Items:  []*types.ResultItem{},
			Total:  0,
			Error:  fmt.Sprintf("Invalid MCP format: %v", err),
		}, nil
	}

	return provider.Search(req)
}
