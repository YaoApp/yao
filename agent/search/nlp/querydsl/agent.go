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

		// Parse the result from response
		genResult, err := p.parseResponse(result)
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
// Returns JSON format for structured communication with the agent
func (p *AgentProvider) buildRequestMessage(input *Input, attempt int, lastLintErrors string) string {
	// Build request data as JSON
	requestData := map[string]interface{}{
		"query":  input.Query,
		"models": input.ModelIDs,
		"limit":  input.Limit,
	}

	// Add schema from extra params if provided
	if input.ExtraParams != nil {
		if schema, ok := input.ExtraParams["schema"]; ok {
			requestData["schema"] = schema
		}
	}

	// Add scenario hint if specified (filter, aggregation, join, complex)
	if input.Scenario != "" {
		requestData["scenario"] = string(input.Scenario)
	}

	// Add allowed fields if specified
	if len(input.AllowedFields) > 0 {
		requestData["allowed_fields"] = input.AllowedFields
	}

	// Add retry context if this is a retry attempt
	if attempt > 1 && lastLintErrors != "" {
		requestData["retry"] = map[string]interface{}{
			"attempt":      attempt,
			"lint_errors":  lastLintErrors,
			"instructions": "The previous QueryDSL was invalid. Please fix the errors and regenerate.",
		}
	}

	jsonBytes, _ := json.Marshal(requestData)
	return string(jsonBytes)
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

// parseResponse extracts QueryDSL from the agent's *context.Response
// Now that agent.Stream() returns *context.Response directly,
// we can access fields without type assertions.
//
// The querydsl agent returns QueryDSL in response.Next field
// Or returns error JSON: {"error": "code", "message": "..."}
func (p *AgentProvider) parseResponse(response *agentContext.Response) (*Result, error) {
	if response == nil {
		return &Result{}, nil
	}

	// Check Next field first (custom hook data)
	if response.Next != nil {
		return p.parseNextData(response.Next)
	}

	// No Next data, return empty result
	return &Result{}, nil
}

// parseNextData extracts QueryDSL from Next hook data
func (p *AgentProvider) parseNextData(next interface{}) (*Result, error) {
	if next == nil {
		return &Result{}, nil
	}

	// Try to convert to map first
	var data map[string]interface{}

	switch v := next.(type) {
	case map[string]interface{}:
		data = v
	case string:
		// Try to parse as JSON
		if err := json.Unmarshal([]byte(v), &data); err != nil {
			return nil, fmt.Errorf("failed to parse agent response: %w", err)
		}
	default:
		// Try to marshal and unmarshal
		jsonBytes, err := json.Marshal(next)
		if err != nil {
			return &Result{}, nil
		}
		if err := json.Unmarshal(jsonBytes, &data); err != nil {
			return &Result{}, nil
		}
	}

	genResult := &Result{}

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

	// Check for "dsl" field wrapper: { dsl: {...} }
	if dsl, ok := data["dsl"]; ok {
		genResult.DSL = p.extractDSL(dsl)
		if explain, ok := data["explain"].(string); ok {
			genResult.Explain = explain
		}
		if warnings, ok := data["warnings"]; ok {
			genResult.Warnings = p.extractWarnings(warnings)
		}
		return genResult, nil
	}

	// Check for "data" field wrapper: { data: { dsl: {...}, explain: "...", warnings: [] } }
	if d, ok := data["data"]; ok {
		if dm, ok := d.(map[string]interface{}); ok {
			// Check if data.data contains dsl field: { data: { dsl: {...} } }
			if dsl, ok := dm["dsl"]; ok {
				genResult.DSL = p.extractDSL(dsl)
			} else if _, hasFrom := dm["from"]; hasFrom {
				// data.data is directly a QueryDSL (from __yao.querydsl Next hook)
				genResult.DSL = p.extractDSL(dm)
			} else if _, hasSelect := dm["select"]; hasSelect {
				// data.data is directly a QueryDSL
				genResult.DSL = p.extractDSL(dm)
			}
			// Extract explain and warnings from data.data
			if explain, ok := dm["explain"].(string); ok {
				genResult.Explain = explain
			}
			if warnings, ok := dm["warnings"]; ok {
				genResult.Warnings = p.extractWarnings(warnings)
			}
			return genResult, nil
		}
	}

	// Fallback: Get explain and warnings from top level
	if explain, ok := data["explain"].(string); ok {
		genResult.Explain = explain
	}
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
