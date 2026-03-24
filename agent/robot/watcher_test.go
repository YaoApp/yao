package robot_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/xun/capsule"
	"github.com/yaoapp/yao/agent/robot/api"
	"github.com/yaoapp/yao/agent/robot/manager"
	"github.com/yaoapp/yao/agent/robot/store"
	"github.com/yaoapp/yao/agent/robot/types"
	"github.com/yaoapp/yao/agent/testutils"
	"github.com/yaoapp/yao/monitor"

	// Trigger watcher registration via init()
	_ "github.com/yaoapp/yao/agent/robot"
)

const watcherTestPrefix = "_test_watcher_"

func TestRobotTasksWatcher(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	t.Run("detects_zombie_running_execution", func(t *testing.T) {
		cleanupWatcherData(t)
		defer cleanupWatcherData(t)

		insertWatcherRobot(t, watcherTestPrefix+"member_001", "team_w")

		m := startWatcherManager(t)
		defer m.Stop()

		// Insert AFTER manager start so recovery doesn't touch it
		oldStart := time.Now().Add(-5 * time.Hour)
		insertWatcherExec(t, watcherTestPrefix+"zombie_001", watcherTestPrefix+"member_001", "team_w", "running", &oldStart, nil)

		watcher := findWatcher(t, "robot-tasks")
		alerts := watcher.Check(context.Background())

		found := false
		for _, a := range alerts {
			if a.Target == "execution:"+watcherTestPrefix+"zombie_001" {
				found = true
				assert.Equal(t, monitor.Warn, a.Level)
				assert.Contains(t, a.Message, "zombie")
				assert.NotNil(t, a.Action)
			}
		}
		assert.True(t, found, "should detect zombie running execution")
	})

	t.Run("ignores_recent_running_execution", func(t *testing.T) {
		cleanupWatcherData(t)
		defer cleanupWatcherData(t)

		insertWatcherRobot(t, watcherTestPrefix+"member_002", "team_w")

		m := startWatcherManager(t)
		defer m.Stop()

		// Insert AFTER start so recovery doesn't mark it failed
		recentStart := time.Now().Add(-30 * time.Minute)
		insertWatcherExec(t, watcherTestPrefix+"recent_001", watcherTestPrefix+"member_002", "team_w", "running", &recentStart, nil)

		watcher := findWatcher(t, "robot-tasks")
		alerts := watcher.Check(context.Background())

		for _, a := range alerts {
			assert.NotEqual(t, "execution:"+watcherTestPrefix+"recent_001", a.Target,
				"should not alert for recent running execution")
		}
	})

	t.Run("detects_waiting_timeout", func(t *testing.T) {
		cleanupWatcherData(t)
		defer cleanupWatcherData(t)

		insertWatcherRobot(t, watcherTestPrefix+"member_003", "team_w")

		m := startWatcherManager(t)
		defer m.Stop()

		startTime := time.Now().Add(-25 * time.Hour)
		oldUpdated := time.Now().Add(-25 * time.Hour)
		insertWatcherExec(t, watcherTestPrefix+"wait_001", watcherTestPrefix+"member_003", "team_w", "waiting", &startTime, &oldUpdated)

		watcher := findWatcher(t, "robot-tasks")
		alerts := watcher.Check(context.Background())

		found := false
		for _, a := range alerts {
			if a.Target == "execution:"+watcherTestPrefix+"wait_001" {
				found = true
				assert.Equal(t, monitor.Warn, a.Level)
				assert.Contains(t, a.Message, "waiting")
				assert.NotNil(t, a.Action)
			}
		}
		assert.True(t, found, "should detect waiting timeout")
	})

	t.Run("ignores_recent_waiting_execution", func(t *testing.T) {
		cleanupWatcherData(t)
		defer cleanupWatcherData(t)

		insertWatcherRobot(t, watcherTestPrefix+"member_004", "team_w")

		m := startWatcherManager(t)
		defer m.Stop()

		startTime := time.Now().Add(-1 * time.Hour)
		recentUpdated := time.Now().Add(-30 * time.Minute)
		insertWatcherExec(t, watcherTestPrefix+"wait_002", watcherTestPrefix+"member_004", "team_w", "waiting", &startTime, &recentUpdated)

		watcher := findWatcher(t, "robot-tasks")
		alerts := watcher.Check(context.Background())

		for _, a := range alerts {
			assert.NotEqual(t, "execution:"+watcherTestPrefix+"wait_002", a.Target,
				"should not alert for recent waiting execution")
		}
	})

	t.Run("detects_confirming_timeout", func(t *testing.T) {
		cleanupWatcherData(t)
		defer cleanupWatcherData(t)

		insertWatcherRobot(t, watcherTestPrefix+"member_005", "team_w")

		m := startWatcherManager(t)
		defer m.Stop()

		startTime := time.Now().Add(-2 * time.Hour)
		oldUpdated := time.Now().Add(-2 * time.Hour)
		insertWatcherExec(t, watcherTestPrefix+"conf_001", watcherTestPrefix+"member_005", "team_w", "confirming", &startTime, &oldUpdated)

		watcher := findWatcher(t, "robot-tasks")
		alerts := watcher.Check(context.Background())

		found := false
		for _, a := range alerts {
			if a.Target == "execution:"+watcherTestPrefix+"conf_001" {
				found = true
				assert.Equal(t, monitor.Info, a.Level)
				assert.Contains(t, a.Message, "confirming")
				assert.NotNil(t, a.Action)
			}
		}
		assert.True(t, found, "should detect confirming timeout")
	})

	t.Run("returns_empty_when_no_issues", func(t *testing.T) {
		cleanupWatcherData(t)
		defer cleanupWatcherData(t)

		insertWatcherRobot(t, watcherTestPrefix+"member_006", "team_w")

		m := startWatcherManager(t)
		defer m.Stop()

		startTime := time.Now().Add(-1 * time.Hour)
		insertWatcherExec(t, watcherTestPrefix+"done_001", watcherTestPrefix+"member_006", "team_w", "completed", &startTime, nil)
		insertWatcherExec(t, watcherTestPrefix+"done_002", watcherTestPrefix+"member_006", "team_w", "failed", &startTime, nil)

		watcher := findWatcher(t, "robot-tasks")
		alerts := watcher.Check(context.Background())

		for _, a := range alerts {
			assert.NotContains(t, a.Target, watcherTestPrefix,
				"should not have alerts for terminal-state executions")
		}
	})

	t.Run("handles_nil_manager_gracefully", func(t *testing.T) {
		cleanupWatcherData(t)
		defer cleanupWatcherData(t)

		// Do NOT start a manager — api.GetManager() returns nil
		api.SetManager(nil)

		watcher := findWatcher(t, "robot-tasks")
		assert.NotPanics(t, func() {
			alerts := watcher.Check(context.Background())
			assert.Empty(t, alerts)
		})
	})
}

// ==================== Helpers ====================

func startWatcherManager(t *testing.T) *manager.Manager {
	t.Helper()
	m := manager.New()
	err := m.Start()
	require.NoError(t, err)
	api.SetManager(m)
	return m
}

func findWatcher(t *testing.T, name string) monitor.Watcher {
	t.Helper()
	w := monitor.GetWatcher(name)
	require.NotNil(t, w, "watcher %q should be registered", name)
	return w
}

func insertWatcherExec(t *testing.T, execID, memberID, teamID, status string, startTime *time.Time, updatedAt *time.Time) {
	t.Helper()
	ctx := context.Background()
	execStore := store.NewExecutionStore()

	record := &store.ExecutionRecord{
		ExecutionID: execID,
		MemberID:    memberID,
		TeamID:      teamID,
		TriggerType: types.TriggerClock,
		Status:      types.ExecStatus(status),
		Phase:       types.PhaseRun,
		StartTime:   startTime,
	}

	err := execStore.Save(ctx, record)
	require.NoError(t, err, "insert execution %s", execID)

	if updatedAt != nil {
		mod := model.Select("__yao.agent.execution")
		require.NotNil(t, mod)
		tableName := mod.MetaData.Table.Name
		qb := capsule.Query()
		_, err := qb.Table(tableName).
			Where("execution_id", execID).
			Update(map[string]interface{}{"updated_at": updatedAt.Format("2006-01-02 15:04:05")})
		require.NoError(t, err, "update updated_at for %s", execID)
	}
}

func insertWatcherRobot(t *testing.T, memberID, teamID string) {
	t.Helper()
	mod := model.Select("__yao.member")
	tableName := mod.MetaData.Table.Name
	qb := capsule.Query()

	err := qb.Table(tableName).Insert([]map[string]interface{}{
		{
			"member_id":       memberID,
			"team_id":         teamID,
			"member_type":     "robot",
			"display_name":    "Watcher Test " + memberID,
			"status":          "active",
			"role_id":         "member",
			"autonomous_mode": true,
			"robot_status":    "idle",
		},
	})
	require.NoError(t, err, "insert robot %s", memberID)
}

func cleanupWatcherData(t *testing.T) {
	t.Helper()

	execMod := model.Select("__yao.agent.execution")
	execTable := execMod.MetaData.Table.Name
	qb := capsule.Query()
	qb.Table(execTable).Where("execution_id", "like", watcherTestPrefix+"%").Delete()

	memberMod := model.Select("__yao.member")
	memberTable := memberMod.MetaData.Table.Name
	qb.Table(memberTable).Where("member_id", "like", watcherTestPrefix+"%").Delete()
	memberMod.DeleteWhere(model.QueryParam{
		Wheres: []model.QueryWhere{
			{Column: "member_id", OP: "like", Value: watcherTestPrefix + "%"},
		},
	})
}
