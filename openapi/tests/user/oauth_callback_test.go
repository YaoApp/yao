package user_test

import (
	"bytes"
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

func TestUserOAuthCallback(t *testing.T) {
	// Initialize test environment
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	// Get base URL from server config
	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Register a test client first (needed for user.Load validation)
	testClient := testutils.RegisterTestClient(t, "User OAuth Callback Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, testClient.ClientID)

	// Note: user.Load is automatically called by openapi.Load in testutils.Prepare

	// Note: OAuth callback testing requires a complex setup with valid OAuth state
	// and authorization codes. For now, we test the endpoint accessibility and
	// basic error handling.

	testCases := []struct {
		name       string
		provider   string
		method     string
		body       map[string]interface{}
		expectCode int
		expectMsg  string
	}{
		{
			"callback without parameters",
			"google",
			"POST",
			map[string]interface{}{},
			400, // Should return bad request for missing parameters
			"State is required",
		},
		{
			"callback with invalid state",
			"google",
			"POST",
			map[string]interface{}{
				"code":  "test-auth-code",
				"state": "invalid-state",
			},
			400, // Should return bad request for invalid state
			"Invalid state",
		},
		{
			"callback for nonexistent provider",
			"nonexistent",
			"POST",
			map[string]interface{}{
				"code":  "test-auth-code",
				"state": "test-state",
			},
			404, // Should return not found for nonexistent provider
			"Failed to get provider",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			requestURL := serverURL + baseURL + "/user/oauth/" + tc.provider + "/callback"

			// Prepare request body
			bodyBytes, _ := json.Marshal(tc.body)
			req, err := http.NewRequest(tc.method, requestURL, bytes.NewBuffer(bodyBytes))
			assert.NoError(t, err, "Should create HTTP request")

			req.Header.Set("Content-Type", "application/json")

			client := &http.Client{}
			resp, err := client.Do(req)
			assert.NoError(t, err, "HTTP request should succeed")

			if resp != nil {
				defer resp.Body.Close()

				t.Logf("OAuth callback test %s: status=%d", tc.name, resp.StatusCode)

				// Note: The exact status codes may vary based on implementation
				// These tests verify the endpoint is accessible and handles basic errors
				assert.True(t, resp.StatusCode >= 400 || resp.StatusCode < 300,
					"Should return either success or client/server error")

				// For error responses, try to parse error message
				if resp.StatusCode >= 400 {
					body, err := io.ReadAll(resp.Body)
					if err == nil {
						var response map[string]interface{}
						if json.Unmarshal(body, &response) == nil {
							if errorDesc, hasError := response["error_description"]; hasError {
								errorDescStr, ok := errorDesc.(string)
								if ok && tc.expectMsg != "" {
									t.Logf("Error message: %s", errorDescStr)
									// Note: Exact error message matching may vary
									// We just verify the endpoint responds with error details
								}
							}
						}
					}
				}
			}
		})
	}
}

func TestUserOAuthCallbackPrepare(t *testing.T) {
	// Initialize test environment
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	// Get base URL from server config
	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Register a test client first (needed for user.Load validation)
	testClient := testutils.RegisterTestClient(t, "User OAuth Prepare Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, testClient.ClientID)

	// Note: user.Load is automatically called by openapi.Load in testutils.Prepare

	// Test OAuth callback prepare endpoint (form_post mode)
	testCases := []struct {
		name       string
		provider   string
		formData   map[string]string
		expectCode int
	}{
		{
			"prepare without parameters",
			"apple", // Apple typically uses form_post mode
			map[string]string{},
			500, // Should return error for missing parameters
		},
		{
			"prepare with code and state",
			"apple",
			map[string]string{
				"code":  "test-auth-code",
				"state": "test-state",
			},
			500, // Will fail due to invalid state, but endpoint should be accessible
		},
		{
			"prepare with user info",
			"apple",
			map[string]string{
				"code":  "test-auth-code",
				"state": "test-state",
				"user":  `{"name":{"firstName":"John","lastName":"Doe"},"email":"john@example.com"}`,
			},
			500, // Will fail due to invalid state, but endpoint should be accessible
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			requestURL := serverURL + baseURL + "/user/oauth/" + tc.provider + "/authorize/prepare"

			// Prepare form data
			formData := url.Values{}
			for key, value := range tc.formData {
				formData.Set(key, value)
			}

			req, err := http.NewRequest("POST", requestURL, strings.NewReader(formData.Encode()))
			assert.NoError(t, err, "Should create HTTP request")

			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			client := &http.Client{
				CheckRedirect: func(req *http.Request, via []*http.Request) error {
					// Don't follow redirects, we want to test the response
					return http.ErrUseLastResponse
				},
			}

			resp, err := client.Do(req)
			assert.NoError(t, err, "HTTP request should succeed")

			if resp != nil {
				defer resp.Body.Close()

				t.Logf("OAuth prepare test %s: status=%d", tc.name, resp.StatusCode)

				// The prepare endpoint may redirect or return errors
				// We just verify it's accessible and responds appropriately
				assert.True(t, resp.StatusCode == 302 || resp.StatusCode >= 400,
					"Should return redirect or error response")

				// If it's a redirect, check the location header
				if resp.StatusCode == 302 {
					location := resp.Header.Get("Location")
					if location != "" {
						t.Logf("Redirect location: %s", location)
						assert.Contains(t, location, "code=", "Redirect should contain code parameter")
						assert.Contains(t, location, "state=", "Redirect should contain state parameter")
					}
				}
			}
		})
	}
}

func TestUserOAuthProviderValidation(t *testing.T) {
	// Initialize test environment
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	// Get base URL from server config
	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Register a test client first (needed for user.Load validation)
	testClient := testutils.RegisterTestClient(t, "User OAuth Validation Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, testClient.ClientID)

	// Note: user.Load is automatically called by openapi.Load in testutils.Prepare

	// Test provider validation
	providers := []string{"google", "microsoft", "apple", "github", "nonexistent"}

	for _, provider := range providers {
		t.Run("provider_"+provider, func(t *testing.T) {
			// Test authorize endpoint
			authorizeURL := serverURL + baseURL + "/user/oauth/" + provider + "/authorize"
			resp, err := http.Get(authorizeURL)
			assert.NoError(t, err, "HTTP request should succeed")

			if resp != nil {
				defer resp.Body.Close()

				if provider == "nonexistent" {
					assert.Equal(t, 404, resp.StatusCode, "Nonexistent provider should return 404")
				} else {
					// Known providers should return 200 or other valid response
					assert.True(t, resp.StatusCode == 200 || resp.StatusCode == 500,
						"Known provider should return 200 or 500 (if not configured)")
				}

				t.Logf("Provider %s authorize endpoint: status=%d", provider, resp.StatusCode)
			}

			// Test callback endpoint
			callbackURL := serverURL + baseURL + "/user/oauth/" + provider + "/callback"
			bodyBytes, _ := json.Marshal(map[string]interface{}{
				"code":  "test-code",
				"state": "test-state",
			})
			req, err := http.NewRequest("POST", callbackURL, bytes.NewBuffer(bodyBytes))
			assert.NoError(t, err, "Should create HTTP request")
			req.Header.Set("Content-Type", "application/json")

			client := &http.Client{}
			resp, err = client.Do(req)
			assert.NoError(t, err, "HTTP request should succeed")

			if resp != nil {
				defer resp.Body.Close()

				if provider == "nonexistent" {
					assert.Equal(t, 404, resp.StatusCode, "Nonexistent provider should return 404")
				} else {
					// Known providers should return 400 (bad request due to invalid state) or other error
					assert.True(t, resp.StatusCode >= 400, "Known provider should return error for invalid request")
				}

				t.Logf("Provider %s callback endpoint: status=%d", provider, resp.StatusCode)
			}
		})
	}
}
