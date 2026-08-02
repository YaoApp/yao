package volume

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"crypto/x509"

	gossh "golang.org/x/crypto/ssh"
)

func generateTestED25519Key(t *testing.T) string {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	raw, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		t.Fatalf("marshal key: %v", err)
	}
	return string(pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: raw,
	}))
}

func TestRemoveSSHHostBlock_RemovesMatch(t *testing.T) {
	content := "Host github.com\n\tIdentityFile /path/key\n\tStrictHostKeyChecking accept-new\nHost gitlab.com\n\tIdentityFile /other/key\n"
	result := removeSSHHostBlock(content, "github.com")

	if strings.Contains(result, "github.com") {
		t.Error("github.com block should be removed")
	}
	if !strings.Contains(result, "gitlab.com") {
		t.Error("gitlab.com block should remain")
	}
}

func TestRemoveSSHHostBlock_NoMatch(t *testing.T) {
	content := "Host github.com\n\tIdentityFile /path/key\n"
	result := removeSSHHostBlock(content, "bitbucket.org")

	if !strings.Contains(result, "github.com") {
		t.Error("content should be unchanged")
	}
}

func TestRemoveSSHHostBlock_CaseInsensitive(t *testing.T) {
	content := "Host GitHub.Com\n\tIdentityFile /path/key\n"
	result := removeSSHHostBlock(content, "github.com")

	if strings.Contains(result, "GitHub") {
		t.Error("case-insensitive match should remove block")
	}
}

func TestRemoveSSHHostBlockByKeyName_RemovesBlock(t *testing.T) {
	content := "Host github.com\n\tIdentityFile /ws/ssh/keys/github.key\nHost gitlab.com\n\tIdentityFile /ws/ssh/keys/gitlab.key\n"
	result := removeSSHHostBlockByKeyName(content, "github.key")

	if strings.Contains(result, "github.com") {
		t.Error("github block should be removed")
	}
	if !strings.Contains(result, "gitlab.com") {
		t.Error("gitlab block should remain")
	}
}

func TestRemoveSSHHostBlockByKeyName_NoMatch(t *testing.T) {
	content := "Host github.com\n\tIdentityFile /ws/ssh/keys/github.key\n"
	result := removeSSHHostBlockByKeyName(content, "bitbucket.key")

	if !strings.Contains(result, "github.com") {
		t.Error("content should be unchanged")
	}
}

func TestRemoveSSHHostBlockByKeyName_EmptyAfterRemoval(t *testing.T) {
	content := "Host github.com\n\tIdentityFile /ws/ssh/keys/only.key\n"
	result := removeSSHHostBlockByKeyName(content, "only.key")

	if result != "" {
		t.Errorf("expected empty, got %q", result)
	}
}

func TestUpdateSSHConfig_AddNew(t *testing.T) {
	base := t.TempDir()
	wsBase := filepath.Join(base, ".workspace")
	_ = ensureConfigDirs(wsBase)

	keyPath := filepath.Join(wsBase, "ssh", "keys", "mykey.key")
	_ = os.WriteFile(keyPath, []byte("fake-key"), 0o600)

	if err := updateSSHConfig(wsBase, "mykey", "github.com"); err != nil {
		t.Fatalf("updateSSHConfig: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(wsBase, "ssh", "config"))
	if !strings.Contains(string(data), "Host github.com") {
		t.Error("config should contain Host block")
	}
	if !strings.Contains(string(data), "IdentityFile") {
		t.Error("config should contain IdentityFile")
	}
}

func TestUpdateSSHConfig_ReplaceExisting(t *testing.T) {
	base := t.TempDir()
	wsBase := filepath.Join(base, ".workspace")
	_ = ensureConfigDirs(wsBase)

	_ = updateSSHConfig(wsBase, "old", "github.com")
	_ = updateSSHConfig(wsBase, "new", "github.com")

	data, _ := os.ReadFile(filepath.Join(wsBase, "ssh", "config"))
	if strings.Count(string(data), "Host github.com") != 1 {
		t.Errorf("should have exactly 1 Host github.com block: %q", data)
	}
	if !strings.Contains(string(data), "new.key") {
		t.Error("should reference new key")
	}
}

func TestDerivePublicKey_ED25519(t *testing.T) {
	pemData := generateTestED25519Key(t)

	pubBytes, err := derivePublicKey([]byte(pemData))
	if err != nil {
		t.Fatalf("derivePublicKey: %v", err)
	}
	if len(pubBytes) == 0 {
		t.Error("empty public key")
	}

	_, _, _, _, err = gossh.ParseAuthorizedKey(pubBytes)
	if err != nil {
		t.Errorf("invalid authorized key format: %v", err)
	}
}

func TestDerivePublicKey_InvalidPEM(t *testing.T) {
	_, err := derivePublicKey([]byte("not a pem key"))
	if err == nil {
		t.Error("expected error for invalid PEM")
	}
}

// ---------------------------------------------------------------------------
// localStorage: GitSSHKeyImport / List / Delete
// ---------------------------------------------------------------------------

func TestLocalGitSSHKey_CRUD(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	ctx := context.Background()
	sid := "ssh-key-crud"
	_ = os.MkdirAll(filepath.Join(dir, sid), 0o755)

	privKey := generateTestED25519Key(t)

	if err := vol.GitSSHKeyImport(ctx, sid, "mykey", privKey, "", "github.com"); err != nil {
		t.Fatalf("GitSSHKeyImport: %v", err)
	}

	keys, err := vol.GitSSHKeyList(ctx, sid)
	if err != nil {
		t.Fatalf("GitSSHKeyList: %v", err)
	}
	if len(keys) != 1 {
		t.Fatalf("expected 1 key, got %d", len(keys))
	}
	if keys[0].Name != "mykey" {
		t.Errorf("name = %q", keys[0].Name)
	}
	if keys[0].Fingerprint == "" {
		t.Error("fingerprint should not be empty")
	}

	if err := vol.GitSSHKeyDelete(ctx, sid, "mykey"); err != nil {
		t.Fatalf("GitSSHKeyDelete: %v", err)
	}

	keys, _ = vol.GitSSHKeyList(ctx, sid)
	if len(keys) != 0 {
		t.Errorf("expected 0 keys after delete, got %d", len(keys))
	}
}

func TestLocalGitSSHKeyImport_Validation(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	ctx := context.Background()
	sid := "ssh-key-valid"
	_ = os.MkdirAll(filepath.Join(dir, sid), 0o755)

	if err := vol.GitSSHKeyImport(ctx, sid, "", "key-data", "", ""); err == nil {
		t.Error("expected error for empty name")
	}
	if err := vol.GitSSHKeyImport(ctx, sid, "mykey", "", "", ""); err == nil {
		t.Error("expected error for empty private_key")
	}
}

func TestLocalGitSSHKeyImport_WithPublicKey(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	ctx := context.Background()
	sid := "ssh-key-withpub"
	_ = os.MkdirAll(filepath.Join(dir, sid), 0o755)

	privKey := generateTestED25519Key(t)
	pubKey := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAItest test@example.com"

	if err := vol.GitSSHKeyImport(ctx, sid, "withpub", privKey, pubKey, ""); err != nil {
		t.Fatalf("GitSSHKeyImport: %v", err)
	}

	wsBase := filepath.Join(dir, sid, ".workspace")
	data, _ := os.ReadFile(filepath.Join(wsBase, "ssh", "keys", "withpub.pub"))
	if string(data) != pubKey {
		t.Errorf("pub content = %q, want %q", data, pubKey)
	}
}

func TestLocalGitSSHKeyImport_NoHost(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	ctx := context.Background()
	sid := "ssh-key-nohost"
	_ = os.MkdirAll(filepath.Join(dir, sid), 0o755)

	privKey := generateTestED25519Key(t)

	if err := vol.GitSSHKeyImport(ctx, sid, "nohost", privKey, "", ""); err != nil {
		t.Fatalf("GitSSHKeyImport: %v", err)
	}

	wsBase := filepath.Join(dir, sid, ".workspace")
	data, _ := os.ReadFile(filepath.Join(wsBase, "ssh", "config"))
	if strings.Contains(string(data), "Host") {
		t.Error("should not add Host block when host is empty")
	}
}

func TestLocalGitSSHKeyDelete_Validation(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	ctx := context.Background()
	sid := "ssh-key-del-valid"
	_ = os.MkdirAll(filepath.Join(dir, sid), 0o755)

	if err := vol.GitSSHKeyDelete(ctx, sid, ""); err == nil {
		t.Error("expected error for empty name")
	}
}

func TestLocalGitSSHKeyDelete_CleansSSHConfig(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	ctx := context.Background()
	sid := "ssh-key-del-config"
	_ = os.MkdirAll(filepath.Join(dir, sid), 0o755)

	privKey := generateTestED25519Key(t)
	_ = vol.GitSSHKeyImport(ctx, sid, "github", privKey, "", "github.com")

	wsBase := filepath.Join(dir, sid, ".workspace")
	cfgData, _ := os.ReadFile(filepath.Join(wsBase, "ssh", "config"))
	if !strings.Contains(string(cfgData), "Host github.com") {
		t.Fatal("SSH config should contain Host block before delete")
	}

	_ = vol.GitSSHKeyDelete(ctx, sid, "github")

	cfgData, _ = os.ReadFile(filepath.Join(wsBase, "ssh", "config"))
	if strings.Contains(string(cfgData), "github.key") {
		t.Error("SSH config should not reference deleted key")
	}
}

func TestLocalGitSSHKeyList_Empty(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir)
	ctx := context.Background()
	sid := "ssh-key-list-empty"
	_ = os.MkdirAll(filepath.Join(dir, sid), 0o755)

	keys, err := vol.GitSSHKeyList(ctx, sid)
	if err != nil {
		t.Fatalf("GitSSHKeyList: %v", err)
	}
	if keys != nil {
		t.Errorf("expected nil, got %v", keys)
	}
}
