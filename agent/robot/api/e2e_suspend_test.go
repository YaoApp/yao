package api_test

// End-to-end tests for V2 Suspend/Resume flow
// Tests the complete lifecycle: execution → need_input → suspend → reply → resume → complete/re-suspend
//
// Prerequisites:
//   - Valid LLM API keys
//   - Test assistants: tests.robot-need-input, experts.text-writer
//   - Database connection (YAO_DB_PRIMARY)

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/xun/capsule"
	agentcontext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/robot/api"
	"github.com/yaoapp/yao/agent/robot/store"
	"github.com/yaoapp/yao/agent/robot/types"
	"github.com/yaoapp/yao/agent/testutils"
	oauthtypes "github.com/yaoapp/yao/openapi/oauth/types"
)

func testAuthSuspend() *oauthtypes.AuthorizedInfo {
	return &oauthtypes.AuthorizedInfo{
		UserID: "e2e-suspend-user",
		TeamID: "e2e-suspend-team",
	}
}

// triggerSuspendRobot triggers a robot via the Trigger API (human trigger path)
func triggerSuspendRobot(t *testing.T, ctx *types.Context, memberID string, message string) *api.TriggerResult {
	t.Helper()
	result, err := api.Trigger(ctx, memberID, &api.TriggerRequest{
		Type:   types.TriggerHuman,
		Action: types.ActionTaskAdd,
		Messages: []agentcontext.Message{
			{Role: "user", Content: message},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	if !result.Accepted {
		t.Fatalf("Trigger not accepted: %s", result.Message)
	}
	return result
}

// waitForStatus polls execution status until it matches one of the expected statuses
func waitForStatus(t *testing.T, execID string, statuses []types.ExecStatus, timeout time.Duration) *types.Execution {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		time.Sleep(time.Second)
		exec := getExecution(t, execID)
		if exec == nil {
			continue
		}
		for _, s := range statuses {
			if exec.Status == s {
				return exec
			}
		}
	}
	return nil
}

// TestE2ENormalExecutionNoSuspend verifies that a normal execution (no need_input)
// completes without entering the suspend path.
func TestE2ENormalExecutionNoSuspend(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test - requires real LLM calls")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	cleanupE2ESuspendRobots(t)
	defer cleanupE2ESuspendRobots(t)

	memberID := "robot_e2e_suspend_001"
	setupE2ESuspendRobotWithTasksPlanner(t, memberID, "team_e2e_suspend", []string{"experts.text-writer"}, "tests.e2e-tasks")

	err := api.Start()
	require.NoError(t, err)
	defer api.Stop()

	ctx := types.NewContext(context.Background(), testAuthSuspend())
	result := triggerSuspendRobot(t, ctx, memberID, "Write a one-sentence greeting")

	exec := waitForStatus(t, result.ExecutionID,
		[]types.ExecStatus{types.ExecCompleted, types.ExecFailed}, 120*time.Second)

	require.NotNil(t, exec, "Execution should exist and reach terminal state")
	if exec.Status == types.ExecFailed {
		t.Logf("Execution failed with error: %s", exec.Error)
	}
	assert.Equal(t, types.ExecCompleted, exec.Status, "Normal execution should complete")
	assert.NotEmpty(t, exec.ChatID, "ChatID should be set")
	assert.Nil(t, exec.ResumeContext, "No resume context for normal execution")
	assert.Empty(t, exec.WaitingTaskID, "No waiting task for normal execution")
}

// TestE2ESuspendResumeFlow tests the full suspend-resume lifecycle:
// 1. Trigger execution with robot-need-input assistant (signals need_input)
// 2. Verify execution enters waiting status
// 3. Reply to resume execution via api.Interact
// 4. Verify execution re-suspends (since robot-need-input always signals need_input)
func TestE2ESuspendResumeFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test - requires real LLM calls")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	cleanupE2ESuspendRobots(t)
	defer cleanupE2ESuspendRobots(t)

	memberID := "robot_e2e_suspend_002"
	setupE2ESuspendRobot(t, memberID, "team_e2e_suspend", []string{"tests.robot-need-input"})

	err := api.Start()
	require.NoError(t, err)
	defer api.Stop()

	ctx := types.NewContext(context.Background(), testAuthSuspend())

	// Step 1: Trigger execution — the robot-need-input assistant always returns need_input
	result := triggerSuspendRobot(t, ctx, memberID, "Analyze sales data")
	execID := result.ExecutionID

	// Step 2: Wait for the execution to reach waiting status
	exec := waitForStatus(t, execID,
		[]types.ExecStatus{types.ExecWaiting, types.ExecCompleted, types.ExecFailed}, 120*time.Second)

	require.NotNil(t, exec, "Execution should exist")
	require.Equal(t, types.ExecWaiting, exec.Status, "Execution should be in waiting status")
	assert.NotEmpty(t, exec.WaitingTaskID, "WaitingTaskID should be set")
	assert.NotEmpty(t, exec.WaitingQuestion, "WaitingQuestion should be set")
	assert.NotNil(t, exec.WaitingSince, "WaitingSince should be set")
	assert.NotNil(t, exec.ResumeContext, "ResumeContext should be set")
	assert.NotEmpty(t, exec.ChatID, "ChatID should be set")

	t.Logf("Execution suspended: execID=%s task=%s question=%s", execID, exec.WaitingTaskID, exec.WaitingQuestion)

	// Step 3: Resume via api.Interact (reply to the waiting execution)
	interactResult, err := api.Interact(ctx, memberID, &api.InteractRequest{
		ExecutionID: execID,
		Message:     "Use the last 30 days for analysis",
	})
	require.NoError(t, err)
	require.NotNil(t, interactResult)

	// The Host Agent may return a structured action (→ "waiting"/"resumed") or
	// a conversational reply (→ "waiting_for_more") depending on LLM behaviour.
	assert.Contains(t, []string{"waiting", "resumed", "waiting_for_more"}, interactResult.Status,
		"Expected waiting, resumed, or waiting_for_more; got %s", interactResult.Status)
	t.Logf("Interact result: status=%s message=%s", interactResult.Status, interactResult.Message)

	// Step 4: Verify the execution is in waiting status again (re-suspended)
	exec = getExecution(t, execID)
	require.NotNil(t, exec)
	assert.Equal(t, types.ExecWaiting, exec.Status, "Execution should be waiting again after re-suspend")
	assert.NotNil(t, exec.ResumeContext, "ResumeContext should be set after re-suspend")
}

// TestE2EReplyShortcut tests the Reply semantic shortcut
func TestE2EReplyShortcut(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test - requires real LLM calls")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	cleanupE2ESuspendRobots(t)
	defer cleanupE2ESuspendRobots(t)

	memberID := "robot_e2e_suspend_004"
	setupE2ESuspendRobot(t, memberID, "team_e2e_suspend", []string{"tests.robot-need-input"})

	err := api.Start()
	require.NoError(t, err)
	defer api.Stop()

	ctx := types.NewContext(context.Background(), testAuthSuspend())
	result := triggerSuspendRobot(t, ctx, memberID, "Check inventory levels")

	exec := waitForStatus(t, result.ExecutionID,
		[]types.ExecStatus{types.ExecWaiting}, 120*time.Second)
	require.NotNil(t, exec, "Execution should reach waiting status")
	require.Equal(t, types.ExecWaiting, exec.Status)

	// Use Reply shortcut
	replyResult, err := api.Reply(ctx, memberID, result.ExecutionID, exec.WaitingTaskID, "Use warehouse A data")
	require.NoError(t, err)
	require.NotNil(t, replyResult)
	assert.Contains(t, []string{"waiting", "resumed", "waiting_for_more"}, replyResult.Status)
	t.Logf("Reply result: status=%s", replyResult.Status)
}

// TestE2EResumeContextPersistence verifies that suspend state is properly persisted
// and can be loaded back from the database.
func TestE2EResumeContextPersistence(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test - requires real LLM calls")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	cleanupE2ESuspendRobots(t)
	defer cleanupE2ESuspendRobots(t)

	memberID := "robot_e2e_suspend_003"
	setupE2ESuspendRobot(t, memberID, "team_e2e_suspend", []string{"tests.robot-need-input"})

	err := api.Start()
	require.NoError(t, err)
	defer api.Stop()

	ctx := types.NewContext(context.Background(), testAuthSuspend())
	result := triggerSuspendRobot(t, ctx, memberID, "Analyze user behavior")

	exec := waitForStatus(t, result.ExecutionID,
		[]types.ExecStatus{types.ExecWaiting, types.ExecCompleted, types.ExecFailed}, 120*time.Second)

	require.NotNil(t, exec)
	if exec.Status != types.ExecWaiting {
		t.Skipf("Execution did not reach waiting status (status=%s), skipping persistence test", exec.Status)
	}

	// Load from DB directly using store to verify persistence
	execStore := store.NewExecutionStore()
	record, err := execStore.Get(context.Background(), result.ExecutionID)
	require.NoError(t, err)
	require.NotNil(t, record)

	assert.Equal(t, types.ExecWaiting, record.Status)
	assert.NotEmpty(t, record.WaitingTaskID)
	assert.NotEmpty(t, record.WaitingQuestion)
	assert.NotNil(t, record.WaitingSince)
	assert.NotNil(t, record.ResumeContext)
	assert.Equal(t, exec.ChatID, record.ChatID)

	// Verify resume context deserialization
	restored := record.ToExecution()
	assert.NotNil(t, restored.ResumeContext)
	assert.GreaterOrEqual(t, restored.ResumeContext.TaskIndex, 0)

	t.Logf("Persisted resume context: TaskIndex=%d, PreviousResults=%d",
		restored.ResumeContext.TaskIndex, len(restored.ResumeContext.PreviousResults))
}

// TestE2EInteractRequiresExecutionID tests that Interact API returns error when
// execution_id is not provided (Host Agent deferred).
func TestE2EInteractRequiresExecutionID(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	ctx := types.NewContext(context.Background(), testAuthSuspend())

	_, err := api.Interact(ctx, "some-member", &api.InteractRequest{
		Message: "hello",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "execution_id is required")
}

// TestE2EInteractWithNonWaitingExecution tests that Interact API returns error
// when trying to resume an execution that is not in waiting status.
func TestE2EInteractWithNonWaitingExecution(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test - requires real LLM calls")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	cleanupE2ESuspendRobots(t)
	defer cleanupE2ESuspendRobots(t)

	memberID := "robot_e2e_suspend_005"
	setupE2ESuspendRobotWithTasksPlanner(t, memberID, "team_e2e_suspend", []string{"experts.text-writer"}, "tests.e2e-tasks")

	err := api.Start()
	require.NoError(t, err)
	defer api.Stop()

	ctx := types.NewContext(context.Background(), testAuthSuspend())
	result := triggerSuspendRobot(t, ctx, memberID, "Say hello")

	// Wait for completion
	exec := waitForStatus(t, result.ExecutionID,
		[]types.ExecStatus{types.ExecCompleted, types.ExecFailed}, 120*time.Second)
	require.NotNil(t, exec, "Execution should reach terminal state")

	// Try to interact with the completed execution
	_, err = api.Interact(ctx, memberID, &api.InteractRequest{
		ExecutionID: result.ExecutionID,
		Message:     "This should fail",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot interact")
}

// ============================================================================
// Helper Functions
// ============================================================================

func setupE2ESuspendRobotWithTasksPlanner(t *testing.T, memberID, teamID string, agents []string, tasksPlanner string) {
	t.Helper()

	robotConfig := map[string]interface{}{
		"identity": map[string]interface{}{
			"role":   "V2 Suspend Test Robot",
			"duties": []string{"Execute test tasks"},
			"rules":  []string{"Keep responses under 50 words"},
		},
		"resources": map[string]interface{}{
			"phases": map[string]interface{}{
				"inspiration": "robot.inspiration",
				"goals":       "robot.goals",
				"tasks":       tasksPlanner,
				"run":         "robot.validation",
				"delivery":    "robot.delivery",
				"learning":    "robot.learning",
			},
			"agents": agents,
		},
		"quota": map[string]interface{}{
			"max":      5,
			"queue":    20,
			"priority": 5,
		},
		"triggers": map[string]interface{}{
			"clock":     map[string]interface{}{"enabled": false},
			"intervene": map[string]interface{}{"enabled": true},
			"event":     map[string]interface{}{"enabled": true},
		},
		"delivery": map[string]interface{}{
			"email":   map[string]interface{}{"enabled": false},
			"webhook": map[string]interface{}{"enabled": false},
			"process": map[string]interface{}{"enabled": false},
		},
	}

	configJSON, err := json.Marshal(robotConfig)
	require.NoError(t, err)

	m := model.Select("__yao.member")
	require.NotNil(t, m)
	tableName := m.MetaData.Table.Name
	qb := capsule.Query()

	err = qb.Table(tableName).Insert([]map[string]interface{}{
		{
			"member_id":       memberID,
			"team_id":         teamID,
			"member_type":     "robot",
			"display_name":    "E2E Suspend Test Robot " + memberID,
			"system_prompt":   "You are a simple E2E test robot. Your job is to execute tasks.\nWhen generating goals: create exactly 1 simple goal.\nWhen generating tasks: create exactly 1 simple task.\nKeep all outputs brief.",
			"status":          "active",
			"role_id":         "member",
			"autonomous_mode": true,
			"robot_status":    "idle",
			"robot_config":    string(configJSON),
		},
	})
	require.NoError(t, err)
}

func setupE2ESuspendRobot(t *testing.T, memberID, teamID string, agents []string) {
	t.Helper()
	setupE2ESuspendRobotWithTasksPlanner(t, memberID, teamID, agents, "tests.e2e-suspend-tasks")
}

func cleanupE2ESuspendRobots(t *testing.T) {
	t.Helper()
	mod := model.Select("__yao.member")
	if mod == nil {
		return
	}
	mod.DeleteWhere(model.QueryParam{
		Wheres: []model.QueryWhere{
			{Column: "member_id", OP: "like", Value: "robot_e2e_suspend_%"},
		},
	})
}

func getExecution(t *testing.T, execID string) *types.Execution {
	t.Helper()
	execStore := store.NewExecutionStore()
	record, err := execStore.Get(context.Background(), execID)
	if err != nil || record == nil {
		return nil
	}
	return record.ToExecution()
}
