package web

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/yaoapp/gou/mcp"
	gouMCPTypes "github.com/yaoapp/gou/mcp/types"
	"github.com/yaoapp/yao/agent/search/types"
)

// MCPProvider implements web search using MCP tool
type MCPProvider struct {
	serverID string // MCP server ID (e.g., "search")
	toolName string // MCP tool name (e.g., "web_search")
}

// NewMCPProvider creates a new MCP provider from "mcp:server.tool" format
func NewMCPProvider(mcpRef string) (*MCPProvider, error) {
	// Parse "server.tool" format
	parts := strings.SplitN(mcpRef, ".", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid MCP format, expected 'server.tool', got '%s'", mcpRef)
	}

	return &MCPProvider{
		serverID: parts[0],
		toolName: parts[1],
	}, nil
}

// Search executes web search via MCP tool
func (p *MCPProvider) Search(req *types.Request) (*types.Result, error) {
	startTime := time.Now()

	// Select MCP client
	client, err := mcp.Select(p.serverID)
	if err != nil {
		return &types.Result{
			Type:     types.SearchTypeWeb,
			Query:    req.Query,
			Source:   req.Source,
			Items:    []*types.ResultItem{},
			Total:    0,
			Duration: time.Since(startTime).Milliseconds(),
			Error:    fmt.Sprintf("MCP client '%s' not found: %v", p.serverID, err),
		}, nil
	}

	// Build MCP tool arguments
	args := map[string]interface{}{
		"query": req.Query,
	}

	if req.Limit > 0 {
		args["limit"] = req.Limit
	}

	if len(req.Sites) > 0 {
		args["sites"] = req.Sites
	}

	if req.TimeRange != "" {
		args["time_range"] = req.TimeRange
	}

	// Call MCP tool
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := client.CallTool(ctx, p.toolName, args)
	if err != nil {
		return &types.Result{
			Type:     types.SearchTypeWeb,
			Query:    req.Query,
			Source:   req.Source,
			Items:    []*types.ResultItem{},
			Total:    0,
			Duration: time.Since(startTime).Milliseconds(),
			Error:    fmt.Sprintf("MCP tool call failed: %v", err),
		}, nil
	}

	// Parse MCP result
	items, total, parseErr := p.parseResult(result, req.Source)
	if parseErr != "" {
		return &types.Result{
			Type:     types.SearchTypeWeb,
			Query:    req.Query,
			Source:   req.Source,
			Items:    []*types.ResultItem{},
			Total:    0,
			Duration: time.Since(startTime).Milliseconds(),
			Error:    parseErr,
		}, nil
	}

	return &types.Result{
		Type:     types.SearchTypeWeb,
		Query:    req.Query,
		Source:   req.Source,
		Items:    items,
		Total:    total,
		Duration: time.Since(startTime).Milliseconds(),
	}, nil
}

// parseResult parses MCP tool result into search result items
func (p *MCPProvider) parseResult(result *gouMCPTypes.CallToolResponse, source types.SourceType) ([]*types.ResultItem, int, string) {
	if result == nil {
		return nil, 0, "MCP returned nil result"
	}

	// Check for errors in result
	if result.IsError {
		errMsg := "MCP tool returned error"
		if len(result.Content) > 0 && result.Content[0].Text != "" {
			errMsg = result.Content[0].Text
		}
		return nil, 0, errMsg
	}

	// Parse content - expect JSON data
	if len(result.Content) == 0 {
		return []*types.ResultItem{}, 0, ""
	}

	// Try to extract data from content
	var data map[string]interface{}

	for _, content := range result.Content {
		// Check text content type
		if content.Type == gouMCPTypes.ToolContentTypeText && content.Text != "" {
			// Try to parse as JSON
			if parsed, ok := parseJSON(content.Text); ok {
				data = parsed
				break
			}
		}
	}

	if data == nil {
		return []*types.ResultItem{}, 0, ""
	}

	// Extract items from data
	items := []*types.ResultItem{}
	total := 0

	if itemsData, ok := data["items"].([]interface{}); ok {
		for _, itemData := range itemsData {
			if item, ok := itemData.(map[string]interface{}); ok {
				resultItem := &types.ResultItem{
					Type:   types.SearchTypeWeb,
					Source: source,
				}

				if title, ok := item["title"].(string); ok {
					resultItem.Title = title
				}
				if content, ok := item["content"].(string); ok {
					resultItem.Content = content
				}
				if url, ok := item["url"].(string); ok {
					resultItem.URL = url
				}
				if score, ok := item["score"].(float64); ok {
					resultItem.Score = score
				}

				items = append(items, resultItem)
			}
		}
	}

	if totalVal, ok := data["total"].(float64); ok {
		total = int(totalVal)
	} else {
		total = len(items)
	}

	return items, total, ""
}

// parseJSON attempts to parse a string as JSON
func parseJSON(s string) (map[string]interface{}, bool) {
	// Simple JSON detection - if it starts with { and ends with }
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "{") || !strings.HasSuffix(s, "}") {
		return nil, false
	}

	// Use encoding/json for parsing
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(s), &result); err != nil {
		return nil, false
	}

	return result, true
}
