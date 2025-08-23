package openapi

import (
	"github.com/yaoapp/yao/openapi/oauth"
	"github.com/yaoapp/yao/openapi/oauth/types"
)

// Config is the configuration for the OpenAPI server
type Config struct {
	BaseURL   string     `json:"baseurl" yaml:"baseurl"`
	Store     string     `json:"store,omitempty" yaml:"store,omitempty"`
	Cache     string     `json:"cache,omitempty" yaml:"cache,omitempty"`
	Providers *Providers `json:"providers,omitempty" yaml:"providers,omitempty"`
	OAuth     *OAuth     `json:"oauth,omitempty" yaml:"oauth,omitempty"`
	root      string     `json:"-" yaml:"-"` // Application root path, not serialized to JSON
}

// Provider is the provider for the OpenAPI server, and in the future will be refactored into a struct
type Provider string

// Providers is the providers for the OpenAPI server
type Providers struct {
	User   Provider `json:"user,omitempty" yaml:"user,omitempty"`
	Client Provider `json:"client,omitempty" yaml:"client,omitempty"`
}

// OAuth is the OAuth configuration for the OpenAPI server
type OAuth struct {
	IssuerURL string               `json:"issuer_url,omitempty" yaml:"issuer_url,omitempty"`
	Signing   types.SigningConfig  `json:"signing,omitempty" yaml:"signing,omitempty"`
	Token     types.TokenConfig    `json:"token,omitempty" yaml:"token,omitempty"`
	Security  types.SecurityConfig `json:"security,omitempty" yaml:"security,omitempty"`
	Client    types.ClientConfig   `json:"client,omitempty" yaml:"client,omitempty"`
	Features  oauth.FeatureFlags   `json:"features,omitempty" yaml:"features,omitempty"`
}

// Temporary config structures for JSON unmarshaling (string duration fields)
// These are used to parse human-readable duration strings from config files
// and convert them to Go time.Duration types for internal use

// TempSigningConfig represents signing configuration with string duration fields
type TempSigningConfig struct {
	SigningCertPath      string   `json:"signing_cert_path"`
	SigningKeyPath       string   `json:"signing_key_path"`
	SigningKeyPassword   string   `json:"signing_key_password,omitempty"`
	SigningAlgorithm     string   `json:"signing_algorithm"`
	VerificationCerts    []string `json:"verification_certs,omitempty"`
	MTLSClientCACertPath string   `json:"mtls_client_ca_cert_path,omitempty"`
	MTLSEnabled          bool     `json:"mtls_enabled"`
	CertRotationEnabled  bool     `json:"cert_rotation_enabled"`
	CertRotationInterval string   `json:"cert_rotation_interval"`
}

// TempTokenConfig represents token configuration with string duration fields
type TempTokenConfig struct {
	AccessTokenLifetime       string   `json:"access_token_lifetime"`
	AccessTokenFormat         string   `json:"access_token_format"`
	AccessTokenSigningAlg     string   `json:"access_token_signing_alg"`
	RefreshTokenLifetime      string   `json:"refresh_token_lifetime"`
	RefreshTokenRotation      bool     `json:"refresh_token_rotation"`
	RefreshTokenFormat        string   `json:"refresh_token_format"`
	AuthorizationCodeLifetime string   `json:"authorization_code_lifetime"`
	AuthorizationCodeLength   int      `json:"authorization_code_length"`
	DeviceCodeLifetime        string   `json:"device_code_lifetime"`
	DeviceCodeLength          int      `json:"device_code_length"`
	UserCodeLength            int      `json:"user_code_length"`
	DeviceCodeInterval        string   `json:"device_code_interval"`
	TokenBindingEnabled       bool     `json:"token_binding_enabled"`
	SupportedBindingTypes     []string `json:"supported_binding_types"`
	DefaultAudience           []string `json:"default_audience"`
	AudienceValidationMode    string   `json:"audience_validation_mode"`
}

// TempSecurityConfig represents security configuration with string duration fields
type TempSecurityConfig struct {
	PKCERequired                bool     `json:"pkce_required"`
	PKCECodeChallengeMethod     []string `json:"pkce_code_challenge_method"`
	PKCECodeVerifierLength      int      `json:"pkce_code_verifier_length"`
	StateParameterRequired      bool     `json:"state_parameter_required"`
	StateParameterLifetime      string   `json:"state_parameter_lifetime"`
	StateParameterLength        int      `json:"state_parameter_length"`
	RateLimitEnabled            bool     `json:"rate_limit_enabled"`
	RateLimitRequests           int      `json:"rate_limit_requests"`
	RateLimitWindow             string   `json:"rate_limit_window"`
	RateLimitByClientID         bool     `json:"rate_limit_by_client_id"`
	BruteForceProtectionEnabled bool     `json:"brute_force_protection_enabled"`
	MaxFailedAttempts           int      `json:"max_failed_attempts"`
	LockoutDuration             string   `json:"lockout_duration"`
	EncryptionKey               string   `json:"encryption_key"`
	EncryptionAlgorithm         string   `json:"encryption_algorithm"`
	IPWhitelist                 []string `json:"ip_whitelist,omitempty"`
	IPBlacklist                 []string `json:"ip_blacklist,omitempty"`
	RequireHTTPS                bool     `json:"require_https"`
	DisableUnsecureEndpoints    bool     `json:"disable_unsecure_endpoints"`
}

// TempClientConfig represents client configuration with string duration fields
type TempClientConfig struct {
	DefaultClientType              string   `json:"default_client_type"`
	DefaultTokenEndpointAuthMethod string   `json:"default_token_endpoint_auth_method"`
	DefaultGrantTypes              []string `json:"default_grant_types"`
	DefaultResponseTypes           []string `json:"default_response_types"`
	DefaultScopes                  []string `json:"default_scopes"`
	ClientIDLength                 int      `json:"client_id_length"`
	ClientSecretLength             int      `json:"client_secret_length"`
	ClientSecretLifetime           string   `json:"client_secret_lifetime"`
	DynamicRegistrationEnabled     bool     `json:"dynamic_registration_enabled"`
	AllowedRedirectURISchemes      []string `json:"allowed_redirect_uri_schemes"`
	AllowedRedirectURIHosts        []string `json:"allowed_redirect_uri_hosts"`
	ClientCertificateRequired      bool     `json:"client_certificate_required"`
	ClientCertificateValidation    string   `json:"client_certificate_validation"`
}

// TempOAuth represents OAuth configuration with string duration fields
type TempOAuth struct {
	IssuerURL string             `json:"issuer_url,omitempty"`
	Signing   TempSigningConfig  `json:"signing,omitempty"`
	Token     TempTokenConfig    `json:"token,omitempty"`
	Security  TempSecurityConfig `json:"security,omitempty"`
	Client    TempClientConfig   `json:"client,omitempty"`
	Features  oauth.FeatureFlags `json:"features,omitempty"`
}

// TempConfig represents the full config structure with string duration fields
type TempConfig struct {
	BaseURL   string     `json:"baseurl"`
	Store     string     `json:"store,omitempty"`
	Cache     string     `json:"cache,omitempty"`
	Providers *Providers `json:"providers,omitempty"`
	OAuth     *TempOAuth `json:"oauth,omitempty"`
}
