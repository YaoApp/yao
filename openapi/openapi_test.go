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

func TestLoad(t *testing.T) {
	serverURL := Prepare(t)
	defer Clean()

	assert.NotNil(t, Server)
	assert.NotEmpty(t, serverURL)
	assert.Contains(t, serverURL, "http://127.0.0.1:")
}
