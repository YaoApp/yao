package manager_test

// Integration tests for Clock trigger modes
// Tests all three clock modes: times, interval, daemon
// Includes timezone handling and day-of-week filtering

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

// createClockTestManager creates a manager with mock executor for clock tests
func createClockTestManager(t *testing.T, tickInterval time.Duration, workerSize, queueSize int) (*manager.Manager, *executor.DryRunExecutor) {
	exec := executor.NewDryRunWithDelay(0)
	config := &manager.Config{
		TickInterval: tickInterval,
		PoolConfig:   &pool.Config{WorkerSize: workerSize, QueueSize: queueSize},
		Executor:     exec,
	}
	m := manager.NewWithConfig(config)
	return m, exec
}

// ==================== Times Mode Tests ====================

// TestIntegrationClockTimesMode tests the times mode clock trigger
func TestIntegrationClockTimesMode(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	cleanupIntegrationRobots(t)
	defer cleanupIntegrationRobots(t)

	t.Run("triggers at configured time", func(t *testing.T) {
		// Clean up before each subtest to ensure isolation
		cleanupIntegrationRobots(t)

		setupClockTestRobot(t, "robot_integ_clock_times1", "team_integ_clock", map[string]interface{}{
			"mode":  "times",
			"times": []string{"09:00", "14:00", "17:00"},
			"days":  []string{"Mon", "Tue", "Wed", "Thu", "Fri"},
			"tz":    "Asia/Shanghai",
		})

		m, exec := createClockTestManager(t, 100*time.Millisecond, 3, 20)

		err := m.Start()
		require.NoError(t, err)
		defer m.Stop()

		// Verify robot is loaded into cache
		robot := m.Cache().Get("robot_integ_clock_times1")
		require.NotNil(t, robot, "Robot should be loaded into cache")

		exec.Reset()

		// Trigger at 09:00 on Wednesday
		loc, _ := time.LoadLocation("Asia/Shanghai")
		now := time.Date(2025, 1, 15, 9, 0, 0, 0, loc) // Wednesday 09:00

		ctx := types.NewContext(context.Background(), nil)
		err = m.Tick(ctx, now)
		assert.NoError(t, err)

		time.Sleep(300 * time.Millisecond)
		assert.GreaterOrEqual(t, exec.ExecCount(), 1, "Should trigger at 09:00")
	})

	t.Run("does not trigger at non-configured time", func(t *testing.T) {
		// Clean up before each subtest to ensure isolation
		cleanupIntegrationRobots(t)

		setupClockTestRobot(t, "robot_integ_clock_times2", "team_integ_clock", map[string]interface{}{
			"mode":  "times",
			"times": []string{"09:00", "14:00"},
			"days":  []string{"Mon", "Tue", "Wed", "Thu", "Fri"},
			"tz":    "Asia/Shanghai",
		})

		m, exec := createClockTestManager(t, 100*time.Millisecond, 3, 20)

		err := m.Start()
		require.NoError(t, err)
		defer m.Stop()

		exec.Reset()

		// Trigger at 10:30 (not configured)
		loc, _ := time.LoadLocation("Asia/Shanghai")
		now := time.Date(2025, 1, 15, 10, 30, 0, 0, loc) // Wednesday 10:30

		ctx := types.NewContext(context.Background(), nil)
		err = m.Tick(ctx, now)
		assert.NoError(t, err)

		time.Sleep(200 * time.Millisecond)
		assert.Equal(t, 0, exec.ExecCount(), "Should not trigger at non-configured time")
	})

	t.Run("does not trigger on non-configured day", func(t *testing.T) {
		// Clean up before each subtest to ensure isolation
		cleanupIntegrationRobots(t)

		setupClockTestRobot(t, "robot_integ_clock_times3", "team_integ_clock", map[string]interface{}{
			"mode":  "times",
			"times": []string{"09:00"},
			"days":  []string{"Mon", "Tue", "Wed", "Thu", "Fri"}, // Weekdays only
			"tz":    "Asia/Shanghai",
		})

		m, exec := createClockTestManager(t, 100*time.Millisecond, 3, 20)

		err := m.Start()
		require.NoError(t, err)
		defer m.Stop()

		exec.Reset()

		// Trigger at 09:00 on Saturday (not configured)
		loc, _ := time.LoadLocation("Asia/Shanghai")
		now := time.Date(2025, 1, 18, 9, 0, 0, 0, loc) // Saturday 09:00

		ctx := types.NewContext(context.Background(), nil)
		err = m.Tick(ctx, now)
		assert.NoError(t, err)

		time.Sleep(200 * time.Millisecond)
		assert.Equal(t, 0, exec.ExecCount(), "Should not trigger on Saturday")
	})

	t.Run("wildcard days matches all days", func(t *testing.T) {
		// Clean up before each subtest to ensure isolation
		cleanupIntegrationRobots(t)

		setupClockTestRobot(t, "robot_integ_clock_times4", "team_integ_clock", map[string]interface{}{
			"mode":  "times",
			"times": []string{"09:00"},
			"days":  []string{"*"}, // All days
			"tz":    "Asia/Shanghai",
		})

		m, exec := createClockTestManager(t, 100*time.Millisecond, 3, 20)

		err := m.Start()
		require.NoError(t, err)
		defer m.Stop()

		exec.Reset()

		// Trigger at 09:00 on Saturday
		loc, _ := time.LoadLocation("Asia/Shanghai")
		now := time.Date(2025, 1, 18, 9, 0, 0, 0, loc) // Saturday 09:00

		ctx := types.NewContext(context.Background(), nil)
		err = m.Tick(ctx, now)
		assert.NoError(t, err)

		time.Sleep(300 * time.Millisecond)
		assert.GreaterOrEqual(t, exec.ExecCount(), 1, "Should trigger on Saturday with wildcard days")
	})

	t.Run("dedup prevents double trigger in same minute", func(t *testing.T) {
		// Clean up before each subtest to ensure isolation
		cleanupIntegrationRobots(t)

		setupClockTestRobot(t, "robot_integ_clock_times5", "team_integ_clock", map[string]interface{}{
			"mode":  "times",
			"times": []string{"09:00"},
			"days":  []string{"*"},
			"tz":    "Asia/Shanghai",
		})

		m, exec := createClockTestManager(t, 100*time.Millisecond, 3, 20)

		err := m.Start()
		require.NoError(t, err)
		defer m.Stop()

		exec.Reset()

		loc, _ := time.LoadLocation("Asia/Shanghai")
		ctx := types.NewContext(context.Background(), nil)

		// First tick at 09:00:00
		now1 := time.Date(2025, 1, 15, 9, 0, 0, 0, loc)
		err = m.Tick(ctx, now1)
		assert.NoError(t, err)
		time.Sleep(200 * time.Millisecond)
		firstCount := exec.ExecCount()
		assert.GreaterOrEqual(t, firstCount, 1, "First tick should trigger")

		// Second tick at 09:00:30 (same minute)
		now2 := time.Date(2025, 1, 15, 9, 0, 30, 0, loc)
		err = m.Tick(ctx, now2)
		assert.NoError(t, err)
		time.Sleep(200 * time.Millisecond)

		// Should not trigger again in same minute
		assert.Equal(t, firstCount, exec.ExecCount(), "Should not trigger twice in same minute")
	})
}

// ==================== Interval Mode Tests ====================

// TestIntegrationClockIntervalMode tests the interval mode clock trigger
func TestIntegrationClockIntervalMode(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	cleanupIntegrationRobots(t)
	defer cleanupIntegrationRobots(t)

	t.Run("triggers on first run", func(t *testing.T) {
		// Clean up before each subtest to ensure isolation
		cleanupIntegrationRobots(t)

		setupClockTestRobot(t, "robot_integ_clock_interval1", "team_integ_clock", map[string]interface{}{
			"mode":  "interval",
			"every": "30m",
		})

		m, exec := createClockTestManager(t, 100*time.Millisecond, 3, 20)

		err := m.Start()
		require.NoError(t, err)
		defer m.Stop()

		exec.Reset()

		ctx := types.NewContext(context.Background(), nil)
		now := time.Now()
		err = m.Tick(ctx, now)
		assert.NoError(t, err)

		time.Sleep(300 * time.Millisecond)
		assert.GreaterOrEqual(t, exec.ExecCount(), 1, "Should trigger on first run")
	})

	t.Run("triggers after interval passed", func(t *testing.T) {
		// Clean up before each subtest to ensure isolation
		cleanupIntegrationRobots(t)

		setupClockTestRobot(t, "robot_integ_clock_interval2", "team_integ_clock", map[string]interface{}{
			"mode":  "interval",
			"every": "100ms", // Short interval for testing
		})

		m, exec := createClockTestManager(t, 50*time.Millisecond, 3, 20)

		err := m.Start()
		require.NoError(t, err)
		defer m.Stop()

		exec.Reset()

		ctx := types.NewContext(context.Background(), nil)

		// First tick
		now1 := time.Now()
		err = m.Tick(ctx, now1)
		assert.NoError(t, err)
		time.Sleep(200 * time.Millisecond)
		firstCount := exec.ExecCount()
		assert.GreaterOrEqual(t, firstCount, 1, "First tick should trigger")

		// Wait for interval to pass
		time.Sleep(150 * time.Millisecond)

		// Second tick after interval
		now2 := time.Now()
		err = m.Tick(ctx, now2)
		assert.NoError(t, err)
		time.Sleep(200 * time.Millisecond)

		// Should have triggered again
		assert.Greater(t, exec.ExecCount(), firstCount, "Should trigger again after interval")
	})

	t.Run("does not trigger before interval passed", func(t *testing.T) {
		// Clean up before each subtest to ensure isolation
		cleanupIntegrationRobots(t)

		setupClockTestRobot(t, "robot_integ_clock_interval3", "team_integ_clock", map[string]interface{}{
			"mode":  "interval",
			"every": "1h", // Long interval
		})

		m, exec := createClockTestManager(t, 100*time.Millisecond, 3, 20)

		err := m.Start()
		require.NoError(t, err)
		defer m.Stop()

		exec.Reset()

		ctx := types.NewContext(context.Background(), nil)

		// First tick
		now1 := time.Now()
		err = m.Tick(ctx, now1)
		assert.NoError(t, err)
		time.Sleep(200 * time.Millisecond)
		firstCount := exec.ExecCount()
		assert.GreaterOrEqual(t, firstCount, 1, "First tick should trigger")

		// Second tick immediately (interval not passed)
		now2 := now1.Add(1 * time.Minute) // Only 1 minute later
		err = m.Tick(ctx, now2)
		assert.NoError(t, err)
		time.Sleep(200 * time.Millisecond)

		// Should not trigger again
		assert.Equal(t, firstCount, exec.ExecCount(), "Should not trigger before interval")
	})
}

// ==================== Daemon Mode Tests ====================

// TestIntegrationClockDaemonMode tests the daemon mode clock trigger
func TestIntegrationClockDaemonMode(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	cleanupIntegrationRobots(t)
	defer cleanupIntegrationRobots(t)

	t.Run("triggers when robot can run", func(t *testing.T) {
		setupClockTestRobot(t, "robot_integ_clock_daemon1", "team_integ_clock", map[string]interface{}{
			"mode":    "daemon",
			"timeout": "5m",
		})

		m, exec := createClockTestManager(t, 100*time.Millisecond, 3, 20)

		err := m.Start()
		require.NoError(t, err)
		defer m.Stop()

		exec.Reset()

		ctx := types.NewContext(context.Background(), nil)
		err = m.Tick(ctx, time.Now())
		assert.NoError(t, err)

		time.Sleep(300 * time.Millisecond)
		assert.GreaterOrEqual(t, exec.ExecCount(), 1, "Daemon should trigger when idle")
	})

	t.Run("respects quota limit", func(t *testing.T) {
		// Create daemon robot with Max=1
		setupClockTestRobotWithQuota(t, "robot_integ_clock_daemon2", "team_integ_clock",
			map[string]interface{}{
				"mode":    "daemon",
				"timeout": "5m",
			},
			1, 5, 5) // Max=1, Queue=5

		m, exec := createClockTestManager(t, 50*time.Millisecond, 5, 20)

		err := m.Start()
		require.NoError(t, err)
		defer m.Stop()

		exec.Reset()

		ctx := types.NewContext(context.Background(), nil)

		// Trigger multiple times rapidly
		for i := 0; i < 5; i++ {
			err = m.Tick(ctx, time.Now())
			assert.NoError(t, err)
			time.Sleep(60 * time.Millisecond)
		}

		// Robot should respect quota (Max=1)
		robot := m.Cache().Get("robot_integ_clock_daemon2")
		assert.NotNil(t, robot)
		// Running count should be at most Max
		assert.LessOrEqual(t, robot.RunningCount(), 1, "Should respect quota limit")
	})
}

// ==================== Timezone Tests ====================

// TestIntegrationClockTimezone tests timezone handling
func TestIntegrationClockTimezone(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	cleanupIntegrationRobots(t)
	defer cleanupIntegrationRobots(t)

	t.Run("respects robot timezone", func(t *testing.T) {
		// Robot configured for Asia/Shanghai (UTC+8)
		setupClockTestRobot(t, "robot_integ_clock_tz1", "team_integ_clock", map[string]interface{}{
			"mode":  "times",
			"times": []string{"09:00"},
			"days":  []string{"*"},
			"tz":    "Asia/Shanghai",
		})

		m, exec := createClockTestManager(t, 100*time.Millisecond, 3, 20)

		err := m.Start()
		require.NoError(t, err)
		defer m.Stop()

		exec.Reset()

		ctx := types.NewContext(context.Background(), nil)

		// 09:00 in Shanghai = 01:00 UTC
		shanghai, _ := time.LoadLocation("Asia/Shanghai")
		shanghaiTime := time.Date(2025, 1, 15, 9, 0, 0, 0, shanghai)

		err = m.Tick(ctx, shanghaiTime)
		assert.NoError(t, err)

		time.Sleep(300 * time.Millisecond)
		assert.GreaterOrEqual(t, exec.ExecCount(), 1, "Should trigger at 09:00 Shanghai time")
	})

	t.Run("different timezone same UTC time", func(t *testing.T) {
		// Robot 1: Asia/Shanghai at 09:00 (UTC+8) = 01:00 UTC
		setupClockTestRobot(t, "robot_integ_clock_tz2", "team_integ_clock", map[string]interface{}{
			"mode":  "times",
			"times": []string{"09:00"},
			"days":  []string{"*"},
			"tz":    "Asia/Shanghai",
		})

		// Robot 2: America/New_York at 09:00 (UTC-5) = 14:00 UTC
		setupClockTestRobot(t, "robot_integ_clock_tz3", "team_integ_clock", map[string]interface{}{
			"mode":  "times",
			"times": []string{"09:00"},
			"days":  []string{"*"},
			"tz":    "America/New_York",
		})

		m, exec := createClockTestManager(t, 100*time.Millisecond, 3, 20)

		err := m.Start()
		require.NoError(t, err)
		defer m.Stop()

		exec.Reset()

		ctx := types.NewContext(context.Background(), nil)

		// Test at 01:00 UTC (09:00 Shanghai)
		utcTime := time.Date(2025, 1, 15, 1, 0, 0, 0, time.UTC)
		err = m.Tick(ctx, utcTime)
		assert.NoError(t, err)

		time.Sleep(300 * time.Millisecond)

		// Only Shanghai robot should trigger
		execCount := exec.ExecCount()
		assert.GreaterOrEqual(t, execCount, 1, "Shanghai robot should trigger")
		// New York robot should not trigger (it's 20:00 in NY)
	})
}

// ==================== Edge Cases ====================

// TestIntegrationClockEdgeCases tests edge cases in clock triggering
func TestIntegrationClockEdgeCases(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	cleanupIntegrationRobots(t)
	defer cleanupIntegrationRobots(t)

	t.Run("robot with clock disabled is skipped", func(t *testing.T) {
		// Create robot with clock trigger disabled
		m := model.Select("__yao.member")
		tableName := m.MetaData.Table.Name
		qb := capsule.Query()

		robotConfig := map[string]interface{}{
			"identity": map[string]interface{}{"role": "Clock Disabled Robot"},
			"triggers": map[string]interface{}{
				"clock": map[string]interface{}{"enabled": false},
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
				"member_id":       "robot_integ_clock_disabled",
				"team_id":         "team_integ_clock",
				"member_type":     "robot",
				"display_name":    "Clock Disabled Robot",
				"status":          "active",
				"role_id":         "member",
				"autonomous_mode": true,
				"robot_status":    "idle",
				"robot_config":    string(configJSON),
			},
		})
		require.NoError(t, err)

		mgr, exec := createClockTestManager(t, 100*time.Millisecond, 3, 20)

		err = mgr.Start()
		require.NoError(t, err)
		defer mgr.Stop()

		exec.Reset()

		// Trigger at matching time
		loc, _ := time.LoadLocation("Asia/Shanghai")
		now := time.Date(2025, 1, 15, 9, 0, 0, 0, loc)

		ctx := types.NewContext(context.Background(), nil)
		err = mgr.Tick(ctx, now)
		assert.NoError(t, err)

		time.Sleep(200 * time.Millisecond)
		assert.Equal(t, 0, exec.ExecCount(), "Clock disabled robot should not trigger")
	})

	t.Run("paused robot is skipped", func(t *testing.T) {
		// Create paused robot
		m := model.Select("__yao.member")
		tableName := m.MetaData.Table.Name
		qb := capsule.Query()

		robotConfig := map[string]interface{}{
			"identity": map[string]interface{}{"role": "Paused Robot"},
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
				"member_id":       "robot_integ_clock_paused",
				"team_id":         "team_integ_clock",
				"member_type":     "robot",
				"display_name":    "Paused Robot",
				"status":          "active",
				"role_id":         "member",
				"autonomous_mode": true,
				"robot_status":    "paused", // Paused status
				"robot_config":    string(configJSON),
			},
		})
		require.NoError(t, err)

		mgr, exec := createClockTestManager(t, 100*time.Millisecond, 3, 20)

		err = mgr.Start()
		require.NoError(t, err)
		defer mgr.Stop()

		exec.Reset()

		// Trigger at matching time
		loc, _ := time.LoadLocation("Asia/Shanghai")
		now := time.Date(2025, 1, 15, 9, 0, 0, 0, loc)

		ctx := types.NewContext(context.Background(), nil)
		err = mgr.Tick(ctx, now)
		assert.NoError(t, err)

		time.Sleep(200 * time.Millisecond)
		assert.Equal(t, 0, exec.ExecCount(), "Paused robot should not trigger")
	})

	t.Run("robot without clock config is skipped", func(t *testing.T) {
		// Create robot without clock config
		m := model.Select("__yao.member")
		tableName := m.MetaData.Table.Name
		qb := capsule.Query()

		robotConfig := map[string]interface{}{
			"identity": map[string]interface{}{"role": "No Clock Robot"},
			"triggers": map[string]interface{}{
				"clock": map[string]interface{}{"enabled": true},
			},
			// No clock config
		}
		configJSON, _ := json.Marshal(robotConfig)

		err := qb.Table(tableName).Insert([]map[string]interface{}{
			{
				"member_id":       "robot_integ_clock_noconfig",
				"team_id":         "team_integ_clock",
				"member_type":     "robot",
				"display_name":    "No Clock Config Robot",
				"status":          "active",
				"role_id":         "member",
				"autonomous_mode": true,
				"robot_status":    "idle",
				"robot_config":    string(configJSON),
			},
		})
		require.NoError(t, err)

		mgr, exec := createClockTestManager(t, 100*time.Millisecond, 3, 20)

		err = mgr.Start()
		require.NoError(t, err)
		defer mgr.Stop()

		exec.Reset()

		ctx := types.NewContext(context.Background(), nil)
		err = mgr.Tick(ctx, time.Now())
		assert.NoError(t, err)

		time.Sleep(200 * time.Millisecond)
		assert.Equal(t, 0, exec.ExecCount(), "Robot without clock config should not trigger")
	})
}

// ==================== Test Data Setup Helpers ====================

// setupClockTestRobot creates a robot with specified clock config
func setupClockTestRobot(t *testing.T, memberID, teamID string, clockConfig map[string]interface{}) {
	setupClockTestRobotWithQuota(t, memberID, teamID, clockConfig, 3, 20, 5)
}

// setupClockTestRobotWithQuota creates a robot with specified clock config and quota
func setupClockTestRobotWithQuota(t *testing.T, memberID, teamID string, clockConfig map[string]interface{}, max, queue, priority int) {
	m := model.Select("__yao.member")
	tableName := m.MetaData.Table.Name
	qb := capsule.Query()

	robotConfig := map[string]interface{}{
		"identity": map[string]interface{}{
			"role": "Clock Test Robot " + memberID,
		},
		"quota": map[string]interface{}{
			"max":      max,
			"queue":    queue,
			"priority": priority,
		},
		"triggers": map[string]interface{}{
			"clock": map[string]interface{}{"enabled": true},
		},
		"clock": clockConfig,
	}
	configJSON, _ := json.Marshal(robotConfig)

	err := qb.Table(tableName).Insert([]map[string]interface{}{
		{
			"member_id":       memberID,
			"team_id":         teamID,
			"member_type":     "robot",
			"display_name":    "Clock Test Robot " + memberID,
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
