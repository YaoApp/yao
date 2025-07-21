package openapi

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/openapi/oauth/types"
)

func TestOAuthToken_AuthorizationCode(t *testing.T) {
	serverURL := Prepare(t)
	defer Clean()

	// Get base URL from server config
	baseURL := ""
	if Server != nil && Server.Config != nil {
		baseURL = Server.Config.BaseURL
	}

	// Register a test client
	client := RegisterTestClient(t, "Token Test Client", []string{"https://localhost/callback"})
	defer CleanupTestClient(t, client.ClientID)

	// Obtain authorization code dynamically
	authInfo := ObtainAuthorizationCode(t, serverURL, client.ClientID, "https://localhost/callback", "openid profile")

	// Test authorization code grant
	t.Run("Valid Authorization Code Grant", func(t *testing.T) {
		// Prepare token request
		data := url.Values{}
		data.Set("grant_type", "authorization_code")
		data.Set("code", authInfo.Code)
		data.Set("redirect_uri", authInfo.RedirectURI)
		data.Set("client_id", client.ClientID)

		// Make token request
		endpoint := serverURL + baseURL + "/oauth/token"
		req, err := http.NewRequest("POST", endpoint, bytes.NewBufferString(data.Encode()))
		assert.NoError(t, err)

		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Authorization", "Basic "+basicAuth(client.ClientID, client.ClientSecret))

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		// Verify response
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Verify OAuth 2.1 security headers
		assert.Equal(t, "no-store", resp.Header.Get("Cache-Control"))
		assert.Equal(t, "no-cache", resp.Header.Get("Pragma"))
		assert.Equal(t, "application/json;charset=UTF-8", resp.Header.Get("Content-Type"))

		// Parse response
		var tokenResp types.Token
		err = json.NewDecoder(resp.Body).Decode(&tokenResp)
		assert.NoError(t, err)

		// Verify token response
		assert.NotEmpty(t, tokenResp.AccessToken)
		assert.Equal(t, "Bearer", tokenResp.TokenType)
		assert.Greater(t, tokenResp.ExpiresIn, 0)
		assert.NotEmpty(t, tokenResp.RefreshToken) // Should have refresh token for authorization code grant

		t.Logf("Token response: AccessToken=%s, TokenType=%s, ExpiresIn=%d",
			tokenResp.AccessToken, tokenResp.TokenType, tokenResp.ExpiresIn)
	})

	t.Run("Invalid Authorization Code", func(t *testing.T) {
		// Test with invalid authorization code - should return error

		// Prepare token request with invalid code
		data := url.Values{}
		data.Set("grant_type", "authorization_code")
		data.Set("code", "invalid-code")
		data.Set("redirect_uri", authInfo.RedirectURI)
		data.Set("client_id", client.ClientID)

		// Make token request
		endpoint := serverURL + baseURL + "/oauth/token"
		req, err := http.NewRequest("POST", endpoint, bytes.NewBufferString(data.Encode()))
		assert.NoError(t, err)

		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Authorization", "Basic "+basicAuth(client.ClientID, client.ClientSecret))

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		// Should return error for invalid authorization code
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		// Verify OAuth 2.1 security headers
		assert.Equal(t, "no-store", resp.Header.Get("Cache-Control"))
		assert.Equal(t, "no-cache", resp.Header.Get("Pragma"))
	})

	t.Run("Missing Required Parameters", func(t *testing.T) {
		// Prepare token request missing redirect_uri
		data := url.Values{}
		data.Set("grant_type", "authorization_code")
		data.Set("code", authInfo.Code)
		// Missing redirect_uri
		data.Set("client_id", client.ClientID)

		// Make token request
		endpoint := serverURL + baseURL + "/oauth/token"
		req, err := http.NewRequest("POST", endpoint, bytes.NewBufferString(data.Encode()))
		assert.NoError(t, err)

		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Authorization", "Basic "+basicAuth(client.ClientID, client.ClientSecret))

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		// Should return error
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

func TestOAuthToken_ClientCredentials(t *testing.T) {
	serverURL := Prepare(t)
	defer Clean()

	// Get base URL from server config
	baseURL := ""
	if Server != nil && Server.Config != nil {
		baseURL = Server.Config.BaseURL
	}

	// Register a test client for client credentials
	client := RegisterTestClient(t, "Client Credentials Test", []string{"https://localhost/callback"})
	defer CleanupTestClient(t, client.ClientID)

	t.Run("Valid Client Credentials Grant", func(t *testing.T) {
		// Prepare token request
		data := url.Values{}
		data.Set("grant_type", "client_credentials")
		data.Set("scope", "api:read api:write")

		// Make token request
		endpoint := serverURL + baseURL + "/oauth/token"
		req, err := http.NewRequest("POST", endpoint, bytes.NewBufferString(data.Encode()))
		assert.NoError(t, err)

		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Authorization", "Basic "+basicAuth(client.ClientID, client.ClientSecret))

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		// Debug: Print response body on failure
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Logf("Client credentials grant failed with status %d: %s", resp.StatusCode, string(body))
		}

		// Verify response
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Verify OAuth 2.1 security headers
		assert.Equal(t, "no-store", resp.Header.Get("Cache-Control"))
		assert.Equal(t, "no-cache", resp.Header.Get("Pragma"))

		// Parse response
		var tokenResp types.Token
		err = json.NewDecoder(resp.Body).Decode(&tokenResp)
		assert.NoError(t, err)

		// Verify token response
		assert.NotEmpty(t, tokenResp.AccessToken)
		assert.Equal(t, "Bearer", tokenResp.TokenType)
		assert.Greater(t, tokenResp.ExpiresIn, 0)
		// Client credentials grant should NOT have refresh token
		assert.Empty(t, tokenResp.RefreshToken)

		t.Logf("Client credentials token: AccessToken=%s, TokenType=%s, ExpiresIn=%d",
			tokenResp.AccessToken, tokenResp.TokenType, tokenResp.ExpiresIn)
	})

	t.Run("Client Credentials Without Authentication", func(t *testing.T) {
		// Prepare token request
		data := url.Values{}
		data.Set("grant_type", "client_credentials")

		// Make token request WITHOUT client authentication
		endpoint := serverURL + baseURL + "/oauth/token"
		req, err := http.NewRequest("POST", endpoint, bytes.NewBufferString(data.Encode()))
		assert.NoError(t, err)

		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		// No Authorization header

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		// Should return error - client credentials grant requires authentication
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

func TestOAuthToken_RefreshToken(t *testing.T) {
	serverURL := Prepare(t)
	defer Clean()

	// Get base URL from server config
	baseURL := ""
	if Server != nil && Server.Config != nil {
		baseURL = Server.Config.BaseURL
	}

	// Register a test client
	client := RegisterTestClient(t, "Refresh Token Test Client", []string{"https://localhost/callback"})
	defer CleanupTestClient(t, client.ClientID)

	// First, get an access token and refresh token using authorization code
	authInfo := ObtainAuthorizationCode(t, serverURL, client.ClientID, "https://localhost/callback", "openid profile")

	// Get initial token
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", authInfo.Code)
	data.Set("redirect_uri", authInfo.RedirectURI)
	data.Set("client_id", client.ClientID)

	endpoint := serverURL + baseURL + "/oauth/token"
	req, err := http.NewRequest("POST", endpoint, bytes.NewBufferString(data.Encode()))
	assert.NoError(t, err)

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Basic "+basicAuth(client.ClientID, client.ClientSecret))

	resp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var initialToken types.Token
	err = json.NewDecoder(resp.Body).Decode(&initialToken)
	assert.NoError(t, err)
	assert.NotEmpty(t, initialToken.RefreshToken)

	t.Run("Valid Refresh Token Grant", func(t *testing.T) {
		// Prepare refresh token request
		data := url.Values{}
		data.Set("grant_type", "refresh_token")
		data.Set("refresh_token", initialToken.RefreshToken)
		data.Set("scope", "openid profile") // Same or narrower scope

		// Make refresh token request
		req, err := http.NewRequest("POST", endpoint, bytes.NewBufferString(data.Encode()))
		assert.NoError(t, err)

		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Authorization", "Basic "+basicAuth(client.ClientID, client.ClientSecret))

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		// Debug: Print response body on failure
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Logf("Refresh token grant failed with status %d: %s", resp.StatusCode, string(body))
		}

		// Verify response
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Verify OAuth 2.1 security headers
		assert.Equal(t, "no-store", resp.Header.Get("Cache-Control"))
		assert.Equal(t, "no-cache", resp.Header.Get("Pragma"))

		// Parse response
		var refreshResp types.RefreshTokenResponse
		err = json.NewDecoder(resp.Body).Decode(&refreshResp)
		assert.NoError(t, err)

		// Verify refresh token response
		assert.NotEmpty(t, refreshResp.AccessToken)
		assert.Equal(t, "Bearer", refreshResp.TokenType)
		assert.Greater(t, refreshResp.ExpiresIn, 0)
		assert.Equal(t, "openid profile", refreshResp.Scope)

		// New access token should be different from original
		assert.NotEqual(t, initialToken.AccessToken, refreshResp.AccessToken)

		t.Logf("Refresh token response: AccessToken=%s, TokenType=%s, ExpiresIn=%d",
			refreshResp.AccessToken, refreshResp.TokenType, refreshResp.ExpiresIn)
	})

	t.Run("Invalid Refresh Token", func(t *testing.T) {
		// Prepare refresh token request with invalid token
		data := url.Values{}
		data.Set("grant_type", "refresh_token")
		data.Set("refresh_token", "invalid-refresh-token")

		// Make refresh token request
		req, err := http.NewRequest("POST", endpoint, bytes.NewBufferString(data.Encode()))
		assert.NoError(t, err)

		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Authorization", "Basic "+basicAuth(client.ClientID, client.ClientSecret))

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		// Should return error
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("Missing Refresh Token", func(t *testing.T) {
		// Prepare refresh token request without refresh_token parameter
		data := url.Values{}
		data.Set("grant_type", "refresh_token")
		// Missing refresh_token

		// Make refresh token request
		req, err := http.NewRequest("POST", endpoint, bytes.NewBufferString(data.Encode()))
		assert.NoError(t, err)

		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Authorization", "Basic "+basicAuth(client.ClientID, client.ClientSecret))

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		// Should return error
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

func TestOAuthToken_InvalidGrantType(t *testing.T) {
	serverURL := Prepare(t)
	defer Clean()

	// Get base URL from server config
	baseURL := ""
	if Server != nil && Server.Config != nil {
		baseURL = Server.Config.BaseURL
	}

	client := RegisterTestClient(t, "Invalid Grant Test", []string{"https://localhost/callback"})
	defer CleanupTestClient(t, client.ClientID)

	t.Run("Unsupported Grant Type", func(t *testing.T) {
		// Prepare token request with unsupported grant type
		data := url.Values{}
		data.Set("grant_type", "unsupported_grant_type")

		// Make token request
		endpoint := serverURL + baseURL + "/oauth/token"
		req, err := http.NewRequest("POST", endpoint, bytes.NewBufferString(data.Encode()))
		assert.NoError(t, err)

		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Authorization", "Basic "+basicAuth(client.ClientID, client.ClientSecret))

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		// Should return unsupported_grant_type error
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		// Verify OAuth 2.1 security headers even for errors
		assert.Equal(t, "no-store", resp.Header.Get("Cache-Control"))
		assert.Equal(t, "no-cache", resp.Header.Get("Pragma"))
	})

	t.Run("Missing Grant Type", func(t *testing.T) {
		// Prepare token request without grant_type
		data := url.Values{}
		// Missing grant_type

		// Make token request
		endpoint := serverURL + baseURL + "/oauth/token"
		req, err := http.NewRequest("POST", endpoint, bytes.NewBufferString(data.Encode()))
		assert.NoError(t, err)

		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Authorization", "Basic "+basicAuth(client.ClientID, client.ClientSecret))

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		// Should return invalid_request error
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

// Helper function to create Basic Auth header
func basicAuth(username, password string) string {
	auth := username + ":" + password
	return base64Encode([]byte(auth))
}

// Simple base64 encoding helper
func base64Encode(data []byte) string {
	const base64Table = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"

	if len(data) == 0 {
		return ""
	}

	// Calculate output length
	outputLen := ((len(data) + 2) / 3) * 4
	result := make([]byte, outputLen)

	for i, j := 0, 0; i < len(data); i += 3 {
		// Get 3 bytes (or less for the last group)
		b1 := data[i]
		var b2, b3 byte
		if i+1 < len(data) {
			b2 = data[i+1]
		}
		if i+2 < len(data) {
			b3 = data[i+2]
		}

		// Convert to 4 base64 characters
		result[j] = base64Table[b1>>2]
		result[j+1] = base64Table[((b1&0x03)<<4)|(b2>>4)]

		if i+1 < len(data) {
			result[j+2] = base64Table[((b2&0x0f)<<2)|(b3>>6)]
		} else {
			result[j+2] = '='
		}

		if i+2 < len(data) {
			result[j+3] = base64Table[b3&0x3f]
		} else {
			result[j+3] = '='
		}

		j += 4
	}

	return string(result)
}
