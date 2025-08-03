package oauth

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/openapi/oauth/types"
)

// =============================================================================
// Basic Service Endpoint Tests
// =============================================================================

func TestAuthorizationServer(t *testing.T) {
	service, _, _, cleanup := setupOAuthTestEnvironment(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("returns correct authorization server URL", func(t *testing.T) {
		url := service.AuthorizationServer(ctx)
		assert.Equal(t, "https://oauth.test.example.com", url)
		assert.Equal(t, service.config.IssuerURL, url)
	})
}

func TestProtectedResource(t *testing.T) {
	service, _, _, cleanup := setupOAuthTestEnvironment(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("returns correct protected resource URL", func(t *testing.T) {
		url := service.ProtectedResource(ctx)
		assert.Equal(t, "https://oauth.test.example.com", url)
		assert.Equal(t, service.config.IssuerURL, url)
	})
}

// =============================================================================
// Authorization Flow Tests
// =============================================================================

func TestAuthorize(t *testing.T) {
	service, _, _, cleanup := setupOAuthTestEnvironment(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("successful authorization code flow", func(t *testing.T) {
		request := &types.AuthorizationRequest{
			ClientID:     GetActualClientID(testClients[0].ClientID), // confidential client
			ResponseType: "code",
			RedirectURI:  "https://localhost/callback",
			Scope:        "openid profile",
			State:        "test-state-123",
		}

		response, err := service.Authorize(ctx, request)
		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.NotEmpty(t, response.Code)
		assert.Equal(t, "test-state-123", response.State)
		assert.Empty(t, response.Error)
	})

	t.Run("authorization with invalid client", func(t *testing.T) {
		request := &types.AuthorizationRequest{
			ClientID:     "invalid-client-id",
			ResponseType: "code",
			RedirectURI:  "https://localhost/callback",
			Scope:        "openid profile",
			State:        "test-state-123",
		}

		response, err := service.Authorize(ctx, request)
		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.Equal(t, types.ErrorInvalidClient, response.Error)
		assert.Equal(t, "Invalid client", response.ErrorDescription)
		assert.Empty(t, response.Code)
	})

	t.Run("authorization with missing redirect URI", func(t *testing.T) {
		request := &types.AuthorizationRequest{
			ClientID:     GetActualClientID(testClients[0].ClientID),
			ResponseType: "code",
			RedirectURI:  "", // Missing redirect URI
			Scope:        "openid profile",
			State:        "test-state-123",
		}

		response, err := service.Authorize(ctx, request)
		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.Equal(t, types.ErrorInvalidRequest, response.Error)
		assert.Equal(t, "Missing redirect URI", response.ErrorDescription)
		assert.Empty(t, response.Code)
	})

	t.Run("authorization with invalid redirect URI", func(t *testing.T) {
		request := &types.AuthorizationRequest{
			ClientID:     GetActualClientID(testClients[0].ClientID),
			ResponseType: "code",
			RedirectURI:  "https://invalid-domain.com/callback", // Invalid redirect URI
			Scope:        "openid profile",
			State:        "test-state-123",
		}

		response, err := service.Authorize(ctx, request)
		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.Equal(t, types.ErrorInvalidRequest, response.Error)
		assert.Equal(t, "Invalid redirect URI", response.ErrorDescription)
		assert.Empty(t, response.Code)
	})

	t.Run("authorization with missing response type", func(t *testing.T) {
		request := &types.AuthorizationRequest{
			ClientID:     GetActualClientID(testClients[0].ClientID),
			ResponseType: "", // Missing response type
			RedirectURI:  "https://localhost/callback",
			Scope:        "openid profile",
			State:        "test-state-123",
		}

		response, err := service.Authorize(ctx, request)
		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.Equal(t, types.ErrorInvalidRequest, response.Error)
		assert.Equal(t, "Missing response type", response.ErrorDescription)
		assert.Empty(t, response.Code)
	})

	t.Run("authorization with unsupported response type", func(t *testing.T) {
		request := &types.AuthorizationRequest{
			ClientID:     GetActualClientID(testClients[0].ClientID),
			ResponseType: "unsupported_type",
			RedirectURI:  "https://localhost/callback",
			Scope:        "openid profile",
			State:        "test-state-123",
		}

		response, err := service.Authorize(ctx, request)
		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.Equal(t, types.ErrorUnsupportedResponseType, response.Error)
		assert.Equal(t, "Unsupported response type", response.ErrorDescription)
		assert.Empty(t, response.Code)
	})

	t.Run("authorization with valid response types", func(t *testing.T) {
		validResponseTypes := []string{"code", "token", "id_token", "code token", "code id_token"}

		for _, responseType := range validResponseTypes {
			request := &types.AuthorizationRequest{
				ClientID:     GetActualClientID(testClients[0].ClientID),
				ResponseType: responseType,
				RedirectURI:  "https://localhost/callback",
				Scope:        "openid profile",
				State:        "test-state-123",
			}

			response, err := service.Authorize(ctx, request)
			assert.NoError(t, err)
			assert.NotNil(t, response)
			assert.NotEmpty(t, response.Code)
			assert.Empty(t, response.Error)
		}
	})

	t.Run("authorization with invalid scope", func(t *testing.T) {
		request := &types.AuthorizationRequest{
			ClientID:     GetActualClientID(testClients[0].ClientID),
			ResponseType: "code",
			RedirectURI:  "https://localhost/callback",
			Scope:        "invalid-scope", // Invalid scope
			State:        "test-state-123",
		}

		response, err := service.Authorize(ctx, request)
		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.Equal(t, types.ErrorInvalidScope, response.Error)
		assert.Equal(t, "Invalid scope", response.ErrorDescription)
		assert.Empty(t, response.Code)
	})

	t.Run("authorization without scope", func(t *testing.T) {
		request := &types.AuthorizationRequest{
			ClientID:     GetActualClientID(testClients[0].ClientID),
			ResponseType: "code",
			RedirectURI:  "https://localhost/callback",
			Scope:        "", // No scope
			State:        "test-state-123",
		}

		response, err := service.Authorize(ctx, request)
		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.NotEmpty(t, response.Code)
		assert.Empty(t, response.Error)
	})
}

// =============================================================================
// Token Exchange Tests
// =============================================================================

func TestToken(t *testing.T) {
	service, _, _, cleanup := setupOAuthTestEnvironment(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("authorization code grant", func(t *testing.T) {
		clientID := GetActualClientID(testClients[0].ClientID) // confidential client

		// Generate a real authorization code using the service
		code, err := service.generateAuthorizationCodeWithInfo(clientID, "test-state", "", "", "")
		assert.NoError(t, err)
		assert.NotEmpty(t, code)

		token, err := service.Token(ctx, types.GrantTypeAuthorizationCode, code, clientID, "")
		assert.NoError(t, err)
		assert.NotNil(t, token)
		assert.NotEmpty(t, token.AccessToken)
		assert.Equal(t, "Bearer", token.TokenType)
		assert.Equal(t, 3600, token.ExpiresIn)
		assert.NotEmpty(t, token.RefreshToken) // Should have refresh token
	})

	t.Run("client credentials grant", func(t *testing.T) {
		clientID := GetActualClientID(testClients[2].ClientID) // client credentials client

		token, err := service.Token(ctx, types.GrantTypeClientCredentials, "", clientID, "")
		assert.NoError(t, err)
		assert.NotNil(t, token)
		assert.NotEmpty(t, token.AccessToken)
		assert.Equal(t, "Bearer", token.TokenType)
		assert.Equal(t, 3600, token.ExpiresIn)
		assert.Empty(t, token.RefreshToken) // Should not have refresh token for client credentials
	})

	t.Run("refresh token grant", func(t *testing.T) {
		clientID := GetActualClientID(testClients[0].ClientID) // confidential client
		refreshToken := "test-refresh-token"

		// Store refresh token using the new method
		err := service.storeRefreshToken(refreshToken, clientID)
		assert.NoError(t, err)

		token, err := service.Token(ctx, types.GrantTypeRefreshToken, refreshToken, clientID, "")
		assert.NoError(t, err)
		assert.NotNil(t, token)
		assert.NotEmpty(t, token.AccessToken)
		assert.Equal(t, "Bearer", token.TokenType)
		assert.Equal(t, 3600, token.ExpiresIn)
		assert.NotEmpty(t, token.RefreshToken) // Should have refresh token
	})

	t.Run("invalid client", func(t *testing.T) {
		clientID := "invalid-client-id"

		// Generate a real authorization code for consistency, even though client validation happens first
		validClientID := GetActualClientID(testClients[0].ClientID)
		code, err := service.generateAuthorizationCodeWithInfo(validClientID, "test-state", "", "", "")
		assert.NoError(t, err)
		assert.NotEmpty(t, code)

		token, err := service.Token(ctx, types.GrantTypeAuthorizationCode, code, clientID, "")
		assert.Error(t, err)
		assert.Nil(t, token)

		oauthErr, ok := err.(*types.ErrorResponse)
		assert.True(t, ok)
		assert.Equal(t, types.ErrorInvalidClient, oauthErr.Code)
		assert.Equal(t, "Invalid client", oauthErr.ErrorDescription)
	})

	t.Run("unsupported grant type", func(t *testing.T) {
		clientID := GetActualClientID(testClients[0].ClientID)

		// Generate a real authorization code for consistency
		code, err := service.generateAuthorizationCodeWithInfo(clientID, "test-state", "", "", "")
		assert.NoError(t, err)
		assert.NotEmpty(t, code)

		token, err := service.Token(ctx, "unsupported_grant_type", code, clientID, "")
		assert.Error(t, err)
		assert.Nil(t, token)

		oauthErr, ok := err.(*types.ErrorResponse)
		assert.True(t, ok)
		assert.Equal(t, types.ErrorUnsupportedGrantType, oauthErr.Code)
		assert.Equal(t, "Unsupported grant type", oauthErr.ErrorDescription)
	})
}

// =============================================================================
// Token Revocation Tests
// =============================================================================

func TestRevoke(t *testing.T) {
	service, _, _, cleanup := setupOAuthTestEnvironment(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("successful token revocation", func(t *testing.T) {
		token := "test-access-token"
		clientID := GetActualClientID(testClients[0].ClientID)

		// Store token using the new method
		err := service.storeAccessToken(token, clientID, "", "", 3600)
		assert.NoError(t, err)

		err = service.Revoke(ctx, token, "access_token")
		assert.NoError(t, err)

		// Verify token is revoked - should not be found in store
		_, err = service.getAccessTokenData(token)
		assert.Error(t, err) // Should return error since token is revoked
	})

	t.Run("revoke non-existent token", func(t *testing.T) {
		token := "non-existent-token"

		// According to OAuth spec, revoking non-existent token should succeed
		err := service.Revoke(ctx, token, "access_token")
		assert.NoError(t, err)

		// Verify token still doesn't exist
		_, err = service.getAccessTokenData(token)
		assert.Error(t, err) // Should return error since token doesn't exist
	})

	t.Run("revoke refresh token", func(t *testing.T) {
		token := "test-refresh-token"
		clientID := GetActualClientID(testClients[0].ClientID)

		// Store refresh token using the new method
		err := service.storeRefreshToken(token, clientID)
		assert.NoError(t, err)

		err = service.Revoke(ctx, token, "refresh_token")
		assert.NoError(t, err)

		// Verify token is revoked - should not be found in store
		_, err = service.getRefreshTokenData(token)
		assert.Error(t, err) // Should return error since token is revoked
	})
}

// =============================================================================
// Refresh Token Tests
// =============================================================================

func TestRefreshToken(t *testing.T) {
	service, _, _, cleanup := setupOAuthTestEnvironment(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("successful refresh token exchange", func(t *testing.T) {
		refreshToken := "test-refresh-token"
		clientID := GetActualClientID(testClients[0].ClientID)
		originalScope := "openid profile email"
		subject := testUsers[0].UserID

		// Store refresh token with scope using storeRefreshTokenWithScope
		err := service.storeRefreshTokenWithScope(refreshToken, clientID, originalScope, subject)
		assert.NoError(t, err)

		response, err := service.RefreshToken(ctx, refreshToken, "openid profile")
		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.NotEmpty(t, response.AccessToken)
		assert.Equal(t, "Bearer", response.TokenType)
		assert.Equal(t, int(service.config.Token.AccessTokenLifetime.Seconds()), response.ExpiresIn)
		assert.Equal(t, "openid profile", response.Scope)
	})

	t.Run("refresh token with rotation enabled", func(t *testing.T) {
		refreshToken := "test-refresh-token-rotation"
		clientID := GetActualClientID(testClients[0].ClientID)
		originalScope := "openid profile"
		subject := testUsers[0].UserID

		// Store refresh token with scope using storeRefreshTokenWithScope
		err := service.storeRefreshTokenWithScope(refreshToken, clientID, originalScope, subject)
		assert.NoError(t, err)

		// Ensure rotation is enabled
		assert.True(t, service.config.Features.RefreshTokenRotationEnabled)

		response, err := service.RefreshToken(ctx, refreshToken)
		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.NotEmpty(t, response.AccessToken)
		assert.NotEmpty(t, response.RefreshToken)
		assert.NotEqual(t, refreshToken, response.RefreshToken) // Should be different

	})

	t.Run("invalid refresh token", func(t *testing.T) {
		refreshToken := "invalid-refresh-token"

		response, err := service.RefreshToken(ctx, refreshToken)
		assert.Error(t, err)
		assert.Nil(t, response)

		oauthErr, ok := err.(*types.ErrorResponse)
		assert.True(t, ok)
		assert.Equal(t, types.ErrorInvalidGrant, oauthErr.Code)
		assert.Equal(t, "Invalid refresh token", oauthErr.ErrorDescription)
	})

	t.Run("refresh token with invalid client", func(t *testing.T) {
		refreshToken := "test-refresh-token-invalid-client"

		// Store refresh token with invalid client
		err := service.storeRefreshToken(refreshToken, "invalid-client-id")
		assert.NoError(t, err)

		response, err := service.RefreshToken(ctx, refreshToken)
		assert.Error(t, err)
		assert.Nil(t, response)

		oauthErr, ok := err.(*types.ErrorResponse)
		assert.True(t, ok)
		assert.Equal(t, types.ErrorInvalidClient, oauthErr.Code)
		assert.Equal(t, "Invalid client", oauthErr.ErrorDescription)
	})

	t.Run("refresh token with invalid scope", func(t *testing.T) {
		refreshToken := "test-refresh-token-invalid-scope"
		clientID := GetActualClientID(testClients[0].ClientID)
		originalScope := "openid profile" // Original scope
		subject := testUsers[0].UserID

		// Store refresh token with limited scope
		err := service.storeRefreshTokenWithScope(refreshToken, clientID, originalScope, subject)
		assert.NoError(t, err)

		// Try to request scope that exceeds the original scope
		response, err := service.RefreshToken(ctx, refreshToken, "openid profile admin")
		assert.Error(t, err)
		assert.Nil(t, response)

		oauthErr, ok := err.(*types.ErrorResponse)
		assert.True(t, ok)
		assert.Equal(t, types.ErrorInvalidScope, oauthErr.Code)
		assert.Equal(t, "Requested scope exceeds originally granted scope", oauthErr.ErrorDescription)
	})

	t.Run("refresh token without scope", func(t *testing.T) {
		refreshToken := "test-refresh-token-no-scope"
		clientID := GetActualClientID(testClients[0].ClientID)

		// Store refresh token
		err := service.storeRefreshToken(refreshToken, clientID)
		assert.NoError(t, err)

		response, err := service.RefreshToken(ctx, refreshToken)
		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.NotEmpty(t, response.AccessToken)
		assert.Empty(t, response.Scope)
	})
}

// =============================================================================
// Refresh Token Rotation Tests
// =============================================================================

func TestRotateRefreshToken(t *testing.T) {
	service, _, _, cleanup := setupOAuthTestEnvironment(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("successful refresh token rotation", func(t *testing.T) {
		oldToken := "old-refresh-token"
		clientID := GetActualClientID(testClients[0].ClientID)
		originalScope := "openid profile"
		subject := testUsers[0].UserID

		// Store old refresh token with scope using storeRefreshTokenWithScope
		err := service.storeRefreshTokenWithScope(oldToken, clientID, originalScope, subject)
		assert.NoError(t, err)

		// Ensure rotation is enabled
		assert.True(t, service.config.Features.RefreshTokenRotationEnabled)

		response, err := service.RotateRefreshToken(ctx, oldToken)
		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.NotEmpty(t, response.AccessToken)
		assert.NotEmpty(t, response.RefreshToken)
		assert.NotEqual(t, oldToken, response.RefreshToken)
		assert.Equal(t, "Bearer", response.TokenType)
		assert.Equal(t, 3600, response.ExpiresIn)

	})

	t.Run("rotation with disabled feature", func(t *testing.T) {
		// Temporarily disable rotation
		originalEnabled := service.config.Features.RefreshTokenRotationEnabled
		service.config.Features.RefreshTokenRotationEnabled = false
		defer func() {
			service.config.Features.RefreshTokenRotationEnabled = originalEnabled
		}()

		oldToken := "old-refresh-token-disabled"

		response, err := service.RotateRefreshToken(ctx, oldToken)
		assert.Error(t, err)
		assert.Nil(t, response)

		oauthErr, ok := err.(*types.ErrorResponse)
		assert.True(t, ok)
		assert.Equal(t, types.ErrorInvalidRequest, oauthErr.Code)
		assert.Equal(t, "Refresh token rotation is not enabled", oauthErr.ErrorDescription)
	})

	t.Run("rotation with invalid token", func(t *testing.T) {
		oldToken := "invalid-refresh-token"

		response, err := service.RotateRefreshToken(ctx, oldToken)
		assert.Error(t, err)
		assert.Nil(t, response)

		oauthErr, ok := err.(*types.ErrorResponse)
		assert.True(t, ok)
		assert.Equal(t, types.ErrorInvalidGrant, oauthErr.Code)
		assert.Equal(t, "Invalid refresh token", oauthErr.ErrorDescription)
	})

	t.Run("rotation with malformed token data", func(t *testing.T) {
		oldToken := "malformed-refresh-token"

		// Store token with malformed data directly in store
		malformedData := map[string]interface{}{
			"invalid_field": "invalid_value",
		}
		err := service.store.Set(service.refreshTokenKey(oldToken), malformedData, 24*time.Hour)
		assert.NoError(t, err)

		response, err := service.RotateRefreshToken(ctx, oldToken)
		assert.Error(t, err)
		assert.Nil(t, response)

		oauthErr, ok := err.(*types.ErrorResponse)
		assert.True(t, ok)
		assert.Equal(t, types.ErrorInvalidGrant, oauthErr.Code)
		assert.Equal(t, "Invalid token format", oauthErr.ErrorDescription)
	})
}

// =============================================================================
// Grant Type Handler Tests
// =============================================================================

func TestHandleAuthorizationCodeGrant(t *testing.T) {
	service, _, _, cleanup := setupOAuthTestEnvironment(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("successful authorization code grant", func(t *testing.T) {
		client := &types.ClientInfo{
			ClientID:   GetActualClientID(testClients[0].ClientID),
			GrantTypes: []string{types.GrantTypeAuthorizationCode, types.GrantTypeRefreshToken},
		}

		// Generate a real authorization code
		code, err := service.generateAuthorizationCodeWithInfo(client.ClientID, "test-state", "", "", "")
		assert.NoError(t, err)
		assert.NotEmpty(t, code)

		token, err := service.handleAuthorizationCodeGrant(ctx, client, code, "")
		assert.NoError(t, err)
		assert.NotNil(t, token)
		assert.NotEmpty(t, token.AccessToken)
		assert.Equal(t, "Bearer", token.TokenType)
		assert.Equal(t, 3600, token.ExpiresIn)
		assert.NotEmpty(t, token.RefreshToken) // Should have refresh token
	})

	t.Run("authorization code grant without refresh token support", func(t *testing.T) {
		client := &types.ClientInfo{
			ClientID:   GetActualClientID(testClients[1].ClientID),
			GrantTypes: []string{types.GrantTypeAuthorizationCode}, // No refresh token
		}

		// Generate a real authorization code
		code, err := service.generateAuthorizationCodeWithInfo(client.ClientID, "test-state", "", "", "")
		assert.NoError(t, err)
		assert.NotEmpty(t, code)

		token, err := service.handleAuthorizationCodeGrant(ctx, client, code, "")
		assert.NoError(t, err)
		assert.NotNil(t, token)
		assert.NotEmpty(t, token.AccessToken)
		assert.Equal(t, "Bearer", token.TokenType)
		assert.Equal(t, 3600, token.ExpiresIn)
		assert.Empty(t, token.RefreshToken) // Should not have refresh token
	})
}

func TestHandleClientCredentialsGrant(t *testing.T) {
	service, _, _, cleanup := setupOAuthTestEnvironment(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("successful client credentials grant", func(t *testing.T) {
		client := &types.ClientInfo{
			ClientID:   GetActualClientID(testClients[2].ClientID),
			GrantTypes: []string{types.GrantTypeClientCredentials},
		}

		token, err := service.handleClientCredentialsGrant(ctx, client)
		assert.NoError(t, err)
		assert.NotNil(t, token)
		assert.NotEmpty(t, token.AccessToken)
		assert.Equal(t, "Bearer", token.TokenType)
		assert.Equal(t, 3600, token.ExpiresIn)
		assert.Empty(t, token.RefreshToken) // Should not have refresh token
	})
}

func TestHandleRefreshTokenGrant(t *testing.T) {
	service, _, _, cleanup := setupOAuthTestEnvironment(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("successful refresh token grant with rotation", func(t *testing.T) {
		client := &types.ClientInfo{
			ClientID:   GetActualClientID(testClients[0].ClientID),
			GrantTypes: []string{types.GrantTypeRefreshToken},
		}

		refreshToken := "test-refresh-token-grant"
		originalScope := "openid profile"
		subject := testUsers[0].UserID

		// Store refresh token with scope using storeRefreshTokenWithScope
		err := service.storeRefreshTokenWithScope(refreshToken, client.ClientID, originalScope, subject)
		assert.NoError(t, err)

		// Ensure rotation is enabled
		assert.True(t, service.config.Features.RefreshTokenRotationEnabled)

		token, err := service.handleRefreshTokenGrant(ctx, client, refreshToken)
		assert.NoError(t, err)
		assert.NotNil(t, token)
		assert.NotEmpty(t, token.AccessToken)
		assert.Equal(t, "Bearer", token.TokenType)
		assert.Equal(t, 3600, token.ExpiresIn)
		assert.NotEmpty(t, token.RefreshToken)
		assert.NotEqual(t, refreshToken, token.RefreshToken) // Should be different

	})

	t.Run("refresh token grant without rotation", func(t *testing.T) {
		// Temporarily disable rotation
		originalEnabled := service.config.Features.RefreshTokenRotationEnabled
		service.config.Features.RefreshTokenRotationEnabled = false
		defer func() {
			service.config.Features.RefreshTokenRotationEnabled = originalEnabled
		}()

		client := &types.ClientInfo{
			ClientID:   GetActualClientID(testClients[0].ClientID),
			GrantTypes: []string{types.GrantTypeRefreshToken},
		}

		refreshToken := "test-refresh-token-no-rotation"
		originalScope := "openid profile"
		subject := testUsers[0].UserID

		// Store refresh token with scope using storeRefreshTokenWithScope
		err := service.storeRefreshTokenWithScope(refreshToken, client.ClientID, originalScope, subject)
		assert.NoError(t, err)

		token, err := service.handleRefreshTokenGrant(ctx, client, refreshToken)
		assert.NoError(t, err)
		assert.NotNil(t, token)
		assert.NotEmpty(t, token.AccessToken)
		assert.Equal(t, "Bearer", token.TokenType)
		assert.Equal(t, 3600, token.ExpiresIn)
		assert.Equal(t, refreshToken, token.RefreshToken) // Should be the same

	})

	t.Run("refresh token grant with invalid token", func(t *testing.T) {
		client := &types.ClientInfo{
			ClientID:   GetActualClientID(testClients[0].ClientID),
			GrantTypes: []string{types.GrantTypeRefreshToken},
		}

		refreshToken := "invalid-refresh-token"

		token, err := service.handleRefreshTokenGrant(ctx, client, refreshToken)
		assert.Error(t, err)
		assert.Nil(t, token)

		oauthErr, ok := err.(*types.ErrorResponse)
		assert.True(t, ok)
		assert.Equal(t, types.ErrorInvalidGrant, oauthErr.Code)
		assert.Equal(t, "Invalid refresh token", oauthErr.ErrorDescription)
	})
}

// =============================================================================
// Integration Tests
// =============================================================================

func TestCoreIntegration(t *testing.T) {
	service, _, _, cleanup := setupOAuthTestEnvironment(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("complete authorization code flow", func(t *testing.T) {
		// Step 1: Authorization
		authRequest := &types.AuthorizationRequest{
			ClientID:     GetActualClientID(testClients[0].ClientID),
			ResponseType: "code",
			RedirectURI:  "https://localhost/callback",
			Scope:        "openid profile",
			State:        "integration-test-state",
		}

		authResponse, err := service.Authorize(ctx, authRequest)
		assert.NoError(t, err)
		assert.NotNil(t, authResponse)
		assert.NotEmpty(t, authResponse.Code)
		assert.Equal(t, "integration-test-state", authResponse.State)

		// Step 2: Token exchange
		token, err := service.Token(ctx, types.GrantTypeAuthorizationCode, authResponse.Code, GetActualClientID(testClients[0].ClientID), "")
		assert.NoError(t, err)
		assert.NotNil(t, token)
		assert.NotEmpty(t, token.AccessToken)
		assert.NotEmpty(t, token.RefreshToken)

		// Step 3: Refresh token (token already stored with proper scope information)
		refreshResponse, err := service.RefreshToken(ctx, token.RefreshToken, "openid profile")
		assert.NoError(t, err)
		assert.NotNil(t, refreshResponse)
		assert.NotEmpty(t, refreshResponse.AccessToken)
		assert.NotEmpty(t, refreshResponse.RefreshToken)
		assert.NotEqual(t, token.RefreshToken, refreshResponse.RefreshToken)

		// Step 4: Revoke token
		err = service.Revoke(ctx, refreshResponse.AccessToken, "access_token")
		assert.NoError(t, err)
	})

	t.Run("client credentials flow", func(t *testing.T) {
		// Token exchange for client credentials
		token, err := service.Token(ctx, types.GrantTypeClientCredentials, "", GetActualClientID(testClients[2].ClientID), "")
		assert.NoError(t, err)
		assert.NotNil(t, token)
		assert.NotEmpty(t, token.AccessToken)
		assert.Empty(t, token.RefreshToken) // No refresh token for client credentials

		// Revoke token
		err = service.Revoke(ctx, token.AccessToken, "access_token")
		assert.NoError(t, err)
	})

	t.Run("error propagation", func(t *testing.T) {
		// Test that errors are properly propagated through the flow

		// Invalid client in authorization
		authRequest := &types.AuthorizationRequest{
			ClientID:     "invalid-client",
			ResponseType: "code",
			RedirectURI:  "https://localhost/callback",
			Scope:        "openid profile",
			State:        "error-test-state",
		}

		authResponse, err := service.Authorize(ctx, authRequest)
		assert.NoError(t, err)
		assert.NotNil(t, authResponse)
		assert.Equal(t, types.ErrorInvalidClient, authResponse.Error)

		// Invalid client in token exchange
		token, err := service.Token(ctx, types.GrantTypeAuthorizationCode, "test-code", "invalid-client", "")
		assert.Error(t, err)
		assert.Nil(t, token)

		oauthErr, ok := err.(*types.ErrorResponse)
		assert.True(t, ok)
		assert.Equal(t, types.ErrorInvalidClient, oauthErr.Code)
	})
}

// =============================================================================
// Edge Cases and Security Tests
// =============================================================================

func TestCoreEdgeCases(t *testing.T) {
	service, _, _, cleanup := setupOAuthTestEnvironment(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("authorization with multiple scopes", func(t *testing.T) {
		request := &types.AuthorizationRequest{
			ClientID:     GetActualClientID(testClients[0].ClientID),
			ResponseType: "code",
			RedirectURI:  "https://localhost/callback",
			Scope:        "openid profile email", // Multiple scopes
			State:        "multi-scope-test",
		}

		response, err := service.Authorize(ctx, request)
		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.NotEmpty(t, response.Code)
		assert.Empty(t, response.Error)
	})

	t.Run("authorization with state parameter preservation", func(t *testing.T) {
		longState := strings.Repeat("test-state-", 10) // Long state parameter

		request := &types.AuthorizationRequest{
			ClientID:     GetActualClientID(testClients[0].ClientID),
			ResponseType: "code",
			RedirectURI:  "https://localhost/callback",
			Scope:        "openid profile",
			State:        longState,
		}

		response, err := service.Authorize(ctx, request)
		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.NotEmpty(t, response.Code)
		assert.Equal(t, longState, response.State)
	})

	t.Run("token generation uniqueness", func(t *testing.T) {
		clientID := GetActualClientID(testClients[0].ClientID)
		tokens := make(map[string]bool)

		// Generate multiple tokens and ensure they're unique
		for i := 0; i < 10; i++ {
			// Generate a new authorization code for each iteration (codes can only be used once)
			code, err := service.generateAuthorizationCodeWithInfo(clientID, "test-state", "", "", "")
			assert.NoError(t, err)
			assert.NotEmpty(t, code)

			token, err := service.Token(ctx, types.GrantTypeAuthorizationCode, code, clientID, "")
			assert.NoError(t, err)
			assert.NotNil(t, token)
			assert.NotEmpty(t, token.AccessToken)

			// Check uniqueness
			assert.False(t, tokens[token.AccessToken], "Access token should be unique")
			tokens[token.AccessToken] = true
		}
	})

	t.Run("refresh token data integrity", func(t *testing.T) {
		refreshToken := "test-refresh-token-integrity"
		clientID := GetActualClientID(testClients[0].ClientID)

		// Store refresh token with additional data directly in store
		tokenData := map[string]interface{}{
			"client_id":  clientID,
			"type":       "refresh_token",
			"user_id":    "test-user-123",
			"issued_at":  time.Now().Unix(),
			"extra_data": "should-be-preserved",
		}
		err := service.store.Set(service.refreshTokenKey(refreshToken), tokenData, 24*time.Hour)
		assert.NoError(t, err)

		response, err := service.RefreshToken(ctx, refreshToken)
		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.NotEmpty(t, response.AccessToken)
	})
}
