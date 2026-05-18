package sandboxv2_test

import (
	"testing"
	"time"

	sandboxv2 "github.com/yaoapp/yao/agent/sandbox/v2"
	"github.com/yaoapp/yao/agent/sandbox/v2/types"
	"github.com/yaoapp/yao/openapi/oauth"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

func TestCacheKey_WithTeam(t *testing.T) {
	testprepare.PrepareUnit(t)
	key := sandboxv2.ExportCacheKey("team-1", "user-1")
	if key != "team-1/user-1" {
		t.Errorf("got %q, want %q", key, "team-1/user-1")
	}
}

func TestCacheKey_NoTeam(t *testing.T) {
	testprepare.PrepareUnit(t)
	key := sandboxv2.ExportCacheKey("", "user-2")
	if key != "user-2" {
		t.Errorf("got %q, want %q", key, "user-2")
	}
}

func TestTokenCacheRoundtrip(t *testing.T) {
	testprepare.PrepareUnit(t)
	tok := &types.SandboxToken{Token: "access-tok", RefreshToken: "refresh-tok"}
	sandboxv2.ExportSetToken("team-x", "user-x", tok, 5*time.Minute)

	got := sandboxv2.ExportGetToken("team-x", "user-x")
	if got == nil {
		t.Fatal("expected cached token, got nil")
	}
	if got.Token != "access-tok" {
		t.Errorf("Token: got %q", got.Token)
	}
	if got.RefreshToken != "refresh-tok" {
		t.Errorf("RefreshToken: got %q", got.RefreshToken)
	}
}

func TestTokenCacheMiss(t *testing.T) {
	testprepare.PrepareUnit(t)
	got := sandboxv2.ExportGetToken("nonexistent-team", "nonexistent-user")
	if got != nil {
		t.Errorf("expected nil for cache miss, got %+v", got)
	}
}

// TestIssueSandboxToken_FreshIssue verifies that IssueSandboxToken behaves
// correctly with respect to the global OAuth service state:
//   - When oauth.OAuth is nil (no openapi loaded): returns (nil, nil) gracefully.
//   - When oauth.OAuth is initialized (full env): returns a non-nil signed token.
//
// PrepareUnit deliberately does not load openapi, so OAuth may be nil here
// unless another test in the same binary already triggered PrepareSandbox.
func TestIssueSandboxToken_FreshIssue(t *testing.T) {
	testprepare.PrepareUnit(t)

	tok, err := sandboxv2.IssueSandboxToken("team-fresh", "user-fresh")
	if err != nil {
		t.Fatalf("IssueSandboxToken: %v", err)
	}

	if oauth.OAuth == nil {
		if tok != nil {
			t.Fatalf("expected nil token when OAuth is nil, got %+v", tok)
		}
		return
	}

	if tok == nil {
		t.Fatal("expected non-nil token when OAuth is initialized")
	}
	if tok.Token == "" {
		t.Error("expected non-empty access token")
	}
	if tok.RefreshToken == "" {
		t.Error("expected non-empty refresh token")
	}
}

func TestIssueSandboxToken_CacheHit(t *testing.T) {
	testprepare.PrepareUnit(t)
	cached := &types.SandboxToken{Token: "cached-tok", RefreshToken: "cached-refresh"}
	sandboxv2.ExportSetToken("team-cache", "user-cache", cached, 5*time.Minute)

	tok, err := sandboxv2.IssueSandboxToken("team-cache", "user-cache")
	if err != nil {
		t.Fatalf("IssueSandboxToken: %v", err)
	}
	if tok == nil {
		t.Fatal("expected cached token")
	}
	if tok.Token != "cached-tok" {
		t.Errorf("Token: got %q, want 'cached-tok'", tok.Token)
	}
}
