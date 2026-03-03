package common

import (
	"testing"
)

func TestVersionSatisfies(t *testing.T) {
	tests := []struct {
		installed  string
		constraint string
		want       bool
	}{
		{"1.0.0", "^1.0.0", true},
		{"1.2.3", "^1.0.0", true},
		{"1.0.1", "^1.0.0", true},
		{"2.0.0", "^1.0.0", false},
		{"0.9.0", "^1.0.0", false},
		{"1.0.0", ">=1.0.0", true},
		{"2.0.0", ">=1.0.0", true},
		{"0.9.0", ">=1.0.0", false},
		{"1.0.0", "1.0.0", true},
		{"1.0.1", "1.0.0", false},
		{"1.0.0", "*", true},
		{"1.0.0", "", true},
		{"1.3.0", "^1.0.0", true},
		{"1.0.0", "^1.3.0", false},
	}

	for _, tt := range tests {
		got := VersionSatisfies(tt.installed, tt.constraint)
		if got != tt.want {
			t.Errorf("VersionSatisfies(%q, %q) = %v, want %v", tt.installed, tt.constraint, got, tt.want)
		}
	}
}

func TestCheckDependencies(t *testing.T) {
	lf := &RegistryYao{
		Packages: map[string]PackageInfo{
			"@yao/rag-tools":  {Type: TypeMCP, Version: "1.3.0"},
			"@yao/title-gen":  {Type: TypeAssistant, Version: "1.0.0"},
			"@yao/old-helper": {Type: TypeAssistant, Version: "0.5.0"},
		},
	}

	deps := map[string]string{
		"@yao/rag-tools":  "^1.0.0",
		"@yao/title-gen":  "^2.0.0", // conflict: installed 1.0.0, needs ^2.0.0
		"@yao/old-helper": "^0.5.0",
		"@yao/new-pkg":    "^1.0.0", // missing
	}

	missing, conflicts, satisfied := CheckDependencies(deps, lf)

	if len(missing) != 1 {
		t.Fatalf("expected 1 missing, got %d", len(missing))
	}
	if missing[0].PackageID != "@yao/new-pkg" {
		t.Errorf("expected @yao/new-pkg missing, got %s", missing[0].PackageID)
	}

	if len(conflicts) != 1 {
		t.Fatalf("expected 1 conflict, got %d", len(conflicts))
	}
	if conflicts[0].PackageID != "@yao/title-gen" {
		t.Errorf("expected @yao/title-gen conflict, got %s", conflicts[0].PackageID)
	}
	if conflicts[0].InstalledVersion != "1.0.0" {
		t.Errorf("expected installed 1.0.0, got %s", conflicts[0].InstalledVersion)
	}

	if len(satisfied) != 2 {
		t.Fatalf("expected 2 satisfied, got %d", len(satisfied))
	}
}

func TestCheckDependenciesEmptyLockfile(t *testing.T) {
	lf := &RegistryYao{Packages: map[string]PackageInfo{}}
	deps := map[string]string{
		"@yao/a": "^1.0.0",
		"@yao/b": "^2.0.0",
	}

	missing, conflicts, satisfied := CheckDependencies(deps, lf)
	if len(missing) != 2 {
		t.Errorf("expected 2 missing, got %d", len(missing))
	}
	if len(conflicts) != 0 {
		t.Errorf("expected 0 conflicts, got %d", len(conflicts))
	}
	if len(satisfied) != 0 {
		t.Errorf("expected 0 satisfied, got %d", len(satisfied))
	}
}

func TestDetectCycle(t *testing.T) {
	installing := map[string]bool{
		"@yao/keeper": true,
	}

	if !DetectCycle(installing, "@yao/keeper") {
		t.Error("expected cycle detected for @yao/keeper")
	}
	if DetectCycle(installing, "@yao/other") {
		t.Error("expected no cycle for @yao/other")
	}
}

func TestCompareVersions(t *testing.T) {
	if compareVersions("1.0.0", "1.0.0") != 0 {
		t.Error("1.0.0 == 1.0.0")
	}
	if compareVersions("2.0.0", "1.0.0") <= 0 {
		t.Error("2.0.0 > 1.0.0")
	}
	if compareVersions("1.0.0", "2.0.0") >= 0 {
		t.Error("1.0.0 < 2.0.0")
	}
	if compareVersions("1.1.0", "1.0.0") <= 0 {
		t.Error("1.1.0 > 1.0.0")
	}
}

func TestParseVersionInvalid(t *testing.T) {
	_, _, _, err := parseVersion("bad")
	if err == nil {
		t.Error("expected error")
	}
	_, _, _, err = parseVersion("1.2")
	if err == nil {
		t.Error("expected error for 2-part version")
	}
}
