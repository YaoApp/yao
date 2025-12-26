package test_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent"
	agenttest "github.com/yaoapp/yao/agent/test"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/test"
)

// TestRunner_AgentDrivenInput tests the complete flow:
// 1. Use generator-agent to generate test cases
// 2. Run the generated tests against simple-greeting agent
func TestRunner_AgentDrivenInput(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Load agents
	err := agent.Load(config.Conf)
	require.NoError(t, err, "Failed to load agents")

	// Test with agent-driven input
	opts := &agenttest.Options{
		Input:     "agents:tests.generator-agent?count=3",
		AgentID:   "tests.simple-greeting",
		Verbose:   true,
		InputMode: agenttest.InputModeFile, // Will be overridden by ParseInputSource
	}
	opts = agenttest.MergeOptions(opts, agenttest.DefaultOptions())

	runner := agenttest.NewRunner(opts)
	report, err := runner.Run()

	require.NoError(t, err, "Runner should not return error")
	require.NotNil(t, report, "Report should not be nil")
	require.NotNil(t, report.Summary, "Summary should not be nil")

	// Verify report
	assert.Greater(t, report.Summary.Total, 0, "Should have at least one test case")
	t.Logf("Total: %d, Passed: %d, Failed: %d",
		report.Summary.Total, report.Summary.Passed, report.Summary.Failed)
}

// TestRunner_AgentDrivenInput_DryRun tests dry-run mode
func TestRunner_AgentDrivenInput_DryRun(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Load agents
	err := agent.Load(config.Conf)
	require.NoError(t, err, "Failed to load agents")

	// Test with dry-run mode
	opts := &agenttest.Options{
		Input:   "agents:tests.generator-agent?count=2",
		AgentID: "tests.simple-greeting",
		DryRun:  true,
		Verbose: true,
	}
	opts = agenttest.MergeOptions(opts, agenttest.DefaultOptions())

	runner := agenttest.NewRunner(opts)
	report, err := runner.Run()

	require.NoError(t, err, "Dry-run should not return error")
	require.NotNil(t, report, "Report should not be nil")

	// In dry-run mode, tests are generated but not executed
	// So Passed and Failed should both be 0, but Total should have the count
	assert.Greater(t, report.Summary.Total, 0, "Should have generated test cases")

	t.Logf("Generated %d test cases in dry-run mode", report.Summary.Total)
}

// TestRunner_FileInput tests loading test cases from JSONL file
func TestRunner_FileInput(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Load agents
	err := agent.Load(config.Conf)
	require.NoError(t, err, "Failed to load agents")

	// Create a temporary JSONL file with test cases
	// Use case-insensitive contains for robustness
	tmpDir := t.TempDir()
	inputFile := filepath.Join(tmpDir, "inputs.jsonl")

	testCases := `{"id": "greeting-hello", "input": "Hello", "assert": {"type": "regex", "value": "(?i)hello"}}
{"id": "greeting-hi", "input": "Hi there", "assert": {"type": "regex", "value": "(?i)(hi|hello)"}}
{"id": "greeting-morning", "input": "Good morning", "assert": {"type": "regex", "value": "(?i)(hello|morning|good)"}}`

	err = os.WriteFile(inputFile, []byte(testCases), 0644)
	require.NoError(t, err, "Failed to write test file")

	// Run tests from file
	opts := &agenttest.Options{
		Input:     inputFile,
		AgentID:   "tests.simple-greeting",
		Verbose:   true,
		InputMode: agenttest.InputModeFile,
	}
	opts = agenttest.MergeOptions(opts, agenttest.DefaultOptions())

	runner := agenttest.NewRunner(opts)
	report, err := runner.Run()

	require.NoError(t, err, "Runner should not return error")
	require.NotNil(t, report, "Report should not be nil")

	// Verify report
	assert.Equal(t, 3, report.Summary.Total, "Should have 3 test cases")
	t.Logf("Total: %d, Passed: %d, Failed: %d",
		report.Summary.Total, report.Summary.Passed, report.Summary.Failed)

	// Check results for debugging
	if report.Results != nil {
		for _, r := range report.Results {
			t.Logf("  [%s] Status: %s, Output: %v", r.ID, r.Status, r.Output)
		}
	}
}

// TestRunner_DirectMessage tests direct message mode
func TestRunner_DirectMessage(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Load agents
	err := agent.Load(config.Conf)
	require.NoError(t, err, "Failed to load agents")

	// Test with direct message
	opts := &agenttest.Options{
		Input:     "Hello, how are you?",
		AgentID:   "tests.simple-greeting",
		Verbose:   true,
		InputMode: agenttest.InputModeMessage,
	}
	opts = agenttest.MergeOptions(opts, agenttest.DefaultOptions())

	runner := agenttest.NewRunner(opts)
	report, err := runner.Run()

	require.NoError(t, err, "Runner should not return error")
	require.NotNil(t, report, "Report should not be nil")

	// Direct message mode returns a minimal report
	assert.Equal(t, 1, report.Summary.Total, "Should have 1 test case")
	assert.Equal(t, 1, report.Summary.Passed, "Direct message should pass")
}

// TestRunner_WithBeforeAfter tests before/after hooks
func TestRunner_WithBeforeAfter(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Load agents
	err := agent.Load(config.Conf)
	require.NoError(t, err, "Failed to load agents")

	// Create a temporary JSONL file with test cases that use hooks
	tmpDir := t.TempDir()
	inputFile := filepath.Join(tmpDir, "inputs.jsonl")

	// Note: hooks-test agent has env_test.ts with Before/After functions
	testCases := `{"id": "hook-test-1", "input": "Hello", "assert": {"type": "contains", "value": "hello"}, "before": "env_test.Before", "after": "env_test.After"}`

	err = os.WriteFile(inputFile, []byte(testCases), 0644)
	require.NoError(t, err, "Failed to write test file")

	// Run tests with hooks (using hooks-test agent which has the hook scripts)
	opts := &agenttest.Options{
		Input:     inputFile,
		AgentID:   "tests.hooks-test",
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

// TestRunner_Parallel tests parallel execution
func TestRunner_Parallel(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Load agents
	err := agent.Load(config.Conf)
	require.NoError(t, err, "Failed to load agents")

	// Create a temporary JSONL file with multiple test cases
	// Use regex for case-insensitive matching
	tmpDir := t.TempDir()
	inputFile := filepath.Join(tmpDir, "inputs.jsonl")

	testCases := `{"id": "parallel-1", "input": "Hello", "assert": {"type": "regex", "value": "(?i)(hello|hi)"}}
{"id": "parallel-2", "input": "Hi", "assert": {"type": "regex", "value": "(?i)(hello|hi)"}}
{"id": "parallel-3", "input": "Hey", "assert": {"type": "regex", "value": "(?i)(hello|hi|hey)"}}
{"id": "parallel-4", "input": "Good day", "assert": {"type": "regex", "value": "(?i)(hello|good|day)"}}`

	err = os.WriteFile(inputFile, []byte(testCases), 0644)
	require.NoError(t, err, "Failed to write test file")

	// Run tests in parallel
	opts := &agenttest.Options{
		Input:     inputFile,
		AgentID:   "tests.simple-greeting",
		Parallel:  2, // Run 2 tests in parallel
		Verbose:   true,
		InputMode: agenttest.InputModeFile,
	}
	opts = agenttest.MergeOptions(opts, agenttest.DefaultOptions())

	runner := agenttest.NewRunner(opts)
	report, err := runner.Run()

	require.NoError(t, err, "Runner should not return error")
	require.NotNil(t, report, "Report should not be nil")

	assert.Equal(t, 4, report.Summary.Total, "Should have 4 test cases")
	t.Logf("Total: %d, Passed: %d, Failed: %d (parallel: 2)",
		report.Summary.Total, report.Summary.Passed, report.Summary.Failed)
}

// TestRunner_FailFast tests fail-fast behavior
func TestRunner_FailFast(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Load agents
	err := agent.Load(config.Conf)
	require.NoError(t, err, "Failed to load agents")

	// Create a temporary JSONL file with a failing test first
	tmpDir := t.TempDir()
	inputFile := filepath.Join(tmpDir, "inputs.jsonl")

	// First test will fail (expects "impossible" which won't be in response)
	testCases := `{"id": "fail-first", "input": "Hello", "assert": {"type": "contains", "value": "IMPOSSIBLE_STRING_12345"}}
{"id": "should-skip", "input": "Hi", "assert": {"type": "contains", "value": "hi"}}`

	err = os.WriteFile(inputFile, []byte(testCases), 0644)
	require.NoError(t, err, "Failed to write test file")

	// Run tests with fail-fast
	opts := &agenttest.Options{
		Input:     inputFile,
		AgentID:   "tests.simple-greeting",
		FailFast:  true,
		Verbose:   true,
		InputMode: agenttest.InputModeFile,
	}
	opts = agenttest.MergeOptions(opts, agenttest.DefaultOptions())

	runner := agenttest.NewRunner(opts)
	report, err := runner.Run()

	require.NoError(t, err, "Runner should not return error (fail-fast is not an error)")
	require.NotNil(t, report, "Report should not be nil")

	// With fail-fast, only the first test should run
	assert.Equal(t, 1, report.Summary.Failed, "First test should fail")
	// The second test might not run due to fail-fast
	t.Logf("Total: %d, Passed: %d, Failed: %d (fail-fast enabled)",
		report.Summary.Total, report.Summary.Passed, report.Summary.Failed)
}
