package openapi_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/openapi"
	"github.com/yaoapp/yao/openapi/tests/testutils"
	"github.com/yaoapp/yao/share"
)

func TestHelloWorldPublic(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	// Get base URL from server config
	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	tests := []struct {
		name   string
		method string
		path   string
	}{
		{
			name:   "GET public endpoint",
			method: "GET",
			path:   baseURL + "/helloworld/public",
		},
		{
			name:   "POST public endpoint",
			method: "POST",
			path:   baseURL + "/helloworld/public",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Make HTTP request
			var resp *http.Response
			var err error

			if tt.method == "GET" {
				resp, err = http.Get(serverURL + tt.path)
			} else {
				resp, err = http.Post(serverURL+tt.path, "application/json", nil)
			}

			assert.NoError(t, err)
			assert.NotNil(t, resp)
			defer resp.Body.Close()

			// Check status code
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			// Parse JSON response
			var response map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&response)
			assert.NoError(t, err)

			// Verify response structure and content
			assert.Equal(t, "HELLO, WORLD", response["MESSAGE"])
			assert.NotEmpty(t, response["SERVER_TIME"])
			assert.Equal(t, share.VERSION, response["VERSION"])
			assert.Equal(t, share.PRVERSION, response["PRVERSION"])
			assert.Equal(t, share.CUI, response["CUI"])
			assert.Equal(t, share.PRCUI, response["PRCUI"])
			assert.Equal(t, share.App.Name, response["APP"])
			assert.Equal(t, share.App.Version, response["APP_VERSION"])

			// Check that SERVER_TIME is a valid timestamp format
			serverTime, ok := response["SERVER_TIME"].(string)
			assert.True(t, ok)
			assert.NotEmpty(t, serverTime)
		})
	}
}

func TestHelloWorldProtected(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	// Get base URL from server config
	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Register a test client for authentication
	client := testutils.RegisterTestClient(t, "Hello World Protected Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)

	// Obtain access token for authentication
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	tests := []struct {
		name   string
		method string
		path   string
	}{
		{
			name:   "GET protected endpoint with valid token",
			method: "GET",
			path:   baseURL + "/helloworld/protected",
		},
		{
			name:   "POST protected endpoint with valid token",
			method: "POST",
			path:   baseURL + "/helloworld/protected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create HTTP request with Bearer token
			var req *http.Request
			var err error

			if tt.method == "GET" {
				req, err = http.NewRequest("GET", serverURL+tt.path, nil)
			} else {
				req, err = http.NewRequest("POST", serverURL+tt.path, nil)
				req.Header.Set("Content-Type", "application/json")
			}
			assert.NoError(t, err)

			// Add Bearer token for authentication
			req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

			// Make HTTP request
			resp, err := http.DefaultClient.Do(req)
			assert.NoError(t, err)
			assert.NotNil(t, resp)
			defer resp.Body.Close()

			// Check status code
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			// Parse JSON response
			var response map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&response)
			assert.NoError(t, err)

			// Verify response structure and content (same as public endpoint)
			assert.Equal(t, "HELLO, WORLD", response["MESSAGE"])
			assert.NotEmpty(t, response["SERVER_TIME"])
			assert.Equal(t, share.VERSION, response["VERSION"])
			assert.Equal(t, share.PRVERSION, response["PRVERSION"])
			assert.Equal(t, share.CUI, response["CUI"])
			assert.Equal(t, share.PRCUI, response["PRCUI"])
			assert.Equal(t, share.App.Name, response["APP"])
			assert.Equal(t, share.App.Version, response["APP_VERSION"])

			// Check that SERVER_TIME is a valid timestamp format
			serverTime, ok := response["SERVER_TIME"].(string)
			assert.True(t, ok)
			assert.NotEmpty(t, serverTime)

			t.Logf("Protected endpoint accessed successfully with token: %s", tokenInfo.AccessToken[:20]+"...")
		})
	}
}

func TestHelloWorldProtectedUnauthorized(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	// Get base URL from server config
	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	tests := []struct {
		name        string
		method      string
		path        string
		description string
	}{
		{
			name:        "GET protected endpoint without token",
			method:      "GET",
			path:        baseURL + "/helloworld/protected",
			description: "No Authorization header",
		},
		{
			name:        "POST protected endpoint without token",
			method:      "POST",
			path:        baseURL + "/helloworld/protected",
			description: "No Authorization header",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create HTTP request without Authorization header
			var req *http.Request
			var err error

			if tt.method == "GET" {
				req, err = http.NewRequest("GET", serverURL+tt.path, nil)
			} else {
				req, err = http.NewRequest("POST", serverURL+tt.path, nil)
				req.Header.Set("Content-Type", "application/json")
			}
			assert.NoError(t, err)

			// Make HTTP request (no Authorization header)
			resp, err := http.DefaultClient.Do(req)
			assert.NoError(t, err)
			assert.NotNil(t, resp)
			defer resp.Body.Close()

			// Should return 401 Unauthorized for protected endpoint without token
			assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

			t.Logf("Protected endpoint correctly rejected unauthorized request: %s", tt.description)
		})
	}
}
