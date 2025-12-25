package test

import (
	"testing"
)

func TestAsserter_JSONPath_ArrayEquality(t *testing.T) {
	asserter := NewAsserter()

	tests := []struct {
		name     string
		tc       *Case
		output   interface{}
		expected bool
		errMsg   string
	}{
		{
			name: "array equals array - same content",
			tc: &Case{
				Assert: map[string]interface{}{
					"type":  "json_path",
					"path":  "search_types",
					"value": []interface{}{"db"},
				},
			},
			output: map[string]interface{}{
				"search_types": []interface{}{"db"},
			},
			expected: true,
		},
		{
			name: "array equals array - different content",
			tc: &Case{
				Assert: map[string]interface{}{
					"type":  "json_path",
					"path":  "search_types",
					"value": []interface{}{"db"},
				},
			},
			output: map[string]interface{}{
				"search_types": []interface{}{"web"},
			},
			expected: false,
		},
		{
			name: "array equals array - multiple elements",
			tc: &Case{
				Assert: map[string]interface{}{
					"type":  "json_path",
					"path":  "search_types",
					"value": []interface{}{"web", "db"},
				},
			},
			output: map[string]interface{}{
				"search_types": []interface{}{"web", "db"},
			},
			expected: true,
		},
		{
			name: "array equals array - different order",
			tc: &Case{
				Assert: map[string]interface{}{
					"type":  "json_path",
					"path":  "search_types",
					"value": []interface{}{"db", "web"},
				},
			},
			output: map[string]interface{}{
				"search_types": []interface{}{"web", "db"},
			},
			expected: false, // Order matters for array equality
		},
		{
			name: "scalar in array - match",
			tc: &Case{
				Assert: map[string]interface{}{
					"type":  "json_path",
					"path":  "status",
					"value": []interface{}{"active", "pending"},
				},
			},
			output: map[string]interface{}{
				"status": "active",
			},
			expected: true, // "active" is one of ["active", "pending"]
		},
		{
			name: "scalar in array - no match",
			tc: &Case{
				Assert: map[string]interface{}{
					"type":  "json_path",
					"path":  "status",
					"value": []interface{}{"active", "pending"},
				},
			},
			output: map[string]interface{}{
				"status": "inactive",
			},
			expected: false,
		},
		{
			name: "simple value comparison",
			tc: &Case{
				Assert: map[string]interface{}{
					"type":  "json_path",
					"path":  "need_search",
					"value": true,
				},
			},
			output: map[string]interface{}{
				"need_search": true,
			},
			expected: true,
		},
		{
			name: "nested path",
			tc: &Case{
				Assert: map[string]interface{}{
					"type":  "json_path",
					"path":  "result.count",
					"value": float64(5),
				},
			},
			output: map[string]interface{}{
				"result": map[string]interface{}{
					"count": float64(5),
				},
			},
			expected: true,
		},
		{
			name: "array index access",
			tc: &Case{
				Assert: map[string]interface{}{
					"type":  "json_path",
					"path":  "items[0].name",
					"value": "first",
				},
			},
			output: map[string]interface{}{
				"items": []interface{}{
					map[string]interface{}{"name": "first"},
					map[string]interface{}{"name": "second"},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			passed, errMsg := asserter.Validate(tt.tc, tt.output)
			if passed != tt.expected {
				t.Errorf("Expected passed=%v, got passed=%v, error: %s", tt.expected, passed, errMsg)
			}
		})
	}
}

func TestAsserter_Contains(t *testing.T) {
	asserter := NewAsserter()

	tests := []struct {
		name     string
		tc       *Case
		output   interface{}
		expected bool
	}{
		{
			name: "string contains substring",
			tc: &Case{
				Assert: map[string]interface{}{
					"type":  "contains",
					"value": "hello",
				},
			},
			output:   "hello world",
			expected: true,
		},
		{
			name: "string does not contain",
			tc: &Case{
				Assert: map[string]interface{}{
					"type":  "contains",
					"value": "goodbye",
				},
			},
			output:   "hello world",
			expected: false,
		},
		{
			name: "JSON contains field",
			tc: &Case{
				Assert: map[string]interface{}{
					"type":  "contains",
					"value": "success",
				},
			},
			output: map[string]interface{}{
				"status": "success",
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			passed, _ := asserter.Validate(tt.tc, tt.output)
			if passed != tt.expected {
				t.Errorf("Expected passed=%v, got passed=%v", tt.expected, passed)
			}
		})
	}
}

func TestAsserter_MultipleAssertions(t *testing.T) {
	asserter := NewAsserter()

	tc := &Case{
		Assert: []interface{}{
			map[string]interface{}{
				"type":  "json_path",
				"path":  "need_search",
				"value": true,
			},
			map[string]interface{}{
				"type":  "json_path",
				"path":  "search_types",
				"value": []interface{}{"web"},
			},
		},
	}

	output := map[string]interface{}{
		"need_search":  true,
		"search_types": []interface{}{"web"},
	}

	passed, errMsg := asserter.Validate(tc, output)
	if !passed {
		t.Errorf("Expected all assertions to pass, got error: %s", errMsg)
	}
}

func TestAsserter_Negate(t *testing.T) {
	asserter := NewAsserter()

	tc := &Case{
		Assert: map[string]interface{}{
			"type":   "contains",
			"value":  "error",
			"negate": true,
		},
	}

	output := "success message"

	passed, _ := asserter.Validate(tc, output)
	if !passed {
		t.Error("Expected negated assertion to pass")
	}
}

func TestAsserter_Regex(t *testing.T) {
	asserter := NewAsserter()

	tests := []struct {
		name     string
		tc       *Case
		output   interface{}
		expected bool
	}{
		{
			name: "regex matches",
			tc: &Case{
				Assert: map[string]interface{}{
					"type":  "regex",
					"value": `\d{3}-\d{4}`,
				},
			},
			output:   "Phone: 123-4567",
			expected: true,
		},
		{
			name: "regex does not match",
			tc: &Case{
				Assert: map[string]interface{}{
					"type":  "regex",
					"value": `\d{3}-\d{4}`,
				},
			},
			output:   "No phone number here",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			passed, _ := asserter.Validate(tt.tc, tt.output)
			if passed != tt.expected {
				t.Errorf("Expected passed=%v, got passed=%v", tt.expected, passed)
			}
		})
	}
}
