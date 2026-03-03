package common

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const lockfileName = "registry.yao"

// LoadLockfile reads registry.yao from appRoot. Returns an empty lockfile if
// the file does not exist.
func LoadLockfile(appRoot string) (*RegistryYao, error) {
	path := filepath.Join(appRoot, lockfileName)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &RegistryYao{
				Scope:    "@local",
				Packages: map[string]PackageInfo{},
			}, nil
		}
		return nil, fmt.Errorf("read %s: %w", lockfileName, err)
	}

	var lf RegistryYao
	if err := json.Unmarshal(data, &lf); err != nil {
		return nil, fmt.Errorf("parse %s: %w", lockfileName, err)
	}
	if lf.Packages == nil {
		lf.Packages = map[string]PackageInfo{}
	}
	if lf.Scope == "" {
		lf.Scope = "@local"
	}
	return &lf, nil
}

// SaveLockfile writes registry.yao to appRoot.
func SaveLockfile(appRoot string, lf *RegistryYao) error {
	path := filepath.Join(appRoot, lockfileName)
	data, err := json.MarshalIndent(lf, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal %s: %w", lockfileName, err)
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0644)
}

// GetPackage returns the package info and existence flag for a given ID.
func (lf *RegistryYao) GetPackage(pkgID string) (PackageInfo, bool) {
	info, ok := lf.Packages[pkgID]
	return info, ok
}

// SetPackage adds or updates a package entry.
func (lf *RegistryYao) SetPackage(pkgID string, info PackageInfo) {
	lf.Packages[pkgID] = info
}

// RemovePackage removes a package entry and cleans up required_by references.
func (lf *RegistryYao) RemovePackage(pkgID string) {
	pkg, ok := lf.Packages[pkgID]
	if !ok {
		return
	}

	// Remove this package from the required_by lists of its dependencies
	for depID := range pkg.Dependencies {
		if dep, exists := lf.Packages[depID]; exists {
			dep.RequiredBy = removeFromSlice(dep.RequiredBy, pkgID)
			lf.Packages[depID] = dep
		}
	}

	delete(lf.Packages, pkgID)
}

// AddRequiredBy adds a reverse dependency reference.
func (lf *RegistryYao) AddRequiredBy(depID, requiredByID string) {
	dep, ok := lf.Packages[depID]
	if !ok {
		return
	}
	for _, id := range dep.RequiredBy {
		if id == requiredByID {
			return
		}
	}
	dep.RequiredBy = append(dep.RequiredBy, requiredByID)
	lf.Packages[depID] = dep
}

// DefaultScope returns the user's default scope (from the "scope" field).
// Returns "local" (without @) for use in directory paths.
func (lf *RegistryYao) DefaultScope() string {
	scope := lf.Scope
	if scope == "" {
		scope = "@local"
	}
	if len(scope) > 0 && scope[0] == '@' {
		return scope[1:]
	}
	return scope
}

func removeFromSlice(s []string, item string) []string {
	result := make([]string, 0, len(s))
	for _, v := range s {
		if v != item {
			result = append(result, v)
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}
