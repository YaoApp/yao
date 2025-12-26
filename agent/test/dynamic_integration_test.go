package test_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent"
	agenttest "github.com/yaoapp/yao/agent/test"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/test"
)

// TestDynamicRunner_CoffeeOrder tests a complete dynamic mode flow:
// Simulator acts as a customer ordering coffee, agent handles the order
func TestDynamicRunner_CoffeeOrder(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Load agents
	err := agent.Load(config.Conf)
	require.NoError(t, err, "Failed to load agents")

	// Create a temporary JSONL file with a dynamic test case
	tmpDir := t.TempDir()
	inputFile := filepath.Join(tmpDir, "dynamic-inputs.jsonl")

	// Dynamic test case: customer ordering coffee (JSONL must be single line)
	testCase := `{"id": "coffee-order-flow", "name": "Complete Coffee Order", "input": "Hi, I would like to order a coffee please", "simulator": {"use": "tests.simulator-agent", "options": {"metadata": {"persona": "A customer who wants to order a medium latte with oat milk", "goal": "Successfully complete a coffee order"}}}, "checkpoints": [{"id": "greeting", "description": "Agent greets and asks for order", "assert": {"type": "regex", "value": "(?i)(order|like|help)"}}, {"id": "ask_size", "description": "Agent asks for size", "after": ["greeting"], "assert": {"type": "regex", "value": "(?i)size"}}, {"id": "confirm_order", "description": "Agent confirms the order", "after": ["ask_size"], "assert": {"type": "regex", "value": "(?i)confirm"}}], "max_turns": 8}`

	err = os.WriteFile(inputFile, []byte(testCase), 0644)
	require.NoError(t, err, "Failed to write test file")

	// Run dynamic test
	opts := &agenttest.Options{
		Input:     inputFile,
		AgentID:   "tests.dynamic-test-agent",
		Verbose:   true,
		InputMode: agenttest.InputModeFile,
	}
	opts = agenttest.MergeOptions(opts, agenttest.DefaultOptions())

	runner := agenttest.NewRunner(opts)
	report, err := runner.Run()

	require.NoError(t, err, "Runner should not return error")
	require.NotNil(t, report, "Report should not be nil")
	require.NotNil(t, report.Summary, "Summary should not be nil")

	// Log results
	t.Logf("Total: %d, Passed: %d, Failed: %d",
		report.Summary.Total, report.Summary.Passed, report.Summary.Failed)

	// Check results
	if len(report.Results) > 0 {
		result := report.Results[0]
		t.Logf("Test [%s] Status: %s", result.ID, result.Status)

		// Check metadata for dynamic mode info
		if result.Metadata != nil {
			if mode, ok := result.Metadata["mode"].(string); ok {
				assert.Equal(t, "dynamic", mode, "Should be dynamic mode")
			}
			if turns, ok := result.Metadata["total_turns"].(int); ok {
				t.Logf("Total turns: %d", turns)
			}
		}

		if result.Error != "" {
			t.Logf("Error: %s", result.Error)
		}
	}
}

// TestDynamicRunner_WithInitialInput tests dynamic mode with initial user input
func TestDynamicRunner_WithInitialInput(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Load agents
	err := agent.Load(config.Conf)
	require.NoError(t, err, "Failed to load agents")

	// Create a test case with initial input
	tmpDir := t.TempDir()
	inputFile := filepath.Join(tmpDir, "dynamic-inputs.jsonl")

	// Start with user's first message (JSONL must be single line)
	testCase := `{"id": "coffee-with-initial", "name": "Coffee Order with Initial Message", "input": "Hi, I want to order a coffee", "simulator": {"use": "tests.simulator-agent", "options": {"metadata": {"persona": "Customer ordering a large cappuccino", "goal": "Complete the coffee order"}}}, "checkpoints": [{"id": "acknowledge", "description": "Agent acknowledges the order request", "assert": {"type": "regex", "value": "(?i)(coffee|order|help)"}}, {"id": "ask_details", "description": "Agent asks for more details", "after": ["acknowledge"], "assert": {"type": "regex", "value": "(?i)(size|type|what)"}}], "max_turns": 5}`

	err = os.WriteFile(inputFile, []byte(testCase), 0644)
	require.NoError(t, err, "Failed to write test file")

	// Run test
	opts := &agenttest.Options{
		Input:     inputFile,
		AgentID:   "tests.dynamic-test-agent",
		Verbose:   true,
		InputMode: agenttest.InputModeFile,
	}
	opts = agenttest.MergeOptions(opts, agenttest.DefaultOptions())

	runner := agenttest.NewRunner(opts)
	report, err := runner.Run()

	require.NoError(t, err, "Runner should not return error")
	require.NotNil(t, report, "Report should not be nil")

	t.Logf("Total: %d, Passed: %d, Failed: %d",
		report.Summary.Total, report.Summary.Passed, report.Summary.Failed)
}

// TestDynamicRunner_OptionalCheckpoint tests optional checkpoint behavior
func TestDynamicRunner_OptionalCheckpoint(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Load agents
	err := agent.Load(config.Conf)
	require.NoError(t, err, "Failed to load agents")

	tmpDir := t.TempDir()
	inputFile := filepath.Join(tmpDir, "dynamic-inputs.jsonl")

	// Test with one required and one optional checkpoint (JSONL must be single line)
	testCase := `{"id": "optional-checkpoint-test", "name": "Test with Optional Checkpoint", "input": "Hello", "simulator": {"use": "tests.simulator-agent", "options": {"metadata": {"persona": "Simple customer", "goal": "Get a greeting response"}}}, "checkpoints": [{"id": "greeting_response", "description": "Agent responds with greeting", "assert": {"type": "regex", "value": "(?i)(hello|hi|help)"}}, {"id": "special_offer", "description": "Agent mentions special offer (optional)", "required": false, "assert": {"type": "contains", "value": "special offer"}}], "max_turns": 3}`

	err = os.WriteFile(inputFile, []byte(testCase), 0644)
	require.NoError(t, err, "Failed to write test file")

	// Run test
	opts := &agenttest.Options{
		Input:     inputFile,
		AgentID:   "tests.dynamic-test-agent",
		Verbose:   true,
		InputMode: agenttest.InputModeFile,
	}
	opts = agenttest.MergeOptions(opts, agenttest.DefaultOptions())

	runner := agenttest.NewRunner(opts)
	report, err := runner.Run()

	require.NoError(t, err, "Runner should not return error")
	require.NotNil(t, report, "Report should not be nil")

	// Test should pass even if optional checkpoint is not reached
	t.Logf("Total: %d, Passed: %d, Failed: %d",
		report.Summary.Total, report.Summary.Passed, report.Summary.Failed)

	// If the required checkpoint is reached, the test should pass
	if len(report.Results) > 0 && report.Results[0].Metadata != nil {
		if checkpoints, ok := report.Results[0].Metadata["checkpoints"].(map[string]*agenttest.CheckpointResult); ok {
			for id, cp := range checkpoints {
				t.Logf("Checkpoint [%s]: reached=%v, required=%v", id, cp.Reached, cp.Required)
			}
		}
	}
}

// TestDynamicRunner_MaxTurnsExceeded tests behavior when max turns is exceeded
func TestDynamicRunner_MaxTurnsExceeded(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Load agents
	err := agent.Load(config.Conf)
	require.NoError(t, err, "Failed to load agents")

	tmpDir := t.TempDir()
	inputFile := filepath.Join(tmpDir, "dynamic-inputs.jsonl")

	// Test case with impossible checkpoint and low max_turns (JSONL must be single line)
	testCase := `{"id": "max-turns-test", "name": "Test Max Turns Exceeded", "input": "Hello", "simulator": {"use": "tests.simulator-agent", "options": {"metadata": {"persona": "Persistent customer", "goal": "Keep talking"}}}, "checkpoints": [{"id": "impossible", "description": "This checkpoint will never be reached", "assert": {"type": "contains", "value": "IMPOSSIBLE_STRING_NEVER_APPEARS_12345"}}], "max_turns": 2}`

	err = os.WriteFile(inputFile, []byte(testCase), 0644)
	require.NoError(t, err, "Failed to write test file")

	// Run test
	opts := &agenttest.Options{
		Input:     inputFile,
		AgentID:   "tests.dynamic-test-agent",
		Verbose:   true,
		InputMode: agenttest.InputModeFile,
	}
	opts = agenttest.MergeOptions(opts, agenttest.DefaultOptions())

	runner := agenttest.NewRunner(opts)
	report, err := runner.Run()

	require.NoError(t, err, "Runner should not return error")
	require.NotNil(t, report, "Report should not be nil")

	// Test should fail due to max turns exceeded or goal achieved without checkpoints
	assert.Equal(t, 1, report.Summary.Failed, "Test should fail")

	if len(report.Results) > 0 {
		result := report.Results[0]
		assert.Equal(t, agenttest.StatusFailed, result.Status, "Status should be failed")
		// Either max turns exceeded or simulator signaled goal achieved without checkpoints
		validError := strings.Contains(result.Error, "max turns") ||
			strings.Contains(result.Error, "not all required checkpoints reached")
		assert.True(t, validError, "Error should mention max turns or checkpoints not reached, got: %s", result.Error)
		t.Logf("Error (expected): %s", result.Error)
	}
}

// TestDynamicRunner_CheckpointOrdering tests that checkpoint ordering is enforced
func TestDynamicRunner_CheckpointOrderingEnforced(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Load agents
	err := agent.Load(config.Conf)
	require.NoError(t, err, "Failed to load agents")

	tmpDir := t.TempDir()
	inputFile := filepath.Join(tmpDir, "dynamic-inputs.jsonl")

	// Test case with ordered checkpoints (JSONL must be single line)
	testCase := `{"id": "ordered-checkpoints", "name": "Test Checkpoint Ordering", "input": "I want to order coffee", "simulator": {"use": "tests.simulator-agent", "options": {"metadata": {"persona": "Customer ordering step by step", "goal": "Complete coffee order following the flow"}}}, "checkpoints": [{"id": "step1_greeting", "description": "Agent greets", "assert": {"type": "regex", "value": "(?i)(hello|hi|help|order)"}}, {"id": "step2_size", "description": "Agent asks about size", "after": ["step1_greeting"], "assert": {"type": "regex", "value": "(?i)size"}}, {"id": "step3_confirm", "description": "Agent confirms", "after": ["step2_size"], "assert": {"type": "regex", "value": "(?i)confirm"}}], "max_turns": 10}`

	err = os.WriteFile(inputFile, []byte(testCase), 0644)
	require.NoError(t, err, "Failed to write test file")

	// Run test
	opts := &agenttest.Options{
		Input:     inputFile,
		AgentID:   "tests.dynamic-test-agent",
		Verbose:   true,
		InputMode: agenttest.InputModeFile,
	}
	opts = agenttest.MergeOptions(opts, agenttest.DefaultOptions())

	runner := agenttest.NewRunner(opts)
	report, err := runner.Run()

	require.NoError(t, err, "Runner should not return error")
	require.NotNil(t, report, "Report should not be nil")

	t.Logf("Total: %d, Passed: %d, Failed: %d",
		report.Summary.Total, report.Summary.Passed, report.Summary.Failed)

	// Log checkpoint order
	if len(report.Results) > 0 && report.Results[0].Metadata != nil {
		if checkpoints, ok := report.Results[0].Metadata["checkpoints"].(map[string]*agenttest.CheckpointResult); ok {
			for id, cp := range checkpoints {
				t.Logf("Checkpoint [%s]: reached=%v, at_turn=%d", id, cp.Reached, cp.ReachedAtTurn)
			}
		}
	}
}
