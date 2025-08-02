package user_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pquerna/otp/totp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/openapi/oauth/types"
)

func TestMFAOperations(t *testing.T) {
	prepare(t)
	defer clean()

	ctx := context.Background()

	// Use UUID to ensure unique identifiers
	testUUID := strings.ReplaceAll(uuid.New().String(), "-", "")[:8]

	// Create test user data dynamically
	testUser := createTestUserData(testUUID)
	_, testUserID := setupTestUser(t, ctx, testUser)

	var mfaSecret string
	var recoveryCodes []string

	// Test complete MFA setup and usage flow
	t.Run("CompleteFlow", func(t *testing.T) {
		// Step 1: Generate MFA Secret
		secret, qrURL, err := testProvider.GenerateMFASecret(ctx, testUserID, nil)
		assert.NoError(t, err)
		assert.NotEmpty(t, secret)
		assert.NotEmpty(t, qrURL)
		assert.Contains(t, qrURL, "otpauth://totp/")
		assert.Contains(t, qrURL, testUserID) // Account name should default to userID

		mfaSecret = secret

		// Verify MFA is not enabled yet
		enabled, err := testProvider.IsMFAEnabled(ctx, testUserID)
		assert.NoError(t, err)
		assert.False(t, enabled)

		// Step 2: Enable MFA
		code, err := totp.GenerateCode(mfaSecret, time.Now())
		require.NoError(t, err)

		err = testProvider.EnableMFA(ctx, testUserID, mfaSecret, code)
		assert.NoError(t, err)

		// Verify MFA is now enabled
		enabled, err = testProvider.IsMFAEnabled(ctx, testUserID)
		assert.NoError(t, err)
		assert.True(t, enabled)

		// Step 3: Verify MFA Code
		validCode, err := totp.GenerateCode(mfaSecret, time.Now())
		require.NoError(t, err)

		valid, err := testProvider.VerifyMFACode(ctx, testUserID, validCode)
		assert.NoError(t, err)
		assert.True(t, valid)

		// Test invalid code
		valid, err = testProvider.VerifyMFACode(ctx, testUserID, "000000")
		assert.NoError(t, err)
		assert.False(t, valid)

		// Step 4: Get MFA Config
		config, err := testProvider.GetMFAConfig(ctx, testUserID)
		assert.NoError(t, err)
		assert.NotNil(t, config)

		assert.Equal(t, testUserID, config["user_id"])
		assert.Equal(t, true, config["mfa_enabled"])
		assert.Equal(t, "Yao App Engine", config["mfa_issuer"]) // Default issuer
		assert.Equal(t, "SHA256", config["mfa_algorithm"])      // Default algorithm
		// Handle database type variations for integers
		if digits, ok := config["mfa_digits"].(int64); ok {
			assert.Equal(t, int64(6), digits) // Default digits
		} else {
			assert.Equal(t, 6, config["mfa_digits"])
		}
		if period, ok := config["mfa_period"].(int64); ok {
			assert.Equal(t, int64(30), period) // Default period
		} else {
			assert.Equal(t, 30, config["mfa_period"])
		}
		assert.NotNil(t, config["mfa_enabled_at"])
		assert.NotNil(t, config["mfa_last_verified_at"]) // Should be set after VerifyMFACode

		// Step 5: Generate Recovery Codes
		codes, err := testProvider.GenerateRecoveryCodes(ctx, testUserID)
		assert.NoError(t, err)
		assert.NotNil(t, codes)
		assert.Len(t, codes, 16) // 16 recovery codes following GitHub standard

		recoveryCodes = codes

		// Verify code format (should be 12 characters with dashes: XXXX-XXXX-XXXX)
		for _, code := range codes {
			assert.Len(t, code, 14) // 12 chars + 2 dashes
			assert.Regexp(t, `^[23456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnpqrstuvwxyz]{4}-[23456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnpqrstuvwxyz]{4}-[23456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnpqrstuvwxyz]{4}$`, code)
		}

		// Verify all codes are unique
		codeSet := make(map[string]bool)
		for _, code := range codes {
			assert.False(t, codeSet[code], "Duplicate recovery code: %s", code)
			codeSet[code] = true
		}

		// Verify MFA config now shows recovery codes available
		config, err = testProvider.GetMFAConfig(ctx, testUserID)
		assert.NoError(t, err)
		assert.Equal(t, 16, config["recovery_codes_available"])

		// Step 6: Test Recovery Code Verification
		testCode := recoveryCodes[0]
		valid, err = testProvider.VerifyRecoveryCode(ctx, testUserID, testCode)
		assert.NoError(t, err)
		assert.True(t, valid)

		// Test same code again (should be consumed/invalid)
		valid, err = testProvider.VerifyRecoveryCode(ctx, testUserID, testCode)
		assert.NoError(t, err)
		assert.False(t, valid)

		// Verify recovery codes available decreased
		config, err = testProvider.GetMFAConfig(ctx, testUserID)
		assert.NoError(t, err)
		assert.Equal(t, 15, config["recovery_codes_available"]) // One less (16-1=15)

		// Test invalid recovery code
		valid, err = testProvider.VerifyRecoveryCode(ctx, testUserID, "invalid-code")
		assert.NoError(t, err)
		assert.False(t, valid)

		// Step 7: Disable MFA
		disableCode, err := totp.GenerateCode(mfaSecret, time.Now())
		require.NoError(t, err)

		err = testProvider.DisableMFA(ctx, testUserID, disableCode)
		assert.NoError(t, err)

		// Verify MFA is now disabled
		enabled, err = testProvider.IsMFAEnabled(ctx, testUserID)
		assert.NoError(t, err)
		assert.False(t, enabled)

		// Verify MFA config reflects disabled state
		config, err = testProvider.GetMFAConfig(ctx, testUserID)
		assert.NoError(t, err)
		assert.Equal(t, false, config["mfa_enabled"])
		// Should not contain MFA-specific fields when disabled
		assert.NotContains(t, config, "mfa_issuer")
		assert.NotContains(t, config, "mfa_algorithm")
		assert.NotContains(t, config, "recovery_codes_available")
	})
}

func TestMFAErrorHandling(t *testing.T) {
	prepare(t)
	defer clean()

	ctx := context.Background()
	testUUID := strings.ReplaceAll(uuid.New().String(), "-", "")[:8]
	nonExistentUserID := "non-existent-user-" + testUUID

	// Create test user for some error tests
	testUser := createTestUserData("mfaerror" + testUUID)
	_, testUserID := setupTestUser(t, ctx, testUser)

	t.Run("GenerateMFASecret_UserNotFound", func(t *testing.T) {
		_, _, err := testProvider.GenerateMFASecret(ctx, nonExistentUserID, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})

	t.Run("EnableMFA_UserNotFound", func(t *testing.T) {
		err := testProvider.EnableMFA(ctx, nonExistentUserID, "testsecret", "000000")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})

	t.Run("EnableMFA_InvalidCode", func(t *testing.T) {
		// Generate MFA secret first
		secret, _, err := testProvider.GenerateMFASecret(ctx, testUserID, nil)
		require.NoError(t, err)

		// Try to enable with invalid code
		err = testProvider.EnableMFA(ctx, testUserID, secret, "000000")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid MFA code")
	})

	t.Run("EnableMFA_NoSecret", func(t *testing.T) {
		// Try to enable MFA without generating secret first (use new user)
		newUser := createTestUserData("nomfasecret" + testUUID)
		_, newUserID := setupTestUser(t, ctx, newUser)

		err := testProvider.EnableMFA(ctx, newUserID, "", "000000")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no MFA secret found")
	})

	t.Run("DisableMFA_UserNotFound", func(t *testing.T) {
		err := testProvider.DisableMFA(ctx, nonExistentUserID, "000000")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})

	t.Run("DisableMFA_NotEnabled", func(t *testing.T) {
		// Use user without MFA enabled
		newUser := createTestUserData("nomfauser" + testUUID)
		_, newUserID := setupTestUser(t, ctx, newUser)

		err := testProvider.DisableMFA(ctx, newUserID, "000000")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "MFA is not enabled")
	})

	t.Run("VerifyMFACode_UserNotFound", func(t *testing.T) {
		_, err := testProvider.VerifyMFACode(ctx, nonExistentUserID, "000000")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})

	t.Run("VerifyMFACode_NotEnabled", func(t *testing.T) {
		// Use user without MFA enabled
		newUser := createTestUserData("nomfaverify" + testUUID)
		_, newUserID := setupTestUser(t, ctx, newUser)

		_, err := testProvider.VerifyMFACode(ctx, newUserID, "000000")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "MFA is not enabled")
	})

	t.Run("GenerateRecoveryCodes_UserNotFound", func(t *testing.T) {
		_, err := testProvider.GenerateRecoveryCodes(ctx, nonExistentUserID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})

	t.Run("GenerateRecoveryCodes_NotEnabled", func(t *testing.T) {
		// Use user without MFA enabled
		newUser := createTestUserData("norecovery" + testUUID)
		_, newUserID := setupTestUser(t, ctx, newUser)

		_, err := testProvider.GenerateRecoveryCodes(ctx, newUserID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "MFA is not enabled")
	})

	t.Run("VerifyRecoveryCode_UserNotFound", func(t *testing.T) {
		_, err := testProvider.VerifyRecoveryCode(ctx, nonExistentUserID, "test-code")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})

	t.Run("VerifyRecoveryCode_NotEnabled", func(t *testing.T) {
		// Use user without MFA enabled
		newUser := createTestUserData("noverifyrecov" + testUUID)
		_, newUserID := setupTestUser(t, ctx, newUser)

		_, err := testProvider.VerifyRecoveryCode(ctx, newUserID, "test-code")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "MFA is not enabled")
	})

	t.Run("VerifyRecoveryCode_NoRecoveryCodes", func(t *testing.T) {
		// Create user, enable MFA, but don't generate recovery codes
		newUser := createTestUserData("norecodes" + testUUID)
		_, newUserID := setupTestUser(t, ctx, newUser)

		// Generate and enable MFA
		secret, _, err := testProvider.GenerateMFASecret(ctx, newUserID, nil)
		require.NoError(t, err)

		code, err := totp.GenerateCode(secret, time.Now())
		require.NoError(t, err)

		err = testProvider.EnableMFA(ctx, newUserID, secret, code)
		require.NoError(t, err)

		// Try to verify recovery code without generating them
		_, err = testProvider.VerifyRecoveryCode(ctx, newUserID, "test-code")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no recovery codes found")
	})

	t.Run("IsMFAEnabled_UserNotFound", func(t *testing.T) {
		_, err := testProvider.IsMFAEnabled(ctx, nonExistentUserID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})

	t.Run("GetMFAConfig_UserNotFound", func(t *testing.T) {
		_, err := testProvider.GetMFAConfig(ctx, nonExistentUserID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})
}

func TestMFACustomOptions(t *testing.T) {
	prepare(t)
	defer clean()

	ctx := context.Background()
	testUUID := strings.ReplaceAll(uuid.New().String(), "-", "")[:8]

	t.Run("CustomMFAOptions", func(t *testing.T) {
		// Create test user
		testUser := createTestUserData("mfaoptions" + testUUID)
		_, testUserID := setupTestUser(t, ctx, testUser)

		// Test with custom options
		customOptions := &types.MFAOptions{
			Issuer:      "Custom MFA Test",
			Algorithm:   "SHA1",
			Digits:      8,
			Period:      60,
			SecretSize:  16,
			AccountName: "custom@test.com",
		}

		// Generate MFA secret with custom options
		secret, qrURL, err := testProvider.GenerateMFASecret(ctx, testUserID, customOptions)
		assert.NoError(t, err)
		assert.NotEmpty(t, secret)
		assert.Contains(t, qrURL, "Custom%20MFA%20Test")
		assert.Contains(t, qrURL, "custom@test.com")
		assert.Contains(t, qrURL, "algorithm=SHA1")
		assert.Contains(t, qrURL, "digits=8")
		assert.Contains(t, qrURL, "period=60")

		// Enable MFA
		code, err := totp.GenerateCode(secret, time.Now())
		require.NoError(t, err)
		err = testProvider.EnableMFA(ctx, testUserID, secret, code)
		assert.NoError(t, err)

		// Verify MFA config shows custom settings
		config, err := testProvider.GetMFAConfig(ctx, testUserID)
		assert.NoError(t, err)
		assert.Equal(t, "Custom MFA Test", config["mfa_issuer"])
		assert.Equal(t, "SHA1", config["mfa_algorithm"])
		// Handle database type variations
		if digits, ok := config["mfa_digits"].(int64); ok {
			assert.Equal(t, int64(8), digits)
		} else {
			assert.Equal(t, 8, config["mfa_digits"])
		}
		if period, ok := config["mfa_period"].(int64); ok {
			assert.Equal(t, int64(60), period)
		} else {
			assert.Equal(t, 60, config["mfa_period"])
		}
	})

	t.Run("PartialCustomOptions", func(t *testing.T) {
		// Create another user for partial options test
		testUser := createTestUserData("mfapartial" + testUUID)
		_, testUserID := setupTestUser(t, ctx, testUser)

		// Test with partial custom options (some fields empty)
		partialOptions := &types.MFAOptions{
			Issuer:      "Partial Test",
			AccountName: "partial@test.com",
			// Other fields empty, should use defaults
		}

		secret, qrURL, err := testProvider.GenerateMFASecret(ctx, testUserID, partialOptions)
		assert.NoError(t, err)
		assert.NotEmpty(t, secret)
		assert.Contains(t, qrURL, "Partial%20Test")
		assert.Contains(t, qrURL, "partial@test.com")
		// Should use defaults for other parameters
		assert.Contains(t, qrURL, "algorithm=SHA256") // Default
		assert.Contains(t, qrURL, "digits=6")         // Default
		assert.Contains(t, qrURL, "period=30")        // Default
	})
}

func TestMFAStateMachine(t *testing.T) {
	prepare(t)
	defer clean()

	ctx := context.Background()
	testUUID := strings.ReplaceAll(uuid.New().String(), "-", "")[:8]

	// Create test user
	testUser := createTestUserData("mfastate" + testUUID)
	_, testUserID := setupTestUser(t, ctx, testUser)

	t.Run("StateTransitions", func(t *testing.T) {
		// State 1: No MFA configured
		enabled, err := testProvider.IsMFAEnabled(ctx, testUserID)
		assert.NoError(t, err)
		assert.False(t, enabled)

		// Should fail to verify code when MFA not enabled
		_, err = testProvider.VerifyMFACode(ctx, testUserID, "000000")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "MFA is not enabled")

		// State 2: Generate secret (but not enabled yet)
		secret, _, err := testProvider.GenerateMFASecret(ctx, testUserID, nil)
		assert.NoError(t, err)

		enabled, err = testProvider.IsMFAEnabled(ctx, testUserID)
		assert.NoError(t, err)
		assert.False(t, enabled) // Still not enabled

		// State 3: Enable MFA
		code, err := totp.GenerateCode(secret, time.Now())
		require.NoError(t, err)

		err = testProvider.EnableMFA(ctx, testUserID, secret, code)
		assert.NoError(t, err)

		enabled, err = testProvider.IsMFAEnabled(ctx, testUserID)
		assert.NoError(t, err)
		assert.True(t, enabled) // Now enabled

		// Should be able to verify codes now
		newCode, err := totp.GenerateCode(secret, time.Now())
		require.NoError(t, err)

		valid, err := testProvider.VerifyMFACode(ctx, testUserID, newCode)
		assert.NoError(t, err)
		assert.True(t, valid)

		// State 4: Regenerate secret (MFA remains enabled but with new secret)
		newSecret, _, err := testProvider.GenerateMFASecret(ctx, testUserID, nil)
		assert.NoError(t, err)
		assert.NotEqual(t, secret, newSecret)

		// MFA should still be enabled
		enabled, err = testProvider.IsMFAEnabled(ctx, testUserID)
		assert.NoError(t, err)
		assert.True(t, enabled) // Still enabled

		// Old codes should not work after regenerating secret, but new codes should work
		newCode2, err := totp.GenerateCode(newSecret, time.Now())
		require.NoError(t, err)

		valid, err = testProvider.VerifyMFACode(ctx, testUserID, newCode2)
		assert.NoError(t, err)
		assert.True(t, valid) // Should work with new secret

		// Old codes should not work
		oldCode, err := totp.GenerateCode(secret, time.Now())
		require.NoError(t, err)

		valid, err = testProvider.VerifyMFACode(ctx, testUserID, oldCode)
		assert.NoError(t, err)
		assert.False(t, valid) // Should fail with old secret

		// State 5: Disable MFA
		disableCode, err := totp.GenerateCode(newSecret, time.Now())
		require.NoError(t, err)

		err = testProvider.DisableMFA(ctx, testUserID, disableCode)
		assert.NoError(t, err)

		enabled, err = testProvider.IsMFAEnabled(ctx, testUserID)
		assert.NoError(t, err)
		assert.False(t, enabled) // Back to disabled

		// Should not be able to verify codes after disabling
		_, err = testProvider.VerifyMFACode(ctx, testUserID, disableCode)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "MFA is not enabled")
	})
}
