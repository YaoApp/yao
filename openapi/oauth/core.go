package oauth

import (
	"context"
	"strings"
	"time"

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
	// TODO:
	//  1. Should validate scope, if not provide, use the default scope
	//  2. If scope has "openid", should be redirect to the login page/mobile app authentication
	//  3. If scope not has "openid", can't visit the userinfo endpoint
	//  4. Security check
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

	// Generate authorization code with authorization information
	// TODO: Future implementation will generate subject here after user authentication
	authCode, err := s.generateAuthorizationCodeWithInfo(
		request.ClientID,
		request.State,
		request.Scope,               // Store the requested scope for validation
		request.CodeChallenge,       // PKCE code challenge
		request.CodeChallengeMethod, // PKCE method
	)
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
	// Try to revoke as access token first
	if tokenTypeHint == "" || tokenTypeHint == "access_token" {
		// Check if it's an access token
		_, err := s.getAccessTokenData(token)
		if err == nil {
			s.revokeAccessToken(token)
			return nil
		}
	}

	// Try to revoke as refresh token
	if tokenTypeHint == "" || tokenTypeHint == "refresh_token" {
		// Check if it's a refresh token
		_, err := s.getRefreshTokenData(token)
		if err == nil {
			s.revokeRefreshToken(token)
			return nil
		}
	}

	// If token not found in either store, still return success (RFC 7009)
	// This prevents information leakage about token existence
	return nil
}

// RefreshToken exchanges a refresh token for a new access token
// This allows clients to obtain fresh access tokens without user interaction
func (s *Service) RefreshToken(ctx context.Context, refreshToken string, scope ...string) (*types.RefreshTokenResponse, error) {
	// Check if refresh token rotation is enabled and call RotateRefreshToken directly
	if s.config.Features.RefreshTokenRotationEnabled {
		return s.RotateRefreshToken(ctx, refreshToken, scope...)
	}

	// Get and validate refresh token data
	tokenInfo, err := s.getRefreshTokenData(refreshToken)
	if err != nil {
		return nil, err
	}

	// Extract client ID from token data
	clientID, ok := tokenInfo["client_id"].(string)
	if !ok {
		return nil, &types.ErrorResponse{
			Code:             types.ErrorInvalidGrant,
			ErrorDescription: "Invalid token format",
		}
	}

	// Validate client exists
	_, err = s.clientProvider.GetClientByID(ctx, clientID)
	if err != nil {
		return nil, &types.ErrorResponse{
			Code:             types.ErrorInvalidClient,
			ErrorDescription: "Invalid client",
		}
	}

	// Extract original scope and subject from refresh token data
	originalScope := ""
	if originalScopeVal, ok := tokenInfo["scope"].(string); ok {
		originalScope = originalScopeVal
	}
	originalSubject := ""
	if originalSubjectVal, ok := tokenInfo["subject"].(string); ok {
		originalSubject = originalSubjectVal
	}

	// Handle scope according to OAuth 2.0 spec:
	// - If scope is omitted, treat as equal to the scope originally granted
	// - If scope is provided, it MUST NOT include any scope not originally granted
	finalScope := originalScope // Default to original scope
	if len(scope) > 0 && scope[0] != "" {
		requestedScope := scope[0]
		// Validate that requested scope doesn't exceed original scope
		requestedScopes := strings.Fields(requestedScope)
		originalScopes := strings.Fields(originalScope)

		// Convert original scopes to a map for easier lookup
		originalScopeMap := make(map[string]bool)
		for _, s := range originalScopes {
			originalScopeMap[s] = true
		}

		// Check that all requested scopes were originally granted
		for _, reqScope := range requestedScopes {
			if !originalScopeMap[reqScope] {
				return nil, &types.ErrorResponse{
					Code:             types.ErrorInvalidScope,
					ErrorDescription: "Requested scope exceeds originally granted scope",
				}
			}
		}

		finalScope = requestedScope
	}

	// Generate new access token with final scope
	expiresIn := int(s.config.Token.AccessTokenLifetime.Seconds())
	newAccessToken, err := s.generateAccessTokenWithScope(clientID, finalScope, originalSubject, expiresIn)
	if err != nil {
		return nil, &types.ErrorResponse{
			Code:             types.ErrorServerError,
			ErrorDescription: "Failed to generate access token",
		}
	}

	response := &types.RefreshTokenResponse{
		AccessToken:  newAccessToken,
		RefreshToken: refreshToken, // Reuse the same refresh token (no rotation)
		TokenType:    "Bearer",
		ExpiresIn:    expiresIn,
	}

	// Include scope if different from originally granted
	if finalScope != originalScope {
		response.Scope = finalScope
	}

	return response, nil
}

// RotateRefreshToken rotates a refresh token and invalidates the old one
// This implements refresh token rotation for enhanced security
func (s *Service) RotateRefreshToken(ctx context.Context, oldToken string, requestedScope ...string) (*types.RefreshTokenResponse, error) {
	// Check if refresh token rotation is enabled
	if !s.config.Features.RefreshTokenRotationEnabled {
		return nil, &types.ErrorResponse{
			Code:             types.ErrorInvalidRequest,
			ErrorDescription: "Refresh token rotation is not enabled",
		}
	}

	// Get and validate refresh token data
	tokenInfo, err := s.getRefreshTokenData(oldToken)
	if err != nil {
		return nil, err
	}

	// Extract client ID from token data
	clientID, ok := tokenInfo["client_id"].(string)
	if !ok {
		return nil, &types.ErrorResponse{
			Code:             types.ErrorInvalidGrant,
			ErrorDescription: "Invalid token format",
		}
	}

	// Validate client exists
	_, err = s.clientProvider.GetClientByID(ctx, clientID)
	if err != nil {
		return nil, &types.ErrorResponse{
			Code:             types.ErrorInvalidClient,
			ErrorDescription: "Invalid client",
		}
	}

	// Extract original scope and subject from refresh token data
	originalScope := ""
	if originalScopeVal, ok := tokenInfo["scope"].(string); ok {
		originalScope = originalScopeVal
	}
	originalSubject := ""
	if originalSubjectVal, ok := tokenInfo["subject"].(string); ok {
		originalSubject = originalSubjectVal
	}

	// Handle scope according to OAuth 2.0 spec:
	// - If scope is omitted, treat as equal to the scope originally granted
	// - If scope is provided, it MUST NOT include any scope not originally granted
	finalScope := originalScope // Default to original scope
	if len(requestedScope) > 0 && requestedScope[0] != "" {
		scope := requestedScope[0]
		// Validate that requested scope doesn't exceed original scope
		requestedScopes := strings.Fields(scope)
		originalScopes := strings.Fields(originalScope)

		// Convert original scopes to a map for easier lookup
		originalScopeMap := make(map[string]bool)
		for _, s := range originalScopes {
			originalScopeMap[s] = true
		}

		// Check that all requested scopes were originally granted
		for _, requestedScopeItem := range requestedScopes {
			if !originalScopeMap[requestedScopeItem] {
				return nil, &types.ErrorResponse{
					Code:             types.ErrorInvalidScope,
					ErrorDescription: "Requested scope exceeds originally granted scope",
				}
			}
		}

		finalScope = scope
	}

	// Generate new tokens with final scope and original subject
	expiresIn := int(s.config.Token.AccessTokenLifetime.Seconds())
	newAccessToken, err := s.generateAccessTokenWithScope(clientID, finalScope, originalSubject, expiresIn)
	if err != nil {
		return nil, &types.ErrorResponse{
			Code:             types.ErrorServerError,
			ErrorDescription: "Failed to generate access token",
		}
	}

	newRefreshToken, err := s.generateRefreshToken(clientID, finalScope, originalSubject)
	if err != nil {
		return nil, &types.ErrorResponse{
			Code:             types.ErrorServerError,
			ErrorDescription: "Failed to generate refresh token",
		}
	}

	// Revoke old token
	s.revokeRefreshToken(oldToken)

	response := &types.RefreshTokenResponse{
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    expiresIn,
	}

	// Include scope if different from originally granted
	if finalScope != originalScope {
		response.Scope = finalScope
	}

	return response, nil
}

// Helper methods for token grant types

// handleAuthorizationCodeGrant handles authorization code grant
func (s *Service) handleAuthorizationCodeGrant(ctx context.Context, client *types.ClientInfo, code string, codeVerifier string) (*types.Token, error) {
	// Get and validate authorization code data
	codeInfo, err := s.getAuthorizationCodeData(code)
	if err != nil {
		return nil, err
	}

	// Validate that the code belongs to the requesting client
	codeClientID, ok := codeInfo["client_id"].(string)
	if !ok || codeClientID != client.ClientID {
		return nil, &types.ErrorResponse{
			Code:             types.ErrorInvalidGrant,
			ErrorDescription: "Authorization code does not belong to this client",
		}
	}

	// Check if code has expired
	expiresAt, ok := codeInfo["expires_at"].(int64)
	if ok && time.Now().Unix() > expiresAt {
		// Clean up expired code
		s.consumeAuthorizationCode(code)
		return nil, &types.ErrorResponse{
			Code:             types.ErrorInvalidGrant,
			ErrorDescription: "Authorization code has expired",
		}
	}

	// PKCE validation (Proof Key for Code Exchange)
	err = s.validatePKCE(ctx, client, codeInfo, codeVerifier)
	if err != nil {
		// Clean up the code since validation failed
		s.consumeAuthorizationCode(code)
		return nil, err
	}

	// Code is valid, consume it (delete it to prevent reuse)
	s.consumeAuthorizationCode(code)

	// Extract scope from authorization code
	scope := ""
	if scopeVal, ok := codeInfo["scope"].(string); ok {
		scope = scopeVal
	}

	// Extract subject from authorization code if available
	subject := ""
	if subjectVal, ok := codeInfo["subject"].(string); ok {
		subject = subjectVal
	}

	// Generate and store access token with proper scope and subject
	expiresIn := int(s.config.Token.AccessTokenLifetime.Seconds())
	accessToken, err := s.generateAccessTokenWithScope(client.ClientID, scope, subject, expiresIn)
	if err != nil {
		return nil, &types.ErrorResponse{
			Code:             types.ErrorServerError,
			ErrorDescription: "Failed to generate access token",
		}
	}

	token := &types.Token{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   expiresIn,
	}

	// Generate refresh token if supported
	if types.Contains(client.GrantTypes, types.GrantTypeRefreshToken) {
		refreshToken, err := s.generateRefreshToken(client.ClientID, scope, subject)
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
	// Use client's configured scope for client credentials grant
	scope := client.Scope

	// Generate and store access token with client's scope (no user subject for client credentials)
	expiresIn := int(s.config.Token.AccessTokenLifetime.Seconds())
	accessToken, err := s.generateAccessTokenWithScope(client.ClientID, scope, "", expiresIn)
	if err != nil {
		return nil, &types.ErrorResponse{
			Code:             types.ErrorServerError,
			ErrorDescription: "Failed to generate access token",
		}
	}

	token := &types.Token{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   expiresIn,
	}

	// Include scope in response if client has configured scope
	if scope != "" {
		token.Scope = scope
	}

	return token, nil
}

// handleRefreshTokenGrant handles refresh token grant
func (s *Service) handleRefreshTokenGrant(ctx context.Context, client *types.ClientInfo, refreshToken string) (*types.Token, error) {
	// Get and validate refresh token data
	refreshTokenInfo, err := s.getRefreshTokenData(refreshToken)
	if err != nil {
		return nil, err
	}

	// Extract scope and subject from refresh token if available
	scope := ""
	if scopeVal, ok := refreshTokenInfo["scope"].(string); ok {
		scope = scopeVal
	}

	subject := ""
	if subjectVal, ok := refreshTokenInfo["subject"].(string); ok {
		subject = subjectVal
	}

	// Generate and store new access token with proper scope and subject
	expiresIn := int(s.config.Token.AccessTokenLifetime.Seconds())
	accessToken, err := s.generateAccessTokenWithScope(client.ClientID, scope, subject, expiresIn)
	if err != nil {
		return nil, &types.ErrorResponse{
			Code:             types.ErrorServerError,
			ErrorDescription: "Failed to generate access token",
		}
	}

	token := &types.Token{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   expiresIn,
	}

	// Include refresh token if rotation is enabled
	if s.config.Features.RefreshTokenRotationEnabled {
		newRefreshToken, err := s.generateRefreshToken(client.ClientID, scope, subject)
		if err != nil {
			return nil, &types.ErrorResponse{
				Code:             types.ErrorServerError,
				ErrorDescription: "Failed to generate refresh token",
			}
		}
		token.RefreshToken = newRefreshToken

		// Revoke old refresh token
		s.revokeRefreshToken(refreshToken)
	} else {
		// Reuse the same refresh token
		token.RefreshToken = refreshToken
	}

	return token, nil
}

// validatePKCE validates PKCE code verifier against stored code challenge
func (s *Service) validatePKCE(ctx context.Context, client *types.ClientInfo, codeInfo map[string]interface{}, codeVerifier string) error {
	// Check if PKCE is required
	isPKCERequired := s.config.Security.PKCERequired

	// For OAuth 2.1, PKCE is mandatory for public clients
	if client.ClientType == types.ClientTypePublic {
		isPKCERequired = true
	}

	// Extract code challenge information from stored authorization code
	codeChallenge := ""
	if challengeVal, ok := codeInfo["code_challenge"].(string); ok {
		codeChallenge = challengeVal
	}

	codeChallengeMethod := ""
	if methodVal, ok := codeInfo["code_challenge_method"].(string); ok {
		codeChallengeMethod = methodVal
	}

	// Check if PKCE is required but not provided
	if isPKCERequired && (codeVerifier == "" || codeChallenge == "") {
		return &types.ErrorResponse{
			Code:             types.ErrorInvalidRequest,
			ErrorDescription: "PKCE is required but code verifier or code challenge is missing",
		}
	}

	// If code verifier is provided, validate it
	if codeVerifier != "" {
		if codeChallenge == "" {
			return &types.ErrorResponse{
				Code:             types.ErrorInvalidGrant,
				ErrorDescription: "Code challenge not found for provided code verifier",
			}
		}

		// Use default method if not specified
		if codeChallengeMethod == "" {
			codeChallengeMethod = types.CodeChallengeMethodS256
		}

		// Validate that the method is supported
		supportedMethods := s.config.Security.PKCECodeChallengeMethod
		if len(supportedMethods) > 0 {
			methodSupported := false
			for _, method := range supportedMethods {
				if method == codeChallengeMethod {
					methodSupported = true
					break
				}
			}
			if !methodSupported {
				return &types.ErrorResponse{
					Code:             types.ErrorInvalidRequest,
					ErrorDescription: "Code challenge method not supported",
				}
			}
		}

		// Validate the code verifier against the challenge
		err := s.ValidateCodeChallenge(ctx, codeVerifier, codeChallenge, codeChallengeMethod)
		if err != nil {
			return &types.ErrorResponse{
				Code:             types.ErrorInvalidGrant,
				ErrorDescription: "Code verifier validation failed",
			}
		}
	}

	return nil
}
