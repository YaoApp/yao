package openapi

import (
	"errors"
	// "fmt"
	"path/filepath"
	"strings"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/store"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/openapi/oauth"
	"github.com/yaoapp/yao/openapi/oauth/providers/client"
	"github.com/yaoapp/yao/openapi/oauth/providers/user"
	"github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/share"
)

// Validate validates the configuration
func (config *Config) Validate() error {
	if config.Providers == nil {
		return errors.New("providers is required")
	}

	if config.BaseURL == "" {
		return errors.New("baseurl is required")
	}

	return nil
}

// MarshalJSON JSON Marshaler
func (config *Config) MarshalJSON() ([]byte, error) {
	// Convert config to temporary structure with string duration fields
	tempConfig := TempConfig{
		BaseURL:   config.BaseURL,
		Store:     config.Store,
		Cache:     config.Cache,
		Providers: config.Providers,
	}

	if config.OAuth != nil {
		tempConfig.OAuth = &TempOAuth{
			IssuerURL: config.OAuth.IssuerURL,
			Features:  config.OAuth.Features,
			Signing: TempSigningConfig{
				SigningCertPath:      convertAbsoluteToRelativePath(config.OAuth.Signing.SigningCertPath, config.root),
				SigningKeyPath:       convertAbsoluteToRelativePath(config.OAuth.Signing.SigningKeyPath, config.root),
				SigningKeyPassword:   config.OAuth.Signing.SigningKeyPassword,
				SigningAlgorithm:     config.OAuth.Signing.SigningAlgorithm,
				VerificationCerts:    config.OAuth.Signing.VerificationCerts,
				MTLSClientCACertPath: convertAbsoluteToRelativePath(config.OAuth.Signing.MTLSClientCACertPath, config.root),
				MTLSEnabled:          config.OAuth.Signing.MTLSEnabled,
				CertRotationEnabled:  config.OAuth.Signing.CertRotationEnabled,
				CertRotationInterval: formatDuration(config.OAuth.Signing.CertRotationInterval),
			},
			Token: TempTokenConfig{
				AccessTokenLifetime:       formatDuration(config.OAuth.Token.AccessTokenLifetime),
				AccessTokenFormat:         config.OAuth.Token.AccessTokenFormat,
				AccessTokenSigningAlg:     config.OAuth.Token.AccessTokenSigningAlg,
				RefreshTokenLifetime:      formatDuration(config.OAuth.Token.RefreshTokenLifetime),
				RefreshTokenRotation:      config.OAuth.Token.RefreshTokenRotation,
				RefreshTokenFormat:        config.OAuth.Token.RefreshTokenFormat,
				AuthorizationCodeLifetime: formatDuration(config.OAuth.Token.AuthorizationCodeLifetime),
				AuthorizationCodeLength:   config.OAuth.Token.AuthorizationCodeLength,
				DeviceCodeLifetime:        formatDuration(config.OAuth.Token.DeviceCodeLifetime),
				DeviceCodeLength:          config.OAuth.Token.DeviceCodeLength,
				UserCodeLength:            config.OAuth.Token.UserCodeLength,
				DeviceCodeInterval:        formatDuration(config.OAuth.Token.DeviceCodeInterval),
				TokenBindingEnabled:       config.OAuth.Token.TokenBindingEnabled,
				SupportedBindingTypes:     config.OAuth.Token.SupportedBindingTypes,
				DefaultAudience:           config.OAuth.Token.DefaultAudience,
				AudienceValidationMode:    config.OAuth.Token.AudienceValidationMode,
			},
			Security: TempSecurityConfig{
				PKCERequired:                config.OAuth.Security.PKCERequired,
				PKCECodeChallengeMethod:     config.OAuth.Security.PKCECodeChallengeMethod,
				PKCECodeVerifierLength:      config.OAuth.Security.PKCECodeVerifierLength,
				StateParameterRequired:      config.OAuth.Security.StateParameterRequired,
				StateParameterLifetime:      formatDuration(config.OAuth.Security.StateParameterLifetime),
				StateParameterLength:        config.OAuth.Security.StateParameterLength,
				RateLimitEnabled:            config.OAuth.Security.RateLimitEnabled,
				RateLimitRequests:           config.OAuth.Security.RateLimitRequests,
				RateLimitWindow:             formatDuration(config.OAuth.Security.RateLimitWindow),
				RateLimitByClientID:         config.OAuth.Security.RateLimitByClientID,
				BruteForceProtectionEnabled: config.OAuth.Security.BruteForceProtectionEnabled,
				MaxFailedAttempts:           config.OAuth.Security.MaxFailedAttempts,
				LockoutDuration:             formatDuration(config.OAuth.Security.LockoutDuration),
				EncryptionKey:               config.OAuth.Security.EncryptionKey,
				EncryptionAlgorithm:         config.OAuth.Security.EncryptionAlgorithm,
				IPWhitelist:                 config.OAuth.Security.IPWhitelist,
				IPBlacklist:                 config.OAuth.Security.IPBlacklist,
				RequireHTTPS:                config.OAuth.Security.RequireHTTPS,
				DisableUnsecureEndpoints:    config.OAuth.Security.DisableUnsecureEndpoints,
			},
			Client: TempClientConfig{
				DefaultClientType:              config.OAuth.Client.DefaultClientType,
				DefaultTokenEndpointAuthMethod: config.OAuth.Client.DefaultTokenEndpointAuthMethod,
				DefaultGrantTypes:              config.OAuth.Client.DefaultGrantTypes,
				DefaultResponseTypes:           config.OAuth.Client.DefaultResponseTypes,
				DefaultScopes:                  config.OAuth.Client.DefaultScopes,
				ClientIDLength:                 config.OAuth.Client.ClientIDLength,
				ClientSecretLength:             config.OAuth.Client.ClientSecretLength,
				ClientSecretLifetime:           formatDuration(config.OAuth.Client.ClientSecretLifetime),
				DynamicRegistrationEnabled:     config.OAuth.Client.DynamicRegistrationEnabled,
				AllowedRedirectURISchemes:      config.OAuth.Client.AllowedRedirectURISchemes,
				AllowedRedirectURIHosts:        config.OAuth.Client.AllowedRedirectURIHosts,
				ClientCertificateRequired:      config.OAuth.Client.ClientCertificateRequired,
				ClientCertificateValidation:    config.OAuth.Client.ClientCertificateValidation,
			},
		}
	}

	return jsoniter.Marshal(tempConfig)
}

// UnmarshalJSON JSON Unmarshaler
func (config *Config) UnmarshalJSON(data []byte) error {
	var tempConfig TempConfig
	err := jsoniter.Unmarshal(data, &tempConfig)
	if err != nil {
		return err
	}

	// Convert temporary config to final config
	config.BaseURL = tempConfig.BaseURL
	config.Store = tempConfig.Store
	config.Cache = tempConfig.Cache
	config.Providers = tempConfig.Providers

	if tempConfig.OAuth != nil {
		config.OAuth = &OAuth{
			IssuerURL: tempConfig.OAuth.IssuerURL,
			Features:  tempConfig.OAuth.Features,
		}

		// fmt.Println("----debug----")
		// fmt.Println("tempConfig.OAuth.IssuerURL", tempConfig.OAuth.IssuerURL)
		// fmt.Println("config.OAuth.IssuerURL", config.OAuth.IssuerURL)
		// fmt.Println("----debug----")

		// Convert signing config with duration parsing
		config.OAuth.Signing = types.SigningConfig{
			SigningCertPath:      tempConfig.OAuth.Signing.SigningCertPath,
			SigningKeyPath:       tempConfig.OAuth.Signing.SigningKeyPath,
			SigningKeyPassword:   tempConfig.OAuth.Signing.SigningKeyPassword,
			SigningAlgorithm:     tempConfig.OAuth.Signing.SigningAlgorithm,
			VerificationCerts:    tempConfig.OAuth.Signing.VerificationCerts,
			MTLSClientCACertPath: tempConfig.OAuth.Signing.MTLSClientCACertPath,
			MTLSEnabled:          tempConfig.OAuth.Signing.MTLSEnabled,
			CertRotationEnabled:  tempConfig.OAuth.Signing.CertRotationEnabled,
		}
		if tempConfig.OAuth.Signing.CertRotationInterval != "" {
			if duration, err := parseDuration(tempConfig.OAuth.Signing.CertRotationInterval); err == nil {
				config.OAuth.Signing.CertRotationInterval = duration
			}
		}

		// Convert token config with duration parsing
		config.OAuth.Token = types.TokenConfig{
			AccessTokenFormat:       tempConfig.OAuth.Token.AccessTokenFormat,
			AccessTokenSigningAlg:   tempConfig.OAuth.Token.AccessTokenSigningAlg,
			RefreshTokenRotation:    tempConfig.OAuth.Token.RefreshTokenRotation,
			RefreshTokenFormat:      tempConfig.OAuth.Token.RefreshTokenFormat,
			AuthorizationCodeLength: tempConfig.OAuth.Token.AuthorizationCodeLength,
			DeviceCodeLength:        tempConfig.OAuth.Token.DeviceCodeLength,
			UserCodeLength:          tempConfig.OAuth.Token.UserCodeLength,
			TokenBindingEnabled:     tempConfig.OAuth.Token.TokenBindingEnabled,
			SupportedBindingTypes:   tempConfig.OAuth.Token.SupportedBindingTypes,
			DefaultAudience:         tempConfig.OAuth.Token.DefaultAudience,
			AudienceValidationMode:  tempConfig.OAuth.Token.AudienceValidationMode,
		}
		if tempConfig.OAuth.Token.AccessTokenLifetime != "" {
			if duration, err := parseDuration(tempConfig.OAuth.Token.AccessTokenLifetime); err == nil {
				config.OAuth.Token.AccessTokenLifetime = duration
			}
		}
		if tempConfig.OAuth.Token.RefreshTokenLifetime != "" {
			if duration, err := parseDuration(tempConfig.OAuth.Token.RefreshTokenLifetime); err == nil {
				config.OAuth.Token.RefreshTokenLifetime = duration
			}
		}
		if tempConfig.OAuth.Token.AuthorizationCodeLifetime != "" {
			if duration, err := parseDuration(tempConfig.OAuth.Token.AuthorizationCodeLifetime); err == nil {
				config.OAuth.Token.AuthorizationCodeLifetime = duration
			}
		}
		if tempConfig.OAuth.Token.DeviceCodeLifetime != "" {
			if duration, err := parseDuration(tempConfig.OAuth.Token.DeviceCodeLifetime); err == nil {
				config.OAuth.Token.DeviceCodeLifetime = duration
			}
		}
		if tempConfig.OAuth.Token.DeviceCodeInterval != "" {
			if duration, err := parseDuration(tempConfig.OAuth.Token.DeviceCodeInterval); err == nil {
				config.OAuth.Token.DeviceCodeInterval = duration
			}
		}

		// Convert security config with duration parsing
		config.OAuth.Security = types.SecurityConfig{
			PKCERequired:                tempConfig.OAuth.Security.PKCERequired,
			PKCECodeChallengeMethod:     tempConfig.OAuth.Security.PKCECodeChallengeMethod,
			PKCECodeVerifierLength:      tempConfig.OAuth.Security.PKCECodeVerifierLength,
			StateParameterRequired:      tempConfig.OAuth.Security.StateParameterRequired,
			StateParameterLength:        tempConfig.OAuth.Security.StateParameterLength,
			RateLimitEnabled:            tempConfig.OAuth.Security.RateLimitEnabled,
			RateLimitRequests:           tempConfig.OAuth.Security.RateLimitRequests,
			RateLimitByClientID:         tempConfig.OAuth.Security.RateLimitByClientID,
			BruteForceProtectionEnabled: tempConfig.OAuth.Security.BruteForceProtectionEnabled,
			MaxFailedAttempts:           tempConfig.OAuth.Security.MaxFailedAttempts,
			EncryptionKey:               tempConfig.OAuth.Security.EncryptionKey,
			EncryptionAlgorithm:         tempConfig.OAuth.Security.EncryptionAlgorithm,
			IPWhitelist:                 tempConfig.OAuth.Security.IPWhitelist,
			IPBlacklist:                 tempConfig.OAuth.Security.IPBlacklist,
			RequireHTTPS:                tempConfig.OAuth.Security.RequireHTTPS,
			DisableUnsecureEndpoints:    tempConfig.OAuth.Security.DisableUnsecureEndpoints,
		}
		if tempConfig.OAuth.Security.StateParameterLifetime != "" {
			if duration, err := parseDuration(tempConfig.OAuth.Security.StateParameterLifetime); err == nil {
				config.OAuth.Security.StateParameterLifetime = duration
			}
		}
		if tempConfig.OAuth.Security.RateLimitWindow != "" {
			if duration, err := parseDuration(tempConfig.OAuth.Security.RateLimitWindow); err == nil {
				config.OAuth.Security.RateLimitWindow = duration
			}
		}
		if tempConfig.OAuth.Security.LockoutDuration != "" {
			if duration, err := parseDuration(tempConfig.OAuth.Security.LockoutDuration); err == nil {
				config.OAuth.Security.LockoutDuration = duration
			}
		}

		// Convert client config with duration parsing
		config.OAuth.Client = types.ClientConfig{
			DefaultClientType:              tempConfig.OAuth.Client.DefaultClientType,
			DefaultTokenEndpointAuthMethod: tempConfig.OAuth.Client.DefaultTokenEndpointAuthMethod,
			DefaultGrantTypes:              tempConfig.OAuth.Client.DefaultGrantTypes,
			DefaultResponseTypes:           tempConfig.OAuth.Client.DefaultResponseTypes,
			DefaultScopes:                  tempConfig.OAuth.Client.DefaultScopes,
			ClientIDLength:                 tempConfig.OAuth.Client.ClientIDLength,
			ClientSecretLength:             tempConfig.OAuth.Client.ClientSecretLength,
			DynamicRegistrationEnabled:     tempConfig.OAuth.Client.DynamicRegistrationEnabled,
			AllowedRedirectURISchemes:      tempConfig.OAuth.Client.AllowedRedirectURISchemes,
			AllowedRedirectURIHosts:        tempConfig.OAuth.Client.AllowedRedirectURIHosts,
			ClientCertificateRequired:      tempConfig.OAuth.Client.ClientCertificateRequired,
			ClientCertificateValidation:    tempConfig.OAuth.Client.ClientCertificateValidation,
		}
		if tempConfig.OAuth.Client.ClientSecretLifetime != "" {
			if duration, err := parseDuration(tempConfig.OAuth.Client.ClientSecretLifetime); err == nil {
				config.OAuth.Client.ClientSecretLifetime = duration
			}
		}
	}

	// Set defaults if needed
	if config.BaseURL == "" {
		config.BaseURL = "/v1"
	}

	// Format the BaseURL should not have trailing slash
	config.BaseURL = strings.TrimSuffix(config.BaseURL, "/")

	if config.Cache == "" {
		config.Cache = "__yao.oauth.cache"
	}

	if config.Store == "" {
		config.Store = "__yao.oauth.store"
	}

	return nil
}

// parseDuration parses a time duration string (e.g., "24h", "1h", "10m") into time.Duration
func parseDuration(durationStr string) (time.Duration, error) {
	if durationStr == "" || durationStr == "0" || durationStr == "0s" {
		return 0, nil
	}
	return time.ParseDuration(durationStr)
}

// formatDuration converts time.Duration to human-readable string format
func formatDuration(duration time.Duration) string {
	if duration == 0 {
		return "0s"
	}
	return duration.String()
}

// convertRelativeToAbsolutePath converts relative certificate path to absolute path
func convertRelativeToAbsolutePath(relativePath, rootPath string) string {
	if relativePath == "" {
		return ""
	}
	// If already absolute path, return as is
	if filepath.IsAbs(relativePath) {
		return relativePath
	}
	// Convert relative path to absolute: Root + "openapi" + "certs" + relativePath
	return filepath.Join(rootPath, "openapi", "certs", relativePath)
}

// convertAbsoluteToRelativePath converts absolute certificate path to relative path
func convertAbsoluteToRelativePath(absolutePath, rootPath string) string {
	if absolutePath == "" {
		return ""
	}
	// If not absolute path, return as is
	if !filepath.IsAbs(absolutePath) {
		return absolutePath
	}

	// Remove Root + "openapi" + "certs" prefix
	certBasePath := filepath.Join(rootPath, "openapi", "certs")
	if strings.HasPrefix(absolutePath, certBasePath) {
		relativePath := strings.TrimPrefix(absolutePath, certBasePath)
		// Remove leading separator
		relativePath = strings.TrimPrefix(relativePath, string(filepath.Separator))
		return relativePath
	}

	// If path doesn't match expected pattern, return as is
	return absolutePath
}

// OAuthConfig converts the configuration to an OAuth configuration
func (config *Config) OAuthConfig(appConfig config.Config) (*oauth.Config, error) {
	var oauthConfig oauth.Config

	var prefix string = share.App.GetPrefix()
	var providers *Providers = config.GetProviders()

	// Store the root path for later use in MarshalJSON
	config.root = appConfig.Root

	cacheStore, err := store.Get(config.Cache)
	if err != nil {
		return nil, err
	}

	dataStore, err := store.Get(config.Store)
	if err != nil {
		return nil, err
	}

	clientStore, err := store.Get(string(providers.Client))
	if err != nil {
		return nil, err
	}

	// Create the User provider
	userProvider := user.NewDefaultUser(&user.DefaultUserOptions{
		Prefix: prefix,
		Model:  string(providers.User),
		Cache:  cacheStore,
	})

	// Create the Client provider
	clientProvider, err := client.NewDefaultClient(&client.DefaultClientOptions{
		Prefix: prefix,
		Store:  clientStore,
		Cache:  cacheStore,
	})

	if err != nil {
		return nil, err
	}

	// Default OAuth configuration
	if config.OAuth == nil {
		config.OAuth = config.GetDefaultOAuthConfig()
	}

	// Convert certificate paths from relative to absolute
	signingConfig := config.OAuth.Signing
	signingConfig.SigningCertPath = convertRelativeToAbsolutePath(signingConfig.SigningCertPath, appConfig.Root)
	signingConfig.SigningKeyPath = convertRelativeToAbsolutePath(signingConfig.SigningKeyPath, appConfig.Root)
	signingConfig.MTLSClientCACertPath = convertRelativeToAbsolutePath(signingConfig.MTLSClientCACertPath, appConfig.Root)

	// Create the OAuth configuration
	oauthConfig = oauth.Config{
		UserProvider:   userProvider,
		ClientProvider: clientProvider,
		Cache:          cacheStore,
		Store:          dataStore,
		IssuerURL:      config.OAuth.IssuerURL,
		Signing:        signingConfig, // Use the converted signing config
		Token:          config.OAuth.Token,
		Security:       config.OAuth.Security,
		Client:         config.OAuth.Client,
		Features:       config.OAuth.Features,
	}

	return &oauthConfig, nil
}

// GetDefaultOAuthConfig Get the default OAuth configuration
func (config *Config) GetDefaultOAuthConfig() *OAuth {
	return &OAuth{
		Signing:  types.SigningConfig{},
		Token:    types.TokenConfig{},
		Security: types.SecurityConfig{},
		Client:   types.ClientConfig{},
		Features: oauth.FeatureFlags{},
	}
}

// GetProviders Get the providers from the configuration
func (config *Config) GetProviders() *Providers {
	if config.Providers == nil {
		config.Providers = &Providers{
			User:   "__yao.user",
			Client: "__yao.oauth.client",
		}

		return config.Providers
	}

	if config.Providers.User == "" {
		config.Providers.User = "__yao.user"
	}

	if config.Providers.Client == "" {
		config.Providers.Client = "__yao.oauth.client"
	}

	return config.Providers
}
