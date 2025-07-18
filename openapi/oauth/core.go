package oauth

import (
	"context"
	"strings"

	"github.com/yaoapp/yao/openapi/oauth/types"
)

// AuthorizationServer returns the authorization server endpoint URL
func (s *Service) AuthorizationServer(ctx context.Context) string {
	return s.config.IssuerURL
}

// ProtectedResource returns the protected resource endpoint URL
func (s *Service) ProtectedResource(ctx context.Context) string {
	return s.config.IssuerURL
}

// Authorize processes an authorization request and returns an authorization code
// The authorization code can be exchanged for an access token
func (s *Service) Authorize(ctx context.Context, request *types.AuthorizationRequest) (*types.AuthorizationResponse, error) {
	// Validate client
	_, err := s.clientProvider.GetClientByID(ctx, request.ClientID)
	if err != nil {
		return &types.AuthorizationResponse{
			Error:            types.ErrorInvalidClient,
			ErrorDescription: "Invalid client",
		}, nil
	}

	// Validate redirect URI
	if request.RedirectURI == "" {
		return &types.AuthorizationResponse{
			Error:            types.ErrorInvalidRequest,
			ErrorDescription: "Missing redirect URI",
		}, nil
	}

	validationResult, err := s.clientProvider.ValidateRedirectURI(ctx, request.ClientID, request.RedirectURI)
	if err != nil || !validationResult.Valid {
		return &types.AuthorizationResponse{
			Error:            types.ErrorInvalidRequest,
			ErrorDescription: "Invalid redirect URI",
		}, nil
	}

	// Validate response type
	if request.ResponseType == "" {
		return &types.AuthorizationResponse{
			Error:            types.ErrorInvalidRequest,
			ErrorDescription: "Missing response type",
		}, nil
	}

	validResponseTypes := []string{"code", "token", "id_token"}
	validResponseType := false
	for _, validType := range validResponseTypes {
		if request.ResponseType == validType || strings.Contains(request.ResponseType, validType) {
			validResponseType = true
			break
		}
	}

	if !validResponseType {
		return &types.AuthorizationResponse{
			Error:            types.ErrorUnsupportedResponseType,
			ErrorDescription: "Unsupported response type",
		}, nil
	}

	// Validate scope if provided
	if request.Scope != "" {
		scopes := strings.Fields(request.Scope)
		scopeValidation, err := s.clientProvider.ValidateScope(ctx, request.ClientID, scopes)
		if err != nil || !scopeValidation.Valid {
			return &types.AuthorizationResponse{
				Error:            types.ErrorInvalidScope,
				ErrorDescription: "Invalid scope",
			}, nil
		}
	}

	// Generate authorization code
	authCode, err := s.generateAuthorizationCode(request.ClientID, request.State)
	if err != nil {
		return &types.AuthorizationResponse{
			Error:            types.ErrorServerError,
			ErrorDescription: "Failed to generate authorization code",
		}, nil
	}

	response := &types.AuthorizationResponse{
		Code:  authCode,
		State: request.State,
	}

	return response, nil
}

// Token exchanges an authorization code for an access token
// This is the core token endpoint functionality
func (s *Service) Token(ctx context.Context, grantType string, code string, clientID string, codeVerifier string) (*types.Token, error) {
	// Validate client
	client, err := s.clientProvider.GetClientByID(ctx, clientID)
	if err != nil {
		return nil, &types.ErrorResponse{
			Code:             types.ErrorInvalidClient,
			ErrorDescription: "Invalid client",
		}
	}

	// Validate grant type
	switch grantType {
	case types.GrantTypeAuthorizationCode:
		return s.handleAuthorizationCodeGrant(ctx, client, code, codeVerifier)
	case types.GrantTypeClientCredentials:
		return s.handleClientCredentialsGrant(ctx, client)
	case types.GrantTypeRefreshToken:
		return s.handleRefreshTokenGrant(ctx, client, code) // code is refresh token in this case
	default:
		return nil, &types.ErrorResponse{
			Code:             types.ErrorUnsupportedGrantType,
			ErrorDescription: "Unsupported grant type",
		}
	}
}

// Revoke revokes an access token or refresh token
// Once revoked, the token cannot be used for accessing protected resources
func (s *Service) Revoke(ctx context.Context, token string, tokenTypeHint string) error {
	// Revoke token using user provider
	if err := s.userProvider.RevokeToken(token); err != nil {
		return &types.ErrorResponse{
			Code:             types.ErrorInvalidToken,
			ErrorDescription: "Failed to revoke token",
		}
	}

	return nil
}

// RefreshToken exchanges a refresh token for a new access token
// This allows clients to obtain fresh access tokens without user interaction
func (s *Service) RefreshToken(ctx context.Context, refreshToken string, scope string) (*types.RefreshTokenResponse, error) {
	// Validate refresh token
	if !s.userProvider.TokenExists(refreshToken) {
		return nil, &types.ErrorResponse{
			Code:             types.ErrorInvalidGrant,
			ErrorDescription: "Invalid refresh token",
		}
	}

	// Get token data
	tokenData, err := s.userProvider.GetTokenData(refreshToken)
	if err != nil {
		return nil, &types.ErrorResponse{
			Code:             types.ErrorInvalidGrant,
			ErrorDescription: "Invalid refresh token",
		}
	}

	// Extract client ID from token data
	clientID, ok := tokenData["client_id"].(string)
	if !ok {
		return nil, &types.ErrorResponse{
			Code:             types.ErrorInvalidGrant,
			ErrorDescription: "Invalid token format",
		}
	}

	// Validate client
	client, err := s.clientProvider.GetClientByID(ctx, clientID)
	if err != nil {
		return nil, &types.ErrorResponse{
			Code:             types.ErrorInvalidClient,
			ErrorDescription: "Invalid client",
		}
	}

	// Validate scope if provided
	if scope != "" {
		scopes := strings.Fields(scope)
		scopeValidation, err := s.clientProvider.ValidateScope(ctx, client.ClientID, scopes)
		if err != nil || !scopeValidation.Valid {
			return nil, &types.ErrorResponse{
				Code:             types.ErrorInvalidScope,
				ErrorDescription: "Invalid scope",
			}
		}
	}

	// Generate new access token
	newAccessToken, err := s.generateAccessToken(clientID)
	if err != nil {
		return nil, &types.ErrorResponse{
			Code:             types.ErrorServerError,
			ErrorDescription: "Failed to generate access token",
		}
	}

	response := &types.RefreshTokenResponse{
		AccessToken: newAccessToken,
		TokenType:   "Bearer",
		ExpiresIn:   3600, // 1 hour
	}

	// Include scope if provided
	if scope != "" {
		response.Scope = scope
	}

	// Include refresh token if rotation is enabled
	if s.config.Features.RefreshTokenRotationEnabled {
		newRefreshToken, err := s.generateRefreshToken(clientID)
		if err != nil {
			return nil, &types.ErrorResponse{
				Code:             types.ErrorServerError,
				ErrorDescription: "Failed to generate refresh token",
			}
		}
		response.RefreshToken = newRefreshToken

		// Revoke old refresh token
		s.userProvider.RevokeToken(refreshToken)
	}

	return response, nil
}

// RotateRefreshToken rotates a refresh token and invalidates the old one
// This implements refresh token rotation for enhanced security
func (s *Service) RotateRefreshToken(ctx context.Context, oldToken string) (*types.RefreshTokenResponse, error) {
	// Check if refresh token rotation is enabled
	if !s.config.Features.RefreshTokenRotationEnabled {
		return nil, &types.ErrorResponse{
			Code:             types.ErrorInvalidRequest,
			ErrorDescription: "Refresh token rotation is not enabled",
		}
	}

	// Validate old token
	if !s.userProvider.TokenExists(oldToken) {
		return nil, &types.ErrorResponse{
			Code:             types.ErrorInvalidGrant,
			ErrorDescription: "Invalid refresh token",
		}
	}

	// Get token data
	tokenData, err := s.userProvider.GetTokenData(oldToken)
	if err != nil {
		return nil, &types.ErrorResponse{
			Code:             types.ErrorInvalidGrant,
			ErrorDescription: "Invalid refresh token",
		}
	}

	// Extract client ID from token data
	clientID, ok := tokenData["client_id"].(string)
	if !ok {
		return nil, &types.ErrorResponse{
			Code:             types.ErrorInvalidGrant,
			ErrorDescription: "Invalid token format",
		}
	}

	// Generate new tokens
	newAccessToken, err := s.generateAccessToken(clientID)
	if err != nil {
		return nil, &types.ErrorResponse{
			Code:             types.ErrorServerError,
			ErrorDescription: "Failed to generate access token",
		}
	}

	newRefreshToken, err := s.generateRefreshToken(clientID)
	if err != nil {
		return nil, &types.ErrorResponse{
			Code:             types.ErrorServerError,
			ErrorDescription: "Failed to generate refresh token",
		}
	}

	// Revoke old token
	err = s.userProvider.RevokeToken(oldToken)
	if err != nil {
		return nil, &types.ErrorResponse{
			Code:             types.ErrorServerError,
			ErrorDescription: "Failed to revoke old token",
		}
	}

	response := &types.RefreshTokenResponse{
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    3600, // 1 hour
	}

	return response, nil
}

// Helper methods for token grant types

// handleAuthorizationCodeGrant handles authorization code grant
func (s *Service) handleAuthorizationCodeGrant(ctx context.Context, client *types.ClientInfo, code string, codeVerifier string) (*types.Token, error) {
	// TODO: Validate authorization code
	// In a real implementation, this would validate the authorization code and extract user info

	// Generate access token
	accessToken, err := s.generateAccessToken(client.ClientID)
	if err != nil {
		return nil, &types.ErrorResponse{
			Code:             types.ErrorServerError,
			ErrorDescription: "Failed to generate access token",
		}
	}

	token := &types.Token{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   3600, // 1 hour
	}

	// Generate refresh token if supported
	if types.Contains(client.GrantTypes, types.GrantTypeRefreshToken) {
		refreshToken, err := s.generateRefreshToken(client.ClientID)
		if err != nil {
			return nil, &types.ErrorResponse{
				Code:             types.ErrorServerError,
				ErrorDescription: "Failed to generate refresh token",
			}
		}
		token.RefreshToken = refreshToken
	}

	return token, nil
}

// handleClientCredentialsGrant handles client credentials grant
func (s *Service) handleClientCredentialsGrant(ctx context.Context, client *types.ClientInfo) (*types.Token, error) {
	// Generate access token
	accessToken, err := s.generateAccessToken(client.ClientID)
	if err != nil {
		return nil, &types.ErrorResponse{
			Code:             types.ErrorServerError,
			ErrorDescription: "Failed to generate access token",
		}
	}

	token := &types.Token{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   3600, // 1 hour
	}

	return token, nil
}

// handleRefreshTokenGrant handles refresh token grant
func (s *Service) handleRefreshTokenGrant(ctx context.Context, client *types.ClientInfo, refreshToken string) (*types.Token, error) {
	// Validate refresh token
	if !s.userProvider.TokenExists(refreshToken) {
		return nil, &types.ErrorResponse{
			Code:             types.ErrorInvalidGrant,
			ErrorDescription: "Invalid refresh token",
		}
	}

	// Generate new access token
	accessToken, err := s.generateAccessToken(client.ClientID)
	if err != nil {
		return nil, &types.ErrorResponse{
			Code:             types.ErrorServerError,
			ErrorDescription: "Failed to generate access token",
		}
	}

	token := &types.Token{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   3600, // 1 hour
	}

	// Include refresh token if rotation is enabled
	if s.config.Features.RefreshTokenRotationEnabled {
		newRefreshToken, err := s.generateRefreshToken(client.ClientID)
		if err != nil {
			return nil, &types.ErrorResponse{
				Code:             types.ErrorServerError,
				ErrorDescription: "Failed to generate refresh token",
			}
		}
		token.RefreshToken = newRefreshToken

		// Revoke old refresh token
		s.userProvider.RevokeToken(refreshToken)
	} else {
		// Reuse the same refresh token
		token.RefreshToken = refreshToken
	}

	return token, nil
}
