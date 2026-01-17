package standard

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/yaoapp/gou/process"
	robottypes "github.com/yaoapp/yao/agent/robot/types"
	"github.com/yaoapp/yao/assert"
)

// Validator handles task result validation using a two-layer approach:
// 1. Rule-based validation: Uses yao/assert for deterministic rules (type, contains, regex, json_path)
// 2. Semantic validation: Calls Validation Agent for semantic understanding (ExpectedOutput)
type Validator struct {
	ctx      *robottypes.Context
	robot    *robottypes.Robot
	config   *RunConfig
	asserter *assert.Asserter
}

// NewValidator creates a new task validator
func NewValidator(ctx *robottypes.Context, robot *robottypes.Robot, config *RunConfig) *Validator {
	v := &Validator{
		ctx:      ctx,
		robot:    robot,
		config:   config,
		asserter: assert.New(),
	}

	// Configure asserter with robot-specific implementations
	v.asserter.WithAgentValidator(&robotAgentValidator{v: v})
	v.asserter.WithScriptRunner(&robotScriptRunner{ctx: ctx})

	return v
}

// Validate validates task output using two-layer validation:
// 1. First, run rule-based assertions (fast, deterministic)
// 2. Then, if ExpectedOutput is set, run semantic validation via Agent
func (v *Validator) Validate(task *robottypes.Task, output interface{}) *robottypes.ValidationResult {
	// If no validation rules and no expected output, return passed
	if task.ExpectedOutput == "" && len(task.ValidationRules) == 0 {
		return &robottypes.ValidationResult{
			Passed: true,
			Score:  1.0,
		}
	}

	result := &robottypes.ValidationResult{
		Passed: true,
		Score:  1.0,
	}

	// Layer 1: Rule-based validation (using yao/assert)
	if len(task.ValidationRules) > 0 {
		ruleResult := v.validateRules(task.ValidationRules, output)
		if !ruleResult.Passed {
			return ruleResult
		}
		// Merge rule validation results
		result.Issues = append(result.Issues, ruleResult.Issues...)
		result.Suggestions = append(result.Suggestions, ruleResult.Suggestions...)
	}

	// Layer 2: Semantic validation (using Validation Agent)
	// Only run if ExpectedOutput is set or there are agent-type rules
	if task.ExpectedOutput != "" || v.hasAgentRules(task.ValidationRules) {
		semanticResult := v.validateSemantic(task, output)
		result = v.mergeResults(result, semanticResult)
	}

	return result
}

// validateRules validates output against rule-based assertions
func (v *Validator) validateRules(rules []string, output interface{}) *robottypes.ValidationResult {
	result := &robottypes.ValidationResult{
		Passed: true,
		Score:  1.0,
	}

	// Parse rules into assertions
	assertions := v.parseRules(rules)
	if len(assertions) == 0 {
		return result
	}

	// Run assertions
	passed, message := v.asserter.Validate(assertions, output)
	if !passed {
		result.Passed = false
		result.Score = 0
		result.Issues = append(result.Issues, message)
	}

	return result
}

// parseRules converts validation rules (strings or JSON) to assertions
// Supports:
// - Simple string rules: "output must be valid JSON" (converted to type check)
// - JSON assertion objects: {"type": "contains", "value": "success"}
func (v *Validator) parseRules(rules []string) []*assert.Assertion {
	var assertions []*assert.Assertion

	for _, rule := range rules {
		// Try to parse as JSON assertion
		if strings.HasPrefix(rule, "{") {
			var assertionMap map[string]interface{}
			if err := json.Unmarshal([]byte(rule), &assertionMap); err == nil {
				parsed := assert.ParseAssertions(assertionMap)
				assertions = append(assertions, parsed...)
				continue
			}
		}

		// Convert common string rules to assertions
		assertion := v.convertStringRule(rule)
		if assertion != nil {
			assertions = append(assertions, assertion)
		}
	}

	return assertions
}

// convertStringRule converts a human-readable rule string to an assertion
// Examples:
// - "output must be valid JSON" -> {"type": "type", "value": "object"}
// - "must contain 'success'" -> {"type": "contains", "value": "success"}
// - "count > 0" -> (passed to semantic validation)
func (v *Validator) convertStringRule(rule string) *assert.Assertion {
	ruleLower := strings.ToLower(rule)

	// JSON type check
	if strings.Contains(ruleLower, "valid json") || strings.Contains(ruleLower, "json object") {
		return &assert.Assertion{
			Type:    "type",
			Value:   "object",
			Message: rule,
		}
	}

	// Array type check
	if strings.Contains(ruleLower, "json array") || strings.Contains(ruleLower, "must be array") {
		return &assert.Assertion{
			Type:    "type",
			Value:   "array",
			Message: rule,
		}
	}

	// Contains check
	if strings.Contains(ruleLower, "contain") {
		// Extract the value in quotes
		if start := strings.Index(rule, "'"); start != -1 {
			if end := strings.Index(rule[start+1:], "'"); end != -1 {
				value := rule[start+1 : start+1+end]
				return &assert.Assertion{
					Type:    "contains",
					Value:   value,
					Message: rule,
				}
			}
		}
		if start := strings.Index(rule, "\""); start != -1 {
			if end := strings.Index(rule[start+1:], "\""); end != -1 {
				value := rule[start+1 : start+1+end]
				return &assert.Assertion{
					Type:    "contains",
					Value:   value,
					Message: rule,
				}
			}
		}
	}

	// Not empty check - use regex to match at least one character
	if strings.Contains(ruleLower, "not empty") || strings.Contains(ruleLower, "non-empty") {
		return &assert.Assertion{
			Type:    "regex",
			Value:   ".+",
			Message: rule,
		}
	}

	// For other rules, return nil (will be handled by semantic validation)
	return nil
}

// hasAgentRules checks if any rule requires agent-based validation
func (v *Validator) hasAgentRules(rules []string) bool {
	for _, rule := range rules {
		if strings.HasPrefix(rule, "{") {
			var assertionMap map[string]interface{}
			if err := json.Unmarshal([]byte(rule), &assertionMap); err == nil {
				if assertionMap["type"] == "agent" {
					return true
				}
			}
		}
	}
	return false
}

// validateSemantic performs semantic validation using the Validation Agent
func (v *Validator) validateSemantic(task *robottypes.Task, output interface{}) *robottypes.ValidationResult {
	// Get validation agent ID
	validationAgentID := "__yao.validation" // default
	if v.robot.Config != nil && v.robot.Config.Resources != nil {
		if customID, ok := v.robot.Config.Resources.Phases["validation"]; ok && customID != "" {
			validationAgentID = customID
		}
	}

	// Build validation prompt
	validationPrompt := v.BuildSemanticPrompt(task, output)

	// Call validation agent
	caller := NewAgentCaller()
	result, err := caller.CallWithMessages(v.ctx, validationAgentID, validationPrompt)
	if err != nil {
		return &robottypes.ValidationResult{
			Passed: false,
			Score:  0,
			Issues: []string{fmt.Sprintf("Validation agent error: %s", err.Error())},
		}
	}

	return v.ParseAgentResult(result)
}

// BuildSemanticPrompt builds the prompt for semantic validation
func (v *Validator) BuildSemanticPrompt(task *robottypes.Task, output interface{}) string {
	var sb strings.Builder

	sb.WriteString("## Task Definition\n\n")
	sb.WriteString(fmt.Sprintf("**Task ID**: %s\n", task.ID))
	sb.WriteString(fmt.Sprintf("**Executor**: %s (%s)\n\n", task.ExecutorID, task.ExecutorType))

	// Task description
	if len(task.Messages) > 0 {
		sb.WriteString("**Task Instructions**:\n")
		for _, msg := range task.Messages {
			if content, ok := msg.Content.(string); ok {
				sb.WriteString(content + "\n")
			}
		}
		sb.WriteString("\n")
	}

	// Expected output (primary criterion for semantic validation)
	if task.ExpectedOutput != "" {
		sb.WriteString(fmt.Sprintf("**Expected Output**: %s\n\n", task.ExpectedOutput))
	}

	// Semantic validation rules (rules that couldn't be converted to assertions)
	semanticRules := v.getSemanticRules(task.ValidationRules)
	if len(semanticRules) > 0 {
		sb.WriteString("**Validation Criteria**:\n")
		for _, rule := range semanticRules {
			sb.WriteString(fmt.Sprintf("- %s\n", rule))
		}
		sb.WriteString("\n")
	}

	// Actual output
	sb.WriteString("## Actual Output\n\n")
	if output != nil {
		outputJSON, err := json.MarshalIndent(output, "", "  ")
		if err == nil {
			sb.WriteString(fmt.Sprintf("```json\n%s\n```\n", string(outputJSON)))
		} else {
			sb.WriteString(fmt.Sprintf("%v\n", output))
		}
	} else {
		sb.WriteString("(no output)\n")
	}

	sb.WriteString("\n## Validation Request\n\n")
	sb.WriteString("Please validate the actual output against the expected output and validation criteria. ")
	sb.WriteString("Focus on semantic correctness and completeness. ")
	sb.WriteString("Return a JSON object with: passed (bool), score (0-1), issues (array), suggestions (array), details (markdown report).\n")

	return sb.String()
}

// getSemanticRules returns rules that need semantic validation (not convertible to assertions)
func (v *Validator) getSemanticRules(rules []string) []string {
	var semanticRules []string
	for _, rule := range rules {
		// Skip JSON assertions (already handled)
		if strings.HasPrefix(rule, "{") {
			continue
		}
		// Skip rules that were converted to assertions
		if v.convertStringRule(rule) == nil {
			semanticRules = append(semanticRules, rule)
		}
	}
	return semanticRules
}

// ParseAgentResult parses the validation agent's response
func (v *Validator) ParseAgentResult(result *CallResult) *robottypes.ValidationResult {
	validation := &robottypes.ValidationResult{
		Passed: false,
		Score:  0,
	}

	// Try to parse as JSON
	data, err := result.GetJSON()
	if err != nil {
		// If not JSON, try to interpret the text response
		text := result.GetText()
		if text != "" {
			validation.Details = text
			// Simple heuristic: check for positive keywords
			textLower := strings.ToLower(text)
			positiveKeywords := []string{"passed", "valid", "correct", "success"}
			for _, keyword := range positiveKeywords {
				if strings.Contains(textLower, keyword) {
					validation.Passed = true
					validation.Score = 0.8
					break
				}
			}
		}
		return validation
	}

	// Parse JSON fields
	if passed, ok := data["passed"].(bool); ok {
		validation.Passed = passed
	}

	if score, ok := data["score"].(float64); ok {
		validation.Score = score
	}

	if issues, ok := data["issues"].([]interface{}); ok {
		for _, issue := range issues {
			if s, ok := issue.(string); ok {
				validation.Issues = append(validation.Issues, s)
			}
		}
	}

	if suggestions, ok := data["suggestions"].([]interface{}); ok {
		for _, suggestion := range suggestions {
			if s, ok := suggestion.(string); ok {
				validation.Suggestions = append(validation.Suggestions, s)
			}
		}
	}

	if details, ok := data["details"].(string); ok {
		validation.Details = details
	}

	return validation
}

// mergeResults merges rule-based and semantic validation results
func (v *Validator) mergeResults(ruleResult, semanticResult *robottypes.ValidationResult) *robottypes.ValidationResult {
	// If either failed, the overall result is failed
	if !ruleResult.Passed || !semanticResult.Passed {
		return &robottypes.ValidationResult{
			Passed:      false,
			Score:       min(ruleResult.Score, semanticResult.Score),
			Issues:      append(ruleResult.Issues, semanticResult.Issues...),
			Suggestions: append(ruleResult.Suggestions, semanticResult.Suggestions...),
			Details:     semanticResult.Details,
		}
	}

	// Both passed
	return &robottypes.ValidationResult{
		Passed:      true,
		Score:       (ruleResult.Score + semanticResult.Score) / 2,
		Issues:      append(ruleResult.Issues, semanticResult.Issues...),
		Suggestions: append(ruleResult.Suggestions, semanticResult.Suggestions...),
		Details:     semanticResult.Details,
	}
}

// ============================================================================
// Robot-specific implementations of assert interfaces
// ============================================================================

// robotAgentValidator implements assert.AgentValidator for robot package
type robotAgentValidator struct {
	v *Validator
}

// Validate validates output using an agent
func (av *robotAgentValidator) Validate(agentID string, output, input, criteria interface{}, options *assert.AssertionOptions) *assert.Result {
	result := &assert.Result{}

	// Build validation request
	validationInput := map[string]interface{}{
		"output": output,
		"input":  input,
	}
	if criteria != nil {
		validationInput["criteria"] = criteria
	}

	inputJSON, err := json.Marshal(validationInput)
	if err != nil {
		result.Passed = false
		result.Message = fmt.Sprintf("failed to marshal validation input: %s", err.Error())
		return result
	}

	// Call agent
	caller := NewAgentCaller()
	callResult, err := caller.CallWithMessages(av.v.ctx, agentID, string(inputJSON))
	if err != nil {
		result.Passed = false
		result.Message = fmt.Sprintf("agent validation error: %s", err.Error())
		return result
	}

	// Parse response
	data, err := callResult.GetJSON()
	if err != nil {
		result.Passed = false
		result.Message = "agent returned invalid response format"
		return result
	}

	if passed, ok := data["passed"].(bool); ok {
		result.Passed = passed
	}
	if reason, ok := data["reason"].(string); ok {
		result.Message = reason
	}
	result.Expected = data

	return result
}

// robotScriptRunner implements assert.ScriptRunner for robot package
type robotScriptRunner struct {
	ctx *robottypes.Context
}

// Run runs an assertion script using Yao process
func (r *robotScriptRunner) Run(scriptName string, output, input, expected interface{}) (bool, string, error) {
	// Build script arguments
	args := []interface{}{output, input, expected}

	// Create and run the process
	proc, err := process.Of(scriptName, args...)
	if err != nil {
		return false, "", fmt.Errorf("failed to create process: %w", err)
	}

	// Set context for timeout and cancellation support
	if r.ctx != nil {
		proc.Context = r.ctx.Context
	}

	if err := proc.Execute(); err != nil {
		return false, "", fmt.Errorf("script execution failed: %w", err)
	}
	defer proc.Release()

	// Parse result - expected format: bool or { "pass": bool, "message": string }
	res := proc.Value()
	switch v := res.(type) {
	case bool:
		if v {
			return true, "script assertion passed", nil
		}
		return false, "script assertion failed", nil

	case map[string]interface{}:
		passed := false
		message := ""
		if pass, ok := v["pass"].(bool); ok {
			passed = pass
		}
		if msg, ok := v["message"].(string); ok {
			message = msg
		}
		return passed, message, nil

	default:
		return false, fmt.Sprintf("script returned unexpected type: %T", res), nil
	}
}
