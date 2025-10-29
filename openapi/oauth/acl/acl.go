package acl

import (
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/openapi/oauth/acl/role"
)

// Global is the global ACL enforcer
var Global Enforcer = nil

// New creates a new ACL enforcer
func New(config *Config) (Enforcer, error) {

	if config == nil {
		config = &DefaultConfig
	}

	acl := &ACL{
		Config: config,
	}

	// Load scope manager if ACL is enabled
	if config.Enabled {

		// Load scope manager
		manager, err := LoadScopes()
		if err != nil {
			return nil, err
		}
		acl.Scope = manager
		log.Info("[ACL] Scope manager loaded successfully")

		// Load feature manager
		featureManager, err := LoadFeatures()
		if err != nil {
			return nil, err
		}
		acl.Feature = featureManager
		log.Info("[ACL] Feature manager loaded successfully")

		// Init Role Manager
		role.RoleManager = role.NewManager(config.Cache, config.Provider)
		log.Info("[ACL] Role manager loaded successfully")

		// Log PathPrefix configuration
		if config.PathPrefix != "" {
			log.Info("[ACL] Path prefix configured: %s (will be stripped from request paths)", config.PathPrefix)
		} else {
			log.Info("[ACL] No path prefix configured")
		}
	}

	return acl, nil
}

// Load loads the ACL enforcer
func Load(config *Config) (Enforcer, error) {
	enforcer, err := New(config)
	if err != nil {
		return nil, err
	}

	// Clear role cache after loading to ensure fresh data
	if config != nil && config.Enabled && role.RoleManager != nil {
		if err := role.RoleManager.ClearCache(); err != nil {
			log.Warn("[ACL] Failed to clear role cache after loading: %v", err)
		} else {
			log.Debug("[ACL] Role cache cleared successfully")
		}
	}

	Global = enforcer
	return Global, nil
}

// Enabled returns true if the ACL is enabled, otherwise false
func (acl *ACL) Enabled() bool {
	return acl.Config.Enabled
}
