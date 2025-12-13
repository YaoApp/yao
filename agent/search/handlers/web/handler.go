package web

import (
	"fmt"
	"strings"

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
func (h *Handler) Search(req *types.Request) (*types.Result, error) {
	switch {
	case h.usesWeb == "builtin" || h.usesWeb == "":
		return h.builtinSearch(req)
	case strings.HasPrefix(h.usesWeb, "mcp:"):
		return h.mcpSearch(req)
	default:
		// Agent mode: delegate to assistant for AI-powered search
		return h.agentSearch(req)
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
func (h *Handler) agentSearch(req *types.Request) (*types.Result, error) {
	// TODO: Implement agent mode
	// 1. Call assistant with search request
	// 2. Assistant understands intent, generates optimized queries
	// 3. Assistant executes searches (may call builtin internally)
	// 4. Assistant analyzes and returns structured results
	return &types.Result{
		Type:   types.SearchTypeWeb,
		Query:  req.Query,
		Source: req.Source,
		Items:  []*types.ResultItem{},
		Total:  0,
		Error:  "Agent mode not yet implemented",
	}, nil
}

// mcpSearch calls external MCP tool
func (h *Handler) mcpSearch(req *types.Request) (*types.Result, error) {
	// TODO: Implement MCP mode
	// Parse "mcp:server.tool"
	mcpRef := strings.TrimPrefix(h.usesWeb, "mcp:")
	parts := strings.SplitN(mcpRef, ".", 2)
	if len(parts) != 2 {
		return &types.Result{
			Type:   types.SearchTypeWeb,
			Query:  req.Query,
			Source: req.Source,
			Items:  []*types.ResultItem{},
			Total:  0,
			Error:  fmt.Sprintf("Invalid MCP format, expected 'mcp:server.tool', got '%s'", h.usesWeb),
		}, nil
	}
	// serverID, toolName := parts[0], parts[1]
	// Call MCP tool
	return &types.Result{
		Type:   types.SearchTypeWeb,
		Query:  req.Query,
		Source: req.Source,
		Items:  []*types.ResultItem{},
		Total:  0,
		Error:  "MCP mode not yet implemented",
	}, nil
}
