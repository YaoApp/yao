package store_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/yao/agent/robot/store"
	"github.com/yaoapp/yao/agent/robot/types"
	"github.com/yaoapp/yao/agent/testutils"
)

// TestExecutionStoreSave tests creating and updating execution records
func TestExecutionStoreSave(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Clean up any existing test data
	cleanupTestExecutions(t)
	defer cleanupTestExecutions(t)

	s := store.NewExecutionStore()
	ctx := context.Background()

	t.Run("creates_new_execution_record", func(t *testing.T) {
		startTime := time.Now()
		record := &store.ExecutionRecord{
			ExecutionID: "exec_test_save_001",
			MemberID:    "member_test_001",
			TeamID:      "team_test_001",
			JobID:       "job_test_001",
			TriggerType: types.TriggerClock,
			Status:      types.ExecPending,
			Phase:       types.PhaseInspiration,
			StartTime:   &startTime,
		}

		err := s.Save(ctx, record)
		require.NoError(t, err)

		// Verify it was created
		saved, err := s.Get(ctx, "exec_test_save_001")
		require.NoError(t, err)
		require.NotNil(t, saved)

		assert.Equal(t, "exec_test_save_001", saved.ExecutionID)
		assert.Equal(t, "member_test_001", saved.MemberID)
		assert.Equal(t, "team_test_001", saved.TeamID)
		assert.Equal(t, "job_test_001", saved.JobID)
		assert.Equal(t, types.TriggerClock, saved.TriggerType)
		assert.Equal(t, types.ExecPending, saved.Status)
		assert.Equal(t, types.PhaseInspiration, saved.Phase)
		assert.NotNil(t, saved.StartTime)
		assert.NotNil(t, saved.CreatedAt)
	})

	t.Run("updates_existing_execution_record", func(t *testing.T) {
		// First create a record
		startTime := time.Now()
		record := &store.ExecutionRecord{
			ExecutionID: "exec_test_save_002",
			MemberID:    "member_test_002",
			TeamID:      "team_test_002",
			TriggerType: types.TriggerHuman,
			Status:      types.ExecPending,
			Phase:       types.PhaseInspiration,
			StartTime:   &startTime,
		}

		err := s.Save(ctx, record)
		require.NoError(t, err)

		// Update the record
		record.Status = types.ExecRunning
		record.Phase = types.PhaseGoals
		record.Goals = &types.Goals{Content: "Test goals content"}

		err = s.Save(ctx, record)
		require.NoError(t, err)

		// Verify the update
		saved, err := s.Get(ctx, "exec_test_save_002")
		require.NoError(t, err)
		require.NotNil(t, saved)

		assert.Equal(t, types.ExecRunning, saved.Status)
		assert.Equal(t, types.PhaseGoals, saved.Phase)
		assert.NotNil(t, saved.Goals)
		assert.Equal(t, "Test goals content", saved.Goals.Content)
	})
}

// TestExecutionStoreGet tests retrieving execution records
func TestExecutionStoreGet(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	cleanupTestExecutions(t)
	defer cleanupTestExecutions(t)

	s := store.NewExecutionStore()
	ctx := context.Background()

	// Create a test record with all fields populated
	setupTestExecution(t, s, ctx)

	t.Run("returns_existing_record", func(t *testing.T) {
		record, err := s.Get(ctx, "exec_test_get_001")
		require.NoError(t, err)
		require.NotNil(t, record)

		assert.Equal(t, "exec_test_get_001", record.ExecutionID)
		assert.Equal(t, "member_test_get", record.MemberID)
		assert.Equal(t, "team_test_get", record.TeamID)
		assert.Equal(t, types.TriggerClock, record.TriggerType)
		assert.Equal(t, types.ExecCompleted, record.Status)
		assert.Equal(t, types.PhaseDelivery, record.Phase)

		// Verify phase outputs
		assert.NotNil(t, record.Inspiration)
		assert.Equal(t, "Test inspiration content", record.Inspiration.Content)
		assert.NotNil(t, record.Goals)
		assert.Equal(t, "Test goals content", record.Goals.Content)
		assert.Len(t, record.Tasks, 2)
		assert.Equal(t, "task_001", record.Tasks[0].ID)
		assert.Len(t, record.Results, 2)
		assert.True(t, record.Results[0].Success)
	})

	t.Run("returns_nil_for_non_existent_record", func(t *testing.T) {
		record, err := s.Get(ctx, "exec_non_existent")
		require.NoError(t, err)
		assert.Nil(t, record)
	})
}

// TestExecutionStoreList tests listing execution records with filters
func TestExecutionStoreList(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	cleanupTestExecutions(t)
	defer cleanupTestExecutions(t)

	s := store.NewExecutionStore()
	ctx := context.Background()

	// Create multiple test records
	setupTestExecutionsForList(t, s, ctx)

	t.Run("lists_all_records_without_filters", func(t *testing.T) {
		records, err := s.List(ctx, nil)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(records), 4)
	})

	t.Run("filters_by_member_id", func(t *testing.T) {
		records, err := s.List(ctx, &store.ListOptions{
			MemberID: "member_list_001",
		})
		require.NoError(t, err)
		assert.Equal(t, 2, len(records))
		for _, r := range records {
			assert.Equal(t, "member_list_001", r.MemberID)
		}
	})

	t.Run("filters_by_team_id", func(t *testing.T) {
		records, err := s.List(ctx, &store.ListOptions{
			TeamID: "team_list_001",
		})
		require.NoError(t, err)
		assert.Equal(t, 3, len(records))
		for _, r := range records {
			assert.Equal(t, "team_list_001", r.TeamID)
		}
	})

	t.Run("filters_by_status", func(t *testing.T) {
		records, err := s.List(ctx, &store.ListOptions{
			Status: types.ExecCompleted,
		})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(records), 2)
		for _, r := range records {
			assert.Equal(t, types.ExecCompleted, r.Status)
		}
	})

	t.Run("filters_by_trigger_type", func(t *testing.T) {
		records, err := s.List(ctx, &store.ListOptions{
			TriggerType: types.TriggerHuman,
		})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(records), 1)
		for _, r := range records {
			assert.Equal(t, types.TriggerHuman, r.TriggerType)
		}
	})

	t.Run("respects_limit", func(t *testing.T) {
		records, err := s.List(ctx, &store.ListOptions{
			Limit: 2,
		})
		require.NoError(t, err)
		assert.Equal(t, 2, len(records))
	})

	t.Run("combines_multiple_filters", func(t *testing.T) {
		records, err := s.List(ctx, &store.ListOptions{
			TeamID: "team_list_001",
			Status: types.ExecCompleted,
		})
		require.NoError(t, err)
		assert.Equal(t, 2, len(records))
		for _, r := range records {
			assert.Equal(t, "team_list_001", r.TeamID)
			assert.Equal(t, types.ExecCompleted, r.Status)
		}
	})
}

// TestExecutionStoreUpdatePhase tests updating phase and phase data
func TestExecutionStoreUpdatePhase(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	cleanupTestExecutions(t)
	defer cleanupTestExecutions(t)

	s := store.NewExecutionStore()
	ctx := context.Background()

	// Create a base record
	startTime := time.Now()
	record := &store.ExecutionRecord{
		ExecutionID: "exec_test_phase_001",
		MemberID:    "member_phase_001",
		TeamID:      "team_phase_001",
		TriggerType: types.TriggerClock,
		Status:      types.ExecRunning,
		Phase:       types.PhaseInspiration,
		StartTime:   &startTime,
	}
	err := s.Save(ctx, record)
	require.NoError(t, err)

	t.Run("updates_inspiration_phase", func(t *testing.T) {
		inspiration := &types.InspirationReport{
			Content: "Updated inspiration content",
		}
		err := s.UpdatePhase(ctx, "exec_test_phase_001", types.PhaseInspiration, inspiration)
		require.NoError(t, err)

		saved, err := s.Get(ctx, "exec_test_phase_001")
		require.NoError(t, err)
		assert.Equal(t, types.PhaseInspiration, saved.Phase)
		assert.NotNil(t, saved.Inspiration)
		assert.Equal(t, "Updated inspiration content", saved.Inspiration.Content)
	})

	t.Run("updates_goals_phase", func(t *testing.T) {
		goals := &types.Goals{
			Content: "Updated goals content",
		}
		err := s.UpdatePhase(ctx, "exec_test_phase_001", types.PhaseGoals, goals)
		require.NoError(t, err)

		saved, err := s.Get(ctx, "exec_test_phase_001")
		require.NoError(t, err)
		assert.Equal(t, types.PhaseGoals, saved.Phase)
		assert.NotNil(t, saved.Goals)
		assert.Equal(t, "Updated goals content", saved.Goals.Content)
	})

	t.Run("updates_tasks_phase", func(t *testing.T) {
		tasks := []types.Task{
			{ID: "task_phase_001", ExecutorType: types.ExecutorAssistant},
			{ID: "task_phase_002", ExecutorType: types.ExecutorProcess},
		}
		err := s.UpdatePhase(ctx, "exec_test_phase_001", types.PhaseTasks, tasks)
		require.NoError(t, err)

		saved, err := s.Get(ctx, "exec_test_phase_001")
		require.NoError(t, err)
		assert.Equal(t, types.PhaseTasks, saved.Phase)
		assert.Len(t, saved.Tasks, 2)
		assert.Equal(t, "task_phase_001", saved.Tasks[0].ID)
	})

	t.Run("updates_run_phase", func(t *testing.T) {
		results := []types.TaskResult{
			{TaskID: "task_phase_001", Success: true, Output: "Result 1"},
			{TaskID: "task_phase_002", Success: false, Error: "Failed"},
		}
		err := s.UpdatePhase(ctx, "exec_test_phase_001", types.PhaseRun, results)
		require.NoError(t, err)

		saved, err := s.Get(ctx, "exec_test_phase_001")
		require.NoError(t, err)
		assert.Equal(t, types.PhaseRun, saved.Phase)
		assert.Len(t, saved.Results, 2)
		assert.True(t, saved.Results[0].Success)
		assert.False(t, saved.Results[1].Success)
	})

	t.Run("updates_delivery_phase", func(t *testing.T) {
		delivery := &types.DeliveryResult{
			Success: true,
		}
		err := s.UpdatePhase(ctx, "exec_test_phase_001", types.PhaseDelivery, delivery)
		require.NoError(t, err)

		saved, err := s.Get(ctx, "exec_test_phase_001")
		require.NoError(t, err)
		assert.Equal(t, types.PhaseDelivery, saved.Phase)
		assert.NotNil(t, saved.Delivery)
		assert.True(t, saved.Delivery.Success)
	})

	t.Run("updates_learning_phase", func(t *testing.T) {
		learning := []types.LearningEntry{
			{Type: types.LearnExecution, Content: "Learned something"},
		}
		err := s.UpdatePhase(ctx, "exec_test_phase_001", types.PhaseLearning, learning)
		require.NoError(t, err)

		saved, err := s.Get(ctx, "exec_test_phase_001")
		require.NoError(t, err)
		assert.Equal(t, types.PhaseLearning, saved.Phase)
		assert.Len(t, saved.Learning, 1)
		assert.Equal(t, "Learned something", saved.Learning[0].Content)
	})
}

// TestExecutionStoreUpdateStatus tests updating execution status
func TestExecutionStoreUpdateStatus(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	cleanupTestExecutions(t)
	defer cleanupTestExecutions(t)

	s := store.NewExecutionStore()
	ctx := context.Background()

	t.Run("updates_status_to_running", func(t *testing.T) {
		startTime := time.Now()
		record := &store.ExecutionRecord{
			ExecutionID: "exec_test_status_001",
			MemberID:    "member_status_001",
			TeamID:      "team_status_001",
			TriggerType: types.TriggerClock,
			Status:      types.ExecPending,
			Phase:       types.PhaseInspiration,
			StartTime:   &startTime,
		}
		err := s.Save(ctx, record)
		require.NoError(t, err)

		err = s.UpdateStatus(ctx, "exec_test_status_001", types.ExecRunning, "")
		require.NoError(t, err)

		saved, err := s.Get(ctx, "exec_test_status_001")
		require.NoError(t, err)
		assert.Equal(t, types.ExecRunning, saved.Status)
		assert.Nil(t, saved.EndTime) // Should not set end_time for running
	})

	t.Run("updates_status_to_completed_with_end_time", func(t *testing.T) {
		startTime := time.Now()
		record := &store.ExecutionRecord{
			ExecutionID: "exec_test_status_002",
			MemberID:    "member_status_002",
			TeamID:      "team_status_002",
			TriggerType: types.TriggerHuman,
			Status:      types.ExecRunning,
			Phase:       types.PhaseDelivery,
			StartTime:   &startTime,
		}
		err := s.Save(ctx, record)
		require.NoError(t, err)

		err = s.UpdateStatus(ctx, "exec_test_status_002", types.ExecCompleted, "")
		require.NoError(t, err)

		saved, err := s.Get(ctx, "exec_test_status_002")
		require.NoError(t, err)
		assert.Equal(t, types.ExecCompleted, saved.Status)
		assert.NotNil(t, saved.EndTime) // Should set end_time for completed
	})

	t.Run("updates_status_to_failed_with_error", func(t *testing.T) {
		startTime := time.Now()
		record := &store.ExecutionRecord{
			ExecutionID: "exec_test_status_003",
			MemberID:    "member_status_003",
			TeamID:      "team_status_003",
			TriggerType: types.TriggerEvent,
			Status:      types.ExecRunning,
			Phase:       types.PhaseRun,
			StartTime:   &startTime,
		}
		err := s.Save(ctx, record)
		require.NoError(t, err)

		err = s.UpdateStatus(ctx, "exec_test_status_003", types.ExecFailed, "Task execution failed: timeout")
		require.NoError(t, err)

		saved, err := s.Get(ctx, "exec_test_status_003")
		require.NoError(t, err)
		assert.Equal(t, types.ExecFailed, saved.Status)
		assert.Equal(t, "Task execution failed: timeout", saved.Error)
		assert.NotNil(t, saved.EndTime) // Should set end_time for failed
	})

	t.Run("updates_status_to_cancelled", func(t *testing.T) {
		startTime := time.Now()
		record := &store.ExecutionRecord{
			ExecutionID: "exec_test_status_004",
			MemberID:    "member_status_004",
			TeamID:      "team_status_004",
			TriggerType: types.TriggerClock,
			Status:      types.ExecRunning,
			Phase:       types.PhaseTasks,
			StartTime:   &startTime,
		}
		err := s.Save(ctx, record)
		require.NoError(t, err)

		err = s.UpdateStatus(ctx, "exec_test_status_004", types.ExecCancelled, "User cancelled")
		require.NoError(t, err)

		saved, err := s.Get(ctx, "exec_test_status_004")
		require.NoError(t, err)
		assert.Equal(t, types.ExecCancelled, saved.Status)
		assert.Equal(t, "User cancelled", saved.Error)
		assert.NotNil(t, saved.EndTime) // Should set end_time for cancelled
	})
}

// TestExecutionStoreUpdateCurrent tests updating current state
func TestExecutionStoreUpdateCurrent(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	cleanupTestExecutions(t)
	defer cleanupTestExecutions(t)

	s := store.NewExecutionStore()
	ctx := context.Background()

	// Create a base record
	startTime := time.Now()
	record := &store.ExecutionRecord{
		ExecutionID: "exec_test_current_001",
		MemberID:    "member_current_001",
		TeamID:      "team_current_001",
		TriggerType: types.TriggerClock,
		Status:      types.ExecRunning,
		Phase:       types.PhaseRun,
		StartTime:   &startTime,
	}
	err := s.Save(ctx, record)
	require.NoError(t, err)

	t.Run("updates_current_state", func(t *testing.T) {
		current := &store.CurrentState{
			TaskIndex: 2,
			Progress:  "3/5 tasks completed",
		}
		err := s.UpdateCurrent(ctx, "exec_test_current_001", current)
		require.NoError(t, err)

		saved, err := s.Get(ctx, "exec_test_current_001")
		require.NoError(t, err)
		assert.NotNil(t, saved.Current)
		assert.Equal(t, 2, saved.Current.TaskIndex)
		assert.Equal(t, "3/5 tasks completed", saved.Current.Progress)
	})
}

// TestExecutionStoreDelete tests deleting execution records
func TestExecutionStoreDelete(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	cleanupTestExecutions(t)
	defer cleanupTestExecutions(t)

	s := store.NewExecutionStore()
	ctx := context.Background()

	t.Run("deletes_existing_record", func(t *testing.T) {
		// Create a record
		startTime := time.Now()
		record := &store.ExecutionRecord{
			ExecutionID: "exec_test_delete_001",
			MemberID:    "member_delete_001",
			TeamID:      "team_delete_001",
			TriggerType: types.TriggerClock,
			Status:      types.ExecCompleted,
			Phase:       types.PhaseDelivery,
			StartTime:   &startTime,
		}
		err := s.Save(ctx, record)
		require.NoError(t, err)

		// Verify it exists
		saved, err := s.Get(ctx, "exec_test_delete_001")
		require.NoError(t, err)
		require.NotNil(t, saved)

		// Delete it
		err = s.Delete(ctx, "exec_test_delete_001")
		require.NoError(t, err)

		// Verify it's gone
		saved, err = s.Get(ctx, "exec_test_delete_001")
		require.NoError(t, err)
		assert.Nil(t, saved)
	})

	t.Run("no_error_for_non_existent_record", func(t *testing.T) {
		err := s.Delete(ctx, "exec_non_existent")
		assert.NoError(t, err)
	})
}

// TestExecutionRecordConversion tests conversion between ExecutionRecord and Execution
func TestExecutionRecordConversion(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	t.Run("converts_from_execution", func(t *testing.T) {
		now := time.Now()
		endTime := now.Add(time.Hour)
		exec := &types.Execution{
			ID:          "exec_convert_001",
			MemberID:    "member_convert_001",
			TeamID:      "team_convert_001",
			JobID:       "job_convert_001",
			TriggerType: types.TriggerHuman,
			Status:      types.ExecCompleted,
			Phase:       types.PhaseDelivery,
			StartTime:   now,
			EndTime:     &endTime,
			Error:       "",
			Inspiration: &types.InspirationReport{Content: "Test inspiration"},
			Goals:       &types.Goals{Content: "Test goals"},
			Tasks: []types.Task{
				{ID: "task_001", ExecutorType: types.ExecutorAssistant},
			},
			Results: []types.TaskResult{
				{TaskID: "task_001", Success: true},
			},
			Current: &types.CurrentState{
				TaskIndex: 1,
				Progress:  "1/1 tasks",
			},
		}

		record := store.FromExecution(exec)

		assert.Equal(t, "exec_convert_001", record.ExecutionID)
		assert.Equal(t, "member_convert_001", record.MemberID)
		assert.Equal(t, "team_convert_001", record.TeamID)
		assert.Equal(t, "job_convert_001", record.JobID)
		assert.Equal(t, types.TriggerHuman, record.TriggerType)
		assert.Equal(t, types.ExecCompleted, record.Status)
		assert.Equal(t, types.PhaseDelivery, record.Phase)
		assert.NotNil(t, record.StartTime)
		assert.NotNil(t, record.EndTime)
		assert.NotNil(t, record.Inspiration)
		assert.NotNil(t, record.Goals)
		assert.Len(t, record.Tasks, 1)
		assert.Len(t, record.Results, 1)
		assert.NotNil(t, record.Current)
		assert.Equal(t, 1, record.Current.TaskIndex)
	})

	t.Run("converts_to_execution", func(t *testing.T) {
		now := time.Now()
		endTime := now.Add(time.Hour)
		record := &store.ExecutionRecord{
			ExecutionID: "exec_convert_002",
			MemberID:    "member_convert_002",
			TeamID:      "team_convert_002",
			JobID:       "job_convert_002",
			TriggerType: types.TriggerClock,
			Status:      types.ExecRunning,
			Phase:       types.PhaseRun,
			StartTime:   &now,
			EndTime:     &endTime,
			Inspiration: &types.InspirationReport{Content: "Test inspiration"},
			Goals:       &types.Goals{Content: "Test goals"},
			Tasks: []types.Task{
				{ID: "task_002", ExecutorType: types.ExecutorProcess},
			},
			Results: []types.TaskResult{
				{TaskID: "task_002", Success: false, Error: "Failed"},
			},
			Current: &store.CurrentState{
				TaskIndex: 0,
				Progress:  "0/1 tasks",
			},
		}

		exec := record.ToExecution()

		assert.Equal(t, "exec_convert_002", exec.ID)
		assert.Equal(t, "member_convert_002", exec.MemberID)
		assert.Equal(t, "team_convert_002", exec.TeamID)
		assert.Equal(t, "job_convert_002", exec.JobID)
		assert.Equal(t, types.TriggerClock, exec.TriggerType)
		assert.Equal(t, types.ExecRunning, exec.Status)
		assert.Equal(t, types.PhaseRun, exec.Phase)
		assert.NotNil(t, exec.Inspiration)
		assert.NotNil(t, exec.Goals)
		assert.Len(t, exec.Tasks, 1)
		assert.Len(t, exec.Results, 1)
		assert.NotNil(t, exec.Current)
		assert.Equal(t, 0, exec.Current.TaskIndex)
	})
}

// Helper functions

func cleanupTestExecutions(t *testing.T) {
	mod := model.Select("__yao.agent.execution")
	if mod == nil {
		return
	}

	// Delete all test execution records
	_, err := mod.DeleteWhere(model.QueryParam{
		Wheres: []model.QueryWhere{
			{Column: "execution_id", OP: "like", Value: "exec_test_%"},
		},
	})
	if err != nil {
		t.Logf("Warning: failed to cleanup test executions: %v", err)
	}
}

func setupTestExecution(t *testing.T, s *store.ExecutionStore, ctx context.Context) {
	startTime := time.Now().Add(-time.Hour)
	endTime := time.Now()

	record := &store.ExecutionRecord{
		ExecutionID: "exec_test_get_001",
		MemberID:    "member_test_get",
		TeamID:      "team_test_get",
		JobID:       "job_test_get",
		TriggerType: types.TriggerClock,
		Status:      types.ExecCompleted,
		Phase:       types.PhaseDelivery,
		StartTime:   &startTime,
		EndTime:     &endTime,
		Inspiration: &types.InspirationReport{
			Content: "Test inspiration content",
		},
		Goals: &types.Goals{
			Content: "Test goals content",
		},
		Tasks: []types.Task{
			{ID: "task_001", ExecutorType: types.ExecutorAssistant, Status: types.TaskCompleted},
			{ID: "task_002", ExecutorType: types.ExecutorProcess, Status: types.TaskCompleted},
		},
		Results: []types.TaskResult{
			{TaskID: "task_001", Success: true, Output: "Result 1"},
			{TaskID: "task_002", Success: true, Output: "Result 2"},
		},
		Delivery: &types.DeliveryResult{
			Success: true,
		},
		Learning: []types.LearningEntry{
			{Type: types.LearnExecution, Content: "Test learning"},
		},
	}

	err := s.Save(ctx, record)
	require.NoError(t, err)
}

func setupTestExecutionsForList(t *testing.T, s *store.ExecutionStore, ctx context.Context) {
	startTime := time.Now()

	records := []*store.ExecutionRecord{
		{
			ExecutionID: "exec_test_list_001",
			MemberID:    "member_list_001",
			TeamID:      "team_list_001",
			TriggerType: types.TriggerClock,
			Status:      types.ExecCompleted,
			Phase:       types.PhaseDelivery,
			StartTime:   &startTime,
		},
		{
			ExecutionID: "exec_test_list_002",
			MemberID:    "member_list_001",
			TeamID:      "team_list_001",
			TriggerType: types.TriggerClock,
			Status:      types.ExecCompleted,
			Phase:       types.PhaseDelivery,
			StartTime:   &startTime,
		},
		{
			ExecutionID: "exec_test_list_003",
			MemberID:    "member_list_002",
			TeamID:      "team_list_001",
			TriggerType: types.TriggerHuman,
			Status:      types.ExecRunning,
			Phase:       types.PhaseRun,
			StartTime:   &startTime,
		},
		{
			ExecutionID: "exec_test_list_004",
			MemberID:    "member_list_002",
			TeamID:      "team_list_002",
			TriggerType: types.TriggerEvent,
			Status:      types.ExecFailed,
			Phase:       types.PhaseRun,
			StartTime:   &startTime,
			Error:       "Test error",
		},
	}

	for _, record := range records {
		err := s.Save(ctx, record)
		require.NoError(t, err)
	}
}
