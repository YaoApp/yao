package mcp

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/yaoapp/yao/registry/manager/common"
)

// AddOptions configures the Add operation.
type AddOptions struct {
	Version string
	Force   bool
}

// Add installs an MCP package from the registry.
// Per DESIGN-MCP.md:
//  1. Check conflict
//  2. Pull from registry
//  3. Check dependencies (MCPs currently have none, but structure supports it)
//  4. Unpack .mcp.yao + mapping/ to mcps/{scope}/{name}/
//  5. Extract scripts/ to project root scripts/{scope}/
//  6. Check script conflicts
//  7. Write registry.yao (files include both MCP dir and scripts)
//  8. Hot-reload
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

	if existing, ok := lf.GetPackage(pkgID); ok && !opts.Force {
		return fmt.Errorf("package %s is already installed (version %s). Use --force to reinstall", pkgID, existing.Version)
	}

	destDir := common.PackageDir(common.TypeMCP, scope, name, m.appRoot)
	if _, err := os.Stat(destDir); err == nil {
		if _, ok := lf.GetPackage(pkgID); !ok {
			return fmt.Errorf("directory %s already exists but is not managed by registry. Please remove or relocate it first", destDir)
		}
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

	// Unpack everything to a temp dir first, then sort into MCP dir and scripts
	tempDir, err := os.MkdirTemp("", "yao-mcp-install-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	allFiles, err := common.UnpackTo(zipData, tempDir)
	if err != nil {
		return fmt.Errorf("unpack: %w", err)
	}

	fileHashes := map[string]string{}
	mcpRelDir := common.PackageDirRel(common.TypeMCP, scope, name)

	for _, f := range allFiles {
		srcPath := filepath.Join(tempDir, f)

		if strings.HasPrefix(f, "scripts/") {
			// Script file → project root scripts/
			destPath := filepath.Join(m.appRoot, f)

			// Check script conflict: exists but not in registry.yao
			if _, err := os.Stat(destPath); err == nil {
				if !isScriptTracked(lf, f) {
					return fmt.Errorf("script file %s already exists and is not managed by registry. Please remove or relocate it first", f)
				}
			}

			if err := copyFileFromTo(srcPath, destPath); err != nil {
				return err
			}
			hash, _ := common.HashFile(destPath)
			fileHashes[f] = hash
		} else {
			// MCP file → mcps/{scope}/{name}/
			destPath := filepath.Join(destDir, f)
			if err := copyFileFromTo(srcPath, destPath); err != nil {
				return err
			}
			relPath := mcpRelDir + "/" + f
			hash, _ := common.HashFile(destPath)
			fileHashes[relPath] = hash
		}
	}

	info := common.PackageInfo{
		Type:         common.TypeMCP,
		Version:      manifest.Version,
		Integrity:    digest,
		Dependencies: manifest.Dependencies,
		Files:        fileHashes,
	}
	lf.SetPackage(pkgID, info)

	for depID := range manifest.Dependencies {
		lf.AddRequiredBy(depID, pkgID)
	}

	if err := common.SaveLockfile(m.appRoot, lf); err != nil {
		return err
	}

	fmt.Printf("✓ Installed %s@%s → %s\n", pkgID, manifest.Version, destDir)
	return nil
}

// isScriptTracked checks if a script path is tracked by any package in the lockfile.
func isScriptTracked(lf *common.RegistryYao, scriptPath string) bool {
	for _, pkg := range lf.Packages {
		if _, ok := pkg.Files[scriptPath]; ok {
			return true
		}
	}
	return false
}

func copyFileFromTo(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}
