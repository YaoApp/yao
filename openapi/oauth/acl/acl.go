package acl

import (
	"github.com/yaoapp/kun/log"
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
	}

	return acl, nil
}

// Load loads the ACL enforcer
func Load(config *Config) (Enforcer, error) {
	enforcer, err := New(config)
	if err != nil {
		return nil, err
	}
	Global = enforcer
	return Global, nil
}

// Enabled returns true if the ACL is enabled, otherwise false
func (acl *ACL) Enabled() bool {
	return acl.Config.Enabled
}
