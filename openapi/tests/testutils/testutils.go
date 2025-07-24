package testutils

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/kb"
	"github.com/yaoapp/yao/openapi"
	"github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/test"
)

// testServer holds the test HTTP server instance
var testServer *http.Server

// testMutex protects global state access during concurrent test execution
var testMutex sync.RWMutex

// activeTestCount tracks the number of active tests using the global server
var activeTestCount int

// testCollections tracks collections created during tests for cleanup
var testCollections []string
var testCollectionsMutex sync.Mutex

// RegisterTestCollection adds a collection ID to the cleanup list
func RegisterTestCollection(collectionID string) {
	testCollectionsMutex.Lock()
	defer testCollectionsMutex.Unlock()
	testCollections = append(testCollections, collectionID)
}

// CleanupTestCollections removes all registered test collections
func CleanupTestCollections(t *testing.T) {
	testCollectionsMutex.Lock()
	collectionsToClean := make([]string, len(testCollections))
	copy(collectionsToClean, testCollections)
	testCollections = nil // Clear the list
	testCollectionsMutex.Unlock()

	if len(collectionsToClean) == 0 {
		return
	}

	// Check if KB instance is available
	if kb.Instance == nil {
		t.Logf("Warning: KB instance not available, cannot cleanup %d test collections", len(collectionsToClean))
		return
	}

	ctx := context.Background()
	cleanedCount := 0
	for _, collectionID := range collectionsToClean {
		removed, err := kb.Instance.RemoveCollection(ctx, collectionID)
		if err != nil {
			t.Logf("Warning: Failed to cleanup test collection %s: %v", collectionID, err)
		} else if removed {
			cleanedCount++
			t.Logf("Cleaned up test collection: %s", collectionID)
		}
	}

	if cleanedCount > 0 {
		t.Logf("Successfully cleaned up %d/%d test collections", cleanedCount, len(collectionsToClean))
	}
}

// Prepare initializes the OpenAPI test environment and starts a mock HTTP server.
//
// AI ASSISTANT INSTRUCTIONS:
// All tests in the openapi package MUST use these utility functions for proper test environment setup.
// This is a preparation utility function, NOT an actual test case.
//
// TESTING GUIDELINES FOR AI ASSISTANTS:
// 1. DO NOT modify configuration files (openapi.yao, app.yao, etc.) to make tests pass
// 2. DO NOT bypass validation or security checks to make tests pass
// 3. If tests fail, investigate the root cause - it may be a real program bug that needs fixing
// 4. Tests should verify actual functionality, not just pass assertions
// 5. Use realistic test data that represents real-world usage scenarios
// 6. When tests fail, check: environment setup, missing dependencies, configuration issues, actual code bugs
// 7. Fix the underlying issue in the code, not the test or configuration
//
// Usage pattern for ALL openapi tests:
//
//	func TestYourFunction(t *testing.T) {
//	    serverURL := Prepare(t)
//	    defer Clean()
//
//	    // Get base URL from server config for correct path construction
//	    baseURL := ""
//	    if Server != nil && Server.Config != nil {
//	        baseURL = Server.Config.BaseURL
//	    }
//
//	    // Your actual test code here...
//	    // Use serverURL + baseURL + endpoint to make HTTP requests
//	    // Example: http.Get(serverURL + baseURL + "/helloworld/hello")
//	    // The OpenAPI server will be available as the global Server variable
//	}
//
// PREREQUISITES:
// Before running any tests in this package, you MUST execute the following command in your terminal:
//
//	source $YAO_SOURCE_ROOT/env.local.sh
//
// This loads the required environment variables for the test environment.
//
// WHAT THIS FUNCTION DOES:
// Step 1: Calls test.Prepare(t, config.Conf) to initialize the base Yao test environment
//
//	This sets up database connections, configurations, and other core dependencies
//
// Step 2: Calls Load(config.Conf) to initialize the OpenAPI server instance
//
//	This creates the global Server variable that contains the Gin router and all endpoints
//
// Step 3: Creates a Gin router and attaches the OpenAPI server to it
//
//	The server uses Server.Config.BaseURL as the base path for all endpoints
//
// Step 4: Starts an HTTP server on a random available port (127.0.0.1:xxxxx)
//
//	This allows actual HTTP testing of the OpenAPI endpoints
//
// RETURN VALUE:
// Returns the server URL in format "http://127.0.0.1:xxxxx" where xxxxx is the random port
// NOTE: You need to append Server.Config.BaseURL to construct the full endpoint URL
//
// ERROR HANDLING:
// If any step fails, the test will fail immediately with a descriptive error message.
func Prepare(t *testing.T) string {
	// Use write lock to protect global state initialization
	testMutex.Lock()
	defer func() {
		activeTestCount++
		testMutex.Unlock()
	}()

	// Step 1: Initialize base test environment with all Yao dependencies
	test.Prepare(t, config.Conf)

	// Step 1.5: Initialize Knowledge Base (must be done before OpenAPI load)
	_, err := kb.Load(config.Conf)
	if err != nil {
		// KB loading failure is not fatal for tests, just log it
		t.Logf("Warning: Failed to load Knowledge Base: %v", err)
		t.Logf("Some KB-related tests may not work properly")
	}

	// Step 2: Initialize OpenAPI server and make it available globally (only if not already initialized)
	if openapi.Server == nil {
		_, err := openapi.Load(config.Conf)
		if err != nil {
			t.Fatalf("Failed to load OpenAPI server: %v", err)
		}
	}

	// Step 3: Create Gin router and attach OpenAPI server
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Attach the OpenAPI server to the router
	if openapi.Server != nil {
		openapi.Server.Attach(router)
	}

	// Step 4: Start HTTP server on random available port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}

	server := &http.Server{
		Handler: router,
	}

	// Start server in background
	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			t.Errorf("Failed to start test server: %v", err)
		}
	}()

	// Wait a moment for server to start
	time.Sleep(10 * time.Millisecond)

	// Store server instance for this test (each test gets its own HTTP server)
	testServer = server

	// Return server URL
	serverURL := fmt.Sprintf("http://%s", listener.Addr().String())
	return serverURL
}

// Clean cleans up the OpenAPI test environment and shuts down the test server.
//
// AI ASSISTANT INSTRUCTIONS:
// This function MUST be called with defer in every test that uses Prepare().
// This is a cleanup utility function, NOT an actual test case.
// Always use: defer Clean()
//
// WHAT THIS FUNCTION DOES:
// Step 1: Gracefully shutdown the HTTP test server if it exists
//
//	This ensures all pending requests are completed and resources are freed
//
// Step 2: Reset the global Server variable to nil
//
//	This ensures no state leakage between tests and prevents memory leaks
//
// Step 3: Clean up GraphRag test collections
//
//	This removes any test collections that were created during the test
//
// Step 4: Calls test.Clean() to clean up the base test environment
//
//	This closes database connections, cleans up temporary files, and resets global state
//
// IMPORTANT NOTES:
// - This function should ALWAYS be called with defer to ensure cleanup happens even if tests panic
// - Proper cleanup prevents test interference and resource leaks
// - The order of cleanup steps is important: HTTP server first, then OpenAPI cleanup, then base cleanup
// - Server shutdown has a 5-second timeout to prevent hanging tests
func Clean() {
	// Step 1: Gracefully shutdown the HTTP test server for this test
	if testServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := testServer.Shutdown(ctx); err != nil {
			// Force close if graceful shutdown fails
			testServer.Close()
		}
		testServer = nil
	}

	// Step 2: Use lock to safely decrement active test count and clean global state if needed
	testMutex.Lock()
	activeTestCount--
	shouldCleanGlobalState := activeTestCount <= 0
	if shouldCleanGlobalState {
		// Reset global state only when no other tests are active
		openapi.Server = nil
		activeTestCount = 0 // Ensure it doesn't go negative
	}
	testMutex.Unlock()

	// Step 3: Clean up GraphRag test collections (only when cleaning global state)
	if shouldCleanGlobalState {
		// Create a dummy test object for logging (since Clean doesn't receive *testing.T)
		// Note: This is a limitation, but we can still attempt cleanup
		ctx := context.Background()
		testCollectionsMutex.Lock()
		collectionsToClean := make([]string, len(testCollections))
		copy(collectionsToClean, testCollections)
		testCollections = nil // Clear the list
		testCollectionsMutex.Unlock()

		if len(collectionsToClean) > 0 && kb.Instance != nil {
			for _, collectionID := range collectionsToClean {
				_, err := kb.Instance.RemoveCollection(ctx, collectionID)
				if err != nil {
					// Can't use t.Logf here, but errors will be visible in test output
					_ = err
				}
			}
		}
	}

	// Step 4: Clean up base test environment
	if shouldCleanGlobalState {
		test.Clean()
	}
}

// RegisterTestClient registers a test OAuth client and returns the client information.
//
// AI ASSISTANT INSTRUCTIONS:
// Use this function to create test OAuth clients for testing OAuth endpoints.
// This function provides realistic test clients that can be used for authentication flows.
// ALWAYS clean up test clients using CleanupTestClient() to prevent test interference.
//
// Usage pattern:
//
//	func TestOAuthEndpoint(t *testing.T) {
//	    serverURL := Prepare(t)
//	    defer Clean()
//
//	    // Register a test client
//	    client := RegisterTestClient(t, "Test Client", []string{"http://localhost/callback"})
//	    defer CleanupTestClient(t, client.ClientID)
//
//	    // Use client.ClientID and client.ClientSecret in your tests
//	    // Example: test OAuth authorize with real client_id
//	}
//
// PARAMETERS:
// - t: The test instance for error reporting
// - clientName: Human-readable name for the client (e.g., "Test Web App")
// - redirectURIs: List of valid redirect URIs for the client
//
// RETURN VALUE:
// Returns a pointer to types.ClientInfo containing:
// - ClientID: Generated unique client identifier
// - ClientSecret: Generated client secret (for confidential clients)
// - RedirectURIs: The provided redirect URIs
// - Other OAuth client metadata
//
// ERROR HANDLING:
// If client registration fails, the test will fail immediately with a descriptive error message.
func RegisterTestClient(t *testing.T, clientName string, redirectURIs []string) *types.ClientInfo {
	testMutex.RLock()
	server := openapi.Server
	testMutex.RUnlock()

	if server == nil || server.OAuth == nil {
		t.Fatal("OpenAPI server not initialized. Call Prepare(t) first.")
	}

	// Create dynamic client registration request
	req := &types.DynamicClientRegistrationRequest{
		ClientName:   clientName,
		RedirectURIs: redirectURIs,
		GrantTypes: []string{
			"authorization_code",
			"refresh_token",
			"client_credentials",
		},
		ResponseTypes: []string{
			"code",
		},
		ApplicationType:         "web",
		TokenEndpointAuthMethod: "client_secret_basic",
		Scope:                   "openid profile email",
	}

	// Register the client using the OAuth service
	ctx := context.Background()
	response, err := server.OAuth.DynamicClientRegistration(ctx, req)
	if err != nil {
		t.Fatalf("Failed to register test client: %v", err)
	}

	// Convert response to ClientInfo for easier usage
	clientInfo := &types.ClientInfo{
		ClientID:                response.ClientID,
		ClientSecret:            response.ClientSecret,
		ClientName:              response.ClientName,
		RedirectURIs:            response.RedirectURIs,
		GrantTypes:              response.GrantTypes,
		ResponseTypes:           response.ResponseTypes,
		ApplicationType:         response.ApplicationType,
		TokenEndpointAuthMethod: response.TokenEndpointAuthMethod,
		Scope:                   response.Scope,
		ClientURI:               response.ClientURI,
		LogoURI:                 response.LogoURI,
		TosURI:                  response.TosURI,
		PolicyURI:               response.PolicyURI,
		Contacts:                response.Contacts,
	}

	t.Logf("Registered test client: %s (ID: %s)", clientName, clientInfo.ClientID)
	return clientInfo
}

// CleanupTestClient removes a test OAuth client from the system.
//
// AI ASSISTANT INSTRUCTIONS:
// ALWAYS call this function to clean up test clients created with RegisterTestClient().
// Use defer to ensure cleanup happens even if tests fail or panic.
// Proper cleanup prevents test interference and maintains a clean test environment.
//
// Usage pattern:
//
//	client := RegisterTestClient(t, "Test Client", []string{"http://localhost/callback"})
//	defer CleanupTestClient(t, client.ClientID)
//
// PARAMETERS:
// - t: The test instance for error reporting
// - clientID: The client ID to remove (obtained from RegisterTestClient return value)
//
// ERROR HANDLING:
// If client deletion fails, logs an error but does not fail the test.
// This prevents cleanup failures from affecting test results.
func CleanupTestClient(t *testing.T, clientID string) {
	testMutex.RLock()
	server := openapi.Server
	testMutex.RUnlock()

	if server == nil || server.OAuth == nil {
		// Server might already be cleaned up, which is OK
		return
	}

	if clientID == "" {
		return
	}

	// Delete the client using the OAuth service
	ctx := context.Background()
	err := server.OAuth.DeleteClient(ctx, clientID)
	if err != nil {
		// Log error but don't fail the test - cleanup should be resilient
		t.Logf("Warning: Failed to cleanup test client %s: %v", clientID, err)
	} else {
		t.Logf("Cleaned up test client: %s", clientID)
	}
}

// CreateTestClientCredentials creates a simple test client with just ID and secret for basic testing.
//
// AI ASSISTANT INSTRUCTIONS:
// Use this function when you need a quick test client without full OAuth registration.
// This is useful for testing non-OAuth endpoints or when you need predictable client credentials.
// This creates an in-memory client that doesn't persist and doesn't need cleanup.
//
// Usage pattern:
//
//	clientID, clientSecret := CreateTestClientCredentials()
//	// Use in Basic Auth or client_credentials grant tests
//
// RETURN VALUES:
// - clientID: A predictable test client ID
// - clientSecret: A predictable test client secret
//
// NOTE: This function creates temporary credentials and doesn't register them with the OAuth service.
// For full OAuth flow testing, use RegisterTestClient() instead.
func CreateTestClientCredentials() (clientID, clientSecret string) {
	return "test-client-id", "test-client-secret"
}

// AuthorizationInfo represents the information needed for OAuth authorization.
// ObtainAuthorizationCode dynamically obtains an authorization code for testing OAuth token endpoints.
//
// AI ASSISTANT INSTRUCTIONS:
// Use this function to get a real authorization code for testing OAuth token exchange.
// This function simulates the complete OAuth authorization flow and returns all necessary information
// for testing the token endpoint with realistic data.
//
// Usage pattern:
//
//	func TestOAuthToken(t *testing.T) {
//	    serverURL := Prepare(t)
//	    defer Clean()
//
//	    // Register a test client
//	    client := RegisterTestClient(t, "Test Client", []string{"https://localhost/callback"})
//	    defer CleanupTestClient(t, client.ClientID)
//
//	    // Obtain authorization code dynamically
//	    authInfo := ObtainAuthorizationCode(t, serverURL, client.ClientID, "https://localhost/callback", "openid profile")
//
//	    // Now test token endpoint with real authorization code
//	    // POST to /oauth/token with grant_type=authorization_code&code=authInfo.Code&...
//	}
//
// PARAMETERS:
// - t: The test instance for error reporting
// - serverURL: The test server URL (from Prepare function)
// - clientID: The OAuth client ID (from RegisterTestClient)
// - redirectURI: The redirect URI (must match client registration)
// - scope: The requested OAuth scope (e.g., "openid profile email")
//
// RETURN VALUE:
// Returns AuthorizationInfo struct containing:
// - Code: The authorization code for token exchange
// - State: The state parameter for CSRF protection
// - RedirectURI: The redirect URI used in the flow
// - ClientID: The client ID used in the flow
// - Scope: The scope requested in the flow
// - CodeVerifier: The PKCE code verifier for token exchange
// - CodeChallenge: The PKCE code challenge used in authorization
// - CodeChallengeMethod: The PKCE challenge method (S256)
//
// WHAT THIS FUNCTION DOES:
// 1. Generates PKCE parameters for OAuth 2.1 compliance
// 2. Creates a realistic authorization request with proper parameters
// 3. Calls the OAuth service directly to simulate user authorization
// 4. Extracts the authorization code from the response
// 5. Returns all information needed for token endpoint testing
//
// ERROR HANDLING:
// If authorization fails, the test will fail immediately with a descriptive error message.
type AuthorizationInfo struct {
	Code                string
	State               string
	RedirectURI         string
	ClientID            string
	Scope               string
	CodeVerifier        string
	CodeChallenge       string
	CodeChallengeMethod string
}

// ObtainAuthorizationCode obtains an authorization code for testing OAuth token endpoints.
func ObtainAuthorizationCode(t *testing.T, serverURL, clientID, redirectURI, scope string) *AuthorizationInfo {
	testMutex.RLock()
	server := openapi.Server
	testMutex.RUnlock()

	if server == nil || server.OAuth == nil {
		t.Fatal("OpenAPI server not initialized. Call Prepare(t) first.")
	}

	// Generate a unique state parameter for CSRF protection
	state := fmt.Sprintf("test-state-%d", time.Now().UnixNano())

	// Generate PKCE parameters for OAuth 2.1 compliance
	codeVerifier := generateCodeVerifier()
	codeChallenge := generateCodeChallenge(codeVerifier)
	codeChallengeMethod := "S256"

	// Create authorization request with PKCE parameters
	authReq := &types.AuthorizationRequest{
		ClientID:            clientID,
		ResponseType:        "code",
		RedirectURI:         redirectURI,
		Scope:               scope,
		State:               state,
		CodeChallenge:       codeChallenge,
		CodeChallengeMethod: codeChallengeMethod,
	}

	// Call OAuth service to process authorization request
	ctx := context.Background()
	authResp, err := server.OAuth.Authorize(ctx, authReq)
	if err != nil {
		t.Fatalf("Failed to obtain authorization code: %v", err)
	}

	// Check if authorization response contains an error
	if authResp.Error != "" {
		t.Fatalf("Authorization failed: %s - %s", authResp.Error, authResp.ErrorDescription)
	}

	// Verify we got an authorization code
	if authResp.Code == "" {
		t.Fatal("Authorization response missing code")
	}

	authInfo := &AuthorizationInfo{
		Code:                authResp.Code,
		State:               authResp.State,
		RedirectURI:         redirectURI,
		ClientID:            clientID,
		Scope:               scope,
		CodeVerifier:        codeVerifier,
		CodeChallenge:       codeChallenge,
		CodeChallengeMethod: codeChallengeMethod,
	}

	t.Logf("Obtained authorization code: %s (state: %s)", authInfo.Code, authInfo.State)
	return authInfo
}

// TokenInfo represents the information needed for OAuth token exchange.
// ObtainAccessToken directly obtains an access token for testing OAuth endpoints that require authentication.
//
// AI ASSISTANT INSTRUCTIONS:
// Use this function to get a real access token for testing OAuth endpoints like introspect, revoke, etc.
// This function handles the complete OAuth flow (authorization + token exchange) and returns a ready-to-use token.
//
// Usage pattern:
//
//	func TestOAuthIntrospect(t *testing.T) {
//	    serverURL := Prepare(t)
//	    defer Clean()
//
//	    // Register a test client
//	    client := RegisterTestClient(t, "Test Client", []string{"https://localhost/callback"})
//	    defer CleanupTestClient(t, client.ClientID)
//
//	    // Obtain access token directly
//	    tokenInfo := ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")
//
//	    // Now test introspect endpoint with real access token
//	    // POST to /oauth/introspect with token=tokenInfo.AccessToken
//	}
//
// PARAMETERS:
// - t: The test instance for error reporting
// - serverURL: The test server URL (from Prepare function)
// - clientID: The OAuth client ID (from RegisterTestClient)
// - clientSecret: The OAuth client secret (from RegisterTestClient)
// - redirectURI: The redirect URI (must match client registration)
// - scope: The requested OAuth scope (e.g., "openid profile email")
//
// RETURN VALUE:
// Returns TokenInfo struct containing:
// - AccessToken: The access token for API calls
// - RefreshToken: The refresh token for token renewal
// - TokenType: The token type (usually "Bearer")
// - ExpiresIn: Token expiration time in seconds
// - Scope: The granted scope
// - ClientID: The client ID used to obtain the token
//
// WHAT THIS FUNCTION DOES:
// 1. Calls ObtainAuthorizationCode to get an authorization code
// 2. Exchanges the authorization code for an access token using the OAuth service
// 3. Returns all token information needed for authenticated API testing
//
// ERROR HANDLING:
// If token exchange fails, the test will fail immediately with a descriptive error message.
type TokenInfo struct {
	AccessToken  string
	RefreshToken string
	TokenType    string
	ExpiresIn    int
	Scope        string
	ClientID     string
}

// ObtainAccessToken obtains an access token for testing OAuth endpoints that require authentication.
func ObtainAccessToken(t *testing.T, serverURL, clientID, clientSecret, redirectURI, scope string) *TokenInfo {
	testMutex.RLock()
	server := openapi.Server
	testMutex.RUnlock()

	if server == nil || server.OAuth == nil {
		t.Fatal("OpenAPI server not initialized. Call Prepare(t) first.")
	}

	// Step 1: Get authorization code with PKCE parameters
	authInfo := ObtainAuthorizationCode(t, serverURL, clientID, redirectURI, scope)

	// Step 2: Exchange authorization code for access token with PKCE code verifier
	ctx := context.Background()
	token, err := server.OAuth.Token(ctx, "authorization_code", authInfo.Code, clientID, authInfo.CodeVerifier)
	if err != nil {
		t.Fatalf("Failed to exchange authorization code for token: %v", err)
	}

	// Verify we got a valid token
	if token.AccessToken == "" {
		t.Fatal("Token response missing access token")
	}

	tokenInfo := &TokenInfo{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		TokenType:    token.TokenType,
		ExpiresIn:    token.ExpiresIn,
		Scope:        token.Scope,
		ClientID:     clientID,
	}

	t.Logf("Obtained access token: %s (type: %s, expires_in: %d)",
		tokenInfo.AccessToken, tokenInfo.TokenType, tokenInfo.ExpiresIn)
	return tokenInfo
}

// generateCodeVerifier generates a cryptographically random code verifier for PKCE
func generateCodeVerifier() string {
	// PKCE code verifier should be 43-128 characters long
	// We'll generate 32 random bytes and base64url encode them (43 characters)
	bytes := make([]byte, 32)
	_, err := rand.Read(bytes)
	if err != nil {
		panic(fmt.Sprintf("Failed to generate random bytes: %v", err))
	}

	// Base64 URL encoding without padding
	return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(bytes)
}

// generateCodeChallenge generates a code challenge from the code verifier using S256 method
func generateCodeChallenge(codeVerifier string) string {
	// SHA256 hash the code verifier
	hash := sha256.Sum256([]byte(codeVerifier))

	// Base64 URL encode the hash without padding
	return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(hash[:])
}
