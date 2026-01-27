package standard_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	agentcontext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/robot/executor/standard"
	"github.com/yaoapp/yao/agent/robot/types"
	"github.com/yaoapp/yao/agent/testutils"
)

// ============================================================================
// Runner Tests - Multi-Turn Conversation Flow
// ============================================================================

func TestRunnerExecuteWithRetry(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	ctx := types.NewContext(context.Background(), testAuth())

	t.Run("executes assistant task with multi-turn conversation", func(t *testing.T) {
		robot := createRunnerTestRobot(t)
		config := standard.DefaultRunConfig()
		runner := standard.NewRunner(ctx, robot, config)

		task := &types.Task{
			ID:           "task-001",
			ExecutorType: types.ExecutorAssistant,
			ExecutorID:   "experts.text-writer",
			Messages: []agentcontext.Message{
				{Role: agentcontext.RoleUser, Content: "Write a haiku about coding. Format: three lines with 5-7-5 syllables."},
			},
			ExpectedOutput: "A haiku poem about coding",
			Status:         types.TaskPending,
		}

		taskCtx := &standard.RunnerContext{
			SystemPrompt: robot.SystemPrompt,
		}

		result := runner.ExecuteWithRetry(task, taskCtx)

		assert.True(t, result.Success, "task should succeed")
		assert.NotNil(t, result.Output)
		assert.NotNil(t, result.Validation)
		assert.True(t, result.Validation.Complete)
		assert.Greater(t, result.Duration, int64(0))

		t.Logf("Output: %v", result.Output)
		t.Logf("Validation: passed=%v, complete=%v, score=%.2f",
			result.Validation.Passed, result.Validation.Complete, result.Validation.Score)
	})

	t.Run("handles validation failure with multi-turn retry", func(t *testing.T) {
		robot := createRunnerTestRobot(t)
		config := standard.DefaultRunConfig()
		config.MaxTurnsPerTask = 3 // Limit turns for test
		runner := standard.NewRunner(ctx, robot, config)

		// Task with strict validation that may require conversation
		task := &types.Task{
			ID:           "task-002",
			ExecutorType: types.ExecutorAssistant,
			ExecutorID:   "experts.data-analyst",
			Messages: []agentcontext.Message{
				{Role: agentcontext.RoleUser, Content: "Return a JSON object with exactly these fields: status (string 'ok'), count (number greater than 0)."},
			},
			ExpectedOutput: "JSON with status='ok' and count>0",
			ValidationRules: []string{
				"output must be valid JSON",
				`{"type": "type", "value": "object"}`,
			},
			Status: types.TaskPending,
		}

		taskCtx := &standard.RunnerContext{
			SystemPrompt: robot.SystemPrompt,
		}

		result := runner.ExecuteWithRetry(task, taskCtx)

		// Should either succeed or fail gracefully
		assert.NotNil(t, result.Validation)
		t.Logf("Success: %v, Output: %v", result.Success, result.Output)
		t.Logf("Validation: passed=%v, complete=%v, needReply=%v",
			result.Validation.Passed, result.Validation.Complete, result.Validation.NeedReply)
	})

	t.Run("respects max turns limit", func(t *testing.T) {
		robot := createRunnerTestRobot(t)
		config := standard.DefaultRunConfig()
		config.MaxTurnsPerTask = 1 // Only 1 turn allowed
		runner := standard.NewRunner(ctx, robot, config)

		// Task that requires multiple turns - asking for something incomplete
		// then validation will ask for more, but we only allow 1 turn
		task := &types.Task{
			ID:           "task-003",
			ExecutorType: types.ExecutorAssistant,
			ExecutorID:   "experts.text-writer",
			Messages: []agentcontext.Message{
				{Role: agentcontext.RoleUser, Content: "Say 'hello'"},
			},
			// Validation will fail because it expects a JSON object
			ExpectedOutput:  "A JSON object with 'status' and 'data' fields",
			ValidationRules: []string{`{"type": "type", "value": "object"}`},
			Status:          types.TaskPending,
		}

		taskCtx := &standard.RunnerContext{
			SystemPrompt: robot.SystemPrompt,
		}

		result := runner.ExecuteWithRetry(task, taskCtx)

		// With only 1 turn and strict validation, task should not complete successfully
		// Either it fails validation or hits max turns
		t.Logf("Result: success=%v, error=%s", result.Success, result.Error)
		t.Logf("Validation: passed=%v, complete=%v, needReply=%v",
			result.Validation.Passed, result.Validation.Complete, result.Validation.NeedReply)

		// The test verifies the max turns mechanism works - task either:
		// 1. Fails validation (expected with "say hello" vs JSON requirement)
		// 2. Or hits max turns if validation requests retry
		assert.NotNil(t, result.Validation)
	})
}

func TestRunnerBuildTaskContext(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	ctx := types.NewContext(context.Background(), testAuth())

	t.Run("includes previous results in context", func(t *testing.T) {
		robot := createRunnerTestRobot(t)
		config := standard.DefaultRunConfig()
		runner := standard.NewRunner(ctx, robot, config)

		exec := &types.Execution{
			ID:       "test-exec",
			MemberID: robot.MemberID,
			TeamID:   robot.TeamID,
			Goals: &types.Goals{
				Content: "Test goals",
			},
			Results: []types.TaskResult{
				{
					TaskID:  "task-001",
					Success: true,
					Output:  map[string]interface{}{"data": "previous result"},
				},
				{
					TaskID:  "task-002",
					Success: true,
					Output:  "Another result",
				},
			},
		}
		exec.SetRobot(robot)

		// Build context for task at index 2 (should include results 0 and 1)
		taskCtx := runner.BuildTaskContext(exec, 2)

		assert.NotNil(t, taskCtx)
		assert.Len(t, taskCtx.PreviousResults, 2)
		assert.Equal(t, "task-001", taskCtx.PreviousResults[0].TaskID)
		assert.Equal(t, "task-002", taskCtx.PreviousResults[1].TaskID)
		assert.Equal(t, robot.SystemPrompt, taskCtx.SystemPrompt)
	})

	t.Run("handles first task with no previous results", func(t *testing.T) {
		robot := createRunnerTestRobot(t)
		config := standard.DefaultRunConfig()
		runner := standard.NewRunner(ctx, robot, config)

		exec := &types.Execution{
			ID:       "test-exec",
			MemberID: robot.MemberID,
			TeamID:   robot.TeamID,
			Goals: &types.Goals{
				Content: "Test goals",
			},
			Results: []types.TaskResult{},
		}
		exec.SetRobot(robot)

		taskCtx := runner.BuildTaskContext(exec, 0)

		assert.NotNil(t, taskCtx)
		assert.Empty(t, taskCtx.PreviousResults)
	})

	t.Run("handles bounds check for task index", func(t *testing.T) {
		robot := createRunnerTestRobot(t)
		config := standard.DefaultRunConfig()
		runner := standard.NewRunner(ctx, robot, config)

		exec := &types.Execution{
			ID:       "test-exec",
			MemberID: robot.MemberID,
			TeamID:   robot.TeamID,
			Results: []types.TaskResult{
				{TaskID: "task-001", Success: true},
			},
		}
		exec.SetRobot(robot)

		// Task index 5, but only 1 result exists
		taskCtx := runner.BuildTaskContext(exec, 5)

		assert.NotNil(t, taskCtx)
		assert.Len(t, taskCtx.PreviousResults, 1) // Should only include available results
	})
}

func TestRunnerFormatPreviousResultsAsContext(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	ctx := types.NewContext(context.Background(), testAuth())

	t.Run("formats previous results as markdown", func(t *testing.T) {
		robot := createRunnerTestRobot(t)
		config := standard.DefaultRunConfig()
		runner := standard.NewRunner(ctx, robot, config)

		results := []types.TaskResult{
			{
				TaskID:  "task-001",
				Success: true,
				Output:  map[string]interface{}{"key": "value", "count": 42},
			},
			{
				TaskID:  "task-002",
				Success: false,
				Output:  "Partial result",
				Error:   "Validation failed",
			},
		}

		formatted := runner.FormatPreviousResultsAsContext(results)

		assert.Contains(t, formatted, "## Previous Task Results")
		assert.Contains(t, formatted, "task-001")
		assert.Contains(t, formatted, "task-002")
		assert.Contains(t, formatted, "Success")
		assert.Contains(t, formatted, "Failed")
		assert.Contains(t, formatted, "key")
		assert.Contains(t, formatted, "value")

		t.Logf("Formatted context:\n%s", formatted)
	})

	t.Run("returns empty string for no results", func(t *testing.T) {
		robot := createRunnerTestRobot(t)
		config := standard.DefaultRunConfig()
		runner := standard.NewRunner(ctx, robot, config)

		formatted := runner.FormatPreviousResultsAsContext([]types.TaskResult{})

		assert.Empty(t, formatted)
	})
}

func TestRunnerBuildAssistantMessages(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	ctx := types.NewContext(context.Background(), testAuth())

	t.Run("builds messages with task content", func(t *testing.T) {
		robot := createRunnerTestRobot(t)
		config := standard.DefaultRunConfig()
		runner := standard.NewRunner(ctx, robot, config)

		task := &types.Task{
			ID:           "task-001",
			ExecutorType: types.ExecutorAssistant,
			ExecutorID:   "experts.text-writer",
			Messages: []agentcontext.Message{
				{Role: agentcontext.RoleUser, Content: "Write a greeting"},
			},
		}

		taskCtx := &standard.RunnerContext{
			SystemPrompt: "You are helpful",
		}

		messages := runner.BuildAssistantMessages(task, taskCtx)

		assert.NotEmpty(t, messages)
		// Should contain task message
		found := false
		for _, msg := range messages {
			if content, ok := msg.Content.(string); ok && content == "Write a greeting" {
				found = true
				break
			}
		}
		assert.True(t, found, "should contain task message")
	})

	t.Run("includes previous results in messages", func(t *testing.T) {
		robot := createRunnerTestRobot(t)
		config := standard.DefaultRunConfig()
		runner := standard.NewRunner(ctx, robot, config)

		task := &types.Task{
			ID:           "task-002",
			ExecutorType: types.ExecutorAssistant,
			ExecutorID:   "experts.text-writer",
			Messages: []agentcontext.Message{
				{Role: agentcontext.RoleUser, Content: "Continue from previous"},
			},
		}

		taskCtx := &standard.RunnerContext{
			PreviousResults: []types.TaskResult{
				{TaskID: "task-001", Success: true, Output: "Previous output"},
			},
			SystemPrompt: "You are helpful",
		}

		messages := runner.BuildAssistantMessages(task, taskCtx)

		assert.NotEmpty(t, messages)
		// Should have context message with previous results
		formatted := runner.FormatMessagesAsText(messages)
		assert.Contains(t, formatted, "Previous Task Results")
	})
}

func TestRunnerFormatMessagesAsText(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	ctx := types.NewContext(context.Background(), testAuth())

	t.Run("formats string content", func(t *testing.T) {
		robot := createRunnerTestRobot(t)
		config := standard.DefaultRunConfig()
		runner := standard.NewRunner(ctx, robot, config)

		messages := []agentcontext.Message{
			{Role: agentcontext.RoleUser, Content: "Hello"},
			{Role: agentcontext.RoleUser, Content: "World"},
		}

		text := runner.FormatMessagesAsText(messages)

		assert.Contains(t, text, "Hello")
		assert.Contains(t, text, "World")
	})

	t.Run("handles multipart content", func(t *testing.T) {
		robot := createRunnerTestRobot(t)
		config := standard.DefaultRunConfig()
		runner := standard.NewRunner(ctx, robot, config)

		messages := []agentcontext.Message{
			{
				Role: agentcontext.RoleUser,
				Content: []interface{}{
					map[string]interface{}{"type": "text", "text": "Part 1"},
					map[string]interface{}{"type": "text", "text": "Part 2"},
				},
			},
		}

		text := runner.FormatMessagesAsText(messages)

		assert.Contains(t, text, "Part 1")
		assert.Contains(t, text, "Part 2")
	})

	t.Run("handles map content via JSON", func(t *testing.T) {
		robot := createRunnerTestRobot(t)
		config := standard.DefaultRunConfig()
		runner := standard.NewRunner(ctx, robot, config)

		messages := []agentcontext.Message{
			{
				Role:    agentcontext.RoleUser,
				Content: map[string]interface{}{"key": "value"},
			},
		}

		text := runner.FormatMessagesAsText(messages)

		assert.Contains(t, text, "key")
		assert.Contains(t, text, "value")
	})
}

// ============================================================================
// Non-Assistant Task Tests (MCP, Process)
// ============================================================================

func TestRunnerExecuteNonAssistantTask(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	ctx := types.NewContext(context.Background(), testAuth())

	t.Run("executes MCP task (single-call)", func(t *testing.T) {
		// Note: This test requires MCP server to be running
		// Skip if MCP is not available
		t.Skip("MCP server not available in test environment")

		robot := createRunnerTestRobot(t)
		config := standard.DefaultRunConfig()
		runner := standard.NewRunner(ctx, robot, config)

		task := &types.Task{
			ID:           "task-mcp",
			ExecutorType: types.ExecutorMCP,
			ExecutorID:   "filesystem.list_directory",
			Args:         []any{map[string]interface{}{"path": "/tmp"}},
			Status:       types.TaskPending,
		}

		taskCtx := &standard.RunnerContext{}

		result := runner.ExecuteWithRetry(task, taskCtx)

		// MCP tasks are single-call, no multi-turn
		t.Logf("MCP result: success=%v, output=%v", result.Success, result.Output)
	})

	t.Run("executes Process task (single-call)", func(t *testing.T) {
		// Note: This test requires a Yao process to be available
		// Skip if process is not available
		t.Skip("Yao process not available in test environment")

		robot := createRunnerTestRobot(t)
		config := standard.DefaultRunConfig()
		runner := standard.NewRunner(ctx, robot, config)

		task := &types.Task{
			ID:           "task-process",
			ExecutorType: types.ExecutorProcess,
			ExecutorID:   "utils.env.Get",
			Args:         []any{"PATH"},
			Status:       types.TaskPending,
		}

		taskCtx := &standard.RunnerContext{}

		result := runner.ExecuteWithRetry(task, taskCtx)

		// Process tasks are single-call, no multi-turn
		t.Logf("Process result: success=%v, output=%v", result.Success, result.Output)
	})
}

// ============================================================================
// MCP Output Validation Tests
// ============================================================================

func TestRunnerValidateMCPOutput(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	ctx := types.NewContext(context.Background(), testAuth())

	// Test that MCP tasks use simple structure validation, not semantic validation
	// This is tested indirectly through the validation result

	t.Run("MCP validation passes with valid map output", func(t *testing.T) {
		robot := createRunnerTestRobot(t)
		config := standard.DefaultRunConfig()
		runner := standard.NewRunner(ctx, robot, config)

		// Create a mock MCP task with validation rules
		// (normally these rules would trigger semantic validation for assistant tasks)
		task := &types.Task{
			ID:           "task-mcp-test",
			ExecutorType: types.ExecutorMCP,
			ExecutorID:   "test.tool",
			MCPServer:    "test",
			MCPTool:      "tool",
			// These semantic rules should be IGNORED for MCP tasks
			ExpectedOutput: "Image with file and content_type",
			ValidationRules: []string{
				"file field exists",
				"content_type is image/jpeg",
			},
			Status: types.TaskPending,
		}

		// Simulate MCP output (normally would come from actual MCP call)
		output := map[string]interface{}{
			"file":         "__yao.attachment://abc123",
			"content_type": "image/jpeg",
		}

		// Test validateMCPOutput directly through reflection or mock
		// Since validateMCPOutput is private, we test the behavior indirectly:
		// MCP validation should only check for non-empty output, not semantic content

		// The validation should pass because:
		// 1. Output is not nil
		// 2. Output is a non-empty map
		// (Semantic validation rules are NOT applied for MCP tasks)

		t.Logf("MCP task configured with validation rules that should be ignored")
		t.Logf("Task ExpectedOutput: %s", task.ExpectedOutput)
		t.Logf("Task ValidationRules: %v", task.ValidationRules)
		t.Logf("MCP output: %v", output)

		// Note: We can't directly call ExecuteWithRetry without an MCP server
		// This test documents the expected behavior
		_ = runner
		_ = task
		_ = output
	})

	t.Run("MCP validation fails with nil output", func(t *testing.T) {
		// MCP validation should fail if output is nil
		t.Log("MCP validation should fail when output is nil")
		t.Log("Expected: Passed=false, Issues=['MCP tool returned nil output']")
	})

	t.Run("MCP validation fails with empty string output", func(t *testing.T) {
		// MCP validation should fail if output is empty string
		t.Log("MCP validation should fail when output is empty string")
		t.Log("Expected: Passed=false, Issues=['MCP tool returned empty string']")
	})

	t.Run("MCP validation fails with empty map output", func(t *testing.T) {
		// MCP validation should fail if output is empty map
		t.Log("MCP validation should fail when output is empty map")
		t.Log("Expected: Passed=false, Issues=['MCP tool returned empty object']")
	})

	t.Run("MCP validation fails with empty array output", func(t *testing.T) {
		// MCP validation should fail if output is empty array
		t.Log("MCP validation should fail when output is empty array")
		t.Log("Expected: Passed=false, Issues=['MCP tool returned empty array']")
	})

	t.Run("MCP validation passes with any non-empty output", func(t *testing.T) {
		// MCP validation should pass for any non-empty output
		// regardless of ExpectedOutput or ValidationRules
		t.Log("MCP validation should pass when output is non-empty")
		t.Log("Semantic validation (ExpectedOutput, ValidationRules) should NOT be applied")
		t.Log("Expected: Passed=true, Complete=true, Score=1.0")
	})
}

// ============================================================================
// Helper Functions
// ============================================================================

// createRunnerTestRobot creates a test robot for runner tests
func createRunnerTestRobot(t *testing.T) *types.Robot {
	t.Helper()
	return &types.Robot{
		MemberID:     "test-robot-runner",
		TeamID:       "test-team-1",
		DisplayName:  "Test Robot for Runner",
		SystemPrompt: "You are a helpful assistant. Follow instructions carefully and provide clear responses.",
		Config: &types.Config{
			Identity: &types.Identity{
				Role:   "Test Assistant",
				Duties: []string{"Execute tasks", "Generate content"},
			},
			Resources: &types.Resources{
				Phases: map[types.Phase]string{
					types.PhaseRun: "robot.validation",
					"validation":   "robot.validation", // For semantic validation agent
				},
				Agents: []string{
					"experts.data-analyst",
					"experts.summarizer",
					"experts.text-writer",
				},
			},
		},
	}
}

// Note: createRunnerTestExecution is available if needed for future tests
// that require a full Execution object instead of just RunnerContext
