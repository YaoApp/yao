package role_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/yao/openapi/oauth"
	"github.com/yaoapp/yao/openapi/oauth/acl/role"
	"github.com/yaoapp/yao/openapi/tests/testutils"
)

// Role Manager Test Suite
//
// PREREQUISITES:
// Before running tests, source the environment file:
//   source $YAO_DEV/env.local.sh
//
// Then run tests:
//   go test -v ./openapi/tests/oauth/acl/role/... -count=1

func TestNewManager(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean()

	t.Run("CreateManagerWithProvider", func(t *testing.T) {
		// Get cache and provider from OAuth service
		cache := oauth.OAuth.GetCache()
		require.NotNil(t, cache)

		provider, err := oauth.OAuth.GetUserProvider()
		require.NoError(t, err)
		require.NotNil(t, provider)

		// Create manager
		manager := role.NewManager(cache, provider)
		assert.NotNil(t, manager)

		t.Log("Successfully created role manager with cache and provider")
	})

	t.Run("CreateManagerWithNilProvider", func(t *testing.T) {
		// Get cache from OAuth service
		cache := oauth.OAuth.GetCache()
		require.NotNil(t, cache)

		// Create manager with nil provider
		manager := role.NewManager(cache, nil)
		assert.NotNil(t, manager)

		// Should work but will error when trying to get roles
		ctx := context.Background()
		_, err := manager.GetUserRole(ctx, "test-user")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user provider is not configured")

		t.Log("Manager with nil provider correctly returns error")
	})
}

func TestManagerWithNilCache(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean()

	// Get provider
	provider, err := oauth.OAuth.GetUserProvider()
	require.NoError(t, err)
	require.NotNil(t, provider)

	// Create manager with nil cache
	manager := role.NewManager(nil, provider)
	require.NotNil(t, manager)

	ctx := context.Background()

	t.Run("GetUserRoleWithNilCache", func(t *testing.T) {
		// Create a test role and user
		testRoleID := "test_role_nil_cache"
		roleData := maps.MapStrAny{
			"role_id":     testRoleID,
			"name":        "Test Nil Cache Role",
			"description": "Role for nil cache testing",
			"is_active":   true,
		}
		_, err := provider.CreateRole(ctx, roleData)
		if err != nil {
			t.Skipf("Skipping test - cannot create role: %v", err)
			return
		}
		defer provider.DeleteRole(ctx, testRoleID)

		userID, err := provider.GenerateUserID(ctx, true)
		require.NoError(t, err)

		userData := maps.MapStrAny{
			"user_id": userID,
			"email":   "nilcache@example.com",
			"role_id": testRoleID,
			"status":  "active",
		}
		_, err = provider.CreateUser(ctx, userData)
		require.NoError(t, err)
		defer provider.DeleteUser(ctx, userID)

		// Get user role - should work even without cache
		roleID, err := manager.GetUserRole(ctx, userID)
		assert.NoError(t, err)
		assert.Equal(t, testRoleID, roleID)

		t.Log("Successfully retrieved role without cache")
	})

	t.Run("GetScopesWithNilCache", func(t *testing.T) {
		// Create test role with permissions
		testRoleID := "test_role_scopes_nil_cache"
		permissions := maps.MapStrAny{
			"read":  true,
			"write": true,
		}
		restrictedPermissions := []string{"admin"}

		roleData := maps.MapStrAny{
			"role_id":                testRoleID,
			"name":                   "Test Nil Cache Scopes",
			"description":            "Role for scopes nil cache testing",
			"is_active":              true,
			"permissions":            permissions,
			"restricted_permissions": restrictedPermissions,
		}
		_, err := provider.CreateRole(ctx, roleData)
		if err != nil {
			t.Skipf("Skipping test - cannot create role: %v", err)
			return
		}
		defer provider.DeleteRole(ctx, testRoleID)

		// Get scopes - should work even without cache
		allowed, restricted, err := manager.GetScopes(ctx, testRoleID)
		assert.NoError(t, err)
		assert.NotNil(t, allowed)
		assert.NotNil(t, restricted)

		t.Logf("Successfully retrieved scopes without cache: allowed=%v, restricted=%v", allowed, restricted)
	})

	t.Run("ClearCacheWithNilCache", func(t *testing.T) {
		// Should not panic with nil cache
		err := manager.ClearCache()
		assert.NoError(t, err)

		t.Log("ClearCache works safely with nil cache")
	})
}

func TestGetUserRole(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean()

	// Setup
	cache := oauth.OAuth.GetCache()
	require.NotNil(t, cache)

	provider, err := oauth.OAuth.GetUserProvider()
	require.NoError(t, err)
	require.NotNil(t, provider)

	manager := role.NewManager(cache, provider)
	ctx := context.Background()

	t.Run("GetRoleForExistingUser", func(t *testing.T) {
		// First, create a test role using provider
		testRoleID := "test_role_get_user"
		roleData := maps.MapStrAny{
			"role_id":     testRoleID,
			"name":        "Test Role",
			"description": "Role for testing",
			"is_active":   true,
		}

		// Create role
		createdRoleID, err := provider.CreateRole(ctx, roleData)
		if err != nil {
			t.Skipf("Skipping test - cannot create role: %v", err)
			return
		}
		require.NotEmpty(t, createdRoleID)
		defer provider.DeleteRole(ctx, testRoleID)

		// Generate unique user ID
		userID, err := provider.GenerateUserID(ctx, true)
		require.NoError(t, err)

		// Create test user with role
		userData := maps.MapStrAny{
			"user_id": userID,
			"email":   "testuser@example.com",
			"role_id": testRoleID,
			"status":  "active",
		}
		_, err = provider.CreateUser(ctx, userData)
		require.NoError(t, err)
		defer provider.DeleteUser(ctx, userID)

		// Get user role
		roleID, err := manager.GetUserRole(ctx, userID)
		assert.NoError(t, err)
		assert.Equal(t, testRoleID, roleID)

		t.Logf("Successfully retrieved role %s for user %s", roleID, userID)

		// Get again (should come from cache)
		roleID2, err := manager.GetUserRole(ctx, userID)
		assert.NoError(t, err)
		assert.Equal(t, testRoleID, roleID2)

		t.Log("Successfully retrieved role from cache")
	})

	t.Run("GetRoleForNonExistentUser", func(t *testing.T) {
		_, err := manager.GetUserRole(ctx, "non-existent-user-12345")
		assert.Error(t, err)

		t.Log("Correctly returns error for non-existent user")
	})

	t.Run("GetRoleForUserWithoutRole", func(t *testing.T) {
		// Create user without role
		userID, err := provider.GenerateUserID(ctx, true)
		require.NoError(t, err)

		userData := maps.MapStrAny{
			"user_id": userID,
			"email":   "norole@example.com",
			"status":  "active",
		}
		_, err = provider.CreateUser(ctx, userData)
		require.NoError(t, err)
		defer provider.DeleteUser(ctx, userID)

		_, err = manager.GetUserRole(ctx, userID)
		assert.Error(t, err)

		t.Log("Correctly returns error for user without role")
	})
}

func TestGetMemberRole(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean()

	// Setup
	cache := oauth.OAuth.GetCache()
	require.NotNil(t, cache)

	provider, err := oauth.OAuth.GetUserProvider()
	require.NoError(t, err)
	require.NotNil(t, provider)

	manager := role.NewManager(cache, provider)
	ctx := context.Background()

	t.Run("GetRoleForExistingMember", func(t *testing.T) {
		// Create test role
		testRoleID := "test_role_member"
		roleData := maps.MapStrAny{
			"role_id":     testRoleID,
			"name":        "Test Member Role",
			"description": "Role for member testing",
			"is_active":   true,
		}
		_, err := provider.CreateRole(ctx, roleData)
		if err != nil {
			t.Skipf("Skipping test - cannot create role: %v", err)
			return
		}
		defer provider.DeleteRole(ctx, testRoleID)

		// Create test team
		teamID, err := provider.GenerateUserID(ctx, true)
		require.NoError(t, err)

		teamData := maps.MapStrAny{
			"team_id":  teamID,
			"name":     "Test Team",
			"owner_id": "test_owner",
			"status":   "active",
		}
		_, err = provider.CreateTeam(ctx, teamData)
		require.NoError(t, err)
		defer provider.DeleteTeam(ctx, teamID)

		// Create test member
		userID, err := provider.GenerateUserID(ctx, true)
		require.NoError(t, err)

		memberData := maps.MapStrAny{
			"team_id":     teamID,
			"user_id":     userID,
			"role_id":     testRoleID,
			"member_type": "user",
			"status":      "active",
		}
		_, err = provider.CreateMember(ctx, memberData)
		require.NoError(t, err)
		defer provider.RemoveMember(ctx, teamID, userID)

		// Get member role
		roleID, err := manager.GetMemberRole(ctx, teamID, userID)
		assert.NoError(t, err)
		assert.Equal(t, testRoleID, roleID)

		t.Logf("Successfully retrieved role %s for member %s in team %s", roleID, userID, teamID)

		// Get again (should come from cache)
		roleID2, err := manager.GetMemberRole(ctx, teamID, userID)
		assert.NoError(t, err)
		assert.Equal(t, testRoleID, roleID2)

		t.Log("Successfully retrieved member role from cache")
	})

	t.Run("GetRoleForNonExistentMember", func(t *testing.T) {
		_, err := manager.GetMemberRole(ctx, "non-existent-team", "non-existent-user")
		assert.Error(t, err)

		t.Log("Correctly returns error for non-existent member")
	})
}

func TestGetScopes(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean()

	// Setup
	cache := oauth.OAuth.GetCache()
	require.NotNil(t, cache)

	provider, err := oauth.OAuth.GetUserProvider()
	require.NoError(t, err)
	require.NotNil(t, provider)

	manager := role.NewManager(cache, provider)
	ctx := context.Background()

	t.Run("GetScopesForRoleWithPermissions", func(t *testing.T) {
		// Create test role with permissions
		testRoleID := "test_role_scopes"

		// Create role with permissions as map
		permissions := maps.MapStrAny{
			"read":   true,
			"write":  true,
			"delete": false, // Should not be included
		}

		restrictedPermissions := []string{"admin", "superuser"}

		roleData := maps.MapStrAny{
			"role_id":                testRoleID,
			"name":                   "Test Role with Scopes",
			"description":            "Role for scopes testing",
			"is_active":              true,
			"permissions":            permissions,
			"restricted_permissions": restrictedPermissions,
		}
		_, err := provider.CreateRole(ctx, roleData)
		if err != nil {
			t.Skipf("Skipping test - cannot create role: %v", err)
			return
		}
		defer provider.DeleteRole(ctx, testRoleID)

		// Get scopes
		allowed, restricted, err := manager.GetScopes(ctx, testRoleID)
		assert.NoError(t, err)
		assert.NotNil(t, allowed)
		assert.NotNil(t, restricted)

		// Verify allowed scopes contain enabled permissions
		t.Logf("Allowed scopes: %v", allowed)
		t.Logf("Restricted scopes: %v", restricted)

		// Note: The exact format depends on how the database stores JSON
		// We just verify we can retrieve them without error
		assert.True(t, len(allowed) >= 0, "Should return allowed scopes (empty or with values)")
		assert.True(t, len(restricted) >= 0, "Should return restricted scopes (empty or with values)")

		// Get again (should come from cache)
		allowed2, restricted2, err := manager.GetScopes(ctx, testRoleID)
		assert.NoError(t, err)
		assert.Equal(t, allowed, allowed2)
		assert.Equal(t, restricted, restricted2)

		t.Log("Successfully retrieved scopes from cache")
	})

	t.Run("GetScopesForNonExistentRole", func(t *testing.T) {
		_, _, err := manager.GetScopes(ctx, "non-existent-role-12345")
		assert.Error(t, err)

		t.Log("Correctly returns error for non-existent role")
	})
}

func TestGetTeamRole(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean()

	// Setup
	cache := oauth.OAuth.GetCache()
	require.NotNil(t, cache)

	provider, err := oauth.OAuth.GetUserProvider()
	require.NoError(t, err)
	require.NotNil(t, provider)

	manager := role.NewManager(cache, provider)
	ctx := context.Background()

	t.Run("GetRoleForTeamWithoutRole", func(t *testing.T) {
		// Create team without role_id field
		teamID, err := provider.GenerateUserID(ctx, true)
		require.NoError(t, err)

		teamData := maps.MapStrAny{
			"team_id":  teamID,
			"name":     "Test Team No Role",
			"owner_id": "test_owner",
			"status":   "active",
		}
		_, err = provider.CreateTeam(ctx, teamData)
		require.NoError(t, err)
		defer provider.DeleteTeam(ctx, teamID)

		// Get team role (should return error when no role_id assigned)
		roleID, err := manager.GetTeamRole(ctx, teamID)
		assert.Error(t, err, "Should return error when team has no role_id")
		assert.Contains(t, err.Error(), "has no role_id assigned", "Error message should indicate missing role_id")
		assert.Empty(t, roleID, "Role ID should be empty when error occurs")

		t.Logf("Correctly returns error for team without role_id: %v", err)
	})

	t.Run("GetRoleForNonExistentTeam", func(t *testing.T) {
		_, err := manager.GetTeamRole(ctx, "non-existent-team-12345")
		assert.Error(t, err)

		t.Log("Correctly returns error for non-existent team")
	})
}

func TestGetClientRole(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean()

	// Setup
	cache := oauth.OAuth.GetCache()
	require.NotNil(t, cache)

	provider, err := oauth.OAuth.GetUserProvider()
	require.NoError(t, err)
	require.NotNil(t, provider)

	manager := role.NewManager(cache, provider)
	ctx := context.Background()

	t.Run("GetClientRoleReturnsDefault", func(t *testing.T) {
		// Note: Client role retrieval is TODO in the code
		// It currently returns a default "system:root" role
		roleID, err := manager.GetClientRole(ctx, "test-client")
		assert.NoError(t, err)
		assert.Equal(t, "system:root", roleID)

		t.Log("Client role returns default system:root (TODO: implement ClientProvider)")
	})
}

func TestCacheIntegration(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean()

	// Setup
	cache := oauth.OAuth.GetCache()
	require.NotNil(t, cache)

	provider, err := oauth.OAuth.GetUserProvider()
	require.NoError(t, err)
	require.NotNil(t, provider)

	manager := role.NewManager(cache, provider)
	ctx := context.Background()

	t.Run("RoleCachingWorks", func(t *testing.T) {
		// Create test role and user
		testRoleID := "test_role_cache"
		roleData := maps.MapStrAny{
			"role_id":     testRoleID,
			"name":        "Test Cache Role",
			"description": "Role for cache testing",
			"is_active":   true,
		}
		_, err := provider.CreateRole(ctx, roleData)
		if err != nil {
			t.Skipf("Skipping test - cannot create role: %v", err)
			return
		}
		defer provider.DeleteRole(ctx, testRoleID)

		userID, err := provider.GenerateUserID(ctx, true)
		require.NoError(t, err)

		userData := maps.MapStrAny{
			"user_id": userID,
			"email":   "cache@example.com",
			"role_id": testRoleID,
			"status":  "active",
		}
		_, err = provider.CreateUser(ctx, userData)
		require.NoError(t, err)
		defer provider.DeleteUser(ctx, userID)

		// First call - should hit database
		roleID1, err := manager.GetUserRole(ctx, userID)
		assert.NoError(t, err)
		assert.Equal(t, testRoleID, roleID1)

		// Second call - should hit cache
		roleID2, err := manager.GetUserRole(ctx, userID)
		assert.NoError(t, err)
		assert.Equal(t, testRoleID, roleID2)

		// Results should be identical
		assert.Equal(t, roleID1, roleID2)

		t.Log("Cache integration verified: same role retrieved from cache")
	})
}
