package oauth

import (
	"context"
	"fmt"

	"github.com/yaoapp/yao/openapi/oauth/types"
)

// JWKS returns the JSON Web Key Set for token verification
// This endpoint provides public keys for validating JWT tokens
func (s *Service) JWKS(ctx context.Context) (*types.JWKSResponse, error) {
	// TODO: Implement JWKS endpoint - this requires certificate/key management
	// For now, return empty JWKS
	return &types.JWKSResponse{
		Keys: []types.JWK{},
	}, nil
}

// Endpoints returns a map of all available OAuth endpoints
// This provides endpoint discovery for clients
func (s *Service) Endpoints(ctx context.Context) (map[string]string, error) {
	baseURL := s.config.IssuerURL

	endpoints := map[string]string{
		"authorization_endpoint":                fmt.Sprintf("%s/oauth/authorize", baseURL),
		"token_endpoint":                        fmt.Sprintf("%s/oauth/token", baseURL),
		"userinfo_endpoint":                     fmt.Sprintf("%s/oauth/userinfo", baseURL),
		"jwks_uri":                              fmt.Sprintf("%s/oauth/jwks", baseURL),
		"registration_endpoint":                 fmt.Sprintf("%s/oauth/register", baseURL),
		"introspection_endpoint":                fmt.Sprintf("%s/oauth/introspect", baseURL),
		"revocation_endpoint":                   fmt.Sprintf("%s/oauth/revoke", baseURL),
		"device_authorization_endpoint":         fmt.Sprintf("%s/oauth/device", baseURL),
		"pushed_authorization_request_endpoint": fmt.Sprintf("%s/oauth/par", baseURL),
	}

	return endpoints, nil
}

// GetServerMetadata returns OAuth 2.0 Authorization Server Metadata
// This implements RFC 8414 for server discovery
func (s *Service) GetServerMetadata(ctx context.Context) (*types.AuthorizationServerMetadata, error) {
	endpoints, err := s.Endpoints(ctx)
	if err != nil {
		return nil, err
	}

	metadata := &types.AuthorizationServerMetadata{
		Issuer:                            s.config.IssuerURL,
		AuthorizationEndpoint:             endpoints["authorization_endpoint"],
		TokenEndpoint:                     endpoints["token_endpoint"],
		UserinfoEndpoint:                  endpoints["userinfo_endpoint"],
		JwksURI:                           endpoints["jwks_uri"],
		RegistrationEndpoint:              endpoints["registration_endpoint"],
		ScopesSupported:                   []string{"openid", "profile", "email", "address", "phone", "offline_access"},
		ResponseTypesSupported:            []string{"code", "token", "id_token", "code token", "code id_token", "token id_token", "code token id_token"},
		ResponseModesSupported:            []string{"query", "fragment", "form_post"},
		GrantTypesSupported:               []string{"authorization_code", "client_credentials", "refresh_token"},
		TokenEndpointAuthMethodsSupported: []string{"client_secret_basic", "client_secret_post", "client_secret_jwt", "private_key_jwt"},
		TokenEndpointAuthSigningAlgValuesSupported: []string{"RS256", "HS256"},
		ServiceDocumentation:                       fmt.Sprintf("%s/docs", s.config.IssuerURL),
		UILocalesSupported:                         []string{"en-US", "en-GB", "en-CA", "fr-FR", "fr-CA"},
		OpPolicyURI:                                fmt.Sprintf("%s/policy", s.config.IssuerURL),
		OpTosURI:                                   fmt.Sprintf("%s/terms", s.config.IssuerURL),
		RevocationEndpoint:                         endpoints["revocation_endpoint"],
		RevocationEndpointAuthMethodsSupported:     []string{"client_secret_basic", "client_secret_post", "client_secret_jwt", "private_key_jwt"},
		IntrospectionEndpoint:                      endpoints["introspection_endpoint"],
		IntrospectionEndpointAuthMethodsSupported:  []string{"client_secret_basic", "client_secret_post", "client_secret_jwt", "private_key_jwt"},
		CodeChallengeMethodsSupported:              []string{"plain", "S256"},
		DeviceAuthorizationEndpoint:                endpoints["device_authorization_endpoint"],
		PushedAuthorizationRequestEndpoint:         endpoints["pushed_authorization_request_endpoint"],
		RequirePushedAuthorizationRequests:         false,
		DPoPSigningAlgValuesSupported:              []string{"RS256", "PS256", "ES256"},
	}

	// Add feature-specific endpoints and capabilities
	if s.config.Features.DeviceFlowEnabled {
		metadata.DeviceAuthorizationEndpoint = endpoints["device_authorization_endpoint"]
		metadata.GrantTypesSupported = append(metadata.GrantTypesSupported, "urn:ietf:params:oauth:grant-type:device_code")
	}

	if s.config.Features.TokenExchangeEnabled {
		metadata.GrantTypesSupported = append(metadata.GrantTypesSupported, "urn:ietf:params:oauth:grant-type:token-exchange")
	}

	if s.config.Features.PushedAuthorizationEnabled {
		metadata.PushedAuthorizationRequestEndpoint = endpoints["pushed_authorization_request_endpoint"]
		metadata.RequirePushedAuthorizationRequests = true
	}

	if s.config.Features.DynamicClientRegistrationEnabled {
		metadata.RegistrationEndpoint = endpoints["registration_endpoint"]
	}

	return metadata, nil
}
