package mcp

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/yaoapp/yao/registry/manager/common"
)

// UpdateOptions configures the Update operation.
type UpdateOptions struct {
	Version string
}

// Update performs a hash-based safe update for an MCP package.
// Same strategy as agent update but also handles scripts under project root.
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

	regType := common.TypeToRegistryType(common.TypeMCP)
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
					warnings = append(warnings, fmt.Sprintf("  %s requires %s ← incompatible", depID, constraint))
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

	// Unpack new version to temp
	tempDir, err := os.MkdirTemp("", "yao-mcp-update-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	newFileList, err := common.UnpackTo(zipData, tempDir)
	if err != nil {
		return err
	}

	mcpRelDir := common.PackageDirRel(common.TypeMCP, scope, name)
	destDir := common.PackageDir(common.TypeMCP, scope, name, m.appRoot)

	// Build new file set with full relative paths
	newFileMap := map[string]string{} // fullRelPath → temp file path
	for _, f := range newFileList {
		if strings.HasPrefix(f, "scripts/") {
			newFileMap[f] = filepath.Join(tempDir, f)
		} else {
			fullRel := mcpRelDir + "/" + f
			newFileMap[fullRel] = filepath.Join(tempDir, f)
		}
	}

	newHashes := map[string]string{}

	// Process each new file
	for fullRel, tempPath := range newFileMap {
		newContent, err := os.ReadFile(tempPath)
		if err != nil {
			return err
		}
		newHash := common.HashBytes(newContent)

		// Determine local path
		var localPath string
		if strings.HasPrefix(fullRel, "scripts/") {
			localPath = filepath.Join(m.appRoot, fullRel)
		} else {
			relInMCP := strings.TrimPrefix(fullRel, mcpRelDir+"/")
			localPath = filepath.Join(destDir, relInMCP)
		}

		oldHash, wasTracked := existing.Files[fullRel]

		if !wasTracked {
			if err := writeFileTo(localPath, newContent); err != nil {
				return err
			}
			fmt.Printf("+ %s — new file, added\n", filepath.Base(fullRel))
			newHashes[fullRel] = newHash
			continue
		}

		localHash, err := common.HashFile(localPath)
		if err != nil {
			if err := writeFileTo(localPath, newContent); err != nil {
				return err
			}
			fmt.Printf("✓ %s — restored (was missing locally)\n", filepath.Base(fullRel))
			newHashes[fullRel] = newHash
			continue
		}

		if localHash == oldHash {
			if err := writeFileTo(localPath, newContent); err != nil {
				return err
			}
			fmt.Printf("✓ %s — unmodified, updated\n", filepath.Base(fullRel))
			newHashes[fullRel] = newHash
		} else {
			newPath := localPath + ".new"
			if err := writeFileTo(newPath, newContent); err != nil {
				return err
			}
			fmt.Printf("✗ %s — locally modified, skipped (new version → .new)\n", filepath.Base(fullRel))
			newHashes[fullRel] = newHash
		}
	}

	// Handle deleted files
	for oldFile, oldHash := range existing.Files {
		if _, inNew := newFileMap[oldFile]; inNew {
			continue
		}
		var localPath string
		if strings.HasPrefix(oldFile, "scripts/") {
			localPath = filepath.Join(m.appRoot, oldFile)
		} else {
			relInMCP := strings.TrimPrefix(oldFile, mcpRelDir+"/")
			localPath = filepath.Join(destDir, relInMCP)
		}
		localHash, err := common.HashFile(localPath)
		if err != nil {
			continue
		}
		if localHash == oldHash {
			os.Remove(localPath)
			fmt.Printf("- %s — removed\n", filepath.Base(oldFile))
		} else {
			fmt.Printf("⚠ %s — locally modified, kept\n", filepath.Base(oldFile))
		}
	}

	existing.Version = manifest.Version
	existing.Integrity = digest
	existing.Dependencies = manifest.Dependencies
	existing.Files = newHashes
	lf.SetPackage(pkgID, existing)

	if err := common.SaveLockfile(m.appRoot, lf); err != nil {
		return err
	}

	fmt.Printf("✓ Updated %s to %s\n", pkgID, manifest.Version)
	return nil
}

func writeFileTo(path string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, content, 0644)
}
