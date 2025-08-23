package user_test

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/yao/openapi/oauth/providers/user"
)

func TestGenerateUserID(t *testing.T) {
	prepare(t)
	defer clean()

	ctx := context.Background()

	t.Run("NanoID_Strategy_Safe_Mode", func(t *testing.T) {
		// Test NanoID strategy with safe mode (default)
		provider := user.NewDefaultUser(&user.DefaultUserOptions{
			Prefix:     "test:",
			IDStrategy: user.NanoIDStrategy,
			IDPrefix:   "user_",
		})

		userID, err := provider.GenerateUserID(ctx, true)
		assert.NoError(t, err)
		assert.NotEmpty(t, userID)
		assert.True(t, strings.HasPrefix(userID, "user_"))
		assert.Greater(t, len(userID), 5) // Should be "user_" + at least some characters
	})

	t.Run("NanoID_Strategy_Unsafe_Mode", func(t *testing.T) {
		provider := user.NewDefaultUser(&user.DefaultUserOptions{
			Prefix:     "test:",
			IDStrategy: user.NanoIDStrategy,
			IDPrefix:   "test_",
		})

		userID, err := provider.GenerateUserID(ctx, false)
		assert.NoError(t, err)
		assert.NotEmpty(t, userID)
		assert.True(t, strings.HasPrefix(userID, "test_"))
	})

	t.Run("UUID_Strategy_Safe_Mode", func(t *testing.T) {
		provider := user.NewDefaultUser(&user.DefaultUserOptions{
			Prefix:     "test:",
			IDStrategy: user.UUIDStrategy,
			IDPrefix:   "uuid_",
		})

		userID, err := provider.GenerateUserID(ctx, true)
		assert.NoError(t, err)
		assert.NotEmpty(t, userID)
		assert.True(t, strings.HasPrefix(userID, "uuid_"))
		// UUID format: xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx (36 chars) + prefix
		assert.Greater(t, len(userID), 40)
	})

	t.Run("UUID_Strategy_Unsafe_Mode", func(t *testing.T) {
		provider := user.NewDefaultUser(&user.DefaultUserOptions{
			Prefix:     "test:",
			IDStrategy: user.UUIDStrategy,
			IDPrefix:   "",
		})

		userID, err := provider.GenerateUserID(ctx, false)
		assert.NoError(t, err)
		assert.NotEmpty(t, userID)
		assert.Len(t, userID, 36) // Standard UUID length without prefix
	})

	t.Run("Default_Safe_Mode_Behavior", func(t *testing.T) {
		// NanoID should default to safe mode
		providerNano := user.NewDefaultUser(&user.DefaultUserOptions{
			Prefix:     "test:",
			IDStrategy: user.NanoIDStrategy,
		})

		userID, err := providerNano.GenerateUserID(ctx)
		assert.NoError(t, err)
		assert.NotEmpty(t, userID)

		// UUID should default to unsafe mode
		providerUUID := user.NewDefaultUser(&user.DefaultUserOptions{
			Prefix:     "test:",
			IDStrategy: user.UUIDStrategy,
		})

		userID2, err := providerUUID.GenerateUserID(ctx)
		assert.NoError(t, err)
		assert.NotEmpty(t, userID2)
		assert.Len(t, userID2, 36) // Standard UUID length
	})

	t.Run("Collision_Detection", func(t *testing.T) {
		provider := user.NewDefaultUser(&user.DefaultUserOptions{
			Prefix:     "test:",
			IDStrategy: user.NanoIDStrategy,
			IDPrefix:   "collision_",
		})

		// Generate first ID and create user with it
		userID1, err := provider.GenerateUserID(ctx, true)
		require.NoError(t, err)

		userData := maps.MapStrAny{
			"user_id":            userID1,
			"preferred_username": "collisiontest",
			"email":              "collision@test.com",
			"password":           "TestPass123!",
			"status":             "active",
		}
		_, err = provider.CreateUser(ctx, userData)
		require.NoError(t, err)

		// Generate second ID - should be different due to collision detection
		userID2, err := provider.GenerateUserID(ctx, true)
		assert.NoError(t, err)
		assert.NotEqual(t, userID1, userID2)
		assert.True(t, strings.HasPrefix(userID2, "collision_"))

		// Clean up
		provider.DeleteUser(ctx, userID1)
	})

	t.Run("Multiple_IDs_Uniqueness", func(t *testing.T) {
		provider := user.NewDefaultUser(&user.DefaultUserOptions{
			Prefix:     "test:",
			IDStrategy: user.NanoIDStrategy,
		})

		ids := make(map[string]bool)
		for i := 0; i < 100; i++ {
			userID, err := provider.GenerateUserID(ctx, false)
			assert.NoError(t, err)
			assert.NotEmpty(t, userID)

			// Check uniqueness
			assert.False(t, ids[userID], "Generated duplicate ID: %s", userID)
			ids[userID] = true
		}
	})
}

// TODO: TestGetOAuthUserID - depends on CreateOAuthAccount implementation
// func TestGetOAuthUserID(t *testing.T) {
//     // Will be implemented after CreateOAuthAccount is implemented
// }

func TestNanoIDGeneration(t *testing.T) {
	t.Run("NanoID_Length_And_Characters", func(t *testing.T) {
		provider := user.NewDefaultUser(&user.DefaultUserOptions{
			Prefix:     "test:",
			IDStrategy: user.NanoIDStrategy,
		})

		// Generate multiple NanoIDs and check their properties
		for i := 0; i < 10; i++ {
			userID, err := provider.GenerateUserID(context.Background(), false)
			assert.NoError(t, err)
			assert.NotEmpty(t, userID)

			// NanoID should be 12 characters (default length)
			assert.Len(t, userID, 12)

			// Should only contain allowed characters (no ambiguous chars like 0, O, 1, l, I)
			allowedChars := "23456789ABCDEFGHJKMNPQRSTUVWXYZabcdefghijkmnpqrstuvwxyz"
			for _, char := range userID {
				assert.Contains(t, allowedChars, string(char), "Invalid character in NanoID: %c", char)
			}

			// Should not contain ambiguous characters
			forbiddenChars := "01OIl"
			for _, char := range forbiddenChars {
				assert.NotContains(t, userID, string(char), "NanoID contains ambiguous character: %c", char)
			}
		}
	})

	t.Run("NanoID_With_Prefix", func(t *testing.T) {
		provider := user.NewDefaultUser(&user.DefaultUserOptions{
			Prefix:     "test:",
			IDStrategy: user.NanoIDStrategy,
			IDPrefix:   "nano_",
		})

		userID, err := provider.GenerateUserID(context.Background(), false)
		assert.NoError(t, err)
		assert.True(t, strings.HasPrefix(userID, "nano_"))
		assert.Len(t, userID, 17) // "nano_" (5) + 12 chars
	})
}

func TestUUIDGeneration(t *testing.T) {
	t.Run("UUID_Format", func(t *testing.T) {
		provider := user.NewDefaultUser(&user.DefaultUserOptions{
			Prefix:     "test:",
			IDStrategy: user.UUIDStrategy,
		})

		// Generate multiple UUIDs and check their format
		for i := 0; i < 10; i++ {
			userID, err := provider.GenerateUserID(context.Background(), false)
			assert.NoError(t, err)
			assert.NotEmpty(t, userID)

			// UUID format: xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx
			assert.Len(t, userID, 36)
			assert.Equal(t, byte('-'), userID[8])
			assert.Equal(t, byte('-'), userID[13])
			assert.Equal(t, byte('-'), userID[18])
			assert.Equal(t, byte('-'), userID[23])

			// Should be version 4 UUID (14th character should be '4')
			assert.Equal(t, byte('4'), userID[14])

			// 19th character should be one of '8', '9', 'a', 'b' (variant bits)
			variant := userID[19]
			assert.True(t, variant == '8' || variant == '9' || variant == 'a' || variant == 'b',
				"Invalid UUID variant: %c", variant)
		}
	})

	t.Run("UUID_With_Prefix", func(t *testing.T) {
		provider := user.NewDefaultUser(&user.DefaultUserOptions{
			Prefix:     "test:",
			IDStrategy: user.UUIDStrategy,
			IDPrefix:   "uuid_",
		})

		userID, err := provider.GenerateUserID(context.Background(), false)
		assert.NoError(t, err)
		assert.True(t, strings.HasPrefix(userID, "uuid_"))
		assert.Len(t, userID, 41) // "uuid_" (5) + 36 chars
	})
}

func TestRandomPasswordGeneration(t *testing.T) {
	prepare(t)
	defer clean()

	ctx := context.Background()

	t.Run("ResetPassword_Generates_Random_Password", func(t *testing.T) {
		// Create test user
		testUserData := createTestUserData("password")
		_, testUserID := setupTestUser(t, ctx, testUserData)

		// Reset password should generate a random 12-character password
		randomPassword, err := testProvider.ResetPassword(ctx, testUserID)
		assert.NoError(t, err)
		assert.NotEmpty(t, randomPassword)
		assert.Len(t, randomPassword, 12)

		// Password should contain mix of characters
		hasUpper := false
		hasLower := false
		hasDigit := false
		hasSpecial := false

		for _, char := range randomPassword {
			switch {
			case char >= 'A' && char <= 'Z':
				hasUpper = true
			case char >= 'a' && char <= 'z':
				hasLower = true
			case char >= '0' && char <= '9':
				hasDigit = true
			case strings.ContainsRune("!@#$%^&*", char):
				hasSpecial = true
			}
		}

		// Should have at least some variety (not enforcing all types for 12 chars)
		varietyCount := 0
		if hasUpper {
			varietyCount++
		}
		if hasLower {
			varietyCount++
		}
		if hasDigit {
			varietyCount++
		}
		if hasSpecial {
			varietyCount++
		}

		assert.Greater(t, varietyCount, 1, "Password should have variety in character types")

		// Clean up
		testProvider.DeleteUser(ctx, testUserID)
	})

	t.Run("Multiple_Random_Passwords_Are_Different", func(t *testing.T) {
		// Create test user
		testUserData := createTestUserData("multipass")
		_, testUserID := setupTestUser(t, ctx, testUserData)

		passwords := make(map[string]bool)
		for i := 0; i < 10; i++ {
			randomPassword, err := testProvider.ResetPassword(ctx, testUserID)
			assert.NoError(t, err)
			assert.NotEmpty(t, randomPassword)

			// Check uniqueness
			assert.False(t, passwords[randomPassword], "Generated duplicate password: %s", randomPassword)
			passwords[randomPassword] = true
		}

		// Clean up
		testProvider.DeleteUser(ctx, testUserID)
	})
}

func TestIDStrategyConfiguration(t *testing.T) {
	t.Run("Default_Strategy_Is_NanoID", func(t *testing.T) {
		provider := user.NewDefaultUser(&user.DefaultUserOptions{
			Prefix: "test:",
			// No IDStrategy specified, should default to NanoID
		})

		userID, err := provider.GenerateUserID(context.Background(), false)
		assert.NoError(t, err)
		assert.NotEmpty(t, userID)
		assert.Len(t, userID, 12) // NanoID length
	})

	t.Run("Explicit_NanoID_Strategy", func(t *testing.T) {
		provider := user.NewDefaultUser(&user.DefaultUserOptions{
			Prefix:     "test:",
			IDStrategy: user.NanoIDStrategy,
		})

		userID, err := provider.GenerateUserID(context.Background(), false)
		assert.NoError(t, err)
		assert.NotEmpty(t, userID)
		assert.Len(t, userID, 12)
	})

	t.Run("Explicit_UUID_Strategy", func(t *testing.T) {
		provider := user.NewDefaultUser(&user.DefaultUserOptions{
			Prefix:     "test:",
			IDStrategy: user.UUIDStrategy,
		})

		userID, err := provider.GenerateUserID(context.Background(), false)
		assert.NoError(t, err)
		assert.NotEmpty(t, userID)
		assert.Len(t, userID, 36)
	})
}
