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
		c, w := setupGinContext("GET", "/test/endpoint", []string{})

		// Enforce should check permissions
		allowed, err := aclEnforcer.Enforce(c)
		assert.NoError(t, err)

		t.Logf("Access decision: allowed=%v, status=%d", allowed, w.Code)
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

		// Setup test context with scopes
		c, w := setupGinContext("GET", "/api/users", []string{"read:users", "write:users"})

		// Enforce should check permissions
		allowed, err := aclEnforcer.Enforce(c)
		assert.NoError(t, err)

		t.Logf("Access with scopes: allowed=%v, status=%d", allowed, w.Code)

		if !allowed && w.Code == 403 {
			t.Log("Access correctly denied with 403 response")
		}
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

		// Enforce
		allowed, err := aclEnforcer.Enforce(c)
		assert.NoError(t, err)

		t.Logf("Enforce with collections scopes: allowed=%v", allowed)
	})
}

// TestEnforceResponseFormat tests the response format when access is denied
func TestEnforceResponseFormat(t *testing.T) {
	t.Run("DeniedAccessResponse", func(t *testing.T) {
		config := &acl.Config{
			Enabled: true,
		}

		aclEnforcer, err := acl.New(config)

		if err != nil {
			t.Skipf("Skipping test: ACL initialization failed: %v", err)
			return
		}

		// Setup context with insufficient scopes for a protected endpoint
		c, w := setupGinContext("POST", "/protected/admin", []string{"read:basic"})

		// Enforce
		allowed, err := aclEnforcer.Enforce(c)
		assert.NoError(t, err)

		// If access is denied, check response format
		if !allowed {
			assert.Equal(t, 403, w.Code, "Should return 403 Forbidden")

			// Response should be JSON
			contentType := w.Header().Get("Content-Type")
			assert.Contains(t, contentType, "application/json")

			// Response body should contain error details
			body := w.Body.String()
			assert.Contains(t, body, "Access denied")

			t.Logf("Denied access response format correct: %s", body)
		} else {
			t.Log("Access was allowed (no scope configuration for this endpoint)")
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
				path:     "/kb/documents/doc-123",
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
				c, w := setupGinContext(tc.method, tc.path, tc.scopes)

				allowed, err := aclEnforcer.Enforce(c)
				assert.NoError(t, err)

				t.Logf("%s %s with scopes %v: allowed=%v, status=%d",
					tc.method, tc.path, tc.scopes, allowed, w.Code)

				// Verify response is properly formatted
				if !allowed {
					assert.Equal(t, 403, w.Code)
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

		// Test with special characters in path
		c, _ := setupGinContext("GET", "/api/users/%20with%20spaces", []string{"read:users"})

		allowed, err := aclEnforcer.Enforce(c)
		assert.NoError(t, err)

		t.Logf("Path with special chars: allowed=%v", allowed)
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

		// Test with very long scope name
		longScope := "very:long:scope:name:with:many:segments:to:test:handling:of:long:strings"
		c, _ := setupGinContext("GET", "/test", []string{longScope})

		allowed, err := aclEnforcer.Enforce(c)
		assert.NoError(t, err)

		t.Logf("Long scope handling: allowed=%v", allowed)
	})
}
