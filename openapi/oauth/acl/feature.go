package acl

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/openapi/oauth/acl/role"
	"gopkg.in/yaml.v3"
)

// ============ Gin Context Integration (Package-Level Functions) ============

// GetFeatures returns all features for the current user/team member from gin context
// Automatically determines whether to use user or team member lookup based on context
// Returns a map for O(1) feature lookup: feature_name -> true
func GetFeatures(c *gin.Context) (map[string]bool, error) {
	// Get ACL instance
	if Global == nil || !Global.Enabled() {
		return make(map[string]bool), nil
	}

	acl, ok := Global.(*ACL)
	if !ok || acl.Feature == nil {
		return make(map[string]bool), nil
	}

	// Get role ID from context
	roleID, err := getRoleFromContext(c)
	if err != nil || roleID == "" {
		return make(map[string]bool), err
	}

	// Get features for this role
	return acl.Feature.Features(roleID), nil
}

// GetFeaturesByDomain returns features filtered by domain from gin context
// Automatically determines whether to use user or team member lookup based on context
// Supports hierarchical matching: "user" includes "user/profile", "user/team", etc.
// Returns a map for O(1) feature lookup: feature_name -> true
func GetFeaturesByDomain(c *gin.Context, domain string) (map[string]bool, error) {
	// Get ACL instance
	if Global == nil || !Global.Enabled() {
		return make(map[string]bool), nil
	}

	acl, ok := Global.(*ACL)
	if !ok || acl.Feature == nil {
		return make(map[string]bool), nil
	}

	// Get role ID from context
	roleID, err := getRoleFromContext(c)
	if err != nil || roleID == "" {
		return make(map[string]bool), err
	}

	// Get features by domain for this role
	return acl.Feature.FeaturesByDomain(roleID, domain), nil
}

// ============ Public API (exported query methods) ============

// Features returns all features for a given role (expands aliases and wildcards)
// Returns a map for O(1) lookup: feature_name -> true
func (m *FeatureManager) Features(roleID string) map[string]bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	features := m.roleFeatures[roleID]
	if features == nil {
		return make(map[string]bool)
	}

	return m.expandFeaturesAsMap(features)
}

// FeaturesForUser returns all features for a user by looking up their role
// Returns a map for O(1) lookup: feature_name -> true
func (m *FeatureManager) FeaturesForUser(ctx context.Context, userID string) (map[string]bool, error) {
	roleID, err := m.getRoleForUser(ctx, userID)
	if err != nil {
		return make(map[string]bool), err
	}
	return m.Features(roleID), nil
}

// FeaturesForUserByDomain returns features for a user filtered by domain
// Returns a map for O(1) lookup: feature_name -> true
func (m *FeatureManager) FeaturesForUserByDomain(ctx context.Context, userID, domain string) (map[string]bool, error) {
	roleID, err := m.getRoleForUser(ctx, userID)
	if err != nil {
		return make(map[string]bool), err
	}
	return m.FeaturesByDomain(roleID, domain), nil
}

// FeaturesForTeamUser returns all features for a team user by looking up their member role
// Returns a map for O(1) lookup: feature_name -> true
func (m *FeatureManager) FeaturesForTeamUser(ctx context.Context, teamID, userID string) (map[string]bool, error) {
	roleID, err := m.getRoleForMember(ctx, teamID, userID)
	if err != nil {
		return make(map[string]bool), err
	}
	return m.Features(roleID), nil
}

// FeaturesForTeamUserByDomain returns features for a team user filtered by domain
// Returns a map for O(1) lookup: feature_name -> true
func (m *FeatureManager) FeaturesForTeamUserByDomain(ctx context.Context, teamID, userID, domain string) (map[string]bool, error) {
	roleID, err := m.getRoleForMember(ctx, teamID, userID)
	if err != nil {
		return make(map[string]bool), err
	}
	return m.FeaturesByDomain(roleID, domain), nil
}

// FeaturesByDomain returns features for a role filtered by domain
// Supports hierarchical matching: querying "user" will include "user/team", "user/profile", etc.
// Returns a map for O(1) lookup: feature_name -> true
func (m *FeatureManager) FeaturesByDomain(roleID, domain string) map[string]bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	features := m.roleFeatures[roleID]
	if features == nil {
		return make(map[string]bool)
	}

	// Expand all features
	expanded := m.expandFeaturesAsMap(features)

	// Filter by domain (supports hierarchical matching)
	result := make(map[string]bool)
	for feature := range expanded {
		featureDomain := m.featureDomain[feature]
		// Exact match OR prefix match (for nested domains)
		// e.g., domain="user" matches "user", "user/team", "user/profile", etc.
		if featureDomain == domain || strings.HasPrefix(featureDomain, domain+"/") {
			result[feature] = true
		}
	}

	return result
}

// DomainFeatures returns all features in a specific domain
// Returns a map for O(1) lookup: feature_name -> true
func (m *FeatureManager) DomainFeatures(domain string) map[string]bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	features := m.domainFeatures[domain]
	if features == nil {
		return make(map[string]bool)
	}

	result := make(map[string]bool, len(features))
	for name := range features {
		result[name] = true
	}

	return result
}

// Domains returns all available domains
func (m *FeatureManager) Domains() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	domains := make([]string, 0, len(m.domainFeatures))
	for domain := range m.domainFeatures {
		domains = append(domains, domain)
	}

	return domains
}

// Definition returns the definition of a specific feature
func (m *FeatureManager) Definition(featureName string) *FeatureDefinition {
	m.mu.RLock()
	defer m.mu.RUnlock()

	domain := m.featureDomain[featureName]
	if domain == "" {
		return nil
	}

	return m.domainFeatures[domain][featureName]
}

// ============ Internal Structures ============

// FeatureManager manages feature definitions and role-to-feature mappings
type FeatureManager struct {
	mu sync.RWMutex

	// Feature definitions by domain
	// domain -> feature_name -> FeatureDefinition
	// Example: "user" -> "profile:read" -> FeatureDefinition
	domainFeatures map[string]map[string]*FeatureDefinition

	// Feature aliases (groups of features)
	// alias_name -> []feature_names
	aliasIndex map[string][]string

	// Role to features mapping
	// role_id -> []feature_names (can include aliases and actual features)
	roleFeatures map[string][]string

	// Domain index for quick lookup
	// feature_name -> domain
	featureDomain map[string]string
}

// FeatureDefinition defines a single feature
type FeatureDefinition struct {
	Name        string `yaml:"-"`
	Description string `yaml:"description"`
}

// FeatureAliasConfig stores feature aliases (alias_name -> feature_names)
type FeatureAliasConfig map[string][]string

// RoleFeatureConfig stores role-to-features mapping (role_id -> feature_names)
type RoleFeatureConfig map[string][]string

// ============ Loading and Configuration ============

// LoadFeatures loads the feature configuration from the openapi/features directory
func LoadFeatures() (*FeatureManager, error) {
	manager := &FeatureManager{
		domainFeatures: make(map[string]map[string]*FeatureDefinition),
		aliasIndex:     make(map[string][]string),
		roleFeatures:   make(map[string][]string),
		featureDomain:  make(map[string]string),
	}

	// Check if features directory exists
	featuresDir := filepath.Join("openapi", "features")
	exists, err := application.App.Exists(featuresDir)
	if err != nil {
		return nil, err
	}
	if !exists {
		log.Warn("[Feature] Features directory not found")
		return manager, nil
	}

	// Step 1: Load feature aliases (alias.yml)
	if err := manager.loadAliasConfig(); err != nil {
		return nil, fmt.Errorf("failed to load alias config: %w", err)
	}

	// Step 2: Load feature definitions from subdirectories (domains)
	if err := manager.loadFeatureDefinitions(); err != nil {
		return nil, fmt.Errorf("failed to load feature definitions: %w", err)
	}

	// Step 3: Load role-to-features mapping (features.yml)
	if err := manager.loadRoleFeaturesConfig(); err != nil {
		return nil, fmt.Errorf("failed to load role features config: %w", err)
	}

	// Step 4: Build indexes
	if err := manager.buildIndexes(); err != nil {
		return nil, fmt.Errorf("failed to build indexes: %w", err)
	}

	log.Info("[Feature] Loaded %d features across %d domains, %d aliases, %d roles",
		len(manager.featureDomain), len(manager.domainFeatures), len(manager.aliasIndex), len(manager.roleFeatures))
	return manager, nil
}

// loadAliasConfig loads the feature aliases from alias.yml
func (m *FeatureManager) loadAliasConfig() error {
	configPath := filepath.Join("openapi", "features", "alias.yml")
	exists, err := application.App.Exists(configPath)
	if err != nil {
		return err
	}
	if !exists {
		log.Warn("[Feature] alias.yml not found")
		return nil
	}

	raw, err := application.App.Read(configPath)
	if err != nil {
		return err
	}

	var config FeatureAliasConfig
	if err := yaml.Unmarshal(raw, &config); err != nil {
		return err
	}

	// Expand aliases (resolve nested aliases)
	for alias := range config {
		expanded, err := m.expandAlias(alias, config, make(map[string]bool))
		if err != nil {
			return fmt.Errorf("failed to expand alias %s: %w", alias, err)
		}
		m.aliasIndex[alias] = expanded
	}

	return nil
}

// expandAlias recursively expands an alias to its features, detecting circular references
func (m *FeatureManager) expandAlias(alias string, config FeatureAliasConfig, visited map[string]bool) ([]string, error) {
	// Check for circular reference
	if visited[alias] {
		return nil, fmt.Errorf("circular alias reference detected: %s", alias)
	}
	visited[alias] = true

	features := config[alias]
	if features == nil {
		// Not an alias, return as is
		return []string{alias}, nil
	}

	var expanded []string
	seen := make(map[string]bool)

	for _, feature := range features {
		// Check if this is another alias
		if config[feature] != nil {
			// Recursively expand
			subFeatures, err := m.expandAlias(feature, config, visited)
			if err != nil {
				return nil, err
			}
			for _, sf := range subFeatures {
				if !seen[sf] {
					expanded = append(expanded, sf)
					seen[sf] = true
				}
			}
		} else {
			if !seen[feature] {
				expanded = append(expanded, feature)
				seen[feature] = true
			}
		}
	}

	delete(visited, alias)
	return expanded, nil
}

// loadFeatureDefinitions loads feature definitions from subdirectories (domains)
// Supports nested directories for hierarchical domain organization
func (m *FeatureManager) loadFeatureDefinitions() error {
	featuresDir := filepath.Join("openapi", "features")

	// Walk through all subdirectories
	err := application.App.Walk(featuresDir, func(root, path string, isdir bool) error {
		// Skip root directory files (alias.yml, features.yml)
		if filepath.Dir(path) == featuresDir {
			return nil
		}

		// Only process .yml files in subdirectories
		if isdir || !strings.HasSuffix(path, ".yml") {
			return nil
		}

		// Extract domain from path (include filename without extension)
		// Example: openapi/features/user/profile.yml -> domain = "user/profile"
		// Example: openapi/features/user/team/members.yml -> domain = "user/team/members"
		relPath, err := filepath.Rel(featuresDir, path)
		if err != nil {
			return err
		}

		// Remove .yml extension to get domain path
		domainPath := strings.TrimSuffix(relPath, ".yml")
		// Convert to forward slashes for consistent domain names
		domain := filepath.ToSlash(domainPath)

		// Load feature definitions from this file
		if err := m.loadFeatureFile(path, domain); err != nil {
			log.Warn("[Feature] Failed to load %s: %v", path, err)
		}

		return nil
	}, "*.yml")

	return err
}

// loadFeatureFile loads feature definitions from a single YAML file
func (m *FeatureManager) loadFeatureFile(filePath, domain string) error {
	raw, err := application.App.Read(filePath)
	if err != nil {
		return err
	}

	// Parse as map of feature definitions
	var featureMap map[string]*FeatureDefinition
	if err := yaml.Unmarshal(raw, &featureMap); err != nil {
		return err
	}

	// Initialize domain map if needed
	if m.domainFeatures[domain] == nil {
		m.domainFeatures[domain] = make(map[string]*FeatureDefinition)
	}

	// Store each feature definition
	for name, def := range featureMap {
		def.Name = name
		m.domainFeatures[domain][name] = def
	}

	return nil
}

// loadRoleFeaturesConfig loads the role-to-features mapping from features.yml
func (m *FeatureManager) loadRoleFeaturesConfig() error {
	configPath := filepath.Join("openapi", "features", "features.yml")
	exists, err := application.App.Exists(configPath)
	if err != nil {
		return err
	}
	if !exists {
		log.Warn("[Feature] features.yml not found")
		return nil
	}

	raw, err := application.App.Read(configPath)
	if err != nil {
		return err
	}

	var config RoleFeatureConfig
	if err := yaml.Unmarshal(raw, &config); err != nil {
		return err
	}

	m.roleFeatures = config
	return nil
}

// buildIndexes builds runtime indexes for efficient querying
func (m *FeatureManager) buildIndexes() error {
	// Build feature-to-domain index
	for domain, features := range m.domainFeatures {
		for featureName := range features {
			m.featureDomain[featureName] = domain
		}
	}

	return nil
}

// expandFeaturesAsMap expands feature list by resolving aliases and wildcards
// Returns a map for efficient lookup
func (m *FeatureManager) expandFeaturesAsMap(features []string) map[string]bool {
	result := make(map[string]bool)

	for _, feature := range features {
		// Check for wildcard
		if m.matchesWildcard(feature) {
			// Add all features
			for f := range m.featureDomain {
				result[f] = true
			}
			continue
		}

		// Check if it's an alias
		if aliasFeatures := m.aliasIndex[feature]; aliasFeatures != nil {
			for _, f := range aliasFeatures {
				result[f] = true
			}
		} else {
			// Regular feature
			result[feature] = true
		}
	}

	return result
}

// matchesWildcard checks if a feature string is a wildcard pattern
func (m *FeatureManager) matchesWildcard(feature string) bool {
	// Full wildcard: *:*:*
	if feature == "*:*:*" {
		return true
	}

	// Could extend to support partial wildcards in the future
	// For now, only support full wildcard
	return false
}

// Reload reloads the feature configuration
func (m *FeatureManager) Reload() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Create a new manager
	newManager, err := LoadFeatures()
	if err != nil {
		return err
	}

	// Replace current data with new data
	m.domainFeatures = newManager.domainFeatures
	m.aliasIndex = newManager.aliasIndex
	m.roleFeatures = newManager.roleFeatures
	m.featureDomain = newManager.featureDomain

	return nil
}

// ============================================================================
// Role Resolution Helper Methods
// ============================================================================

// getRoleForUser gets the role ID for a user from role manager
func (m *FeatureManager) getRoleForUser(ctx context.Context, userID string) (string, error) {
	if role.RoleManager == nil {
		return "", fmt.Errorf("role manager is not initialized")
	}
	return role.RoleManager.GetUserRole(ctx, userID)
}

// getRoleForMember gets the role ID for a team member from role manager
func (m *FeatureManager) getRoleForMember(ctx context.Context, teamID, userID string) (string, error) {
	if role.RoleManager == nil {
		return "", fmt.Errorf("role manager is not initialized")
	}
	return role.RoleManager.GetMemberRole(ctx, teamID, userID)
}

// ============================================================================
// Internal Helper Functions
// ============================================================================

// getRoleFromContext extracts role ID from gin context
// Automatically determines whether to use user or team member role lookup
func getRoleFromContext(c *gin.Context) (string, error) {
	// Get context for queries
	ctx := c.Request.Context()

	// Check if this is a team context (has team_id)
	teamID, hasTeam := c.Get("__team_id")
	userID, hasUser := c.Get("__user_id")

	if !hasUser {
		// No user_id, cannot get role
		return "", nil
	}

	userIDStr, ok := userID.(string)
	if !ok {
		return "", fmt.Errorf("invalid user_id type")
	}

	// If team_id exists, get member role
	if hasTeam && teamID != nil {
		if teamIDStr, ok := teamID.(string); ok && teamIDStr != "" {
			// Get member role from role manager
			if role.RoleManager != nil {
				return role.RoleManager.GetMemberRole(ctx, teamIDStr, userIDStr)
			}
		}
	}

	// No team_id, get user role
	if role.RoleManager != nil {
		return role.RoleManager.GetUserRole(ctx, userIDStr)
	}

	return "", fmt.Errorf("role manager is not initialized")
}
