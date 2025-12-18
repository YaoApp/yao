package querydsl

import (
	"encoding/json"
	"fmt"

	"github.com/yaoapp/gou/query/gou"
	"github.com/yaoapp/gou/query/linter"
	"github.com/yaoapp/yao/agent/caller"
	agentContext "github.com/yaoapp/yao/agent/context"
)

// AgentProvider delegates QueryDSL generation to an LLM-powered assistant
// The assistant can understand context and generate semantically correct QueryDSL
type AgentProvider struct {
	agentID string // Assistant ID to delegate to
}

// NewAgentProvider creates a new agent-based QueryDSL generator
func NewAgentProvider(agentID string) *AgentProvider {
	return &AgentProvider{
		agentID: agentID,
	}
}

// Generate generates QueryDSL by calling the target agent with retry and lint validation
// The agent receives the query and schema, returns generated QueryDSL
func (p *AgentProvider) Generate(ctx *agentContext.Context, input *Input) (*Result, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context is required for agent QueryDSL generation")
	}

	// Check if AgentGetterFunc is initialized
	if caller.AgentGetterFunc == nil {
		return nil, fmt.Errorf("AgentGetterFunc not initialized")
	}

	// Get the agent
	agent, err := caller.AgentGetterFunc(p.agentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent %s: %w", p.agentID, err)
	}

	var lastError error
	var lastLintErrors string

	for attempt := 1; attempt <= MaxRetries; attempt++ {
		// Build the request message in the format expected by querydsl agent
		requestMessage := p.buildRequestMessage(input, attempt, lastLintErrors)

		// Create message for the agent
		messages := []agentContext.Message{
			{
				Role:    "user",
				Content: requestMessage,
			},
		}

		// Call the agent with skip options (no history, no output)
		options := &agentContext.Options{
			Skip: &agentContext.Skip{
				History: true,
				Output:  true,
			},
		}

		result, err := agent.Stream(ctx, messages, options)
		if err != nil {
			lastError = fmt.Errorf("agent call failed: %w", err)
			continue
		}

		// Parse the result
		genResult, err := p.parseResult(result)
		if err != nil {
			lastError = err
			continue
		}

		// Validate with linter if DSL is present
		if genResult.DSL != nil {
			lintResult := p.validateDSL(genResult.DSL)
			if lintResult.Valid {
				return genResult, nil
			}

			// Lint failed, prepare error message for retry
			lastLintErrors = lintResult.FormatDiagnostics()
			lastError = fmt.Errorf("QueryDSL validation failed: %s", lastLintErrors)

			// Add lint warnings to result warnings
			for _, diag := range lintResult.Diagnostics {
				genResult.Warnings = append(genResult.Warnings, fmt.Sprintf("[%s] %s: %s", diag.Code, diag.Path, diag.Message))
			}
			continue
		}

		// No DSL returned
		lastError = fmt.Errorf("no QueryDSL returned from agent")
	}

	return nil, fmt.Errorf("QueryDSL generation failed after %d attempts: %w", MaxRetries, lastError)
}

// buildRequestMessage constructs the request message for the agent
// Format follows the querydsl agent prompts.yml:
// "用户查询\nSchema:\n{schema JSON}"
func (p *AgentProvider) buildRequestMessage(input *Input, attempt int, lastLintErrors string) string {
	// Build schema from extra params if provided
	var schemaJSON string
	if input.ExtraParams != nil {
		if schema, ok := input.ExtraParams["schema"]; ok {
			schemaBytes, _ := json.Marshal(schema)
			schemaJSON = string(schemaBytes)
		}
	}

	// Build message in the expected format
	message := input.Query
	if schemaJSON != "" {
		message = fmt.Sprintf("%s\nSchema:\n%s", input.Query, schemaJSON)
	}

	// Add scenario hint if specified (filter, aggregation, join, complex)
	if input.Scenario != "" {
		message = fmt.Sprintf("%s\nScenario: %s", message, input.Scenario)
	}

	// Add retry context if this is a retry attempt
	if attempt > 1 && lastLintErrors != "" {
		message = fmt.Sprintf("%s\n\nPrevious attempt failed with errors:\n%s\n\nPlease fix the errors and regenerate.", message, lastLintErrors)
	}

	return message
}

// validateDSL validates the generated QueryDSL using the linter
func (p *AgentProvider) validateDSL(dsl *gou.QueryDSL) *linter.LintResult {
	// Marshal DSL to JSON for linting
	jsonBytes, err := json.Marshal(dsl)
	if err != nil {
		result := &linter.LintResult{Valid: false}
		return result
	}

	_, lintResult := linter.Parse(string(jsonBytes))
	return lintResult
}

// parseResult extracts QueryDSL from the agent's response
// The querydsl agent returns QueryDSL JSON directly (not wrapped in {dsl: ...})
// Or returns error JSON: {"error": "code", "message": "..."}
// Stream() returns *context.Response with QueryDSL in "next" field
func (p *AgentProvider) parseResult(result interface{}) (*Result, error) {
	if result == nil {
		return &Result{}, nil
	}

	// Handle *context.Response directly (most common case from Stream())
	if resp, ok := result.(*agentContext.Response); ok {
		genResult := &Result{}
		if resp.Next != nil {
			// Next contains the QueryDSL from hook
			genResult.DSL = p.extractDSL(resp.Next)
		}
		return genResult, nil
	}

	// Try to convert to map first
	var data map[string]interface{}

	switch v := result.(type) {
	case map[string]interface{}:
		data = v
	case string:
		// Try to parse as JSON
		if err := json.Unmarshal([]byte(v), &data); err != nil {
			return nil, fmt.Errorf("failed to parse agent response: %w", err)
		}
	default:
		// Try to marshal and unmarshal
		jsonBytes, err := json.Marshal(result)
		if err != nil {
			return &Result{}, nil
		}
		if err := json.Unmarshal(jsonBytes, &data); err != nil {
			return &Result{}, nil
		}
	}

	genResult := &Result{}

	// Check for Stream() wrapper: { content: "...", next: {...} }
	// The actual response is in "content" field as a string
	if content, hasContent := data["content"]; hasContent && content != nil {
		if contentStr, ok := content.(string); ok && contentStr != "" {
			// Parse the content string as JSON
			var contentData map[string]interface{}
			if err := json.Unmarshal([]byte(contentStr), &contentData); err == nil {
				data = contentData
			}
		}
	}

	// Check for error response: {"error": "code", "message": "..."}
	if errCode, hasError := data["error"]; hasError {
		errMsg := ""
		if msg, ok := data["message"].(string); ok {
			errMsg = msg
		}
		return nil, fmt.Errorf("QueryDSL generation error [%v]: %s", errCode, errMsg)
	}

	// Check if this is a direct QueryDSL (has "from" or "select" field)
	// The querydsl agent returns QueryDSL directly, e.g., {"select": [...], "from": "table", ...}
	if _, hasFrom := data["from"]; hasFrom {
		genResult.DSL = p.extractDSL(data)
		return genResult, nil
	}
	if _, hasSelect := data["select"]; hasSelect {
		genResult.DSL = p.extractDSL(data)
		return genResult, nil
	}

	// Fallback: check for wrapped formats
	// Check for "next" field (custom hook data from NextHookResponse)
	if next, hasNext := data["next"]; hasNext && next != nil {
		if nextMap, ok := next.(map[string]interface{}); ok {
			data = nextMap
		} else if nextStr, ok := next.(string); ok {
			if err := json.Unmarshal([]byte(nextStr), &data); err == nil {
				// Check if parsed data is a QueryDSL
				if _, hasFrom := data["from"]; hasFrom {
					genResult.DSL = p.extractDSL(data)
					return genResult, nil
				}
			}
		}
	}

	// Check for "dsl" field wrapper
	if dsl, ok := data["dsl"]; ok {
		genResult.DSL = p.extractDSL(dsl)
	} else if d, ok := data["data"]; ok {
		if dm, ok := d.(map[string]interface{}); ok {
			// Check if data.data is a wrapped DSL: { dsl: {...} }
			if dsl, ok := dm["dsl"]; ok {
				genResult.DSL = p.extractDSL(dsl)
			} else if _, hasFrom := dm["from"]; hasFrom {
				// data.data is directly a QueryDSL (from __yao.querydsl Next hook)
				genResult.DSL = p.extractDSL(dm)
			} else if _, hasSelect := dm["select"]; hasSelect {
				// data.data is directly a QueryDSL
				genResult.DSL = p.extractDSL(dm)
			}
			if explain, ok := dm["explain"].(string); ok {
				genResult.Explain = explain
			}
			if warnings, ok := dm["warnings"]; ok {
				genResult.Warnings = p.extractWarnings(warnings)
			}
		}
	}

	// Get explain if present
	if explain, ok := data["explain"].(string); ok {
		genResult.Explain = explain
	}

	// Get warnings if present
	if warnings, ok := data["warnings"]; ok {
		genResult.Warnings = p.extractWarnings(warnings)
	}

	return genResult, nil
}

// extractDSL converts interface{} to gou.QueryDSL
func (p *AgentProvider) extractDSL(v interface{}) *gou.QueryDSL {
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
func (p *AgentProvider) extractWarnings(v interface{}) []string {
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
