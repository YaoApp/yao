package volume

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseConfigKey_TwoParts(t *testing.T) {
	sec, sub, key := parseConfigKey("user.name")
	if sec != "user" || sub != "" || key != "name" {
		t.Errorf("got (%q,%q,%q)", sec, sub, key)
	}
}

func TestParseConfigKey_ThreeParts(t *testing.T) {
	sec, sub, key := parseConfigKey("remote.origin.url")
	if sec != "remote" || sub != "origin" || key != "url" {
		t.Errorf("got (%q,%q,%q)", sec, sub, key)
	}
}

func TestParseConfigKey_SinglePart(t *testing.T) {
	sec, sub, key := parseConfigKey("standalone")
	if sec != "standalone" || sub != "" || key != "" {
		t.Errorf("got (%q,%q,%q)", sec, sub, key)
	}
}

func TestInitGitConfig_CreatesCredentialHelper(t *testing.T) {
	base := t.TempDir()
	wsBase := filepath.Join(base, ".workspace")
	_ = ensureConfigDirs(wsBase)

	if err := initGitConfig(wsBase); err != nil {
		t.Fatalf("initGitConfig: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(wsBase, "git", "config"))
	content := string(data)
	if !strings.Contains(content, "[credential]") {
		t.Error("config should contain [credential]")
	}
	if strings.Contains(content, "--file=") {
		t.Error("credential helper should NOT contain --file=")
	}
	assertHelperChain(t, content)
}

func TestInitGitConfig_Idempotent(t *testing.T) {
	base := t.TempDir()
	wsBase := filepath.Join(base, ".workspace")
	_ = ensureConfigDirs(wsBase)

	_ = initGitConfig(wsBase)
	data1, _ := os.ReadFile(filepath.Join(wsBase, "git", "config"))

	_ = initGitConfig(wsBase)
	data2, _ := os.ReadFile(filepath.Join(wsBase, "git", "config"))

	if strings.Count(string(data2), "[credential]") != 1 {
		t.Errorf("credential section duplicated: %q", data2)
	}
	if string(data1) != string(data2) {
		t.Errorf("second call changed config:\nbefore: %q\nafter:  %q", data1, data2)
	}
}

func TestInitGitConfig_UpgradeFormatA(t *testing.T) {
	base := t.TempDir()
	wsBase := filepath.Join(base, ".workspace")
	_ = ensureConfigDirs(wsBase)
	cfgPath := filepath.Join(wsBase, "git", "config")
	_ = os.WriteFile(cfgPath, []byte("[credential]\n\thelper = store\n"), 0o600)

	if err := initGitConfig(wsBase); err != nil {
		t.Fatalf("initGitConfig: %v", err)
	}

	data, _ := os.ReadFile(cfgPath)
	assertHelperChain(t, string(data))
}

func TestInitGitConfig_UpgradeFormatB(t *testing.T) {
	base := t.TempDir()
	wsBase := filepath.Join(base, ".workspace")
	_ = ensureConfigDirs(wsBase)
	cfgPath := filepath.Join(wsBase, "git", "config")
	_ = os.WriteFile(cfgPath, []byte("[credential]\n\thelper = store --file=/some/path/credentials\n"), 0o600)

	if err := initGitConfig(wsBase); err != nil {
		t.Fatalf("initGitConfig: %v", err)
	}

	data, _ := os.ReadFile(cfgPath)
	content := string(data)
	assertHelperChain(t, content)
	if strings.Contains(content, "--file=") {
		t.Error("upgraded config should NOT contain --file=")
	}
}

func TestInitGitConfig_UpgradeFormatC(t *testing.T) {
	base := t.TempDir()
	wsBase := filepath.Join(base, ".workspace")
	_ = ensureConfigDirs(wsBase)
	cfgPath := filepath.Join(wsBase, "git", "config")
	_ = os.WriteFile(cfgPath, []byte("[credential]\n\thelper =\n\thelper = store\n"), 0o600)

	if err := initGitConfig(wsBase); err != nil {
		t.Fatalf("initGitConfig: %v", err)
	}

	data, _ := os.ReadFile(cfgPath)
	assertHelperChain(t, string(data))
}

func TestInitGitConfig_UpgradePreservesNonHelper(t *testing.T) {
	base := t.TempDir()
	wsBase := filepath.Join(base, ".workspace")
	_ = ensureConfigDirs(wsBase)
	cfgPath := filepath.Join(wsBase, "git", "config")
	_ = os.WriteFile(cfgPath, []byte("[user]\n\tname = Test\n[credential]\n\thelper = store\n\tusername = testuser\n"), 0o600)

	if err := initGitConfig(wsBase); err != nil {
		t.Fatalf("initGitConfig: %v", err)
	}

	data, _ := os.ReadFile(cfgPath)
	content := string(data)
	assertHelperChain(t, content)
	if !strings.Contains(content, "username = testuser") {
		t.Errorf("non-helper option should be preserved, got:\n%s", content)
	}
	if !strings.Contains(content, "name = Test") {
		t.Errorf("other sections should be preserved, got:\n%s", content)
	}
}

func extractHelpers(content string) []string {
	var helpers []string
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "helper") || !strings.Contains(trimmed, "=") {
			continue
		}
		parts := strings.SplitN(trimmed, "=", 2)
		if len(parts) == 2 {
			helpers = append(helpers, strings.TrimSpace(parts[1]))
		}
	}
	return helpers
}

func assertHelperChain(t *testing.T, content string) {
	t.Helper()
	expected := credentialHelpers()
	got := extractHelpers(content)
	if len(got) != len(expected) {
		t.Errorf("helper count: got %d %v, want %d %v\nconfig:\n%s",
			len(got), got, len(expected), expected, content)
		return
	}
	for i := range expected {
		if got[i] != expected[i] {
			t.Errorf("helper[%d]: got %q, want %q", i, got[i], expected[i])
		}
	}
}

func TestReadWriteCredentialFile_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "credentials")

	entries := []credEntry{
		{Host: "github.com", Username: "user1", Token: "token1"},
		{Host: "gitlab.com", Username: "x-access-token", Token: "glpat-abc"},
	}

	if err := writeCredentialFile(path, entries); err != nil {
		t.Fatalf("write: %v", err)
	}

	got, err := readCredentialFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d entries, want 2", len(got))
	}
	if got[0].Host != "github.com" || got[0].Token != "token1" {
		t.Errorf("entry[0] = %+v", got[0])
	}
	if got[1].Host != "gitlab.com" || got[1].Token != "glpat-abc" {
		t.Errorf("entry[1] = %+v", got[1])
	}
}

func TestReadCredentialFile_NonExistent(t *testing.T) {
	entries, err := readCredentialFile("/nonexistent/credentials")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entries != nil {
		t.Errorf("expected nil, got %v", entries)
	}
}

func TestReadCredentialFile_SkipsInvalidLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "creds")
	_ = os.WriteFile(path, []byte("# comment\ninvalid line\nhttps://user:pass@host.com\n"), 0o600)

	entries, err := readCredentialFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("expected 1 valid entry, got %d", len(entries))
	}
}

// ---------------------------------------------------------------------------
// localStorage: GitConfigGet / GitConfigSet
// ---------------------------------------------------------------------------

func TestLocalGitConfigSet_And_Get(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	ctx := context.Background()
	sid := "git-config-setget"
	_ = os.MkdirAll(filepath.Join(dir, sid), 0o755)

	if err := vol.GitConfigSet(ctx, sid, "user.name", "Test User"); err != nil {
		t.Fatalf("GitConfigSet name: %v", err)
	}
	if err := vol.GitConfigSet(ctx, sid, "user.email", "test@example.com"); err != nil {
		t.Fatalf("GitConfigSet email: %v", err)
	}

	values, err := vol.GitConfigGet(ctx, sid, "user.name")
	if err != nil {
		t.Fatalf("GitConfigGet: %v", err)
	}
	if values["user.name"] != "Test User" {
		t.Errorf("user.name = %q", values["user.name"])
	}
}

func TestLocalGitConfigGet_All(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	ctx := context.Background()
	sid := "git-config-getall"
	_ = os.MkdirAll(filepath.Join(dir, sid), 0o755)

	_ = vol.GitConfigSet(ctx, sid, "user.name", "N")
	_ = vol.GitConfigSet(ctx, sid, "user.email", "E")

	values, err := vol.GitConfigGet(ctx, sid, "")
	if err != nil {
		t.Fatalf("GitConfigGet all: %v", err)
	}
	if len(values) < 2 {
		t.Errorf("expected >= 2 entries, got %d", len(values))
	}
}

func TestLocalGitConfigGet_NonExistent(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	ctx := context.Background()
	sid := "git-config-noexist"
	_ = os.MkdirAll(filepath.Join(dir, sid), 0o755)

	values, err := vol.GitConfigGet(ctx, sid, "user.name")
	if err != nil {
		t.Fatalf("GitConfigGet: %v", err)
	}
	if len(values) != 0 {
		t.Errorf("expected empty map, got %d entries", len(values))
	}
}

func TestLocalGitConfigSet_EmptyKey(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	ctx := context.Background()
	sid := "git-config-emptykey"
	_ = os.MkdirAll(filepath.Join(dir, sid), 0o755)

	err := vol.GitConfigSet(ctx, sid, "", "value")
	if err == nil {
		t.Error("expected error for empty key")
	}
}

func TestLocalGitConfigSet_Subsection(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	ctx := context.Background()
	sid := "git-config-subsec"
	_ = os.MkdirAll(filepath.Join(dir, sid), 0o755)

	if err := vol.GitConfigSet(ctx, sid, "remote.origin.url", "https://example.com/repo.git"); err != nil {
		t.Fatalf("GitConfigSet subsection: %v", err)
	}

	values, err := vol.GitConfigGet(ctx, sid, "remote.origin.url")
	if err != nil {
		t.Fatalf("GitConfigGet: %v", err)
	}
	if values["remote.origin.url"] != "https://example.com/repo.git" {
		t.Errorf("url = %q", values["remote.origin.url"])
	}
}

// ---------------------------------------------------------------------------
// localStorage: GitCredentialSet / List / Delete
// ---------------------------------------------------------------------------

func TestLocalGitCredential_CRUD(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	ctx := context.Background()
	sid := "git-cred-crud"
	_ = os.MkdirAll(filepath.Join(dir, sid), 0o755)

	if err := vol.GitCredentialSet(ctx, sid, "github.com", "myuser", "ghp_token123"); err != nil {
		t.Fatalf("GitCredentialSet: %v", err)
	}
	if err := vol.GitCredentialSet(ctx, sid, "gitlab.com", "", "glpat-xyz"); err != nil {
		t.Fatalf("GitCredentialSet gitlab: %v", err)
	}

	list, err := vol.GitCredentialList(ctx, sid)
	if err != nil {
		t.Fatalf("GitCredentialList: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 credentials, got %d", len(list))
	}

	foundGH := false
	for _, c := range list {
		if c.Host == "github.com" {
			foundGH = true
			if c.Username != "myuser" {
				t.Errorf("username = %q, want 'myuser'", c.Username)
			}
		}
		if c.Host == "gitlab.com" && c.Username != "x-access-token" {
			t.Errorf("default username = %q, want 'x-access-token'", c.Username)
		}
	}
	if !foundGH {
		t.Error("github.com credential not found")
	}

	if err := vol.GitCredentialDelete(ctx, sid, "github.com"); err != nil {
		t.Fatalf("GitCredentialDelete: %v", err)
	}

	list, _ = vol.GitCredentialList(ctx, sid)
	if len(list) != 1 {
		t.Errorf("expected 1 credential after delete, got %d", len(list))
	}
}

func TestLocalGitCredentialSet_ReplaceExisting(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	ctx := context.Background()
	sid := "git-cred-replace"
	_ = os.MkdirAll(filepath.Join(dir, sid), 0o755)

	_ = vol.GitCredentialSet(ctx, sid, "github.com", "old", "old-token")
	_ = vol.GitCredentialSet(ctx, sid, "github.com", "new", "new-token")

	list, _ := vol.GitCredentialList(ctx, sid)
	if len(list) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(list))
	}
	if list[0].Username != "new" {
		t.Errorf("username = %q, want 'new'", list[0].Username)
	}
}

func TestLocalGitCredentialSet_Validation(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	ctx := context.Background()
	sid := "git-cred-valid"
	_ = os.MkdirAll(filepath.Join(dir, sid), 0o755)

	if err := vol.GitCredentialSet(ctx, sid, "", "", "token"); err == nil {
		t.Error("expected error for empty host")
	}
	if err := vol.GitCredentialSet(ctx, sid, "host", "", ""); err == nil {
		t.Error("expected error for empty token")
	}
}

func TestLocalGitCredentialDelete_NonExistent(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	ctx := context.Background()
	sid := "git-cred-del-noexist"
	_ = os.MkdirAll(filepath.Join(dir, sid), 0o755)

	err := vol.GitCredentialDelete(ctx, sid, "nonexistent.com")
	if err != nil {
		t.Fatalf("GitCredentialDelete: %v", err)
	}
}

func TestLocalGitCredentialDelete_EmptyHost(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	ctx := context.Background()
	sid := "git-cred-del-empty"
	_ = os.MkdirAll(filepath.Join(dir, sid), 0o755)

	err := vol.GitCredentialDelete(ctx, sid, "")
	if err == nil {
		t.Error("expected error for empty host")
	}
}

func TestLocalGitCredentialList_Empty(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	ctx := context.Background()
	sid := "git-cred-list-empty"
	_ = os.MkdirAll(filepath.Join(dir, sid), 0o755)

	list, err := vol.GitCredentialList(ctx, sid)
	if err != nil {
		t.Fatalf("GitCredentialList: %v", err)
	}
	if list != nil {
		t.Errorf("expected nil, got %v", list)
	}
}
