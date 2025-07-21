package openapi

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/openapi/oauth/types"
)

func TestOAuthRegister(t *testing.T) {
	serverURL := Prepare(t)
	defer Clean()

	// Debug: Check if Server is properly initialized
	if Server == nil {
		t.Fatal("OpenAPI Server is nil")
	}

	if Server.Config == nil {
		t.Fatal("OpenAPI Server.Config is nil")
	}

	if Server.OAuth == nil {
		t.Fatal("OpenAPI Server.OAuth is nil")
	}

	t.Logf("Server initialized with BaseURL: %s", Server.Config.BaseURL)

	// Get base URL from server config
	baseURL := ""
	if Server != nil && Server.Config != nil {
		baseURL = Server.Config.BaseURL
	}

	endpoint := serverURL + baseURL + "/oauth/register"
	t.Logf("Testing endpoint: %s", endpoint)

	t.Run("Valid Client Registration", func(t *testing.T) {
		// Minimal valid registration request to isolate the issue
		req := types.DynamicClientRegistrationRequest{
			RedirectURIs: []string{
				"http://localhost/callback",
			},
			ClientName: "Test Client",
		}

		// Convert to JSON
		jsonData, err := json.Marshal(req)
		assert.NoError(t, err)
		t.Logf("Request JSON: %s", string(jsonData))

		// Make POST request
		t.Logf("Making POST request to: %s", endpoint)
		resp, err := http.Post(endpoint, "application/json", bytes.NewBuffer(jsonData))
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		t.Logf("Response status code: %d", resp.StatusCode)

		// Verify OAuth 2.1 security headers are present
		assert.Equal(t, "no-store", resp.Header.Get("Cache-Control"), "Cache-Control header should be set")
		assert.Equal(t, "no-cache", resp.Header.Get("Pragma"), "Pragma header should be set")
		assert.Equal(t, "nosniff", resp.Header.Get("X-Content-Type-Options"), "X-Content-Type-Options header should be set")
		assert.Equal(t, "DENY", resp.Header.Get("X-Frame-Options"), "X-Frame-Options header should be set")
		assert.Equal(t, "no-referrer", resp.Header.Get("Referrer-Policy"), "Referrer-Policy header should be set")
		assert.Equal(t, "application/json;charset=UTF-8", resp.Header.Get("Content-Type"), "Content-Type header should be set")

		// Read the complete response body for debugging
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Logf("Complete response body: %s", string(bodyBytes))

		// Reset the response body for JSON decoding
		resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))

		// Check status code
		if resp.StatusCode != http.StatusCreated {
			// Read error response for debugging
			body, _ := io.ReadAll(resp.Body)
			t.Logf("Expected 201, got %d. Response: %s", resp.StatusCode, string(body))
		}
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		// Parse response
		t.Logf("Parsing response body...")
		var response types.DynamicClientRegistrationResponse
		err = json.NewDecoder(resp.Body).Decode(&response)
		if err != nil {
			t.Logf("Failed to decode response: %v", err)
		}
		assert.NoError(t, err)

		t.Logf("Response ClientID: %s", response.ClientID)
		t.Logf("Response ClientSecret: %s", response.ClientSecret)

		// Verify response contains generated client credentials
		assert.NotEmpty(t, response.ClientID)
		assert.NotEmpty(t, response.ClientSecret)

		// Verify request data is preserved in response
		if response.DynamicClientRegistrationRequest != nil {
			assert.Equal(t, req.ClientName, response.DynamicClientRegistrationRequest.ClientName)
			assert.Equal(t, req.RedirectURIs, response.DynamicClientRegistrationRequest.RedirectURIs)

			// Verify that default values were applied when not specified in request
			assert.NotEmpty(t, response.DynamicClientRegistrationRequest.GrantTypes, "Server should apply default grant types")
			assert.NotEmpty(t, response.DynamicClientRegistrationRequest.ResponseTypes, "Server should apply default response types")
			assert.Equal(t, "web", response.DynamicClientRegistrationRequest.ApplicationType, "Server should apply default application type")
			assert.Equal(t, "client_secret_basic", response.DynamicClientRegistrationRequest.TokenEndpointAuthMethod, "Server should apply default auth method")
		}
	})
}
