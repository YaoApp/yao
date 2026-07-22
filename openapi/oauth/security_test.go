package oauth

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/openapi/oauth/types"
)

// =============================================================================
// PKCE Code Challenge Tests
// =============================================================================

func TestGenerateCodeChallenge(t *testing.T) {
	service, _, _, cleanup := setupOAuthTestEnvironment(t)
	defer cleanup()

	ctx := context.Background()
	codeVerifier := "test_code_verifier_123456789"

	t.Run("S256 method", func(t *testing.T) {
		challenge, err := service.GenerateCodeChallenge(ctx, codeVerifier, "S256")
		assert.NoError(t, err)
		assert.NotEmpty(t, challenge)
		assert.NotEqual(t, codeVerifier, challenge)

		// Should be base64 URL encoded
		assert.NotContains(t, challenge, "=")
		assert.NotContains(t, challenge, "+")
		assert.NotContains(t, challenge, "/")
	})

	t.Run("plain method", func(t *testing.T) {
		challenge, err := service.GenerateCodeChallenge(ctx, codeVerifier, "plain")
		assert.NoError(t, err)
		assert.Equal(t, codeVerifier, challenge)
	})

	t.Run("unsupported method", func(t *testing.T) {
		challenge, err := service.GenerateCodeChallenge(ctx, codeVerifier, "unsupported")
		assert.Error(t, err)
		assert.Empty(t, challenge)
		assert.Contains(t, err.Error(), "unsupported code challenge method")
	})

	t.Run("consistency check", func(t *testing.T) {
		// Same verifier should generate same challenge
		challenge1, err := service.GenerateCodeChallenge(ctx, codeVerifier, "S256")
		assert.NoError(t, err)

		challenge2, err := service.GenerateCodeChallenge(ctx, codeVerifier, "S256")
		assert.NoError(t, err)

		assert.Equal(t, challenge1, challenge2)
	})
}

func TestValidateCodeChallenge(t *testing.T) {
	service, _, _, cleanup := setupOAuthTestEnvironment(t)
	defer cleanup()

	ctx := context.Background()
	codeVerifier := "test_code_verifier_123456789"

	t.Run("valid S256 challenge", func(t *testing.T) {
		challenge, err := service.GenerateCodeChallenge(ctx, codeVerifier, "S256")
		require.NoError(t, err)

		err = service.ValidateCodeChallenge(ctx, codeVerifier, challenge, "S256")
		assert.NoError(t, err)
	})

	t.Run("valid plain challenge", func(t *testing.T) {
		challenge, err := service.GenerateCodeChallenge(ctx, codeVerifier, "plain")
		require.NoError(t, err)

		err = service.ValidateCodeChallenge(ctx, codeVerifier, challenge, "plain")
		assert.NoError(t, err)
	})

	t.Run("invalid S256 challenge", func(t *testing.T) {
		err := service.ValidateCodeChallenge(ctx, codeVerifier, "invalid_challenge", "S256")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "code challenge verification failed")
	})

	t.Run("invalid plain challenge", func(t *testing.T) {
		err := service.ValidateCodeChallenge(ctx, "wrong_verifier", codeVerifier, "plain")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "code challenge verification failed")
	})

	t.Run("unsupported method", func(t *testing.T) {
		err := service.ValidateCodeChallenge(ctx, codeVerifier, "challenge", "unsupported")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported code challenge method")
	})
}

// =============================================================================
// State Parameter Tests
// =============================================================================

func TestGenerateStateParameter(t *testing.T) {
	service, _, _, cleanup := setupOAuthTestEnvironment(t)
	defer cleanup()

	ctx := context.Background()
	clientID := GetActualClientID(testClients[0].ClientID)

	t.Run("generate valid state parameter", func(t *testing.T) {
		stateParam, err := service.GenerateStateParameter(ctx, clientID)
		assert.NoError(t, err)
		assert.NotNil(t, stateParam)
		assert.NotEmpty(t, stateParam.Value)
		assert.Equal(t, clientID, stateParam.ClientID)
		assert.True(t, stateParam.ExpiresAt.After(time.Now()))
	})

	t.Run("generate unique state parameters", func(t *testing.T) {
		state1, err := service.GenerateStateParameter(ctx, clientID)
		assert.NoError(t, err)

		state2, err := service.GenerateStateParameter(ctx, clientID)
		assert.NoError(t, err)

		assert.NotEqual(t, state1.Value, state2.Value)
	})

	t.Run("state parameter format", func(t *testing.T) {
		stateParam, err := service.GenerateStateParameter(ctx, clientID)
		assert.NoError(t, err)

		// Should be base64 URL encoded
		assert.NotContains(t, stateParam.Value, "=")
		assert.NotContains(t, stateParam.Value, "+")
		assert.NotContains(t, stateParam.Value, "/")
		assert.True(t, len(stateParam.Value) > 0)
	})

	t.Run("empty client ID", func(t *testing.T) {
		stateParam, err := service.GenerateStateParameter(ctx, "")
		assert.NoError(t, err)
		assert.NotNil(t, stateParam)
		assert.Empty(t, stateParam.ClientID)
	})
}

func TestValidateStateParameter(t *testing.T) {
	service, _, _, cleanup := setupOAuthTestEnvironment(t)
	defer cleanup()

	ctx := context.Background()
	clientID := GetActualClientID(testClients[0].ClientID)

	t.Run("validate valid state parameter", func(t *testing.T) {
		stateParam, err := service.GenerateStateParameter(ctx, clientID)
		require.NoError(t, err)

		result, err := service.ValidateStateParameter(ctx, stateParam.Value, clientID)
		assert.NoError(t, err)
		assert.True(t, result.Valid)
		assert.Empty(t, result.Errors)
	})

	t.Run("validate non-existent state parameter", func(t *testing.T) {
		result, err := service.ValidateStateParameter(ctx, "non_existent_state", clientID)
		assert.NoError(t, err)
		assert.False(t, result.Valid)
		assert.Contains(t, result.Errors, "State parameter not found")
	})

	t.Run("validate state parameter with wrong client", func(t *testing.T) {
		stateParam, err := service.GenerateStateParameter(ctx, clientID)
		require.NoError(t, err)

		wrongClientID := GetActualClientID(testClients[1].ClientID)
		result, err := service.ValidateStateParameter(ctx, stateParam.Value, wrongClientID)
		assert.NoError(t, err)
		assert.False(t, result.Valid)
		assert.Contains(t, result.Errors, "State parameter not found")
	})

	t.Run("validate expired state parameter", func(t *testing.T) {
		// Create state parameter with very short lifetime
		originalConfig := service.config.Security.StateParameterLifetime
		service.config.Security.StateParameterLifetime = 1 * time.Nanosecond

		stateParam, err := service.GenerateStateParameter(ctx, clientID)
		require.NoError(t, err)

		// Restore original config
		service.config.Security.StateParameterLifetime = originalConfig

		// Wait for expiration
		time.Sleep(2 * time.Millisecond)

		result, err := service.ValidateStateParameter(ctx, stateParam.Value, clientID)
		assert.NoError(t, err)
		assert.False(t, result.Valid)
		// Note: The implementation might store the state parameter in cache,
		// so it could return different error messages depending on cache state
		assert.NotEmpty(t, result.Errors)
	})

	t.Run("validate empty state parameter", func(t *testing.T) {
		result, err := service.ValidateStateParameter(ctx, "", clientID)
		assert.NoError(t, err)
		assert.False(t, result.Valid)
		assert.Contains(t, result.Errors, "State parameter not found")
	})
}

// =============================================================================
// Redirect URI Validation Tests
// =============================================================================

func TestValidateRedirectURI(t *testing.T) {
	service, _, _, cleanup := setupOAuthTestEnvironment(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("valid redirect URI", func(t *testing.T) {
		redirectURI := "https://example.com/callback"
		registeredURIs := []string{
			"https://example.com/callback",
			"https://example.com/other",
		}

		result, err := service.ValidateRedirectURI(ctx, redirectURI, registeredURIs)
		assert.NoError(t, err)
		assert.True(t, result.Valid)
		assert.Empty(t, result.Errors)
	})

	t.Run("invalid redirect URI", func(t *testing.T) {
		redirectURI := "https://malicious.com/callback"
		registeredURIs := []string{
			"https://example.com/callback",
			"https://example.com/other",
		}

		result, err := service.ValidateRedirectURI(ctx, redirectURI, registeredURIs)
		assert.NoError(t, err)
		assert.False(t, result.Valid)
		assert.Contains(t, result.Errors, "Redirect URI not found in registered URIs")
	})

	t.Run("no registered URIs", func(t *testing.T) {
		redirectURI := "https://example.com/callback"
		registeredURIs := []string{}

		result, err := service.ValidateRedirectURI(ctx, redirectURI, registeredURIs)
		assert.NoError(t, err)
		assert.False(t, result.Valid)
		assert.Contains(t, result.Errors, "No registered URIs provided")
	})

	t.Run("nil registered URIs", func(t *testing.T) {
		redirectURI := "https://example.com/callback"
		var registeredURIs []string

		result, err := service.ValidateRedirectURI(ctx, redirectURI, registeredURIs)
		assert.NoError(t, err)
		assert.False(t, result.Valid)
		assert.Contains(t, result.Errors, "No registered URIs provided")
	})

	t.Run("empty redirect URI", func(t *testing.T) {
		redirectURI := ""
		registeredURIs := []string{"https://example.com/callback"}

		result, err := service.ValidateRedirectURI(ctx, redirectURI, registeredURIs)
		assert.NoError(t, err)
		assert.False(t, result.Valid)
		assert.Contains(t, result.Errors, "Redirect URI not found in registered URIs")
	})

	t.Run("exact match required", func(t *testing.T) {
		redirectURI := "https://example.com/callback/extra"
		registeredURIs := []string{"https://example.com/callback"}

		result, err := service.ValidateRedirectURI(ctx, redirectURI, registeredURIs)
		assert.NoError(t, err)
		assert.False(t, result.Valid)
		assert.Contains(t, result.Errors, "Redirect URI not found in registered URIs")
	})
}

func TestValidateRedirectURIForClient(t *testing.T) {
	service, _, _, cleanup := setupOAuthTestEnvironment(t)
	defer cleanup()

	ctx := context.Background()
	clientID := GetActualClientID(testClients[0].ClientID)
	validRedirectURI := testClients[0].RedirectURIs[0]

	t.Run("valid redirect URI for client", func(t *testing.T) {
		result, err := service.ValidateRedirectURIForClient(ctx, clientID, validRedirectURI)
		assert.NoError(t, err)
		assert.True(t, result.Valid)
		assert.Empty(t, result.Errors)
	})

	t.Run("invalid redirect URI for client", func(t *testing.T) {
		invalidRedirectURI := "https://malicious.com/callback"
		result, err := service.ValidateRedirectURIForClient(ctx, clientID, invalidRedirectURI)
		assert.NoError(t, err)
		assert.False(t, result.Valid)
		assert.NotEmpty(t, result.Errors)
	})

	t.Run("non-existent client", func(t *testing.T) {
		result, err := service.ValidateRedirectURIForClient(ctx, "non-existent-client", validRedirectURI)
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

// =============================================================================
// Pushed Authorization Request Tests
// =============================================================================

func TestPushAuthorizationRequest(t *testing.T) {
	service, _, _, cleanup := setupOAuthTestEnvironment(t)
	defer cleanup()

	ctx := context.Background()
	clientID := GetActualClientID(testClients[0].ClientID)
	redirectURI := testClients[0].RedirectURIs[0]

	t.Run("successful pushed authorization request", func(t *testing.T) {
		request := &types.PushedAuthorizationRequest{
			ClientID:     clientID,
			RedirectURI:  redirectURI,
			ResponseType: types.ResponseTypeCode,
			Scope:        "openid profile",
			State:        "test_state",
		}

		response, err := service.PushAuthorizationRequest(ctx, request)
		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.NotEmpty(t, response.RequestURI)
		assert.True(t, strings.HasPrefix(response.RequestURI, "urn:ietf:params:oauth:request_uri:"))
		assert.Equal(t, 600, response.ExpiresIn)
	})

	t.Run("invalid client", func(t *testing.T) {
		request := &types.PushedAuthorizationRequest{
			ClientID:     "invalid-client",
			RedirectURI:  redirectURI,
			ResponseType: types.ResponseTypeCode,
			Scope:        "openid profile",
			State:        "test_state",
		}

		response, err := service.PushAuthorizationRequest(ctx, request)
		assert.Error(t, err)
		assert.Nil(t, response)

		errorResp, ok := err.(*types.ErrorResponse)
		assert.True(t, ok)
		assert.Equal(t, types.ErrorInvalidClient, errorResp.Code)
	})

	t.Run("invalid redirect URI", func(t *testing.T) {
		request := &types.PushedAuthorizationRequest{
			ClientID:     clientID,
			RedirectURI:  "https://malicious.com/callback",
			ResponseType: types.ResponseTypeCode,
			Scope:        "openid profile",
			State:        "test_state",
		}

		response, err := service.PushAuthorizationRequest(ctx, request)
		assert.Error(t, err)
		assert.Nil(t, response)

		errorResp, ok := err.(*types.ErrorResponse)
		assert.True(t, ok)
		assert.Equal(t, types.ErrorInvalidRequest, errorResp.Code)
	})

	t.Run("invalid scope", func(t *testing.T) {
		request := &types.PushedAuthorizationRequest{
			ClientID:     clientID,
			RedirectURI:  redirectURI,
			ResponseType: types.ResponseTypeCode,
			Scope:        "invalid_scope",
			State:        "test_state",
		}

		response, err := service.PushAuthorizationRequest(ctx, request)
		assert.Error(t, err)
		assert.Nil(t, response)

		errorResp, ok := err.(*types.ErrorResponse)
		assert.True(t, ok)
		assert.Equal(t, types.ErrorInvalidScope, errorResp.Code)
	})

	t.Run("request without scope", func(t *testing.T) {
		request := &types.PushedAuthorizationRequest{
			ClientID:     clientID,
			RedirectURI:  redirectURI,
			ResponseType: types.ResponseTypeCode,
			State:        "test_state",
		}

		response, err := service.PushAuthorizationRequest(ctx, request)
		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.NotEmpty(t, response.RequestURI)
	})

	t.Run("request URI uniqueness", func(t *testing.T) {
		request := &types.PushedAuthorizationRequest{
			ClientID:     clientID,
			RedirectURI:  redirectURI,
			ResponseType: types.ResponseTypeCode,
			Scope:        "openid profile",
			State:        "test_state",
		}

		response1, err := service.PushAuthorizationRequest(ctx, request)
		assert.NoError(t, err)

		response2, err := service.PushAuthorizationRequest(ctx, request)
		assert.NoError(t, err)

		assert.NotEqual(t, response1.RequestURI, response2.RequestURI)
	})
}

// =============================================================================
// Helper Method Tests
// =============================================================================

func TestSecurityHelperMethods(t *testing.T) {
	service, _, _, cleanup := setupOAuthTestEnvironment(t)
	defer cleanup()

	clientID := GetActualClientID(testClients[0].ClientID)

	t.Run("state parameter key generation", func(t *testing.T) {
		state := "test_state"
		key := service.stateParameterKey(clientID, state)

		assert.NotEmpty(t, key)
		assert.Contains(t, key, "oauth:state")
		assert.Contains(t, key, clientID)
		assert.Contains(t, key, state)
	})

	t.Run("pushed auth request key generation", func(t *testing.T) {
		requestURI := "test_request_uri"
		key := service.pushedAuthRequestKey(requestURI)

		assert.NotEmpty(t, key)
		assert.Contains(t, key, "oauth:par:")
		// After SHA-256 hashing the key contains a 64-char hex digest, not the raw URI
		assert.Len(t, key, len(service.prefix)+len("oauth:par:")+64)
	})

	t.Run("request URI generation", func(t *testing.T) {
		requestURI := service.generateRequestURI()

		assert.NotEmpty(t, requestURI)
		assert.True(t, strings.HasPrefix(requestURI, "urn:ietf:params:oauth:request_uri:"))

		// Should be base64 URL encoded
		parts := strings.Split(requestURI, ":")
		assert.True(t, len(parts) >= 4)
		encodedPart := parts[len(parts)-1]
		assert.NotContains(t, encodedPart, "=")
		assert.NotContains(t, encodedPart, "+")
		assert.NotContains(t, encodedPart, "/")
	})

	t.Run("request URI uniqueness", func(t *testing.T) {
		uri1 := service.generateRequestURI()
		uri2 := service.generateRequestURI()

		assert.NotEqual(t, uri1, uri2)
	})
}

// =============================================================================
// Integration Tests
// =============================================================================

func TestSecurityIntegration(t *testing.T) {
	service, _, _, cleanup := setupOAuthTestEnvironment(t)
	defer cleanup()

	ctx := context.Background()
	clientID := GetActualClientID(testClients[0].ClientID)

	t.Run("complete PKCE flow", func(t *testing.T) {
		codeVerifier := "test_code_verifier_123456789"

		// Generate code challenge
		challenge, err := service.GenerateCodeChallenge(ctx, codeVerifier, "S256")
		assert.NoError(t, err)

		// Validate code challenge
		err = service.ValidateCodeChallenge(ctx, codeVerifier, challenge, "S256")
		assert.NoError(t, err)

		// Test with wrong verifier
		err = service.ValidateCodeChallenge(ctx, "wrong_verifier", challenge, "S256")
		assert.Error(t, err)
	})

	t.Run("complete state parameter flow", func(t *testing.T) {
		// Generate state parameter
		stateParam, err := service.GenerateStateParameter(ctx, clientID)
		assert.NoError(t, err)

		// Validate state parameter
		result, err := service.ValidateStateParameter(ctx, stateParam.Value, clientID)
		assert.NoError(t, err)
		assert.True(t, result.Valid)

		// Test with wrong client
		wrongClientID := GetActualClientID(testClients[1].ClientID)
		result, err = service.ValidateStateParameter(ctx, stateParam.Value, wrongClientID)
		assert.NoError(t, err)
		assert.False(t, result.Valid)
	})

	t.Run("complete pushed authorization flow", func(t *testing.T) {
		// Create pushed authorization request
		request := &types.PushedAuthorizationRequest{
			ClientID:     clientID,
			RedirectURI:  testClients[0].RedirectURIs[0],
			ResponseType: types.ResponseTypeCode,
			Scope:        "openid profile",
			State:        "test_state",
		}

		// Push authorization request
		response, err := service.PushAuthorizationRequest(ctx, request)
		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.NotEmpty(t, response.RequestURI)

		// Request URI should be stored and retrievable
		key := service.pushedAuthRequestKey(response.RequestURI)
		data, ok := service.store.Get(key)
		assert.True(t, ok)
		assert.NotNil(t, data)
	})
}

// =============================================================================
// Edge Cases and Error Handling
// =============================================================================

func TestSecurityEdgeCases(t *testing.T) {
	service, _, _, cleanup := setupOAuthTestEnvironment(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("PKCE with empty code verifier", func(t *testing.T) {
		challenge, err := service.GenerateCodeChallenge(ctx, "", "S256")
		assert.NoError(t, err)
		assert.NotEmpty(t, challenge)

		err = service.ValidateCodeChallenge(ctx, "", challenge, "S256")
		assert.NoError(t, err)
	})

	t.Run("PKCE with very long code verifier", func(t *testing.T) {
		longVerifier := strings.Repeat("a", 1000)
		challenge, err := service.GenerateCodeChallenge(ctx, longVerifier, "S256")
		assert.NoError(t, err)
		assert.NotEmpty(t, challenge)

		err = service.ValidateCodeChallenge(ctx, longVerifier, challenge, "S256")
		assert.NoError(t, err)
	})

	t.Run("state parameter with special characters", func(t *testing.T) {
		clientID := "client-with-special-chars-!@#$%"
		stateParam, err := service.GenerateStateParameter(ctx, clientID)
		assert.NoError(t, err)
		assert.NotNil(t, stateParam)

		result, err := service.ValidateStateParameter(ctx, stateParam.Value, clientID)
		assert.NoError(t, err)
		assert.True(t, result.Valid)
	})

	t.Run("redirect URI with query parameters", func(t *testing.T) {
		redirectURI := "https://example.com/callback?param=value"
		registeredURIs := []string{redirectURI}

		result, err := service.ValidateRedirectURI(ctx, redirectURI, registeredURIs)
		assert.NoError(t, err)
		assert.True(t, result.Valid)
	})

	t.Run("pushed authorization request with empty fields", func(t *testing.T) {
		request := &types.PushedAuthorizationRequest{
			ClientID:     GetActualClientID(testClients[0].ClientID),
			RedirectURI:  testClients[0].RedirectURIs[0],
			ResponseType: "",
			Scope:        "",
			State:        "",
		}

		response, err := service.PushAuthorizationRequest(ctx, request)
		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.NotEmpty(t, response.RequestURI)
	})
}

// =============================================================================
// Token Key Hashing Tests
// =============================================================================

func TestTokenKeyHashing(t *testing.T) {
	service, _, _, cleanup := setupOAuthTestEnvironment(t)
	defer cleanup()

	t.Run("hashKey deterministic", func(t *testing.T) {
		input := "some-token-value"
		h1 := service.hashKey(input)
		h2 := service.hashKey(input)
		assert.Equal(t, h1, h2)
		assert.Len(t, h1, 64) // SHA-256 hex = 64 chars
	})

	t.Run("hashKey different inputs yield different hashes", func(t *testing.T) {
		h1 := service.hashKey("token-a")
		h2 := service.hashKey("token-b")
		assert.NotEqual(t, h1, h2)
	})

	t.Run("all key functions produce keys within 255 chars", func(t *testing.T) {
		longJWT := strings.Repeat("x", 1000)

		keys := []string{
			service.accessTokenKey(longJWT),
			service.refreshTokenKey(longJWT),
			service.authorizationCodeKey(longJWT),
			service.pushedAuthRequestKey(longJWT),
			service.deviceCodeKey(longJWT),
			service.userCodeKey(longJWT),
		}

		for _, key := range keys {
			assert.LessOrEqual(t, len(key), 255, "key exceeds 255 chars: %s", key)
		}
	})

	t.Run("key functions contain correct type prefix", func(t *testing.T) {
		token := "test-token"

		assert.Contains(t, service.accessTokenKey(token), "oauth:access_token:")
		assert.Contains(t, service.refreshTokenKey(token), "oauth:refresh_token:")
		assert.Contains(t, service.authorizationCodeKey(token), "oauth:auth_code:")
		assert.Contains(t, service.pushedAuthRequestKey(token), "oauth:par:")
		assert.Contains(t, service.deviceCodeKey(token), "oauth:device_code:")
		assert.Contains(t, service.userCodeKey(token), "oauth:user_code:")
	})

	t.Run("same token produces same key", func(t *testing.T) {
		token := "test-token-for-consistency"
		k1 := service.accessTokenKey(token)
		k2 := service.accessTokenKey(token)
		assert.Equal(t, k1, k2)
	})

	t.Run("store round-trip with hashed keys", func(t *testing.T) {
		token := "round-trip-test-token"
		key := service.accessTokenKey(token)
		expected := map[string]interface{}{"client_id": "test-client"}

		err := service.store.Set(key, expected, 60*time.Second)
		assert.NoError(t, err)

		got, ok := service.store.Get(key)
		assert.True(t, ok)
		assert.NotNil(t, got)
	})
}
