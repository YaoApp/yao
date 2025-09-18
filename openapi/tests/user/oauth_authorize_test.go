package user_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/openapi"
	"github.com/yaoapp/yao/openapi/tests/testutils"
)

func TestUserOAuthAuthorizationURL(t *testing.T) {
	// Initialize test environment
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	// Get base URL from server config
	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Register a test client first (needed for user.Load validation)
	testClient := testutils.RegisterTestClient(t, "User OAuth Authorize Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, testClient.ClientID)

	// Note: user.Load is automatically called by openapi.Load in testutils.Prepare

	// Test OAuth authorization URL endpoints
	// Note: These should return 200 when OAuth client credentials are properly configured
	// (which they are in this test environment). Only nonexistent providers should return 404.
	testCases := []struct {
		name           string
		provider       string
		query          string
		expectCode     int
		expectErrorMsg string
	}{
		{"get google oauth url", "google", "", 200, ""},
		{"get microsoft oauth url", "microsoft", "", 200, ""},
		{"get apple oauth url", "apple", "", 200, ""},
		{"get github oauth url", "github", "", 200, ""},
		{"get oauth url with redirect_uri", "google", "?redirect_uri=https://example.com/callback", 200, ""},
		{"get oauth url with state", "google", "?state=test-state-123", 200, ""},
		{"get oauth url for nonexistent provider", "nonexistent", "", 404, "Failed to get provider"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			requestURL := serverURL + baseURL + "/user/oauth/" + tc.provider + "/authorize" + tc.query
			resp, err := http.Get(requestURL)
			assert.NoError(t, err, "HTTP request should succeed")

			if resp != nil {
				defer resp.Body.Close()
				assert.Equal(t, tc.expectCode, resp.StatusCode, "Expected status code %d", tc.expectCode)

				// Parse response body
				body, err := io.ReadAll(resp.Body)
				assert.NoError(t, err, "Should read response body")

				t.Logf("Response for %s: status=%d, body=%s", tc.provider, resp.StatusCode, string(body))

				var response map[string]interface{}
				err = json.Unmarshal(body, &response)
				assert.NoError(t, err, "Should parse JSON response")

				if tc.expectCode == 200 {
					// Success case - should have authorization_url
					if authURL, hasAuthURL := response["authorization_url"]; hasAuthURL {
						authURLStr, ok := authURL.(string)
						assert.True(t, ok, "authorization_url should be string")
						assert.NotEmpty(t, authURLStr, "authorization_url should not be empty")
						t.Logf("Authorization URL generated successfully for %s", tc.provider)

						// Verify the URL contains expected OAuth parameters
						assert.Contains(t, authURLStr, "client_id=", "Authorization URL should contain client_id")
						assert.Contains(t, authURLStr, "response_type=code", "Authorization URL should contain response_type=code")
						assert.Contains(t, authURLStr, "redirect_uri=", "Authorization URL should contain redirect_uri")
						assert.Contains(t, authURLStr, "state=", "Authorization URL should contain state")

						// Check for state in response
						if state, hasState := response["state"]; hasState {
							stateStr, ok := state.(string)
							assert.True(t, ok, "state should be string")
							assert.NotEmpty(t, stateStr, "state should not be empty")
							t.Logf("State generated: %s", stateStr)
						}

						// Check for warnings (optional)
						if warnings, hasWarnings := response["warnings"]; hasWarnings {
							warningsSlice, ok := warnings.([]interface{})
							if ok && len(warningsSlice) > 0 {
								t.Logf("Warnings: %v", warningsSlice)
							}
						}
					} else {
						t.Errorf("Success response should contain authorization_url field")
					}
				} else {
					// Error case - should have error fields
					if errorDescription, hasError := response["error_description"]; hasError {
						errorDescStr, ok := errorDescription.(string)
						assert.True(t, ok, "error_description should be string")
						if tc.expectErrorMsg != "" {
							assert.Contains(t, errorDescStr, tc.expectErrorMsg, "Error message should contain expected text")
						}
					} else {
						t.Errorf("Error response should contain error_description field")
					}

					// Verify error code is present
					if errorCode, hasErrorCode := response["error"]; hasErrorCode {
						assert.Equal(t, "invalid_request", errorCode, "Error code should be invalid_request")
					} else {
						t.Errorf("Error response should contain error field")
					}
				}
			}
		})
	}
}

func TestUserOAuthAuthorizationURLParameters(t *testing.T) {
	// Initialize test environment
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	// Get base URL from server config
	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Register a test client first (needed for user.Load validation)
	testClient := testutils.RegisterTestClient(t, "User OAuth URL Params Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, testClient.ClientID)

	// Note: user.Load is automatically called by openapi.Load in testutils.Prepare

	// Test various OAuth parameters
	testCases := []struct {
		name        string
		provider    string
		redirectURI string
		state       string
		expectCode  int
	}{
		{
			"with custom redirect_uri",
			"google",
			"https://myapp.example.com/callback",
			"",
			200,
		},
		{
			"with custom state",
			"google",
			"",
			"my-custom-state-12345",
			200,
		},
		{
			"with both redirect_uri and state",
			"google",
			"https://myapp.example.com/callback",
			"my-custom-state-12345",
			200,
		},
		{
			"with UUID state format",
			"google",
			"",
			"550e8400-e29b-41d4-a716-446655440000",
			200,
		},
		{
			"with non-UUID state format",
			"google",
			"",
			"simple-state",
			200,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Build query parameters
			query := ""
			params := []string{}
			if tc.redirectURI != "" {
				params = append(params, "redirect_uri="+tc.redirectURI)
			}
			if tc.state != "" {
				params = append(params, "state="+tc.state)
			}
			if len(params) > 0 {
				query = "?" + strings.Join(params, "&")
			}

			requestURL := serverURL + baseURL + "/user/oauth/" + tc.provider + "/authorize" + query
			resp, err := http.Get(requestURL)
			assert.NoError(t, err, "HTTP request should succeed")

			if resp != nil {
				defer resp.Body.Close()
				assert.Equal(t, tc.expectCode, resp.StatusCode, "Expected status code %d", tc.expectCode)

				// Parse response body
				body, err := io.ReadAll(resp.Body)
				assert.NoError(t, err, "Should read response body")

				var response map[string]interface{}
				err = json.Unmarshal(body, &response)
				assert.NoError(t, err, "Should parse JSON response")

				if tc.expectCode == 200 {
					// Verify authorization URL is generated
					if authURL, hasAuthURL := response["authorization_url"]; hasAuthURL {
						authURLStr, ok := authURL.(string)
						assert.True(t, ok, "authorization_url should be string")
						assert.NotEmpty(t, authURLStr, "authorization_url should not be empty")

						// Verify custom parameters are included in the URL
						if tc.redirectURI != "" {
							// Parse the authorization URL and check parameters
							parsedURL, err := url.Parse(authURLStr)
							assert.NoError(t, err, "Authorization URL should be valid")

							// Check if redirect_uri parameter matches
							redirectURI := parsedURL.Query().Get("redirect_uri")
							assert.Equal(t, tc.redirectURI, redirectURI, "Authorization URL should contain custom redirect_uri")
						}

						// Verify state parameter
						if state, hasState := response["state"]; hasState {
							stateStr, ok := state.(string)
							assert.True(t, ok, "state should be string")
							assert.NotEmpty(t, stateStr, "state should not be empty")

							if tc.state != "" {
								assert.Equal(t, tc.state, stateStr, "State should match provided state")
							}

							// Check for warnings about non-UUID state
							if warnings, hasWarnings := response["warnings"]; hasWarnings {
								warningsSlice, ok := warnings.([]interface{})
								if ok {
									t.Logf("Warnings for state '%s': %v", stateStr, warningsSlice)
								}
							}
						}

						t.Logf("Test %s passed: URL=%s", tc.name, authURLStr)
					}
				}
			}
		})
	}
}
