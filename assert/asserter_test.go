package assert

import (
	"errors"
	"testing"
)

func TestAsserterEquals(t *testing.T) {
	a := New()

	tests := []struct {
		name     string
		value    interface{}
		output   interface{}
		expected bool
	}{
		{"string match", "hello", "hello", true},
		{"string mismatch", "hello", "world", false},
		{"number match", 42, 42, true},
		{"number mismatch", 42, 43, false},
		{"bool match", true, true, true},
		{"bool mismatch", true, false, false},
		{"map match", map[string]interface{}{"a": 1}, map[string]interface{}{"a": 1}, true},
		{"map mismatch", map[string]interface{}{"a": 1}, map[string]interface{}{"a": 2}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertion := &Assertion{
				Type:  "equals",
				Value: tt.value,
			}
			result := a.Evaluate(assertion, tt.output, nil)
			if result.Passed != tt.expected {
				t.Errorf("expected passed=%v, got passed=%v", tt.expected, result.Passed)
			}
		})
	}
}

func TestAsserterContains(t *testing.T) {
	a := New()

	tests := []struct {
		name     string
		value    string
		output   string
		expected bool
	}{
		{"contains substring", "world", "hello world", true},
		{"does not contain", "foo", "hello world", false},
		{"exact match", "hello", "hello", true},
		{"empty string", "", "hello", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertion := &Assertion{
				Type:  "contains",
				Value: tt.value,
			}
			result := a.Evaluate(assertion, tt.output, nil)
			if result.Passed != tt.expected {
				t.Errorf("expected passed=%v, got passed=%v", tt.expected, result.Passed)
			}
		})
	}
}

func TestAsserterNotContains(t *testing.T) {
	a := New()

	tests := []struct {
		name     string
		value    string
		output   string
		expected bool
	}{
		{"does not contain", "foo", "hello world", true},
		{"contains substring", "world", "hello world", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertion := &Assertion{
				Type:  "not_contains",
				Value: tt.value,
			}
			result := a.Evaluate(assertion, tt.output, nil)
			if result.Passed != tt.expected {
				t.Errorf("expected passed=%v, got passed=%v", tt.expected, result.Passed)
			}
		})
	}
}

func TestAsserterJSONPath(t *testing.T) {
	a := New()

	output := map[string]interface{}{
		"name":  "test",
		"count": 42,
		"nested": map[string]interface{}{
			"value": "deep",
		},
		"items": []interface{}{"a", "b", "c"},
	}

	tests := []struct {
		name     string
		path     string
		value    interface{}
		expected bool
	}{
		{"simple field", "name", "test", true},
		{"number field", "count", float64(42), true},
		{"nested field", "nested.value", "deep", true},
		{"array index", "items[0]", "a", true},
		{"array index 2", "items[2]", "c", true},
		{"wrong value", "name", "wrong", false},
		{"non-existent path", "missing", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertion := &Assertion{
				Type:  "json_path",
				Path:  tt.path,
				Value: tt.value,
			}
			result := a.Evaluate(assertion, output, nil)
			if result.Passed != tt.expected {
				t.Errorf("expected passed=%v, got passed=%v, message=%s", tt.expected, result.Passed, result.Message)
			}
		})
	}
}

func TestAsserterRegex(t *testing.T) {
	a := New()

	tests := []struct {
		name     string
		pattern  string
		output   string
		expected bool
	}{
		{"simple match", "hello", "hello world", true},
		{"regex pattern", "^\\d+$", "12345", true},
		{"regex no match", "^\\d+$", "abc", false},
		{"email pattern", `\w+@\w+\.\w+`, "test@example.com", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertion := &Assertion{
				Type:  "regex",
				Value: tt.pattern,
			}
			result := a.Evaluate(assertion, tt.output, nil)
			if result.Passed != tt.expected {
				t.Errorf("expected passed=%v, got passed=%v", tt.expected, result.Passed)
			}
		})
	}
}

func TestAsserterType(t *testing.T) {
	a := New()

	tests := []struct {
		name         string
		expectedType string
		output       interface{}
		expected     bool
	}{
		{"string type", "string", "hello", true},
		{"number type", "number", 42, true},
		{"number type float", "number", 3.14, true},
		{"boolean type", "boolean", true, true},
		{"array type", "array", []interface{}{1, 2, 3}, true},
		{"object type", "object", map[string]interface{}{"a": 1}, true},
		{"null type", "null", nil, true},
		{"wrong type", "string", 42, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertion := &Assertion{
				Type:  "type",
				Value: tt.expectedType,
			}
			result := a.Evaluate(assertion, tt.output, nil)
			if result.Passed != tt.expected {
				t.Errorf("expected passed=%v, got passed=%v", tt.expected, result.Passed)
			}
		})
	}
}

func TestAsserterTypeWithPath(t *testing.T) {
	a := New()

	// Test data with nested structure
	output := map[string]interface{}{
		"name":    "test",
		"count":   float64(42),
		"items":   []interface{}{"a", "b", "c"},
		"enabled": true,
		"nested": map[string]interface{}{
			"value": "nested_value",
		},
	}

	tests := []struct {
		name         string
		path         string
		expectedType string
		expected     bool
	}{
		{"string field", "name", "string", true},
		{"number field", "count", "number", true},
		{"array field", "items", "array", true},
		{"boolean field", "enabled", "boolean", true},
		{"object field", "nested", "object", true},
		{"nested string field", "nested.value", "string", true},
		{"wrong type for field", "name", "number", false},
		{"non-existent path", "missing", "string", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertion := &Assertion{
				Type:  "type",
				Path:  tt.path,
				Value: tt.expectedType,
			}
			result := a.Evaluate(assertion, output, nil)
			if result.Passed != tt.expected {
				t.Errorf("expected passed=%v, got passed=%v, message=%s", tt.expected, result.Passed, result.Message)
			}
		})
	}
}

func TestAsserterNegate(t *testing.T) {
	a := New()

	// Test negation
	assertion := &Assertion{
		Type:   "equals",
		Value:  "hello",
		Negate: true,
	}

	// Should fail because "hello" == "hello", but negate inverts it
	result := a.Evaluate(assertion, "hello", nil)
	if result.Passed {
		t.Error("negated equals should fail when values match")
	}

	// Should pass because "hello" != "world", and negate inverts it
	result = a.Evaluate(assertion, "world", nil)
	if !result.Passed {
		t.Error("negated equals should pass when values don't match")
	}
}

func TestAsserterValidate(t *testing.T) {
	a := New()

	assertions := []*Assertion{
		{Type: "type", Value: "object"},
		{Type: "json_path", Path: "name", Value: "test"},
		{Type: "json_path", Path: "count", Value: float64(42)},
	}

	output := map[string]interface{}{
		"name":  "test",
		"count": 42,
	}

	passed, message := a.Validate(assertions, output)
	if !passed {
		t.Errorf("validation should pass, got message: %s", message)
	}

	// Test with failing assertion
	assertions = append(assertions, &Assertion{
		Type:  "json_path",
		Path:  "name",
		Value: "wrong",
	})

	passed, message = a.Validate(assertions, output)
	if passed {
		t.Error("validation should fail with wrong value")
	}
}

func TestParseAssertions(t *testing.T) {
	// Test map input
	input := map[string]interface{}{
		"type":  "contains",
		"value": "hello",
	}
	assertions := ParseAssertions(input)
	if len(assertions) != 1 {
		t.Errorf("expected 1 assertion, got %d", len(assertions))
	}
	if assertions[0].Type != "contains" {
		t.Errorf("expected type 'contains', got '%s'", assertions[0].Type)
	}

	// Test array input
	input2 := []interface{}{
		map[string]interface{}{"type": "equals", "value": 1},
		map[string]interface{}{"type": "contains", "value": "test"},
	}
	assertions = ParseAssertions(input2)
	if len(assertions) != 2 {
		t.Errorf("expected 2 assertions, got %d", len(assertions))
	}

	// Test string input
	assertions = ParseAssertions("contains")
	if len(assertions) != 1 {
		t.Errorf("expected 1 assertion, got %d", len(assertions))
	}
	if assertions[0].Type != "contains" {
		t.Errorf("expected type 'contains', got '%s'", assertions[0].Type)
	}
}

func TestExtractPath(t *testing.T) {
	data := map[string]interface{}{
		"name": "test",
		"nested": map[string]interface{}{
			"value": "deep",
		},
		"items": []interface{}{
			map[string]interface{}{"id": 1},
			map[string]interface{}{"id": 2},
		},
	}

	tests := []struct {
		path     string
		expected interface{}
	}{
		{"name", "test"},
		{"nested.value", "deep"},
		{"items[0].id", 1},
		{"items[1].id", 2},
		{"missing", nil},
		{"nested.missing", nil},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := ExtractPath(data, tt.path)
			if !ValidateOutput(result, tt.expected) {
				t.Errorf("path '%s': expected %v, got %v", tt.path, tt.expected, result)
			}
		})
	}
}

// ============================================================================
// Additional tests for improved coverage
// ============================================================================

// Mock implementations for testing
type mockScriptRunner struct {
	passed  bool
	message string
	err     error
}

func (m *mockScriptRunner) Run(scriptName string, output, input, expected interface{}) (bool, string, error) {
	return m.passed, m.message, m.err
}

type mockAgentValidator struct {
	result *Result
}

func (m *mockAgentValidator) Validate(agentID string, output, input, criteria interface{}, options *AssertionOptions) *Result {
	return m.result
}

// Test WithAgentValidator and WithScriptRunner
func TestAsserterConfiguration(t *testing.T) {
	a := New()

	// Test chaining
	mockAgent := &mockAgentValidator{}
	mockScript := &mockScriptRunner{}

	result := a.WithAgentValidator(mockAgent).WithScriptRunner(mockScript)

	if result != a {
		t.Error("WithAgentValidator should return the same asserter for chaining")
	}
	if a.agentValidator != mockAgent {
		t.Error("agentValidator should be set")
	}
	if a.scriptRunner != mockScript {
		t.Error("scriptRunner should be set")
	}
}

// Test ValidateWithDetails
func TestAsserterValidateWithDetails(t *testing.T) {
	a := New()

	t.Run("empty assertions", func(t *testing.T) {
		result := a.ValidateWithDetails([]*Assertion{}, "output")
		if !result.Passed {
			t.Error("empty assertions should pass")
		}
	})

	t.Run("single assertion pass", func(t *testing.T) {
		result := a.ValidateWithDetails([]*Assertion{
			{Type: "equals", Value: "hello"},
		}, "hello")
		if !result.Passed {
			t.Error("single matching assertion should pass")
		}
	})

	t.Run("single assertion fail", func(t *testing.T) {
		result := a.ValidateWithDetails([]*Assertion{
			{Type: "equals", Value: "hello"},
		}, "world")
		if result.Passed {
			t.Error("single non-matching assertion should fail")
		}
	})

	t.Run("multiple assertions with custom message", func(t *testing.T) {
		result := a.ValidateWithDetails([]*Assertion{
			{Type: "equals", Value: "hello"},
			{Type: "contains", Value: "world", Message: "custom failure message"},
		}, "hello")
		if result.Passed {
			t.Error("should fail when one assertion fails")
		}
		if result.Message != "custom failure message" {
			t.Errorf("should use custom message, got: %s", result.Message)
		}
	})

	t.Run("multiple assertions all pass", func(t *testing.T) {
		result := a.ValidateWithDetails([]*Assertion{
			{Type: "contains", Value: "hello"},
			{Type: "contains", Value: "world"},
		}, "hello world")
		if !result.Passed {
			t.Error("all matching assertions should pass")
		}
	})
}

// Test unknown assertion type
func TestAsserterUnknownType(t *testing.T) {
	a := New()

	assertion := &Assertion{
		Type:  "unknown_type",
		Value: "test",
	}
	result := a.Evaluate(assertion, "test", nil)
	if result.Passed {
		t.Error("unknown assertion type should fail")
	}
	if result.Message != "unknown assertion type: unknown_type" {
		t.Errorf("unexpected message: %s", result.Message)
	}
}

// Test default type (empty string = equals)
func TestAsserterDefaultType(t *testing.T) {
	a := New()

	assertion := &Assertion{
		Type:  "", // empty = equals
		Value: "hello",
	}
	result := a.Evaluate(assertion, "hello", nil)
	if !result.Passed {
		t.Error("empty type should default to equals")
	}
}

// Test assertScript
func TestAsserterScript(t *testing.T) {
	t.Run("no script runner configured", func(t *testing.T) {
		a := New()
		assertion := &Assertion{
			Type:   "script",
			Script: "test.script",
		}
		result := a.Evaluate(assertion, "output", nil)
		if result.Passed {
			t.Error("should fail without script runner")
		}
		if result.Message != "script assertions require a ScriptRunner to be configured" {
			t.Errorf("unexpected message: %s", result.Message)
		}
	})

	t.Run("empty script name", func(t *testing.T) {
		a := New().WithScriptRunner(&mockScriptRunner{})
		assertion := &Assertion{
			Type:   "script",
			Script: "",
		}
		result := a.Evaluate(assertion, "output", nil)
		if result.Passed {
			t.Error("should fail with empty script name")
		}
		if result.Message != "script assertion requires a script name" {
			t.Errorf("unexpected message: %s", result.Message)
		}
	})

	t.Run("script execution error", func(t *testing.T) {
		a := New().WithScriptRunner(&mockScriptRunner{
			err: errors.New("execution failed"),
		})
		assertion := &Assertion{
			Type:   "script",
			Script: "test.script",
		}
		result := a.Evaluate(assertion, "output", nil)
		if result.Passed {
			t.Error("should fail on script error")
		}
		if result.Message != "script execution failed: execution failed" {
			t.Errorf("unexpected message: %s", result.Message)
		}
	})

	t.Run("script passes", func(t *testing.T) {
		a := New().WithScriptRunner(&mockScriptRunner{
			passed:  true,
			message: "script passed",
		})
		assertion := &Assertion{
			Type:   "script",
			Script: "test.script",
		}
		result := a.Evaluate(assertion, "output", nil)
		if !result.Passed {
			t.Error("should pass when script passes")
		}
		if result.Message != "script passed" {
			t.Errorf("unexpected message: %s", result.Message)
		}
	})

	t.Run("script fails", func(t *testing.T) {
		a := New().WithScriptRunner(&mockScriptRunner{
			passed:  false,
			message: "validation failed",
		})
		assertion := &Assertion{
			Type:   "script",
			Script: "test.script",
		}
		result := a.Evaluate(assertion, "output", nil)
		if result.Passed {
			t.Error("should fail when script fails")
		}
	})
}

// Test assertAgent
func TestAsserterAgent(t *testing.T) {
	t.Run("no agent validator configured", func(t *testing.T) {
		a := New()
		assertion := &Assertion{
			Type: "agent",
			Use:  "agents:validator",
		}
		result := a.Evaluate(assertion, "output", nil)
		if result.Passed {
			t.Error("should fail without agent validator")
		}
		if result.Message != "agent assertions require an AgentValidator to be configured" {
			t.Errorf("unexpected message: %s", result.Message)
		}
	})

	t.Run("invalid use field format", func(t *testing.T) {
		a := New().WithAgentValidator(&mockAgentValidator{})
		assertion := &Assertion{
			Type: "agent",
			Use:  "invalid_format",
		}
		result := a.Evaluate(assertion, "output", nil)
		if result.Passed {
			t.Error("should fail with invalid use format")
		}
		if result.Message != "agent assertion requires 'use' field with 'agents:' prefix" {
			t.Errorf("unexpected message: %s", result.Message)
		}
	})

	t.Run("agent validation passes", func(t *testing.T) {
		a := New().WithAgentValidator(&mockAgentValidator{
			result: &Result{Passed: true, Message: "agent validated"},
		})
		assertion := &Assertion{
			Type: "agent",
			Use:  "agents:validator",
		}
		result := a.Evaluate(assertion, "output", nil)
		if !result.Passed {
			t.Error("should pass when agent validates")
		}
	})

	t.Run("agent validation fails", func(t *testing.T) {
		a := New().WithAgentValidator(&mockAgentValidator{
			result: &Result{Passed: false, Message: "agent rejected"},
		})
		assertion := &Assertion{
			Type: "agent",
			Use:  "agents:validator",
		}
		result := a.Evaluate(assertion, "output", nil)
		if result.Passed {
			t.Error("should fail when agent rejects")
		}
	})
}

// Test assertJSONPath edge cases
func TestAsserterJSONPathEdgeCases(t *testing.T) {
	a := New()

	t.Run("string output with valid JSON", func(t *testing.T) {
		assertion := &Assertion{
			Type:  "json_path",
			Path:  "name",
			Value: "test",
		}
		result := a.Evaluate(assertion, `{"name": "test"}`, nil)
		if !result.Passed {
			t.Errorf("should pass with valid JSON string, message: %s", result.Message)
		}
	})

	t.Run("string output with invalid JSON", func(t *testing.T) {
		assertion := &Assertion{
			Type:  "json_path",
			Path:  "name",
			Value: "test",
		}
		result := a.Evaluate(assertion, "not json", nil)
		if result.Passed {
			t.Error("should fail with invalid JSON string")
		}
	})

	t.Run("non-JSON output type", func(t *testing.T) {
		assertion := &Assertion{
			Type:  "json_path",
			Path:  "name",
			Value: "test",
		}
		result := a.Evaluate(assertion, 12345, nil)
		if result.Passed {
			t.Error("should fail with non-JSON type")
		}
	})

	t.Run("IN semantics with array expected", func(t *testing.T) {
		assertion := &Assertion{
			Type:  "json_path",
			Path:  "status",
			Value: []interface{}{"active", "pending", "completed"},
		}
		output := map[string]interface{}{"status": "pending"}
		result := a.Evaluate(assertion, output, nil)
		if !result.Passed {
			t.Errorf("should pass with IN semantics, message: %s", result.Message)
		}
	})

	t.Run("IN semantics no match", func(t *testing.T) {
		assertion := &Assertion{
			Type:  "json_path",
			Path:  "status",
			Value: []interface{}{"active", "completed"},
		}
		output := map[string]interface{}{"status": "pending"}
		result := a.Evaluate(assertion, output, nil)
		if result.Passed {
			t.Error("should fail when value not in expected array")
		}
	})

	t.Run("path with $. prefix", func(t *testing.T) {
		assertion := &Assertion{
			Type:  "json_path",
			Path:  "$.name",
			Value: "test",
		}
		output := map[string]interface{}{"name": "test"}
		result := a.Evaluate(assertion, output, nil)
		if !result.Passed {
			t.Errorf("should handle $. prefix, message: %s", result.Message)
		}
	})

	t.Run("array output", func(t *testing.T) {
		assertion := &Assertion{
			Type:  "json_path",
			Path:  "[0]",
			Value: "first",
		}
		output := []interface{}{"first", "second"}
		result := a.Evaluate(assertion, output, nil)
		if !result.Passed {
			t.Errorf("should work with array output, message: %s", result.Message)
		}
	})
}

// Test assertRegex edge cases
func TestAsserterRegexEdgeCases(t *testing.T) {
	a := New()

	t.Run("non-string pattern", func(t *testing.T) {
		assertion := &Assertion{
			Type:  "regex",
			Value: 12345, // not a string
		}
		result := a.Evaluate(assertion, "test", nil)
		if result.Passed {
			t.Error("should fail with non-string pattern")
		}
		if result.Message != "regex pattern must be a string" {
			t.Errorf("unexpected message: %s", result.Message)
		}
	})

	t.Run("invalid regex pattern", func(t *testing.T) {
		assertion := &Assertion{
			Type:  "regex",
			Value: "[invalid",
		}
		result := a.Evaluate(assertion, "test", nil)
		if result.Passed {
			t.Error("should fail with invalid regex")
		}
	})
}

// Test assertType edge cases
func TestAsserterTypeEdgeCases(t *testing.T) {
	a := New()

	t.Run("non-string type value", func(t *testing.T) {
		assertion := &Assertion{
			Type:  "type",
			Value: 12345, // not a string
		}
		result := a.Evaluate(assertion, "test", nil)
		if result.Passed {
			t.Error("should fail with non-string type value")
		}
		if result.Message != "type assertion value must be a string" {
			t.Errorf("unexpected message: %s", result.Message)
		}
	})

	t.Run("type with path from JSON string", func(t *testing.T) {
		assertion := &Assertion{
			Type:  "type",
			Path:  "items",
			Value: "array",
		}
		result := a.Evaluate(assertion, `{"items": [1, 2, 3]}`, nil)
		if !result.Passed {
			t.Errorf("should pass with JSON string input, message: %s", result.Message)
		}
	})

	t.Run("type with path from invalid JSON string", func(t *testing.T) {
		assertion := &Assertion{
			Type:  "type",
			Path:  "items",
			Value: "array",
		}
		result := a.Evaluate(assertion, "not json", nil)
		if result.Passed {
			t.Error("should fail with invalid JSON string")
		}
	})

	t.Run("type with path from non-JSON type", func(t *testing.T) {
		assertion := &Assertion{
			Type:  "type",
			Path:  "items",
			Value: "array",
		}
		result := a.Evaluate(assertion, 12345, nil)
		if result.Passed {
			t.Error("should fail with non-JSON type")
		}
	})

	t.Run("type with path from array", func(t *testing.T) {
		assertion := &Assertion{
			Type:  "type",
			Path:  "[0]",
			Value: "string",
		}
		result := a.Evaluate(assertion, []interface{}{"hello"}, nil)
		if !result.Passed {
			t.Errorf("should work with array, message: %s", result.Message)
		}
	})
}

// Test Validate with custom message
func TestAsserterValidateWithCustomMessage(t *testing.T) {
	a := New()

	assertions := []*Assertion{
		{Type: "equals", Value: "expected", Message: "custom failure"},
	}

	passed, message := a.Validate(assertions, "actual")
	if passed {
		t.Error("should fail")
	}
	if message != "custom failure" {
		t.Errorf("should use custom message, got: %s", message)
	}
}

// Test ParseAssertions edge cases
func TestParseAssertionsEdgeCases(t *testing.T) {
	t.Run("nil input", func(t *testing.T) {
		result := ParseAssertions(nil)
		if result != nil {
			t.Error("nil input should return nil")
		}
	})

	t.Run("array with non-map items", func(t *testing.T) {
		input := []interface{}{
			"string item",
			map[string]interface{}{"type": "equals"},
		}
		result := ParseAssertions(input)
		if len(result) != 1 {
			t.Errorf("should only parse map items, got %d", len(result))
		}
	})

	t.Run("map with all fields", func(t *testing.T) {
		input := map[string]interface{}{
			"type":    "agent",
			"value":   "criteria",
			"path":    "$.field",
			"script":  "test.script",
			"use":     "agents:validator",
			"message": "custom message",
			"negate":  true,
			"options": map[string]interface{}{
				"connector": "openai",
				"metadata":  map[string]interface{}{"key": "value"},
			},
		}
		result := ParseAssertions(input)
		if len(result) != 1 {
			t.Fatalf("expected 1 assertion, got %d", len(result))
		}
		a := result[0]
		if a.Type != "agent" {
			t.Errorf("type mismatch: %s", a.Type)
		}
		if a.Path != "$.field" {
			t.Errorf("path mismatch: %s", a.Path)
		}
		if a.Script != "test.script" {
			t.Errorf("script mismatch: %s", a.Script)
		}
		if a.Use != "agents:validator" {
			t.Errorf("use mismatch: %s", a.Use)
		}
		if a.Message != "custom message" {
			t.Errorf("message mismatch: %s", a.Message)
		}
		if !a.Negate {
			t.Error("negate should be true")
		}
		if a.Options == nil {
			t.Fatal("options should not be nil")
		}
		if a.Options.Connector != "openai" {
			t.Errorf("connector mismatch: %s", a.Options.Connector)
		}
		if a.Options.Metadata["key"] != "value" {
			t.Error("metadata mismatch")
		}
	})
}

// Test helper functions
func TestToString(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{"nil", nil, ""},
		{"string", "hello", "hello"},
		{"bytes", []byte("hello"), "hello"},
		{"number", 42, "42"},
		{"map", map[string]interface{}{"a": 1}, `{"a":1}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToString(tt.input)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestGetType(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{"nil", nil, "null"},
		{"string", "hello", "string"},
		{"float64", float64(3.14), "number"},
		{"float32", float32(3.14), "number"},
		{"int", 42, "number"},
		{"int64", int64(42), "number"},
		{"int32", int32(42), "number"},
		{"bool", true, "boolean"},
		{"array", []interface{}{1, 2}, "array"},
		{"object", map[string]interface{}{"a": 1}, "object"},
		{"other", struct{}{}, "struct {}"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetType(tt.input)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestTruncateOutput(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		maxLen   int
		expected string
	}{
		{"nil", nil, 10, "<nil>"},
		{"short string", "hello", 10, "hello"},
		{"long string", "hello world", 5, "hello..."},
		{"object", map[string]interface{}{"a": 1}, 100, `{"a":1}`},
		{"long object", map[string]interface{}{"key": "value"}, 5, `{"key...`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TruncateOutput(tt.input, tt.maxLen)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestExtractJSON(t *testing.T) {
	// Test basic JSON extraction
	result := ExtractJSON(`{"name": "test"}`)
	if result == nil {
		t.Error("should extract JSON")
	}
	if m, ok := result.(map[string]interface{}); ok {
		if m["name"] != "test" {
			t.Error("should extract correct value")
		}
	} else {
		t.Error("should return map")
	}
}

func TestExtractPathEdgeCases(t *testing.T) {
	t.Run("invalid array index", func(t *testing.T) {
		data := map[string]interface{}{
			"items": []interface{}{"a", "b"},
		}
		result := ExtractPath(data, "items[abc]")
		if result != nil {
			t.Error("invalid index should return nil")
		}
	})

	t.Run("array index on non-array", func(t *testing.T) {
		data := map[string]interface{}{
			"name": "test",
		}
		result := ExtractPath(data, "name[0]")
		if result != nil {
			t.Error("array index on non-array should return nil")
		}
	})

	t.Run("negative array index", func(t *testing.T) {
		data := map[string]interface{}{
			"items": []interface{}{"a", "b"},
		}
		result := ExtractPath(data, "items[-1]")
		if result != nil {
			t.Error("negative index should return nil")
		}
	})

	t.Run("out of bounds array index", func(t *testing.T) {
		data := map[string]interface{}{
			"items": []interface{}{"a", "b"},
		}
		result := ExtractPath(data, "items[99]")
		if result != nil {
			t.Error("out of bounds index should return nil")
		}
	})

	t.Run("field access on non-map", func(t *testing.T) {
		data := map[string]interface{}{
			"name": "test",
		}
		result := ExtractPath(data, "name.field")
		if result != nil {
			t.Error("field access on non-map should return nil")
		}
	})

	t.Run("empty path segment", func(t *testing.T) {
		data := map[string]interface{}{
			"name": "test",
		}
		result := ExtractPath(data, ".name")
		if result != "test" {
			t.Errorf("should handle leading dot, got: %v", result)
		}
	})
}

func TestValidateOutputEdgeCases(t *testing.T) {
	// Test with unmarshalable types (channels, functions)
	ch := make(chan int)
	result := ValidateOutput(ch, ch)
	if result {
		t.Error("unmarshalable types should return false")
	}
}
