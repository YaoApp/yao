package common

import (
	"testing"
)

func TestParsePackageID(t *testing.T) {
	tests := []struct {
		input     string
		scope     string
		name      string
		expectErr bool
	}{
		{"@yao/keeper", "yao", "keeper", false},
		{"@max/tools.search", "max", "tools.search", false},
		{"@local/my-mcp", "local", "my-mcp", false},
		{"yao/keeper", "", "", true},
		{"@/keeper", "", "", true},
		{"@yao/", "", "", true},
		{"@yao", "", "", true},
		{"", "", "", true},
	}

	for _, tt := range tests {
		scope, name, err := ParsePackageID(tt.input)
		if tt.expectErr {
			if err == nil {
				t.Errorf("ParsePackageID(%q) expected error", tt.input)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParsePackageID(%q) unexpected error: %v", tt.input, err)
			continue
		}
		if scope != tt.scope || name != tt.name {
			t.Errorf("ParsePackageID(%q) = (%q, %q), want (%q, %q)", tt.input, scope, name, tt.scope, tt.name)
		}
	}
}

func TestFormatPackageID(t *testing.T) {
	if got := FormatPackageID("yao", "keeper"); got != "@yao/keeper" {
		t.Errorf("FormatPackageID = %q, want @yao/keeper", got)
	}
}

func TestPackageDir(t *testing.T) {
	got := PackageDir(TypeAssistant, "yao", "keeper", "/app")
	want := "/app/assistants/yao/keeper"
	if got != want {
		t.Errorf("PackageDir = %q, want %q", got, want)
	}

	got = PackageDir(TypeMCP, "max", "rag-tools", "/app")
	want = "/app/mcps/max/rag-tools"
	if got != want {
		t.Errorf("PackageDir = %q, want %q", got, want)
	}

	got = PackageDir(TypeAssistant, "max", "tools.search", "/app")
	want = "/app/assistants/max/tools/search"
	if got != want {
		t.Errorf("PackageDir nested = %q, want %q", got, want)
	}
}

func TestPackageDirRel(t *testing.T) {
	got := PackageDirRel(TypeAssistant, "yao", "keeper")
	want := "assistants/yao/keeper"
	if got != want {
		t.Errorf("PackageDirRel = %q, want %q", got, want)
	}
}

func TestIDFromYaoID(t *testing.T) {
	tests := []struct {
		input     string
		scope     string
		name      string
		expectErr bool
	}{
		{"yao.keeper", "yao", "keeper", false},
		{"max.tools.search", "max", "tools.search", false},
		{"yao", "", "", true},
		{".keeper", "", "", true},
		{"yao.", "", "", true},
	}

	for _, tt := range tests {
		scope, name, err := IDFromYaoID(tt.input)
		if tt.expectErr {
			if err == nil {
				t.Errorf("IDFromYaoID(%q) expected error", tt.input)
			}
			continue
		}
		if err != nil {
			t.Errorf("IDFromYaoID(%q) unexpected error: %v", tt.input, err)
			continue
		}
		if scope != tt.scope || name != tt.name {
			t.Errorf("IDFromYaoID(%q) = (%q, %q), want (%q, %q)", tt.input, scope, name, tt.scope, tt.name)
		}
	}
}

func TestYaoIDFromPackageID(t *testing.T) {
	got, err := YaoIDFromPackageID("@yao/keeper")
	if err != nil {
		t.Fatal(err)
	}
	if got != "yao.keeper" {
		t.Errorf("YaoIDFromPackageID = %q, want yao.keeper", got)
	}

	_, err = YaoIDFromPackageID("bad")
	if err == nil {
		t.Error("expected error for invalid input")
	}
}

func TestPackageIDFromYaoID(t *testing.T) {
	got, err := PackageIDFromYaoID("yao.keeper")
	if err != nil {
		t.Fatal(err)
	}
	if got != "@yao/keeper" {
		t.Errorf("PackageIDFromYaoID = %q, want @yao/keeper", got)
	}

	_, err = PackageIDFromYaoID("bad")
	if err == nil {
		t.Error("expected error for invalid input")
	}
}

func TestScopeFromPath(t *testing.T) {
	got, err := ScopeFromPath("assistants/yao/keeper")
	if err != nil {
		t.Fatal(err)
	}
	if got != "yao" {
		t.Errorf("ScopeFromPath = %q, want yao", got)
	}

	_, err = ScopeFromPath("single")
	if err == nil {
		t.Error("expected error for single-element path")
	}
}

func TestIsLocalScope(t *testing.T) {
	if !IsLocalScope("local") {
		t.Error("expected local to be local scope")
	}
	if !IsLocalScope("@local") {
		t.Error("expected @local to be local scope")
	}
	if IsLocalScope("yao") {
		t.Error("expected yao to NOT be local scope")
	}
}
