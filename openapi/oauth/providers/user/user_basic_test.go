package user_test

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/kun/maps"
)

func TestUserBasicOperations(t *testing.T) {
	prepare(t)
	defer clean()

	ctx := context.Background()

	// Use UUID to ensure unique identifiers
	testUUID := strings.ReplaceAll(uuid.New().String(), "-", "")[:8] // 8 char UUID

	// Create test user data dynamically
	testUser := &TestUserData{
		PreferredUsername: "testuser" + testUUID,
		Email:             "testuser" + testUUID + "@example.com",
		Password:          "TestPass123!",
		Name:              "Test User " + testUUID,
		GivenName:         "Test",
		FamilyName:        "User",
		Status:            "active",
		RoleID:            "user",
		TypeID:            "regular",
		EmailVerified:     true,
		Metadata:          map[string]interface{}{"source": "test"},
	}

	var testUserID string // Store the auto-generated user_id

	// Test CreateUser
	t.Run("CreateUser", func(t *testing.T) {
		userMap := maps.MapStrAny{
			"preferred_username": testUser.PreferredUsername,
			"email":              testUser.Email,
			"password":           testUser.Password,
			"name":               testUser.Name,
			"given_name":         testUser.GivenName,
			"family_name":        testUser.FamilyName,
			"status":             testUser.Status,
			"role_id":            testUser.RoleID,
			"type_id":            testUser.TypeID,
			"email_verified":     testUser.EmailVerified,
			"metadata":           testUser.Metadata,
		}

		id, err := testProvider.CreateUser(ctx, userMap)
		assert.NoError(t, err)
		assert.NotNil(t, id)

		// Verify user was created with auto-generated user_id
		assert.Contains(t, userMap, "user_id")
		assert.NotEmpty(t, userMap["user_id"])

		// Store generated user_id for subsequent tests
		testUserID = userMap["user_id"].(string)
	})

	// Test GetUser
	t.Run("GetUser", func(t *testing.T) {
		user, err := testProvider.GetUser(ctx, testUserID)
		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, testUser.PreferredUsername, user["preferred_username"])
		assert.Equal(t, testUser.Email, user["email"])
		assert.Equal(t, testUser.Name, user["name"])

		// Should not contain password_hash in public fields
		assert.NotContains(t, user, "password_hash")
	})

	// Test GetUserByPreferredUsername
	t.Run("GetUserByPreferredUsername", func(t *testing.T) {
		user, err := testProvider.GetUserByPreferredUsername(ctx, testUser.PreferredUsername)
		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, testUserID, user["user_id"])
		assert.Equal(t, testUser.Email, user["email"])
	})

	// Test GetUserByEmail
	t.Run("GetUserByEmail", func(t *testing.T) {
		user, err := testProvider.GetUserByEmail(ctx, testUser.Email)
		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, testUserID, user["user_id"])
		assert.Equal(t, testUser.PreferredUsername, user["preferred_username"])
	})

	// Test GetUserForAuth
	t.Run("GetUserForAuth", func(t *testing.T) {
		user, err := testProvider.GetUserForAuth(ctx, testUserID, "user_id")
		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, testUser.PreferredUsername, user["preferred_username"])

		// Should contain password_hash for auth
		assert.Contains(t, user, "password_hash")
		assert.NotEmpty(t, user["password_hash"])
	})

	// Test VerifyPassword
	t.Run("VerifyPassword", func(t *testing.T) {
		// Get user auth data first
		user, err := testProvider.GetUserForAuth(ctx, testUserID, "user_id")
		assert.NoError(t, err)

		passwordHash := user["password_hash"].(string)

		// Test correct password
		valid, err := testProvider.VerifyPassword(ctx, testUser.Password, passwordHash)
		assert.NoError(t, err)
		assert.True(t, valid)

		// Test incorrect password
		valid, err = testProvider.VerifyPassword(ctx, "wrongpassword", passwordHash)
		assert.NoError(t, err)
		assert.False(t, valid)

		// Test empty password hash
		valid, err = testProvider.VerifyPassword(ctx, testUser.Password, "")
		assert.Error(t, err)
		assert.False(t, valid)
		assert.Contains(t, err.Error(), "no password hash found")
	})

	// Test UpdateUser
	t.Run("UpdateUser", func(t *testing.T) {
		updateData := maps.MapStrAny{
			"name":        "Updated Test User",
			"given_name":  "Updated",
			"family_name": "User",
			"metadata":    map[string]interface{}{"updated": true},
		}

		err := testProvider.UpdateUser(ctx, testUserID, updateData)
		assert.NoError(t, err)

		// Verify update
		user, err := testProvider.GetUser(ctx, testUserID)
		assert.NoError(t, err)
		assert.Equal(t, "Updated Test User", user["name"])
		assert.Equal(t, "Updated", user["given_name"])

		// Test updating sensitive fields (should be ignored)
		sensitiveData := maps.MapStrAny{
			"password":      "newpassword",
			"password_hash": "newhash",
			"mfa_secret":    "newsecret",
		}

		err = testProvider.UpdateUser(ctx, testUserID, sensitiveData)
		assert.NoError(t, err) // Should not error, just ignore sensitive fields
	})

	// Test UpdatePassword
	t.Run("UpdatePassword", func(t *testing.T) {
		newPassword := "NewTestPass789!"

		err := testProvider.UpdatePassword(ctx, testUserID, newPassword)
		assert.NoError(t, err)

		// Verify password was updated
		user, err := testProvider.GetUserForAuth(ctx, testUserID, "user_id")
		assert.NoError(t, err)

		passwordHash := user["password_hash"].(string)
		valid, err := testProvider.VerifyPassword(ctx, newPassword, passwordHash)
		assert.NoError(t, err)
		assert.True(t, valid)

		// Old password should not work
		valid, err = testProvider.VerifyPassword(ctx, testUser.Password, passwordHash)
		assert.NoError(t, err)
		assert.False(t, valid)
	})

	// Test ResetPassword
	t.Run("ResetPassword", func(t *testing.T) {
		randomPassword, err := testProvider.ResetPassword(ctx, testUserID)
		assert.NoError(t, err)
		assert.NotEmpty(t, randomPassword)
		assert.Len(t, randomPassword, 12) // Should be 12 characters

		// Verify random password works
		user, err := testProvider.GetUserForAuth(ctx, testUserID, "user_id")
		assert.NoError(t, err)

		passwordHash := user["password_hash"].(string)
		valid, err := testProvider.VerifyPassword(ctx, randomPassword, passwordHash)
		assert.NoError(t, err)
		assert.True(t, valid)
	})

	// Test UpdateUserLastLogin
	t.Run("UpdateUserLastLogin", func(t *testing.T) {
		err := testProvider.UpdateUserLastLogin(ctx, testUserID, "127.0.0.1")
		assert.NoError(t, err)

		// Verify last_login_at was updated
		user, err := testProvider.GetUser(ctx, testUserID)
		assert.NoError(t, err)
		assert.NotNil(t, user["last_login_at"])
	})

	// Test UpdateUserStatus
	t.Run("UpdateUserStatus", func(t *testing.T) {
		err := testProvider.UpdateUserStatus(ctx, testUserID, "suspended")
		assert.NoError(t, err)

		// Verify status was updated
		user, err := testProvider.GetUser(ctx, testUserID)
		assert.NoError(t, err)
		assert.Equal(t, "suspended", user["status"])
	})

	// Test DeleteUser (at the end)
	t.Run("DeleteUser", func(t *testing.T) {
		err := testProvider.DeleteUser(ctx, testUserID)
		assert.NoError(t, err)

		// Verify user was deleted
		_, err = testProvider.GetUser(ctx, testUserID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})
}

func TestUserErrorHandling(t *testing.T) {
	prepare(t)
	defer clean()

	ctx := context.Background()
	// Use UUID to avoid conflicts
	testUUID := strings.ReplaceAll(uuid.New().String(), "-", "")[:8]
	nonExistentUserID := "non-existent-user-id-" + testUUID

	t.Run("GetUser_NotFound", func(t *testing.T) {
		_, err := testProvider.GetUser(ctx, nonExistentUserID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})

	t.Run("GetUserByPreferredUsername_NotFound", func(t *testing.T) {
		_, err := testProvider.GetUserByPreferredUsername(ctx, "nonexistent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})

	t.Run("GetUserByEmail_NotFound", func(t *testing.T) {
		_, err := testProvider.GetUserByEmail(ctx, "nonexistent@example.com")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})

	t.Run("GetUserForAuth_InvalidIdentifierType", func(t *testing.T) {
		_, err := testProvider.GetUserForAuth(ctx, "test", "invalid_type")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid identifier type")
	})

	t.Run("UpdateUser_NotFound", func(t *testing.T) {
		updateData := maps.MapStrAny{"name": "Test"}
		err := testProvider.UpdateUser(ctx, nonExistentUserID, updateData)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})

	t.Run("UpdatePassword_NotFound", func(t *testing.T) {
		err := testProvider.UpdatePassword(ctx, nonExistentUserID, "newpassword")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})

	t.Run("ResetPassword_NotFound", func(t *testing.T) {
		_, err := testProvider.ResetPassword(ctx, nonExistentUserID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})

	t.Run("DeleteUser_NotFound", func(t *testing.T) {
		err := testProvider.DeleteUser(ctx, nonExistentUserID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})
}

// NOTE: TestIDGeneration moved to utils_test.go (tests utils.go methods)
// NOTE: TestFieldListConfiguration moved to default_test.go (tests configuration, not basic operations)
