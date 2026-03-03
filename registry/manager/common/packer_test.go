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

func TestPackDirBuiltinIgnore(t *testing.T) {
	srcDir := t.TempDir()
	os.WriteFile(filepath.Join(srcDir, "package.yao"), []byte(`{}`), 0644)
	os.WriteFile(filepath.Join(srcDir, "prompts.md"), []byte("hello"), 0644)

	// Files that should be excluded by built-in defaults
	os.WriteFile(filepath.Join(srcDir, ".DS_Store"), []byte{}, 0644)
	os.WriteFile(filepath.Join(srcDir, "debug.swp"), []byte{}, 0644)
	os.WriteFile(filepath.Join(srcDir, "notes.bak"), []byte{}, 0644)
	os.MkdirAll(filepath.Join(srcDir, ".git", "objects"), 0755)
	os.WriteFile(filepath.Join(srcDir, ".git", "config"), []byte{}, 0644)
	os.MkdirAll(filepath.Join(srcDir, ".vscode"), 0755)
	os.WriteFile(filepath.Join(srcDir, ".vscode", "settings.json"), []byte{}, 0644)
	os.MkdirAll(filepath.Join(srcDir, "node_modules", "foo"), 0755)
	os.WriteFile(filepath.Join(srcDir, "node_modules", "foo", "index.js"), []byte{}, 0644)

	manifest := &PkgManifest{Type: TypeAssistant, Scope: "test", Name: "ign", Version: "1.0.0"}
	zipData, err := PackDir(srcDir, manifest, nil)
	if err != nil {
		t.Fatalf("PackDir: %v", err)
	}

	files, err := ListZipFiles(zipData)
	if err != nil {
		t.Fatal(err)
	}

	fileSet := map[string]bool{}
	for _, f := range files {
		fileSet[f] = true
	}

	if !fileSet["package.yao"] {
		t.Error("expected package.yao in zip")
	}
	if !fileSet["prompts.md"] {
		t.Error("expected prompts.md in zip")
	}
	for _, excluded := range []string{".DS_Store", "debug.swp", "notes.bak", ".git/config", ".vscode/settings.json", "node_modules/foo/index.js"} {
		if fileSet[excluded] {
			t.Errorf("expected %s to be excluded from zip", excluded)
		}
	}
}

func TestPackDirYaoignoreFile(t *testing.T) {
	srcDir := t.TempDir()
	os.WriteFile(filepath.Join(srcDir, "package.yao"), []byte(`{}`), 0644)
	os.WriteFile(filepath.Join(srcDir, "keep.txt"), []byte("keep"), 0644)
	os.WriteFile(filepath.Join(srcDir, "secret.key"), []byte("secret"), 0644)
	os.MkdirAll(filepath.Join(srcDir, "drafts"), 0755)
	os.WriteFile(filepath.Join(srcDir, "drafts", "notes.md"), []byte("draft"), 0644)
	os.WriteFile(filepath.Join(srcDir, "test.log"), []byte("log"), 0644)

	// .yaoignore excludes *.key and drafts/
	os.WriteFile(filepath.Join(srcDir, ".yaoignore"), []byte("*.key\ndrafts/\n"), 0644)

	manifest := &PkgManifest{Type: TypeAssistant, Scope: "test", Name: "ign2", Version: "1.0.0"}
	zipData, err := PackDir(srcDir, manifest, nil)
	if err != nil {
		t.Fatalf("PackDir: %v", err)
	}

	files, err := ListZipFiles(zipData)
	if err != nil {
		t.Fatal(err)
	}

	fileSet := map[string]bool{}
	for _, f := range files {
		fileSet[f] = true
	}

	if !fileSet["package.yao"] {
		t.Error("expected package.yao")
	}
	if !fileSet["keep.txt"] {
		t.Error("expected keep.txt")
	}
	if fileSet["secret.key"] {
		t.Error("secret.key should be excluded by .yaoignore")
	}
	if fileSet["drafts/notes.md"] {
		t.Error("drafts/notes.md should be excluded by .yaoignore")
	}
	if fileSet["test.log"] {
		t.Error("test.log should be excluded by built-in *.log pattern")
	}
	if fileSet[".yaoignore"] {
		t.Error(".yaoignore itself should be excluded")
	}
}

func TestPackDirYaoignoreNegation(t *testing.T) {
	srcDir := t.TempDir()
	os.WriteFile(filepath.Join(srcDir, "package.yao"), []byte(`{}`), 0644)
	os.WriteFile(filepath.Join(srcDir, "a.tmp"), []byte("tmp"), 0644)
	os.WriteFile(filepath.Join(srcDir, "important.tmp"), []byte("keep"), 0644)

	// *.tmp is in defaults, but negate important.tmp
	os.WriteFile(filepath.Join(srcDir, ".yaoignore"), []byte("!important.tmp\n"), 0644)

	manifest := &PkgManifest{Type: TypeAssistant, Scope: "test", Name: "neg", Version: "1.0.0"}
	zipData, err := PackDir(srcDir, manifest, nil)
	if err != nil {
		t.Fatalf("PackDir: %v", err)
	}

	files, err := ListZipFiles(zipData)
	if err != nil {
		t.Fatal(err)
	}

	fileSet := map[string]bool{}
	for _, f := range files {
		fileSet[f] = true
	}

	if fileSet["a.tmp"] {
		t.Error("a.tmp should be excluded by built-in *.tmp")
	}
	if !fileSet["important.tmp"] {
		t.Error("important.tmp should be included via negation in .yaoignore")
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
