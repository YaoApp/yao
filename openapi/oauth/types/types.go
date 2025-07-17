package types

import (
	"time"
)

// ErrorResponse represents an OAuth 2.1 error response
type ErrorResponse struct {
	Code             string `json:"error"`
	ErrorDescription string `json:"error_description,omitempty"`
	ErrorURI         string `json:"error_uri,omitempty"`
	State            string `json:"state,omitempty"`
}

// Error implements the error interface
func (e *ErrorResponse) Error() string {
	if e.ErrorDescription != "" {
		return e.Code + ": " + e.ErrorDescription
	}
	return e.Code
}

// OAuth 2.1 Grant Types
const (
	GrantTypeAuthorizationCode = "authorization_code"
	GrantTypeClientCredentials = "client_credentials"
	GrantTypeRefreshToken      = "refresh_token"
	GrantTypeDeviceCode        = "urn:ietf:params:oauth:grant-type:device_code"
	GrantTypeTokenExchange     = "urn:ietf:params:oauth:grant-type:token-exchange"
)

// OAuth 2.1 Response Types
const (
	ResponseTypeCode = "code"
)

// OAuth 2.1 Token Types
const (
	TokenTypeBearer = "Bearer"
	TokenTypeMAC    = "MAC"
	TokenTypeDPoP   = "DPoP"
)

// OAuth 2.1 Client Types
const (
	ClientTypeConfidential = "confidential"
	ClientTypePublic       = "public"
	ClientTypeCredentialed = "credentialed"
)

// PKCE Code Challenge Methods
const (
	CodeChallengeMethodS256  = "S256"
	CodeChallengeMethodPlain = "plain"
)

// OAuth 2.1 Error Codes
const (
	ErrorInvalidRequest          = "invalid_request"
	ErrorInvalidClient           = "invalid_client"
	ErrorInvalidGrant            = "invalid_grant"
	ErrorUnauthorizedClient      = "unauthorized_client"
	ErrorUnsupportedGrantType    = "unsupported_grant_type"
	ErrorInvalidScope            = "invalid_scope"
	ErrorAccessDenied            = "access_denied"
	ErrorUnsupportedResponseType = "unsupported_response_type"
	ErrorServerError             = "server_error"
	ErrorTemporarilyUnavailable  = "temporarily_unavailable"
	ErrorInvalidToken            = "invalid_token"
	ErrorInsufficientScope       = "insufficient_scope"
	ErrorExpiredToken            = "expired_token"
	ErrorAuthorizationPending    = "authorization_pending"
	ErrorSlowDown                = "slow_down"
)

// Token Binding Types
const (
	TokenBindingTypeDPoP        = "dpop"
	TokenBindingTypeMTLS        = "mtls"
	TokenBindingTypeCertificate = "certificate"
)

// Application Types
const (
	ApplicationTypeWeb    = "web"
	ApplicationTypeNative = "native"
)

// Token Endpoint Authentication Methods
const (
	TokenEndpointAuthNone          = "none"
	TokenEndpointAuthPost          = "client_secret_post"
	TokenEndpointAuthBasic         = "client_secret_basic"
	TokenEndpointAuthJWT           = "client_secret_jwt"
	TokenEndpointAuthPrivateKeyJWT = "private_key_jwt"
	TokenEndpointAuthTLSClientAuth = "tls_client_auth"
	TokenEndpointAuthSelfSignedTLS = "self_signed_tls_client_auth"
)

// Response Modes
const (
	ResponseModeQuery    = "query"
	ResponseModeFragment = "fragment"
	ResponseModeFormPost = "form_post"
)

// Standard OAuth Scopes
const (
	ScopeOpenID  = "openid"
	ScopeProfile = "profile"
	ScopeEmail   = "email"
	ScopeAddress = "address"
	ScopePhone   = "phone"
	ScopeOffline = "offline_access"
)

// MCP Specific Constants
const (
	MCPResourceParameter = "resource"
	MCPBearerTokenHeader = "Authorization"
	MCPBearerTokenPrefix = "Bearer "
)

// WWW-Authenticate Schemes
const (
	WWWAuthenticateSchemeBearer = "Bearer"
	WWWAuthenticateSchemeBasic  = "Basic"
	WWWAuthenticateSchemeDPoP   = "DPoP"
)

// Token represents an OAuth 2.1 access token
type Token struct {
	AccessToken  string    `json:"access_token"`
	TokenType    string    `json:"token_type"`
	ExpiresIn    int       `json:"expires_in"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	Scope        string    `json:"scope,omitempty"`
	IssuedAt     time.Time `json:"issued_at"`
	ExpiresAt    time.Time `json:"expires_at"`
	Audience     []string  `json:"audience,omitempty"`
	Subject      string    `json:"subject,omitempty"`
	Issuer       string    `json:"issuer,omitempty"`
	ClientID     string    `json:"client_id,omitempty"`
}

// RefreshTokenResponse represents the response from refresh token endpoint
type RefreshTokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
}

// DeviceAuthorizationResponse represents device authorization response
type DeviceAuthorizationResponse struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete,omitempty"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval,omitempty"`
}

// UserInfo represents user information from userinfo endpoint
type UserInfo struct {
	Subject           string                 `json:"sub"`
	Name              string                 `json:"name,omitempty"`
	GivenName         string                 `json:"given_name,omitempty"`
	FamilyName        string                 `json:"family_name,omitempty"`
	MiddleName        string                 `json:"middle_name,omitempty"`
	Nickname          string                 `json:"nickname,omitempty"`
	PreferredUsername string                 `json:"preferred_username,omitempty"`
	Profile           string                 `json:"profile,omitempty"`
	Picture           string                 `json:"picture,omitempty"`
	Website           string                 `json:"website,omitempty"`
	Email             string                 `json:"email,omitempty"`
	EmailVerified     bool                   `json:"email_verified,omitempty"`
	Gender            string                 `json:"gender,omitempty"`
	Birthdate         string                 `json:"birthdate,omitempty"`
	Zoneinfo          string                 `json:"zoneinfo,omitempty"`
	Locale            string                 `json:"locale,omitempty"`
	PhoneNumber       string                 `json:"phone_number,omitempty"`
	PhoneVerified     bool                   `json:"phone_number_verified,omitempty"`
	Address           *UserAddress           `json:"address,omitempty"`
	UpdatedAt         int64                  `json:"updated_at,omitempty"`
	CustomClaims      map[string]interface{} `json:"-"`
}

// UserAddress represents user address information
type UserAddress struct {
	Formatted     string `json:"formatted,omitempty"`
	StreetAddress string `json:"street_address,omitempty"`
	Locality      string `json:"locality,omitempty"`
	Region        string `json:"region,omitempty"`
	PostalCode    string `json:"postal_code,omitempty"`
	Country       string `json:"country,omitempty"`
}

// ClientInfo represents OAuth client information
type ClientInfo struct {
	ClientID                string                 `json:"client_id"`
	ClientSecret            string                 `json:"client_secret,omitempty"`
	ClientName              string                 `json:"client_name,omitempty"`
	ClientType              string                 `json:"client_type"` // "confidential", "public", "credentialed"
	RedirectURIs            []string               `json:"redirect_uris"`
	ResponseTypes           []string               `json:"response_types,omitempty"`
	GrantTypes              []string               `json:"grant_types,omitempty"`
	ApplicationType         string                 `json:"application_type,omitempty"`
	Contacts                []string               `json:"contacts,omitempty"`
	ClientURI               string                 `json:"client_uri,omitempty"`
	LogoURI                 string                 `json:"logo_uri,omitempty"`
	Scope                   string                 `json:"scope,omitempty"`
	TosURI                  string                 `json:"tos_uri,omitempty"`
	PolicyURI               string                 `json:"policy_uri,omitempty"`
	JwksURI                 string                 `json:"jwks_uri,omitempty"`
	JwksValue               string                 `json:"jwks,omitempty"`
	TokenEndpointAuthMethod string                 `json:"token_endpoint_auth_method,omitempty"`
	CreatedAt               time.Time              `json:"created_at,omitempty"`
	UpdatedAt               time.Time              `json:"updated_at,omitempty"`
	Extra                   map[string]interface{} `json:"extra,omitempty"` // Extra fields for custom client properties
}

// AuthorizationServerMetadata represents OAuth 2.0 Authorization Server Metadata (RFC 8414)
type AuthorizationServerMetadata struct {
	Issuer                                     string   `json:"issuer"`
	AuthorizationEndpoint                      string   `json:"authorization_endpoint"`
	TokenEndpoint                              string   `json:"token_endpoint"`
	JwksURI                                    string   `json:"jwks_uri,omitempty"`
	RegistrationEndpoint                       string   `json:"registration_endpoint,omitempty"`
	ScopesSupported                            []string `json:"scopes_supported,omitempty"`
	ResponseTypesSupported                     []string `json:"response_types_supported"`
	ResponseModesSupported                     []string `json:"response_modes_supported,omitempty"`
	GrantTypesSupported                        []string `json:"grant_types_supported,omitempty"`
	TokenEndpointAuthMethodsSupported          []string `json:"token_endpoint_auth_methods_supported,omitempty"`
	TokenEndpointAuthSigningAlgValuesSupported []string `json:"token_endpoint_auth_signing_alg_values_supported,omitempty"`
	ServiceDocumentation                       string   `json:"service_documentation,omitempty"`
	UILocalesSupported                         []string `json:"ui_locales_supported,omitempty"`
	OpPolicyURI                                string   `json:"op_policy_uri,omitempty"`
	OpTosURI                                   string   `json:"op_tos_uri,omitempty"`
	RevocationEndpoint                         string   `json:"revocation_endpoint,omitempty"`
	RevocationEndpointAuthMethodsSupported     []string `json:"revocation_endpoint_auth_methods_supported,omitempty"`
	IntrospectionEndpoint                      string   `json:"introspection_endpoint,omitempty"`
	IntrospectionEndpointAuthMethodsSupported  []string `json:"introspection_endpoint_auth_methods_supported,omitempty"`
	CodeChallengeMethodsSupported              []string `json:"code_challenge_methods_supported,omitempty"`
	DeviceAuthorizationEndpoint                string   `json:"device_authorization_endpoint,omitempty"`
	UserinfoEndpoint                           string   `json:"userinfo_endpoint,omitempty"`
	PushedAuthorizationRequestEndpoint         string   `json:"pushed_authorization_request_endpoint,omitempty"`
	RequirePushedAuthorizationRequests         bool     `json:"require_pushed_authorization_requests,omitempty"`
	DPoPSigningAlgValuesSupported              []string `json:"dpop_signing_alg_values_supported,omitempty"`
}

// ProtectedResourceMetadata represents OAuth 2.0 Protected Resource Metadata (RFC 9728)
type ProtectedResourceMetadata struct {
	Resource               string   `json:"resource"`
	AuthorizationServers   []string `json:"authorization_servers"`
	JwksURI                string   `json:"jwks_uri,omitempty"`
	BearerMethodsSupported []string `json:"bearer_methods_supported,omitempty"`
	ResourceDocumentation  string   `json:"resource_documentation,omitempty"`
}

// TokenIntrospectionResponse represents token introspection response
type TokenIntrospectionResponse struct {
	Active    bool     `json:"active"`
	Scope     string   `json:"scope,omitempty"`
	ClientID  string   `json:"client_id,omitempty"`
	Username  string   `json:"username,omitempty"`
	TokenType string   `json:"token_type,omitempty"`
	ExpiresAt int64    `json:"exp,omitempty"`
	IssuedAt  int64    `json:"iat,omitempty"`
	NotBefore int64    `json:"nbf,omitempty"`
	Subject   string   `json:"sub,omitempty"`
	Audience  []string `json:"aud,omitempty"`
	Issuer    string   `json:"iss,omitempty"`
	JwtID     string   `json:"jti,omitempty"`
}

// PushedAuthorizationRequest represents PAR request
type PushedAuthorizationRequest struct {
	ClientID            string `json:"client_id"`
	ResponseType        string `json:"response_type"`
	RedirectURI         string `json:"redirect_uri"`
	Scope               string `json:"scope,omitempty"`
	State               string `json:"state,omitempty"`
	CodeChallenge       string `json:"code_challenge,omitempty"`
	CodeChallengeMethod string `json:"code_challenge_method,omitempty"`
	Resource            string `json:"resource,omitempty"`
	RequestURI          string `json:"request_uri,omitempty"`
	Request             string `json:"request,omitempty"`
}

// PushedAuthorizationResponse represents PAR response
type PushedAuthorizationResponse struct {
	RequestURI string `json:"request_uri"`
	ExpiresIn  int    `json:"expires_in"`
}

// TokenExchangeResponse represents token exchange response
type TokenExchangeResponse struct {
	AccessToken     string `json:"access_token"`
	IssuedTokenType string `json:"issued_token_type"`
	TokenType       string `json:"token_type"`
	ExpiresIn       int    `json:"expires_in,omitempty"`
	Scope           string `json:"scope,omitempty"`
	RefreshToken    string `json:"refresh_token,omitempty"`
}

// DynamicClientRegistrationRequest represents dynamic client registration request
type DynamicClientRegistrationRequest struct {
	RedirectURIs                []string `json:"redirect_uris"`
	ResponseTypes               []string `json:"response_types,omitempty"`
	GrantTypes                  []string `json:"grant_types,omitempty"`
	ApplicationType             string   `json:"application_type,omitempty"`
	Contacts                    []string `json:"contacts,omitempty"`
	ClientName                  string   `json:"client_name,omitempty"`
	LogoURI                     string   `json:"logo_uri,omitempty"`
	ClientURI                   string   `json:"client_uri,omitempty"`
	PolicyURI                   string   `json:"policy_uri,omitempty"`
	TosURI                      string   `json:"tos_uri,omitempty"`
	JwksURI                     string   `json:"jwks_uri,omitempty"`
	Jwks                        string   `json:"jwks,omitempty"`
	Scope                       string   `json:"scope,omitempty"`
	TokenEndpointAuthMethod     string   `json:"token_endpoint_auth_method,omitempty"`
	TokenEndpointAuthSigningAlg string   `json:"token_endpoint_auth_signing_alg,omitempty"`
	DefaultMaxAge               int      `json:"default_max_age,omitempty"`
	RequireAuthTime             bool     `json:"require_auth_time,omitempty"`
	DefaultACRValues            []string `json:"default_acr_values,omitempty"`
	InitiateLoginURI            string   `json:"initiate_login_uri,omitempty"`
	RequestURIs                 []string `json:"request_uris,omitempty"`
	SoftwareID                  string   `json:"software_id,omitempty"`
	SoftwareVersion             string   `json:"software_version,omitempty"`
	SoftwareStatement           string   `json:"software_statement,omitempty"`
}

// DynamicClientRegistrationResponse represents dynamic client registration response
type DynamicClientRegistrationResponse struct {
	ClientID                string `json:"client_id"`
	ClientSecret            string `json:"client_secret,omitempty"`
	ClientSecretExpiresAt   int64  `json:"client_secret_expires_at,omitempty"`
	RegistrationAccessToken string `json:"registration_access_token,omitempty"`
	RegistrationClientURI   string `json:"registration_client_uri,omitempty"`
	ClientIDIssuedAt        int64  `json:"client_id_issued_at,omitempty"`
	*DynamicClientRegistrationRequest
}

// WWWAuthenticateChallenge represents WWW-Authenticate challenge
type WWWAuthenticateChallenge struct {
	Scheme     string            `json:"scheme"`
	Realm      string            `json:"realm,omitempty"`
	Scope      string            `json:"scope,omitempty"`
	Error      string            `json:"error,omitempty"`
	ErrorDesc  string            `json:"error_description,omitempty"`
	ErrorURI   string            `json:"error_uri,omitempty"`
	Resource   string            `json:"resource,omitempty"`
	Parameters map[string]string `json:"parameters,omitempty"`
}

// StateParameter represents OAuth state parameter
type StateParameter struct {
	Value     string    `json:"value"`
	ExpiresAt time.Time `json:"expires_at"`
	ClientID  string    `json:"client_id"`
	Nonce     string    `json:"nonce,omitempty"`
}

// TokenBinding represents token binding information
type TokenBinding struct {
	TokenID      string                 `json:"token_id"`
	ClientID     string                 `json:"client_id"`
	BindingType  string                 `json:"binding_type"` // "dpop", "mtls", "certificate"
	BindingValue string                 `json:"binding_value"`
	BindingData  map[string]interface{} `json:"binding_data,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
	ExpiresAt    time.Time              `json:"expires_at"`
}

// ResourceParameter represents OAuth 2.0 resource parameter
type ResourceParameter struct {
	Resource    string    `json:"resource"`
	Canonical   string    `json:"canonical"`
	Audiences   []string  `json:"audiences,omitempty"`
	Scopes      []string  `json:"scopes,omitempty"`
	ValidatedAt time.Time `json:"validated_at"`
}

// ValidationResult represents validation result
type ValidationResult struct {
	Valid   bool              `json:"valid"`
	Errors  []string          `json:"errors,omitempty"`
	Details map[string]string `json:"details,omitempty"`
}

// AuthorizationRequest represents authorization request
type AuthorizationRequest struct {
	ClientID            string `json:"client_id"`
	ResponseType        string `json:"response_type"`
	RedirectURI         string `json:"redirect_uri"`
	Scope               string `json:"scope,omitempty"`
	State               string `json:"state,omitempty"`
	CodeChallenge       string `json:"code_challenge,omitempty"`
	CodeChallengeMethod string `json:"code_challenge_method,omitempty"`
	Resource            string `json:"resource,omitempty"`
	Nonce               string `json:"nonce,omitempty"`
}

// AuthorizationResponse represents authorization response
type AuthorizationResponse struct {
	Code             string `json:"code,omitempty"`
	State            string `json:"state,omitempty"`
	Error            string `json:"error,omitempty"`
	ErrorDescription string `json:"error_description,omitempty"`
}

// JWKSResponse represents JWKS response
type JWKSResponse struct {
	Keys []JWK `json:"keys"`
}

// JWK represents JSON Web Key
type JWK struct {
	Kty     string   `json:"kty"`
	Use     string   `json:"use,omitempty"`
	KeyOps  []string `json:"key_ops,omitempty"`
	Alg     string   `json:"alg,omitempty"`
	Kid     string   `json:"kid,omitempty"`
	X5U     string   `json:"x5u,omitempty"`
	X5C     []string `json:"x5c,omitempty"`
	X5T     string   `json:"x5t,omitempty"`
	X5TS256 string   `json:"x5t#S256,omitempty"`
	// RSA
	N  string `json:"n,omitempty"`
	E  string `json:"e,omitempty"`
	D  string `json:"d,omitempty"`
	P  string `json:"p,omitempty"`
	Q  string `json:"q,omitempty"`
	DP string `json:"dp,omitempty"`
	DQ string `json:"dq,omitempty"`
	QI string `json:"qi,omitempty"`
	// EC
	Crv string `json:"crv,omitempty"`
	X   string `json:"x,omitempty"`
	Y   string `json:"y,omitempty"`
	// Symmetric
	K string `json:"k,omitempty"`
}

// OAuth Service Configuration Types

// SigningConfig represents signing configuration for OAuth service
type SigningConfig struct {
	// Token signing certificate and key (for JWT tokens)
	SigningCertPath    string `json:"signing_cert_path"`
	SigningKeyPath     string `json:"signing_key_path"`
	SigningKeyPassword string `json:"signing_key_password,omitempty"`
	SigningAlgorithm   string `json:"signing_algorithm"` // RS256, RS384, RS512, ES256, ES384, ES512

	// Token verification certificates (for token validation)
	VerificationCerts []string `json:"verification_certs,omitempty"`

	// mTLS client certificate validation
	MTLSClientCACertPath string `json:"mtls_client_ca_cert_path,omitempty"`
	MTLSEnabled          bool   `json:"mtls_enabled"`

	// Certificate rotation settings
	CertRotationEnabled  bool          `json:"cert_rotation_enabled"`
	CertRotationInterval time.Duration `json:"cert_rotation_interval"`
}

// TokenConfig represents token-related configuration
type TokenConfig struct {
	// Access token settings
	AccessTokenLifetime   time.Duration `json:"access_token_lifetime"`    // 1h
	AccessTokenFormat     string        `json:"access_token_format"`      // jwt, opaque
	AccessTokenSigningAlg string        `json:"access_token_signing_alg"` // RS256

	// Refresh token settings
	RefreshTokenLifetime time.Duration `json:"refresh_token_lifetime"` // 24h
	RefreshTokenRotation bool          `json:"refresh_token_rotation"` // true for OAuth 2.1
	RefreshTokenFormat   string        `json:"refresh_token_format"`   // opaque, jwt

	// Authorization code settings
	AuthorizationCodeLifetime time.Duration `json:"authorization_code_lifetime"` // 10m
	AuthorizationCodeLength   int           `json:"authorization_code_length"`   // 32

	// Device code settings
	DeviceCodeLifetime time.Duration `json:"device_code_lifetime"` // 15m
	DeviceCodeLength   int           `json:"device_code_length"`   // 8
	UserCodeLength     int           `json:"user_code_length"`     // 8
	DeviceCodeInterval time.Duration `json:"device_code_interval"` // 5s

	// Token binding settings
	TokenBindingEnabled   bool     `json:"token_binding_enabled"`
	SupportedBindingTypes []string `json:"supported_binding_types"` // dpop, mtls

	// Token audience settings
	DefaultAudience        []string `json:"default_audience"`
	AudienceValidationMode string   `json:"audience_validation_mode"` // strict, relaxed
}

// SecurityConfig represents security-related configuration
type SecurityConfig struct {
	// PKCE settings (mandatory for OAuth 2.1)
	PKCERequired            bool     `json:"pkce_required"`              // true for OAuth 2.1
	PKCECodeChallengeMethod []string `json:"pkce_code_challenge_method"` // S256
	PKCECodeVerifierLength  int      `json:"pkce_code_verifier_length"`  // 128

	// State parameter settings
	StateParameterRequired bool          `json:"state_parameter_required"`
	StateParameterLifetime time.Duration `json:"state_parameter_lifetime"` // 10m
	StateParameterLength   int           `json:"state_parameter_length"`   // 32

	// Rate limiting
	RateLimitEnabled    bool          `json:"rate_limit_enabled"`
	RateLimitRequests   int           `json:"rate_limit_requests"` // requests per window
	RateLimitWindow     time.Duration `json:"rate_limit_window"`   // 1m
	RateLimitByClientID bool          `json:"rate_limit_by_client_id"`

	// Brute force protection
	BruteForceProtectionEnabled bool          `json:"brute_force_protection_enabled"`
	MaxFailedAttempts           int           `json:"max_failed_attempts"` // 5
	LockoutDuration             time.Duration `json:"lockout_duration"`    // 15m

	// Encryption settings
	EncryptionKey       string `json:"encryption_key"`       // for encrypting sensitive data
	EncryptionAlgorithm string `json:"encryption_algorithm"` // AES-256-GCM

	// Additional security features
	IPWhitelist              []string `json:"ip_whitelist,omitempty"`
	IPBlacklist              []string `json:"ip_blacklist,omitempty"`
	RequireHTTPS             bool     `json:"require_https"`
	DisableUnsecureEndpoints bool     `json:"disable_unsecure_endpoints"`
}

// ClientConfig represents default client configuration
type ClientConfig struct {
	// Default client settings
	DefaultClientType              string   `json:"default_client_type"`                // confidential, public
	DefaultTokenEndpointAuthMethod string   `json:"default_token_endpoint_auth_method"` // client_secret_basic, client_secret_post, private_key_jwt
	DefaultGrantTypes              []string `json:"default_grant_types"`                // authorization_code, refresh_token
	DefaultResponseTypes           []string `json:"default_response_types"`             // code
	DefaultScopes                  []string `json:"default_scopes"`                     // openid, profile, email

	// Client validation settings
	ClientIDLength       int           `json:"client_id_length"`       // 32
	ClientSecretLength   int           `json:"client_secret_length"`   // 64
	ClientSecretLifetime time.Duration `json:"client_secret_lifetime"` // 0 (never expires)

	// Dynamic client registration
	DynamicRegistrationEnabled bool     `json:"dynamic_registration_enabled"`
	AllowedRedirectURISchemes  []string `json:"allowed_redirect_uri_schemes"` // https, http (for dev)
	AllowedRedirectURIHosts    []string `json:"allowed_redirect_uri_hosts"`   // localhost (for dev)

	// Client certificate settings
	ClientCertificateRequired   bool   `json:"client_certificate_required"`
	ClientCertificateValidation string `json:"client_certificate_validation"` // none, optional, required
}
