package web

import (
	"fmt"
	"strings"

	agentContext "github.com/yaoapp/yao/agent/context"
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
// ctx is optional and only required for agent mode
func (h *Handler) Search(req *types.Request) (*types.Result, error) {
	return h.SearchWithContext(nil, req)
}

// SearchWithContext executes web search with optional agent context
// ctx is required for agent mode, optional for builtin and MCP modes
func (h *Handler) SearchWithContext(ctx *agentContext.Context, req *types.Request) (*types.Result, error) {
	switch {
	case h.usesWeb == "builtin" || h.usesWeb == "":
		return h.builtinSearch(req)
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

// builtinSearch uses Tavily/Serper/SerpAPI directly
func (h *Handler) builtinSearch(req *types.Request) (*types.Result, error) {
	// Determine provider from config
	providerName := "tavily" // default
	if h.config != nil && h.config.Provider != "" {
		providerName = h.config.Provider
	}

	switch providerName {
	case "tavily":
		return NewTavilyProvider(h.config).Search(req)
	case "serper":
		// Serper (serper.dev) - POST request with X-API-KEY header
		return NewSerperProvider(h.config).Search(req)
	case "serpapi":
		// SerpAPI (serpapi.com) - GET request with api_key parameter
		return NewSerpAPIProvider(h.config).Search(req)
	default:
		return &types.Result{
			Type:   types.SearchTypeWeb,
			Query:  req.Query,
			Source: req.Source,
			Items:  []*types.ResultItem{},
			Total:  0,
			Error:  fmt.Sprintf("Unknown provider: %s (supported: tavily, serper, serpapi)", providerName),
		}, nil
	}
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
