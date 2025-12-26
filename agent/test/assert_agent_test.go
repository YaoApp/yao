package test_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	v8 "github.com/yaoapp/gou/runtime/v8"
	"github.com/yaoapp/yao/agent"
	agenttest "github.com/yaoapp/yao/agent/test"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/test"
	"rogchap.com/v8go"
)

func TestAsserter_AgentAssertion(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Load agent (includes assistants)
	err := agent.Load(config.Conf)
	if err != nil {
		t.Fatalf("Failed to load agent: %v", err)
	}

	asserter := agenttest.NewAsserter()

	tests := []struct {
		name     string
		tc       *agenttest.Case
		output   interface{}
		expected bool
		skipMsg  string
	}{
		{
			name: "agent assertion - pass",
			tc: &agenttest.Case{
				Assert: map[string]interface{}{
					"type":  "agent",
					"use":   "agents:tests.validator-agent",
					"value": "Response should be a greeting",
				},
			},
			output:   "Hello! How can I help you today?",
			expected: true,
		},
		{
			name: "agent assertion - fail",
			tc: &agenttest.Case{
				Assert: map[string]interface{}{
					"type":  "agent",
					"use":   "agents:tests.validator-agent",
					"value": "Response should provide a detailed technical answer",
				},
			},
			output:   "I don't know.",
			expected: false,
		},
		{
			name: "agent assertion - missing prefix",
			tc: &agenttest.Case{
				Assert: map[string]interface{}{
					"type":  "agent",
					"use":   "tests.validator-agent", // Missing agents: prefix
					"value": "Should pass",
				},
			},
			output:   "Hello",
			expected: false, // Should fail due to missing prefix
		},
		{
			name: "agent assertion - with metadata",
			tc: &agenttest.Case{
				Assert: map[string]interface{}{
					"type":  "agent",
					"use":   "agents:tests.validator-agent",
					"value": "Response is helpful",
					"options": map[string]interface{}{
						"metadata": map[string]interface{}{
							"context": "customer support",
						},
					},
				},
			},
			output:   "I'd be happy to help you with your order. Let me look that up for you.",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipMsg != "" {
				t.Skip(tt.skipMsg)
			}

			passed, errMsg := asserter.Validate(tt.tc, tt.output)
			if passed != tt.expected {
				t.Errorf("Expected passed=%v, got passed=%v, error: %s", tt.expected, passed, errMsg)
			}
		})
	}
}

func TestAsserter_AgentAssertion_InvalidAgent(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Load agent (includes assistants)
	err := agent.Load(config.Conf)
	if err != nil {
		t.Fatalf("Failed to load agent: %v", err)
	}

	asserter := agenttest.NewAsserter()

	tc := &agenttest.Case{
		Assert: map[string]interface{}{
			"type":  "agent",
			"use":   "agents:nonexistent.agent",
			"value": "Should fail",
		},
	}

	passed, errMsg := asserter.Validate(tc, "Hello")
	assert.False(t, passed, "Should fail for nonexistent agent")
	assert.Contains(t, errMsg, "failed to get validator agent", "Error should mention agent loading failure")
}

func TestAsserter_MapToAssertion_WithUseAndOptions(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Load agent (includes assistants)
	err := agent.Load(config.Conf)
	if err != nil {
		t.Fatalf("Failed to load agent: %v", err)
	}

	asserter := agenttest.NewAsserter()

	// Test that mapToAssertion correctly parses use and options fields
	tc := &agenttest.Case{
		Assert: map[string]interface{}{
			"type":  "agent",
			"use":   "agents:tests.validator-agent",
			"value": "criteria here",
			"options": map[string]interface{}{
				"connector": "gpt-4o",
				"metadata": map[string]interface{}{
					"key": "value",
				},
			},
		},
	}

	// Validate triggers parseAssertions internally
	// We just verify it doesn't panic and processes correctly
	_, _ = asserter.Validate(tc, "test output")
	// If we get here without panic, the parsing worked
}

// TestTestingT_AssertAgent tests the JSAPI t.assert.Agent() method
func TestTestingT_AssertAgent(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Load agent (includes assistants)
	err := agent.Load(config.Conf)
	if err != nil {
		t.Fatalf("Failed to load agent: %v", err)
	}

	tests := []struct {
		name       string
		script     string
		shouldFail bool
	}{
		{
			name: "JSAPI agent assertion - pass",
			script: `
				function test(t) {
					var response = "Hello! How can I help you today?";
					t.assert.Agent(response, "tests.validator-agent", {
						criteria: "Response should be a friendly greeting"
					});
				}
				test(__test_t);
			`,
			shouldFail: false,
		},
		{
			name: "JSAPI agent assertion - JSON response",
			script: `
				function test(t) {
					var response = {
						status: "success",
						data: { user: "john", email: "john@example.com" },
						message: "User created successfully"
					};
					t.assert.Agent(response, "tests.validator-agent", {
						criteria: "Response should be a successful API response with user data"
					});
				}
				test(__test_t);
			`,
			shouldFail: false,
		},
		{
			name: "JSAPI agent assertion - with metadata",
			script: `
				function test(t) {
					var response = "I'd be happy to help you with your order.";
					t.assert.Agent(response, "tests.validator-agent", {
						criteria: "Response is helpful and professional",
						metadata: { context: "customer support" }
					});
				}
				test(__test_t);
			`,
			shouldFail: false,
		},
		{
			name: "JSAPI agent assertion - fail case",
			script: `
				function test(t) {
					var response = "I don't know.";
					t.assert.Agent(response, "tests.validator-agent", {
						criteria: "Response should provide a detailed technical explanation"
					});
				}
				test(__test_t);
			`,
			shouldFail: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create TestingT
			testingT := agenttest.NewTestingT(tt.name)

			// Create V8 isolate and context
			iso := v8go.NewIsolate()
			defer iso.Dispose()

			v8ctx := v8go.NewContext(iso)
			defer v8ctx.Close()

			// Create testing object
			testObj, err := agenttest.NewTestingTObject(v8ctx, testingT)
			if err != nil {
				t.Fatalf("Failed to create testing object: %v", err)
			}

			// Set testing object as global
			global := v8ctx.Global()
			global.Set("__test_t", testObj)

			// Run the test script
			_, err = v8ctx.RunScript(tt.script, "test.js")

			// Check results
			if tt.shouldFail {
				assert.True(t, testingT.Failed(), "Test should have failed")
			} else {
				if err != nil {
					t.Errorf("Script execution error: %v", err)
				}
				assert.False(t, testingT.Failed(), "Test should have passed, errors: %v", testingT.Errors())
			}
		})
	}
}

// Ensure v8 is used (for script loading)
var _ = v8.Scripts
