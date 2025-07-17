package oauth

import (
	"context"
	"time"

	"github.com/yaoapp/gou/store"
)

// Service OAuth service
type Service struct {
	config       *Config
	store        store.Store
	userProvider UserProvider
}

// Config OAuth service configuration
type Config struct {
	// Core storage interface
	Store store.Store `json:"-"`

	// User provider interface
	UserProvider UserProvider `json:"-"`

	// Certificate and key management
	Signing SigningConfig `json:"signing"`

	// Token management settings
	Token TokenConfig `json:"token"`

	// Security configuration
	Security SecurityConfig `json:"security"`

	// Default client settings
	Client ClientConfig `json:"client"`

	// Feature flags
	Features FeatureFlags `json:"features"`

	// OAuth server metadata
	IssuerURL string `json:"issuer_url"` // JWT token issuer URL
}

// FeatureFlags represents feature toggle configuration
type FeatureFlags struct {
	// OAuth 2.1 features
	OAuth21Enabled              bool `json:"oauth21_enabled"`
	PKCEEnforced                bool `json:"pkce_enforced"`
	RefreshTokenRotationEnabled bool `json:"refresh_token_rotation_enabled"`

	// Advanced features
	DeviceFlowEnabled                bool `json:"device_flow_enabled"`
	TokenExchangeEnabled             bool `json:"token_exchange_enabled"`
	PushedAuthorizationEnabled       bool `json:"pushed_authorization_enabled"`
	DynamicClientRegistrationEnabled bool `json:"dynamic_client_registration_enabled"`

	// MCP features
	MCPComplianceEnabled     bool `json:"mcp_compliance_enabled"`
	ResourceParameterEnabled bool `json:"resource_parameter_enabled"`

	// Security features
	TokenBindingEnabled bool `json:"token_binding_enabled"`
	MTLSEnabled         bool `json:"mtls_enabled"`
	DPoPEnabled         bool `json:"dpop_enabled"`

	// Experimental features
	JWTIntrospectionEnabled bool `json:"jwt_introspection_enabled"`
	TokenRevocationEnabled  bool `json:"token_revocation_enabled"`
	UserInfoJWTEnabled      bool `json:"userinfo_jwt_enabled"`
}

// NewService creates a new OAuth service with the given configuration
func NewService(config *Config) (*Service, error) {
	if config == nil {
		return nil, ErrInvalidConfiguration
	}

	// Set default values if not provided
	if err := setConfigDefaults(config); err != nil {
		return nil, err
	}

	// Validate configuration
	if err := validateConfig(config); err != nil {
		return nil, err
	}

	// Use UserProvider from config, or create a default one if not provided
	userProvider := config.UserProvider
	if userProvider == nil {
		userProvider = NewDefaultUserProvider(nil, nil, nil)
	}

	service := &Service{
		config:       config,
		store:        config.Store,
		userProvider: userProvider,
	}

	return service, nil
}

// GetConfig returns the service configuration
func (s *Service) GetConfig() *Config {
	return s.config
}

// GetUserProvider returns the user provider for the service
func (s *Service) GetUserProvider() UserProvider {
	return s.userProvider
}

// setConfigDefaults sets default values for configuration
func setConfigDefaults(config *Config) error {
	// Certificate defaults
	if config.Signing.SigningAlgorithm == "" {
		config.Signing.SigningAlgorithm = "RS256"
	}

	// Token defaults
	if config.Token.AccessTokenLifetime == 0 {
		config.Token.AccessTokenLifetime = time.Hour
	}
	if config.Token.RefreshTokenLifetime == 0 {
		config.Token.RefreshTokenLifetime = 24 * time.Hour
	}
	if config.Token.AuthorizationCodeLifetime == 0 {
		config.Token.AuthorizationCodeLifetime = 10 * time.Minute
	}
	if config.Token.DeviceCodeLifetime == 0 {
		config.Token.DeviceCodeLifetime = 15 * time.Minute
	}
	if config.Token.AccessTokenFormat == "" {
		config.Token.AccessTokenFormat = "jwt"
	}
	if config.Token.RefreshTokenFormat == "" {
		config.Token.RefreshTokenFormat = "opaque"
	}

	// Security defaults
	if len(config.Security.PKCECodeChallengeMethod) == 0 {
		config.Security.PKCECodeChallengeMethod = []string{"S256"}
	}
	if config.Security.PKCECodeVerifierLength == 0 {
		config.Security.PKCECodeVerifierLength = 128
	}
	if config.Security.StateParameterLifetime == 0 {
		config.Security.StateParameterLifetime = 10 * time.Minute
	}
	if config.Security.StateParameterLength == 0 {
		config.Security.StateParameterLength = 32
	}

	// Client defaults
	if config.Client.DefaultClientType == "" {
		config.Client.DefaultClientType = "confidential"
	}
	if config.Client.DefaultTokenEndpointAuthMethod == "" {
		config.Client.DefaultTokenEndpointAuthMethod = "client_secret_basic"
	}
	if len(config.Client.DefaultGrantTypes) == 0 {
		config.Client.DefaultGrantTypes = []string{"authorization_code", "refresh_token"}
	}
	if len(config.Client.DefaultResponseTypes) == 0 {
		config.Client.DefaultResponseTypes = []string{"code"}
	}
	if config.Client.ClientIDLength == 0 {
		config.Client.ClientIDLength = 32
	}
	if config.Client.ClientSecretLength == 0 {
		config.Client.ClientSecretLength = 64
	}

	// Feature flags defaults - enable OAuth 2.1 features by default
	config.Features.OAuth21Enabled = true
	config.Features.PKCEEnforced = true
	config.Features.RefreshTokenRotationEnabled = true

	return nil
}

// validateConfig validates the configuration
func validateConfig(config *Config) error {
	if config.Store == nil {
		return ErrStoreMissing
	}

	// Validate issuer URL
	if config.IssuerURL == "" {
		return ErrIssuerURLMissing
	}

	// Validate certificate configuration
	if config.Signing.SigningCertPath == "" || config.Signing.SigningKeyPath == "" {
		return ErrCertificateMissing
	}

	// Validate token configuration
	if config.Token.AccessTokenLifetime <= 0 {
		return ErrInvalidTokenLifetime
	}

	// Validate security configuration
	if config.Security.PKCERequired && len(config.Security.PKCECodeChallengeMethod) == 0 {
		return ErrPKCEConfigurationInvalid
	}

	return nil
}

// Error definitions
var (
	ErrInvalidConfiguration     = &ErrorResponse{Code: "invalid_configuration", ErrorDescription: "Invalid OAuth service configuration"}
	ErrStoreMissing             = &ErrorResponse{Code: "store_missing", ErrorDescription: "Store is required for OAuth service"}
	ErrIssuerURLMissing         = &ErrorResponse{Code: "issuer_url_missing", ErrorDescription: "Issuer URL is required for OAuth service"}
	ErrCertificateMissing       = &ErrorResponse{Code: "certificate_missing", ErrorDescription: "JWT signing certificate and key are required"}
	ErrInvalidTokenLifetime     = &ErrorResponse{Code: "invalid_token_lifetime", ErrorDescription: "Token lifetime must be greater than 0"}
	ErrPKCEConfigurationInvalid = &ErrorResponse{Code: "pkce_configuration_invalid", ErrorDescription: "PKCE configuration is invalid"}
)

// AuthorizationServer returns the authorization server endpoint URL
func (s *Service) AuthorizationServer(ctx context.Context) string {
	return s.config.IssuerURL
}

// ProtectedResource returns the protected resource endpoint URL
func (s *Service) ProtectedResource(ctx context.Context) string {
	return s.config.IssuerURL
}

// UserInfo returns user information for a given access token
func (s *Service) UserInfo(ctx context.Context, accessToken string) (interface{}, error) {
	return s.userProvider.GetUserByAccessToken(ctx, accessToken)
}
