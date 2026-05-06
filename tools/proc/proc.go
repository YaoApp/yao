package proc

import (
	_ "embed"
	"strings"
	"sync"

	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/log"
)

//go:embed schema.json
var SchemaJSON []byte

//go:embed allowed.json
var AllowedSchemaJSON []byte

// Config is the tools.yml configuration structure.
// Extensible for future tool settings (web_search, web_fetch, etc.)
type Config struct {
	ProcessCall ProcessCallConfig `json:"process_call" yaml:"process_call"`
}

// ProcessCallConfig defines the allowed process list for process_call.
type ProcessCallConfig struct {
	Allowed []string `json:"allowed" yaml:"allowed"`
}

// Default allowed prefixes when no tools.yml is present.
var defaultAllowed = []string{
	"http.",
	"encoding.",
	"json.",
	"text.",
}

var (
	config     *Config
	configOnce sync.Once
)

// LoadConfig loads tools.yml from the application root.
// If tools.yml exists, its process_call.allowed completely replaces the default list.
// If tools.yml does not exist, the default safe list is used.
func LoadConfig() {
	configOnce.Do(func() {
		if application.App == nil {
			return
		}

		data, err := application.App.Read("tools.yml")
		if err != nil {
			return
		}

		cfg := &Config{}
		if err := application.Parse("tools.yml", data, cfg); err != nil {
			log.Error("[tools] failed to parse tools.yml: %s", err.Error())
			return
		}

		config = cfg
		log.Info("[tools] loaded tools.yml with %d process_call rules", len(cfg.ProcessCall.Allowed))
	})
}

// Handler is the tools.process_call process handler.
// Args[0]: name (string — process name, e.g. "models.user.Find")
// Args[1]: args ([]interface{} — process arguments, optional)
func Handler(p *process.Process) interface{} {
	LoadConfig()
	name := p.ArgsString(0)

	if !isAllowedProcess(name) {
		exception.New("process %s is not allowed", 403, name).Throw()
	}

	var args []interface{}
	if len(p.Args) > 1 {
		if arr, ok := p.Args[1].([]interface{}); ok {
			args = arr
		}
	}

	target, err := process.Of(name, args...)
	if err != nil {
		exception.New("process %s not found: %s", 404, name, err.Error()).Throw()
	}
	if p.Authorized != nil {
		target.WithAuthorized(p.Authorized)
	}
	target.WithSID(p.Sid)
	target.WithContext(p.Context)
	if err := target.Execute(); err != nil {
		exception.New("process %s execution failed: %s", 500, name, err.Error()).Throw()
	}
	defer target.Release()
	return target.Value()
}

// AllowedHandler is the tools.process_allowed process handler.
// Without args: returns the current allowed rules list.
// Args[0]: name (string) — check if a specific process is allowed, returns {"allowed": bool, "name": string}.
func AllowedHandler(p *process.Process) interface{} {
	LoadConfig()

	name := ""
	if len(p.Args) > 0 {
		name = p.ArgsString(0)
	}

	if name != "" {
		return map[string]interface{}{
			"name":    name,
			"allowed": isAllowedProcess(name),
		}
	}

	rules := defaultAllowed
	if config != nil && len(config.ProcessCall.Allowed) > 0 {
		rules = config.ProcessCall.Allowed
	}
	return map[string]interface{}{
		"rules": rules,
	}
}

func isAllowedProcess(name string) bool {
	lower := strings.ToLower(name)

	// If tools.yml was loaded, use its rules exclusively
	if config != nil && len(config.ProcessCall.Allowed) > 0 {
		return matchRules(lower, config.ProcessCall.Allowed)
	}

	// Otherwise use default safe list
	return matchRules(lower, defaultAllowed)
}

// matchRules checks name against a list of rules.
// Rules ending with "*" do prefix matching (e.g. "models.*" matches "models.user.find").
// Rules ending with "." also do prefix matching (e.g. "http." matches "http.get").
// Other rules do exact matching (e.g. "models.user.Find" matches only that).
func matchRules(lower string, rules []string) bool {
	for _, rule := range rules {
		r := strings.ToLower(rule)
		if strings.HasSuffix(r, ".*") {
			// "models.*" → prefix match on "models."
			prefix := r[:len(r)-1] // "models."
			if strings.HasPrefix(lower, prefix) {
				return true
			}
		} else if strings.HasSuffix(r, ".") {
			// "http." → prefix match
			if strings.HasPrefix(lower, r) {
				return true
			}
		} else {
			// Exact match
			if lower == r {
				return true
			}
		}
	}
	return false
}
