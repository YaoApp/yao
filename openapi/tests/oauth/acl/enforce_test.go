package acl_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/openapi/oauth"
	"github.com/yaoapp/yao/openapi/oauth/acl"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
	"github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/openapi/tests/testutils"
)

// setupGinContext creates a test gin context with authorized info
func setupGinContext(method, path string, scopes []string) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Create test request
	req, _ := http.NewRequest(method, path, nil)
	c.Request = req

	// Set authorized info in context
	authInfo := &types.AuthorizedInfo{
		Subject:  "test-subject",
		ClientID: "test-client",
		UserID:   "test-user",
		Scope:    joinScopes(scopes),
	}

	// Simulate what authorized.SetInfo would do
	c.Set("__subject", authInfo.Subject)
	c.Set("__client_id", authInfo.ClientID)
	c.Set("__user_id", authInfo.UserID)
	c.Set("__scope", authInfo.Scope)

	return c, w
}

// joinScopes joins scopes array into space-separated string
func joinScopes(scopes []string) string {
	if len(scopes) == 0 {
		return ""
	}
	result := scopes[0]
	for i := 1; i < len(scopes); i++ {
		result += " " + scopes[i]
	}
	return result
}

// TestEnforce tests the Enforce method
func TestEnforce(t *testing.T) {
	t.Run("EnforceWithDisabledACL", func(t *testing.T) {
		// Create disabled ACL
		config := &acl.Config{
			Enabled: false,
		}

		aclEnforcer, err := acl.New(config)
		assert.NoError(t, err)

		// Setup test context
		c, _ := setupGinContext("GET", "/test/endpoint", []string{"read:test"})

		// Enforce should allow access when ACL is disabled
		allowed, err := aclEnforcer.Enforce(c)
		assert.NoError(t, err)
		assert.True(t, allowed, "Should allow access when ACL is disabled")

		t.Log("Disabled ACL correctly allows all access")
	})

	t.Run("EnforceWithEnabledACLNoScope", func(t *testing.T) {
		testutils.Prepare(t)
		defer testutils.Clean()

		// Create enabled ACL
		config := &acl.Config{
			Enabled: true,
		}

		aclEnforcer, err := acl.New(config)

		// May fail if scopes directory doesn't exist
		if err != nil {
			t.Skipf("Skipping test: ACL initialization failed (expected if scopes directory missing): %v", err)
			return
		}

		// Setup test context with no scopes
		c, _ := setupGinContext("GET", "/test/endpoint", []string{})

		// Enforce should deny access and return error (no scope for unmatched endpoint)
		allowed, err := aclEnforcer.Enforce(c)
		assert.False(t, allowed, "Should deny access for unmatched endpoint")
		assert.Error(t, err, "Should return error when access is denied")

		if err != nil {
			t.Logf("Access correctly denied: %v", err)
		}
	})

	t.Run("EnforceWithScopes", func(t *testing.T) {
		testutils.Prepare(t)
		defer testutils.Clean()

		// Create enabled ACL
		config := &acl.Config{
			Enabled: true,
		}

		aclEnforcer, err := acl.New(config)

		if err != nil {
			t.Skipf("Skipping test: ACL initialization failed: %v", err)
			return
		}

		// Setup test context with scopes for unmatched endpoint
		c, _ := setupGinContext("GET", "/api/users", []string{"read:users", "write:users"})

		// Enforce should deny (unmatched endpoint, default policy: deny)
		allowed, err := aclEnforcer.Enforce(c)
		assert.False(t, allowed, "Should deny access for unmatched endpoint")
		assert.Error(t, err, "Should return error when access is denied")

		t.Logf("Access correctly denied: %v", err)
	})

	t.Run("EnforceChecksContext", func(t *testing.T) {
		testutils.Prepare(t)
		defer testutils.Clean()

		// Test that Enforce extracts info from context correctly
		config := &acl.Config{
			Enabled: true,
		}

		aclEnforcer, err := acl.New(config)

		if err != nil {
			t.Skipf("Skipping test: ACL initialization failed: %v", err)
			return
		}

		// Setup context with specific authorized info
		c, _ := setupGinContext("POST", "/kb/collections", []string{
			"collections:create",
			"collections:read",
		})

		// Verify authorized info can be extracted
		authInfo := authorized.GetInfo(c)
		assert.NotNil(t, authInfo)
		assert.Equal(t, "test-user", authInfo.UserID)
		assert.Equal(t, "test-client", authInfo.ClientID)
		assert.Contains(t, authInfo.Scope, "collections:create")

		// Enforce - should deny because required scope is "collections:write:all"
		allowed, err := aclEnforcer.Enforce(c)
		assert.False(t, allowed, "Should deny access without required scope")
		assert.Error(t, err, "Should return error for missing required scopes")

		t.Logf("Access correctly denied: %v", err)
	})

	t.Run("EnforceUpdatesConstraints", func(t *testing.T) {
		testutils.Prepare(t)
		defer testutils.Clean()

		// Get user provider and set up test data
		ctx := context.Background()
		provider, err := oauth.OAuth.GetUserProvider()
		if err != nil || provider == nil {
			t.Skip("Skipping: user provider not available")
			return
		}

		// Set up test data
		testData := setupACLTestData(t, ctx, provider)
		defer cleanupACLTestData(t, ctx, provider, testData)

		// Use global ACL instance
		aclEnforcer := acl.Global
		if aclEnforcer == nil || !aclEnforcer.Enabled() {
			t.Skip("Skipping: ACL not enabled")
			return
		}

		// Test case 1: OwnerOnly constraint (profile:read:own has owner: true)
		t.Run("OwnerOnlyConstraint", func(t *testing.T) {
			c, _ := setupGinContext("GET", "/user/profile", []string{"profile:read:own"})

			allowed, err := aclEnforcer.Enforce(c)
			assert.NoError(t, err, "Should not return error")
			assert.True(t, allowed, "Should allow access with profile:read:own")

			// Get updated authorized info from context
			authInfo := authorized.GetInfo(c)
			assert.NotNil(t, authInfo, "AuthInfo should not be nil")

			// Verify OwnerOnly constraint was set
			assert.True(t, authInfo.Constraints.OwnerOnly,
				"OwnerOnly should be true for profile:read:own endpoint")
			assert.False(t, authInfo.Constraints.TeamOnly,
				"TeamOnly should be false for profile endpoint")

			t.Logf("✓ OwnerOnly constraint correctly set: OwnerOnly=%v, TeamOnly=%v",
				authInfo.Constraints.OwnerOnly, authInfo.Constraints.TeamOnly)
		})

		// Test case 2: TeamOnly constraint (collections:read:team has team: true)
		t.Run("TeamOnlyConstraint", func(t *testing.T) {
			c, _ := setupGinContext("GET", "/kb/collections/team", []string{"collections:read:team"})

			allowed, err := aclEnforcer.Enforce(c)
			assert.NoError(t, err, "Should not return error")
			assert.True(t, allowed, "Should allow access with collections:read:team")

			// Get updated authorized info from context
			authInfo := authorized.GetInfo(c)
			assert.NotNil(t, authInfo, "AuthInfo should not be nil")

			// Verify TeamOnly constraint was set
			assert.True(t, authInfo.Constraints.TeamOnly,
				"TeamOnly should be true for collections:read:team endpoint")
			assert.False(t, authInfo.Constraints.OwnerOnly,
				"OwnerOnly should be false for team endpoint")

			t.Logf("✓ TeamOnly constraint correctly set: OwnerOnly=%v, TeamOnly=%v",
				authInfo.Constraints.OwnerOnly, authInfo.Constraints.TeamOnly)
		})

		// Test case 3: No constraints (collections:read:all has no owner/team flags)
		t.Run("NoConstraints", func(t *testing.T) {
			c, _ := setupGinContext("GET", "/kb/collections", []string{"collections:read:all"})

			allowed, err := aclEnforcer.Enforce(c)
			assert.NoError(t, err, "Should not return error")
			assert.True(t, allowed, "Should allow access with collections:read:all")

			// Get updated authorized info from context
			authInfo := authorized.GetInfo(c)
			assert.NotNil(t, authInfo, "AuthInfo should not be nil")

			// Verify no constraints were set
			assert.False(t, authInfo.Constraints.OwnerOnly,
				"OwnerOnly should be false for unrestricted endpoint")
			assert.False(t, authInfo.Constraints.TeamOnly,
				"TeamOnly should be false for unrestricted endpoint")

			t.Logf("✓ No constraints for unrestricted endpoint: OwnerOnly=%v, TeamOnly=%v",
				authInfo.Constraints.OwnerOnly, authInfo.Constraints.TeamOnly)
		})

		// Test case 4: Both constraints (if such endpoint exists)
		t.Run("BothOwnerAndTeamConstraints", func(t *testing.T) {
			c, _ := setupGinContext("GET", "/kb/collections/own", []string{"collections:read:own"})

			allowed, err := aclEnforcer.Enforce(c)
			assert.NoError(t, err, "Should not return error")
			assert.True(t, allowed, "Should allow access with collections:read:own")

			// Get updated authorized info from context
			authInfo := authorized.GetInfo(c)
			assert.NotNil(t, authInfo, "AuthInfo should not be nil")

			// Verify OwnerOnly constraint was set (collections:read:own has owner: true)
			assert.True(t, authInfo.Constraints.OwnerOnly,
				"OwnerOnly should be true for collections:read:own endpoint")

			t.Logf("✓ Owner constraint for own collections: OwnerOnly=%v, TeamOnly=%v",
				authInfo.Constraints.OwnerOnly, authInfo.Constraints.TeamOnly)
		})
	})
}

// TestEnforceReturnValues tests Enforce return values when access is denied
// Note: HTTP response format is handled by Guard middleware, not by Enforce
func TestEnforceReturnValues(t *testing.T) {
	t.Run("DeniedAccessReturnValues", func(t *testing.T) {
		testutils.Prepare(t)
		defer testutils.Clean()

		config := &acl.Config{
			Enabled: true,
		}

		aclEnforcer, err := acl.New(config)

		if err != nil {
			t.Skipf("Skipping test: ACL initialization failed: %v", err)
			return
		}

		// Setup context with scopes for an unmatched endpoint
		c, _ := setupGinContext("POST", "/protected/admin", []string{"read:basic"})

		// Enforce should return false and an error
		allowed, err := aclEnforcer.Enforce(c)
		assert.False(t, allowed, "Should deny access")
		assert.Error(t, err, "Should return error when access is denied")

		// Verify error contains ACL error information
		if err != nil {
			aclErr, ok := err.(*acl.Error)
			assert.True(t, ok, "Error should be ACL Error type")
			if aclErr != nil {
				assert.NotEmpty(t, aclErr.Message, "Error should have message")
				t.Logf("Access correctly denied with error: %v", aclErr.Message)
			}
		}
	})
}

// TestGetScopes tests the internal getScopes function behavior
func TestGetScopes(t *testing.T) {
	t.Run("ExtractScopesFromContext", func(t *testing.T) {
		// Setup context with scopes
		c, _ := setupGinContext("GET", "/test", []string{
			"scope1",
			"scope2",
			"scope3",
		})

		// Get authorized info (which getScopes would use)
		authInfo := authorized.GetInfo(c)
		assert.NotNil(t, authInfo)

		// Verify scope string contains all scopes
		assert.Contains(t, authInfo.Scope, "scope1")
		assert.Contains(t, authInfo.Scope, "scope2")
		assert.Contains(t, authInfo.Scope, "scope3")

		t.Logf("Scope string: %s", authInfo.Scope)
	})

	t.Run("EmptyScopes", func(t *testing.T) {
		// Setup context with no scopes
		c, _ := setupGinContext("GET", "/test", []string{})

		authInfo := authorized.GetInfo(c)
		assert.NotNil(t, authInfo)
		assert.Empty(t, authInfo.Scope)

		t.Log("Empty scopes handled correctly")
	})

	t.Run("SingleScope", func(t *testing.T) {
		// Setup context with single scope
		c, _ := setupGinContext("GET", "/test", []string{"single:scope"})

		authInfo := authorized.GetInfo(c)
		assert.NotNil(t, authInfo)
		assert.Equal(t, "single:scope", authInfo.Scope)

		t.Log("Single scope handled correctly")
	})
}

// setupACLTestRoles creates test roles with permissions for ACL testing
func setupACLTestRoles(t *testing.T, ctx context.Context, provider types.UserProvider) {
	roles := []struct {
		roleID      string
		name        string
		description string
		permissions []string
		restricted  []string
	}{
		{
			roleID:      "system:root",
			name:        "System Root",
			description: "System root role with full access",
			permissions: []string{"*:*:*"},
			restricted:  []string{},
		},
		{
			roleID:      "acl_test_user",
			name:        "ACL Test User Role",
			description: "Role for ACL user testing",
			permissions: []string{
				"profile:read:own",
				"profile:write:own",
				"collections:read:all",
				"collections:write:all",
			},
			restricted: []string{},
		},
		{
			roleID:      "acl_test_team",
			name:        "ACL Test Team Role",
			description: "Role for ACL team testing",
			permissions: []string{
				"team:read:all",
				"team:write:all",
				"collections:read:team",
			},
			restricted: []string{},
		},
		{
			roleID:      "acl_test_member",
			name:        "ACL Test Member Role",
			description: "Role for ACL member testing",
			permissions: []string{
				"member:read:own",
				"member:write:own",
				"collections:read:own",
			},
			restricted: []string{
				"admin:access",
			},
		},
	}

	for _, role := range roles {
		// Create role
		roleData := map[string]interface{}{
			"role_id":     role.roleID,
			"name":        role.name,
			"description": role.description,
			"status":      "active",
		}
		_, err := provider.CreateRole(ctx, roleData)
		if err != nil {
			t.Logf("Warning: Failed to create role %s (may already exist): %v", role.roleID, err)
		}

		// Set role permissions
		permissions := map[string]interface{}{
			"permissions":            role.permissions,
			"restricted_permissions": role.restricted,
		}
		err = provider.SetRolePermissions(ctx, role.roleID, permissions)
		if err != nil {
			t.Logf("Warning: Failed to set permissions for role %s: %v", role.roleID, err)
		}
	}

	t.Log("Set up ACL test roles and permissions")
}

// cleanupACLTestRoles removes test roles created for ACL testing
func cleanupACLTestRoles(t *testing.T, ctx context.Context, provider types.UserProvider) {
	roles := []string{"system:root", "acl_test_user", "acl_test_team", "acl_test_member"}
	for _, roleID := range roles {
		err := provider.DeleteRole(ctx, roleID)
		if err != nil {
			t.Logf("Warning: Failed to delete test role %s: %v", roleID, err)
		}
	}
	t.Log("Cleaned up ACL test roles")
}

// setupACLTestData creates test users, teams, members and roles for ACL testing
func setupACLTestData(t *testing.T, ctx context.Context, provider types.UserProvider) *ACLTestData {
	data := &ACLTestData{
		UserIDs:   make([]string, 0),
		TeamIDs:   make([]string, 0),
		MemberIDs: make([]int64, 0),
	}

	// Set up roles first
	setupACLTestRoles(t, ctx, provider)

	// Create test users with different roles
	users := []struct {
		userID   string
		email    string
		username string
		roleID   string
	}{
		{"test-user", "testuser@acl.test", "testuser", "system:root"},          // For test-client in setupGinContext
		{"acl-user-1", "acluser1@acl.test", "acluser1", "acl_test_user"},       // Regular user
		{"acl-owner", "aclowner@acl.test", "aclowner", "acl_test_user"},        // Team owner
		{"acl-member-1", "aclmember1@acl.test", "aclmember1", "acl_test_user"}, // Team member
	}

	for _, u := range users {
		userData := map[string]interface{}{
			"user_id":            u.userID,
			"email":              u.email,
			"preferred_username": u.username,
			"password_hash":      "test_hash",
			"status":             "active",
			"role_id":            u.roleID,
		}
		userID, err := provider.CreateUser(ctx, userData)
		if err != nil {
			t.Logf("Warning: Failed to create user %s: %v", u.userID, err)
		} else {
			data.UserIDs = append(data.UserIDs, userID)
		}
	}

	// Create test team
	teamData := map[string]interface{}{
		"name":        "ACL Test Team",
		"description": "Team for ACL testing",
		"owner_id":    "acl-owner",
		"status":      "active",
		"role_id":     "acl_test_team",
	}
	teamID, err := provider.CreateTeam(ctx, teamData)
	if err != nil {
		t.Logf("Warning: Failed to create team: %v", err)
	} else {
		data.TeamIDs = append(data.TeamIDs, teamID)

		// Create team members
		members := []struct {
			userID string
			roleID string
		}{
			{"acl-owner", "acl_test_member"},    // Owner as member
			{"acl-member-1", "acl_test_member"}, // Regular member
		}

		for _, m := range members {
			memberData := map[string]interface{}{
				"team_id":     teamID,
				"user_id":     m.userID,
				"role_id":     m.roleID,
				"member_type": "user",
				"status":      "active",
			}
			memberID, err := provider.CreateMember(ctx, memberData)
			if err != nil {
				t.Logf("Warning: Failed to create member %s: %v", m.userID, err)
			} else {
				data.MemberIDs = append(data.MemberIDs, memberID)
			}
		}
	}

	t.Logf("Created ACL test data: %d users, %d teams, %d members",
		len(data.UserIDs), len(data.TeamIDs), len(data.MemberIDs))
	return data
}

// cleanupACLTestData removes all test data created for ACL testing
func cleanupACLTestData(t *testing.T, ctx context.Context, provider types.UserProvider, data *ACLTestData) {
	if data == nil {
		return
	}

	// Remove teams (this will cascade remove members)
	for _, teamID := range data.TeamIDs {
		err := provider.DeleteTeam(ctx, teamID)
		if err != nil {
			t.Logf("Warning: Failed to delete test team %s: %v", teamID, err)
		}
	}

	// Remove users
	for _, userID := range data.UserIDs {
		err := provider.DeleteUser(ctx, userID)
		if err != nil {
			t.Logf("Warning: Failed to delete test user %s: %v", userID, err)
		}
	}

	// Remove roles
	cleanupACLTestRoles(t, ctx, provider)

	t.Logf("Cleaned up ACL test data: %d users, %d teams",
		len(data.UserIDs), len(data.TeamIDs))
}

// ACLTestData holds test data created for ACL testing
type ACLTestData struct {
	UserIDs   []string
	TeamIDs   []string
	MemberIDs []int64
}

// TestEnforceIntegration tests the complete enforcement flow
// Note: This test only validates scope-based ACL when RoleManager is not configured
// For full role-based testing, see role package tests
func TestEnforceIntegration(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean()

	t.Run("ScopeBasedFlow", func(t *testing.T) {
		// Use the global ACL instance created by testutils.Prepare
		// This tests the real configuration loaded from the application
		aclEnforcer := acl.Global

		if aclEnforcer == nil || !aclEnforcer.Enabled() {
			t.Skip("Skipping integration test: ACL is not enabled")
			return
		}

		// Get user provider and set up complete test data
		ctx := context.Background()
		provider, err := oauth.OAuth.GetUserProvider()
		if err != nil || provider == nil {
			t.Skip("Skipping: user provider not available")
			return
		}

		// Set up test data and ensure cleanup
		testData := setupACLTestData(t, ctx, provider)
		defer cleanupACLTestData(t, ctx, provider, testData)

		// Test cases with real endpoints from yao-dev-app scopes configuration
		testCases := []struct {
			name     string
			method   string
			path     string
			scopes   []string
			expected string // "allow" or "deny"
		}{
			{
				name:     "PublicEndpoint",
				method:   "GET",
				path:     "/user/entry",
				scopes:   []string{},
				expected: "allow", // Public endpoint from scopes.yml
			},
			{
				name:     "PublicCaptcha",
				method:   "GET",
				path:     "/user/entry/captcha",
				scopes:   []string{},
				expected: "allow", // Public endpoint
			},
			{
				name:     "KBReadWithScope",
				method:   "GET",
				path:     "/kb/collections",
				scopes:   []string{"collections:read:all"},
				expected: "allow", // Should be allowed with scope
			},
			{
				name:     "KBWriteWithoutScope",
				method:   "POST",
				path:     "/kb/collections",
				scopes:   []string{"collections:read:all"},
				expected: "deny", // POST requires write scope
			},
			{
				name:     "KBWriteWithScope",
				method:   "POST",
				path:     "/kb/collections",
				scopes:   []string{"collections:write:all"},
				expected: "allow", // Should be allowed with write scope
			},
			{
				name:     "ProfileReadWithScope",
				method:   "GET",
				path:     "/user/profile",
				scopes:   []string{"profile:read:own"},
				expected: "allow", // Should be allowed
			},
			{
				name:     "WildcardAllowedRead",
				method:   "GET",
				path:     "/kb/some-other-resource/item-123",
				scopes:   []string{},
				expected: "allow", // GET /kb/* allow from scopes.yml
			},
			{
				name:     "UnmatchedEndpoint",
				method:   "GET",
				path:     "/unmatched/endpoint",
				scopes:   []string{"some:scope"},
				expected: "deny", // Default policy is deny
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				c, _ := setupGinContext(tc.method, tc.path, tc.scopes)

				allowed, err := aclEnforcer.Enforce(c)

				t.Logf("%s %s with scopes %v: allowed=%v",
					tc.method, tc.path, tc.scopes, allowed)

				// Verify Enforce return values
				if tc.expected == "allow" {
					assert.True(t, allowed, "Should allow access")
					assert.NoError(t, err, "Should not return error when access is allowed")
				} else {
					assert.False(t, allowed, "Should deny access")
					assert.Error(t, err, "Should return error when access is denied")
				}
			})
		}
	})
}

// TestEnforcerInterface tests that ACL implements the Enforcer interface
func TestEnforcerInterface(t *testing.T) {
	t.Run("ImplementsInterface", func(t *testing.T) {
		config := &acl.Config{
			Enabled: false,
		}

		enforcer, err := acl.New(config)
		assert.NoError(t, err)

		// Should implement Enforcer interface methods
		assert.Implements(t, (*acl.Enforcer)(nil), enforcer)

		t.Log("ACL correctly implements Enforcer interface")
	})
}

// TestEnforceEdgeCases tests edge cases in enforcement
func TestEnforceEdgeCases(t *testing.T) {
	t.Run("NilContext", func(t *testing.T) {
		config := &acl.Config{
			Enabled: false,
		}

		aclEnforcer, err := acl.New(config)
		assert.NoError(t, err)

		// Disabled ACL should handle nil context gracefully
		// (though this shouldn't happen in practice)
		c, _ := setupGinContext("GET", "/test", []string{})
		c.Request = nil // Simulate edge case

		// Should not panic
		assert.NotPanics(t, func() {
			// Disabled ACL returns early, so won't access c.Request
			_, _ = aclEnforcer.Enforce(c)
		})
	})

	t.Run("SpecialCharactersInPath", func(t *testing.T) {
		testutils.Prepare(t)
		defer testutils.Clean()

		config := &acl.Config{
			Enabled: true,
		}

		aclEnforcer, err := acl.New(config)

		if err != nil {
			t.Skipf("Skipping test: %v", err)
			return
		}

		// Test with special characters in path (unmatched endpoint)
		c, _ := setupGinContext("GET", "/api/users/%20with%20spaces", []string{"read:users"})

		allowed, err := aclEnforcer.Enforce(c)
		assert.False(t, allowed, "Should deny unmatched endpoint")
		assert.Error(t, err, "Should return error for unmatched endpoint")

		t.Logf("Path with special chars correctly denied: %v", err)
	})

	t.Run("VeryLongScope", func(t *testing.T) {
		testutils.Prepare(t)
		defer testutils.Clean()

		config := &acl.Config{
			Enabled: true,
		}

		aclEnforcer, err := acl.New(config)

		if err != nil {
			t.Skipf("Skipping test: %v", err)
			return
		}

		// Test with very long scope name (unmatched endpoint)
		longScope := "very:long:scope:name:with:many:segments:to:test:handling:of:long:strings"
		c, _ := setupGinContext("GET", "/test", []string{longScope})

		allowed, err := aclEnforcer.Enforce(c)
		assert.False(t, allowed, "Should deny unmatched endpoint")
		assert.Error(t, err, "Should return error for unmatched endpoint")

		t.Logf("Long scope correctly handled: %v", err)
	})
}
