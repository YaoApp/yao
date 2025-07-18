package oauth

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// UserInfo Tests
// =============================================================================

func TestUserInfo(t *testing.T) {
	service, _, _, cleanup := setupOAuthTestEnvironment(t)
	defer cleanup()

	ctx := context.Background()
	userProvider := service.GetUserProvider()

	t.Run("get user info with valid access token", func(t *testing.T) {
		// Create a valid access token for the first test user
		testUser := testUsers[0]
		accessToken := "valid_access_token_123"

		// Store token data in user provider
		tokenData := map[string]interface{}{
			"token":      accessToken,
			"user_id":    testUser.ID,
			"subject":    testUser.Subject,
			"username":   testUser.Username,
			"email":      testUser.Email,
			"first_name": testUser.FirstName,
			"last_name":  testUser.LastName,
			"full_name":  testUser.FullName,
			"scopes":     testUser.Scopes,
			"status":     testUser.Status,
			"exp":        time.Now().Add(time.Hour).Unix(),
			"iat":        time.Now().Unix(),
			"token_type": "Bearer",
		}

		// Store the token data
		err := userProvider.StoreToken(accessToken, tokenData, time.Hour)
		require.NoError(t, err)

		// Get user info using the access token
		userInfo, err := service.UserInfo(ctx, accessToken)
		assert.NoError(t, err)
		assert.NotNil(t, userInfo)

		// Verify the user info contains expected data
		if userInfoMap, ok := userInfo.(map[string]interface{}); ok {
			assert.Equal(t, testUser.Subject, userInfoMap["subject"])
			assert.Equal(t, testUser.Username, userInfoMap["username"])
			assert.Equal(t, testUser.Email, userInfoMap["email"])
		}
	})

	t.Run("get user info with invalid access token", func(t *testing.T) {
		invalidToken := "invalid_access_token_xyz"

		userInfo, err := service.UserInfo(ctx, invalidToken)
		assert.Error(t, err)
		assert.Nil(t, userInfo)
	})

	t.Run("get user info with non-existent access token", func(t *testing.T) {
		nonExistentToken := "non_existent_token_abc"

		userInfo, err := service.UserInfo(ctx, nonExistentToken)
		assert.Error(t, err)
		assert.Nil(t, userInfo)
	})

	t.Run("get user info with empty access token", func(t *testing.T) {
		emptyToken := ""

		userInfo, err := service.UserInfo(ctx, emptyToken)
		assert.Error(t, err)
		assert.Nil(t, userInfo)
	})

	t.Run("get user info with expired access token", func(t *testing.T) {
		testUser := testUsers[1]
		expiredToken := "expired_access_token_456"

		// Store expired token data
		tokenData := map[string]interface{}{
			"token":      expiredToken,
			"user_id":    testUser.ID,
			"subject":    testUser.Subject,
			"username":   testUser.Username,
			"email":      testUser.Email,
			"scopes":     testUser.Scopes,
			"status":     testUser.Status,
			"exp":        time.Now().Add(-time.Hour).Unix(), // Expired 1 hour ago
			"iat":        time.Now().Add(-2 * time.Hour).Unix(),
			"token_type": "Bearer",
		}

		err := userProvider.StoreToken(expiredToken, tokenData, time.Hour)
		require.NoError(t, err)

		userInfo, err := service.UserInfo(ctx, expiredToken)
		// UserInfo method returns user data regardless of token expiry
		assert.NoError(t, err)
		assert.NotNil(t, userInfo)

		// Verify user info contains expected data
		if userInfoMap, ok := userInfo.(map[string]interface{}); ok {
			assert.Equal(t, testUser.Subject, userInfoMap["subject"])
			assert.Equal(t, testUser.Username, userInfoMap["username"])
		}
	})

	t.Run("get user info with inactive user", func(t *testing.T) {
		// Use the inactive test user
		inactiveUser := testUsers[4] // inactive.user
		inactiveToken := "inactive_user_token_789"

		tokenData := map[string]interface{}{
			"token":      inactiveToken,
			"user_id":    inactiveUser.ID,
			"subject":    inactiveUser.Subject,
			"username":   inactiveUser.Username,
			"email":      inactiveUser.Email,
			"scopes":     inactiveUser.Scopes,
			"status":     inactiveUser.Status, // inactive
			"exp":        time.Now().Add(time.Hour).Unix(),
			"iat":        time.Now().Unix(),
			"token_type": "Bearer",
		}

		err := userProvider.StoreToken(inactiveToken, tokenData, time.Hour)
		require.NoError(t, err)

		userInfo, err := service.UserInfo(ctx, inactiveToken)
		// UserInfo method returns user data regardless of user status
		assert.NoError(t, err)
		assert.NotNil(t, userInfo)

		// Verify user info contains expected data
		if userInfoMap, ok := userInfo.(map[string]interface{}); ok {
			assert.Equal(t, inactiveUser.Subject, userInfoMap["subject"])
			assert.Equal(t, inactiveUser.Username, userInfoMap["username"])
			assert.Equal(t, inactiveUser.Status, userInfoMap["status"])
		}
	})

	t.Run("get user info with limited scope user", func(t *testing.T) {
		// Use the limited scope test user
		limitedUser := testUsers[5] // limited.user
		limitedToken := "limited_scope_token_101"

		tokenData := map[string]interface{}{
			"token":      limitedToken,
			"user_id":    limitedUser.ID,
			"subject":    limitedUser.Subject,
			"username":   limitedUser.Username,
			"email":      limitedUser.Email,
			"scopes":     limitedUser.Scopes, // Only openid
			"status":     limitedUser.Status,
			"exp":        time.Now().Add(time.Hour).Unix(),
			"iat":        time.Now().Unix(),
			"token_type": "Bearer",
		}

		err := userProvider.StoreToken(limitedToken, tokenData, time.Hour)
		require.NoError(t, err)

		userInfo, err := service.UserInfo(ctx, limitedToken)
		assert.NoError(t, err)
		assert.NotNil(t, userInfo)

		// Verify limited user info
		if userInfoMap, ok := userInfo.(map[string]interface{}); ok {
			assert.Equal(t, limitedUser.Subject, userInfoMap["subject"])
			assert.Equal(t, limitedUser.Username, userInfoMap["username"])
			// Should only have basic scopes
			if scopes, ok := userInfoMap["scopes"].([]string); ok {
				assert.Contains(t, scopes, "openid")
				assert.Len(t, scopes, 1)
			}
		}
	})

	t.Run("get user info with admin user", func(t *testing.T) {
		// Use the admin test user
		adminUser := testUsers[0] // admin
		adminToken := "admin_token_202"

		tokenData := map[string]interface{}{
			"token":              adminToken,
			"user_id":            adminUser.ID,
			"subject":            adminUser.Subject,
			"username":           adminUser.Username,
			"email":              adminUser.Email,
			"first_name":         adminUser.FirstName,
			"last_name":          adminUser.LastName,
			"full_name":          adminUser.FullName,
			"scopes":             adminUser.Scopes,
			"status":             adminUser.Status,
			"email_verified":     adminUser.EmailVerified,
			"mobile_verified":    adminUser.MobileVerified,
			"two_factor_enabled": adminUser.TwoFactorEnabled,
			"exp":                time.Now().Add(time.Hour).Unix(),
			"iat":                time.Now().Unix(),
			"token_type":         "Bearer",
		}

		err := userProvider.StoreToken(adminToken, tokenData, time.Hour)
		require.NoError(t, err)

		userInfo, err := service.UserInfo(ctx, adminToken)
		assert.NoError(t, err)
		assert.NotNil(t, userInfo)

		// Verify admin user info
		if userInfoMap, ok := userInfo.(map[string]interface{}); ok {
			assert.Equal(t, adminUser.Subject, userInfoMap["subject"])
			assert.Equal(t, adminUser.Username, userInfoMap["username"])
			assert.Equal(t, adminUser.Email, userInfoMap["email"])
			assert.True(t, userInfoMap["email_verified"].(bool))
			assert.True(t, userInfoMap["two_factor_enabled"].(bool))

			// Should have admin scopes
			if scopes, ok := userInfoMap["scopes"].([]string); ok {
				assert.Contains(t, scopes, "admin")
				assert.Contains(t, scopes, "openid")
				assert.Contains(t, scopes, "profile")
				assert.Contains(t, scopes, "email")
			}
		}
	})
}

// =============================================================================
// Integration Tests
// =============================================================================

func TestUserInfoIntegration(t *testing.T) {
	service, _, _, cleanup := setupOAuthTestEnvironment(t)
	defer cleanup()

	ctx := context.Background()
	userProvider := service.GetUserProvider()

	t.Run("complete user info flow", func(t *testing.T) {
		// Use different test users for comprehensive testing
		testCases := []struct {
			name        string
			user        *TestUser
			tokenSuffix string
		}{
			{"regular_user", testUsers[1], "regular"},
			{"verified_user", testUsers[2], "verified"},
			{"secure_user", testUsers[6], "secure"},
			{"api_user", testUsers[7], "api"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				token := "integration_token_" + tc.tokenSuffix

				tokenData := map[string]interface{}{
					"token":      token,
					"user_id":    tc.user.ID,
					"subject":    tc.user.Subject,
					"username":   tc.user.Username,
					"email":      tc.user.Email,
					"scopes":     tc.user.Scopes,
					"status":     tc.user.Status,
					"exp":        time.Now().Add(time.Hour).Unix(),
					"iat":        time.Now().Unix(),
					"token_type": "Bearer",
				}

				err := userProvider.StoreToken(token, tokenData, time.Hour)
				require.NoError(t, err)

				userInfo, err := service.UserInfo(ctx, token)
				assert.NoError(t, err)
				assert.NotNil(t, userInfo)

				// Verify basic user info structure
				if userInfoMap, ok := userInfo.(map[string]interface{}); ok {
					assert.Equal(t, tc.user.Subject, userInfoMap["subject"])
					assert.Equal(t, tc.user.Username, userInfoMap["username"])
					assert.Equal(t, tc.user.Email, userInfoMap["email"])
					assert.Equal(t, tc.user.Status, userInfoMap["status"])
				}
			})
		}
	})

	t.Run("concurrent user info requests", func(t *testing.T) {
		// Test concurrent access to user info
		const numRequests = 10

		// Create tokens for concurrent testing
		tokens := make([]string, numRequests)
		for i := 0; i < numRequests; i++ {
			tokens[i] = fmt.Sprintf("concurrent_token_%d", i)
			testUser := testUsers[i%len(testUsers)]

			tokenData := map[string]interface{}{
				"token":      tokens[i],
				"user_id":    testUser.ID,
				"subject":    testUser.Subject,
				"username":   testUser.Username,
				"email":      testUser.Email,
				"scopes":     testUser.Scopes,
				"status":     testUser.Status,
				"exp":        time.Now().Add(time.Hour).Unix(),
				"iat":        time.Now().Unix(),
				"token_type": "Bearer",
			}

			err := userProvider.StoreToken(tokens[i], tokenData, time.Hour)
			require.NoError(t, err)
		}

		// Make concurrent requests
		results := make(chan error, numRequests)
		for i := 0; i < numRequests; i++ {
			go func(token string) {
				userInfo, err := service.UserInfo(ctx, token)
				if err != nil {
					results <- err
					return
				}
				if userInfo == nil {
					results <- fmt.Errorf("user info is nil")
					return
				}
				results <- nil
			}(tokens[i])
		}

		// Collect results
		for i := 0; i < numRequests; i++ {
			err := <-results
			assert.NoError(t, err)
		}
	})
}

// =============================================================================
// Edge Cases and Error Handling
// =============================================================================

func TestUserInfoEdgeCases(t *testing.T) {
	service, _, _, cleanup := setupOAuthTestEnvironment(t)
	defer cleanup()

	ctx := context.Background()
	userProvider := service.GetUserProvider()

	t.Run("malformed token data", func(t *testing.T) {
		malformedToken := "malformed_token_data"

		// Store malformed token data
		tokenData := map[string]interface{}{
			"token":      malformedToken,
			"user_id":    "invalid_user_id",
			"subject":    nil,            // Invalid subject
			"username":   "",             // Empty username
			"exp":        "not_a_number", // Invalid expiration
			"iat":        time.Now().Unix(),
			"token_type": "Bearer",
		}

		err := userProvider.StoreToken(malformedToken, tokenData, time.Hour)
		require.NoError(t, err)

		userInfo, err := service.UserInfo(ctx, malformedToken)
		assert.Error(t, err)
		assert.Nil(t, userInfo)
	})

	t.Run("very long access token", func(t *testing.T) {
		// Create a very long token
		longToken := "very_long_token_" + strings.Repeat("a", 1000)

		userInfo, err := service.UserInfo(ctx, longToken)
		assert.Error(t, err)
		assert.Nil(t, userInfo)
	})

	t.Run("special characters in token", func(t *testing.T) {
		specialToken := "special_token_!@#$%^&*()_+{}[]|\\:;\"'<>?,./`~"

		userInfo, err := service.UserInfo(ctx, specialToken)
		assert.Error(t, err)
		assert.Nil(t, userInfo)
	})

	t.Run("token with only whitespace", func(t *testing.T) {
		whitespaceToken := "   \t\n\r   "

		userInfo, err := service.UserInfo(ctx, whitespaceToken)
		assert.Error(t, err)
		assert.Nil(t, userInfo)
	})

	t.Run("token with minimal valid data", func(t *testing.T) {
		minimalToken := "minimal_token_999"
		testUser := testUsers[9] // test.user

		// Store minimal token data
		tokenData := map[string]interface{}{
			"token":      minimalToken,
			"user_id":    testUser.ID,
			"subject":    testUser.Subject,
			"username":   testUser.Username,
			"exp":        time.Now().Add(time.Hour).Unix(),
			"iat":        time.Now().Unix(),
			"token_type": "Bearer",
		}

		err := userProvider.StoreToken(minimalToken, tokenData, time.Hour)
		require.NoError(t, err)

		userInfo, err := service.UserInfo(ctx, minimalToken)
		assert.NoError(t, err)
		assert.NotNil(t, userInfo)

		// Verify minimal user info
		if userInfoMap, ok := userInfo.(map[string]interface{}); ok {
			assert.Equal(t, testUser.Subject, userInfoMap["subject"])
			assert.Equal(t, testUser.Username, userInfoMap["username"])
		}
	})
}
