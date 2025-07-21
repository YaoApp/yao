package oauth

import (
	"context"
	"os"
	"path/filepath"
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
	"github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/test"
)

// =============================================================================
// Test Environment Setup
// =============================================================================

// **IMPORTANT**
// Before running any OAuth tests, you must source the env.local.sh file.
// $YAO_SOURCE_ROOT is the root directory of the Yao source code.
// source $YAO_SOURCE_ROOT/env.local.sh

// Test certificate paths - created once and reused across tests
var (
	testCertPath string
	testKeyPath  string
)

// Store configuration for parameterized tests
type StoreConfig struct {
	Name    string
	GetFunc func(*testing.T) store.Store
}

// TestClient represents a test OAuth client
// AI: Use this standard test client structure for all OAuth functionality tests
type TestClient struct {
	ClientID      string
	ClientSecret  string
	ClientName    string
	ClientType    string
	RedirectURIs  []string
	GrantTypes    []string
	ResponseTypes []string
	Scope         string
	Description   string // For test identification
}

// TestUser represents a test user
// AI: Use this standard test user structure for all OAuth functionality tests
type TestUser struct {
	ID               int64
	Subject          string
	Username         string
	Email            string
	PasswordHash     string
	FirstName        string
	LastName         string
	FullName         string
	Scopes           []string
	Status           string
	EmailVerified    bool
	MobileVerified   bool
	TwoFactorEnabled bool
	Description      string // For test identification
}

// OAuth Test Environment Setup
// AI: This is the foundational environment setup for all OAuth unit tests.
// Use this environment setup function directly when building other OAuth tests.
// It provides pre-configured stores, clients, and users for comprehensive testing.

// Standard Test Data Sets
// AI: All subsequent functionality tests should use these pre-defined test data sets.
// These provide consistent, well-structured test data for OAuth operations.

// Test clients - 3 different types for comprehensive testing
var testClients = []*TestClient{
	{
		ClientID:      "test-confidential-client",
		ClientSecret:  "confidential-secret-12345",
		ClientName:    "Test Confidential Client",
		ClientType:    types.ClientTypeConfidential,
		RedirectURIs:  []string{"https://localhost/callback"},
		GrantTypes:    []string{types.GrantTypeAuthorizationCode, types.GrantTypeRefreshToken},
		ResponseTypes: []string{types.ResponseTypeCode},
		Scope:         "openid profile email",
		Description:   "Confidential client for authorization code flow",
	},
	{
		ClientID:      "test-public-client",
		ClientSecret:  "", // Public clients don't have secrets
		ClientName:    "Test Public Client",
		ClientType:    types.ClientTypePublic,
		RedirectURIs:  []string{"https://localhost/callback"},
		GrantTypes:    []string{types.GrantTypeAuthorizationCode},
		ResponseTypes: []string{types.ResponseTypeCode},
		Scope:         "openid profile",
		Description:   "Public client for mobile/SPA applications",
	},
	{
		ClientID:      "test-credentials-client",
		ClientSecret:  "credentials-secret-67890",
		ClientName:    "Test Client Credentials Client",
		ClientType:    types.ClientTypeConfidential,
		RedirectURIs:  []string{"https://localhost/callback"},
		GrantTypes:    []string{types.GrantTypeClientCredentials},
		ResponseTypes: []string{types.ResponseTypeCode},
		Scope:         "api:read api:write",
		Description:   "Client for server-to-server authentication",
	},
}

// Test users - 10 users with different characteristics
var testUsers = []*TestUser{
	{
		Subject:          "user-admin-001",
		Username:         "admin",
		Email:            "admin@example.com",
		PasswordHash:     "admin-hash-001",
		FirstName:        "Admin",
		LastName:         "User",
		FullName:         "Admin User",
		Scopes:           []string{"openid", "profile", "email", "admin"},
		Status:           "active",
		EmailVerified:    true,
		MobileVerified:   true,
		TwoFactorEnabled: true,
		Description:      "Administrator user with full privileges",
	},
	{
		Subject:          "user-regular-001",
		Username:         "john.doe",
		Email:            "john.doe@example.com",
		PasswordHash:     "john-hash-001",
		FirstName:        "John",
		LastName:         "Doe",
		FullName:         "John Doe",
		Scopes:           []string{"openid", "profile", "email"},
		Status:           "active",
		EmailVerified:    true,
		MobileVerified:   false,
		TwoFactorEnabled: false,
		Description:      "Regular user with basic privileges",
	},
	{
		Subject:          "user-regular-002",
		Username:         "jane.smith",
		Email:            "jane.smith@example.com",
		PasswordHash:     "jane-hash-001",
		FirstName:        "Jane",
		LastName:         "Smith",
		FullName:         "Jane Smith",
		Scopes:           []string{"openid", "profile", "email"},
		Status:           "active",
		EmailVerified:    true,
		MobileVerified:   true,
		TwoFactorEnabled: false,
		Description:      "Regular user with verified mobile",
	},
	{
		Subject:          "user-pending-001",
		Username:         "pending.user",
		Email:            "pending@example.com",
		PasswordHash:     "pending-hash-001",
		FirstName:        "Pending",
		LastName:         "User",
		FullName:         "Pending User",
		Scopes:           []string{"openid", "profile"},
		Status:           "pending",
		EmailVerified:    false,
		MobileVerified:   false,
		TwoFactorEnabled: false,
		Description:      "User with pending verification",
	},
	{
		Subject:          "user-inactive-001",
		Username:         "inactive.user",
		Email:            "inactive@example.com",
		PasswordHash:     "inactive-hash-001",
		FirstName:        "Inactive",
		LastName:         "User",
		FullName:         "Inactive User",
		Scopes:           []string{"openid"},
		Status:           "inactive",
		EmailVerified:    true,
		MobileVerified:   false,
		TwoFactorEnabled: false,
		Description:      "Inactive user account",
	},
	{
		Subject:          "user-limited-001",
		Username:         "limited.user",
		Email:            "limited@example.com",
		PasswordHash:     "limited-hash-001",
		FirstName:        "Limited",
		LastName:         "User",
		FullName:         "Limited User",
		Scopes:           []string{"openid"},
		Status:           "active",
		EmailVerified:    true,
		MobileVerified:   false,
		TwoFactorEnabled: false,
		Description:      "User with limited scope access",
	},
	{
		Subject:          "user-2fa-001",
		Username:         "secure.user",
		Email:            "secure@example.com",
		PasswordHash:     "secure-hash-001",
		FirstName:        "Secure",
		LastName:         "User",
		FullName:         "Secure User",
		Scopes:           []string{"openid", "profile", "email"},
		Status:           "active",
		EmailVerified:    true,
		MobileVerified:   true,
		TwoFactorEnabled: true,
		Description:      "Security-focused user with 2FA enabled",
	},
	{
		Subject:          "user-api-001",
		Username:         "api.user",
		Email:            "api@example.com",
		PasswordHash:     "api-hash-001",
		FirstName:        "API",
		LastName:         "User",
		FullName:         "API User",
		Scopes:           []string{"api:read", "api:write"},
		Status:           "active",
		EmailVerified:    true,
		MobileVerified:   false,
		TwoFactorEnabled: false,
		Description:      "User for API access testing",
	},
	{
		Subject:          "user-guest-001",
		Username:         "guest.user",
		Email:            "guest@example.com",
		PasswordHash:     "guest-hash-001",
		FirstName:        "Guest",
		LastName:         "User",
		FullName:         "Guest User",
		Scopes:           []string{"openid"},
		Status:           "active",
		EmailVerified:    false,
		MobileVerified:   false,
		TwoFactorEnabled: false,
		Description:      "Guest user with minimal access",
	},
	{
		Subject:          "user-test-001",
		Username:         "test.user",
		Email:            "test@example.com",
		PasswordHash:     "test-hash-001",
		FirstName:        "Test",
		LastName:         "User",
		FullName:         "Test User",
		Scopes:           []string{"openid", "profile", "email", "test"},
		Status:           "active",
		EmailVerified:    true,
		MobileVerified:   true,
		TwoFactorEnabled: false,
		Description:      "General purpose test user",
	},
}

// setupOAuthTestEnvironment sets up the foundational environment for OAuth unit tests
// AI: This is the core environment setup function. Use this directly in other OAuth tests.
// It provides everything needed: stores, clients, users, and proper cleanup.
func setupOAuthTestEnvironment(t *testing.T) (*Service, store.Store, store.Store, func()) {
	// Initialize test environment
	test.Prepare(t, config.Conf)

	// Get store configurations
	storeConfigs := getStoreConfigs()

	// Use the first available store (prefer MongoDB, fallback to Badger)
	var mainStore store.Store
	var storeConfig StoreConfig

	for _, config := range storeConfigs {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Logf("Store %s not available: %v", config.Name, r)
				}
			}()

			testStore := config.GetFunc(t)
			if testStore != nil {
				mainStore = testStore
				storeConfig = config
				return
			}
		}()

		if mainStore != nil {
			break
		}
	}

	// Fallback to Badger if no other store is available
	if mainStore == nil {
		mainStore = getBadgerStore(t)
		storeConfig = StoreConfig{Name: "Badger", GetFunc: getBadgerStore}
	}

	// Create cache
	cache := getLRUCache(t)

	// Create test certificates once if not already created
	if testCertPath == "" || testKeyPath == "" {
		createTestCertificatesOnce(t)
	}

	// Create OAuth service configuration
	oauthConfig := &Config{
		Store: mainStore,
		Cache: cache,
		Signing: types.SigningConfig{
			SigningAlgorithm: "RS256",
			SigningCertPath:  testCertPath,
			SigningKeyPath:   testKeyPath,
		},
		Token: types.TokenConfig{
			AccessTokenLifetime:       time.Hour,
			RefreshTokenLifetime:      24 * time.Hour,
			AuthorizationCodeLifetime: 10 * time.Minute,
			DeviceCodeLifetime:        15 * time.Minute,
			AccessTokenFormat:         "jwt",
			RefreshTokenFormat:        "opaque",
		},
		Security: types.SecurityConfig{
			PKCECodeChallengeMethod: []string{"S256"},
			PKCECodeVerifierLength:  128,
			StateParameterLifetime:  10 * time.Minute,
			StateParameterLength:    32,
		},
		Client: types.ClientConfig{
			DefaultClientType:              types.ClientTypeConfidential,
			DefaultTokenEndpointAuthMethod: "client_secret_basic",
			DefaultGrantTypes:              []string{types.GrantTypeAuthorizationCode, types.GrantTypeRefreshToken},
			DefaultResponseTypes:           []string{types.ResponseTypeCode},
			ClientIDLength:                 32,
			ClientSecretLength:             64,
			DynamicRegistrationEnabled:     true,
			AllowedRedirectURISchemes:      []string{"https", "http"},
			AllowedRedirectURIHosts:        []string{"localhost", "127.0.0.1"},
		},
		Features: FeatureFlags{
			OAuth21Enabled:                   true,
			PKCEEnforced:                     true,
			RefreshTokenRotationEnabled:      true,
			DeviceFlowEnabled:                true,
			TokenExchangeEnabled:             true,
			PushedAuthorizationEnabled:       true,
			DynamicClientRegistrationEnabled: true,
			MCPComplianceEnabled:             true,
			ResourceParameterEnabled:         true,
			TokenBindingEnabled:              true,
			MTLSEnabled:                      false,
			DPoPEnabled:                      false,
			JWTIntrospectionEnabled:          true,
			TokenRevocationEnabled:           true,
			UserInfoJWTEnabled:               true,
		},
		IssuerURL: "https://oauth.test.example.com",
	}

	// Create OAuth service
	service, err := NewService(oauthConfig)
	require.NoError(t, err, "Failed to create OAuth service")
	require.NotNil(t, service, "OAuth service should not be nil")

	// Setup test data
	setupTestData(t, service)

	// Return cleanup function
	cleanup := func() {
		cleanupTestData(t, service)
		test.Clean()

		// Close stores if they support it
		if closer, ok := mainStore.(interface{ Close() error }); ok {
			closer.Close()
		}
		if closer, ok := cache.(interface{ Close() error }); ok {
			closer.Close()
		}
	}

	t.Logf("OAuth test environment initialized with %s store", storeConfig.Name)
	return service, mainStore, cache, cleanup
}

// setupTestData initializes the standard test data set
func setupTestData(t *testing.T, service *Service) {
	ctx := context.Background()

	// Clean up any existing test data first
	cleanupTestData(t, service)

	// Create test clients
	clientProvider := service.GetClientProvider()
	for i, testClient := range testClients {
		clientInfo := &types.ClientInfo{
			ClientID:                testClient.ClientID,
			ClientSecret:            testClient.ClientSecret,
			ClientName:              testClient.ClientName,
			ClientType:              testClient.ClientType,
			RedirectURIs:            testClient.RedirectURIs,
			GrantTypes:              testClient.GrantTypes,
			ResponseTypes:           testClient.ResponseTypes,
			Scope:                   testClient.Scope,
			ApplicationType:         types.ApplicationTypeWeb,
			TokenEndpointAuthMethod: "client_secret_basic",
		}

		// Set appropriate auth method for public clients
		if testClient.ClientType == types.ClientTypePublic {
			clientInfo.TokenEndpointAuthMethod = "none"
		}

		createdClient, err := clientProvider.CreateClient(ctx, clientInfo)
		require.NoError(t, err, "Failed to create test client %d: %s", i, testClient.Description)
		require.NotNil(t, createdClient, "Created client should not be nil")

		t.Logf("Created test client: %s (%s)", testClient.ClientID, testClient.Description)
	}

	// Create test users
	userProvider := service.GetUserProvider()
	for i, testUser := range testUsers {
		userData := map[string]interface{}{
			"subject":            testUser.Subject,
			"username":           testUser.Username,
			"email":              testUser.Email,
			"password_hash":      testUser.PasswordHash,
			"first_name":         testUser.FirstName,
			"last_name":          testUser.LastName,
			"full_name":          testUser.FullName,
			"scopes":             testUser.Scopes,
			"status":             testUser.Status,
			"email_verified":     testUser.EmailVerified,
			"mobile_verified":    testUser.MobileVerified,
			"two_factor_enabled": testUser.TwoFactorEnabled,
		}

		createdUserID, err := userProvider.CreateUser(userData)
		require.NoError(t, err, "Failed to create test user %d: %s", i, testUser.Description)
		require.NotNil(t, createdUserID, "Created user ID should not be nil")

		// Update the test user with the created ID
		if userID, ok := createdUserID.(int64); ok {
			testUser.ID = userID
		} else if userID, ok := createdUserID.(int); ok {
			testUser.ID = int64(userID)
		} else {
			testUser.ID = int64(0) // Fallback for interface{} types
		}

		t.Logf("Created test user: %s (%s)", testUser.Username, testUser.Description)
	}

	t.Logf("Test data setup complete: %d clients, %d users", len(testClients), len(testUsers))
}

// cleanupTestData removes all test data
func cleanupTestData(t *testing.T, service *Service) {
	ctx := context.Background()

	// Clean up test clients
	clientProvider := service.GetClientProvider()
	for _, testClient := range testClients {
		err := clientProvider.DeleteClient(ctx, testClient.ClientID)
		if err != nil {
			t.Logf("Warning: Failed to delete test client %s: %v", testClient.ClientID, err)
		}
	}

	// Clean up test users
	m := model.Select("__yao.user")
	for _, testUser := range testUsers {
		if testUser.ID > 0 {
			err := m.Destroy(testUser.ID)
			if err != nil {
				t.Logf("Warning: Failed to delete test user %d: %v", testUser.ID, err)
			}
		}
	}

	// Clean up any remaining test data by patterns
	_, err := m.DestroyWhere(model.QueryParam{
		Wheres: []model.QueryWhere{
			{Column: "subject", OP: "like", Value: "user-%"},
		},
	})
	if err != nil {
		t.Logf("Warning: Failed to cleanup test users by pattern: %v", err)
	}
}

// createTestCertificatesOnce creates temporary certificate pair for all tests
func createTestCertificatesOnce(t *testing.T) {
	// Generate temporary certificates (auto-generate with empty paths)
	config := &types.SigningConfig{
		SigningAlgorithm: "RS256",
		SigningCertPath:  "",
		SigningKeyPath:   "",
	}

	certs, err := LoadSigningCertificates(config)
	if err != nil {
		t.Fatalf("Failed to generate test certificates: %v", err)
	}

	testCertPath = certs.SigningCertPath
	testKeyPath = certs.SigningKeyPath

	t.Logf("Created test certificates: cert=%s, key=%s", testCertPath, testKeyPath)
}

// Helper functions for store setup (same as in other test files)

func getMongoStore(t *testing.T) store.Store {
	host := os.Getenv("MONGO_TEST_HOST")
	if host == "" {
		t.Skip("MongoDB not available - set MONGO_TEST_HOST environment variable")
	}

	mongoConnector, err := connector.New("mongo", "oauth_test", []byte(`{
		"name": "OAuth Test MongoDB",
		"type": "mongo",
		"options": {
			"db": "oauth_test",
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
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_oauth_badger")

	badgerStore, err := badger.New(dbPath)
	require.NoError(t, err)

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

func getStoreConfigs() []StoreConfig {
	return []StoreConfig{
		{Name: "MongoDB", GetFunc: getMongoStore},
		{Name: "Badger", GetFunc: getBadgerStore},
	}
}

// =============================================================================
// OAuth Service Tests
// =============================================================================

func TestMain(m *testing.M) {
	// Run tests
	code := m.Run()

	// Cleanup global test certificates
	cleanupGlobalTestCertificates()

	os.Exit(code)
}

// cleanupGlobalTestCertificates removes global test certificates
func cleanupGlobalTestCertificates() {
	if testCertPath != "" {
		if _, err := os.Stat(testCertPath); !os.IsNotExist(err) {
			os.Remove(testCertPath)
		}
	}
	if testKeyPath != "" {
		if _, err := os.Stat(testKeyPath); !os.IsNotExist(err) {
			os.Remove(testKeyPath)
		}
	}
}

func TestNewService(t *testing.T) {
	t.Run("create service with valid config", func(t *testing.T) {
		service, _, _, cleanup := setupOAuthTestEnvironment(t)
		defer cleanup()

		assert.NotNil(t, service)
		assert.NotNil(t, service.config)
		assert.NotNil(t, service.store)
		assert.NotNil(t, service.cache)
		assert.NotNil(t, service.userProvider)
		assert.NotNil(t, service.clientProvider)
		assert.NotEmpty(t, service.prefix)
	})

	t.Run("create service with nil config", func(t *testing.T) {
		service, err := NewService(nil)
		assert.Error(t, err)
		assert.Nil(t, service)
		assert.Equal(t, types.ErrInvalidConfiguration, err)
	})

	t.Run("create service with missing store", func(t *testing.T) {
		config := &Config{
			IssuerURL: "https://test.example.com",
		}

		service, err := NewService(config)
		assert.Error(t, err)
		assert.Nil(t, service)
		assert.Equal(t, types.ErrStoreMissing, err)
	})

	t.Run("create service with missing issuer URL", func(t *testing.T) {
		store := getBadgerStore(t)
		config := &Config{
			Store: store,
			Signing: types.SigningConfig{
				SigningCertPath: testCertPath,
				SigningKeyPath:  testKeyPath,
			},
		}

		service, err := NewService(config)
		assert.Error(t, err)
		assert.Nil(t, service)
		assert.Equal(t, types.ErrIssuerURLMissing, err)
	})
}

func TestServiceGetters(t *testing.T) {
	service, _, _, cleanup := setupOAuthTestEnvironment(t)
	defer cleanup()

	t.Run("get config", func(t *testing.T) {
		config := service.GetConfig()
		assert.NotNil(t, config)
		assert.Equal(t, "https://oauth.test.example.com", config.IssuerURL)
		assert.True(t, config.Features.OAuth21Enabled)
	})

	t.Run("get user provider", func(t *testing.T) {
		userProvider := service.GetUserProvider()
		assert.NotNil(t, userProvider)
		assert.Implements(t, (*types.UserProvider)(nil), userProvider)
	})

	t.Run("get client provider", func(t *testing.T) {
		clientProvider := service.GetClientProvider()
		assert.NotNil(t, clientProvider)
		assert.Implements(t, (*types.ClientProvider)(nil), clientProvider)
	})
}

func TestConfigDefaults(t *testing.T) {
	store := getBadgerStore(t)

	t.Run("set default values", func(t *testing.T) {
		config := &Config{
			Store:     store,
			IssuerURL: "https://test.example.com",
			Signing: types.SigningConfig{
				SigningCertPath: testCertPath,
				SigningKeyPath:  testKeyPath,
			},
		}

		service, err := NewService(config)
		assert.NoError(t, err)
		assert.NotNil(t, service)

		// Check that defaults were set
		assert.Equal(t, "RS256", config.Signing.SigningAlgorithm)
		assert.Equal(t, time.Hour, config.Token.AccessTokenLifetime)
		assert.Equal(t, 24*time.Hour, config.Token.RefreshTokenLifetime)
		assert.Equal(t, 10*time.Minute, config.Token.AuthorizationCodeLifetime)
		assert.Equal(t, 15*time.Minute, config.Token.DeviceCodeLifetime)
		assert.Equal(t, "jwt", config.Token.AccessTokenFormat)
		assert.Equal(t, "opaque", config.Token.RefreshTokenFormat)
		assert.Equal(t, []string{"S256"}, config.Security.PKCECodeChallengeMethod)
		assert.Equal(t, 128, config.Security.PKCECodeVerifierLength)
		assert.Equal(t, 10*time.Minute, config.Security.StateParameterLifetime)
		assert.Equal(t, 32, config.Security.StateParameterLength)
		assert.Equal(t, types.ClientTypeConfidential, config.Client.DefaultClientType)
		assert.Equal(t, "client_secret_basic", config.Client.DefaultTokenEndpointAuthMethod)
		assert.Equal(t, []string{"authorization_code", "refresh_token"}, config.Client.DefaultGrantTypes)
		assert.Equal(t, []string{"code"}, config.Client.DefaultResponseTypes)
		assert.Equal(t, 32, config.Client.ClientIDLength)
		assert.Equal(t, 64, config.Client.ClientSecretLength)
		assert.True(t, config.Features.OAuth21Enabled)
		assert.True(t, config.Features.PKCEEnforced)
		assert.True(t, config.Features.RefreshTokenRotationEnabled)
	})
}

func TestFeatureFlags(t *testing.T) {
	service, _, _, cleanup := setupOAuthTestEnvironment(t)
	defer cleanup()

	config := service.GetConfig()

	t.Run("oauth 2.1 features enabled", func(t *testing.T) {
		assert.True(t, config.Features.OAuth21Enabled)
		assert.True(t, config.Features.PKCEEnforced)
		assert.True(t, config.Features.RefreshTokenRotationEnabled)
	})

	t.Run("advanced features enabled", func(t *testing.T) {
		assert.True(t, config.Features.DeviceFlowEnabled)
		assert.True(t, config.Features.TokenExchangeEnabled)
		assert.True(t, config.Features.PushedAuthorizationEnabled)
		assert.True(t, config.Features.DynamicClientRegistrationEnabled)
	})

	t.Run("mcp features enabled", func(t *testing.T) {
		assert.True(t, config.Features.MCPComplianceEnabled)
		assert.True(t, config.Features.ResourceParameterEnabled)
	})

	t.Run("security features configured", func(t *testing.T) {
		assert.True(t, config.Features.TokenBindingEnabled)
		assert.False(t, config.Features.MTLSEnabled)
		assert.False(t, config.Features.DPoPEnabled)
	})

	t.Run("experimental features enabled", func(t *testing.T) {
		assert.True(t, config.Features.JWTIntrospectionEnabled)
		assert.True(t, config.Features.TokenRevocationEnabled)
		assert.True(t, config.Features.UserInfoJWTEnabled)
	})
}

func TestProviderInitialization(t *testing.T) {
	t.Run("default providers created when not provided", func(t *testing.T) {
		store := getBadgerStore(t)
		cache := getLRUCache(t)

		config := &Config{
			Store:     store,
			Cache:     cache,
			IssuerURL: "https://test.example.com",
			Signing: types.SigningConfig{
				SigningCertPath: testCertPath,
				SigningKeyPath:  testKeyPath,
			},
		}

		service, err := NewService(config)
		assert.NoError(t, err)
		assert.NotNil(t, service)

		// Check that default providers were created
		assert.NotNil(t, service.userProvider)
		assert.NotNil(t, service.clientProvider)

		// Verify they implement the correct interfaces
		assert.Implements(t, (*types.UserProvider)(nil), service.userProvider)
		assert.Implements(t, (*types.ClientProvider)(nil), service.clientProvider)
	})

	t.Run("custom providers used when provided", func(t *testing.T) {
		store := getBadgerStore(t)
		cache := getLRUCache(t)

		// Create a temporary service to get default providers for testing
		tempConfig := &Config{
			Store:     store,
			Cache:     cache,
			IssuerURL: "https://test.example.com",
			Signing: types.SigningConfig{
				SigningCertPath: testCertPath,
				SigningKeyPath:  testKeyPath,
			},
		}

		tempService, err := NewService(tempConfig)
		require.NoError(t, err)

		// Create custom providers (for this test, we'll use the default ones)
		customUserProvider := tempService.GetUserProvider()
		customClientProvider := tempService.GetClientProvider()

		config := &Config{
			Store:          store,
			Cache:          cache,
			UserProvider:   customUserProvider,
			ClientProvider: customClientProvider,
			IssuerURL:      "https://test.example.com",
			Signing: types.SigningConfig{
				SigningCertPath: testCertPath,
				SigningKeyPath:  testKeyPath,
			},
		}

		service, err := NewService(config)
		assert.NoError(t, err)
		assert.NotNil(t, service)

		// Check that custom providers were used
		assert.Equal(t, customUserProvider, service.userProvider)
		assert.Equal(t, customClientProvider, service.clientProvider)
	})
}

func TestServiceIntegration(t *testing.T) {
	service, _, _, cleanup := setupOAuthTestEnvironment(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("verify test clients are accessible", func(t *testing.T) {
		clientProvider := service.GetClientProvider()

		for _, testClient := range testClients {
			client, err := clientProvider.GetClientByID(ctx, testClient.ClientID)
			assert.NoError(t, err, "Failed to get client %s", testClient.ClientID)
			assert.NotNil(t, client, "Client %s should not be nil", testClient.ClientID)
			assert.Equal(t, testClient.ClientName, client.ClientName)
			assert.Equal(t, testClient.ClientType, client.ClientType)
		}
	})

	t.Run("verify test users are accessible", func(t *testing.T) {
		userProvider := service.GetUserProvider()

		for _, testUser := range testUsers {
			user, err := userProvider.GetUserBySubject(ctx, testUser.Subject)
			assert.NoError(t, err, "Failed to get user %s", testUser.Subject)
			assert.NotNil(t, user, "User %s should not be nil", testUser.Subject)

			// Note: Skip detailed verification as user structure may vary by provider
			// The important thing is that the user exists and can be retrieved
		}
	})
}

// =============================================================================
// Configuration Validation Tests
// =============================================================================

func TestConfigValidation(t *testing.T) {
	t.Run("valid configuration", func(t *testing.T) {
		config := &Config{
			Store:     getBadgerStore(t),
			IssuerURL: "https://test.example.com",
			Signing: types.SigningConfig{
				SigningCertPath: testCertPath,
				SigningKeyPath:  testKeyPath,
			},
			Token: types.TokenConfig{
				AccessTokenLifetime:       time.Hour,
				RefreshTokenLifetime:      24 * time.Hour,
				AuthorizationCodeLifetime: 10 * time.Minute,
			},
		}

		// Set defaults first (like NewService does)
		err := setConfigDefaults(config)
		assert.NoError(t, err)

		// Then validate
		err = validateConfig(config)
		assert.NoError(t, err)
	})

	t.Run("missing store", func(t *testing.T) {
		config := &Config{
			IssuerURL: "https://test.example.com",
			Signing: types.SigningConfig{
				SigningCertPath: testCertPath,
				SigningKeyPath:  testKeyPath,
			},
		}

		// Set defaults first (like NewService does)
		err := setConfigDefaults(config)
		assert.NoError(t, err)

		// Then validate
		err = validateConfig(config)
		assert.Error(t, err)
		assert.Equal(t, types.ErrStoreMissing, err)
	})

	t.Run("missing issuer URL", func(t *testing.T) {
		config := &Config{
			Store: getBadgerStore(t),
			Signing: types.SigningConfig{
				SigningCertPath: testCertPath,
				SigningKeyPath:  testKeyPath,
			},
		}

		// Set defaults first (like NewService does)
		err := setConfigDefaults(config)
		assert.NoError(t, err)

		// Then validate
		err = validateConfig(config)
		assert.Error(t, err)
		assert.Equal(t, types.ErrIssuerURLMissing, err)
	})

	t.Run("partial certificate configuration", func(t *testing.T) {
		config := &Config{
			Store:     getBadgerStore(t),
			IssuerURL: "https://test.example.com",
			Signing: types.SigningConfig{
				SigningCertPath: testCertPath, // Only cert path, missing key path
				SigningKeyPath:  "",
			},
		}

		// Set defaults first (like NewService does)
		err := setConfigDefaults(config)
		assert.NoError(t, err)

		// Then validate
		err = validateConfig(config)
		assert.Error(t, err)
		assert.Equal(t, types.ErrCertificateMissing, err)
	})

	t.Run("invalid token lifetime", func(t *testing.T) {
		config := &Config{
			Store:     getBadgerStore(t),
			IssuerURL: "https://test.example.com",
			Signing: types.SigningConfig{
				SigningCertPath: testCertPath,
				SigningKeyPath:  testKeyPath,
			},
			Token: types.TokenConfig{
				AccessTokenLifetime: -1 * time.Hour, // Invalid negative lifetime
			},
		}

		// Set defaults first (like NewService does)
		err := setConfigDefaults(config)
		assert.NoError(t, err)

		// Then validate
		err = validateConfig(config)
		assert.Error(t, err)
		assert.Equal(t, types.ErrInvalidTokenLifetime, err)
	})
}

func TestDefaultConfigValues(t *testing.T) {
	t.Run("test all default values", func(t *testing.T) {
		config := &Config{}

		err := setConfigDefaults(config)
		assert.NoError(t, err)

		// Test signing defaults
		assert.Equal(t, "RS256", config.Signing.SigningAlgorithm)

		// Test token defaults
		assert.Equal(t, time.Hour, config.Token.AccessTokenLifetime)
		assert.Equal(t, 24*time.Hour, config.Token.RefreshTokenLifetime)
		assert.Equal(t, 10*time.Minute, config.Token.AuthorizationCodeLifetime)
		assert.Equal(t, 15*time.Minute, config.Token.DeviceCodeLifetime)
		assert.Equal(t, "jwt", config.Token.AccessTokenFormat)
		assert.Equal(t, "opaque", config.Token.RefreshTokenFormat)

		// Test security defaults
		assert.Equal(t, []string{"S256"}, config.Security.PKCECodeChallengeMethod)
		assert.Equal(t, 128, config.Security.PKCECodeVerifierLength)
		assert.Equal(t, 10*time.Minute, config.Security.StateParameterLifetime)
		assert.Equal(t, 32, config.Security.StateParameterLength)

		// Test client defaults
		assert.Equal(t, types.ClientTypeConfidential, config.Client.DefaultClientType)
		assert.Equal(t, "client_secret_basic", config.Client.DefaultTokenEndpointAuthMethod)
		assert.Equal(t, []string{"authorization_code", "refresh_token"}, config.Client.DefaultGrantTypes)
		assert.Equal(t, []string{"code"}, config.Client.DefaultResponseTypes)
		assert.Equal(t, 32, config.Client.ClientIDLength)
		assert.Equal(t, 64, config.Client.ClientSecretLength)

		// Test feature flags defaults
		assert.True(t, config.Features.OAuth21Enabled)
		assert.True(t, config.Features.PKCEEnforced)
		assert.True(t, config.Features.RefreshTokenRotationEnabled)
	})
}
