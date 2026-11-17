package user

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

//go:embed providers/tao.yml
var builtinTaoConfig []byte

// registerBuiltinTaoProvider registers the system Tao provider. Tao is always
// present; availability depends on TAO_OAUTH_ISSUER being configured.
func registerBuiltinTaoProvider() error {
	if len(builtinTaoConfig) == 0 {
		return fmt.Errorf("builtin tao config is empty")
	}
	var provider Provider
	if err := json.Unmarshal(builtinTaoConfig, &provider); err != nil {
		return fmt.Errorf("parse builtin tao config: %w", err)
	}
	provider.ID = "tao"
	provider.ClientID = replaceENVVar(provider.ClientID)
	provider.ClientSecret = replaceENVVar(provider.ClientSecret)
	provider.Issuer = replaceENVVar(provider.Issuer)
	if provider.Issuer != "" {
		provider.Available = true
		if provider.Endpoints == nil {
			provider.Endpoints = &Endpoints{}
		}
		provider.Endpoints.Authorization = replaceENVVar(provider.Endpoints.Authorization)
		provider.Endpoints.Token = replaceENVVar(provider.Endpoints.Token)
		provider.Endpoints.UserInfo = replaceENVVar(provider.Endpoints.UserInfo)
		provider.Endpoints.JWKS = replaceENVVar(provider.Endpoints.JWKS)
		base := strings.TrimRight(provider.Issuer, "/")
		if provider.Endpoints.Authorization == "" {
			provider.Endpoints.Authorization = base + "/oauth/authorize"
		}
		if provider.Endpoints.Token == "" {
			provider.Endpoints.Token = base + "/oauth/token"
		}
		if provider.Endpoints.UserInfo == "" {
			provider.Endpoints.UserInfo = base + "/oauth/userinfo"
		}
		if provider.Endpoints.JWKS == "" {
			provider.Endpoints.JWKS = base + "/.well-known/jwks.json"
		}
	}
	providers["tao"] = &provider
	return nil
}

// LocalLoginEnabled reports whether Yao's local username/password login is on.
func LocalLoginEnabled() bool {
	configMutex.RLock()
	defer configMutex.RUnlock()
	if yaoClientConfig == nil || yaoClientConfig.User == nil || yaoClientConfig.User.LocalLogin == nil {
		return false
	}
	return yaoClientConfig.User.LocalLogin.Enabled
}

// TaoConfigured reports whether the Tao provider has an issuer.
func TaoConfigured() bool {
	configMutex.RLock()
	defer configMutex.RUnlock()
	p := providers["tao"]
	return p != nil && p.Issuer != ""
}

// PublicProviders returns sanitized provider metadata for frontends.
func PublicProviders() []*Provider {
	configMutex.RLock()
	defer configMutex.RUnlock()
	ids := make([]string, 0, len(providers))
	for id := range providers {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	result := make([]*Provider, 0, len(ids))
	for _, id := range ids {
		src := providers[id]
		if src == nil {
			continue
		}
		if id != "tao" && (!src.Available && (src.ClientID == "" || src.Endpoints == nil)) {
			continue
		}
		clone := *src
		clone.ClientSecret = ""
		clone.DeviceClientSecret = ""
		clone.ClientSecretGenerator = nil
		clone.Available = id == "tao" && src.Issuer != ""
		if id != "tao" {
			clone.Available = src.ClientID != "" && src.Endpoints != nil && src.Endpoints.Authorization != ""
		}
		result = append(result, &clone)
	}
	return result
}
