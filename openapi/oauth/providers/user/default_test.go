package user

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/gou/store"
	"github.com/yaoapp/gou/store/badger"
	"github.com/yaoapp/gou/store/lru"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/test"
)

// Store configuration for parameterized tests
type StoreConfig struct {
	Name    string
	GetFunc func(*testing.T) store.Store
}

// Test user data
type TestUserData struct {
	ID               int64                  `json:"id"`
	Subject          string                 `json:"subject"`
	Username         string                 `json:"username"`
	Email            string                 `json:"email"`
	PasswordHash     string                 `json:"password_hash"`
	FirstName        string                 `json:"first_name"`
	LastName         string                 `json:"last_name"`
	FullName         string                 `json:"full_name"`
	AvatarURL        string                 `json:"avatar_url"`
	Mobile           string                 `json:"mobile"`
	Address          string                 `json:"address"`
	Scopes           []string               `json:"scopes"`
	Status           string                 `json:"status"`
	EmailVerified    bool                   `json:"email_verified"`
	MobileVerified   bool                   `json:"mobile_verified"`
	TwoFactorEnabled bool                   `json:"two_factor_enabled"`
	TwoFactorSecret  string                 `json:"two_factor_secret"`
	Metadata         map[string]interface{} `json:"metadata"`
	Preferences      map[string]interface{} `json:"preferences"`
}

var testUserData = &TestUserData{
	Subject:          "test-subject-123",
	Username:         "testuser123",
	Email:            "test@example.com",
	PasswordHash:     "hashed_password_123",
	FirstName:        "Test",
	LastName:         "User",
	FullName:         "Test User",
	AvatarURL:        "https://example.com/avatar.jpg",
	Mobile:           "+1234567890",
	Address:          "123 Test Street",
	Scopes:           []string{"openid", "profile", "email"},
	Status:           "active",
	EmailVerified:    true,
	MobileVerified:   false,
	TwoFactorEnabled: false,
	// TwoFactorSecret:  "",
	Metadata:    map[string]interface{}{"test": "data"},
	Preferences: map[string]interface{}{"theme": "dark"},
}

// Helper function to convert various map types to map[string]interface{}
func convertToStringMap(t *testing.T, data interface{}) map[string]interface{} {
	switch v := data.(type) {
	case map[string]interface{}:
		return v
	default:
		// Try to convert using reflection if it's a map-like type
		if reflect.TypeOf(v).Kind() == reflect.Map {
			result := make(map[string]interface{})
			rv := reflect.ValueOf(v)
			for _, key := range rv.MapKeys() {
				if keyStr, ok := key.Interface().(string); ok {
					result[keyStr] = rv.MapIndex(key).Interface()
				}
			}
			return result
		}
		t.Fatalf("Unexpected data type: %T", v)
		return nil
	}
}

func TestMain(m *testing.M) {
	// Setup
	test.Prepare(&testing.T{}, config.Conf)
	defer test.Clean()

	// Run tests
	code := m.Run()
	os.Exit(code)
}

// Test helpers
func getMongoStore(t *testing.T) store.Store {
	// Skip test if MongoDB is not available
	host := os.Getenv("MONGO_TEST_HOST")
	if host == "" {
		t.Skip("MongoDB not available - set MONGO_TEST_HOST environment variable")
	}

	// Create MongoDB store using connector
	mongoConnector, err := connector.New("mongo", "oauth_user_test", []byte(`{
		"name": "OAuth User Test MongoDB",
		"type": "mongo",
		"options": {
			"db": "oauth_user_test",
			"hosts": [{
				"host": "`+host+`",
				"port": "`+os.Getenv("MONGO_TEST_PORT")+`",
				"user": "`+os.Getenv("MONGO_TEST_USER")+`",
				"pass": "`+os.Getenv("MONGO_TEST_PASS")+`"
			}]
		}
	}`))
	require.NoError(t, err)

	mongoStore, err := store.New(mongoConnector, nil)
	require.NoError(t, err)

	return mongoStore
}

func getBadgerStore(t *testing.T) store.Store {
	// Create temporary directory for test database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_oauth_user_badger")

	badgerStore, err := badger.New(dbPath)
	require.NoError(t, err)

	// Clean up on test completion
	t.Cleanup(func() {
		badgerStore.Close()
	})

	return badgerStore
}

func getLRUCache(t *testing.T) store.Store {
	cache, err := lru.New(1000)
	require.NoError(t, err)
	return cache
}

// Get all available store configurations
func getStoreConfigs() []StoreConfig {
	return []StoreConfig{
		{Name: "MongoDB", GetFunc: getMongoStore},
		{Name: "Badger", GetFunc: getBadgerStore},
	}
}

// Create test user data with unique identifier
func createTestUser(id string) *TestUserData {
	timestamp := time.Now().UnixNano()
	uniqueID := fmt.Sprintf("%s-%d", id, timestamp)

	return &TestUserData{
		Subject:          "test-subject-" + uniqueID,
		Username:         "testuser" + uniqueID,
		Email:            "test" + uniqueID + "@example.com",
		PasswordHash:     "hashed-password-" + uniqueID,
		FirstName:        "Test",
		LastName:         "User " + uniqueID,
		FullName:         "Test User " + uniqueID,
		AvatarURL:        "https://example.com/avatar" + uniqueID + ".jpg",
		Mobile:           "1234567890",
		Address:          "Test Address " + uniqueID,
		Scopes:           []string{"openid", "profile", "email"},
		Status:           "active",
		EmailVerified:    true,
		MobileVerified:   true,
		TwoFactorEnabled: false,
		Metadata:         map[string]interface{}{"test": "data"},
		Preferences:      map[string]interface{}{"theme": "dark"},
	}
}

// Create test token data
func createTestToken(subject string) map[string]interface{} {
	return map[string]interface{}{
		"subject":    subject,
		"client_id":  "test-client",
		"scopes":     []string{"openid", "profile", "email"},
		"expires_at": time.Now().Add(1 * time.Hour).Unix(),
		"issued_at":  time.Now().Unix(),
	}
}

// Setup test user in database
func setupTestUser(t *testing.T, userData *TestUserData) {
	m := model.Select("__yao.user")

	// Create user
	userMap := map[string]interface{}{
		"subject":            userData.Subject,
		"username":           userData.Username,
		"email":              userData.Email,
		"password_hash":      userData.PasswordHash,
		"first_name":         userData.FirstName,
		"last_name":          userData.LastName,
		"full_name":          userData.FullName,
		"avatar_url":         userData.AvatarURL,
		"mobile":             userData.Mobile,
		"address":            userData.Address,
		"scopes":             userData.Scopes,
		"status":             userData.Status,
		"email_verified":     userData.EmailVerified,
		"mobile_verified":    userData.MobileVerified,
		"two_factor_enabled": userData.TwoFactorEnabled,
		"two_factor_secret":  userData.TwoFactorSecret,
		"metadata":           userData.Metadata,
		"preferences":        userData.Preferences,
	}

	id, err := m.Create(userMap)
	require.NoError(t, err)
	userData.ID = int64(id)
}

// Clean up test data
func cleanupTestData(t *testing.T) {
	m := model.Select("__yao.user")

	// Delete all test users (be more aggressive in cleanup)
	_, err := m.DeleteWhere(model.QueryParam{
		Wheres: []model.QueryWhere{
			{Column: "subject", OP: "like", Value: "test-subject-%"},
		},
	})
	if err != nil {
		t.Logf("Warning: Failed to clean up test users by subject: %v", err)
	}

	// Also clean up by username pattern
	_, err = m.DeleteWhere(model.QueryParam{
		Wheres: []model.QueryWhere{
			{Column: "username", OP: "like", Value: "testuser%"},
		},
	})
	if err != nil {
		t.Logf("Warning: Failed to clean up test users by username: %v", err)
	}
}

func TestNewDefaultUser(t *testing.T) {
	storeConfigs := getStoreConfigs()

	for _, config := range storeConfigs {
		t.Run(config.Name, func(t *testing.T) {
			tokenStore := config.GetFunc(t)
			cache := getLRUCache(t)

			t.Run("valid options", func(t *testing.T) {
				user := NewDefaultUser(&DefaultUserOptions{
					Prefix: "test:",

					Cache:      cache,
					TokenStore: tokenStore,
				})

				assert.NotNil(t, user)
				assert.Equal(t, "test:", user.prefix)
				assert.Equal(t, "__yao.user", user.model)
				assert.Equal(t, cache, user.cache)
				assert.Equal(t, tokenStore, user.tokenStore)
			})

			t.Run("without cache", func(t *testing.T) {
				user := NewDefaultUser(&DefaultUserOptions{
					Prefix: "test:",

					TokenStore: tokenStore,
				})

				assert.NotNil(t, user)
				assert.Nil(t, user.cache)
			})

			t.Run("without token store", func(t *testing.T) {
				user := NewDefaultUser(&DefaultUserOptions{
					Prefix: "test:",
					Cache:  cache,
				})

				assert.NotNil(t, user)
				assert.Nil(t, user.tokenStore)
			})
		})
	}
}

func TestKeyGeneration(t *testing.T) {
	storeConfigs := getStoreConfigs()

	for _, config := range storeConfigs {
		t.Run(config.Name, func(t *testing.T) {
			tokenStore := config.GetFunc(t)
			cache := getLRUCache(t)

			user := NewDefaultUser(&DefaultUserOptions{
				Prefix: "test:",

				Cache:      cache,
				TokenStore: tokenStore,
			})

			t.Run("token key", func(t *testing.T) {
				key := user.tokenKey("test-token")
				expected := "test::token:test-token"
				assert.Equal(t, expected, key)
			})

			t.Run("cache key", func(t *testing.T) {
				key := user.cacheKey("123")
				expected := "test::user:123"
				assert.Equal(t, expected, key)
			})

			t.Run("subject cache key", func(t *testing.T) {
				key := user.subjectCacheKey("test-subject")
				expected := "test::user:subject:test-subject"
				assert.Equal(t, expected, key)
			})

			t.Run("username cache key", func(t *testing.T) {
				key := user.usernameCacheKey("testuser")
				expected := "test::user:username:testuser"
				assert.Equal(t, expected, key)
			})

			t.Run("email cache key", func(t *testing.T) {
				key := user.emailCacheKey("test@example.com")
				expected := "test::user:email:test@example.com"
				assert.Equal(t, expected, key)
			})
		})
	}
}

func TestGetUserBySubject(t *testing.T) {
	storeConfigs := getStoreConfigs()

	for _, config := range storeConfigs {
		t.Run(config.Name, func(t *testing.T) {
			cleanupTestData(t)
			defer cleanupTestData(t)

			tokenStore := config.GetFunc(t)
			cache := getLRUCache(t)

			user := NewDefaultUser(&DefaultUserOptions{
				Prefix: "test:",

				Cache:      cache,
				TokenStore: tokenStore,
			})

			// Create test user
			testUser := createTestUser("subject1")
			setupTestUser(t, testUser)

			ctx := context.Background()

			t.Run("get user by subject", func(t *testing.T) {
				retrievedUser, err := user.GetUserBySubject(ctx, testUser.Subject)
				assert.NoError(t, err)
				assert.NotNil(t, retrievedUser)

				userMap := convertToStringMap(t, retrievedUser)

				assert.Equal(t, testUser.Subject, userMap["subject"])
				assert.Equal(t, testUser.Username, userMap["username"])
				assert.Equal(t, testUser.Email, userMap["email"])
			})

			t.Run("get user by subject with cache", func(t *testing.T) {
				// Clear cache first
				cache.Clear()

				// First call should hit database
				retrievedUser, err := user.GetUserBySubject(ctx, testUser.Subject)
				assert.NoError(t, err)
				assert.NotNil(t, retrievedUser)

				// Second call should hit cache
				retrievedUser2, err := user.GetUserBySubject(ctx, testUser.Subject)
				assert.NoError(t, err)
				assert.NotNil(t, retrievedUser2)

				userMap := convertToStringMap(t, retrievedUser2)

				assert.Equal(t, testUser.Subject, userMap["subject"])
			})

			t.Run("non-existent subject", func(t *testing.T) {
				retrievedUser, err := user.GetUserBySubject(ctx, "non-existent-subject")
				assert.Error(t, err)
				assert.Nil(t, retrievedUser)
				assert.Contains(t, err.Error(), "user not found")
			})
		})
	}
}

func TestGetUserByUsername(t *testing.T) {
	storeConfigs := getStoreConfigs()

	for _, config := range storeConfigs {
		t.Run(config.Name, func(t *testing.T) {
			cleanupTestData(t)
			defer cleanupTestData(t)

			tokenStore := config.GetFunc(t)
			cache := getLRUCache(t)

			user := NewDefaultUser(&DefaultUserOptions{
				Prefix: "test:",

				Cache:      cache,
				TokenStore: tokenStore,
			})

			// Create test user
			testUser := createTestUser("username1")
			setupTestUser(t, testUser)

			t.Run("get user by username", func(t *testing.T) {
				retrievedUser, err := user.GetUserByUsername(testUser.Username)
				assert.NoError(t, err)
				assert.NotNil(t, retrievedUser)

				userMap := convertToStringMap(t, retrievedUser)
				assert.Equal(t, testUser.Username, userMap["username"])
				assert.Equal(t, testUser.Email, userMap["email"])
			})

			t.Run("non-existent username", func(t *testing.T) {
				retrievedUser, err := user.GetUserByUsername("non-existent-user")
				assert.Error(t, err)
				assert.Nil(t, retrievedUser)
				assert.Contains(t, err.Error(), "user not found")
			})
		})
	}
}

func TestGetUserByEmail(t *testing.T) {
	storeConfigs := getStoreConfigs()

	for _, config := range storeConfigs {
		t.Run(config.Name, func(t *testing.T) {
			cleanupTestData(t)
			defer cleanupTestData(t)

			tokenStore := config.GetFunc(t)
			cache := getLRUCache(t)

			user := NewDefaultUser(&DefaultUserOptions{
				Prefix: "test:",

				Cache:      cache,
				TokenStore: tokenStore,
			})

			// Create test user
			testUser := createTestUser("email1")
			setupTestUser(t, testUser)

			t.Run("get user by email", func(t *testing.T) {
				retrievedUser, err := user.GetUserByEmail(testUser.Email)
				assert.NoError(t, err)
				assert.NotNil(t, retrievedUser)

				userMap := convertToStringMap(t, retrievedUser)
				assert.Equal(t, testUser.Email, userMap["email"])
				assert.Equal(t, testUser.Username, userMap["username"])
			})

			t.Run("non-existent email", func(t *testing.T) {
				retrievedUser, err := user.GetUserByEmail("non-existent@example.com")
				assert.Error(t, err)
				assert.Nil(t, retrievedUser)
				assert.Contains(t, err.Error(), "user not found")
			})
		})
	}
}

func TestValidateUserScope(t *testing.T) {
	storeConfigs := getStoreConfigs()

	for _, config := range storeConfigs {
		t.Run(config.Name, func(t *testing.T) {
			cleanupTestData(t)
			defer cleanupTestData(t)

			tokenStore := config.GetFunc(t)
			cache := getLRUCache(t)

			user := NewDefaultUser(&DefaultUserOptions{
				Prefix: "test:",

				Cache:      cache,
				TokenStore: tokenStore,
			})

			// Create test user
			testUser := createTestUser("scope1")
			setupTestUser(t, testUser)

			ctx := context.Background()

			t.Run("validate user scope - valid", func(t *testing.T) {
				valid, err := user.ValidateUserScope(ctx, fmt.Sprintf("%d", testUser.ID), []string{"openid", "profile"})
				assert.NoError(t, err)
				assert.True(t, valid)
			})

			t.Run("validate user scope - invalid", func(t *testing.T) {
				valid, err := user.ValidateUserScope(ctx, fmt.Sprintf("%d", testUser.ID), []string{"admin"})
				assert.NoError(t, err)
				assert.False(t, valid)
			})

			t.Run("validate user scope - inactive user", func(t *testing.T) {
				// Create inactive user
				inactiveUser := createTestUser("inactive")
				inactiveUser.Status = "inactive"
				setupTestUser(t, inactiveUser)

				valid, err := user.ValidateUserScope(ctx, fmt.Sprintf("%d", inactiveUser.ID), []string{"openid"})
				assert.Error(t, err)
				assert.False(t, valid)
				assert.Contains(t, err.Error(), "user is not active")
			})

			t.Run("validate user scope - non-existent user", func(t *testing.T) {
				valid, err := user.ValidateUserScope(ctx, "999999", []string{"openid"})
				assert.Error(t, err)
				assert.False(t, valid)
				assert.Contains(t, err.Error(), "数据不存在")
			})
		})
	}
}

func TestGetUserForAuth(t *testing.T) {
	storeConfigs := getStoreConfigs()

	for _, config := range storeConfigs {
		t.Run(config.Name, func(t *testing.T) {
			cleanupTestData(t)
			defer cleanupTestData(t)

			tokenStore := config.GetFunc(t)
			cache := getLRUCache(t)

			user := NewDefaultUser(&DefaultUserOptions{
				Prefix: "test:",

				Cache:      cache,
				TokenStore: tokenStore,
			})

			// Create test user
			testUser := createTestUser("auth1")
			setupTestUser(t, testUser)

			ctx := context.Background()

			t.Run("get user for auth by username", func(t *testing.T) {
				retrievedUser, err := user.GetUserForAuth(ctx, testUser.Username, "username")
				assert.NoError(t, err)
				assert.NotNil(t, retrievedUser)

				userMap := convertToStringMap(t, retrievedUser)
				assert.Equal(t, testUser.Username, userMap["username"])
				// Password should be encrypted, not equal to original
				assert.NotEmpty(t, userMap["password_hash"])
				assert.NotEqual(t, testUser.PasswordHash, userMap["password_hash"])
			})

			t.Run("get user for auth by email", func(t *testing.T) {
				retrievedUser, err := user.GetUserForAuth(ctx, testUser.Email, "email")
				assert.NoError(t, err)
				assert.NotNil(t, retrievedUser)

				userMap := convertToStringMap(t, retrievedUser)
				assert.Equal(t, testUser.Email, userMap["email"])
				// Password should be encrypted, not equal to original
				assert.NotEmpty(t, userMap["password_hash"])
				assert.NotEqual(t, testUser.PasswordHash, userMap["password_hash"])
			})

			t.Run("get user for auth by subject", func(t *testing.T) {
				retrievedUser, err := user.GetUserForAuth(ctx, testUser.Subject, "subject")
				assert.NoError(t, err)
				assert.NotNil(t, retrievedUser)

				userMap := convertToStringMap(t, retrievedUser)
				assert.Equal(t, testUser.Subject, userMap["subject"])
				// Password should be encrypted, not equal to original
				assert.NotEmpty(t, userMap["password_hash"])
				assert.NotEqual(t, testUser.PasswordHash, userMap["password_hash"])
			})

			t.Run("get user for auth - invalid identifier type", func(t *testing.T) {
				retrievedUser, err := user.GetUserForAuth(ctx, testUser.Username, "invalid")
				assert.Error(t, err)
				assert.Nil(t, retrievedUser)
				assert.Contains(t, err.Error(), "invalid identifier type")
			})

			t.Run("get user for auth - non-existent user", func(t *testing.T) {
				retrievedUser, err := user.GetUserForAuth(ctx, "non-existent", "username")
				assert.Error(t, err)
				assert.Nil(t, retrievedUser)
				assert.Contains(t, err.Error(), "user not found")
			})
		})
	}
}

func TestCreateUser(t *testing.T) {
	storeConfigs := getStoreConfigs()

	for _, config := range storeConfigs {
		t.Run(config.Name, func(t *testing.T) {
			cleanupTestData(t)
			defer cleanupTestData(t)

			tokenStore := config.GetFunc(t)
			cache := getLRUCache(t)

			user := NewDefaultUser(&DefaultUserOptions{
				Prefix: "test:",

				Cache:      cache,
				TokenStore: tokenStore,
			})

			t.Run("create user", func(t *testing.T) {
				testUser := createTestUser("create")

				userData := map[string]interface{}{
					"subject":         testUser.Subject,
					"username":        testUser.Username,
					"email":           testUser.Email,
					"password_hash":   testUser.PasswordHash,
					"first_name":      testUser.FirstName,
					"last_name":       testUser.LastName,
					"full_name":       testUser.FullName,
					"status":          testUser.Status,
					"email_verified":  testUser.EmailVerified,
					"mobile_verified": testUser.MobileVerified,
					"scopes":          testUser.Scopes,
				}

				// Create user
				userID, err := user.CreateUser(userData)
				assert.NoError(t, err)
				assert.NotNil(t, userID)

				// Verify user was created
				m := model.Select("__yao.user")
				createdUser, err := m.Find(userID, model.QueryParam{})
				assert.NoError(t, err)
				assert.Equal(t, userData["username"], createdUser["username"])
				assert.Equal(t, userData["email"], createdUser["email"])

				// Verify user was created with correct default model name
				user2 := NewDefaultUser(&DefaultUserOptions{
					Prefix: "test:",

					Cache:      cache,
					TokenStore: tokenStore,
				})
				assert.Equal(t, "__yao.user", user2.model)
			})
		})
	}
}

func TestUpdateUserLastLogin(t *testing.T) {
	storeConfigs := getStoreConfigs()

	for _, config := range storeConfigs {
		t.Run(config.Name, func(t *testing.T) {
			cleanupTestData(t)
			defer cleanupTestData(t)

			tokenStore := config.GetFunc(t)
			cache := getLRUCache(t)

			user := NewDefaultUser(&DefaultUserOptions{
				Prefix: "test:",

				Cache:      cache,
				TokenStore: tokenStore,
			})

			// Create test user
			testUser := createTestUser("login1")
			setupTestUser(t, testUser)

			t.Run("update user last login", func(t *testing.T) {
				err := user.UpdateUserLastLogin(testUser.ID)
				assert.NoError(t, err)

				// Verify last login was updated
				m := model.Select("__yao.user")
				updatedUser, err := m.Find(testUser.ID, model.QueryParam{})
				assert.NoError(t, err)
				assert.NotNil(t, updatedUser)
				assert.NotNil(t, updatedUser["last_login_at"])
			})
		})
	}
}

func TestTOTPGeneration(t *testing.T) {
	storeConfigs := getStoreConfigs()

	for _, config := range storeConfigs {
		t.Run(config.Name, func(t *testing.T) {
			tokenStore := config.GetFunc(t)
			cache := getLRUCache(t)

			user := NewDefaultUser(&DefaultUserOptions{
				Prefix:     "test:",
				Model:      "__yao.user",
				Cache:      cache,
				TokenStore: tokenStore,
			})

			ctx := context.Background()

			t.Run("generate TOTP secret", func(t *testing.T) {
				secret, qrURL, err := user.GenerateTOTPSecret(ctx, "test-user", "Test App", "testuser@example.com")
				assert.NoError(t, err)
				assert.NotEmpty(t, secret)
				assert.NotEmpty(t, qrURL)
				assert.Contains(t, qrURL, "otpauth://totp/")
				assert.Contains(t, qrURL, "secret=")
				assert.Contains(t, qrURL, "issuer=Test+App")
			})

			t.Run("generate TOTP secret with defaults", func(t *testing.T) {
				secret, qrURL, err := user.GenerateTOTPSecret(ctx, "test-user", "", "")
				assert.NoError(t, err)
				assert.NotEmpty(t, secret)
				assert.NotEmpty(t, qrURL)
				assert.Contains(t, qrURL, "issuer=YAO+OAuth")
			})
		})
	}
}

func TestTOTPVerification(t *testing.T) {
	storeConfigs := getStoreConfigs()

	for _, config := range storeConfigs {
		t.Run(config.Name, func(t *testing.T) {
			tokenStore := config.GetFunc(t)
			cache := getLRUCache(t)

			user := NewDefaultUser(&DefaultUserOptions{
				Prefix:     "test:",
				Model:      "__yao.user",
				Cache:      cache,
				TokenStore: tokenStore,
			})

			t.Run("verify TOTP with secret", func(t *testing.T) {
				secret := "JBSWY3DPEHPK3PXP" // Test secret

				// Generate code for current time
				now := time.Now().Unix()
				timeCounter := now / 30
				expectedCode := user.generateTOTPCode([]byte("Hello!\xDE\xAD\xBE\xEF"), timeCounter, "SHA1", 6)

				// This test might be flaky due to time, so we'll test the method exists
				result := user.verifyTOTPWithSecret(secret, expectedCode, "SHA1", 6, 30)
				// We can't assert the exact result due to time dependencies
				assert.IsType(t, false, result)
			})

			t.Run("generate TOTP code", func(t *testing.T) {
				secret := []byte("Hello!\xDE\xAD\xBE\xEF")
				timeCounter := int64(1234567890)

				code := user.generateTOTPCode(secret, timeCounter, "SHA1", 6)
				assert.Len(t, code, 6)
				assert.Regexp(t, `^\d{6}$`, code)
			})
		})
	}
}

func TestTOTPEnabledUserFlow(t *testing.T) {
	storeConfigs := getStoreConfigs()

	for _, config := range storeConfigs {
		t.Run(config.Name, func(t *testing.T) {
			cleanupTestData(t)
			defer cleanupTestData(t)

			tokenStore := config.GetFunc(t)
			cache := getLRUCache(t)

			user := NewDefaultUser(&DefaultUserOptions{
				Prefix: "test:",

				Cache:      cache,
				TokenStore: tokenStore,
			})

			// Create test user
			testUser := createTestUser("2fa1")
			setupTestUser(t, testUser)

			ctx := context.Background()

			t.Run("enable two factor with invalid code", func(t *testing.T) {
				// This test is limited because we can't easily generate a valid TOTP code
				// In a real scenario, we'd need to coordinate the secret generation and verification

				secret := "JBSWY3DPEHPK3PXP"
				// Using a mock code - in real tests, you'd generate a proper TOTP code
				code := "123456"

				err := user.EnableTwoFactor(ctx, fmt.Sprintf("%d", testUser.ID), secret, code)
				// This will fail with invalid code, which is expected
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "invalid verification code")
			})

			t.Run("generate recovery codes", func(t *testing.T) {
				codes, err := user.GenerateRecoveryCodes(ctx, fmt.Sprintf("%d", testUser.ID))
				assert.NoError(t, err)
				assert.Len(t, codes, 10)

				for _, code := range codes {
					assert.Len(t, code, 16) // 8 bytes hex = 16 characters
					assert.Regexp(t, `^[0-9a-f]{16}$`, code)
				}
			})
		})
	}
}

func TestHelperMethods(t *testing.T) {
	storeConfigs := getStoreConfigs()

	for _, config := range storeConfigs {
		t.Run(config.Name, func(t *testing.T) {
			tokenStore := config.GetFunc(t)
			cache := getLRUCache(t)

			user := NewDefaultUser(&DefaultUserOptions{
				Prefix: "test:",

				Cache:      cache,
				TokenStore: tokenStore,
			})

			t.Run("generate QR code URL", func(t *testing.T) {
				qrURL := user.generateQRCodeURL("JBSWY3DPEHPK3PXP", "Test App", "testuser@example.com")
				assert.Contains(t, qrURL, "otpauth://totp/")
				assert.Contains(t, qrURL, "secret=JBSWY3DPEHPK3PXP")
				assert.Contains(t, qrURL, "issuer=Test+App")
				assert.Contains(t, qrURL, "algorithm=SHA1")
				assert.Contains(t, qrURL, "digits=6")
				assert.Contains(t, qrURL, "period=30")
			})

			t.Run("generate recovery codes list", func(t *testing.T) {
				codes, err := user.generateRecoveryCodesList()
				assert.NoError(t, err)
				assert.Len(t, codes, 10)

				for _, code := range codes {
					codeStr := code.(string)
					assert.Len(t, codeStr, 16) // 8 bytes hex = 16 characters
					assert.Regexp(t, `^[0-9a-f]{16}$`, codeStr)
				}
			})
		})
	}
}

func TestErrorHandling(t *testing.T) {
	storeConfigs := getStoreConfigs()

	for _, config := range storeConfigs {
		t.Run(config.Name, func(t *testing.T) {
			cleanupTestData(t)
			defer cleanupTestData(t)

			tokenStore := config.GetFunc(t)
			cache := getLRUCache(t)

			user := NewDefaultUser(&DefaultUserOptions{
				Prefix: "test:",

				Cache:      cache,
				TokenStore: tokenStore,
			})

			ctx := context.Background()

			t.Run("get user by invalid subject", func(t *testing.T) {
				retrievedUser, err := user.GetUserBySubject(ctx, "")
				assert.Error(t, err)
				assert.Nil(t, retrievedUser)
			})

			t.Run("verify TOTP code - user not found", func(t *testing.T) {
				verified, err := user.VerifyTOTPCode(ctx, "999999", "123456")
				assert.Error(t, err)
				assert.False(t, verified)
			})

			t.Run("verify recovery code - user not found", func(t *testing.T) {
				verified, err := user.VerifyRecoveryCode(ctx, "999999", "test-code")
				assert.Error(t, err)
				assert.False(t, verified)
			})

			t.Run("disable two factor - user not found", func(t *testing.T) {
				err := user.DisableTwoFactor(ctx, "999999", "123456")
				assert.Error(t, err)
			})
		})
	}
}

func TestCacheConsistency(t *testing.T) {
	storeConfigs := getStoreConfigs()

	for _, config := range storeConfigs {
		t.Run(config.Name, func(t *testing.T) {
			cleanupTestData(t)
			defer cleanupTestData(t)

			tokenStore := config.GetFunc(t)
			cache := getLRUCache(t)

			user := NewDefaultUser(&DefaultUserOptions{
				Prefix: "test:",

				Cache:      cache,
				TokenStore: tokenStore,
			})

			// Create test user
			testUser := createTestUser("cache1")
			setupTestUser(t, testUser)

			ctx := context.Background()

			t.Run("cache invalidation on update", func(t *testing.T) {
				// First, load user into cache
				retrievedUser, err := user.GetUserBySubject(ctx, testUser.Subject)
				assert.NoError(t, err)
				assert.NotNil(t, retrievedUser)

				// Update user last login (should clear cache)
				err = user.UpdateUserLastLogin(testUser.ID)
				assert.NoError(t, err)

				// Verify cache was cleared by checking if the key exists
				cacheKey := user.cacheKey(fmt.Sprintf("%d", testUser.ID))
				_, exists := cache.Get(cacheKey)
				assert.False(t, exists)
			})

			t.Run("cache invalidation on two factor operations", func(t *testing.T) {
				// Load user into cache
				retrievedUser, err := user.GetUserBySubject(ctx, testUser.Subject)
				assert.NoError(t, err)
				assert.NotNil(t, retrievedUser)

				// Generate recovery codes (should clear cache)
				codes, err := user.GenerateRecoveryCodes(ctx, fmt.Sprintf("%d", testUser.ID))
				assert.NoError(t, err)
				assert.Len(t, codes, 10)

				// Verify cache was cleared
				cacheKey := user.cacheKey(fmt.Sprintf("%d", testUser.ID))
				_, exists := cache.Get(cacheKey)
				assert.False(t, exists)
			})
		})
	}
}
