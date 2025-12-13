package rerank

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/yaoapp/gou/mcp"
	gouMCPTypes "github.com/yaoapp/gou/mcp/types"
	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/search/types"
)

// MCPProvider implements reranking by calling an MCP tool
type MCPProvider struct {
	serverID string // MCP server ID
	toolName string // Tool name
}

// NewMCPProvider creates a new MCP reranker
// mcpRef format: "server_id.tool_name"
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

// Rerank calls MCP tool to rerank items
func (p *MCPProvider) Rerank(ctx *context.Context, query string, items []*types.ResultItem, opts *types.RerankOptions) ([]*types.ResultItem, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context is required for MCP rerank")
	}

	// Get MCP client
	client, err := mcp.Select(p.serverID)
	if err != nil {
		return nil, fmt.Errorf("MCP server %s not found: %w", p.serverID, err)
	}

	// Build arguments for MCP tool
	args := map[string]interface{}{
		"query": query,
		"items": items,
		"top_n": opts.TopN,
	}

	// Call MCP tool
	result, err := client.CallTool(ctx.Context, p.toolName, args)
	if err != nil {
		return nil, fmt.Errorf("MCP tool call failed: %w", err)
	}

	// Parse result
	return p.parseResult(result, items, opts)
}

// parseResult extracts reranked items from MCP response
func (p *MCPProvider) parseResult(result *gouMCPTypes.CallToolResponse, originalItems []*types.ResultItem, opts *types.RerankOptions) ([]*types.ResultItem, error) {
	if result == nil || len(result.Content) == 0 {
		return originalItems, nil
	}

	// Build index map for quick lookup
	itemMap := make(map[string]*types.ResultItem)
	for _, item := range originalItems {
		if item.CitationID != "" {
			itemMap[item.CitationID] = item
		}
	}

	// Extract text content from MCP response
	var textContent string
	for _, content := range result.Content {
		if content.Type == gouMCPTypes.ToolContentTypeText && content.Text != "" {
			textContent = content.Text
			break
		}
	}

	if textContent == "" {
		return originalItems, nil
	}

	// Parse JSON response
	var response map[string]interface{}
	if err := json.Unmarshal([]byte(textContent), &response); err != nil {
		// Try parsing as array of IDs
		var orderList []string
		if err := json.Unmarshal([]byte(textContent), &orderList); err == nil {
			return p.reorderByIDs(orderList, itemMap, originalItems, opts)
		}
		return originalItems, nil
	}

	// Try "order" field (list of citation IDs)
	if order, ok := response["order"]; ok {
		if orderList := toStringSlice(order); len(orderList) > 0 {
			return p.reorderByIDs(orderList, itemMap, originalItems, opts)
		}
	}

	// Try "items" field
	if items, ok := response["items"]; ok {
		if itemsList := toItemsList(items); len(itemsList) > 0 {
			return p.reorderByItems(itemsList, itemMap, originalItems, opts)
		}
	}

	return originalItems, nil
}

// reorderByIDs reorders items based on list of citation IDs
func (p *MCPProvider) reorderByIDs(order []string, itemMap map[string]*types.ResultItem, originalItems []*types.ResultItem, opts *types.RerankOptions) ([]*types.ResultItem, error) {
	var result []*types.ResultItem

	// Add items in specified order
	for _, id := range order {
		if item, exists := itemMap[id]; exists {
			result = append(result, item)
			delete(itemMap, id)
		}
	}

	// Append remaining items
	for _, item := range originalItems {
		if _, exists := itemMap[item.CitationID]; exists {
			result = append(result, item)
		}
	}

	// Apply top N
	if opts.TopN > 0 && opts.TopN < len(result) {
		result = result[:opts.TopN]
	}

	return result, nil
}

// reorderByItems reorders items based on list of item references
func (p *MCPProvider) reorderByItems(itemsList []map[string]interface{}, itemMap map[string]*types.ResultItem, originalItems []*types.ResultItem, opts *types.RerankOptions) ([]*types.ResultItem, error) {
	var result []*types.ResultItem

	// Add items in specified order
	for _, respItem := range itemsList {
		if citationID, ok := respItem["citation_id"].(string); ok {
			if item, exists := itemMap[citationID]; exists {
				result = append(result, item)
				delete(itemMap, citationID)
			}
		}
	}

	// Append remaining items
	for _, item := range originalItems {
		if _, exists := itemMap[item.CitationID]; exists {
			result = append(result, item)
		}
	}

	// Apply top N
	if opts.TopN > 0 && opts.TopN < len(result) {
		result = result[:opts.TopN]
	}

	return result, nil
}
