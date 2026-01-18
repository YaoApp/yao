// Package assert provides a universal assertion/validation library for Yao.
// It can be used by agent/robot, flow, pipe, widget, and other modules.
//
// Design:
// - Independent implementation (no dependency on agent/test)
// - Supports both rule-based and semantic validation
// - Extensible through interfaces (AgentValidator, ScriptRunner)
package assert

// Assertion represents a single assertion rule
type Assertion struct {
	// Type is the assertion type:
	// - "equals": exact match (default if expected is set)
	// - "contains": output contains the expected string/value
	// - "not_contains": output does not contain the string/value
	// - "json_path": extract value using JSON path and compare
	// - "regex": match output against regex pattern
	// - "type": check output type (string, object, array, number, boolean)
	// - "script": run a custom assertion script (requires ScriptRunner)
	// - "agent": use an agent to validate (requires AgentValidator)
	Type string `json:"type"`

	// Value is the expected value or pattern (depends on type)
	Value interface{} `json:"value,omitempty"`

	// Path is the JSON path for json_path assertions (e.g., "$.count", "items[0].name")
	Path string `json:"path,omitempty"`

	// Script is the script/process name for script assertions
	Script string `json:"script,omitempty"`

	// Use specifies the agent for validation (e.g., "agents:validator")
	Use string `json:"use,omitempty"`

	// Options for agent-driven assertions
	Options *AssertionOptions `json:"options,omitempty"`

	// Message is a custom failure message
	Message string `json:"message,omitempty"`

	// Negate inverts the assertion result
	Negate bool `json:"negate,omitempty"`
}

// AssertionOptions for agent-driven assertions
type AssertionOptions struct {
	// Connector overrides the agent's default connector
	Connector string `json:"connector,omitempty"`

	// Metadata contains custom data passed to the validator
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// Result represents the result of an assertion
type Result struct {
	// Passed indicates whether the assertion passed
	Passed bool `json:"passed"`

	// Message describes the assertion result
	Message string `json:"message,omitempty"`

	// Assertion is the original assertion that was evaluated
	Assertion *Assertion `json:"assertion,omitempty"`

	// Actual is the actual value that was compared
	Actual interface{} `json:"actual,omitempty"`

	// Expected is the expected value
	Expected interface{} `json:"expected,omitempty"`
}

// AgentValidator is an interface for agent-based validation
// Implementations should call an AI agent to perform semantic validation
type AgentValidator interface {
	// Validate validates output using an agent
	// agentID: the agent identifier (e.g., "validator")
	// output: the output to validate
	// input: the original input (for context)
	// criteria: validation criteria from assertion.Value
	// options: assertion options
	Validate(agentID string, output, input, criteria interface{}, options *AssertionOptions) *Result
}

// ScriptRunner is an interface for running assertion scripts
// Implementations should call a Yao process to perform validation
type ScriptRunner interface {
	// Run runs an assertion script
	// scriptName: the script/process name
	// output: the output to validate
	// input: the original input
	// expected: the expected value from assertion.Value
	// Returns (passed, message, error)
	Run(scriptName string, output, input, expected interface{}) (bool, string, error)
}
