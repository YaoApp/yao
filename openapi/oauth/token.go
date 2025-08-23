package oauth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/yaoapp/yao/openapi/oauth/types"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Introspect returns information about an access token
// This endpoint allows resource servers to validate tokens
func (s *Service) Introspect(ctx context.Context, token string) (*types.TokenIntrospectionResponse, error) {
	// Try to verify token using signature verification first
	tokenClaims, err := s.VerifyToken(token)
	if err != nil {
		// If signature verification fails, try to get from store (for opaque tokens)
		return s.introspectFromStore(token)
	}

	// Token is valid, build response from verified claims
	response := &types.TokenIntrospectionResponse{
		Active:    true,
		ClientID:  tokenClaims.ClientID,
		Subject:   tokenClaims.Subject,
		Scope:     tokenClaims.Scope,
		TokenType: "Bearer",
		ExpiresAt: tokenClaims.ExpiresAt.Unix(),
		IssuedAt:  tokenClaims.IssuedAt.Unix(),
	}

	// Check if token is expired
	if !tokenClaims.ExpiresAt.IsZero() && time.Now().After(tokenClaims.ExpiresAt) {
		response.Active = false
	}

	return response, nil
}

// introspectFromStore fallback method for token introspection from store
func (s *Service) introspectFromStore(token string) (*types.TokenIntrospectionResponse, error) {
	// Try to get token data from OAuth store
	tokenInfo, err := s.getAccessTokenData(token)
	if err != nil {
		return &types.TokenIntrospectionResponse{Active: false}, nil
	}

	// Check if token exists and is valid
	if tokenInfo == nil {
		return &types.TokenIntrospectionResponse{Active: false}, nil
	}

	// Extract token information
	response := &types.TokenIntrospectionResponse{
		Active: true,
	}

	// Extract standard fields from token data
	if clientID, ok := tokenInfo["client_id"].(string); ok {
		response.ClientID = clientID
	}
	if subject, ok := tokenInfo["subject"].(string); ok {
		response.Subject = subject
	}
	if tokenType, ok := tokenInfo["token_type"].(string); ok {
		response.TokenType = tokenType
	} else {
		response.TokenType = "Bearer"
	}
	if scope, ok := tokenInfo["scope"].(string); ok {
		response.Scope = scope
	}
	if exp, ok := tokenInfo["expires_at"].(int64); ok {
		response.ExpiresAt = exp
	}
	if iat, ok := tokenInfo["issued_at"].(int64); ok {
		response.IssuedAt = iat
	}

	// Check if token is expired
	if response.ExpiresAt > 0 && time.Now().Unix() > response.ExpiresAt {
		response.Active = false
	}

	return response, nil
}

// TokenExchange exchanges one token for another token
// This implements RFC 8693 for token exchange scenarios
func (s *Service) TokenExchange(ctx context.Context, subjectToken string, subjectTokenType string, audience string, scope string) (*types.TokenExchangeResponse, error) {
	// Check if token exchange is enabled
	if !s.config.Features.TokenExchangeEnabled {
		return nil, &types.ErrorResponse{
			Code:             types.ErrorUnsupportedGrantType,
			ErrorDescription: "Token exchange is not enabled",
		}
	}

	// Validate subject token
	introspectionResult, err := s.Introspect(ctx, subjectToken)
	if err != nil {
		return nil, &types.ErrorResponse{
			Code:             types.ErrorInvalidGrant,
			ErrorDescription: "Invalid subject token",
		}
	}

	if !introspectionResult.Active {
		return nil, &types.ErrorResponse{
			Code:             types.ErrorInvalidGrant,
			ErrorDescription: "Subject token is not active",
		}
	}

	// Validate audience if provided
	if audience != "" {
		if err := s.validateAudience(audience); err != nil {
			return nil, &types.ErrorResponse{
				Code:             types.ErrorInvalidRequest,
				ErrorDescription: "Invalid audience",
			}
		}
	}

	// Validate scope if provided
	if scope != "" {
		scopes := strings.Fields(scope)
		if introspectionResult.ClientID != "" {
			scopeValidation, err := s.clientProvider.ValidateScope(ctx, introspectionResult.ClientID, scopes)
			if err != nil {
				return nil, err
			}
			if !scopeValidation.Valid {
				return nil, &types.ErrorResponse{
					Code:             types.ErrorInvalidScope,
					ErrorDescription: "Invalid scope",
				}
			}
		}
	}

	// Generate new token for exchange
	newToken, err := s.generateExchangedToken(subjectToken, audience)
	if err != nil {
		return nil, &types.ErrorResponse{
			Code:             types.ErrorServerError,
			ErrorDescription: "Failed to generate exchanged token",
		}
	}

	response := &types.TokenExchangeResponse{
		AccessToken:     newToken,
		IssuedTokenType: "urn:ietf:params:oauth:token-type:access_token",
		TokenType:       "Bearer",
		ExpiresIn:       int(s.config.Token.AccessTokenLifetime.Seconds()),
	}

	if scope != "" {
		response.Scope = scope
	}

	return response, nil
}

// ValidateTokenAudience validates token audience claims
// This ensures tokens are only used with their intended audiences
func (s *Service) ValidateTokenAudience(ctx context.Context, token string, expectedAudience string) (*types.ValidationResult, error) {
	result := &types.ValidationResult{Valid: false}

	// Get token introspection
	introspectionResult, err := s.Introspect(ctx, token)
	if err != nil {
		return nil, err
	}

	if !introspectionResult.Active {
		result.Errors = append(result.Errors, "Token is not active")
		return result, nil
	}

	// Check audience
	if len(introspectionResult.Audience) == 0 {
		// If no audience is specified in token, allow access
		result.Valid = true
		return result, nil
	}

	// Check if expected audience is in token audience list
	for _, aud := range introspectionResult.Audience {
		if aud == expectedAudience {
			result.Valid = true
			return result, nil
		}
	}

	result.Errors = append(result.Errors, "Token audience does not match expected audience")
	return result, nil
}

// ValidateTokenBinding validates token binding information
// This ensures tokens are bound to the correct client or device
func (s *Service) ValidateTokenBinding(ctx context.Context, token string, binding *types.TokenBinding) (*types.ValidationResult, error) {
	result := &types.ValidationResult{Valid: false}

	// Check if token binding is enabled
	if !s.config.Features.TokenBindingEnabled {
		result.Valid = true // If not enabled, always valid
		return result, nil
	}

	// Get token introspection
	introspectionResult, err := s.Introspect(ctx, token)
	if err != nil {
		return nil, err
	}

	if !introspectionResult.Active {
		result.Errors = append(result.Errors, "Token is not active")
		return result, nil
	}

	// Validate binding type
	switch binding.BindingType {
	case types.TokenBindingTypeDPoP:
		// DPoP binding validation would go here
		result.Valid = true // Placeholder
	case types.TokenBindingTypeMTLS:
		// mTLS binding validation would go here
		result.Valid = true // Placeholder
	case types.TokenBindingTypeCertificate:
		// Certificate binding validation would go here
		result.Valid = true // Placeholder
	default:
		result.Errors = append(result.Errors, "Unknown token binding type")
		return result, nil
	}

	return result, nil
}

// ============================================================================
// Public Token helper methods for internal use
// ============================================================================

// MakeAccessToken generates a new access token with specific parameters and stores it
func (s *Service) MakeAccessToken(clientID, scope, subject string, expiresIn int) (string, error) {
	return s.generateAccessTokenWithScope(clientID, scope, subject, expiresIn)
}

// MakeRefreshToken generates a new refresh token with specific parameters and stores it
func (s *Service) MakeRefreshToken(clientID, scope, subject string, expiresIn ...int) (string, error) {
	return s.generateRefreshToken(clientID, scope, subject, expiresIn...)
}

// Subject converts a userID to a subject using NanoID fingerprint
func (s *Service) Subject(clientID, userID string) (string, error) {
	// Check if mapping already exists for this clientID+userID
	mappingKey := s.userMappingKey(clientID, userID)
	if existingNanoID, exists := s.store.Get(mappingKey); exists {
		if nanoIDStr, ok := existingNanoID.(string); ok {
			return nanoIDStr, nil
		}
	}

	maxRetries := 5
	for i := 0; i < maxRetries; i++ {
		// Generate 16-character NanoID
		nanoID, err := generateNumericID(16)
		if err != nil {
			return "", fmt.Errorf("failed to generate NanoID: %w", err)
		}

		// Check if this NanoID already exists for this client
		key := s.userFingerprintKey(clientID, nanoID)
		_, exists := s.store.Get(key)
		if !exists {
			// Store both mappings
			// 1. clientID:nanoID -> userID
			if err := s.store.Set(key, userID, 0); err != nil {
				return "", fmt.Errorf("failed to store user fingerprint: %w", err)
			}
			// 2. clientID:userID -> nanoID (for checking existing mapping)
			if err := s.store.Set(mappingKey, nanoID, 0); err != nil {
				return "", fmt.Errorf("failed to store user mapping: %w", err)
			}
			return nanoID, nil
		}
	}

	return "", fmt.Errorf("failed to generate unique NanoID after %d retries", maxRetries)
}

// UserID converts a subject to a userID using fingerprint lookup
func (s *Service) UserID(clientID, subject string) (string, error) {
	key := s.userFingerprintKey(clientID, subject)
	userID, exists := s.store.Get(key)
	if !exists {
		return "", fmt.Errorf("fingerprint not found")
	}

	userIDStr, ok := userID.(string)
	if !ok {
		return "", fmt.Errorf("invalid userID format")
	}

	return userIDStr, nil
}

// MakeAuthorizationCode generates a new authorization code with specific parameters and stores it

// ============================================================================
// Helper methods
// ============================================================================

// validateAudience validates if an audience is valid
func (s *Service) validateAudience(audience string) error {
	// Basic audience validation
	if audience == "" {
		return &types.ErrorResponse{
			Code:             types.ErrorInvalidRequest,
			ErrorDescription: "Audience cannot be empty",
		}
	}

	// Add more sophisticated audience validation here
	// For example, checking against a whitelist of valid audiences

	return nil
}

// Token generation helper methods

// generateAccessToken generates a new access token
func (s *Service) generateAccessToken(clientID string) (string, error) {
	expiresIn := int(s.config.Token.AccessTokenLifetime.Seconds())
	return s.generateAccessTokenWithScope(clientID, "", "", expiresIn)
}

// generateAccessTokenWithScope generates a new access token with specific parameters and stores it
func (s *Service) generateAccessTokenWithScope(clientID, scope, subject string, expiresIn int) (string, error) {
	// Use the new signing mechanism based on configuration
	accessToken, err := s.SignToken("access_token", clientID, scope, subject, expiresIn)
	if err != nil {
		return "", err
	}

	// Store access token with metadata
	err = s.storeAccessToken(accessToken, clientID, scope, subject, expiresIn)
	if err != nil {
		return "", err
	}

	return accessToken, nil
}

// storeAccessToken stores access token with metadata and specified expiration
func (s *Service) storeAccessToken(accessToken, clientID string, scope string, subject string, expiresIn int) error {
	now := time.Now()
	expiresAt := now.Add(time.Duration(expiresIn) * time.Second).Unix()

	tokenData := map[string]interface{}{
		"client_id":  clientID,
		"type":       "access_token",
		"scope":      scope,
		"subject":    subject,
		"token_type": "Bearer",
		"issued_at":  now.Unix(),
		"expires_at": expiresAt,
	}

	ttl := time.Duration(expiresIn) * time.Second
	return s.store.Set(s.accessTokenKey(accessToken), tokenData, ttl)
}

// getAccessTokenData retrieves access token data
func (s *Service) getAccessTokenData(accessToken string) (map[string]interface{}, error) {
	tokenData, exists := s.store.Get(s.accessTokenKey(accessToken))
	if !exists {
		return nil, &types.ErrorResponse{
			Code:             types.ErrorInvalidToken,
			ErrorDescription: "Invalid access token",
		}
	}

	// Convert to map[string]interface{} if needed
	tokenInfo, ok := tokenData.(map[string]interface{})
	if !ok {
		// Try primitive.M for MongoDB store compatibility
		if primitiveM, isPrimitiveM := tokenData.(primitive.M); isPrimitiveM {
			// Convert primitive.M to map[string]interface{}
			tokenInfo = make(map[string]interface{})
			for k, v := range primitiveM {
				tokenInfo[k] = v
			}
		} else {
			return nil, &types.ErrorResponse{
				Code:             types.ErrorInvalidToken,
				ErrorDescription: "Invalid token format",
			}
		}
	}

	return tokenInfo, nil
}

// revokeAccessToken deletes access token from store
func (s *Service) revokeAccessToken(accessToken string) error {
	s.store.Del(s.accessTokenKey(accessToken))
	return nil
}

// generateRefreshToken generates and stores a new refresh token with scope and subject
func (s *Service) generateRefreshToken(clientID, scope, subject string, expiresIn ...int) (string, error) {
	refreshToken, err := s.generateToken("rfk", clientID)
	if err != nil {
		return "", err
	}

	// Store refresh token with metadata
	err = s.storeRefreshTokenWithScope(refreshToken, clientID, scope, subject, expiresIn...)
	if err != nil {
		return "", err
	}

	return refreshToken, nil
}

// generateAuthorizationCodeWithInfo generates a new authorization code with authorization information
func (s *Service) generateAuthorizationCodeWithInfo(clientID, state, scope, codeChallenge, codeChallengeMethod string, subject ...string) (string, error) {
	authCode, err := s.generateToken("ac", clientID)
	if err != nil {
		return "", err
	}

	// Store authorization code with metadata for later validation
	err = s.storeAuthorizationCode(authCode, clientID, state, scope, codeChallenge, codeChallengeMethod, subject...)
	if err != nil {
		return "", fmt.Errorf("failed to store authorization code: %w", err)
	}

	return authCode, nil
}

// storeAuthorizationCode stores authorization code with metadata
func (s *Service) storeAuthorizationCode(code, clientID, state, scope, codeChallenge, codeChallengeMethod string, subject ...string) error {
	codeData := map[string]interface{}{
		"client_id":  clientID,
		"state":      state,
		"type":       "authorization_code",
		"issued_at":  time.Now().Unix(),
		"expires_at": time.Now().Add(s.config.Token.AuthorizationCodeLifetime).Unix(),
	}

	// Add scope if provided
	if scope != "" {
		codeData["scope"] = scope
	}

	// Add subject if provided (optional parameter)
	if len(subject) > 0 && subject[0] != "" {
		codeData["subject"] = subject[0]
	}

	// Add PKCE information if provided
	if codeChallenge != "" {
		codeData["code_challenge"] = codeChallenge
		if codeChallengeMethod != "" {
			codeData["code_challenge_method"] = codeChallengeMethod
		} else {
			// Default to S256 if not specified
			codeData["code_challenge_method"] = types.CodeChallengeMethodS256
		}
	}

	return s.store.Set(s.authorizationCodeKey(code), codeData, s.config.Token.AuthorizationCodeLifetime)
}

// getAuthorizationCodeData retrieves and validates authorization code data
func (s *Service) getAuthorizationCodeData(code string) (map[string]interface{}, error) {
	codeData, exists := s.store.Get(s.authorizationCodeKey(code))
	if !exists {
		return nil, &types.ErrorResponse{
			Code:             types.ErrorInvalidGrant,
			ErrorDescription: "Invalid or expired authorization code",
		}
	}

	// Convert to map[string]interface{} if needed
	codeInfo, ok := codeData.(map[string]interface{})
	if !ok {
		// Try primitive.M for MongoDB store compatibility
		if primitiveM, isPrimitiveM := codeData.(primitive.M); isPrimitiveM {
			// Convert primitive.M to map[string]interface{}
			codeInfo = make(map[string]interface{})
			for k, v := range primitiveM {
				codeInfo[k] = v
			}
		} else {
			return nil, &types.ErrorResponse{
				Code:             types.ErrorInvalidGrant,
				ErrorDescription: "Invalid authorization code format",
			}
		}
	}

	return codeInfo, nil
}

// consumeAuthorizationCode retrieves and deletes authorization code (prevents reuse)
func (s *Service) consumeAuthorizationCode(code string) error {
	s.store.Del(s.authorizationCodeKey(code))
	return nil
}

// storeRefreshToken stores refresh token with metadata
func (s *Service) storeRefreshToken(refreshToken, clientID string) error {
	tokenData := map[string]interface{}{
		"client_id": clientID,
		"type":      "refresh_token",
		"issued_at": time.Now().Unix(),
	}

	return s.store.Set(s.refreshTokenKey(refreshToken), tokenData, s.config.Token.RefreshTokenLifetime)
}

// storeRefreshTokenWithScope stores refresh token with metadata including scope and subject
func (s *Service) storeRefreshTokenWithScope(refreshToken, clientID, scope, subject string, expiresIn ...int) error {
	tokenData := map[string]interface{}{
		"client_id": clientID,
		"scope":     scope,
		"subject":   subject,
		"type":      "refresh_token",
		"issued_at": time.Now().Unix(),
	}

	expires := s.config.Token.RefreshTokenLifetime
	if len(expiresIn) > 0 && expiresIn[0] > 0 {
		expires = time.Duration(expiresIn[0]) * time.Second
	}

	return s.store.Set(s.refreshTokenKey(refreshToken), tokenData, expires)
}

// getRefreshTokenData retrieves refresh token data
func (s *Service) getRefreshTokenData(refreshToken string) (map[string]interface{}, error) {
	tokenData, exists := s.store.Get(s.refreshTokenKey(refreshToken))
	if !exists {
		return nil, &types.ErrorResponse{
			Code:             types.ErrorInvalidGrant,
			ErrorDescription: "Invalid refresh token",
		}
	}

	// Convert to map[string]interface{} if needed
	tokenInfo, ok := tokenData.(map[string]interface{})
	if !ok {
		// Try primitive.M for MongoDB store compatibility
		if primitiveM, isPrimitiveM := tokenData.(primitive.M); isPrimitiveM {
			// Convert primitive.M to map[string]interface{}
			tokenInfo = make(map[string]interface{})
			for k, v := range primitiveM {
				tokenInfo[k] = v
			}
		} else {
			return nil, &types.ErrorResponse{
				Code:             types.ErrorInvalidGrant,
				ErrorDescription: "Invalid token format",
			}
		}
	}

	return tokenInfo, nil
}

// revokeRefreshToken deletes refresh token from store
func (s *Service) revokeRefreshToken(refreshToken string) error {
	s.store.Del(s.refreshTokenKey(refreshToken))
	return nil
}

// authorizationCodeKey generates a key for authorization code storage
func (s *Service) authorizationCodeKey(code string) string {
	return fmt.Sprintf("%soauth:auth_code:%s", s.prefix, code)
}

// refreshTokenKey generates a key for refresh token storage
func (s *Service) refreshTokenKey(refreshToken string) string {
	return fmt.Sprintf("%soauth:refresh_token:%s", s.prefix, refreshToken)
}

// accessTokenKey generates a key for access token storage
func (s *Service) accessTokenKey(accessToken string) string {
	return fmt.Sprintf("%soauth:access_token:%s", s.prefix, accessToken)
}

// userFingerprintKey generates a key for user fingerprint storage
func (s *Service) userFingerprintKey(clientID, nanoID string) string {
	return fmt.Sprintf("%soauth:user_fingerprint:%s:%s", s.prefix, clientID, nanoID)
}

// userMappingKey generates a key for reverse user mapping (clientID+userID -> nanoID)
func (s *Service) userMappingKey(clientID, userID string) string {
	return fmt.Sprintf("%soauth:user_mapping:%s:%s", s.prefix, clientID, userID)
}

// generateExchangedToken generates a new token for token exchange
func (s *Service) generateExchangedToken(subjectToken string, audience string) (string, error) {
	// Extract token prefix for tracking purposes
	tokenPrefix := subjectToken
	if len(subjectToken) > 20 {
		tokenPrefix = subjectToken[:20]
	}

	// Generate a more secure token using the same pattern as other tokens
	// For now, we'll use a simple concatenation approach
	// In a real implementation, this would generate a JWT or opaque token
	exchangedToken := "exchanged_" + tokenPrefix + "_" + audience

	return exchangedToken, nil
}

// generateToken generates a token with the specified type and client ID
func (s *Service) generateToken(tokenType string, clientID string) (string, error) {
	// Generate random bytes for token
	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// Create token with type, client ID, timestamp, and random component
	randomPart := base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(randomBytes)
	// Replace underscores with hyphens to avoid conflicts with our delimiter
	randomPart = strings.ReplaceAll(randomPart, "_", "-")
	// Remove any spaces or newlines that might be in the base64 encoding
	randomPart = strings.ReplaceAll(randomPart, " ", "")
	randomPart = strings.ReplaceAll(randomPart, "\n", "")
	randomPart = strings.ReplaceAll(randomPart, "\t", "")
	timestamp := time.Now().Format("20060102150405")

	return fmt.Sprintf("%s_%s_%s_%s", tokenType, clientID, timestamp, randomPart), nil
}

// ============================================================================
// User Fingerprint Methods
// ============================================================================

// generateNumericID generates a deterministic numeric ID using simple hash mapping
func generateNumericID(length int) (string, error) {
	if length <= 0 || length > 16 {
		return "", fmt.Errorf("length must be between 1 and 16")
	}
	// Use only digits 0-9 for numeric ID
	// This provides 10^length possible combinations
	// For 16 digits, that's 10^16 = 10,000,000,000,000,000 possibilities
	const numericAlphabet = "0123456789"
	return gonanoid.Generate(numericAlphabet, length)
}

// DeleteUserFingerprint removes a fingerprint mapping
func (s *Service) DeleteUserFingerprint(clientID, nanoID string) error {
	key := s.userFingerprintKey(clientID, nanoID)
	s.store.Del(key)
	return nil
}
