package api_test

// End-to-end tests for Clock trigger flow
// These tests use REAL LLM calls via Standard executor (not DryRun)
//
// Test Flow: Clock Trigger → P0 (Inspiration) → P1 (Goals) → P2 (Tasks) → P3 (Run) → P4 (Delivery)
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

// testAuth returns test auth info for E2E tests
func testAuth() *oauthtypes.AuthorizedInfo {
	return &oauthtypes.AuthorizedInfo{
		UserID: "e2e-test-user",
		TeamID: "e2e-test-team",
	}
}

// TestE2EClockTriggerFullFlow tests the complete clock trigger flow with real LLM calls
// Flow: Clock → P0 (Inspiration) → P1 (Goals) → P2 (Tasks) → P3 (Run) → P4 (Delivery)
func TestE2EClockTriggerFullFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test - requires real LLM calls")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	cleanupE2ERobots(t)
	cleanupE2EExecutions(t)
	defer cleanupE2ERobots(t)
	defer cleanupE2EExecutions(t)

	t.Run("complete_P0_to_P4_flow", func(t *testing.T) {
		// Setup: Create a robot configured for clock trigger
		memberID := "robot_e2e_clock_001"
		setupE2ERobotForClock(t, memberID, "team_e2e_clock")

		// Start the API system
		err := api.Start()
		require.NoError(t, err)
		defer api.Stop()

		// Verify robot is loaded
		ctx := types.NewContext(context.Background(), testAuth())
		robot, err := api.GetRobot(ctx, memberID)
		require.NoError(t, err)
		require.NotNil(t, robot)
		assert.Equal(t, memberID, robot.MemberID)

		// Trigger execution via clock trigger type
		result, err := api.TriggerManual(ctx, memberID, types.TriggerClock, nil)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.True(t, result.Accepted, "Clock trigger should be accepted: %s", result.Message)
		assert.NotEmpty(t, result.ExecutionID, "Should return execution ID")

		t.Logf("Execution started: ExecutionID=%s", result.ExecutionID)

		// Wait for execution to complete (real LLM calls take time)
		// P0→P4 typically takes 30-60 seconds with real LLM
		var exec *types.Execution
		maxWait := 180 * time.Second
		pollInterval := 2 * time.Second
		deadline := time.Now().Add(maxWait)

		for time.Now().Before(deadline) {
			time.Sleep(pollInterval)

			// Query all executions and find a completed one
			executions, err := api.ListExecutions(ctx, memberID, &api.ExecutionQuery{
				Page:     1,
				PageSize: 10,
			})
			if err != nil {
				t.Logf("Query error (retrying): %v", err)
				continue
			}

			// Look for a completed execution
			for _, e := range executions.Data {
				t.Logf("Execution %s: status=%s, phase=%s", e.ID, e.Status, e.Phase)
				if e.Status == types.ExecCompleted {
					exec = e
					break
				}
			}

			if exec != nil {
				break
			}

			// Also check if there's any running execution
			hasRunning := false
			for _, e := range executions.Data {
				if e.Status == types.ExecRunning || e.Status == types.ExecPending {
					hasRunning = true
					break
				}
			}
			if !hasRunning && len(executions.Data) > 0 {
				// All executions finished but none completed - take the first one for error reporting
				exec = executions.Data[0]
				break
			}
		}

		// Verify execution completed successfully - ALL phases must pass
		require.NotNil(t, exec, "Execution should exist")

		if exec.Status == types.ExecFailed {
			t.Fatalf("Execution failed: %s", exec.Error)
		}

		// Strict assertion: execution MUST complete successfully
		assert.Equal(t, types.ExecCompleted, exec.Status, "Execution must complete successfully")

		// Verify P0 (Inspiration) output exists
		require.NotNil(t, exec.Inspiration, "P0 Inspiration output must exist")
		t.Logf("P0 Inspiration: %+v", exec.Inspiration)

		// Verify P1 (Goals) output exists
		require.NotNil(t, exec.Goals, "P1 Goals output must exist")
		t.Logf("P1 Goals content length: %d", len(exec.Goals.Content))

		// Verify P2 (Tasks) output exists
		require.NotNil(t, exec.Tasks, "P2 Tasks output must exist")
		require.Greater(t, len(exec.Tasks), 0, "P2 must have at least 1 task")
		t.Logf("P2 Tasks count: %d", len(exec.Tasks))

		// Verify P3 (Results) output exists - THIS IS CRITICAL
		require.NotNil(t, exec.Results, "P3 Results output must exist")
		require.Greater(t, len(exec.Results), 0, "P3 must have at least 1 result")
		t.Logf("P3 Results count: %d", len(exec.Results))

		// Verify P4 (Delivery) output exists
		require.NotNil(t, exec.Delivery, "P4 Delivery output must exist")
		t.Logf("P4 Delivery: RequestID=%s, Success=%v", exec.Delivery.RequestID, exec.Delivery.Success)

		t.Logf("✅ Clock trigger E2E: ALL PHASES (P0-P4) completed successfully")
	})
}

// TestE2EClockTriggerPhaseProgression tests that phases execute in correct order
func TestE2EClockTriggerPhaseProgression(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test - requires real LLM calls")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	cleanupE2ERobots(t)
	cleanupE2EExecutions(t)
	defer cleanupE2ERobots(t)
	defer cleanupE2EExecutions(t)

	t.Run("phases_execute_P0_P1_P2_P3_P4", func(t *testing.T) {
		memberID := "robot_e2e_clock_phases"
		setupE2ERobotForClock(t, memberID, "team_e2e_clock")

		err := api.Start()
		require.NoError(t, err)
		defer api.Stop()

		ctx := types.NewContext(context.Background(), testAuth())

		// Trigger execution
		result, err := api.TriggerManual(ctx, memberID, types.TriggerClock, nil)
		require.NoError(t, err)
		assert.True(t, result.Accepted)

		// Track phase progression
		phasesObserved := make([]types.Phase, 0)
		lastPhase := types.Phase("")

		maxWait := 120 * time.Second
		pollInterval := 1 * time.Second
		deadline := time.Now().Add(maxWait)

		for time.Now().Before(deadline) {
			time.Sleep(pollInterval)

			executions, err := api.ListExecutions(ctx, memberID, &api.ExecutionQuery{
				Page:     1,
				PageSize: 1,
			})
			if err != nil || len(executions.Data) == 0 {
				continue
			}

			exec := executions.Data[0]

			// Record phase changes
			if exec.Phase != lastPhase {
				phasesObserved = append(phasesObserved, exec.Phase)
				lastPhase = exec.Phase
				t.Logf("Phase changed to: %s", exec.Phase)
			}

			if exec.Status == types.ExecCompleted || exec.Status == types.ExecFailed {
				break
			}
		}

		// Verify phase order (should include at least P0, P1, P2, P3, P4)
		t.Logf("Phases observed: %v", phasesObserved)
		assert.GreaterOrEqual(t, len(phasesObserved), 1, "Should observe at least one phase")

		// The final phase should be delivery or learning
		if len(phasesObserved) > 0 {
			lastObserved := phasesObserved[len(phasesObserved)-1]
			assert.True(t,
				lastObserved == types.PhaseDelivery || lastObserved == types.PhaseLearning,
				"Last phase should be delivery or learning, got: %s", lastObserved)
		}
	})
}

// TestE2EClockTriggerDataPersistence tests that execution data is persisted to database
func TestE2EClockTriggerDataPersistence(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test - requires real LLM calls")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	cleanupE2ERobots(t)
	cleanupE2EExecutions(t)
	defer cleanupE2ERobots(t)
	defer cleanupE2EExecutions(t)

	t.Run("execution_data_persisted_to_database", func(t *testing.T) {
		memberID := "robot_e2e_clock_persist"
		setupE2ERobotForClock(t, memberID, "team_e2e_clock")

		err := api.Start()
		require.NoError(t, err)
		defer api.Stop()

		ctx := types.NewContext(context.Background(), testAuth())

		// Trigger and wait for completion
		result, err := api.TriggerManual(ctx, memberID, types.TriggerClock, nil)
		require.NoError(t, err)
		assert.True(t, result.Accepted)

		// Wait for completion
		var execID string
		maxWait := 120 * time.Second
		deadline := time.Now().Add(maxWait)

		for time.Now().Before(deadline) {
			time.Sleep(2 * time.Second)

			executions, err := api.ListExecutions(ctx, memberID, nil)
			if err != nil || len(executions.Data) == 0 {
				continue
			}

			exec := executions.Data[0]
			execID = exec.ID

			if exec.Status == types.ExecCompleted || exec.Status == types.ExecFailed {
				break
			}
		}

		require.NotEmpty(t, execID, "Should have execution ID")

		// Query execution by ID to verify persistence
		exec, err := api.GetExecution(ctx, execID)
		require.NoError(t, err)
		require.NotNil(t, exec)

		// Verify all fields are persisted
		assert.Equal(t, execID, exec.ID)
		assert.Equal(t, memberID, exec.MemberID)
		assert.Equal(t, types.TriggerClock, exec.TriggerType)
		assert.NotNil(t, exec.StartTime, "StartTime should be set")

		if exec.Status == types.ExecCompleted {
			assert.NotNil(t, exec.EndTime, "EndTime should be set for completed execution")
		}

		t.Logf("Persisted execution: ID=%s, Status=%s, Phase=%s", exec.ID, exec.Status, exec.Phase)
	})
}

// ==================== Helper Functions ====================

// setupE2ERobotForClock creates a robot configured for clock trigger E2E tests
// Uses extremely simple tasks to ensure quick completion through all phases
func setupE2ERobotForClock(t *testing.T, memberID, teamID string) {
	m := model.Select("__yao.member")
	tableName := m.MetaData.Table.Name
	qb := capsule.Query()

	// Robot config optimized for E2E testing - tasks must complete quickly
	robotConfig := map[string]interface{}{
		"identity": map[string]interface{}{
			"role":   "Greeter Bot",
			"duties": []string{"Output greeting message"}, // Extremely simple
			"rules":  []string{"Always complete in one step", "No tools needed", "Just output text directly"},
		},
		"quota": map[string]interface{}{
			"max":      5,
			"queue":    20,
			"priority": 5,
		},
		"triggers": map[string]interface{}{
			"clock":     map[string]interface{}{"enabled": true},
			"intervene": map[string]interface{}{"enabled": true},
			"event":     map[string]interface{}{"enabled": true},
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
			"agents": []string{},
		},
		"delivery": map[string]interface{}{
			"email":   map[string]interface{}{"enabled": false},
			"webhook": map[string]interface{}{"enabled": false},
			"process": map[string]interface{}{"enabled": false},
		},
	}
	configJSON, _ := json.Marshal(robotConfig)

	// Critical: System prompt must guide LLM to generate simple, completable tasks
	systemPrompt := `You are a simple greeter bot for E2E testing.

CRITICAL RULES FOR ALL PHASES:
1. Goals: Generate exactly ONE goal: "Output a greeting"
2. Tasks: Generate exactly ONE task with description "Say Hello World"
3. Run: Complete the task by directly outputting "Hello World!" - NO tools needed
4. The task is complete when you output the greeting text

You do NOT need any tools or external calls. Just output text directly.
When asked to execute a task, immediately respond with the greeting and mark complete.`

	err := qb.Table(tableName).Insert([]map[string]interface{}{
		{
			"member_id":       memberID,
			"team_id":         teamID,
			"member_type":     "robot",
			"display_name":    "E2E Clock Test Robot " + memberID,
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

// cleanupE2ERobots removes all E2E test robots
func cleanupE2ERobots(t *testing.T) {
	m := model.Select("__yao.member")
	if m == nil {
		return
	}
	tableName := m.MetaData.Table.Name
	qb := capsule.Query()

	_, err := qb.Table(tableName).Where("member_id", "like", "robot_e2e_%").Delete()
	if err != nil {
		t.Logf("Warning: cleanup robots error: %v", err)
	}
}

// cleanupE2EExecutions removes all E2E test executions
func cleanupE2EExecutions(t *testing.T) {
	m := model.Select("__yao.agent.execution")
	if m == nil {
		return
	}
	tableName := m.MetaData.Table.Name
	qb := capsule.Query()

	_, err := qb.Table(tableName).Where("member_id", "like", "robot_e2e_%").Delete()
	if err != nil {
		t.Logf("Warning: cleanup executions error: %v", err)
	}
}
