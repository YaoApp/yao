package volume

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestIsSSH(t *testing.T) {
	cases := []struct {
		url  string
		want bool
	}{
		{"git@github.com:user/repo.git", true},
		{"ssh://git@github.com/repo.git", true},
		{"https://github.com/repo.git", false},
		{"http://github.com/repo.git", false},
		{"", false},
	}
	for _, c := range cases {
		if got := isSSH(c.url); got != c.want {
			t.Errorf("isSSH(%q) = %v, want %v", c.url, got, c.want)
		}
	}
}

func TestIsHTTPS(t *testing.T) {
	cases := []struct {
		url  string
		want bool
	}{
		{"https://github.com/repo.git", true},
		{"http://github.com/repo.git", false},
		{"git@github.com:user/repo.git", false},
		{"ssh://git@github.com/repo.git", false},
		{"", false},
	}
	for _, c := range cases {
		if got := isHTTPS(c.url); got != c.want {
			t.Errorf("isHTTPS(%q) = %v, want %v", c.url, got, c.want)
		}
	}
}

func TestExtractSSHHost(t *testing.T) {
	cases := []struct {
		url  string
		want string
	}{
		{"git@github.com:user/repo.git", "github.com"},
		{"ssh://git@gitlab.com/repo.git", "gitlab.com"},
		{"git@bitbucket.org:team/project.git", "bitbucket.org"},
		{"https://github.com/repo.git", ""},
		{"", ""},
	}
	for _, c := range cases {
		if got := extractSSHHost(c.url); got != c.want {
			t.Errorf("extractSSHHost(%q) = %q, want %q", c.url, got, c.want)
		}
	}
}

func TestExtractHost(t *testing.T) {
	cases := []struct {
		url  string
		want string
	}{
		{"https://github.com/repo.git", "github.com"},
		{"https://gitlab.example.com:8443/repo.git", "gitlab.example.com"},
		{"invalid-url", ""},
	}
	for _, c := range cases {
		if got := extractHost(c.url); got != c.want {
			t.Errorf("extractHost(%q) = %q, want %q", c.url, got, c.want)
		}
	}
}

func TestSanitizeRemoteURL(t *testing.T) {
	cases := []struct {
		url  string
		want string
	}{
		{"https://user:token@github.com/repo.git", "https://github.com/repo.git"},
		{"https://github.com/repo.git", "https://github.com/repo.git"},
		{"git@github.com:user/repo.git", "git@github.com:user/repo.git"},
		{"ssh://git@github.com/repo.git", "ssh://git@github.com/repo.git"},
	}
	for _, c := range cases {
		if got := sanitizeRemoteURL(c.url); got != c.want {
			t.Errorf("sanitizeRemoteURL(%q) = %q, want %q", c.url, got, c.want)
		}
	}
}

func TestMatchSSHKey(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config")

	content := "Host github.com\n\tIdentityFile /keys/github.key\nHost gitlab.com\n\tIdentityFile /keys/gitlab.key\n"
	_ = os.WriteFile(cfgPath, []byte(content), 0o600)

	path, err := matchSSHKey(cfgPath, "github.com")
	if err != nil {
		t.Fatalf("matchSSHKey: %v", err)
	}
	if path != "/keys/github.key" {
		t.Errorf("path = %q", path)
	}

	path, err = matchSSHKey(cfgPath, "gitlab.com")
	if err != nil {
		t.Fatalf("matchSSHKey: %v", err)
	}
	if path != "/keys/gitlab.key" {
		t.Errorf("path = %q", path)
	}

	_, err = matchSSHKey(cfgPath, "bitbucket.org")
	if err == nil {
		t.Error("expected error for unmatched host")
	}
}

func TestMatchSSHKey_NonExistentFile(t *testing.T) {
	_, err := matchSSHKey("/nonexistent/config", "github.com")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

// ---------------------------------------------------------------------------
// resolveAuth integration tests
// ---------------------------------------------------------------------------

func TestResolveAuth_HTTPSWithCredential(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	ctx := context.Background()
	sid := "auth-https"
	_ = os.MkdirAll(filepath.Join(dir, sid), 0o755)

	_ = vol.GitCredentialSet(ctx, sid, "github.com", "x-access-token", "ghp_test123")

	ls := vol.(*localStorage)
	auth, err := ls.resolveAuth(sid, "https://github.com/user/repo.git")
	if err != nil {
		t.Fatalf("resolveAuth: %v", err)
	}
	if auth == nil {
		t.Fatal("expected auth for HTTPS with configured credential")
	}
}

func TestResolveAuth_HTTPSNoCredential(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	sid := "auth-https-none"
	_ = os.MkdirAll(filepath.Join(dir, sid), 0o755)

	ls := vol.(*localStorage)
	auth, err := ls.resolveAuth(sid, "https://github.com/user/repo.git")
	if err != nil {
		t.Fatalf("resolveAuth: %v", err)
	}
	if auth != nil {
		t.Error("expected nil auth when no credential configured")
	}
}

func TestResolveAuth_SSHWithKey(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	ctx := context.Background()
	sid := "auth-ssh"
	_ = os.MkdirAll(filepath.Join(dir, sid), 0o755)

	privKey := generateTestED25519Key(t)
	_ = vol.GitSSHKeyImport(ctx, sid, "github", privKey, "", "github.com")

	ls := vol.(*localStorage)
	auth, err := ls.resolveAuth(sid, "git@github.com:user/repo.git")
	if err != nil {
		t.Fatalf("resolveAuth: %v", err)
	}
	if auth == nil {
		t.Fatal("expected auth for SSH with configured key")
	}
}

func TestResolveAuth_SSHFallbackToFirstKey(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	ctx := context.Background()
	sid := "auth-ssh-fallback"
	_ = os.MkdirAll(filepath.Join(dir, sid), 0o755)

	privKey := generateTestED25519Key(t)
	_ = vol.GitSSHKeyImport(ctx, sid, "generic", privKey, "", "")

	ls := vol.(*localStorage)
	auth, err := ls.resolveAuth(sid, "git@bitbucket.org:user/repo.git")
	if err != nil {
		t.Fatalf("resolveAuth: %v", err)
	}
	if auth == nil {
		t.Fatal("expected auth from fallback key")
	}
}

func TestResolveAuth_UnknownProtocol(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	sid := "auth-unknown"
	_ = os.MkdirAll(filepath.Join(dir, sid), 0o755)

	ls := vol.(*localStorage)
	auth, err := ls.resolveAuth(sid, "ftp://example.com/repo.git")
	if err != nil {
		t.Fatalf("resolveAuth: %v", err)
	}
	if auth != nil {
		t.Error("expected nil auth for unknown protocol")
	}
}
