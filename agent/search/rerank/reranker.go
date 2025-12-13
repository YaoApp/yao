// Package rerank provides result reranking for search module
// Supports three modes via uses.rerank configuration:
//   - "builtin": Simple score-based sorting (no external dependencies)
//   - "<assistant-id>": Delegate to an LLM-powered assistant for semantic reranking
//   - "mcp:<server>.<tool>": Call external MCP tool
//
// For production use cases requiring high accuracy, use Agent or MCP mode.
package rerank

import (
	"strings"

	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/search/types"
)

// Reranker reorders search results by relevance
// Mode is determined by uses.rerank configuration
type Reranker struct {
	usesRerank string              // "builtin", "<assistant-id>", "mcp:<server>.<tool>"
	config     *types.RerankConfig // Rerank options
}

// NewReranker creates a new reranker
// usesRerank: value from uses.rerank config
// cfg: rerank options from search config
func NewReranker(usesRerank string, cfg *types.RerankConfig) *Reranker {
	return &Reranker{
		usesRerank: usesRerank,
		config:     cfg,
	}
}

// Rerank reorders results based on configured mode
// Returns reordered items, potentially truncated to top N
func (r *Reranker) Rerank(ctx *context.Context, query string, items []*types.ResultItem, opts *types.RerankOptions) ([]*types.ResultItem, error) {
	if len(items) == 0 {
		return items, nil
	}

	// Merge options with config defaults
	mergedOpts := r.mergeOptions(opts)

	switch {
	case r.usesRerank == "builtin" || r.usesRerank == "":
		return r.builtinRerank(query, items, mergedOpts)
	case strings.HasPrefix(r.usesRerank, "mcp:"):
		return r.mcpRerank(ctx, query, items, mergedOpts)
	default:
		// Assume it's an assistant ID for Agent mode
		return r.agentRerank(ctx, query, items, mergedOpts)
	}
}

// mergeOptions merges runtime options with config defaults
func (r *Reranker) mergeOptions(opts *types.RerankOptions) *types.RerankOptions {
	result := &types.RerankOptions{
		TopN: 10, // default
	}

	// Apply config defaults
	if r.config != nil {
		if r.config.TopN > 0 {
			result.TopN = r.config.TopN
		}
	}

	// Apply runtime options (highest priority)
	if opts != nil {
		if opts.TopN > 0 {
			result.TopN = opts.TopN
		}
	}

	return result
}

// builtinRerank uses simple score-based sorting
func (r *Reranker) builtinRerank(query string, items []*types.ResultItem, opts *types.RerankOptions) ([]*types.ResultItem, error) {
	reranker := NewBuiltinReranker()
	return reranker.Rerank(query, items, opts)
}

// agentRerank delegates to an LLM-powered assistant
func (r *Reranker) agentRerank(ctx *context.Context, query string, items []*types.ResultItem, opts *types.RerankOptions) ([]*types.ResultItem, error) {
	provider := NewAgentProvider(r.usesRerank)
	return provider.Rerank(ctx, query, items, opts)
}

// mcpRerank calls an external MCP tool
func (r *Reranker) mcpRerank(ctx *context.Context, query string, items []*types.ResultItem, opts *types.RerankOptions) ([]*types.ResultItem, error) {
	mcpRef := strings.TrimPrefix(r.usesRerank, "mcp:")
	provider, err := NewMCPProvider(mcpRef)
	if err != nil {
		// Fallback to builtin on invalid MCP format
		return r.builtinRerank(query, items, opts)
	}
	return provider.Rerank(ctx, query, items, opts)
}
