package manager_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/xun/capsule"
	"github.com/yaoapp/yao/agent/robot/manager"
	"github.com/yaoapp/yao/agent/testutils"
)

const recoveryTestPrefix = "_test_recovery_"

func TestRecoveryOnRestart(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	t.Run("marks_running_as_failed_on_restart", func(t *testing.T) {
		cleanupRecoveryData(t)
		defer cleanupRecoveryData(t)

		insertRecoveryExec(t, recoveryTestPrefix+"run_001", recoveryTestPrefix+"member_001", "team_r", "running")
		insertRecoveryRobot(t, recoveryTestPrefix+"member_001", "team_r")

		m := manager.New()
		err := m.Start()
		require.NoError(t, err)
		defer m.Stop()

		rec := getExecRecord(t, recoveryTestPrefix+"run_001")
		require.NotNil(t, rec)
		assert.Equal(t, "failed", rec["status"])
		errMsg, _ := rec["error"].(string)
		assert.Contains(t, errMsg, "server restart")
	})

	t.Run("keeps_waiting_on_restart", func(t *testing.T) {
		cleanupRecoveryData(t)
		defer cleanupRecoveryData(t)

		insertRecoveryExec(t, recoveryTestPrefix+"wait_001", recoveryTestPrefix+"member_002", "team_r", "waiting")
		insertRecoveryRobot(t, recoveryTestPrefix+"member_002", "team_r")

		m := manager.New()
		err := m.Start()
		require.NoError(t, err)
		defer m.Stop()

		rec := getExecRecord(t, recoveryTestPrefix+"wait_001")
		require.NotNil(t, rec)
		assert.Equal(t, "waiting", rec["status"])
	})

	t.Run("keeps_confirming_on_restart", func(t *testing.T) {
		cleanupRecoveryData(t)
		defer cleanupRecoveryData(t)

		insertRecoveryExec(t, recoveryTestPrefix+"conf_001", recoveryTestPrefix+"member_003", "team_r", "confirming")
		insertRecoveryRobot(t, recoveryTestPrefix+"member_003", "team_r")

		m := manager.New()
		err := m.Start()
		require.NoError(t, err)
		defer m.Stop()

		rec := getExecRecord(t, recoveryTestPrefix+"conf_001")
		require.NotNil(t, rec)
		assert.Equal(t, "confirming", rec["status"])
	})

	t.Run("marks_paused_as_failed_on_restart", func(t *testing.T) {
		cleanupRecoveryData(t)
		defer cleanupRecoveryData(t)

		insertRecoveryExec(t, recoveryTestPrefix+"pause_001", recoveryTestPrefix+"member_004", "team_r", "paused")
		insertRecoveryRobot(t, recoveryTestPrefix+"member_004", "team_r")

		m := manager.New()
		err := m.Start()
		require.NoError(t, err)
		defer m.Stop()

		rec := getExecRecord(t, recoveryTestPrefix+"pause_001")
		require.NotNil(t, rec)
		assert.Equal(t, "failed", rec["status"])
	})

	t.Run("no_active_executions_starts_normally", func(t *testing.T) {
		cleanupRecoveryData(t)
		defer cleanupRecoveryData(t)

		m := manager.New()
		err := m.Start()
		require.NoError(t, err)
		defer m.Stop()

		assert.True(t, m.IsStarted())
	})

	t.Run("updates_robot_status_to_idle_after_fail", func(t *testing.T) {
		cleanupRecoveryData(t)
		defer cleanupRecoveryData(t)

		insertRecoveryExec(t, recoveryTestPrefix+"run_002", recoveryTestPrefix+"member_005", "team_r", "running")
		insertRecoveryRobot(t, recoveryTestPrefix+"member_005", "team_r")
		setRobotStatus(t, recoveryTestPrefix+"member_005", "working")

		m := manager.New()
		err := m.Start()
		require.NoError(t, err)
		defer m.Stop()

		rec := getExecRecord(t, recoveryTestPrefix+"run_002")
		require.NotNil(t, rec)
		assert.Equal(t, "failed", rec["status"])

		robot := getRobotRecord(t, recoveryTestPrefix+"member_005")
		require.NotNil(t, robot)
		assert.Equal(t, "idle", robot["robot_status"])
	})

	t.Run("keeps_robot_status_if_other_waiting", func(t *testing.T) {
		cleanupRecoveryData(t)
		defer cleanupRecoveryData(t)

		insertRecoveryExec(t, recoveryTestPrefix+"run_003", recoveryTestPrefix+"member_006", "team_r", "running")
		insertRecoveryExec(t, recoveryTestPrefix+"wait_003", recoveryTestPrefix+"member_006", "team_r", "waiting")
		insertRecoveryRobot(t, recoveryTestPrefix+"member_006", "team_r")
		setRobotStatus(t, recoveryTestPrefix+"member_006", "working")

		m := manager.New()
		err := m.Start()
		require.NoError(t, err)
		defer m.Stop()

		// Running should be failed
		rec := getExecRecord(t, recoveryTestPrefix+"run_003")
		require.NotNil(t, rec)
		assert.Equal(t, "failed", rec["status"])

		// Waiting should remain
		rec2 := getExecRecord(t, recoveryTestPrefix+"wait_003")
		require.NotNil(t, rec2)
		assert.Equal(t, "waiting", rec2["status"])

		// Robot should NOT be set to idle because waiting exec still exists
		robot := getRobotRecord(t, recoveryTestPrefix+"member_006")
		require.NotNil(t, robot)
		assert.NotEqual(t, "idle", robot["robot_status"],
			"robot should not be idle when waiting execution exists")
	})

	t.Run("idempotent_on_double_restart", func(t *testing.T) {
		cleanupRecoveryData(t)
		defer cleanupRecoveryData(t)

		insertRecoveryExec(t, recoveryTestPrefix+"run_004", recoveryTestPrefix+"member_007", "team_r", "running")
		insertRecoveryRobot(t, recoveryTestPrefix+"member_007", "team_r")

		// First start
		m1 := manager.New()
		err := m1.Start()
		require.NoError(t, err)
		m1.Stop()

		rec := getExecRecord(t, recoveryTestPrefix+"run_004")
		require.NotNil(t, rec)
		assert.Equal(t, "failed", rec["status"])

		// Second start — should not panic or error
		m2 := manager.New()
		err = m2.Start()
		require.NoError(t, err)
		defer m2.Stop()

		rec2 := getExecRecord(t, recoveryTestPrefix+"run_004")
		require.NotNil(t, rec2)
		assert.Equal(t, "failed", rec2["status"])
	})
}

// ==================== Helpers ====================

func insertRecoveryExec(t *testing.T, execID, memberID, teamID, status string) {
	t.Helper()
	mod := model.Select("__yao.agent.execution")
	tableName := mod.MetaData.Table.Name
	qb := capsule.Query()

	now := time.Now()
	err := qb.Table(tableName).Insert([]map[string]interface{}{
		{
			"execution_id": execID,
			"member_id":    memberID,
			"team_id":      teamID,
			"trigger_type": "clock",
			"status":       status,
			"phase":        "run",
			"start_time":   now.Add(-1 * time.Hour),
		},
	})
	require.NoError(t, err, "insert execution %s", execID)
}

func insertRecoveryRobot(t *testing.T, memberID, teamID string) {
	t.Helper()
	mod := model.Select("__yao.member")
	tableName := mod.MetaData.Table.Name
	qb := capsule.Query()

	robotConfig := map[string]interface{}{
		"identity": map[string]interface{}{"role": "Recovery Test Robot"},
		"triggers": map[string]interface{}{
			"clock": map[string]interface{}{"enabled": false},
		},
	}
	configJSON, _ := json.Marshal(robotConfig)

	err := qb.Table(tableName).Insert([]map[string]interface{}{
		{
			"member_id":       memberID,
			"team_id":         teamID,
			"member_type":     "robot",
			"display_name":    "Recovery Test " + memberID,
			"status":          "active",
			"role_id":         "member",
			"autonomous_mode": true,
			"robot_status":    "idle",
			"robot_config":    string(configJSON),
		},
	})
	require.NoError(t, err, "insert robot %s", memberID)
}

func setRobotStatus(t *testing.T, memberID, status string) {
	t.Helper()
	mod := model.Select("__yao.member")
	tableName := mod.MetaData.Table.Name
	qb := capsule.Query()
	_, err := qb.Table(tableName).Where("member_id", memberID).Update(map[string]interface{}{
		"robot_status": status,
	})
	require.NoError(t, err)
}

func getExecRecord(t *testing.T, execID string) map[string]interface{} {
	t.Helper()
	mod := model.Select("__yao.agent.execution")
	tableName := mod.MetaData.Table.Name
	qb := capsule.Query()

	rows, err := qb.Table(tableName).Where("execution_id", execID).Limit(1).Get()
	require.NoError(t, err)
	if len(rows) == 0 {
		return nil
	}
	return map[string]interface{}(rows[0])
}

func getRobotRecord(t *testing.T, memberID string) map[string]interface{} {
	t.Helper()
	mod := model.Select("__yao.member")
	tableName := mod.MetaData.Table.Name
	qb := capsule.Query()

	rows, err := qb.Table(tableName).Where("member_id", memberID).Limit(1).Get()
	require.NoError(t, err)
	if len(rows) == 0 {
		return nil
	}
	return map[string]interface{}(rows[0])
}

func cleanupRecoveryData(t *testing.T) {
	t.Helper()

	// Clean executions
	execMod := model.Select("__yao.agent.execution")
	execTable := execMod.MetaData.Table.Name
	qb := capsule.Query()
	qb.Table(execTable).Where("execution_id", "like", recoveryTestPrefix+"%").Delete()

	// Clean robots
	memberMod := model.Select("__yao.member")
	memberTable := memberMod.MetaData.Table.Name
	qb.Table(memberTable).Where("member_id", "like", recoveryTestPrefix+"%").Delete()
	// Also clean via model (soft delete)
	memberMod.DeleteWhere(model.QueryParam{
		Wheres: []model.QueryWhere{
			{Column: "member_id", OP: "like", Value: recoveryTestPrefix + "%"},
		},
	})
}
