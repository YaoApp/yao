package acl

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/kun/log"
	"gopkg.in/yaml.v3"
)

// builtinScopes stores scopes registered by code (before configuration loading)
var builtinScopes = make(map[string]*ScopeDefinition)
var builtinScopesMutex sync.RWMutex

// Register registers built-in scopes that will be automatically loaded
// This should be called in init() functions before the ACL system is initialized
// Supports registering multiple scopes at once
//
// Example:
//
//	acl.Register(
//	    &acl.ScopeDefinition{
//	        Name:        "builtin:mfa:verification",
//	        Description: "MFA verification - temporary access for MFA setup",
//	        Endpoints:   []string{"POST /user/mfa/totp/enable", "POST /user/mfa/totp/verify"},
//	    },
//	    &acl.ScopeDefinition{
//	        Name:        "builtin:team:selection",
//	        Description: "Team selection scope",
//	        Endpoints:   []string{"POST /user/teams/select"},
//	    },
//	)
func Register(scopes ...*ScopeDefinition) {
	builtinScopesMutex.Lock()
	defer builtinScopesMutex.Unlock()

	for _, scope := range scopes {
		builtinScopes[scope.Name] = scope
		log.Trace("[ACL] Registered builtin scope: %s (%d endpoints)", scope.Name, len(scope.Endpoints))
	}
}

// LoadScopes loads the scope configuration from the openapi/scopes directory
func LoadScopes() (*ScopeManager, error) {
	manager := &ScopeManager{
		defaultAction: "deny",
		publicPaths:   make(map[string]struct{}),
		endpointIndex: make(map[string]*PathMatcher),
		scopeIndex:    make(map[string]*Scope),
		aliasIndex:    make(map[string][]string),
		scopes:        make(map[string]*ScopeDefinition),
	}

	// Step 1: Load builtin scopes first (registered by code)
	builtinScopesMutex.RLock()
	builtinCount := len(builtinScopes)
	for name, scopeDef := range builtinScopes {
		// Create a copy to avoid mutation
		defCopy := *scopeDef
		manager.scopes[name] = &defCopy
	}
	builtinScopesMutex.RUnlock()

	if builtinCount > 0 {
		log.Info("[ACL] Loaded %d builtin scopes from code registration", builtinCount)
	}

	// Check if scopes directory exists
	scopesDir := filepath.Join("openapi", "scopes")
	exists, err := application.App.Exists(scopesDir)
	if err != nil {
		return nil, err
	}
	if !exists {
		log.Warn("[ACL] Scopes directory not found, using default deny policy")
		// Still build indexes for builtin scopes
		if builtinCount > 0 {
			if err := manager.buildIndexes(); err != nil {
				return nil, fmt.Errorf("failed to build indexes for builtin scopes: %w", err)
			}
		}
		return manager, nil
	}

	// Step 2: Load global configuration (scopes.yml)
	if err := manager.loadGlobalConfig(); err != nil {
		return nil, fmt.Errorf("failed to load global config: %w", err)
	}

	// Step 3: Load alias configuration (alias.yml)
	if err := manager.loadAliasConfig(); err != nil {
		return nil, fmt.Errorf("failed to load alias config: %w", err)
	}

	// Step 4: Load scope definitions from subdirectories
	// This will merge with builtin scopes (file scopes override builtin if same name)
	if err := manager.loadScopeDefinitions(); err != nil {
		return nil, fmt.Errorf("failed to load scope definitions: %w", err)
	}

	// Step 5: Build runtime indexes
	if err := manager.buildIndexes(); err != nil {
		return nil, fmt.Errorf("failed to build indexes: %w", err)
	}

	log.Info("[ACL] Loaded %d scopes (%d builtin, %d from files), %d aliases",
		len(manager.scopeIndex), builtinCount, len(manager.scopes)-builtinCount, len(manager.aliasIndex))
	return manager, nil
}

// loadGlobalConfig loads the global scopes configuration from scopes.yml
func (m *ScopeManager) loadGlobalConfig() error {
	configPath := filepath.Join("openapi", "scopes", "scopes.yml")
	exists, err := application.App.Exists(configPath)
	if err != nil {
		return err
	}
	if !exists {
		log.Warn("[ACL] scopes.yml not found, using default configuration")
		return nil
	}

	raw, err := application.App.Read(configPath)
	if err != nil {
		return err
	}

	var config GlobalConfig
	if err := yaml.Unmarshal(raw, &config); err != nil {
		return err
	}

	m.globalConfig = &config

	// Set default action
	if config.Default != "" {
		m.defaultAction = config.Default
	}

	// Parse public endpoints
	for _, endpoint := range config.Public {
		// Format: METHOD /path
		parts := strings.Fields(endpoint)
		if len(parts) == 2 {
			key := parts[0] + " " + parts[1]
			m.publicPaths[key] = struct{}{}
		}
	}

	return nil
}

// loadAliasConfig loads the alias configuration from alias.yml
func (m *ScopeManager) loadAliasConfig() error {
	configPath := filepath.Join("openapi", "scopes", "alias.yml")
	exists, err := application.App.Exists(configPath)
	if err != nil {
		return err
	}
	if !exists {
		log.Warn("[ACL] alias.yml not found")
		return nil
	}

	raw, err := application.App.Read(configPath)
	if err != nil {
		return err
	}

	var config AliasConfig
	if err := yaml.Unmarshal(raw, &config); err != nil {
		return err
	}

	m.aliasConfig = config

	// Expand aliases (resolve nested aliases)
	for alias := range config {
		expanded, err := m.expandAlias(alias, make(map[string]bool))
		if err != nil {
			return fmt.Errorf("failed to expand alias %s: %w", alias, err)
		}
		m.aliasIndex[alias] = expanded
	}

	return nil
}

// expandAlias recursively expands an alias to its scopes, detecting circular references
func (m *ScopeManager) expandAlias(alias string, visited map[string]bool) ([]string, error) {
	// Check for circular reference
	if visited[alias] {
		return nil, fmt.Errorf("circular alias reference detected: %s", alias)
	}
	visited[alias] = true

	scopes := m.aliasConfig[alias]
	if scopes == nil {
		// Not an alias, return as is
		return []string{alias}, nil
	}

	var expanded []string
	for _, scope := range scopes {
		// Check if this is another alias
		if m.aliasConfig[scope] != nil {
			// Recursively expand
			subScopes, err := m.expandAlias(scope, visited)
			if err != nil {
				return nil, err
			}
			expanded = append(expanded, subScopes...)
		} else {
			expanded = append(expanded, scope)
		}
	}

	delete(visited, alias)
	return expanded, nil
}

// loadScopeDefinitions loads scope definitions from subdirectories
func (m *ScopeManager) loadScopeDefinitions() error {
	scopesDir := filepath.Join("openapi", "scopes")

	// Subdirectories to scan
	subDirs := []string{"kb", "job", "file", "user"}

	for _, subDir := range subDirs {
		dirPath := filepath.Join(scopesDir, subDir)
		exists, err := application.App.Exists(dirPath)
		if err != nil {
			return err
		}
		if !exists {
			continue
		}

		// Walk through all .yml files in the directory
		err = application.App.Walk(dirPath, func(root, filename string, isdir bool) error {
			if isdir {
				return nil
			}

			if !strings.HasSuffix(filename, ".yml") {
				return nil
			}

			if err := m.loadScopeFile(filename); err != nil {
				log.Warn("[ACL] Failed to load %s: %v", filename, err)
			}

			return nil
		}, "*.yml")

		if err != nil {
			return err
		}
	}

	return nil
}

// loadScopeFile loads scope definitions from a single YAML file
func (m *ScopeManager) loadScopeFile(filePath string) error {
	raw, err := application.App.Read(filePath)
	if err != nil {
		return err
	}

	// Parse as map of scope definitions
	var scopeMap map[string]*ScopeDefinition
	if err := yaml.Unmarshal(raw, &scopeMap); err != nil {
		return err
	}

	// Store each scope definition
	for name, def := range scopeMap {
		def.Name = name
		m.scopes[name] = def
	}

	return nil
}

// buildIndexes builds runtime indexes for efficient querying
func (m *ScopeManager) buildIndexes() error {
	// Build scope index
	for name, def := range m.scopes {
		m.scopeIndex[name] = &Scope{
			Name:        name,
			Description: def.Description,
			Owner:       def.Owner,
			Creator:     def.Creator,
			Editor:      def.Editor,
			Team:        def.Team,
			Extra:       def.Extra,
			Endpoints:   def.Endpoints,
		}
	}

	// Build endpoint index from global config
	if m.globalConfig != nil {
		for _, rule := range m.globalConfig.Endpoints {
			if err := m.addEndpointRule(rule.Method, rule.Path, rule.Action, nil); err != nil {
				return err
			}
		}
	}

	// Build endpoint index from scope definitions
	for name, def := range m.scopes {
		for _, endpoint := range def.Endpoints {
			// Format: METHOD /path
			parts := strings.Fields(endpoint)
			if len(parts) != 2 {
				log.Warn("[ACL] Invalid endpoint format: %s", endpoint)
				continue
			}

			method, path := parts[0], parts[1]
			if err := m.addEndpointRule(method, path, "require-scopes", []string{name}); err != nil {
				return err
			}
		}
	}

	// Sort wildcard paths by prefix length (longer first)
	for _, matcher := range m.endpointIndex {
		sort.Slice(matcher.wildcardPaths, func(i, j int) bool {
			return len(matcher.wildcardPaths[i].Prefix) > len(matcher.wildcardPaths[j].Prefix)
		})
	}

	return nil
}

// addEndpointRule adds an endpoint rule to the index
func (m *ScopeManager) addEndpointRule(method, path, action string, scopes []string) error {
	// Normalize path: remove trailing slash (except for root path)
	path = normalizePath(path)

	// Get or create PathMatcher for this method
	matcher := m.endpointIndex[method]
	if matcher == nil {
		matcher = &PathMatcher{
			exactPaths:    make(map[string]*EndpointInfo),
			paramPaths:    make(map[string]*EndpointInfo),
			wildcardPaths: []*WildcardPath{},
		}
		m.endpointIndex[method] = matcher
	}

	// Determine policy
	var policy EndpointPolicy
	switch action {
	case "allow":
		policy = PolicyAllow
	case "deny":
		policy = PolicyDeny
	case "require-scopes":
		policy = PolicyRequireScopes
	default:
		return fmt.Errorf("unknown action: %s", action)
	}

	// Create endpoint info
	info := &EndpointInfo{
		Method:         method,
		Path:           path,
		Policy:         policy,
		RequiredScopes: scopes,
	}

	// Set constraints from scope definitions
	if len(scopes) > 0 {
		for _, scopeName := range scopes {
			if def := m.scopes[scopeName]; def != nil {
				// Built-in constraints
				if def.Owner {
					info.OwnerOnly = true
				}
				if def.Creator {
					info.CreatorOnly = true
				}
				if def.Editor {
					info.EditorOnly = true
				}
				if def.Team {
					info.TeamOnly = true
				}

				// Merge extra constraints
				if len(def.Extra) > 0 {
					if info.Extra == nil {
						info.Extra = make(map[string]interface{})
					}
					for key, value := range def.Extra {
						info.Extra[key] = value
					}
				}
			}
		}
	}

	// Classify path type and add to appropriate index
	if strings.Contains(path, "*") {
		// Wildcard path
		prefix := strings.TrimSuffix(path, "*")
		matcher.wildcardPaths = append(matcher.wildcardPaths, &WildcardPath{
			Pattern:  path,
			Prefix:   prefix,
			Endpoint: info,
		})
	} else if strings.Contains(path, ":") {
		// Parameter path - merge with existing if present (support multiple scopes per endpoint)
		if existing := matcher.paramPaths[path]; existing != nil {
			// Endpoint already exists, merge scopes and constraints
			existing.RequiredScopes = append(existing.RequiredScopes, info.RequiredScopes...)
			// Merge constraints (OR logic: if any scope requires it, set to true)
			existing.OwnerOnly = existing.OwnerOnly || info.OwnerOnly
			existing.CreatorOnly = existing.CreatorOnly || info.CreatorOnly
			existing.EditorOnly = existing.EditorOnly || info.EditorOnly
			existing.TeamOnly = existing.TeamOnly || info.TeamOnly
			// Merge extra constraints
			if info.Extra != nil {
				if existing.Extra == nil {
					existing.Extra = make(map[string]interface{})
				}
				for key, value := range info.Extra {
					existing.Extra[key] = value
				}
			}
			log.Trace("[ACL] Merged endpoint %s %s: scopes=%v, owner=%v, team=%v",
				method, path, existing.RequiredScopes, existing.OwnerOnly, existing.TeamOnly)
		} else {
			matcher.paramPaths[path] = info
			log.Trace("[ACL] Added endpoint %s %s: scopes=%v, owner=%v, team=%v",
				method, path, info.RequiredScopes, info.OwnerOnly, info.TeamOnly)
		}
	} else {
		// Exact path - merge with existing if present (support multiple scopes per endpoint)
		if existing := matcher.exactPaths[path]; existing != nil {
			// Endpoint already exists, merge scopes and constraints
			existing.RequiredScopes = append(existing.RequiredScopes, info.RequiredScopes...)
			existing.OwnerOnly = existing.OwnerOnly || info.OwnerOnly
			existing.CreatorOnly = existing.CreatorOnly || info.CreatorOnly
			existing.EditorOnly = existing.EditorOnly || info.EditorOnly
			existing.TeamOnly = existing.TeamOnly || info.TeamOnly
			if info.Extra != nil {
				if existing.Extra == nil {
					existing.Extra = make(map[string]interface{})
				}
				for key, value := range info.Extra {
					existing.Extra[key] = value
				}
			}
			log.Trace("[ACL] Merged endpoint %s %s: scopes=%v, owner=%v, team=%v",
				method, path, existing.RequiredScopes, existing.OwnerOnly, existing.TeamOnly)
		} else {
			matcher.exactPaths[path] = info
			log.Trace("[ACL] Added endpoint %s %s: scopes=%v, owner=%v, team=%v",
				method, path, info.RequiredScopes, info.OwnerOnly, info.TeamOnly)
		}
	}

	return nil
}

// Check checks if the request scopes satisfy the endpoint requirements
func (m *ScopeManager) Check(req *AccessRequest) *AccessDecision {
	m.mu.RLock()
	defer m.mu.RUnlock()

	decision := &AccessDecision{
		Allowed:    false,
		UserScopes: req.Scopes,
	}

	// Normalize the request path
	normalizedPath := normalizePath(req.Path)

	// 1. Check if it's a public endpoint
	publicKey := req.Method + " " + normalizedPath
	if _, ok := m.publicPaths[publicKey]; ok {
		decision.Allowed = true
		decision.Reason = "public endpoint"
		return decision
	}

	// 2. Find matching endpoint (matchEndpoint will normalize the path again, but it's idempotent)
	endpoint, pattern := m.matchEndpoint(req.Method, normalizedPath)
	if endpoint == nil {
		// No match found, use default policy
		decision.Allowed = m.defaultAction == "allow"
		decision.Reason = fmt.Sprintf("no match, default policy: %s", m.defaultAction)
		return decision
	}

	decision.MatchedEndpoint = endpoint
	decision.MatchedPattern = pattern

	// 3. Check policy
	switch endpoint.Policy {
	case PolicyAllow:
		decision.Allowed = true
		decision.Reason = "policy: allow"
		return decision

	case PolicyDeny:
		decision.Allowed = false
		decision.Reason = "policy: deny"
		return decision

	case PolicyRequireScopes:
		// Expand user scopes (include aliases)
		expandedScopes := m.expandUserScopes(req.Scopes)

		// Check if user has any required scope (OR relationship)
		decision.RequiredScopes = endpoint.RequiredScopes
		var matchedScope string
		for _, required := range endpoint.RequiredScopes {
			for _, userScope := range expandedScopes {
				// Check for exact match or wildcard match
				if userScope == required || m.matchesWildcardScope(userScope, required) {
					matchedScope = required
					break
				}
			}
			if matchedScope != "" {
				break
			}
		}

		if matchedScope == "" {
			decision.Allowed = false
			decision.Reason = "missing required scopes"
			decision.MissingScopes = m.findMissingScopes(expandedScopes, endpoint.RequiredScopes)
			return decision
		}

		// Record which scope was matched
		decision.MatchedScope = matchedScope
		decision.Allowed = true
		decision.Reason = "scope matched"
		return decision
	}

	decision.Allowed = false
	decision.Reason = "unknown policy"
	return decision
}

// CheckRestricted checks if the endpoint is restricted by any of the given scopes
// Returns true if the endpoint is restricted (should be denied)
func (m *ScopeManager) CheckRestricted(req *AccessRequest) *AccessDecision {
	m.mu.RLock()
	defer m.mu.RUnlock()

	decision := &AccessDecision{
		Allowed:    true, // Default to allowed (not restricted)
		UserScopes: req.Scopes,
	}

	// Normalize the request path
	normalizedPath := normalizePath(req.Path)

	// 1. Check if it's a public endpoint - public endpoints cannot be restricted
	publicKey := req.Method + " " + normalizedPath
	if _, ok := m.publicPaths[publicKey]; ok {
		decision.Allowed = true
		decision.Reason = "public endpoint"
		return decision
	}

	// 2. Find matching endpoint (matchEndpoint will normalize the path again, but it's idempotent)
	endpoint, pattern := m.matchEndpoint(req.Method, normalizedPath)
	if endpoint == nil {
		// No match found - not restricted
		decision.Allowed = true
		decision.Reason = "no restriction match"
		return decision
	}

	decision.MatchedEndpoint = endpoint
	decision.MatchedPattern = pattern

	// 3. Check if any user scope matches the endpoint's required scopes
	// If it matches, this endpoint IS restricted by these scopes
	switch endpoint.Policy {
	case PolicyDeny:
		// Explicit deny policy - this is restricted
		decision.Allowed = false
		decision.Reason = "policy: deny (restricted)"
		return decision

	case PolicyRequireScopes:
		// Expand user scopes (include aliases)
		expandedScopes := m.expandUserScopes(req.Scopes)

		// Check if this endpoint requires any of the user's scopes
		// If yes, this endpoint is restricted by these scopes
		for _, required := range endpoint.RequiredScopes {
			for _, userScope := range expandedScopes {
				// Check for exact match or wildcard match
				if userScope == required || m.matchesWildcardScope(userScope, required) {
					// This endpoint is restricted by this scope
					decision.Allowed = false
					decision.Reason = "endpoint restricted by scope: " + required
					decision.RequiredScopes = []string{required}
					return decision
				}
			}
		}

		// No restriction match
		decision.Allowed = true
		decision.Reason = "no restriction match"
		return decision
	}

	// Default: not restricted
	decision.Allowed = true
	decision.Reason = "not restricted"
	return decision
}

// matchEndpoint finds the matching endpoint for a request
func (m *ScopeManager) matchEndpoint(method, path string) (*EndpointInfo, string) {
	// Normalize path: remove trailing slash (except for root path)
	path = normalizePath(path)

	matcher := m.endpointIndex[method]
	if matcher == nil {
		return nil, ""
	}

	// 1. Try exact match
	if info := matcher.exactPaths[path]; info != nil {
		return info, path
	}

	// 2. Try parameter match
	for pattern, info := range matcher.paramPaths {
		if m.matchParameterPath(pattern, path) {
			return info, pattern
		}
	}

	// 3. Try wildcard match (already sorted by prefix length)
	for _, wildcard := range matcher.wildcardPaths {
		if strings.HasPrefix(path, wildcard.Prefix) {
			return wildcard.Endpoint, wildcard.Pattern
		}
	}

	return nil, ""
}

// matchParameterPath checks if a path matches a parameter pattern
func (m *ScopeManager) matchParameterPath(pattern, path string) bool {
	patternParts := strings.Split(strings.Trim(pattern, "/"), "/")
	pathParts := strings.Split(strings.Trim(path, "/"), "/")

	// Must have same number of segments
	if len(patternParts) != len(pathParts) {
		return false
	}

	for i := range patternParts {
		// Parameter segment (starts with :)
		if strings.HasPrefix(patternParts[i], ":") {
			continue
		}
		// Exact match required
		if patternParts[i] != pathParts[i] {
			return false
		}
	}

	return true
}

// expandUserScopes expands user scopes by resolving aliases
func (m *ScopeManager) expandUserScopes(scopes []string) []string {
	var expanded []string
	seen := make(map[string]bool)

	for _, scope := range scopes {
		// Check if it's an alias
		if aliasScopes := m.aliasIndex[scope]; aliasScopes != nil {
			for _, s := range aliasScopes {
				if !seen[s] {
					expanded = append(expanded, s)
					seen[s] = true
				}
			}
		} else {
			if !seen[scope] {
				expanded = append(expanded, scope)
				seen[scope] = true
			}
		}
	}

	return expanded
}

// matchesWildcardScope checks if a user scope (potentially with wildcards) matches a required scope
// Supports patterns like:
//   - *:*:* matches everything
//   - resource:*:* matches resource:action:level
//   - resource:action:* matches resource:action:level
func (m *ScopeManager) matchesWildcardScope(userScope, requiredScope string) bool {
	// No wildcard, no match (exact match already checked)
	if !strings.Contains(userScope, "*") {
		return false
	}

	// Split both scopes into parts
	userParts := strings.Split(userScope, ":")
	requiredParts := strings.Split(requiredScope, ":")

	// If lengths don't match and user scope isn't full wildcard, no match
	if len(userParts) != len(requiredParts) {
		return false
	}

	// Check each part
	for i := range userParts {
		if userParts[i] == "*" {
			// Wildcard matches anything
			continue
		}
		if userParts[i] != requiredParts[i] {
			// Not a match
			return false
		}
	}

	return true
}

// findMissingScopes finds which scopes are missing
func (m *ScopeManager) findMissingScopes(userScopes, requiredScopes []string) []string {
	userScopeSet := make(map[string]bool)
	for _, s := range userScopes {
		userScopeSet[s] = true
	}

	var missing []string
	for _, required := range requiredScopes {
		// Check exact match
		if userScopeSet[required] {
			continue
		}

		// Check wildcard match
		matched := false
		for _, userScope := range userScopes {
			if m.matchesWildcardScope(userScope, required) {
				matched = true
				break
			}
		}

		if !matched {
			missing = append(missing, required)
		}
	}

	return missing
}

// GetScopeConstraints returns the constraints for a specific scope
// This allows getting the original constraints for a matched scope,
// instead of using merged constraints from multiple scopes
func (m *ScopeManager) GetScopeConstraints(scopeName string, method, path string) *EndpointInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Get the scope definition
	scopeDef := m.scopes[scopeName]
	if scopeDef == nil {
		return nil
	}

	// Create an EndpointInfo with this scope's constraints
	info := &EndpointInfo{
		Method:         method,
		Path:           path,
		Policy:         PolicyRequireScopes,
		RequiredScopes: []string{scopeName},
		OwnerOnly:      scopeDef.Owner,
		CreatorOnly:    scopeDef.Creator,
		EditorOnly:     scopeDef.Editor,
		TeamOnly:       scopeDef.Team,
	}

	// Copy extra constraints
	if len(scopeDef.Extra) > 0 {
		info.Extra = make(map[string]interface{})
		for key, value := range scopeDef.Extra {
			info.Extra[key] = value
		}
	}

	return info
}

// Reload reloads the scope configuration
func (m *ScopeManager) Reload() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Create a new manager
	newManager, err := LoadScopes()
	if err != nil {
		return err
	}

	// Replace current data with new data
	m.defaultAction = newManager.defaultAction
	m.publicPaths = newManager.publicPaths
	m.endpointIndex = newManager.endpointIndex
	m.scopeIndex = newManager.scopeIndex
	m.aliasIndex = newManager.aliasIndex
	m.globalConfig = newManager.globalConfig
	m.aliasConfig = newManager.aliasConfig
	m.scopes = newManager.scopes

	return nil
}

// normalizePath normalizes a path by removing trailing slashes (except for root path "/")
// This ensures consistent path matching regardless of whether the request or definition has a trailing slash
// Examples:
//   - "/user/teams/" -> "/user/teams"
//   - "/user/teams" -> "/user/teams"
//   - "/" -> "/"
//   - "" -> ""
func normalizePath(path string) string {
	// Empty path or root path - return as is
	if path == "" || path == "/" {
		return path
	}

	// Remove trailing slash
	return strings.TrimSuffix(path, "/")
}
