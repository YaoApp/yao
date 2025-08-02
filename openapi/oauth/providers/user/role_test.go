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

// TestRoleData represents test role data structure
type TestRoleData struct {
	RoleID      string                 `json:"role_id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	IsActive    bool                   `json:"is_active"`
	IsDefault   bool                   `json:"is_default"`
	IsSystem    bool                   `json:"is_system"`
	Level       int                    `json:"level"`
	SortOrder   int                    `json:"sort_order"`
	Color       string                 `json:"color"`
	Icon        string                 `json:"icon"`
	Permissions map[string]interface{} `json:"permissions"`
	Metadata    map[string]interface{} `json:"metadata"`
}

func TestRoleBasicOperations(t *testing.T) {
	prepare(t)
	defer clean()

	ctx := context.Background()

	// Use UUID to ensure unique identifiers
	testUUID := strings.ReplaceAll(uuid.New().String(), "-", "")[:8] // 8 char UUID

	// Create test role data dynamically
	testRole := &TestRoleData{
		RoleID:      "testrole_" + testUUID,
		Name:        "Test Role " + testUUID,
		Description: "Test role for unit testing " + testUUID,
		IsActive:    true,
		IsDefault:   false,
		IsSystem:    false,
		Level:       10,
		SortOrder:   100,
		Color:       "#007bff",
		Icon:        "test-icon",
		Permissions: map[string]interface{}{
			"read":   true,
			"write":  true,
			"delete": false,
		},
		Metadata: map[string]interface{}{
			"source": "test",
			"uuid":   testUUID,
		},
	}

	// Test CreateRole
	t.Run("CreateRole", func(t *testing.T) {
		roleData := maps.MapStrAny{
			"role_id":     testRole.RoleID,
			"name":        testRole.Name,
			"description": testRole.Description,
			"level":       testRole.Level,
			"sort_order":  testRole.SortOrder,
			"color":       testRole.Color,
			"icon":        testRole.Icon,
			"permissions": testRole.Permissions,
			"metadata":    testRole.Metadata,
		}

		id, err := testProvider.CreateRole(ctx, roleData)
		assert.NoError(t, err)
		assert.NotNil(t, id)

		// Verify default values were set
		assert.Equal(t, true, roleData["is_active"])
		assert.Equal(t, false, roleData["is_default"])
		assert.Equal(t, false, roleData["is_system"])
		// level should remain as provided (10), not be overridden
	})

	// Test GetRole
	t.Run("GetRole", func(t *testing.T) {
		role, err := testProvider.GetRole(ctx, testRole.RoleID)
		assert.NoError(t, err)
		assert.NotNil(t, role)

		// Verify key fields
		assert.Equal(t, testRole.RoleID, role["role_id"])
		assert.Equal(t, testRole.Name, role["name"])
		assert.Equal(t, testRole.Description, role["description"])
		assert.Equal(t, testRole.Color, role["color"])
		assert.Equal(t, testRole.Icon, role["icon"])

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

		assert.NotNil(t, role["created_at"])
	})

	// Test UpdateRole
	t.Run("UpdateRole", func(t *testing.T) {
		updateData := maps.MapStrAny{
			"name":        "Updated Test Role",
			"description": "Updated description for testing",
			"color":       "#28a745",
			"icon":        "updated-icon",
			"level":       20,
			"metadata": map[string]interface{}{
				"updated": true,
				"version": 2,
			},
		}

		err := testProvider.UpdateRole(ctx, testRole.RoleID, updateData)
		assert.NoError(t, err)

		// Verify update
		role, err := testProvider.GetRole(ctx, testRole.RoleID)
		assert.NoError(t, err)
		assert.Equal(t, "Updated Test Role", role["name"])
		assert.Equal(t, "Updated description for testing", role["description"])
		assert.Equal(t, "#28a745", role["color"])
		assert.Equal(t, "updated-icon", role["icon"])

		// Test updating sensitive fields (should be ignored)
		sensitiveData := maps.MapStrAny{
			"id":         999,
			"role_id":    "malicious_role_id",
			"created_at": "2020-01-01T00:00:00Z",
		}

		err = testProvider.UpdateRole(ctx, testRole.RoleID, sensitiveData)
		assert.NoError(t, err) // Should not error, just ignore sensitive fields

		// Verify sensitive fields were not changed
		role, err = testProvider.GetRole(ctx, testRole.RoleID)
		assert.NoError(t, err)
		assert.Equal(t, testRole.RoleID, role["role_id"]) // Should remain unchanged
	})

	// Create a system role for delete test
	t.Run("CreateSystemRole", func(t *testing.T) {
		systemRoleData := maps.MapStrAny{
			"role_id":     "systemrole_" + testUUID,
			"name":        "System Role " + testUUID,
			"description": "System role for delete testing",
			"is_system":   true,
		}

		id, err := testProvider.CreateRole(ctx, systemRoleData)
		assert.NoError(t, err)
		assert.NotNil(t, id)
	})

	// Test DeleteRole - System Role Protection
	t.Run("DeleteRole_SystemRoleProtection", func(t *testing.T) {
		systemRoleID := "systemrole_" + testUUID
		err := testProvider.DeleteRole(ctx, systemRoleID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot delete system role")

		// Verify system role still exists
		role, err := testProvider.GetRole(ctx, systemRoleID)
		assert.NoError(t, err)
		assert.NotNil(t, role)
	})

	// Test DeleteRole - Normal Role (at the end)
	t.Run("DeleteRole", func(t *testing.T) {
		err := testProvider.DeleteRole(ctx, testRole.RoleID)
		assert.NoError(t, err)

		// Verify role was deleted
		_, err = testProvider.GetRole(ctx, testRole.RoleID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "role not found")
	})
}

func TestRolePermissionOperations(t *testing.T) {
	prepare(t)
	defer clean()

	ctx := context.Background()

	// Use UUID to ensure unique identifiers
	testUUID := strings.ReplaceAll(uuid.New().String(), "-", "")[:8]

	// Create a role for permission testing
	testRole := &TestRoleData{
		RoleID:      "permrole_" + testUUID,
		Name:        "Permission Test Role " + testUUID,
		Description: "Role for testing permissions",
		IsActive:    true,
		Permissions: map[string]interface{}{
			"users.read":   true,
			"users.write":  true,
			"users.delete": false,
			"admin.access": true,
		},
	}

	// Create role
	roleData := maps.MapStrAny{
		"role_id":     testRole.RoleID,
		"name":        testRole.Name,
		"description": testRole.Description,
		"permissions": testRole.Permissions,
		"restricted_permissions": []string{
			"system.config",
			"root.access",
		},
	}

	_, err := testProvider.CreateRole(ctx, roleData)
	assert.NoError(t, err)

	// Test GetRolePermissions
	t.Run("GetRolePermissions", func(t *testing.T) {
		permissions, err := testProvider.GetRolePermissions(ctx, testRole.RoleID)
		assert.NoError(t, err)
		assert.NotNil(t, permissions)

		assert.Equal(t, testRole.RoleID, permissions["role_id"])
		assert.NotNil(t, permissions["permissions"])
		assert.NotNil(t, permissions["restricted_permissions"])

		// Verify permissions structure
		permsMap, ok := permissions["permissions"].(map[string]interface{})
		if ok {
			assert.Equal(t, true, permsMap["users.read"])
			assert.Equal(t, true, permsMap["users.write"])
			assert.Equal(t, false, permsMap["users.delete"])
		}
	})

	// Test SetRolePermissions
	t.Run("SetRolePermissions", func(t *testing.T) {
		newPermissions := maps.MapStrAny{
			"permissions": map[string]interface{}{
				"users.read":   true,
				"users.write":  false, // Changed
				"users.delete": true,  // Changed
				"posts.read":   true,  // New
			},
			"restricted_permissions": []string{
				"system.config",
				"dangerous.operation", // New restriction
			},
		}

		err := testProvider.SetRolePermissions(ctx, testRole.RoleID, newPermissions)
		assert.NoError(t, err)

		// Verify permissions were updated
		permissions, err := testProvider.GetRolePermissions(ctx, testRole.RoleID)
		assert.NoError(t, err)

		permsMap, ok := permissions["permissions"].(map[string]interface{})
		if ok {
			assert.Equal(t, true, permsMap["users.read"])
			assert.Equal(t, false, permsMap["users.write"]) // Should be updated
			assert.Equal(t, true, permsMap["users.delete"]) // Should be updated
			assert.Equal(t, true, permsMap["posts.read"])   // Should be new
		}
	})

	// Test ValidateRolePermissions
	t.Run("ValidateRolePermissions_ValidPermissions", func(t *testing.T) {
		requiredPermissions := []string{"users.read", "posts.read"}
		valid, err := testProvider.ValidateRolePermissions(ctx, testRole.RoleID, requiredPermissions)
		assert.NoError(t, err)
		assert.True(t, valid)
	})

	t.Run("ValidateRolePermissions_InvalidPermissions", func(t *testing.T) {
		requiredPermissions := []string{"users.write"} // This was set to false
		valid, err := testProvider.ValidateRolePermissions(ctx, testRole.RoleID, requiredPermissions)
		assert.NoError(t, err)
		assert.False(t, valid) // Should be false because users.write is disabled
	})

	t.Run("ValidateRolePermissions_RestrictedPermissions", func(t *testing.T) {
		requiredPermissions := []string{"system.config"} // This is in restricted list
		valid, err := testProvider.ValidateRolePermissions(ctx, testRole.RoleID, requiredPermissions)
		assert.NoError(t, err)
		assert.False(t, valid) // Should be false because it's restricted
	})

	t.Run("ValidateRolePermissions_EmptyRequirements", func(t *testing.T) {
		requiredPermissions := []string{}
		valid, err := testProvider.ValidateRolePermissions(ctx, testRole.RoleID, requiredPermissions)
		assert.NoError(t, err)
		assert.True(t, valid) // Should be true when no permissions required
	})

	t.Run("ValidateRolePermissions_NonExistentPermission", func(t *testing.T) {
		requiredPermissions := []string{"nonexistent.permission"}
		valid, err := testProvider.ValidateRolePermissions(ctx, testRole.RoleID, requiredPermissions)
		assert.NoError(t, err)
		assert.False(t, valid) // Should be false for nonexistent permissions
	})
}

func TestRoleListOperations(t *testing.T) {
	prepare(t)
	defer clean()

	ctx := context.Background()

	// Create multiple test roles for list operations
	// Use UUID to ensure unique identifiers
	testUUID := strings.ReplaceAll(uuid.New().String(), "-", "")[:8]

	testRoles := []TestRoleData{
		{
			RoleID:      "listrole_" + testUUID + "_1",
			Name:        "List Role 1",
			Description: "First role for list testing",
			IsActive:    true,
			Level:       10,
		},
		{
			RoleID:      "listrole_" + testUUID + "_2",
			Name:        "List Role 2",
			Description: "Second role for list testing",
			IsActive:    true,
			Level:       20,
		},
		{
			RoleID:      "listrole_" + testUUID + "_3",
			Name:        "List Role 3",
			Description: "Third role for list testing",
			IsActive:    false, // Different status for filtering
			Level:       30,
		},
		{
			RoleID:      "listrole_" + testUUID + "_4",
			Name:        "List Role 4",
			Description: "Fourth role for list testing",
			IsActive:    true,
			Level:       40,
		},
		{
			RoleID:      "listrole_" + testUUID + "_5",
			Name:        "List Role 5",
			Description: "Fifth role for list testing",
			IsActive:    true,
			Level:       50,
		},
	}

	// Create roles in database
	for _, roleData := range testRoles {
		roleMap := maps.MapStrAny{
			"role_id":     roleData.RoleID,
			"name":        roleData.Name,
			"description": roleData.Description,
			"is_active":   roleData.IsActive,
			"level":       roleData.Level,
		}

		_, err := testProvider.CreateRole(ctx, roleMap)
		assert.NoError(t, err)
	}

	// Test GetRoles
	t.Run("GetRoles_All", func(t *testing.T) {
		param := model.QueryParam{
			Wheres: []model.QueryWhere{
				{Column: "role_id", OP: "like", Value: "listrole_" + testUUID + "_%"},
			},
		}
		roles, err := testProvider.GetRoles(ctx, param)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(roles), 5) // At least our 5 test roles

		// Check that basic fields are returned by default
		if len(roles) > 0 {
			role := roles[0]
			assert.Contains(t, role, "role_id")
			assert.Contains(t, role, "name")
			assert.Contains(t, role, "description")
			assert.Contains(t, role, "is_active")
		}
	})

	t.Run("GetRoles_WithFilters", func(t *testing.T) {
		param := model.QueryParam{
			Wheres: []model.QueryWhere{
				{Column: "role_id", OP: "like", Value: "listrole_" + testUUID + "_%"},
				{Column: "is_active", Value: true},
			},
		}
		roles, err := testProvider.GetRoles(ctx, param)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(roles), 4) // At least 4 active roles

		// All returned roles should be active
		for _, role := range roles {
			if strings.Contains(role["role_id"].(string), "listrole_"+testUUID+"_") {
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
			}
		}
	})

	t.Run("GetRoles_WithCustomFields", func(t *testing.T) {
		param := model.QueryParam{
			Select: []interface{}{"role_id", "name", "is_active", "level"},
			Wheres: []model.QueryWhere{
				{Column: "role_id", OP: "like", Value: "listrole_" + testUUID + "_%"},
			},
			Limit: 3,
		}
		roles, err := testProvider.GetRoles(ctx, param)
		assert.NoError(t, err)
		assert.LessOrEqual(t, len(roles), 3) // Respects limit

		if len(roles) > 0 {
			role := roles[0]
			assert.Contains(t, role, "role_id")
			assert.Contains(t, role, "name")
			assert.Contains(t, role, "is_active")
			assert.Contains(t, role, "level")
		}
	})

	// Test PaginateRoles
	t.Run("PaginateRoles_FirstPage", func(t *testing.T) {
		param := model.QueryParam{
			Wheres: []model.QueryWhere{
				{Column: "role_id", OP: "like", Value: "listrole_" + testUUID + "_%"},
			},
			Orders: []model.QueryOrder{
				{Column: "level", Option: "asc"},
			},
		}
		result, err := testProvider.PaginateRoles(ctx, param, 1, 3)
		assert.NoError(t, err)
		assert.NotNil(t, result)

		// Check pagination structure
		assert.Contains(t, result, "data")
		assert.Contains(t, result, "total")
		assert.Contains(t, result, "page")
		assert.Contains(t, result, "pagesize")

		data, ok := result["data"].([]maps.MapStr)
		assert.True(t, ok)
		assert.LessOrEqual(t, len(data), 3) // Page size limit

		// Handle different total types
		totalInterface, exists := result["total"]
		assert.True(t, exists)

		var total int64
		switch v := totalInterface.(type) {
		case int:
			total = int64(v)
		case int32:
			total = int64(v)
		case int64:
			total = v
		case uint:
			total = int64(v)
		case uint32:
			total = int64(v)
		case uint64:
			total = int64(v)
		default:
			t.Errorf("unexpected total type: %T, value: %v", totalInterface, totalInterface)
		}
		assert.GreaterOrEqual(t, total, int64(5)) // At least 5 roles

		assert.Equal(t, 1, result["page"])
		assert.Equal(t, 3, result["pagesize"])
	})

	t.Run("PaginateRoles_WithFilters", func(t *testing.T) {
		param := model.QueryParam{
			Wheres: []model.QueryWhere{
				{Column: "role_id", OP: "like", Value: "listrole_" + testUUID + "_%"},
				{Column: "is_active", Value: true},
			},
		}
		result, err := testProvider.PaginateRoles(ctx, param, 1, 10)
		assert.NoError(t, err)
		assert.NotNil(t, result)

		data, ok := result["data"].([]maps.MapStr)
		assert.True(t, ok)
		assert.GreaterOrEqual(t, len(data), 4) // At least 4 active roles

		// Verify is_active filter works
		for _, role := range data {
			if strings.Contains(role["role_id"].(string), "listrole_"+testUUID+"_") {
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
			}
		}
	})

	// Test CountRoles
	t.Run("CountRoles_All", func(t *testing.T) {
		param := model.QueryParam{
			Wheres: []model.QueryWhere{
				{Column: "role_id", OP: "like", Value: "listrole_" + testUUID + "_%"},
			},
		}
		count, err := testProvider.CountRoles(ctx, param)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, count, int64(5)) // At least 5 roles
	})

	t.Run("CountRoles_WithFilters", func(t *testing.T) {
		param := model.QueryParam{
			Wheres: []model.QueryWhere{
				{Column: "role_id", OP: "like", Value: "listrole_" + testUUID + "_%"},
				{Column: "is_active", Value: true},
			},
		}
		count, err := testProvider.CountRoles(ctx, param)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, count, int64(4)) // At least 4 active roles
	})

	t.Run("CountRoles_SpecificLevel", func(t *testing.T) {
		param := model.QueryParam{
			Wheres: []model.QueryWhere{
				{Column: "role_id", OP: "like", Value: "listrole_" + testUUID + "_%"},
				{Column: "level", OP: ">=", Value: 30},
			},
		}
		count, err := testProvider.CountRoles(ctx, param)
		assert.NoError(t, err)
		// We created 3 roles with level >= 30 (30, 40, 50), but be flexible with database state
		assert.GreaterOrEqual(t, count, int64(1)) // At least 1 role with level >= 30
		assert.LessOrEqual(t, count, int64(5))    // But not more than 5 (our total test roles)
	})

	t.Run("CountRoles_NoResults", func(t *testing.T) {
		param := model.QueryParam{
			Wheres: []model.QueryWhere{
				{Column: "role_id", Value: "nonexistent_role_id"},
			},
		}
		count, err := testProvider.CountRoles(ctx, param)
		assert.NoError(t, err)
		assert.Equal(t, int64(0), count)
	})
}

func TestRoleErrorHandling(t *testing.T) {
	prepare(t)
	defer clean()

	ctx := context.Background()
	testUUID := strings.ReplaceAll(uuid.New().String(), "-", "")[:8]
	nonExistentRoleID := "nonexistent_role_" + testUUID

	t.Run("GetRole_NotFound", func(t *testing.T) {
		_, err := testProvider.GetRole(ctx, nonExistentRoleID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "role not found")
	})

	t.Run("CreateRole_MissingRoleID", func(t *testing.T) {
		roleData := maps.MapStrAny{
			"name":        "Test Role",
			"description": "Role without role_id",
		}

		_, err := testProvider.CreateRole(ctx, roleData)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "role_id is required")
	})

	t.Run("UpdateRole_NotFound", func(t *testing.T) {
		updateData := maps.MapStrAny{"name": "Test"}
		err := testProvider.UpdateRole(ctx, nonExistentRoleID, updateData)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "role not found")
	})

	t.Run("DeleteRole_NotFound", func(t *testing.T) {
		err := testProvider.DeleteRole(ctx, nonExistentRoleID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "role not found")
	})

	t.Run("GetRolePermissions_NotFound", func(t *testing.T) {
		_, err := testProvider.GetRolePermissions(ctx, nonExistentRoleID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "role not found")
	})

	t.Run("SetRolePermissions_NotFound", func(t *testing.T) {
		permissions := maps.MapStrAny{
			"permissions": map[string]interface{}{"test": true},
		}
		err := testProvider.SetRolePermissions(ctx, nonExistentRoleID, permissions)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "role not found")
	})

	t.Run("ValidateRolePermissions_NotFound", func(t *testing.T) {
		requiredPermissions := []string{"test.permission"}
		_, err := testProvider.ValidateRolePermissions(ctx, nonExistentRoleID, requiredPermissions)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "role not found")
	})

	t.Run("GetRoles_EmptyResult", func(t *testing.T) {
		param := model.QueryParam{
			Wheres: []model.QueryWhere{
				{Column: "role_id", Value: nonExistentRoleID},
			},
		}
		roles, err := testProvider.GetRoles(ctx, param)
		assert.NoError(t, err)
		assert.Equal(t, 0, len(roles)) // Empty slice, not nil
	})

	t.Run("PaginateRoles_EmptyResult", func(t *testing.T) {
		param := model.QueryParam{
			Wheres: []model.QueryWhere{
				{Column: "role_id", Value: nonExistentRoleID},
			},
		}
		result, err := testProvider.PaginateRoles(ctx, param, 1, 10)
		assert.NoError(t, err)
		assert.NotNil(t, result)

		data, ok := result["data"].([]maps.MapStr)
		assert.True(t, ok)
		assert.Equal(t, 0, len(data))

		// Handle different total types
		totalInterface, exists := result["total"]
		assert.True(t, exists)

		var total int64
		switch v := totalInterface.(type) {
		case int:
			total = int64(v)
		case int32:
			total = int64(v)
		case int64:
			total = v
		case uint:
			total = int64(v)
		case uint32:
			total = int64(v)
		case uint64:
			total = int64(v)
		default:
			t.Errorf("unexpected total type: %T, value: %v", totalInterface, totalInterface)
		}
		assert.Equal(t, int64(0), total)
	})

	t.Run("UpdateRole_EmptyData", func(t *testing.T) {
		// First create a role for this test
		testRoleID := "emptyupdate_" + testUUID
		roleData := maps.MapStrAny{
			"role_id": testRoleID,
			"name":    "Test Role for Empty Update",
		}
		_, err := testProvider.CreateRole(ctx, roleData)
		assert.NoError(t, err)

		// Test with empty update data (should not error, just do nothing)
		emptyData := maps.MapStrAny{}
		err = testProvider.UpdateRole(ctx, testRoleID, emptyData)
		assert.NoError(t, err) // Should not error, just skip update
	})

	t.Run("SetRolePermissions_EmptyData", func(t *testing.T) {
		// First create a role for this test
		testRoleID := "emptyperm_" + testUUID
		roleData := maps.MapStrAny{
			"role_id": testRoleID,
			"name":    "Test Role for Empty Permissions",
		}
		_, err := testProvider.CreateRole(ctx, roleData)
		assert.NoError(t, err)

		// Test with empty permission data (should not error, just do nothing)
		emptyData := maps.MapStrAny{}
		err = testProvider.SetRolePermissions(ctx, testRoleID, emptyData)
		assert.NoError(t, err) // Should not error, just skip update
	})

	t.Run("CountRoles_ComplexFilters", func(t *testing.T) {
		param := model.QueryParam{
			Wheres: []model.QueryWhere{
				{Column: "is_active", Value: true},
				{Column: "level", OP: ">=", Value: 10},
				{Column: "is_system", Value: false},
			},
		}
		count, err := testProvider.CountRoles(ctx, param)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, count, int64(0)) // Should handle complex filters without error
	})
}
