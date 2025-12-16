package querydsl

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/yaoapp/gou/mcp"
	gouMCPTypes "github.com/yaoapp/gou/mcp/types"
	"github.com/yaoapp/gou/query/gou"
	agentContext "github.com/yaoapp/yao/agent/context"
)

// MCPProvider delegates QueryDSL generation to an MCP tool
type MCPProvider struct {
	serverID string // MCP server ID
	toolName string // Tool name to call
}

// NewMCPProvider creates a new MCP-based QueryDSL generator
// mcpRef format: "server.tool" (e.g., "nlp.generate_querydsl")
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

// Generate generates QueryDSL by calling the MCP tool
func (p *MCPProvider) Generate(ctx *agentContext.Context, input *Input) (*Result, error) {
	// Get MCP client
	client, err := mcp.Select(p.serverID)
	if err != nil {
		return nil, fmt.Errorf("MCP server '%s' not found: %w", p.serverID, err)
	}

	// Build arguments for the MCP tool
	// Note: model metadata is loaded internally by the MCP tool
	arguments := map[string]interface{}{
		"query":  input.Query,
		"models": input.ModelIDs,
		"limit":  input.Limit,
	}

	// Add optional fields
	if len(input.Wheres) > 0 {
		arguments["wheres"] = input.Wheres
	}
	if len(input.Orders) > 0 {
		arguments["orders"] = input.Orders
	}
	if len(input.AllowedFields) > 0 {
		arguments["allowed_fields"] = input.AllowedFields
	}
	if len(input.ExtraParams) > 0 {
		arguments["extra"] = input.ExtraParams
	}

	// Call the MCP tool (ctx embeds context.Context)
	callResult, err := client.CallTool(ctx, p.toolName, arguments)
	if err != nil {
		return nil, fmt.Errorf("MCP tool call failed: %w", err)
	}

	// Parse the result
	return p.parseResult(callResult)
}

// parseResult extracts QueryDSL from the MCP tool response
func (p *MCPProvider) parseResult(result *gouMCPTypes.CallToolResponse) (*Result, error) {
	if result == nil {
		return &Result{}, nil
	}

	// Check for errors in result
	if result.IsError {
		errMsg := "MCP tool returned error"
		if len(result.Content) > 0 && result.Content[0].Text != "" {
			errMsg = result.Content[0].Text
		}
		return nil, fmt.Errorf("%s", errMsg)
	}

	// Parse content - expect JSON data with "dsl" field
	if len(result.Content) == 0 {
		return &Result{}, nil
	}

	genResult := &Result{}

	// Try to extract QueryDSL from content
	for _, content := range result.Content {
		// Check text content type
		if content.Type == gouMCPTypes.ToolContentTypeText && content.Text != "" {
			// Try to parse as JSON
			var data map[string]interface{}
			if err := json.Unmarshal([]byte(content.Text), &data); err == nil {
				// Look for "dsl" field
				if dsl, ok := data["dsl"]; ok {
					genResult.DSL = p.extractDSL(dsl)
				}
				if explain, ok := data["explain"].(string); ok {
					genResult.Explain = explain
				}
				if warnings, ok := data["warnings"]; ok {
					genResult.Warnings = p.extractWarnings(warnings)
				}
				return genResult, nil
			}

			// Try to parse as direct QueryDSL
			var dsl gou.QueryDSL
			if err := json.Unmarshal([]byte(content.Text), &dsl); err == nil {
				genResult.DSL = &dsl
				return genResult, nil
			}
		}
	}

	return genResult, nil
}

// extractDSL converts interface{} to gou.QueryDSL
func (p *MCPProvider) extractDSL(v interface{}) *gou.QueryDSL {
	if v == nil {
		return nil
	}

	// Marshal and unmarshal to gou.QueryDSL
	jsonBytes, err := json.Marshal(v)
	if err != nil {
		return nil
	}

	var dsl gou.QueryDSL
	if err := json.Unmarshal(jsonBytes, &dsl); err != nil {
		return nil
	}

	return &dsl
}

// extractWarnings extracts warnings array from various types
func (p *MCPProvider) extractWarnings(v interface{}) []string {
	switch w := v.(type) {
	case []string:
		return w
	case []interface{}:
		warnings := make([]string, 0, len(w))
		for _, item := range w {
			if s, ok := item.(string); ok {
				warnings = append(warnings, s)
			}
		}
		return warnings
	case string:
		return []string{w}
	}
	return nil
}
