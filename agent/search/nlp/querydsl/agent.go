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
		// Build the request message
		requestData := p.buildRequestData(input, attempt, lastLintErrors)
		requestJSON, _ := json.Marshal(requestData)

		// Create message for the agent
		messages := []agentContext.Message{
			{
				Role:    "user",
				Content: string(requestJSON),
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

// buildRequestData constructs the request data for the agent
func (p *AgentProvider) buildRequestData(input *Input, attempt int, lastLintErrors string) map[string]interface{} {
	requestData := map[string]interface{}{
		"query":  input.Query,
		"models": input.ModelIDs,
		"limit":  input.Limit,
	}

	// Add optional fields
	if len(input.Wheres) > 0 {
		requestData["wheres"] = input.Wheres
	}
	if len(input.Orders) > 0 {
		requestData["orders"] = input.Orders
	}
	if len(input.AllowedFields) > 0 {
		requestData["allowed_fields"] = input.AllowedFields
	}
	if len(input.ExtraParams) > 0 {
		requestData["extra"] = input.ExtraParams
	}

	// Add retry context if this is a retry attempt
	if attempt > 1 && lastLintErrors != "" {
		requestData["retry"] = map[string]interface{}{
			"attempt":      attempt,
			"lint_errors":  lastLintErrors,
			"instructions": "The previous QueryDSL was invalid. Please fix the errors and regenerate.",
		}
	}

	return requestData
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
// The agent should return data in NextHookResponse format: { data: { dsl: {...}, explain: "..." } }
// The Stream() response wraps this in: { next: { data: { dsl: {...} } } }
func (p *AgentProvider) parseResult(result interface{}) (*Result, error) {
	if result == nil {
		return &Result{}, nil
	}

	// Try to convert to map first (most common case)
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

	// Check for "next" field (custom hook data from NextHookResponse)
	// Stream() returns: { next: { data: { dsl: {...} } } }
	if next, hasNext := data["next"]; hasNext && next != nil {
		if nextMap, ok := next.(map[string]interface{}); ok {
			data = nextMap
		} else if nextStr, ok := next.(string); ok {
			if err := json.Unmarshal([]byte(nextStr), &data); err != nil {
				return &Result{}, nil
			}
		}
	}

	// Extract QueryDSL from data
	// Try common field names: "dsl", "data", "data.dsl"
	genResult := &Result{}

	// Get explain if present
	if explain, ok := data["explain"].(string); ok {
		genResult.Explain = explain
	}

	// Get warnings if present
	if warnings, ok := data["warnings"]; ok {
		genResult.Warnings = p.extractWarnings(warnings)
	}

	// Get DSL
	if dsl, ok := data["dsl"]; ok {
		genResult.DSL = p.extractDSL(dsl)
	} else if d, ok := data["data"]; ok {
		if dm, ok := d.(map[string]interface{}); ok {
			if dsl, ok := dm["dsl"]; ok {
				genResult.DSL = p.extractDSL(dsl)
			}
			if explain, ok := dm["explain"].(string); ok {
				genResult.Explain = explain
			}
			if warnings, ok := dm["warnings"]; ok {
				genResult.Warnings = p.extractWarnings(warnings)
			}
		}
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
