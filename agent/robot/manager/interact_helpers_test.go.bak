package manager

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/robot/cache"
	"github.com/yaoapp/yao/agent/robot/store"
	"github.com/yaoapp/yao/agent/robot/types"
	"github.com/yaoapp/yao/agent/testutils"
)

// mockExecutor is a minimal Executor for unit testing
type mockExecutor struct {
	resumeErr error
}

func (m *mockExecutor) ExecuteWithControl(ctx *types.Context, robot *types.Robot, trigger types.TriggerType, data interface{}, execID string, control types.ExecutionControl) (*types.Execution, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockExecutor) ExecuteWithID(ctx *types.Context, robot *types.Robot, trigger types.TriggerType, data interface{}, execID string) (*types.Execution, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockExecutor) Execute(ctx *types.Context, robot *types.Robot, trigger types.TriggerType, data interface{}) (*types.Execution, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockExecutor) Resume(ctx *types.Context, execID string, reply string) error {
	return m.resumeErr
}
func (m *mockExecutor) ExecCount() int    { return 0 }
func (m *mockExecutor) CurrentCount() int { return 0 }
func (m *mockExecutor) Reset()            {}

// HL1: createConfirmingExecution
func TestCreateConfirmingExecution(t *testing.T) {
	m := &Manager{}

	t.Run("creates record with correct fields", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Requires database")
		}
		testutils.PrepareAgent(t)
		defer testutils.Clean(t)
		ctx := types.NewContext(context.Background(), nil)
		robot := &types.Robot{MemberID: "member-hl1", TeamID: "team-hl1"}
		req := &InteractRequest{Message: "do something"}
		execStore := store.NewExecutionStore()

		record, chatID, err := m.createConfirmingExecution(ctx, robot, req, execStore)
		require.NoError(t, err)
		assert.NotEmpty(t, record.ExecutionID)
		assert.Equal(t, "member-hl1", record.MemberID)
		assert.Equal(t, "team-hl1", record.TeamID)
		assert.Equal(t, types.ExecConfirming, record.Status)
		assert.Equal(t, types.TriggerHuman, record.TriggerType)
		assert.Equal(t, types.PhaseGoals, record.Phase)
		assert.Contains(t, chatID, "robot_member-hl1_")
		assert.Equal(t, chatID, record.ChatID)
		assert.NotNil(t, record.Input)
		assert.Equal(t, types.ActionTaskAdd, record.Input.Action)
		assert.Len(t, record.Input.Messages, 1)
		assert.Equal(t, "do something", record.Input.Messages[0].Content)
		assert.NotNil(t, record.StartTime)
	})

	t.Run("UserID empty when auth is nil", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Requires database")
		}
		testutils.PrepareAgent(t)
		defer testutils.Clean(t)
		ctx := types.NewContext(context.Background(), nil)
		robot := &types.Robot{MemberID: "member-hl1b", TeamID: "team-hl1b"}
		req := &InteractRequest{Message: "test"}
		execStore := store.NewExecutionStore()

		record, _, err := m.createConfirmingExecution(ctx, robot, req, execStore)
		require.NoError(t, err)
		assert.Empty(t, record.Input.UserID)
	})
}

// HL2-HL4: adjustExecution
func TestAdjustExecution(t *testing.T) {
	m := &Manager{}

	t.Run("adjusts goals from string", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Requires database")
		}
		testutils.PrepareAgent(t)
		defer testutils.Clean(t)
		ctx := types.NewContext(context.Background(), nil)
		record := &store.ExecutionRecord{
			ExecutionID: "exec-hl2",
			MemberID:    "member-hl2",
			TriggerType: types.TriggerHuman,
			Status:      types.ExecPending,
			Phase:       types.PhaseInspiration,
		}
		execStore := store.NewExecutionStore()
		require.NoError(t, execStore.Save(ctx.Context, record))

		actionData := map[string]interface{}{"goals": "updated goals content"}
		err := m.adjustExecution(ctx, record, actionData, execStore)
		require.NoError(t, err)
		require.NotNil(t, record.Goals)
		assert.Equal(t, "updated goals content", record.Goals.Content)
	})

	t.Run("adjusts tasks from array", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Requires database")
		}
		testutils.PrepareAgent(t)
		defer testutils.Clean(t)
		ctx := types.NewContext(context.Background(), nil)
		record := &store.ExecutionRecord{
			ExecutionID: "exec-hl3",
			MemberID:    "member-hl3",
			TriggerType: types.TriggerHuman,
			Status:      types.ExecPending,
			Phase:       types.PhaseInspiration,
		}
		execStore := store.NewExecutionStore()
		require.NoError(t, execStore.Save(ctx.Context, record))

		tasks := []map[string]interface{}{
			{"id": "t1", "name": "Task 1"},
			{"id": "t2", "name": "Task 2"},
		}
		actionData := map[string]interface{}{"tasks": tasks}
		err := m.adjustExecution(ctx, record, actionData, execStore)
		require.NoError(t, err)
		assert.Len(t, record.Tasks, 2)
	})

	t.Run("nil action data is noop", func(t *testing.T) {
		ctx := types.NewContext(context.Background(), nil)
		record := &store.ExecutionRecord{}
		execStore := store.NewExecutionStore()

		err := m.adjustExecution(ctx, record, nil, execStore)
		require.NoError(t, err)
		assert.Nil(t, record.Goals)
	})

	t.Run("non-map action data handled gracefully", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Requires database")
		}
		testutils.PrepareAgent(t)
		defer testutils.Clean(t)
		ctx := types.NewContext(context.Background(), nil)
		record := &store.ExecutionRecord{
			ExecutionID: "exec-hl4",
			MemberID:    "member-hl4",
			TriggerType: types.TriggerHuman,
			Status:      types.ExecPending,
			Phase:       types.PhaseInspiration,
		}
		execStore := store.NewExecutionStore()
		require.NoError(t, execStore.Save(ctx.Context, record))

		err := m.adjustExecution(ctx, record, "not a map", execStore)
		require.NoError(t, err)
	})
}

// HL5-HL6: injectTask
func TestInjectTask(t *testing.T) {
	m := &Manager{}

	t.Run("appends new task with auto-generated ID", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Requires database")
		}
		testutils.PrepareAgent(t)
		defer testutils.Clean(t)
		ctx := types.NewContext(context.Background(), nil)
		record := &store.ExecutionRecord{
			ExecutionID: "exec-hl5",
			MemberID:    "member-hl5",
			TriggerType: types.TriggerHuman,
			Status:      types.ExecPending,
			Phase:       types.PhaseInspiration,
		}
		execStore := store.NewExecutionStore()
		require.NoError(t, execStore.Save(ctx.Context, record))

		taskData := map[string]interface{}{"name": "New Task"}
		err := m.injectTask(ctx, record, taskData, execStore)
		require.NoError(t, err)
		require.Len(t, record.Tasks, 1)
		assert.Contains(t, record.Tasks[0].ID, "injected-")
		assert.Equal(t, types.TaskPending, record.Tasks[0].Status)
	})

	t.Run("preserves existing tasks", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Requires database")
		}
		testutils.PrepareAgent(t)
		defer testutils.Clean(t)
		ctx := types.NewContext(context.Background(), nil)
		record := &store.ExecutionRecord{
			ExecutionID: "exec-hl6",
			MemberID:    "member-hl6",
			TriggerType: types.TriggerHuman,
			Status:      types.ExecPending,
			Phase:       types.PhaseInspiration,
			Tasks: []types.Task{
				{ID: "existing-1", Description: "Existing"},
			},
		}
		execStore := store.NewExecutionStore()
		require.NoError(t, execStore.Save(ctx.Context, record))

		taskData := map[string]interface{}{"name": "Added Task"}
		err := m.injectTask(ctx, record, taskData, execStore)
		require.NoError(t, err)
		assert.Len(t, record.Tasks, 2)
		assert.Equal(t, "existing-1", record.Tasks[0].ID)
	})

	t.Run("nil action data returns error", func(t *testing.T) {
		ctx := types.NewContext(context.Background(), nil)
		record := &store.ExecutionRecord{}
		execStore := store.NewExecutionStore()

		err := m.injectTask(ctx, record, nil, execStore)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "task data is required")
	})

	t.Run("respects provided task ID", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Requires database")
		}
		testutils.PrepareAgent(t)
		defer testutils.Clean(t)
		ctx := types.NewContext(context.Background(), nil)
		record := &store.ExecutionRecord{
			ExecutionID: "exec-hl6b",
			MemberID:    "member-hl6b",
			TriggerType: types.TriggerHuman,
			Status:      types.ExecPending,
			Phase:       types.PhaseInspiration,
		}
		execStore := store.NewExecutionStore()
		require.NoError(t, execStore.Save(ctx.Context, record))

		taskData := map[string]interface{}{"id": "custom-id", "name": "Custom"}
		err := m.injectTask(ctx, record, taskData, execStore)
		require.NoError(t, err)
		assert.Equal(t, "custom-id", record.Tasks[0].ID)
	})
}

// HL7: callHostAgentForScenario
func TestCallHostAgentForScenario(t *testing.T) {
	m := &Manager{}

	t.Run("no host agent returns error", func(t *testing.T) {
		ctx := types.NewContext(context.Background(), nil)
		robot := &types.Robot{MemberID: "member-hl7"}

		_, err := m.callHostAgentForScenario(ctx, robot, "assign", "test", nil, "chat-1")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no Host Agent configured")
	})

	t.Run("robot with nil config returns error", func(t *testing.T) {
		ctx := types.NewContext(context.Background(), nil)
		robot := &types.Robot{MemberID: "member-hl7b", Config: nil}

		_, err := m.callHostAgentForScenario(ctx, robot, "assign", "test", nil, "chat-1")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no Host Agent configured")
	})
}

// HL8: directAssign (needs pool — tested in processHostAction)
// HL9-HL10: directResume (needs executor — tested in processHostAction)

// Updated buildRobotStatusSnapshot tests
func TestBuildRobotStatusSnapshotV2(t *testing.T) {
	m := &Manager{}

	t.Run("nil robot returns nil", func(t *testing.T) {
		snap := m.buildRobotStatusSnapshot(nil)
		assert.Nil(t, snap)
	})

	t.Run("populates MemberID and Status", func(t *testing.T) {
		robot := &types.Robot{
			MemberID: "member-snap",
			Status:   types.RobotWorking,
		}
		snap := m.buildRobotStatusSnapshot(robot)
		require.NotNil(t, snap)
		assert.Equal(t, "member-snap", snap.MemberID)
		assert.Equal(t, types.RobotWorking, snap.Status)
	})

	t.Run("uses ActiveCount and WaitingCount", func(t *testing.T) {
		robot := &types.Robot{MemberID: "member-snap2"}
		exec1 := &types.Execution{ID: "e1", Status: types.ExecRunning}
		exec2 := &types.Execution{ID: "e2", Status: types.ExecWaiting}
		robot.AddExecution(exec1)
		robot.AddExecution(exec2)

		snap := m.buildRobotStatusSnapshot(robot)
		require.NotNil(t, snap)
		assert.Equal(t, 1, snap.ActiveCount)
		assert.Equal(t, 1, snap.WaitingCount)
	})

	t.Run("populates ActiveExecs briefs", func(t *testing.T) {
		robot := &types.Robot{MemberID: "member-snap3"}
		exec := &types.Execution{ID: "e-brief", Status: types.ExecRunning, Name: "Test Exec"}
		robot.AddExecution(exec)

		snap := m.buildRobotStatusSnapshot(robot)
		require.NotNil(t, snap)
		require.Len(t, snap.ActiveExecs, 1)
		assert.Equal(t, "e-brief", snap.ActiveExecs[0].ID)
	})

	t.Run("uses robot MaxQuota", func(t *testing.T) {
		robot := &types.Robot{
			MemberID: "member-snap4",
			Config:   &types.Config{Quota: &types.Quota{Max: 7}},
		}
		snap := m.buildRobotStatusSnapshot(robot)
		require.NotNil(t, snap)
		assert.Equal(t, 7, snap.MaxQuota)
	})
}

// Test processHostAction — adjust branch
func TestProcessHostActionAdjust(t *testing.T) {
	if testing.Short() {
		t.Skip("Requires database")
	}
	testutils.PrepareAgent(t)
	defer testutils.Clean(t)
	m := &Manager{}
	ctx := types.NewContext(context.Background(), nil)
	robot := &types.Robot{MemberID: "member-pa-adj"}

	t.Run("adjust with goals", func(t *testing.T) {
		record := &store.ExecutionRecord{
			ExecutionID: "exec-pa2",
			MemberID:    "member-pa-adj",
			TriggerType: types.TriggerHuman,
			Status:      types.ExecConfirming,
			Phase:       types.PhaseInspiration,
		}
		execStore := store.NewExecutionStore()
		require.NoError(t, execStore.Save(ctx.Context, record))

		output := &types.HostOutput{
			Reply:      "Plan adjusted",
			Action:     types.HostActionAdjust,
			ActionData: map[string]interface{}{"goals": "new goals"},
		}

		resp, err := m.processHostAction(ctx, robot, record, output, execStore)
		require.NoError(t, err)
		assert.Equal(t, "adjusted", resp.Status)
		require.NotNil(t, record.Goals)
		assert.Equal(t, "new goals", record.Goals.Content)
	})

	t.Run("adjust with tasks", func(t *testing.T) {
		record := &store.ExecutionRecord{
			ExecutionID: "exec-pa3",
			MemberID:    "member-pa-adj",
			TriggerType: types.TriggerHuman,
			Status:      types.ExecConfirming,
			Phase:       types.PhaseInspiration,
		}
		execStore := store.NewExecutionStore()
		require.NoError(t, execStore.Save(ctx.Context, record))

		tasksJSON := []map[string]interface{}{{"id": "t1", "name": "Adjusted Task"}}
		output := &types.HostOutput{
			Reply:      "Tasks updated",
			Action:     types.HostActionAdjust,
			ActionData: map[string]interface{}{"tasks": tasksJSON},
		}

		resp, err := m.processHostAction(ctx, robot, record, output, execStore)
		require.NoError(t, err)
		assert.Equal(t, "adjusted", resp.Status)
		assert.Len(t, record.Tasks, 1)
	})

	t.Run("adjust with nil data is noop", func(t *testing.T) {
		record := &store.ExecutionRecord{
			ExecutionID: "exec-pa4",
			MemberID:    "member-pa-adj",
			TriggerType: types.TriggerHuman,
			Status:      types.ExecConfirming,
			Phase:       types.PhaseInspiration,
		}
		execStore := store.NewExecutionStore()
		require.NoError(t, execStore.Save(ctx.Context, record))

		output := &types.HostOutput{
			Reply:  "No changes",
			Action: types.HostActionAdjust,
		}

		resp, err := m.processHostAction(ctx, robot, record, output, execStore)
		require.NoError(t, err)
		assert.Equal(t, "adjusted", resp.Status)
	})
}

// Test processHostAction — add_task branch
func TestProcessHostActionAddTask(t *testing.T) {
	if testing.Short() {
		t.Skip("Requires database")
	}
	testutils.PrepareAgent(t)
	defer testutils.Clean(t)
	m := &Manager{}
	ctx := types.NewContext(context.Background(), nil)
	robot := &types.Robot{MemberID: "member-pa-at"}

	t.Run("add task success", func(t *testing.T) {
		record := &store.ExecutionRecord{
			ExecutionID: "exec-pa5",
			MemberID:    "member-pa-at",
			TriggerType: types.TriggerHuman,
			Status:      types.ExecConfirming,
			Phase:       types.PhaseInspiration,
		}
		execStore := store.NewExecutionStore()
		require.NoError(t, execStore.Save(ctx.Context, record))

		output := &types.HostOutput{
			Reply:      "Task added",
			Action:     types.HostActionAddTask,
			ActionData: map[string]interface{}{"name": "New task"},
		}

		resp, err := m.processHostAction(ctx, robot, record, output, execStore)
		require.NoError(t, err)
		assert.Equal(t, "task_added", resp.Status)
		assert.Len(t, record.Tasks, 1)
	})

	t.Run("add task nil data returns error", func(t *testing.T) {
		record := &store.ExecutionRecord{
			ExecutionID: "exec-pa6",
			MemberID:    "member-pa-at",
		}
		execStore := store.NewExecutionStore()

		output := &types.HostOutput{
			Reply:  "Add task",
			Action: types.HostActionAddTask,
		}

		_, err := m.processHostAction(ctx, robot, record, output, execStore)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "task data is required")
	})
}

// Test processHostAction — skip branch
func TestProcessHostActionSkip(t *testing.T) {
	m := &Manager{}
	ctx := types.NewContext(context.Background(), nil)
	robot := &types.Robot{MemberID: "member-pa-skip"}

	t.Run("skip without waiting task returns error", func(t *testing.T) {
		record := &store.ExecutionRecord{
			ExecutionID: "exec-pa8",
			MemberID:    "member-pa-skip",
		}
		execStore := store.NewExecutionStore()

		output := &types.HostOutput{
			Reply:  "Skip it",
			Action: types.HostActionSkip,
		}

		_, err := m.processHostAction(ctx, robot, record, output, execStore)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no task is waiting")
	})
}

// Test processHostAction — wait_for_more and default
func TestProcessHostActionWaitForMoreAndDefault(t *testing.T) {
	m := &Manager{}
	ctx := types.NewContext(context.Background(), nil)
	robot := &types.Robot{MemberID: "member-pa-wfm"}

	t.Run("wait_for_more", func(t *testing.T) {
		record := &store.ExecutionRecord{}
		execStore := store.NewExecutionStore()

		output := &types.HostOutput{
			Reply:       "More details please",
			WaitForMore: true,
		}

		resp, err := m.processHostAction(ctx, robot, record, output, execStore)
		require.NoError(t, err)
		assert.Equal(t, "waiting_for_more", resp.Status)
		assert.Equal(t, "More details please", resp.Reply)
		assert.True(t, resp.WaitForMore)
	})

	t.Run("unknown action returns acknowledged", func(t *testing.T) {
		record := &store.ExecutionRecord{}
		execStore := store.NewExecutionStore()

		output := &types.HostOutput{
			Reply:  "OK",
			Action: "unknown_action",
		}

		resp, err := m.processHostAction(ctx, robot, record, output, execStore)
		require.NoError(t, err)
		assert.Equal(t, "acknowledged", resp.Status)
		assert.Equal(t, "OK", resp.Message)
	})
}

// Test processHostAction — cancel branch
func TestProcessHostActionCancel(t *testing.T) {
	if testing.Short() {
		t.Skip("Requires database")
	}
	testutils.PrepareAgent(t)
	defer testutils.Clean(t)

	t.Run("cancel waiting execution", func(t *testing.T) {
		// Cannot fully test without a started manager; verify the error path
		m := &Manager{started: false}
		ctx := types.NewContext(context.Background(), nil)
		robot := &types.Robot{MemberID: "member-pa-cancel"}

		record := &store.ExecutionRecord{
			ExecutionID: "exec-pa11",
			MemberID:    "member-pa-cancel",
		}
		execStore := store.NewExecutionStore()

		output := &types.HostOutput{
			Reply:  "Cancel it",
			Action: types.HostActionCancel,
		}

		_, err := m.processHostAction(ctx, robot, record, output, execStore)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "manager not started")
	})
}

// Test HandleInteract validation
func TestHandleInteractValidationExtended(t *testing.T) {
	t.Run("manager not started returns error", func(t *testing.T) {
		m := &Manager{started: false}
		_, err := m.HandleInteract(types.NewContext(context.Background(), nil), "member-1", &InteractRequest{Message: "test"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "manager not started")
	})

	t.Run("empty member_id returns error", func(t *testing.T) {
		m := &Manager{started: true}
		_, err := m.HandleInteract(types.NewContext(context.Background(), nil), "", &InteractRequest{Message: "test"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "member_id is required")
	})

	t.Run("nil request returns error", func(t *testing.T) {
		m := &Manager{started: true}
		_, err := m.HandleInteract(types.NewContext(context.Background(), nil), "member-1", nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "message is required")
	})

	t.Run("empty message returns error", func(t *testing.T) {
		m := &Manager{started: true}
		_, err := m.HandleInteract(types.NewContext(context.Background(), nil), "member-1", &InteractRequest{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "message is required")
	})

	t.Run("non-interactable status returns error", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Requires database and cache")
		}
		testutils.PrepareAgent(t)
		defer testutils.Clean(t)
		// Would require a full Manager with cache — tested via E2E
	})
}

// Test CancelExecution validation
func TestCancelExecutionValidationExtended(t *testing.T) {
	t.Run("manager not started", func(t *testing.T) {
		m := &Manager{started: false}
		err := m.CancelExecution(types.NewContext(context.Background(), nil), "exec-1")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "manager not started")
	})
}

// Test buildHostContext JSON output
func TestBuildHostContextJSON(t *testing.T) {
	m := &Manager{}

	robot := &types.Robot{MemberID: "member-ctx"}
	record := &store.ExecutionRecord{
		Goals:           &types.Goals{Content: "test goals"},
		Tasks:           []types.Task{{ID: "t1"}},
		WaitingQuestion: "What time?",
	}
	waitingTask := &types.Task{ID: "t1", Status: types.TaskWaitingInput}

	hostCtx := m.buildHostContext(robot, record, waitingTask)
	require.NotNil(t, hostCtx)

	data, err := json.Marshal(hostCtx)
	require.NoError(t, err)

	var parsed map[string]interface{}
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	// Goals is a struct, not a plain string
	goalsRaw, ok := parsed["goals"]
	require.True(t, ok)
	goalsMap, ok := goalsRaw.(map[string]interface{})
	require.True(t, ok, "Goals should be a JSON object, not a string")
	assert.Equal(t, "test goals", goalsMap["content"])

	assert.Equal(t, "What time?", parsed["agent_reply"])
}

// ==================== processHostAction -- confirm branch (PA1) ====================

func TestProcessHostActionConfirmRequiresPool(t *testing.T) {
	if testing.Short() {
		t.Skip("Requires database")
	}
	testutils.PrepareAgent(t)
	defer testutils.Clean(t)
	m := &Manager{started: false}
	ctx := types.NewContext(context.Background(), nil)
	robot := &types.Robot{MemberID: "member-pa1"}
	record := &store.ExecutionRecord{
		ExecutionID: "exec-pa1",
		MemberID:    "member-pa1",
		Status:      types.ExecConfirming,
	}
	execStore := store.NewExecutionStore()

	output := &types.HostOutput{
		Reply:  "Confirmed",
		Action: types.HostActionConfirm,
	}

	assert.Panics(t, func() {
		m.processHostAction(ctx, robot, record, output, execStore)
	}, "should panic because pool/executor are nil")
}

// ==================== processHostAction -- inject_ctx branch (PA9-PA10) ====================

func TestProcessHostActionInjectCtx(t *testing.T) {
	t.Run("nil executor panics", func(t *testing.T) {
		m := &Manager{}
		ctx := types.NewContext(context.Background(), nil)
		robot := &types.Robot{MemberID: "member-pa9"}
		record := &store.ExecutionRecord{
			ExecutionID: "exec-pa9",
			MemberID:    "member-pa9",
			Status:      types.ExecWaiting,
		}
		execStore := store.NewExecutionStore()

		output := &types.HostOutput{
			Reply:      "Here's context",
			Action:     types.HostActionInjectCtx,
			ActionData: "additional context data",
		}

		assert.Panics(t, func() {
			m.processHostAction(ctx, robot, record, output, execStore)
		})
	})

	t.Run("with mock executor delegates resume", func(t *testing.T) {
		mockExec := &mockExecutor{resumeErr: fmt.Errorf("mock error")}
		m := &Manager{executor: mockExec}
		ctx := types.NewContext(context.Background(), nil)
		robot := &types.Robot{MemberID: "member-pa10"}
		record := &store.ExecutionRecord{
			ExecutionID: "exec-pa10",
			MemberID:    "member-pa10",
		}
		execStore := store.NewExecutionStore()

		output := &types.HostOutput{
			Reply:      "Resume with data",
			Action:     types.HostActionInjectCtx,
			ActionData: map[string]interface{}{"reply": "detailed info"},
		}

		_, err := m.processHostAction(ctx, robot, record, output, execStore)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "mock error")
	})

	t.Run("ErrExecutionSuspended returns waiting status", func(t *testing.T) {
		mockExec := &mockExecutor{resumeErr: types.ErrExecutionSuspended}
		m := &Manager{executor: mockExec}
		ctx := types.NewContext(context.Background(), nil)
		robot := &types.Robot{MemberID: "member-pa10b"}
		record := &store.ExecutionRecord{
			ExecutionID: "exec-pa10b",
			MemberID:    "member-pa10b",
		}
		execStore := store.NewExecutionStore()

		output := &types.HostOutput{
			Reply:      "Resume",
			Action:     types.HostActionInjectCtx,
			ActionData: "context",
		}

		resp, err := m.processHostAction(ctx, robot, record, output, execStore)
		require.NoError(t, err)
		assert.Equal(t, "waiting", resp.Status)
	})
}

// ==================== HandleInteract routing (HI5-HI8) ====================

func TestHandleInteractRouting(t *testing.T) {
	t.Run("HI5: non-existent execution_id returns error", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Requires database and cache")
		}
		testutils.PrepareAgent(t)
		defer testutils.Clean(t)

		m := &Manager{started: true, cache: cache.New()}
		ctx := types.NewContext(context.Background(), nil)
		_, err := m.HandleInteract(ctx, "member-hi5", &InteractRequest{
			ExecutionID: "nonexistent-exec",
			Message:     "test",
		})
		assert.Error(t, err)
	})

	t.Run("HI6: non-existent robot returns error", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Requires database and cache")
		}
		testutils.PrepareAgent(t)
		defer testutils.Clean(t)
		m := &Manager{started: true, cache: cache.New()}
		ctx := types.NewContext(context.Background(), nil)
		_, err := m.HandleInteract(ctx, "nonexistent-robot", &InteractRequest{
			Message: "test",
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "robot not found")
	})
}

// ==================== CancelExecution validation (CE2-CE5) ====================

func TestCancelExecutionStatusValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("Requires database")
	}
	testutils.PrepareAgent(t)
	defer testutils.Clean(t)

	t.Run("CE2: non-existent execution returns error", func(t *testing.T) {
		m := &Manager{started: true}
		ctx := types.NewContext(context.Background(), nil)
		err := m.CancelExecution(ctx, "nonexistent-exec")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "execution not found")
	})

	t.Run("CE3: running execution cannot be cancelled", func(t *testing.T) {
		m := &Manager{started: true}
		ctx := types.NewContext(context.Background(), nil)

		execStore := store.NewExecutionStore()
		record := &store.ExecutionRecord{
			ExecutionID: "exec-ce3",
			MemberID:    "member-ce3",
			Status:      types.ExecRunning,
			TriggerType: types.TriggerHuman,
			Phase:       types.PhaseInspiration,
		}
		require.NoError(t, execStore.Save(ctx.Context, record))

		err := m.CancelExecution(ctx, "exec-ce3")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "only waiting/confirming can be cancelled")
	})

	t.Run("CE4: completed execution cannot be cancelled", func(t *testing.T) {
		m := &Manager{started: true}
		ctx := types.NewContext(context.Background(), nil)

		execStore := store.NewExecutionStore()
		record := &store.ExecutionRecord{
			ExecutionID: "exec-ce4",
			MemberID:    "member-ce4",
			Status:      types.ExecCompleted,
			TriggerType: types.TriggerHuman,
			Phase:       types.PhaseInspiration,
		}
		require.NoError(t, execStore.Save(ctx.Context, record))

		err := m.CancelExecution(ctx, "exec-ce4")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "only waiting/confirming can be cancelled")
	})
}

// ==================== InteractRequest/InteractResponse struct validation ====================

func TestInteractRequestStructFields(t *testing.T) {
	req := &InteractRequest{
		ExecutionID: "exec-1",
		TaskID:      "task-1",
		Source:      types.InteractSourceUI,
		Message:     "do something",
		Action:      "confirm",
	}
	assert.Equal(t, "exec-1", req.ExecutionID)
	assert.Equal(t, "task-1", req.TaskID)
	assert.Equal(t, types.InteractSourceUI, req.Source)
	assert.Equal(t, "do something", req.Message)
	assert.Equal(t, "confirm", req.Action)
}

func TestInteractResponseStructFields(t *testing.T) {
	resp := &InteractResponse{
		ExecutionID: "exec-1",
		Status:      "confirmed",
		Message:     "Done",
		ChatID:      "chat-1",
		Reply:       "I'll do it",
		WaitForMore: true,
	}
	assert.Equal(t, "exec-1", resp.ExecutionID)
	assert.Equal(t, "confirmed", resp.Status)
	assert.Equal(t, "Done", resp.Message)
	assert.Equal(t, "chat-1", resp.ChatID)
	assert.Equal(t, "I'll do it", resp.Reply)
	assert.True(t, resp.WaitForMore)
}

// ==================== executeResume helper ====================

func TestExecuteResumeNilExecutor(t *testing.T) {
	m := &Manager{}
	ctx := types.NewContext(context.Background(), nil)

	assert.Panics(t, func() {
		_ = m.executeResume(ctx, "exec-test", "reply")
	})
}

func TestExecuteResumeWithMock(t *testing.T) {
	t.Run("delegates to executor Resume", func(t *testing.T) {
		mockExec := &mockExecutor{resumeErr: nil}
		m := &Manager{executor: mockExec}
		ctx := types.NewContext(context.Background(), nil)

		err := m.executeResume(ctx, "exec-test", "reply")
		assert.NoError(t, err)
	})

	t.Run("propagates error", func(t *testing.T) {
		mockExec := &mockExecutor{resumeErr: fmt.Errorf("resume failed")}
		m := &Manager{executor: mockExec}
		ctx := types.NewContext(context.Background(), nil)

		err := m.executeResume(ctx, "exec-test", "reply")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "resume failed")
	})

	t.Run("propagates ErrExecutionSuspended", func(t *testing.T) {
		mockExec := &mockExecutor{resumeErr: types.ErrExecutionSuspended}
		m := &Manager{executor: mockExec}
		ctx := types.NewContext(context.Background(), nil)

		err := m.executeResume(ctx, "exec-test", "reply")
		assert.Equal(t, types.ErrExecutionSuspended, err)
	})
}

// ==================== skipWaitingTask and directResume with mock ====================

func TestSkipWaitingTaskWithMock(t *testing.T) {
	t.Run("no waiting task returns error", func(t *testing.T) {
		mockExec := &mockExecutor{}
		m := &Manager{executor: mockExec}
		ctx := types.NewContext(context.Background(), nil)
		record := &store.ExecutionRecord{
			ExecutionID: "exec-skip",
		}
		execStore := store.NewExecutionStore()

		err := m.skipWaitingTask(ctx, record, execStore)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no task is waiting")
	})

	t.Run("marks waiting task as skipped and resumes", func(t *testing.T) {
		mockExec := &mockExecutor{resumeErr: nil}
		m := &Manager{executor: mockExec}
		ctx := types.NewContext(context.Background(), nil)
		record := &store.ExecutionRecord{
			ExecutionID:   "exec-skip2",
			WaitingTaskID: "task-w",
			Tasks: []types.Task{
				{ID: "task-w", Status: types.TaskWaitingInput},
			},
		}
		execStore := store.NewExecutionStore()

		err := m.skipWaitingTask(ctx, record, execStore)
		assert.NoError(t, err)
		assert.Equal(t, types.TaskSkipped, record.Tasks[0].Status)
	})
}

func TestDirectResumeWithMock(t *testing.T) {
	t.Run("successful resume", func(t *testing.T) {
		mockExec := &mockExecutor{resumeErr: nil}
		m := &Manager{executor: mockExec}
		ctx := types.NewContext(context.Background(), nil)
		record := &store.ExecutionRecord{
			ExecutionID: "exec-dr",
			ChatID:      "chat-dr",
		}
		req := &InteractRequest{Message: "continue"}

		resp, err := m.directResume(ctx, record, req)
		require.NoError(t, err)
		assert.Equal(t, "resumed", resp.Status)
		assert.Equal(t, "exec-dr", resp.ExecutionID)
		assert.Equal(t, "chat-dr", resp.ChatID)
	})

	t.Run("suspended again", func(t *testing.T) {
		mockExec := &mockExecutor{resumeErr: types.ErrExecutionSuspended}
		m := &Manager{executor: mockExec}
		ctx := types.NewContext(context.Background(), nil)
		record := &store.ExecutionRecord{
			ExecutionID: "exec-dr2",
			ChatID:      "chat-dr2",
		}
		req := &InteractRequest{Message: "continue"}

		resp, err := m.directResume(ctx, record, req)
		require.NoError(t, err)
		assert.Equal(t, "waiting", resp.Status)
	})

	t.Run("error propagated", func(t *testing.T) {
		mockExec := &mockExecutor{resumeErr: fmt.Errorf("resume failed")}
		m := &Manager{executor: mockExec}
		ctx := types.NewContext(context.Background(), nil)
		record := &store.ExecutionRecord{
			ExecutionID: "exec-dr3",
		}
		req := &InteractRequest{Message: "continue"}

		_, err := m.directResume(ctx, record, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "resume failed")
	})
}
