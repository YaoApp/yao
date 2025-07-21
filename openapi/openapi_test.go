package openapi

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/test"
)

// testServer holds the test HTTP server instance
var testServer *http.Server

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
	// Step 1: Initialize base test environment with all Yao dependencies
	test.Prepare(t, config.Conf)

	// Step 2: Initialize OpenAPI server and make it available globally
	_, err := Load(config.Conf)
	if err != nil {
		t.Fatalf("Failed to load OpenAPI server: %v", err)
	}

	// Step 3: Create Gin router and attach OpenAPI server
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Attach the OpenAPI server to the router
	if Server != nil {
		Server.Attach(router)
	}

	// Step 4: Start HTTP server on random available port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}

	testServer = &http.Server{
		Handler: router,
	}

	// Start server in background
	go func() {
		if err := testServer.Serve(listener); err != nil && err != http.ErrServerClosed {
			t.Errorf("Failed to start test server: %v", err)
		}
	}()

	// Wait a moment for server to start
	time.Sleep(10 * time.Millisecond)

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
// Step 3: Calls test.Clean() to clean up the base test environment
//
//	This closes database connections, cleans up temporary files, and resets global state
//
// IMPORTANT NOTES:
// - This function should ALWAYS be called with defer to ensure cleanup happens even if tests panic
// - Proper cleanup prevents test interference and resource leaks
// - The order of cleanup steps is important: HTTP server first, then OpenAPI cleanup, then base cleanup
// - Server shutdown has a 5-second timeout to prevent hanging tests
func Clean() {
	// Step 1: Gracefully shutdown the HTTP test server
	if testServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := testServer.Shutdown(ctx); err != nil {
			// Force close if graceful shutdown fails
			testServer.Close()
		}
		testServer = nil
	}

	// Step 2: Reset OpenAPI server instance to prevent state leakage
	Server = nil

	// Step 3: Clean up base test environment and all dependencies
	test.Clean()
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
	if Server == nil || Server.OAuth == nil {
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
	response, err := Server.OAuth.DynamicClientRegistration(ctx, req)
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
	if Server == nil || Server.OAuth == nil {
		// Server might already be cleaned up, which is OK
		return
	}

	if clientID == "" {
		return
	}

	// Delete the client using the OAuth service
	ctx := context.Background()
	err := Server.OAuth.DeleteClient(ctx, clientID)
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

func TestLoad(t *testing.T) {
	serverURL := Prepare(t)
	defer Clean()

	assert.NotNil(t, Server)
	assert.NotEmpty(t, serverURL)
	assert.Contains(t, serverURL, "http://127.0.0.1:")
}
