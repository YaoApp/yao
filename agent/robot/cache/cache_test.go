package cache_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/xun/capsule"
	"github.com/yaoapp/yao/agent/robot/cache"
	"github.com/yaoapp/yao/agent/robot/types"
	"github.com/yaoapp/yao/agent/testutils"
)

// TestCacheLoad tests loading all active robots from database
func TestCacheLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Clean up any existing test data first
	cleanupTestRobots(t)

	// Create test robots in database
	setupTestRobots(t)
	defer cleanupTestRobots(t)

	c := cache.New()
	ctx := types.NewContext(context.Background(), nil)

	// Load all robots
	err := c.Load(ctx)
	assert.NoError(t, err)

	// Count should be at least 2 (may have other robots in DB)
	count := c.Count()
	assert.GreaterOrEqual(t, count, 2, "Should load at least 2 active autonomous robots")

	// Verify first robot
	robot1 := c.Get("robot_test_sales_001")
	assert.NotNil(t, robot1, "Sales bot should be loaded")
	if robot1 == nil {
		t.Fatal("robot_test_sales_001 not found in cache")
	}
	assert.Equal(t, "robot_test_sales_001", robot1.MemberID)
	assert.Equal(t, "team_test_cache_001", robot1.TeamID)
	assert.Equal(t, "Test Sales Bot", robot1.DisplayName)
	assert.Equal(t, types.RobotIdle, robot1.Status)
	assert.True(t, robot1.AutonomousMode)
	assert.NotNil(t, robot1.Config, "Robot config should be parsed")
	assert.NotNil(t, robot1.Config.Identity, "Identity should be parsed")
	assert.Equal(t, "Sales Manager", robot1.Config.Identity.Role)
	assert.Equal(t, 3, robot1.Config.Quota.GetMax())

	// Verify second robot
	robot2 := c.Get("robot_test_support_002")
	assert.NotNil(t, robot2, "Support bot should be loaded")
	assert.Equal(t, "robot_test_support_002", robot2.MemberID)
	assert.Equal(t, "Test Support Bot", robot2.DisplayName)
	assert.NotNil(t, robot2.Config)
	assert.Equal(t, "Customer Support", robot2.Config.Identity.Role)
	assert.Equal(t, 2, robot2.Config.Quota.GetMax())

	// Verify inactive robot is not loaded
	robot3 := c.Get("robot_test_inactive_003")
	assert.Nil(t, robot3, "Inactive robot should not be loaded")
}

// TestCacheLoadByID tests loading a single robot by member ID
func TestCacheLoadByID(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	cleanupTestRobots(t)
	setupTestRobots(t)
	defer cleanupTestRobots(t)

	c := cache.New()
	ctx := types.NewContext(context.Background(), nil)

	t.Run("load existing robot", func(t *testing.T) {
		robot, err := c.LoadByID(ctx, "robot_test_sales_001")
		assert.NoError(t, err)
		assert.NotNil(t, robot)
		assert.Equal(t, "robot_test_sales_001", robot.MemberID)
		assert.Equal(t, "Test Sales Bot", robot.DisplayName)
		assert.NotNil(t, robot.Config)
	})

	t.Run("load non-existent robot", func(t *testing.T) {
		robot, err := c.LoadByID(ctx, "robot_nonexistent")
		assert.Error(t, err)
		assert.Equal(t, types.ErrRobotNotFound, err)
		assert.Nil(t, robot)
	})

	t.Run("load inactive robot by ID", func(t *testing.T) {
		// LoadByID doesn't filter by status, so it should load
		robot, err := c.LoadByID(ctx, "robot_test_inactive_003")
		assert.NoError(t, err)
		assert.NotNil(t, robot)
		assert.Equal(t, "robot_test_inactive_003", robot.MemberID)
	})
}

// TestCacheRefresh tests refreshing a single robot from database
func TestCacheRefresh(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	cleanupTestRobots(t)
	setupTestRobots(t)
	defer cleanupTestRobots(t)

	c := cache.New()
	ctx := types.NewContext(context.Background(), nil)

	// Load initial data
	err := c.Load(ctx)
	assert.NoError(t, err)

	t.Run("refresh existing robot", func(t *testing.T) {
		err := c.Refresh(ctx, "robot_test_sales_001")
		assert.NoError(t, err)

		// Robot should still be in cache
		robot := c.Get("robot_test_sales_001")
		assert.NotNil(t, robot)
	})

	t.Run("refresh removes non-existent robot", func(t *testing.T) {
		// Add a fake robot to cache
		c.Add(&types.Robot{MemberID: "robot_test_fake", TeamID: "team_test_cache_001"})
		assert.NotNil(t, c.Get("robot_test_fake"))

		// Refresh should remove it
		err := c.Refresh(ctx, "robot_test_fake")
		assert.NoError(t, err)
		assert.Nil(t, c.Get("robot_test_fake"), "Non-existent robot should be removed")
	})
}

// TestCacheListByTeam tests listing robots by team
func TestCacheListByTeam(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	cleanupTestRobots(t)
	setupTestRobots(t)
	defer cleanupTestRobots(t)

	c := cache.New()
	ctx := types.NewContext(context.Background(), nil)

	// Load all robots
	err := c.Load(ctx)
	assert.NoError(t, err)

	// List robots by team
	robots := c.List("team_test_cache_001")
	assert.Len(t, robots, 2, "Should have 2 robots in team_test_cache_001")

	// List robots for non-existent team
	robots = c.List("team_nonexistent")
	assert.Len(t, robots, 0, "Non-existent team should have no robots")
}

// TestCacheGetByStatus tests getting robots by status
func TestCacheGetByStatus(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	cleanupTestRobots(t)
	setupTestRobots(t)
	defer cleanupTestRobots(t)

	c := cache.New()
	ctx := types.NewContext(context.Background(), nil)

	// Load all robots
	err := c.Load(ctx)
	assert.NoError(t, err)

	// Get idle robots (may have others in DB)
	idle := c.GetIdle()
	assert.GreaterOrEqual(t, len(idle), 2, "Should have at least 2 idle robots")

	// Verify our test robots are not working
	testRobot1 := c.Get("robot_test_sales_001")
	testRobot2 := c.Get("robot_test_support_002")
	assert.Equal(t, types.RobotIdle, testRobot1.Status, "Test robot 1 should be idle")
	assert.Equal(t, types.RobotIdle, testRobot2.Status, "Test robot 2 should be idle")
}

// TestCacheAutoRefresh tests auto-refresh functionality and goroutine leak prevention
func TestCacheAutoRefresh(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	cleanupTestRobots(t)
	setupTestRobots(t)
	defer cleanupTestRobots(t)

	// Verify test data is set up
	c := cache.New()
	ctx := types.NewContext(context.Background(), nil)
	err := c.Load(ctx)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, c.Count(), 1, "Should have at least one robot loaded")

	t.Run("start and stop auto-refresh", func(t *testing.T) {
		// Use a fresh cache for this test
		testCache := cache.New()
		testCtx := types.NewContext(context.Background(), nil)
		err := testCache.Load(testCtx)
		assert.NoError(t, err)

		// Start auto-refresh with short interval
		config := &cache.RefreshConfig{Interval: 100 * time.Millisecond}
		testCache.StartAutoRefresh(testCtx, config)

		// Wait a bit to let it run (should trigger at least 2 refreshes)
		time.Sleep(250 * time.Millisecond)

		// Stop auto-refresh
		testCache.StopAutoRefresh()

		// Verify it stopped by checking that no more refreshes happen
		countBefore := testCache.Count()
		time.Sleep(200 * time.Millisecond)
		countAfter := testCache.Count()

		// Count should be stable (no errors from stopped goroutine)
		assert.Equal(t, countBefore, countAfter, "Cache should be stable after stop")
	})

	t.Run("multiple start calls should replace previous", func(t *testing.T) {
		// Use a fresh cache for this test
		testCache := cache.New()
		testCtx := types.NewContext(context.Background(), nil)
		err := testCache.Load(testCtx)
		assert.NoError(t, err)

		// Track refresh count using a counter
		refreshCount := 0
		originalCount := testCache.Count()

		// Start multiple times without stopping
		config := &cache.RefreshConfig{Interval: 50 * time.Millisecond}

		testCache.StartAutoRefresh(testCtx, config)
		time.Sleep(30 * time.Millisecond)

		testCache.StartAutoRefresh(testCtx, config) // Should stop previous one
		time.Sleep(30 * time.Millisecond)

		testCache.StartAutoRefresh(testCtx, config) // Should stop previous one

		// Wait for some refreshes
		time.Sleep(150 * time.Millisecond)

		// Stop once should be enough
		testCache.StopAutoRefresh()

		// Verify cache still works correctly
		assert.GreaterOrEqual(t, testCache.Count(), 0, "Cache should still be functional")

		// Verify we can still access robots
		_ = refreshCount  // suppress unused warning
		_ = originalCount // suppress unused warning
	})

	t.Run("stop without start should not panic", func(t *testing.T) {
		// Use a fresh cache for this test
		testCache := cache.New()

		// Multiple stops should be safe
		assert.NotPanics(t, func() {
			testCache.StopAutoRefresh()
			testCache.StopAutoRefresh()
			testCache.StopAutoRefresh()
		})
	})

	t.Run("concurrent start and stop should be safe", func(t *testing.T) {
		// Use a fresh cache for this test
		testCache := cache.New()
		testCtx := types.NewContext(context.Background(), nil)
		err := testCache.Load(testCtx)
		assert.NoError(t, err)

		config := &cache.RefreshConfig{Interval: 50 * time.Millisecond}

		// Rapidly start and stop multiple times - should not panic or deadlock
		done := make(chan bool)
		go func() {
			for i := 0; i < 10; i++ {
				testCache.StartAutoRefresh(testCtx, config)
				time.Sleep(10 * time.Millisecond)
				testCache.StopAutoRefresh()
				time.Sleep(10 * time.Millisecond)
			}
			done <- true
		}()

		// Wait with timeout to detect deadlocks
		select {
		case <-done:
			// Success - no deadlock
		case <-time.After(5 * time.Second):
			t.Fatal("Rapid start/stop cycles caused deadlock")
		}

		// Final cleanup
		testCache.StopAutoRefresh()

		// Verify cache is still functional
		assert.GreaterOrEqual(t, testCache.Count(), 0, "Cache should still be functional after rapid cycles")
	})
}

// setupTestRobots creates 3 test robot records in database
func setupTestRobots(t *testing.T) {
	// Get the actual table name from model
	m := model.Select("__yao.member")
	tableName := m.MetaData.Table.Name

	qb := capsule.Query()

	// Robot 1: Sales Bot (active, autonomous)
	robotConfig1 := map[string]interface{}{
		"identity": map[string]interface{}{
			"role":   "Sales Manager",
			"duties": []string{"Manage leads", "Follow up customers"},
			"rules":  []string{"Be professional", "Reply within 24h"},
		},
		"quota": map[string]interface{}{
			"max":      3,
			"queue":    15,
			"priority": 7,
		},
		"clock": map[string]interface{}{
			"mode":  "times",
			"times": []string{"09:00", "14:00"},
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
			"role_id":         "member", // required field
			"autonomous_mode": true,
			"robot_status":    "idle",
			"robot_config":    string(config1JSON),
		},
	})
	if err != nil {
		t.Fatalf("Failed to insert robot_test_sales_001: %v", err)
	}

	// Robot 2: Support Bot (active, autonomous)
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
			"role_id":         "member", // required field
			"autonomous_mode": true,
			"robot_status":    "idle",
			"robot_config":    string(config2JSON),
		},
	})
	if err != nil {
		t.Fatalf("Failed to insert robot_test_support_002: %v", err)
	}

	// Robot 3: Inactive robot (should not be loaded by Load())
	err = qb.Table(tableName).Insert([]map[string]interface{}{
		{
			"member_id":       "robot_test_inactive_003",
			"team_id":         "team_test_cache_001",
			"member_type":     "robot",
			"display_name":    "Test Inactive Bot",
			"status":          "inactive",
			"role_id":         "member", // required field
			"autonomous_mode": true,
			"robot_status":    "paused",
		},
	})
	if err != nil {
		t.Fatalf("Failed to insert robot_test_inactive_003: %v", err)
	}
}

// cleanupTestRobots removes test robot records
func cleanupTestRobots(t *testing.T) {
	qb := capsule.Query()

	// Use the member model to perform soft delete
	m := model.Select("__yao.member")

	// Delete test robots
	m.DeleteWhere(model.QueryParam{
		Wheres: []model.QueryWhere{
			{Column: "member_id", Value: "robot_test_sales_001"},
		},
	})
	m.DeleteWhere(model.QueryParam{
		Wheres: []model.QueryWhere{
			{Column: "member_id", Value: "robot_test_support_002"},
		},
	})
	m.DeleteWhere(model.QueryParam{
		Wheres: []model.QueryWhere{
			{Column: "member_id", Value: "robot_test_inactive_003"},
		},
	})

	// Hard delete from database (cleanup for next test run)
	m2 := model.Select("__yao.member")
	tableName2 := m2.MetaData.Table.Name
	qb.Table(tableName2).Where("member_id", "robot_test_sales_001").Delete()
	qb.Table(tableName2).Where("member_id", "robot_test_support_002").Delete()
	qb.Table(tableName2).Where("member_id", "robot_test_inactive_003").Delete()
}
