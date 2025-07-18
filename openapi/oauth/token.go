package oauth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/yaoapp/yao/openapi/oauth/types"
)

// Introspect returns information about an access token
// This endpoint allows resource servers to validate tokens
func (s *Service) Introspect(ctx context.Context, token string) (*types.TokenIntrospectionResponse, error) {
	// Try to get token data from user provider
	tokenData, err := s.userProvider.GetTokenData(token)
	if err != nil {
		return &types.TokenIntrospectionResponse{Active: false}, nil
	}

	// Check if token exists and is valid
	if tokenData == nil {
		return &types.TokenIntrospectionResponse{Active: false}, nil
	}

	// Extract token information
	response := &types.TokenIntrospectionResponse{
		Active: true,
	}

	// Extract standard fields from token data
	if clientID, ok := tokenData["client_id"].(string); ok {
		response.ClientID = clientID
	}
	if username, ok := tokenData["username"].(string); ok {
		response.Username = username
	}
	if subject, ok := tokenData["sub"].(string); ok {
		response.Subject = subject
	}
	if tokenType, ok := tokenData["token_type"].(string); ok {
		response.TokenType = tokenType
	} else {
		response.TokenType = "Bearer"
	}
	if scope, ok := tokenData["scope"].(string); ok {
		response.Scope = scope
	}
	if exp, ok := tokenData["exp"].(int64); ok {
		response.ExpiresAt = exp
	}
	if iat, ok := tokenData["iat"].(int64); ok {
		response.IssuedAt = iat
	}
	if nbf, ok := tokenData["nbf"].(int64); ok {
		response.NotBefore = nbf
	}
	if aud, ok := tokenData["aud"].([]string); ok {
		response.Audience = aud
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

	// Generate new token (placeholder implementation)
	// In a real implementation, this would generate a JWT or opaque token
	newToken := "exchanged_" + subjectToken[:20] + "_" + audience

	response := &types.TokenExchangeResponse{
		AccessToken:     newToken,
		IssuedTokenType: "urn:ietf:params:oauth:token-type:access_token",
		TokenType:       "Bearer",
		ExpiresIn:       3600, // 1 hour
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

// Helper methods

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
	return s.generateToken("ak", clientID)
}

// generateRefreshToken generates a new refresh token
func (s *Service) generateRefreshToken(clientID string) (string, error) {
	return s.generateToken("rfk", clientID)
}

// generateAuthorizationCode generates a new authorization code
func (s *Service) generateAuthorizationCode(clientID string, state string) (string, error) {
	return s.generateToken("ac", clientID)
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
	timestamp := time.Now().Format("20060102150405")

	return fmt.Sprintf("%s_%s_%s_%s", tokenType, clientID, timestamp, randomPart), nil
}
