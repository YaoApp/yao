package test_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/agent"
	agenttest "github.com/yaoapp/yao/agent/test"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/test"
)

func TestParseInputSource(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantType   agenttest.InputSourceType
		wantValue  string
		wantParams map[string]interface{}
	}{
		{
			name:      "JSONL file",
			input:     "./tests/inputs.jsonl",
			wantType:  agenttest.InputSourceFile,
			wantValue: "./tests/inputs.jsonl",
		},
		{
			name:      "JSON file",
			input:     "./tests/inputs.json",
			wantType:  agenttest.InputSourceFile,
			wantValue: "./tests/inputs.json",
		},
		{
			name:      "direct message",
			input:     "Hello, how are you?",
			wantType:  agenttest.InputSourceMessage,
			wantValue: "Hello, how are you?",
		},
		{
			name:      "agent source simple",
			input:     "agents:tests.generator-agent",
			wantType:  agenttest.InputSourceAgent,
			wantValue: "tests.generator-agent",
		},
		{
			name:      "agent source with params",
			input:     "agents:tests.generator-agent?count=10&focus=edge-cases",
			wantType:  agenttest.InputSourceAgent,
			wantValue: "tests.generator-agent",
			wantParams: map[string]interface{}{
				"count": 10,
				"focus": "edge-cases",
			},
		},
		{
			name:      "agent source with boolean param",
			input:     "agents:tests.generator-agent?verbose=true",
			wantType:  agenttest.InputSourceAgent,
			wantValue: "tests.generator-agent",
			wantParams: map[string]interface{}{
				"verbose": true,
			},
		},
		{
			name:      "script source with prefix",
			input:     "scripts:tests.gen.Generate",
			wantType:  agenttest.InputSourceScript,
			wantValue: "tests.gen.Generate",
		},
		{
			name:      "script test mode",
			input:     "scripts.tests.gen",
			wantType:  agenttest.InputSourceScript,
			wantValue: "scripts.tests.gen",
		},
		{
			name:      "path with separator",
			input:     "/path/to/inputs.jsonl",
			wantType:  agenttest.InputSourceFile,
			wantValue: "/path/to/inputs.jsonl",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := agenttest.ParseInputSource(tt.input)

			assert.Equal(t, tt.wantType, source.Type, "Type mismatch")
			assert.Equal(t, tt.wantValue, source.Value, "Value mismatch")

			if tt.wantParams != nil {
				for k, v := range tt.wantParams {
					assert.Equal(t, v, source.Params[k], "Param %s mismatch", k)
				}
			}
		})
	}
}

func TestInputSource_ToInputMode(t *testing.T) {
	tests := []struct {
		name     string
		source   *agenttest.InputSource
		wantMode agenttest.InputMode
	}{
		{
			name:     "file source",
			source:   &agenttest.InputSource{Type: agenttest.InputSourceFile},
			wantMode: agenttest.InputModeFile,
		},
		{
			name:     "message source",
			source:   &agenttest.InputSource{Type: agenttest.InputSourceMessage},
			wantMode: agenttest.InputModeMessage,
		},
		{
			name:     "script source",
			source:   &agenttest.InputSource{Type: agenttest.InputSourceScript},
			wantMode: agenttest.InputModeScript,
		},
		{
			name:     "agent source",
			source:   &agenttest.InputSource{Type: agenttest.InputSourceAgent},
			wantMode: agenttest.InputModeFile, // Agent generates cases, then runs in file mode
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mode := tt.source.ToInputMode()
			assert.Equal(t, tt.wantMode, mode)
		})
	}
}

func TestGenerateTestCases(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Load agent (includes assistants)
	err := agent.Load(config.Conf)
	if err != nil {
		t.Fatalf("Failed to load agent: %v", err)
	}

	// Test generating test cases from the generator agent
	targetInfo := &agenttest.TargetAgentInfo{
		ID:          "tests.next",
		Description: "A simple test agent for greeting",
	}

	params := map[string]interface{}{
		"count": 3,
		"focus": "happy-path",
	}

	cases, err := agenttest.GenerateTestCases("tests.generator-agent", targetInfo, params)
	if err != nil {
		t.Fatalf("Failed to generate test cases: %v", err)
	}

	// Verify we got some test cases
	assert.NotEmpty(t, cases, "Should generate at least one test case")

	// Verify each case has required fields
	for _, tc := range cases {
		assert.NotEmpty(t, tc.ID, "Test case should have ID")
		assert.NotNil(t, tc.Input, "Test case should have Input")
	}

	t.Logf("Generated %d test cases", len(cases))
	for _, tc := range cases {
		t.Logf("  - %s", tc.ID)
	}
}

func TestMapToCaseOptions(t *testing.T) {
	// Test that options map is correctly converted
	source := agenttest.ParseInputSource("agents:test?count=5")
	assert.Equal(t, 5, source.Params["count"])
}
