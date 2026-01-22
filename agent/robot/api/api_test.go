package api_test

// Integration tests for the Robot Agent API
// These tests verify the complete API functionality with real database operations.
//
// Test Structure:
//   - api_test.go: Core API integration tests (this file)
//   - lifecycle_test.go: Start/Stop lifecycle tests
//   - robot_test.go: Robot query tests
//   - trigger_test.go: Trigger tests
//   - execution_test.go: Execution query/control tests
//
// Test Data:
//   All tests use real database records in __yao.member and agent_execution tables
//   Test robot IDs are prefixed with "robot_api_" for easy cleanup
//   Test execution IDs are prefixed with "exec_api_" for easy cleanup

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
	"github.com/yaoapp/yao/agent/robot/store"
	"github.com/yaoapp/yao/agent/robot/types"
	"github.com/yaoapp/yao/agent/testutils"
)

// ==================== Full Lifecycle Integration Tests ====================

// TestAPIFullLifecycle tests the complete API workflow:
// Start → Create Robot → Query Robot → Trigger → Query Execution → Stop
func TestAPIFullLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Cleanup test data
	cleanupAPITestRobots(t)
	cleanupAPITestExecutions(t)
	defer cleanupAPITestRobots(t)
	defer cleanupAPITestExecutions(t)

	t.Run("complete workflow", func(t *testing.T) {
		// 1. Setup: Create test robot in database
		setupAPITestRobot(t, "robot_api_lifecycle_001", "team_api_001")

		// 2. Start the API system
		err := api.Start()
		require.NoError(t, err)
		assert.True(t, api.IsRunning())
		defer api.Stop()

		// 3. Query the robot via API
		ctx := types.NewContext(context.Background(), nil)
		robot, err := api.GetRobot(ctx, "robot_api_lifecycle_001")
		require.NoError(t, err)
		require.NotNil(t, robot)
		assert.Equal(t, "robot_api_lifecycle_001", robot.MemberID)
		assert.Equal(t, "team_api_001", robot.TeamID)
		assert.Equal(t, "API Test Robot robot_api_lifecycle_001", robot.DisplayName)

		// 4. Get robot status
		status, err := api.GetRobotStatus(ctx, "robot_api_lifecycle_001")
		require.NoError(t, err)
		require.NotNil(t, status)
		assert.Equal(t, "robot_api_lifecycle_001", status.MemberID)
		assert.Equal(t, types.RobotIdle, status.Status)
		assert.Equal(t, 0, status.Running)
		assert.Equal(t, 5, status.MaxRunning)

		// 5. List robots
		listResult, err := api.ListRobots(ctx, &api.ListQuery{
			TeamID:   "team_api_001",
			Page:     1,
			PageSize: 10,
		})
		require.NoError(t, err)
		require.NotNil(t, listResult)
		assert.GreaterOrEqual(t, listResult.Total, 1)

		// Find our robot in the list
		found := false
		for _, r := range listResult.Data {
			if r.MemberID == "robot_api_lifecycle_001" {
				found = true
				break
			}
		}
		assert.True(t, found, "Robot should be in list")

		// 6. Trigger manual execution
		triggerResult, err := api.TriggerManual(ctx, "robot_api_lifecycle_001", types.TriggerClock, nil)
		require.NoError(t, err)
		require.NotNil(t, triggerResult)
		assert.True(t, triggerResult.Accepted)
		assert.NotEmpty(t, triggerResult.ExecutionID)

		// 7. Wait for execution to complete
		time.Sleep(500 * time.Millisecond)

		// 8. Stop the system
		err = api.Stop()
		require.NoError(t, err)
		assert.False(t, api.IsRunning())
	})
}

// TestAPIRobotQueryWithData tests robot query APIs with real data
func TestAPIRobotQueryWithData(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	cleanupAPITestRobots(t)
	defer cleanupAPITestRobots(t)

	// Setup: Create multiple robots
	setupAPITestRobot(t, "robot_api_query_001", "team_api_query")
	setupAPITestRobot(t, "robot_api_query_002", "team_api_query")
	setupAPITestRobot(t, "robot_api_query_003", "team_api_other")

	ctx := types.NewContext(context.Background(), nil)

	t.Run("GetRobot returns correct robot", func(t *testing.T) {
		robot, err := api.GetRobot(ctx, "robot_api_query_001")
		require.NoError(t, err)
		require.NotNil(t, robot)

		assert.Equal(t, "robot_api_query_001", robot.MemberID)
		assert.Equal(t, "team_api_query", robot.TeamID)
		assert.Equal(t, "API Test Robot robot_api_query_001", robot.DisplayName)
		assert.True(t, robot.AutonomousMode)
		assert.Equal(t, types.RobotIdle, robot.Status)
	})

	t.Run("ListRobots filters by team", func(t *testing.T) {
		result, err := api.ListRobots(ctx, &api.ListQuery{
			TeamID:   "team_api_query",
			Page:     1,
			PageSize: 10,
		})
		require.NoError(t, err)
		require.NotNil(t, result)

		// Should have at least 2 robots from team_api_query
		// (might have more if other tests created robots in this team)
		assert.GreaterOrEqual(t, result.Total, 2)
		assert.GreaterOrEqual(t, len(result.Data), 2)

		// Verify all returned robots are from the correct team
		for _, robot := range result.Data {
			assert.Equal(t, "team_api_query", robot.TeamID)
		}
	})

	t.Run("ListRobots pagination works", func(t *testing.T) {
		// Page 1 with size 1
		result1, err := api.ListRobots(ctx, &api.ListQuery{
			TeamID:   "team_api_query",
			Page:     1,
			PageSize: 1,
		})
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(result1.Data), 1, "Should have at least 1 robot on page 1")

		// Page 2 with size 1
		result2, err := api.ListRobots(ctx, &api.ListQuery{
			TeamID:   "team_api_query",
			Page:     2,
			PageSize: 1,
		})
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(result2.Data), 1, "Should have at least 1 robot on page 2")

		// Should be different robots
		assert.NotEqual(t, result1.Data[0].MemberID, result2.Data[0].MemberID)
	})

	t.Run("ListRobots filters by keywords", func(t *testing.T) {
		result, err := api.ListRobots(ctx, &api.ListQuery{
			Keywords: "robot_api_query_001",
			Page:     1,
			PageSize: 10,
		})
		require.NoError(t, err)
		require.NotNil(t, result)

		// Should find at least 1 robot matching keywords
		assert.GreaterOrEqual(t, result.Total, 1)
		for _, robot := range result.Data {
			assert.Contains(t, robot.DisplayName, "robot_api_query_001")
		}
	})
}

// TestListRobotsAutonomousModeFilter tests the autonomous_mode filter
func TestListRobotsAutonomousModeFilter(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	cleanupAPITestRobots(t)
	defer cleanupAPITestRobots(t)

	// Setup: Create robots with different autonomous_mode settings
	setupAPITestRobotWithMode(t, "robot_api_auto_001", "team_api_mode", true)    // autonomous
	setupAPITestRobotWithMode(t, "robot_api_auto_002", "team_api_mode", true)    // autonomous
	setupAPITestRobotWithMode(t, "robot_api_demand_001", "team_api_mode", false) // on-demand

	ctx := types.NewContext(context.Background(), nil)

	t.Run("ListRobots returns all robots when autonomous_mode is nil", func(t *testing.T) {
		result, err := api.ListRobots(ctx, &api.ListQuery{
			TeamID:   "team_api_mode",
			Page:     1,
			PageSize: 10,
		})
		require.NoError(t, err)
		require.NotNil(t, result)

		// Should have all 3 robots
		assert.Equal(t, 3, result.Total)
	})

	t.Run("ListRobots filters by autonomous_mode=true", func(t *testing.T) {
		autonomousMode := true
		result, err := api.ListRobots(ctx, &api.ListQuery{
			TeamID:         "team_api_mode",
			AutonomousMode: &autonomousMode,
			Page:           1,
			PageSize:       10,
		})
		require.NoError(t, err)
		require.NotNil(t, result)

		// Should have only 2 autonomous robots
		assert.Equal(t, 2, result.Total)
		for _, robot := range result.Data {
			assert.True(t, robot.AutonomousMode, "All returned robots should be autonomous")
		}
	})

	t.Run("ListRobots filters by autonomous_mode=false", func(t *testing.T) {
		autonomousMode := false
		result, err := api.ListRobots(ctx, &api.ListQuery{
			TeamID:         "team_api_mode",
			AutonomousMode: &autonomousMode,
			Page:           1,
			PageSize:       10,
		})
		require.NoError(t, err)
		require.NotNil(t, result)

		// Should have only 1 on-demand robot
		assert.Equal(t, 1, result.Total)
		for _, robot := range result.Data {
			assert.False(t, robot.AutonomousMode, "All returned robots should be on-demand")
		}
	})
}

// TestAPIExecutionQueryWithData tests execution query APIs with real data
func TestAPIExecutionQueryWithData(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	cleanupAPITestExecutions(t)
	defer cleanupAPITestExecutions(t)

	// Setup: Create test executions
	setupAPITestExecution(t, "exec_api_query_001", "member_api_exec", types.TriggerClock, types.ExecCompleted)
	setupAPITestExecution(t, "exec_api_query_002", "member_api_exec", types.TriggerHuman, types.ExecRunning)
	setupAPITestExecution(t, "exec_api_query_003", "member_api_exec", types.TriggerClock, types.ExecFailed)
	setupAPITestExecution(t, "exec_api_query_004", "member_api_other", types.TriggerEvent, types.ExecCompleted)

	ctx := types.NewContext(context.Background(), nil)

	t.Run("GetExecution returns correct execution", func(t *testing.T) {
		exec, err := api.GetExecution(ctx, "exec_api_query_001")
		require.NoError(t, err)
		require.NotNil(t, exec)

		assert.Equal(t, "exec_api_query_001", exec.ID)
		assert.Equal(t, "member_api_exec", exec.MemberID)
		assert.Equal(t, types.TriggerClock, exec.TriggerType)
		assert.Equal(t, types.ExecCompleted, exec.Status)
	})

	t.Run("ListExecutions filters by member", func(t *testing.T) {
		result, err := api.ListExecutions(ctx, "member_api_exec", &api.ExecutionQuery{
			Page:     1,
			PageSize: 10,
		})
		require.NoError(t, err)
		require.NotNil(t, result)

		// Should have 3 executions for member_api_exec
		assert.Equal(t, 3, result.Total)
		assert.Len(t, result.Data, 3)

		// Verify all returned executions are for the correct member
		for _, exec := range result.Data {
			assert.Equal(t, "member_api_exec", exec.MemberID)
		}
	})

	t.Run("ListExecutions filters by status", func(t *testing.T) {
		result, err := api.ListExecutions(ctx, "member_api_exec", &api.ExecutionQuery{
			Status:   types.ExecCompleted,
			Page:     1,
			PageSize: 10,
		})
		require.NoError(t, err)
		require.NotNil(t, result)

		// Should have only completed executions
		assert.Equal(t, 1, result.Total)
		for _, exec := range result.Data {
			assert.Equal(t, types.ExecCompleted, exec.Status)
		}
	})

	t.Run("ListExecutions filters by trigger type", func(t *testing.T) {
		result, err := api.ListExecutions(ctx, "member_api_exec", &api.ExecutionQuery{
			Trigger:  types.TriggerClock,
			Page:     1,
			PageSize: 10,
		})
		require.NoError(t, err)
		require.NotNil(t, result)

		// Should have only clock trigger executions
		assert.Equal(t, 2, result.Total)
		for _, exec := range result.Data {
			assert.Equal(t, types.TriggerClock, exec.TriggerType)
		}
	})

	t.Run("ListExecutions pagination works", func(t *testing.T) {
		// Page 1 with size 2
		result1, err := api.ListExecutions(ctx, "member_api_exec", &api.ExecutionQuery{
			Page:     1,
			PageSize: 2,
		})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(result1.Data), 2, "Should have at least 2 executions on page 1")

		// Page 2 with size 2
		result2, err := api.ListExecutions(ctx, "member_api_exec", &api.ExecutionQuery{
			Page:     2,
			PageSize: 2,
		})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(result2.Data), 1, "Should have at least 1 execution on page 2")
	})
}

// TestAPITriggerWithData tests trigger APIs with real robots
func TestAPITriggerWithData(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	cleanupAPITestRobots(t)
	defer cleanupAPITestRobots(t)

	// Setup: Create test robot
	setupAPITestRobot(t, "robot_api_trigger_001", "team_api_trigger")

	// Start manager
	err := api.Start()
	require.NoError(t, err)
	defer api.Stop()

	ctx := types.NewContext(context.Background(), nil)

	t.Run("TriggerManual accepts valid robot", func(t *testing.T) {
		result, err := api.TriggerManual(ctx, "robot_api_trigger_001", types.TriggerClock, nil)
		require.NoError(t, err)
		require.NotNil(t, result)

		assert.True(t, result.Accepted)
		assert.NotEmpty(t, result.ExecutionID)
		assert.Contains(t, result.Message, "submitted")
	})

	t.Run("Trigger with human type", func(t *testing.T) {
		result, err := api.Trigger(ctx, "robot_api_trigger_001", &api.TriggerRequest{
			Type:   types.TriggerHuman,
			Action: types.ActionTaskAdd,
		})
		require.NoError(t, err)
		require.NotNil(t, result)

		// Result should be returned (accepted or not depends on robot state)
		// The important thing is that the API doesn't error
		t.Logf("Trigger result: accepted=%v, message=%s", result.Accepted, result.Message)
	})

	t.Run("Trigger with event type", func(t *testing.T) {
		result, err := api.Trigger(ctx, "robot_api_trigger_001", &api.TriggerRequest{
			Type:      types.TriggerEvent,
			Source:    types.EventWebhook,
			EventType: "test.event",
			Data:      map[string]interface{}{"key": "value"},
		})
		require.NoError(t, err)
		require.NotNil(t, result)

		// Should be accepted (robot exists and has event enabled)
		assert.True(t, result.Accepted)
	})

	t.Run("Trigger rejects non-existent robot", func(t *testing.T) {
		result, err := api.Trigger(ctx, "robot_api_nonexistent", &api.TriggerRequest{
			Type: types.TriggerHuman,
		})
		// Should not return error, but result should show not accepted
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.False(t, result.Accepted)
	})
}

// ==================== Helper Functions ====================

// setupAPITestRobotWithMode creates a test robot with specific autonomous_mode setting
func setupAPITestRobotWithMode(t *testing.T, memberID, teamID string, autonomousMode bool) {
	m := model.Select("__yao.member")
	tableName := m.MetaData.Table.Name
	qb := capsule.Query()

	robotConfig := map[string]interface{}{
		"identity": map[string]interface{}{
			"role":   "API Test Robot",
			"duties": []string{"Testing API functions"},
		},
		"quota": map[string]interface{}{
			"max":      5,
			"queue":    20,
			"priority": 5,
		},
	}
	configJSON, _ := json.Marshal(robotConfig)

	err := qb.Table(tableName).Insert([]map[string]interface{}{
		{
			"member_id":       memberID,
			"team_id":         teamID,
			"member_type":     "robot",
			"display_name":    "API Test Robot " + memberID,
			"system_prompt":   "You are an API test robot.",
			"status":          "active",
			"role_id":         "member",
			"autonomous_mode": autonomousMode,
			"robot_status":    "idle",
			"robot_config":    string(configJSON),
		},
	})
	if err != nil {
		t.Fatalf("Failed to insert robot %s: %v", memberID, err)
	}
}

// setupAPITestRobot creates a test robot in the database
func setupAPITestRobot(t *testing.T, memberID, teamID string) {
	m := model.Select("__yao.member")
	tableName := m.MetaData.Table.Name
	qb := capsule.Query()

	robotConfig := map[string]interface{}{
		"identity": map[string]interface{}{
			"role":   "API Test Robot",
			"duties": []string{"Testing API functions"},
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
		"clock": map[string]interface{}{
			"mode":    "interval",
			"every":   "1h",
			"timeout": "30m",
		},
	}
	configJSON, _ := json.Marshal(robotConfig)

	err := qb.Table(tableName).Insert([]map[string]interface{}{
		{
			"member_id":       memberID,
			"team_id":         teamID,
			"member_type":     "robot",
			"display_name":    "API Test Robot " + memberID,
			"system_prompt":   "You are an API test robot.",
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

// setupAPITestExecution creates a test execution in the database
func setupAPITestExecution(t *testing.T, execID, memberID string, triggerType types.TriggerType, status types.ExecStatus) {
	s := store.NewExecutionStore()
	ctx := context.Background()

	startTime := time.Now().Add(-1 * time.Hour)
	record := &store.ExecutionRecord{
		ExecutionID: execID,
		MemberID:    memberID,
		TeamID:      "team_api_exec",
		TriggerType: triggerType,
		Status:      status,
		Phase:       types.PhaseDelivery,
		StartTime:   &startTime,
	}

	if status == types.ExecCompleted || status == types.ExecFailed {
		endTime := time.Now()
		record.EndTime = &endTime
	}

	err := s.Save(ctx, record)
	if err != nil {
		t.Fatalf("Failed to insert execution %s: %v", execID, err)
	}
}

// cleanupAPITestRobots removes all API test robots
func cleanupAPITestRobots(t *testing.T) {
	m := model.Select("__yao.member")
	if m == nil {
		return
	}
	tableName := m.MetaData.Table.Name
	qb := capsule.Query()

	// Delete all robots with member_id starting with "robot_api_" or "api_robot_"
	_, err := qb.Table(tableName).Where("member_id", "like", "robot_api_%").Delete()
	if err != nil {
		t.Logf("Warning: cleanup robots error: %v", err)
	}

	// Also delete "api_robot_" prefixed robots (new tests)
	_, err = qb.Table(tableName).Where("member_id", "like", "api_robot_%").Delete()
	if err != nil {
		t.Logf("Warning: cleanup robots error: %v", err)
	}
}

// cleanupAPITestExecutions removes all API test executions
func cleanupAPITestExecutions(t *testing.T) {
	m := model.Select("__yao.agent.execution")
	if m == nil {
		t.Logf("Warning: model __yao.agent.execution not found, skipping cleanup")
		return
	}
	tableName := m.MetaData.Table.Name
	qb := capsule.Query()

	// Delete all executions with execution_id starting with "exec_api_"
	_, err := qb.Table(tableName).Where("execution_id", "like", "exec_api_%").Delete()
	if err != nil {
		t.Logf("Warning: cleanup executions error: %v", err)
	}

	// Also delete executions for API test members
	_, err = qb.Table(tableName).Where("member_id", "like", "member_api_%").Delete()
	if err != nil {
		t.Logf("Warning: cleanup executions error: %v", err)
	}
}
