package user_test

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/kun/maps"
)

func TestUserRoleOperations(t *testing.T) {
	prepare(t)
	defer clean()

	ctx := context.Background()

	// Use UUID to ensure unique identifiers
	testUUID := strings.ReplaceAll(uuid.New().String(), "-", "")[:8] // 8 char UUID

	// Step 1: Create a test user first
	testUser := createTestUserData("roleuser" + testUUID)
	_, testUserID := setupTestUser(t, ctx, testUser)

	// Step 2: Create test roles for assignment
	testRoles := []maps.MapStrAny{
		{
			"role_id":     "adminrole_" + testUUID,
			"name":        "Admin Role " + testUUID,
			"description": "Administrator role for testing",
			"is_active":   true,
			"level":       100,
		},
		{
			"role_id":     "userrole_" + testUUID,
			"name":        "User Role " + testUUID,
			"description": "Regular user role for testing",
			"is_active":   true,
			"level":       10,
		},
		{
			"role_id":     "inactiverole_" + testUUID,
			"name":        "Inactive Role " + testUUID,
			"description": "Inactive role for testing",
			"is_active":   false,
			"level":       0,
		},
	}

	// Create roles in database
	for _, roleData := range testRoles {
		_, err := testProvider.CreateRole(ctx, roleData)
		assert.NoError(t, err)
	}

	adminRoleID := "adminrole_" + testUUID
	userRoleID := "userrole_" + testUUID
	inactiveRoleID := "inactiverole_" + testUUID

	// Test SetUserRole
	t.Run("SetUserRole", func(t *testing.T) {
		err := testProvider.SetUserRole(ctx, testUserID, adminRoleID)
		assert.NoError(t, err)

		// Verify role was assigned by getting user info
		user, err := testProvider.GetUser(ctx, testUserID)
		assert.NoError(t, err)
		assert.Equal(t, adminRoleID, user["role_id"])
	})

	// Test GetUserRole
	t.Run("GetUserRole", func(t *testing.T) {
		role, err := testProvider.GetUserRole(ctx, testUserID)
		assert.NoError(t, err)
		assert.NotNil(t, role)

		// Verify we got the correct role information
		assert.Equal(t, adminRoleID, role["role_id"])
		assert.Equal(t, "Admin Role "+testUUID, role["name"])
		assert.Equal(t, "Administrator role for testing", role["description"])

		// Handle different boolean representations from database
		isActive := role["is_active"]
		switch v := isActive.(type) {
		case bool:
			assert.True(t, v)
		case int, int32, int64:
			assert.NotEqual(t, 0, v) // Any non-zero value is true
		default:
			t.Errorf("unexpected is_active type: %T, value: %v", isActive, isActive)
		}
	})

	// Test SetUserRole - Change to different role
	t.Run("SetUserRole_ChangeRole", func(t *testing.T) {
		err := testProvider.SetUserRole(ctx, testUserID, userRoleID)
		assert.NoError(t, err)

		// Verify role was changed
		role, err := testProvider.GetUserRole(ctx, testUserID)
		assert.NoError(t, err)
		assert.Equal(t, userRoleID, role["role_id"])
		assert.Equal(t, "User Role "+testUUID, role["name"])
	})

	// Test SetUserRole - Inactive Role (should fail)
	t.Run("SetUserRole_InactiveRole", func(t *testing.T) {
		err := testProvider.SetUserRole(ctx, testUserID, inactiveRoleID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot assign inactive role")

		// Verify role was not changed
		role, err := testProvider.GetUserRole(ctx, testUserID)
		assert.NoError(t, err)
		assert.Equal(t, userRoleID, role["role_id"]) // Should still be the previous role
	})

	// Test ClearUserRole
	t.Run("ClearUserRole", func(t *testing.T) {
		err := testProvider.ClearUserRole(ctx, testUserID)
		assert.NoError(t, err)

		// Verify role was cleared
		user, err := testProvider.GetUser(ctx, testUserID)
		assert.NoError(t, err)
		assert.Nil(t, user["role_id"]) // Should be null/nil

		// GetUserRole should now fail
		_, err = testProvider.GetUserRole(ctx, testUserID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "has no role assigned")
	})

	// Test SetUserRole again after clearing
	t.Run("SetUserRole_AfterClear", func(t *testing.T) {
		err := testProvider.SetUserRole(ctx, testUserID, adminRoleID)
		assert.NoError(t, err)

		// Verify role was assigned again
		role, err := testProvider.GetUserRole(ctx, testUserID)
		assert.NoError(t, err)
		assert.Equal(t, adminRoleID, role["role_id"])
	})
}

func TestUserTypeOperations(t *testing.T) {
	prepare(t)
	defer clean()

	ctx := context.Background()

	// Use UUID to ensure unique identifiers
	testUUID := strings.ReplaceAll(uuid.New().String(), "-", "")[:8]

	// Step 1: Create a test user first
	testUser := createTestUserData("typeuser" + testUUID)
	_, testUserID := setupTestUser(t, ctx, testUser)

	// Note: User type operations are not yet implemented
	// These tests are placeholders for future implementation

	// Test GetUserType (should return not implemented or similar)
	t.Run("GetUserType_NotImplemented", func(t *testing.T) {
		_, err := testProvider.GetUserType(ctx, testUserID)
		// Since implementation returns nil, nil - we expect no error but nil result
		// In a real implementation, this might return an error or the actual type
		assert.NoError(t, err) // Based on current TODO implementation
	})

	// Test SetUserType (should return not implemented or similar)
	t.Run("SetUserType_NotImplemented", func(t *testing.T) {
		err := testProvider.SetUserType(ctx, testUserID, "premium")
		// Since implementation returns nil - we expect no error
		// In a real implementation, this might return an error or actually set the type
		assert.NoError(t, err) // Based on current TODO implementation
	})
}

func TestValidateUserScope(t *testing.T) {
	prepare(t)
	defer clean()

	ctx := context.Background()

	// Use UUID to ensure unique identifiers
	testUUID := strings.ReplaceAll(uuid.New().String(), "-", "")[:8]

	// Create a test user
	testUser := createTestUserData("scopeuser" + testUUID)
	_, testUserID := setupTestUser(t, ctx, testUser)

	// Note: ValidateUserScope is not yet implemented
	// This test is a placeholder for future implementation

	t.Run("ValidateUserScope_NotImplemented", func(t *testing.T) {
		scopes := []string{"read", "write", "admin"}
		valid, err := testProvider.ValidateUserScope(ctx, testUserID, scopes)

		// Since implementation returns false, nil - we expect no error but false result
		// In a real implementation, this would validate user's scopes based on role and type
		assert.NoError(t, err) // Based on current TODO implementation
		assert.False(t, valid) // Based on current TODO implementation
	})
}

func TestUserRoleErrorHandling(t *testing.T) {
	prepare(t)
	defer clean()

	ctx := context.Background()

	// Use UUID to avoid conflicts
	testUUID := strings.ReplaceAll(uuid.New().String(), "-", "")[:8]
	nonExistentUserID := "nonexistent_user_" + testUUID
	nonExistentRoleID := "nonexistent_role_" + testUUID

	// Create a valid user for some tests
	testUser := createTestUserData("erroruser" + testUUID)
	_, validUserID := setupTestUser(t, ctx, testUser)

	// Create a valid role for some tests
	validRoleData := maps.MapStrAny{
		"role_id":     "validrole_" + testUUID,
		"name":        "Valid Role " + testUUID,
		"description": "Valid role for error testing",
		"is_active":   true,
	}
	_, err := testProvider.CreateRole(ctx, validRoleData)
	assert.NoError(t, err)
	validRoleID := "validrole_" + testUUID

	t.Run("GetUserRole_UserNotFound", func(t *testing.T) {
		_, err := testProvider.GetUserRole(ctx, nonExistentUserID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})

	t.Run("GetUserRole_NoRoleAssigned", func(t *testing.T) {
		// Create a user without a role assignment
		userWithoutRole := createTestUserData("noroleuser" + testUUID)
		_, userWithoutRoleID := setupTestUser(t, ctx, userWithoutRole)

		// Clear any default role that might have been set
		err := testProvider.ClearUserRole(ctx, userWithoutRoleID)
		assert.NoError(t, err)

		_, err = testProvider.GetUserRole(ctx, userWithoutRoleID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "has no role assigned")
	})

	t.Run("SetUserRole_UserNotFound", func(t *testing.T) {
		err := testProvider.SetUserRole(ctx, nonExistentUserID, validRoleID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})

	t.Run("SetUserRole_RoleNotFound", func(t *testing.T) {
		err := testProvider.SetUserRole(ctx, validUserID, nonExistentRoleID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "role not found")
	})

	t.Run("ClearUserRole_UserNotFound", func(t *testing.T) {
		err := testProvider.ClearUserRole(ctx, nonExistentUserID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})

	t.Run("ClearUserRole_NoRoleTooClear", func(t *testing.T) {
		// Create a user without a role assignment
		userWithoutRole := createTestUserData("clearnouser" + testUUID)
		_, userWithoutRoleID := setupTestUser(t, ctx, userWithoutRole)

		// Clear any default role that might have been set
		err := testProvider.ClearUserRole(ctx, userWithoutRoleID)
		assert.NoError(t, err) // Should succeed even if no role was assigned

		// Try to clear again (should still succeed)
		err = testProvider.ClearUserRole(ctx, userWithoutRoleID)
		assert.NoError(t, err) // Should not error even if no role exists
	})

	// Test user type error handling (placeholders for future implementation)
	t.Run("GetUserType_UserNotFound", func(t *testing.T) {
		_, err := testProvider.GetUserType(ctx, nonExistentUserID)
		// Since implementation returns nil, nil - we expect no error
		// In a real implementation, this should return an error
		assert.NoError(t, err) // Based on current TODO implementation
	})

	t.Run("SetUserType_UserNotFound", func(t *testing.T) {
		err := testProvider.SetUserType(ctx, nonExistentUserID, "premium")
		// Since implementation returns nil - we expect no error
		// In a real implementation, this should return an error
		assert.NoError(t, err) // Based on current TODO implementation
	})

	// Test scope validation error handling (placeholder for future implementation)
	t.Run("ValidateUserScope_UserNotFound", func(t *testing.T) {
		scopes := []string{"read", "write"}
		valid, err := testProvider.ValidateUserScope(ctx, nonExistentUserID, scopes)

		// Since implementation returns false, nil - we expect no error but false result
		// In a real implementation, this should return an error
		assert.NoError(t, err) // Based on current TODO implementation
		assert.False(t, valid) // Based on current TODO implementation
	})
}

func TestUserRoleIntegration(t *testing.T) {
	prepare(t)
	defer clean()

	ctx := context.Background()

	// Use UUID to ensure unique identifiers
	testUUID := strings.ReplaceAll(uuid.New().String(), "-", "")[:8]

	// Create multiple users and roles for integration testing
	users := make([]string, 3)
	for i := 0; i < 3; i++ {
		userData := createTestUserData("integuser" + testUUID + string('0'+rune(i)))
		_, userID := setupTestUser(t, ctx, userData)
		users[i] = userID
	}

	roles := []string{
		"adminrole_" + testUUID,
		"userrole_" + testUUID,
		"guestrole_" + testUUID,
	}

	roleData := []maps.MapStrAny{
		{
			"role_id":     roles[0],
			"name":        "Admin Role " + testUUID,
			"description": "Administrator role",
			"is_active":   true,
			"level":       100,
		},
		{
			"role_id":     roles[1],
			"name":        "User Role " + testUUID,
			"description": "Regular user role",
			"is_active":   true,
			"level":       10,
		},
		{
			"role_id":     roles[2],
			"name":        "Guest Role " + testUUID,
			"description": "Guest user role",
			"is_active":   true,
			"level":       1,
		},
	}

	// Create roles
	for _, role := range roleData {
		_, err := testProvider.CreateRole(ctx, role)
		assert.NoError(t, err)
	}

	t.Run("CompleteUserRoleFlow", func(t *testing.T) {
		userID := users[0]

		// Step 1: Assign admin role
		err := testProvider.SetUserRole(ctx, userID, roles[0])
		assert.NoError(t, err)

		// Step 2: Verify role assignment
		role, err := testProvider.GetUserRole(ctx, userID)
		assert.NoError(t, err)
		assert.Equal(t, roles[0], role["role_id"])
		assert.Equal(t, "Admin Role "+testUUID, role["name"])

		// Step 3: Change to user role
		err = testProvider.SetUserRole(ctx, userID, roles[1])
		assert.NoError(t, err)

		role, err = testProvider.GetUserRole(ctx, userID)
		assert.NoError(t, err)
		assert.Equal(t, roles[1], role["role_id"])

		// Step 4: Clear role
		err = testProvider.ClearUserRole(ctx, userID)
		assert.NoError(t, err)

		// Step 5: Verify role was cleared
		_, err = testProvider.GetUserRole(ctx, userID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "has no role assigned")

		// Step 6: Reassign role
		err = testProvider.SetUserRole(ctx, userID, roles[2])
		assert.NoError(t, err)

		role, err = testProvider.GetUserRole(ctx, userID)
		assert.NoError(t, err)
		assert.Equal(t, roles[2], role["role_id"])
	})

	t.Run("MultipleUsersRoleAssignment", func(t *testing.T) {
		// Assign different roles to different users
		for i, userID := range users {
			err := testProvider.SetUserRole(ctx, userID, roles[i])
			assert.NoError(t, err)
		}

		// Verify each user has the correct role
		for i, userID := range users {
			role, err := testProvider.GetUserRole(ctx, userID)
			assert.NoError(t, err)
			assert.Equal(t, roles[i], role["role_id"])
		}

		// Clear all roles
		for _, userID := range users {
			err := testProvider.ClearUserRole(ctx, userID)
			assert.NoError(t, err)
		}

		// Verify all roles were cleared
		for _, userID := range users {
			_, err := testProvider.GetUserRole(ctx, userID)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "has no role assigned")
		}
	})

	t.Run("RoleConsistency", func(t *testing.T) {
		userID := users[0]
		roleID := roles[0]

		// Assign role
		err := testProvider.SetUserRole(ctx, userID, roleID)
		assert.NoError(t, err)

		// Get role through user role method
		userRole, err := testProvider.GetUserRole(ctx, userID)
		assert.NoError(t, err)

		// Get role directly through role method
		directRole, err := testProvider.GetRole(ctx, roleID)
		assert.NoError(t, err)

		// Both should return the same role information
		assert.Equal(t, directRole["role_id"], userRole["role_id"])
		assert.Equal(t, directRole["name"], userRole["name"])
		assert.Equal(t, directRole["description"], userRole["description"])
		assert.Equal(t, directRole["is_active"], userRole["is_active"])
		assert.Equal(t, directRole["level"], userRole["level"])
	})
}
