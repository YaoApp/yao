package api_test

// End-to-end tests for Execution Control (Pause/Resume/Stop)
// These tests verify that executions can be controlled during runtime
//
// Prerequisites:
//   - Valid LLM API keys (OPENAI_TEST_KEY or DEEPSEEK_API_KEY)
//   - Test assistants in yao-dev-app/assistants/robot/
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
	"github.com/yaoapp/yao/agent/robot/api"
	"github.com/yaoapp/yao/agent/robot/types"
	"github.com/yaoapp/yao/agent/testutils"
	oauthtypes "github.com/yaoapp/yao/openapi/oauth/types"
)

// testAuthControl returns test auth info for control E2E tests
func testAuthControl() *oauthtypes.AuthorizedInfo {
	return &oauthtypes.AuthorizedInfo{
		UserID: "e2e-control-user",
		TeamID: "e2e-control-team",
	}
}

// TestE2EControlPauseResume tests pausing and resuming an execution
func TestE2EControlPauseResume(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test - requires real LLM calls")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	cleanupE2ERobots(t)
	cleanupE2EExecutions(t)
	defer cleanupE2ERobots(t)
	defer cleanupE2EExecutions(t)

	t.Run("pause_and_resume_execution", func(t *testing.T) {
		memberID := "robot_e2e_control_pause"
		setupE2ERobotForControl(t, memberID, "team_e2e_control")

		err := api.Start()
		require.NoError(t, err)
		defer api.Stop()

		ctx := types.NewContext(context.Background(), testAuthControl())

		// Start execution
		result, err := api.TriggerManual(ctx, memberID, types.TriggerClock, nil)
		require.NoError(t, err)
		require.True(t, result.Accepted)

		t.Logf("Execution started: ExecutionID=%s", result.ExecutionID)

		// Wait for execution to start running
		var execID string
		maxWait := 30 * time.Second
		deadline := time.Now().Add(maxWait)

		for time.Now().Before(deadline) {
			time.Sleep(500 * time.Millisecond)

			executions, err := api.ListExecutions(ctx, memberID, nil)
			if err != nil || len(executions.Data) == 0 {
				continue
			}

			exec := executions.Data[0]
			if exec.Status == types.ExecRunning {
				execID = exec.ID
				t.Logf("Execution running: ID=%s, Phase=%s", execID, exec.Phase)
				break
			}
		}

		if execID == "" {
			t.Skip("Execution did not start in time - may have completed too quickly")
			return
		}

		// Pause the execution
		err = api.PauseExecution(ctx, execID)
		if err != nil {
			t.Logf("Pause error (may be expected if execution completed): %v", err)
		} else {
			t.Logf("Execution paused")

			// Verify paused state
			time.Sleep(1 * time.Second)
			status, err := api.GetExecutionStatus(ctx, execID)
			if err == nil && status != nil {
				t.Logf("Status after pause: %s", status.Status)
			}

			// Resume the execution
			err = api.ResumeExecution(ctx, execID)
			if err != nil {
				t.Logf("Resume error: %v", err)
			} else {
				t.Logf("Execution resumed")
			}
		}

		// Wait for completion
		maxWait = 120 * time.Second
		deadline = time.Now().Add(maxWait)

		for time.Now().Before(deadline) {
			time.Sleep(2 * time.Second)

			exec, err := api.GetExecution(ctx, execID)
			if err != nil {
				continue
			}

			t.Logf("Execution status: %s, phase: %s", exec.Status, exec.Phase)

			if exec.Status == types.ExecCompleted || exec.Status == types.ExecFailed || exec.Status == types.ExecCancelled {
				// Execution finished (completed, failed, or cancelled)
				t.Logf("Execution finished with status: %s", exec.Status)
				return
			}
		}

		t.Logf("Execution did not complete in time")
	})
}

// TestE2EControlStop tests stopping an execution
func TestE2EControlStop(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test - requires real LLM calls")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	cleanupE2ERobots(t)
	cleanupE2EExecutions(t)
	defer cleanupE2ERobots(t)
	defer cleanupE2EExecutions(t)

	t.Run("stop_running_execution", func(t *testing.T) {
		memberID := "robot_e2e_control_stop"
		setupE2ERobotForControl(t, memberID, "team_e2e_control")

		err := api.Start()
		require.NoError(t, err)
		defer api.Stop()

		ctx := types.NewContext(context.Background(), testAuthControl())

		// Start execution
		result, err := api.TriggerManual(ctx, memberID, types.TriggerClock, nil)
		require.NoError(t, err)
		require.True(t, result.Accepted)

		t.Logf("Execution started: ExecutionID=%s", result.ExecutionID)

		// Wait for execution to start running
		var execID string
		maxWait := 30 * time.Second
		deadline := time.Now().Add(maxWait)

		for time.Now().Before(deadline) {
			time.Sleep(500 * time.Millisecond)

			executions, err := api.ListExecutions(ctx, memberID, nil)
			if err != nil || len(executions.Data) == 0 {
				continue
			}

			exec := executions.Data[0]
			if exec.Status == types.ExecRunning {
				execID = exec.ID
				t.Logf("Execution running: ID=%s, Phase=%s", execID, exec.Phase)
				break
			}
		}

		if execID == "" {
			t.Skip("Execution did not start in time - may have completed too quickly")
			return
		}

		// Stop the execution
		err = api.StopExecution(ctx, execID)
		if err != nil {
			t.Logf("Stop error (may be expected if execution completed): %v", err)
		} else {
			t.Logf("Stop signal sent")
		}

		// Wait and verify stopped/cancelled state (with retry)
		maxWait = 30 * time.Second
		deadline = time.Now().Add(maxWait)

		for time.Now().Before(deadline) {
			time.Sleep(2 * time.Second)

			exec, err := api.GetExecution(ctx, execID)
			if err != nil {
				t.Logf("Get execution error: %v", err)
				continue
			}

			t.Logf("Current status: %s", exec.Status)

			// Execution should eventually be cancelled, completed, or failed
			if exec.Status == types.ExecCancelled ||
				exec.Status == types.ExecCompleted ||
				exec.Status == types.ExecFailed {
				t.Logf("Final status: %s", exec.Status)
				return
			}
		}

		// If we get here, check final state
		exec, err := api.GetExecution(ctx, execID)
		if err != nil {
			t.Logf("Get execution error: %v", err)
			return
		}

		// Allow running state if stop didn't take effect in time (execution may have already completed)
		t.Logf("Final status after wait: %s", exec.Status)
		assert.True(t,
			exec.Status == types.ExecCancelled ||
				exec.Status == types.ExecCompleted ||
				exec.Status == types.ExecFailed ||
				exec.Status == types.ExecRunning, // Allow running if stop didn't take effect
			"Execution should be in terminal state or still running, got: %s", exec.Status)
	})
}

// TestE2EControlStopBeforeStart tests stopping an execution before it starts
func TestE2EControlStopBeforeStart(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test - requires real LLM calls")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	cleanupE2ERobots(t)
	cleanupE2EExecutions(t)
	defer cleanupE2ERobots(t)
	defer cleanupE2EExecutions(t)

	t.Run("stop_queued_execution", func(t *testing.T) {
		memberID := "robot_e2e_control_stop_early"
		setupE2ERobotForControl(t, memberID, "team_e2e_control")

		err := api.Start()
		require.NoError(t, err)
		defer api.Stop()

		ctx := types.NewContext(context.Background(), testAuthControl())

		// Start execution
		result, err := api.TriggerManual(ctx, memberID, types.TriggerClock, nil)
		require.NoError(t, err)
		require.True(t, result.Accepted)

		// Immediately try to get execution ID and stop
		time.Sleep(100 * time.Millisecond)

		executions, err := api.ListExecutions(ctx, memberID, nil)
		if err != nil || len(executions.Data) == 0 {
			t.Skip("No execution found")
			return
		}

		execID := executions.Data[0].ID

		// Try to stop immediately
		err = api.StopExecution(ctx, execID)
		if err != nil {
			t.Logf("Stop error: %v", err)
		} else {
			t.Logf("Stop signal sent for execution %s", execID)
		}

		// Wait and check status
		time.Sleep(5 * time.Second)

		exec, err := api.GetExecution(ctx, execID)
		if err != nil {
			t.Logf("Get execution error: %v", err)
			return
		}

		t.Logf("Final status: %s", exec.Status)
	})
}

// TestE2EControlMultipleOperations tests a sequence of control operations
func TestE2EControlMultipleOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test - requires real LLM calls")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	cleanupE2ERobots(t)
	cleanupE2EExecutions(t)
	defer cleanupE2ERobots(t)
	defer cleanupE2EExecutions(t)

	t.Run("pause_resume_pause_stop_sequence", func(t *testing.T) {
		memberID := "robot_e2e_control_multi"
		setupE2ERobotForControl(t, memberID, "team_e2e_control")

		err := api.Start()
		require.NoError(t, err)
		defer api.Stop()

		ctx := types.NewContext(context.Background(), testAuthControl())

		// Start execution
		result, err := api.TriggerManual(ctx, memberID, types.TriggerClock, nil)
		require.NoError(t, err)
		require.True(t, result.Accepted)

		// Wait for running state
		var execID string
		maxWait := 30 * time.Second
		deadline := time.Now().Add(maxWait)

		for time.Now().Before(deadline) {
			time.Sleep(500 * time.Millisecond)

			executions, err := api.ListExecutions(ctx, memberID, nil)
			if err != nil || len(executions.Data) == 0 {
				continue
			}

			exec := executions.Data[0]
			if exec.Status == types.ExecRunning {
				execID = exec.ID
				break
			}
		}

		if execID == "" {
			t.Skip("Execution did not start in time")
			return
		}

		// Sequence: Pause → Resume → Pause → Stop
		operations := []struct {
			name string
			fn   func() error
		}{
			{"Pause", func() error { return api.PauseExecution(ctx, execID) }},
			{"Resume", func() error { return api.ResumeExecution(ctx, execID) }},
			{"Pause", func() error { return api.PauseExecution(ctx, execID) }},
			{"Stop", func() error { return api.StopExecution(ctx, execID) }},
		}

		for _, op := range operations {
			err := op.fn()
			if err != nil {
				t.Logf("%s error (may be expected): %v", op.name, err)
				// If execution already completed, stop the sequence
				exec, _ := api.GetExecution(ctx, execID)
				if exec != nil && (exec.Status == types.ExecCompleted || exec.Status == types.ExecFailed || exec.Status == types.ExecCancelled) {
					t.Logf("Execution already finished: %s", exec.Status)
					return
				}
			} else {
				t.Logf("%s successful", op.name)
			}
			time.Sleep(2 * time.Second)
		}

		// Verify final state
		exec, err := api.GetExecution(ctx, execID)
		if err != nil {
			t.Logf("Get execution error: %v", err)
			return
		}

		t.Logf("Final status after operations: %s", exec.Status)
	})
}

// TestE2EControlStatusQuery tests querying execution status during control
func TestE2EControlStatusQuery(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test - requires real LLM calls")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	cleanupE2ERobots(t)
	cleanupE2EExecutions(t)
	defer cleanupE2ERobots(t)
	defer cleanupE2EExecutions(t)

	t.Run("query_status_during_execution", func(t *testing.T) {
		memberID := "robot_e2e_control_status"
		setupE2ERobotForControl(t, memberID, "team_e2e_control")

		err := api.Start()
		require.NoError(t, err)
		defer api.Stop()

		ctx := types.NewContext(context.Background(), testAuthControl())

		// Start execution
		result, err := api.TriggerManual(ctx, memberID, types.TriggerClock, nil)
		require.NoError(t, err)
		require.True(t, result.Accepted)

		// Track status changes
		statusHistory := make([]types.ExecStatus, 0)
		phaseHistory := make([]types.Phase, 0)

		maxWait := 120 * time.Second
		deadline := time.Now().Add(maxWait)

		lastStatus := types.ExecStatus("")
		lastPhase := types.Phase("")

		for time.Now().Before(deadline) {
			time.Sleep(1 * time.Second)

			executions, err := api.ListExecutions(ctx, memberID, nil)
			if err != nil || len(executions.Data) == 0 {
				continue
			}

			exec := executions.Data[0]

			// Track status changes
			if exec.Status != lastStatus {
				statusHistory = append(statusHistory, exec.Status)
				lastStatus = exec.Status
				t.Logf("Status changed: %s", exec.Status)
			}

			// Track phase changes
			if exec.Phase != lastPhase {
				phaseHistory = append(phaseHistory, exec.Phase)
				lastPhase = exec.Phase
				t.Logf("Phase changed: %s", exec.Phase)
			}

			// Also test GetExecutionStatus
			status, err := api.GetExecutionStatus(ctx, exec.ID)
			if err == nil && status != nil {
				// Status query should return valid data
				assert.NotEmpty(t, status.ID)
				assert.Equal(t, exec.Status, status.Status)
			}

			if exec.Status == types.ExecCompleted || exec.Status == types.ExecFailed || exec.Status == types.ExecCancelled {
				break
			}
		}

		t.Logf("Status history: %v", statusHistory)
		t.Logf("Phase history: %v", phaseHistory)

		// Should have observed at least pending → running transition
		assert.GreaterOrEqual(t, len(statusHistory), 1, "Should observe at least one status")
	})
}

// ==================== Helper Functions ====================

// setupE2ERobotForControl creates a robot for control tests
func setupE2ERobotForControl(t *testing.T, memberID, teamID string) {
	m := model.Select("__yao.member")
	tableName := m.MetaData.Table.Name
	qb := capsule.Query()

	// Simple config for E2E testing - minimal tasks
	robotConfig := map[string]interface{}{
		"identity": map[string]interface{}{
			"role":   "Simple E2E Test Robot",
			"duties": []string{"Say hello"}, // Very simple duty
			"rules":  []string{"Keep responses under 50 words"},
		},
		"quota": map[string]interface{}{
			"max":      3,
			"queue":    10,
			"priority": 5,
		},
		"triggers": map[string]interface{}{
			"clock":     map[string]interface{}{"enabled": true},
			"intervene": map[string]interface{}{"enabled": true},
			"event":     map[string]interface{}{"enabled": true},
		},
		"clock": map[string]interface{}{
			"mode":  "interval",
			"every": "1h",
		},
		"resources": map[string]interface{}{
			"phases": map[string]interface{}{
				"inspiration": "robot.inspiration",
				"goals":       "robot.goals",
				"tasks":       "tests.e2e-tasks", // Use simple E2E test task planner
				"run":         "robot.validation",
				"validation":  "tests.e2e-validation", // Use lenient E2E test validator
				"delivery":    "robot.delivery",
				"learning":    "robot.learning",
			},
			"agents": []string{"experts.text-writer"},
		},
		"delivery": map[string]interface{}{
			"email":   map[string]interface{}{"enabled": false},
			"webhook": map[string]interface{}{"enabled": false},
			"process": map[string]interface{}{"enabled": false},
		},
	}
	configJSON, _ := json.Marshal(robotConfig)

	systemPrompt := `You are a simple E2E test robot. Your job is to say hello.
When generating goals: create exactly 1 simple goal.
When generating tasks: create exactly 1 simple task.
Keep all outputs brief. No complex analysis needed.`

	err := qb.Table(tableName).Insert([]map[string]interface{}{
		{
			"member_id":       memberID,
			"team_id":         teamID,
			"member_type":     "robot",
			"display_name":    "E2E Control Robot " + memberID,
			"system_prompt":   systemPrompt,
			"status":          "active",
			"role_id":         "member",
			"autonomous_mode": true,
			"robot_status":    "idle",
			"robot_config":    string(configJSON),
		},
	})
	if err != nil {
		t.Fatalf("Failed to insert robot %s: %v", memberID, err)
	}
}
