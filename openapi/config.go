package openapi

import (
	"errors"

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
	return jsoniter.Marshal(config)
}

// UnmarshalJSON JSON Unmarshaler
func (config *Config) UnmarshalJSON(data []byte) error {
	return jsoniter.Unmarshal(data, config)
}

// OAuthConfig converts the configuration to an OAuth configuration
func (config *Config) OAuthConfig(appConfig *config.Config) (*oauth.Config, error) {
	var oauthConfig oauth.Config

	var prefix string = share.App.GetPrefix()
	var providers *Providers = config.GetProviders()

	cacheStore, err := store.Get(string(providers.Cache))
	if err != nil {
		return nil, err
	}

	dataStore, err := store.Get(string(providers.Client))
	if err != nil {
		return nil, err
	}

	// Create the User provider
	userProvider := user.NewDefaultUser(&user.DefaultUserOptions{
		Prefix:     prefix,
		Model:      string(providers.User),
		Cache:      cacheStore,
		TokenStore: dataStore,
	})

	// Create the Client provider
	clientProvider, err := client.NewDefaultClient(&client.DefaultClientOptions{
		Prefix: prefix,
		Store:  dataStore,
		Cache:  cacheStore,
	})

	if err != nil {
		return nil, err
	}

	// Default OAuth configuration
	if config.OAuth == nil {
		config.OAuth = config.GetDefaultOAuthConfig()
	}

	// Create the OAuth configuration
	oauthConfig = oauth.Config{
		UserProvider:   userProvider,
		ClientProvider: clientProvider,
		Cache:          cacheStore,
		Store:          dataStore,
		IssuerURL:      config.BaseURL,
		Signing:        config.OAuth.Signing,
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
			Cache:  "__yao.oauth.cache",
			Client: "__yao.oauth.client",
		}

		return config.Providers
	}

	if config.Providers.User == "" {
		config.Providers.User = "__yao.user"
	}

	if config.Providers.Cache == "" {
		config.Providers.Cache = "__yao.oauth.cache"
	}

	if config.Providers.Client == "" {
		config.Providers.Client = "__yao.oauth.client"
	}

	return config.Providers
}
