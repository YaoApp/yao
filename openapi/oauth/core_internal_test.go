package oauth

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/openapi/oauth/types"
)

// =============================================================================
// Yao OAuth Server identity claims (Phase 2)
// =============================================================================

func TestAuthorizationCodeIdentityClaims(t *testing.T) {
	service := newUnitTestService(t)
	client := createUnitTestClient(t, service, "identity-client", []string{"https://client.example.com/callback"})
	ctx := context.Background()

	subject, err := service.Subject(client.ClientID, "user-identity")
	require.NoError(t, err)
	verifier := "test-verifier-0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	challenge, err := service.GenerateCodeChallenge(ctx, verifier, types.CodeChallengeMethodS256)
	require.NoError(t, err)

	identityRequest := func() *types.AuthorizationRequest {
		return &types.AuthorizationRequest{
			ClientID:            client.ClientID,
			ResponseType:        types.ResponseTypeCode,
			RedirectURI:         "https://client.example.com/callback",
			Scope:               "openid profile",
			State:               "state-identity",
			CodeChallenge:       challenge,
			CodeChallengeMethod: types.CodeChallengeMethodS256,
			Subject:             subject,
			AuthSource:          types.AuthSourceTao,
			IdentitySource:      types.IdentitySourceTao,
		}
	}

	exchangeCode := func(t *testing.T) *types.Token {
		t.Helper()
		resp, err := service.Authorize(ctx, identityRequest())
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Empty(t, resp.Error)
		require.NotEmpty(t, resp.Code)

		token, err := service.Token(ctx, types.GrantTypeAuthorizationCode, resp.Code, client.ClientID, verifier)
		require.NoError(t, err)
		require.NotNil(t, token)
		return token
	}

	t.Run("authorization code carries subject and identity metadata", func(t *testing.T) {
		resp, err := service.Authorize(ctx, identityRequest())
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotEmpty(t, resp.Code)

		codeData, err := service.getAuthorizationCodeData(resp.Code)
		require.NoError(t, err)
		assert.Equal(t, subject, codeData["subject"])
		assert.Equal(t, types.AuthSourceTao, codeData[types.ClaimAuthSource])
		assert.Equal(t, types.IdentitySourceTao, codeData[types.ClaimIdentitySource])
	})

	t.Run("code exchange propagates identity claims and no Team claims", func(t *testing.T) {
		token := exchangeCode(t)

		assert.NotEmpty(t, token.AccessToken)
		assert.NotEmpty(t, token.RefreshToken)

		// No ID token is ever issued by the Yao OAuth Server.
		rawToken, err := json.Marshal(token)
		require.NoError(t, err)
		assert.NotContains(t, string(rawToken), "id_token")

		claims, err := service.VerifyToken(token.AccessToken)
		require.NoError(t, err)
		require.NotNil(t, claims)
		assert.Equal(t, subject, claims.Subject)
		assert.Equal(t, types.AuthSourceTao, claims.Extra[types.ClaimAuthSource])
		assert.Equal(t, types.IdentitySourceTao, claims.Extra[types.ClaimIdentitySource])
		assert.Equal(t, client.ClientID, claims.ClientID)

		// Team/Tenant claims must never be part of OAuth identity.
		for _, forbidden := range []string{"team_id", "team_ids", "tenant", "tenant_id", "Team"} {
			assert.NotContains(t, claims.Extra, forbidden)
		}

		refreshData, err := service.getRefreshTokenData(token.RefreshToken)
		require.NoError(t, err)
		assert.Equal(t, subject, refreshData["subject"])
		assert.Equal(t, types.AuthSourceTao, refreshData[types.ClaimAuthSource])
		assert.Equal(t, types.IdentitySourceTao, refreshData[types.ClaimIdentitySource])
		for _, forbidden := range []string{"team_id", "team_ids", "tenant", "tenant_id"} {
			assert.NotContains(t, refreshData, forbidden)
		}
	})

	t.Run("refresh grant keeps identity claims", func(t *testing.T) {
		token := exchangeCode(t)

		refreshed, err := service.Token(ctx, types.GrantTypeRefreshToken, token.RefreshToken, client.ClientID, "")
		require.NoError(t, err)
		require.NotNil(t, refreshed)
		assert.NotEmpty(t, refreshed.AccessToken)

		claims, err := service.VerifyToken(refreshed.AccessToken)
		require.NoError(t, err)
		assert.Equal(t, subject, claims.Subject)
		assert.Equal(t, types.AuthSourceTao, claims.Extra[types.ClaimAuthSource])
		assert.Equal(t, types.IdentitySourceTao, claims.Extra[types.ClaimIdentitySource])
		assert.NotContains(t, claims.Extra, "team_id")
		assert.NotContains(t, claims.Extra, "tenant_id")
	})

	t.Run("authorization without a session subject still works", func(t *testing.T) {
		resp, err := service.Authorize(ctx, &types.AuthorizationRequest{
			ClientID:            client.ClientID,
			ResponseType:        types.ResponseTypeCode,
			RedirectURI:         "https://client.example.com/callback",
			Scope:               "openid profile",
			CodeChallenge:       challenge,
			CodeChallengeMethod: types.CodeChallengeMethodS256,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Empty(t, resp.Error)
		require.NotEmpty(t, resp.Code)

		codeData, err := service.getAuthorizationCodeData(resp.Code)
		require.NoError(t, err)
		assert.NotContains(t, codeData, "subject")
		assert.NotContains(t, codeData, types.ClaimAuthSource)
		assert.NotContains(t, codeData, types.ClaimIdentitySource)

		token, err := service.Token(ctx, types.GrantTypeAuthorizationCode, resp.Code, client.ClientID, verifier)
		require.NoError(t, err)
		claims, err := service.VerifyToken(token.AccessToken)
		require.NoError(t, err)
		assert.Empty(t, claims.Subject)
		assert.NotContains(t, claims.Extra, types.ClaimAuthSource)
		assert.NotContains(t, claims.Extra, types.ClaimIdentitySource)
	})
}

func TestValidateAuthorizationRequest(t *testing.T) {
	service := newUnitTestService(t)
	client := createUnitTestClient(t, service, "validate-client", []string{"https://client.example.com/callback"})
	ctx := context.Background()

	tests := []struct {
		name        string
		request     *types.AuthorizationRequest
		wantCode    string
		wantMessage string
	}{
		{
			name:        "nil request",
			request:     nil,
			wantCode:    types.ErrorInvalidRequest,
			wantMessage: "Missing authorization request",
		},
		{
			name: "unknown client",
			request: &types.AuthorizationRequest{
				ClientID: "missing", ResponseType: types.ResponseTypeCode,
				RedirectURI: "https://client.example.com/callback",
			},
			wantCode:    types.ErrorInvalidClient,
			wantMessage: "Invalid client",
		},
		{
			name: "missing redirect URI",
			request: &types.AuthorizationRequest{
				ClientID: client.ClientID, ResponseType: types.ResponseTypeCode,
			},
			wantCode:    types.ErrorInvalidRequest,
			wantMessage: "Missing redirect URI",
		},
		{
			name: "unregistered redirect URI",
			request: &types.AuthorizationRequest{
				ClientID: client.ClientID, ResponseType: types.ResponseTypeCode,
				RedirectURI: "https://evil.example.com/callback",
			},
			wantCode:    types.ErrorInvalidRequest,
			wantMessage: "Invalid redirect URI",
		},
		{
			name: "missing response type",
			request: &types.AuthorizationRequest{
				ClientID: client.ClientID, RedirectURI: "https://client.example.com/callback",
			},
			wantCode:    types.ErrorInvalidRequest,
			wantMessage: "Missing response type",
		},
		{
			name: "unsupported response type",
			request: &types.AuthorizationRequest{
				ClientID: client.ClientID, ResponseType: "unsupported",
				RedirectURI: "https://client.example.com/callback",
			},
			wantCode:    types.ErrorUnsupportedResponseType,
			wantMessage: "Unsupported response type",
		},
		{
			name: "invalid scope",
			request: &types.AuthorizationRequest{
				ClientID: client.ClientID, ResponseType: types.ResponseTypeCode,
				RedirectURI: "https://client.example.com/callback", Scope: "admin",
			},
			wantCode:    types.ErrorInvalidScope,
			wantMessage: "Invalid scope",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errResp := service.ValidateAuthorizationRequest(ctx, tt.request)
			require.NotNil(t, errResp)
			assert.Equal(t, tt.wantCode, errResp.Error)
			assert.Equal(t, tt.wantMessage, errResp.ErrorDescription)
			assert.Empty(t, errResp.Code)
		})
	}

	t.Run("valid request passes without error", func(t *testing.T) {
		errResp := service.ValidateAuthorizationRequest(ctx, &types.AuthorizationRequest{
			ClientID:     client.ClientID,
			ResponseType: types.ResponseTypeCode,
			RedirectURI:  "https://client.example.com/callback",
			Scope:        "openid profile",
		})
		assert.Nil(t, errResp)
	})
}
