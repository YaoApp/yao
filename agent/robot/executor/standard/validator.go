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

// Validate validates task output using two-layer validation (without multi-turn context)
// Equivalent to ValidateWithContext(task, output, nil)
// Use ValidateWithContext when you have a CallResult for better multi-turn support
func (v *Validator) Validate(task *robottypes.Task, output interface{}) *robottypes.ValidationResult {
	return v.ValidateWithContext(task, output, nil)
}

// ValidateWithContext validates task output and determines execution state for multi-turn conversation.
// It extends basic validation with:
// - Complete: whether expected result is obtained
// - NeedReply: whether to continue conversation
// - ReplyContent: content for next turn
//
// Parameters:
// - task: the task being executed
// - output: the output from assistant/mcp/process
// - callResult: the full call result (for detecting assistant's need for more info)
func (v *Validator) ValidateWithContext(task *robottypes.Task, output interface{}, callResult *CallResult) *robottypes.ValidationResult {
	// If no validation rules and no expected output, return passed and complete
	if task.ExpectedOutput == "" && len(task.ValidationRules) == 0 {
		return &robottypes.ValidationResult{
			Passed:   true,
			Score:    1.0,
			Complete: v.hasValidOutput(output),
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
			// Rule validation failed - check if we should retry with feedback
			ruleResult.Complete = false
			ruleResult.NeedReply, ruleResult.ReplyContent = v.checkNeedReplyOnFailure(task, ruleResult)
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

	// Determine execution state
	result.Complete = v.isComplete(task, output, result)
	result.NeedReply, result.ReplyContent = v.checkNeedReply(task, output, callResult, result)

	return result
}

// hasValidOutput checks if output is non-empty and valid
func (v *Validator) hasValidOutput(output interface{}) bool {
	if output == nil {
		return false
	}
	switch o := output.(type) {
	case string:
		return strings.TrimSpace(o) != ""
	case []interface{}:
		return len(o) > 0
	case map[string]interface{}:
		return len(o) > 0
	default:
		return true
	}
}

// isComplete determines if the expected result has been obtained
func (v *Validator) isComplete(task *robottypes.Task, output interface{}, result *robottypes.ValidationResult) bool {
	// If validation failed, not complete
	if !result.Passed {
		return false
	}

	// Must have valid output
	if !v.hasValidOutput(output) {
		return false
	}

	// If score is below threshold, consider incomplete
	if result.Score < v.config.ValidationThreshold {
		return false
	}

	return true
}

// checkNeedReply determines if conversation should continue and generates reply content
func (v *Validator) checkNeedReply(task *robottypes.Task, output interface{}, callResult *CallResult, result *robottypes.ValidationResult) (bool, string) {
	// If already complete, no need to reply
	if result.Complete {
		return false, ""
	}

	// Scenario 1: Assistant explicitly asks for more information
	if callResult != nil {
		text := callResult.GetText()
		if v.detectNeedMoreInfo(text) {
			return true, v.generateClarificationReply(task, text)
		}
	}

	// Scenario 2: Validation passed but output is incomplete/empty
	if result.Passed && !v.hasValidOutput(output) {
		return true, "Please continue and provide the complete result as specified in the task."
	}

	// Scenario 3: Validation failed with suggestions - can retry with feedback
	if !result.Passed && len(result.Suggestions) > 0 {
		return true, v.generateFeedbackReply(result)
	}

	// Scenario 4: Low confidence score - ask for improvement
	if result.Passed && result.Score < v.config.ValidationThreshold {
		return true, fmt.Sprintf("The result is partially correct (score: %.2f), but needs improvement. Please refine your response to better match the expected output: %s", result.Score, task.ExpectedOutput)
	}

	// No need to continue
	return false, ""
}

// checkNeedReplyOnFailure handles the case when rule validation fails
func (v *Validator) checkNeedReplyOnFailure(task *robottypes.Task, result *robottypes.ValidationResult) (bool, string) {
	// If there are suggestions, we can try to fix
	if len(result.Suggestions) > 0 {
		return true, v.generateFeedbackReply(result)
	}

	// If there are issues, provide feedback
	if len(result.Issues) > 0 {
		var sb strings.Builder
		sb.WriteString("Your response did not pass validation. Please fix the following issues:\n\n")
		for _, issue := range result.Issues {
			sb.WriteString(fmt.Sprintf("- %s\n", issue))
		}
		sb.WriteString(fmt.Sprintf("\nExpected output: %s", task.ExpectedOutput))
		return true, sb.String()
	}

	return false, ""
}

// detectNeedMoreInfo checks if assistant's response indicates need for more information
func (v *Validator) detectNeedMoreInfo(text string) bool {
	if text == "" {
		return false
	}

	textLower := strings.ToLower(text)
	keywords := []string{
		"need more information",
		"please clarify",
		"could you provide",
		"can you specify",
		"what is the",
		"which one",
		"please provide",
		"i need to know",
		"could you tell me",
		"what do you mean",
	}

	for _, kw := range keywords {
		if strings.Contains(textLower, kw) {
			return true
		}
	}

	// Check for question marks at the end (likely asking for clarification)
	// Note: We require 2+ question marks to avoid false positives from rhetorical questions
	// or questions that are part of the output (e.g., "How can I help you?")
	// Single questions are often just conversational and don't need clarification
	trimmed := strings.TrimSpace(text)
	if strings.HasSuffix(trimmed, "?") {
		if strings.Count(text, "?") >= 2 {
			return true
		}
	}

	return false
}

// generateClarificationReply generates a reply when assistant asks for clarification
func (v *Validator) generateClarificationReply(task *robottypes.Task, assistantText string) string {
	var sb strings.Builder
	sb.WriteString("Please proceed with the task based on the available information.\n\n")

	if task.ExpectedOutput != "" {
		sb.WriteString(fmt.Sprintf("**Expected Output**: %s\n\n", task.ExpectedOutput))
	}

	sb.WriteString("If you need to make assumptions, please state them clearly and proceed with the most reasonable interpretation.")

	return sb.String()
}

// generateFeedbackReply generates a reply with validation feedback
func (v *Validator) generateFeedbackReply(result *robottypes.ValidationResult) string {
	var sb strings.Builder
	sb.WriteString("## Validation Feedback\n\n")
	sb.WriteString("Your previous response needs improvement. Please address the following:\n\n")

	if len(result.Issues) > 0 {
		sb.WriteString("### Issues\n")
		for _, issue := range result.Issues {
			sb.WriteString(fmt.Sprintf("- %s\n", issue))
		}
		sb.WriteString("\n")
	}

	if len(result.Suggestions) > 0 {
		sb.WriteString("### Suggestions\n")
		for _, suggestion := range result.Suggestions {
			sb.WriteString(fmt.Sprintf("- %s\n", suggestion))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("Please provide an improved response that addresses these points.")

	return sb.String()
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
// Format matches the Validation Agent's expected input structure:
// 1. Task: task definition with expected_output and validation_rules
// 2. Result: actual output from task execution
// 3. Success Criteria: overall criteria (optional)
func (v *Validator) BuildSemanticPrompt(task *robottypes.Task, output interface{}) string {
	var sb strings.Builder

	// Section 1: Task (matches Agent's expected "Task" input)
	sb.WriteString("## Task\n\n")
	sb.WriteString(fmt.Sprintf("**Task ID**: %s\n", task.ID))
	sb.WriteString(fmt.Sprintf("**Executor**: %s (%s)\n\n", task.ExecutorID, task.ExecutorType))

	// Task description (instructions)
	if len(task.Messages) > 0 {
		sb.WriteString("**Instructions**:\n")
		for _, msg := range task.Messages {
			if content, ok := msg.Content.(string); ok {
				sb.WriteString(content + "\n")
			}
		}
		sb.WriteString("\n")
	}

	// Expected output (primary criterion for semantic validation)
	if task.ExpectedOutput != "" {
		sb.WriteString(fmt.Sprintf("**expected_output**: %s\n\n", task.ExpectedOutput))
	}

	// Validation rules
	semanticRules := v.getSemanticRules(task.ValidationRules)
	if len(semanticRules) > 0 {
		sb.WriteString("**validation_rules**:\n")
		for _, rule := range semanticRules {
			sb.WriteString(fmt.Sprintf("- %s\n", rule))
		}
		sb.WriteString("\n")
	}

	// Section 2: Result (matches Agent's expected "Result" input)
	sb.WriteString("## Result\n\n")
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

	// Section 3: Success Criteria (optional, from goals if available)
	// Note: This could be extended to include criteria from exec.Goals if needed
	sb.WriteString("\n## Success Criteria\n\n")
	if task.ExpectedOutput != "" {
		sb.WriteString(fmt.Sprintf("The task should produce: %s\n", task.ExpectedOutput))
	} else {
		sb.WriteString("Complete the task successfully with valid output.\n")
	}

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
