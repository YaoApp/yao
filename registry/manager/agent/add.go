package agent

import (
	"fmt"
	"os"
	"strings"

	"github.com/yaoapp/yao/registry/manager/common"
)

// AddOptions configures the Add operation.
type AddOptions struct {
	Version string // version or dist-tag, default "latest"
	Force   bool   // force reinstall even if already installed
}

// Add installs an assistant package from the registry.
// Flow per DESIGN-AGENT.md:
//  1. Parse @scope/name
//  2. Check target path conflict
//  3. Pull from registry
//  4. Check and install dependencies (recursive)
//  5. Unpack to assistants/{scope}/{name}/
//  6. Compute file hashes, write registry.yao
//  7. Hot-reload
func (m *Manager) Add(pkgID string, opts AddOptions) error {
	if opts.Version == "" {
		opts.Version = "latest"
	}

	scope, name, err := common.ParsePackageID(pkgID)
	if err != nil {
		return err
	}

	lf, err := common.LoadLockfile(m.appRoot)
	if err != nil {
		return err
	}

	// Check if already installed
	if existing, ok := lf.GetPackage(pkgID); ok && !opts.Force {
		return fmt.Errorf("package %s is already installed (version %s). Use --force to reinstall", pkgID, existing.Version)
	}

	// Check directory conflict
	destDir := common.PackageDir(common.TypeAssistant, scope, name, m.appRoot)
	if _, err := os.Stat(destDir); err == nil {
		if _, ok := lf.GetPackage(pkgID); !ok {
			return fmt.Errorf("directory %s already exists but is not managed by registry. Please remove or relocate it first", destDir)
		}
	}

	// Pull from registry
	regType := common.TypeToRegistryType(common.TypeAssistant)
	zipData, digest, err := m.client.Pull(regType, "@"+scope, name, opts.Version)
	if err != nil {
		return fmt.Errorf("pull %s: %w", pkgID, err)
	}

	// Read manifest
	manifest, err := common.ReadManifest(zipData)
	if err != nil {
		return fmt.Errorf("read manifest: %w", err)
	}

	// Install dependencies first
	if len(manifest.Dependencies) > 0 {
		if err := m.installDependencies(manifest.Dependencies, lf, pkgID, map[string]bool{pkgID: true}); err != nil {
			return err
		}
	}

	// Unpack to destination
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
	}
	if _, err := common.UnpackTo(zipData, destDir); err != nil {
		return fmt.Errorf("unpack: %w", err)
	}

	// Compute file hashes
	relDir := common.PackageDirRel(common.TypeAssistant, scope, name)
	files, err := common.HashDir(destDir, relDir)
	if err != nil {
		return fmt.Errorf("hash files: %w", err)
	}

	// Update lockfile
	info := common.PackageInfo{
		Type:         common.TypeAssistant,
		Version:      manifest.Version,
		Integrity:    digest,
		Dependencies: manifest.Dependencies,
		Files:        files,
	}
	lf.SetPackage(pkgID, info)

	// Update required_by on dependencies
	for depID := range manifest.Dependencies {
		lf.AddRequiredBy(depID, pkgID)
	}

	if err := common.SaveLockfile(m.appRoot, lf); err != nil {
		return err
	}

	// Hot-reload: in production this calls assistant.LoadPath().
	// For the manager package we keep it as a no-op since LoadPath requires
	// the full engine runtime. The CLI layer will handle hot-reload.
	fmt.Printf("✓ Installed %s@%s → %s\n", pkgID, manifest.Version, destDir)
	return nil
}

// installDependencies recursively installs missing dependencies.
func (m *Manager) installDependencies(deps map[string]string, lf *common.RegistryYao, parentID string, installing map[string]bool) error {
	missing, conflicts, _ := common.CheckDependencies(deps, lf)

	// Handle conflicts
	for _, c := range conflicts {
		msg := fmt.Sprintf(
			"⚠ %s is currently %s (required by %s)\n  %s requires %s\n",
			c.PackageID, c.InstalledVersion, parentID, parentID, c.RequiredVersion,
		)
		options := []string{
			fmt.Sprintf("Upgrade %s (may break other dependents)", c.PackageID),
			"Keep current version",
			"Abort installation",
		}
		choice := m.prompter.Choose(msg, options)
		switch choice {
		case 0:
			// Upgrade: treat as missing so it gets reinstalled
			missing = append(missing, c)
		case 1:
			// Keep current
			continue
		default:
			return fmt.Errorf("installation aborted by user")
		}
	}

	if len(missing) == 0 {
		return nil
	}

	// Build summary
	var summary strings.Builder
	summary.WriteString("The following dependencies need to be installed:\n")
	for _, dep := range missing {
		summary.WriteString(fmt.Sprintf("  %s %s\n", dep.PackageID, dep.RequiredVersion))
	}
	if !m.prompter.Confirm(summary.String() + "Install?") {
		return fmt.Errorf("dependency installation declined, aborting")
	}

	for _, dep := range missing {
		if common.DetectCycle(installing, dep.PackageID) {
			continue
		}
		installing[dep.PackageID] = true

		// Determine type from package ID by trying to pull and reading manifest
		depScope, depName, err := common.ParsePackageID(dep.PackageID)
		if err != nil {
			return err
		}

		// Try assistant type first, then mcp
		var installed bool
		for _, regType := range []string{common.TypeDirAssistants, common.TypeDirMCPs} {
			zipData, digest, err := m.client.Pull(regType, "@"+depScope, depName, "latest")
			if err != nil {
				continue
			}
			manifest, err := common.ReadManifest(zipData)
			if err != nil {
				continue
			}

			pkgType := manifest.Type
			destDir := common.PackageDir(pkgType, depScope, depName, m.appRoot)
			if err := os.MkdirAll(destDir, 0755); err != nil {
				return err
			}
			if _, err := common.UnpackTo(zipData, destDir); err != nil {
				return fmt.Errorf("unpack dependency %s: %w", dep.PackageID, err)
			}

			relDir := common.PackageDirRel(pkgType, depScope, depName)
			files, err := common.HashDir(destDir, relDir)
			if err != nil {
				return err
			}

			info := common.PackageInfo{
				Type:         pkgType,
				Version:      manifest.Version,
				Integrity:    digest,
				Dependencies: manifest.Dependencies,
				Files:        files,
			}
			lf.SetPackage(dep.PackageID, info)

			// Recursively install this dep's dependencies
			if len(manifest.Dependencies) > 0 {
				if err := m.installDependencies(manifest.Dependencies, lf, dep.PackageID, installing); err != nil {
					return err
				}
			}

			fmt.Printf("  ✓ Dependency %s@%s installed\n", dep.PackageID, manifest.Version)
			installed = true
			break
		}

		if !installed {
			return fmt.Errorf("failed to install dependency %s: not found in registry", dep.PackageID)
		}
	}

	return nil
}
