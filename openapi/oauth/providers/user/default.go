package user

import (
	"context"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"hash"
	"math"
	"net/url"
	"reflect"
	"strings"
	"time"

	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/gou/store"
)

// Safe user fields that can be displayed to users
var (
	// PublicUserFields contains fields that can be safely returned to users
	PublicUserFields = []interface{}{
		"id", "subject", "username", "email", "first_name", "last_name",
		"full_name", "avatar_url", "mobile", "address", "scopes", "status",
		"email_verified", "mobile_verified", "two_factor_enabled",
		"last_login_at", "metadata", "preferences", "created_at", "updated_at",
	}

	// BasicUserFields contains minimal fields for basic user info
	BasicUserFields = []interface{}{
		"id", "subject", "username", "email", "first_name", "last_name",
		"full_name", "avatar_url", "status", "email_verified", "mobile_verified",
	}

	// AuthUserFields contains fields needed for authentication
	AuthUserFields = []interface{}{
		"id", "subject", "username", "email", "password_hash", "scopes", "status",
		"email_verified", "mobile_verified", "two_factor_enabled", "last_login_at",
	}

	// TwoFactorUserFields contains fields needed for two-factor authentication
	TwoFactorUserFields = []interface{}{
		"id", "two_factor_enabled", "two_factor_secret", "two_factor_algorithm",
		"two_factor_digits", "two_factor_period", "two_factor_recovery_codes",
	}
)

// DefaultUser provides a default implementation of UserProvider
type DefaultUser struct {
	prefix     string
	model      string
	cache      store.Store
	tokenStore store.Store
}

// DefaultUserOptions provides options for the DefaultUser
type DefaultUserOptions struct {
	Prefix     string
	Model      string // bind to a specific user model
	Cache      store.Store
	TokenStore store.Store // store for OAuth tokens
}

// NewDefaultUser creates a new DefaultUser
func NewDefaultUser(options *DefaultUserOptions) *DefaultUser {
	// Set default model name if not specified
	modelName := options.Model
	if modelName == "" {
		modelName = "__yao.user"
	}

	return &DefaultUser{
		prefix:     options.Prefix,
		model:      modelName,
		cache:      options.Cache,
		tokenStore: options.TokenStore,
	}
}

// Key generation methods

func (u *DefaultUser) tokenKey(accessToken string) string {
	return fmt.Sprintf("%s:token:%s", u.prefix, accessToken)
}

func (u *DefaultUser) cacheKey(userID string) string {
	return fmt.Sprintf("%s:user:%s", u.prefix, userID)
}

func (u *DefaultUser) subjectCacheKey(subject string) string {
	return fmt.Sprintf("%s:user:subject:%s", u.prefix, subject)
}

func (u *DefaultUser) usernameCacheKey(username string) string {
	return fmt.Sprintf("%s:user:username:%s", u.prefix, username)
}

func (u *DefaultUser) emailCacheKey(email string) string {
	return fmt.Sprintf("%s:user:email:%s", u.prefix, email)
}

// GetUserByAccessToken retrieves user information using an access token
func (u *DefaultUser) GetUserByAccessToken(ctx context.Context, accessToken string) (interface{}, error) {
	// Get token information from tokenStore
	tokenData, exists := u.tokenStore.Get(u.tokenKey(accessToken))
	if !exists {
		return nil, fmt.Errorf("token not found")
	}

	// Parse token data to get user subject
	var tokenInfo map[string]interface{}
	var ok bool

	// Try to convert to map[string]interface{} directly
	if tokenInfo, ok = tokenData.(map[string]interface{}); !ok {
		// If direct conversion fails, try to handle other possible types
		switch v := tokenData.(type) {
		case map[interface{}]interface{}:
			// Convert map[interface{}]interface{} to map[string]interface{}
			tokenInfo = make(map[string]interface{})
			for key, val := range v {
				if keyStr, ok := key.(string); ok {
					tokenInfo[keyStr] = val
				}
			}
		default:
			// Try to convert using map[string]interface{} casting
			// This handles primitive.M and other MongoDB types
			if reflect.TypeOf(v).Kind() == reflect.Map {
				tokenInfo = make(map[string]interface{})
				rv := reflect.ValueOf(v)
				for _, key := range rv.MapKeys() {
					if keyStr, ok := key.Interface().(string); ok {
						tokenInfo[keyStr] = rv.MapIndex(key).Interface()
					}
				}
				if len(tokenInfo) == 0 {
					return nil, fmt.Errorf("invalid token data format: %T", tokenData)
				}
			} else {
				return nil, fmt.Errorf("invalid token data format: %T", tokenData)
			}
		}
	}

	subject, ok := tokenInfo["subject"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid subject in token")
	}

	// Get user by subject
	return u.GetUserBySubject(ctx, subject)
}

// GetUserBySubject retrieves user information using a subject identifier
func (u *DefaultUser) GetUserBySubject(ctx context.Context, subject string) (interface{}, error) {
	// Try cache first if available
	if u.cache != nil {
		if cached, ok := u.cache.Get(u.subjectCacheKey(subject)); ok {
			return cached, nil
		}
	}

	// Get user from database using the model
	m := model.Select(u.model)

	user, err := m.Get(model.QueryParam{
		Select: PublicUserFields,
		Wheres: []model.QueryWhere{
			{Column: "subject", Value: subject},
		},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get user by subject: %w", err)
	}

	if len(user) == 0 {
		return nil, fmt.Errorf("user not found")
	}

	userData := user[0]

	// Cache the result if cache is available
	if u.cache != nil {
		u.cache.Set(u.subjectCacheKey(subject), userData, 5*time.Minute)
	}

	return userData, nil
}

// ValidateUserScope validates if a user has access to requested scopes
func (u *DefaultUser) ValidateUserScope(ctx context.Context, userID string, scopes []string) (bool, error) {
	var user interface{}
	var err error

	// Try cache first if available
	if u.cache != nil {
		if cached, ok := u.cache.Get(u.cacheKey(userID)); ok {
			user = cached
		}
	}

	// If not in cache, get from database
	if user == nil {
		m := model.Select(u.model)
		user, err = m.Find(userID, model.QueryParam{
			Select: []interface{}{"scopes", "status"},
		})

		if err != nil {
			return false, fmt.Errorf("failed to get user: %w", err)
		}

		// Cache the result if cache is available
		if u.cache != nil {
			u.cache.Set(u.cacheKey(userID), user, 5*time.Minute)
		}
	}

	// Check if user data is valid
	if user == nil {
		return false, fmt.Errorf("user not found")
	}

	// Convert user to map for indexing
	var userMap map[string]interface{}
	switch v := user.(type) {
	case map[string]interface{}:
		userMap = v
	default:
		// Try to convert using reflection if it's a map-like type
		if reflect.TypeOf(v).Kind() == reflect.Map {
			userMap = make(map[string]interface{})
			rv := reflect.ValueOf(v)
			for _, key := range rv.MapKeys() {
				if keyStr, ok := key.Interface().(string); ok {
					userMap[keyStr] = rv.MapIndex(key).Interface()
				}
			}
		} else {
			return false, fmt.Errorf("invalid user data format")
		}
	}

	// Check if user is active
	if status, ok := userMap["status"].(string); ok && status != "active" {
		return false, fmt.Errorf("user is not active")
	}

	// Get user scopes
	userScopes, ok := userMap["scopes"].([]interface{})
	if !ok {
		// If no scopes defined, deny access
		return false, nil
	}

	// Convert user scopes to string slice
	userScopeStrings := make([]string, len(userScopes))
	for i, scope := range userScopes {
		if scopeStr, ok := scope.(string); ok {
			userScopeStrings[i] = scopeStr
		}
	}

	// Check if user has all requested scopes
	for _, requestedScope := range scopes {
		hasScope := false
		for _, userScope := range userScopeStrings {
			if userScope == requestedScope {
				hasScope = true
				break
			}
		}
		if !hasScope {
			return false, nil
		}
	}

	return true, nil
}

// // StoreToken stores a token in the token store with expiration time
// func (u *DefaultUser) StoreToken(accessToken string, tokenData map[string]interface{}, expiration time.Duration) error {
// 	return u.tokenStore.Set(u.tokenKey(accessToken), tokenData, expiration)
// }

// // RevokeToken revokes a token by removing it from the token store
// func (u *DefaultUser) RevokeToken(accessToken string) error {
// 	u.tokenStore.Del(u.tokenKey(accessToken))
// 	return nil
// }

// // TokenExists checks if a token exists in the token store
// func (u *DefaultUser) TokenExists(accessToken string) bool {
// 	_, exists := u.tokenStore.Get(u.tokenKey(accessToken))
// 	return exists
// }

// // GetTokenData retrieves token data from the token store
// func (u *DefaultUser) GetTokenData(accessToken string) (map[string]interface{}, error) {
// 	tokenData, exists := u.tokenStore.Get(u.tokenKey(accessToken))
// 	if !exists {
// 		return nil, fmt.Errorf("token not found")
// 	}

// 	// Try to convert to map[string]interface{} directly
// 	if tokenInfo, ok := tokenData.(map[string]interface{}); ok {
// 		return tokenInfo, nil
// 	}

// 	// If direct conversion fails, try to handle other possible types
// 	// This handles cases where MongoDB might return different types
// 	switch v := tokenData.(type) {
// 	case map[string]interface{}:
// 		return v, nil
// 	case map[interface{}]interface{}:
// 		// Convert map[interface{}]interface{} to map[string]interface{}
// 		result := make(map[string]interface{})
// 		for key, val := range v {
// 			if keyStr, ok := key.(string); ok {
// 				result[keyStr] = val
// 			}
// 		}
// 		return result, nil
// 	default:
// 		// Try to convert using map[string]interface{} casting
// 		// This handles primitive.M and other MongoDB types
// 		if reflect.TypeOf(v).Kind() == reflect.Map {
// 			result := make(map[string]interface{})
// 			rv := reflect.ValueOf(v)
// 			for _, key := range rv.MapKeys() {
// 				if keyStr, ok := key.Interface().(string); ok {
// 					result[keyStr] = rv.MapIndex(key).Interface()
// 				}
// 			}
// 			if len(result) > 0 {
// 				return result, nil
// 			}
// 		}
// 		return nil, fmt.Errorf("invalid token data format: %T", tokenData)
// 	}
// }

// CreateUser creates a new user in the database
func (u *DefaultUser) CreateUser(userData map[string]interface{}) (interface{}, error) {
	m := model.Select(u.model)
	userID, err := m.Create(userData)
	if err != nil {
		return nil, err
	}

	// Note: No need to cache newly created user data since it will be cached
	// when accessed for the first time through other methods

	return userID, nil
}

// UpdateUserLastLogin updates the user's last login timestamp
func (u *DefaultUser) UpdateUserLastLogin(userID interface{}) error {
	m := model.Select(u.model)
	err := m.Update(userID, map[string]interface{}{
		"last_login_at": time.Now(),
	})

	if err != nil {
		return err
	}

	// Clear cache for this user since data has changed
	if u.cache != nil {
		userIDStr := fmt.Sprintf("%v", userID)
		u.cache.Del(u.cacheKey(userIDStr))
	}

	return nil
}

// GetUserByUsername retrieves user by username
func (u *DefaultUser) GetUserByUsername(username string) (interface{}, error) {
	// Try cache first if available
	if u.cache != nil {
		if cached, ok := u.cache.Get(u.usernameCacheKey(username)); ok {
			return cached, nil
		}
	}

	m := model.Select(u.model)

	users, err := m.Get(model.QueryParam{
		Select: PublicUserFields,
		Wheres: []model.QueryWhere{
			{Column: "username", Value: username},
		},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get user by username: %w", err)
	}

	if len(users) == 0 {
		return nil, fmt.Errorf("user not found")
	}

	userData := users[0]

	// Cache the result if cache is available
	if u.cache != nil {
		u.cache.Set(u.usernameCacheKey(username), userData, 5*time.Minute)
	}

	return userData, nil
}

// GetUserByEmail retrieves user by email
func (u *DefaultUser) GetUserByEmail(email string) (interface{}, error) {
	// Try cache first if available
	if u.cache != nil {
		if cached, ok := u.cache.Get(u.emailCacheKey(email)); ok {
			return cached, nil
		}
	}

	m := model.Select(u.model)

	users, err := m.Get(model.QueryParam{
		Select: PublicUserFields,
		Wheres: []model.QueryWhere{
			{Column: "email", Value: email},
		},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	if len(users) == 0 {
		return nil, fmt.Errorf("user not found")
	}

	userData := users[0]

	// Cache the result if cache is available
	if u.cache != nil {
		u.cache.Set(u.emailCacheKey(email), userData, 5*time.Minute)
	}

	return userData, nil
}

// GenerateTOTPSecret generates a new TOTP secret for user
func (u *DefaultUser) GenerateTOTPSecret(ctx context.Context, userID string, issuer string, accountName string) (string, string, error) {
	// Generate a random 20-byte secret
	secret := make([]byte, 20)
	if _, err := rand.Read(secret); err != nil {
		return "", "", fmt.Errorf("failed to generate secret: %w", err)
	}

	// Encode secret as Base32
	secretBase32 := base32.StdEncoding.EncodeToString(secret)
	secretBase32 = strings.TrimRight(secretBase32, "=") // Remove padding

	// Set default values
	if issuer == "" {
		issuer = "YAO OAuth"
	}
	if accountName == "" {
		accountName = userID
	}

	// Generate QR code URL
	qrURL := u.generateQRCodeURL(secretBase32, issuer, accountName)

	return secretBase32, qrURL, nil
}

// EnableTwoFactor enables two-factor authentication for user
func (u *DefaultUser) EnableTwoFactor(ctx context.Context, userID string, secret string, code string) error {
	// Verify the provided code with the secret
	if !u.verifyTOTPWithSecret(secret, code, "SHA1", 6, 30) {
		return fmt.Errorf("invalid verification code")
	}

	// Generate recovery codes
	recoveryCodes, err := u.generateRecoveryCodesList()
	if err != nil {
		return fmt.Errorf("failed to generate recovery codes: %w", err)
	}

	// Update user record
	m := model.Select(u.model)
	now := time.Now()
	err = m.Update(userID, map[string]interface{}{
		"two_factor_enabled":          true,
		"two_factor_secret":           secret,
		"two_factor_recovery_codes":   recoveryCodes,
		"two_factor_enabled_at":       now,
		"two_factor_last_verified_at": now,
	})

	if err != nil {
		return fmt.Errorf("failed to enable two-factor authentication: %w", err)
	}

	// Clear user cache
	if u.cache != nil {
		u.cache.Del(u.cacheKey(userID))
	}

	return nil
}

// DisableTwoFactor disables two-factor authentication for user
func (u *DefaultUser) DisableTwoFactor(ctx context.Context, userID string, code string) error {
	// Get current user data
	m := model.Select(u.model)
	user, err := m.Find(userID, model.QueryParam{
		Select: []interface{}{"two_factor_secret", "two_factor_recovery_codes"},
	})
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	if user == nil {
		return fmt.Errorf("user not found")
	}

	// Verify code (either TOTP or recovery code)
	verified := false
	if secret, ok := user["two_factor_secret"].(string); ok && secret != "" {
		verified = u.verifyTOTPWithSecret(secret, code, "SHA1", 6, 30)
	}

	if !verified {
		// Try recovery code
		if recoveryCodes, ok := user["two_factor_recovery_codes"].([]interface{}); ok {
			for _, rc := range recoveryCodes {
				if rcStr, ok := rc.(string); ok && rcStr == code {
					verified = true
					break
				}
			}
		}
	}

	if !verified {
		return fmt.Errorf("invalid verification code")
	}

	// Disable two-factor authentication
	err = m.Update(userID, map[string]interface{}{
		"two_factor_enabled":          false,
		"two_factor_secret":           nil,
		"two_factor_recovery_codes":   nil,
		"two_factor_enabled_at":       nil,
		"two_factor_last_verified_at": nil,
	})

	if err != nil {
		return fmt.Errorf("failed to disable two-factor authentication: %w", err)
	}

	// Clear user cache
	if u.cache != nil {
		u.cache.Del(u.cacheKey(userID))
	}

	return nil
}

// VerifyTOTPCode verifies a TOTP code for user
func (u *DefaultUser) VerifyTOTPCode(ctx context.Context, userID string, code string) (bool, error) {
	// Get user data
	m := model.Select(u.model)
	user, err := m.Find(userID, model.QueryParam{
		Select: []interface{}{"two_factor_enabled", "two_factor_secret", "two_factor_algorithm", "two_factor_digits", "two_factor_period"},
	})
	if err != nil {
		return false, fmt.Errorf("failed to get user: %w", err)
	}

	if user == nil {
		return false, fmt.Errorf("user not found")
	}

	// Check if two-factor is enabled
	if enabled, ok := user["two_factor_enabled"].(bool); !ok || !enabled {
		return false, fmt.Errorf("two-factor authentication is not enabled")
	}

	// Get TOTP parameters
	secret, _ := user["two_factor_secret"].(string)
	algorithm, _ := user["two_factor_algorithm"].(string)
	digits, _ := user["two_factor_digits"].(int)
	period, _ := user["two_factor_period"].(int)

	// Set defaults
	if algorithm == "" {
		algorithm = "SHA1"
	}
	if digits == 0 {
		digits = 6
	}
	if period == 0 {
		period = 30
	}

	// Verify code
	verified := u.verifyTOTPWithSecret(secret, code, algorithm, digits, period)

	if verified {
		// Update last verified time
		m.Update(userID, map[string]interface{}{
			"two_factor_last_verified_at": time.Now(),
		})

		// Clear user cache
		if u.cache != nil {
			u.cache.Del(u.cacheKey(userID))
		}
	}

	return verified, nil
}

// GenerateRecoveryCodes generates new recovery codes for user
func (u *DefaultUser) GenerateRecoveryCodes(ctx context.Context, userID string) ([]string, error) {
	// Generate new recovery codes
	recoveryCodes, err := u.generateRecoveryCodesList()
	if err != nil {
		return nil, fmt.Errorf("failed to generate recovery codes: %w", err)
	}

	// Update user record
	m := model.Select(u.model)
	err = m.Update(userID, map[string]interface{}{
		"two_factor_recovery_codes": recoveryCodes,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to update recovery codes: %w", err)
	}

	// Clear user cache
	if u.cache != nil {
		u.cache.Del(u.cacheKey(userID))
	}

	// Convert to string slice for return
	result := make([]string, len(recoveryCodes))
	for i, code := range recoveryCodes {
		result[i] = code.(string)
	}

	return result, nil
}

// VerifyRecoveryCode verifies and consumes a recovery code
func (u *DefaultUser) VerifyRecoveryCode(ctx context.Context, userID string, code string) (bool, error) {
	// Get user data
	m := model.Select(u.model)
	user, err := m.Find(userID, model.QueryParam{
		Select: []interface{}{"two_factor_enabled", "two_factor_recovery_codes"},
	})
	if err != nil {
		return false, fmt.Errorf("failed to get user: %w", err)
	}

	if user == nil {
		return false, fmt.Errorf("user not found")
	}

	// Check if two-factor is enabled
	if enabled, ok := user["two_factor_enabled"].(bool); !ok || !enabled {
		return false, fmt.Errorf("two-factor authentication is not enabled")
	}

	// Get recovery codes
	recoveryCodes, ok := user["two_factor_recovery_codes"].([]interface{})
	if !ok {
		return false, fmt.Errorf("no recovery codes found")
	}

	// Find and remove the used code
	var newRecoveryCodes []interface{}
	found := false
	for _, rc := range recoveryCodes {
		if rcStr, ok := rc.(string); ok && rcStr == code {
			found = true
			// Don't add this code to the new list (consume it)
		} else {
			newRecoveryCodes = append(newRecoveryCodes, rc)
		}
	}

	if !found {
		return false, nil
	}

	// Update user record with remaining codes
	err = m.Update(userID, map[string]interface{}{
		"two_factor_recovery_codes":   newRecoveryCodes,
		"two_factor_last_verified_at": time.Now(),
	})

	if err != nil {
		return false, fmt.Errorf("failed to update recovery codes: %w", err)
	}

	// Clear user cache
	if u.cache != nil {
		u.cache.Del(u.cacheKey(userID))
	}

	return true, nil
}

// Helper methods for TOTP

// generateQRCodeURL generates a QR code URL for TOTP setup
func (u *DefaultUser) generateQRCodeURL(secret, issuer, accountName string) string {
	// Build the otpauth URL
	params := url.Values{}
	params.Set("secret", secret)
	params.Set("issuer", issuer)
	params.Set("algorithm", "SHA1")
	params.Set("digits", "6")
	params.Set("period", "30")

	label := fmt.Sprintf("%s:%s", issuer, accountName)
	qrURL := fmt.Sprintf("otpauth://totp/%s?%s", url.QueryEscape(label), params.Encode())

	return qrURL
}

// generateRecoveryCodesList generates a list of recovery codes
func (u *DefaultUser) generateRecoveryCodesList() ([]interface{}, error) {
	codes := make([]interface{}, 10) // Generate 10 recovery codes

	for i := 0; i < 10; i++ {
		// Generate 8-character recovery code
		code := make([]byte, 8)
		if _, err := rand.Read(code); err != nil {
			return nil, err
		}

		// Convert to hex string
		codeStr := fmt.Sprintf("%x", code)
		codes[i] = codeStr
	}

	return codes, nil
}

// verifyTOTPWithSecret verifies a TOTP code with given parameters
func (u *DefaultUser) verifyTOTPWithSecret(secret, code, algorithm string, digits, period int) bool {
	// Decode secret
	secretBytes, err := base32.StdEncoding.DecodeString(secret)
	if err != nil {
		return false
	}

	// Get current time
	now := time.Now().Unix()

	// Check current time window and previous/next windows for clock skew
	for i := -1; i <= 1; i++ {
		timeCounter := (now + int64(i*period)) / int64(period)
		expectedCode := u.generateTOTPCode(secretBytes, timeCounter, algorithm, digits)

		if expectedCode == code {
			return true
		}
	}

	return false
}

// generateTOTPCode generates a TOTP code
func (u *DefaultUser) generateTOTPCode(secret []byte, timeCounter int64, algorithm string, digits int) string {
	// Convert time counter to byte array
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(timeCounter))

	// Choose hash algorithm
	var h hash.Hash
	switch algorithm {
	case "SHA256":
		h = sha256.New()
	case "SHA512":
		h = sha512.New()
	default:
		h = sha1.New()
	}

	// HMAC
	for i := 0; i < len(secret); i++ {
		h.Write([]byte{secret[i] ^ 0x36})
	}
	for i := len(secret); i < h.BlockSize(); i++ {
		h.Write([]byte{0x36})
	}
	h.Write(buf)
	innerHash := h.Sum(nil)

	h.Reset()
	for i := 0; i < len(secret); i++ {
		h.Write([]byte{secret[i] ^ 0x5c})
	}
	for i := len(secret); i < h.BlockSize(); i++ {
		h.Write([]byte{0x5c})
	}
	h.Write(innerHash)
	hmacHash := h.Sum(nil)

	// Dynamic truncation
	offset := hmacHash[len(hmacHash)-1] & 0x0f
	binCode := binary.BigEndian.Uint32(hmacHash[offset:offset+4]) & 0x7fffffff

	// Generate digits
	code := binCode % uint32(math.Pow10(digits))

	return fmt.Sprintf("%0*d", digits, code)
}

// GetUserForAuth retrieves user information for authentication purposes (internal use only)
// This method includes sensitive fields like password_hash and should not be exposed to external APIs
func (u *DefaultUser) GetUserForAuth(ctx context.Context, identifier string, identifierType string) (interface{}, error) {
	// Get user from database using the model
	m := model.Select(u.model)

	var column string
	switch identifierType {
	case "username":
		column = "username"
	case "email":
		column = "email"
	case "subject":
		column = "subject"
	default:
		return nil, fmt.Errorf("invalid identifier type: %s", identifierType)
	}

	user, err := m.Get(model.QueryParam{
		Select: AuthUserFields,
		Wheres: []model.QueryWhere{
			{Column: column, Value: identifier},
		},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get user for auth: %w", err)
	}

	if len(user) == 0 {
		return nil, fmt.Errorf("user not found")
	}

	return user[0], nil
}
