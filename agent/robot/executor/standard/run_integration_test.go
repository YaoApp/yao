//go:build integration

package standard_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	agentcontext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/robot/executor/standard"
	robottypes "github.com/yaoapp/yao/agent/robot/types"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

// ============================================================================
// P3 Run Phase Tests - RunExecution
// ============================================================================

func TestRunExecutionBasic(t *testing.T) {
	identity := testprepare.PrepareSandbox(t)
	ctx := testCtx(identity)

	t.Run("executes_single_task_successfully", func(t *testing.T) {
		robot := newTestRobot(t, identity)
		exec := createRunExecution(robot)
		exec.Tasks = []robottypes.Task{
			{
				ID: "task-001", ExecutorType: robottypes.ExecutorAssistant,
				ExecutorID: "experts.text-writer",
				Messages: []agentcontext.Message{
					{Role: agentcontext.RoleUser, Content: "Write a short greeting message. Keep it under 50 words."},
				},
				ExpectedOutput: "A friendly greeting message",
				Order:          0, Status: robottypes.TaskPending,
			},
		}

		e := standard.New()
		err := e.RunExecution(ctx, exec, nil)

		require.NoError(t, err)
		require.Len(t, exec.Results, 1)

		result := exec.Results[0]
		assert.Equal(t, "task-001", result.TaskID)
		assert.True(t, result.Success)
		assert.NotNil(t, result.Output)
		assert.Greater(t, result.Duration, int64(0))
		assert.Equal(t, robottypes.TaskCompleted, exec.Tasks[0].Status)
	})

	t.Run("executes_multiple_tasks_in_order", func(t *testing.T) {
		robot := newTestRobot(t, identity)
		exec := createRunExecution(robot)
		exec.Tasks = []robottypes.Task{
			{
				ID: "task-001", ExecutorType: robottypes.ExecutorAssistant,
				ExecutorID: "experts.data-analyst",
				Messages: []agentcontext.Message{
					{Role: agentcontext.RoleUser, Content: "Analyze this data: Q1: $100K, Q2: $150K, Q3: $120K, Q4: $180K. Calculate total and average."},
				},
				Order: 0, Status: robottypes.TaskPending,
			},
			{
				ID: "task-002", ExecutorType: robottypes.ExecutorAssistant,
				ExecutorID: "experts.text-writer",
				Messages: []agentcontext.Message{
					{Role: agentcontext.RoleUser, Content: "Summarize the key findings from the previous analysis in 2-3 sentences."},
				},
				Order: 1, Status: robottypes.TaskPending,
			},
		}

		e := standard.New()
		err := e.RunExecution(ctx, exec, nil)

		require.NoError(t, err)
		require.Len(t, exec.Results, 2)
		assert.True(t, exec.Results[0].Success)
		assert.True(t, exec.Results[1].Success)
		assert.Equal(t, robottypes.TaskCompleted, exec.Tasks[0].Status)
		assert.Equal(t, robottypes.TaskCompleted, exec.Tasks[1].Status)
	})
}

func TestRunExecutionContinueOnFailure(t *testing.T) {
	identity := testprepare.PrepareSandbox(t)
	ctx := testCtx(identity)

	t.Run("continues_on_failure_with_default_config", func(t *testing.T) {
		robot := newTestRobot(t, identity)
		exec := createRunExecution(robot)
		exec.Tasks = []robottypes.Task{
			{
				ID: "task-001", ExecutorType: robottypes.ExecutorAssistant,
				ExecutorID: "non.existent.assistant.xyz123",
				Messages:   []agentcontext.Message{{Role: agentcontext.RoleUser, Content: "This will fail"}},
				Order:      0, Status: robottypes.TaskPending,
			},
			{
				ID: "task-002", ExecutorType: robottypes.ExecutorAssistant,
				ExecutorID: "experts.text-writer",
				Messages:   []agentcontext.Message{{Role: agentcontext.RoleUser, Content: "Say hello"}},
				Order:      1, Status: robottypes.TaskPending,
			},
		}

		e := standard.New()
		err := e.RunExecution(ctx, exec, nil)

		require.NoError(t, err)
		assert.Equal(t, robottypes.TaskFailed, exec.Tasks[0].Status)
		assert.Equal(t, robottypes.TaskCompleted, exec.Tasks[1].Status)
		assert.Len(t, exec.Results, 2)
		assert.False(t, exec.Results[0].Success)
		assert.True(t, exec.Results[1].Success)
	})

	t.Run("stops_on_first_failure_when_disabled", func(t *testing.T) {
		robot := newTestRobot(t, identity)
		exec := createRunExecution(robot)
		exec.Tasks = []robottypes.Task{
			{
				ID: "task-001", ExecutorType: robottypes.ExecutorAssistant,
				ExecutorID: "non.existent.assistant.xyz123",
				Messages:   []agentcontext.Message{{Role: agentcontext.RoleUser, Content: "This will fail"}},
				Order:      0, Status: robottypes.TaskPending,
			},
			{
				ID: "task-002", ExecutorType: robottypes.ExecutorAssistant,
				ExecutorID: "experts.text-writer",
				Messages:   []agentcontext.Message{{Role: agentcontext.RoleUser, Content: "Write greeting"}},
				Order:      1, Status: robottypes.TaskPending,
			},
		}

		config := &standard.RunConfig{ContinueOnFailure: false}
		e := standard.New()
		err := e.RunExecution(ctx, exec, config)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "task-001")
		assert.Equal(t, robottypes.TaskFailed, exec.Tasks[0].Status)
		assert.Equal(t, robottypes.TaskSkipped, exec.Tasks[1].Status)
	})
}

func TestRunExecutionErrorHandling(t *testing.T) {
	identity := testprepare.PrepareSandbox(t)
	ctx := testCtx(identity)

	t.Run("returns_error_when_robot_is_nil", func(t *testing.T) {
		exec := &robottypes.Execution{
			ID: "test-exec-run-norobot", TriggerType: robottypes.TriggerClock,
			Tasks: []robottypes.Task{{ID: "task-001", ExecutorID: "test"}},
		}

		e := standard.New()
		err := e.RunExecution(ctx, exec, nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "robot not found")
	})

	t.Run("returns_error_when_no_tasks", func(t *testing.T) {
		robot := newTestRobot(t, identity)
		exec := createRunExecution(robot)
		exec.Tasks = []robottypes.Task{}

		e := standard.New()
		err := e.RunExecution(ctx, exec, nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no tasks")
	})

	t.Run("records_failure_for_non_existent_assistant", func(t *testing.T) {
		robot := newTestRobot(t, identity)
		exec := createRunExecution(robot)
		exec.Tasks = []robottypes.Task{
			{
				ID: "task-001", ExecutorType: robottypes.ExecutorAssistant,
				ExecutorID: "non.existent.agent",
				Messages:   []agentcontext.Message{{Role: agentcontext.RoleUser, Content: "Test"}},
				Order:      0, Status: robottypes.TaskPending,
			},
		}

		e := standard.New()
		err := e.RunExecution(ctx, exec, nil)

		assert.NoError(t, err)
		assert.Equal(t, robottypes.TaskFailed, exec.Tasks[0].Status)
		assert.Len(t, exec.Results, 1)
		assert.False(t, exec.Results[0].Success)
	})
}

func TestRunExecutionNoValidation(t *testing.T) {
	identity := testprepare.PrepareSandbox(t)
	ctx := testCtx(identity)

	t.Run("V2_runner_does_not_set_Validation_on_results", func(t *testing.T) {
		robot := newTestRobot(t, identity)
		exec := createRunExecution(robot)
		exec.Tasks = []robottypes.Task{
			{
				ID: "task-001", ExecutorType: robottypes.ExecutorAssistant,
				ExecutorID: "experts.data-analyst",
				Messages: []agentcontext.Message{
					{Role: agentcontext.RoleUser, Content: "Return a JSON object with fields: name (string), count (number)."},
				},
				ExpectedOutput: "JSON object with name and count fields",
				Order:          0, Status: robottypes.TaskPending,
			},
		}

		e := standard.New()
		err := e.RunExecution(ctx, exec, nil)

		require.NoError(t, err)
		require.Len(t, exec.Results, 1)

		assert.True(t, exec.Results[0].Success)
		assert.NotNil(t, exec.Results[0].Output)
		assert.Nil(t, exec.Results[0].Validation, "V2 runner does not run validation")
	})
}

// ============================================================================
// ResumeContext Tests
// ============================================================================

func TestRunExecutionResumeContext(t *testing.T) {
	identity := testprepare.PrepareSandbox(t)
	ctx := testCtx(identity)

	t.Run("resumes_from_task_index_with_previous_results", func(t *testing.T) {
		robot := newTestRobot(t, identity)
		exec := createRunExecution(robot)
		exec.Tasks = []robottypes.Task{
			{
				ID: "task-001", ExecutorType: robottypes.ExecutorAssistant,
				ExecutorID: "experts.text-writer",
				Messages:   []agentcontext.Message{{Role: agentcontext.RoleUser, Content: "Write 'hello'"}},
				Order:      0, Status: robottypes.TaskCompleted,
			},
			{
				ID: "task-002", ExecutorType: robottypes.ExecutorAssistant,
				ExecutorID: "experts.text-writer",
				Messages:   []agentcontext.Message{{Role: agentcontext.RoleUser, Content: "Write 'world'"}},
				Order:      1, Status: robottypes.TaskPending,
			},
		}

		exec.ResumeContext = &robottypes.ResumeContext{
			TaskIndex:       1,
			PreviousResults: []robottypes.TaskResult{{TaskID: "task-001", Success: true, Output: "hello", Duration: 100}},
		}

		e := standard.New()
		err := e.RunExecution(ctx, exec, nil)

		require.NoError(t, err)
		require.Len(t, exec.Results, 2)
		assert.Equal(t, "task-001", exec.Results[0].TaskID)
		assert.Equal(t, "task-002", exec.Results[1].TaskID)
		assert.True(t, exec.Results[1].Success)
		assert.Nil(t, exec.ResumeContext)
	})
}

// ============================================================================
// Helpers
// ============================================================================

func createRunExecution(robot *robottypes.Robot) *robottypes.Execution {
	exec := &robottypes.Execution{
		ID:          "test-exec-run-" + time.Now().Format("150405.000"),
		MemberID:    robot.MemberID,
		TeamID:      robot.TeamID,
		TriggerType: robottypes.TriggerClock,
		StartTime:   time.Now(),
		Status:      robottypes.ExecRunning,
		Phase:       robottypes.PhaseRun,
		Goals:       &robottypes.Goals{Content: "## Goals\n\n1. Execute test tasks"},
	}
	exec.SetRobot(robot)
	return exec
}
