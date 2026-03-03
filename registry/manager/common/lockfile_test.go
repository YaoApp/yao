package common

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadLockfileNotExist(t *testing.T) {
	dir := t.TempDir()
	lf, err := LoadLockfile(dir)
	if err != nil {
		t.Fatal(err)
	}
	if lf.Scope != "@local" {
		t.Errorf("expected @local scope, got %q", lf.Scope)
	}
	if len(lf.Packages) != 0 {
		t.Errorf("expected empty packages")
	}
}

func TestLoadAndSaveLockfile(t *testing.T) {
	dir := t.TempDir()

	lf := &RegistryYao{
		Scope: "@local",
		Packages: map[string]PackageInfo{
			"@yao/keeper": {
				Type:    TypeAssistant,
				Version: "2.0.0",
				Files:   map[string]string{"package.yao": "sha256-aaa"},
			},
		},
	}

	if err := SaveLockfile(dir, lf); err != nil {
		t.Fatal(err)
	}

	// Verify file exists
	data, err := os.ReadFile(filepath.Join(dir, "registry.yao"))
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Fatal("expected non-empty file")
	}

	// Reload
	lf2, err := LoadLockfile(dir)
	if err != nil {
		t.Fatal(err)
	}
	if lf2.Scope != "@local" {
		t.Errorf("scope mismatch: %q", lf2.Scope)
	}
	pkg, ok := lf2.GetPackage("@yao/keeper")
	if !ok {
		t.Fatal("expected @yao/keeper in lockfile")
	}
	if pkg.Version != "2.0.0" {
		t.Errorf("version mismatch: %q", pkg.Version)
	}
	if pkg.Files["package.yao"] != "sha256-aaa" {
		t.Error("files hash mismatch")
	}
}

func TestLoadLockfileInvalid(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "registry.yao"), []byte("not json"), 0644)

	_, err := LoadLockfile(dir)
	if err == nil {
		t.Error("expected parse error")
	}
}

func TestSetAndGetPackage(t *testing.T) {
	lf := &RegistryYao{Packages: map[string]PackageInfo{}}
	lf.SetPackage("@yao/test", PackageInfo{Type: TypeMCP, Version: "1.0.0"})

	pkg, ok := lf.GetPackage("@yao/test")
	if !ok {
		t.Fatal("expected package")
	}
	if pkg.Type != TypeMCP {
		t.Errorf("type mismatch: %q", pkg.Type)
	}
}

func TestRemovePackage(t *testing.T) {
	lf := &RegistryYao{
		Packages: map[string]PackageInfo{
			"@yao/keeper": {
				Type:         TypeAssistant,
				Version:      "1.0.0",
				Dependencies: map[string]string{"@yao/rag-tools": "^1.0.0"},
			},
			"@yao/rag-tools": {
				Type:       TypeMCP,
				Version:    "1.0.0",
				RequiredBy: []string{"@yao/keeper"},
			},
		},
	}

	lf.RemovePackage("@yao/keeper")

	if _, ok := lf.GetPackage("@yao/keeper"); ok {
		t.Error("expected @yao/keeper removed")
	}

	dep, ok := lf.GetPackage("@yao/rag-tools")
	if !ok {
		t.Fatal("expected @yao/rag-tools still present")
	}
	if len(dep.RequiredBy) != 0 {
		t.Errorf("expected required_by cleaned up, got %v", dep.RequiredBy)
	}
}

func TestAddRequiredBy(t *testing.T) {
	lf := &RegistryYao{
		Packages: map[string]PackageInfo{
			"@yao/rag-tools": {Type: TypeMCP, Version: "1.0.0"},
		},
	}

	lf.AddRequiredBy("@yao/rag-tools", "@yao/keeper")
	lf.AddRequiredBy("@yao/rag-tools", "@yao/keeper") // duplicate, should not add again

	dep, _ := lf.GetPackage("@yao/rag-tools")
	if len(dep.RequiredBy) != 1 {
		t.Errorf("expected 1 required_by entry, got %d", len(dep.RequiredBy))
	}
	if dep.RequiredBy[0] != "@yao/keeper" {
		t.Errorf("expected @yao/keeper, got %q", dep.RequiredBy[0])
	}

	// Non-existent package should be a no-op
	lf.AddRequiredBy("@nonexistent/pkg", "@yao/keeper")
}

func TestDefaultScope(t *testing.T) {
	lf := &RegistryYao{Scope: "@local"}
	if lf.DefaultScope() != "local" {
		t.Errorf("expected local, got %q", lf.DefaultScope())
	}

	lf.Scope = "@max"
	if lf.DefaultScope() != "max" {
		t.Errorf("expected max, got %q", lf.DefaultScope())
	}

	lf.Scope = ""
	if lf.DefaultScope() != "local" {
		t.Errorf("expected local for empty scope, got %q", lf.DefaultScope())
	}
}

func TestIsManaged(t *testing.T) {
	p := PackageInfo{}
	if !p.IsManaged() {
		t.Error("nil managed should be true")
	}

	p.Managed = BoolPtr(false)
	if p.IsManaged() {
		t.Error("false managed should be false")
	}

	p.Managed = BoolPtr(true)
	if !p.IsManaged() {
		t.Error("true managed should be true")
	}
}
