package test_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	v8 "github.com/yaoapp/gou/runtime/v8"
	agenttest "github.com/yaoapp/yao/agent/test"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/test"
)

const hooksTestAgent = "assistants/tests/hooks-test"

func TestParseHookRef(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantFile  string
		wantFunc  string
		expectErr bool
	}{
		{
			name:     "function only",
			input:    "Before",
			wantFile: "",
			wantFunc: "Before",
		},
		{
			name:     "with script file",
			input:    "env_test.Before",
			wantFile: "env_test.ts",
			wantFunc: "Before",
		},
		{
			name:     "with src prefix",
			input:    "src/env_test.Before",
			wantFile: "env_test.ts",
			wantFunc: "Before",
		},
		{
			name:     "nested path",
			input:    "setup/db_test.Before",
			wantFile: "setup/db_test.ts",
			wantFunc: "Before",
		},
		{
			name:      "empty string",
			input:     "",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ref, err := agenttest.ParseHookRef(tt.input)
			if tt.expectErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.wantFile, ref.ScriptFile)
			assert.Equal(t, tt.wantFunc, ref.Function)
		})
	}
}

func TestHookExecutorLoadTestScripts(t *testing.T) {
	// Prepare test environment
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Load agent test scripts using the utility function
	scripts := test.LoadAgentTestScripts(t, hooksTestAgent)

	assert.NotEmpty(t, scripts, "Should load at least one test script")

	// Verify the script was loaded into V8
	found := false
	for _, scriptID := range scripts {
		if _, ok := v8.Scripts[scriptID]; ok {
			found = true
			t.Logf("Loaded script: %s", scriptID)
			break
		}
	}
	assert.True(t, found, "At least one script should be loaded into V8")
}

func TestHookExecutorExecuteBefore(t *testing.T) {
	// Prepare test environment
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Load agent test scripts
	test.LoadAgentTestScripts(t, hooksTestAgent)

	executor := agenttest.NewHookExecutor(true)

	testCase := &agenttest.Case{
		ID:    "TEST001",
		Input: "Hello World",
	}

	// Execute Before hook
	beforeData, err := executor.ExecuteBefore("env_test.Before", testCase, hooksTestAgent)
	assert.NoError(t, err)
	assert.NotNil(t, beforeData)

	// Verify returned data
	dataMap, ok := beforeData.(map[string]interface{})
	assert.True(t, ok, "beforeData should be a map")
	assert.Equal(t, "TEST001", dataMap["test_id"])
	assert.NotEmpty(t, dataMap["mock_user_id"])
	assert.NotEmpty(t, dataMap["mock_session_id"])
}

func TestHookExecutorExecuteAfter(t *testing.T) {
	// Prepare test environment
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Load agent test scripts
	test.LoadAgentTestScripts(t, hooksTestAgent)

	executor := agenttest.NewHookExecutor(true)

	testCase := &agenttest.Case{
		ID:    "TEST002",
		Input: "Test input",
	}

	result := &agenttest.Result{
		ID:         "TEST002",
		Status:     agenttest.StatusPassed,
		DurationMs: 100,
	}

	beforeData := map[string]interface{}{
		"test_id":         "TEST002",
		"mock_user_id":    "user_TEST002_12345",
		"mock_session_id": "session_12345",
	}

	// Execute After hook
	err := executor.ExecuteAfter("env_test.After", testCase, result, beforeData, hooksTestAgent)
	assert.NoError(t, err)
}

func TestHookExecutorExecuteBeforeAll(t *testing.T) {
	// Prepare test environment
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Load agent test scripts
	test.LoadAgentTestScripts(t, hooksTestAgent)

	executor := agenttest.NewHookExecutor(true)

	testCases := []*agenttest.Case{
		{ID: "T001", Input: "Test 1"},
		{ID: "T002", Input: "Test 2"},
		{ID: "T003", Input: "Test 3"},
	}

	// Execute BeforeAll hook
	globalData, err := executor.ExecuteBeforeAll("env_test.BeforeAll", testCases, hooksTestAgent)
	assert.NoError(t, err)
	assert.NotNil(t, globalData)

	// Verify returned data
	dataMap, ok := globalData.(map[string]interface{})
	assert.True(t, ok, "globalData should be a map")
	assert.NotEmpty(t, dataMap["suite_id"])
	assert.Equal(t, float64(3), dataMap["test_count"]) // JSON numbers are float64
}

func TestHookExecutorExecuteAfterAll(t *testing.T) {
	// Prepare test environment
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Load agent test scripts
	test.LoadAgentTestScripts(t, hooksTestAgent)

	executor := agenttest.NewHookExecutor(true)

	results := []*agenttest.Result{
		{ID: "T001", Status: agenttest.StatusPassed, DurationMs: 100},
		{ID: "T002", Status: agenttest.StatusFailed, DurationMs: 200, Error: "assertion failed"},
		{ID: "T003", Status: agenttest.StatusPassed, DurationMs: 150},
	}

	globalData := map[string]interface{}{
		"suite_id":   "suite_12345",
		"test_count": 3,
	}

	// Execute AfterAll hook
	err := executor.ExecuteAfterAll("env_test.AfterAll", results, globalData, hooksTestAgent)
	assert.NoError(t, err)
}

func TestHookExecutorFunctionNotFound(t *testing.T) {
	// Prepare test environment
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Load agent test scripts
	test.LoadAgentTestScripts(t, hooksTestAgent)

	executor := agenttest.NewHookExecutor(true)

	testCase := &agenttest.Case{
		ID:    "TEST001",
		Input: "Hello",
	}

	// Try to execute non-existent function
	_, err := executor.ExecuteBefore("env_test.NonExistent", testCase, hooksTestAgent)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not defined")
}

func TestHookExecutorScriptNotFound(t *testing.T) {
	// Prepare test environment
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Load agent test scripts
	test.LoadAgentTestScripts(t, hooksTestAgent)

	executor := agenttest.NewHookExecutor(true)

	testCase := &agenttest.Case{
		ID:    "TEST001",
		Input: "Hello",
	}

	// Try to execute from non-existent script
	_, err := executor.ExecuteBefore("nonexistent_test.Before", testCase, hooksTestAgent)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}
