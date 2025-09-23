package user

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"

	"github.com/google/uuid"
	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/yaoapp/gou/model"
)

// Utils

// GenerateUserID generates a new unique user_id for user creation
// safe: optional parameter, if true check for collisions and retry if needed
//
//	defaults to true for NanoID, false for UUID
func (u *DefaultUser) GenerateUserID(ctx context.Context, safe ...bool) (string, error) {
	// Determine safe mode: default based on strategy, or use provided value
	var safeMode bool
	if len(safe) > 0 {
		safeMode = safe[0] // Use provided value
	} else {
		// Default: if idStrategy is Numeric or NanoID, use safe mode.
		safeMode = (u.idStrategy == NumericStrategy) || (u.idStrategy == NanoIDStrategy)
	}

	if !safeMode {
		// Direct generation without collision detection (UUID case)
		return u.generateUserID()
	}

	// Safe generation with collision detection (NanoID case)
	const maxRetries = 10 // Prevent infinite loops

	for i := 0; i < maxRetries; i++ {
		// Generate new ID
		id, err := u.generateUserID()
		if err != nil {
			return "", fmt.Errorf(ErrFailedToGenerateUserID, err)
		}

		// Check if ID already exists
		exists, err := u.userIDExists(ctx, id)
		if err != nil {
			return "", fmt.Errorf("failed to check user_id existence: %w", err)
		}

		if !exists {
			return id, nil // Found unique ID
		}

		// ID exists, retry with new generation
	}

	return "", fmt.Errorf("failed to generate unique user_id after %d retries", maxRetries)
}

// generateUserID generates a new user_id based on configured strategy (internal use)
func (u *DefaultUser) generateUserID() (string, error) {
	var id string
	var err error

	switch u.idStrategy {
	case UUIDStrategy:
		id, err = generateUUID()
	case NanoIDStrategy:
		id, err = generateNanoID(12) // 12 characters, URL-safe, readable
	case NumericStrategy:
		id, err = generateNumericID(12) // 12 characters, numeric, readable (default)
	default:
		id, err = generateNumericID(12) // 12 characters, URL-safe, readable
	}

	if err != nil {
		return "", err
	}

	// Add prefix if configured
	if u.idPrefix != "" {
		return u.idPrefix + id, nil
	}

	return id, nil
}

// generateInvitationID generates a new invitation_id based on configured strategy (internal use)
func (u *DefaultUser) generateInvitationID() (string, error) {
	var id string
	var err error

	switch u.idStrategy {
	case UUIDStrategy:
		id, err = generateUUID()
	case NanoIDStrategy:
		id, err = generateNanoID(12) // 12 characters, URL-safe, readable
	case NumericStrategy:
		id, err = generateNumericID(12) // 12 characters, numeric, readable (default)
	default:
		id, err = generateNumericID(12) // 12 characters, URL-safe, readable
	}

	if err != nil {
		return "", err
	}

	// Add prefix if configured (could be different from user prefix)
	prefix := "inv_" // Default invitation prefix
	if u.idPrefix != "" {
		prefix = u.idPrefix + "inv_"
	}

	return prefix + id, nil
}

// userIDExists checks if a user_id already exists in the database
func (u *DefaultUser) userIDExists(ctx context.Context, userID string) (bool, error) {
	m := model.Select(u.model)
	users, err := m.Get(model.QueryParam{
		Select: []interface{}{"id"}, // Just get primary key, minimal data
		Wheres: []model.QueryWhere{
			{Column: "user_id", Value: userID},
		},
		Limit: 1,
	})

	if err != nil {
		return false, err
	}

	return len(users) > 0, nil
}

// GetOAuthUserID quickly retrieves user_id by OAuth provider and subject
func (u *DefaultUser) GetOAuthUserID(ctx context.Context, provider string, subject string) (string, error) {
	m := model.Select(u.oauthAccountModel)
	accounts, err := m.Get(model.QueryParam{
		Select: []interface{}{"user_id"},
		Wheres: []model.QueryWhere{
			{Column: "provider", Value: provider},
			{Column: "sub", Value: subject},
		},
		Limit: 1,
	})

	if err != nil {
		return "", fmt.Errorf(ErrFailedToGetOAuthAccount, err)
	}

	if len(accounts) == 0 {
		return "", fmt.Errorf(ErrOAuthAccountNotFound)
	}

	userID, ok := accounts[0]["user_id"].(string)
	if !ok {
		return "", fmt.Errorf(ErrInvalidUserIDInOAuth)
	}

	return userID, nil
}

// generateNanoID generates a Nano ID using the library
func generateNanoID(length int) (string, error) {
	// URL-safe alphabet (no ambiguous characters like 0/O, 1/l/I)
	const alphabet = "23456789ABCDEFGHJKMNPQRSTUVWXYZabcdefghijkmnpqrstuvwxyz"
	return gonanoid.Generate(alphabet, length)
}

// generateNumericID generates a numeric ID
func generateNumericID(length int) (string, error) {
	if length <= 0 || length > 16 {
		return "", fmt.Errorf("length must be between 1 and 16")
	}
	return gonanoid.Generate("0123456789", length)
}

// generateUUID generates a traditional UUID using Google's library
func generateUUID() (string, error) {
	return uuid.NewString(), nil
}

// generateRandomPassword generates a random password with specified length
func generateRandomPassword(length int) (string, error) {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789!@#$%^&*"
	bytes := make([]byte, length)

	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	for i, b := range bytes {
		bytes[i] = charset[b%byte(len(charset))]
	}

	return string(bytes), nil
}

// parseTimeFromDB parses time values from database fields, handling different formats and types
func parseTimeFromDB(value interface{}) (*time.Time, error) {
	if value == nil {
		return nil, nil
	}

	switch v := value.(type) {
	case time.Time:
		return &v, nil
	case string:
		if v == "" {
			return nil, nil
		}
		// Try parsing common time formats - assume local timezone for database timestamps
		if parsedTime, err := time.ParseInLocation("2006-01-02 15:04:05", v, time.Local); err == nil {
			return &parsedTime, nil
		}
		if parsedTime, err := time.Parse(time.RFC3339, v); err == nil {
			return &parsedTime, nil
		}
		if parsedTime, err := time.ParseInLocation("2006-01-02T15:04:05", v, time.Local); err == nil {
			return &parsedTime, nil
		}
		if parsedTime, err := time.ParseInLocation("2006-01-02 15:04:05.000000", v, time.Local); err == nil {
			return &parsedTime, nil
		}
		return nil, fmt.Errorf("unable to parse time format: %s", v)
	default:
		return nil, fmt.Errorf("unsupported time type: %T", value)
	}
}

// parseIntFromDB parses integer values from database fields, handling different integer types
func parseIntFromDB(value interface{}) (int64, error) {
	if value == nil {
		return 0, fmt.Errorf("value is nil")
	}

	switch v := value.(type) {
	case int64:
		return v, nil
	case int:
		return int64(v), nil
	case int32:
		return int64(v), nil
	case uint:
		return int64(v), nil
	case uint32:
		return int64(v), nil
	case uint64:
		// Check for overflow
		if v > 9223372036854775807 { // max int64
			return 0, fmt.Errorf("value too large for int64: %d", v)
		}
		return int64(v), nil
	case float64:
		// Handle cases where database returns numbers as floats
		return int64(v), nil
	case string:
		// Try to parse string as integer
		if parsed, err := fmt.Sscanf(v, "%d", new(int64)); err == nil && parsed == 1 {
			var result int64
			fmt.Sscanf(v, "%d", &result)
			return result, nil
		}
		return 0, fmt.Errorf("unable to parse string as integer: %s", v)
	default:
		return 0, fmt.Errorf("unsupported integer type: %T", value)
	}
}

// checkTimeExpired checks if a time field from database indicates expiration
func checkTimeExpired(value interface{}) (bool, error) {
	parsedTime, err := parseTimeFromDB(value)
	if err != nil {
		return false, err
	}
	if parsedTime == nil {
		return false, nil // No expiry time set
	}
	return time.Now().After(*parsedTime), nil
}
