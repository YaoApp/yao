package job_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yaoapp/yao/agent/robot/job"
	"github.com/yaoapp/yao/agent/robot/types"
	"github.com/yaoapp/yao/agent/testutils"
)

// TestCreateExecution tests creating a new execution
func TestCreateExecution(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	ctx := types.NewContext(context.Background(), nil)

	t.Run("create execution with clock trigger", func(t *testing.T) {
		robot := createTestRobot("test_exec_create_001")

		exec, err := job.CreateExecution(ctx, &job.CreateOptions{
			Robot:       robot,
			TriggerType: types.TriggerClock,
		})

		require.NoError(t, err)
		assert.NotNil(t, exec)
		assert.NotEmpty(t, exec.ID)
		assert.NotEmpty(t, exec.JobID)
		assert.Equal(t, robot.MemberID, exec.MemberID)
		assert.Equal(t, robot.TeamID, exec.TeamID)
		assert.Equal(t, types.TriggerClock, exec.TriggerType)
		assert.Equal(t, types.ExecPending, exec.Status)
		// Clock trigger starts from P0 (Inspiration)
		assert.Equal(t, types.PhaseInspiration, exec.Phase)
		assert.False(t, exec.StartTime.IsZero())
	})

	t.Run("create execution with human trigger starts from P1", func(t *testing.T) {
		robot := createTestRobot("test_exec_create_002")

		exec, err := job.CreateExecution(ctx, &job.CreateOptions{
			Robot:       robot,
			TriggerType: types.TriggerHuman,
		})

		require.NoError(t, err)
		assert.NotNil(t, exec)
		// Human trigger skips P0, starts from P1 (Goals)
		assert.Equal(t, types.PhaseGoals, exec.Phase)
	})

	t.Run("create execution with event trigger starts from P1", func(t *testing.T) {
		robot := createTestRobot("test_exec_create_003")

		exec, err := job.CreateExecution(ctx, &job.CreateOptions{
			Robot:       robot,
			TriggerType: types.TriggerEvent,
		})

		require.NoError(t, err)
		assert.NotNil(t, exec)
		// Event trigger skips P0, starts from P1 (Goals)
		assert.Equal(t, types.PhaseGoals, exec.Phase)
	})

	t.Run("create execution with input", func(t *testing.T) {
		robot := createTestRobot("test_exec_create_004")
		input := &types.TriggerInput{
			Action: types.ActionTaskAdd,
			UserID: "test_user_001",
		}

		exec, err := job.CreateExecution(ctx, &job.CreateOptions{
			Robot:       robot,
			TriggerType: types.TriggerHuman,
			Input:       input,
		})

		require.NoError(t, err)
		assert.NotNil(t, exec)
		assert.NotNil(t, exec.Input)
		assert.Equal(t, types.ActionTaskAdd, exec.Input.Action)
	})

	t.Run("create execution with optional fields", func(t *testing.T) {
		robot := createTestRobot("test_exec_create_005")
		timeout := 300
		scheduledAt := time.Now().Add(1 * time.Hour)

		exec, err := job.CreateExecution(ctx, &job.CreateOptions{
			Robot:             robot,
			TriggerType:       types.TriggerClock,
			Priority:          5,
			TimeoutSeconds:    &timeout,
			ParentExecutionID: "parent_exec_001",
			ScheduledAt:       &scheduledAt,
			Metadata: map[string]interface{}{
				"source": "test",
			},
		})

		require.NoError(t, err)
		assert.NotNil(t, exec)
	})

	t.Run("create execution with nil robot returns error", func(t *testing.T) {
		_, err := job.CreateExecution(ctx, &job.CreateOptions{
			Robot:       nil,
			TriggerType: types.TriggerClock,
		})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "robot is required")
	})

	t.Run("create execution with empty trigger type returns error", func(t *testing.T) {
		robot := createTestRobot("test_exec_create_006")

		_, err := job.CreateExecution(ctx, &job.CreateOptions{
			Robot:       robot,
			TriggerType: "",
		})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "trigger type is required")
	})

	t.Run("create execution with nil options returns error", func(t *testing.T) {
		_, err := job.CreateExecution(ctx, nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "options is nil")
	})
}

// TestUpdatePhase tests updating execution phase
func TestUpdatePhase(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	ctx := types.NewContext(context.Background(), nil)

	t.Run("update phase successfully", func(t *testing.T) {
		robot := createTestRobot("test_phase_001")
		exec, err := job.CreateExecution(ctx, &job.CreateOptions{
			Robot:       robot,
			TriggerType: types.TriggerClock,
		})
		require.NoError(t, err)

		// Update to Goals phase
		err = job.UpdatePhase(ctx, exec, types.PhaseGoals)
		require.NoError(t, err)
		assert.Equal(t, types.PhaseGoals, exec.Phase)

		// Verify job was updated
		j, err := job.Get(exec.JobID)
		require.NoError(t, err)
		assert.Equal(t, string(types.PhaseGoals), j.Config["current_phase"])
	})

	t.Run("update through all phases", func(t *testing.T) {
		robot := createTestRobot("test_phase_002")
		exec, err := job.CreateExecution(ctx, &job.CreateOptions{
			Robot:       robot,
			TriggerType: types.TriggerClock,
		})
		require.NoError(t, err)

		phases := []types.Phase{
			types.PhaseGoals,
			types.PhaseTasks,
			types.PhaseRun,
			types.PhaseDelivery,
			types.PhaseLearning,
		}

		for _, phase := range phases {
			err = job.UpdatePhase(ctx, exec, phase)
			require.NoError(t, err)
			assert.Equal(t, phase, exec.Phase)
		}
	})

	t.Run("update phase with nil execution returns error", func(t *testing.T) {
		err := job.UpdatePhase(ctx, nil, types.PhaseGoals)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid execution")
	})

	t.Run("update phase with empty execution ID returns error", func(t *testing.T) {
		exec := &types.Execution{
			ID:    "",
			JobID: "some_job_id",
		}
		err := job.UpdatePhase(ctx, exec, types.PhaseGoals)
		assert.Error(t, err)
	})

	t.Run("update phase with empty job ID returns error", func(t *testing.T) {
		exec := &types.Execution{
			ID:    "some_exec_id",
			JobID: "",
		}
		err := job.UpdatePhase(ctx, exec, types.PhaseGoals)
		assert.Error(t, err)
	})
}

// TestUpdateStatus tests updating execution status
func TestUpdateStatus(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	ctx := types.NewContext(context.Background(), nil)

	t.Run("update status to running", func(t *testing.T) {
		robot := createTestRobot("test_status_001")
		exec, err := job.CreateExecution(ctx, &job.CreateOptions{
			Robot:       robot,
			TriggerType: types.TriggerClock,
		})
		require.NoError(t, err)

		err = job.UpdateStatus(ctx, exec, types.ExecRunning)
		require.NoError(t, err)
		assert.Equal(t, types.ExecRunning, exec.Status)

		// Verify job was updated
		j, err := job.Get(exec.JobID)
		require.NoError(t, err)
		assert.Equal(t, "running", j.Status)
	})

	t.Run("update status with nil execution returns error", func(t *testing.T) {
		err := job.UpdateStatus(ctx, nil, types.ExecRunning)
		assert.Error(t, err)
	})
}

// TestCompleteExecution tests completing an execution
func TestCompleteExecution(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	ctx := types.NewContext(context.Background(), nil)

	t.Run("complete execution successfully", func(t *testing.T) {
		robot := createTestRobot("test_complete_001")
		exec, err := job.CreateExecution(ctx, &job.CreateOptions{
			Robot:       robot,
			TriggerType: types.TriggerClock,
		})
		require.NoError(t, err)

		// Simulate execution progress
		exec.Delivery = &types.DeliveryResult{
			RequestID: "test-delivery-001",
			Content: &types.DeliveryContent{
				Summary: "Test delivery completed",
				Body:    "# Test Delivery\n\nThis is a test delivery result.",
			},
			Success: true,
		}

		err = job.CompleteExecution(ctx, exec)
		require.NoError(t, err)

		assert.Equal(t, types.ExecCompleted, exec.Status)
		assert.NotNil(t, exec.EndTime)

		// Verify job was completed
		j, err := job.Get(exec.JobID)
		require.NoError(t, err)
		assert.Equal(t, "completed", j.Status)

		// Verify job execution was updated
		jobExec, err := job.GetExecution(exec.ID)
		require.NoError(t, err)
		assert.Equal(t, "completed", jobExec.Status)
		assert.Equal(t, 100, jobExec.Progress)
		assert.NotNil(t, jobExec.EndedAt)
	})

	t.Run("complete execution with nil execution returns error", func(t *testing.T) {
		err := job.CompleteExecution(ctx, nil)
		assert.Error(t, err)
	})
}

// TestFailExecution tests failing an execution
func TestFailExecution(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	ctx := types.NewContext(context.Background(), nil)

	t.Run("fail execution with error", func(t *testing.T) {
		robot := createTestRobot("test_fail_001")
		exec, err := job.CreateExecution(ctx, &job.CreateOptions{
			Robot:       robot,
			TriggerType: types.TriggerClock,
		})
		require.NoError(t, err)

		testErr := errors.New("task execution failed")
		err = job.FailExecution(ctx, exec, testErr)
		require.NoError(t, err)

		assert.Equal(t, types.ExecFailed, exec.Status)
		assert.NotNil(t, exec.EndTime)
		assert.Equal(t, testErr.Error(), exec.Error)

		// Verify job was failed
		j, err := job.Get(exec.JobID)
		require.NoError(t, err)
		assert.Equal(t, "failed", j.Status)
	})

	t.Run("fail execution without error", func(t *testing.T) {
		robot := createTestRobot("test_fail_002")
		exec, err := job.CreateExecution(ctx, &job.CreateOptions{
			Robot:       robot,
			TriggerType: types.TriggerClock,
		})
		require.NoError(t, err)

		err = job.FailExecution(ctx, exec, nil)
		require.NoError(t, err)

		assert.Equal(t, types.ExecFailed, exec.Status)
		assert.Empty(t, exec.Error)
	})
}

// TestCancelExecution tests cancelling an execution
func TestCancelExecution(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	ctx := types.NewContext(context.Background(), nil)

	t.Run("cancel execution successfully", func(t *testing.T) {
		robot := createTestRobot("test_cancel_001")
		exec, err := job.CreateExecution(ctx, &job.CreateOptions{
			Robot:       robot,
			TriggerType: types.TriggerClock,
		})
		require.NoError(t, err)

		err = job.CancelExecution(ctx, exec)
		require.NoError(t, err)

		assert.Equal(t, types.ExecCancelled, exec.Status)
		assert.NotNil(t, exec.EndTime)

		// Verify job was cancelled
		j, err := job.Get(exec.JobID)
		require.NoError(t, err)
		assert.Equal(t, "cancelled", j.Status)
	})
}

// TestGetExecution tests retrieving an execution
func TestGetExecution(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	ctx := types.NewContext(context.Background(), nil)

	t.Run("get existing execution", func(t *testing.T) {
		robot := createTestRobot("test_get_exec_001")
		exec, err := job.CreateExecution(ctx, &job.CreateOptions{
			Robot:       robot,
			TriggerType: types.TriggerClock,
		})
		require.NoError(t, err)

		jobExec, err := job.GetExecution(exec.ID)
		require.NoError(t, err)
		assert.NotNil(t, jobExec)
		assert.Equal(t, exec.ID, jobExec.ExecutionID)
		assert.Equal(t, exec.JobID, jobExec.JobID)
	})

	t.Run("get non-existent execution returns error", func(t *testing.T) {
		_, err := job.GetExecution("non_existent_exec_id")
		assert.Error(t, err)
	})

	t.Run("get with empty execution ID returns error", func(t *testing.T) {
		_, err := job.GetExecution("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "execution ID is empty")
	})
}

// TestListExecutions tests listing executions for a job
func TestListExecutions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	ctx := types.NewContext(context.Background(), nil)

	t.Run("list executions for job", func(t *testing.T) {
		robot := createTestRobot("test_list_exec_001")
		exec, err := job.CreateExecution(ctx, &job.CreateOptions{
			Robot:       robot,
			TriggerType: types.TriggerClock,
		})
		require.NoError(t, err)

		execs, err := job.ListExecutions(exec.JobID)
		require.NoError(t, err)
		assert.NotEmpty(t, execs)
		assert.Equal(t, 1, len(execs))
		assert.Equal(t, exec.ID, execs[0].ExecutionID)
	})

	t.Run("list with empty job ID returns error", func(t *testing.T) {
		_, err := job.ListExecutions("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "job ID is empty")
	})
}

// TestPhaseToProgress tests phase to progress mapping
func TestPhaseToProgress(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	ctx := types.NewContext(context.Background(), nil)

	testCases := []struct {
		phase            types.Phase
		expectedProgress int
	}{
		{types.PhaseInspiration, 10},
		{types.PhaseGoals, 25},
		{types.PhaseTasks, 40},
		{types.PhaseRun, 60},
		{types.PhaseDelivery, 80},
		{types.PhaseLearning, 95},
	}

	for _, tc := range testCases {
		t.Run(string(tc.phase), func(t *testing.T) {
			robot := createTestRobot("test_progress_" + string(tc.phase))
			exec, err := job.CreateExecution(ctx, &job.CreateOptions{
				Robot:       robot,
				TriggerType: types.TriggerClock,
			})
			require.NoError(t, err)

			err = job.UpdatePhase(ctx, exec, tc.phase)
			require.NoError(t, err)

			// Verify progress in job execution
			jobExec, err := job.GetExecution(exec.ID)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedProgress, jobExec.Progress)
		})
	}
}

// TestExecutionDuration tests execution duration calculation
func TestExecutionDuration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	ctx := types.NewContext(context.Background(), nil)

	t.Run("duration calculated on completion", func(t *testing.T) {
		robot := createTestRobot("test_duration_001")
		exec, err := job.CreateExecution(ctx, &job.CreateOptions{
			Robot:       robot,
			TriggerType: types.TriggerClock,
		})
		require.NoError(t, err)

		// Wait a bit to ensure measurable duration
		time.Sleep(50 * time.Millisecond)

		err = job.CompleteExecution(ctx, exec)
		require.NoError(t, err)

		// Verify duration was calculated
		jobExec, err := job.GetExecution(exec.ID)
		require.NoError(t, err)
		assert.NotNil(t, jobExec.Duration)
		assert.Greater(t, *jobExec.Duration, 0)
	})

	t.Run("duration calculated on failure", func(t *testing.T) {
		robot := createTestRobot("test_duration_002")
		exec, err := job.CreateExecution(ctx, &job.CreateOptions{
			Robot:       robot,
			TriggerType: types.TriggerClock,
		})
		require.NoError(t, err)

		time.Sleep(50 * time.Millisecond)

		err = job.FailExecution(ctx, exec, errors.New("test error"))
		require.NoError(t, err)

		jobExec, err := job.GetExecution(exec.ID)
		require.NoError(t, err)
		assert.NotNil(t, jobExec.Duration)
		assert.Greater(t, *jobExec.Duration, 0)
	})
}
