package acl

// Global is the global ACL enforcer
var Global Enforcer = nil

// New creates a new ACL enforcer
func New(config *Config) Enforcer {

	if config == nil {
		config = &DefaultConfig
	}

	return &ACL{
		Config: config,
	}
}

// Load loads the ACL enforcer
func Load(config *Config) (Enforcer, error) {
	Global = New(config)
	return Global, nil
}

// Enabled returns true if the ACL is enabled, otherwise false
func (acl *ACL) Enabled() bool {
	return acl.Config.Enabled
}
