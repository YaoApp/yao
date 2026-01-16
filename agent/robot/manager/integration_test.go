package manager_test

// Integration tests for the Robot Agent scheduling system
// These tests verify the complete end-to-end flow:
//   Trigger → Manager → Cache → Pool → Worker → Executor → Job
//
// Test Structure:
//   - integration_test.go:       Core scheduling flow tests
//   - integration_clock_test.go: Clock trigger mode tests (times/interval/daemon)
//   - integration_human_test.go: Human intervention trigger tests
//   - integration_event_test.go: Event trigger tests
//   - integration_concurrent_test.go: Concurrent execution & quota tests
//   - integration_control_test.go: Pause/Resume/Stop tests
//
// Test Data:
//   All tests use real database records in __yao.member table
//   Test robot IDs are prefixed with "robot_integ_" for easy cleanup

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/xun/capsule"
	"github.com/yaoapp/yao/agent/robot/executor"
	"github.com/yaoapp/yao/agent/robot/manager"
	"github.com/yaoapp/yao/agent/robot/pool"
	"github.com/yaoapp/yao/agent/robot/types"
	"github.com/yaoapp/yao/agent/testutils"
)

// ==================== Core Scheduling Flow Tests ====================

// TestIntegrationSchedulingFlow tests the complete scheduling flow:
// Create robot → Start manager → Trigger → Verify execution
func TestIntegrationSchedulingFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	cleanupIntegrationRobots(t)
	defer cleanupIntegrationRobots(t)

	t.Run("complete clock trigger flow", func(t *testing.T) {
		// Setup: Create a robot with times mode clock config
		setupIntegrationRobotTimes(t, "robot_integ_flow_clock", "team_integ_flow")

		// Create manager with fast tick interval for testing
		config := &manager.Config{
			TickInterval: 100 * time.Millisecond,
			PoolConfig:   &pool.Config{WorkerSize: 5, QueueSize: 50},
		}
		m := manager.NewWithConfig(config)

		// Start manager
		err := m.Start()
		require.NoError(t, err)
		defer m.Stop()

		// Verify robot is loaded into cache
		robot := m.Cache().Get("robot_integ_flow_clock")
		require.NotNil(t, robot, "Robot should be loaded into cache")
		assert.Equal(t, "robot_integ_flow_clock", robot.MemberID)
		assert.Equal(t, types.RobotIdle, robot.Status)

		// Simulate clock trigger at matching time (09:00 on Wednesday)
		loc, _ := time.LoadLocation("Asia/Shanghai")
		triggerTime := time.Date(2025, 1, 15, 9, 0, 0, 0, loc) // Wednesday 09:00

		ctx := types.NewContext(context.Background(), nil)
		err = m.Tick(ctx, triggerTime)
		assert.NoError(t, err)

		// Wait for execution to complete
		time.Sleep(500 * time.Millisecond)

		// Verify execution happened
		execCount := m.Executor().ExecCount()
		assert.GreaterOrEqual(t, execCount, 1, "Should have at least 1 execution")
	})

	t.Run("robot loaded from database", func(t *testing.T) {
		// Setup: Create multiple robots
		setupIntegrationRobotTimes(t, "robot_integ_flow_db1", "team_integ_flow")
		setupIntegrationRobotInterval(t, "robot_integ_flow_db2", "team_integ_flow")

		m := manager.New()
		err := m.Start()
		require.NoError(t, err)
		defer m.Stop()

		// Verify both robots are in cache
		robot1 := m.Cache().Get("robot_integ_flow_db1")
		robot2 := m.Cache().Get("robot_integ_flow_db2")
		assert.NotNil(t, robot1, "Robot 1 should be loaded")
		assert.NotNil(t, robot2, "Robot 2 should be loaded")

		// Verify config is parsed correctly
		assert.NotNil(t, robot1.Config)
		assert.NotNil(t, robot1.Config.Clock)
		assert.Equal(t, types.ClockTimes, robot1.Config.Clock.Mode)

		assert.NotNil(t, robot2.Config)
		assert.NotNil(t, robot2.Config.Clock)
		assert.Equal(t, types.ClockInterval, robot2.Config.Clock.Mode)
	})

	t.Run("inactive robot not loaded", func(t *testing.T) {
		// Setup: Create an inactive robot
		setupIntegrationRobotInactive(t, "robot_integ_flow_inactive", "team_integ_flow")

		m := manager.New()
		err := m.Start()
		require.NoError(t, err)
		defer m.Stop()

		// Inactive robot should not be in cache
		robot := m.Cache().Get("robot_integ_flow_inactive")
		assert.Nil(t, robot, "Inactive robot should not be loaded")
	})

	t.Run("robot with autonomous_mode=false not loaded", func(t *testing.T) {
		// Setup: Create a robot with autonomous_mode=false
		setupIntegrationRobotNonAutonomous(t, "robot_integ_flow_nonauto", "team_integ_flow")

		m := manager.New()
		err := m.Start()
		require.NoError(t, err)
		defer m.Stop()

		// Non-autonomous robot should not be in cache
		robot := m.Cache().Get("robot_integ_flow_nonauto")
		assert.Nil(t, robot, "Non-autonomous robot should not be loaded")
	})
}

// TestIntegrationJobSubmission tests job submission to pool and execution
func TestIntegrationJobSubmission(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	cleanupIntegrationRobots(t)
	defer cleanupIntegrationRobots(t)

	t.Run("job submitted to pool and executed", func(t *testing.T) {
		setupIntegrationRobotTimes(t, "robot_integ_submit", "team_integ_submit")

		config := &manager.Config{
			TickInterval: 100 * time.Millisecond,
			PoolConfig:   &pool.Config{WorkerSize: 3, QueueSize: 20},
		}
		m := manager.NewWithConfig(config)

		err := m.Start()
		require.NoError(t, err)
		defer m.Stop()

		// Manually trigger execution
		ctx := types.NewContext(context.Background(), nil)
		execID, err := m.TriggerManual(ctx, "robot_integ_submit", types.TriggerClock, nil)
		assert.NoError(t, err)
		assert.NotEmpty(t, execID, "Should return execution ID")

		// Wait for execution
		time.Sleep(500 * time.Millisecond)

		// Verify execution completed
		assert.GreaterOrEqual(t, m.Executor().ExecCount(), 1)
	})

	t.Run("multiple jobs queued and executed in order", func(t *testing.T) {
		setupIntegrationRobotHighQuota(t, "robot_integ_queue", "team_integ_submit")

		config := &manager.Config{
			TickInterval: 100 * time.Millisecond,
			PoolConfig:   &pool.Config{WorkerSize: 2, QueueSize: 50},
		}
		m := manager.NewWithConfig(config)

		err := m.Start()
		require.NoError(t, err)
		defer m.Stop()

		ctx := types.NewContext(context.Background(), nil)

		// Submit multiple jobs
		execIDs := make([]string, 5)
		for i := 0; i < 5; i++ {
			execID, err := m.TriggerManual(ctx, "robot_integ_queue", types.TriggerClock, nil)
			assert.NoError(t, err)
			execIDs[i] = execID
		}

		// All should have valid IDs
		for i, id := range execIDs {
			assert.NotEmpty(t, id, "Execution %d should have valid ID", i)
		}

		// Wait for all to complete (longer wait for slow execution)
		time.Sleep(2 * time.Second)

		// All jobs should have executed
		execCount := m.Executor().ExecCount()
		assert.GreaterOrEqual(t, execCount, 5, "Expected at least 5 executions, got %d", execCount)
	})
}

// TestIntegrationPhaseProgression tests that execution progresses through all phases
func TestIntegrationPhaseProgression(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	cleanupIntegrationRobots(t)
	defer cleanupIntegrationRobots(t)

	t.Run("clock trigger executes all phases P0-P5", func(t *testing.T) {
		setupIntegrationRobotTimes(t, "robot_integ_phases_clock", "team_integ_phases")

		// Track phases executed
		phasesExecuted := make([]types.Phase, 0)
		exec := executor.NewDryRunWithConfig(executor.DryRunConfig{
			Config: executor.Config{
				OnPhaseStart: func(phase types.Phase) {
					phasesExecuted = append(phasesExecuted, phase)
				},
			},
		})

		config := &manager.Config{
			TickInterval: 100 * time.Millisecond,
			PoolConfig:   &pool.Config{WorkerSize: 2, QueueSize: 20},
			Executor:     exec,
		}
		m := manager.NewWithConfig(config)

		err := m.Start()
		require.NoError(t, err)
		defer m.Stop()

		// Trigger execution
		ctx := types.NewContext(context.Background(), nil)
		_, err = m.TriggerManual(ctx, "robot_integ_phases_clock", types.TriggerClock, nil)
		assert.NoError(t, err)

		// Wait for execution
		time.Sleep(500 * time.Millisecond)

		// Verify all 6 phases executed (P0-P5)
		assert.Len(t, phasesExecuted, 6, "Should execute all 6 phases for clock trigger")
		assert.Equal(t, types.PhaseInspiration, phasesExecuted[0], "Should start with P0")
		assert.Equal(t, types.PhaseLearning, phasesExecuted[5], "Should end with P5")
	})

	t.Run("human trigger skips P0 and executes P1-P5", func(t *testing.T) {
		setupIntegrationRobotIntervene(t, "robot_integ_phases_human", "team_integ_phases")

		// Track phases executed
		phasesExecuted := make([]types.Phase, 0)
		exec := executor.NewDryRunWithConfig(executor.DryRunConfig{
			Config: executor.Config{
				OnPhaseStart: func(phase types.Phase) {
					phasesExecuted = append(phasesExecuted, phase)
				},
			},
		})

		config := &manager.Config{
			TickInterval: 100 * time.Millisecond,
			PoolConfig:   &pool.Config{WorkerSize: 2, QueueSize: 20},
			Executor:     exec,
		}
		m := manager.NewWithConfig(config)

		err := m.Start()
		require.NoError(t, err)
		defer m.Stop()

		// Trigger execution via human trigger
		ctx := types.NewContext(context.Background(), nil)
		_, err = m.TriggerManual(ctx, "robot_integ_phases_human", types.TriggerHuman, nil)
		assert.NoError(t, err)

		// Wait for execution
		time.Sleep(500 * time.Millisecond)

		// Verify 5 phases executed (P1-P5, skipping P0)
		assert.Len(t, phasesExecuted, 5, "Should execute 5 phases for human trigger")
		assert.Equal(t, types.PhaseGoals, phasesExecuted[0], "Should start with P1 (Goals)")
		assert.Equal(t, types.PhaseLearning, phasesExecuted[4], "Should end with P5")
	})
}

// TestIntegrationCacheRefresh tests that cache refresh works correctly
func TestIntegrationCacheRefresh(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	cleanupIntegrationRobots(t)
	defer cleanupIntegrationRobots(t)

	t.Run("cache refresh loads new robots", func(t *testing.T) {
		// Start with one robot
		setupIntegrationRobotTimes(t, "robot_integ_refresh1", "team_integ_refresh")

		m := manager.New()
		err := m.Start()
		require.NoError(t, err)
		defer m.Stop()

		// Verify first robot is loaded
		robot1 := m.Cache().Get("robot_integ_refresh1")
		assert.NotNil(t, robot1)

		// Add another robot to database
		setupIntegrationRobotTimes(t, "robot_integ_refresh2", "team_integ_refresh")

		// Manually refresh cache
		ctx := types.NewContext(context.Background(), nil)
		err = m.Cache().Load(ctx)
		assert.NoError(t, err)

		// Verify new robot is now in cache
		robot2 := m.Cache().Get("robot_integ_refresh2")
		assert.NotNil(t, robot2, "New robot should be loaded after refresh")
	})
}

// ==================== Test Data Setup Helpers ====================

// setupIntegrationRobotTimes creates a robot with times mode clock config
func setupIntegrationRobotTimes(t *testing.T, memberID, teamID string) {
	m := model.Select("__yao.member")
	tableName := m.MetaData.Table.Name
	qb := capsule.Query()

	robotConfig := map[string]interface{}{
		"identity": map[string]interface{}{
			"role":   "Integration Test Robot (Times)",
			"duties": []string{"Test scheduling"},
		},
		"quota": map[string]interface{}{
			"max":      3,
			"queue":    20,
			"priority": 5,
		},
		"triggers": map[string]interface{}{
			"clock":     map[string]interface{}{"enabled": true},
			"intervene": map[string]interface{}{"enabled": true},
			"event":     map[string]interface{}{"enabled": true},
		},
		"clock": map[string]interface{}{
			"mode":    "times",
			"times":   []string{"09:00", "14:00", "17:00"},
			"days":    []string{"Mon", "Tue", "Wed", "Thu", "Fri"},
			"tz":      "Asia/Shanghai",
			"timeout": "30m",
		},
	}
	configJSON, _ := json.Marshal(robotConfig)

	err := qb.Table(tableName).Insert([]map[string]interface{}{
		{
			"member_id":       memberID,
			"team_id":         teamID,
			"member_type":     "robot",
			"display_name":    "Test Robot " + memberID,
			"system_prompt":   "You are an integration test robot.",
			"status":          "active",
			"role_id":         "member",
			"autonomous_mode": true,
			"robot_status":    "idle",
			"robot_config":    string(configJSON),
		},
	})
	if err != nil {
		t.Fatalf("Failed to insert %s: %v", memberID, err)
	}
}

// setupIntegrationRobotInterval creates a robot with interval mode clock config
func setupIntegrationRobotInterval(t *testing.T, memberID, teamID string) {
	m := model.Select("__yao.member")
	tableName := m.MetaData.Table.Name
	qb := capsule.Query()

	robotConfig := map[string]interface{}{
		"identity": map[string]interface{}{
			"role": "Integration Test Robot (Interval)",
		},
		"quota": map[string]interface{}{
			"max":      2,
			"queue":    10,
			"priority": 5,
		},
		"triggers": map[string]interface{}{
			"clock": map[string]interface{}{"enabled": true},
		},
		"clock": map[string]interface{}{
			"mode":    "interval",
			"every":   "30m",
			"timeout": "10m",
		},
	}
	configJSON, _ := json.Marshal(robotConfig)

	err := qb.Table(tableName).Insert([]map[string]interface{}{
		{
			"member_id":       memberID,
			"team_id":         teamID,
			"member_type":     "robot",
			"display_name":    "Test Robot " + memberID,
			"status":          "active",
			"role_id":         "member",
			"autonomous_mode": true,
			"robot_status":    "idle",
			"robot_config":    string(configJSON),
		},
	})
	if err != nil {
		t.Fatalf("Failed to insert %s: %v", memberID, err)
	}
}

// setupIntegrationRobotHighQuota creates a robot with high quota for queue tests
func setupIntegrationRobotHighQuota(t *testing.T, memberID, teamID string) {
	m := model.Select("__yao.member")
	tableName := m.MetaData.Table.Name
	qb := capsule.Query()

	robotConfig := map[string]interface{}{
		"identity": map[string]interface{}{
			"role": "Integration Test Robot (High Quota)",
		},
		"quota": map[string]interface{}{
			"max":      10,
			"queue":    50,
			"priority": 5,
		},
		"triggers": map[string]interface{}{
			"clock": map[string]interface{}{"enabled": true},
		},
		"clock": map[string]interface{}{
			"mode":  "times",
			"times": []string{"09:00"},
			"tz":    "Asia/Shanghai",
		},
	}
	configJSON, _ := json.Marshal(robotConfig)

	err := qb.Table(tableName).Insert([]map[string]interface{}{
		{
			"member_id":       memberID,
			"team_id":         teamID,
			"member_type":     "robot",
			"display_name":    "Test Robot " + memberID,
			"status":          "active",
			"role_id":         "member",
			"autonomous_mode": true,
			"robot_status":    "idle",
			"robot_config":    string(configJSON),
		},
	})
	if err != nil {
		t.Fatalf("Failed to insert %s: %v", memberID, err)
	}
}

// setupIntegrationRobotIntervene creates a robot with intervene trigger enabled
func setupIntegrationRobotIntervene(t *testing.T, memberID, teamID string) {
	m := model.Select("__yao.member")
	tableName := m.MetaData.Table.Name
	qb := capsule.Query()

	robotConfig := map[string]interface{}{
		"identity": map[string]interface{}{
			"role": "Integration Test Robot (Intervene)",
		},
		"quota": map[string]interface{}{
			"max":      5,
			"queue":    20,
			"priority": 5,
		},
		"triggers": map[string]interface{}{
			"clock":     map[string]interface{}{"enabled": false},
			"intervene": map[string]interface{}{"enabled": true},
		},
	}
	configJSON, _ := json.Marshal(robotConfig)

	err := qb.Table(tableName).Insert([]map[string]interface{}{
		{
			"member_id":       memberID,
			"team_id":         teamID,
			"member_type":     "robot",
			"display_name":    "Test Robot " + memberID,
			"status":          "active",
			"role_id":         "member",
			"autonomous_mode": true,
			"robot_status":    "idle",
			"robot_config":    string(configJSON),
		},
	})
	if err != nil {
		t.Fatalf("Failed to insert %s: %v", memberID, err)
	}
}

// setupIntegrationRobotInactive creates an inactive robot (should not be loaded)
func setupIntegrationRobotInactive(t *testing.T, memberID, teamID string) {
	m := model.Select("__yao.member")
	tableName := m.MetaData.Table.Name
	qb := capsule.Query()

	robotConfig := map[string]interface{}{
		"identity": map[string]interface{}{
			"role": "Inactive Robot",
		},
		"triggers": map[string]interface{}{
			"clock": map[string]interface{}{"enabled": true},
		},
	}
	configJSON, _ := json.Marshal(robotConfig)

	err := qb.Table(tableName).Insert([]map[string]interface{}{
		{
			"member_id":       memberID,
			"team_id":         teamID,
			"member_type":     "robot",
			"display_name":    "Inactive Robot " + memberID,
			"status":          "inactive", // Inactive status
			"role_id":         "member",
			"autonomous_mode": true,
			"robot_status":    "paused",
			"robot_config":    string(configJSON),
		},
	})
	if err != nil {
		t.Fatalf("Failed to insert %s: %v", memberID, err)
	}
}

// setupIntegrationRobotNonAutonomous creates a robot with autonomous_mode=false
func setupIntegrationRobotNonAutonomous(t *testing.T, memberID, teamID string) {
	m := model.Select("__yao.member")
	tableName := m.MetaData.Table.Name
	qb := capsule.Query()

	robotConfig := map[string]interface{}{
		"identity": map[string]interface{}{
			"role": "Non-Autonomous Robot",
		},
	}
	configJSON, _ := json.Marshal(robotConfig)

	err := qb.Table(tableName).Insert([]map[string]interface{}{
		{
			"member_id":       memberID,
			"team_id":         teamID,
			"member_type":     "robot",
			"display_name":    "Non-Autonomous Robot " + memberID,
			"status":          "active",
			"role_id":         "member",
			"autonomous_mode": false, // Not autonomous
			"robot_status":    "idle",
			"robot_config":    string(configJSON),
		},
	})
	if err != nil {
		t.Fatalf("Failed to insert %s: %v", memberID, err)
	}
}

// cleanupIntegrationRobots removes all integration test robots
func cleanupIntegrationRobots(t *testing.T) {
	qb := capsule.Query()
	m := model.Select("__yao.member")
	tableName := m.MetaData.Table.Name

	// Delete all robots with member_id starting with "robot_integ_"
	// Using LIKE pattern for cleanup
	_, err := qb.Table(tableName).Where("member_id", "like", "robot_integ_%").Delete()
	if err != nil {
		// Log but don't fail - cleanup errors are not critical
		t.Logf("Warning: cleanup error: %v", err)
	}
}
