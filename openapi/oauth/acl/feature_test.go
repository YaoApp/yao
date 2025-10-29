package acl

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/application"
)

// setupFeatureTest initializes test environment
func setupFeatureTest(t *testing.T) *FeatureManager {
	// Set test application path
	testApp := os.Getenv("YAO_TEST_APPLICATION")
	if testApp == "" {
		t.Skip("YAO_TEST_APPLICATION not set, skipping feature tests")
	}

	// Initialize application
	app, err := application.OpenFromDisk(testApp)
	if err != nil {
		t.Fatalf("Failed to open application: %v", err)
	}
	application.Load(app)

	// Load features
	manager, err := LoadFeatures()
	if err != nil {
		t.Fatalf("Failed to load features: %v", err)
	}

	return manager
}

func TestLoadFeatures(t *testing.T) {
	manager := setupFeatureTest(t)
	assert.NotNil(t, manager)
}

func TestFeatures_SystemRoot(t *testing.T) {
	manager := setupFeatureTest(t)

	// Test system:root role with wildcard *:*:*
	features := manager.Features("system:root")
	assert.NotEmpty(t, features, "system:root should have features")

	// Verify it has all features from all domains
	assert.True(t, features["profile:read"], "should have profile:read")
	assert.True(t, features["profile:edit"], "should have profile:edit")
	assert.True(t, features["team:edit"], "should have team:edit")
	assert.True(t, features["collections:create"], "should have collections:create")
	assert.True(t, features["document:create"], "should have document:create")
	assert.True(t, features["meta:edit"], "should have meta:edit")
}

func TestFeatures_OwnerFree(t *testing.T) {
	manager := setupFeatureTest(t)

	// Test owner:free role
	features := manager.Features("owner:free")
	assert.NotEmpty(t, features, "owner:free should have features")

	// Should have profile:manage alias expanded
	assert.True(t, features["profile:read"], "should have profile:read from profile:manage alias")
	assert.True(t, features["profile:edit"], "should have profile:edit from profile:manage alias")

	// Should have team:manage alias expanded
	assert.True(t, features["team:edit"], "should have team:edit from team:manage alias")
	assert.True(t, features["team:member:invite"], "should have team:member:invite from team:manage alias")
	assert.True(t, features["team:member:remove"], "should have team:member:remove from team:manage alias")

	// Should have collections:create
	assert.True(t, features["collections:create"], "should have collections:create")
}

func TestFeatures_OwnerPro(t *testing.T) {
	manager := setupFeatureTest(t)

	// Test owner:pro role with user:full alias
	features := manager.Features("owner:pro")
	assert.NotEmpty(t, features, "owner:pro should have features")

	// user:full includes profile:manage, team:manage, kb:manage
	assert.True(t, features["profile:read"], "should have profile:read")
	assert.True(t, features["profile:edit"], "should have profile:edit")
	assert.True(t, features["team:edit"], "should have team:edit")
	assert.True(t, features["collections:create"], "should have collections:create from kb:manage")
}

func TestFeaturesByDomain_User(t *testing.T) {
	manager := setupFeatureTest(t)

	// Query "user" domain - should include all user/* files
	features := manager.FeaturesByDomain("system:root", "user")
	assert.NotEmpty(t, features, "user domain should have features")

	// Should include features from user/profile.yml
	assert.True(t, features["profile:read"], "should have profile:read from user/profile")
	assert.True(t, features["profile:edit"], "should have profile:edit from user/profile")

	// Should include features from user/team.yml
	assert.True(t, features["team:edit"], "should have team:edit from user/team")

	// Should include features from user/team/settings.yml (nested)
	assert.True(t, features["team:settings:view"], "should have team:settings:view from user/team/settings")
	assert.True(t, features["team:settings:edit"], "should have team:settings:edit from user/team/settings")

	// Should include features from user/team/members.yml (nested)
	assert.True(t, features["team:members:list"], "should have team:members:list from user/team/members")
	assert.True(t, features["team:members:add"], "should have team:members:add from user/team/members")

	// Should NOT include kb features
	assert.False(t, features["collections:create"], "should not have kb features")
}

func TestFeaturesByDomain_UserTeam(t *testing.T) {
	manager := setupFeatureTest(t)

	// Query "user/team" domain - should include user/team.yml (exact match) AND user/team/* files (prefix match)
	features := manager.FeaturesByDomain("system:root", "user/team")
	assert.NotEmpty(t, features, "user/team domain should have features")

	// Should NOT include user/profile features
	assert.False(t, features["profile:read"], "should not have profile:read")

	// Should include user/team.yml features (exact match on domain="user/team")
	assert.True(t, features["team:edit"], "should have team:edit from user/team.yml (exact match)")
	assert.True(t, features["team:member:invite"], "should have team:member:invite from user/team.yml")

	// Should include features from user/team/settings.yml (prefix match)
	assert.True(t, features["team:settings:view"], "should have team:settings:view from user/team/settings")
	assert.True(t, features["team:settings:edit"], "should have team:settings:edit from user/team/settings")

	// Should include features from user/team/members.yml (prefix match)
	assert.True(t, features["team:members:list"], "should have team:members:list from user/team/members")
	assert.True(t, features["team:members:add"], "should have team:members:add from user/team/members")
}

func TestFeaturesByDomain_UserProfile(t *testing.T) {
	manager := setupFeatureTest(t)

	// Query "user/profile" domain - should only include user/profile.yml
	features := manager.FeaturesByDomain("system:root", "user/profile")
	assert.NotEmpty(t, features, "user/profile domain should have features")

	// Should include features from user/profile.yml
	assert.True(t, features["profile:read"], "should have profile:read")
	assert.True(t, features["profile:edit"], "should have profile:edit")

	// Should NOT include team features
	assert.False(t, features["team:edit"], "should not have team:edit")
	assert.False(t, features["team:settings:view"], "should not have team:settings:view")
}

func TestFeaturesByDomain_KB(t *testing.T) {
	manager := setupFeatureTest(t)

	// Query "kb" domain - should include all kb/* files
	features := manager.FeaturesByDomain("system:root", "kb")
	assert.NotEmpty(t, features, "kb domain should have features")

	// Should include features from kb/collections.yml
	assert.True(t, features["collections:create"], "should have collections:create")

	// Should include features from kb/collections/document.yml
	assert.True(t, features["document:create"], "should have document:create")
	assert.True(t, features["document:edit"], "should have document:edit")
	assert.True(t, features["document:delete"], "should have document:delete")

	// Should include features from kb/collections/advanced/meta.yml (deep nested)
	assert.True(t, features["meta:edit"], "should have meta:edit from kb/collections/advanced/meta")
	assert.True(t, features["meta:view"], "should have meta:view from kb/collections/advanced/meta")
	assert.True(t, features["meta:export"], "should have meta:export from kb/collections/advanced/meta")

	// Should NOT include user features
	assert.False(t, features["profile:read"], "should not have user features")
}

func TestFeaturesByDomain_KBCollections(t *testing.T) {
	manager := setupFeatureTest(t)

	// Query "kb/collections" domain - should include kb/collections.yml and kb/collections/* files
	features := manager.FeaturesByDomain("system:root", "kb/collections")
	assert.NotEmpty(t, features, "kb/collections domain should have features")

	// Should include features from kb/collections.yml (exact match)
	assert.True(t, features["collections:create"], "should have collections:create from kb/collections")

	// Should include features from kb/collections/document.yml (nested)
	assert.True(t, features["document:create"], "should have document:create")
	assert.True(t, features["document:edit"], "should have document:edit")

	// Should include features from kb/collections/advanced/meta.yml (deep nested)
	assert.True(t, features["meta:edit"], "should have meta:edit")
	assert.True(t, features["meta:view"], "should have meta:view")
}

func TestFeaturesByDomain_KBCollectionsAdvanced(t *testing.T) {
	manager := setupFeatureTest(t)

	// Query "kb/collections/advanced" domain - should include kb/collections/advanced/* files
	features := manager.FeaturesByDomain("system:root", "kb/collections/advanced")
	assert.NotEmpty(t, features, "kb/collections/advanced domain should have features")

	// Should include features from kb/collections/advanced/meta.yml
	assert.True(t, features["meta:edit"], "should have meta:edit")
	assert.True(t, features["meta:view"], "should have meta:view")
	assert.True(t, features["meta:export"], "should have meta:export")

	// Should NOT include kb/collections.yml features
	assert.False(t, features["collections:create"], "should not have collections:create")

	// Should NOT include kb/collections/document.yml features
	assert.False(t, features["document:create"], "should not have document:create")
}

func TestDomainFeatures(t *testing.T) {
	manager := setupFeatureTest(t)

	// Test exact domain match - user/profile
	features := manager.DomainFeatures("user/profile")
	assert.NotEmpty(t, features, "user/profile domain should have features")
	assert.True(t, features["profile:read"], "should have profile:read")
	assert.True(t, features["profile:edit"], "should have profile:edit")

	// Should NOT include nested domains
	assert.False(t, features["team:edit"], "should not include user/team features")
	assert.False(t, features["team:settings:view"], "should not include user/team/settings features")
}

func TestDomains(t *testing.T) {
	manager := setupFeatureTest(t)

	domains := manager.Domains()
	assert.NotEmpty(t, domains, "should have domains")

	// Check expected domains exist
	expectedDomains := []string{
		"user/profile",
		"user/team",
		"user/team/settings",
		"user/team/members",
		"kb/collections",
		"kb/collections/document",
		"kb/collections/advanced/meta",
	}

	domainMap := make(map[string]bool)
	for _, d := range domains {
		domainMap[d] = true
	}

	for _, expected := range expectedDomains {
		assert.True(t, domainMap[expected], "should have domain: %s", expected)
	}
}

func TestDefinition(t *testing.T) {
	manager := setupFeatureTest(t)

	// Test feature definition lookup
	def := manager.Definition("profile:read")
	assert.NotNil(t, def, "should find profile:read definition")
	assert.Equal(t, "Read own profile", def.Description, "description should match")

	// Test nested domain feature
	def = manager.Definition("team:settings:view")
	assert.NotNil(t, def, "should find team:settings:view definition")
	assert.Equal(t, "View team settings", def.Description, "description should match")

	// Test deep nested feature
	def = manager.Definition("meta:edit")
	assert.NotNil(t, def, "should find meta:edit definition")
	assert.Equal(t, "Edit document metadata", def.Description, "description should match")

	// Test non-existent feature
	def = manager.Definition("nonexistent:feature")
	assert.Nil(t, def, "should return nil for non-existent feature")
}

func TestAliasExpansion(t *testing.T) {
	manager := setupFeatureTest(t)

	// Test that aliases are properly expanded
	features := manager.Features("owner:free")

	// profile:manage should expand to profile:read + profile:edit
	assert.True(t, features["profile:read"], "profile:manage alias should include profile:read")
	assert.True(t, features["profile:edit"], "profile:manage alias should include profile:edit")

	// team:manage should expand to multiple team features
	assert.True(t, features["team:edit"], "team:manage alias should include team:edit")
	assert.True(t, features["team:member:invite"], "team:manage alias should include team:member:invite")
	assert.True(t, features["team:member:robot:create"], "team:manage alias should include team:member:robot:create")
	assert.True(t, features["team:member:robot:edit"], "team:manage alias should include team:member:robot:edit")
	assert.True(t, features["team:member:remove"], "team:manage alias should include team:member:remove")
}

func TestNestedAliasExpansion(t *testing.T) {
	manager := setupFeatureTest(t)

	// Test nested alias: user:full -> profile:manage, team:manage, kb:manage
	features := manager.Features("owner:pro")

	// Should expand all nested aliases
	assert.True(t, features["profile:read"], "should have profile:read from profile:manage")
	assert.True(t, features["profile:edit"], "should have profile:edit from profile:manage")
	assert.True(t, features["team:edit"], "should have team:edit from team:manage")
	assert.True(t, features["team:member:invite"], "should have team:member:invite from team:manage")
	assert.True(t, features["collections:create"], "should have collections:create from kb:manage")
}

func TestEmptyRole(t *testing.T) {
	manager := setupFeatureTest(t)

	// Test non-existent role
	features := manager.Features("nonexistent:role")
	assert.Empty(t, features, "non-existent role should return empty map")

	// Test by domain with non-existent role
	features = manager.FeaturesByDomain("nonexistent:role", "user")
	assert.Empty(t, features, "non-existent role should return empty map for domain query")
}

func TestEmptyDomain(t *testing.T) {
	manager := setupFeatureTest(t)

	// Test non-existent domain
	features := manager.FeaturesByDomain("system:root", "nonexistent")
	assert.Empty(t, features, "non-existent domain should return empty map")

	// Test DomainFeatures with non-existent domain
	features = manager.DomainFeatures("nonexistent")
	assert.Empty(t, features, "non-existent domain should return empty map")
}

func TestFeatureMapReturnType(t *testing.T) {
	manager := setupFeatureTest(t)

	// Verify return type is map[string]bool for O(1) lookups
	features := manager.Features("owner:free")

	// Should be able to check existence with simple map lookup
	if features["profile:read"] {
		// Feature exists
		assert.True(t, true)
	} else {
		t.Error("profile:read should exist for owner:free")
	}

	// Check non-existent feature
	if features["nonexistent:feature"] {
		t.Error("nonexistent:feature should not exist")
	}
}

func TestHierarchicalQueryBehavior(t *testing.T) {
	manager := setupFeatureTest(t)

	// Verify hierarchical behavior: querying parent includes children
	userFeatures := manager.FeaturesByDomain("system:root", "user")
	userTeamFeatures := manager.FeaturesByDomain("system:root", "user/team")
	userProfileFeatures := manager.FeaturesByDomain("system:root", "user/profile")

	// user should include more features than user/team and user/profile
	assert.Greater(t, len(userFeatures), len(userTeamFeatures), "user should have more features than user/team")
	assert.Greater(t, len(userFeatures), len(userProfileFeatures), "user should have more features than user/profile")

	// user/team should NOT include user/profile features
	assert.True(t, userProfileFeatures["profile:read"], "user/profile should have profile:read")
	assert.False(t, userTeamFeatures["profile:read"], "user/team should NOT have profile:read")

	// user should include both
	assert.True(t, userFeatures["profile:read"], "user should have profile:read")
	assert.True(t, userFeatures["team:settings:view"], "user should have team:settings:view")
}

func TestConvenienceMethods(t *testing.T) {
	manager := setupFeatureTest(t)
	ctx := context.Background()

	// Note: These tests will skip if role manager is not properly initialized
	// In production, role manager would be initialized with a real provider

	// Test FeaturesForUser (will fail if role manager not initialized, which is expected in test)
	_, err := manager.FeaturesForUser(ctx, "test-user-123")
	// We expect an error because role manager might not be initialized
	if err != nil {
		assert.Contains(t, err.Error(), "role manager", "should indicate role manager issue")
	}

	// Test FeaturesForUserByDomain
	_, err = manager.FeaturesForUserByDomain(ctx, "test-user-123", "user")
	if err != nil {
		assert.Contains(t, err.Error(), "role manager", "should indicate role manager issue")
	}

	// Test FeaturesForTeamUser
	_, err = manager.FeaturesForTeamUser(ctx, "test-team-456", "test-user-123")
	if err != nil {
		assert.Contains(t, err.Error(), "role manager", "should indicate role manager issue")
	}

	// Test FeaturesForTeamUserByDomain
	_, err = manager.FeaturesForTeamUserByDomain(ctx, "test-team-456", "test-user-123", "user")
	if err != nil {
		assert.Contains(t, err.Error(), "role manager", "should indicate role manager issue")
	}
}
