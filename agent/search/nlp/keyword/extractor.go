// Package keyword provides keyword extraction for web search optimization
// Supports three modes via uses.keyword configuration:
//   - "builtin": Simple frequency-based extraction (no external dependencies)
//   - "<assistant-id>": Delegate to an LLM-powered assistant for high-quality extraction
//   - "mcp:<server>.<tool>": Call external MCP tool
//
// For production use cases requiring high accuracy, use Agent or MCP mode.
package keyword

import (
	"strings"

	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/search/types"
)

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
// Returns a list of keywords optimized for search queries
func (e *Extractor) Extract(ctx *context.Context, content string, opts *types.KeywordOptions) ([]string, error) {
	// Merge options with config defaults
	mergedOpts := e.mergeOptions(opts)

	switch {
	case e.usesKeyword == "builtin" || e.usesKeyword == "":
		return e.builtinExtract(content, mergedOpts)
	case strings.HasPrefix(e.usesKeyword, "mcp:"):
		return e.mcpExtract(ctx, content, mergedOpts)
	default:
		// Assume it's an assistant ID for Agent mode
		return e.agentExtract(ctx, content, mergedOpts)
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

// builtinExtract uses simple frequency-based extraction
// This is a lightweight implementation with no external dependencies.
// For better results, use Agent or MCP mode.
func (e *Extractor) builtinExtract(content string, opts *types.KeywordOptions) ([]string, error) {
	extractor := NewBuiltinExtractor()
	return extractor.ExtractAsStrings(content, opts.MaxKeywords), nil
}

// agentExtract delegates to an LLM-powered assistant
// The assistant can understand context and extract semantically relevant keywords
func (e *Extractor) agentExtract(ctx *context.Context, content string, opts *types.KeywordOptions) ([]string, error) {
	provider := NewAgentProvider(e.usesKeyword)
	return provider.Extract(ctx, content, opts)
}

// mcpExtract calls an external MCP tool
// Format: "mcp:<server>.<tool>"
func (e *Extractor) mcpExtract(ctx *context.Context, content string, opts *types.KeywordOptions) ([]string, error) {
	mcpRef := strings.TrimPrefix(e.usesKeyword, "mcp:")
	provider, err := NewMCPProvider(mcpRef)
	if err != nil {
		// Fallback to builtin on invalid MCP format
		return e.builtinExtract(content, opts)
	}
	return provider.Extract(ctx, content, opts)
}
