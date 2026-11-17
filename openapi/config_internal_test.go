package openapi

import "testing"

func TestResolveOAuthIssuerURL(t *testing.T) {
	t.Setenv("YAO_OAUTH_ISSUER_URL", "")
	if got := resolveOAuthIssuerURL("http://127.0.0.1:5098"); got != "http://127.0.0.1:5098" {
		t.Fatalf("fallback issuer = %q", got)
	}
	t.Setenv("YAO_OAUTH_ISSUER_URL", "http://192.168.3.110:5099/")
	if got := resolveOAuthIssuerURL("http://127.0.0.1:5098"); got != "http://192.168.3.110:5099" {
		t.Fatalf("env issuer = %q", got)
	}
}
