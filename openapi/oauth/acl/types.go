package acl

import (
	"fmt"
	"strings"
	"sync"

	"github.com/yaoapp/gou/store"
	"github.com/yaoapp/yao/openapi/oauth/types"
)

// DefaultConfig is the default configuration for the ACL
var DefaultConfig = Config{
	Enabled: false,
}

// Config is the configuration for the ACL
type Config struct {
	Enabled    bool               `json:"enabled"`
	PathPrefix string             `json:"path_prefix"` // BaseURL prefix to strip from request paths (e.g., "/v1")
	Cache      store.Store        `json:"-"`
	Provider   types.UserProvider `json:"-"`
}

// ACL is the ACL checker
type ACL struct {
	Config  *Config
	Scope   *ScopeManager
	Feature *FeatureManager
}

// ============ Configuration Structures (loaded from config files) ============

// GlobalConfig represents global scopes configuration (from scopes.yml)
type GlobalConfig struct {
	Default   string         `json:"default" yaml:"default"`     // "allow" or "deny" - default policy
	Public    []string       `json:"public" yaml:"public"`       // Public endpoints (no authentication required)
	Endpoints []EndpointRule `json:"endpoints" yaml:"endpoints"` // Default endpoint rules
}

// EndpointRule represents an endpoint rule (format: METHOD /path action)
type EndpointRule struct {
	Method string // HTTP method (GET, POST, PUT, DELETE, etc.)
	Path   string // URL path (supports wildcard *)
	Action string // "allow" or "deny"
}

// UnmarshalYAML implements custom YAML unmarshaling to support simple string format
// Supports both formats:
//   - "GET /api/users allow"  (simple string format)
//   - {method: GET, path: /api/users, action: allow}  (struct format)
func (e *EndpointRule) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// Try to unmarshal as string first (simple format)
	var str string
	if err := unmarshal(&str); err == nil {
		// Parse string format: "METHOD /path action"
		parts := strings.Fields(str)
		if len(parts) != 3 {
			return fmt.Errorf("invalid endpoint rule format: %q (expected: METHOD /path action)", str)
		}
		e.Method = parts[0]
		e.Path = parts[1]
		e.Action = parts[2]
		return nil
	}

	// Fallback to struct format
	type endpointRule EndpointRule // Create alias to avoid recursion
	var rule endpointRule
	if err := unmarshal(&rule); err != nil {
		return err
	}
	*e = EndpointRule(rule)
	return nil
}

// AliasConfig represents alias configuration (from alias.yml)
// Format: alias_name -> [scope1, scope2, ...]
type AliasConfig map[string][]string

// ScopeDefinition represents a scope definition (from subdirectory yml files)
type ScopeDefinition struct {
	Name        string                 `json:"name" yaml:"name"`                       // Scope name (e.g. collections:read:all)
	Description string                 `json:"description" yaml:"description"`         // Description
	Owner       bool                   `json:"owner" yaml:"owner"`                     // Owner only (current owner)
	Creator     bool                   `json:"creator" yaml:"creator"`                 // Creator only (who created)
	Editor      bool                   `json:"editor" yaml:"editor"`                   // Editor only (who last updated)
	Team        bool                   `json:"team" yaml:"team"`                       // Team only
	Extra       map[string]interface{} `json:"extra,omitempty" yaml:"extra,omitempty"` // Extra constraints
	Endpoints   []string               `json:"endpoints" yaml:"endpoints"`             // Endpoint list (format: METHOD /path)
}

// ============ Runtime Structures (optimized for querying) ============

// ScopeManager is the permission manager - global singleton, supports efficient querying and dynamic updates
type ScopeManager struct {
	mu sync.RWMutex // Read-write lock for concurrent safety

	// Global configuration
	defaultAction string              // Default policy: allow or deny
	publicPaths   map[string]struct{} // Public path set (fast lookup)

	// Runtime indexes (optimized for performance)
	endpointIndex map[string]*PathMatcher // method -> PathMatcher
	scopeIndex    map[string]*Scope       // scope_name -> Scope details
	aliasIndex    map[string][]string     // alias -> expanded scopes

	// Original configuration (for reloading)
	globalConfig *GlobalConfig
	aliasConfig  AliasConfig
	scopes       map[string]*ScopeDefinition
}

// PathMatcher stores path rules by priority
type PathMatcher struct {
	// Exact match paths (highest priority)
	// key: full path (e.g. "/kb/collections")
	// value: endpoint info
	exactPaths map[string]*EndpointInfo

	// Parameter paths (medium priority)
	// Grouped by segment count, supports :param placeholder
	// key: path pattern (e.g. "/kb/collections/:collectionID")
	// value: endpoint info
	paramPaths map[string]*EndpointInfo

	// Wildcard paths (lowest priority)
	// Sorted by prefix length (longer first)
	// e.g. ["/kb/collections/*", "/kb/*"]
	wildcardPaths []*WildcardPath
}

// WildcardPath represents a wildcard path rule
type WildcardPath struct {
	Pattern  string        // Original pattern (e.g. "/kb/*")
	Prefix   string        // Match prefix (e.g. "/kb/")
	Endpoint *EndpointInfo // Endpoint info
}

// EndpointInfo stores access control policy for an endpoint
type EndpointInfo struct {
	Method string // HTTP method
	Path   string // Original path pattern

	// Access control policy
	Policy EndpointPolicy // allow / deny / require-scopes

	// If Policy is require-scopes, the scopes required to access
	RequiredScopes []string // Scope list (OR relationship, any one satisfied)

	// Built-in resource constraints (common cases)
	OwnerOnly   bool // Owner only (current owner of the resource)
	CreatorOnly bool // Creator only (who created the resource)
	EditorOnly  bool // Editor only (who last updated the resource)
	TeamOnly    bool // Team only

	// Extra constraints (user-defined, flexible extension)
	// Examples: "department_only", "region_only", "project_only"
	// Value can be bool, string, or other types for complex constraints
	Extra map[string]interface{} `json:"extra,omitempty" yaml:"extra,omitempty"`
}

// GetConstraints returns all data access constraints as a map
// This allows flexible extension without changing method signatures
func (e *EndpointInfo) GetConstraints() map[string]interface{} {
	if e == nil {
		return map[string]interface{}{}
	}

	constraints := make(map[string]interface{})

	// Built-in constraints
	if e.OwnerOnly {
		constraints["owner_only"] = true
	}

	if e.CreatorOnly {
		constraints["creator_only"] = true
	}

	if e.EditorOnly {
		constraints["editor_only"] = true
	}

	if e.TeamOnly {
		constraints["team_only"] = true
	}

	// Merge extra constraints
	if e.Extra != nil {
		for key, value := range e.Extra {
			constraints[key] = value
		}
	}

	return constraints
}

// EndpointPolicy represents the endpoint policy
type EndpointPolicy int

const (
	// PolicyDeny denies access to the endpoint
	PolicyDeny EndpointPolicy = iota
	// PolicyAllow allows access to the endpoint without scope check
	PolicyAllow
	// PolicyRequireScopes requires specific scopes to access the endpoint
	PolicyRequireScopes
)

// Scope represents a permission scope
type Scope struct {
	Name        string                 // Scope name
	Description string                 // Description
	Owner       bool                   // Owner only (current owner)
	Creator     bool                   // Creator only (who created)
	Editor      bool                   // Editor only (who last updated)
	Team        bool                   // Team only
	Extra       map[string]interface{} // Extra constraints
	Endpoints   []string               // Associated endpoint list
}

// ============ Request Context (permission check context) ============

// AccessRequest represents an access request for scope-based access control
// It focuses on the resource being accessed and the available scopes
type AccessRequest struct {
	Method string   // HTTP method
	Path   string   // Request path
	Scopes []string // User's scopes (should be resolved externally including user, team, and client scopes)
}

// AccessDecision represents the access decision result
type AccessDecision struct {
	Allowed bool   // Whether access is allowed
	Reason  string // Decision reason (for debugging)

	// Matched endpoint info
	MatchedEndpoint *EndpointInfo
	MatchedPattern  string // Matched path pattern
	MatchedScope    string // Which scope was actually matched (for constraint lookup)

	// Permission check details
	RequiredScopes []string // Required scopes
	UserScopes     []string // User's scopes
	MissingScopes  []string // Missing scopes
}

// ============ Enforcement Stage (permission check stages) ============

// EnforcementStage represents the stage where permission check failed
type EnforcementStage string

const (
	// EnforcementStageClient indicates client permission check failed
	EnforcementStageClient EnforcementStage = "client"

	// EnforcementStageScope indicates scope permission check failed
	EnforcementStageScope EnforcementStage = "scope"

	// EnforcementStageTeam indicates team permission check failed
	EnforcementStageTeam EnforcementStage = "team"

	// EnforcementStageMember indicates member permission check failed
	EnforcementStageMember EnforcementStage = "member"

	// EnforcementStageUser indicates user permission check failed
	EnforcementStageUser EnforcementStage = "user"
)
