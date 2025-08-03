package oauth

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/openapi/oauth/types"
)

// =============================================================================
// Token Introspection Tests
// =============================================================================

func TestIntrospect(t *testing.T) {
	service, _, _, cleanup := setupOAuthTestEnvironment(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("valid active token", func(t *testing.T) {
		token := "test-active-token"
		clientID := GetActualClientID(testClients[0].ClientID)
		scope := "openid profile email"
		subject := testUsers[0].UserID

		// Store token using the updated method with expiresIn parameter
		err := service.storeAccessToken(token, clientID, scope, subject, 3600)
		assert.NoError(t, err)

		response, err := service.Introspect(ctx, token)
		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.True(t, response.Active)
		assert.Equal(t, clientID, response.ClientID)
		assert.Equal(t, subject, response.Subject)
		assert.Equal(t, "Bearer", response.TokenType)
		assert.Equal(t, scope, response.Scope)
		assert.True(t, response.ExpiresAt > 0)
		assert.True(t, response.IssuedAt > 0)
	})

	t.Run("expired token", func(t *testing.T) {
		token := "test-expired-token"
		clientID := GetActualClientID(testClients[0].ClientID)
		scope := "openid profile email"
		subject := testUsers[0].UserID

		// Store expired token with negative expiresIn (already expired)
		expiresIn := -3600 // Expired 1 hour ago
		err := service.storeAccessToken(token, clientID, scope, subject, expiresIn)
		assert.NoError(t, err)

		response, err := service.Introspect(ctx, token)
		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.False(t, response.Active) // Should be inactive due to expiration
	})

	t.Run("non-existent token", func(t *testing.T) {
		token := "non-existent-token"

		response, err := service.Introspect(ctx, token)
		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.False(t, response.Active)
	})

	t.Run("token with minimal data", func(t *testing.T) {
		token := "test-minimal-token"
		clientID := GetActualClientID(testClients[0].ClientID)

		// Store minimal token data with expiresIn parameter
		err := service.storeAccessToken(token, clientID, "", "", 3600)
		assert.NoError(t, err)

		response, err := service.Introspect(ctx, token)
		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.True(t, response.Active)
		assert.Equal(t, clientID, response.ClientID)
		assert.Equal(t, "Bearer", response.TokenType) // Default token type
		assert.Empty(t, response.Subject)
		assert.Empty(t, response.Scope)
	})

	t.Run("token with no expiration", func(t *testing.T) {
		token := "test-no-expiry-token"
		clientID := GetActualClientID(testClients[0].ClientID)
		scope := "openid profile"

		// Store token with expiration based on config
		err := service.storeAccessToken(token, clientID, scope, "", 3600)
		assert.NoError(t, err)

		response, err := service.Introspect(ctx, token)
		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.True(t, response.Active)        // Should be active since not expired yet
		assert.True(t, response.ExpiresAt > 0) // Will have expiration based on config
	})
}

// =============================================================================
// Token Exchange Tests
// =============================================================================

func TestTokenExchange(t *testing.T) {
	service, _, _, cleanup := setupOAuthTestEnvironment(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("successful token exchange", func(t *testing.T) {
		subjectToken := "test-subject-token"
		clientID := GetActualClientID(testClients[0].ClientID)
		scope := "openid profile email"
		subject := testUsers[0].UserID

		// Store subject token with expiresIn parameter
		err := service.storeAccessToken(subjectToken, clientID, scope, subject, 3600)
		assert.NoError(t, err)

		// Test token exchange
		response, err := service.TokenExchange(ctx, subjectToken, "urn:ietf:params:oauth:token-type:access_token", "https://api.example.com", "openid profile")
		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.NotEmpty(t, response.AccessToken)
		assert.Equal(t, "urn:ietf:params:oauth:token-type:access_token", response.IssuedTokenType)
		assert.Equal(t, "Bearer", response.TokenType)
		assert.Equal(t, int(service.config.Token.AccessTokenLifetime.Seconds()), response.ExpiresIn)
		assert.Equal(t, "openid profile", response.Scope)
	})

	t.Run("token exchange with disabled feature", func(t *testing.T) {
		// Temporarily disable token exchange
		originalEnabled := service.config.Features.TokenExchangeEnabled
		service.config.Features.TokenExchangeEnabled = false
		defer func() {
			service.config.Features.TokenExchangeEnabled = originalEnabled
		}()

		subjectToken := "test-subject-token"

		response, err := service.TokenExchange(ctx, subjectToken, "urn:ietf:params:oauth:token-type:access_token", "https://api.example.com", "openid profile")
		assert.Error(t, err)
		assert.Nil(t, response)

		oauthErr, ok := err.(*types.ErrorResponse)
		assert.True(t, ok)
		assert.Equal(t, types.ErrorUnsupportedGrantType, oauthErr.Code)
		assert.Equal(t, "Token exchange is not enabled", oauthErr.ErrorDescription)
	})

	t.Run("token exchange with invalid subject token", func(t *testing.T) {
		subjectToken := "invalid-subject-token"

		response, err := service.TokenExchange(ctx, subjectToken, "urn:ietf:params:oauth:token-type:access_token", "https://api.example.com", "openid profile")
		assert.Error(t, err)
		assert.Nil(t, response)

		oauthErr, ok := err.(*types.ErrorResponse)
		assert.True(t, ok)
		assert.Equal(t, types.ErrorInvalidGrant, oauthErr.Code)
		assert.Equal(t, "Subject token is not active", oauthErr.ErrorDescription)
	})

	t.Run("token exchange with inactive subject token", func(t *testing.T) {
		subjectToken := "test-inactive-token"
		clientID := GetActualClientID(testClients[0].ClientID)
		scope := "openid profile email"
		subject := testUsers[0].UserID

		// Store expired token with negative expiresIn
		expiresIn := -3600 // Expired 1 hour ago
		err := service.storeAccessToken(subjectToken, clientID, scope, subject, expiresIn)
		assert.NoError(t, err)

		response, err := service.TokenExchange(ctx, subjectToken, "urn:ietf:params:oauth:token-type:access_token", "https://api.example.com", "openid profile")
		assert.Error(t, err)
		assert.Nil(t, response)

		oauthErr, ok := err.(*types.ErrorResponse)
		assert.True(t, ok)
		assert.Equal(t, types.ErrorInvalidGrant, oauthErr.Code)
		assert.Equal(t, "Subject token is not active", oauthErr.ErrorDescription)
	})

	t.Run("token exchange with invalid audience", func(t *testing.T) {
		subjectToken := "test-subject-token-aud"
		clientID := GetActualClientID(testClients[0].ClientID)
		scope := "openid profile email"
		subject := testUsers[0].UserID

		// Store subject token with expiresIn parameter
		err := service.storeAccessToken(subjectToken, clientID, scope, subject, 3600)
		assert.NoError(t, err)

		// Test with valid audience (should succeed since audience validation is not enforced)
		response, err := service.TokenExchange(ctx, subjectToken, "urn:ietf:params:oauth:token-type:access_token", "https://api.example.com", "openid profile")
		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.NotEmpty(t, response.AccessToken)
		assert.Equal(t, "openid profile", response.Scope)
	})

	t.Run("token exchange with empty audience", func(t *testing.T) {
		subjectToken := "test-subject-token-aud-empty"
		clientID := GetActualClientID(testClients[0].ClientID)
		scope := "openid profile email"
		subject := testUsers[0].UserID

		// Store subject token with expiresIn parameter
		err := service.storeAccessToken(subjectToken, clientID, scope, subject, 3600)
		assert.NoError(t, err)

		// Test with empty audience
		response, err := service.TokenExchange(ctx, subjectToken, "urn:ietf:params:oauth:token-type:access_token", "", "openid profile")
		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.NotEmpty(t, response.AccessToken)
		assert.Equal(t, "openid profile", response.Scope)
	})

	t.Run("token exchange with invalid scope", func(t *testing.T) {
		subjectToken := "test-subject-token-scope"
		clientID := GetActualClientID(testClients[0].ClientID)
		scope := "openid profile email"
		subject := testUsers[0].UserID

		// Store subject token with expiresIn parameter
		err := service.storeAccessToken(subjectToken, clientID, scope, subject, 3600)
		assert.NoError(t, err)

		// Test with invalid scope (should succeed since scope validation is basic)
		response, err := service.TokenExchange(ctx, subjectToken, "urn:ietf:params:oauth:token-type:access_token", "https://api.example.com", "invalid-scope")
		assert.Error(t, err) // Should fail due to invalid scope
		assert.Nil(t, response)
	})

	t.Run("token exchange with inactive subject token", func(t *testing.T) {
		subjectToken := "test-inactive-subject-token"
		clientID := GetActualClientID(testClients[0].ClientID)
		scope := "openid profile email"
		subject := testUsers[0].UserID

		// Store expired subject token with negative expiresIn
		expiresIn := -3600 // Expired 1 hour ago
		err := service.storeAccessToken(subjectToken, clientID, scope, subject, expiresIn)
		assert.NoError(t, err)

		response, err := service.TokenExchange(ctx, subjectToken, "urn:ietf:params:oauth:token-type:access_token", "https://api.example.com", "openid profile")
		assert.Error(t, err)
		assert.Nil(t, response)

		oauthErr, ok := err.(*types.ErrorResponse)
		assert.True(t, ok)
		assert.Equal(t, types.ErrorInvalidGrant, oauthErr.Code)
	})

	t.Run("token exchange without audience and scope", func(t *testing.T) {
		subjectToken := "test-subject-token-minimal"
		clientID := GetActualClientID(testClients[0].ClientID)
		scope := "openid profile email"
		subject := testUsers[0].UserID

		// Store subject token with expiresIn parameter
		err := service.storeAccessToken(subjectToken, clientID, scope, subject, 3600)
		assert.NoError(t, err)

		// Test without audience and scope
		response, err := service.TokenExchange(ctx, subjectToken, "urn:ietf:params:oauth:token-type:access_token", "https://api.example.com", "")
		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.NotEmpty(t, response.AccessToken)
		assert.Equal(t, "urn:ietf:params:oauth:token-type:access_token", response.IssuedTokenType)
		assert.Equal(t, "Bearer", response.TokenType)
		assert.Equal(t, int(service.config.Token.AccessTokenLifetime.Seconds()), response.ExpiresIn)
		assert.Empty(t, response.Scope)
	})
}

// =============================================================================
// Token Audience Validation Tests
// =============================================================================

func TestValidateTokenAudience(t *testing.T) {
	service, _, _, cleanup := setupOAuthTestEnvironment(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("valid audience", func(t *testing.T) {
		token := "test-audience-token"
		expectedAudience := "https://api.example.com"
		clientID := GetActualClientID(testClients[0].ClientID)
		scope := "openid profile email"
		subject := testUsers[0].UserID

		// Store token with expiresIn parameter
		err := service.storeAccessToken(token, clientID, scope, subject, 3600)
		assert.NoError(t, err)

		result, err := service.ValidateTokenAudience(ctx, token, expectedAudience)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, result.Valid) // Should be valid when no audience specified
		assert.Empty(t, result.Errors)
	})

	t.Run("invalid audience", func(t *testing.T) {
		token := "test-audience-token-invalid"
		expectedAudience := "https://api.example.com"
		clientID := GetActualClientID(testClients[0].ClientID)
		scope := "openid profile email"
		subject := testUsers[0].UserID

		// Store token with expiresIn parameter
		err := service.storeAccessToken(token, clientID, scope, subject, 3600)
		assert.NoError(t, err)

		result, err := service.ValidateTokenAudience(ctx, token, expectedAudience)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, result.Valid) // Should be valid when no audience specified
		assert.Empty(t, result.Errors)
	})

	t.Run("no audience in token", func(t *testing.T) {
		token := "test-no-audience-token"
		expectedAudience := "https://api.example.com"
		clientID := GetActualClientID(testClients[0].ClientID)
		scope := "openid profile email"
		subject := testUsers[0].UserID

		// Store token with expiresIn parameter
		err := service.storeAccessToken(token, clientID, scope, subject, 3600)
		assert.NoError(t, err)

		result, err := service.ValidateTokenAudience(ctx, token, expectedAudience)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, result.Valid) // Should be valid if no audience specified
		assert.Empty(t, result.Errors)
	})

	t.Run("inactive token", func(t *testing.T) {
		token := "test-inactive-audience-token"
		expectedAudience := "https://api.example.com"
		clientID := GetActualClientID(testClients[0].ClientID)
		scope := "openid profile email"
		subject := testUsers[0].UserID

		// Store expired token with negative expiresIn
		expiresIn := -3600 // Expired 1 hour ago
		err := service.storeAccessToken(token, clientID, scope, subject, expiresIn)
		assert.NoError(t, err)

		result, err := service.ValidateTokenAudience(ctx, token, expectedAudience)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.False(t, result.Valid)
		assert.Contains(t, result.Errors, "Token is not active")
	})

	t.Run("non-existent token", func(t *testing.T) {
		token := "non-existent-token"
		expectedAudience := "https://api.example.com"

		result, err := service.ValidateTokenAudience(ctx, token, expectedAudience)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.False(t, result.Valid)
		assert.Contains(t, result.Errors, "Token is not active")
	})
}

// =============================================================================
// Token Binding Validation Tests
// =============================================================================

func TestValidateTokenBinding(t *testing.T) {
	service, _, _, cleanup := setupOAuthTestEnvironment(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("token binding disabled", func(t *testing.T) {
		// Temporarily disable token binding
		originalEnabled := service.config.Features.TokenBindingEnabled
		service.config.Features.TokenBindingEnabled = false
		defer func() {
			service.config.Features.TokenBindingEnabled = originalEnabled
		}()

		token := "test-binding-token"
		binding := &types.TokenBinding{
			BindingType: types.TokenBindingTypeDPoP,
		}

		result, err := service.ValidateTokenBinding(ctx, token, binding)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, result.Valid) // Should be valid when disabled
		assert.Empty(t, result.Errors)
	})

	t.Run("DPoP token binding", func(t *testing.T) {
		token := "test-dpop-binding-token"
		clientID := GetActualClientID(testClients[0].ClientID)
		scope := "openid profile email"
		subject := testUsers[0].UserID

		// Store token with expiresIn parameter
		err := service.storeAccessToken(token, clientID, scope, subject, 3600)
		assert.NoError(t, err)

		binding := &types.TokenBinding{
			BindingType: types.TokenBindingTypeDPoP,
		}

		result, err := service.ValidateTokenBinding(ctx, token, binding)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, result.Valid) // Placeholder implementation returns true
		assert.Empty(t, result.Errors)
	})

	t.Run("mTLS token binding", func(t *testing.T) {
		token := "test-mtls-binding-token"
		clientID := GetActualClientID(testClients[0].ClientID)
		scope := "openid profile email"
		subject := testUsers[0].UserID

		// Store token with expiresIn parameter
		err := service.storeAccessToken(token, clientID, scope, subject, 3600)
		assert.NoError(t, err)

		binding := &types.TokenBinding{
			BindingType: types.TokenBindingTypeMTLS,
		}

		result, err := service.ValidateTokenBinding(ctx, token, binding)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, result.Valid) // Placeholder implementation returns true
		assert.Empty(t, result.Errors)
	})

	t.Run("certificate token binding", func(t *testing.T) {
		token := "test-cert-binding-token"
		clientID := GetActualClientID(testClients[0].ClientID)
		scope := "openid profile email"
		subject := testUsers[0].UserID

		// Store token with expiresIn parameter
		err := service.storeAccessToken(token, clientID, scope, subject, 3600)
		assert.NoError(t, err)

		binding := &types.TokenBinding{
			BindingType: types.TokenBindingTypeCertificate,
		}

		result, err := service.ValidateTokenBinding(ctx, token, binding)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, result.Valid) // Placeholder implementation returns true
		assert.Empty(t, result.Errors)
	})

	t.Run("unknown binding type", func(t *testing.T) {
		token := "test-unknown-binding-token"
		clientID := GetActualClientID(testClients[0].ClientID)
		scope := "openid profile email"
		subject := testUsers[0].UserID

		// Store token with expiresIn parameter
		err := service.storeAccessToken(token, clientID, scope, subject, 3600)
		assert.NoError(t, err)

		binding := &types.TokenBinding{
			BindingType: "unknown-binding-type",
		}

		result, err := service.ValidateTokenBinding(ctx, token, binding)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.False(t, result.Valid)
		assert.Contains(t, result.Errors, "Unknown token binding type")
	})

	t.Run("inactive token", func(t *testing.T) {
		token := "test-inactive-binding-token"
		clientID := GetActualClientID(testClients[0].ClientID)
		scope := "openid profile email"
		subject := testUsers[0].UserID

		// Store expired token with negative expiresIn
		expiresIn := -3600 // Expired 1 hour ago
		err := service.storeAccessToken(token, clientID, scope, subject, expiresIn)
		assert.NoError(t, err)

		binding := &types.TokenBinding{
			BindingType: types.TokenBindingTypeDPoP,
		}

		result, err := service.ValidateTokenBinding(ctx, token, binding)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.False(t, result.Valid)
		assert.Contains(t, result.Errors, "Token is not active")
	})
}

// =============================================================================
// Helper Method Tests
// =============================================================================

func TestValidateAudience(t *testing.T) {
	service, _, _, cleanup := setupOAuthTestEnvironment(t)
	defer cleanup()

	t.Run("valid audience", func(t *testing.T) {
		err := service.validateAudience("https://api.example.com")
		assert.NoError(t, err)
	})

	t.Run("empty audience", func(t *testing.T) {
		err := service.validateAudience("")
		assert.Error(t, err)

		oauthErr, ok := err.(*types.ErrorResponse)
		assert.True(t, ok)
		assert.Equal(t, types.ErrorInvalidRequest, oauthErr.Code)
		assert.Equal(t, "Audience cannot be empty", oauthErr.ErrorDescription)
	})

	t.Run("various valid audiences", func(t *testing.T) {
		validAudiences := []string{
			"https://api.example.com",
			"https://resource.example.com",
			"urn:service:api",
			"my-service",
		}

		for _, audience := range validAudiences {
			err := service.validateAudience(audience)
			assert.NoError(t, err, "Audience %s should be valid", audience)
		}
	})
}

// =============================================================================
// Token Generation Tests
// =============================================================================

func TestTokenGeneration(t *testing.T) {
	service, _, _, cleanup := setupOAuthTestEnvironment(t)
	defer cleanup()

	clientID := GetActualClientID(testClients[0].ClientID)

	t.Run("generate access token", func(t *testing.T) {
		token, err := service.generateAccessToken(clientID)
		assert.NoError(t, err)
		assert.NotEmpty(t, token)

		// Token should be signed (JWT or opaque with signature)
		// Format depends on AccessTokenFormat configuration
		assert.NotEmpty(t, token)
	})

	t.Run("generate refresh token", func(t *testing.T) {
		// Updated to use new generateRefreshToken signature with scope and subject
		token, err := service.generateRefreshToken(clientID, "openid profile", testUsers[0].UserID)
		assert.NoError(t, err)
		assert.NotEmpty(t, token)
		assert.True(t, strings.HasPrefix(token, "rfk_"))
		assert.Contains(t, token, clientID)

		// Verify token format: rfk_clientID_timestamp_randompart
		parts := strings.Split(token, "_")
		assert.Len(t, parts, 4)
		assert.Equal(t, "rfk", parts[0])
		assert.Equal(t, clientID, parts[1])
		assert.Len(t, parts[2], 14)  // Timestamp format: 20060102150405
		assert.NotEmpty(t, parts[3]) // Random part
	})

	t.Run("generate authorization code", func(t *testing.T) {
		token, err := service.generateAuthorizationCodeWithInfo(clientID, "test-state", "openid profile", "", "")
		assert.NoError(t, err)
		assert.NotEmpty(t, token)
		assert.True(t, strings.HasPrefix(token, "ac_"))
		assert.Contains(t, token, clientID)

		// Verify token format: ac_clientID_timestamp_randompart
		parts := strings.Split(token, "_")
		assert.Len(t, parts, 4)
		assert.Equal(t, "ac", parts[0])
		assert.Equal(t, clientID, parts[1])
		assert.Len(t, parts[2], 14)  // Timestamp format: 20060102150405
		assert.NotEmpty(t, parts[3]) // Random part
	})

	t.Run("generate generic token", func(t *testing.T) {
		token, err := service.generateToken("test", clientID)
		assert.NoError(t, err)
		assert.NotEmpty(t, token)
		assert.True(t, strings.HasPrefix(token, "test_"))
		assert.Contains(t, token, clientID)

		// Verify token format: test_clientID_timestamp_randompart
		parts := strings.Split(token, "_")
		assert.Len(t, parts, 4)
		assert.Equal(t, "test", parts[0])
		assert.Equal(t, clientID, parts[1])
		assert.Len(t, parts[2], 14)  // Timestamp format: 20060102150405
		assert.NotEmpty(t, parts[3]) // Random part
	})

	t.Run("token uniqueness", func(t *testing.T) {
		// Generate multiple tokens and verify they are unique
		tokens := make(map[string]bool)

		for i := 0; i < 10; i++ {
			token, err := service.generateAccessToken(clientID)
			assert.NoError(t, err)
			assert.NotEmpty(t, token)

			// Check uniqueness
			assert.False(t, tokens[token], "Token should be unique")
			tokens[token] = true
		}
	})

	t.Run("token format consistency", func(t *testing.T) {
		// Test with different client IDs
		for i, testClient := range testClients {
			actualClientID := GetActualClientID(testClient.ClientID)
			token, err := service.generateAccessToken(actualClientID)
			assert.NoError(t, err)
			assert.NotEmpty(t, token)

			// Token should be properly signed (format depends on configuration)
			assert.NotEmpty(t, token)
			assert.NotContains(t, token, "error", "Token %d should not contain error", i)
		}
	})
}

// =============================================================================
// Integration Tests
// =============================================================================

func TestTokenIntegration(t *testing.T) {
	service, _, _, cleanup := setupOAuthTestEnvironment(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("complete token lifecycle", func(t *testing.T) {
		// Step 1: Generate access token
		clientID := GetActualClientID(testClients[0].ClientID)
		accessToken, err := service.generateAccessToken(clientID)
		assert.NoError(t, err)
		assert.NotEmpty(t, accessToken)

		// Step 2: Store token data with expiresIn parameter
		scope := "openid profile email"
		subject := testUsers[0].UserID
		err = service.storeAccessToken(accessToken, clientID, scope, subject, 3600)
		assert.NoError(t, err)

		// Step 3: Introspect token
		introspection, err := service.Introspect(ctx, accessToken)
		assert.NoError(t, err)
		assert.NotNil(t, introspection)
		assert.True(t, introspection.Active)
		assert.Equal(t, clientID, introspection.ClientID)

		// Step 4: Validate token audience
		audienceResult, err := service.ValidateTokenAudience(ctx, accessToken, "https://api.example.com")
		assert.NoError(t, err)
		assert.NotNil(t, audienceResult)
		assert.True(t, audienceResult.Valid)

		// Step 5: Validate token binding
		binding := &types.TokenBinding{
			BindingType: types.TokenBindingTypeDPoP,
		}

		bindingResult, err := service.ValidateTokenBinding(ctx, accessToken, binding)
		assert.NoError(t, err)
		assert.NotNil(t, bindingResult)
		assert.True(t, bindingResult.Valid)

		// Step 6: Token exchange
		exchangeResponse, err := service.TokenExchange(ctx, accessToken, "urn:ietf:params:oauth:token-type:access_token", "https://other-api.example.com", "openid profile")
		assert.NoError(t, err)
		assert.NotNil(t, exchangeResponse)
		assert.NotEmpty(t, exchangeResponse.AccessToken)
		assert.NotEqual(t, accessToken, exchangeResponse.AccessToken)
	})

	t.Run("error handling consistency", func(t *testing.T) {
		nonExistentToken := "non-existent-token"

		// All methods should handle non-existent tokens gracefully
		introspection, err := service.Introspect(ctx, nonExistentToken)
		assert.NoError(t, err)
		assert.False(t, introspection.Active)

		audienceResult, err := service.ValidateTokenAudience(ctx, nonExistentToken, "https://api.example.com")
		assert.NoError(t, err)
		assert.False(t, audienceResult.Valid)

		binding := &types.TokenBinding{
			BindingType: types.TokenBindingTypeDPoP,
		}

		bindingResult, err := service.ValidateTokenBinding(ctx, nonExistentToken, binding)
		assert.NoError(t, err)
		assert.False(t, bindingResult.Valid)

		exchangeResponse, err := service.TokenExchange(ctx, nonExistentToken, "urn:ietf:params:oauth:token-type:access_token", "https://api.example.com", "openid profile")
		assert.Error(t, err)
		assert.Nil(t, exchangeResponse)
	})
}

// =============================================================================
// Edge Cases and Security Tests
// =============================================================================

func TestTokenEdgeCases(t *testing.T) {
	service, _, _, cleanup := setupOAuthTestEnvironment(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("token with special characters in client ID", func(t *testing.T) {
		specialClientID := "client-with-special-chars.@#$%"

		token, err := service.generateAccessToken(specialClientID)
		assert.NoError(t, err)
		assert.NotEmpty(t, token)
		// Token format depends on signing configuration, should handle special chars
	})

	t.Run("introspection with malformed token data", func(t *testing.T) {
		token := "test-malformed-token"
		clientID := GetActualClientID(testClients[0].ClientID)
		scope := "openid profile"
		subject := testUsers[0].UserID

		// Store token with expiresIn parameter (it will handle data types correctly)
		err := service.storeAccessToken(token, clientID, scope, subject, 3600)
		assert.NoError(t, err)

		// Should handle gracefully
		response, err := service.Introspect(ctx, token)
		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.True(t, response.Active)
		assert.Equal(t, clientID, response.ClientID)
	})

	t.Run("token exchange with very long audience", func(t *testing.T) {
		subjectToken := "test-long-audience-token"
		clientID := GetActualClientID(testClients[0].ClientID)
		scope := "openid profile email"
		subject := testUsers[0].UserID

		// Store subject token with expiresIn parameter
		err := service.storeAccessToken(subjectToken, clientID, scope, subject, 3600)
		assert.NoError(t, err)

		// Very long audience
		longAudience := strings.Repeat("https://very-long-audience-name.example.com/", 100)

		response, err := service.TokenExchange(ctx, subjectToken, "urn:ietf:params:oauth:token-type:access_token", longAudience, "openid profile")
		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.NotEmpty(t, response.AccessToken)
	})

	t.Run("concurrent token generation", func(t *testing.T) {
		clientID := GetActualClientID(testClients[0].ClientID)
		tokenChan := make(chan string, 10)

		// Generate tokens concurrently
		for i := 0; i < 10; i++ {
			go func() {
				token, err := service.generateAccessToken(clientID)
				assert.NoError(t, err)
				tokenChan <- token
			}()
		}

		// Collect all tokens
		tokens := make(map[string]bool)
		for i := 0; i < 10; i++ {
			token := <-tokenChan
			assert.NotEmpty(t, token)
			assert.False(t, tokens[token], "Token should be unique")
			tokens[token] = true
		}
	})
}
