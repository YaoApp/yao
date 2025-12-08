package types

import (
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// LoginContext represents the context information for login
type LoginContext struct {
	IP         string `json:"ip,omitempty"`          // Client IP address
	UserAgent  string `json:"user_agent,omitempty"`  // Client user agent
	Device     string `json:"device,omitempty"`      // Device type (e.g., "mobile", "desktop", "tablet")
	Platform   string `json:"platform,omitempty"`    // Platform (e.g., "ios", "android", "web")
	Location   string `json:"location,omitempty"`    // Geographic location (optional)
	RememberMe bool   `json:"remember_me,omitempty"` // Remember Me flag for extended session
}

// MFAOptions contains configuration for MFA operations
type MFAOptions struct {
	Issuer         string // Issuer name displayed in authenticator app
	Algorithm      string // TOTP algorithm: "SHA1", "SHA256", "SHA512"
	Digits         int    // Number of digits in TOTP code (6 or 8)
	Period         int    // TOTP time period in seconds (usually 30)
	SecretSize     int    // Secret key size in bytes (usually 32)
	RecoveryCount  int    // Number of recovery codes to generate
	RecoveryLength int    // Length of each recovery code
	AccountName    string // Optional account name (defaults to userID)
}

// ErrorResponse represents an OAuth 2.1 error response
type ErrorResponse struct {
	Code             string `json:"error"`
	ErrorDescription string `json:"error_description,omitempty"`
	ErrorURI         string `json:"error_uri,omitempty"`
	State            string `json:"state,omitempty"`

	// Extended fields for ACL and permission errors (optional, following OAuth 2.0 extensibility)
	Reason         string   `json:"reason,omitempty"`          // Detailed reason for denial
	RequiredScopes []string `json:"required_scopes,omitempty"` // Required scopes for access
	MissingScopes  []string `json:"missing_scopes,omitempty"`  // Scopes that are missing
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
	ResponseTypeCode    = "code"
	ResponseTypeToken   = "token"
	ResponseTypeIDToken = "id_token"
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

// User Status Constants
const (
	UserStatusPending         = "pending"
	UserStatusActive          = "active"
	UserStatusDisabled        = "disabled"
	UserStatusSuspended       = "suspended"
	UserStatusLocked          = "locked"
	UserStatusPasswordExpired = "password_expired"
	UserStatusEmailUnverified = "email_unverified"
	UserStatusArchived        = "archived"
)

// MFA Algorithm Constants
const (
	MFAAlgorithmSHA1   = "SHA1"
	MFAAlgorithmSHA256 = "SHA256"
	MFAAlgorithmSHA512 = "SHA512"
)

// OAuth Provider Constants
const (
	ProviderLocal     = "local"
	ProviderGoogle    = "google"
	ProviderApple     = "apple"
	ProviderGitHub    = "github"
	ProviderMicrosoft = "microsoft"
	ProviderWeChat    = "wechat"
	ProviderGeneric   = "generic"
)

// User Identifier Types
const (
	IdentifierTypeUserID            = "user_id"
	IdentifierTypeSubject           = "subject"
	IdentifierTypePreferredUsername = "preferred_username"
	IdentifierTypeEmail             = "email"
	IdentifierTypePhoneNumber       = "phone_number"
)

// Login Methods
const (
	LoginMethodPassword = "password"
	LoginMethodOAuth    = "oauth"
	LoginMethodMFA      = "mfa"
	LoginMethodRecovery = "recovery"
	LoginMethodSSO      = "sso"
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
	ClientID                    string   `json:"client_id,omitempty"` // Optional: Client ID to use for registration, if not provided, a new client ID will be generated
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
	// Token signing certificate and key (for JWT and opaque tokens)
	SigningCertPath    string `json:"signing_cert_path"`              // Required: Path to token signing certificate (public key)
	SigningKeyPath     string `json:"signing_key_path"`               // Required: Path to token signing private key
	SigningKeyPassword string `json:"signing_key_password,omitempty"` // Optional: Password for encrypted private key
	SigningAlgorithm   string `json:"signing_algorithm"`              // Optional: Token signing algorithm (default: RS256)

	// Token verification certificates (for token validation)
	VerificationCerts []string `json:"verification_certs,omitempty"` // Optional: Additional certificates for token verification

	// mTLS client certificate validation
	MTLSClientCACertPath string `json:"mtls_client_ca_cert_path,omitempty"` // Optional: CA certificate path for mTLS client validation
	MTLSEnabled          bool   `json:"mtls_enabled"`                       // Optional: Enable mutual TLS authentication (default: false)

	// Certificate rotation settings
	CertRotationEnabled  bool          `json:"cert_rotation_enabled"`  // Optional: Enable automatic certificate rotation (default: false)
	CertRotationInterval time.Duration `json:"cert_rotation_interval"` // Optional: Certificate rotation interval (default: 24h)
}

// TokenConfig represents token-related configuration
type TokenConfig struct {
	// Access token settings
	AccessTokenLifetime   time.Duration `json:"access_token_lifetime"`    // Optional: Access token validity period (default: 1h)
	AccessTokenFormat     string        `json:"access_token_format"`      // Optional: Access token format - jwt, opaque (default: jwt)
	AccessTokenSigningAlg string        `json:"access_token_signing_alg"` // Optional: Access token signing algorithm (default: RS256)

	// Refresh token settings
	RefreshTokenLifetime time.Duration `json:"refresh_token_lifetime"` // Optional: Refresh token validity period (default: 24h)
	RefreshTokenRotation bool          `json:"refresh_token_rotation"` // Optional: Enable refresh token rotation for OAuth 2.1 (default: true)
	RefreshTokenFormat   string        `json:"refresh_token_format"`   // Optional: Refresh token format - opaque, jwt (default: opaque)

	// Authorization code settings
	AuthorizationCodeLifetime time.Duration `json:"authorization_code_lifetime"` // Optional: Authorization code validity period (default: 10m)
	AuthorizationCodeLength   int           `json:"authorization_code_length"`   // Optional: Authorization code length in bytes (default: 32)

	// Device code settings
	DeviceCodeLifetime time.Duration `json:"device_code_lifetime"` // Optional: Device code validity period (default: 15m)
	DeviceCodeLength   int           `json:"device_code_length"`   // Optional: Device code length in bytes (default: 8)
	UserCodeLength     int           `json:"user_code_length"`     // Optional: User code length for device flow (default: 8)
	DeviceCodeInterval time.Duration `json:"device_code_interval"` // Optional: Device code polling interval (default: 5s)

	// Token binding settings
	TokenBindingEnabled   bool     `json:"token_binding_enabled"`   // Optional: Enable token binding to client certificates (default: false)
	SupportedBindingTypes []string `json:"supported_binding_types"` // Optional: Supported token binding types - dpop, mtls (default: [dpop, mtls])

	// Token audience settings
	DefaultAudience        []string `json:"default_audience"`         // Optional: Default token audience (default: [])
	AudienceValidationMode string   `json:"audience_validation_mode"` // Optional: Audience validation mode - strict, relaxed (default: strict)
}

// SecurityConfig represents security-related configuration
type SecurityConfig struct {
	// PKCE settings (mandatory for OAuth 2.1)
	PKCERequired            bool     `json:"pkce_required"`              // Optional: Require PKCE for OAuth 2.1 compliance (default: true)
	PKCECodeChallengeMethod []string `json:"pkce_code_challenge_method"` // Optional: Supported PKCE code challenge methods (default: [S256])
	PKCECodeVerifierLength  int      `json:"pkce_code_verifier_length"`  // Optional: PKCE code verifier length (default: 128)

	// State parameter settings
	StateParameterRequired bool          `json:"state_parameter_required"` // Optional: Require state parameter for CSRF protection (default: false)
	StateParameterLifetime time.Duration `json:"state_parameter_lifetime"` // Optional: State parameter validity period (default: 10m)
	StateParameterLength   int           `json:"state_parameter_length"`   // Optional: State parameter length in bytes (default: 32)

	// Rate limiting
	RateLimitEnabled    bool          `json:"rate_limit_enabled"`      // Optional: Enable rate limiting (default: false)
	RateLimitRequests   int           `json:"rate_limit_requests"`     // Optional: Number of requests per window (default: 100)
	RateLimitWindow     time.Duration `json:"rate_limit_window"`       // Optional: Rate limit time window (default: 1m)
	RateLimitByClientID bool          `json:"rate_limit_by_client_id"` // Optional: Enable per-client rate limiting (default: false)

	// Brute force protection
	BruteForceProtectionEnabled bool          `json:"brute_force_protection_enabled"` // Optional: Enable brute force attack protection (default: false)
	MaxFailedAttempts           int           `json:"max_failed_attempts"`            // Optional: Maximum failed login attempts (default: 5)
	LockoutDuration             time.Duration `json:"lockout_duration"`               // Optional: Account lockout duration (default: 15m)

	// Encryption settings
	EncryptionKey       string `json:"encryption_key"`       // Optional: Key for encrypting sensitive data (default: "")
	EncryptionAlgorithm string `json:"encryption_algorithm"` // Optional: Encryption algorithm for sensitive data (default: AES-256-GCM)

	// Additional security features
	IPWhitelist              []string `json:"ip_whitelist,omitempty"`     // Optional: IP addresses allowed to access (default: [])
	IPBlacklist              []string `json:"ip_blacklist,omitempty"`     // Optional: IP addresses blocked from access (default: [])
	RequireHTTPS             bool     `json:"require_https"`              // Optional: Require HTTPS for all endpoints (default: true)
	DisableUnsecureEndpoints bool     `json:"disable_unsecure_endpoints"` // Optional: Disable non-HTTPS endpoints (default: false)
}

// TokenClaims represents decoded token claims for both JWT and opaque tokens
type TokenClaims struct {
	Subject   string    `json:"sub,omitempty"`   // Subject identifier
	ClientID  string    `json:"client_id"`       // OAuth client ID
	Scope     string    `json:"scope,omitempty"` // Access scope
	TokenType string    `json:"token_type"`      // Token type (access_token, refresh_token, etc.)
	ExpiresAt time.Time `json:"exp,omitempty"`   // Expiration time
	IssuedAt  time.Time `json:"iat,omitempty"`   // Issued at time
	Issuer    string    `json:"iss,omitempty"`   // Token issuer
	Audience  []string  `json:"aud,omitempty"`   // Token audience
	JTI       string    `json:"jti,omitempty"`   // JWT ID (for JWT tokens)

	// Extended claims for multi-tenancy and team support
	TeamID   string `json:"team_id,omitempty"`   // Team identifier
	TenantID string `json:"tenant_id,omitempty"` // Tenant identifier

	// Extra claims for flexibility
	Extra map[string]interface{} `json:"-"` // Additional custom claims (not serialized directly)
}

// DataConstraints represents data access constraints
// These constraints are set by ACL enforcement and used by API handlers to filter data
type DataConstraints struct {
	// Built-in constraints
	OwnerOnly   bool `json:"owner_only,omitempty"`   // Only access owner's data (current owner)
	CreatorOnly bool `json:"creator_only,omitempty"` // Only access creator's data (who created)
	EditorOnly  bool `json:"editor_only,omitempty"`  // Only access editor's data (who last updated)
	TeamOnly    bool `json:"team_only,omitempty"`    // Only access team's data (filter by TeamID)

	// Extra constraints (user-defined, flexible extension)
	// Examples: department_only, region_only, project_only
	Extra map[string]interface{} `json:"extra,omitempty"` // Extra constraints
}

// AuthorizedInfo represents authorized information
type AuthorizedInfo struct {
	Subject   string `json:"sub,omitempty"`        // Subject identifier
	ClientID  string `json:"client_id"`            // OAuth client ID
	Scope     string `json:"scope,omitempty"`      // Access scope
	SessionID string `json:"session_id,omitempty"` // Session ID
	UserID    string `json:"user_id,omitempty"`    // User ID

	// Extended fields for multi-tenancy and team support
	TeamID     string `json:"team_id,omitempty"`     // Team identifier
	TenantID   string `json:"tenant_id,omitempty"`   // Tenant identifier
	RememberMe bool   `json:"remember_me,omitempty"` // Remember Me flag preserved from login

	// Data access constraints (set by ACL enforcement)
	Constraints DataConstraints `json:"constraints,omitempty"`
}

// AuthorizedToMap converts AuthorizedInfo to map[string]interface{}
// This is useful for passing authorized information to runtime bridges (e.g., V8)
func (auth *AuthorizedInfo) AuthorizedToMap() map[string]interface{} {
	if auth == nil {
		return nil
	}

	result := make(map[string]interface{})

	if auth.Subject != "" {
		result["sub"] = auth.Subject
	}
	if auth.ClientID != "" {
		result["client_id"] = auth.ClientID
	}
	if auth.Scope != "" {
		result["scope"] = auth.Scope
	}
	if auth.SessionID != "" {
		result["session_id"] = auth.SessionID
	}
	if auth.UserID != "" {
		result["user_id"] = auth.UserID
	}
	if auth.TeamID != "" {
		result["team_id"] = auth.TeamID
	}
	if auth.TenantID != "" {
		result["tenant_id"] = auth.TenantID
	}
	if auth.RememberMe {
		result["remember_me"] = auth.RememberMe
	}

	// Add constraints if any are set
	if auth.Constraints.OwnerOnly || auth.Constraints.CreatorOnly || auth.Constraints.EditorOnly || auth.Constraints.TeamOnly || len(auth.Constraints.Extra) > 0 {
		constraints := make(map[string]interface{})
		if auth.Constraints.OwnerOnly {
			constraints["owner_only"] = true
		}
		if auth.Constraints.CreatorOnly {
			constraints["creator_only"] = true
		}
		if auth.Constraints.EditorOnly {
			constraints["editor_only"] = true
		}
		if auth.Constraints.TeamOnly {
			constraints["team_only"] = true
		}
		if len(auth.Constraints.Extra) > 0 {
			constraints["extra"] = auth.Constraints.Extra
		}
		result["constraints"] = constraints
	}

	return result
}

// JWTClaims represents JWT-specific claims structure
type JWTClaims struct {
	jwt.StandardClaims
	ClientID  string `json:"client_id"`       // OAuth client ID
	Scope     string `json:"scope,omitempty"` // Access scope
	TokenType string `json:"token_type"`      // Token type

	// Extended claims for multi-tenancy and team support
	TeamID   string `json:"team_id,omitempty"`   // Team identifier
	TenantID string `json:"tenant_id,omitempty"` // Tenant identifier
}

// ClientConfig represents default client configuration
type ClientConfig struct {
	// Default client settings
	DefaultClientType              string   `json:"default_client_type"`                // Optional: Default client type - confidential, public (default: confidential)
	DefaultTokenEndpointAuthMethod string   `json:"default_token_endpoint_auth_method"` // Optional: Default client authentication method (default: client_secret_basic)
	DefaultGrantTypes              []string `json:"default_grant_types"`                // Optional: Default supported grant types (default: [authorization_code, refresh_token])
	DefaultResponseTypes           []string `json:"default_response_types"`             // Optional: Default supported response types (default: [code])
	DefaultScopes                  []string `json:"default_scopes"`                     // Optional: Default OAuth scopes (default: [openid, profile, email])

	// Client validation settings
	ClientIDLength       int           `json:"client_id_length"`       // Optional: Client ID length in bytes (default: 32)
	ClientSecretLength   int           `json:"client_secret_length"`   // Optional: Client secret length in bytes (default: 64)
	ClientSecretLifetime time.Duration `json:"client_secret_lifetime"` // Optional: Client secret lifetime, 0 = never expires (default: 0s)

	// Dynamic client registration
	DynamicRegistrationEnabled bool     `json:"dynamic_registration_enabled"` // Optional: Enable dynamic client registration (default: true)
	AllowedRedirectURISchemes  []string `json:"allowed_redirect_uri_schemes"` // Optional: Allowed redirect URI schemes (default: [https, http])
	AllowedRedirectURIHosts    []string `json:"allowed_redirect_uri_hosts"`   // Optional: Allowed redirect URI hosts (default: [localhost, 127.0.0.1])

	// Client certificate settings
	ClientCertificateRequired   bool   `json:"client_certificate_required"`   // Optional: Require client certificates (default: false)
	ClientCertificateValidation string `json:"client_certificate_validation"` // Optional: Client certificate validation mode - none, optional, required (default: none)
}

// OIDC Standard Types

// OIDCIDToken represents ID Token claims based on OIDC standard
// https://openid.net/specs/openid-connect-core-1_0.html#IDToken
type OIDCIDToken struct {
	// REQUIRED ID Token Claims
	Iss string `json:"iss"` // Issuer Identifier for the Issuer of the response
	Sub string `json:"sub"` // Subject Identifier - locally unique identifier for the End-User
	Aud string `json:"aud"` // Audience - OAuth 2.0 client_id of the Relying Party
	Exp int64  `json:"exp"` // Expiration time - seconds from 1970-01-01T00:00:00Z UTC
	Iat int64  `json:"iat"` // Issued at time - seconds from 1970-01-01T00:00:00Z UTC

	// OPTIONAL ID Token Claims
	AuthTime *int64   `json:"auth_time,omitempty"` // Time when End-User authentication occurred
	Nonce    string   `json:"nonce,omitempty"`     // String value to associate Client session with ID Token
	Acr      string   `json:"acr,omitempty"`       // Authentication Context Class Reference
	Amr      []string `json:"amr,omitempty"`       // Authentication Methods References
	Azp      string   `json:"azp,omitempty"`       // Authorized party - party to which ID Token was issued

	// Hash Claims for token validation
	AtHash string `json:"at_hash,omitempty"` // Access Token hash value
	CHash  string `json:"c_hash,omitempty"`  // Code hash value
}

// OIDCUserInfo represents user information based on OIDC standard
type OIDCUserInfo struct {
	// OIDC Standard Claims (https://openid.net/specs/openid-connect-core-1_0.html#StandardClaims)
	Sub                 string `json:"sub"`                             // Subject identifier (required)
	Name                string `json:"name,omitempty"`                  // Full name
	GivenName           string `json:"given_name,omitempty"`            // Given name(s) or first name(s)
	FamilyName          string `json:"family_name,omitempty"`           // Surname(s) or last name(s)
	MiddleName          string `json:"middle_name,omitempty"`           // Middle name(s)
	Nickname            string `json:"nickname,omitempty"`              // Casual name
	PreferredUsername   string `json:"preferred_username,omitempty"`    // Shorthand name
	Profile             string `json:"profile,omitempty"`               // Profile page URL
	Picture             string `json:"picture,omitempty"`               // Profile picture URL
	Website             string `json:"website,omitempty"`               // Web page or blog URL
	Email               string `json:"email,omitempty"`                 // Email address
	EmailVerified       *bool  `json:"email_verified,omitempty"`        // Email verification status
	Gender              string `json:"gender,omitempty"`                // Gender
	Birthdate           string `json:"birthdate,omitempty"`             // Birthday (YYYY-MM-DD format)
	Zoneinfo            string `json:"zoneinfo,omitempty"`              // Time zone info
	Locale              string `json:"locale,omitempty"`                // Locale (language-country)
	PhoneNumber         string `json:"phone_number,omitempty"`          // Phone number
	PhoneNumberVerified *bool  `json:"phone_number_verified,omitempty"` // Phone verification status
	UpdatedAt           *int64 `json:"updated_at,omitempty"`            // Time of last update (seconds since epoch)

	// OIDC Address Claim (structured)
	Address *OIDCAddress `json:"address,omitempty"` // Physical mailing address

	// Additional custom claims with namespace
	YaoUserID   string          `json:"yao:user_id,omitempty"`   // Yao user ID (original user ID)
	YaoTenantID string          `json:"yao:tenant_id,omitempty"` // Yao tenant ID
	YaoTeamID   string          `json:"yao:team_id,omitempty"`   // Yao team ID
	YaoTeam     *OIDCTeamInfo   `json:"yao:team,omitempty"`      // Yao team info
	YaoIsOwner  *bool           `json:"yao:is_owner,omitempty"`  // Yao is owner
	YaoTypeID   string          `json:"yao:type_id,omitempty"`   // Yao user type ID
	YaoType     *OIDCTypeInfo   `json:"yao:type,omitempty"`      // Yao user type info
	YaoMember   *OIDCMemberInfo `json:"yao:member,omitempty"`    // Yao member profile info (for team context)

	// Raw response for debugging and custom processing
	Raw map[string]interface{} `json:"raw,omitempty"` // Original provider response
}

// OIDCTeamInfo represents team information based on OIDC standard
type OIDCTeamInfo struct {
	TeamID      string `json:"team_id,omitempty"`     // Team identifier
	Logo        string `json:"logo,omitempty"`        // Team logo
	Name        string `json:"name,omitempty"`        // Team name
	OwnerID     string `json:"owner_id,omitempty"`    // Team owner ID
	Description string `json:"description,omitempty"` // Team description
	UpdatedAt   *int64 `json:"updated_at,omitempty"`  // Team updated at (seconds since epoch)
}

// OIDCTypeInfo represents user type information based on OIDC standard
type OIDCTypeInfo struct {
	TypeID string `json:"type_id,omitempty"` // User type identifier
	Name   string `json:"name,omitempty"`    // User type name
	Locale string `json:"locale,omitempty"`  // User type locale
}

// OIDCMemberInfo represents team member profile information
type OIDCMemberInfo struct {
	MemberID    string `json:"member_id,omitempty"`    // Member's unique identifier in team
	DisplayName string `json:"display_name,omitempty"` // Member's display name in team
	Bio         string `json:"bio,omitempty"`          // Member's bio in team
	Avatar      string `json:"avatar,omitempty"`       // Member's avatar in team
	Email       string `json:"email,omitempty"`        // Member's email in team
}

// OIDCAddress represents the OIDC address claim structure
type OIDCAddress struct {
	Formatted     string `json:"formatted,omitempty"`      // Full mailing address
	StreetAddress string `json:"street_address,omitempty"` // Street address
	Locality      string `json:"locality,omitempty"`       // City or locality
	Region        string `json:"region,omitempty"`         // State, province, prefecture, or region
	PostalCode    string `json:"postal_code,omitempty"`    // Zip code or postal code
	Country       string `json:"country,omitempty"`        // Country name
}
