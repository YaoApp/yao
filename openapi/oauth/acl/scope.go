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

// getFirstLevelSubdirs returns all first-level subdirectories in a given directory
func getFirstLevelSubdirs(baseDir string) ([]string, error) {
	var subdirs []string

	err := application.App.Walk(baseDir, func(root, filename string, isdir bool) error {
		// Only process directories
		if !isdir {
			return nil
		}

		// Skip the root directory itself
		if filename == baseDir {
			return nil
		}

		// Get relative path from baseDir
		relPath := strings.TrimPrefix(filename, baseDir)
		relPath = strings.TrimPrefix(relPath, string(filepath.Separator))

		// Only include first-level directories (no nested paths)
		if !strings.Contains(relPath, string(filepath.Separator)) && relPath != "" {
			subdirs = append(subdirs, relPath)
		}

		return nil
	}, "")

	if err != nil {
		return nil, err
	}

	return subdirs, nil
}

// loadScopeDefinitions loads scope definitions from subdirectories
func (m *ScopeManager) loadScopeDefinitions() error {
	scopesDir := filepath.Join("openapi", "scopes")

	// Get all subdirectories in the scopes directory
	subDirs, err := getFirstLevelSubdirs(scopesDir)
	if err != nil {
		return fmt.Errorf("failed to scan scopes directory: %w", err)
	}

	// Load scope definitions from each subdirectory
	for _, subDir := range subDirs {
		dirPath := filepath.Join(scopesDir, subDir)

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

				// Merge extra constraints (deep copy to prevent shared state)
				if len(def.Extra) > 0 {
					if info.Extra == nil {
						info.Extra = make(map[string]interface{})
					}
					for key, value := range def.Extra {
						info.Extra[key] = deepCopyValue(value)
					}
				}
			}
		}
	}

	// Classify path type and add to appropriate index
	if strings.Contains(path, "*") {
		// Wildcard path - merge with existing if present (support multiple scopes per endpoint)
		prefix := strings.TrimSuffix(path, "*")
		merged := false
		for _, existing := range matcher.wildcardPaths {
			if existing.Pattern == path {
				// Endpoint already exists, merge scopes and constraints
				existing.Endpoint.RequiredScopes = append(existing.Endpoint.RequiredScopes, info.RequiredScopes...)
				existing.Endpoint.OwnerOnly = existing.Endpoint.OwnerOnly || info.OwnerOnly
				existing.Endpoint.CreatorOnly = existing.Endpoint.CreatorOnly || info.CreatorOnly
				existing.Endpoint.EditorOnly = existing.Endpoint.EditorOnly || info.EditorOnly
				existing.Endpoint.TeamOnly = existing.Endpoint.TeamOnly || info.TeamOnly
				if info.Extra != nil {
					if existing.Endpoint.Extra == nil {
						existing.Endpoint.Extra = make(map[string]interface{})
					}
					for key, value := range info.Extra {
						existing.Endpoint.Extra[key] = deepCopyValue(value)
					}
				}
				log.Trace("[ACL] Merged wildcard endpoint %s %s: scopes=%v, owner=%v, team=%v",
					method, path, existing.Endpoint.RequiredScopes, existing.Endpoint.OwnerOnly, existing.Endpoint.TeamOnly)
				merged = true
				break
			}
		}
		if !merged {
			matcher.wildcardPaths = append(matcher.wildcardPaths, &WildcardPath{
				Pattern:  path,
				Prefix:   prefix,
				Endpoint: info,
			})
			log.Trace("[ACL] Added wildcard endpoint %s %s: scopes=%v, owner=%v, team=%v",
				method, path, info.RequiredScopes, info.OwnerOnly, info.TeamOnly)
		}
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
			// Merge extra constraints (deep copy to prevent shared state)
			if info.Extra != nil {
				if existing.Extra == nil {
					existing.Extra = make(map[string]interface{})
				}
				for key, value := range info.Extra {
					existing.Extra[key] = deepCopyValue(value)
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
			// Merge extra constraints (deep copy to prevent shared state)
			if info.Extra != nil {
				if existing.Extra == nil {
					existing.Extra = make(map[string]interface{})
				}
				for key, value := range info.Extra {
					existing.Extra[key] = deepCopyValue(value)
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
// Returns a defensive copy to prevent accidental mutations of shared endpoint data
func (m *ScopeManager) matchEndpoint(method, path string) (*EndpointInfo, string) {
	// Normalize path: remove trailing slash (except for root path)
	path = normalizePath(path)

	matcher := m.endpointIndex[method]
	if matcher == nil {
		return nil, ""
	}

	// 1. Try exact match
	if info := matcher.exactPaths[path]; info != nil {
		return m.copyEndpointInfo(info), path
	}

	// 2. Try parameter match
	for pattern, info := range matcher.paramPaths {
		if m.matchParameterPath(pattern, path) {
			return m.copyEndpointInfo(info), pattern
		}
	}

	// 3. Try wildcard match (already sorted by prefix length)
	for _, wildcard := range matcher.wildcardPaths {
		if strings.HasPrefix(path, wildcard.Prefix) {
			return m.copyEndpointInfo(wildcard.Endpoint), wildcard.Pattern
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

// expandUserScopes expands user scopes by resolving aliases recursively
// This allows nested aliases like: system:root -> *:*:* -> matches any scope
func (m *ScopeManager) expandUserScopes(scopes []string) []string {
	var expanded []string
	seen := make(map[string]bool)

	// Use a queue for iterative expansion to avoid deep recursion
	queue := make([]string, len(scopes))
	copy(queue, scopes)

	for len(queue) > 0 {
		scope := queue[0]
		queue = queue[1:]

		// Skip if already processed
		if seen[scope] {
			continue
		}
		seen[scope] = true

		// Check if it's an alias
		if aliasScopes := m.aliasIndex[scope]; aliasScopes != nil {
			// Add alias expansions to queue for further processing
			for _, s := range aliasScopes {
				if !seen[s] {
					queue = append(queue, s)
				}
			}
		} else {
			// Not an alias, add to expanded list
			expanded = append(expanded, scope)
		}
	}

	return expanded
}

// matchesWildcardScope checks if a user scope (potentially with wildcards) matches a required scope
// Supports patterns like:
//   - *:*:* matches any 3-part scope (e.g., sui:run:execute)
//   - *:*:*:* matches any 4-part scope (e.g., sui:run:execute:all)
//   - *:*:*:*:* matches any 5-part scope
//   - resource:*:* matches resource:action:level
//   - resource:action:* matches resource:action:level
//   - resource:*:*:* matches resource:action:level:sublevel
//
// The wildcard pattern must have the same number of parts as the required scope.
// Use multiple wildcard patterns in alias.yml to cover different scope depths.
func (m *ScopeManager) matchesWildcardScope(userScope, requiredScope string) bool {
	// No wildcard, no match (exact match already checked)
	if !strings.Contains(userScope, "*") {
		return false
	}

	// Split both scopes into parts
	userParts := strings.Split(userScope, ":")
	requiredParts := strings.Split(requiredScope, ":")

	// Lengths must match for wildcard matching
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

	// Deep copy extra constraints to prevent shared state issues
	if len(scopeDef.Extra) > 0 {
		info.Extra = deepCopyMap(scopeDef.Extra)
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

// copyEndpointInfo creates a defensive copy of an EndpointInfo object
// This prevents accidental mutations of shared endpoint data
func (m *ScopeManager) copyEndpointInfo(src *EndpointInfo) *EndpointInfo {
	if src == nil {
		return nil
	}

	// Create new EndpointInfo with copied fields
	dst := &EndpointInfo{
		Method:      src.Method,
		Path:        src.Path,
		Policy:      src.Policy,
		OwnerOnly:   src.OwnerOnly,
		CreatorOnly: src.CreatorOnly,
		EditorOnly:  src.EditorOnly,
		TeamOnly:    src.TeamOnly,
	}

	// Deep copy RequiredScopes slice
	if len(src.RequiredScopes) > 0 {
		dst.RequiredScopes = make([]string, len(src.RequiredScopes))
		copy(dst.RequiredScopes, src.RequiredScopes)
	}

	// Deep copy Extra map
	if len(src.Extra) > 0 {
		dst.Extra = deepCopyMap(src.Extra)
	}

	return dst
}

// deepCopyMap creates a deep copy of a map[string]interface{}
// This handles common types: primitives, strings, slices, and nested maps
func deepCopyMap(src map[string]interface{}) map[string]interface{} {
	if src == nil {
		return nil
	}

	dst := make(map[string]interface{}, len(src))
	for key, value := range src {
		dst[key] = deepCopyValue(value)
	}
	return dst
}

// deepCopyValue creates a deep copy of an interface{} value
// Handles common types: primitives, strings, slices, maps
func deepCopyValue(src interface{}) interface{} {
	if src == nil {
		return nil
	}

	switch v := src.(type) {
	case map[string]interface{}:
		// Recursively copy nested maps
		return deepCopyMap(v)
	case []interface{}:
		// Copy slices
		dst := make([]interface{}, len(v))
		for i, item := range v {
			dst[i] = deepCopyValue(item)
		}
		return dst
	case []string:
		// Copy string slices
		dst := make([]string, len(v))
		copy(dst, v)
		return dst
	default:
		// For primitives (bool, int, float64, string), direct assignment is safe
		// Note: If you have custom types or pointers in Extra, they may need special handling
		return v
	}
}
