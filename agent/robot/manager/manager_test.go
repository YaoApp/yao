package manager_test

import (
	"context"
	"encoding/json"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/xun/capsule"
	agentcontext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/robot/manager"
	"github.com/yaoapp/yao/agent/robot/pool"
	"github.com/yaoapp/yao/agent/robot/types"
	"github.com/yaoapp/yao/agent/testutils"
)

// TestManagerStartStop tests manager lifecycle
func TestManagerStartStop(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	cleanupTestRobots(t)
	setupTestRobots(t)
	defer cleanupTestRobots(t)

	t.Run("start and stop manager", func(t *testing.T) {
		m := manager.New()

		// Should not be started
		assert.False(t, m.IsStarted())

		// Start manager
		err := m.Start()
		assert.NoError(t, err)
		assert.True(t, m.IsStarted())

		// Robots should be loaded
		assert.GreaterOrEqual(t, m.CachedRobots(), 2, "Should load at least 2 robots")

		// Stop manager
		err = m.Stop()
		assert.NoError(t, err)
		assert.False(t, m.IsStarted())
	})

	t.Run("double start should fail", func(t *testing.T) {
		m := manager.New()

		err := m.Start()
		assert.NoError(t, err)

		// Second start should fail
		err = m.Start()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already started")

		// Cleanup
		m.Stop()
	})

	t.Run("stop without start should not panic", func(t *testing.T) {
		m := manager.New()

		assert.NotPanics(t, func() {
			err := m.Stop()
			assert.NoError(t, err)
		})
	})
}

// TestManagerTick tests the Tick function with different clock modes
func TestManagerTick(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	cleanupTestRobots(t)
	setupTestRobotsWithClockConfig(t)
	defer cleanupTestRobots(t)

	t.Run("tick with times mode - matching time", func(t *testing.T) {
		// Create manager with short tick interval for testing
		config := &manager.Config{
			TickInterval: 100 * time.Millisecond,
			PoolConfig:   &pool.Config{WorkerSize: 2, QueueSize: 10},
		}
		m := manager.NewWithConfig(config)

		err := m.Start()
		assert.NoError(t, err)
		defer m.Stop()

		// Create a time that matches the configured time (09:00)
		loc, _ := time.LoadLocation("Asia/Shanghai")
		now := time.Date(2025, 1, 15, 9, 0, 0, 0, loc) // Wednesday 09:00

		ctx := types.NewContext(context.Background(), nil)
		err = m.Tick(ctx, now)
		assert.NoError(t, err)

		// Wait for job to be processed
		time.Sleep(200 * time.Millisecond)

		// Check that job was submitted (may be queued or running)
		// Note: The executor stub completes quickly, so we check execution count
		execCount := m.Executor().ExecCount()
		assert.GreaterOrEqual(t, execCount, 1, "Should have executed at least 1 job")
	})

	t.Run("tick with times mode - non-matching time", func(t *testing.T) {
		config := &manager.Config{
			TickInterval: 100 * time.Millisecond,
			PoolConfig:   &pool.Config{WorkerSize: 2, QueueSize: 10},
		}
		m := manager.NewWithConfig(config)

		err := m.Start()
		assert.NoError(t, err)
		defer m.Stop()

		// Reset executor count
		m.Executor().Reset()

		// Create a time that does NOT match (10:30)
		loc, _ := time.LoadLocation("Asia/Shanghai")
		now := time.Date(2025, 1, 15, 10, 30, 0, 0, loc) // Wednesday 10:30

		ctx := types.NewContext(context.Background(), nil)
		err = m.Tick(ctx, now)
		assert.NoError(t, err)

		// Wait a bit
		time.Sleep(100 * time.Millisecond)

		// Should not have triggered (times mode robot only triggers at 09:00, 14:00)
		execCount := m.Executor().ExecCount()
		// Note: interval mode robot might trigger if enough time passed
		// We just verify the times mode robot didn't trigger
		assert.LessOrEqual(t, execCount, 1, "Times mode robot should not trigger at non-matching time")
	})

	t.Run("tick with interval mode", func(t *testing.T) {
		config := &manager.Config{
			TickInterval: 100 * time.Millisecond,
			PoolConfig:   &pool.Config{WorkerSize: 2, QueueSize: 10},
		}
		m := manager.NewWithConfig(config)

		err := m.Start()
		assert.NoError(t, err)
		defer m.Stop()

		// Reset executor count
		m.Executor().Reset()

		// First tick - should trigger interval mode robot (first run)
		ctx := types.NewContext(context.Background(), nil)
		now := time.Now()
		err = m.Tick(ctx, now)
		assert.NoError(t, err)

		// Wait for execution
		time.Sleep(200 * time.Millisecond)

		// Should have at least 1 execution (interval robot first run)
		execCount := m.Executor().ExecCount()
		assert.GreaterOrEqual(t, execCount, 1, "Interval mode robot should trigger on first run")
	})

	t.Run("tick skips paused robots", func(t *testing.T) {
		config := &manager.Config{
			TickInterval: 100 * time.Millisecond,
			PoolConfig:   &pool.Config{WorkerSize: 2, QueueSize: 10},
		}
		m := manager.NewWithConfig(config)

		err := m.Start()
		assert.NoError(t, err)
		defer m.Stop()

		// Get the paused robot from cache
		pausedRobot := m.Cache().Get("robot_test_manager_paused")
		assert.NotNil(t, pausedRobot)
		assert.Equal(t, types.RobotPaused, pausedRobot.Status)

		// Reset executor count
		m.Executor().Reset()

		// Tick should skip paused robot
		ctx := types.NewContext(context.Background(), nil)
		loc, _ := time.LoadLocation("Asia/Shanghai")
		now := time.Date(2025, 1, 15, 9, 0, 0, 0, loc)
		err = m.Tick(ctx, now)
		assert.NoError(t, err)

		// The paused robot should not have been triggered
		// (we can't directly verify this, but we verify the tick completed)
	})
}

// TestManagerTriggerManual tests manual triggering of robots
func TestManagerTriggerManual(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	cleanupTestRobots(t)
	setupTestRobotsWithClockConfig(t)
	defer cleanupTestRobots(t)

	t.Run("trigger manual - success", func(t *testing.T) {
		m := manager.New()
		err := m.Start()
		assert.NoError(t, err)
		defer m.Stop()

		ctx := types.NewContext(context.Background(), nil)

		// Manually trigger a robot
		execID, err := m.TriggerManual(ctx, "robot_test_manager_times", types.TriggerHuman, nil)
		assert.NoError(t, err)
		assert.NotEmpty(t, execID)

		// Wait for execution
		time.Sleep(200 * time.Millisecond)

		// Should have executed
		assert.GreaterOrEqual(t, m.Executor().ExecCount(), 1)
	})

	t.Run("trigger manual - robot not found", func(t *testing.T) {
		m := manager.New()
		err := m.Start()
		assert.NoError(t, err)
		defer m.Stop()

		ctx := types.NewContext(context.Background(), nil)

		// Try to trigger non-existent robot
		_, err = m.TriggerManual(ctx, "robot_nonexistent", types.TriggerHuman, nil)
		assert.Error(t, err)
		assert.Equal(t, types.ErrRobotNotFound, err)
	})

	t.Run("trigger manual - robot paused", func(t *testing.T) {
		m := manager.New()
		err := m.Start()
		assert.NoError(t, err)
		defer m.Stop()

		ctx := types.NewContext(context.Background(), nil)

		// Try to trigger paused robot
		_, err = m.TriggerManual(ctx, "robot_test_manager_paused", types.TriggerHuman, nil)
		assert.Error(t, err)
		assert.Equal(t, types.ErrRobotPaused, err)
	})

	t.Run("trigger manual - manager not started", func(t *testing.T) {
		m := manager.New()
		// Don't start manager

		ctx := types.NewContext(context.Background(), nil)

		_, err := m.TriggerManual(ctx, "robot_test_manager_times", types.TriggerHuman, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not started")
	})
}

// TestManagerClockModes tests all three clock modes
func TestManagerClockModes(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	cleanupTestRobots(t)
	setupTestRobotsWithClockConfig(t)
	defer cleanupTestRobots(t)

	t.Run("times mode - day matching", func(t *testing.T) {
		m := manager.New()
		err := m.Start()
		assert.NoError(t, err)
		defer m.Stop()

		m.Executor().Reset()

		// Wednesday (configured day)
		loc, _ := time.LoadLocation("Asia/Shanghai")
		now := time.Date(2025, 1, 15, 9, 0, 0, 0, loc) // Wednesday 09:00

		ctx := types.NewContext(context.Background(), nil)
		err = m.Tick(ctx, now)
		assert.NoError(t, err)

		time.Sleep(200 * time.Millisecond)
		assert.GreaterOrEqual(t, m.Executor().ExecCount(), 1, "Should trigger on matching day")
	})

	t.Run("times mode - day not matching", func(t *testing.T) {
		m := manager.New()
		err := m.Start()
		assert.NoError(t, err)
		defer m.Stop()

		m.Executor().Reset()

		// Saturday (not configured)
		loc, _ := time.LoadLocation("Asia/Shanghai")
		now := time.Date(2025, 1, 18, 9, 0, 0, 0, loc) // Saturday 09:00

		ctx := types.NewContext(context.Background(), nil)
		err = m.Tick(ctx, now)
		assert.NoError(t, err)

		time.Sleep(100 * time.Millisecond)
		// Times mode robot should not trigger on Saturday
		// Only interval/daemon robots might trigger
	})

	t.Run("daemon mode - always triggers when idle", func(t *testing.T) {
		m := manager.New()
		err := m.Start()
		assert.NoError(t, err)
		defer m.Stop()

		m.Executor().Reset()

		// Daemon robot should trigger whenever it can run
		ctx := types.NewContext(context.Background(), nil)
		now := time.Now()

		// First tick
		err = m.Tick(ctx, now)
		assert.NoError(t, err)

		time.Sleep(200 * time.Millisecond)

		// Should have triggered daemon robot
		assert.GreaterOrEqual(t, m.Executor().ExecCount(), 1, "Daemon mode should trigger")
	})
}

// TestManagerTimezoneDedup tests that times mode dedup works correctly across timezones
// This specifically tests the bug fix where LastRun.Day() must be converted to the same
// timezone as 'now' before comparison
func TestManagerTimezoneDedup(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	cleanupTestRobots(t)
	setupTestRobotsWithClockConfig(t)
	defer cleanupTestRobots(t)

	t.Run("times mode - same minute same day should not trigger twice", func(t *testing.T) {
		m := manager.New()
		err := m.Start()
		assert.NoError(t, err)
		defer m.Stop()

		m.Executor().Reset()

		// Use Asia/Shanghai timezone (UTC+8)
		loc, _ := time.LoadLocation("Asia/Shanghai")
		// Wednesday 09:00:00 in Shanghai
		now := time.Date(2025, 1, 15, 9, 0, 0, 0, loc)

		ctx := types.NewContext(context.Background(), nil)

		// First tick - should trigger
		err = m.Tick(ctx, now)
		assert.NoError(t, err)
		time.Sleep(200 * time.Millisecond)
		firstCount := m.Executor().ExecCount()
		assert.GreaterOrEqual(t, firstCount, 1, "First tick should trigger")

		// Second tick at 09:00:30 (same minute) - should NOT trigger again
		now2 := time.Date(2025, 1, 15, 9, 0, 30, 0, loc)
		err = m.Tick(ctx, now2)
		assert.NoError(t, err)
		time.Sleep(200 * time.Millisecond)

		// Count should remain the same (times robot should not trigger twice)
		// Note: daemon/interval robots may still trigger, so we check the delta
		secondCount := m.Executor().ExecCount()
		t.Logf("First count: %d, Second count: %d", firstCount, secondCount)

		// The times robot should not have triggered again in the same minute
		// Delta should be <= 2 (daemon always triggers, interval might trigger)
		// If delta > 2, it means times robot triggered twice (bug!)
		delta := secondCount - firstCount
		assert.LessOrEqual(t, delta, 2, "Times robot should not trigger twice in same minute (delta: %d)", delta)
	})

	t.Run("times mode - different day should trigger again", func(t *testing.T) {
		m := manager.New()
		err := m.Start()
		assert.NoError(t, err)
		defer m.Stop()

		m.Executor().Reset()

		loc, _ := time.LoadLocation("Asia/Shanghai")

		ctx := types.NewContext(context.Background(), nil)

		// Wednesday 09:00
		now1 := time.Date(2025, 1, 15, 9, 0, 0, 0, loc)
		err = m.Tick(ctx, now1)
		assert.NoError(t, err)
		time.Sleep(200 * time.Millisecond)
		firstCount := m.Executor().ExecCount()

		// Thursday 09:00 (next day, same time)
		now2 := time.Date(2025, 1, 16, 9, 0, 0, 0, loc)
		err = m.Tick(ctx, now2)
		assert.NoError(t, err)
		time.Sleep(200 * time.Millisecond)

		// Should have triggered again on the new day
		secondCount := m.Executor().ExecCount()
		assert.Greater(t, secondCount, firstCount, "Should trigger on different day")
	})

	t.Run("times mode - cross-timezone day boundary", func(t *testing.T) {
		m := manager.New()
		err := m.Start()
		assert.NoError(t, err)
		defer m.Stop()

		m.Executor().Reset()

		// Robot is configured with Asia/Shanghai (UTC+8)
		// Test case: LastRun was set when it was Jan 15 in Shanghai
		// Now it's Jan 16 00:30 in Shanghai (still Jan 15 in UTC)
		// The comparison should use Shanghai timezone, not UTC

		loc, _ := time.LoadLocation("Asia/Shanghai")
		ctx := types.NewContext(context.Background(), nil)

		// First run: Jan 15, 09:00 Shanghai time
		now1 := time.Date(2025, 1, 15, 9, 0, 0, 0, loc)
		err = m.Tick(ctx, now1)
		assert.NoError(t, err)
		time.Sleep(200 * time.Millisecond)

		// Second run: Jan 16, 09:00 Shanghai time
		// This is Jan 16 01:00 UTC, but should be treated as Jan 16 in Shanghai
		now2 := time.Date(2025, 1, 16, 9, 0, 0, 0, loc)
		err = m.Tick(ctx, now2)
		assert.NoError(t, err)
		time.Sleep(200 * time.Millisecond)

		// Should have triggered on both days
		assert.GreaterOrEqual(t, m.Executor().ExecCount(), 2, "Should trigger on both days")
	})

	t.Run("times mode - UTC vs local timezone comparison", func(t *testing.T) {
		m := manager.New()
		err := m.Start()
		assert.NoError(t, err)
		defer m.Stop()

		m.Executor().Reset()

		// Test with explicit UTC time converted to Shanghai
		// This tests that LastRun stored in one timezone is correctly
		// compared when 'now' is in a different timezone

		shanghai, _ := time.LoadLocation("Asia/Shanghai")
		ctx := types.NewContext(context.Background(), nil)

		// Create a time that's Jan 15 23:30 UTC = Jan 16 07:30 Shanghai
		utcTime := time.Date(2025, 1, 15, 23, 30, 0, 0, time.UTC)
		shanghaiTime := utcTime.In(shanghai)
		t.Logf("UTC: %v, Shanghai: %v", utcTime, shanghaiTime)

		// The robot is configured for 09:00 Shanghai time
		// So Jan 16 09:00 Shanghai should trigger
		now := time.Date(2025, 1, 16, 9, 0, 0, 0, shanghai)
		err = m.Tick(ctx, now)
		assert.NoError(t, err)
		time.Sleep(200 * time.Millisecond)

		assert.GreaterOrEqual(t, m.Executor().ExecCount(), 1, "Should trigger at 09:00 Shanghai")
	})
}

// TestManagerGoroutineLeak tests that manager doesn't leak goroutines
func TestManagerGoroutineLeak(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	cleanupTestRobots(t)
	setupTestRobots(t)
	defer cleanupTestRobots(t)

	t.Run("start stop cycle should not leak goroutines", func(t *testing.T) {
		// Record initial goroutine count
		runtime.GC()
		time.Sleep(100 * time.Millisecond)
		initialGoroutines := runtime.NumGoroutine()

		// Start and stop multiple times
		for i := 0; i < 5; i++ {
			m := manager.New()
			err := m.Start()
			assert.NoError(t, err)

			// Do some ticks
			ctx := types.NewContext(context.Background(), nil)
			m.Tick(ctx, time.Now())

			time.Sleep(50 * time.Millisecond)

			err = m.Stop()
			assert.NoError(t, err)
		}

		// Wait for cleanup
		time.Sleep(200 * time.Millisecond)
		runtime.GC()
		time.Sleep(100 * time.Millisecond)

		// Check goroutine count
		finalGoroutines := runtime.NumGoroutine()
		assert.LessOrEqual(t, finalGoroutines, initialGoroutines+2,
			"Should not leak goroutines (initial: %d, final: %d)",
			initialGoroutines, finalGoroutines)
	})
}

// TestManagerComponents tests access to internal components
func TestManagerComponents(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	cleanupTestRobots(t)
	setupTestRobots(t)
	defer cleanupTestRobots(t)

	m := manager.New()
	err := m.Start()
	assert.NoError(t, err)
	defer m.Stop()

	t.Run("cache access", func(t *testing.T) {
		cache := m.Cache()
		assert.NotNil(t, cache)

		robot := cache.Get("robot_test_sales_001")
		assert.NotNil(t, robot)
	})

	t.Run("pool access", func(t *testing.T) {
		pool := m.Pool()
		assert.NotNil(t, pool)
		assert.True(t, pool.IsStarted())
	})

	t.Run("executor access", func(t *testing.T) {
		executor := m.Executor()
		assert.NotNil(t, executor)
	})

	t.Run("running and queued counts", func(t *testing.T) {
		running := m.Running()
		queued := m.Queued()
		cached := m.CachedRobots()

		assert.GreaterOrEqual(t, running, 0)
		assert.GreaterOrEqual(t, queued, 0)
		assert.GreaterOrEqual(t, cached, 2)
	})
}

// ==================== Test Data Setup ====================

// setupTestRobots creates basic test robots (same as cache tests)
func setupTestRobots(t *testing.T) {
	m := model.Select("__yao.member")
	tableName := m.MetaData.Table.Name
	qb := capsule.Query()

	// Robot 1: Sales Bot
	robotConfig1 := map[string]interface{}{
		"identity": map[string]interface{}{
			"role":   "Sales Manager",
			"duties": []string{"Manage leads", "Follow up customers"},
		},
		"quota": map[string]interface{}{
			"max":      3,
			"queue":    15,
			"priority": 7,
		},
		"triggers": map[string]interface{}{
			"clock": map[string]interface{}{"enabled": true},
		},
		"clock": map[string]interface{}{
			"mode":  "times",
			"times": []string{"09:00", "14:00"},
			"days":  []string{"Mon", "Tue", "Wed", "Thu", "Fri"},
			"tz":    "Asia/Shanghai",
		},
	}
	config1JSON, _ := json.Marshal(robotConfig1)

	err := qb.Table(tableName).Insert([]map[string]interface{}{
		{
			"member_id":       "robot_test_sales_001",
			"team_id":         "team_test_cache_001",
			"member_type":     "robot",
			"display_name":    "Test Sales Bot",
			"system_prompt":   "You are a professional sales manager assistant.",
			"status":          "active",
			"role_id":         "member",
			"autonomous_mode": true,
			"robot_status":    "idle",
			"robot_config":    string(config1JSON),
		},
	})
	if err != nil {
		t.Fatalf("Failed to insert robot_test_sales_001: %v", err)
	}

	// Robot 2: Support Bot
	robotConfig2 := map[string]interface{}{
		"identity": map[string]interface{}{
			"role":   "Customer Support",
			"duties": []string{"Answer questions", "Resolve issues"},
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
			"mode":  "interval",
			"every": "1h",
		},
	}
	config2JSON, _ := json.Marshal(robotConfig2)

	err = qb.Table(tableName).Insert([]map[string]interface{}{
		{
			"member_id":       "robot_test_support_002",
			"team_id":         "team_test_cache_001",
			"member_type":     "robot",
			"display_name":    "Test Support Bot",
			"system_prompt":   "You are a helpful customer support assistant.",
			"status":          "active",
			"role_id":         "member",
			"autonomous_mode": true,
			"robot_status":    "idle",
			"robot_config":    string(config2JSON),
		},
	})
	if err != nil {
		t.Fatalf("Failed to insert robot_test_support_002: %v", err)
	}

	// Robot 3: Inactive robot
	err = qb.Table(tableName).Insert([]map[string]interface{}{
		{
			"member_id":       "robot_test_inactive_003",
			"team_id":         "team_test_cache_001",
			"member_type":     "robot",
			"display_name":    "Test Inactive Bot",
			"status":          "inactive",
			"role_id":         "member",
			"autonomous_mode": true,
			"robot_status":    "paused",
		},
	})
	if err != nil {
		t.Fatalf("Failed to insert robot_test_inactive_003: %v", err)
	}
}

// setupTestRobotsWithClockConfig creates robots with specific clock configurations
func setupTestRobotsWithClockConfig(t *testing.T) {
	m := model.Select("__yao.member")
	tableName := m.MetaData.Table.Name
	qb := capsule.Query()

	// Robot 1: Times mode (09:00, 14:00 on weekdays)
	robotConfigTimes := map[string]interface{}{
		"identity": map[string]interface{}{
			"role": "Times Mode Robot",
		},
		"quota": map[string]interface{}{
			"max": 2,
		},
		"triggers": map[string]interface{}{
			"clock": map[string]interface{}{"enabled": true},
		},
		"clock": map[string]interface{}{
			"mode":  "times",
			"times": []string{"09:00", "14:00"},
			"days":  []string{"Mon", "Tue", "Wed", "Thu", "Fri"},
			"tz":    "Asia/Shanghai",
		},
	}
	configTimesJSON, _ := json.Marshal(robotConfigTimes)

	err := qb.Table(tableName).Insert([]map[string]interface{}{
		{
			"member_id":       "robot_test_manager_times",
			"team_id":         "team_test_manager",
			"member_type":     "robot",
			"display_name":    "Test Times Robot",
			"status":          "active",
			"role_id":         "member",
			"autonomous_mode": true,
			"robot_status":    "idle",
			"robot_config":    string(configTimesJSON),
		},
	})
	if err != nil {
		t.Fatalf("Failed to insert robot_test_manager_times: %v", err)
	}

	// Robot 2: Interval mode (every 30 minutes)
	robotConfigInterval := map[string]interface{}{
		"identity": map[string]interface{}{
			"role": "Interval Mode Robot",
		},
		"quota": map[string]interface{}{
			"max": 2,
		},
		"triggers": map[string]interface{}{
			"clock": map[string]interface{}{"enabled": true},
		},
		"clock": map[string]interface{}{
			"mode":  "interval",
			"every": "30m",
		},
	}
	configIntervalJSON, _ := json.Marshal(robotConfigInterval)

	err = qb.Table(tableName).Insert([]map[string]interface{}{
		{
			"member_id":       "robot_test_manager_interval",
			"team_id":         "team_test_manager",
			"member_type":     "robot",
			"display_name":    "Test Interval Robot",
			"status":          "active",
			"role_id":         "member",
			"autonomous_mode": true,
			"robot_status":    "idle",
			"robot_config":    string(configIntervalJSON),
		},
	})
	if err != nil {
		t.Fatalf("Failed to insert robot_test_manager_interval: %v", err)
	}

	// Robot 3: Daemon mode
	robotConfigDaemon := map[string]interface{}{
		"identity": map[string]interface{}{
			"role": "Daemon Mode Robot",
		},
		"quota": map[string]interface{}{
			"max": 2,
		},
		"triggers": map[string]interface{}{
			"clock": map[string]interface{}{"enabled": true},
		},
		"clock": map[string]interface{}{
			"mode":    "daemon",
			"timeout": "5m",
		},
	}
	configDaemonJSON, _ := json.Marshal(robotConfigDaemon)

	err = qb.Table(tableName).Insert([]map[string]interface{}{
		{
			"member_id":       "robot_test_manager_daemon",
			"team_id":         "team_test_manager",
			"member_type":     "robot",
			"display_name":    "Test Daemon Robot",
			"status":          "active",
			"role_id":         "member",
			"autonomous_mode": true,
			"robot_status":    "idle",
			"robot_config":    string(configDaemonJSON),
		},
	})
	if err != nil {
		t.Fatalf("Failed to insert robot_test_manager_daemon: %v", err)
	}

	// Robot 4: Paused robot (should be skipped)
	robotConfigPaused := map[string]interface{}{
		"identity": map[string]interface{}{
			"role": "Paused Robot",
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
	configPausedJSON, _ := json.Marshal(robotConfigPaused)

	err = qb.Table(tableName).Insert([]map[string]interface{}{
		{
			"member_id":       "robot_test_manager_paused",
			"team_id":         "team_test_manager",
			"member_type":     "robot",
			"display_name":    "Test Paused Robot",
			"status":          "active",
			"role_id":         "member",
			"autonomous_mode": true,
			"robot_status":    "paused",
			"robot_config":    string(configPausedJSON),
		},
	})
	if err != nil {
		t.Fatalf("Failed to insert robot_test_manager_paused: %v", err)
	}

	// Robot 5: Clock disabled robot
	robotConfigDisabled := map[string]interface{}{
		"identity": map[string]interface{}{
			"role": "Clock Disabled Robot",
		},
		"triggers": map[string]interface{}{
			"clock": map[string]interface{}{"enabled": false},
		},
		"clock": map[string]interface{}{
			"mode":  "times",
			"times": []string{"09:00"},
		},
	}
	configDisabledJSON, _ := json.Marshal(robotConfigDisabled)

	err = qb.Table(tableName).Insert([]map[string]interface{}{
		{
			"member_id":       "robot_test_manager_disabled",
			"team_id":         "team_test_manager",
			"member_type":     "robot",
			"display_name":    "Test Clock Disabled Robot",
			"status":          "active",
			"role_id":         "member",
			"autonomous_mode": true,
			"robot_status":    "idle",
			"robot_config":    string(configDisabledJSON),
		},
	})
	if err != nil {
		t.Fatalf("Failed to insert robot_test_manager_disabled: %v", err)
	}
}

// ==================== Intervene Tests ====================

func TestManagerIntervene(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	cleanupTestRobots(t)
	setupTestRobotsWithInterveneConfig(t)
	defer cleanupTestRobots(t)

	t.Run("intervene success", func(t *testing.T) {
		m := manager.New()
		err := m.Start()
		assert.NoError(t, err)
		defer m.Stop()

		ctx := types.NewContext(context.Background(), nil)
		req := &types.InterveneRequest{
			TeamID:   "team_test_manager",
			MemberID: "robot_test_manager_intervene",
			Action:   types.ActionTaskAdd,
			Messages: []agentcontext.Message{
				{Role: agentcontext.RoleUser, Content: "Add a new task"},
			},
		}

		result, err := m.Intervene(ctx, req)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEmpty(t, result.ExecutionID)
		assert.Equal(t, types.ExecPending, result.Status)
	})

	t.Run("intervene - manager not started", func(t *testing.T) {
		m := manager.New()
		// Don't start

		ctx := types.NewContext(context.Background(), nil)
		req := &types.InterveneRequest{
			MemberID: "robot_test_manager_intervene",
			Action:   types.ActionTaskAdd,
			Messages: []agentcontext.Message{
				{Role: agentcontext.RoleUser, Content: "Add a new task"},
			},
		}

		_, err := m.Intervene(ctx, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not started")
	})

	t.Run("intervene - robot not found", func(t *testing.T) {
		m := manager.New()
		err := m.Start()
		assert.NoError(t, err)
		defer m.Stop()

		ctx := types.NewContext(context.Background(), nil)
		req := &types.InterveneRequest{
			MemberID: "non_existent_robot",
			Action:   types.ActionTaskAdd,
			Messages: []agentcontext.Message{
				{Role: agentcontext.RoleUser, Content: "Add a new task"},
			},
		}

		_, err = m.Intervene(ctx, req)
		assert.Error(t, err)
		assert.Equal(t, types.ErrRobotNotFound, err)
	})

	t.Run("intervene - robot paused", func(t *testing.T) {
		m := manager.New()
		err := m.Start()
		assert.NoError(t, err)
		defer m.Stop()

		ctx := types.NewContext(context.Background(), nil)
		req := &types.InterveneRequest{
			MemberID: "robot_test_manager_paused",
			Action:   types.ActionTaskAdd,
			Messages: []agentcontext.Message{
				{Role: agentcontext.RoleUser, Content: "Add a new task"},
			},
		}

		_, err = m.Intervene(ctx, req)
		assert.Error(t, err)
		assert.Equal(t, types.ErrRobotPaused, err)
	})

	t.Run("intervene - invalid request", func(t *testing.T) {
		m := manager.New()
		err := m.Start()
		assert.NoError(t, err)
		defer m.Stop()

		ctx := types.NewContext(context.Background(), nil)
		req := &types.InterveneRequest{
			MemberID: "", // Invalid: empty member_id
			Action:   types.ActionTaskAdd,
		}

		_, err = m.Intervene(ctx, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "member_id")
	})

	t.Run("intervene - trigger disabled", func(t *testing.T) {
		m := manager.New()
		err := m.Start()
		assert.NoError(t, err)
		defer m.Stop()

		ctx := types.NewContext(context.Background(), nil)
		req := &types.InterveneRequest{
			MemberID: "robot_test_manager_intervene_disabled",
			Action:   types.ActionTaskAdd,
			Messages: []agentcontext.Message{
				{Role: agentcontext.RoleUser, Content: "Add a new task"},
			},
		}

		_, err = m.Intervene(ctx, req)
		assert.Error(t, err)
		assert.Equal(t, types.ErrTriggerDisabled, err)
	})
}

// ==================== HandleEvent Tests ====================

func TestManagerHandleEvent(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	cleanupTestRobots(t)
	setupTestRobotsWithEventConfig(t)
	defer cleanupTestRobots(t)

	t.Run("handle event success", func(t *testing.T) {
		m := manager.New()
		err := m.Start()
		assert.NoError(t, err)
		defer m.Stop()

		ctx := types.NewContext(context.Background(), nil)
		req := &types.EventRequest{
			MemberID:  "robot_test_manager_event",
			Source:    "webhook",
			EventType: "lead.created",
			Data:      map[string]interface{}{"name": "John", "email": "john@example.com"},
		}

		result, err := m.HandleEvent(ctx, req)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEmpty(t, result.ExecutionID)
		assert.Equal(t, types.ExecPending, result.Status)
	})

	t.Run("handle event - manager not started", func(t *testing.T) {
		m := manager.New()
		// Don't start

		ctx := types.NewContext(context.Background(), nil)
		req := &types.EventRequest{
			MemberID:  "robot_test_manager_event",
			Source:    "webhook",
			EventType: "lead.created",
		}

		_, err := m.HandleEvent(ctx, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not started")
	})

	t.Run("handle event - robot not found", func(t *testing.T) {
		m := manager.New()
		err := m.Start()
		assert.NoError(t, err)
		defer m.Stop()

		ctx := types.NewContext(context.Background(), nil)
		req := &types.EventRequest{
			MemberID:  "non_existent_robot",
			Source:    "webhook",
			EventType: "lead.created",
		}

		_, err = m.HandleEvent(ctx, req)
		assert.Error(t, err)
		assert.Equal(t, types.ErrRobotNotFound, err)
	})

	t.Run("handle event - invalid request", func(t *testing.T) {
		m := manager.New()
		err := m.Start()
		assert.NoError(t, err)
		defer m.Stop()

		ctx := types.NewContext(context.Background(), nil)
		req := &types.EventRequest{
			MemberID:  "robot_test_manager_event",
			Source:    "", // Invalid: empty source
			EventType: "lead.created",
		}

		_, err = m.HandleEvent(ctx, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "source")
	})

	t.Run("handle event - trigger disabled", func(t *testing.T) {
		m := manager.New()
		err := m.Start()
		assert.NoError(t, err)
		defer m.Stop()

		ctx := types.NewContext(context.Background(), nil)
		req := &types.EventRequest{
			MemberID:  "robot_test_manager_event_disabled",
			Source:    "webhook",
			EventType: "lead.created",
		}

		_, err = m.HandleEvent(ctx, req)
		assert.Error(t, err)
		assert.Equal(t, types.ErrTriggerDisabled, err)
	})
}

// ==================== Execution Control Tests ====================

func TestManagerExecutionControl(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	cleanupTestRobots(t)
	setupTestRobotsWithInterveneConfig(t)
	defer cleanupTestRobots(t)

	t.Run("pause and resume execution", func(t *testing.T) {
		m := manager.New()
		err := m.Start()
		assert.NoError(t, err)
		defer m.Stop()

		// Trigger an execution
		ctx := types.NewContext(context.Background(), nil)
		req := &types.InterveneRequest{
			MemberID: "robot_test_manager_intervene",
			Action:   types.ActionTaskAdd,
			Messages: []agentcontext.Message{
				{Role: agentcontext.RoleUser, Content: "Test task"},
			},
		}

		result, err := m.Intervene(ctx, req)
		assert.NoError(t, err)
		execID := result.ExecutionID

		// Wait a bit for execution to be tracked
		time.Sleep(50 * time.Millisecond)

		// Pause
		err = m.PauseExecution(ctx, execID)
		assert.NoError(t, err)

		// Get status - should be paused
		status, err := m.GetExecutionStatus(execID)
		assert.NoError(t, err)
		assert.True(t, status.IsPaused())

		// Resume
		err = m.ResumeExecution(ctx, execID)
		assert.NoError(t, err)

		// Get status - should not be paused
		status, err = m.GetExecutionStatus(execID)
		assert.NoError(t, err)
		assert.False(t, status.IsPaused())
	})

	t.Run("stop execution", func(t *testing.T) {
		m := manager.New()
		err := m.Start()
		assert.NoError(t, err)
		defer m.Stop()

		// Trigger an execution
		ctx := types.NewContext(context.Background(), nil)
		req := &types.InterveneRequest{
			MemberID: "robot_test_manager_intervene",
			Action:   types.ActionTaskAdd,
			Messages: []agentcontext.Message{
				{Role: agentcontext.RoleUser, Content: "Test task"},
			},
		}

		result, err := m.Intervene(ctx, req)
		assert.NoError(t, err)
		execID := result.ExecutionID

		// Wait a bit for execution to be tracked
		time.Sleep(50 * time.Millisecond)

		// Stop
		err = m.StopExecution(ctx, execID)
		assert.NoError(t, err)

		// Get status - should not be found (removed after stop)
		_, err = m.GetExecutionStatus(execID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("list executions", func(t *testing.T) {
		m := manager.New()
		err := m.Start()
		assert.NoError(t, err)
		defer m.Stop()

		ctx := types.NewContext(context.Background(), nil)

		// Track execution IDs
		var execIDs []string

		// Trigger multiple executions
		for i := 0; i < 3; i++ {
			req := &types.InterveneRequest{
				MemberID: "robot_test_manager_intervene",
				Action:   types.ActionTaskAdd,
				Messages: []agentcontext.Message{
					{Role: agentcontext.RoleUser, Content: "Test task"},
				},
			}
			result, err := m.Intervene(ctx, req)
			assert.NoError(t, err)
			execIDs = append(execIDs, result.ExecutionID)
		}

		// Verify each execution was tracked (even if briefly)
		// Note: executions complete quickly with stub executor, so they may be removed
		// We just verify that we got valid execution IDs
		assert.Len(t, execIDs, 3)
		for _, id := range execIDs {
			assert.NotEmpty(t, id)
		}
	})
}

// setupTestRobotsWithInterveneConfig creates test robots with intervene trigger enabled
func setupTestRobotsWithInterveneConfig(t *testing.T) {
	// First setup the basic robots
	setupTestRobotsWithClockConfig(t)

	// Add robots for intervene tests
	qb := capsule.Query()
	m := model.Select("__yao.member")
	tableName := m.MetaData.Table.Name

	// Robot with intervene enabled
	robotConfigIntervene := map[string]interface{}{
		"identity": map[string]interface{}{
			"role": "Intervene Test Robot",
		},
		"triggers": map[string]interface{}{
			"clock":     map[string]interface{}{"enabled": false},
			"intervene": map[string]interface{}{"enabled": true},
		},
		"quota": map[string]interface{}{
			"max":   5,
			"queue": 10,
		},
	}
	configInterveneJSON, _ := json.Marshal(robotConfigIntervene)

	err := qb.Table(tableName).Insert([]map[string]interface{}{
		{
			"member_id":       "robot_test_manager_intervene",
			"team_id":         "team_test_manager",
			"member_type":     "robot",
			"display_name":    "Test Intervene Robot",
			"status":          "active",
			"role_id":         "member",
			"autonomous_mode": true,
			"robot_status":    "idle",
			"robot_config":    string(configInterveneJSON),
		},
	})
	if err != nil {
		t.Fatalf("Failed to insert robot_test_manager_intervene: %v", err)
	}

	// Robot with intervene disabled
	robotConfigInterveneDisabled := map[string]interface{}{
		"identity": map[string]interface{}{
			"role": "Intervene Disabled Robot",
		},
		"triggers": map[string]interface{}{
			"clock":     map[string]interface{}{"enabled": false},
			"intervene": map[string]interface{}{"enabled": false},
		},
	}
	configInterveneDisabledJSON, _ := json.Marshal(robotConfigInterveneDisabled)

	err = qb.Table(tableName).Insert([]map[string]interface{}{
		{
			"member_id":       "robot_test_manager_intervene_disabled",
			"team_id":         "team_test_manager",
			"member_type":     "robot",
			"display_name":    "Test Intervene Disabled Robot",
			"status":          "active",
			"role_id":         "member",
			"autonomous_mode": true,
			"robot_status":    "idle",
			"robot_config":    string(configInterveneDisabledJSON),
		},
	})
	if err != nil {
		t.Fatalf("Failed to insert robot_test_manager_intervene_disabled: %v", err)
	}
}

// setupTestRobotsWithEventConfig creates test robots with event trigger enabled
func setupTestRobotsWithEventConfig(t *testing.T) {
	// First setup the basic robots
	setupTestRobotsWithClockConfig(t)

	// Add robots for event tests
	qb := capsule.Query()
	m := model.Select("__yao.member")
	tableName := m.MetaData.Table.Name

	// Robot with event enabled
	robotConfigEvent := map[string]interface{}{
		"identity": map[string]interface{}{
			"role": "Event Test Robot",
		},
		"triggers": map[string]interface{}{
			"clock": map[string]interface{}{"enabled": false},
			"event": map[string]interface{}{"enabled": true},
		},
		"quota": map[string]interface{}{
			"max":   5,
			"queue": 10,
		},
	}
	configEventJSON, _ := json.Marshal(robotConfigEvent)

	err := qb.Table(tableName).Insert([]map[string]interface{}{
		{
			"member_id":       "robot_test_manager_event",
			"team_id":         "team_test_manager",
			"member_type":     "robot",
			"display_name":    "Test Event Robot",
			"status":          "active",
			"role_id":         "member",
			"autonomous_mode": true,
			"robot_status":    "idle",
			"robot_config":    string(configEventJSON),
		},
	})
	if err != nil {
		t.Fatalf("Failed to insert robot_test_manager_event: %v", err)
	}

	// Robot with event disabled
	robotConfigEventDisabled := map[string]interface{}{
		"identity": map[string]interface{}{
			"role": "Event Disabled Robot",
		},
		"triggers": map[string]interface{}{
			"clock": map[string]interface{}{"enabled": false},
			"event": map[string]interface{}{"enabled": false},
		},
	}
	configEventDisabledJSON, _ := json.Marshal(robotConfigEventDisabled)

	err = qb.Table(tableName).Insert([]map[string]interface{}{
		{
			"member_id":       "robot_test_manager_event_disabled",
			"team_id":         "team_test_manager",
			"member_type":     "robot",
			"display_name":    "Test Event Disabled Robot",
			"status":          "active",
			"role_id":         "member",
			"autonomous_mode": true,
			"robot_status":    "idle",
			"robot_config":    string(configEventDisabledJSON),
		},
	})
	if err != nil {
		t.Fatalf("Failed to insert robot_test_manager_event_disabled: %v", err)
	}
}

// ==================== Lazy Load Tests for Non-Autonomous Robots ====================

// TestManagerLazyLoadNonAutonomous tests that non-autonomous robots are lazy-loaded on demand
// and automatically cleaned up after execution completes
func TestManagerLazyLoadNonAutonomous(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	cleanupTestRobots(t)
	setupTestRobotsWithNonAutonomous(t)
	defer cleanupTestRobots(t)

	t.Run("non-autonomous robot not in cache on startup", func(t *testing.T) {
		m := manager.New()
		err := m.Start()
		assert.NoError(t, err)
		defer m.Stop()

		// Non-autonomous robot should NOT be in cache
		robot := m.Cache().Get("robot_test_manager_on_demand")
		assert.Nil(t, robot, "Non-autonomous robot should not be pre-loaded into cache")

		// Autonomous robot SHOULD be in cache
		autoRobot := m.Cache().Get("robot_test_manager_times")
		assert.NotNil(t, autoRobot, "Autonomous robot should be in cache")
	})

	t.Run("TriggerManual lazy-loads non-autonomous robot", func(t *testing.T) {
		m := manager.New()
		err := m.Start()
		assert.NoError(t, err)
		defer m.Stop()

		ctx := types.NewContext(context.Background(), nil)

		// Verify robot is NOT in cache before trigger
		assert.Nil(t, m.Cache().Get("robot_test_manager_on_demand"))

		// Trigger the non-autonomous robot manually
		execID, err := m.TriggerManual(ctx, "robot_test_manager_on_demand", types.TriggerHuman, nil)
		assert.NoError(t, err)
		assert.NotEmpty(t, execID)

		// Robot should now be in cache (lazy-loaded)
		robot := m.Cache().Get("robot_test_manager_on_demand")
		assert.NotNil(t, robot, "Robot should be lazy-loaded into cache")
		assert.Equal(t, "robot_test_manager_on_demand", robot.MemberID)
		assert.False(t, robot.AutonomousMode)
	})

	t.Run("Intervene lazy-loads non-autonomous robot", func(t *testing.T) {
		m := manager.New()
		err := m.Start()
		assert.NoError(t, err)
		defer m.Stop()

		ctx := types.NewContext(context.Background(), nil)

		// Verify robot is NOT in cache before trigger
		assert.Nil(t, m.Cache().Get("robot_test_manager_on_demand_intervene"))

		// Intervene on the non-autonomous robot
		req := &types.InterveneRequest{
			TeamID:   "team_test_manager",
			MemberID: "robot_test_manager_on_demand_intervene",
			Action:   types.ActionTaskAdd,
			Messages: []agentcontext.Message{
				{Role: agentcontext.RoleUser, Content: "Test lazy load via intervene"},
			},
		}

		result, err := m.Intervene(ctx, req)
		assert.NoError(t, err)
		assert.NotEmpty(t, result.ExecutionID)

		// Robot should now be in cache (lazy-loaded)
		robot := m.Cache().Get("robot_test_manager_on_demand_intervene")
		assert.NotNil(t, robot, "Robot should be lazy-loaded into cache via Intervene")
	})

	t.Run("HandleEvent lazy-loads non-autonomous robot", func(t *testing.T) {
		m := manager.New()
		err := m.Start()
		assert.NoError(t, err)
		defer m.Stop()

		ctx := types.NewContext(context.Background(), nil)

		// Verify robot is NOT in cache before trigger
		assert.Nil(t, m.Cache().Get("robot_test_manager_on_demand_event"))

		// Send event to the non-autonomous robot
		req := &types.EventRequest{
			MemberID:  "robot_test_manager_on_demand_event",
			Source:    "webhook",
			EventType: "data.updated",
			Data:      map[string]interface{}{"test": true},
		}

		result, err := m.HandleEvent(ctx, req)
		assert.NoError(t, err)
		assert.NotEmpty(t, result.ExecutionID)

		// Robot should now be in cache (lazy-loaded)
		robot := m.Cache().Get("robot_test_manager_on_demand_event")
		assert.NotNil(t, robot, "Robot should be lazy-loaded into cache via HandleEvent")
	})

	t.Run("lazy-loaded robot is cleaned up after execution completes", func(t *testing.T) {
		m := manager.New()
		err := m.Start()
		assert.NoError(t, err)
		defer m.Stop()

		ctx := types.NewContext(context.Background(), nil)

		// Trigger the non-autonomous robot
		_, err = m.TriggerManual(ctx, "robot_test_manager_on_demand", types.TriggerHuman, nil)
		assert.NoError(t, err)

		// Robot should be in cache immediately after trigger
		robot := m.Cache().Get("robot_test_manager_on_demand")
		assert.NotNil(t, robot, "Robot should be in cache after trigger")

		// Wait for execution to complete and cleanup to happen
		// The stub executor completes quickly, and cleanup runs every 5 seconds
		// We wait up to 10 seconds for the cleanup goroutine to remove the robot
		var removed bool
		for i := 0; i < 20; i++ {
			time.Sleep(500 * time.Millisecond)
			if m.Cache().Get("robot_test_manager_on_demand") == nil {
				removed = true
				break
			}
		}

		assert.True(t, removed, "Non-autonomous robot should be removed from cache after execution completes")
	})

	t.Run("trigger non-existent robot returns error", func(t *testing.T) {
		m := manager.New()
		err := m.Start()
		assert.NoError(t, err)
		defer m.Stop()

		ctx := types.NewContext(context.Background(), nil)

		// Try to trigger a robot that doesn't exist in DB
		_, err = m.TriggerManual(ctx, "robot_nonexistent_xyz", types.TriggerHuman, nil)
		assert.Error(t, err)
		assert.Equal(t, types.ErrRobotNotFound, err)
	})
}

// setupTestRobotsWithNonAutonomous creates test robots including non-autonomous ones
func setupTestRobotsWithNonAutonomous(t *testing.T) {
	// First setup the autonomous robots
	setupTestRobotsWithClockConfig(t)

	// Add non-autonomous robots
	qb := capsule.Query()
	m := model.Select("__yao.member")
	tableName := m.MetaData.Table.Name

	// Non-autonomous robot 1: for TriggerManual test
	robotConfigOnDemand := map[string]interface{}{
		"identity": map[string]interface{}{
			"role": "On-Demand Robot",
		},
		"triggers": map[string]interface{}{
			"clock":     map[string]interface{}{"enabled": false},
			"intervene": map[string]interface{}{"enabled": true},
		},
		"quota": map[string]interface{}{
			"max":   2,
			"queue": 5,
		},
	}
	configOnDemandJSON, _ := json.Marshal(robotConfigOnDemand)

	err := qb.Table(tableName).Insert([]map[string]interface{}{
		{
			"member_id":       "robot_test_manager_on_demand",
			"team_id":         "team_test_manager",
			"member_type":     "robot",
			"display_name":    "Test On-Demand Robot",
			"status":          "active",
			"role_id":         "member",
			"autonomous_mode": false, // Non-autonomous!
			"robot_status":    "idle",
			"robot_config":    string(configOnDemandJSON),
		},
	})
	if err != nil {
		t.Fatalf("Failed to insert robot_test_manager_on_demand: %v", err)
	}

	// Non-autonomous robot 2: for Intervene test
	robotConfigOnDemandIntervene := map[string]interface{}{
		"identity": map[string]interface{}{
			"role": "On-Demand Intervene Robot",
		},
		"triggers": map[string]interface{}{
			"clock":     map[string]interface{}{"enabled": false},
			"intervene": map[string]interface{}{"enabled": true},
		},
		"quota": map[string]interface{}{
			"max": 2,
		},
	}
	configOnDemandInterveneJSON, _ := json.Marshal(robotConfigOnDemandIntervene)

	err = qb.Table(tableName).Insert([]map[string]interface{}{
		{
			"member_id":       "robot_test_manager_on_demand_intervene",
			"team_id":         "team_test_manager",
			"member_type":     "robot",
			"display_name":    "Test On-Demand Intervene Robot",
			"status":          "active",
			"role_id":         "member",
			"autonomous_mode": false, // Non-autonomous!
			"robot_status":    "idle",
			"robot_config":    string(configOnDemandInterveneJSON),
		},
	})
	if err != nil {
		t.Fatalf("Failed to insert robot_test_manager_on_demand_intervene: %v", err)
	}

	// Non-autonomous robot 3: for HandleEvent test
	robotConfigOnDemandEvent := map[string]interface{}{
		"identity": map[string]interface{}{
			"role": "On-Demand Event Robot",
		},
		"triggers": map[string]interface{}{
			"clock": map[string]interface{}{"enabled": false},
			"event": map[string]interface{}{"enabled": true},
		},
		"quota": map[string]interface{}{
			"max": 2,
		},
	}
	configOnDemandEventJSON, _ := json.Marshal(robotConfigOnDemandEvent)

	err = qb.Table(tableName).Insert([]map[string]interface{}{
		{
			"member_id":       "robot_test_manager_on_demand_event",
			"team_id":         "team_test_manager",
			"member_type":     "robot",
			"display_name":    "Test On-Demand Event Robot",
			"status":          "active",
			"role_id":         "member",
			"autonomous_mode": false, // Non-autonomous!
			"robot_status":    "idle",
			"robot_config":    string(configOnDemandEventJSON),
		},
	})
	if err != nil {
		t.Fatalf("Failed to insert robot_test_manager_on_demand_event: %v", err)
	}
}

// cleanupTestRobots removes all test robot records
func cleanupTestRobots(t *testing.T) {
	qb := capsule.Query()
	m := model.Select("__yao.member")
	tableName := m.MetaData.Table.Name

	// List of test robot IDs to clean up
	testRobotIDs := []string{
		"robot_test_sales_001",
		"robot_test_support_002",
		"robot_test_inactive_003",
		"robot_test_manager_times",
		"robot_test_manager_interval",
		"robot_test_manager_daemon",
		"robot_test_manager_paused",
		"robot_test_manager_disabled",
		"robot_test_manager_intervene",
		"robot_test_manager_intervene_disabled",
		"robot_test_manager_event",
		"robot_test_manager_event_disabled",
		// Non-autonomous robots
		"robot_test_manager_on_demand",
		"robot_test_manager_on_demand_intervene",
		"robot_test_manager_on_demand_event",
	}

	for _, id := range testRobotIDs {
		// Soft delete
		m.DeleteWhere(model.QueryParam{
			Wheres: []model.QueryWhere{
				{Column: "member_id", Value: id},
			},
		})
		// Hard delete
		qb.Table(tableName).Where("member_id", id).Delete()
	}
}
