package test

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/process"
	goutext "github.com/yaoapp/gou/text"
	"github.com/yaoapp/yao/agent/assistant"
	"github.com/yaoapp/yao/agent/context"
)

// Asserter handles test assertions
type Asserter struct {
	// response holds the current response for tool-related assertions
	response *context.Response
}

// NewAsserter creates a new asserter
func NewAsserter() *Asserter {
	return &Asserter{}
}

// WithResponse sets the response for tool-related assertions
func (a *Asserter) WithResponse(response *context.Response) *Asserter {
	a.response = response
	return a
}

// Validate validates the output against the test case's assertions
// Returns (passed, error message)
func (a *Asserter) Validate(tc *Case, output interface{}) (bool, string) {
	// If assert is defined, use assertion rules
	if tc.Assert != nil {
		return a.validateAssertions(tc, output)
	}

	// If expected is defined, use simple comparison
	if tc.Expected != nil {
		if validateOutput(output, tc.Expected) {
			return true, ""
		}
		return false, "output does not match expected"
	}

	// No assertions defined - pass if we got output without error
	return true, ""
}

// ValidateWithDetails validates the output and returns detailed results
// This is useful for agent assertions where we want to capture the validator's response
func (a *Asserter) ValidateWithDetails(tc *Case, output interface{}) *AssertionResult {
	if tc.Assert == nil {
		return &AssertionResult{Passed: true}
	}

	assertions := a.parseAssertions(tc.Assert)
	if len(assertions) == 0 {
		return &AssertionResult{Passed: true}
	}

	// For single assertion, return its full result
	if len(assertions) == 1 {
		return a.evaluateAssertion(assertions[0], output, tc.Input)
	}

	// For multiple assertions, combine results
	var failures []string
	for _, assertion := range assertions {
		result := a.evaluateAssertion(assertion, output, tc.Input)
		if !result.Passed {
			msg := result.Message
			if assertion.Message != "" {
				msg = assertion.Message
			}
			failures = append(failures, msg)
		}
	}

	if len(failures) > 0 {
		return &AssertionResult{
			Passed:  false,
			Message: strings.Join(failures, "; "),
		}
	}
	return &AssertionResult{Passed: true}
}

// validateAssertions validates output against assertion rules
func (a *Asserter) validateAssertions(tc *Case, output interface{}) (bool, string) {
	assertions := a.parseAssertions(tc.Assert)
	if len(assertions) == 0 {
		return true, ""
	}

	var failures []string
	for _, assertion := range assertions {
		result := a.evaluateAssertion(assertion, output, tc.Input)
		if !result.Passed {
			msg := result.Message
			if assertion.Message != "" {
				msg = assertion.Message
			}
			failures = append(failures, msg)
		}
	}

	if len(failures) > 0 {
		return false, strings.Join(failures, "; ")
	}
	return true, ""
}

// parseAssertions parses the assert field into a list of assertions
func (a *Asserter) parseAssertions(assert interface{}) []*Assertion {
	if assert == nil {
		return nil
	}

	var assertions []*Assertion

	switch v := assert.(type) {
	case map[string]interface{}:
		// Single assertion object
		assertion := a.mapToAssertion(v)
		if assertion != nil {
			assertions = append(assertions, assertion)
		}

	case []interface{}:
		// Array of assertions
		for _, item := range v {
			if m, ok := item.(map[string]interface{}); ok {
				assertion := a.mapToAssertion(m)
				if assertion != nil {
					assertions = append(assertions, assertion)
				}
			}
		}

	case string:
		// Shorthand: just a type name (e.g., "contains")
		assertions = append(assertions, &Assertion{Type: v})
	}

	return assertions
}

// mapToAssertion converts a map to an Assertion
func (a *Asserter) mapToAssertion(m map[string]interface{}) *Assertion {
	assertion := &Assertion{}

	if t, ok := m["type"].(string); ok {
		assertion.Type = t
	}
	if v, ok := m["value"]; ok {
		assertion.Value = v
	}
	if p, ok := m["path"].(string); ok {
		assertion.Path = p
	}
	if s, ok := m["script"].(string); ok {
		assertion.Script = s
	}
	if u, ok := m["use"].(string); ok {
		assertion.Use = u
	}
	if msg, ok := m["message"].(string); ok {
		assertion.Message = msg
	}
	if n, ok := m["negate"].(bool); ok {
		assertion.Negate = n
	}

	// Parse options for agent assertions
	if opts, ok := m["options"].(map[string]interface{}); ok {
		assertion.Options = &AssertionOptions{}
		if c, ok := opts["connector"].(string); ok {
			assertion.Options.Connector = c
		}
		if meta, ok := opts["metadata"].(map[string]interface{}); ok {
			assertion.Options.Metadata = meta
		}
	}

	return assertion
}

// evaluateAssertion evaluates a single assertion
func (a *Asserter) evaluateAssertion(assertion *Assertion, output, input interface{}) *AssertionResult {
	result := &AssertionResult{
		Assertion: assertion,
		Expected:  assertion.Value,
	}

	switch assertion.Type {
	case "equals", "":
		result = a.assertEquals(assertion, output)
	case "contains":
		result = a.assertContains(assertion, output)
	case "not_contains":
		result = a.assertNotContains(assertion, output)
	case "json_path":
		result = a.assertJSONPath(assertion, output)
	case "regex":
		result = a.assertRegex(assertion, output)
	case "type":
		result = a.assertType(assertion, output)
	case "script":
		result = a.assertScript(assertion, output, input)
	case "agent":
		result = a.assertAgent(assertion, output, input)
	case "tool_called":
		result = a.assertToolCalled(assertion)
	case "tool_result":
		result = a.assertToolResult(assertion)
	default:
		result.Passed = false
		result.Message = fmt.Sprintf("unknown assertion type: %s", assertion.Type)
	}

	// Apply negate
	if assertion.Negate {
		result.Passed = !result.Passed
		if result.Passed {
			result.Message = "negated assertion passed"
		} else {
			result.Message = "negated: " + result.Message
		}
	}

	return result
}

// assertEquals checks for exact equality
func (a *Asserter) assertEquals(assertion *Assertion, output interface{}) *AssertionResult {
	result := &AssertionResult{
		Assertion: assertion,
		Actual:    output,
		Expected:  assertion.Value,
	}

	if validateOutput(output, assertion.Value) {
		result.Passed = true
		result.Message = "values are equal"
	} else {
		result.Passed = false
		result.Message = fmt.Sprintf("expected %v, got %v", assertion.Value, output)
	}

	return result
}

// assertContains checks if output contains the expected value
func (a *Asserter) assertContains(assertion *Assertion, output interface{}) *AssertionResult {
	result := &AssertionResult{
		Assertion: assertion,
		Actual:    output,
		Expected:  assertion.Value,
	}

	outputStr := a.toString(output)
	expectedStr := a.toString(assertion.Value)

	if strings.Contains(outputStr, expectedStr) {
		result.Passed = true
		result.Message = fmt.Sprintf("output contains '%s'", expectedStr)
	} else {
		result.Passed = false
		result.Message = fmt.Sprintf("output does not contain '%s'", expectedStr)
	}

	return result
}

// assertNotContains checks if output does not contain the expected value
func (a *Asserter) assertNotContains(assertion *Assertion, output interface{}) *AssertionResult {
	result := a.assertContains(assertion, output)
	result.Passed = !result.Passed
	if result.Passed {
		result.Message = fmt.Sprintf("output does not contain '%s'", a.toString(assertion.Value))
	} else {
		result.Message = fmt.Sprintf("output should not contain '%s'", a.toString(assertion.Value))
	}
	return result
}

// assertJSONPath extracts a value using JSON path and compares
func (a *Asserter) assertJSONPath(assertion *Assertion, output interface{}) *AssertionResult {
	result := &AssertionResult{
		Assertion: assertion,
		Expected:  assertion.Value,
	}

	// Convert output to JSON if needed
	var jsonData interface{}
	switch v := output.(type) {
	case string:
		// Use gou/text to extract JSON (handles markdown, auto-repair, etc.)
		extracted := goutext.ExtractJSON(v)
		if extracted != nil {
			jsonData = extracted
		} else {
			result.Passed = false
			result.Message = fmt.Sprintf("output is not valid JSON: %s", v)
			return result
		}
	case map[string]interface{}, []interface{}:
		jsonData = v
	default:
		result.Passed = false
		result.Message = fmt.Sprintf("output is not a JSON object or array, got: %T = %v", output, truncateOutput(output, 200))
		return result
	}

	// Extract value using simple path (e.g., "$.need_search" or "need_search")
	path := strings.TrimPrefix(assertion.Path, "$.")
	actual := a.extractPath(jsonData, path)
	result.Actual = actual

	// Compare expected value with actual value
	// First try direct comparison (handles array-to-array comparison)
	if validateOutput(actual, assertion.Value) {
		result.Passed = true
		result.Message = fmt.Sprintf("path '%s' equals expected value", assertion.Path)
		return result
	}

	// If expected is an array and direct comparison failed, check if actual matches ANY element (IN semantics)
	// This is for cases like: expected: ["a", "b"], actual: "a" (actual is one of expected)
	if expectedArr, ok := assertion.Value.([]interface{}); ok {
		// Only apply IN semantics if actual is NOT an array (otherwise it was already compared above)
		if _, actualIsArr := actual.([]interface{}); !actualIsArr {
			for _, expectedItem := range expectedArr {
				if validateOutput(actual, expectedItem) {
					result.Passed = true
					result.Message = fmt.Sprintf("path '%s' equals one of expected values", assertion.Path)
					return result
				}
			}
		}
		result.Passed = false
		result.Message = fmt.Sprintf("path '%s': expected %v, got %v", assertion.Path, assertion.Value, actual)
	} else {
		// Direct comparison already failed above
		result.Passed = false
		result.Message = fmt.Sprintf("path '%s': expected %v, got %v", assertion.Path, assertion.Value, actual)
	}

	return result
}

// truncateOutput truncates output for error messages
func truncateOutput(output interface{}, maxLen int) string {
	var s string
	switch v := output.(type) {
	case string:
		s = v
	case nil:
		return "<nil>"
	default:
		bytes, err := jsoniter.Marshal(v)
		if err != nil {
			s = fmt.Sprintf("%v", v)
		} else {
			s = string(bytes)
		}
	}

	if len(s) > maxLen {
		return s[:maxLen] + "..."
	}
	return s
}

// extractPath extracts a value from JSON using dot-notation path with array index support
// Supports: "field", "field.nested", "field[0]", "field[0].nested", "field.nested[0].value"
func (a *Asserter) extractPath(data interface{}, path string) interface{} {
	current := data

	// Parse path into segments, handling both dots and array indices
	// e.g., "wheres[0].like" -> ["wheres", "[0]", "like"]
	segments := parsePathSegments(path)

	for _, segment := range segments {
		if segment == "" {
			continue
		}

		// Check if this is an array index like "[0]"
		if strings.HasPrefix(segment, "[") && strings.HasSuffix(segment, "]") {
			indexStr := segment[1 : len(segment)-1]
			index, err := strconv.Atoi(indexStr)
			if err != nil {
				return nil
			}

			arr, ok := current.([]interface{})
			if !ok {
				return nil
			}

			if index < 0 || index >= len(arr) {
				return nil
			}
			current = arr[index]
		} else {
			// Regular field access
			switch v := current.(type) {
			case map[string]interface{}:
				current = v[segment]
			default:
				return nil
			}
		}
	}

	return current
}

// parsePathSegments splits a path like "wheres[0].like" into ["wheres", "[0]", "like"]
func parsePathSegments(path string) []string {
	var segments []string
	var current strings.Builder

	for i := 0; i < len(path); i++ {
		ch := path[i]
		switch ch {
		case '.':
			if current.Len() > 0 {
				segments = append(segments, current.String())
				current.Reset()
			}
		case '[':
			if current.Len() > 0 {
				segments = append(segments, current.String())
				current.Reset()
			}
			// Find the closing bracket
			j := i + 1
			for j < len(path) && path[j] != ']' {
				j++
			}
			if j < len(path) {
				segments = append(segments, path[i:j+1]) // Include "[" and "]"
				i = j
			}
		default:
			current.WriteByte(ch)
		}
	}

	if current.Len() > 0 {
		segments = append(segments, current.String())
	}

	return segments
}

// assertRegex checks if output matches a regex pattern
func (a *Asserter) assertRegex(assertion *Assertion, output interface{}) *AssertionResult {
	result := &AssertionResult{
		Assertion: assertion,
		Actual:    output,
		Expected:  assertion.Value,
	}

	pattern, ok := assertion.Value.(string)
	if !ok {
		result.Passed = false
		result.Message = "regex pattern must be a string"
		return result
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		result.Passed = false
		result.Message = fmt.Sprintf("invalid regex pattern: %s", err.Error())
		return result
	}

	outputStr := a.toString(output)
	if re.MatchString(outputStr) {
		result.Passed = true
		result.Message = fmt.Sprintf("output matches pattern '%s'", pattern)
	} else {
		result.Passed = false
		result.Message = fmt.Sprintf("output does not match pattern '%s'", pattern)
	}

	return result
}

// assertType checks the type of the output
func (a *Asserter) assertType(assertion *Assertion, output interface{}) *AssertionResult {
	result := &AssertionResult{
		Assertion: assertion,
		Actual:    output,
		Expected:  assertion.Value,
	}

	expectedType, ok := assertion.Value.(string)
	if !ok {
		result.Passed = false
		result.Message = "type assertion value must be a string"
		return result
	}

	actualType := a.getType(output)
	result.Actual = actualType

	if actualType == expectedType {
		result.Passed = true
		result.Message = fmt.Sprintf("output is of type '%s'", expectedType)
	} else {
		result.Passed = false
		result.Message = fmt.Sprintf("expected type '%s', got '%s'", expectedType, actualType)
	}

	return result
}

// getType returns the type name of a value
func (a *Asserter) getType(v interface{}) string {
	if v == nil {
		return "null"
	}

	switch v.(type) {
	case string:
		return "string"
	case float64, float32, int, int64, int32:
		return "number"
	case bool:
		return "boolean"
	case []interface{}:
		return "array"
	case map[string]interface{}:
		return "object"
	default:
		return fmt.Sprintf("%T", v)
	}
}

// assertAgent uses an agent to validate the output
func (a *Asserter) assertAgent(assertion *Assertion, output, input interface{}) *AssertionResult {
	result := &AssertionResult{
		Assertion: assertion,
		Actual:    output,
	}

	// Parse use field: "agents:tests.validator-agent"
	if !strings.HasPrefix(assertion.Use, "agents:") {
		result.Passed = false
		result.Message = "agent assertion requires 'use' field with 'agents:' prefix"
		return result
	}

	agentID := strings.TrimPrefix(assertion.Use, "agents:")

	// Get assistant
	ast, err := assistant.Get(agentID)
	if err != nil {
		result.Passed = false
		result.Message = fmt.Sprintf("failed to get validator agent: %s", err.Error())
		return result
	}

	// Build validation request
	validationInput := map[string]interface{}{
		"output": output,
		"input":  input,
	}

	// Add criteria from Value field
	if assertion.Value != nil {
		validationInput["criteria"] = assertion.Value
	}

	// Add metadata from options
	if assertion.Options != nil && assertion.Options.Metadata != nil {
		for k, v := range assertion.Options.Metadata {
			validationInput[k] = v
		}
	}

	// Build context options - skip history and trace for validator
	opts := &context.Options{
		Skip: &context.Skip{
			History: true,
			Trace:   true,
			Output:  true,
		},
		Metadata: map[string]interface{}{
			"test_mode": "validator",
		},
	}
	if assertion.Options != nil && assertion.Options.Connector != "" {
		opts.Connector = assertion.Options.Connector
	}

	// Create context and call agent
	env := NewEnvironment("", "")
	ctx := NewTestContext("validator", agentID, env)
	defer ctx.Release()

	// Convert validation input to JSON string for the message
	inputJSON, err := json.Marshal(validationInput)
	if err != nil {
		result.Passed = false
		result.Message = fmt.Sprintf("failed to marshal validation input: %s", err.Error())
		return result
	}

	messages := []context.Message{{
		Role:    context.RoleUser,
		Content: string(inputJSON),
	}}

	response, err := ast.Stream(ctx, messages, opts)
	if err != nil {
		result.Passed = false
		result.Message = fmt.Sprintf("validator agent error: %s", err.Error())
		return result
	}

	// Parse response
	return a.parseValidatorResponse(response, result)
}

// parseValidatorResponse parses the validator agent's response
func (a *Asserter) parseValidatorResponse(response *context.Response, result *AssertionResult) *AssertionResult {
	output := extractValidatorOutput(response)

	// Expected format: { "passed": bool, "reason": string, "score": float, "suggestions": [] }
	if outputMap, ok := output.(map[string]interface{}); ok {
		if passed, ok := outputMap["passed"].(bool); ok {
			result.Passed = passed
		} else {
			result.Passed = false
			result.Message = "validator response missing 'passed' field"
			return result
		}
		if reason, ok := outputMap["reason"].(string); ok {
			result.Message = reason
		}
		// Store score and suggestions in expected field for reference
		result.Expected = outputMap
	} else {
		result.Passed = false
		result.Message = "validator agent returned invalid response format"
	}

	return result
}

// extractValidatorOutput extracts the output from a validator response
func extractValidatorOutput(response *context.Response) interface{} {
	if response == nil || response.Completion == nil {
		return nil
	}

	// Get content from completion
	content := response.Completion.Content
	if content == nil {
		return nil
	}

	// Try to get text content
	var text string
	switch v := content.(type) {
	case string:
		text = v
	default:
		// Try to marshal and use as-is
		data, err := json.Marshal(content)
		if err != nil {
			return nil
		}
		text = string(data)
	}

	if text == "" {
		return nil
	}

	// Use gou/text to extract JSON (handles markdown code blocks, auto-repair, etc.)
	result := goutext.ExtractJSON(text)
	if result != nil {
		return result
	}

	// Return raw text if extraction fails
	return text
}

// assertScript runs a custom assertion script
func (a *Asserter) assertScript(assertion *Assertion, output, input interface{}) *AssertionResult {
	result := &AssertionResult{
		Assertion: assertion,
		Actual:    output,
	}

	if assertion.Script == "" {
		result.Passed = false
		result.Message = "script assertion requires a script name"
		return result
	}

	// Build script arguments
	args := []interface{}{
		output,
		input,
		assertion.Value,
	}

	// Run the script as a process
	p, err := process.Of(assertion.Script, args...)
	if err != nil {
		result.Passed = false
		result.Message = fmt.Sprintf("failed to create process: %s", err.Error())
		return result
	}

	res, err := p.Exec()
	if err != nil {
		result.Passed = false
		result.Message = fmt.Sprintf("script execution failed: %s", err.Error())
		return result
	}

	// Parse script result
	// Expected format: { "pass": bool, "message": string }
	switch v := res.(type) {
	case bool:
		result.Passed = v
		if v {
			result.Message = "script assertion passed"
		} else {
			result.Message = "script assertion failed"
		}

	case map[string]interface{}:
		if pass, ok := v["pass"].(bool); ok {
			result.Passed = pass
		}
		if msg, ok := v["message"].(string); ok {
			result.Message = msg
		}

	default:
		result.Passed = false
		result.Message = fmt.Sprintf("script returned unexpected type: %T", res)
	}

	return result
}

// assertToolCalled checks if a specific tool was called
// value can be:
// - string: exact tool name to match
// - []string: any of the tool names
// - map with "name" and optional "arguments" for more specific matching
func (a *Asserter) assertToolCalled(assertion *Assertion) *AssertionResult {
	result := &AssertionResult{
		Assertion: assertion,
		Expected:  assertion.Value,
	}

	if a.response == nil {
		result.Passed = false
		result.Message = "no response available for tool_called assertion"
		return result
	}

	if len(a.response.Tools) == 0 {
		result.Passed = false
		result.Message = "no tools were called"
		return result
	}

	// Get tool names that were called
	calledTools := make([]string, 0, len(a.response.Tools))
	for _, tool := range a.response.Tools {
		calledTools = append(calledTools, tool.Tool)
	}
	result.Actual = calledTools

	switch v := assertion.Value.(type) {
	case string:
		// Simple case: check if tool name matches (supports prefix matching)
		for _, tool := range a.response.Tools {
			if matchToolName(tool.Tool, v) {
				result.Passed = true
				result.Message = fmt.Sprintf("tool '%s' was called", v)
				return result
			}
		}
		result.Passed = false
		result.Message = fmt.Sprintf("tool '%s' was not called, called: %v", v, calledTools)

	case []interface{}:
		// Check if any of the specified tools were called
		for _, expected := range v {
			if expectedStr, ok := expected.(string); ok {
				for _, tool := range a.response.Tools {
					if matchToolName(tool.Tool, expectedStr) {
						result.Passed = true
						result.Message = fmt.Sprintf("tool '%s' was called", expectedStr)
						return result
					}
				}
			}
		}
		result.Passed = false
		result.Message = fmt.Sprintf("none of the expected tools were called, called: %v", calledTools)

	case map[string]interface{}:
		// Advanced case: match name and optionally arguments
		expectedName, _ := v["name"].(string)
		expectedArgs := v["arguments"]

		for _, tool := range a.response.Tools {
			if matchToolName(tool.Tool, expectedName) {
				// If arguments specified, check them too
				if expectedArgs != nil {
					if matchArguments(tool.Arguments, expectedArgs) {
						result.Passed = true
						result.Message = fmt.Sprintf("tool '%s' was called with matching arguments", expectedName)
						return result
					}
				} else {
					result.Passed = true
					result.Message = fmt.Sprintf("tool '%s' was called", expectedName)
					return result
				}
			}
		}
		result.Passed = false
		if expectedArgs != nil {
			result.Message = fmt.Sprintf("tool '%s' was not called with expected arguments", expectedName)
		} else {
			result.Message = fmt.Sprintf("tool '%s' was not called, called: %v", expectedName, calledTools)
		}

	default:
		result.Passed = false
		result.Message = fmt.Sprintf("invalid tool_called value type: %T", assertion.Value)
	}

	return result
}

// assertToolResult checks the result of a tool call
// value should be a map with "tool" (name) and "result" (expected result pattern)
func (a *Asserter) assertToolResult(assertion *Assertion) *AssertionResult {
	result := &AssertionResult{
		Assertion: assertion,
		Expected:  assertion.Value,
	}

	if a.response == nil {
		result.Passed = false
		result.Message = "no response available for tool_result assertion"
		return result
	}

	if len(a.response.Tools) == 0 {
		result.Passed = false
		result.Message = "no tools were called"
		return result
	}

	spec, ok := assertion.Value.(map[string]interface{})
	if !ok {
		result.Passed = false
		result.Message = "tool_result assertion requires a map with 'tool' and 'result' fields"
		return result
	}

	toolName, _ := spec["tool"].(string)
	expectedResult := spec["result"]

	if toolName == "" {
		result.Passed = false
		result.Message = "tool_result assertion requires 'tool' field"
		return result
	}

	// Find the tool call
	for _, tool := range a.response.Tools {
		if matchToolName(tool.Tool, toolName) {
			result.Actual = tool.Result

			// Check if there was an error
			if tool.Error != "" {
				result.Passed = false
				result.Message = fmt.Sprintf("tool '%s' returned error: %s", toolName, tool.Error)
				return result
			}

			// If no expected result specified, just check success (no error)
			if expectedResult == nil {
				result.Passed = true
				result.Message = fmt.Sprintf("tool '%s' executed successfully", toolName)
				return result
			}

			// Match result
			if matchResult(tool.Result, expectedResult) {
				result.Passed = true
				result.Message = fmt.Sprintf("tool '%s' result matches expected", toolName)
				return result
			}

			result.Passed = false
			result.Message = fmt.Sprintf("tool '%s' result does not match expected", toolName)
			return result
		}
	}

	result.Passed = false
	result.Message = fmt.Sprintf("tool '%s' was not called", toolName)
	return result
}

// matchToolName checks if a tool name matches the expected pattern
// Supports exact match and suffix match (e.g., "setup" matches "agents_expense_tools__setup")
func matchToolName(actual, expected string) bool {
	if actual == expected {
		return true
	}
	// Support suffix matching (tool name without namespace prefix)
	if strings.HasSuffix(actual, "__"+expected) || strings.HasSuffix(actual, "."+expected) {
		return true
	}
	// Support contains matching for partial names
	if strings.Contains(actual, expected) {
		return true
	}
	return false
}

// matchArguments checks if tool arguments match expected pattern
func matchArguments(actual, expected interface{}) bool {
	expectedMap, ok := expected.(map[string]interface{})
	if !ok {
		return false
	}

	actualMap, ok := actual.(map[string]interface{})
	if !ok {
		// Try parsing as JSON string
		if actualStr, ok := actual.(string); ok {
			var parsed map[string]interface{}
			if err := jsoniter.UnmarshalFromString(actualStr, &parsed); err == nil {
				actualMap = parsed
			} else {
				return false
			}
		} else {
			return false
		}
	}

	// Check that all expected keys exist and match
	for key, expectedVal := range expectedMap {
		actualVal, exists := actualMap[key]
		if !exists {
			return false
		}
		if !validateOutput(actualVal, expectedVal) {
			return false
		}
	}
	return true
}

// matchResult checks if tool result matches expected pattern
func matchResult(actual, expected interface{}) bool {
	switch exp := expected.(type) {
	case map[string]interface{}:
		actualMap, ok := actual.(map[string]interface{})
		if !ok {
			return false
		}
		// Check that all expected keys exist and match
		for key, expectedVal := range exp {
			actualVal, exists := actualMap[key]
			if !exists {
				return false
			}
			if !matchResult(actualVal, expectedVal) {
				return false
			}
		}
		return true

	case string:
		// Support regex pattern matching for strings
		if strings.HasPrefix(exp, "regex:") {
			pattern := strings.TrimPrefix(exp, "regex:")
			re, err := regexp.Compile(pattern)
			if err != nil {
				return false
			}
			actualStr := fmt.Sprintf("%v", actual)
			return re.MatchString(actualStr)
		}
		return fmt.Sprintf("%v", actual) == exp

	case bool:
		actualBool, ok := actual.(bool)
		return ok && actualBool == exp

	default:
		return validateOutput(actual, expected)
	}
}

// toString converts a value to string for comparison
func (a *Asserter) toString(v interface{}) string {
	if v == nil {
		return ""
	}

	switch val := v.(type) {
	case string:
		return val
	case []byte:
		return string(val)
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", v)
		}
		return string(b)
	}
}
