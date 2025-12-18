// Package keyword provides keyword extraction for web search optimization
// Supports three modes via uses.keyword configuration:
//   - "builtin" or "": Uses __yao.keyword system agent (LLM-powered)
//   - "<assistant-id>": Delegate to a custom LLM-powered assistant
//   - "mcp:<server>.<tool>": Call external MCP tool
package keyword

import (
	"fmt"
	"strings"

	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/search/types"
)

// SystemKeywordAgent is the default system agent for keyword extraction
const SystemKeywordAgent = "__yao.keyword"

// Extractor extracts keywords from text
// Mode is determined by uses.keyword configuration
type Extractor struct {
	usesKeyword string               // "builtin", "<assistant-id>", "mcp:<server>.<tool>"
	config      *types.KeywordConfig // Keyword extraction options
}

// NewExtractor creates a new keyword extractor
// usesKeyword: value from uses.keyword config
// cfg: keyword extraction options from search config
func NewExtractor(usesKeyword string, cfg *types.KeywordConfig) *Extractor {
	return &Extractor{
		usesKeyword: usesKeyword,
		config:      cfg,
	}
}

// Extract extracts keywords from content based on configured mode
// Returns a list of keywords with weights optimized for search queries
func (e *Extractor) Extract(ctx *context.Context, content string, opts *types.KeywordOptions) ([]types.Keyword, error) {
	// Merge options with config defaults
	mergedOpts := e.mergeOptions(opts)

	switch {
	case e.usesKeyword == "builtin" || e.usesKeyword == "":
		// Use system keyword agent
		return e.agentExtract(ctx, content, SystemKeywordAgent, mergedOpts)
	case strings.HasPrefix(e.usesKeyword, "mcp:"):
		return e.mcpExtract(ctx, content, mergedOpts)
	default:
		// Assume it's an assistant ID for Agent mode
		return e.agentExtract(ctx, content, e.usesKeyword, mergedOpts)
	}
}

// mergeOptions merges runtime options with config defaults
func (e *Extractor) mergeOptions(opts *types.KeywordOptions) *types.KeywordOptions {
	result := &types.KeywordOptions{
		MaxKeywords: 10,     // default
		Language:    "auto", // default
	}

	// Apply config defaults
	if e.config != nil {
		if e.config.MaxKeywords > 0 {
			result.MaxKeywords = e.config.MaxKeywords
		}
		if e.config.Language != "" {
			result.Language = e.config.Language
		}
	}

	// Apply runtime options (highest priority)
	if opts != nil {
		if opts.MaxKeywords > 0 {
			result.MaxKeywords = opts.MaxKeywords
		}
		if opts.Language != "" {
			result.Language = opts.Language
		}
	}

	return result
}

// agentExtract delegates to an LLM-powered assistant
// The assistant can understand context and extract semantically relevant keywords
func (e *Extractor) agentExtract(ctx *context.Context, content string, agentID string, opts *types.KeywordOptions) ([]types.Keyword, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context is required for keyword extraction")
	}
	provider := NewAgentProvider(agentID)
	return provider.Extract(ctx, content, opts)
}

// mcpExtract calls an external MCP tool
// Format: "mcp:<server>.<tool>"
func (e *Extractor) mcpExtract(ctx *context.Context, content string, opts *types.KeywordOptions) ([]types.Keyword, error) {
	mcpRef := strings.TrimPrefix(e.usesKeyword, "mcp:")
	provider, err := NewMCPProvider(mcpRef)
	if err != nil {
		// Fallback to system agent on invalid MCP format
		return e.agentExtract(ctx, content, SystemKeywordAgent, e.mergeOptions(nil))
	}
	return provider.Extract(ctx, content, opts)
}
