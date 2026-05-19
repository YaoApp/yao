//go:build integration

package standard_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	agentcontext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/robot/executor/standard"
	"github.com/yaoapp/yao/agent/robot/executor/types"
	"github.com/yaoapp/yao/agent/robot/store"
	robottypes "github.com/yaoapp/yao/agent/robot/types"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

// ============================================================================
// Suspend Tests
// ============================================================================

func TestSuspendExecution(t *testing.T) {
	_ = testprepare.PrepareSandbox(t)

	t.Run("suspend_sets_waiting_fields_and_returns_ErrExecutionSuspended", func(t *testing.T) {
		robot := &robottypes.Robot{
			MemberID:    "test-robot-suspend",
			TeamID:      "test-team-1",
			DisplayName: "Suspend Test Robot",
		}

		exec := &robottypes.Execution{
			ID: "exec-suspend-001", MemberID: robot.MemberID, TeamID: robot.TeamID,
			Status: robottypes.ExecRunning, Phase: robottypes.PhaseRun,
			Tasks: []robottypes.Task{
				{ID: "task-001", Status: robottypes.TaskRunning},
				{ID: "task-002", Status: robottypes.TaskPending},
			},
			Results: []robottypes.TaskResult{},
		}
		exec.SetRobot(robot)

		e := standard.NewWithConfig(types.Config{SkipPersistence: true})
		err := e.Suspend(robottypes.NewContext(context.Background(), nil), exec, 0, "What time range?")

		assert.ErrorIs(t, err, robottypes.ErrExecutionSuspended)
		assert.Equal(t, robottypes.ExecWaiting, exec.Status)
		assert.Equal(t, "task-001", exec.WaitingTaskID)
		assert.Equal(t, "What time range?", exec.WaitingQuestion)
		assert.NotNil(t, exec.WaitingSince)
		assert.NotNil(t, exec.ResumeContext)
		assert.Equal(t, 0, exec.ResumeContext.TaskIndex)
		assert.Equal(t, robottypes.TaskWaitingInput, exec.Tasks[0].Status)
	})

	t.Run("suspend_with_out_of_range_taskIndex_is_safe", func(t *testing.T) {
		robot := &robottypes.Robot{MemberID: "test-robot-suspend-2", TeamID: "test-team-1"}
		exec := &robottypes.Execution{
			ID: "exec-suspend-002", MemberID: robot.MemberID, TeamID: robot.TeamID,
			Status: robottypes.ExecRunning, Tasks: []robottypes.Task{}, Results: []robottypes.TaskResult{},
		}
		exec.SetRobot(robot)

		e := standard.NewWithConfig(types.Config{SkipPersistence: true})
		err := e.Suspend(robottypes.NewContext(context.Background(), nil), exec, 5, "some question")

		assert.ErrorIs(t, err, robottypes.ErrExecutionSuspended)
		assert.Equal(t, robottypes.ExecWaiting, exec.Status)
		assert.Empty(t, exec.WaitingTaskID)
	})
}

// ============================================================================
// ResumeContext data structure tests
// ============================================================================

func TestResumeContext(t *testing.T) {
	_ = testprepare.PrepareSandbox(t)

	t.Run("stores_task_index_and_previous_results", func(t *testing.T) {
		rc := &robottypes.ResumeContext{
			TaskIndex: 2,
			PreviousResults: []robottypes.TaskResult{
				{TaskID: "t1", Success: true, Output: "out1"},
				{TaskID: "t2", Success: false, Error: "some error"},
			},
		}
		assert.Equal(t, 2, rc.TaskIndex)
		assert.Len(t, rc.PreviousResults, 2)
		assert.True(t, rc.PreviousResults[0].Success)
		assert.False(t, rc.PreviousResults[1].Success)
	})
}

// ============================================================================
// TaskResult NeedInput Tests
// ============================================================================

func TestTaskResultNeedInput(t *testing.T) {
	_ = testprepare.PrepareSandbox(t)

	t.Run("NeedInput_fields_are_populated_correctly", func(t *testing.T) {
		result := robottypes.TaskResult{
			TaskID: "task-001", Success: true, Output: "some output",
			NeedInput: true, InputQuestion: "What time range?",
		}
		assert.True(t, result.NeedInput)
		assert.Equal(t, "What time range?", result.InputQuestion)
	})
}

// ============================================================================
// Execution Status Transitions
// ============================================================================

func TestExecutionStatusTransitions(t *testing.T) {
	_ = testprepare.PrepareSandbox(t)

	t.Run("ExecWaiting_is_a_valid_status", func(t *testing.T) {
		exec := &robottypes.Execution{ID: "exec-001", Status: robottypes.ExecWaiting}
		assert.Equal(t, robottypes.ExecStatus("waiting"), exec.Status)
	})

	t.Run("TaskWaitingInput_is_a_valid_task_status", func(t *testing.T) {
		task := robottypes.Task{ID: "task-001", Status: robottypes.TaskWaitingInput}
		assert.Equal(t, robottypes.TaskStatus("waiting_input"), task.Status)
	})

	t.Run("Execution_V2_fields_are_accessible", func(t *testing.T) {
		now := time.Now()
		exec := &robottypes.Execution{
			ID: "exec-v2-001", ChatID: "robot_member1_exec001",
			WaitingTaskID: "task-002", WaitingQuestion: "What period?",
			WaitingSince: &now,
			ResumeContext: &robottypes.ResumeContext{
				TaskIndex:       1,
				PreviousResults: []robottypes.TaskResult{{TaskID: "task-001", Success: true}},
			},
		}
		assert.Equal(t, "robot_member1_exec001", exec.ChatID)
		assert.Equal(t, "task-002", exec.WaitingTaskID)
		assert.Equal(t, "What period?", exec.WaitingQuestion)
		assert.NotNil(t, exec.WaitingSince)
		assert.NotNil(t, exec.ResumeContext)
		assert.Equal(t, 1, exec.ResumeContext.TaskIndex)
	})
}

// ============================================================================
// Resume Tests
// ============================================================================

func TestResume(t *testing.T) {
	t.Run("R1_Resume_with_empty_execID_returns_error", func(t *testing.T) {
		_ = testprepare.PrepareSandbox(t)
		e := standard.New()
		ctx := robottypes.NewContext(context.Background(), nil)

		err := e.Resume(ctx, "", "some reply")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "empty")
	})

	t.Run("R2_Resume_with_non_existent_execID_returns_error", func(t *testing.T) {
		identity := testprepare.PrepareSandbox(t)
		e := standard.New()
		ctx := testCtx(identity)

		err := e.Resume(ctx, "non-existent-exec-id-12345", "reply")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "execution not found")
	})

	t.Run("R3_Resume_with_execution_not_in_waiting_status_returns_error", func(t *testing.T) {
		identity := testprepare.PrepareSandbox(t)
		ctx := testCtx(identity)
		robot := newResumeTestRobot(t, identity)
		exec := newResumeTestExecution(robot)
		exec.Status = robottypes.ExecRunning

		record := store.FromExecution(exec)
		require.NoError(t, store.NewExecutionStore().Save(ctx.Context, record))

		robotRecord := store.FromRobot(robot)
		require.NoError(t, store.NewRobotStore().Save(ctx.Context, robotRecord))

		e := standard.New()
		err := e.Resume(ctx, exec.ID, "reply")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "not in waiting status")
	})

	t.Run("R4_Resume_loads_execution_from_store", func(t *testing.T) {
		identity := testprepare.PrepareSandbox(t)
		ctx := testCtx(identity)
		robot := newResumeTestRobot(t, identity)
		exec := newSuspendedResumeExecution(robot)

		execStore := store.NewExecutionStore()
		robotStore := store.NewRobotStore()

		require.NoError(t, execStore.Save(ctx.Context, store.FromExecution(exec)))
		require.NoError(t, robotStore.Save(ctx.Context, store.FromRobot(robot)))

		e := standard.New()
		err := e.Resume(ctx, exec.ID, "User provided answer")

		require.NoError(t, err)

		loaded, err := execStore.Get(ctx.Context, exec.ID)
		require.NoError(t, err)
		require.NotNil(t, loaded)
		assert.Equal(t, robottypes.ExecCompleted, loaded.Status)
	})

	t.Run("R8_Resume_counter_returns_to_zero_after_completion", func(t *testing.T) {
		identity := testprepare.PrepareSandbox(t)
		ctx := testCtx(identity)
		robot := newResumeTestRobot(t, identity)
		exec := newSuspendedResumeExecution(robot)

		execStore := store.NewExecutionStore()
		robotStore := store.NewRobotStore()
		require.NoError(t, execStore.Save(ctx.Context, store.FromExecution(exec)))
		require.NoError(t, robotStore.Save(ctx.Context, store.FromRobot(robot)))

		e := standard.New()
		e.Reset()

		err := e.Resume(ctx, exec.ID, "answer")
		require.NoError(t, err)
		assert.Equal(t, 0, e.CurrentCount())
	})

	t.Run("R10_Resume_with_nil_context_returns_error", func(t *testing.T) {
		_ = testprepare.PrepareSandbox(t)
		e := standard.NewWithConfig(types.Config{SkipPersistence: true})

		err := e.Resume(nil, "some-exec-id", "reply")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "context")
	})
}

// ============================================================================
// Resume Helpers
// ============================================================================

func newResumeTestRobot(t *testing.T, identity *testprepare.TestIdentity) *robottypes.Robot {
	t.Helper()
	name := t.Name()
	if len(name) > 30 {
		name = name[len(name)-30:]
	}
	return &robottypes.Robot{
		MemberID:     "rr-" + name,
		TeamID:       identity.AlphaTeamID,
		DisplayName:  "Resume Test Robot",
		SystemPrompt: "You are a helpful assistant.",
		Config: &robottypes.Config{
			Identity: &robottypes.Identity{Role: "Test", Duties: []string{"Execute tasks"}},
			Resources: &robottypes.Resources{
				Phases: map[robottypes.Phase]string{
					robottypes.PhaseDelivery: "tests.robot-delivery",
					robottypes.PhaseLearning: "tests.robot-learning",
				},
				Agents: []string{"experts.text-writer"},
			},
			Quota: &robottypes.Quota{Max: 5},
		},
	}
}

func newResumeTestExecution(robot *robottypes.Robot) *robottypes.Execution {
	exec := &robottypes.Execution{
		ID:          "test-exec-resume-" + time.Now().Format("150405.000"),
		MemberID:    robot.MemberID,
		TeamID:      robot.TeamID,
		TriggerType: robottypes.TriggerClock,
		StartTime:   time.Now(),
		Status:      robottypes.ExecRunning,
		Phase:       robottypes.PhaseRun,
		Goals:       &robottypes.Goals{Content: "## Goals\n\n1. Test resume"},
		Tasks: []robottypes.Task{
			{
				ID: "task-001", ExecutorType: robottypes.ExecutorAssistant,
				ExecutorID: "experts.text-writer",
				Messages:   []agentcontext.Message{{Role: agentcontext.RoleUser, Content: "Write 'hello'"}},
				Order:      0, Status: robottypes.TaskPending,
			},
		},
		ChatID: "robot_" + robot.MemberID + "_test-exec-resume",
	}
	exec.SetRobot(robot)
	return exec
}

func newSuspendedResumeExecution(robot *robottypes.Robot) *robottypes.Execution {
	exec := newResumeTestExecution(robot)
	exec.Status = robottypes.ExecWaiting
	exec.WaitingTaskID = "task-001"
	exec.WaitingQuestion = "What should we do?"
	now := time.Now()
	exec.WaitingSince = &now
	exec.ResumeContext = &robottypes.ResumeContext{TaskIndex: 0, PreviousResults: []robottypes.TaskResult{}}
	exec.Tasks[0].Status = robottypes.TaskWaitingInput
	return exec
}
