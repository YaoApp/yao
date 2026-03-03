package common

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHashFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(path, []byte("hello world"), 0644); err != nil {
		t.Fatal(err)
	}

	hash, err := HashFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(hash, "sha256-") {
		t.Errorf("expected sha256- prefix, got %q", hash)
	}
	if len(hash) != 7+64 { // "sha256-" + 64 hex chars
		t.Errorf("unexpected hash length: %d", len(hash))
	}

	// Same content should produce same hash
	hash2, _ := HashFile(path)
	if hash != hash2 {
		t.Error("same file should produce same hash")
	}

	// Non-existent file
	_, err = HashFile(filepath.Join(dir, "nope.txt"))
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestHashBytes(t *testing.T) {
	h := HashBytes([]byte("hello world"))
	if !strings.HasPrefix(h, "sha256-") {
		t.Errorf("expected sha256- prefix, got %q", h)
	}

	h2 := HashBytes([]byte("hello world"))
	if h != h2 {
		t.Error("same bytes should produce same hash")
	}

	h3 := HashBytes([]byte("different"))
	if h == h3 {
		t.Error("different bytes should produce different hash")
	}
}

func TestHashDir(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "sub"), 0755)
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("aaa"), 0644)
	os.WriteFile(filepath.Join(dir, "sub", "b.txt"), []byte("bbb"), 0644)

	// Without prefix
	hashes, err := HashDir(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(hashes) != 2 {
		t.Errorf("expected 2 files, got %d", len(hashes))
	}
	if _, ok := hashes["a.txt"]; !ok {
		t.Error("expected a.txt in hashes")
	}
	if _, ok := hashes["sub/b.txt"]; !ok {
		t.Error("expected sub/b.txt in hashes")
	}

	// With prefix
	hashes, err = HashDir(dir, "mcps/yao/rag-tools")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := hashes["mcps/yao/rag-tools/a.txt"]; !ok {
		t.Error("expected prefixed path")
	}
}
