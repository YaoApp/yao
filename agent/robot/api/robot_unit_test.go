//go:build unit

package api_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/agent/robot/api"
	"github.com/yaoapp/yao/agent/robot/types"
)

func TestMatchQueryFiltering(t *testing.T) {
	t.Run("empty query matches all robots", func(t *testing.T) {
		query := &api.ListQuery{}
		query.ApplyDefaults()

		assert.Equal(t, 1, query.Page)
		assert.Equal(t, 20, query.PageSize)
	})

	t.Run("default pagination values", func(t *testing.T) {
		query := &api.ListQuery{Page: 0, PageSize: 0}
		query.ApplyDefaults()

		assert.Equal(t, 1, query.Page)
		assert.Equal(t, 20, query.PageSize)
	})

	t.Run("caps pagesize at 100", func(t *testing.T) {
		query := &api.ListQuery{Page: 1, PageSize: 500}
		query.ApplyDefaults()

		assert.Equal(t, 100, query.PageSize)
	})
}

func TestPaginateRobots(t *testing.T) {
	robots := make([]*types.Robot, 5)
	for i := range robots {
		robots[i] = &types.Robot{MemberID: "robot-" + string(rune('a'+i))}
	}

	t.Run("first page", func(t *testing.T) {
		result := api.PaginateRobotsForTest(robots, &api.ListQuery{Page: 1, PageSize: 2})
		assert.Equal(t, 5, result.Total)
		assert.Len(t, result.Data, 2)
		assert.Equal(t, "robot-a", result.Data[0].MemberID)
	})

	t.Run("second page", func(t *testing.T) {
		result := api.PaginateRobotsForTest(robots, &api.ListQuery{Page: 2, PageSize: 2})
		assert.Equal(t, 5, result.Total)
		assert.Len(t, result.Data, 2)
		assert.Equal(t, "robot-c", result.Data[0].MemberID)
	})

	t.Run("last page partial", func(t *testing.T) {
		result := api.PaginateRobotsForTest(robots, &api.ListQuery{Page: 3, PageSize: 2})
		assert.Equal(t, 5, result.Total)
		assert.Len(t, result.Data, 1)
	})

	t.Run("page beyond total returns empty", func(t *testing.T) {
		result := api.PaginateRobotsForTest(robots, &api.ListQuery{Page: 10, PageSize: 2})
		assert.Equal(t, 5, result.Total)
		assert.Len(t, result.Data, 0)
	})
}

func TestLifecycleIsRunning(t *testing.T) {
	t.Run("not running by default", func(t *testing.T) {
		assert.False(t, api.IsRunning())
	})
}

func TestGetRobotValidationUnit(t *testing.T) {
	t.Run("returns error for empty member_id", func(t *testing.T) {
		ctx := types.NewContext(context.Background(), nil)
		robot, err := api.GetRobot(ctx, "")
		assert.Error(t, err)
		assert.Nil(t, robot)
		assert.Contains(t, err.Error(), "member_id is required")
	})
}

func TestGetRobotStatusValidationUnit(t *testing.T) {
	t.Run("returns error for empty member_id", func(t *testing.T) {
		ctx := types.NewContext(context.Background(), nil)
		status, err := api.GetRobotStatus(ctx, "")
		assert.Error(t, err)
		assert.Nil(t, status)
		assert.Contains(t, err.Error(), "member_id is required")
	})
}

func TestCreateRobotValidationUnit(t *testing.T) {
	ctx := types.NewContext(context.Background(), nil)

	t.Run("returns error for empty team_id", func(t *testing.T) {
		req := &api.CreateRobotRequest{
			MemberID:    "robot_unit_test_001",
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
			MemberID:    "robot_unit_test_001",
			TeamID:      "team_001",
			DisplayName: "",
		}
		result, err := api.CreateRobot(ctx, req)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "display_name is required")
	})
}

func TestRemoveRobotValidationUnit(t *testing.T) {
	t.Run("returns error for empty member_id", func(t *testing.T) {
		ctx := types.NewContext(context.Background(), nil)
		err := api.RemoveRobot(ctx, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "member_id is required")
	})
}

func TestUpdateRobotValidationUnit(t *testing.T) {
	t.Run("returns error for empty member_id", func(t *testing.T) {
		ctx := types.NewContext(context.Background(), nil)
		req := &api.UpdateRobotRequest{}
		result, err := api.UpdateRobot(ctx, "", req)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "member_id is required")
	})
}
