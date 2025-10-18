package acl

// DefaultConfig is the default configuration for the ACL
var DefaultConfig = Config{
	Enabled: false,
}

// Config is the configuration for the ACL
type Config struct {
	Enabled bool `json:"enabled"`
}

// ACL is the ACL checker
type ACL struct {
	Config *Config
}
