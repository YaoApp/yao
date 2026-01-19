package api_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/agent/robot/api"
	"github.com/yaoapp/yao/agent/robot/types"
	"github.com/yaoapp/yao/agent/testutils"
)

// TestGetRobotValidation tests parameter validation for GetRobot
func TestGetRobotValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	t.Run("returns error for empty member_id", func(t *testing.T) {
		ctx := types.NewContext(context.Background(), nil)
		robot, err := api.GetRobot(ctx, "")
		assert.Error(t, err)
		assert.Nil(t, robot)
		assert.Contains(t, err.Error(), "member_id is required")
	})

	t.Run("returns error for non-existent robot", func(t *testing.T) {
		ctx := types.NewContext(context.Background(), nil)
		robot, err := api.GetRobot(ctx, "non_existent_member_id_xyz")
		assert.Error(t, err)
		assert.Nil(t, robot)
	})
}

// TestListRobotsValidation tests parameter validation for ListRobots
func TestListRobotsValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	ctx := types.NewContext(context.Background(), nil)

	t.Run("applies default pagination when query is nil", func(t *testing.T) {
		result, err := api.ListRobots(ctx, nil)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 1, result.Page)
		assert.Equal(t, 20, result.PageSize)
	})

	t.Run("applies default pagination when values are zero", func(t *testing.T) {
		result, err := api.ListRobots(ctx, &api.ListQuery{
			Page:     0,
			PageSize: 0,
		})
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 1, result.Page)
		assert.Equal(t, 20, result.PageSize)
	})

	t.Run("caps pagesize at 100", func(t *testing.T) {
		result, err := api.ListRobots(ctx, &api.ListQuery{
			Page:     1,
			PageSize: 500,
		})
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 100, result.PageSize)
	})
}

// TestGetRobotStatusValidation tests parameter validation for GetRobotStatus
func TestGetRobotStatusValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	t.Run("returns error for empty member_id", func(t *testing.T) {
		ctx := types.NewContext(context.Background(), nil)
		status, err := api.GetRobotStatus(ctx, "")
		assert.Error(t, err)
		assert.Nil(t, status)
		assert.Contains(t, err.Error(), "member_id is required")
	})

	t.Run("returns error for non-existent robot", func(t *testing.T) {
		ctx := types.NewContext(context.Background(), nil)
		status, err := api.GetRobotStatus(ctx, "non_existent_member_id_xyz")
		assert.Error(t, err)
		assert.Nil(t, status)
	})
}
