// Package common provides shared types and utilities for the registry manager.
package common

import "encoding/json"

// RegistryYao represents the registry.yao lockfile that tracks installed packages.
type RegistryYao struct {
	Scope    string                 `json:"scope"`
	Packages map[string]PackageInfo `json:"packages"`
}

// PackageInfo describes an installed package in registry.yao.
type PackageInfo struct {
	Type         string            `json:"type"`
	Version      string            `json:"version"`
	Integrity    string            `json:"integrity"`
	Dependencies map[string]string `json:"dependencies,omitempty"`
	RequiredBy   []string          `json:"required_by,omitempty"`
	Files        map[string]string `json:"files,omitempty"`
	Managed      *bool             `json:"managed,omitempty"`
	ForkedFrom   string            `json:"forked_from,omitempty"`
	MemberID     string            `json:"member_id,omitempty"`
	TeamID       string            `json:"team_id,omitempty"`
}

// IsManaged returns true if the package is managed by the registry (not forked).
func (p *PackageInfo) IsManaged() bool {
	if p.Managed == nil {
		return true
	}
	return *p.Managed
}

// PkgManifest represents the pkg.yao file inside a .yao.zip package.
// Dependencies can come from the registry in array format [{type,scope,name,version}]
// and are normalized to map["@scope/name"] = "version" after loading.
type PkgManifest struct {
	Type            string            `json:"type"`
	Scope           string            `json:"scope"`
	Name            string            `json:"name"`
	Version         string            `json:"version"`
	Description     string            `json:"description,omitempty"`
	Dependencies    map[string]string `json:"-"`
	RawDependencies json.RawMessage   `json:"dependencies,omitempty"`
	Keywords        []string          `json:"keywords,omitempty"`
	License         string            `json:"license,omitempty"`
	Author          *ManifestAuthor   `json:"author,omitempty"`
	Engines         map[string]string `json:"engines,omitempty"`
}

// ManifestDep represents a dependency entry in the array format from the registry.
type ManifestDep struct {
	Type    string `json:"type"`
	Scope   string `json:"scope"`
	Name    string `json:"name"`
	Version string `json:"version"`
}

// NormalizeDependencies parses RawDependencies into the Dependencies map.
// Supports both formats:
//   - Array: [{"type":"mcp","scope":"@test","name":"dep","version":"^1.0.0"}]
//   - Map: {"@test/dep": "^1.0.0"}
func (m *PkgManifest) NormalizeDependencies() {
	if m.Dependencies != nil || len(m.RawDependencies) == 0 {
		return
	}
	m.Dependencies = map[string]string{}

	// Try array format first
	var arrDeps []ManifestDep
	if err := json.Unmarshal(m.RawDependencies, &arrDeps); err == nil {
		for _, d := range arrDeps {
			scope := d.Scope
			if len(scope) > 0 && scope[0] != '@' {
				scope = "@" + scope
			}
			pkgID := scope + "/" + d.Name
			m.Dependencies[pkgID] = d.Version
		}
		return
	}

	// Try map format
	var mapDeps map[string]string
	if err := json.Unmarshal(m.RawDependencies, &mapDeps); err == nil {
		m.Dependencies = mapDeps
	}
}

// PrepareMarshal syncs Dependencies map into RawDependencies for JSON serialization.
// Converts the internal map["@scope/name"] = "version" format to the array format
// required by the registry server: [{"type":"...","scope":"@...","name":"...","version":"..."}].
func (m *PkgManifest) PrepareMarshal() {
	if len(m.Dependencies) == 0 {
		return
	}
	var arr []ManifestDep
	for pkgID, ver := range m.Dependencies {
		scope, name, err := ParsePackageID(pkgID)
		if err != nil {
			continue
		}
		depType := TypeDirMCPs
		arr = append(arr, ManifestDep{
			Type:    depType,
			Scope:   "@" + scope,
			Name:    name,
			Version: ver,
		})
	}
	data, err := json.Marshal(arr)
	if err == nil {
		m.RawDependencies = data
	}
}

// ManifestAuthor holds author information in pkg.yao.
type ManifestAuthor struct {
	Name  string `json:"name"`
	Email string `json:"email,omitempty"`
}

// PackageType constants map to registry API type strings and local directory names.
const (
	TypeAssistant = "assistant"
	TypeMCP       = "mcp"
	TypeRobot     = "robot"

	TypeDirAssistants = "assistants"
	TypeDirMCPs       = "mcps"
	TypeDirRobots     = "robots"
)

// TypeToDir maps package type to its top-level directory name.
func TypeToDir(pkgType string) string {
	switch pkgType {
	case TypeAssistant:
		return TypeDirAssistants
	case TypeMCP:
		return TypeDirMCPs
	case TypeRobot:
		return TypeDirRobots
	default:
		return pkgType
	}
}

// TypeToRegistryType maps package type to the registry API type string.
func TypeToRegistryType(pkgType string) string {
	switch pkgType {
	case TypeAssistant:
		return TypeDirAssistants
	case TypeMCP:
		return TypeDirMCPs
	case TypeRobot:
		return TypeDirRobots
	default:
		return pkgType
	}
}

// BoolPtr returns a pointer to a bool value.
func BoolPtr(v bool) *bool {
	return &v
}
