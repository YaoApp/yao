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

// TestExecutionStoreUpdateUIFields tests updating UI display fields
func TestExecutionStoreUpdateUIFields(t *testing.T) {
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
		ExecutionID: "exec_test_uifields_001",
		MemberID:    "member_uifields_001",
		TeamID:      "team_uifields_001",
		TriggerType: types.TriggerHuman,
		Status:      types.ExecRunning,
		Phase:       types.PhaseInspiration,
		StartTime:   &startTime,
	}
	err := s.Save(ctx, record)
	require.NoError(t, err)

	t.Run("updates_name_only", func(t *testing.T) {
		err := s.UpdateUIFields(ctx, "exec_test_uifields_001", "Analyze sales data", "")
		require.NoError(t, err)

		saved, err := s.Get(ctx, "exec_test_uifields_001")
		require.NoError(t, err)
		assert.Equal(t, "Analyze sales data", saved.Name)
		assert.Equal(t, "", saved.CurrentTaskName)
	})

	t.Run("updates_current_task_name_only", func(t *testing.T) {
		err := s.UpdateUIFields(ctx, "exec_test_uifields_001", "", "Analyzing context...")
		require.NoError(t, err)

		saved, err := s.Get(ctx, "exec_test_uifields_001")
		require.NoError(t, err)
		assert.Equal(t, "Analyze sales data", saved.Name) // Previous value retained
		assert.Equal(t, "Analyzing context...", saved.CurrentTaskName)
	})

	t.Run("updates_both_fields", func(t *testing.T) {
		err := s.UpdateUIFields(ctx, "exec_test_uifields_001", "Generate monthly report", "Task 1/3: Collect data")
		require.NoError(t, err)

		saved, err := s.Get(ctx, "exec_test_uifields_001")
		require.NoError(t, err)
		assert.Equal(t, "Generate monthly report", saved.Name)
		assert.Equal(t, "Task 1/3: Collect data", saved.CurrentTaskName)
	})

	t.Run("does_nothing_when_both_empty", func(t *testing.T) {
		// Get current values
		before, err := s.Get(ctx, "exec_test_uifields_001")
		require.NoError(t, err)

		// Update with empty strings
		err = s.UpdateUIFields(ctx, "exec_test_uifields_001", "", "")
		require.NoError(t, err)

		// Values should remain unchanged
		after, err := s.Get(ctx, "exec_test_uifields_001")
		require.NoError(t, err)
		assert.Equal(t, before.Name, after.Name)
		assert.Equal(t, before.CurrentTaskName, after.CurrentTaskName)
	})

	t.Run("handles_chinese_content", func(t *testing.T) {
		err := s.UpdateUIFields(ctx, "exec_test_uifields_001", "生成月度报告", "任务 2/3: 分析数据")
		require.NoError(t, err)

		saved, err := s.Get(ctx, "exec_test_uifields_001")
		require.NoError(t, err)
		assert.Equal(t, "生成月度报告", saved.Name)
		assert.Equal(t, "任务 2/3: 分析数据", saved.CurrentTaskName)
	})

	t.Run("handles_long_content", func(t *testing.T) {
		longName := "This is a very long execution name that might come from a detailed user instruction about what they want the robot to accomplish in this particular run cycle"
		longTask := "Task 1/5: Processing a complex multi-step operation with various sub-tasks that need to be completed..."

		err := s.UpdateUIFields(ctx, "exec_test_uifields_001", longName, longTask)
		require.NoError(t, err)

		saved, err := s.Get(ctx, "exec_test_uifields_001")
		require.NoError(t, err)
		assert.Equal(t, longName, saved.Name)
		assert.Equal(t, longTask, saved.CurrentTaskName)
	})
}

// TestExecutionStoreUpdateTasks tests updating tasks array with status
func TestExecutionStoreUpdateTasks(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	cleanupTestExecutions(t)
	defer cleanupTestExecutions(t)

	s := store.NewExecutionStore()
	ctx := context.Background()

	// Create a base record with initial tasks
	startTime := time.Now()
	record := &store.ExecutionRecord{
		ExecutionID: "exec_test_tasks_001",
		MemberID:    "member_tasks_001",
		TeamID:      "team_tasks_001",
		TriggerType: types.TriggerClock,
		Status:      types.ExecRunning,
		Phase:       types.PhaseRun,
		StartTime:   &startTime,
		Tasks: []types.Task{
			{ID: "task_001", ExecutorType: types.ExecutorAssistant, Status: types.TaskPending, Order: 0},
			{ID: "task_002", ExecutorType: types.ExecutorProcess, Status: types.TaskPending, Order: 1},
			{ID: "task_003", ExecutorType: types.ExecutorAssistant, Status: types.TaskPending, Order: 2},
		},
	}
	err := s.Save(ctx, record)
	require.NoError(t, err)

	t.Run("updates_task_status_to_running", func(t *testing.T) {
		// Update first task to running
		tasks := []types.Task{
			{ID: "task_001", ExecutorType: types.ExecutorAssistant, Status: types.TaskRunning, Order: 0},
			{ID: "task_002", ExecutorType: types.ExecutorProcess, Status: types.TaskPending, Order: 1},
			{ID: "task_003", ExecutorType: types.ExecutorAssistant, Status: types.TaskPending, Order: 2},
		}
		current := &store.CurrentState{TaskIndex: 0, Progress: "1/3 tasks"}

		err := s.UpdateTasks(ctx, "exec_test_tasks_001", tasks, current)
		require.NoError(t, err)

		saved, err := s.Get(ctx, "exec_test_tasks_001")
		require.NoError(t, err)
		require.Len(t, saved.Tasks, 3)

		assert.Equal(t, types.TaskRunning, saved.Tasks[0].Status)
		assert.Equal(t, types.TaskPending, saved.Tasks[1].Status)
		assert.Equal(t, types.TaskPending, saved.Tasks[2].Status)

		assert.NotNil(t, saved.Current)
		assert.Equal(t, 0, saved.Current.TaskIndex)
	})

	t.Run("updates_task_status_to_completed", func(t *testing.T) {
		// First task completed, second running
		tasks := []types.Task{
			{ID: "task_001", ExecutorType: types.ExecutorAssistant, Status: types.TaskCompleted, Order: 0},
			{ID: "task_002", ExecutorType: types.ExecutorProcess, Status: types.TaskRunning, Order: 1},
			{ID: "task_003", ExecutorType: types.ExecutorAssistant, Status: types.TaskPending, Order: 2},
		}
		current := &store.CurrentState{TaskIndex: 1, Progress: "2/3 tasks"}

		err := s.UpdateTasks(ctx, "exec_test_tasks_001", tasks, current)
		require.NoError(t, err)

		saved, err := s.Get(ctx, "exec_test_tasks_001")
		require.NoError(t, err)

		assert.Equal(t, types.TaskCompleted, saved.Tasks[0].Status)
		assert.Equal(t, types.TaskRunning, saved.Tasks[1].Status)
		assert.Equal(t, types.TaskPending, saved.Tasks[2].Status)
		assert.Equal(t, 1, saved.Current.TaskIndex)
	})

	t.Run("updates_task_status_to_failed_with_skipped", func(t *testing.T) {
		// Second task failed, third skipped
		tasks := []types.Task{
			{ID: "task_001", ExecutorType: types.ExecutorAssistant, Status: types.TaskCompleted, Order: 0},
			{ID: "task_002", ExecutorType: types.ExecutorProcess, Status: types.TaskFailed, Order: 1},
			{ID: "task_003", ExecutorType: types.ExecutorAssistant, Status: types.TaskSkipped, Order: 2},
		}
		current := &store.CurrentState{TaskIndex: 1, Progress: "Failed at 2/3"}

		err := s.UpdateTasks(ctx, "exec_test_tasks_001", tasks, current)
		require.NoError(t, err)

		saved, err := s.Get(ctx, "exec_test_tasks_001")
		require.NoError(t, err)

		assert.Equal(t, types.TaskCompleted, saved.Tasks[0].Status)
		assert.Equal(t, types.TaskFailed, saved.Tasks[1].Status)
		assert.Equal(t, types.TaskSkipped, saved.Tasks[2].Status)
	})

	t.Run("updates_with_nil_current", func(t *testing.T) {
		// All tasks completed, no current
		tasks := []types.Task{
			{ID: "task_001", ExecutorType: types.ExecutorAssistant, Status: types.TaskCompleted, Order: 0},
			{ID: "task_002", ExecutorType: types.ExecutorProcess, Status: types.TaskCompleted, Order: 1},
			{ID: "task_003", ExecutorType: types.ExecutorAssistant, Status: types.TaskCompleted, Order: 2},
		}

		err := s.UpdateTasks(ctx, "exec_test_tasks_001", tasks, nil)
		require.NoError(t, err)

		saved, err := s.Get(ctx, "exec_test_tasks_001")
		require.NoError(t, err)

		assert.Equal(t, types.TaskCompleted, saved.Tasks[0].Status)
		assert.Equal(t, types.TaskCompleted, saved.Tasks[1].Status)
		assert.Equal(t, types.TaskCompleted, saved.Tasks[2].Status)
	})

	t.Run("preserves_task_description", func(t *testing.T) {
		// Create a new record with descriptions
		record2 := &store.ExecutionRecord{
			ExecutionID: "exec_test_tasks_002",
			MemberID:    "member_tasks_002",
			TeamID:      "team_tasks_002",
			TriggerType: types.TriggerHuman,
			Status:      types.ExecRunning,
			Phase:       types.PhaseRun,
			StartTime:   &startTime,
			Tasks: []types.Task{
				{ID: "task_d01", Description: "Analyze data", ExecutorType: types.ExecutorAssistant, Status: types.TaskPending, Order: 0},
				{ID: "task_d02", Description: "Generate report", ExecutorType: types.ExecutorAssistant, Status: types.TaskPending, Order: 1},
			},
		}
		err := s.Save(ctx, record2)
		require.NoError(t, err)

		// Update status preserving description
		tasks := []types.Task{
			{ID: "task_d01", Description: "Analyze data", ExecutorType: types.ExecutorAssistant, Status: types.TaskCompleted, Order: 0},
			{ID: "task_d02", Description: "Generate report", ExecutorType: types.ExecutorAssistant, Status: types.TaskRunning, Order: 1},
		}

		err = s.UpdateTasks(ctx, "exec_test_tasks_002", tasks, &store.CurrentState{TaskIndex: 1})
		require.NoError(t, err)

		saved, err := s.Get(ctx, "exec_test_tasks_002")
		require.NoError(t, err)

		assert.Equal(t, "Analyze data", saved.Tasks[0].Description)
		assert.Equal(t, "Generate report", saved.Tasks[1].Description)
		assert.Equal(t, types.TaskCompleted, saved.Tasks[0].Status)
		assert.Equal(t, types.TaskRunning, saved.Tasks[1].Status)
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
			ID:              "exec_convert_001",
			MemberID:        "member_convert_001",
			TeamID:          "team_convert_001",
			TriggerType:     types.TriggerHuman,
			Status:          types.ExecCompleted,
			Phase:           types.PhaseDelivery,
			StartTime:       now,
			EndTime:         &endTime,
			Error:           "",
			Name:            "Analyze sales data",
			CurrentTaskName: "Task 1/3: Processing",
			Inspiration:     &types.InspirationReport{Content: "Test inspiration"},
			Goals:           &types.Goals{Content: "Test goals"},
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
		// Verify UI fields conversion
		assert.Equal(t, "Analyze sales data", record.Name)
		assert.Equal(t, "Task 1/3: Processing", record.CurrentTaskName)
	})

	t.Run("converts_to_execution", func(t *testing.T) {
		now := time.Now()
		endTime := now.Add(time.Hour)
		record := &store.ExecutionRecord{
			ExecutionID:     "exec_convert_002",
			MemberID:        "member_convert_002",
			TeamID:          "team_convert_002",
			TriggerType:     types.TriggerClock,
			Status:          types.ExecRunning,
			Phase:           types.PhaseRun,
			StartTime:       &now,
			EndTime:         &endTime,
			Name:            "定时执行",
			CurrentTaskName: "任务 1/2: 数据分析",
			Inspiration:     &types.InspirationReport{Content: "Test inspiration"},
			Goals:           &types.Goals{Content: "Test goals"},
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
		assert.Equal(t, types.TriggerClock, exec.TriggerType)
		assert.Equal(t, types.ExecRunning, exec.Status)
		assert.Equal(t, types.PhaseRun, exec.Phase)
		assert.NotNil(t, exec.Inspiration)
		assert.NotNil(t, exec.Goals)
		assert.Len(t, exec.Tasks, 1)
		assert.Len(t, exec.Results, 1)
		assert.NotNil(t, exec.Current)
		assert.Equal(t, 0, exec.Current.TaskIndex)
		// Verify UI fields conversion
		assert.Equal(t, "定时执行", exec.Name)
		assert.Equal(t, "任务 1/2: 数据分析", exec.CurrentTaskName)
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

// ==================== Results & Activities Tests ====================

// TestExecutionStoreListResults tests listing execution results (deliveries)
func TestExecutionStoreListResults(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	cleanupTestExecutions(t)
	defer cleanupTestExecutions(t)

	s := store.NewExecutionStore()
	ctx := context.Background()

	// Setup test data with delivery content
	setupTestResultsData(t, s, ctx)

	t.Run("lists_results_without_filters", func(t *testing.T) {
		result, err := s.ListResults(ctx, &store.ResultListOptions{
			MemberID: "member_result_001",
		})
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 2, result.Total)
		assert.Len(t, result.Data, 2)
		// Should be ordered by end_time desc
		for _, r := range result.Data {
			assert.NotNil(t, r.Delivery)
			assert.NotNil(t, r.Delivery.Content)
		}
	})

	t.Run("filters_by_trigger_type", func(t *testing.T) {
		result, err := s.ListResults(ctx, &store.ResultListOptions{
			MemberID:    "member_result_001",
			TriggerType: types.TriggerClock,
		})
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 1, result.Total)
		assert.Len(t, result.Data, 1)
		assert.Equal(t, types.TriggerClock, result.Data[0].TriggerType)
	})

	t.Run("filters_by_keyword", func(t *testing.T) {
		result, err := s.ListResults(ctx, &store.ResultListOptions{
			MemberID: "member_result_001",
			Keyword:  "Weekly",
		})
		require.NoError(t, err)
		require.NotNil(t, result)
		// Should match "Weekly Sales Report"
		assert.GreaterOrEqual(t, result.Total, 1)
	})

	t.Run("respects_pagination", func(t *testing.T) {
		result, err := s.ListResults(ctx, &store.ResultListOptions{
			MemberID: "member_result_001",
			Limit:    1,
			Offset:   0,
		})
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 1, len(result.Data))
		assert.Equal(t, 2, result.Total)
		assert.Equal(t, 1, result.Page)
	})

	t.Run("excludes_executions_without_delivery", func(t *testing.T) {
		result, err := s.ListResults(ctx, &store.ResultListOptions{
			MemberID: "member_result_002", // Has no delivery content
		})
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 0, result.Total)
		assert.Empty(t, result.Data)
	})
}

// TestExecutionStoreCountResults tests counting results
func TestExecutionStoreCountResults(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	cleanupTestExecutions(t)
	defer cleanupTestExecutions(t)

	s := store.NewExecutionStore()
	ctx := context.Background()

	// Setup test data with delivery content
	setupTestResultsData(t, s, ctx)

	t.Run("counts_all_results_for_member", func(t *testing.T) {
		count, err := s.CountResults(ctx, &store.ResultListOptions{
			MemberID: "member_result_001",
		})
		require.NoError(t, err)
		assert.Equal(t, 2, count)
	})

	t.Run("counts_filtered_results", func(t *testing.T) {
		count, err := s.CountResults(ctx, &store.ResultListOptions{
			MemberID:    "member_result_001",
			TriggerType: types.TriggerHuman,
		})
		require.NoError(t, err)
		assert.Equal(t, 1, count)
	})

	t.Run("returns_zero_for_no_results", func(t *testing.T) {
		count, err := s.CountResults(ctx, &store.ResultListOptions{
			MemberID: "member_result_002",
		})
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})
}

// TestExecutionStoreListActivities tests listing activities
func TestExecutionStoreListActivities(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	cleanupTestExecutions(t)
	defer cleanupTestExecutions(t)

	s := store.NewExecutionStore()
	ctx := context.Background()

	// Setup test data
	setupTestActivitiesData(t, s, ctx)

	t.Run("lists_activities_for_team", func(t *testing.T) {
		activities, err := s.ListActivities(ctx, &store.ActivityListOptions{
			TeamID: "team_activity_001",
		})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(activities), 3)
	})

	t.Run("respects_limit", func(t *testing.T) {
		activities, err := s.ListActivities(ctx, &store.ActivityListOptions{
			TeamID: "team_activity_001",
			Limit:  2,
		})
		require.NoError(t, err)
		assert.LessOrEqual(t, len(activities), 2)
	})

	t.Run("filters_by_since", func(t *testing.T) {
		// Without since, should get all activities
		activitiesAll, err := s.ListActivities(ctx, &store.ActivityListOptions{
			TeamID: "team_activity_001",
		})
		require.NoError(t, err)
		allCount := len(activitiesAll)
		assert.GreaterOrEqual(t, allCount, 3, "should have at least 3 activities without filter")

		// Use a time in the future to ensure we get no results
		future := time.Now().Add(24 * time.Hour)
		activitiesFuture, err := s.ListActivities(ctx, &store.ActivityListOptions{
			TeamID: "team_activity_001",
			Since:  &future,
		})
		require.NoError(t, err)
		assert.Equal(t, 0, len(activitiesFuture), "should get no results with future since time")
	})

	t.Run("generates_correct_activity_types", func(t *testing.T) {
		activities, err := s.ListActivities(ctx, &store.ActivityListOptions{
			TeamID: "team_activity_001",
		})
		require.NoError(t, err)

		// Should have activities of different types
		typeCount := make(map[store.ActivityType]int)
		for _, a := range activities {
			typeCount[a.Type]++
		}

		// We should have at least completed and failed types
		assert.Greater(t, typeCount[store.ActivityExecutionCompleted], 0, "should have completed activities")
		assert.Greater(t, typeCount[store.ActivityExecutionFailed], 0, "should have failed activities")
	})

	t.Run("filters_by_type_completed", func(t *testing.T) {
		activities, err := s.ListActivities(ctx, &store.ActivityListOptions{
			TeamID: "team_activity_001",
			Type:   store.ActivityExecutionCompleted,
		})
		require.NoError(t, err)

		// All returned activities should be of type completed
		for _, a := range activities {
			assert.Equal(t, store.ActivityExecutionCompleted, a.Type, "all activities should be completed type")
		}
		assert.Greater(t, len(activities), 0, "should have at least one completed activity")
	})

	t.Run("filters_by_type_failed", func(t *testing.T) {
		activities, err := s.ListActivities(ctx, &store.ActivityListOptions{
			TeamID: "team_activity_001",
			Type:   store.ActivityExecutionFailed,
		})
		require.NoError(t, err)

		// All returned activities should be of type failed
		for _, a := range activities {
			assert.Equal(t, store.ActivityExecutionFailed, a.Type, "all activities should be failed type")
		}
		assert.Greater(t, len(activities), 0, "should have at least one failed activity")
	})

	t.Run("filters_by_type_invalid_returns_empty", func(t *testing.T) {
		activities, err := s.ListActivities(ctx, &store.ActivityListOptions{
			TeamID: "team_activity_001",
			Type:   store.ActivityType("invalid.type"),
		})
		require.NoError(t, err)

		// Invalid type should return empty result
		assert.Equal(t, 0, len(activities), "invalid type should return empty result")
	})

	t.Run("includes_execution_name_in_message", func(t *testing.T) {
		activities, err := s.ListActivities(ctx, &store.ActivityListOptions{
			TeamID: "team_activity_001",
		})
		require.NoError(t, err)

		// Find a completed activity
		var completedActivity *store.Activity
		for _, a := range activities {
			if a.Type == store.ActivityExecutionCompleted && a.Message != "" {
				completedActivity = a
				break
			}
		}

		require.NotNil(t, completedActivity, "should find a completed activity")
		assert.Contains(t, completedActivity.Message, "Completed")
	})
}

// Helper function to setup test results data
func setupTestResultsData(t *testing.T, s *store.ExecutionStore, ctx context.Context) {
	startTime := time.Now().Add(-2 * time.Hour)
	endTime := time.Now().Add(-1 * time.Hour)
	endTime2 := time.Now().Add(-30 * time.Minute)

	records := []*store.ExecutionRecord{
		{
			ExecutionID: "exec_test_result_001",
			MemberID:    "member_result_001",
			TeamID:      "team_result_001",
			TriggerType: types.TriggerClock,
			Status:      types.ExecCompleted,
			Phase:       types.PhaseDelivery,
			Name:        "Weekly Sales Report",
			StartTime:   &startTime,
			EndTime:     &endTime,
			Delivery: &types.DeliveryResult{
				Success: true,
				Content: &types.DeliveryContent{
					Summary: "Weekly sales report generated successfully",
					Body:    "## Weekly Sales Report\n\nTotal sales: $50,000",
				},
			},
		},
		{
			ExecutionID: "exec_test_result_002",
			MemberID:    "member_result_001",
			TeamID:      "team_result_001",
			TriggerType: types.TriggerHuman,
			Status:      types.ExecCompleted,
			Phase:       types.PhaseDelivery,
			Name:        "Custom Analysis",
			StartTime:   &startTime,
			EndTime:     &endTime2,
			Delivery: &types.DeliveryResult{
				Success: true,
				Content: &types.DeliveryContent{
					Summary: "Custom analysis completed",
					Body:    "## Analysis Results\n\nFindings...",
					Attachments: []types.DeliveryAttachment{
						{Title: "Report.pdf", File: "__attachment://file_001"},
					},
				},
			},
		},
		{
			// Completed but no delivery content - should be excluded
			ExecutionID: "exec_test_result_003",
			MemberID:    "member_result_002",
			TeamID:      "team_result_001",
			TriggerType: types.TriggerClock,
			Status:      types.ExecCompleted,
			Phase:       types.PhaseDelivery,
			Name:        "No Delivery Content",
			StartTime:   &startTime,
			EndTime:     &endTime,
			// No Delivery field
		},
		{
			// Running - should be excluded from results
			ExecutionID: "exec_test_result_004",
			MemberID:    "member_result_001",
			TeamID:      "team_result_001",
			TriggerType: types.TriggerClock,
			Status:      types.ExecRunning,
			Phase:       types.PhaseRun,
			Name:        "Running Task",
			StartTime:   &startTime,
		},
	}

	for _, record := range records {
		err := s.Save(ctx, record)
		require.NoError(t, err)
	}
}

// Helper function to setup test activities data
func setupTestActivitiesData(t *testing.T, s *store.ExecutionStore, ctx context.Context) {
	startTime := time.Now().Add(-2 * time.Hour)
	endTime := time.Now().Add(-1 * time.Hour)
	endTimeFailed := time.Now().Add(-45 * time.Minute)

	records := []*store.ExecutionRecord{
		{
			ExecutionID: "exec_test_activity_001",
			MemberID:    "member_activity_001",
			TeamID:      "team_activity_001",
			TriggerType: types.TriggerClock,
			Status:      types.ExecCompleted,
			Phase:       types.PhaseDelivery,
			Name:        "Daily Report",
			StartTime:   &startTime,
			EndTime:     &endTime,
		},
		{
			ExecutionID: "exec_test_activity_002",
			MemberID:    "member_activity_001",
			TeamID:      "team_activity_001",
			TriggerType: types.TriggerHuman,
			Status:      types.ExecFailed,
			Phase:       types.PhaseRun,
			Name:        "Custom Task",
			StartTime:   &startTime,
			EndTime:     &endTimeFailed,
			Error:       "Task timeout",
		},
		{
			ExecutionID: "exec_test_activity_003",
			MemberID:    "member_activity_002",
			TeamID:      "team_activity_001",
			TriggerType: types.TriggerEvent,
			Status:      types.ExecCancelled,
			Phase:       types.PhaseTasks,
			Name:        "Lead Processing",
			StartTime:   &startTime,
			EndTime:     &endTime,
		},
		{
			ExecutionID: "exec_test_activity_004",
			MemberID:    "member_activity_002",
			TeamID:      "team_activity_001",
			TriggerType: types.TriggerClock,
			Status:      types.ExecRunning,
			Phase:       types.PhaseRun,
			Name:        "Data Analysis",
			StartTime:   &startTime,
		},
	}

	for _, record := range records {
		err := s.Save(ctx, record)
		require.NoError(t, err)
	}
}
