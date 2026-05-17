//go:build e2e

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
// Runner Tests - V2 Simplified Execution (single call, no validation loop)
// ============================================================================

func TestRunnerExecuteTask(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.PrepareAgent(t)
	testutils.RequireE2EKeys(t)
	defer testutils.Clean(t)

	ctx := types.NewContext(context.Background(), testAuth())

	t.Run("executes assistant task successfully", func(t *testing.T) {
		robot := createRunnerTestRobot(t)
		config := standard.DefaultRunConfig()
		runner := standard.NewRunner(ctx, robot, config, "", "test")

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

		result := runner.ExecuteTask(task, taskCtx)

		assert.True(t, result.Success, "task should succeed")
		assert.NotNil(t, result.Output, "output should not be nil")
		assert.Empty(t, result.Error, "error should be empty on success")
		assert.Greater(t, result.Duration, int64(0), "duration should be positive")

		t.Logf("Output: %v", result.Output)
	})

	t.Run("returns success without validation for assistant tasks", func(t *testing.T) {
		robot := createRunnerTestRobot(t)
		config := standard.DefaultRunConfig()
		runner := standard.NewRunner(ctx, robot, config, "", "test")

		task := &types.Task{
			ID:           "task-002",
			ExecutorType: types.ExecutorAssistant,
			ExecutorID:   "experts.data-analyst",
			Messages: []agentcontext.Message{
				{Role: agentcontext.RoleUser, Content: "Return a JSON object with exactly these fields: status (string 'ok'), count (number greater than 0)."},
			},
			ExpectedOutput: "JSON with status='ok' and count>0",
			Status:         types.TaskPending,
		}

		taskCtx := &standard.RunnerContext{
			SystemPrompt: robot.SystemPrompt,
		}

		result := runner.ExecuteTask(task, taskCtx)

		// V2: success is determined by the call succeeding, not by validation
		assert.True(t, result.Success, "task should succeed if assistant call returns")
		assert.NotNil(t, result.Output, "output should not be nil")
		assert.Nil(t, result.Validation, "V2 does not set Validation in runner")

		t.Logf("Success: %v, Output: %v", result.Success, result.Output)
	})

	t.Run("handles empty messages gracefully", func(t *testing.T) {
		robot := createRunnerTestRobot(t)
		config := standard.DefaultRunConfig()
		runner := standard.NewRunner(ctx, robot, config, "", "test")

		task := &types.Task{
			ID:           "task-003",
			ExecutorType: types.ExecutorAssistant,
			ExecutorID:   "experts.text-writer",
			Messages:     []agentcontext.Message{},
			Status:       types.TaskPending,
		}

		taskCtx := &standard.RunnerContext{
			SystemPrompt: robot.SystemPrompt,
		}

		result := runner.ExecuteTask(task, taskCtx)

		assert.False(t, result.Success, "task should fail with empty messages")
		assert.NotEmpty(t, result.Error, "error should describe the failure")
		t.Logf("Error: %s", result.Error)
	})
}

func TestRunnerBuildTaskContext(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.PrepareAgent(t)
	testutils.RequireE2EKeys(t)
	defer testutils.Clean(t)

	ctx := types.NewContext(context.Background(), testAuth())

	t.Run("includes previous results in context", func(t *testing.T) {
		robot := createRunnerTestRobot(t)
		config := standard.DefaultRunConfig()
		runner := standard.NewRunner(ctx, robot, config, "", "test")

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
		runner := standard.NewRunner(ctx, robot, config, "", "test")

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
		runner := standard.NewRunner(ctx, robot, config, "", "test")

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

	testutils.PrepareAgent(t)
	testutils.RequireE2EKeys(t)
	defer testutils.Clean(t)

	ctx := types.NewContext(context.Background(), testAuth())

	t.Run("formats previous results as markdown", func(t *testing.T) {
		robot := createRunnerTestRobot(t)
		config := standard.DefaultRunConfig()
		runner := standard.NewRunner(ctx, robot, config, "", "test")

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
		runner := standard.NewRunner(ctx, robot, config, "", "test")

		formatted := runner.FormatPreviousResultsAsContext([]types.TaskResult{})

		assert.Empty(t, formatted)
	})
}

func TestRunnerBuildAssistantMessages(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.PrepareAgent(t)
	testutils.RequireE2EKeys(t)
	defer testutils.Clean(t)

	ctx := types.NewContext(context.Background(), testAuth())

	t.Run("builds messages with task content", func(t *testing.T) {
		robot := createRunnerTestRobot(t)
		config := standard.DefaultRunConfig()
		runner := standard.NewRunner(ctx, robot, config, "", "test")

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
		runner := standard.NewRunner(ctx, robot, config, "", "test")

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

	testutils.PrepareAgent(t)
	testutils.RequireE2EKeys(t)
	defer testutils.Clean(t)

	ctx := types.NewContext(context.Background(), testAuth())

	t.Run("formats string content", func(t *testing.T) {
		robot := createRunnerTestRobot(t)
		config := standard.DefaultRunConfig()
		runner := standard.NewRunner(ctx, robot, config, "", "test")

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
		runner := standard.NewRunner(ctx, robot, config, "", "test")

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
		runner := standard.NewRunner(ctx, robot, config, "", "test")

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

	testutils.PrepareAgent(t)
	testutils.RequireE2EKeys(t)
	defer testutils.Clean(t)

	t.Run("executes unsupported type returns error", func(t *testing.T) {
		ctx := types.NewContext(context.Background(), testAuth())
		robot := createRunnerTestRobot(t)
		config := standard.DefaultRunConfig()
		runner := standard.NewRunner(ctx, robot, config, "", "test")

		task := &types.Task{
			ID:           "task-unknown",
			ExecutorType: "unsupported",
			ExecutorID:   "anything",
			Status:       types.TaskPending,
		}

		taskCtx := &standard.RunnerContext{}
		result := runner.ExecuteTask(task, taskCtx)

		assert.False(t, result.Success, "unsupported executor type should fail")
		assert.Contains(t, result.Error, "unsupported executor type")
		assert.Nil(t, result.Validation, "V2 does not set Validation in runner")
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
					types.PhaseRun: "robot.run",
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
