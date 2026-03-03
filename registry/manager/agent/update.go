package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/yaoapp/yao/registry/manager/common"
)

// UpdateOptions configures the Update operation.
type UpdateOptions struct {
	Version string // target version or dist-tag, default "latest"
}

// Update performs a hash-based safe update per DESIGN.md Update Strategy:
//  1. Confirm installed and managed
//  2. Pull new version
//  3. Check required_by compatibility
//  4. Per-file hash comparison: overwrite unmodified, skip modified (.new), add new, delete removed
//  5. Update registry.yao
//  6. Hot-reload
func (m *Manager) Update(pkgID string, opts UpdateOptions) error {
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

	existing, ok := lf.GetPackage(pkgID)
	if !ok {
		return fmt.Errorf("package %s is not installed", pkgID)
	}
	if !existing.IsManaged() {
		return fmt.Errorf("package %s is forked (from %s) and not managed by registry", pkgID, existing.ForkedFrom)
	}

	// Pull new version
	regType := common.TypeToRegistryType(common.TypeAssistant)
	zipData, digest, err := m.client.Pull(regType, "@"+scope, name, opts.Version)
	if err != nil {
		return fmt.Errorf("pull %s: %w", pkgID, err)
	}

	manifest, err := common.ReadManifest(zipData)
	if err != nil {
		return fmt.Errorf("read manifest: %w", err)
	}

	// Check required_by compatibility
	if len(existing.RequiredBy) > 0 {
		var warnings []string
		for _, depID := range existing.RequiredBy {
			depPkg, depOK := lf.GetPackage(depID)
			if !depOK {
				continue
			}
			if constraint, has := depPkg.Dependencies[pkgID]; has {
				if !common.VersionSatisfies(manifest.Version, constraint) {
					warnings = append(warnings, fmt.Sprintf("  %s requires %s ← ⚠ incompatible", depID, constraint))
				}
			}
		}
		if len(warnings) > 0 {
			msg := fmt.Sprintf("%s is depended on by:\n%s\nContinue update?", pkgID, strings.Join(warnings, "\n"))
			if !m.prompter.Confirm(msg) {
				return fmt.Errorf("update aborted by user")
			}
		}
	}

	destDir := common.PackageDir(common.TypeAssistant, scope, name, m.appRoot)
	relDir := common.PackageDirRel(common.TypeAssistant, scope, name)

	// Get list of new files from zip
	newFiles, err := common.ListZipFiles(zipData)
	if err != nil {
		return err
	}

	// Build a set of new file relative paths (with full relDir prefix)
	newFileSet := map[string]bool{}
	for _, f := range newFiles {
		newFileSet[relDir+"/"+f] = true
	}

	newHashes := map[string]string{}

	// Per-file comparison
	for _, f := range newFiles {
		fullRel := relDir + "/" + f
		localPath := filepath.Join(destDir, f)
		oldHash, wasTracked := existing.Files[fullRel]

		// Read new file content from zip
		newContent, err := common.ExtractFile(zipData, f)
		if err != nil {
			return err
		}
		newHash := common.HashBytes(newContent)

		if !wasTracked {
			// New file in new version → add
			if err := writeFile(localPath, newContent); err != nil {
				return err
			}
			fmt.Printf("+ %s — new file, added\n", f)
			newHashes[fullRel] = newHash
			continue
		}

		// Check if local file was modified
		localHash, err := common.HashFile(localPath)
		if err != nil {
			// File might have been deleted locally, just write it
			if err := writeFile(localPath, newContent); err != nil {
				return err
			}
			fmt.Printf("✓ %s — restored (was missing locally)\n", f)
			newHashes[fullRel] = newHash
			continue
		}

		if localHash == oldHash {
			// Unmodified → overwrite
			if err := writeFile(localPath, newContent); err != nil {
				return err
			}
			fmt.Printf("✓ %s — unmodified, updated\n", f)
			newHashes[fullRel] = newHash
		} else {
			// Locally modified → skip, save new version as .new
			newPath := localPath + ".new"
			if err := writeFile(newPath, newContent); err != nil {
				return err
			}
			fmt.Printf("✗ %s — locally modified, skipped (new version → %s.new)\n", f, f)
			// Update hash to new version's hash per DESIGN.md
			newHashes[fullRel] = newHash
		}
	}

	// Handle deleted files (in old but not in new)
	for oldFile, oldHash := range existing.Files {
		if newFileSet[oldFile] {
			continue
		}
		localPath := filepath.Join(m.appRoot, filepath.FromSlash(oldFile))
		localHash, err := common.HashFile(localPath)
		if err != nil {
			// Already gone
			continue
		}
		if localHash == oldHash {
			// Unmodified → delete
			os.Remove(localPath)
			fmt.Printf("- %s — removed (deleted in new version)\n", filepath.Base(oldFile))
		} else {
			fmt.Printf("⚠ %s — locally modified, kept (deleted in new version)\n", filepath.Base(oldFile))
			// Keep it, but don't track it anymore
		}
	}

	// Check new version dependencies
	if len(manifest.Dependencies) > 0 {
		if err := m.installDependencies(manifest.Dependencies, lf, pkgID, map[string]bool{pkgID: true}); err != nil {
			return err
		}
	}

	// Update lockfile
	existing.Version = manifest.Version
	existing.Integrity = digest
	existing.Dependencies = manifest.Dependencies
	existing.Files = newHashes
	lf.SetPackage(pkgID, existing)

	for depID := range manifest.Dependencies {
		lf.AddRequiredBy(depID, pkgID)
	}

	if err := common.SaveLockfile(m.appRoot, lf); err != nil {
		return err
	}

	fmt.Printf("✓ Updated %s to %s\n", pkgID, manifest.Version)
	return nil
}

func writeFile(path string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, content, 0644)
}
