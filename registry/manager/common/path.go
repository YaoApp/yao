package common

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ParsePackageID parses "@scope/name" into (scope, name).
// The leading "@" on scope is stripped.
// Examples:
//
//	"@yao/keeper"       → ("yao", "keeper")
//	"@max/tools.search" → ("max", "tools.search")
func ParsePackageID(id string) (scope, name string, err error) {
	if !strings.HasPrefix(id, "@") {
		return "", "", fmt.Errorf("invalid package ID %q: must start with @", id)
	}

	parts := strings.SplitN(id[1:], "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid package ID %q: expected @scope/name", id)
	}
	return parts[0], parts[1], nil
}

// FormatPackageID formats scope and name into "@scope/name".
func FormatPackageID(scope, name string) string {
	return "@" + scope + "/" + name
}

// PackageDir returns the installation directory for a package relative to appRoot.
// For assistants: assistants/{scope}/{name}/
// For mcps:       mcps/{scope}/{name}/
func PackageDir(pkgType, scope, name, appRoot string) string {
	dir := TypeToDir(pkgType)
	namePath := strings.ReplaceAll(name, ".", "/")
	return filepath.Join(appRoot, dir, scope, namePath)
}

// PackageDirRel returns the installation directory relative to appRoot (no leading appRoot prefix).
func PackageDirRel(pkgType, scope, name string) string {
	dir := TypeToDir(pkgType)
	namePath := strings.ReplaceAll(name, ".", "/")
	return filepath.Join(dir, scope, namePath)
}

// IDFromYaoID converts a Yao dot-separated ID to (scope, name).
// "yao.keeper"         → ("yao", "keeper")
// "max.tools.search"   → ("max", "tools.search")
func IDFromYaoID(yaoID string) (scope, name string, err error) {
	idx := strings.Index(yaoID, ".")
	if idx <= 0 || idx >= len(yaoID)-1 {
		return "", "", fmt.Errorf("invalid Yao ID %q: expected scope.name", yaoID)
	}
	return yaoID[:idx], yaoID[idx+1:], nil
}

// YaoIDFromPackageID converts "@scope/name" to "scope.name" (Yao dot-separated ID).
func YaoIDFromPackageID(pkgID string) (string, error) {
	scope, name, err := ParsePackageID(pkgID)
	if err != nil {
		return "", err
	}
	return scope + "." + name, nil
}

// PackageIDFromYaoID converts "scope.name" to "@scope/name".
func PackageIDFromYaoID(yaoID string) (string, error) {
	scope, name, err := IDFromYaoID(yaoID)
	if err != nil {
		return "", err
	}
	return FormatPackageID(scope, name), nil
}

// ScopeFromPath extracts the scope from a file path relative to appRoot.
// "assistants/yao/keeper/" → "yao"
// "mcps/max/rag-tools/"    → "max"
func ScopeFromPath(relPath string) (string, error) {
	parts := strings.Split(filepath.ToSlash(relPath), "/")
	if len(parts) < 2 {
		return "", fmt.Errorf("cannot extract scope from path %q", relPath)
	}
	return parts[1], nil
}

// IsLocalScope returns true if the scope is "@local" or "local".
func IsLocalScope(scope string) bool {
	return scope == "local" || scope == "@local"
}
