//go:build integration

package api_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/robot/api"
	"github.com/yaoapp/yao/agent/robot/executor"
	"github.com/yaoapp/yao/agent/robot/manager"
	"github.com/yaoapp/yao/agent/robot/types"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

func TestAPILifecycle(t *testing.T) {
	testprepare.PrepareSandbox(t)

	t.Run("start and stop", func(t *testing.T) {
		config := &manager.Config{
			TickInterval: 10 * time.Second,
			Executor:     executor.NewDryRun(),
		}
		err := api.StartWithConfig(config)
		require.NoError(t, err)
		assert.True(t, api.IsRunning())

		err = api.Stop()
		require.NoError(t, err)
		assert.False(t, api.IsRunning())
	})

	t.Run("double start returns error", func(t *testing.T) {
		config := &manager.Config{
			TickInterval: 10 * time.Second,
			Executor:     executor.NewDryRun(),
		}
		err := api.StartWithConfig(config)
		require.NoError(t, err)
		defer api.Stop()

		err = api.StartWithConfig(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already started")
	})

	t.Run("stop when not started is safe", func(t *testing.T) {
		err := api.Stop()
		assert.NoError(t, err)
	})
}

func TestAPIGetRobot(t *testing.T) {
	testprepare.PrepareSandbox(t)

	ctx := types.NewContext(context.Background(), nil)

	t.Run("returns error for empty member_id", func(t *testing.T) {
		robot, err := api.GetRobot(ctx, "")
		assert.Error(t, err)
		assert.Nil(t, robot)
		assert.Contains(t, err.Error(), "member_id is required")
	})

	t.Run("returns error for non-existent robot", func(t *testing.T) {
		robot, err := api.GetRobot(ctx, "non_existent_member_id_xyz")
		assert.Error(t, err)
		assert.Nil(t, robot)
	})
}

func TestAPIGetRobotStatus(t *testing.T) {
	testprepare.PrepareSandbox(t)

	ctx := types.NewContext(context.Background(), nil)

	t.Run("returns error for empty member_id", func(t *testing.T) {
		status, err := api.GetRobotStatus(ctx, "")
		assert.Error(t, err)
		assert.Nil(t, status)
		assert.Contains(t, err.Error(), "member_id is required")
	})

	t.Run("returns error for non-existent robot", func(t *testing.T) {
		status, err := api.GetRobotStatus(ctx, "non_existent_member_id_xyz")
		assert.Error(t, err)
		assert.Nil(t, status)
	})
}

func TestAPIListAllRobots(t *testing.T) {
	testprepare.PrepareSandbox(t)

	ctx := types.NewContext(context.Background(), nil)

	t.Run("applies default pagination when query is nil", func(t *testing.T) {
		result, err := api.ListAllRobots(ctx, nil)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 1, result.Page)
		assert.Equal(t, 20, result.PageSize)
	})

	t.Run("applies default pagination when values are zero", func(t *testing.T) {
		result, err := api.ListAllRobots(ctx, &api.ListQuery{
			Page:     0,
			PageSize: 0,
		})
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 1, result.Page)
		assert.Equal(t, 20, result.PageSize)
	})

	t.Run("caps pagesize at 100", func(t *testing.T) {
		result, err := api.ListAllRobots(ctx, &api.ListQuery{
			Page:     1,
			PageSize: 500,
		})
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 100, result.PageSize)
	})
}

func TestAPICreateRobotValidation(t *testing.T) {
	testprepare.PrepareSandbox(t)

	ctx := types.NewContext(context.Background(), nil)

	t.Run("returns error for empty team_id", func(t *testing.T) {
		req := &api.CreateRobotRequest{
			MemberID:    "robot_integ_test_001",
			TeamID:      "",
			DisplayName: "Test Robot",
		}
		result, err := api.CreateRobot(ctx, req)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "team_id is required")
	})

	t.Run("returns error for empty display_name", func(t *testing.T) {
		req := &api.CreateRobotRequest{
			MemberID:    "robot_integ_test_001",
			TeamID:      "team_001",
			DisplayName: "",
		}
		result, err := api.CreateRobot(ctx, req)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "display_name is required")
	})
}

func TestAPICreateAndRemoveRobot(t *testing.T) {
	testprepare.PrepareSandbox(t)

	ctx := types.NewContext(context.Background(), nil)

	t.Run("create and remove robot", func(t *testing.T) {
		req := &api.CreateRobotRequest{
			MemberID:    "robot_integ_crud_001",
			TeamID:      "team_integ_crud",
			DisplayName: "Integration CRUD Robot",
		}

		result, err := api.CreateRobot(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "robot_integ_crud_001", result.MemberID)
		assert.Equal(t, "team_integ_crud", result.TeamID)
		assert.Equal(t, "Integration CRUD Robot", result.DisplayName)
		assert.Equal(t, "active", result.Status)
		assert.Equal(t, "idle", result.RobotStatus)

		err = api.RemoveRobot(ctx, "robot_integ_crud_001")
		require.NoError(t, err)

		_, err = api.GetRobot(ctx, "robot_integ_crud_001")
		assert.Error(t, err)
	})

	t.Run("create duplicate returns error", func(t *testing.T) {
		req := &api.CreateRobotRequest{
			MemberID:    "robot_integ_crud_dup",
			TeamID:      "team_integ_crud",
			DisplayName: "Dup Robot",
		}

		_, err := api.CreateRobot(ctx, req)
		require.NoError(t, err)
		defer api.RemoveRobot(ctx, "robot_integ_crud_dup")

		_, err = api.CreateRobot(ctx, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})
}

func TestAPIUpdateRobot(t *testing.T) {
	testprepare.PrepareSandbox(t)

	ctx := types.NewContext(context.Background(), nil)

	createReq := &api.CreateRobotRequest{
		MemberID:    "robot_integ_update_001",
		TeamID:      "team_integ_update",
		DisplayName: "Original Name",
		Bio:         "Original bio",
	}
	_, err := api.CreateRobot(ctx, createReq)
	require.NoError(t, err)
	defer api.RemoveRobot(ctx, "robot_integ_update_001")

	t.Run("returns error for empty member_id", func(t *testing.T) {
		req := &api.UpdateRobotRequest{}
		result, err := api.UpdateRobot(ctx, "", req)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "member_id is required")
	})

	t.Run("returns error for non-existent robot", func(t *testing.T) {
		newName := "New Name"
		req := &api.UpdateRobotRequest{
			DisplayName: &newName,
		}
		result, err := api.UpdateRobot(ctx, "non_existent_robot", req)
		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("updates display_name", func(t *testing.T) {
		newName := "Updated Name"
		req := &api.UpdateRobotRequest{
			DisplayName: &newName,
		}

		result, err := api.UpdateRobot(ctx, "robot_integ_update_001", req)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "Updated Name", result.DisplayName)
		assert.Equal(t, "Original bio", result.Bio)
	})
}

func TestAPITriggerManual(t *testing.T) {
	testprepare.PrepareSandbox(t)

	config := &manager.Config{
		TickInterval: 10 * time.Second,
		Executor:     executor.NewDryRun(),
	}
	err := api.StartWithConfig(config)
	require.NoError(t, err)
	defer api.Stop()

	ctx := types.NewContext(context.Background(), nil)

	t.Run("rejects non-existent robot", func(t *testing.T) {
		result, err := api.TriggerManual(ctx, "robot_nonexistent_xyz", types.TriggerClock, nil)
		if err == nil {
			assert.False(t, result.Accepted)
		}
	})
}
