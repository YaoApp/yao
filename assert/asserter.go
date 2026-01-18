package assert

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/yaoapp/gou/text"
)

// Asserter handles assertions/validations
type Asserter struct {
	agentValidator AgentValidator
	scriptRunner   ScriptRunner
}

// New creates a new Asserter
func New() *Asserter {
	return &Asserter{}
}

// WithAgentValidator sets the agent validator for agent-type assertions
func (a *Asserter) WithAgentValidator(v AgentValidator) *Asserter {
	a.agentValidator = v
	return a
}

// WithScriptRunner sets the script runner for script-type assertions
func (a *Asserter) WithScriptRunner(r ScriptRunner) *Asserter {
	a.scriptRunner = r
	return a
}

// Validate validates output against a list of assertions
// Returns (passed, error message)
func (a *Asserter) Validate(assertions []*Assertion, output interface{}) (bool, string) {
	if len(assertions) == 0 {
		return true, ""
	}

	var failures []string
	for _, assertion := range assertions {
		result := a.Evaluate(assertion, output, nil)
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

// ValidateWithDetails validates output and returns detailed results
func (a *Asserter) ValidateWithDetails(assertions []*Assertion, output interface{}) *Result {
	if len(assertions) == 0 {
		return &Result{Passed: true}
	}

	if len(assertions) == 1 {
		return a.Evaluate(assertions[0], output, nil)
	}

	var failures []string
	for _, assertion := range assertions {
		result := a.Evaluate(assertion, output, nil)
		if !result.Passed {
			msg := result.Message
			if assertion.Message != "" {
				msg = assertion.Message
			}
			failures = append(failures, msg)
		}
	}

	if len(failures) > 0 {
		return &Result{
			Passed:  false,
			Message: strings.Join(failures, "; "),
		}
	}
	return &Result{Passed: true}
}

// Evaluate evaluates a single assertion
func (a *Asserter) Evaluate(assertion *Assertion, output, input interface{}) *Result {
	result := &Result{
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
func (a *Asserter) assertEquals(assertion *Assertion, output interface{}) *Result {
	result := &Result{
		Assertion: assertion,
		Actual:    output,
		Expected:  assertion.Value,
	}

	if ValidateOutput(output, assertion.Value) {
		result.Passed = true
		result.Message = "values are equal"
	} else {
		result.Passed = false
		result.Message = fmt.Sprintf("expected %v, got %v", assertion.Value, output)
	}

	return result
}

// assertContains checks if output contains the expected value
func (a *Asserter) assertContains(assertion *Assertion, output interface{}) *Result {
	result := &Result{
		Assertion: assertion,
		Actual:    output,
		Expected:  assertion.Value,
	}

	outputStr := ToString(output)
	expectedStr := ToString(assertion.Value)

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
func (a *Asserter) assertNotContains(assertion *Assertion, output interface{}) *Result {
	result := a.assertContains(assertion, output)
	result.Passed = !result.Passed
	if result.Passed {
		result.Message = fmt.Sprintf("output does not contain '%s'", ToString(assertion.Value))
	} else {
		result.Message = fmt.Sprintf("output should not contain '%s'", ToString(assertion.Value))
	}
	return result
}

// assertJSONPath extracts a value using JSON path and compares
func (a *Asserter) assertJSONPath(assertion *Assertion, output interface{}) *Result {
	result := &Result{
		Assertion: assertion,
		Expected:  assertion.Value,
	}

	// Convert output to JSON if needed
	var jsonData interface{}
	switch v := output.(type) {
	case string:
		extracted := text.ExtractJSON(v)
		if extracted != nil {
			jsonData = extracted
		} else {
			result.Passed = false
			result.Message = fmt.Sprintf("output is not valid JSON: %s", TruncateOutput(v, 100))
			return result
		}
	case map[string]interface{}, []interface{}:
		jsonData = v
	default:
		result.Passed = false
		result.Message = fmt.Sprintf("output is not a JSON object or array, got: %T", output)
		return result
	}

	// Extract value using path
	path := strings.TrimPrefix(assertion.Path, "$.")
	actual := ExtractPath(jsonData, path)
	result.Actual = actual

	// Compare
	if ValidateOutput(actual, assertion.Value) {
		result.Passed = true
		result.Message = fmt.Sprintf("path '%s' equals expected value", assertion.Path)
		return result
	}

	// IN semantics: if expected is array, check if actual matches any element
	if expectedArr, ok := assertion.Value.([]interface{}); ok {
		if _, actualIsArr := actual.([]interface{}); !actualIsArr {
			for _, expectedItem := range expectedArr {
				if ValidateOutput(actual, expectedItem) {
					result.Passed = true
					result.Message = fmt.Sprintf("path '%s' equals one of expected values", assertion.Path)
					return result
				}
			}
		}
	}

	result.Passed = false
	result.Message = fmt.Sprintf("path '%s': expected %v, got %v", assertion.Path, assertion.Value, actual)
	return result
}

// assertRegex checks if output matches a regex pattern
func (a *Asserter) assertRegex(assertion *Assertion, output interface{}) *Result {
	result := &Result{
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

	outputStr := ToString(output)
	if re.MatchString(outputStr) {
		result.Passed = true
		result.Message = fmt.Sprintf("output matches pattern '%s'", pattern)
	} else {
		result.Passed = false
		result.Message = fmt.Sprintf("output does not match pattern '%s'", pattern)
	}

	return result
}

// assertType checks the type of the output (or a nested field if path is specified)
func (a *Asserter) assertType(assertion *Assertion, output interface{}) *Result {
	result := &Result{
		Assertion: assertion,
		Expected:  assertion.Value,
	}

	expectedType, ok := assertion.Value.(string)
	if !ok {
		result.Passed = false
		result.Message = "type assertion value must be a string"
		return result
	}

	// If path is specified, extract the value first
	var valueToCheck interface{} = output
	if assertion.Path != "" {
		// Convert output to JSON if needed
		var jsonData interface{}
		switch v := output.(type) {
		case string:
			extracted := text.ExtractJSON(v)
			if extracted != nil {
				jsonData = extracted
			} else {
				result.Passed = false
				result.Message = fmt.Sprintf("output is not valid JSON: %s", TruncateOutput(v, 100))
				return result
			}
		case map[string]interface{}, []interface{}:
			jsonData = v
		default:
			result.Passed = false
			result.Message = fmt.Sprintf("output is not a JSON object or array, got: %T", output)
			return result
		}

		// Extract value using path
		path := strings.TrimPrefix(assertion.Path, "$.")
		valueToCheck = ExtractPath(jsonData, path)
		if valueToCheck == nil {
			result.Passed = false
			result.Actual = nil
			result.Message = fmt.Sprintf("path '%s' not found in output", assertion.Path)
			return result
		}
	}

	result.Actual = valueToCheck
	actualType := GetType(valueToCheck)

	if actualType == expectedType {
		result.Passed = true
		if assertion.Path != "" {
			result.Message = fmt.Sprintf("path '%s' is of type '%s'", assertion.Path, expectedType)
		} else {
			result.Message = fmt.Sprintf("output is of type '%s'", expectedType)
		}
	} else {
		result.Passed = false
		if assertion.Path != "" {
			result.Message = fmt.Sprintf("path '%s': expected type '%s', got '%s'", assertion.Path, expectedType, actualType)
		} else {
			result.Message = fmt.Sprintf("expected type '%s', got '%s'", expectedType, actualType)
		}
	}

	return result
}

// assertScript runs a custom assertion script
func (a *Asserter) assertScript(assertion *Assertion, output, input interface{}) *Result {
	result := &Result{
		Assertion: assertion,
		Actual:    output,
	}

	if a.scriptRunner == nil {
		result.Passed = false
		result.Message = "script assertions require a ScriptRunner to be configured"
		return result
	}

	scriptName := assertion.Script
	if scriptName == "" {
		result.Passed = false
		result.Message = "script assertion requires a script name"
		return result
	}

	passed, message, err := a.scriptRunner.Run(scriptName, output, input, assertion.Value)
	if err != nil {
		result.Passed = false
		result.Message = fmt.Sprintf("script execution failed: %s", err.Error())
		return result
	}

	result.Passed = passed
	result.Message = message
	return result
}

// assertAgent uses an agent to validate the output
func (a *Asserter) assertAgent(assertion *Assertion, output, input interface{}) *Result {
	result := &Result{
		Assertion: assertion,
		Actual:    output,
	}

	if a.agentValidator == nil {
		result.Passed = false
		result.Message = "agent assertions require an AgentValidator to be configured"
		return result
	}

	// Parse use field: "agents:validator"
	if !strings.HasPrefix(assertion.Use, "agents:") {
		result.Passed = false
		result.Message = "agent assertion requires 'use' field with 'agents:' prefix"
		return result
	}

	agentID := strings.TrimPrefix(assertion.Use, "agents:")
	return a.agentValidator.Validate(agentID, output, input, assertion.Value, assertion.Options)
}

// ParseAssertions parses assertion definitions into Assertion objects
func ParseAssertions(input interface{}) []*Assertion {
	if input == nil {
		return nil
	}

	var assertions []*Assertion

	switch v := input.(type) {
	case map[string]interface{}:
		assertion := mapToAssertion(v)
		if assertion != nil {
			assertions = append(assertions, assertion)
		}

	case []interface{}:
		for _, item := range v {
			if m, ok := item.(map[string]interface{}); ok {
				assertion := mapToAssertion(m)
				if assertion != nil {
					assertions = append(assertions, assertion)
				}
			}
		}

	case string:
		assertions = append(assertions, &Assertion{Type: v})
	}

	return assertions
}

// mapToAssertion converts a map to an Assertion
func mapToAssertion(m map[string]interface{}) *Assertion {
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
