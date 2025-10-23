package acl

import (
	"testing"
)

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "path with trailing slash",
			input:    "/user/teams/",
			expected: "/user/teams",
		},
		{
			name:     "path without trailing slash",
			input:    "/user/teams",
			expected: "/user/teams",
		},
		{
			name:     "root path",
			input:    "/",
			expected: "/",
		},
		{
			name:     "empty path",
			input:    "",
			expected: "",
		},
		{
			name:     "nested path with trailing slash",
			input:    "/user/teams/members/",
			expected: "/user/teams/members",
		},
		{
			name:     "nested path without trailing slash",
			input:    "/user/teams/members",
			expected: "/user/teams/members",
		},
		{
			name:     "path with multiple trailing slashes",
			input:    "/user/teams//",
			expected: "/user/teams/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizePath(tt.input)
			if result != tt.expected {
				t.Errorf("normalizePath(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestPathMatchingWithTrailingSlash tests that paths match correctly regardless of trailing slashes
func TestPathMatchingWithTrailingSlash(t *testing.T) {
	manager := &ScopeManager{
		endpointIndex: make(map[string]*PathMatcher),
		publicPaths:   make(map[string]struct{}),
	}

	// Add a test endpoint without trailing slash
	err := manager.addEndpointRule("POST", "/user/teams", "require-scopes", []string{"teams:write:own"})
	if err != nil {
		t.Fatalf("Failed to add endpoint rule: %v", err)
	}

	// Test that matching works with trailing slash
	endpoint, pattern := manager.matchEndpoint("POST", "/user/teams/")
	if endpoint == nil {
		t.Errorf("Expected to match endpoint POST /user/teams/, but got nil")
	}
	if pattern != "/user/teams" {
		t.Errorf("Expected pattern /user/teams, got %s", pattern)
	}

	// Test that matching works without trailing slash
	endpoint, pattern = manager.matchEndpoint("POST", "/user/teams")
	if endpoint == nil {
		t.Errorf("Expected to match endpoint POST /user/teams, but got nil")
	}
	if pattern != "/user/teams" {
		t.Errorf("Expected pattern /user/teams, got %s", pattern)
	}
}

// TestPathMatchingExactPaths tests exact path matching with normalization
func TestPathMatchingExactPaths(t *testing.T) {
	manager := &ScopeManager{
		endpointIndex: make(map[string]*PathMatcher),
		publicPaths:   make(map[string]struct{}),
	}

	// Add endpoints with different trailing slash patterns
	testCases := []struct {
		method       string
		definedPath  string
		requestPaths []string
		shouldMatch  bool
	}{
		{
			method:       "POST",
			definedPath:  "/user/teams",
			requestPaths: []string{"/user/teams", "/user/teams/"},
			shouldMatch:  true,
		},
		{
			method:       "GET",
			definedPath:  "/user/teams/",
			requestPaths: []string{"/user/teams", "/user/teams/"},
			shouldMatch:  true,
		},
		{
			method:       "DELETE",
			definedPath:  "/user/profile",
			requestPaths: []string{"/user/profile", "/user/profile/"},
			shouldMatch:  true,
		},
	}

	for _, tc := range testCases {
		// Add the endpoint
		err := manager.addEndpointRule(tc.method, tc.definedPath, "allow", nil)
		if err != nil {
			t.Fatalf("Failed to add endpoint rule %s %s: %v", tc.method, tc.definedPath, err)
		}

		// Test all request paths
		for _, requestPath := range tc.requestPaths {
			endpoint, pattern := manager.matchEndpoint(tc.method, requestPath)
			if tc.shouldMatch && endpoint == nil {
				t.Errorf("Expected %s %s to match defined path %s, but got nil",
					tc.method, requestPath, tc.definedPath)
			} else if tc.shouldMatch && endpoint != nil {
				// Both paths should normalize to the same pattern
				expectedPattern := normalizePath(tc.definedPath)
				if pattern != expectedPattern {
					t.Errorf("Expected pattern %s, got %s (defined: %s, request: %s)",
						expectedPattern, pattern, tc.definedPath, requestPath)
				}
			}
		}
	}
}
