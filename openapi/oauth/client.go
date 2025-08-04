package oauth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"

	"github.com/yaoapp/yao/openapi/oauth/types"
)

// Register registers a new OAuth client with the authorization server
func (s *Service) Register(ctx context.Context, clientInfo *types.ClientInfo) (*types.ClientInfo, error) {
	return s.clientProvider.CreateClient(ctx, clientInfo)
}

// UpdateClient updates an existing OAuth client configuration
func (s *Service) UpdateClient(ctx context.Context, clientID string, clientInfo *types.ClientInfo) (*types.ClientInfo, error) {
	return s.clientProvider.UpdateClient(ctx, clientID, clientInfo)
}

// DeleteClient removes an OAuth client from the authorization server
func (s *Service) DeleteClient(ctx context.Context, clientID string) error {
	return s.clientProvider.DeleteClient(ctx, clientID)
}

// ValidateScope validates requested scopes against available scopes
func (s *Service) ValidateScope(ctx context.Context, requestedScopes []string, clientID string) (*types.ValidationResult, error) {
	return s.clientProvider.ValidateScope(ctx, clientID, requestedScopes)
}

// DynamicClientRegistration handles dynamic client registration
// This implements RFC 7591 for automatic client registration
func (s *Service) DynamicClientRegistration(ctx context.Context, request *types.DynamicClientRegistrationRequest) (*types.DynamicClientRegistrationResponse, error) {
	// Check if dynamic client registration is enabled
	if !s.config.Features.DynamicClientRegistrationEnabled {
		return nil, &types.ErrorResponse{
			Code:             types.ErrorInvalidRequest,
			ErrorDescription: "Dynamic client registration is not enabled",
		}
	}

	// Validate the request
	if err := s.validateDynamicClientRegistrationRequest(request); err != nil {
		return nil, err
	}

	// Generate client ID and secret (use the client ID from the request if provided or generate a new one)
	clientID := request.ClientID
	var err error
	if clientID == "" {
		var err error
		clientID, err = s.GenerateClientID()
		if err != nil {
			return nil, &types.ErrorResponse{
				Code:             types.ErrorServerError,
				ErrorDescription: "Failed to generate client ID",
			}
		}
	}

	clientSecret := ""
	// Determine client type based on token endpoint auth method
	clientType := types.ClientTypePublic
	if request.TokenEndpointAuthMethod == "" ||
		request.TokenEndpointAuthMethod == types.TokenEndpointAuthBasic ||
		request.TokenEndpointAuthMethod == types.TokenEndpointAuthPost ||
		request.TokenEndpointAuthMethod == types.TokenEndpointAuthJWT {
		clientType = types.ClientTypeConfidential
		clientSecret, err = s.GenerateClientSecret()
		if err != nil {
			return nil, &types.ErrorResponse{
				Code:             types.ErrorServerError,
				ErrorDescription: "Failed to generate client secret",
			}
		}
	}

	// Create client info from request
	clientInfo := &types.ClientInfo{
		ClientID:                clientID,
		ClientSecret:            clientSecret,
		ClientName:              request.ClientName,
		ClientType:              clientType,
		RedirectURIs:            request.RedirectURIs,
		ResponseTypes:           request.ResponseTypes,
		GrantTypes:              request.GrantTypes,
		ApplicationType:         request.ApplicationType,
		Contacts:                request.Contacts,
		ClientURI:               request.ClientURI,
		LogoURI:                 request.LogoURI,
		Scope:                   request.Scope,
		TosURI:                  request.TosURI,
		PolicyURI:               request.PolicyURI,
		JwksURI:                 request.JwksURI,
		JwksValue:               request.Jwks,
		TokenEndpointAuthMethod: request.TokenEndpointAuthMethod,
	}

	// Set defaults if not provided
	if len(clientInfo.GrantTypes) == 0 {
		clientInfo.GrantTypes = s.config.Client.DefaultGrantTypes
	}
	if len(clientInfo.ResponseTypes) == 0 {
		clientInfo.ResponseTypes = s.config.Client.DefaultResponseTypes
	}
	if clientInfo.ApplicationType == "" {
		clientInfo.ApplicationType = types.ApplicationTypeWeb
	}
	if clientInfo.TokenEndpointAuthMethod == "" {
		clientInfo.TokenEndpointAuthMethod = s.config.Client.DefaultTokenEndpointAuthMethod
	}

	// Update the request with defaults for the response
	if len(request.GrantTypes) == 0 {
		request.GrantTypes = clientInfo.GrantTypes
	}
	if len(request.ResponseTypes) == 0 {
		request.ResponseTypes = clientInfo.ResponseTypes
	}
	if request.ApplicationType == "" {
		request.ApplicationType = clientInfo.ApplicationType
	}
	if request.TokenEndpointAuthMethod == "" {
		request.TokenEndpointAuthMethod = clientInfo.TokenEndpointAuthMethod
	}

	// Create the client
	createdClient, err := s.clientProvider.CreateClient(ctx, clientInfo)
	if err != nil {
		return nil, err
	}

	// Create response
	response := &types.DynamicClientRegistrationResponse{
		ClientID:                         createdClient.ClientID,
		ClientSecret:                     createdClient.ClientSecret,
		ClientIDIssuedAt:                 createdClient.CreatedAt.Unix(),
		DynamicClientRegistrationRequest: request,
	}

	// Set client secret expiration (0 means it never expires)
	if s.config.Client.ClientSecretLifetime > 0 {
		response.ClientSecretExpiresAt = createdClient.CreatedAt.Add(s.config.Client.ClientSecretLifetime).Unix()
	}

	return response, nil
}

// GenerateClientID generates a random client ID
func (s *Service) GenerateClientID() (string, error) {
	length := s.config.Client.ClientIDLength
	if length == 0 {
		length = 32
	}

	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	// Use base64 URL encoding without padding
	return strings.TrimRight(base64.URLEncoding.EncodeToString(bytes), "="), nil
}

// ValidateClientID validates the client ID
func (s *Service) ValidateClientID(clientID string) error {
	if clientID == "" {
		return &types.ErrorResponse{
			Code:             types.ErrorInvalidRequest,
			ErrorDescription: "Client ID is required",
		}
	}
	length := s.config.Client.ClientIDLength
	if length == 0 {
		length = 32
	}

	if len(clientID) != length {
		return &types.ErrorResponse{
			Code:             types.ErrorInvalidRequest,
			ErrorDescription: fmt.Sprintf("Client ID must be %d characters long", length),
		}
	}
	return nil
}

// GenerateClientSecret generates a random client secret
func (s *Service) GenerateClientSecret() (string, error) {
	length := s.config.Client.ClientSecretLength
	if length == 0 {
		length = 64
	}

	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	// Use base64 URL encoding without padding
	return strings.TrimRight(base64.URLEncoding.EncodeToString(bytes), "="), nil
}

// validateDynamicClientRegistrationRequest validates the dynamic client registration request
func (s *Service) validateDynamicClientRegistrationRequest(request *types.DynamicClientRegistrationRequest) error {
	// Validate redirect URIs
	if len(request.RedirectURIs) == 0 && (strings.Contains(request.Scope, "openid") || strings.Contains(request.Scope, "profile") || strings.Contains(request.Scope, "email")) {
		return &types.ErrorResponse{
			Code:             types.ErrorInvalidRequest,
			ErrorDescription: "At least one redirect URI is required",
		}
	}

	// Validate redirect URI schemes and hosts
	for _, uri := range request.RedirectURIs {
		if err := s.validateRedirectURIForRegistration(uri); err != nil {
			return err
		}
	}

	// Validate grant types
	if len(request.GrantTypes) > 0 {
		for _, grantType := range request.GrantTypes {
			if !s.isValidGrantType(grantType) {
				return &types.ErrorResponse{
					Code:             types.ErrorInvalidRequest,
					ErrorDescription: fmt.Sprintf("Invalid grant type: %s", grantType),
				}
			}
		}
	}

	// Validate response types
	if len(request.ResponseTypes) > 0 {
		for _, responseType := range request.ResponseTypes {
			if !s.isValidResponseType(responseType) {
				return &types.ErrorResponse{
					Code:             types.ErrorInvalidRequest,
					ErrorDescription: fmt.Sprintf("Invalid response type: %s", responseType),
				}
			}
		}
	}

	// Validate application type
	if request.ApplicationType != "" {
		if request.ApplicationType != types.ApplicationTypeWeb && request.ApplicationType != types.ApplicationTypeNative {
			return &types.ErrorResponse{
				Code:             types.ErrorInvalidRequest,
				ErrorDescription: "Invalid application type",
			}
		}
	}

	return nil
}

// validateRedirectURIForRegistration validates redirect URI for dynamic registration
func (s *Service) validateRedirectURIForRegistration(uri string) error {
	parsedURI, err := url.Parse(uri)
	if err != nil {
		return &types.ErrorResponse{
			Code:             types.ErrorInvalidRequest,
			ErrorDescription: "Invalid redirect URI format",
		}
	}

	// Check allowed schemes
	if len(s.config.Client.AllowedRedirectURISchemes) > 0 {
		schemeAllowed := false
		for _, scheme := range s.config.Client.AllowedRedirectURISchemes {
			if parsedURI.Scheme == scheme {
				schemeAllowed = true
				break
			}
		}
		if !schemeAllowed {
			return &types.ErrorResponse{
				Code:             types.ErrorInvalidRequest,
				ErrorDescription: fmt.Sprintf("Redirect URI scheme '%s' is not allowed", parsedURI.Scheme),
			}
		}
	}

	// Check allowed hosts
	if len(s.config.Client.AllowedRedirectURIHosts) > 0 {
		hostAllowed := false
		for _, host := range s.config.Client.AllowedRedirectURIHosts {
			if parsedURI.Host == host {
				hostAllowed = true
				break
			}
		}
		if !hostAllowed {
			return &types.ErrorResponse{
				Code:             types.ErrorInvalidRequest,
				ErrorDescription: fmt.Sprintf("Redirect URI host '%s' is not allowed", parsedURI.Host),
			}
		}
	}

	return nil
}

// isValidGrantType checks if a grant type is valid
func (s *Service) isValidGrantType(grantType string) bool {
	validGrantTypes := []string{
		types.GrantTypeAuthorizationCode,
		types.GrantTypeRefreshToken,
		types.GrantTypeClientCredentials,
		types.GrantTypeDeviceCode,
		types.GrantTypeTokenExchange,
	}

	for _, valid := range validGrantTypes {
		if grantType == valid {
			return true
		}
	}
	return false
}

// isValidResponseType checks if a response type is valid
func (s *Service) isValidResponseType(responseType string) bool {
	validResponseTypes := []string{
		types.ResponseTypeCode,
		types.ResponseTypeToken,
		types.ResponseTypeIDToken,
		"code token",
		"code id_token",
		"token id_token",
		"code token id_token",
	}

	for _, valid := range validResponseTypes {
		if responseType == valid {
			return true
		}
	}
	return false
}
