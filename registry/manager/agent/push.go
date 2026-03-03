package agent

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/yaoapp/yao/registry/manager/common"
)

// PushOptions configures the Push operation.
type PushOptions struct {
	Version string // required semver
	Force   bool   // delete existing version before push
}

// Push packages and uploads an assistant to the registry.
// Flow per DESIGN-AGENT.md:
//  1. Yao ID → path
//  2. Validate package.yao exists
//  3. Derive scope/name from path
//  4. Reject @local
//  5. Pack directory (including embedded mcps/)
//  6. Scan external dependencies
//  7. Generate pkg.yao manifest
//  8. Push to registry
func (m *Manager) Push(yaoID string, opts PushOptions) error {
	if opts.Version == "" {
		return fmt.Errorf("--version is required for push")
	}

	scope, name, err := common.IDFromYaoID(yaoID)
	if err != nil {
		return fmt.Errorf("invalid assistant ID %q: %w", yaoID, err)
	}

	if common.IsLocalScope(scope) {
		return fmt.Errorf("cannot push @local packages. Fork to your own scope first")
	}

	assistantDir := common.PackageDir(common.TypeAssistant, scope, name, m.appRoot)

	// Validate package.yao exists
	pkgYaoPath := filepath.Join(assistantDir, "package.yao")
	if _, err := os.Stat(pkgYaoPath); err != nil {
		return fmt.Errorf("package.yao not found at %s", pkgYaoPath)
	}

	// Scan external dependencies
	scannedDeps, err := ScanDependencies(assistantDir, m.appRoot)
	if err != nil {
		fmt.Printf("⚠ Warning: could not scan dependencies: %v\n", err)
		scannedDeps = map[string]string{}
	}

	manifest := &common.PkgManifest{
		Type:         common.TypeAssistant,
		Scope:        "@" + scope,
		Name:         name,
		Version:      opts.Version,
		Dependencies: scannedDeps,
	}

	// Pack the directory
	zipData, err := common.PackDir(assistantDir, manifest, nil)
	if err != nil {
		return fmt.Errorf("pack: %w", err)
	}

	regType := common.TypeToRegistryType(common.TypeAssistant)

	// Force: delete existing version first (ignore 404)
	if opts.Force {
		m.client.DeleteVersion(regType, "@"+scope, name, opts.Version)
	}

	// Push to registry
	result, err := m.client.Push(regType, "@"+scope, name, opts.Version, zipData)
	if err != nil {
		return fmt.Errorf("push: %w", err)
	}

	fmt.Printf("✓ Pushed %s@%s (digest: %s)\n", common.FormatPackageID(scope, name), result.Version, result.Digest)
	return nil
}
