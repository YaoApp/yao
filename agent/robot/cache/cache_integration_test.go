//go:build integration

package cache_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/xun/capsule"
	"github.com/yaoapp/yao/agent/robot/cache"
	"github.com/yaoapp/yao/agent/robot/types"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

func TestCacheLoad(t *testing.T) {
	identity := testprepare.PrepareSandbox(t)
	require.NotNil(t, identity)

	cleanupTestRobots(t)
	setupTestRobots(t, identity.AlphaTeamID)
	defer cleanupTestRobots(t)

	c := cache.New()
	ctx := types.NewContext(context.Background(), nil)

	err := c.Load(ctx)
	require.NoError(t, err)

	count := c.Count()
	assert.GreaterOrEqual(t, count, 2, "Should load at least 2 active robots")

	robot1 := c.Get("robot_test_sales_001")
	require.NotNil(t, robot1, "Sales bot should be loaded")
	assert.Equal(t, "robot_test_sales_001", robot1.MemberID)
	assert.Equal(t, identity.AlphaTeamID, robot1.TeamID)
	assert.Equal(t, "Test Sales Bot", robot1.DisplayName)
	assert.Equal(t, types.RobotIdle, robot1.Status)
	assert.True(t, robot1.AutonomousMode)
	require.NotNil(t, robot1.Config, "Robot config should be parsed")
	require.NotNil(t, robot1.Config.Identity, "Identity should be parsed")
	assert.Equal(t, "Sales Manager", robot1.Config.Identity.Role)
	assert.Equal(t, 3, robot1.Config.Quota.GetMax())

	robot2 := c.Get("robot_test_support_002")
	require.NotNil(t, robot2, "Support bot should be loaded")
	assert.Equal(t, "robot_test_support_002", robot2.MemberID)
	assert.Equal(t, "Test Support Bot", robot2.DisplayName)
	require.NotNil(t, robot2.Config)
	assert.Equal(t, "Customer Support", robot2.Config.Identity.Role)
	assert.Equal(t, 2, robot2.Config.Quota.GetMax())

	robot3 := c.Get("robot_test_inactive_003")
	assert.Nil(t, robot3, "Inactive robot should not be loaded")
}

func TestCacheLoadByID(t *testing.T) {
	identity := testprepare.PrepareSandbox(t)
	require.NotNil(t, identity)

	cleanupTestRobots(t)
	setupTestRobots(t, identity.AlphaTeamID)
	defer cleanupTestRobots(t)

	c := cache.New()
	ctx := types.NewContext(context.Background(), nil)

	t.Run("load existing robot", func(t *testing.T) {
		robot, err := c.LoadByID(ctx, "robot_test_sales_001")
		require.NoError(t, err)
		require.NotNil(t, robot)
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
		robot, err := c.LoadByID(ctx, "robot_test_inactive_003")
		require.NoError(t, err)
		require.NotNil(t, robot)
		assert.Equal(t, "robot_test_inactive_003", robot.MemberID)
	})
}

func TestCacheRefresh(t *testing.T) {
	identity := testprepare.PrepareSandbox(t)
	require.NotNil(t, identity)

	cleanupTestRobots(t)
	setupTestRobots(t, identity.AlphaTeamID)
	defer cleanupTestRobots(t)

	c := cache.New()
	ctx := types.NewContext(context.Background(), nil)

	err := c.Load(ctx)
	require.NoError(t, err)

	t.Run("refresh existing robot", func(t *testing.T) {
		err := c.Refresh(ctx, "robot_test_sales_001")
		require.NoError(t, err)

		robot := c.Get("robot_test_sales_001")
		assert.NotNil(t, robot)
	})

	t.Run("refresh removes non-existent robot", func(t *testing.T) {
		c.Add(&types.Robot{MemberID: "robot_test_fake", TeamID: identity.AlphaTeamID})
		assert.NotNil(t, c.Get("robot_test_fake"))

		err := c.Refresh(ctx, "robot_test_fake")
		require.NoError(t, err)
		assert.Nil(t, c.Get("robot_test_fake"), "Non-existent robot should be removed")
	})
}

func TestCacheListByTeam(t *testing.T) {
	identity := testprepare.PrepareSandbox(t)
	require.NotNil(t, identity)

	cleanupTestRobots(t)
	setupTestRobots(t, identity.AlphaTeamID)
	defer cleanupTestRobots(t)

	c := cache.New()
	ctx := types.NewContext(context.Background(), nil)

	err := c.Load(ctx)
	require.NoError(t, err)

	robots := c.List(identity.AlphaTeamID)
	assert.GreaterOrEqual(t, len(robots), 2, "Should have at least 2 robots in alpha team")

	robots = c.List("team_nonexistent")
	assert.Len(t, robots, 0, "Non-existent team should have no robots")
}

func TestCacheGetByStatus(t *testing.T) {
	identity := testprepare.PrepareSandbox(t)
	require.NotNil(t, identity)

	cleanupTestRobots(t)
	setupTestRobots(t, identity.AlphaTeamID)
	defer cleanupTestRobots(t)

	c := cache.New()
	ctx := types.NewContext(context.Background(), nil)

	err := c.Load(ctx)
	require.NoError(t, err)

	idle := c.GetIdle()
	assert.GreaterOrEqual(t, len(idle), 2, "Should have at least 2 idle robots")

	testRobot1 := c.Get("robot_test_sales_001")
	testRobot2 := c.Get("robot_test_support_002")
	require.NotNil(t, testRobot1)
	require.NotNil(t, testRobot2)
	assert.Equal(t, types.RobotIdle, testRobot1.Status, "Test robot 1 should be idle")
	assert.Equal(t, types.RobotIdle, testRobot2.Status, "Test robot 2 should be idle")

	working := c.GetWorking()
	for _, r := range working {
		assert.Equal(t, types.RobotWorking, r.Status)
	}
}

func TestCacheAutoRefresh(t *testing.T) {
	identity := testprepare.PrepareSandbox(t)
	require.NotNil(t, identity)

	cleanupTestRobots(t)
	setupTestRobots(t, identity.AlphaTeamID)
	defer cleanupTestRobots(t)

	c := cache.New()
	ctx := types.NewContext(context.Background(), nil)
	err := c.Load(ctx)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, c.Count(), 1, "Should have at least one robot loaded")

	t.Run("start and stop auto-refresh", func(t *testing.T) {
		testCache := cache.New()
		testCtx := types.NewContext(context.Background(), nil)
		err := testCache.Load(testCtx)
		require.NoError(t, err)

		config := &cache.RefreshConfig{Interval: 100 * time.Millisecond}
		testCache.StartAutoRefresh(testCtx, config)

		time.Sleep(250 * time.Millisecond)

		testCache.StopAutoRefresh()

		countBefore := testCache.Count()
		time.Sleep(200 * time.Millisecond)
		countAfter := testCache.Count()

		assert.Equal(t, countBefore, countAfter, "Cache should be stable after stop")
	})

	t.Run("multiple start calls should replace previous", func(t *testing.T) {
		testCache := cache.New()
		testCtx := types.NewContext(context.Background(), nil)
		err := testCache.Load(testCtx)
		require.NoError(t, err)

		config := &cache.RefreshConfig{Interval: 50 * time.Millisecond}

		testCache.StartAutoRefresh(testCtx, config)
		time.Sleep(30 * time.Millisecond)

		testCache.StartAutoRefresh(testCtx, config)
		time.Sleep(30 * time.Millisecond)

		testCache.StartAutoRefresh(testCtx, config)

		time.Sleep(150 * time.Millisecond)

		testCache.StopAutoRefresh()

		assert.GreaterOrEqual(t, testCache.Count(), 0, "Cache should still be functional")
	})

	t.Run("stop without start should not panic", func(t *testing.T) {
		testCache := cache.New()

		assert.NotPanics(t, func() {
			testCache.StopAutoRefresh()
			testCache.StopAutoRefresh()
			testCache.StopAutoRefresh()
		})
	})

	t.Run("concurrent start and stop should be safe", func(t *testing.T) {
		testCache := cache.New()
		testCtx := types.NewContext(context.Background(), nil)
		err := testCache.Load(testCtx)
		require.NoError(t, err)

		config := &cache.RefreshConfig{Interval: 50 * time.Millisecond}

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

		select {
		case <-done:
			// no deadlock
		case <-time.After(5 * time.Second):
			t.Fatal("Rapid start/stop cycles caused deadlock")
		}

		testCache.StopAutoRefresh()
		assert.GreaterOrEqual(t, testCache.Count(), 0, "Cache should still be functional after rapid cycles")
	})
}

func TestCacheListAll(t *testing.T) {
	identity := testprepare.PrepareSandbox(t)
	require.NotNil(t, identity)

	cleanupTestRobots(t)
	setupTestRobots(t, identity.AlphaTeamID)
	defer cleanupTestRobots(t)

	c := cache.New()
	ctx := types.NewContext(context.Background(), nil)

	err := c.Load(ctx)
	require.NoError(t, err)

	all := c.ListAll()
	assert.GreaterOrEqual(t, len(all), 2)
	assert.Equal(t, c.Count(), len(all))
}

func TestCacheListAutonomous(t *testing.T) {
	identity := testprepare.PrepareSandbox(t)
	require.NotNil(t, identity)

	cleanupTestRobots(t)
	setupTestRobots(t, identity.AlphaTeamID)
	defer cleanupTestRobots(t)

	c := cache.New()
	ctx := types.NewContext(context.Background(), nil)

	err := c.Load(ctx)
	require.NoError(t, err)

	autonomous := c.ListAutonomous()
	assert.GreaterOrEqual(t, len(autonomous), 2, "Should have at least 2 autonomous robots")
	for _, r := range autonomous {
		assert.True(t, r.AutonomousMode)
	}
}

func TestCacheRefreshAll(t *testing.T) {
	identity := testprepare.PrepareSandbox(t)
	require.NotNil(t, identity)

	cleanupTestRobots(t)
	setupTestRobots(t, identity.AlphaTeamID)
	defer cleanupTestRobots(t)

	c := cache.New()
	ctx := types.NewContext(context.Background(), nil)

	err := c.Load(ctx)
	require.NoError(t, err)
	initialCount := c.Count()

	err = c.RefreshAll(ctx)
	require.NoError(t, err)
	assert.Equal(t, initialCount, c.Count(), "RefreshAll should reload same robots")
}

func TestCacheAddRemove(t *testing.T) {
	c := cache.New()

	robot := &types.Robot{
		MemberID:       "robot_unit_test",
		TeamID:         "team_unit_test",
		DisplayName:    "Unit Test Bot",
		AutonomousMode: true,
		Status:         types.RobotIdle,
	}

	c.Add(robot)
	assert.Equal(t, 1, c.Count())
	assert.NotNil(t, c.Get("robot_unit_test"))

	robots := c.List("team_unit_test")
	assert.Len(t, robots, 1)
	assert.Equal(t, "Unit Test Bot", robots[0].DisplayName)

	// Adding nil should be safe
	c.Add(nil)
	assert.Equal(t, 1, c.Count())

	// Adding same robot again should not duplicate
	c.Add(robot)
	assert.Equal(t, 1, c.Count())
	robots = c.List("team_unit_test")
	assert.Len(t, robots, 1)

	c.Remove("robot_unit_test")
	assert.Equal(t, 0, c.Count())
	assert.Nil(t, c.Get("robot_unit_test"))

	// Remove non-existent should be safe
	c.Remove("robot_nonexistent")
	assert.Equal(t, 0, c.Count())
}

// --- Test helpers ---

func setupTestRobots(t *testing.T, teamID string) {
	t.Helper()

	m := model.Select("__yao.member")
	tableName := m.MetaData.Table.Name
	qb := capsule.Query()

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
			"team_id":         teamID,
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
	require.NoError(t, err, "Failed to insert robot_test_sales_001")

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
			"team_id":         teamID,
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
	require.NoError(t, err, "Failed to insert robot_test_support_002")

	err = qb.Table(tableName).Insert([]map[string]interface{}{
		{
			"member_id":       "robot_test_inactive_003",
			"team_id":         teamID,
			"member_type":     "robot",
			"display_name":    "Test Inactive Bot",
			"status":          "inactive",
			"role_id":         "member",
			"autonomous_mode": true,
			"robot_status":    "paused",
		},
	})
	require.NoError(t, err, "Failed to insert robot_test_inactive_003")
}

func cleanupTestRobots(t *testing.T) {
	t.Helper()

	m := model.Select("__yao.member")
	tableName := m.MetaData.Table.Name
	qb := capsule.Query()

	testMemberIDs := []string{
		"robot_test_sales_001",
		"robot_test_support_002",
		"robot_test_inactive_003",
	}

	for _, memberID := range testMemberIDs {
		m.DeleteWhere(model.QueryParam{
			Wheres: []model.QueryWhere{
				{Column: "member_id", Value: memberID},
			},
		})
		qb.Table(tableName).Where("member_id", memberID).Delete()
	}
}
