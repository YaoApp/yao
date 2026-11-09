package openapi_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/openapi"
	"github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/openapi/tests/testutils"
)

func TestOAuthRegister(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	// Debug: Check if openapi.Server is properly initialized
	if openapi.Server == nil {
		t.Fatal("OpenAPI openapi.Server is nil")
	}

	if openapi.Server.Config == nil {
		t.Fatal("OpenAPI openapi.Server.Config is nil")
	}

	if openapi.Server.OAuth == nil {
		t.Fatal("OpenAPI openapi.Server.OAuth is nil")
	}

	t.Logf("openapi.Server initialized with BaseURL: %s", openapi.Server.Config.BaseURL)

	// Get base URL from server config
	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
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
		assert.Equal(t, "application/json", resp.Header.Get("Content-Type"), "Content-Type header should be set")

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
			assert.NotEmpty(t, response.DynamicClientRegistrationRequest.GrantTypes, "openapi.Server should apply default grant types")
			assert.NotEmpty(t, response.DynamicClientRegistrationRequest.ResponseTypes, "openapi.Server should apply default response types")
			assert.Equal(t, "web", response.DynamicClientRegistrationRequest.ApplicationType, "openapi.Server should apply default application type")
			assert.Equal(t, "client_secret_basic", response.DynamicClientRegistrationRequest.TokenEndpointAuthMethod, "openapi.Server should apply default auth method")
		}
	})
}

func TestOAuthAuthorize(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	// Register a test client for realistic testing
	testClient := testutils.RegisterTestClient(t, "OAuth Test Client", []string{"http://localhost/callback"})
	defer testutils.CleanupTestClient(t, testClient.ClientID)

	// testutils.Prepare test data
	endpoint := serverURL + openapi.Server.Config.BaseURL + "/oauth/authorize"
	t.Logf("Testing authorize endpoint: %s", endpoint)

	t.Run("Valid Authorization Request", func(t *testing.T) {
		// Test valid authorization request with real client
		params := url.Values{}
		params.Set("client_id", testClient.ClientID) // Use real registered client ID
		params.Set("response_type", "code")
		params.Set("redirect_uri", testClient.RedirectURIs[0]) // Use registered redirect URI
		params.Set("scope", "openid profile")
		params.Set("state", "test-state-123")

		requestURL := endpoint + "?" + params.Encode()
		t.Logf("Making GET request to: %s", requestURL)

		// Configure HTTP client to not follow redirects automatically
		client := &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}

		resp, err := client.Get(requestURL)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		t.Logf("Response status code: %d", resp.StatusCode)

		// Should redirect with either success (302) or error (302)
		assert.Equal(t, http.StatusFound, resp.StatusCode)

		// Check redirect location
		location := resp.Header.Get("Location")
		assert.NotEmpty(t, location, "Location header should be present")
		t.Logf("Redirect location: %s", location)

		// Parse redirect URL to check parameters
		redirectURL, err := url.Parse(location)
		assert.NoError(t, err)

		// Should contain either 'code' (success) or 'error' (failure) parameter
		query := redirectURL.Query()
		hasCode := query.Get("code") != ""
		hasError := query.Get("error") != ""
		assert.True(t, hasCode || hasError, "Redirect should contain either 'code' or 'error' parameter")

		// State parameter should be preserved
		assert.Equal(t, "test-state-123", query.Get("state"), "State parameter should be preserved")

		t.Logf("Authorization result - Code: %s, Error: %s", query.Get("code"), query.Get("error"))
	})

	t.Run("Invalid Client ID", func(t *testing.T) {
		// Test with invalid client ID
		params := url.Values{}
		params.Set("client_id", "invalid-client-id")
		params.Set("response_type", "code")
		params.Set("redirect_uri", "http://localhost/callback")
		params.Set("scope", "openid profile")
		params.Set("state", "test-state-456")

		requestURL := endpoint + "?" + params.Encode()

		client := &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}

		resp, err := client.Get(requestURL)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusFound, resp.StatusCode)

		location := resp.Header.Get("Location")
		redirectURL, err := url.Parse(location)
		assert.NoError(t, err)

		query := redirectURL.Query()
		assert.Equal(t, "invalid_client", query.Get("error"), "Should return invalid_client error")
		assert.Equal(t, "test-state-456", query.Get("state"), "State should be preserved")
	})

	t.Run("Valid Authorization Request via POST", func(t *testing.T) {
		// Test valid authorization request with POST method
		form := url.Values{}
		form.Set("client_id", testClient.ClientID)
		form.Set("response_type", "code")
		form.Set("redirect_uri", testClient.RedirectURIs[0])
		form.Set("scope", "openid profile")
		form.Set("state", "test-post-state-789")

		t.Logf("Making POST request to: %s", endpoint)

		// Configure HTTP client to not follow redirects automatically
		client := &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}

		resp, err := client.PostForm(endpoint, form)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		t.Logf("Response status code: %d", resp.StatusCode)

		// Should redirect with either success (302) or error (302)
		assert.Equal(t, http.StatusFound, resp.StatusCode)

		// Check redirect location
		location := resp.Header.Get("Location")
		assert.NotEmpty(t, location, "Location header should be present")
		t.Logf("Redirect location: %s", location)

		// Parse redirect URL to check parameters
		redirectURL, err := url.Parse(location)
		assert.NoError(t, err)

		// Should contain either 'code' (success) or 'error' (failure) parameter
		query := redirectURL.Query()
		hasCode := query.Get("code") != ""
		hasError := query.Get("error") != ""
		assert.True(t, hasCode || hasError, "Redirect should contain either 'code' or 'error' parameter")

		// State parameter should be preserved
		assert.Equal(t, "test-post-state-789", query.Get("state"), "State parameter should be preserved")

		t.Logf("Authorization result (POST) - Code: %s, Error: %s", query.Get("code"), query.Get("error"))
	})

	t.Run("Invalid Response Type via POST", func(t *testing.T) {
		// Test with invalid response type using POST
		form := url.Values{}
		form.Set("client_id", testClient.ClientID)
		form.Set("response_type", "token") // Implicit flow - deprecated in OAuth 2.1
		form.Set("redirect_uri", testClient.RedirectURIs[0])
		form.Set("scope", "openid profile")
		form.Set("state", "test-invalid-response-type")

		client := &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}

		resp, err := client.PostForm(endpoint, form)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusFound, resp.StatusCode)

		location := resp.Header.Get("Location")
		redirectURL, err := url.Parse(location)
		assert.NoError(t, err)

		query := redirectURL.Query()
		assert.Equal(t, "unsupported_response_type", query.Get("error"), "Should return unsupported_response_type error")
		assert.Equal(t, "test-invalid-response-type", query.Get("state"), "State should be preserved")
	})

	t.Run("Missing Required Parameters via POST", func(t *testing.T) {
		// Test with missing client_id using POST
		form := url.Values{}
		// Missing client_id
		form.Set("response_type", "code")
		form.Set("redirect_uri", testClient.RedirectURIs[0])
		form.Set("scope", "openid profile")
		form.Set("state", "test-missing-client-id")

		client := &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}

		resp, err := client.PostForm(endpoint, form)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusFound, resp.StatusCode)

		location := resp.Header.Get("Location")
		redirectURL, err := url.Parse(location)
		assert.NoError(t, err)

		query := redirectURL.Query()
		assert.Equal(t, "invalid_request", query.Get("error"), "Should return invalid_request error")
		assert.Equal(t, "test-missing-client-id", query.Get("state"), "State should be preserved")
	})
}

func TestOAuthJWKS(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	// Debug: Check if openapi.Server is properly initialized
	if openapi.Server == nil {
		t.Fatal("OpenAPI openapi.Server is nil")
	}

	if openapi.Server.Config == nil {
		t.Fatal("OpenAPI openapi.Server.Config is nil")
	}

	if openapi.Server.OAuth == nil {
		t.Fatal("OpenAPI openapi.Server.OAuth is nil")
	}

	t.Logf("openapi.Server initialized with BaseURL: %s", openapi.Server.Config.BaseURL)

	// Get base URL from server config
	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	endpoint := serverURL + baseURL + "/oauth/jwks"
	t.Logf("Testing JWKS endpoint: %s", endpoint)

	t.Run("Valid JWKS Request", func(t *testing.T) {
		// Make GET request to JWKS endpoint
		resp, err := http.Get(endpoint)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		t.Logf("Response status code: %d", resp.StatusCode)

		// Should return 200 OK
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Verify Content-Type header
		contentType := resp.Header.Get("Content-Type")
		assert.Equal(t, "application/json", contentType, "Content-Type should be application/json")

		// Verify OAuth 2.1 security headers are present
		assert.Equal(t, "no-store", resp.Header.Get("Cache-Control"), "Cache-Control header should be set")
		assert.Equal(t, "no-cache", resp.Header.Get("Pragma"), "Pragma header should be set")
		assert.Equal(t, "nosniff", resp.Header.Get("X-Content-Type-Options"), "X-Content-Type-Options header should be set")
		assert.Equal(t, "DENY", resp.Header.Get("X-Frame-Options"), "X-Frame-Options header should be set")
		assert.Equal(t, "no-referrer", resp.Header.Get("Referrer-Policy"), "Referrer-Policy header should be set")

		// Read and parse response body
		bodyBytes, err := io.ReadAll(resp.Body)
		assert.NoError(t, err)
		t.Logf("JWKS response body: %s", string(bodyBytes))

		// Parse JWKS response directly as per RFC 7517
		var jwks types.JWKSResponse
		err = json.Unmarshal(bodyBytes, &jwks)
		assert.NoError(t, err, "Response should be valid JWKS JSON")

		// Verify JWKS structure
		assert.NotNil(t, jwks.Keys, "JWKS should have keys array")
		assert.Equal(t, 1, len(jwks.Keys), "Should have exactly 1 key (matching 1 certificate pair)")

		// Verify the single JWK entry
		jwk := jwks.Keys[0]

		// Verify required JWK fields
		assert.Equal(t, "RSA", jwk.Kty, "Key type should be RSA")
		assert.Equal(t, "sig", jwk.Use, "Key use should be sig (signature)")
		assert.NotEmpty(t, jwk.Kid, "Key ID should not be empty")
		assert.Equal(t, "RS256", jwk.Alg, "Algorithm should be RS256")
		assert.NotEmpty(t, jwk.N, "RSA modulus (n) should not be empty")
		assert.NotEmpty(t, jwk.E, "RSA exponent (e) should not be empty")

		t.Logf("JWK Details - Kty: %s, Use: %s, Kid: %s, Alg: %s", jwk.Kty, jwk.Use, jwk.Kid, jwk.Alg)
		t.Logf("RSA Modulus length: %d, Exponent: %s", len(jwk.N), jwk.E)

		// Verify base64url encoding (basic validation)
		// Base64URL should not contain padding or invalid characters
		assert.NotContains(t, jwk.N, "=", "RSA modulus should be base64url encoded (no padding)")
		assert.NotContains(t, jwk.E, "=", "RSA exponent should be base64url encoded (no padding)")
		assert.NotContains(t, jwk.N, "+", "RSA modulus should be base64url encoded (no + chars)")
		assert.NotContains(t, jwk.E, "+", "RSA exponent should be base64url encoded (no + chars)")
		assert.NotContains(t, jwk.N, "/", "RSA modulus should be base64url encoded (no / chars)")
		assert.NotContains(t, jwk.E, "/", "RSA exponent should be base64url encoded (no / chars)")

		// Verify optional JWK fields are not present (as they're not needed for basic JWT signing)
		assert.Empty(t, jwk.D, "Private key components should not be exposed in JWKS")
		assert.Empty(t, jwk.P, "Private key components should not be exposed in JWKS")
		assert.Empty(t, jwk.Q, "Private key components should not be exposed in JWKS")
		assert.Empty(t, jwk.DP, "Private key components should not be exposed in JWKS")
		assert.Empty(t, jwk.DQ, "Private key components should not be exposed in JWKS")
		assert.Empty(t, jwk.QI, "Private key components should not be exposed in JWKS")
	})

	t.Run("JWKS Response Format Compliance", func(t *testing.T) {
		// Test that JWKS response is RFC 7517 compliant
		resp, err := http.Get(endpoint)
		assert.NoError(t, err)
		defer resp.Body.Close()

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)

		// RFC 7517: JWKS MUST have "keys" member
		keys, exists := response["keys"]
		assert.True(t, exists, "JWKS must have 'keys' member")

		// Keys should be an array
		keysArray, ok := keys.([]interface{})
		assert.True(t, ok, "Keys should be an array")
		assert.Equal(t, 1, len(keysArray), "Should have exactly one key")

		// Verify the key is a JSON object
		keyObj, ok := keysArray[0].(map[string]interface{})
		assert.True(t, ok, "Key should be a JSON object")

		// Verify required RSA JWK parameters are present
		requiredParams := []string{"kty", "use", "kid", "alg", "n", "e"}
		for _, param := range requiredParams {
			_, exists := keyObj[param]
			assert.True(t, exists, "JWK should have required parameter: %s", param)
		}
	})

	t.Run("JWKS Endpoint Security Headers", func(t *testing.T) {
		// Test that security headers are properly set for JWKS endpoint
		resp, err := http.Get(endpoint)
		assert.NoError(t, err)
		defer resp.Body.Close()

		// Verify all required security headers for OAuth 2.1 compliance
		expectedHeaders := map[string]string{
			"Cache-Control":          "no-store",
			"Pragma":                 "no-cache",
			"X-Content-Type-Options": "nosniff",
			"X-Frame-Options":        "DENY",
			"Referrer-Policy":        "no-referrer",
			"Content-Type":           "application/json",
		}

		for header, expectedValue := range expectedHeaders {
			actualValue := resp.Header.Get(header)
			assert.Equal(t, expectedValue, actualValue, "Header %s should be set correctly", header)
		}
	})
}
