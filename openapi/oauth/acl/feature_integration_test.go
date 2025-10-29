package acl

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/gou/store"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/openapi/oauth/acl/role"
	"github.com/yaoapp/yao/openapi/oauth/providers/user"
	"github.com/yaoapp/yao/test"
)

var (
	integrationTestProvider *user.DefaultUser
	integrationTestCache    store.Store
)

// createMockGinContext creates a mock gin.Context for testing
// If teamID is empty, creates a user context; otherwise creates a team member context
func createMockGinContext(userID, teamID string) *gin.Context {
	// Create a test HTTP request
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	// Create gin context
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	// Set user_id in context
	c.Set("__user_id", userID)

	// Set team_id if provided
	if teamID != "" {
		c.Set("__team_id", teamID)
	}

	return c
}

// prepareIntegrationTest initializes the integration test environment
func prepareIntegrationTest(t *testing.T) (*FeatureManager, string, string, string, string) {
	// Initialize test environment
	test.Prepare(t, config.Conf)

	// Set test application path
	testApp := os.Getenv("YAO_TEST_APPLICATION")
	if testApp == "" {
		t.Skip("YAO_TEST_APPLICATION not set, skipping integration tests")
	}

	// Initialize application
	app, err := application.OpenFromDisk(testApp)
	require.NoError(t, err)
	application.Load(app)

	// Initialize provider
	integrationTestProvider = user.NewDefaultUser(&user.DefaultUserOptions{
		Prefix:     "test:",
		IDStrategy: user.NanoIDStrategy,
		IDPrefix:   "test_",
	})

	// Initialize cache for role manager (use system store if available, nil otherwise)
	integrationTestCache, _ = store.Get("system")

	// Initialize role manager (cache can be nil, role manager handles it gracefully)
	role.RoleManager = role.NewManager(integrationTestCache, integrationTestProvider)

	// Load features
	manager, err := LoadFeatures()
	require.NoError(t, err)

	ctx := context.Background()
	testUUID := strings.ReplaceAll(uuid.New().String(), "-", "")[:8]

	// Note: We assume roles owner:free and team:admin already exist in features.yml
	// If they don't exist in the database, create them (but ignore duplicate errors)
	roleOwnerFree := maps.MapStrAny{
		"role_id":     "owner:free",
		"name":        "Owner Free",
		"description": "Free tier owner",
		"is_active":   true,
		"level":       10,
	}
	integrationTestProvider.CreateRole(ctx, roleOwnerFree) // Ignore error if exists

	roleMemberAdmin := maps.MapStrAny{
		"role_id":     "team:admin",
		"name":        "Team Admin",
		"description": "Team administrator",
		"is_active":   true,
		"level":       50,
	}
	integrationTestProvider.CreateRole(ctx, roleMemberAdmin) // Ignore error if exists

	// Create test user
	userMap := maps.MapStrAny{
		"preferred_username": "featureuser" + testUUID,
		"email":              "featureuser" + testUUID + "@example.com",
		"password":           "TestPass123!",
		"name":               "Feature Test User",
		"status":             "active",
		"role_id":            "owner:free",
		"type_id":            "regular",
		"email_verified":     true,
	}
	_, err = integrationTestProvider.CreateUser(ctx, userMap)
	require.NoError(t, err)
	userID := userMap["user_id"].(string)

	// Create test team
	teamMap := maps.MapStrAny{
		"name":         "Feature Test Team " + testUUID,
		"display_name": "Feature Test Team",
		"description":  "Test team for feature integration",
		"owner_id":     userID,
		"status":       "active",
		"type":         "corporation",
		"type_id":      "business",
	}
	teamID, err := integrationTestProvider.CreateTeam(ctx, teamMap)
	require.NoError(t, err)

	// Create another user as team member
	memberUserMap := maps.MapStrAny{
		"preferred_username": "featuremember" + testUUID,
		"email":              "featuremember" + testUUID + "@example.com",
		"password":           "TestPass123!",
		"name":               "Feature Member User",
		"status":             "active",
		"role_id":            "owner:free",
		"type_id":            "regular",
		"email_verified":     true,
	}
	_, err = integrationTestProvider.CreateUser(ctx, memberUserMap)
	require.NoError(t, err)
	memberUserID := memberUserMap["user_id"].(string)

	// Add member to team
	memberData := maps.MapStrAny{
		"team_id":     teamID,
		"user_id":     memberUserID,
		"member_type": "user",
		"role_id":     "team:admin",
		"status":      "active",
	}
	_, err = integrationTestProvider.CreateMember(ctx, memberData)
	require.NoError(t, err)

	return manager, testUUID, userID, teamID, memberUserID
}

// cleanIntegrationTest cleans up integration test data
func cleanIntegrationTest(testUUID string) {
	if integrationTestProvider == nil {
		return
	}

	// Clean users
	userModel := model.Select("__yao.user")
	userModel.DestroyWhere(model.QueryParam{
		Wheres: []model.QueryWhere{
			{Column: "preferred_username", OP: "like", Value: "%featureuser" + testUUID + "%"},
		},
	})
	userModel.DestroyWhere(model.QueryParam{
		Wheres: []model.QueryWhere{
			{Column: "preferred_username", OP: "like", Value: "%featuremember" + testUUID + "%"},
		},
	})

	// Clean teams
	teamModel := model.Select("__yao.team")
	teamModel.DestroyWhere(model.QueryParam{
		Wheres: []model.QueryWhere{
			{Column: "name", OP: "like", Value: "Feature Test Team " + testUUID},
		},
	})

	// Reset globals
	integrationTestProvider = nil
	integrationTestCache = nil
	role.RoleManager = nil

	// Clean base test environment
	test.Clean()
}

func TestFeaturesForUser_Integration(t *testing.T) {
	manager, testUUID, userID, _, _ := prepareIntegrationTest(t)
	defer cleanIntegrationTest(testUUID)

	ctx := context.Background()

	// Test FeaturesForUser
	t.Run("FeaturesForUser", func(t *testing.T) {
		features, err := manager.FeaturesForUser(ctx, userID)
		require.NoError(t, err)
		assert.NotEmpty(t, features, "user should have features")

		// owner:free should have profile:manage expanded
		assert.True(t, features["profile:read"], "should have profile:read from profile:manage alias")
		assert.True(t, features["profile:edit"], "should have profile:edit from profile:manage alias")

		// Should have team:manage expanded
		assert.True(t, features["team:edit"], "should have team:edit from team:manage alias")
		assert.True(t, features["team:member:invite"], "should have team:member:invite")

		// Should have collections:create
		assert.True(t, features["collections:create"], "should have collections:create")
	})

	// Test FeaturesForUserByDomain
	t.Run("FeaturesForUserByDomain", func(t *testing.T) {
		// Query user domain
		userFeatures, err := manager.FeaturesForUserByDomain(ctx, userID, "user")
		require.NoError(t, err)
		assert.NotEmpty(t, userFeatures, "user domain should have features")

		// Should include profile features (from user/profile.yml)
		assert.True(t, userFeatures["profile:read"], "should have profile:read in user domain")
		assert.True(t, userFeatures["profile:edit"], "should have profile:edit in user domain")

		// Should include team features (from user/team.yml via team:manage alias)
		assert.True(t, userFeatures["team:edit"], "should have team:edit in user domain")
		assert.True(t, userFeatures["team:member:invite"], "should have team:member:invite in user domain")

		// Note: team:settings:view is NOT in owner:free role
		// It's in user/team/settings.yml but not included in team:manage alias
		// If owner:free role had access to all user/* features, we would see it

		// Should NOT include kb features
		assert.False(t, userFeatures["collections:create"], "should not have kb features in user domain")

		// Query specific subdomain
		profileFeatures, err := manager.FeaturesForUserByDomain(ctx, userID, "user/profile")
		require.NoError(t, err)
		assert.True(t, profileFeatures["profile:read"], "should have profile:read in user/profile domain")
		assert.False(t, profileFeatures["team:edit"], "should not have team features in user/profile domain")
	})
}

func TestFeaturesForTeamUser_Integration(t *testing.T) {
	manager, testUUID, _, teamID, memberUserID := prepareIntegrationTest(t)
	defer cleanIntegrationTest(testUUID)

	ctx := context.Background()

	// Test FeaturesForTeamUser
	t.Run("FeaturesForTeamUser", func(t *testing.T) {
		features, err := manager.FeaturesForTeamUser(ctx, teamID, memberUserID)
		require.NoError(t, err)
		assert.NotEmpty(t, features, "team member should have features")

		// team:admin should have profile:manage, team:manage, collections:create
		assert.True(t, features["profile:read"], "should have profile:read")
		assert.True(t, features["profile:edit"], "should have profile:edit")
		assert.True(t, features["team:edit"], "should have team:edit")
		assert.True(t, features["team:member:invite"], "should have team:member:invite")
		assert.True(t, features["collections:create"], "should have collections:create")
	})

	// Test FeaturesForTeamUserByDomain
	t.Run("FeaturesForTeamUserByDomain", func(t *testing.T) {
		// Query user domain for team member
		userFeatures, err := manager.FeaturesForTeamUserByDomain(ctx, teamID, memberUserID, "user")
		require.NoError(t, err)
		assert.NotEmpty(t, userFeatures, "team member should have user domain features")

		// Should include user/* features that are in team:admin role
		assert.True(t, userFeatures["profile:read"], "should have profile:read")
		assert.True(t, userFeatures["team:edit"], "should have team:edit")
		assert.True(t, userFeatures["team:member:invite"], "should have team:member:invite")

		// Note: team:settings:view and team:members:list are NOT in team:admin role
		// because they're not included in the aliases that team:admin has

		// Should NOT include kb features
		assert.False(t, userFeatures["collections:create"], "should not have kb features in user domain")

		// Query kb domain
		kbFeatures, err := manager.FeaturesForTeamUserByDomain(ctx, teamID, memberUserID, "kb")
		require.NoError(t, err)
		assert.True(t, kbFeatures["collections:create"], "should have collections:create in kb domain")
		assert.False(t, kbFeatures["profile:read"], "should not have user features in kb domain")
	})
}

func TestConvenienceMethodsWithCache_Integration(t *testing.T) {
	manager, testUUID, userID, teamID, memberUserID := prepareIntegrationTest(t)
	defer cleanIntegrationTest(testUUID)

	ctx := context.Background()

	// First call should query from database
	features1, err := manager.FeaturesForUser(ctx, userID)
	require.NoError(t, err)
	assert.NotEmpty(t, features1)

	// Second call should use cache
	features2, err := manager.FeaturesForUser(ctx, userID)
	require.NoError(t, err)
	assert.NotEmpty(t, features2)
	assert.Equal(t, features1, features2, "cached results should match")

	// Test team member cache
	teamFeatures1, err := manager.FeaturesForTeamUser(ctx, teamID, memberUserID)
	require.NoError(t, err)
	assert.NotEmpty(t, teamFeatures1)

	teamFeatures2, err := manager.FeaturesForTeamUser(ctx, teamID, memberUserID)
	require.NoError(t, err)
	assert.Equal(t, teamFeatures1, teamFeatures2, "cached team member results should match")
}

func TestGetFeaturesFromGinContext_Integration(t *testing.T) {
	manager, testUUID, userID, teamID, memberUserID := prepareIntegrationTest(t)
	defer cleanIntegrationTest(testUUID)

	// Load ACL with feature manager
	config := &Config{
		Enabled: true,
	}
	acl := &ACL{
		Config:  config,
		Feature: manager,
	}
	Global = acl

	// Test GetFeatures for user (no team_id)
	t.Run("GetFeatures_User", func(t *testing.T) {
		// Create mock gin.Context
		c := createMockGinContext(userID, "")

		// Call GetFeatures
		features, err := GetFeatures(c)
		require.NoError(t, err)
		assert.NotEmpty(t, features, "user should have features")

		// Verify features
		assert.True(t, features["profile:read"], "should have profile:read")
		assert.True(t, features["profile:edit"], "should have profile:edit")
		assert.True(t, features["team:edit"], "should have team:edit")
		assert.True(t, features["collections:create"], "should have collections:create")
	})

	// Test GetFeaturesByDomain for user
	t.Run("GetFeaturesByDomain_User", func(t *testing.T) {
		// Create mock gin.Context
		c := createMockGinContext(userID, "")

		// Call GetFeaturesByDomain with "user" domain
		userFeatures, err := GetFeaturesByDomain(c, "user")
		require.NoError(t, err)
		assert.NotEmpty(t, userFeatures, "user domain should have features")

		// Should include user domain features
		assert.True(t, userFeatures["profile:read"], "should have profile:read")
		assert.True(t, userFeatures["team:edit"], "should have team:edit")

		// Should NOT include kb features
		assert.False(t, userFeatures["collections:create"], "should not have kb features in user domain")

		// Call GetFeaturesByDomain with "kb" domain
		kbFeatures, err := GetFeaturesByDomain(c, "kb")
		require.NoError(t, err)
		assert.True(t, kbFeatures["collections:create"], "should have collections:create in kb domain")
		assert.False(t, kbFeatures["profile:read"], "should not have user features in kb domain")
	})

	// Test GetFeatures for team member (with team_id)
	t.Run("GetFeatures_TeamMember", func(t *testing.T) {
		// Create mock gin.Context with team_id
		c := createMockGinContext(memberUserID, teamID)

		// Call GetFeatures
		features, err := GetFeatures(c)
		require.NoError(t, err)
		assert.NotEmpty(t, features, "team member should have features")

		// Verify team:admin features
		assert.True(t, features["profile:read"], "should have profile:read")
		assert.True(t, features["team:edit"], "should have team:edit")
		assert.True(t, features["collections:create"], "should have collections:create")
	})

	// Test GetFeaturesByDomain for team member
	t.Run("GetFeaturesByDomain_TeamMember", func(t *testing.T) {
		// Create mock gin.Context with team_id
		c := createMockGinContext(memberUserID, teamID)

		// Call GetFeaturesByDomain with "user" domain
		userFeatures, err := GetFeaturesByDomain(c, "user")
		require.NoError(t, err)
		assert.NotEmpty(t, userFeatures, "team member should have user domain features")

		// Should include user domain features
		assert.True(t, userFeatures["profile:read"], "should have profile:read")
		assert.True(t, userFeatures["team:edit"], "should have team:edit")

		// Should NOT include kb features
		assert.False(t, userFeatures["collections:create"], "should not have kb features in user domain")
	})
}
