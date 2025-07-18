package oauth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/openapi/oauth/types"
)

// generateUniqueClientID generates a unique client ID for testing
func generateUniqueClientID(prefix string) string {
	bytes := make([]byte, 4)
	rand.Read(bytes)
	return prefix + "-" + hex.EncodeToString(bytes)
}

// =============================================================================
// Client Management Tests
// =============================================================================

func TestRegister(t *testing.T) {
	service, _, _, cleanup := setupOAuthTestEnvironment(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("register new client successfully", func(t *testing.T) {
		clientInfo := &types.ClientInfo{
			ClientID:      generateUniqueClientID("test-register-new-client"),
			ClientSecret:  "test-secret",
			ClientName:    "Test Registration Client",
			ClientType:    types.ClientTypeConfidential,
			RedirectURIs:  []string{"https://localhost/callback"},
			GrantTypes:    []string{types.GrantTypeAuthorizationCode},
			ResponseTypes: []string{types.ResponseTypeCode},
			Scope:         "openid profile",
		}

		result, err := service.Register(ctx, clientInfo)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, clientInfo.ClientID, result.ClientID)
		assert.Equal(t, clientInfo.ClientName, result.ClientName)
		assert.Equal(t, clientInfo.ClientType, result.ClientType)
		assert.Equal(t, clientInfo.RedirectURIs, result.RedirectURIs)
		assert.Equal(t, clientInfo.GrantTypes, result.GrantTypes)
		assert.Equal(t, clientInfo.ResponseTypes, result.ResponseTypes)
		assert.Equal(t, clientInfo.Scope, result.Scope)
	})

	t.Run("register with nil client info", func(t *testing.T) {
		result, err := service.Register(ctx, nil)
		assert.Error(t, err)
		assert.Nil(t, result)

		// Check that the error is related to nil client info
		assert.Contains(t, err.Error(), "Client information is required")
	})

	t.Run("register with empty client info", func(t *testing.T) {
		clientInfo := &types.ClientInfo{}

		result, err := service.Register(ctx, clientInfo)
		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("register client with existing ID", func(t *testing.T) {
		// Use one of the pre-existing test clients
		clientInfo := &types.ClientInfo{
			ClientID:      testClients[0].ClientID, // This client already exists
			ClientSecret:  "new-secret",
			ClientName:    "Duplicate Client",
			ClientType:    types.ClientTypeConfidential,
			RedirectURIs:  []string{"https://localhost/callback"},
			GrantTypes:    []string{types.GrantTypeAuthorizationCode},
			ResponseTypes: []string{types.ResponseTypeCode},
			Scope:         "openid profile",
		}

		result, err := service.Register(ctx, clientInfo)
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestUpdateClient(t *testing.T) {
	service, _, _, cleanup := setupOAuthTestEnvironment(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("update existing client successfully", func(t *testing.T) {
		// Use first test client
		clientID := testClients[0].ClientID
		updatedInfo := &types.ClientInfo{
			ClientID:      clientID,
			ClientSecret:  "updated-secret",
			ClientName:    "Updated Client Name",
			ClientType:    types.ClientTypeConfidential,
			RedirectURIs:  []string{"https://localhost/callback"},
			GrantTypes:    []string{types.GrantTypeAuthorizationCode, types.GrantTypeRefreshToken},
			ResponseTypes: []string{types.ResponseTypeCode},
			Scope:         "openid profile email",
		}

		result, err := service.UpdateClient(ctx, clientID, updatedInfo)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, clientID, result.ClientID)
		assert.Equal(t, updatedInfo.ClientName, result.ClientName)
		assert.Equal(t, updatedInfo.RedirectURIs, result.RedirectURIs)
		assert.Equal(t, updatedInfo.Scope, result.Scope)
	})

	t.Run("update non-existing client", func(t *testing.T) {
		clientID := "non-existing-client"
		updatedInfo := &types.ClientInfo{
			ClientID:   clientID,
			ClientName: "Non-existing Client",
		}

		result, err := service.UpdateClient(ctx, clientID, updatedInfo)
		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("update with nil client info", func(t *testing.T) {
		clientID := testClients[0].ClientID

		result, err := service.UpdateClient(ctx, clientID, nil)
		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("update with empty client ID", func(t *testing.T) {
		updatedInfo := &types.ClientInfo{
			ClientName: "Updated Client",
		}

		result, err := service.UpdateClient(ctx, "", updatedInfo)
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestDeleteClient(t *testing.T) {
	service, _, _, cleanup := setupOAuthTestEnvironment(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("delete existing client successfully", func(t *testing.T) {
		// Create a client to delete
		clientInfo := &types.ClientInfo{
			ClientID:      generateUniqueClientID("test-delete-client"),
			ClientSecret:  "test-secret",
			ClientName:    "Test Delete Client",
			ClientType:    types.ClientTypeConfidential,
			RedirectURIs:  []string{"https://localhost/callback"},
			GrantTypes:    []string{types.GrantTypeAuthorizationCode},
			ResponseTypes: []string{types.ResponseTypeCode},
			Scope:         "openid profile",
		}

		_, err := service.Register(ctx, clientInfo)
		require.NoError(t, err)

		// Delete the client
		err = service.DeleteClient(ctx, clientInfo.ClientID)
		assert.NoError(t, err)

		// Verify client is deleted
		clientProvider := service.GetClientProvider()
		deletedClient, err := clientProvider.GetClientByID(ctx, clientInfo.ClientID)
		assert.Error(t, err)
		assert.Nil(t, deletedClient)
	})

	t.Run("delete non-existing client", func(t *testing.T) {
		err := service.DeleteClient(ctx, "non-existing-client")
		assert.Error(t, err)
	})

	t.Run("delete with empty client ID", func(t *testing.T) {
		err := service.DeleteClient(ctx, "")
		assert.Error(t, err)
	})
}

func TestValidateScope(t *testing.T) {
	service, _, _, cleanup := setupOAuthTestEnvironment(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("validate valid scopes", func(t *testing.T) {
		clientID := testClients[0].ClientID
		requestedScopes := []string{"openid", "profile"}

		result, err := service.ValidateScope(ctx, requestedScopes, clientID)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, result.Valid)
	})

	t.Run("validate with non-existing client", func(t *testing.T) {
		clientID := "non-existing-client"
		requestedScopes := []string{"openid", "profile"}

		result, err := service.ValidateScope(ctx, requestedScopes, clientID)
		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("validate with empty scopes", func(t *testing.T) {
		clientID := testClients[0].ClientID
		requestedScopes := []string{}

		result, err := service.ValidateScope(ctx, requestedScopes, clientID)
		assert.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("validate with empty client ID", func(t *testing.T) {
		requestedScopes := []string{"openid", "profile"}

		result, err := service.ValidateScope(ctx, requestedScopes, "")
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

// =============================================================================
// Dynamic Client Registration Tests
// =============================================================================

func TestDynamicClientRegistration(t *testing.T) {
	service, _, _, cleanup := setupOAuthTestEnvironment(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("register confidential client successfully", func(t *testing.T) {
		request := &types.DynamicClientRegistrationRequest{
			ClientName:              "Dynamic Test Client",
			RedirectURIs:            []string{"https://localhost/callback"},
			GrantTypes:              []string{types.GrantTypeAuthorizationCode, types.GrantTypeRefreshToken},
			ResponseTypes:           []string{types.ResponseTypeCode},
			ApplicationType:         types.ApplicationTypeWeb,
			TokenEndpointAuthMethod: types.TokenEndpointAuthBasic,
			Scope:                   "openid profile email",
			ClientURI:               "https://localhost",
			LogoURI:                 "https://localhost/logo.png",
			TosURI:                  "https://localhost/tos",
			PolicyURI:               "https://localhost/policy",
			Contacts:                []string{"admin@localhost"},
		}

		response, err := service.DynamicClientRegistration(ctx, request)
		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.NotEmpty(t, response.ClientID)
		assert.NotEmpty(t, response.ClientSecret)
		assert.Greater(t, response.ClientIDIssuedAt, int64(0))
		assert.Equal(t, request.ClientName, response.DynamicClientRegistrationRequest.ClientName)
		assert.Equal(t, request.RedirectURIs, response.DynamicClientRegistrationRequest.RedirectURIs)
		assert.Equal(t, request.GrantTypes, response.DynamicClientRegistrationRequest.GrantTypes)
		assert.Equal(t, request.ResponseTypes, response.DynamicClientRegistrationRequest.ResponseTypes)
		assert.Equal(t, request.ApplicationType, response.DynamicClientRegistrationRequest.ApplicationType)
		assert.Equal(t, request.TokenEndpointAuthMethod, response.DynamicClientRegistrationRequest.TokenEndpointAuthMethod)
		assert.Equal(t, request.Scope, response.DynamicClientRegistrationRequest.Scope)
		assert.Equal(t, request.ClientURI, response.DynamicClientRegistrationRequest.ClientURI)
		assert.Equal(t, request.LogoURI, response.DynamicClientRegistrationRequest.LogoURI)
		assert.Equal(t, request.TosURI, response.DynamicClientRegistrationRequest.TosURI)
		assert.Equal(t, request.PolicyURI, response.DynamicClientRegistrationRequest.PolicyURI)
		assert.Equal(t, request.Contacts, response.DynamicClientRegistrationRequest.Contacts)
	})

	t.Run("register public client successfully", func(t *testing.T) {
		request := &types.DynamicClientRegistrationRequest{
			ClientName:              "Dynamic Public Client",
			RedirectURIs:            []string{"https://localhost/callback"},
			GrantTypes:              []string{types.GrantTypeAuthorizationCode},
			ResponseTypes:           []string{types.ResponseTypeCode},
			ApplicationType:         types.ApplicationTypeNative,
			TokenEndpointAuthMethod: "none",
			Scope:                   "openid profile",
		}

		response, err := service.DynamicClientRegistration(ctx, request)
		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.NotEmpty(t, response.ClientID)
		assert.Empty(t, response.ClientSecret) // Public clients don't have secrets
		assert.Greater(t, response.ClientIDIssuedAt, int64(0))
		assert.Equal(t, request.ClientName, response.DynamicClientRegistrationRequest.ClientName)
		assert.Equal(t, request.RedirectURIs, response.DynamicClientRegistrationRequest.RedirectURIs)
		assert.Equal(t, request.GrantTypes, response.DynamicClientRegistrationRequest.GrantTypes)
		assert.Equal(t, request.ResponseTypes, response.DynamicClientRegistrationRequest.ResponseTypes)
		assert.Equal(t, request.ApplicationType, response.DynamicClientRegistrationRequest.ApplicationType)
		assert.Equal(t, request.TokenEndpointAuthMethod, response.DynamicClientRegistrationRequest.TokenEndpointAuthMethod)
		assert.Equal(t, request.Scope, response.DynamicClientRegistrationRequest.Scope)
	})

	t.Run("register with default values", func(t *testing.T) {
		request := &types.DynamicClientRegistrationRequest{
			ClientName:   "Minimal Client",
			RedirectURIs: []string{"https://localhost/callback"},
		}

		response, err := service.DynamicClientRegistration(ctx, request)
		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.NotEmpty(t, response.ClientID)
		assert.NotEmpty(t, response.ClientSecret)
		assert.Equal(t, request.ClientName, response.DynamicClientRegistrationRequest.ClientName)
		assert.Equal(t, request.RedirectURIs, response.DynamicClientRegistrationRequest.RedirectURIs)
		// Check defaults were applied
		assert.Equal(t, service.config.Client.DefaultGrantTypes, response.DynamicClientRegistrationRequest.GrantTypes)
		assert.Equal(t, service.config.Client.DefaultResponseTypes, response.DynamicClientRegistrationRequest.ResponseTypes)
		assert.Equal(t, types.ApplicationTypeWeb, response.DynamicClientRegistrationRequest.ApplicationType)
		assert.Equal(t, service.config.Client.DefaultTokenEndpointAuthMethod, response.DynamicClientRegistrationRequest.TokenEndpointAuthMethod)
	})

	t.Run("register with dynamic registration disabled", func(t *testing.T) {
		// Temporarily disable dynamic registration
		originalEnabled := service.config.Features.DynamicClientRegistrationEnabled
		service.config.Features.DynamicClientRegistrationEnabled = false
		defer func() {
			service.config.Features.DynamicClientRegistrationEnabled = originalEnabled
		}()

		request := &types.DynamicClientRegistrationRequest{
			ClientName:   "Disabled Feature Client",
			RedirectURIs: []string{"https://localhost/callback"},
		}

		response, err := service.DynamicClientRegistration(ctx, request)
		assert.Error(t, err)
		assert.Nil(t, response)

		oauthErr, ok := err.(*types.ErrorResponse)
		assert.True(t, ok)
		assert.Equal(t, types.ErrorInvalidRequest, oauthErr.Code)
		assert.Contains(t, oauthErr.ErrorDescription, "Dynamic client registration is not enabled")
	})

	t.Run("register with invalid request", func(t *testing.T) {
		request := &types.DynamicClientRegistrationRequest{
			ClientName:   "Invalid Client",
			RedirectURIs: []string{}, // Empty redirect URIs should cause error
		}

		response, err := service.DynamicClientRegistration(ctx, request)
		assert.Error(t, err)
		assert.Nil(t, response)

		oauthErr, ok := err.(*types.ErrorResponse)
		assert.True(t, ok)
		assert.Equal(t, types.ErrorInvalidRequest, oauthErr.Code)
		assert.Contains(t, oauthErr.ErrorDescription, "At least one redirect URI is required")
	})

	t.Run("register with disallowed redirect URI host", func(t *testing.T) {
		request := &types.DynamicClientRegistrationRequest{
			ClientName:   "Disallowed Host Client",
			RedirectURIs: []string{"https://example.com/callback"}, // example.com is not in allowed hosts
		}

		response, err := service.DynamicClientRegistration(ctx, request)
		assert.Error(t, err)
		assert.Nil(t, response)

		oauthErr, ok := err.(*types.ErrorResponse)
		assert.True(t, ok)
		assert.Equal(t, types.ErrorInvalidRequest, oauthErr.Code)
		assert.Contains(t, oauthErr.ErrorDescription, "Redirect URI host 'example.com' is not allowed")
	})

	t.Run("register with disallowed redirect URI scheme", func(t *testing.T) {
		// Temporarily restrict schemes to only HTTPS
		originalSchemes := service.config.Client.AllowedRedirectURISchemes
		service.config.Client.AllowedRedirectURISchemes = []string{"https"}
		defer func() {
			service.config.Client.AllowedRedirectURISchemes = originalSchemes
		}()

		request := &types.DynamicClientRegistrationRequest{
			ClientName:   "Disallowed Scheme Client",
			RedirectURIs: []string{"http://localhost/callback"}, // HTTP is not allowed when only HTTPS is permitted
		}

		response, err := service.DynamicClientRegistration(ctx, request)
		assert.Error(t, err)
		assert.Nil(t, response)

		oauthErr, ok := err.(*types.ErrorResponse)
		assert.True(t, ok)
		assert.Equal(t, types.ErrorInvalidRequest, oauthErr.Code)
		assert.Contains(t, oauthErr.ErrorDescription, "Redirect URI scheme 'http' is not allowed")
	})
}

// =============================================================================
// Client ID and Secret Generation Tests
// =============================================================================

func TestGenerateClientID(t *testing.T) {
	service, _, _, cleanup := setupOAuthTestEnvironment(t)
	defer cleanup()

	t.Run("generate client ID with default length", func(t *testing.T) {
		clientID, err := service.generateClientID()
		assert.NoError(t, err)
		assert.NotEmpty(t, clientID)
		assert.Greater(t, len(clientID), 0)

		// Check it's valid base64url
		assert.NotContains(t, clientID, "=")
		assert.NotContains(t, clientID, "+")
		assert.NotContains(t, clientID, "/")
	})

	t.Run("generate multiple client IDs are unique", func(t *testing.T) {
		clientIDs := make(map[string]bool)

		for i := 0; i < 100; i++ {
			clientID, err := service.generateClientID()
			assert.NoError(t, err)
			assert.NotEmpty(t, clientID)

			// Check uniqueness
			assert.False(t, clientIDs[clientID], "Client ID should be unique")
			clientIDs[clientID] = true
		}
	})

	t.Run("generate client ID with custom length", func(t *testing.T) {
		// Set custom length
		originalLength := service.config.Client.ClientIDLength
		service.config.Client.ClientIDLength = 16
		defer func() {
			service.config.Client.ClientIDLength = originalLength
		}()

		clientID, err := service.generateClientID()
		assert.NoError(t, err)
		assert.NotEmpty(t, clientID)

		// Length should be approximately 16 * 4/3 (base64 encoding)
		expectedMinLength := 16
		assert.GreaterOrEqual(t, len(clientID), expectedMinLength)
	})
}

func TestGenerateClientSecret(t *testing.T) {
	service, _, _, cleanup := setupOAuthTestEnvironment(t)
	defer cleanup()

	t.Run("generate client secret with default length", func(t *testing.T) {
		clientSecret, err := service.generateClientSecret()
		assert.NoError(t, err)
		assert.NotEmpty(t, clientSecret)
		assert.Greater(t, len(clientSecret), 0)

		// Check it's valid base64url
		assert.NotContains(t, clientSecret, "=")
		assert.NotContains(t, clientSecret, "+")
		assert.NotContains(t, clientSecret, "/")
	})

	t.Run("generate multiple client secrets are unique", func(t *testing.T) {
		clientSecrets := make(map[string]bool)

		for i := 0; i < 100; i++ {
			clientSecret, err := service.generateClientSecret()
			assert.NoError(t, err)
			assert.NotEmpty(t, clientSecret)

			// Check uniqueness
			assert.False(t, clientSecrets[clientSecret], "Client secret should be unique")
			clientSecrets[clientSecret] = true
		}
	})

	t.Run("generate client secret with custom length", func(t *testing.T) {
		// Set custom length
		originalLength := service.config.Client.ClientSecretLength
		service.config.Client.ClientSecretLength = 32
		defer func() {
			service.config.Client.ClientSecretLength = originalLength
		}()

		clientSecret, err := service.generateClientSecret()
		assert.NoError(t, err)
		assert.NotEmpty(t, clientSecret)

		// Length should be approximately 32 * 4/3 (base64 encoding)
		expectedMinLength := 32
		assert.GreaterOrEqual(t, len(clientSecret), expectedMinLength)
	})
}

// =============================================================================
// Request Validation Tests
// =============================================================================

func TestValidateDynamicClientRegistrationRequest(t *testing.T) {
	service, _, _, cleanup := setupOAuthTestEnvironment(t)
	defer cleanup()

	t.Run("validate valid request", func(t *testing.T) {
		request := &types.DynamicClientRegistrationRequest{
			ClientName:      "Valid Client",
			RedirectURIs:    []string{"https://localhost/callback"},
			GrantTypes:      []string{types.GrantTypeAuthorizationCode},
			ResponseTypes:   []string{types.ResponseTypeCode},
			ApplicationType: types.ApplicationTypeWeb,
		}

		err := service.validateDynamicClientRegistrationRequest(request)
		assert.NoError(t, err)
	})

	t.Run("validate request with no redirect URIs", func(t *testing.T) {
		request := &types.DynamicClientRegistrationRequest{
			ClientName:   "No Redirect URIs",
			RedirectURIs: []string{},
		}

		err := service.validateDynamicClientRegistrationRequest(request)
		assert.Error(t, err)

		oauthErr, ok := err.(*types.ErrorResponse)
		assert.True(t, ok)
		assert.Equal(t, types.ErrorInvalidRequest, oauthErr.Code)
		assert.Contains(t, oauthErr.ErrorDescription, "At least one redirect URI is required")
	})

	t.Run("validate request with invalid redirect URI", func(t *testing.T) {
		request := &types.DynamicClientRegistrationRequest{
			ClientName:   "Invalid Redirect URI",
			RedirectURIs: []string{"://invalid-uri"},
		}

		err := service.validateDynamicClientRegistrationRequest(request)
		assert.Error(t, err)

		oauthErr, ok := err.(*types.ErrorResponse)
		assert.True(t, ok)
		assert.Equal(t, types.ErrorInvalidRequest, oauthErr.Code)
		assert.Contains(t, oauthErr.ErrorDescription, "Invalid redirect URI format")
	})

	t.Run("validate request with invalid grant type", func(t *testing.T) {
		request := &types.DynamicClientRegistrationRequest{
			ClientName:   "Invalid Grant Type",
			RedirectURIs: []string{"https://localhost/callback"},
			GrantTypes:   []string{"invalid_grant_type"},
		}

		err := service.validateDynamicClientRegistrationRequest(request)
		assert.Error(t, err)

		oauthErr, ok := err.(*types.ErrorResponse)
		assert.True(t, ok)
		assert.Equal(t, types.ErrorInvalidRequest, oauthErr.Code)
		assert.Contains(t, oauthErr.ErrorDescription, "Invalid grant type: invalid_grant_type")
	})

	t.Run("validate request with invalid response type", func(t *testing.T) {
		request := &types.DynamicClientRegistrationRequest{
			ClientName:    "Invalid Response Type",
			RedirectURIs:  []string{"https://localhost/callback"},
			ResponseTypes: []string{"invalid_response_type"},
		}

		err := service.validateDynamicClientRegistrationRequest(request)
		assert.Error(t, err)

		oauthErr, ok := err.(*types.ErrorResponse)
		assert.True(t, ok)
		assert.Equal(t, types.ErrorInvalidRequest, oauthErr.Code)
		assert.Contains(t, oauthErr.ErrorDescription, "Invalid response type: invalid_response_type")
	})

	t.Run("validate request with invalid application type", func(t *testing.T) {
		request := &types.DynamicClientRegistrationRequest{
			ClientName:      "Invalid Application Type",
			RedirectURIs:    []string{"https://localhost/callback"},
			ApplicationType: "invalid_application_type",
		}

		err := service.validateDynamicClientRegistrationRequest(request)
		assert.Error(t, err)

		oauthErr, ok := err.(*types.ErrorResponse)
		assert.True(t, ok)
		assert.Equal(t, types.ErrorInvalidRequest, oauthErr.Code)
		assert.Contains(t, oauthErr.ErrorDescription, "Invalid application type")
	})
}

func TestValidateRedirectURIForRegistration(t *testing.T) {
	service, _, _, cleanup := setupOAuthTestEnvironment(t)
	defer cleanup()

	t.Run("validate valid HTTPS URI", func(t *testing.T) {
		err := service.validateRedirectURIForRegistration("https://localhost/callback")
		assert.NoError(t, err)
	})

	t.Run("validate valid HTTP URI", func(t *testing.T) {
		err := service.validateRedirectURIForRegistration("http://localhost/callback")
		assert.NoError(t, err)
	})

	t.Run("validate invalid URI format", func(t *testing.T) {
		err := service.validateRedirectURIForRegistration("://invalid-uri")
		assert.Error(t, err)

		oauthErr, ok := err.(*types.ErrorResponse)
		assert.True(t, ok)
		assert.Equal(t, types.ErrorInvalidRequest, oauthErr.Code)
		assert.Contains(t, oauthErr.ErrorDescription, "Invalid redirect URI format")
	})

	t.Run("validate with scheme restrictions", func(t *testing.T) {
		// Set restricted schemes
		originalSchemes := service.config.Client.AllowedRedirectURISchemes
		service.config.Client.AllowedRedirectURISchemes = []string{"https"}
		defer func() {
			service.config.Client.AllowedRedirectURISchemes = originalSchemes
		}()

		// HTTPS should be allowed
		err := service.validateRedirectURIForRegistration("https://localhost/callback")
		assert.NoError(t, err)

		// HTTP should be rejected
		err = service.validateRedirectURIForRegistration("http://example.com/callback")
		assert.Error(t, err)

		oauthErr, ok := err.(*types.ErrorResponse)
		assert.True(t, ok)
		assert.Equal(t, types.ErrorInvalidRequest, oauthErr.Code)
		assert.Contains(t, oauthErr.ErrorDescription, "Redirect URI scheme 'http' is not allowed")
	})

	t.Run("validate with host restrictions", func(t *testing.T) {
		// Set restricted hosts
		originalHosts := service.config.Client.AllowedRedirectURIHosts
		service.config.Client.AllowedRedirectURIHosts = []string{"localhost", "127.0.0.1"}
		defer func() {
			service.config.Client.AllowedRedirectURIHosts = originalHosts
		}()

		// Localhost should be allowed
		err := service.validateRedirectURIForRegistration("https://localhost/callback")
		assert.NoError(t, err)

		// 127.0.0.1 should be allowed
		err = service.validateRedirectURIForRegistration("https://127.0.0.1/callback")
		assert.NoError(t, err)

		// Other hosts should be rejected
		err = service.validateRedirectURIForRegistration("https://example.com/callback")
		assert.Error(t, err)

		oauthErr, ok := err.(*types.ErrorResponse)
		assert.True(t, ok)
		assert.Equal(t, types.ErrorInvalidRequest, oauthErr.Code)
		assert.Contains(t, oauthErr.ErrorDescription, "Redirect URI host 'example.com' is not allowed")
	})
}

// =============================================================================
// Grant Type and Response Type Validation Tests
// =============================================================================

func TestIsValidGrantType(t *testing.T) {
	service, _, _, cleanup := setupOAuthTestEnvironment(t)
	defer cleanup()

	t.Run("validate valid grant types", func(t *testing.T) {
		validGrantTypes := []string{
			types.GrantTypeAuthorizationCode,
			types.GrantTypeRefreshToken,
			types.GrantTypeClientCredentials,
			types.GrantTypeDeviceCode,
			types.GrantTypeTokenExchange,
		}

		for _, grantType := range validGrantTypes {
			assert.True(t, service.isValidGrantType(grantType), "Grant type %s should be valid", grantType)
		}
	})

	t.Run("validate invalid grant types", func(t *testing.T) {
		invalidGrantTypes := []string{
			"invalid_grant_type",
			"password", // Not supported in OAuth 2.1
			"implicit", // Not supported in OAuth 2.1
			"",
			"authorization_code_invalid",
		}

		for _, grantType := range invalidGrantTypes {
			assert.False(t, service.isValidGrantType(grantType), "Grant type %s should be invalid", grantType)
		}
	})
}

func TestIsValidResponseType(t *testing.T) {
	service, _, _, cleanup := setupOAuthTestEnvironment(t)
	defer cleanup()

	t.Run("validate valid response types", func(t *testing.T) {
		validResponseTypes := []string{
			types.ResponseTypeCode,
			types.ResponseTypeToken,
			types.ResponseTypeIDToken,
			"code token",
			"code id_token",
			"token id_token",
			"code token id_token",
		}

		for _, responseType := range validResponseTypes {
			assert.True(t, service.isValidResponseType(responseType), "Response type %s should be valid", responseType)
		}
	})

	t.Run("validate invalid response types", func(t *testing.T) {
		invalidResponseTypes := []string{
			"invalid_response_type",
			"code invalid",
			"",
			"token code",    // Wrong order
			"id_token code", // Wrong order
		}

		for _, responseType := range invalidResponseTypes {
			assert.False(t, service.isValidResponseType(responseType), "Response type %s should be invalid", responseType)
		}
	})
}

// =============================================================================
// Client Secret Expiration Tests
// =============================================================================

func TestClientSecretExpiration(t *testing.T) {
	service, _, _, cleanup := setupOAuthTestEnvironment(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("client secret with expiration", func(t *testing.T) {
		// Set client secret lifetime
		originalLifetime := service.config.Client.ClientSecretLifetime
		service.config.Client.ClientSecretLifetime = 24 * time.Hour
		defer func() {
			service.config.Client.ClientSecretLifetime = originalLifetime
		}()

		request := &types.DynamicClientRegistrationRequest{
			ClientName:   "Expiring Secret Client",
			RedirectURIs: []string{"https://localhost/callback"},
		}

		response, err := service.DynamicClientRegistration(ctx, request)
		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.NotEmpty(t, response.ClientSecret)
		assert.Greater(t, response.ClientSecretExpiresAt, int64(0))

		// Check that expiration is approximately 24 hours from now
		expectedExpiration := time.Now().Add(24 * time.Hour).Unix()
		assert.InDelta(t, expectedExpiration, response.ClientSecretExpiresAt, 60) // Within 1 minute
	})

	t.Run("client secret without expiration", func(t *testing.T) {
		// Set client secret lifetime to 0 (no expiration)
		originalLifetime := service.config.Client.ClientSecretLifetime
		service.config.Client.ClientSecretLifetime = 0
		defer func() {
			service.config.Client.ClientSecretLifetime = originalLifetime
		}()

		request := &types.DynamicClientRegistrationRequest{
			ClientName:   "Non-Expiring Secret Client",
			RedirectURIs: []string{"https://localhost/callback"},
		}

		response, err := service.DynamicClientRegistration(ctx, request)
		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.NotEmpty(t, response.ClientSecret)
		assert.Equal(t, int64(0), response.ClientSecretExpiresAt)
	})
}

// =============================================================================
// Edge Cases and Error Handling Tests
// =============================================================================

func TestClientRegistrationEdgeCases(t *testing.T) {
	service, _, _, cleanup := setupOAuthTestEnvironment(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("register with very long client name", func(t *testing.T) {
		request := &types.DynamicClientRegistrationRequest{
			ClientName:   strings.Repeat("A", 1000), // Very long name
			RedirectURIs: []string{"https://localhost/callback"},
		}

		response, err := service.DynamicClientRegistration(ctx, request)
		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.Equal(t, request.ClientName, response.ClientName)
	})

	t.Run("register with many redirect URIs", func(t *testing.T) {
		redirectURIs := make([]string, 10)
		for i := 0; i < 10; i++ {
			redirectURIs[i] = "https://localhost/callback" + string(rune('0'+i))
		}

		request := &types.DynamicClientRegistrationRequest{
			ClientName:   "Many Redirect URIs Client",
			RedirectURIs: redirectURIs,
		}

		response, err := service.DynamicClientRegistration(ctx, request)
		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.Equal(t, redirectURIs, response.RedirectURIs)
	})

	t.Run("register with mixed grant types", func(t *testing.T) {
		request := &types.DynamicClientRegistrationRequest{
			ClientName:   "Mixed Grant Types Client",
			RedirectURIs: []string{"https://localhost/callback"},
			GrantTypes: []string{
				types.GrantTypeAuthorizationCode,
				types.GrantTypeRefreshToken,
				types.GrantTypeClientCredentials,
			},
		}

		response, err := service.DynamicClientRegistration(ctx, request)
		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.Equal(t, request.GrantTypes, response.GrantTypes)
	})

	t.Run("register with JWT token endpoint auth method", func(t *testing.T) {
		request := &types.DynamicClientRegistrationRequest{
			ClientName:              "JWT Auth Client",
			RedirectURIs:            []string{"https://localhost/callback"},
			TokenEndpointAuthMethod: types.TokenEndpointAuthJWT,
		}

		response, err := service.DynamicClientRegistration(ctx, request)
		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.NotEmpty(t, response.ClientSecret)
		assert.Equal(t, types.TokenEndpointAuthJWT, response.TokenEndpointAuthMethod)
	})
}

func TestClientRegistrationWithCustomConfiguration(t *testing.T) {
	service, _, _, cleanup := setupOAuthTestEnvironment(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("register with custom client ID and secret lengths", func(t *testing.T) {
		// Set custom lengths
		originalIDLength := service.config.Client.ClientIDLength
		originalSecretLength := service.config.Client.ClientSecretLength
		service.config.Client.ClientIDLength = 16
		service.config.Client.ClientSecretLength = 32
		defer func() {
			service.config.Client.ClientIDLength = originalIDLength
			service.config.Client.ClientSecretLength = originalSecretLength
		}()

		request := &types.DynamicClientRegistrationRequest{
			ClientName:   "Custom Length Client",
			RedirectURIs: []string{"https://localhost/callback"},
		}

		response, err := service.DynamicClientRegistration(ctx, request)
		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.NotEmpty(t, response.ClientID)
		assert.NotEmpty(t, response.ClientSecret)

		// Check that the lengths are appropriate for the custom settings
		assert.Greater(t, len(response.ClientID), 10)
		assert.Greater(t, len(response.ClientSecret), 20)
	})

	t.Run("register with custom default values", func(t *testing.T) {
		// Set custom defaults
		originalGrantTypes := service.config.Client.DefaultGrantTypes
		originalResponseTypes := service.config.Client.DefaultResponseTypes
		originalAuthMethod := service.config.Client.DefaultTokenEndpointAuthMethod

		service.config.Client.DefaultGrantTypes = []string{types.GrantTypeClientCredentials}
		service.config.Client.DefaultResponseTypes = []string{types.ResponseTypeCode}
		service.config.Client.DefaultTokenEndpointAuthMethod = types.TokenEndpointAuthPost

		defer func() {
			service.config.Client.DefaultGrantTypes = originalGrantTypes
			service.config.Client.DefaultResponseTypes = originalResponseTypes
			service.config.Client.DefaultTokenEndpointAuthMethod = originalAuthMethod
		}()

		request := &types.DynamicClientRegistrationRequest{
			ClientName:   "Custom Defaults Client",
			RedirectURIs: []string{"https://localhost/callback"},
		}

		response, err := service.DynamicClientRegistration(ctx, request)
		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.Equal(t, []string{types.GrantTypeClientCredentials}, response.GrantTypes)
		assert.Equal(t, []string{types.ResponseTypeCode}, response.ResponseTypes)
		assert.Equal(t, types.TokenEndpointAuthPost, response.TokenEndpointAuthMethod)
	})
}
