package api_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// ==================== Robot CRUD API Tests ====================

// TestCreateRobotValidation tests parameter validation for CreateRobot
func TestCreateRobotValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	ctx := types.NewContext(context.Background(), nil)

	t.Run("auto_generates_member_id_when_empty", func(t *testing.T) {
		req := &api.CreateRobotRequest{
			MemberID:    "",
			TeamID:      "team_001",
			DisplayName: "Test Robot Auto ID",
		}
		result, err := api.CreateRobot(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Verify member_id was auto-generated (12-digit numeric)
		assert.NotEmpty(t, result.MemberID)
		assert.Len(t, result.MemberID, 12, "Auto-generated member_id should be 12 digits")

		// Cleanup
		_ = api.RemoveRobot(ctx, result.MemberID)
	})

	t.Run("returns_error_for_empty_team_id", func(t *testing.T) {
		req := &api.CreateRobotRequest{
			MemberID:    "robot_test_001",
			TeamID:      "",
			DisplayName: "Test Robot",
		}
		result, err := api.CreateRobot(ctx, req)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "team_id is required")
	})

	t.Run("returns_error_for_empty_display_name", func(t *testing.T) {
		req := &api.CreateRobotRequest{
			MemberID:    "robot_test_001",
			TeamID:      "team_001",
			DisplayName: "",
		}
		result, err := api.CreateRobot(ctx, req)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "display_name is required")
	})
}

// TestCreateRobot tests the CreateRobot API function
func TestCreateRobot(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Cleanup before and after
	cleanupAPITestRobots(t)
	defer cleanupAPITestRobots(t)

	ctx := types.NewContext(context.Background(), nil)

	t.Run("creates_robot_with_required_fields", func(t *testing.T) {
		req := &api.CreateRobotRequest{
			MemberID:    "api_robot_create_001",
			TeamID:      "api_team_001",
			DisplayName: "API Test Robot",
		}

		result, err := api.CreateRobot(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, result)

		assert.Equal(t, "api_robot_create_001", result.MemberID)
		assert.Equal(t, "api_team_001", result.TeamID)
		assert.Equal(t, "API Test Robot", result.DisplayName)
		assert.Equal(t, "active", result.Status)
		assert.Equal(t, "idle", result.RobotStatus)
	})

	t.Run("creates_robot_with_all_fields", func(t *testing.T) {
		autonomousMode := true
		req := &api.CreateRobotRequest{
			MemberID:       "api_robot_create_002",
			TeamID:         "api_team_002",
			DisplayName:    "Full Robot",
			Bio:            "A fully configured robot",
			SystemPrompt:   "You are a helpful assistant",
			Avatar:         "https://example.com/avatar.png",
			RoleID:         "admin",
			ManagerID:      "user_001",
			AutonomousMode: &autonomousMode,
			RobotEmail:     "fullrobot@test.com",
			LanguageModel:  "gpt-4",
			CostLimit:      100.0,
			RobotConfig: map[string]interface{}{
				"clock_mode":     "on",
				"max_concurrent": 3,
			},
		}

		result, err := api.CreateRobot(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, result)

		assert.Equal(t, "api_robot_create_002", result.MemberID)
		assert.Equal(t, "Full Robot", result.DisplayName)
		assert.Equal(t, "A fully configured robot", result.Bio)
		assert.Equal(t, "You are a helpful assistant", result.SystemPrompt)
		assert.Equal(t, "admin", result.RoleID)
		assert.True(t, result.AutonomousMode)
		assert.Equal(t, "fullrobot@test.com", result.RobotEmail)
		assert.Equal(t, "gpt-4", result.LanguageModel)
		assert.Equal(t, 100.0, result.CostLimit)
	})

	t.Run("creates_robot_with_auth_scope", func(t *testing.T) {
		req := &api.CreateRobotRequest{
			MemberID:    "api_robot_create_003",
			TeamID:      "api_team_003",
			DisplayName: "Robot with Auth",
			AuthScope: &api.AuthScope{
				CreatedBy: "user_123",
				TeamID:    "perm_team_001",
				TenantID:  "tenant_001",
			},
		}

		result, err := api.CreateRobot(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, result)

		assert.Equal(t, "api_robot_create_003", result.MemberID)
		// InvitedBy should be set from AuthScope.CreatedBy
		assert.Equal(t, "user_123", result.InvitedBy)
	})

	t.Run("returns_error_for_duplicate_member_id", func(t *testing.T) {
		req := &api.CreateRobotRequest{
			MemberID:    "api_robot_create_001", // Already created above
			TeamID:      "api_team_001",
			DisplayName: "Duplicate Robot",
		}

		result, err := api.CreateRobot(ctx, req)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "already exists")
	})
}

// TestUpdateRobot tests the UpdateRobot API function
func TestUpdateRobot(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	cleanupAPITestRobots(t)
	defer cleanupAPITestRobots(t)

	ctx := types.NewContext(context.Background(), nil)

	// Create a robot to update
	createReq := &api.CreateRobotRequest{
		MemberID:    "api_robot_update_001",
		TeamID:      "api_team_update",
		DisplayName: "Original Name",
		Bio:         "Original bio",
	}
	_, err := api.CreateRobot(ctx, createReq)
	require.NoError(t, err)

	t.Run("returns_error_for_empty_member_id", func(t *testing.T) {
		req := &api.UpdateRobotRequest{}
		result, err := api.UpdateRobot(ctx, "", req)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "member_id is required")
	})

	t.Run("returns_error_for_non_existent_robot", func(t *testing.T) {
		newName := "New Name"
		req := &api.UpdateRobotRequest{
			DisplayName: &newName,
		}
		result, err := api.UpdateRobot(ctx, "non_existent_robot", req)
		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("updates_display_name", func(t *testing.T) {
		newName := "Updated Name"
		req := &api.UpdateRobotRequest{
			DisplayName: &newName,
		}

		result, err := api.UpdateRobot(ctx, "api_robot_update_001", req)
		require.NoError(t, err)
		require.NotNil(t, result)

		assert.Equal(t, "Updated Name", result.DisplayName)
		// Bio should be unchanged
		assert.Equal(t, "Original bio", result.Bio)
	})

	t.Run("updates_multiple_fields", func(t *testing.T) {
		newBio := "New bio description"
		newPrompt := "Updated system prompt"
		autonomousMode := true

		req := &api.UpdateRobotRequest{
			Bio:            &newBio,
			SystemPrompt:   &newPrompt,
			AutonomousMode: &autonomousMode,
		}

		result, err := api.UpdateRobot(ctx, "api_robot_update_001", req)
		require.NoError(t, err)
		require.NotNil(t, result)

		assert.Equal(t, "New bio description", result.Bio)
		assert.Equal(t, "Updated system prompt", result.SystemPrompt)
		assert.True(t, result.AutonomousMode)
	})

	t.Run("updates_robot_status", func(t *testing.T) {
		newStatus := "working"
		req := &api.UpdateRobotRequest{
			RobotStatus: &newStatus,
		}

		result, err := api.UpdateRobot(ctx, "api_robot_update_001", req)
		require.NoError(t, err)
		require.NotNil(t, result)

		assert.Equal(t, "working", result.RobotStatus)
	})

	t.Run("updates_config", func(t *testing.T) {
		newConfig := map[string]interface{}{
			"clock_mode":     "off",
			"max_concurrent": 5,
		}
		req := &api.UpdateRobotRequest{
			RobotConfig: newConfig,
		}

		result, err := api.UpdateRobot(ctx, "api_robot_update_001", req)
		require.NoError(t, err)
		require.NotNil(t, result)

		assert.NotNil(t, result.RobotConfig)
	})
}

// TestRemoveRobot tests the RemoveRobot API function
func TestRemoveRobot(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	cleanupAPITestRobots(t)
	defer cleanupAPITestRobots(t)

	ctx := types.NewContext(context.Background(), nil)

	t.Run("returns_error_for_empty_member_id", func(t *testing.T) {
		err := api.RemoveRobot(ctx, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "member_id is required")
	})

	t.Run("returns_error_for_non_existent_robot", func(t *testing.T) {
		err := api.RemoveRobot(ctx, "non_existent_robot")
		assert.Error(t, err)
	})

	t.Run("removes_existing_robot", func(t *testing.T) {
		// Create a robot
		createReq := &api.CreateRobotRequest{
			MemberID:    "api_robot_remove_001",
			TeamID:      "api_team_remove",
			DisplayName: "Robot to Remove",
		}
		_, err := api.CreateRobot(ctx, createReq)
		require.NoError(t, err)

		// Verify it exists
		robot, err := api.GetRobot(ctx, "api_robot_remove_001")
		require.NoError(t, err)
		require.NotNil(t, robot)

		// Remove it
		err = api.RemoveRobot(ctx, "api_robot_remove_001")
		require.NoError(t, err)

		// Verify it's gone
		robot, err = api.GetRobot(ctx, "api_robot_remove_001")
		assert.Error(t, err) // Should return error for non-existent
	})
}

// TestGetRobotResponse tests the GetRobotResponse API function
func TestGetRobotResponse(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	cleanupAPITestRobots(t)
	defer cleanupAPITestRobots(t)

	ctx := types.NewContext(context.Background(), nil)

	// Create a robot
	autonomousMode := true
	createReq := &api.CreateRobotRequest{
		MemberID:       "api_robot_response_001",
		TeamID:         "api_team_response",
		DisplayName:    "Response Test Robot",
		Bio:            "Test bio for response",
		SystemPrompt:   "Test prompt",
		AutonomousMode: &autonomousMode,
		RobotEmail:     "response@test.com",
		CostLimit:      50.0,
	}
	_, err := api.CreateRobot(ctx, createReq)
	require.NoError(t, err)

	t.Run("returns_robot_response_format", func(t *testing.T) {
		result, err := api.GetRobotResponse(ctx, "api_robot_response_001")
		require.NoError(t, err)
		require.NotNil(t, result)

		// Verify all fields are present in response
		assert.Equal(t, "api_robot_response_001", result.MemberID)
		assert.Equal(t, "api_team_response", result.TeamID)
		assert.Equal(t, "Response Test Robot", result.DisplayName)
		assert.Equal(t, "Test bio for response", result.Bio)
		assert.Equal(t, "Test prompt", result.SystemPrompt)
		assert.True(t, result.AutonomousMode)
		assert.Equal(t, "response@test.com", result.RobotEmail)
		assert.Equal(t, 50.0, result.CostLimit)
		assert.Equal(t, "active", result.Status)
		assert.Equal(t, "idle", result.RobotStatus)
	})

	t.Run("returns_error_for_non_existent", func(t *testing.T) {
		result, err := api.GetRobotResponse(ctx, "non_existent")
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

// Note: cleanupAPITestRobots is defined in api_test.go (shared helper)
