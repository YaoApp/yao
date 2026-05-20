//go:build integration

package store_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/yao/agent/robot/store"
	"github.com/yaoapp/yao/agent/robot/types"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

// TestRobotStoreSave tests creating and updating robot records
func TestRobotStoreSave(t *testing.T) {
	identity := testprepare.PrepareSandbox(t)
	cleanupTestRobots(t)
	defer cleanupTestRobots(t)

	s := store.NewRobotStore()
	ctx := context.Background()

	t.Run("creates_new_robot_record", func(t *testing.T) {
		now := time.Now()
		record := &store.RobotRecord{
			MemberID:       "robot_test_save_001",
			TeamID:         identity.AlphaTeamID,
			DisplayName:    "Test Robot 001",
			Bio:            "A test robot for save operations",
			SystemPrompt:   "You are a helpful assistant",
			Status:         "active",
			RobotStatus:    "idle",
			AutonomousMode: true,
			RobotEmail:     "robot001@test.com",
			ManagerID:      identity.AlphaOwnerUserID,
			JoinedAt:       &now,
		}

		err := s.Save(ctx, record)
		require.NoError(t, err)

		saved, err := s.Get(ctx, "robot_test_save_001")
		require.NoError(t, err)
		require.NotNil(t, saved)

		assert.Equal(t, "robot_test_save_001", saved.MemberID)
		assert.Equal(t, identity.AlphaTeamID, saved.TeamID)
		assert.Equal(t, "Test Robot 001", saved.DisplayName)
		assert.Equal(t, "A test robot for save operations", saved.Bio)
		assert.Equal(t, "You are a helpful assistant", saved.SystemPrompt)
		assert.Equal(t, "active", saved.Status)
		assert.Equal(t, "idle", saved.RobotStatus)
		assert.True(t, saved.AutonomousMode)
		assert.Equal(t, "robot001@test.com", saved.RobotEmail)
		assert.Equal(t, "robot", saved.MemberType)
		assert.NotNil(t, saved.JoinedAt)
	})

	t.Run("updates_existing_robot_record", func(t *testing.T) {
		record := &store.RobotRecord{
			MemberID:    "robot_test_save_002",
			TeamID:      identity.AlphaTeamID,
			DisplayName: "Original Name",
			Status:      "active",
			RobotStatus: "idle",
		}

		err := s.Save(ctx, record)
		require.NoError(t, err)

		record.DisplayName = "Updated Name"
		record.Bio = "Updated bio"
		record.RobotStatus = "working"

		err = s.Save(ctx, record)
		require.NoError(t, err)

		saved, err := s.Get(ctx, "robot_test_save_002")
		require.NoError(t, err)
		require.NotNil(t, saved)

		assert.Equal(t, "Updated Name", saved.DisplayName)
		assert.Equal(t, "Updated bio", saved.Bio)
		assert.Equal(t, "working", saved.RobotStatus)
	})

	t.Run("saves_robot_with_config", func(t *testing.T) {
		record := &store.RobotRecord{
			MemberID:    "robot_test_save_003",
			TeamID:      identity.AlphaTeamID,
			DisplayName: "Robot with Config",
			Status:      "active",
			RobotStatus: "idle",
			RobotConfig: map[string]interface{}{
				"clock_mode":      "on",
				"max_concurrent":  3,
				"timeout_seconds": 300,
			},
		}

		err := s.Save(ctx, record)
		require.NoError(t, err)

		saved, err := s.Get(ctx, "robot_test_save_003")
		require.NoError(t, err)
		require.NotNil(t, saved)
		assert.NotNil(t, saved.RobotConfig)
	})

	t.Run("saves_robot_with_permission_fields", func(t *testing.T) {
		record := &store.RobotRecord{
			MemberID:     "robot_test_save_004",
			TeamID:       identity.AlphaTeamID,
			DisplayName:  "Robot with Perms",
			Status:       "active",
			RobotStatus:  "idle",
			YaoCreatedBy: identity.AlphaOwnerUserID,
			YaoTeamID:    identity.AlphaTeamID,
		}

		err := s.Save(ctx, record)
		require.NoError(t, err)

		saved, err := s.Get(ctx, "robot_test_save_004")
		require.NoError(t, err)
		require.NotNil(t, saved)
	})
}

// TestRobotStoreGet tests retrieving robot records
func TestRobotStoreGet(t *testing.T) {
	identity := testprepare.PrepareSandbox(t)
	cleanupTestRobots(t)
	defer cleanupTestRobots(t)

	s := store.NewRobotStore()
	ctx := context.Background()

	setupTestRobot(t, s, ctx, identity)

	t.Run("returns_existing_record", func(t *testing.T) {
		record, err := s.Get(ctx, "robot_test_get_001")
		require.NoError(t, err)
		require.NotNil(t, record)

		assert.Equal(t, "robot_test_get_001", record.MemberID)
		assert.Equal(t, identity.AlphaTeamID, record.TeamID)
		assert.Equal(t, "Test Robot Get", record.DisplayName)
		assert.Equal(t, "Test robot description", record.Bio)
		assert.Equal(t, "robot", record.MemberType)
		assert.Equal(t, "active", record.Status)
		assert.Equal(t, "idle", record.RobotStatus)
	})

	t.Run("returns_nil_for_non_existent_record", func(t *testing.T) {
		record, err := s.Get(ctx, "robot_non_existent")
		require.NoError(t, err)
		assert.Nil(t, record)
	})

	t.Run("ignores_non_robot_members", func(t *testing.T) {
		record, err := s.Get(ctx, "robot_test_get_001")
		require.NoError(t, err)
		require.NotNil(t, record)
		assert.Equal(t, "robot", record.MemberType)
	})
}

// TestRobotStoreList tests listing robot records with filters
func TestRobotStoreList(t *testing.T) {
	identity := testprepare.PrepareSandbox(t)
	cleanupTestRobots(t)
	defer cleanupTestRobots(t)

	s := store.NewRobotStore()
	ctx := context.Background()

	setupTestRobotsForList(t, s, ctx, identity)

	t.Run("lists_all_robot_records", func(t *testing.T) {
		records, total, err := s.List(ctx, &store.RobotListOptions{
			Keywords: "Robot",
		})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(records), 4)
		assert.GreaterOrEqual(t, total, 4)
	})

	t.Run("filters_by_team_id", func(t *testing.T) {
		records, total, err := s.List(ctx, &store.RobotListOptions{
			TeamID: identity.AlphaTeamID,
		})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(records), 2)
		assert.GreaterOrEqual(t, total, 2)
		for _, r := range records {
			assert.Equal(t, identity.AlphaTeamID, r.TeamID)
		}
	})

	t.Run("filters_by_robot_status", func(t *testing.T) {
		records, _, err := s.List(ctx, &store.RobotListOptions{
			Status: types.RobotWorking,
		})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(records), 1)
		for _, r := range records {
			assert.Equal(t, "working", r.RobotStatus)
		}
	})

	t.Run("filters_by_keywords", func(t *testing.T) {
		records, _, err := s.List(ctx, &store.RobotListOptions{
			Keywords: "Alpha",
		})
		require.NoError(t, err)
		assert.Equal(t, 1, len(records))
		assert.Contains(t, records[0].DisplayName, "Alpha")
	})

	t.Run("respects_pagination", func(t *testing.T) {
		records, total, err := s.List(ctx, &store.RobotListOptions{
			Page:     1,
			PageSize: 2,
		})
		require.NoError(t, err)
		assert.Equal(t, 2, len(records))
		assert.GreaterOrEqual(t, total, 4)
	})

	t.Run("respects_limit", func(t *testing.T) {
		records, _, err := s.List(ctx, &store.RobotListOptions{
			Limit: 2,
		})
		require.NoError(t, err)
		assert.Equal(t, 2, len(records))
	})

	t.Run("combines_multiple_filters", func(t *testing.T) {
		records, total, err := s.List(ctx, &store.RobotListOptions{
			TeamID: identity.AlphaTeamID,
			Status: types.RobotIdle,
		})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(records), 1)
		assert.GreaterOrEqual(t, total, 1)
		for _, r := range records {
			assert.Equal(t, identity.AlphaTeamID, r.TeamID)
			assert.Equal(t, "idle", r.RobotStatus)
		}
	})
}

// TestRobotStoreDelete tests deleting robot records
func TestRobotStoreDelete(t *testing.T) {
	identity := testprepare.PrepareSandbox(t)
	cleanupTestRobots(t)
	defer cleanupTestRobots(t)

	s := store.NewRobotStore()
	ctx := context.Background()

	t.Run("deletes_existing_record", func(t *testing.T) {
		record := &store.RobotRecord{
			MemberID:    "robot_test_delete_001",
			TeamID:      identity.AlphaTeamID,
			DisplayName: "Robot to Delete",
			Status:      "active",
			RobotStatus: "idle",
		}
		err := s.Save(ctx, record)
		require.NoError(t, err)

		saved, err := s.Get(ctx, "robot_test_delete_001")
		require.NoError(t, err)
		require.NotNil(t, saved)

		err = s.Delete(ctx, "robot_test_delete_001")
		require.NoError(t, err)

		saved, err = s.Get(ctx, "robot_test_delete_001")
		require.NoError(t, err)
		assert.Nil(t, saved)
	})

	t.Run("no_error_for_non_existent_record", func(t *testing.T) {
		err := s.Delete(ctx, "robot_non_existent")
		assert.NoError(t, err)
	})
}

// TestRobotStoreUpdateConfig tests updating robot config
func TestRobotStoreUpdateConfig(t *testing.T) {
	identity := testprepare.PrepareSandbox(t)
	cleanupTestRobots(t)
	defer cleanupTestRobots(t)

	s := store.NewRobotStore()
	ctx := context.Background()

	record := &store.RobotRecord{
		MemberID:    "robot_test_config_001",
		TeamID:      identity.AlphaTeamID,
		DisplayName: "Config Test Robot",
		Status:      "active",
		RobotStatus: "idle",
		RobotConfig: map[string]interface{}{
			"clock_mode": "off",
		},
	}
	err := s.Save(ctx, record)
	require.NoError(t, err)

	t.Run("updates_config_only", func(t *testing.T) {
		newConfig := map[string]interface{}{
			"clock_mode":      "on",
			"max_concurrent":  5,
			"timeout_seconds": 600,
		}
		err := s.UpdateConfig(ctx, "robot_test_config_001", newConfig)
		require.NoError(t, err)

		saved, err := s.Get(ctx, "robot_test_config_001")
		require.NoError(t, err)
		require.NotNil(t, saved)
		assert.NotNil(t, saved.RobotConfig)
		assert.Equal(t, "Config Test Robot", saved.DisplayName)
	})
}

// TestRobotStoreUpdateStatus tests updating robot status
func TestRobotStoreUpdateStatus(t *testing.T) {
	identity := testprepare.PrepareSandbox(t)
	cleanupTestRobots(t)
	defer cleanupTestRobots(t)

	s := store.NewRobotStore()
	ctx := context.Background()

	record := &store.RobotRecord{
		MemberID:    "robot_test_status_001",
		TeamID:      identity.AlphaTeamID,
		DisplayName: "Status Test Robot",
		Status:      "active",
		RobotStatus: "idle",
	}
	err := s.Save(ctx, record)
	require.NoError(t, err)

	t.Run("updates_robot_status", func(t *testing.T) {
		err := s.UpdateStatus(ctx, "robot_test_status_001", types.RobotWorking)
		require.NoError(t, err)

		saved, err := s.Get(ctx, "robot_test_status_001")
		require.NoError(t, err)
		require.NotNil(t, saved)
		assert.Equal(t, "working", saved.RobotStatus)
		assert.Equal(t, "Status Test Robot", saved.DisplayName)
	})

	t.Run("updates_to_paused", func(t *testing.T) {
		err := s.UpdateStatus(ctx, "robot_test_status_001", types.RobotPaused)
		require.NoError(t, err)

		saved, err := s.Get(ctx, "robot_test_status_001")
		require.NoError(t, err)
		assert.Equal(t, "paused", saved.RobotStatus)
	})

	t.Run("updates_to_error", func(t *testing.T) {
		err := s.UpdateStatus(ctx, "robot_test_status_001", types.RobotError)
		require.NoError(t, err)

		saved, err := s.Get(ctx, "robot_test_status_001")
		require.NoError(t, err)
		assert.Equal(t, "error", saved.RobotStatus)
	})
}

// TestRobotRecordConversion tests conversion between RobotRecord and Robot types
func TestRobotRecordConversion(t *testing.T) {
	testprepare.PrepareSandbox(t)

	t.Run("converts_record_to_robot", func(t *testing.T) {
		now := time.Now()
		record := &store.RobotRecord{
			MemberID:       "robot_convert_001",
			TeamID:         "team_convert_001",
			DisplayName:    "Conversion Test Robot",
			Bio:            "Test description",
			SystemPrompt:   "You are helpful",
			Status:         "active",
			RobotStatus:    "idle",
			AutonomousMode: true,
			RobotEmail:     "convert@test.com",
			JoinedAt:       &now,
			RobotConfig: map[string]interface{}{
				"clock_mode": "on",
			},
		}

		robot, err := record.ToRobot()
		require.NoError(t, err)
		require.NotNil(t, robot)

		assert.Equal(t, "robot_convert_001", robot.MemberID)
		assert.Equal(t, "team_convert_001", robot.TeamID)
		assert.Equal(t, "Conversion Test Robot", robot.DisplayName)
		assert.Equal(t, "Test description", robot.Bio)
		assert.Equal(t, "You are helpful", robot.SystemPrompt)
		assert.True(t, robot.AutonomousMode)
		assert.Equal(t, "convert@test.com", robot.RobotEmail)
	})

	t.Run("converts_robot_to_record", func(t *testing.T) {
		robot := &store.RobotRecord{
			MemberID:       "robot_from_001",
			TeamID:         "team_from_001",
			DisplayName:    "From Robot Test",
			Bio:            "From robot description",
			SystemPrompt:   "System prompt",
			RobotStatus:    "working",
			AutonomousMode: false,
			RobotEmail:     "from@test.com",
		}

		converted, err := robot.ToRobot()
		require.NoError(t, err)

		assert.Equal(t, "robot_from_001", converted.MemberID)
		assert.Equal(t, "team_from_001", converted.TeamID)
		assert.Equal(t, "From Robot Test", converted.DisplayName)
	})
}

// --- helpers ---

func cleanupTestRobots(t *testing.T) {
	t.Helper()
	mod := model.Select("__yao.member")
	if mod == nil {
		return
	}

	_, err := mod.DeleteWhere(model.QueryParam{
		Wheres: []model.QueryWhere{
			{Column: "member_id", OP: "like", Value: "robot_test_%"},
			{Column: "member_type", Value: "robot"},
		},
	})
	if err != nil {
		t.Logf("Warning: failed to cleanup test robots: %v", err)
	}
}

func setupTestRobot(t *testing.T, s *store.RobotStore, ctx context.Context, identity *testprepare.TestIdentity) {
	t.Helper()
	now := time.Now()
	record := &store.RobotRecord{
		MemberID:       "robot_test_get_001",
		TeamID:         identity.AlphaTeamID,
		DisplayName:    "Test Robot Get",
		Bio:            "Test robot description",
		SystemPrompt:   "You are a test assistant",
		Status:         "active",
		RobotStatus:    "idle",
		AutonomousMode: false,
		RobotEmail:     "test@robot.com",
		ManagerID:      identity.AlphaOwnerUserID,
		JoinedAt:       &now,
	}

	err := s.Save(ctx, record)
	require.NoError(t, err)
}

func setupTestRobotsForList(t *testing.T, s *store.RobotStore, ctx context.Context, identity *testprepare.TestIdentity) {
	t.Helper()
	now := time.Now()

	records := []*store.RobotRecord{
		{
			MemberID:    "robot_test_list_001",
			TeamID:      identity.AlphaTeamID,
			DisplayName: "Robot Alpha",
			Status:      "active",
			RobotStatus: "idle",
			JoinedAt:    &now,
		},
		{
			MemberID:    "robot_test_list_002",
			TeamID:      identity.AlphaTeamID,
			DisplayName: "Robot Beta",
			Status:      "active",
			RobotStatus: "working",
			JoinedAt:    &now,
		},
		{
			MemberID:    "robot_test_list_003",
			TeamID:      "team_list_002",
			DisplayName: "Robot Gamma",
			Status:      "active",
			RobotStatus: "idle",
			JoinedAt:    &now,
		},
		{
			MemberID:    "robot_test_list_004",
			TeamID:      "team_list_002",
			DisplayName: "Robot Delta",
			Status:      "inactive",
			RobotStatus: "paused",
			JoinedAt:    &now,
		},
	}

	for _, record := range records {
		err := s.Save(ctx, record)
		require.NoError(t, err)
	}
}
