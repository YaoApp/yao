package keyword

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/yaoapp/gou/mcp"
	gouMCPTypes "github.com/yaoapp/gou/mcp/types"
	agentContext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/search/types"
)

// MCPProvider delegates keyword extraction to an MCP tool
type MCPProvider struct {
	serverID string // MCP server ID
	toolName string // Tool name to call
}

// NewMCPProvider creates a new MCP-based keyword extractor
// mcpRef format: "server.tool" (e.g., "nlp.extract_keywords")
func NewMCPProvider(mcpRef string) (*MCPProvider, error) {
	parts := strings.SplitN(mcpRef, ".", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid MCP format, expected 'server.tool', got '%s'", mcpRef)
	}
	return &MCPProvider{
		serverID: parts[0],
		toolName: parts[1],
	}, nil
}

// Extract extracts keywords by calling the MCP tool
func (p *MCPProvider) Extract(ctx *agentContext.Context, content string, opts *types.KeywordOptions) ([]types.Keyword, error) {
	// Get MCP client
	client, err := mcp.Select(p.serverID)
	if err != nil {
		return nil, fmt.Errorf("MCP server '%s' not found: %w", p.serverID, err)
	}

	// Build arguments for the MCP tool
	arguments := map[string]interface{}{
		"content":      content,
		"max_keywords": opts.MaxKeywords,
		"language":     opts.Language,
	}

	// Call the MCP tool (ctx embeds context.Context)
	callResult, err := client.CallTool(ctx, p.toolName, arguments)
	if err != nil {
		return nil, fmt.Errorf("MCP tool call failed: %w", err)
	}

	// Parse the result
	return p.parseResult(callResult)
}

// parseResult extracts keywords from the MCP tool response
func (p *MCPProvider) parseResult(result *gouMCPTypes.CallToolResponse) ([]types.Keyword, error) {
	if result == nil {
		return []types.Keyword{}, nil
	}

	// Check for errors in result
	if result.IsError {
		errMsg := "MCP tool returned error"
		if len(result.Content) > 0 && result.Content[0].Text != "" {
			errMsg = result.Content[0].Text
		}
		return nil, fmt.Errorf("%s", errMsg)
	}

	// Parse content - expect JSON data with "keywords" field
	if len(result.Content) == 0 {
		return []types.Keyword{}, nil
	}

	// Try to extract keywords from content
	for _, content := range result.Content {
		// Check text content type
		if content.Type == gouMCPTypes.ToolContentTypeText && content.Text != "" {
			// Try to parse as JSON
			var data map[string]interface{}
			if err := json.Unmarshal([]byte(content.Text), &data); err == nil {
				// Look for "keywords" field
				if kw, ok := data["keywords"]; ok {
					return p.extractKeywordsFromValue(kw)
				}
			}

			// Try to parse as direct array of keywords
			var keywords []types.Keyword
			if err := json.Unmarshal([]byte(content.Text), &keywords); err == nil {
				return keywords, nil
			}
		}
	}

	return []types.Keyword{}, nil
}

// extractKeywordsFromValue extracts Keyword array from various types
func (p *MCPProvider) extractKeywordsFromValue(v interface{}) ([]types.Keyword, error) {
	switch kw := v.(type) {
	case []types.Keyword:
		return kw, nil
	case []interface{}:
		keywords := make([]types.Keyword, 0, len(kw))
		for _, item := range kw {
			switch v := item.(type) {
			case map[string]interface{}:
				// Handle {k: "keyword", w: 0.9} format
				k, _ := v["k"].(string)
				w, _ := v["w"].(float64)
				if k != "" {
					if w == 0 {
						w = 0.5 // Default weight
					}
					keywords = append(keywords, types.Keyword{K: k, W: w})
				}
			case string:
				// Plain string, use default weight
				if v != "" {
					keywords = append(keywords, types.Keyword{K: v, W: 0.5})
				}
			}
		}
		return keywords, nil
	case string:
		var keywords []types.Keyword
		if err := json.Unmarshal([]byte(kw), &keywords); err == nil {
			return keywords, nil
		}
		return []types.Keyword{{K: kw, W: 0.5}}, nil
	}
	return []types.Keyword{}, nil
}
