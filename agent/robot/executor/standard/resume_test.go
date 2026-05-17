//go:build e2e

package standard_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	agentcontext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/robot/executor/standard"
	executortypes "github.com/yaoapp/yao/agent/robot/executor/types"
	"github.com/yaoapp/yao/agent/robot/store"
	"github.com/yaoapp/yao/agent/robot/types"
	"github.com/yaoapp/yao/agent/testutils"
)

// ============================================================================
// Resume method tests (R1-R10)
// ============================================================================

func TestResume(t *testing.T) {
	// R1: Resume with empty execID returns error
	t.Run("R1: Resume with empty execID returns error", func(t *testing.T) {
		e := standard.New()
		ctx := types.NewContext(context.Background(), testAuth())

		err := e.Resume(ctx, "", "some reply")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "empty")
	})

	// R2: Resume with non-existent execID returns error (requires DB)
	t.Run("R2: Resume with non-existent execID returns error", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Requires database")
		}

		testutils.PrepareAgent(t)
	testutils.RequireE2EKeys(t)
		defer testutils.Clean(t)

		e := standard.New()
		ctx := types.NewContext(context.Background(), testAuth())

		err := e.Resume(ctx, "non-existent-exec-id-12345", "reply")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "execution not found")
	})

	// R3: Resume with execution not in waiting status returns error (requires DB)
	t.Run("R3: Resume with execution not in waiting status returns error", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Requires database")
		}

		testutils.PrepareAgent(t)
	testutils.RequireE2EKeys(t)
		defer testutils.Clean(t)

		ctx := types.NewContext(context.Background(), testAuth())
		robot := createResumeTestRobot(t)
		exec := createResumeTestExecution(robot)
		exec.Status = types.ExecRunning // Not waiting

		record := store.FromExecution(exec)
		require.NoError(t, store.NewExecutionStore().Save(ctx.Context, record))

		// Save robot to robot store for Resume to load
		robotRecord := store.FromRobot(robot)
		require.NoError(t, store.NewRobotStore().Save(ctx.Context, robotRecord))

		e := standard.New()
		err := e.Resume(ctx, exec.ID, "reply")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "not in waiting status")
	})

	// R4: Verify Resume loads execution from store (requires DB)
	t.Run("R4: Resume loads execution from store", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Requires database")
		}

		testutils.PrepareAgent(t)
	testutils.RequireE2EKeys(t)
		defer testutils.Clean(t)

		ctx := types.NewContext(context.Background(), testAuth())
		robot := createResumeTestRobot(t)
		exec := createSuspendedResumeTestExecution(robot)

		execStore := store.NewExecutionStore()
		robotStore := store.NewRobotStore()

		record := store.FromExecution(exec)
		require.NoError(t, execStore.Save(ctx.Context, record))

		robotRecord := store.FromRobot(robot)
		require.NoError(t, robotStore.Save(ctx.Context, robotRecord))

		e := standard.New()
		err := e.Resume(ctx, exec.ID, "User provided answer")

		require.NoError(t, err)

		// Verify execution was loaded and completed
		loaded, err := execStore.Get(ctx.Context, exec.ID)
		require.NoError(t, err)
		require.NotNil(t, loaded)
		assert.Equal(t, types.ExecCompleted, loaded.Status)
	})

	// R5: Resume restores robot from execution record (requires DB)
	t.Run("R5: Resume restores robot from execution record", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Requires database")
		}

		testutils.PrepareAgent(t)
	testutils.RequireE2EKeys(t)
		defer testutils.Clean(t)

		ctx := types.NewContext(context.Background(), testAuth())
		robot := createResumeTestRobot(t)
		exec := createSuspendedResumeTestExecution(robot)

		execStore := store.NewExecutionStore()
		robotStore := store.NewRobotStore()

		record := store.FromExecution(exec)
		require.NoError(t, execStore.Save(ctx.Context, record))

		robotRecord := store.FromRobot(robot)
		require.NoError(t, robotStore.Save(ctx.Context, robotRecord))

		e := standard.New()
		err := e.Resume(ctx, exec.ID, "Answer for the question")

		require.NoError(t, err)
		// If we get here without "robot not found", Resume successfully restored robot
	})

	// R6: Resume with __skip__ reply marks task as skipped (requires DB)
	t.Run("R6: Resume with __skip__ reply marks task as skipped", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Requires database")
		}

		testutils.PrepareAgent(t)
	testutils.RequireE2EKeys(t)
		defer testutils.Clean(t)

		ctx := types.NewContext(context.Background(), testAuth())
		robot := createResumeTestRobot(t)
		exec := createSuspendedResumeTestExecution(robot)
		// Ensure we have a task at index 0 that is waiting
		exec.Tasks[0].Status = types.TaskWaitingInput
		exec.ResumeContext = &types.ResumeContext{
			TaskIndex:       0,
			PreviousResults: []types.TaskResult{},
		}

		execStore := store.NewExecutionStore()
		robotStore := store.NewRobotStore()

		record := store.FromExecution(exec)
		require.NoError(t, execStore.Save(ctx.Context, record))

		robotRecord := store.FromRobot(robot)
		require.NoError(t, robotStore.Save(ctx.Context, robotRecord))

		e := standard.New()
		err := e.Resume(ctx, exec.ID, "__skip__")

		require.NoError(t, err)

		loaded, err := execStore.Get(ctx.Context, exec.ID)
		require.NoError(t, err)
		require.NotNil(t, loaded)
		require.Len(t, loaded.Tasks, 1)
		assert.Equal(t, types.TaskSkipped, loaded.Tasks[0].Status)
	})

	// R7: Resume sends ErrExecutionSuspended when execution suspends again (requires DB)
	t.Run("R7: Resume sends ErrExecutionSuspended when execution suspends again", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Requires database")
		}

		testutils.PrepareAgent(t)
	testutils.RequireE2EKeys(t)
		defer testutils.Clean(t)

		// Use robot-need-input assistant that suspends
		ctx := types.NewContext(context.Background(), testAuth())
		robot := createResumeNeedInputRobot(t)
		exec := createSuspendedResumeNeedInputExecution(robot)

		execStore := store.NewExecutionStore()
		robotStore := store.NewRobotStore()

		record := store.FromExecution(exec)
		require.NoError(t, execStore.Save(ctx.Context, record))

		robotRecord := store.FromRobot(robot)
		require.NoError(t, robotStore.Save(ctx.Context, robotRecord))

		e := standard.New()
		err := e.Resume(ctx, exec.ID, "some reply")

		// May return ErrExecutionSuspended if assistant suspends again
		if err != nil {
			assert.ErrorIs(t, err, types.ErrExecutionSuspended)
		}
	})

	// R8: Resume increments exec counter
	t.Run("R8: Resume increments exec counter", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Requires database")
		}

		testutils.PrepareAgent(t)
	testutils.RequireE2EKeys(t)
		defer testutils.Clean(t)

		ctx := types.NewContext(context.Background(), testAuth())
		robot := createResumeTestRobot(t)
		exec := createSuspendedResumeTestExecution(robot)

		execStore := store.NewExecutionStore()
		robotStore := store.NewRobotStore()

		record := store.FromExecution(exec)
		require.NoError(t, execStore.Save(ctx.Context, record))

		robotRecord := store.FromRobot(robot)
		require.NoError(t, robotStore.Save(ctx.Context, robotRecord))

		e := standard.New()
		e.Reset()

		before := e.CurrentCount()
		err := e.Resume(ctx, exec.ID, "answer")
		after := e.CurrentCount()

		require.NoError(t, err)
		// During Resume, currentCount was incremented; after completion it's decremented
		assert.Equal(t, before, after, "currentCount should be back to original after Resume completes")
	})

	// R9: Resume decrements exec counter on completion
	t.Run("R9: Resume decrements exec counter on completion", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Requires database")
		}

		testutils.PrepareAgent(t)
	testutils.RequireE2EKeys(t)
		defer testutils.Clean(t)

		ctx := types.NewContext(context.Background(), testAuth())
		robot := createResumeTestRobot(t)
		exec := createSuspendedResumeTestExecution(robot)

		execStore := store.NewExecutionStore()
		robotStore := store.NewRobotStore()

		record := store.FromExecution(exec)
		require.NoError(t, execStore.Save(ctx.Context, record))

		robotRecord := store.FromRobot(robot)
		require.NoError(t, robotStore.Save(ctx.Context, robotRecord))

		e := standard.New()
		e.Reset()

		err := e.Resume(ctx, exec.ID, "reply")
		require.NoError(t, err)

		// After Resume completes, currentCount should be 0 (no leak)
		assert.Equal(t, 0, e.CurrentCount())
	})

	// R10: Resume with nil context returns error
	t.Run("R10: Resume with nil context returns error", func(t *testing.T) {
		e := standard.NewWithConfig(executortypes.Config{SkipPersistence: true})

		err := e.Resume(nil, "some-exec-id", "reply")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "context")
	})
}

// ============================================================================
// Helpers for Resume tests
// ============================================================================

func createResumeTestRobot(t *testing.T) *types.Robot {
	t.Helper()
	return &types.Robot{
		MemberID:     "test-robot-resume",
		TeamID:       "test-team-1",
		DisplayName:  "Resume Test Robot",
		SystemPrompt: "You are a helpful assistant.",
		Config: &types.Config{
			Identity: &types.Identity{
				Role:   "Test",
				Duties: []string{"Execute tasks"},
			},
			Resources: &types.Resources{
				Phases: map[types.Phase]string{
					types.PhaseDelivery: "robot.delivery",
					types.PhaseLearning: "robot.learning",
				},
				Agents: []string{"experts.text-writer"},
			},
			Quota: &types.Quota{Max: 5},
		},
	}
}

func createResumeTestExecution(robot *types.Robot) *types.Execution {
	exec := &types.Execution{
		ID:          "test-exec-resume-1",
		MemberID:    robot.MemberID,
		TeamID:      robot.TeamID,
		TriggerType: types.TriggerClock,
		StartTime:   time.Now(),
		Status:      types.ExecRunning,
		Phase:       types.PhaseRun,
		Goals:       &types.Goals{Content: "## Goals\n\n1. Test resume"},
		Tasks: []types.Task{
			{
				ID:           "task-001",
				ExecutorType: types.ExecutorAssistant,
				ExecutorID:   "experts.text-writer",
				Messages: []agentcontext.Message{
					{Role: agentcontext.RoleUser, Content: "Write 'hello'"},
				},
				Order:  0,
				Status: types.TaskPending,
			},
		},
		ChatID: "robot_test-robot-resume_test-exec-resume-1",
	}
	exec.SetRobot(robot)
	return exec
}

func createSuspendedResumeTestExecution(robot *types.Robot) *types.Execution {
	exec := createResumeTestExecution(robot)
	exec.Status = types.ExecWaiting
	exec.WaitingTaskID = "task-001"
	exec.WaitingQuestion = "What should we do?"
	now := time.Now()
	exec.WaitingSince = &now
	exec.ResumeContext = &types.ResumeContext{
		TaskIndex:       0,
		PreviousResults: []types.TaskResult{},
	}
	exec.Tasks[0].Status = types.TaskWaitingInput
	return exec
}

func createResumeNeedInputRobot(t *testing.T) *types.Robot {
	t.Helper()
	return &types.Robot{
		MemberID:     "test-robot-resume-need-input",
		TeamID:       "test-team-1",
		DisplayName:  "Resume Need Input Robot",
		SystemPrompt: "You are a helpful assistant.",
		Config: &types.Config{
			Identity: &types.Identity{
				Role:   "Test",
				Duties: []string{"Execute tasks"},
			},
			Resources: &types.Resources{
				Phases: map[types.Phase]string{
					types.PhaseDelivery: "robot.delivery",
					types.PhaseLearning: "robot.learning",
				},
				Agents: []string{"tests.robot-need-input"},
			},
			Quota: &types.Quota{Max: 5},
		},
	}
}

func createSuspendedResumeNeedInputExecution(robot *types.Robot) *types.Execution {
	exec := &types.Execution{
		ID:          "test-exec-resume-need-input-1",
		MemberID:    robot.MemberID,
		TeamID:      robot.TeamID,
		TriggerType: types.TriggerClock,
		StartTime:   time.Now(),
		Status:      types.ExecWaiting,
		Phase:       types.PhaseRun,
		Goals:       &types.Goals{Content: "## Goals\n\n1. Test need input"},
		Tasks: []types.Task{
			{
				ID:           "task-001",
				ExecutorType: types.ExecutorAssistant,
				ExecutorID:   "tests.robot-need-input",
				Messages: []agentcontext.Message{
					{Role: agentcontext.RoleUser, Content: "Need input test"},
				},
				Order:  0,
				Status: types.TaskWaitingInput,
			},
		},
		ChatID:          "robot_test-robot-resume-need-input_test-exec-resume-need-input-1",
		WaitingTaskID:   "task-001",
		WaitingQuestion: "What period?",
		ResumeContext: &types.ResumeContext{
			TaskIndex:       0,
			PreviousResults: []types.TaskResult{},
		},
	}
	now := time.Now()
	exec.WaitingSince = &now
	exec.SetRobot(robot)
	return exec
}
