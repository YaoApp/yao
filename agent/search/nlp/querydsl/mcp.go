package querydsl

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/yaoapp/gou/mcp"
	gouMCPTypes "github.com/yaoapp/gou/mcp/types"
	"github.com/yaoapp/gou/query/gou"
	"github.com/yaoapp/gou/query/linter"
	agentContext "github.com/yaoapp/yao/agent/context"
)

// MaxRetries is the maximum number of retry attempts for QueryDSL generation
const MaxRetries = 3

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

// Generate generates QueryDSL by calling the MCP tool with retry and lint validation
func (p *MCPProvider) Generate(ctx *agentContext.Context, input *Input) (*Result, error) {
	// Get MCP client
	client, err := mcp.Select(p.serverID)
	if err != nil {
		return nil, fmt.Errorf("MCP server '%s' not found: %w", p.serverID, err)
	}

	var lastError error
	var lastLintErrors string

	for attempt := 1; attempt <= MaxRetries; attempt++ {
		// Build arguments for the MCP tool
		arguments := p.buildArguments(input, attempt, lastLintErrors)

		// Call the MCP tool
		callResult, err := client.CallTool(ctx, p.toolName, arguments)
		if err != nil {
			lastError = fmt.Errorf("MCP tool call failed: %w", err)
			continue
		}

		// Parse the result
		result, err := p.parseResult(callResult)
		if err != nil {
			lastError = err
			continue
		}

		// Validate with linter if DSL is present
		if result.DSL != nil {
			lintResult := p.validateDSL(result.DSL)
			if lintResult.Valid {
				return result, nil
			}

			// Lint failed, prepare error message for retry
			lastLintErrors = lintResult.FormatDiagnostics()
			lastError = fmt.Errorf("QueryDSL validation failed: %s", lastLintErrors)

			// Add lint warnings to result warnings
			for _, diag := range lintResult.Diagnostics {
				result.Warnings = append(result.Warnings, fmt.Sprintf("[%s] %s: %s", diag.Code, diag.Path, diag.Message))
			}
			continue
		}

		// No DSL returned
		lastError = fmt.Errorf("no QueryDSL returned from MCP tool")
	}

	return nil, fmt.Errorf("QueryDSL generation failed after %d attempts: %w", MaxRetries, lastError)
}

// buildArguments constructs the MCP tool arguments
func (p *MCPProvider) buildArguments(input *Input, attempt int, lastLintErrors string) map[string]interface{} {
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

	// Add retry context if this is a retry attempt
	if attempt > 1 && lastLintErrors != "" {
		arguments["retry"] = map[string]interface{}{
			"attempt":      attempt,
			"lint_errors":  lastLintErrors,
			"instructions": "The previous QueryDSL was invalid. Please fix the errors and regenerate.",
		}
	}

	return arguments
}

// validateDSL validates the generated QueryDSL using the linter
func (p *MCPProvider) validateDSL(dsl *gou.QueryDSL) *linter.LintResult {
	// Marshal DSL to JSON for linting
	jsonBytes, err := json.Marshal(dsl)
	if err != nil {
		result := &linter.LintResult{Valid: false}
		return result
	}

	_, lintResult := linter.Parse(string(jsonBytes))
	return lintResult
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
