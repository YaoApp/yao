package user_test

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/model"
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

	// Step 2: Create test types for assignment
	testTypes := []maps.MapStrAny{
		{
			"type_id":     "basictype_" + testUUID,
			"name":        "Basic Type " + testUUID,
			"description": "Basic user type for testing",
			"is_active":   true,
			"sort_order":  10,
		},
		{
			"type_id":     "premiumtype_" + testUUID,
			"name":        "Premium Type " + testUUID,
			"description": "Premium user type for testing",
			"is_active":   true,
			"sort_order":  20,
		},
		{
			"type_id":     "inactivetype_" + testUUID,
			"name":        "Inactive Type " + testUUID,
			"description": "Inactive type for testing",
			"is_active":   false,
			"sort_order":  0,
		},
	}

	// Create types in database
	for _, typeData := range testTypes {
		_, err := testProvider.CreateType(ctx, typeData)
		assert.NoError(t, err)
	}

	basicTypeID := "basictype_" + testUUID
	premiumTypeID := "premiumtype_" + testUUID
	inactiveTypeID := "inactivetype_" + testUUID

	// Test SetUserType
	t.Run("SetUserType", func(t *testing.T) {
		err := testProvider.SetUserType(ctx, testUserID, basicTypeID)
		assert.NoError(t, err)

		// Verify type was assigned by getting user info
		user, err := testProvider.GetUser(ctx, testUserID)
		assert.NoError(t, err)
		assert.Equal(t, basicTypeID, user["type_id"])
	})

	// Test GetUserType
	t.Run("GetUserType", func(t *testing.T) {
		userType, err := testProvider.GetUserType(ctx, testUserID)
		assert.NoError(t, err)
		assert.NotNil(t, userType)

		// Verify we got the correct type information
		assert.Equal(t, basicTypeID, userType["type_id"])
		assert.Equal(t, "Basic Type "+testUUID, userType["name"])
		assert.Equal(t, "Basic user type for testing", userType["description"])

		// Handle different boolean representations from database
		isActive := userType["is_active"]
		switch v := isActive.(type) {
		case bool:
			assert.True(t, v)
		case int, int32, int64:
			assert.NotEqual(t, 0, v) // Any non-zero value is true
		default:
			t.Errorf("unexpected is_active type: %T, value: %v", isActive, isActive)
		}
	})

	// Test SetUserType - Change to different type
	t.Run("SetUserType_ChangeType", func(t *testing.T) {
		err := testProvider.SetUserType(ctx, testUserID, premiumTypeID)
		assert.NoError(t, err)

		// Verify type was changed
		userType, err := testProvider.GetUserType(ctx, testUserID)
		assert.NoError(t, err)
		assert.Equal(t, premiumTypeID, userType["type_id"])
		assert.Equal(t, "Premium Type "+testUUID, userType["name"])
	})

	// Test SetUserType - Inactive Type (should fail)
	t.Run("SetUserType_InactiveType", func(t *testing.T) {
		err := testProvider.SetUserType(ctx, testUserID, inactiveTypeID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot assign inactive type")

		// Verify type was not changed
		userType, err := testProvider.GetUserType(ctx, testUserID)
		assert.NoError(t, err)
		assert.Equal(t, premiumTypeID, userType["type_id"]) // Should still be the previous type
	})

	// Test ClearUserType
	t.Run("ClearUserType", func(t *testing.T) {
		err := testProvider.ClearUserType(ctx, testUserID)
		assert.NoError(t, err)

		// Verify type was cleared
		_, err = testProvider.GetUserType(ctx, testUserID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "has no type assigned")

		// Verify user still exists
		user, err := testProvider.GetUser(ctx, testUserID)
		assert.NoError(t, err)
		assert.Equal(t, testUserID, user["user_id"])
		assert.Nil(t, user["type_id"]) // type_id should be null
	})

	// Test SetUserType - After Clear
	t.Run("SetUserType_AfterClear", func(t *testing.T) {
		// Re-assign a type after clearing
		err := testProvider.SetUserType(ctx, testUserID, basicTypeID)
		assert.NoError(t, err)

		// Verify type was assigned
		userType, err := testProvider.GetUserType(ctx, testUserID)
		assert.NoError(t, err)
		assert.Equal(t, basicTypeID, userType["type_id"])
		assert.Equal(t, "Basic Type "+testUUID, userType["name"])
	})
}

func TestValidateUserScope(t *testing.T) {
	prepare(t)
	defer clean()

	ctx := context.Background()

	// Use UUID to ensure unique identifiers
	testUUID := strings.ReplaceAll(uuid.New().String(), "-", "")[:8]

	// Step 1: Create test role with specific permissions
	testRole := maps.MapStrAny{
		"role_id":     "scoperole_" + testUUID,
		"name":        "Scope Test Role " + testUUID,
		"description": "Role for testing scope validation",
		"is_active":   true,
		"permissions": map[string]interface{}{
			"read":        true,
			"write":       true,
			"admin.read":  true,
			"admin.write": false,
			"delete":      false,
		},
		"restricted_permissions": []string{
			"system.config",
			"root.access",
		},
	}

	_, err := testProvider.CreateRole(ctx, testRole)
	assert.NoError(t, err)

	// Step 2: Create test type with scope limitations
	testType := maps.MapStrAny{
		"type_id":     "scopetype_" + testUUID,
		"name":        "Scope Test Type " + testUUID,
		"description": "Type for testing scope validation",
		"is_active":   true,
		"features": map[string]interface{}{
			"api_access": true,
			"scope_limits": []interface{}{
				"read", "write", "admin.read", // Allowed scopes
			},
		},
	}

	_, err = testProvider.CreateType(ctx, testType)
	assert.NoError(t, err)

	// Step 3: Create test user and assign role and type
	testUser := createTestUserData("scopeuser" + testUUID)
	_, testUserID := setupTestUser(t, ctx, testUser)

	roleID := "scoperole_" + testUUID
	typeID := "scopetype_" + testUUID

	// Assign role and type to user
	err = testProvider.SetUserRole(ctx, testUserID, roleID)
	assert.NoError(t, err)

	err = testProvider.SetUserType(ctx, testUserID, typeID)
	assert.NoError(t, err)

	// Test various scope validation scenarios
	t.Run("ValidateUserScope_EmptyScopes", func(t *testing.T) {
		valid, err := testProvider.ValidateUserScope(ctx, testUserID, []string{})
		assert.NoError(t, err)
		assert.True(t, valid) // Empty scopes should always be valid
	})

	t.Run("ValidateUserScope_ValidSingleScope", func(t *testing.T) {
		valid, err := testProvider.ValidateUserScope(ctx, testUserID, []string{"read"})
		assert.NoError(t, err)
		assert.True(t, valid) // "read" is allowed by both role and type
	})

	t.Run("ValidateUserScope_ValidMultipleScopes", func(t *testing.T) {
		valid, err := testProvider.ValidateUserScope(ctx, testUserID, []string{"read", "write"})
		assert.NoError(t, err)
		assert.True(t, valid) // Both "read" and "write" are allowed
	})

	t.Run("ValidateUserScope_ValidAdminReadScope", func(t *testing.T) {
		valid, err := testProvider.ValidateUserScope(ctx, testUserID, []string{"admin.read"})
		assert.NoError(t, err)
		assert.True(t, valid) // "admin.read" is allowed by both role and type
	})

	t.Run("ValidateUserScope_InvalidRolePermission", func(t *testing.T) {
		valid, err := testProvider.ValidateUserScope(ctx, testUserID, []string{"admin.write"})
		assert.NoError(t, err)
		assert.False(t, valid) // "admin.write" is denied by role permissions
	})

	t.Run("ValidateUserScope_RestrictedPermission", func(t *testing.T) {
		valid, err := testProvider.ValidateUserScope(ctx, testUserID, []string{"system.config"})
		assert.NoError(t, err)
		assert.False(t, valid) // "system.config" is in restricted permissions
	})

	t.Run("ValidateUserScope_TypeScopeLimitation", func(t *testing.T) {
		valid, err := testProvider.ValidateUserScope(ctx, testUserID, []string{"delete"})
		assert.NoError(t, err)
		assert.False(t, valid) // "delete" is not in type's scope_limits
	})

	t.Run("ValidateUserScope_MixedValidInvalid", func(t *testing.T) {
		valid, err := testProvider.ValidateUserScope(ctx, testUserID, []string{"read", "delete"})
		assert.NoError(t, err)
		assert.False(t, valid) // Should fail because "delete" is not allowed
	})

	t.Run("ValidateUserScope_NonExistentScope", func(t *testing.T) {
		valid, err := testProvider.ValidateUserScope(ctx, testUserID, []string{"nonexistent.permission"})
		assert.NoError(t, err)
		assert.False(t, valid) // Non-existent permissions should be denied
	})

	// Test user without role
	t.Run("ValidateUserScope_UserWithoutRole", func(t *testing.T) {
		// Create a user without role assignment
		userWithoutRole := createTestUserData("noroleuser" + testUUID)
		_, userWithoutRoleID := setupTestUser(t, ctx, userWithoutRole)

		// Clear any default role that might have been set
		err := testProvider.ClearUserRole(ctx, userWithoutRoleID)
		assert.NoError(t, err)

		// User without role should only have access to empty scopes
		valid, err := testProvider.ValidateUserScope(ctx, userWithoutRoleID, []string{})
		assert.NoError(t, err)
		assert.True(t, valid) // Empty scopes should be valid

		// Users without roles have minimal access (empty scopes only)
		valid, err = testProvider.ValidateUserScope(ctx, userWithoutRoleID, []string{"read"})
		assert.NoError(t, err)
		assert.False(t, valid) // Should return false - users without roles can only access empty scopes
	})

	// Test user without type (type restrictions should not apply)
	t.Run("ValidateUserScope_UserWithoutType", func(t *testing.T) {
		// Create a user with role but without type
		userWithoutType := createTestUserData("notypeuser" + testUUID)
		userWithoutType.TypeID = "" // Explicitly clear type_id
		_, userWithoutTypeID := setupTestUser(t, ctx, userWithoutType)

		// Assign role but no type
		err := testProvider.SetUserRole(ctx, userWithoutTypeID, roleID)
		assert.NoError(t, err)

		// Manually clear type_id to ensure user has no type
		userModel := model.Select("__yao.user")
		_, err = userModel.UpdateWhere(model.QueryParam{
			Wheres: []model.QueryWhere{
				{Column: "user_id", Value: userWithoutTypeID},
			},
			Limit: 1,
		}, maps.MapStrAny{
			"type_id": nil, // Set type_id to null
		})
		assert.NoError(t, err)

		// Verify user has no type assigned
		_, err = testProvider.GetUserType(ctx, userWithoutTypeID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "has no type assigned")

		// Should be able to access permissions allowed by role (no type restrictions)
		valid, err := testProvider.ValidateUserScope(ctx, userWithoutTypeID, []string{"read", "write"})
		assert.NoError(t, err)
		assert.True(t, valid) // Role allows these, no type restrictions

		// Should still be restricted by role permissions
		valid, err = testProvider.ValidateUserScope(ctx, userWithoutTypeID, []string{"admin.write"})
		assert.NoError(t, err)
		assert.False(t, valid) // Role denies this
	})

	// Test type without scope limits
	t.Run("ValidateUserScope_TypeWithoutScopeLimits", func(t *testing.T) {
		// Create a type without scope limits
		openType := maps.MapStrAny{
			"type_id":     "opentype_" + testUUID,
			"name":        "Open Type " + testUUID,
			"description": "Type without scope limitations",
			"is_active":   true,
			"features": map[string]interface{}{
				"api_access": true,
				// No scope_limits - should allow anything the role permits
			},
		}

		_, err := testProvider.CreateType(ctx, openType)
		assert.NoError(t, err)

		// Create user with role and open type
		openUser := createTestUserData("openuser" + testUUID)
		_, openUserID := setupTestUser(t, ctx, openUser)

		err = testProvider.SetUserRole(ctx, openUserID, roleID)
		assert.NoError(t, err)

		err = testProvider.SetUserType(ctx, openUserID, "opentype_"+testUUID)
		assert.NoError(t, err)

		// Should be able to access any permission allowed by role
		valid, err := testProvider.ValidateUserScope(ctx, openUserID, []string{"read", "write", "admin.read"})
		assert.NoError(t, err)
		assert.True(t, valid) // Type has no limitations, role allows these

		// Should still be restricted by role permissions
		valid, err = testProvider.ValidateUserScope(ctx, openUserID, []string{"admin.write"})
		assert.NoError(t, err)
		assert.False(t, valid) // Role denies this
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

	// Create a valid type for some tests
	validTypeData := maps.MapStrAny{
		"type_id":     "validtype_" + testUUID,
		"name":        "Valid Type " + testUUID,
		"description": "Valid type for error testing",
		"is_active":   true,
	}
	_, err = testProvider.CreateType(ctx, validTypeData)
	assert.NoError(t, err)
	validTypeID := "validtype_" + testUUID

	// Test user type error handling
	t.Run("GetUserType_UserNotFound", func(t *testing.T) {
		_, err := testProvider.GetUserType(ctx, nonExistentUserID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})

	t.Run("GetUserType_NoTypeAssigned", func(t *testing.T) {
		// Create a user without a type assignment
		userWithoutType := createTestUserData("notypeuser" + testUUID)
		userWithoutType.TypeID = "" // Explicitly clear type_id
		_, userWithoutTypeID := setupTestUser(t, ctx, userWithoutType)

		// Manually clear type_id to ensure user has no type
		userModel := model.Select("__yao.user")
		_, err = userModel.UpdateWhere(model.QueryParam{
			Wheres: []model.QueryWhere{
				{Column: "user_id", Value: userWithoutTypeID},
			},
			Limit: 1,
		}, maps.MapStrAny{
			"type_id": nil, // Set type_id to null
		})
		assert.NoError(t, err)

		_, err = testProvider.GetUserType(ctx, userWithoutTypeID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "has no type assigned")
	})

	t.Run("SetUserType_UserNotFound", func(t *testing.T) {
		err := testProvider.SetUserType(ctx, nonExistentUserID, validTypeID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})

	t.Run("SetUserType_TypeNotFound", func(t *testing.T) {
		nonExistentTypeID := "nonexistent_type_" + testUUID
		err := testProvider.SetUserType(ctx, validUserID, nonExistentTypeID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "type not found")
	})

	t.Run("ClearUserType_UserNotFound", func(t *testing.T) {
		err := testProvider.ClearUserType(ctx, nonExistentUserID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})

	t.Run("ClearUserType_NoTypeTooClear", func(t *testing.T) {
		// Create a user without type assignment
		userWithoutType := createTestUserData("clearnotypeuser" + testUUID)
		userWithoutType.TypeID = "" // Explicitly clear type_id
		_, userWithoutTypeID := setupTestUser(t, ctx, userWithoutType)

		// Manually clear type_id to ensure user has no type
		userModel := model.Select("__yao.user")
		_, err = userModel.UpdateWhere(model.QueryParam{
			Wheres: []model.QueryWhere{
				{Column: "user_id", Value: userWithoutTypeID},
			},
			Limit: 1,
		}, maps.MapStrAny{
			"type_id": nil, // Set type_id to null
		})
		assert.NoError(t, err)

		// Try to clear again (should still succeed)
		err = testProvider.ClearUserType(ctx, userWithoutTypeID)
		assert.NoError(t, err) // Should not error even if no type exists
	})

	// Test scope validation error handling
	t.Run("ValidateUserScope_UserNotFound", func(t *testing.T) {
		scopes := []string{"read", "write"}
		valid, err := testProvider.ValidateUserScope(ctx, nonExistentUserID, scopes)
		assert.Error(t, err)
		assert.False(t, valid)
		assert.Contains(t, err.Error(), "user not found")
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
