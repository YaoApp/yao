package acl_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
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
}

// TestEnforceReturnValues tests Enforce return values when access is denied
// Note: HTTP response format is handled by Guard middleware, not by Enforce
func TestEnforceReturnValues(t *testing.T) {
	t.Run("DeniedAccessReturnValues", func(t *testing.T) {
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

// TestEnforceIntegration tests the complete enforcement flow
func TestEnforceIntegration(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean()

	t.Run("CompleteFlow", func(t *testing.T) {
		// Create enabled ACL
		config := &acl.Config{
			Enabled: true,
		}

		aclEnforcer, err := acl.New(config)

		if err != nil {
			t.Skipf("Skipping integration test: ACL initialization failed: %v", err)
			return
		}

		// Test cases with real endpoints from yao-dev-app scopes configuration
		testCases := []struct {
			name     string
			method   string
			path     string
			scopes   []string
			expected string // "allow" or "deny" or "unknown"
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
				expected: "allow", // GET /kb/* allow from scopes.yml (wildcard match, no specific scope defined)
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

				// Verify Enforce return values (not HTTP response format)
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
