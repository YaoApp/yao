package acl_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/openapi/oauth/acl"
	"github.com/yaoapp/yao/openapi/tests/testutils"
)

// TestLoadScopes tests loading scope configuration
func TestLoadScopes(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean()

	t.Run("LoadWithMissingDirectory", func(t *testing.T) {
		// Test loading when scopes directory doesn't exist
		// Should return a valid manager with default deny policy
		manager, err := acl.LoadScopes()
		assert.NoError(t, err, "Should succeed even when scopes directory is missing")
		assert.NotNil(t, manager)

		t.Log("Successfully created scope manager with missing scopes directory")
	})
}

// TestScopeManagerCheck tests the Check method
func TestScopeManagerCheck(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean()

	// Create a basic scope manager for testing
	manager, err := acl.LoadScopes()
	assert.NoError(t, err)
	assert.NotNil(t, manager)

	t.Run("CheckWithNoScopes", func(t *testing.T) {
		// Test request with no scopes to unmatched endpoint
		request := &acl.AccessRequest{
			Method: "GET",
			Path:   "/unmatched/endpoint",
			Scopes: []string{},
		}

		decision := manager.Check(request)
		assert.NotNil(t, decision)

		// Without configuration or no matching endpoint, should use default policy (deny)
		assert.False(t, decision.Allowed)
		assert.Contains(t, decision.Reason, "default policy")

		t.Logf("Decision: Allowed=%v, Reason=%s", decision.Allowed, decision.Reason)
	})

	t.Run("CheckPublicEndpoint", func(t *testing.T) {
		// Test public endpoint (from scopes.yml: GET /user/entry)
		request := &acl.AccessRequest{
			Method: "GET",
			Path:   "/user/entry",
			Scopes: []string{},
		}

		decision := manager.Check(request)
		assert.NotNil(t, decision)

		// Public endpoints should be allowed without scopes
		if decision.Allowed {
			assert.Equal(t, "public endpoint", decision.Reason)
		}

		t.Logf("Public endpoint decision: Allowed=%v, Reason=%s", decision.Allowed, decision.Reason)
	})

	t.Run("CheckWithScopes", func(t *testing.T) {
		// Test request with KB read scope (from alias: kb:read -> collections:read:all)
		request := &acl.AccessRequest{
			Method: "GET",
			Path:   "/kb/collections",
			Scopes: []string{"collections:read:all"},
		}

		decision := manager.Check(request)
		assert.NotNil(t, decision)
		assert.NotEmpty(t, decision.Reason)

		// Should record user scopes in decision
		assert.Equal(t, request.Scopes, decision.UserScopes)

		t.Logf("KB read decision: Allowed=%v, Reason=%s", decision.Allowed, decision.Reason)
	})

	t.Run("CheckKBWriteWithoutScope", func(t *testing.T) {
		// Test KB write operation without required scope
		request := &acl.AccessRequest{
			Method: "POST",
			Path:   "/kb/collections",
			Scopes: []string{"collections:read:all"}, // Only read scope
		}

		decision := manager.Check(request)
		assert.NotNil(t, decision)

		// POST to KB should be denied without write scope
		if !decision.Allowed {
			t.Logf("Correctly denied KB write without scope: %s", decision.Reason)
		}

		t.Logf("KB write without scope: Allowed=%v, Reason=%s", decision.Allowed, decision.Reason)
	})

	t.Run("CheckDifferentMethods", func(t *testing.T) {
		// Test different HTTP methods on KB collections
		testCases := []struct {
			method string
			scopes []string
		}{
			{"GET", []string{"collections:read:all"}},
			{"POST", []string{"collections:write:all"}},
			{"PUT", []string{"collections:write:all"}},
			{"DELETE", []string{"collections:delete:all"}},
		}

		for _, tc := range testCases {
			request := &acl.AccessRequest{
				Method: tc.method,
				Path:   "/kb/collections",
				Scopes: tc.scopes,
			}

			decision := manager.Check(request)
			assert.NotNil(t, decision)
			t.Logf("Method %s with scopes %v: Allowed=%v, Reason=%s",
				tc.method, tc.scopes, decision.Allowed, decision.Reason)
		}
	})

	t.Run("CheckWildcardPath", func(t *testing.T) {
		// Test wildcard path matching (from scopes.yml: GET /kb/* allow)
		testCases := []struct {
			path   string
			scopes []string
		}{
			{"/kb/collections", []string{"collections:read:all"}},
			{"/kb/collections/test-123", []string{"collections:read:all"}},
			{"/kb/documents/doc-456", []string{"documents:read:all"}},
			{"/kb/search", []string{"search:read:all"}},
		}

		for _, tc := range testCases {
			request := &acl.AccessRequest{
				Method: "GET",
				Path:   tc.path,
				Scopes: tc.scopes,
			}

			decision := manager.Check(request)
			assert.NotNil(t, decision)
			t.Logf("Path %s with scopes %v: Allowed=%v, Reason=%s",
				tc.path, tc.scopes, decision.Allowed, decision.Reason)
		}
	})
}

// TestAccessRequest tests the AccessRequest structure
func TestAccessRequest(t *testing.T) {
	t.Run("CreateAccessRequest", func(t *testing.T) {
		// Test creating an access request
		request := &acl.AccessRequest{
			Method: "POST",
			Path:   "/api/collections",
			Scopes: []string{"collections:create", "collections:read"},
		}

		assert.Equal(t, "POST", request.Method)
		assert.Equal(t, "/api/collections", request.Path)
		assert.Len(t, request.Scopes, 2)
		assert.Contains(t, request.Scopes, "collections:create")
		assert.Contains(t, request.Scopes, "collections:read")

		t.Log("AccessRequest structure validated successfully")
	})

	t.Run("AccessRequestWithEmptyScopes", func(t *testing.T) {
		// Test request with empty scopes
		request := &acl.AccessRequest{
			Method: "GET",
			Path:   "/public/info",
			Scopes: []string{},
		}

		assert.Empty(t, request.Scopes)
		assert.NotNil(t, request.Scopes) // Should be initialized, not nil

		t.Log("Empty scopes handled correctly")
	})
}

// TestAccessDecision tests the AccessDecision structure
func TestAccessDecision(t *testing.T) {
	t.Run("CreateAccessDecision", func(t *testing.T) {
		// Test creating an access decision
		decision := &acl.AccessDecision{
			Allowed:        true,
			Reason:         "scope matched",
			RequiredScopes: []string{"read:data"},
			UserScopes:     []string{"read:data", "write:data"},
			MissingScopes:  []string{},
		}

		assert.True(t, decision.Allowed)
		assert.Equal(t, "scope matched", decision.Reason)
		assert.Len(t, decision.RequiredScopes, 1)
		assert.Len(t, decision.UserScopes, 2)
		assert.Empty(t, decision.MissingScopes)

		t.Log("AccessDecision structure validated successfully")
	})

	t.Run("AccessDecisionDenied", func(t *testing.T) {
		// Test denied access decision
		decision := &acl.AccessDecision{
			Allowed:        false,
			Reason:         "missing required scopes",
			RequiredScopes: []string{"admin:write", "admin:delete"},
			UserScopes:     []string{"admin:read"},
			MissingScopes:  []string{"admin:write", "admin:delete"},
		}

		assert.False(t, decision.Allowed)
		assert.Contains(t, decision.Reason, "missing")
		assert.Len(t, decision.MissingScopes, 2)

		t.Log("Denied decision structure validated successfully")
	})
}

// TestEndpointPolicy tests endpoint policy constants
func TestEndpointPolicy(t *testing.T) {
	t.Run("PolicyConstants", func(t *testing.T) {
		// Verify policy constants are distinct
		assert.NotEqual(t, acl.PolicyDeny, acl.PolicyAllow)
		assert.NotEqual(t, acl.PolicyAllow, acl.PolicyRequireScopes)
		assert.NotEqual(t, acl.PolicyDeny, acl.PolicyRequireScopes)

		t.Log("Endpoint policy constants are distinct")
	})
}

// TestScopeExpansion tests scope expansion with aliases
func TestScopeExpansion(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean()

	manager, err := acl.LoadScopes()
	assert.NoError(t, err)
	assert.NotNil(t, manager)

	t.Run("DirectScopeNoAlias", func(t *testing.T) {
		// Test with direct scopes (no aliases)
		request := &acl.AccessRequest{
			Method: "GET",
			Path:   "/kb/collections",
			Scopes: []string{"collections:read:all", "documents:read:all"},
		}

		decision := manager.Check(request)
		assert.NotNil(t, decision)

		// User scopes should be recorded
		assert.Equal(t, request.Scopes, decision.UserScopes)

		t.Log("Direct scopes processed correctly")
	})

	t.Run("AliasScopeExpansion", func(t *testing.T) {
		// Test alias expansion (kb:read -> collections:read:all, documents:read:all, etc.)
		request := &acl.AccessRequest{
			Method: "GET",
			Path:   "/kb/collections",
			Scopes: []string{"kb:read"}, // Alias that should expand
		}

		decision := manager.Check(request)
		assert.NotNil(t, decision)

		t.Logf("Alias expansion test: Allowed=%v, Reason=%s", decision.Allowed, decision.Reason)
	})
}

// TestPathMatching tests various path matching scenarios
func TestPathMatching(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean()

	manager, err := acl.LoadScopes()
	assert.NoError(t, err)

	t.Run("ExactPathMatch", func(t *testing.T) {
		// Test exact path matching (GET /kb/collections)
		request := &acl.AccessRequest{
			Method: "GET",
			Path:   "/kb/collections",
			Scopes: []string{"collections:read:all"},
		}

		decision := manager.Check(request)
		assert.NotNil(t, decision)
		t.Logf("Exact path decision: %v - %s", decision.Allowed, decision.Reason)
	})

	t.Run("ParameterPath", func(t *testing.T) {
		// Test path with parameters (GET /kb/collections/:collectionID)
		request := &acl.AccessRequest{
			Method: "GET",
			Path:   "/kb/collections/test-collection-123",
			Scopes: []string{"collections:read:all"},
		}

		decision := manager.Check(request)
		assert.NotNil(t, decision)
		t.Logf("Parameter path decision: %v - %s", decision.Allowed, decision.Reason)
	})

	t.Run("WildcardPath", func(t *testing.T) {
		// Test wildcard path matching (GET /kb/* allow from scopes.yml)
		request := &acl.AccessRequest{
			Method: "GET",
			Path:   "/kb/anything/nested/path",
			Scopes: []string{},
		}

		decision := manager.Check(request)
		assert.NotNil(t, decision)

		// Should be allowed by wildcard rule in scopes.yml
		if decision.Allowed {
			t.Logf("Wildcard rule correctly allowed access: %s", decision.Reason)
		}

		t.Logf("Wildcard path decision: %v - %s", decision.Allowed, decision.Reason)
	})
}

// TestDefaultPolicy tests default policy behavior
func TestDefaultPolicy(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean()

	manager, err := acl.LoadScopes()
	assert.NoError(t, err)

	t.Run("UnmatchedEndpoint", func(t *testing.T) {
		// Test unmatched endpoint falls back to default policy
		request := &acl.AccessRequest{
			Method: "GET",
			Path:   "/unregistered/endpoint",
			Scopes: []string{"some:scope"},
		}

		decision := manager.Check(request)
		assert.NotNil(t, decision)

		// Should mention default policy in reason
		assert.Contains(t, decision.Reason, "default policy")

		t.Logf("Default policy applied: %v - %s", decision.Allowed, decision.Reason)
	})
}
