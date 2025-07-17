package oauth

import (
	"context"
)

// OAuth interface defines the complete OAuth 2.1 and MCP authorization server functionality
type OAuth interface {
	// AuthorizationServer returns the authorization server endpoint URL
	// This endpoint is used to initiate the authorization flow
	AuthorizationServer(ctx context.Context) string

	// ProtectedResource returns the protected resource endpoint URL
	// This endpoint is used to access protected resources using access tokens
	ProtectedResource(ctx context.Context) string

	// Authorize processes an authorization request and returns an authorization code
	// The authorization code can be exchanged for an access token
	Authorize(ctx context.Context, request *AuthorizationRequest) (*AuthorizationResponse, error)

	// Token exchanges an authorization code for an access token
	// This is the core token endpoint functionality
	Token(ctx context.Context, grantType string, code string, clientID string, codeVerifier string) (*Token, error)

	// Revoke revokes an access token or refresh token
	// Once revoked, the token cannot be used for accessing protected resources
	Revoke(ctx context.Context, token string, tokenTypeHint string) error

	// Introspect returns information about an access token
	// This endpoint allows resource servers to validate tokens
	Introspect(ctx context.Context, token string) (*TokenIntrospectionResponse, error)

	// Register registers a new OAuth client with the authorization server
	// This is used for static client registration
	Register(ctx context.Context, clientInfo *ClientInfo) (*ClientInfo, error)

	// JWKS returns the JSON Web Key Set for token verification
	// This endpoint provides public keys for validating JWT tokens
	JWKS(ctx context.Context) (*JWKSResponse, error)

	// Endpoints returns a map of all available OAuth endpoints
	// This provides endpoint discovery for clients
	Endpoints(ctx context.Context) (map[string]string, error)

	// RefreshToken exchanges a refresh token for a new access token
	// This allows clients to obtain fresh access tokens without user interaction
	RefreshToken(ctx context.Context, refreshToken string, scope string) (*RefreshTokenResponse, error)

	// DeviceAuthorization initiates the device authorization flow
	// This is used for devices with limited input capabilities
	DeviceAuthorization(ctx context.Context, clientID string, scope string) (*DeviceAuthorizationResponse, error)

	// UserInfo returns user information for a given access token
	// This endpoint provides user profile information in the format defined by the UserProvider
	UserInfo(ctx context.Context, accessToken string) (interface{}, error)

	// GenerateCodeChallenge generates a code challenge from a code verifier
	// This is used for PKCE (Proof Key for Code Exchange) flow
	GenerateCodeChallenge(ctx context.Context, codeVerifier string, method string) (string, error)

	// ValidateCodeChallenge validates a code verifier against a code challenge
	// This verifies the PKCE code challenge during token exchange
	ValidateCodeChallenge(ctx context.Context, codeVerifier string, codeChallenge string, method string) error

	// PushAuthorizationRequest processes a pushed authorization request
	// This implements RFC 9126 for enhanced security
	PushAuthorizationRequest(ctx context.Context, request *PushedAuthorizationRequest) (*PushedAuthorizationResponse, error)

	// TokenExchange exchanges one token for another token
	// This implements RFC 8693 for token exchange scenarios
	TokenExchange(ctx context.Context, subjectToken string, subjectTokenType string, audience string, scope string) (*TokenExchangeResponse, error)

	// UpdateClient updates an existing OAuth client configuration
	// This allows modification of client metadata
	UpdateClient(ctx context.Context, clientID string, clientInfo *ClientInfo) (*ClientInfo, error)

	// DeleteClient removes an OAuth client from the authorization server
	// This permanently deletes the client and invalidates all associated tokens
	DeleteClient(ctx context.Context, clientID string) error

	// ValidateScope validates requested scopes against available scopes
	// This ensures clients only request permitted scopes
	ValidateScope(ctx context.Context, requestedScopes []string, clientID string) (*ValidationResult, error)

	// GetServerMetadata returns OAuth 2.0 Authorization Server Metadata
	// This implements RFC 8414 for server discovery
	GetServerMetadata(ctx context.Context) (*AuthorizationServerMetadata, error)

	// MCP Requirements

	// ValidateResourceParameter validates an OAuth 2.0 resource parameter
	// This ensures the resource parameter is valid and properly formatted
	ValidateResourceParameter(ctx context.Context, resource string) (*ValidationResult, error)

	// GetCanonicalResourceURI returns the canonical form of a resource URI
	// This normalizes resource URIs for consistent processing
	GetCanonicalResourceURI(ctx context.Context, serverURI string) (string, error)

	// GetProtectedResourceMetadata returns OAuth 2.0 Protected Resource Metadata
	// This implements RFC 9728 for MCP server discovery
	GetProtectedResourceMetadata(ctx context.Context) (*ProtectedResourceMetadata, error)

	// HandleWWWAuthenticate processes WWW-Authenticate challenges
	// This handles authentication challenges from protected resources
	HandleWWWAuthenticate(ctx context.Context, challenge string) (*WWWAuthenticateChallenge, error)

	// DynamicClientRegistration handles dynamic client registration
	// This implements RFC 7591 for automatic client registration
	DynamicClientRegistration(ctx context.Context, request *DynamicClientRegistrationRequest) (*DynamicClientRegistrationResponse, error)

	// ValidateStateParameter validates OAuth state parameters
	// This prevents CSRF attacks by verifying state parameters
	ValidateStateParameter(ctx context.Context, state string, clientID string) (*ValidationResult, error)

	// GenerateStateParameter generates a secure state parameter
	// This creates cryptographically secure state values for CSRF protection
	GenerateStateParameter(ctx context.Context, clientID string) (*StateParameter, error)

	// ValidateTokenAudience validates token audience claims
	// This ensures tokens are only used with their intended audiences
	ValidateTokenAudience(ctx context.Context, token string, expectedAudience string) (*ValidationResult, error)

	// MCP Security Requirements

	// ValidateRedirectURI validates redirect URIs against registered URIs
	// This prevents open redirect attacks by enforcing exact URI matching
	ValidateRedirectURI(ctx context.Context, redirectURI string, registeredURIs []string) (*ValidationResult, error)

	// RotateRefreshToken rotates a refresh token and invalidates the old one
	// This implements refresh token rotation for enhanced security
	RotateRefreshToken(ctx context.Context, oldToken string) (*RefreshTokenResponse, error)

	// ValidateTokenBinding validates token binding information
	// This ensures tokens are bound to the correct client or device
	ValidateTokenBinding(ctx context.Context, token string, binding *TokenBinding) (*ValidationResult, error)
}

// UserProvider interface for user information retrieval
type UserProvider interface {
	// GetUserByAccessToken retrieves user information using an access token
	GetUserByAccessToken(ctx context.Context, accessToken string) (interface{}, error)

	// GetUserBySubject retrieves user information using a subject identifier
	GetUserBySubject(ctx context.Context, subject string) (interface{}, error)

	// ValidateUserScope validates if a user has access to requested scopes
	ValidateUserScope(ctx context.Context, userID string, scopes []string) (bool, error)
}
