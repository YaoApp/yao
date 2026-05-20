//go:build integration

package standard_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	agentcontext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/robot/executor/standard"
	robottypes "github.com/yaoapp/yao/agent/robot/types"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

// ============================================================================
// Runner Tests - V2 Simplified Execution
// ============================================================================

func TestRunnerExecuteTask(t *testing.T) {
	identity := testprepare.PrepareSandbox(t)
	ctx := testCtx(identity)

	t.Run("executes_assistant_task_successfully", func(t *testing.T) {
		robot := newTestRobot(t, identity)
		config := standard.DefaultRunConfig()
		runner := standard.NewRunner(ctx, robot, config, "", "test-runner")

		task := &robottypes.Task{
			ID:           "task-001",
			ExecutorType: robottypes.ExecutorAssistant,
			ExecutorID:   "experts.text-writer",
			Messages: []agentcontext.Message{
				{Role: agentcontext.RoleUser, Content: "Write a haiku about coding. Format: three lines with 5-7-5 syllables."},
			},
			ExpectedOutput: "A haiku poem about coding",
			Status:         robottypes.TaskPending,
		}

		taskCtx := &standard.RunnerContext{SystemPrompt: robot.SystemPrompt}
		result := runner.ExecuteTask(task, taskCtx)

		assert.True(t, result.Success, "task should succeed")
		assert.NotNil(t, result.Output)
		assert.Empty(t, result.Error)
		assert.Greater(t, result.Duration, int64(0))
	})

	t.Run("handles_empty_messages", func(t *testing.T) {
		robot := newTestRobot(t, identity)
		config := standard.DefaultRunConfig()
		runner := standard.NewRunner(ctx, robot, config, "", "test-runner")

		task := &robottypes.Task{
			ID:           "task-empty-msgs",
			ExecutorType: robottypes.ExecutorAssistant,
			ExecutorID:   "experts.text-writer",
			Messages:     []agentcontext.Message{},
			Status:       robottypes.TaskPending,
		}

		taskCtx := &standard.RunnerContext{SystemPrompt: robot.SystemPrompt}
		result := runner.ExecuteTask(task, taskCtx)

		assert.False(t, result.Success)
		assert.NotEmpty(t, result.Error)
	})

	t.Run("unsupported_executor_type_fails", func(t *testing.T) {
		robot := newTestRobot(t, identity)
		config := standard.DefaultRunConfig()
		runner := standard.NewRunner(ctx, robot, config, "", "test-runner")

		task := &robottypes.Task{
			ID:           "task-unknown-type",
			ExecutorType: "unsupported",
			ExecutorID:   "anything",
			Status:       robottypes.TaskPending,
		}

		taskCtx := &standard.RunnerContext{}
		result := runner.ExecuteTask(task, taskCtx)

		assert.False(t, result.Success)
		assert.Contains(t, result.Error, "unsupported executor type")
	})
}

func TestRunnerBuildTaskContext(t *testing.T) {
	identity := testprepare.PrepareSandbox(t)
	ctx := testCtx(identity)

	t.Run("includes_previous_results_in_context", func(t *testing.T) {
		robot := newTestRobot(t, identity)
		config := standard.DefaultRunConfig()
		runner := standard.NewRunner(ctx, robot, config, "", "test-runner")

		exec := &robottypes.Execution{
			ID: "test-exec-ctx", MemberID: robot.MemberID, TeamID: robot.TeamID,
			Goals: &robottypes.Goals{Content: "Test goals"},
			Results: []robottypes.TaskResult{
				{TaskID: "task-001", Success: true, Output: map[string]interface{}{"data": "previous result"}},
				{TaskID: "task-002", Success: true, Output: "Another result"},
			},
		}
		exec.SetRobot(robot)

		taskCtx := runner.BuildTaskContext(exec, 2)

		require.NotNil(t, taskCtx)
		assert.Len(t, taskCtx.PreviousResults, 2)
		assert.Equal(t, "task-001", taskCtx.PreviousResults[0].TaskID)
		assert.Equal(t, "task-002", taskCtx.PreviousResults[1].TaskID)
	})

	t.Run("handles_first_task_with_no_previous_results", func(t *testing.T) {
		robot := newTestRobot(t, identity)
		config := standard.DefaultRunConfig()
		runner := standard.NewRunner(ctx, robot, config, "", "test-runner")

		exec := &robottypes.Execution{
			ID: "test-exec-ctx-0", MemberID: robot.MemberID, TeamID: robot.TeamID,
			Goals:   &robottypes.Goals{Content: "Test goals"},
			Results: []robottypes.TaskResult{},
		}
		exec.SetRobot(robot)

		taskCtx := runner.BuildTaskContext(exec, 0)

		require.NotNil(t, taskCtx)
		assert.Empty(t, taskCtx.PreviousResults)
	})
}

func TestRunnerFormatPreviousResultsAsContext(t *testing.T) {
	identity := testprepare.PrepareSandbox(t)
	ctx := testCtx(identity)

	t.Run("formats_previous_results_as_markdown", func(t *testing.T) {
		robot := newTestRobot(t, identity)
		config := standard.DefaultRunConfig()
		runner := standard.NewRunner(ctx, robot, config, "", "test-runner")

		results := []robottypes.TaskResult{
			{TaskID: "task-001", Success: true, Output: map[string]interface{}{"key": "value", "count": 42}},
			{TaskID: "task-002", Success: false, Output: "Partial result", Error: "Validation failed"},
		}

		formatted := runner.FormatPreviousResultsAsContext(results)

		assert.Contains(t, formatted, "## Previous Task Results")
		assert.Contains(t, formatted, "task-001")
		assert.Contains(t, formatted, "task-002")
		assert.Contains(t, formatted, "Success")
		assert.Contains(t, formatted, "Failed")
	})

	t.Run("returns_empty_for_no_results", func(t *testing.T) {
		robot := newTestRobot(t, identity)
		config := standard.DefaultRunConfig()
		runner := standard.NewRunner(ctx, robot, config, "", "test-runner")

		assert.Empty(t, runner.FormatPreviousResultsAsContext([]robottypes.TaskResult{}))
	})
}

func TestRunnerBuildAssistantMessages(t *testing.T) {
	identity := testprepare.PrepareSandbox(t)
	ctx := testCtx(identity)

	t.Run("builds_messages_with_task_content", func(t *testing.T) {
		robot := newTestRobot(t, identity)
		config := standard.DefaultRunConfig()
		runner := standard.NewRunner(ctx, robot, config, "", "test-runner")

		task := &robottypes.Task{
			ID:           "task-001",
			ExecutorType: robottypes.ExecutorAssistant,
			ExecutorID:   "experts.text-writer",
			Messages: []agentcontext.Message{
				{Role: agentcontext.RoleUser, Content: "Write a greeting"},
			},
		}

		taskCtx := &standard.RunnerContext{SystemPrompt: "You are helpful"}
		messages := runner.BuildAssistantMessages(task, taskCtx)

		require.NotEmpty(t, messages)
		found := false
		for _, msg := range messages {
			if content, ok := msg.Content.(string); ok && content == "Write a greeting" {
				found = true
				break
			}
		}
		assert.True(t, found, "should contain task message")
	})
}

func TestRunnerFormatMessagesAsText(t *testing.T) {
	identity := testprepare.PrepareSandbox(t)
	ctx := testCtx(identity)

	t.Run("formats_string_content", func(t *testing.T) {
		robot := newTestRobot(t, identity)
		config := standard.DefaultRunConfig()
		runner := standard.NewRunner(ctx, robot, config, "", "test-runner")

		messages := []agentcontext.Message{
			{Role: agentcontext.RoleUser, Content: "Hello"},
			{Role: agentcontext.RoleUser, Content: "World"},
		}

		text := runner.FormatMessagesAsText(messages)
		assert.Contains(t, text, "Hello")
		assert.Contains(t, text, "World")
	})

	t.Run("handles_multipart_content", func(t *testing.T) {
		robot := newTestRobot(t, identity)
		config := standard.DefaultRunConfig()
		runner := standard.NewRunner(ctx, robot, config, "", "test-runner")

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
}
