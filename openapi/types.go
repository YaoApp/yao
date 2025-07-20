package openapi

import (
	"github.com/yaoapp/yao/openapi/oauth"
	"github.com/yaoapp/yao/openapi/oauth/types"
)

// Config is the configuration for the OpenAPI server
type Config struct {
	BaseURL   string     `json:"baseurl" yaml:"baseurl"`
	Providers *Providers `json:"providers,omitempty" yaml:"providers,omitempty"`
	OAuth     *OAuth     `json:"oauth,omitempty" yaml:"oauth,omitempty"`
}

// Provider is the provider for the OpenAPI server, and in the future will be refactored into a struct
type Provider string

// Providers is the providers for the OpenAPI server
type Providers struct {
	User   Provider `json:"user,omitempty" yaml:"user,omitempty"`
	Cache  Provider `json:"cache,omitempty" yaml:"cache,omitempty"`
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
