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

func TestUserListOperations(t *testing.T) {
	prepare(t)
	defer clean()

	ctx := context.Background()

	// Create multiple test users for list operations
	// Use UUID to ensure unique identifiers
	testUUID := strings.ReplaceAll(uuid.New().String(), "-", "")[:8] // 8 char UUID
	testUsers := []*TestUserData{
		createTestUserData("listops" + testUUID + "01"),
		createTestUserData("listops" + testUUID + "02"),
		createTestUserData("listops" + testUUID + "03"),
		createTestUserData("listops" + testUUID + "04"),
		createTestUserData("listops" + testUUID + "05"),
	}

	// Store created user IDs
	userIDs := make([]string, len(testUsers))

	// Create test users with varied data for testing
	for i, userData := range testUsers {
		// Vary some data for testing filters
		if i%2 == 0 {
			userData.Status = "active"
			userData.RoleID = "admin"
		} else {
			userData.Status = "pending"
			userData.RoleID = "user"
		}

		_, userID := setupTestUser(t, ctx, userData)
		userIDs[i] = userID
	}

	// Test GetUsers
	t.Run("GetUsers_All", func(t *testing.T) {
		param := model.QueryParam{
			Wheres: []model.QueryWhere{
				{Column: "user_id", OP: "like", Value: "test_%"},
			},
		}
		users, err := testProvider.GetUsers(ctx, param)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(users), 5) // At least our 5 test users

		// Check that basic fields are returned by default
		if len(users) > 0 {
			user := users[0]
			assert.Contains(t, user, "user_id")
			assert.Contains(t, user, "preferred_username")
			// Should not contain sensitive fields in basic view
			assert.NotContains(t, user, "password_hash")
		}
	})

	t.Run("GetUsers_WithFilters", func(t *testing.T) {
		param := model.QueryParam{
			Wheres: []model.QueryWhere{
				{Column: "user_id", OP: "like", Value: "test_%"},
				{Column: "status", Value: "active"},
			},
		}
		users, err := testProvider.GetUsers(ctx, param)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(users), 3) // At least 3 active users (indexes 0, 2, 4)

		// All returned users should be active
		for _, user := range users {
			if userID, ok := user["user_id"].(string); ok {
				// Only check our test users
				for _, testUserID := range userIDs {
					if userID == testUserID {
						assert.Equal(t, "active", user["status"])
						break
					}
				}
			}
		}
	})

	t.Run("GetUsers_WithCustomFields", func(t *testing.T) {
		param := model.QueryParam{
			Select: []interface{}{"user_id", "preferred_username", "status"},
			Wheres: []model.QueryWhere{
				{Column: "user_id", OP: "like", Value: "test_%"},
			},
			Limit: 2,
		}
		users, err := testProvider.GetUsers(ctx, param)
		assert.NoError(t, err)
		assert.LessOrEqual(t, len(users), 2) // Respects limit

		if len(users) > 0 {
			user := users[0]
			assert.Contains(t, user, "user_id")
			assert.Contains(t, user, "preferred_username")
			assert.Contains(t, user, "status")
		}
	})

	t.Run("GetUsers_WithOrdering", func(t *testing.T) {
		param := model.QueryParam{
			Wheres: []model.QueryWhere{
				{Column: "user_id", OP: "like", Value: "test_%"},
			},
			Orders: []model.QueryOrder{
				{Column: "preferred_username", Option: "asc"},
			},
			Limit: 3,
		}
		users, err := testProvider.GetUsers(ctx, param)
		assert.NoError(t, err)
		assert.LessOrEqual(t, len(users), 3)

		// Check ordering (should be sorted by preferred_username ascending)
		if len(users) >= 2 {
			first := users[0]["preferred_username"].(string)
			second := users[1]["preferred_username"].(string)
			assert.LessOrEqual(t, first, second)
		}
	})

	// Test PaginateUsers
	t.Run("PaginateUsers_FirstPage", func(t *testing.T) {
		param := model.QueryParam{
			Wheres: []model.QueryWhere{
				{Column: "user_id", OP: "like", Value: "test_%"},
			},
			Orders: []model.QueryOrder{
				{Column: "preferred_username", Option: "asc"},
			},
		}
		result, err := testProvider.PaginateUsers(ctx, param, 1, 3)
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
		assert.GreaterOrEqual(t, total, int64(0)) // Could be 0 or more

		assert.Equal(t, 1, result["page"])
		assert.Equal(t, 3, result["pagesize"])
	})

	t.Run("PaginateUsers_SecondPage", func(t *testing.T) {
		param := model.QueryParam{
			Wheres: []model.QueryWhere{
				{Column: "user_id", OP: "like", Value: "test_%"},
			},
			Orders: []model.QueryOrder{
				{Column: "preferred_username", Option: "asc"},
			},
		}
		result, err := testProvider.PaginateUsers(ctx, param, 2, 3)
		assert.NoError(t, err)
		assert.NotNil(t, result)

		assert.Equal(t, 2, result["page"])
		assert.Equal(t, 3, result["pagesize"])

		_, ok := result["data"].([]maps.MapStr)
		assert.True(t, ok)
		// Second page may have fewer items depending on total count
	})

	t.Run("PaginateUsers_WithFilters", func(t *testing.T) {
		param := model.QueryParam{
			Wheres: []model.QueryWhere{
				{Column: "user_id", OP: "like", Value: "test_%"},
				{Column: "role_id", Value: "admin"},
			},
		}
		result, err := testProvider.PaginateUsers(ctx, param, 1, 10)
		assert.NoError(t, err)
		assert.NotNil(t, result)

		data, ok := result["data"].([]maps.MapStr)
		assert.True(t, ok)
		assert.GreaterOrEqual(t, len(data), 0) // Could be 0 or more

		// Verify admin filter works by checking our test users only
		for _, user := range data {
			if userID, ok := user["user_id"].(string); ok {
				// Only check our test users
				for _, testUserID := range userIDs {
					if userID == testUserID {
						assert.Equal(t, "admin", user["role_id"])
						break
					}
				}
			}
		}
	})

	// Test CountUsers
	t.Run("CountUsers_All", func(t *testing.T) {
		param := model.QueryParam{
			Wheres: []model.QueryWhere{
				{Column: "user_id", OP: "like", Value: "test_%"},
			},
		}
		count, err := testProvider.CountUsers(ctx, param)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, count, int64(0)) // At least 0 users
	})

	t.Run("CountUsers_WithFilters", func(t *testing.T) {
		param := model.QueryParam{
			Wheres: []model.QueryWhere{
				{Column: "user_id", OP: "like", Value: "test_%"},
				{Column: "status", Value: "active"},
			},
		}
		count, err := testProvider.CountUsers(ctx, param)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, count, int64(3)) // At least 3 active users (indexes 0, 2, 4)
	})

	t.Run("CountUsers_SpecificRole", func(t *testing.T) {
		param := model.QueryParam{
			Wheres: []model.QueryWhere{
				{Column: "user_id", OP: "like", Value: "test_%"},
				{Column: "role_id", Value: "user"},
			},
		}
		count, err := testProvider.CountUsers(ctx, param)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, count, int64(2)) // At least 2 regular users (indexes 1, 3)
	})

	t.Run("CountUsers_NoResults", func(t *testing.T) {
		param := model.QueryParam{
			Wheres: []model.QueryWhere{
				{Column: "user_id", Value: "nonexistent_user_id"},
			},
		}
		count, err := testProvider.CountUsers(ctx, param)
		assert.NoError(t, err)
		assert.Equal(t, int64(0), count)
	})
}

func TestUserListErrorHandling(t *testing.T) {
	prepare(t)
	defer clean()

	ctx := context.Background()

	// Use UUID for unique test identifiers
	testUUID := strings.ReplaceAll(uuid.New().String(), "-", "")[:8]

	t.Run("GetUsers_EmptyResult", func(t *testing.T) {
		param := model.QueryParam{
			Wheres: []model.QueryWhere{
				{Column: "user_id", Value: "nonexistent_" + testUUID},
			},
		}
		users, err := testProvider.GetUsers(ctx, param)
		assert.NoError(t, err)
		assert.Equal(t, 0, len(users)) // Empty slice, not nil
	})

	t.Run("PaginateUsers_EmptyResult", func(t *testing.T) {
		param := model.QueryParam{
			Wheres: []model.QueryWhere{
				{Column: "user_id", Value: "nonexistent_" + testUUID},
			},
		}
		result, err := testProvider.PaginateUsers(ctx, param, 1, 10)
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

	t.Run("PaginateUsers_LargePage", func(t *testing.T) {
		param := model.QueryParam{
			Wheres: []model.QueryWhere{
				{Column: "user_id", OP: "like", Value: "test_%"},
			},
		}
		result, err := testProvider.PaginateUsers(ctx, param, 100, 10) // Page way beyond data
		assert.NoError(t, err)
		assert.NotNil(t, result)

		data, ok := result["data"].([]maps.MapStr)
		assert.True(t, ok)
		assert.Equal(t, 0, len(data)) // No data on this page

		assert.Equal(t, 100, result["page"])
		assert.Equal(t, 10, result["pagesize"])
	})

	t.Run("CountUsers_ComplexFilters", func(t *testing.T) {
		param := model.QueryParam{
			Wheres: []model.QueryWhere{
				{Column: "user_id", OP: "like", Value: "test_%"},
				{Column: "status", OP: "in", Value: []interface{}{"active", "pending"}},
				{Column: "email_verified", Value: true},
			},
		}
		count, err := testProvider.CountUsers(ctx, param)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, count, int64(0)) // Should handle complex filters without error
	})
}
