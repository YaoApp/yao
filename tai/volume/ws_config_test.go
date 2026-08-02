package volume

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestEnsureConfigDirs_CreatesStructure(t *testing.T) {
	base := t.TempDir()
	wsBase := filepath.Join(base, ".workspace")

	if err := ensureConfigDirs(wsBase); err != nil {
		t.Fatalf("ensureConfigDirs: %v", err)
	}

	dirs := []string{
		filepath.Join(wsBase, "git"),
		filepath.Join(wsBase, "ssh", "keys"),
	}
	for _, d := range dirs {
		info, err := os.Stat(d)
		if err != nil {
			t.Errorf("dir %s not created: %v", d, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("%s is not a directory", d)
		}
	}

	emptyFiles := []string{
		filepath.Join(wsBase, "git", "config"),
		filepath.Join(wsBase, "ssh", "config"),
	}
	for _, f := range emptyFiles {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Errorf("file %s not created: %v", f, err)
			continue
		}
		if len(data) != 0 {
			t.Errorf("file %s should be empty, got %d bytes", f, len(data))
		}
	}
}

func TestEnsureConfigDirs_Idempotent(t *testing.T) {
	base := t.TempDir()
	wsBase := filepath.Join(base, ".workspace")

	if err := ensureConfigDirs(wsBase); err != nil {
		t.Fatalf("first call: %v", err)
	}

	cfgPath := filepath.Join(wsBase, "git", "config")
	if err := os.WriteFile(cfgPath, []byte("[user]\n\tname = Test\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := ensureConfigDirs(wsBase); err != nil {
		t.Fatalf("second call: %v", err)
	}

	data, _ := os.ReadFile(cfgPath)
	if string(data) != "[user]\n\tname = Test\n" {
		t.Errorf("existing config was overwritten: %q", data)
	}
}

func TestWriteSecureFile_Basic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "secret.txt")

	if err := writeSecureFile(path, []byte("sensitive data")); err != nil {
		t.Fatalf("writeSecureFile: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(data) != "sensitive data" {
		t.Errorf("data = %q", data)
	}

	if runtime.GOOS != "windows" {
		info, _ := os.Stat(path)
		if info.Mode().Perm() != 0o600 {
			t.Errorf("perm = %o, want 0600", info.Mode().Perm())
		}
	}
}

func TestWriteSecureFile_Overwrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "overwrite.txt")

	_ = os.WriteFile(path, []byte("old"), 0o644)

	if err := writeSecureFile(path, []byte("new")); err != nil {
		t.Fatalf("writeSecureFile: %v", err)
	}

	data, _ := os.ReadFile(path)
	if string(data) != "new" {
		t.Errorf("data = %q, want 'new'", data)
	}
}

func TestWriteSecureFile_NoTmpLeftBehind(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")

	_ = writeSecureFile(path, []byte("ok"))

	tmpPath := path + ".tmp"
	if _, err := os.Stat(tmpPath); err == nil {
		t.Error("temp file should be removed after write")
	}
}

func TestWsBase(t *testing.T) {
	dir := t.TempDir()
	vol := NewLocal(dir).(*localStorage)
	sid := "ws-base-test"

	_ = os.MkdirAll(filepath.Join(dir, sid), 0o755)

	wsBase, err := vol.wsBase(sid)
	if err != nil {
		t.Fatalf("wsBase: %v", err)
	}

	expected := filepath.Join(dir, sid, ".workspace")
	if wsBase != expected {
		t.Errorf("wsBase = %q, want %q", wsBase, expected)
	}
}
