package common

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestPackAndUnpack(t *testing.T) {
	// Create source directory
	srcDir := t.TempDir()
	os.WriteFile(filepath.Join(srcDir, "package.yao"), []byte(`{"name":"test"}`), 0644)
	os.MkdirAll(filepath.Join(srcDir, "prompts"), 0755)
	os.WriteFile(filepath.Join(srcDir, "prompts", "main.md"), []byte("You are a test."), 0644)

	manifest := &PkgManifest{
		Type:    TypeAssistant,
		Scope:   "test",
		Name:    "demo",
		Version: "1.0.0",
	}

	zipData, err := PackDir(srcDir, manifest, nil)
	if err != nil {
		t.Fatalf("PackDir: %v", err)
	}
	if len(zipData) == 0 {
		t.Fatal("expected non-empty zip")
	}

	// Read manifest from zip
	m, err := ReadManifest(zipData)
	if err != nil {
		t.Fatalf("ReadManifest: %v", err)
	}
	if m.Type != TypeAssistant || m.Version != "1.0.0" {
		t.Errorf("unexpected manifest: %+v", m)
	}

	// Unpack
	destDir := t.TempDir()
	files, err := UnpackTo(zipData, destDir)
	if err != nil {
		t.Fatalf("UnpackTo: %v", err)
	}

	sort.Strings(files)
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d: %v", len(files), files)
	}
	if files[0] != "package.yao" || files[1] != "prompts/main.md" {
		t.Errorf("unexpected files: %v", files)
	}

	// Verify content
	data, err := os.ReadFile(filepath.Join(destDir, "package.yao"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `{"name":"test"}` {
		t.Errorf("unexpected content: %s", data)
	}
}

func TestPackDirWithExtraFiles(t *testing.T) {
	srcDir := t.TempDir()
	os.WriteFile(filepath.Join(srcDir, "main.mcp.yao"), []byte("{}"), 0644)

	// Create an extra file in a separate location
	extraDir := t.TempDir()
	os.MkdirAll(filepath.Join(extraDir, "scripts", "yao"), 0755)
	scriptPath := filepath.Join(extraDir, "scripts", "yao", "rag.ts")
	os.WriteFile(scriptPath, []byte("export function Search() {}"), 0644)

	manifest := &PkgManifest{
		Type:    TypeMCP,
		Scope:   "yao",
		Name:    "rag-tools",
		Version: "1.0.0",
	}

	extraFiles := map[string]string{
		"scripts/yao/rag.ts": scriptPath,
	}

	zipData, err := PackDir(srcDir, manifest, extraFiles)
	if err != nil {
		t.Fatalf("PackDir with extras: %v", err)
	}

	files, err := ListZipFiles(zipData)
	if err != nil {
		t.Fatal(err)
	}

	hasScript := false
	hasMCP := false
	for _, f := range files {
		if f == "scripts/yao/rag.ts" {
			hasScript = true
		}
		if f == "main.mcp.yao" {
			hasMCP = true
		}
	}
	if !hasScript {
		t.Error("expected scripts/yao/rag.ts in zip")
	}
	if !hasMCP {
		t.Error("expected main.mcp.yao in zip")
	}
}

func TestReadManifestMissing(t *testing.T) {
	// Create a zip without pkg.yao
	srcDir := t.TempDir()
	os.WriteFile(filepath.Join(srcDir, "test.txt"), []byte("hello"), 0644)

	manifest := &PkgManifest{Type: "test", Version: "1.0.0"}
	zipData, err := PackDir(srcDir, manifest, nil)
	if err != nil {
		t.Fatal(err)
	}

	// This should succeed because PackDir always writes pkg.yao
	m, err := ReadManifest(zipData)
	if err != nil {
		t.Fatalf("ReadManifest should succeed: %v", err)
	}
	if m.Type != "test" {
		t.Errorf("unexpected type: %s", m.Type)
	}
}

func TestExtractFile(t *testing.T) {
	srcDir := t.TempDir()
	os.WriteFile(filepath.Join(srcDir, "data.json"), []byte(`{"key":"value"}`), 0644)

	manifest := &PkgManifest{Type: "test", Version: "1.0.0"}
	zipData, err := PackDir(srcDir, manifest, nil)
	if err != nil {
		t.Fatal(err)
	}

	data, err := ExtractFile(zipData, "data.json")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `{"key":"value"}` {
		t.Errorf("unexpected: %s", data)
	}

	_, err = ExtractFile(zipData, "missing.json")
	if err == nil {
		t.Error("expected error for missing file")
	}
}
