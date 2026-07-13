package agent

import (
	"log"
	"path/filepath"

	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/helper"
	"github.com/yaoapp/yao/setting"
)

type toolProviderConfig struct {
	Default   string                       `yaml:"default,omitempty"`
	Providers map[string]map[string]string `yaml:"providers,omitempty"`
}

var validWebSearchDefaults = map[string]bool{
	"tavily": true, "serper": true, "cloud": true,
}

var validWebFetchDefaults = map[string]bool{
	"brightdata": true, "cloud": true, "direct": true,
}

// validProviderKeys lists providers that store credentials in search.providers.*.
// "cloud" is excluded — its credentials live in the cloud namespace.
var validProviderKeys = map[string]bool{
	"tavily": true, "serper": true, "brightdata": true,
}

// SyncSearchDefaults reads agent/websearch.yml and agent/webfetch.yml,
// resolves $ENV.XXX references, encrypts sensitive fields, and writes
// the results into setting.Global at system scope.
// Must be called after setting.Init().
func SyncSearchDefaults() error {
	if setting.Global == nil {
		return nil
	}

	wsCfg, wsErr := loadToolConfig(filepath.Join("agent", "websearch.yml"))
	if wsErr != nil {
		log.Printf("[SyncSearchDefaults] websearch.yml: %v", wsErr)
	}
	wfCfg, wfErr := loadToolConfig(filepath.Join("agent", "webfetch.yml"))
	if wfErr != nil {
		log.Printf("[SyncSearchDefaults] webfetch.yml: %v", wfErr)
	}

	if wsCfg == nil && wfCfg == nil {
		return nil
	}

	// Validate default values
	if wsCfg != nil && wsCfg.Default != "" && !validWebSearchDefaults[wsCfg.Default] {
		log.Printf("[SyncSearchDefaults] websearch.yml: unknown default %q, ignoring", wsCfg.Default)
		wsCfg.Default = ""
	}
	if wfCfg != nil && wfCfg.Default != "" && !validWebFetchDefaults[wfCfg.Default] {
		log.Printf("[SyncSearchDefaults] webfetch.yml: unknown default %q, ignoring", wfCfg.Default)
		wfCfg.Default = ""
	}

	// Merge tool_assignment: read existing, overlay new values
	assignment := map[string]interface{}{}
	if existing, err := setting.Global.Get(setting.ScopeID{Scope: setting.ScopeSystem}, "search.tool_assignment"); err == nil {
		for k, v := range existing {
			assignment[k] = v
		}
	}

	hasAssignment := false
	if wsCfg != nil && wsCfg.Default != "" {
		assignment["web_search"] = wsCfg.Default
		hasAssignment = true
	}
	if wfCfg != nil && wfCfg.Default != "" {
		assignment["web_scrape"] = wfCfg.Default
		hasAssignment = true
	}
	if hasAssignment {
		if _, err := setting.Global.Set(setting.ScopeID{Scope: setting.ScopeSystem}, "search.tool_assignment", assignment); err != nil {
			log.Printf("[SyncSearchDefaults] failed to write search.tool_assignment: %v", err)
		}
	}

	// Sync provider credentials
	for _, cfg := range []*toolProviderConfig{wsCfg, wfCfg} {
		if cfg == nil {
			continue
		}
		for key, fields := range cfg.Providers {
			if !validProviderKeys[key] {
				log.Printf("[SyncSearchDefaults] unknown provider key %q, skipping", key)
				continue
			}
			syncToolProvider(key, fields)
		}
	}

	return nil
}

// syncToolProvider encrypts sensitive fields and writes a single provider
// entry into setting.Global[system, "search.providers.<key>"].
func syncToolProvider(presetKey string, fields map[string]string) {
	if setting.Global == nil || len(fields) == 0 {
		return
	}

	// Skip providers without a valid api_key (missing or empty after ENV resolution)
	if fields["api_key"] == "" {
		return
	}

	fieldValues := map[string]interface{}{}
	for k, v := range fields {
		if isPasswordField(k) {
			fieldValues[k] = setting.Encrypt(v)
		} else {
			fieldValues[k] = v
		}
	}

	data := map[string]interface{}{
		"field_values": fieldValues,
		"enabled":      true,
		"status":       "connected",
	}

	ns := "search.providers." + presetKey
	if _, err := setting.Global.Set(setting.ScopeID{Scope: setting.ScopeSystem}, ns, data); err != nil {
		log.Printf("[SyncSearchDefaults] failed to write %s: %v", ns, err)
	}
}

// loadToolConfig reads a YAML config file from the application VFS and
// resolves $ENV.XXX references in all string values.
// Returns (nil, nil) if the file does not exist.
// Returns (nil, err) if the file exists but cannot be parsed.
func loadToolConfig(path string) (*toolProviderConfig, error) {
	if exists, _ := application.App.Exists(path); !exists {
		return nil, nil
	}

	bytes, err := application.App.Read(path)
	if err != nil {
		return nil, err
	}

	var cfg toolProviderConfig
	if err := application.Parse(filepath.Base(path), bytes, &cfg); err != nil {
		return nil, err
	}

	// Resolve $ENV.XXX in default
	cfg.Default = helper.EnvString(cfg.Default)

	// Resolve $ENV.XXX in provider field values
	for key, fields := range cfg.Providers {
		resolved := make(map[string]string, len(fields))
		for k, v := range fields {
			resolved[k] = helper.EnvString(v)
		}
		cfg.Providers[key] = resolved
	}

	return &cfg, nil
}

func isPasswordField(key string) bool {
	return key == "api_key"
}
