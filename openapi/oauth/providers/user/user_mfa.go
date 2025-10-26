package user

import (
	"context"
	"crypto/rand"
	"fmt"
	"strings"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/yao/openapi/oauth/types"
	"golang.org/x/crypto/bcrypt"
)

// User MFA Management

// GenerateMFASecret generates a new TOTP secret for user
func (u *DefaultUser) GenerateMFASecret(ctx context.Context, userID string, options *types.MFAOptions) (string, string, error) {
	// Verify user exists
	m := model.Select(u.model)
	users, err := m.Get(model.QueryParam{
		Select: []interface{}{"user_id", "mfa_enabled"},
		Wheres: []model.QueryWhere{
			{Column: "user_id", Value: userID},
		},
		Limit: 1,
	})

	if err != nil {
		return "", "", fmt.Errorf(ErrFailedToGetUser, err)
	}

	if len(users) == 0 {
		return "", "", fmt.Errorf(ErrUserNotFound)
	}

	// Use provided options or fallback to instance defaults
	if options == nil {
		options = u.mfaOptions
	}

	// Apply defaults for individual fields if not specified
	issuer := options.Issuer
	if issuer == "" {
		issuer = u.mfaOptions.Issuer
	}

	accountName := options.AccountName
	if accountName == "" {
		accountName = userID // Default to userID
	}

	algorithm := options.Algorithm
	if algorithm == "" {
		algorithm = u.mfaOptions.Algorithm
	}

	digits := options.Digits
	if digits == 0 {
		digits = u.mfaOptions.Digits
	}

	period := options.Period
	if period == 0 {
		period = u.mfaOptions.Period
	}

	secretSize := options.SecretSize
	if secretSize == 0 {
		secretSize = u.mfaOptions.SecretSize
	}

	// Convert algorithm string to otp.Algorithm
	var otpAlgorithm otp.Algorithm
	switch algorithm {
	case "SHA1":
		otpAlgorithm = otp.AlgorithmSHA1
	case "SHA256":
		otpAlgorithm = otp.AlgorithmSHA256
	case "SHA512":
		otpAlgorithm = otp.AlgorithmSHA512
	default:
		otpAlgorithm = otp.AlgorithmSHA256 // Default fallback
	}

	// Convert digits to otp.Digits
	var otpDigits otp.Digits
	if digits == 8 {
		otpDigits = otp.DigitsEight
	} else {
		otpDigits = otp.DigitsSix // Default
	}

	// Generate TOTP key
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      issuer,
		AccountName: accountName,
		SecretSize:  uint(secretSize),
		Algorithm:   otpAlgorithm,
		Digits:      otpDigits,
		Period:      uint(period),
	})

	if err != nil {
		return "", "", fmt.Errorf(ErrFailedToGenerateMFASecret, err)
	}

	secret := key.Secret()
	qrCodeURL := key.URL()

	// Store the secret temporarily (not enabled yet until user verifies)
	updateData := maps.MapStrAny{
		"mfa_secret":    secret,
		"mfa_issuer":    issuer,
		"mfa_algorithm": algorithm,
		"mfa_digits":    digits,
		"mfa_period":    period,
	}

	_, err = m.UpdateWhere(model.QueryParam{
		Wheres: []model.QueryWhere{
			{Column: "user_id", Value: userID},
		},
		Limit: 1,
	}, updateData)

	if err != nil {
		return "", "", fmt.Errorf(ErrFailedToUpdateMFAStatus, err)
	}

	return secret, qrCodeURL, nil
}

// EnableMFA enables multi-factor authentication for user
func (u *DefaultUser) EnableMFA(ctx context.Context, userID string, secret string, code string) error {
	// Get user and current MFA status
	m := model.Select(u.model)
	users, err := m.Get(model.QueryParam{
		Select: []interface{}{"user_id", "mfa_enabled", "mfa_secret"},
		Wheres: []model.QueryWhere{
			{Column: "user_id", Value: userID},
		},
		Limit: 1,
	})

	if err != nil {
		return fmt.Errorf(ErrFailedToGetUser, err)
	}

	if len(users) == 0 {
		return fmt.Errorf(ErrUserNotFound)
	}

	user := users[0]

	// Check if MFA is already enabled
	if mfaEnabled, ok := user["mfa_enabled"].(bool); ok && mfaEnabled {
		return fmt.Errorf(ErrMFAAlreadyEnabled)
	}
	// Handle different boolean types from database
	if mfaEnabledInt, ok := user["mfa_enabled"].(int64); ok && mfaEnabledInt != 0 {
		return fmt.Errorf(ErrMFAAlreadyEnabled)
	}

	// Use stored secret if not provided
	if secret == "" {
		if storedSecret, ok := user["mfa_secret"].(string); ok && storedSecret != "" {
			secret = storedSecret
		} else {
			return fmt.Errorf("no MFA secret found, please generate one first")
		}
	}

	// Verify the provided code
	valid := totp.Validate(code, secret)
	if !valid {
		return fmt.Errorf(ErrInvalidMFACode)
	}

	// Enable MFA
	updateData := maps.MapStrAny{
		"mfa_enabled":    true,
		"mfa_secret":     secret, // Store the verified secret
		"mfa_enabled_at": time.Now(),
	}

	affected, err := m.UpdateWhere(model.QueryParam{
		Wheres: []model.QueryWhere{
			{Column: "user_id", Value: userID},
		},
		Limit: 1,
	}, updateData)

	if err != nil {
		return fmt.Errorf(ErrFailedToUpdateMFAStatus, err)
	}

	if affected == 0 {
		// Check if user exists
		exists, checkErr := u.UserExists(ctx, userID)
		if checkErr != nil {
			return fmt.Errorf(ErrFailedToUpdateMFAStatus, checkErr)
		}
		if !exists {
			return fmt.Errorf(ErrUserNotFound)
		}
		// User exists but no changes were made (already enabled with same secret)
	}

	return nil
}

// DisableMFA disables multi-factor authentication for user
func (u *DefaultUser) DisableMFA(ctx context.Context, userID string, code string) error {
	// Get user and current MFA status
	m := model.Select(u.model)
	users, err := m.Get(model.QueryParam{
		Select: []interface{}{"user_id", "mfa_enabled", "mfa_secret"},
		Wheres: []model.QueryWhere{
			{Column: "user_id", Value: userID},
		},
		Limit: 1,
	})

	if err != nil {
		return fmt.Errorf(ErrFailedToGetUser, err)
	}

	if len(users) == 0 {
		return fmt.Errorf(ErrUserNotFound)
	}

	user := users[0]

	// Check if MFA is enabled
	mfaEnabled := false
	if enabled, ok := user["mfa_enabled"].(bool); ok {
		mfaEnabled = enabled
	} else if enabledInt, ok := user["mfa_enabled"].(int64); ok {
		mfaEnabled = enabledInt != 0
	}

	if !mfaEnabled {
		return fmt.Errorf(ErrMFANotEnabled)
	}

	// Get stored secret
	secret, ok := user["mfa_secret"].(string)
	if !ok || secret == "" {
		return fmt.Errorf("no MFA secret found")
	}

	// Verify the provided code
	valid := totp.Validate(code, secret)
	if !valid {
		return fmt.Errorf(ErrInvalidMFACode)
	}

	// Disable MFA and clear sensitive data
	updateData := maps.MapStrAny{
		"mfa_enabled":          false,
		"mfa_secret":           nil, // Clear the secret
		"mfa_recovery_hash":    nil, // Clear recovery codes
		"mfa_enabled_at":       nil,
		"mfa_last_verified_at": nil,
	}

	affected, err := m.UpdateWhere(model.QueryParam{
		Wheres: []model.QueryWhere{
			{Column: "user_id", Value: userID},
		},
		Limit: 1,
	}, updateData)

	if err != nil {
		return fmt.Errorf(ErrFailedToUpdateMFAStatus, err)
	}

	if affected == 0 {
		// Check if user exists
		exists, checkErr := u.UserExists(ctx, userID)
		if checkErr != nil {
			return fmt.Errorf(ErrFailedToUpdateMFAStatus, checkErr)
		}
		if !exists {
			return fmt.Errorf(ErrUserNotFound)
		}
		// User exists but no changes were made (already disabled)
	}

	return nil
}

// VerifyMFACode verifies a TOTP code for user
func (u *DefaultUser) VerifyMFACode(ctx context.Context, userID string, code string) (bool, error) {
	// Get user and MFA status
	m := model.Select(u.model)
	users, err := m.Get(model.QueryParam{
		Select: []interface{}{"user_id", "mfa_enabled", "mfa_secret"},
		Wheres: []model.QueryWhere{
			{Column: "user_id", Value: userID},
		},
		Limit: 1,
	})

	if err != nil {
		return false, fmt.Errorf(ErrFailedToGetUser, err)
	}

	if len(users) == 0 {
		return false, fmt.Errorf(ErrUserNotFound)
	}

	user := users[0]

	// Check if MFA is enabled
	mfaEnabled := false
	if enabled, ok := user["mfa_enabled"].(bool); ok {
		mfaEnabled = enabled
	} else if enabledInt, ok := user["mfa_enabled"].(int64); ok {
		mfaEnabled = enabledInt != 0
	}

	if !mfaEnabled {
		return false, fmt.Errorf(ErrMFANotEnabled)
	}

	// Get stored secret
	secret, ok := user["mfa_secret"].(string)
	if !ok || secret == "" {
		return false, fmt.Errorf("no MFA secret found")
	}

	// Verify the code
	valid := totp.Validate(code, secret)
	if valid {
		// Update last verified timestamp
		_, err = m.UpdateWhere(model.QueryParam{
			Wheres: []model.QueryWhere{
				{Column: "user_id", Value: userID},
			},
			Limit: 1,
		}, maps.MapStrAny{
			"mfa_last_verified_at": time.Now(),
		})
		// Don't fail verification if timestamp update fails
		if err != nil {
			// Log the error but continue
		}
	}

	return valid, nil
}

// GenerateRecoveryCodes generates new recovery codes for user and stores their hash
func (u *DefaultUser) GenerateRecoveryCodes(ctx context.Context, userID string) ([]string, error) {
	// Verify user exists and MFA is enabled
	m := model.Select(u.model)
	users, err := m.Get(model.QueryParam{
		Select: []interface{}{"user_id", "mfa_enabled"},
		Wheres: []model.QueryWhere{
			{Column: "user_id", Value: userID},
		},
		Limit: 1,
	})

	if err != nil {
		return nil, fmt.Errorf(ErrFailedToGetUser, err)
	}

	if len(users) == 0 {
		return nil, fmt.Errorf(ErrUserNotFound)
	}

	user := users[0]

	// Check if MFA is enabled
	mfaEnabled := false
	if enabled, ok := user["mfa_enabled"].(bool); ok {
		mfaEnabled = enabled
	} else if enabledInt, ok := user["mfa_enabled"].(int64); ok {
		mfaEnabled = enabledInt != 0
	}

	if !mfaEnabled {
		return nil, fmt.Errorf(ErrMFANotEnabled)
	}

	// Generate multiple recovery codes and store bcrypt hashes (512 char limit)
	recoveryCount := u.mfaOptions.RecoveryCount
	recoveryLength := u.mfaOptions.RecoveryLength

	recoveryCodes := make([]string, recoveryCount)

	for i := 0; i < recoveryCount; i++ {
		code, err := generateRecoveryCode(recoveryLength)
		if err != nil {
			return nil, fmt.Errorf("failed to generate recovery code: %w", err)
		}
		recoveryCodes[i] = code
	}

	// Hash each recovery code with bcrypt and store all hashes
	recoveryHashes := make([]string, recoveryCount)
	for i, code := range recoveryCodes {
		hashedCode, err := bcrypt.GenerateFromPassword([]byte(code), bcrypt.DefaultCost)
		if err != nil {
			return nil, fmt.Errorf("failed to hash recovery code: %w", err)
		}
		recoveryHashes[i] = string(hashedCode)
	}

	// Join all bcrypt hashes (~60 bytes each, 8 hashes = ~480 bytes, under 512 limit)
	allHashesStr := strings.Join(recoveryHashes, "|||")

	updateData := maps.MapStrAny{
		"mfa_recovery_hash": allHashesStr, // Store bcrypt hashes (~480 bytes, under 512 limit)
	}

	affected, err := m.UpdateWhere(model.QueryParam{
		Wheres: []model.QueryWhere{
			{Column: "user_id", Value: userID},
		},
		Limit: 1,
	}, updateData)

	if err != nil {
		return nil, fmt.Errorf(ErrFailedToUpdateMFAStatus, err)
	}

	if affected == 0 {
		// Check if user exists
		exists, checkErr := u.UserExists(ctx, userID)
		if checkErr != nil {
			return nil, fmt.Errorf(ErrFailedToUpdateMFAStatus, checkErr)
		}
		if !exists {
			return nil, fmt.Errorf(ErrUserNotFound)
		}
		// User exists but no changes were made
	}

	// Return all generated recovery codes
	return recoveryCodes, nil
}

// VerifyRecoveryCode verifies and consumes a recovery code
func (u *DefaultUser) VerifyRecoveryCode(ctx context.Context, userID string, code string) (bool, error) {
	// Get user and MFA status
	m := model.Select(u.model)
	users, err := m.Get(model.QueryParam{
		Select: []interface{}{"user_id", "mfa_enabled", "mfa_recovery_hash"},
		Wheres: []model.QueryWhere{
			{Column: "user_id", Value: userID},
		},
		Limit: 1,
	})

	if err != nil {
		return false, fmt.Errorf(ErrFailedToGetUser, err)
	}

	if len(users) == 0 {
		return false, fmt.Errorf(ErrUserNotFound)
	}

	user := users[0]

	// Check if MFA is enabled
	mfaEnabled := false
	if enabled, ok := user["mfa_enabled"].(bool); ok {
		mfaEnabled = enabled
	} else if enabledInt, ok := user["mfa_enabled"].(int64); ok {
		mfaEnabled = enabledInt != 0
	}

	if !mfaEnabled {
		return false, fmt.Errorf(ErrMFANotEnabled)
	}

	// Get stored bcrypt hashes string (512 char limit - no problem!)
	recoveryHashesStr, ok := user["mfa_recovery_hash"].(string)
	if !ok || recoveryHashesStr == "" {
		return false, fmt.Errorf("no recovery codes found")
	}

	// Split into hashes list
	recoveryHashes := strings.Split(recoveryHashesStr, "|||")
	if len(recoveryHashes) == 0 {
		return false, fmt.Errorf("no recovery codes found")
	}

	// Check if user input code matches any stored bcrypt hash
	matchIndex := -1
	for i, storedHash := range recoveryHashes {
		if storedHash == "" {
			continue // Skip already used codes
		}
		// Verify user input against stored bcrypt hash
		err := bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(code))
		if err == nil {
			matchIndex = i
			break
		}
	}

	if matchIndex == -1 {
		return false, nil // Invalid code
	}

	// Mark the code as used by clearing its hash
	recoveryHashes[matchIndex] = ""
	updatedHashesStr := strings.Join(recoveryHashes, "|||")

	// Update recovery codes in database and mark verification time
	updateData := maps.MapStrAny{
		"mfa_recovery_hash":    updatedHashesStr,
		"mfa_last_verified_at": time.Now(),
	}

	_, err = m.UpdateWhere(model.QueryParam{
		Wheres: []model.QueryWhere{
			{Column: "user_id", Value: userID},
		},
		Limit: 1,
	}, updateData)

	if err != nil {
		return false, fmt.Errorf(ErrFailedToUpdateMFAStatus, err)
	}

	return true, nil
}

// IsMFAEnabled checks if MFA is enabled for a user
func (u *DefaultUser) IsMFAEnabled(ctx context.Context, userID string) (bool, error) {
	m := model.Select(u.model)
	users, err := m.Get(model.QueryParam{
		Select: []interface{}{"user_id", "mfa_enabled"},
		Wheres: []model.QueryWhere{
			{Column: "user_id", Value: userID},
		},
		Limit: 1,
	})

	if err != nil {
		return false, fmt.Errorf(ErrFailedToGetUser, err)
	}

	if len(users) == 0 {
		return false, fmt.Errorf(ErrUserNotFound)
	}

	user := users[0]

	// Check MFA status
	if enabled, ok := user["mfa_enabled"].(bool); ok {
		return enabled, nil
	}
	// Handle different boolean types from database
	if enabledInt, ok := user["mfa_enabled"].(int64); ok {
		return enabledInt != 0, nil
	}

	return false, nil
}

// GetMFAConfig retrieves MFA configuration for a user
func (u *DefaultUser) GetMFAConfig(ctx context.Context, userID string) (maps.MapStrAny, error) {
	m := model.Select(u.model)
	users, err := m.Get(model.QueryParam{
		Select: []interface{}{
			"user_id", "mfa_enabled", "mfa_issuer", "mfa_algorithm",
			"mfa_digits", "mfa_period", "mfa_enabled_at", "mfa_last_verified_at",
			"mfa_recovery_hash", // Include recovery hash field
		},
		Wheres: []model.QueryWhere{
			{Column: "user_id", Value: userID},
		},
		Limit: 1,
	})

	if err != nil {
		return nil, fmt.Errorf(ErrFailedToGetUser, err)
	}

	if len(users) == 0 {
		return nil, fmt.Errorf(ErrUserNotFound)
	}

	user := users[0]

	// Check if MFA is enabled
	mfaEnabled := false
	if enabled, ok := user["mfa_enabled"].(bool); ok {
		mfaEnabled = enabled
	} else if enabledInt, ok := user["mfa_enabled"].(int64); ok {
		mfaEnabled = enabledInt != 0
	}

	config := maps.MapStrAny{
		"user_id":     userID,
		"mfa_enabled": mfaEnabled,
	}

	if mfaEnabled {
		// Include MFA configuration details (but not the secret)
		config["mfa_issuer"] = user["mfa_issuer"]
		config["mfa_algorithm"] = user["mfa_algorithm"]
		config["mfa_digits"] = user["mfa_digits"]
		config["mfa_period"] = user["mfa_period"]
		config["mfa_enabled_at"] = user["mfa_enabled_at"]
		config["mfa_last_verified_at"] = user["mfa_last_verified_at"]

		// Check how many recovery codes are available (bcrypt hash storage)
		if recoveryHashesStr, ok := user["mfa_recovery_hash"].(string); ok && recoveryHashesStr != "" {
			hashes := strings.Split(recoveryHashesStr, "|||")
			remainingCodes := 0
			for _, hash := range hashes {
				if hash != "" {
					remainingCodes++
				}
			}
			config["recovery_codes_available"] = remainingCodes
		} else {
			config["recovery_codes_available"] = 0
		}
	}

	return config, nil
}

// Helper function to generate recovery codes
func generateRecoveryCode(length int) (string, error) {
	// Use alphanumeric charset (excluding similar-looking characters for better UX)
	// Excludes: 0, O, 1, I, l to avoid confusion
	const charset = "23456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnpqrstuvwxyz"
	b := make([]byte, length)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}

	for i := range b {
		b[i] = charset[b[i]%byte(len(charset))]
	}

	// Format with dashes for better readability (like GitHub)
	// For 8-character codes: XXXX-XXXX
	// For 12-character codes: XXXX-XXXX-XXXX
	result := string(b)
	if length == 8 {
		return fmt.Sprintf("%s-%s", result[:4], result[4:]), nil
	} else if length >= 12 {
		return fmt.Sprintf("%s-%s-%s", result[:4], result[4:8], result[8:]), nil
	}

	return result, nil
}
