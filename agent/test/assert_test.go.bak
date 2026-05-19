package test

import (
	"testing"

	"github.com/yaoapp/yao/agent/context"
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

func TestAsserter_ToolCalled(t *testing.T) {
	tests := []struct {
		name     string
		tc       *Case
		response *context.Response
		expected bool
		errMsg   string
	}{
		{
			name: "tool called - exact match",
			tc: &Case{
				Assert: map[string]interface{}{
					"type":  "tool_called",
					"value": "agents_expense_tools__setup",
				},
			},
			response: &context.Response{
				Tools: []context.ToolCallResponse{
					{Tool: "agents_expense_tools__setup", Result: map[string]interface{}{"success": true}},
				},
			},
			expected: true,
		},
		{
			name: "tool called - suffix match",
			tc: &Case{
				Assert: map[string]interface{}{
					"type":  "tool_called",
					"value": "setup",
				},
			},
			response: &context.Response{
				Tools: []context.ToolCallResponse{
					{Tool: "agents_expense_tools__setup", Result: map[string]interface{}{"success": true}},
				},
			},
			expected: true,
		},
		{
			name: "tool not called",
			tc: &Case{
				Assert: map[string]interface{}{
					"type":  "tool_called",
					"value": "setup",
				},
			},
			response: &context.Response{
				Tools: []context.ToolCallResponse{},
			},
			expected: false,
		},
		{
			name: "tool called - wrong tool",
			tc: &Case{
				Assert: map[string]interface{}{
					"type":  "tool_called",
					"value": "setup",
				},
			},
			response: &context.Response{
				Tools: []context.ToolCallResponse{
					{Tool: "agents_expense_tools__submit", Result: map[string]interface{}{"success": true}},
				},
			},
			expected: false,
		},
		{
			name: "tool called - any of multiple",
			tc: &Case{
				Assert: map[string]interface{}{
					"type":  "tool_called",
					"value": []interface{}{"setup", "init"},
				},
			},
			response: &context.Response{
				Tools: []context.ToolCallResponse{
					{Tool: "agents_expense_tools__init", Result: map[string]interface{}{"success": true}},
				},
			},
			expected: true,
		},
		{
			name: "tool called - with arguments (map)",
			tc: &Case{
				Assert: map[string]interface{}{
					"type": "tool_called",
					"value": map[string]interface{}{
						"name": "setup",
						"arguments": map[string]interface{}{
							"action": "init",
						},
					},
				},
			},
			response: &context.Response{
				Tools: []context.ToolCallResponse{
					{
						Tool:      "agents_expense_tools__setup",
						Arguments: map[string]interface{}{"action": "init", "config": map[string]interface{}{}},
						Result:    map[string]interface{}{"success": true},
					},
				},
			},
			expected: true,
		},
		{
			name: "tool called - with arguments (JSON string)",
			tc: &Case{
				Assert: map[string]interface{}{
					"type": "tool_called",
					"value": map[string]interface{}{
						"name": "setup",
						"arguments": map[string]interface{}{
							"action": "init",
						},
					},
				},
			},
			response: &context.Response{
				Tools: []context.ToolCallResponse{
					{
						Tool:      "agents_expense_tools__setup",
						Arguments: `{"action":"init","config":{"default_currency":"USD"}}`,
						Result:    map[string]interface{}{"success": true},
					},
				},
			},
			expected: true,
		},
		{
			name: "tool called - wrong arguments",
			tc: &Case{
				Assert: map[string]interface{}{
					"type": "tool_called",
					"value": map[string]interface{}{
						"name": "setup",
						"arguments": map[string]interface{}{
							"action": "update",
						},
					},
				},
			},
			response: &context.Response{
				Tools: []context.ToolCallResponse{
					{
						Tool:      "agents_expense_tools__setup",
						Arguments: map[string]interface{}{"action": "init"},
						Result:    map[string]interface{}{"success": true},
					},
				},
			},
			expected: false,
		},
		{
			name: "no response",
			tc: &Case{
				Assert: map[string]interface{}{
					"type":  "tool_called",
					"value": "setup",
				},
			},
			response: nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			asserter := NewAsserter().WithResponse(tt.response)
			passed, errMsg := asserter.Validate(tt.tc, nil)
			if passed != tt.expected {
				t.Errorf("Expected passed=%v, got passed=%v, error: %s", tt.expected, passed, errMsg)
			}
		})
	}
}

func TestAsserter_ToolResult(t *testing.T) {
	tests := []struct {
		name     string
		tc       *Case
		response *context.Response
		expected bool
	}{
		{
			name: "tool result - success check",
			tc: &Case{
				Assert: map[string]interface{}{
					"type": "tool_result",
					"value": map[string]interface{}{
						"tool": "setup",
						"result": map[string]interface{}{
							"success": true,
						},
					},
				},
			},
			response: &context.Response{
				Tools: []context.ToolCallResponse{
					{
						Tool:   "agents_expense_tools__setup",
						Result: map[string]interface{}{"success": true, "message": "Setup complete"},
					},
				},
			},
			expected: true,
		},
		{
			name: "tool result - message check with regex",
			tc: &Case{
				Assert: map[string]interface{}{
					"type": "tool_result",
					"value": map[string]interface{}{
						"tool": "setup",
						"result": map[string]interface{}{
							"message": "regex:(?i)setup.*complete",
						},
					},
				},
			},
			response: &context.Response{
				Tools: []context.ToolCallResponse{
					{
						Tool:   "agents_expense_tools__setup",
						Result: map[string]interface{}{"success": true, "message": "Setup complete!"},
					},
				},
			},
			expected: true,
		},
		{
			name: "tool result - no expected result (just check no error)",
			tc: &Case{
				Assert: map[string]interface{}{
					"type": "tool_result",
					"value": map[string]interface{}{
						"tool": "setup",
					},
				},
			},
			response: &context.Response{
				Tools: []context.ToolCallResponse{
					{
						Tool:   "agents_expense_tools__setup",
						Result: map[string]interface{}{"success": true},
					},
				},
			},
			expected: true,
		},
		{
			name: "tool result - tool has error",
			tc: &Case{
				Assert: map[string]interface{}{
					"type": "tool_result",
					"value": map[string]interface{}{
						"tool": "setup",
					},
				},
			},
			response: &context.Response{
				Tools: []context.ToolCallResponse{
					{
						Tool:  "agents_expense_tools__setup",
						Error: "permission denied",
					},
				},
			},
			expected: false,
		},
		{
			name: "tool result - result mismatch",
			tc: &Case{
				Assert: map[string]interface{}{
					"type": "tool_result",
					"value": map[string]interface{}{
						"tool": "setup",
						"result": map[string]interface{}{
							"success": true,
						},
					},
				},
			},
			response: &context.Response{
				Tools: []context.ToolCallResponse{
					{
						Tool:   "agents_expense_tools__setup",
						Result: map[string]interface{}{"success": false},
					},
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			asserter := NewAsserter().WithResponse(tt.response)
			passed, errMsg := asserter.Validate(tt.tc, nil)
			if passed != tt.expected {
				t.Errorf("Expected passed=%v, got passed=%v, error: %s", tt.expected, passed, errMsg)
			}
		})
	}
}

func TestAsserter_MultipleToolAssertions(t *testing.T) {
	// This tests the exact scenario from setup-006: tool_called + tool_result
	response := &context.Response{
		Tools: []context.ToolCallResponse{
			{
				Tool:      "agents_expense_tools__setup",
				Arguments: `{"action":"init","config":{"default_currency":"USD","categories":[{"id":"meals","name":"Meals","daily_limit":100}]}}`,
				Result: map[string]interface{}{
					"success": true,
					"action":  "init",
					"config": map[string]interface{}{
						"default_currency": "USD",
					},
					"message": "Setup complete!",
				},
			},
		},
	}

	asserter := NewAsserter().WithResponse(response)

	tc := &Case{
		Assert: []interface{}{
			map[string]interface{}{
				"type": "tool_called",
				"value": map[string]interface{}{
					"name": "setup",
					"arguments": map[string]interface{}{
						"action": "init",
					},
				},
			},
			map[string]interface{}{
				"type": "tool_result",
				"value": map[string]interface{}{
					"tool": "setup",
					"result": map[string]interface{}{
						"success": true,
					},
				},
			},
		},
	}

	result := asserter.ValidateWithDetails(tc, nil)
	if !result.Passed {
		t.Errorf("Expected multiple tool assertions to pass, got: %s", result.Message)
	}
}

func TestAsserter_SharedAsserterWithResponse(t *testing.T) {
	// Test that a shared asserter correctly uses WithResponse
	asserter := NewAsserter()

	// First call without response - should fail
	tc := &Case{
		Assert: map[string]interface{}{
			"type":  "tool_called",
			"value": "setup",
		},
	}
	result := asserter.ValidateWithDetails(tc, nil)
	if result.Passed {
		t.Error("Expected tool_called to fail without response")
	}
	if result.Message != "no response available for tool_called assertion" {
		t.Errorf("Unexpected message: %s", result.Message)
	}

	// Now set response
	response := &context.Response{
		Tools: []context.ToolCallResponse{
			{
				Tool:   "agents_expense_tools__setup",
				Result: map[string]interface{}{"success": true},
			},
		},
	}
	asserter.WithResponse(response)

	// Should pass now
	result = asserter.ValidateWithDetails(tc, nil)
	if !result.Passed {
		t.Errorf("Expected tool_called to pass with response, got: %s", result.Message)
	}
}

func TestAsserter_Setup006Scenario(t *testing.T) {
	// Exact reproduction of setup-006 scenario
	// Turn 2: tool was called with action: init, result has success: true
	response := &context.Response{
		Tools: []context.ToolCallResponse{
			{
				Tool:      "agents_expense_tools__setup",
				Arguments: `{"action":"init","config":{"default_currency":"USD","categories":[{"id":"meals","name":"Meals","daily_limit":100},{"id":"travel","name":"Travel","daily_limit":500}]}}`,
				Result: map[string]interface{}{
					"config": map[string]interface{}{
						"categories": []interface{}{
							map[string]interface{}{"daily_limit": float64(100), "id": "meals", "name": "Meals"},
							map[string]interface{}{"daily_limit": float64(500), "id": "travel", "name": "Travel"},
						},
						"default_currency": "USD",
					},
					"message": "Setup complete! The expense system has been initialized successfully with the configured settings. You can now start submitting expenses.",
					"success": true,
					"action":  "init",
				},
			},
		},
	}

	// This is the exact assert from setup-006's quick_complete checkpoint
	assertDef := []interface{}{
		map[string]interface{}{
			"type": "tool_called",
			"value": map[string]interface{}{
				"name": "setup",
				"arguments": map[string]interface{}{
					"action": "init",
				},
			},
		},
		map[string]interface{}{
			"type": "tool_result",
			"value": map[string]interface{}{
				"tool": "setup",
				"result": map[string]interface{}{
					"success": true,
				},
			},
		},
	}

	asserter := NewAsserter().WithResponse(response)
	tc := &Case{Assert: assertDef}

	result := asserter.ValidateWithDetails(tc, nil)
	if !result.Passed {
		t.Errorf("Expected setup-006 scenario to pass, got: %s", result.Message)
	}

	// Also test individual assertions
	t.Run("tool_called only", func(t *testing.T) {
		tc2 := &Case{Assert: assertDef[0]}
		result2 := asserter.ValidateWithDetails(tc2, nil)
		if !result2.Passed {
			t.Errorf("Expected tool_called to pass, got: %s", result2.Message)
		}
	})

	t.Run("tool_result only", func(t *testing.T) {
		tc3 := &Case{Assert: assertDef[1]}
		result3 := asserter.ValidateWithDetails(tc3, nil)
		if !result3.Passed {
			t.Errorf("Expected tool_result to pass, got: %s", result3.Message)
		}
	})
}

func TestAsserter_Setup003Scenario(t *testing.T) {
	// Exact reproduction of setup-003 scenario
	// Turn 3: tool was called with action: update
	response := &context.Response{
		Tools: []context.ToolCallResponse{
			{
				Tool:      "agents_expense_tools__setup",
				Arguments: `{"action":"update","config":{"categories":[{"daily_limit":500,"id":"meals","name":"Business Meals"}]}}`,
				Result: map[string]interface{}{
					"action": "update",
					"config": map[string]interface{}{
						"categories": []interface{}{
							map[string]interface{}{"daily_limit": float64(500), "id": "meals", "name": "Business Meals"},
						},
					},
					"message": "Configuration updated successfully! Your changes have been saved.",
					"success": true,
				},
			},
		},
	}

	// This is the exact assert from setup-003's update_complete checkpoint
	assertDef := []interface{}{
		map[string]interface{}{
			"type": "tool_called",
			"value": map[string]interface{}{
				"name": "setup",
				"arguments": map[string]interface{}{
					"action": "update",
				},
			},
		},
		map[string]interface{}{
			"type": "tool_result",
			"value": map[string]interface{}{
				"tool": "setup",
				"result": map[string]interface{}{
					"success": true,
				},
			},
		},
	}

	asserter := NewAsserter().WithResponse(response)
	tc := &Case{Assert: assertDef}

	result := asserter.ValidateWithDetails(tc, nil)
	if !result.Passed {
		t.Errorf("Expected setup-003 scenario to pass, got: %s", result.Message)
	}

	// Test individual assertions
	t.Run("tool_called with action:update", func(t *testing.T) {
		tc2 := &Case{Assert: assertDef[0]}
		result2 := asserter.ValidateWithDetails(tc2, nil)
		if !result2.Passed {
			t.Errorf("Expected tool_called to pass, got: %s", result2.Message)
		}
	})
}

func TestMatchToolName(t *testing.T) {
	tests := []struct {
		actual   string
		expected string
		match    bool
	}{
		{"agents_expense_tools__setup", "agents_expense_tools__setup", true},
		{"agents_expense_tools__setup", "setup", true},
		{"agents.expense.tools.setup", "setup", true},
		{"agents_expense_tools__setup", "init", false},
		{"setup", "setup", true},
	}

	for _, tt := range tests {
		t.Run(tt.actual+"_"+tt.expected, func(t *testing.T) {
			if matchToolName(tt.actual, tt.expected) != tt.match {
				t.Errorf("matchToolName(%q, %q) = %v, want %v", tt.actual, tt.expected, !tt.match, tt.match)
			}
		})
	}
}
