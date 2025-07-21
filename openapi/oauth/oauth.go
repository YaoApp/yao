package oauth

import (
	"fmt"
	"time"

	"github.com/yaoapp/gou/store"
	"github.com/yaoapp/yao/openapi/oauth/providers/client"
	"github.com/yaoapp/yao/openapi/oauth/providers/user"
	"github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/share"
)

// Service OAuth service
type Service struct {
	config         *Config
	store          store.Store
	cache          store.Store
	userProvider   types.UserProvider
	clientProvider types.ClientProvider
	prefix         string
	// Signing certificates for JWT token signing and verification
	signingCerts *SigningCertificates
}

// Config OAuth service configuration
type Config struct {
	// Core storage interface
	Store store.Store `json:"-"`

	// Cache store
	Cache store.Store `json:"-"`

	// User provider interface
	UserProvider types.UserProvider `json:"-"`

	// Client provider interface
	ClientProvider types.ClientProvider `json:"-"`

	// Certificate and key management
	Signing types.SigningConfig `json:"signing"`

	// Token management settings
	Token types.TokenConfig `json:"token"`

	// Security configuration
	Security types.SecurityConfig `json:"security"`

	// Default client settings
	Client types.ClientConfig `json:"client"`

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
		return nil, types.ErrInvalidConfiguration
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
	keyPrefix := fmt.Sprintf("%s:", share.App.Prefix)
	userProvider := config.UserProvider
	if userProvider == nil {
		userProvider = user.NewDefaultUser(&user.DefaultUserOptions{
			Prefix:     keyPrefix,
			Model:      "__yao.user",
			Cache:      config.Cache,
			TokenStore: config.Store,
		})
	}

	// Use ClientProvider from config, or create a default one if not provided
	clientProvider := config.ClientProvider
	if clientProvider == nil {
		var err error
		clientProvider, err = client.NewDefaultClient(&client.DefaultClientOptions{
			Prefix: keyPrefix,
			Store:  config.Store,
			Cache:  config.Cache,
		})
		if err != nil {
			return nil, err
		}
	}

	// Load Certificates
	signingCerts, err := LoadSigningCertificates(&config.Signing)
	if err != nil {
		return nil, fmt.Errorf("failed to load signing certificates: %w", err)
	}

	// Validate the loaded certificates
	if err := signingCerts.ValidateCertificate(); err != nil {
		return nil, fmt.Errorf("certificate validation failed: %w", err)
	}

	service := &Service{
		config:         config,
		store:          config.Store,
		cache:          config.Cache,
		userProvider:   userProvider,
		clientProvider: clientProvider,
		prefix:         keyPrefix,
		signingCerts:   signingCerts,
	}

	return service, nil
}

// GetConfig returns the service configuration
func (s *Service) GetConfig() *Config {
	return s.config
}

// GetUserProvider returns the user provider for the service
func (s *Service) GetUserProvider() types.UserProvider {
	return s.userProvider
}

// GetClientProvider returns the client provider for the service
func (s *Service) GetClientProvider() types.ClientProvider {
	return s.clientProvider
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
		return types.ErrStoreMissing
	}

	// Validate issuer URL
	if config.IssuerURL == "" {
		return types.ErrIssuerURLMissing
	}

	// Certificate configuration validation
	// If both cert and key paths are provided, they must both exist or be empty
	certPathProvided := config.Signing.SigningCertPath != ""
	keyPathProvided := config.Signing.SigningKeyPath != ""

	if certPathProvided != keyPathProvided {
		return types.ErrCertificateMissing // Both paths must be provided together or not at all
	}

	// If paths are not provided, temporary certificates will be generated automatically

	// Validate token configuration
	if config.Token.AccessTokenLifetime <= 0 {
		return types.ErrInvalidTokenLifetime
	}

	// Validate security configuration
	if config.Security.PKCERequired && len(config.Security.PKCECodeChallengeMethod) == 0 {
		return types.ErrPKCEConfigurationInvalid
	}

	return nil
}
