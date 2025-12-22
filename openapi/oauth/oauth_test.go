package oauth

import (
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/gou/store"
	"github.com/yaoapp/gou/store/lru"
	"github.com/yaoapp/gou/store/xun"
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
// Updated to match the latest user model and provider interfaces
type TestUser struct {
	ID                int64                  `json:"id"`                 // Database ID (auto-generated)
	UserID            string                 `json:"user_id"`            // Global unique user identifier (auto-generated)
	PreferredUsername string                 `json:"preferred_username"` // OIDC preferred username
	Email             string                 `json:"email"`              // OIDC email address
	Password          string                 `json:"password"`           // Plain password (will be hashed by Yao)
	Name              string                 `json:"name"`               // OIDC full name
	GivenName         string                 `json:"given_name"`         // OIDC given name(s) or first name(s)
	FamilyName        string                 `json:"family_name"`        // OIDC surname(s) or last name(s)
	Status            string                 `json:"status"`             // User account status (pending, active, disabled, etc.)
	RoleID            string                 `json:"role_id"`            // User role identifier
	TypeID            string                 `json:"type_id"`            // User type identifier
	EmailVerified     bool                   `json:"email_verified"`     // OIDC email verification status
	MFAEnabled        bool                   `json:"mfa_enabled"`        // Whether multi-factor authentication is enabled
	Metadata          map[string]interface{} `json:"metadata"`           // Extended user metadata and custom fields
	Description       string                 `json:"description"`        // For test identification
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
// Updated to match the latest user model and provider interfaces
var testUsers = []*TestUser{
	{
		UserID:            "", // Will be auto-generated by CreateUser
		PreferredUsername: "admin",
		Email:             "admin@example.com",
		Password:          "Admin123!@#", // Plain password (will be hashed by Yao)
		Name:              "Admin User",
		GivenName:         "Admin",
		FamilyName:        "User",
		Status:            "active",
		RoleID:            "admin",    // Administrator role
		TypeID:            "internal", // Internal user type
		EmailVerified:     true,
		MFAEnabled:        true,
		Metadata: map[string]interface{}{
			"department":  "IT",
			"permissions": []string{"admin", "user_management", "system_config"},
		},
		Description: "Administrator user with full privileges",
	},
	{
		UserID:            "", // Will be auto-generated by CreateUser
		PreferredUsername: "john.doe",
		Email:             "john.doe@example.com",
		Password:          "JohnDoe123!",
		Name:              "John Doe",
		GivenName:         "John",
		FamilyName:        "Doe",
		Status:            "active",
		RoleID:            "user",     // Regular user role
		TypeID:            "external", // External user type
		EmailVerified:     true,
		MFAEnabled:        false,
		Metadata: map[string]interface{}{
			"company":   "Example Corp",
			"job_title": "Software Engineer",
		},
		Description: "Regular user with basic privileges",
	},
	{
		UserID:            "", // Will be auto-generated by CreateUser
		PreferredUsername: "jane.smith",
		Email:             "jane.smith@example.com",
		Password:          "JaneSmith456!",
		Name:              "Jane Smith",
		GivenName:         "Jane",
		FamilyName:        "Smith",
		Status:            "active",
		RoleID:            "user",     // Regular user role
		TypeID:            "external", // External user type
		EmailVerified:     true,
		MFAEnabled:        false,
		Metadata: map[string]interface{}{
			"company":         "Tech Solutions",
			"job_title":       "Product Manager",
			"mobile_verified": true,
		},
		Description: "Regular user with verified mobile",
	},
	{
		UserID:            "", // Will be auto-generated by CreateUser
		PreferredUsername: "pending.user",
		Email:             "pending@example.com",
		Password:          "Pending789!",
		Name:              "Pending User",
		GivenName:         "Pending",
		FamilyName:        "User",
		Status:            "pending",  // Awaiting verification
		RoleID:            "user",     // Regular user role
		TypeID:            "external", // External user type
		EmailVerified:     false,
		MFAEnabled:        false,
		Metadata: map[string]interface{}{
			"registration_source":   "web_signup",
			"verification_required": true,
		},
		Description: "User with pending verification",
	},
	{
		UserID:            "", // Will be auto-generated by CreateUser
		PreferredUsername: "inactive.user",
		Email:             "inactive@example.com",
		Password:          "Inactive123!",
		Name:              "Inactive User",
		GivenName:         "Inactive",
		FamilyName:        "User",
		Status:            "disabled", // Changed from "inactive" to match model enum
		RoleID:            "user",     // Regular user role
		TypeID:            "external", // External user type
		EmailVerified:     true,
		MFAEnabled:        false,
		Metadata: map[string]interface{}{
			"deactivation_reason": "admin_action",
			"deactivated_at":      "2024-01-01T00:00:00Z",
		},
		Description: "Disabled user account",
	},
	{
		UserID:            "", // Will be auto-generated by CreateUser
		PreferredUsername: "limited.user",
		Email:             "limited@example.com",
		Password:          "Limited456!",
		Name:              "Limited User",
		GivenName:         "Limited",
		FamilyName:        "User",
		Status:            "active",
		RoleID:            "guest", // Limited guest role
		TypeID:            "guest", // Guest user type
		EmailVerified:     true,
		MFAEnabled:        false,
		Metadata: map[string]interface{}{
			"access_level": "read_only",
			"restrictions": []string{"no_data_export", "limited_api_access"},
		},
		Description: "User with limited access privileges",
	},
	{
		UserID:            "", // Will be auto-generated by CreateUser
		PreferredUsername: "secure.user",
		Email:             "secure@example.com",
		Password:          "SecureUser789!@#",
		Name:              "Secure User",
		GivenName:         "Secure",
		FamilyName:        "User",
		Status:            "active",
		RoleID:            "user",     // Regular user role
		TypeID:            "internal", // Internal user type
		EmailVerified:     true,
		MFAEnabled:        true, // Security-focused with MFA
		Metadata: map[string]interface{}{
			"security_clearance": "high",
			"department":         "Security",
			"mobile_verified":    true,
		},
		Description: "Security-focused user with 2FA enabled",
	},
	{
		UserID:            "", // Will be auto-generated by CreateUser
		PreferredUsername: "api.user",
		Email:             "api@example.com",
		Password:          "ApiUser123!@#",
		Name:              "API User",
		GivenName:         "API",
		FamilyName:        "User",
		Status:            "active",
		RoleID:            "api",     // API access role
		TypeID:            "service", // Service account type
		EmailVerified:     true,
		MFAEnabled:        false, // Service accounts typically don't use MFA
		Metadata: map[string]interface{}{
			"api_scopes":   []string{"api:read", "api:write"},
			"service_type": "automated_system",
		},
		Description: "User for API access testing",
	},
	{
		UserID:            "", // Will be auto-generated by CreateUser
		PreferredUsername: "guest.user",
		Email:             "guest@example.com",
		Password:          "GuestUser456!",
		Name:              "Guest User",
		GivenName:         "Guest",
		FamilyName:        "User",
		Status:            "active",
		RoleID:            "guest", // Guest role
		TypeID:            "guest", // Guest user type
		EmailVerified:     false,   // Guests may not verify email
		MFAEnabled:        false,
		Metadata: map[string]interface{}{
			"access_level":     "minimal",
			"temporary_access": true,
		},
		Description: "Guest user with minimal access",
	},
	{
		UserID:            "", // Will be auto-generated by CreateUser
		PreferredUsername: "test.user",
		Email:             "test@example.com",
		Password:          "TestUser789!",
		Name:              "Test User",
		GivenName:         "Test",
		FamilyName:        "User",
		Status:            "active",
		RoleID:            "user",     // Regular user role
		TypeID:            "external", // External user type
		EmailVerified:     true,
		MFAEnabled:        false,
		Metadata: map[string]interface{}{
			"test_account":    true,
			"test_scopes":     []string{"openid", "profile", "email", "test"},
			"mobile_verified": true,
		},
		Description: "General purpose test user",
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

	// Use the first available store (prefer MongoDB, fallback to Xun)
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

	// Fallback to Xun if no other store is available
	if mainStore == nil {
		mainStore = getXunStore(t)
		storeConfig = StoreConfig{Name: "Xun", GetFunc: getXunStore}
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

	// Generate unique test suffix for this test run to avoid conflicts in parallel execution
	testSuffix := generateTestSuffix(t)
	t.Logf("Using test suffix: %s", testSuffix)

	// Create local copies of test clients (don't modify global arrays)
	clientProvider := service.GetClientProvider()
	createdClientIDs := make([]string, len(testClients))

	for i, testClient := range testClients {
		// Make client ID unique for this test run
		uniqueClientID := testClient.ClientID + "-" + testSuffix
		clientInfo := &types.ClientInfo{
			ClientID:                uniqueClientID,
			ClientSecret:            testClient.ClientSecret,
			ClientName:              testClient.ClientName + " (" + testSuffix + ")",
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

		// Store the unique ID for cleanup (without modifying global array)
		createdClientIDs[i] = uniqueClientID

		// Store mapping for other test files to use
		actualTestClientIDs[testClient.ClientID] = uniqueClientID

		t.Logf("Created test client: %s (%s)", uniqueClientID, testClient.Description)
	}

	// Note: cleanup will be handled by cleanupTestData before next test setup

	// Create local copies of test users (don't modify global arrays)
	userProvider, _ := service.GetUserProvider()
	createdUserIDs := make([]string, len(testUsers))
	createdUserEmails := make([]string, len(testUsers))
	createdUsernames := make([]string, len(testUsers))

	for i, testUser := range testUsers {
		// Make email and username unique for this test run
		uniqueEmail := generateUniqueEmail(testUser.Email, testSuffix)
		uniqueUsername := testUser.PreferredUsername + "-" + testSuffix

		// Convert TestUser to the format expected by CreateUser
		userData := map[string]interface{}{
			// Note: user_id is auto-generated by CreateUser, don't include it
			"preferred_username": uniqueUsername,
			"email":              uniqueEmail,
			"password":           testUser.Password, // Plain password (will be hashed by Yao)
			"name":               testUser.Name,
			"given_name":         testUser.GivenName,
			"family_name":        testUser.FamilyName,
			"status":             testUser.Status,
			"role_id":            testUser.RoleID,
			"type_id":            testUser.TypeID,
			"email_verified":     testUser.EmailVerified,
			"mfa_enabled":        testUser.MFAEnabled,
			"metadata":           testUser.Metadata,
		}

		createdUserID, err := userProvider.CreateUser(ctx, userData)
		require.NoError(t, err, "Failed to create test user %d: %s", i, testUser.Description)
		require.NotEmpty(t, createdUserID, "Created user ID should not be empty")

		// Store the unique identifiers for cleanup (without modifying global array)
		createdUserIDs[i] = createdUserID
		createdUserEmails[i] = uniqueEmail
		createdUsernames[i] = uniqueUsername

		// Store mapping for other test files to use
		actualTestUserEmails[testUser.Email] = uniqueEmail

		t.Logf("Created test user: %s (ID: %s, Email: %s, %s)", uniqueUsername, createdUserID, uniqueEmail, testUser.Description)
	}

	// Note: cleanup will be handled by cleanupTestData before next test setup

	t.Logf("Test data setup complete: %d clients, %d users", len(testClients), len(testUsers))
}

// cleanupTestData removes all test data
func cleanupTestData(t *testing.T, service *Service) {
	// This function is now mainly for general cleanup and doesn't modify global arrays
	// Specific cleanup is handled by t.Cleanup() in setupTestData

	// Clean up any remaining test data by patterns with comprehensive cleanup
	m := model.Select("__yao.user")
	cleanupPatterns := []string{
		"user-%",     // Original pattern
		"admin-%",    // Admin users with suffix
		"john.doe-%", // John Doe users with suffix
		"jane.smith-%",
		"pending.user-%",
		"inactive.user-%",
		"limited.user-%",
		"secure.user-%",
		"api.user-%",
		"guest.user-%",
		"test.user-%",
		"%test-confidential-client-%", // Client patterns
		"%test-public-client-%",
		"%test-credentials-client-%",
		"%t%-%", // General pattern for timestamp-based suffixes
	}

	for _, pattern := range cleanupPatterns {
		_, err := m.DestroyWhere(model.QueryParam{
			Wheres: []model.QueryWhere{
				{Column: "user_id", OP: "like", Value: pattern},
			},
		})
		if err != nil {
			t.Logf("Warning: Failed to cleanup test users by pattern %s: %v", pattern, err)
		}

		// Also clean by email pattern
		_, err = m.DestroyWhere(model.QueryParam{
			Wheres: []model.QueryWhere{
				{Column: "email", OP: "like", Value: pattern},
			},
		})
		if err != nil {
			t.Logf("Warning: Failed to cleanup test users by email pattern %s: %v", pattern, err)
		}

		// Also clean by username pattern
		_, err = m.DestroyWhere(model.QueryParam{
			Wheres: []model.QueryWhere{
				{Column: "preferred_username", OP: "like", Value: pattern},
			},
		})
		if err != nil {
			t.Logf("Warning: Failed to cleanup test users by username pattern %s: %v", pattern, err)
		}
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

func getXunStore(t *testing.T) store.Store {
	// Use test.Prepare to ensure database connection is initialized
	test.Prepare(t, config.Conf)

	// Create xun store using default database connection
	xunStore, err := xun.New(xun.Option{
		Table:     "__yao_oauth_test",
		Connector: "default",
		CacheSize: 1024,
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		xunStore.Clear()
		xunStore.Close()
		test.Clean()
	})

	return xunStore
}

func getLRUCache(t *testing.T) store.Store {
	cache, err := lru.New(1000)
	require.NoError(t, err)
	return cache
}

func getStoreConfigs() []StoreConfig {
	return []StoreConfig{
		{Name: "MongoDB", GetFunc: getMongoStore},
		{Name: "Xun", GetFunc: getXunStore},
	}
}

// Global mapping for actual created IDs (for use by other test files)
var actualTestClientIDs = make(map[string]string)  // original -> actual ID with suffix
var actualTestUserEmails = make(map[string]string) // original -> actual email with suffix

// GetActualClientID returns the actual client ID with suffix (for use by other test files)
func GetActualClientID(originalID string) string {
	if actualID, exists := actualTestClientIDs[originalID]; exists {
		return actualID
	}
	return originalID // fallback to original if not found
}

// GetActualUserEmail returns the actual user email with suffix (for use by other test files)
func GetActualUserEmail(originalEmail string) string {
	if actualEmail, exists := actualTestUserEmails[originalEmail]; exists {
		return actualEmail
	}
	return originalEmail // fallback to original if not found
}

// generateTestSuffix creates a unique suffix for test isolation in parallel execution
func generateTestSuffix(t *testing.T) string {
	// Add random component for uniqueness
	b := make([]byte, 6)
	rand.Read(b)
	randomSuffix := fmt.Sprintf("%x", b)

	// Create a short but unique suffix using just timestamp and random
	timestamp := time.Now().UnixNano() / 1e6 // milliseconds
	suffix := fmt.Sprintf("t%d-%s", timestamp, randomSuffix)

	// Keep it short and simple for better readability
	if len(suffix) > 20 {
		suffix = suffix[:20]
	}

	return suffix
}

// generateUniqueEmail creates a unique email address for test isolation
func generateUniqueEmail(originalEmail, suffix string) string {
	parts := strings.Split(originalEmail, "@")
	if len(parts) != 2 {
		// Fallback for malformed emails
		return fmt.Sprintf("test-%s@example.com", suffix)
	}

	// Insert suffix before @domain
	return fmt.Sprintf("%s-%s@%s", parts[0], suffix, parts[1])
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
		store := getXunStore(t)
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
		userProvider, _ := service.GetUserProvider()
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
	store := getXunStore(t)

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
		store := getXunStore(t)
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
		store := getXunStore(t)
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
		customUserProvider, _ := tempService.GetUserProvider()
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
			actualClientID := GetActualClientID(testClient.ClientID)
			client, err := clientProvider.GetClientByID(ctx, actualClientID)
			assert.NoError(t, err, "Failed to get client %s", actualClientID)
			assert.NotNil(t, client, "Client %s should not be nil", actualClientID)
			// Client name should contain the suffix, but should still match the client type exactly
			assert.Contains(t, client.ClientName, testClient.ClientName)
			assert.Equal(t, testClient.ClientType, client.ClientType)
		}
	})

	// t.Run("verify test users are accessible", func(t *testing.T) {
	// 	userProvider := service.GetUserProvider()

	// 	for _, testUser := range testUsers {
	// 		user, err := userProvider.GetUser(ctx, testUser.Subject)
	// 		assert.NoError(t, err, "Failed to get user %s", testUser.Subject)
	// 		assert.NotNil(t, user, "User %s should not be nil", testUser.Subject)

	// 		// Note: Skip detailed verification as user structure may vary by provider
	// 		// The important thing is that the user exists and can be retrieved
	// 	}
	// })
}

// =============================================================================
// Configuration Validation Tests
// =============================================================================

func TestConfigValidation(t *testing.T) {
	t.Run("valid configuration", func(t *testing.T) {
		config := &Config{
			Store:     getXunStore(t),
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
			Store: getXunStore(t),
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
			Store:     getXunStore(t),
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
			Store:     getXunStore(t),
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
