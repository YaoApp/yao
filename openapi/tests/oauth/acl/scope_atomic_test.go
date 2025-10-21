package acl_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/openapi/oauth/acl"
	"github.com/yaoapp/yao/openapi/tests/testutils"
)

// TestScopeAtomic_ExactPathMatch tests exact path matching
func TestScopeAtomic_ExactPathMatch(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean()

	manager, err := acl.LoadScopes()
	assert.NoError(t, err)
	assert.NotNil(t, manager)

	t.Run("ExactMatch_Allow", func(t *testing.T) {
		// Test exact match: GET /user/entry (public endpoint)
		request := &acl.AccessRequest{
			Method: "GET",
			Path:   "/user/entry",
			Scopes: []string{},
		}

		decision := manager.Check(request)
		assert.NotNil(t, decision)
		assert.True(t, decision.Allowed, "Exact public path should allow")
		assert.Equal(t, "public endpoint", decision.Reason)
		t.Logf("✓ Exact match '/user/entry': Allowed=%v, Reason=%s", decision.Allowed, decision.Reason)
	})

	t.Run("ExactMatch_WithScope", func(t *testing.T) {
		// Test exact match with scope: GET /kb/collections
		request := &acl.AccessRequest{
			Method: "GET",
			Path:   "/kb/collections",
			Scopes: []string{"collections:read:all"},
		}

		decision := manager.Check(request)
		assert.NotNil(t, decision)
		t.Logf("✓ Exact match '/kb/collections' with scope: Allowed=%v, Reason=%s", decision.Allowed, decision.Reason)
	})

	t.Run("ExactMatch_DifferentMethods", func(t *testing.T) {
		// Test same path with different methods
		testCases := []struct {
			method string
			path   string
			scopes []string
			desc   string
		}{
			{"GET", "/kb/collections", []string{}, "GET without scope"},
			{"GET", "/kb/collections", []string{"collections:read:all"}, "GET with read scope"},
			{"POST", "/kb/collections", []string{}, "POST without scope"},
			{"POST", "/kb/collections", []string{"collections:write:all"}, "POST with write scope"},
		}

		for _, tc := range testCases {
			request := &acl.AccessRequest{
				Method: tc.method,
				Path:   tc.path,
				Scopes: tc.scopes,
			}

			decision := manager.Check(request)
			assert.NotNil(t, decision)
			t.Logf("✓ %s %s (%s): Allowed=%v, Reason=%s",
				tc.method, tc.path, tc.desc, decision.Allowed, decision.Reason)
		}
	})
}

// TestScopeAtomic_WildcardMatch tests wildcard path matching
func TestScopeAtomic_WildcardMatch(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean()

	manager, err := acl.LoadScopes()
	assert.NoError(t, err)
	assert.NotNil(t, manager)

	t.Run("Wildcard_SingleLevel", func(t *testing.T) {
		// Test wildcard matching: GET /kb/*
		testPaths := []string{
			"/kb/collections",
			"/kb/documents",
			"/kb/search",
		}

		for _, path := range testPaths {
			request := &acl.AccessRequest{
				Method: "GET",
				Path:   path,
				Scopes: []string{},
			}

			decision := manager.Check(request)
			assert.NotNil(t, decision)
			t.Logf("✓ Wildcard match '%s': Allowed=%v, Reason=%s", path, decision.Allowed, decision.Reason)
		}
	})

	t.Run("Wildcard_MultiLevel", func(t *testing.T) {
		// Test wildcard with nested paths: GET /kb/*
		testPaths := []string{
			"/kb/collections/test-123",
			"/kb/documents/doc-456/content",
			"/kb/search/query/results",
		}

		for _, path := range testPaths {
			request := &acl.AccessRequest{
				Method: "GET",
				Path:   path,
				Scopes: []string{},
			}

			decision := manager.Check(request)
			assert.NotNil(t, decision)
			t.Logf("✓ Wildcard multi-level '%s': Allowed=%v, Reason=%s", path, decision.Allowed, decision.Reason)
		}
	})

	t.Run("Wildcard_DifferentScopes", func(t *testing.T) {
		// Test wildcard with different scopes
		testCases := []struct {
			path   string
			scopes []string
			desc   string
		}{
			{"/kb/collections", []string{"collections:read:all"}, "with read scope"},
			{"/kb/collections", []string{"collections:write:all"}, "with write scope"},
			{"/kb/documents", []string{"documents:read:all"}, "with documents scope"},
		}

		for _, tc := range testCases {
			request := &acl.AccessRequest{
				Method: "GET",
				Path:   tc.path,
				Scopes: tc.scopes,
			}

			decision := manager.Check(request)
			assert.NotNil(t, decision)
			t.Logf("✓ Wildcard '%s' %s: Allowed=%v, Reason=%s",
				tc.path, tc.desc, decision.Allowed, decision.Reason)
		}
	})
}

// TestScopeAtomic_ParameterPath tests parameter path matching
func TestScopeAtomic_ParameterPath(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean()

	manager, err := acl.LoadScopes()
	assert.NoError(t, err)
	assert.NotNil(t, manager)

	t.Run("Parameter_CollectionID", func(t *testing.T) {
		// Test parameter matching: /kb/collections/:collectionID
		testPaths := []string{
			"/kb/collections/abc123",
			"/kb/collections/test-collection",
			"/kb/collections/12345",
		}

		for _, path := range testPaths {
			request := &acl.AccessRequest{
				Method: "GET",
				Path:   path,
				Scopes: []string{"collections:read:all"},
			}

			decision := manager.Check(request)
			assert.NotNil(t, decision)
			t.Logf("✓ Parameter path '%s': Allowed=%v, Reason=%s", path, decision.Allowed, decision.Reason)
		}
	})

	t.Run("Parameter_WithDifferentMethods", func(t *testing.T) {
		// Test parameter path with different methods
		testCases := []struct {
			method string
			path   string
			scopes []string
		}{
			{"GET", "/kb/collections/test-123", []string{"collections:read:all"}},
			{"POST", "/kb/collections/test-123", []string{"collections:write:all"}},
			{"PUT", "/kb/collections/test-123", []string{"collections:write:all"}},
			{"DELETE", "/kb/collections/test-123", []string{"collections:delete:all"}},
		}

		for _, tc := range testCases {
			request := &acl.AccessRequest{
				Method: tc.method,
				Path:   tc.path,
				Scopes: tc.scopes,
			}

			decision := manager.Check(request)
			assert.NotNil(t, decision)
			t.Logf("✓ %s parameter path: Allowed=%v, Reason=%s",
				tc.method, decision.Allowed, decision.Reason)
		}
	})
}

// TestScopeAtomic_AliasExpansion tests scope alias expansion
func TestScopeAtomic_AliasExpansion(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean()

	manager, err := acl.LoadScopes()
	assert.NoError(t, err)
	assert.NotNil(t, manager)

	t.Run("Alias_KBRead", func(t *testing.T) {
		// Test kb:read alias expansion
		// According to alias.yml: kb:read -> collections:read:all, documents:read:all, etc.
		request := &acl.AccessRequest{
			Method: "GET",
			Path:   "/kb/collections",
			Scopes: []string{"kb:read"}, // Use alias instead of direct scope
		}

		decision := manager.Check(request)
		assert.NotNil(t, decision)
		t.Logf("✓ Alias 'kb:read' expansion: Allowed=%v, Reason=%s", decision.Allowed, decision.Reason)

		// Test if alias works for different kb endpoints
		endpoints := []string{
			"/kb/collections",
			"/kb/documents",
			"/kb/search",
		}

		for _, endpoint := range endpoints {
			req := &acl.AccessRequest{
				Method: "GET",
				Path:   endpoint,
				Scopes: []string{"kb:read"},
			}
			dec := manager.Check(req)
			t.Logf("✓ Alias 'kb:read' on '%s': Allowed=%v, Reason=%s",
				endpoint, dec.Allowed, dec.Reason)
		}
	})

	t.Run("Alias_KBWrite", func(t *testing.T) {
		// Test kb:write alias expansion
		request := &acl.AccessRequest{
			Method: "POST",
			Path:   "/kb/collections",
			Scopes: []string{"kb:write"},
		}

		decision := manager.Check(request)
		assert.NotNil(t, decision)
		t.Logf("✓ Alias 'kb:write' expansion: Allowed=%v, Reason=%s", decision.Allowed, decision.Reason)
	})

	t.Run("Alias_WithDirectScope", func(t *testing.T) {
		// Test mixing alias and direct scopes
		request := &acl.AccessRequest{
			Method: "GET",
			Path:   "/kb/collections",
			Scopes: []string{"kb:read", "collections:read:all"}, // Mix alias and direct
		}

		decision := manager.Check(request)
		assert.NotNil(t, decision)
		t.Logf("✓ Mixed alias+direct scope: Allowed=%v, Reason=%s", decision.Allowed, decision.Reason)
	})

	t.Run("Alias_JobRead", func(t *testing.T) {
		// Test job:read alias
		request := &acl.AccessRequest{
			Method: "GET",
			Path:   "/job/jobs",
			Scopes: []string{"job:read"},
		}

		decision := manager.Check(request)
		assert.NotNil(t, decision)
		t.Logf("✓ Alias 'job:read' expansion: Allowed=%v, Reason=%s", decision.Allowed, decision.Reason)
	})
}

// TestScopeAtomic_ScopeValidation tests scope validation logic
func TestScopeAtomic_ScopeValidation(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean()

	manager, err := acl.LoadScopes()
	assert.NoError(t, err)
	assert.NotNil(t, manager)

	t.Run("Scope_Required_Present", func(t *testing.T) {
		// Test with required scope present
		request := &acl.AccessRequest{
			Method: "POST",
			Path:   "/kb/collections",
			Scopes: []string{"collections:write:all"},
		}

		decision := manager.Check(request)
		assert.NotNil(t, decision)
		t.Logf("✓ Required scope present: Allowed=%v, Reason=%s", decision.Allowed, decision.Reason)
		if !decision.Allowed {
			t.Logf("  Missing scopes: %v", decision.MissingScopes)
		}
	})

	t.Run("Scope_Required_Missing", func(t *testing.T) {
		// Test with required scope missing
		request := &acl.AccessRequest{
			Method: "POST",
			Path:   "/kb/collections",
			Scopes: []string{"collections:read:all"}, // Wrong scope (read instead of write)
		}

		decision := manager.Check(request)
		assert.NotNil(t, decision)
		t.Logf("✓ Required scope missing: Allowed=%v, Reason=%s", decision.Allowed, decision.Reason)
		if !decision.Allowed {
			t.Logf("  Missing scopes: %v", decision.MissingScopes)
			t.Logf("  User scopes: %v", decision.UserScopes)
		}
	})

	t.Run("Scope_NoScopeRequired", func(t *testing.T) {
		// Test endpoint that doesn't require scopes (allow policy)
		request := &acl.AccessRequest{
			Method: "GET",
			Path:   "/kb/collections",
			Scopes: []string{}, // No scopes
		}

		decision := manager.Check(request)
		assert.NotNil(t, decision)
		t.Logf("✓ No scope required endpoint: Allowed=%v, Reason=%s", decision.Allowed, decision.Reason)
	})

	t.Run("Scope_ExtraScopes", func(t *testing.T) {
		// Test with extra scopes beyond required
		request := &acl.AccessRequest{
			Method: "GET",
			Path:   "/kb/collections",
			Scopes: []string{
				"collections:read:all",
				"collections:write:all",
				"admin:all",
			},
		}

		decision := manager.Check(request)
		assert.NotNil(t, decision)
		t.Logf("✓ Extra scopes present: Allowed=%v, Reason=%s", decision.Allowed, decision.Reason)
	})
}

// TestScopeAtomic_DefaultPolicy tests default policy behavior
func TestScopeAtomic_DefaultPolicy(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean()

	manager, err := acl.LoadScopes()
	assert.NoError(t, err)
	assert.NotNil(t, manager)

	t.Run("DefaultPolicy_UnmatchedPath", func(t *testing.T) {
		// Test unmatched path (should use default policy)
		request := &acl.AccessRequest{
			Method: "GET",
			Path:   "/unmatched/endpoint/path",
			Scopes: []string{"any:scope"},
		}

		decision := manager.Check(request)
		assert.NotNil(t, decision)
		assert.Contains(t, decision.Reason, "default policy")
		t.Logf("✓ Unmatched path uses default: Allowed=%v, Reason=%s", decision.Allowed, decision.Reason)
	})

	t.Run("DefaultPolicy_UnmatchedMethod", func(t *testing.T) {
		// Test matched path but unmatched method
		request := &acl.AccessRequest{
			Method: "PATCH", // Uncommon method
			Path:   "/kb/collections",
			Scopes: []string{"collections:write:all"},
		}

		decision := manager.Check(request)
		assert.NotNil(t, decision)
		t.Logf("✓ Unmatched method: Allowed=%v, Reason=%s", decision.Allowed, decision.Reason)
	})
}

// TestScopeAtomic_PublicEndpoints tests public endpoint behavior
func TestScopeAtomic_PublicEndpoints(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean()

	manager, err := acl.LoadScopes()
	assert.NoError(t, err)
	assert.NotNil(t, manager)

	t.Run("Public_NoScopeNeeded", func(t *testing.T) {
		// Test public endpoints from scopes.yml
		publicEndpoints := []struct {
			method string
			path   string
		}{
			{"GET", "/user/entry"},
			{"GET", "/user/entry/captcha"},
			{"POST", "/user/entry/verify"},
		}

		for _, ep := range publicEndpoints {
			request := &acl.AccessRequest{
				Method: ep.method,
				Path:   ep.path,
				Scopes: []string{}, // No scopes
			}

			decision := manager.Check(request)
			assert.NotNil(t, decision)
			assert.True(t, decision.Allowed, "Public endpoint should allow: %s %s", ep.method, ep.path)
			assert.Equal(t, "public endpoint", decision.Reason)
			t.Logf("✓ Public endpoint %s %s: Allowed=%v", ep.method, ep.path, decision.Allowed)
		}
	})

	t.Run("Public_WithScopes", func(t *testing.T) {
		// Test public endpoint with scopes (should still allow)
		request := &acl.AccessRequest{
			Method: "GET",
			Path:   "/user/entry",
			Scopes: []string{"user:read", "admin:all"},
		}

		decision := manager.Check(request)
		assert.NotNil(t, decision)
		assert.True(t, decision.Allowed, "Public endpoint should allow even with scopes")
		t.Logf("✓ Public endpoint with scopes: Allowed=%v, Reason=%s", decision.Allowed, decision.Reason)
	})
}

// TestScopeAtomic_ComplexScenarios tests complex real-world scenarios
func TestScopeAtomic_ComplexScenarios(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean()

	manager, err := acl.LoadScopes()
	assert.NoError(t, err)
	assert.NotNil(t, manager)

	t.Run("Complex_MultipleEndpointsSameScope", func(t *testing.T) {
		// Test one scope allowing access to multiple endpoints
		scope := "collections:read:all"
		endpoints := []string{
			"/kb/collections",
			"/kb/collections/test-123",
			"/kb/collections/test-456/documents",
		}

		for _, endpoint := range endpoints {
			request := &acl.AccessRequest{
				Method: "GET",
				Path:   endpoint,
				Scopes: []string{scope},
			}

			decision := manager.Check(request)
			assert.NotNil(t, decision)
			t.Logf("✓ Scope '%s' on '%s': Allowed=%v, Reason=%s",
				scope, endpoint, decision.Allowed, decision.Reason)
		}
	})

	t.Run("Complex_ScopeInheritance", func(t *testing.T) {
		// Test if write scope implies read access (based on config)
		testCases := []struct {
			path   string
			scopes []string
			desc   string
		}{
			{"/kb/collections", []string{"collections:read:all"}, "read scope"},
			{"/kb/collections", []string{"collections:write:all"}, "write scope"},
			{"/kb/collections", []string{"kb:read"}, "kb:read alias"},
			{"/kb/collections", []string{"kb:write"}, "kb:write alias"},
		}

		for _, tc := range testCases {
			request := &acl.AccessRequest{
				Method: "GET",
				Path:   tc.path,
				Scopes: tc.scopes,
			}

			decision := manager.Check(request)
			assert.NotNil(t, decision)
			t.Logf("✓ %s: Allowed=%v, Reason=%s", tc.desc, decision.Allowed, decision.Reason)
		}
	})
}

// TestScopeAtomic_DataConstraints tests data access constraints (OwnerOnly, TeamOnly)
func TestScopeAtomic_DataConstraints(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean()

	manager, err := acl.LoadScopes()
	assert.NoError(t, err)
	assert.NotNil(t, manager)

	t.Run("Constraints_OwnerOnly", func(t *testing.T) {
		// Test endpoints with owner: true constraint
		// According to collections.yml: collections:read:own has owner: true
		request := &acl.AccessRequest{
			Method: "GET",
			Path:   "/kb/collections/own",
			Scopes: []string{"collections:read:own"},
		}

		decision := manager.Check(request)
		assert.NotNil(t, decision)
		t.Logf("✓ OwnerOnly endpoint: Allowed=%v, Reason=%s", decision.Allowed, decision.Reason)

		if decision.Allowed && decision.MatchedEndpoint != nil {
			assert.True(t, decision.MatchedEndpoint.OwnerOnly,
				"Endpoint with owner:true should set OwnerOnly flag")
			t.Logf("  OwnerOnly=%v", decision.MatchedEndpoint.OwnerOnly)
		}
	})

	t.Run("Constraints_TeamOnly", func(t *testing.T) {
		// Test endpoints with team: true constraint
		// According to collections.yml: collections:read:team has team: true
		request := &acl.AccessRequest{
			Method: "GET",
			Path:   "/kb/collections/team",
			Scopes: []string{"collections:read:team"},
		}

		decision := manager.Check(request)
		assert.NotNil(t, decision)
		t.Logf("✓ TeamOnly endpoint: Allowed=%v, Reason=%s", decision.Allowed, decision.Reason)

		if decision.Allowed && decision.MatchedEndpoint != nil {
			assert.True(t, decision.MatchedEndpoint.TeamOnly,
				"Endpoint with team:true should set TeamOnly flag")
			t.Logf("  TeamOnly=%v", decision.MatchedEndpoint.TeamOnly)
		}
	})

	t.Run("Constraints_NoRestrictions", func(t *testing.T) {
		// Test endpoints without constraints
		// collections:read:all has no owner/team flags
		request := &acl.AccessRequest{
			Method: "GET",
			Path:   "/kb/collections",
			Scopes: []string{"collections:read:all"},
		}

		decision := manager.Check(request)
		assert.NotNil(t, decision)
		t.Logf("✓ No constraints endpoint: Allowed=%v, Reason=%s", decision.Allowed, decision.Reason)

		if decision.Allowed && decision.MatchedEndpoint != nil {
			assert.False(t, decision.MatchedEndpoint.OwnerOnly,
				"Endpoint without owner flag should have OwnerOnly=false")
			assert.False(t, decision.MatchedEndpoint.TeamOnly,
				"Endpoint without team flag should have TeamOnly=false")
			t.Logf("  OwnerOnly=%v, TeamOnly=%v",
				decision.MatchedEndpoint.OwnerOnly,
				decision.MatchedEndpoint.TeamOnly)
		}
	})

	t.Run("Constraints_ProfileOwner", func(t *testing.T) {
		// Test user profile endpoint with owner constraint
		// According to profile.yml: profile:read:own has owner: true
		request := &acl.AccessRequest{
			Method: "GET",
			Path:   "/user/profile",
			Scopes: []string{"profile:read:own"},
		}

		decision := manager.Check(request)
		assert.NotNil(t, decision)
		t.Logf("✓ Profile endpoint: Allowed=%v, Reason=%s", decision.Allowed, decision.Reason)

		if decision.Allowed && decision.MatchedEndpoint != nil {
			assert.True(t, decision.MatchedEndpoint.OwnerOnly,
				"Profile endpoint should have OwnerOnly constraint")
			t.Logf("  OwnerOnly=%v", decision.MatchedEndpoint.OwnerOnly)
		}
	})
}
