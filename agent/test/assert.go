package test

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/process"
)

// Asserter handles test assertions
type Asserter struct{}

// NewAsserter creates a new asserter
func NewAsserter() *Asserter {
	return &Asserter{}
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
	if msg, ok := m["message"].(string); ok {
		assertion.Message = msg
	}
	if n, ok := m["negate"].(bool); ok {
		assertion.Negate = n
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
		// Try to parse as JSON
		if err := jsoniter.Unmarshal([]byte(v), &jsonData); err != nil {
			// Try to extract JSON from markdown code blocks
			extracted := extractJSONFromText(v)
			if extracted != nil {
				jsonData = extracted
			} else {
				result.Passed = false
				result.Message = fmt.Sprintf("output is not valid JSON: %s", err.Error())
				return result
			}
		}
	case map[string]interface{}, []interface{}:
		jsonData = v
	default:
		result.Passed = false
		result.Message = "output is not a JSON object or array"
		return result
	}

	// Extract value using simple path (e.g., "$.need_search" or "need_search")
	path := strings.TrimPrefix(assertion.Path, "$.")
	actual := a.extractPath(jsonData, path)
	result.Actual = actual

	if validateOutput(actual, assertion.Value) {
		result.Passed = true
		result.Message = fmt.Sprintf("path '%s' equals expected value", assertion.Path)
	} else {
		result.Passed = false
		result.Message = fmt.Sprintf("path '%s': expected %v, got %v", assertion.Path, assertion.Value, actual)
	}

	return result
}

// extractPath extracts a value from JSON using a simple dot-notation path
func (a *Asserter) extractPath(data interface{}, path string) interface{} {
	parts := strings.Split(path, ".")
	current := data

	for _, part := range parts {
		if part == "" {
			continue
		}

		switch v := current.(type) {
		case map[string]interface{}:
			current = v[part]
		default:
			return nil
		}
	}

	return current
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

// extractJSONFromText tries to extract JSON from text (e.g., markdown code blocks)
func extractJSONFromText(text string) interface{} {
	// Try to find JSON in code blocks
	patterns := []string{
		"```json\n",
		"```\n",
	}

	for _, start := range patterns {
		if idx := strings.Index(text, start); idx >= 0 {
			text = text[idx+len(start):]
			if endIdx := strings.Index(text, "```"); endIdx >= 0 {
				text = text[:endIdx]
			}
			break
		}
	}

	// Try to parse
	var result interface{}
	if err := jsoniter.Unmarshal([]byte(strings.TrimSpace(text)), &result); err == nil {
		return result
	}

	return nil
}
