package user_test

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/kun/maps"
)

// TestExistsMethods tests all resource existence check methods
func TestExistsMethods(t *testing.T) {
	prepare(t)
	defer clean()

	ctx := context.Background()

	// Use UUID to ensure unique identifiers across test runs
	testUUID := strings.ReplaceAll(uuid.New().String(), "-", "")[:8] // 8 char UUID

	testUserID := "test-user-for-exists-" + testUUID
	testUsername := "testexistsuser" + testUUID
	testEmail := "testexists" + testUUID + "@example.com"

	t.Run("UserExists", func(t *testing.T) {
		// Test non-existent user
		exists, err := testProvider.UserExists(ctx, "nonexistent-user-id")
		assert.NoError(t, err)
		assert.False(t, exists)

		// Create a test user
		userData := maps.MapStrAny{
			"user_id":            testUserID,
			"preferred_username": testUsername,
			"email":              testEmail,
			"password":           "password123",
			"status":             "active",
		}

		_, err = testProvider.CreateUser(ctx, userData)
		assert.NoError(t, err)

		// Test existing user
		exists, err = testProvider.UserExists(ctx, testUserID)
		assert.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("UserExistsByEmail", func(t *testing.T) {
		// Test non-existent email
		exists, err := testProvider.UserExistsByEmail(ctx, "nonexistent@example.com")
		assert.NoError(t, err)
		assert.False(t, exists)

		// Test existing email (using unique email from test setup)
		exists, err = testProvider.UserExistsByEmail(ctx, testEmail)
		assert.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("UserExistsByPreferredUsername", func(t *testing.T) {
		// Test non-existent username
		exists, err := testProvider.UserExistsByPreferredUsername(ctx, "nonexistentuser")
		assert.NoError(t, err)
		assert.False(t, exists)

		// Test existing username (using unique username from test setup)
		exists, err = testProvider.UserExistsByPreferredUsername(ctx, testUsername)
		assert.NoError(t, err)
		assert.True(t, exists)
	})

	// Define unique IDs for roles and types
	testRoleID := "test-role-for-exists-" + testUUID
	testTypeID := "test-type-for-exists-" + testUUID

	t.Run("RoleExists", func(t *testing.T) {
		// Test non-existent role
		exists, err := testProvider.RoleExists(ctx, "nonexistent-role")
		assert.NoError(t, err)
		assert.False(t, exists)

		// Create a test role
		roleData := maps.MapStrAny{
			"role_id":     testRoleID,
			"name":        "Test Exists Role " + testUUID,
			"description": "Role for testing exists method",
			"is_active":   true,
		}

		_, err = testProvider.CreateRole(ctx, roleData)
		assert.NoError(t, err)

		// Test existing role
		exists, err = testProvider.RoleExists(ctx, testRoleID)
		assert.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("TypeExists", func(t *testing.T) {
		// Test non-existent type
		exists, err := testProvider.TypeExists(ctx, "nonexistent-type")
		assert.NoError(t, err)
		assert.False(t, exists)

		// Create a test type
		typeData := maps.MapStrAny{
			"type_id":     testTypeID,
			"name":        "Test Exists Type " + testUUID,
			"description": "Type for testing exists method",
			"is_active":   true,
		}

		_, err = testProvider.CreateType(ctx, typeData)
		assert.NoError(t, err)

		// Test existing type
		exists, err = testProvider.TypeExists(ctx, testTypeID)
		assert.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("OAuthAccountExists", func(t *testing.T) {
		// Test non-existent OAuth account
		exists, err := testProvider.OAuthAccountExists(ctx, "nonexistent-provider", "nonexistent-subject")
		assert.NoError(t, err)
		assert.False(t, exists)

		// Create a test OAuth account (using unique identifiers)
		testOAuthProvider := "test-provider-" + testUUID
		testSubject := "test-subject-for-exists-" + testUUID
		oauthData := maps.MapStrAny{
			"provider":  testOAuthProvider,
			"sub":       testSubject,
			"name":      "Test OAuth User " + testUUID,
			"email":     "testoauth" + testUUID + "@example.com",
			"is_active": true,
		}

		_, err = testProvider.CreateOAuthAccount(ctx, testUserID, oauthData)
		assert.NoError(t, err)

		// Test existing OAuth account
		exists, err = testProvider.OAuthAccountExists(ctx, testOAuthProvider, testSubject)
		assert.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("UserHasRole", func(t *testing.T) {
		// Test user without role
		hasRole, err := testProvider.UserHasRole(ctx, testUserID)
		assert.NoError(t, err)
		assert.False(t, hasRole)

		// Assign role to user
		err = testProvider.SetUserRole(ctx, testUserID, testRoleID)
		assert.NoError(t, err)

		// Test user with role
		hasRole, err = testProvider.UserHasRole(ctx, testUserID)
		assert.NoError(t, err)
		assert.True(t, hasRole)

		// Test non-existent user
		_, err = testProvider.UserHasRole(ctx, "nonexistent-user")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})

	t.Run("UserHasType", func(t *testing.T) {
		// Test user without type
		hasType, err := testProvider.UserHasType(ctx, testUserID)
		assert.NoError(t, err)
		assert.False(t, hasType)

		// Assign type to user
		err = testProvider.SetUserType(ctx, testUserID, testTypeID)
		assert.NoError(t, err)

		// Test user with type
		hasType, err = testProvider.UserHasType(ctx, testUserID)
		assert.NoError(t, err)
		assert.True(t, hasType)

		// Test non-existent user
		_, err = testProvider.UserHasType(ctx, "nonexistent-user")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})
}

// TestExistsPerformance tests the performance benefit of Exists methods vs full Get methods
func TestExistsPerformance(t *testing.T) {
	prepare(t)
	defer clean()

	ctx := context.Background()

	// Use UUID to ensure unique identifiers
	perfUUID := strings.ReplaceAll(uuid.New().String(), "-", "")[:8] // 8 char UUID
	perfUserID := "perf-test-user-" + perfUUID
	perfUsername := "perfuser" + perfUUID
	perfEmail := "perf" + perfUUID + "@example.com"

	// Create a test user for performance comparison
	userData := maps.MapStrAny{
		"user_id":            perfUserID,
		"preferred_username": perfUsername,
		"email":              perfEmail,
		"password":           "password123",
		"status":             "active",
	}

	_, err := testProvider.CreateUser(ctx, userData)
	assert.NoError(t, err)

	t.Run("UserExists_vs_GetUser", func(t *testing.T) {
		// Both should work, but UserExists should be more efficient
		// (we can't easily measure performance in unit tests, but we verify functionality)

		// Test UserExists
		exists, err := testProvider.UserExists(ctx, perfUserID)
		assert.NoError(t, err)
		assert.True(t, exists)

		// Test GetUser (more expensive)
		user, err := testProvider.GetUser(ctx, perfUserID)
		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, perfUserID, user["user_id"])

		// Both methods should give consistent results for existence
		exists, err = testProvider.UserExists(ctx, "nonexistent-user")
		assert.NoError(t, err)
		assert.False(t, exists)

		_, err = testProvider.GetUser(ctx, "nonexistent-user")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})
}
