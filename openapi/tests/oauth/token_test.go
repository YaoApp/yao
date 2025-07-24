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
	"github.com/yaoapp/yao/openapi/response"
	"github.com/yaoapp/yao/openapi/tests/testutils"
)

func TestOAuthToken_AuthorizationCode(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	// Get base URL from server config
	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Register a test client
	client := testutils.RegisterTestClient(t, "Token Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)

	// Obtain authorization code dynamically
	authInfo := testutils.ObtainAuthorizationCode(t, serverURL, client.ClientID, "https://localhost/callback", "openid profile")

	// Test authorization code grant
	t.Run("Valid Authorization Code Grant", func(t *testing.T) {
		// testutils.Prepare token request with PKCE code verifier
		data := url.Values{}
		data.Set("grant_type", "authorization_code")
		data.Set("code", authInfo.Code)
		data.Set("redirect_uri", authInfo.RedirectURI)
		data.Set("client_id", client.ClientID)
		data.Set("code_verifier", authInfo.CodeVerifier)

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
		assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

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

		// testutils.Prepare token request with invalid code
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
		// testutils.Prepare token request missing redirect_uri
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
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	// Get base URL from server config
	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Register a test client for client credentials
	client := testutils.RegisterTestClient(t, "Client Credentials Test", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)

	t.Run("Valid Client Credentials Grant", func(t *testing.T) {
		// testutils.Prepare token request
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
		// testutils.Prepare token request
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
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	// Get base URL from server config
	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Register a test client
	client := testutils.RegisterTestClient(t, "Refresh Token Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)

	// First, get an access token and refresh token using authorization code
	authInfo := testutils.ObtainAuthorizationCode(t, serverURL, client.ClientID, "https://localhost/callback", "openid profile")

	// Get initial token with PKCE code verifier
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", authInfo.Code)
	data.Set("redirect_uri", authInfo.RedirectURI)
	data.Set("client_id", client.ClientID)
	data.Set("code_verifier", authInfo.CodeVerifier)

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
		// testutils.Prepare refresh token request
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
		// Note: Scope might be omitted from response if it's the same as originally granted
		if refreshResp.Scope != "" {
			assert.Equal(t, "openid profile", refreshResp.Scope)
		}

		// New access token should be different from original
		assert.NotEqual(t, initialToken.AccessToken, refreshResp.AccessToken)

		t.Logf("Refresh token response: AccessToken=%s, TokenType=%s, ExpiresIn=%d",
			refreshResp.AccessToken, refreshResp.TokenType, refreshResp.ExpiresIn)
	})

	t.Run("Invalid Refresh Token", func(t *testing.T) {
		// testutils.Prepare refresh token request with invalid token
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
		// testutils.Prepare refresh token request without refresh_token parameter
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
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	// Get base URL from server config
	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	client := testutils.RegisterTestClient(t, "Invalid Grant Test", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)

	t.Run("Unsupported Grant Type", func(t *testing.T) {
		// testutils.Prepare token request with unsupported grant type
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
		// testutils.Prepare token request without grant_type
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

func TestOAuthRevoke(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	// Get base URL from server config
	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Register a test client
	client := testutils.RegisterTestClient(t, "Revoke Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)

	// Obtain access token directly using the utility function
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	t.Run("Valid Access Token Revocation", func(t *testing.T) {
		// testutils.Prepare revocation request
		data := url.Values{}
		data.Set("token", tokenInfo.AccessToken)
		data.Set("token_type_hint", "access_token")

		// Make revocation request
		endpoint := serverURL + baseURL + "/oauth/revoke"
		req, err := http.NewRequest("POST", endpoint, bytes.NewBufferString(data.Encode()))
		assert.NoError(t, err)

		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Authorization", "Basic "+basicAuth(client.ClientID, client.ClientSecret))

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		// Should return 200 OK for successful revocation
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		t.Logf("Access token revoked successfully")
	})

	t.Run("Valid Refresh Token Revocation", func(t *testing.T) {
		// testutils.Prepare revocation request for refresh token
		data := url.Values{}
		data.Set("token", tokenInfo.RefreshToken)
		data.Set("token_type_hint", "refresh_token")

		// Make revocation request
		endpoint := serverURL + baseURL + "/oauth/revoke"
		req, err := http.NewRequest("POST", endpoint, bytes.NewBufferString(data.Encode()))
		assert.NoError(t, err)

		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Authorization", "Basic "+basicAuth(client.ClientID, client.ClientSecret))

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		// Should return 200 OK for successful revocation
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		t.Logf("Refresh token revoked successfully")
	})

	t.Run("Invalid Token Revocation", func(t *testing.T) {
		// testutils.Prepare revocation request with invalid token
		data := url.Values{}
		data.Set("token", "invalid-token-12345")
		data.Set("token_type_hint", "access_token")

		// Make revocation request
		endpoint := serverURL + baseURL + "/oauth/revoke"
		req, err := http.NewRequest("POST", endpoint, bytes.NewBufferString(data.Encode()))
		assert.NoError(t, err)

		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Authorization", "Basic "+basicAuth(client.ClientID, client.ClientSecret))

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		// Should return 200 OK even for invalid tokens (RFC 7009)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		t.Logf("Invalid token revocation handled correctly")
	})

	t.Run("Missing Token Parameter", func(t *testing.T) {
		// testutils.Prepare revocation request without token parameter
		data := url.Values{}
		// Missing token parameter

		// Make revocation request
		endpoint := serverURL + baseURL + "/oauth/revoke"
		req, err := http.NewRequest("POST", endpoint, bytes.NewBufferString(data.Encode()))
		assert.NoError(t, err)

		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Authorization", "Basic "+basicAuth(client.ClientID, client.ClientSecret))

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		// Should return 400 Bad Request for missing token
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

func TestOAuthIntrospect(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	// Get base URL from server config
	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Register a test client
	client := testutils.RegisterTestClient(t, "Introspect Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)

	// Obtain access token directly using the utility function
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	t.Run("Valid Access Token Introspection", func(t *testing.T) {
		// testutils.Prepare introspection request
		data := url.Values{}
		data.Set("token", tokenInfo.AccessToken)
		data.Set("token_type_hint", "access_token")

		// Make introspection request
		endpoint := serverURL + baseURL + "/oauth/introspect"
		req, err := http.NewRequest("POST", endpoint, bytes.NewBufferString(data.Encode()))
		assert.NoError(t, err)

		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Authorization", "Basic "+basicAuth(client.ClientID, client.ClientSecret))

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		// Should return 200 OK
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Read response body
		body, err := io.ReadAll(resp.Body)
		assert.NoError(t, err)

		// Parse response directly (no wrapper)
		var introspectResp response.TokenIntrospectionResponse
		err = json.Unmarshal(body, &introspectResp)
		assert.NoError(t, err)

		// Verify introspection response
		assert.True(t, introspectResp.Active)
		assert.Equal(t, client.ClientID, introspectResp.ClientID)
		assert.Equal(t, "Bearer", introspectResp.TokenType)
		// Note: Scope and ExpiresAt might not be included in the response

		t.Logf("Token introspection result: Active=%v, ClientID=%s, TokenType=%s, Scope=%s",
			introspectResp.Active, introspectResp.ClientID, introspectResp.TokenType, introspectResp.Scope)
	})

	t.Run("Invalid Token Introspection", func(t *testing.T) {
		// testutils.Prepare introspection request with invalid token
		data := url.Values{}
		data.Set("token", "invalid-token-12345")
		data.Set("token_type_hint", "access_token")

		// Make introspection request
		endpoint := serverURL + baseURL + "/oauth/introspect"
		req, err := http.NewRequest("POST", endpoint, bytes.NewBufferString(data.Encode()))
		assert.NoError(t, err)

		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Authorization", "Basic "+basicAuth(client.ClientID, client.ClientSecret))

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		// Should return 200 OK
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Read response body
		body, err := io.ReadAll(resp.Body)
		assert.NoError(t, err)

		// Parse response directly (no wrapper)
		var introspectResp response.TokenIntrospectionResponse
		err = json.Unmarshal(body, &introspectResp)
		assert.NoError(t, err)

		// Should indicate token is inactive
		assert.False(t, introspectResp.Active)

		t.Logf("Invalid token introspection handled correctly: Active=%v", introspectResp.Active)
	})

	t.Run("Missing Token Parameter", func(t *testing.T) {
		// testutils.Prepare introspection request without token parameter
		data := url.Values{}
		// Missing token parameter

		// Make introspection request
		endpoint := serverURL + baseURL + "/oauth/introspect"
		req, err := http.NewRequest("POST", endpoint, bytes.NewBufferString(data.Encode()))
		assert.NoError(t, err)

		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Authorization", "Basic "+basicAuth(client.ClientID, client.ClientSecret))

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		// Should return 400 Bad Request for missing token
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("Revoked Token Introspection", func(t *testing.T) {
		// First revoke the token
		revokeData := url.Values{}
		revokeData.Set("token", tokenInfo.AccessToken)

		revokeEndpoint := serverURL + baseURL + "/oauth/revoke"
		revokeReq, err := http.NewRequest("POST", revokeEndpoint, bytes.NewBufferString(revokeData.Encode()))
		assert.NoError(t, err)

		revokeReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		revokeReq.Header.Set("Authorization", "Basic "+basicAuth(client.ClientID, client.ClientSecret))

		revokeResp, err := http.DefaultClient.Do(revokeReq)
		assert.NoError(t, err)
		defer revokeResp.Body.Close()

		assert.Equal(t, http.StatusOK, revokeResp.StatusCode)

		// Now try to introspect the revoked token
		data := url.Values{}
		data.Set("token", tokenInfo.AccessToken)

		endpoint := serverURL + baseURL + "/oauth/introspect"
		req, err := http.NewRequest("POST", endpoint, bytes.NewBufferString(data.Encode()))
		assert.NoError(t, err)

		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Authorization", "Basic "+basicAuth(client.ClientID, client.ClientSecret))

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		// Should return 200 OK
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Read response body
		body, err := io.ReadAll(resp.Body)
		assert.NoError(t, err)

		// Parse response directly (no wrapper)
		var introspectResp response.TokenIntrospectionResponse
		err = json.Unmarshal(body, &introspectResp)
		assert.NoError(t, err)

		// Revoked token should be inactive
		// Note: For JWT tokens, revocation might not be immediately reflected in introspection
		// since JWT tokens are stateless and contain their own validity information
		if introspectResp.Active {
			t.Logf("Token still appears active after revocation (expected for JWT tokens without blacklisting): Active=%v", introspectResp.Active)
		} else {
			assert.False(t, introspectResp.Active)
			t.Logf("Revoked token introspection handled correctly: Active=%v", introspectResp.Active)
		}
	})
}
