package oauth

import (
	"context"
	"strings"
	"time"

	"github.com/yaoapp/yao/openapi/oauth/types"
	"go.mongodb.org/mongo-driver/bson/primitive"
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
	// Validate client, redirect URI, response type and scope
	if errResp := s.ValidateAuthorizationRequest(ctx, request); errResp != nil {
		return errResp, nil
	}

	// Generate authorization code with authorization information.
	// Subject and identity metadata are resolved by the authorize endpoint from
	// the Yao session; they are empty when Authorize is called directly.
	extraClaims := map[string]interface{}{}
	if request.TeamID != "" {
		extraClaims["team_id"] = request.TeamID
	}
	if request.TenantID != "" {
		extraClaims["tenant_id"] = request.TenantID
	}
	if len(extraClaims) == 0 {
		extraClaims = nil
	}
	authCode, err := s.generateAuthorizationCodeWithExtra(
		request.ClientID,
		request.State,
		request.Scope,               // Store the requested scope for validation
		request.CodeChallenge,       // PKCE code challenge
		request.CodeChallengeMethod, // PKCE method
		request.Subject,             // Opaque OAuth subject resolved from the Yao session
		request.AuthSource,          // auth_source claim (tao | password)
		request.IdentitySource,      // user.identity_source (tao | local)
		extraClaims,
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

// ValidateAuthorizationRequest validates an authorization request without
// issuing an authorization code. It returns nil when the request is valid and
// an AuthorizationResponse carrying the OAuth error otherwise.
//
// The authorize endpoint uses this to reject invalid clients/redirect
// URIs/response types/scopes before any login redirect is performed.
func (s *Service) ValidateAuthorizationRequest(ctx context.Context, request *types.AuthorizationRequest) *types.AuthorizationResponse {
	if request == nil {
		return &types.AuthorizationResponse{
			Error:            types.ErrorInvalidRequest,
			ErrorDescription: "Missing authorization request",
		}
	}

	// Validate client
	client, err := s.clientProvider.GetClientByID(ctx, request.ClientID)
	if err != nil {
		return &types.AuthorizationResponse{
			Error:            types.ErrorInvalidClient,
			ErrorDescription: "Invalid client",
		}
	}
	// RFC 6749 allows the server to use a default scope when the client
	// omits it. Use the scope registered on the client so a pre-registered
	// desktop public client does not need to duplicate the list in its UI.
	if request.Scope == "" && client != nil {
		request.Scope = client.Scope
	}

	// Validate redirect URI
	if request.RedirectURI == "" {
		return &types.AuthorizationResponse{
			Error:            types.ErrorInvalidRequest,
			ErrorDescription: "Missing redirect URI",
		}
	}

	validationResult, err := s.clientProvider.ValidateRedirectURI(ctx, request.ClientID, request.RedirectURI)
	if err != nil || !validationResult.Valid {
		return &types.AuthorizationResponse{
			Error:            types.ErrorInvalidRequest,
			ErrorDescription: "Invalid redirect URI",
		}
	}

	// Validate response type
	if request.ResponseType == "" {
		return &types.AuthorizationResponse{
			Error:            types.ErrorInvalidRequest,
			ErrorDescription: "Missing response type",
		}
	}

	validResponseTypes := []string{"code"}
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
		}
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
			}
		}
	}

	return nil
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
	case types.GrantTypeDeviceCode:
		return s.handleDeviceCodeGrant(ctx, client, code) // code is device_code in this case
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

	extraClaims := extractExtraClaims(tokenInfo)

	// Generate new access token with final scope
	expiresIn := int(s.config.Token.AccessTokenLifetime.Seconds())
	newAccessToken, err := s.generateAccessTokenWithScope(clientID, finalScope, originalSubject, expiresIn, extraClaims)
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

	extraClaims := extractExtraClaims(tokenInfo)

	// Generate new tokens with final scope and original subject
	expiresIn := int(s.config.Token.AccessTokenLifetime.Seconds())
	newAccessToken, err := s.generateAccessTokenWithScope(clientID, finalScope, originalSubject, expiresIn, extraClaims)
	if err != nil {
		return nil, &types.ErrorResponse{
			Code:             types.ErrorServerError,
			ErrorDescription: "Failed to generate access token",
		}
	}

	newRefreshToken, err := s.generateRefreshToken(clientID, finalScope, originalSubject, 0, extraClaims)
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

	// Carry the identity metadata recorded on the authorization code into the
	// issued tokens. Team/Tenant claims are propagated so profile/team context
	// selected during authorization is available on the token.
	extraClaims := authorizationCodeIdentityClaims(codeInfo)

	// Generate and store access token with proper scope and subject
	expiresIn := int(s.config.Token.AccessTokenLifetime.Seconds())
	accessToken, err := s.generateAccessTokenWithScope(client.ClientID, scope, subject, expiresIn, extraClaims)
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
	if s.userProvider != nil && strings.Contains(" "+scope+" ", " openid ") && subject != "" {
		if userID, idErr := s.UserID(client.ClientID, subject); idErr == nil {
			if user, userErr := s.userProvider.GetUser(ctx, userID); userErr == nil && user != nil {
				userInfo := types.MakeOIDCUserInfo(user)
				userInfo.Sub = subject
				userInfo.YaoUserID = userID
				if idToken, signErr := s.SignIDToken(client.ClientID, scope, expiresIn, userInfo); signErr == nil {
					token.IDToken = idToken
				}
			}
		}
	}

	// Generate refresh token if supported
	if types.Contains(client.GrantTypes, types.GrantTypeRefreshToken) {
		refreshToken, err := s.generateRefreshToken(client.ClientID, scope, subject, 0, extraClaims)
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
	accessToken, err := s.generateAccessTokenWithScope(client.ClientID, scope, "", expiresIn, nil)
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

	scope, _ := refreshTokenInfo["scope"].(string)
	subject, _ := refreshTokenInfo["subject"].(string)
	extraClaims := extractExtraClaims(refreshTokenInfo)

	expiresIn := int(s.config.Token.AccessTokenLifetime.Seconds())
	accessToken, err := s.generateAccessTokenWithScope(client.ClientID, scope, subject, expiresIn, extraClaims)
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

	if s.config.Features.RefreshTokenRotationEnabled {
		newRefreshToken, err := s.generateRefreshToken(client.ClientID, scope, subject, 0, extraClaims)
		if err != nil {
			return nil, &types.ErrorResponse{
				Code:             types.ErrorServerError,
				ErrorDescription: "Failed to generate refresh token",
			}
		}
		token.RefreshToken = newRefreshToken
		s.revokeRefreshToken(refreshToken)
	} else {
		token.RefreshToken = refreshToken
	}

	return token, nil
}

// validatePKCE validates PKCE code verifier against stored code challenge
func (s *Service) validatePKCE(ctx context.Context, client *types.ClientInfo, codeInfo map[string]interface{}, codeVerifier string) error {
	// OAuth 2.1 requires PKCE for public AND confidential clients.
	// ClientInfo.RequirePKCE records the policy on the client; Yao forces it
	// for every authorization-code exchange.
	isPKCERequired := true

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

// handleDeviceCodeGrant handles the device_code grant type (RFC 8628 Section 3.4).
func (s *Service) handleDeviceCodeGrant(ctx context.Context, client *types.ClientInfo, deviceCode string) (*types.Token, error) {
	if !s.config.Features.DeviceFlowEnabled {
		return nil, &types.ErrorResponse{
			Code:             types.ErrorUnsupportedGrantType,
			ErrorDescription: "Device flow is not enabled",
		}
	}

	codeData, err := s.getDeviceCodeData(deviceCode)
	if err != nil {
		return nil, err
	}

	storedClientID, _ := codeData["client_id"].(string)
	if storedClientID != client.ClientID {
		return nil, &types.ErrorResponse{
			Code:             types.ErrorInvalidGrant,
			ErrorDescription: "Device code was issued to a different client",
		}
	}

	expiresAt, _ := codeData["expires_at"].(int64)
	if expiresAt == 0 {
		if f, ok := codeData["expires_at"].(float64); ok {
			expiresAt = int64(f)
		}
	}
	if expiresAt > 0 && time.Now().Unix() > expiresAt {
		s.consumeDeviceCode(deviceCode)
		return nil, &types.ErrorResponse{
			Code:             types.ErrorExpiredToken,
			ErrorDescription: "Device code has expired",
		}
	}

	status, _ := codeData["status"].(string)
	switch status {
	case "pending":
		return nil, &types.ErrorResponse{
			Code:             types.ErrorAuthorizationPending,
			ErrorDescription: "The authorization request is still pending",
		}

	case "authorized":
		scope, _ := codeData["scope"].(string)
		subject, _ := codeData["subject"].(string)
		s.consumeDeviceCode(deviceCode)

		var extraClaims map[string]interface{}
		if ec, ok := codeData["extra_claims"]; ok {
			switch v := ec.(type) {
			case map[string]interface{}:
				extraClaims = v
			case primitive.M:
				extraClaims = map[string]interface{}(v)
			}
		}

		expiresIn := int(s.config.Token.AccessTokenLifetime.Seconds())
		accessToken, err := s.generateAccessTokenWithScope(client.ClientID, scope, subject, expiresIn, extraClaims)
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

		if types.Contains(client.GrantTypes, types.GrantTypeRefreshToken) {
			refreshToken, err := s.generateRefreshToken(client.ClientID, scope, subject, 0, extraClaims)
			if err != nil {
				return nil, &types.ErrorResponse{
					Code:             types.ErrorServerError,
					ErrorDescription: "Failed to generate refresh token",
				}
			}
			token.RefreshToken = refreshToken
		}

		return token, nil

	default:
		return nil, &types.ErrorResponse{
			Code:             types.ErrorInvalidGrant,
			ErrorDescription: "Invalid device code status",
		}
	}
}

// authorizationCodeIdentityClaims returns the identity metadata stored on an
// authorization code that must be carried into the issued access/refresh
// tokens. Only auth_source and identity_source are propagated: Team/Tenant
// claims are deliberately not part of OAuth identity.
func authorizationCodeIdentityClaims(codeInfo map[string]interface{}) map[string]interface{} {
	if codeInfo == nil {
		return nil
	}

	var claims map[string]interface{}
	for _, key := range []string{types.ClaimAuthSource, types.ClaimIdentitySource, "team_id", "tenant_id"} {
		value, ok := codeInfo[key].(string)
		if !ok || value == "" {
			continue
		}
		if claims == nil {
			claims = make(map[string]interface{}, 2)
		}
		claims[key] = value
	}
	return claims
}

// extractExtraClaims pulls non-reserved fields from a token info map so they
// can be propagated into newly generated access/refresh tokens.
func extractExtraClaims(tokenInfo map[string]interface{}) map[string]interface{} {
	reserved := map[string]bool{
		"client_id": true, "scope": true, "subject": true,
		"type": true, "issued_at": true, "expires_at": true,
	}
	var extra map[string]interface{}
	for k, v := range tokenInfo {
		if reserved[k] {
			continue
		}
		if extra == nil {
			extra = make(map[string]interface{})
		}
		extra[k] = v
	}
	return extra
}
