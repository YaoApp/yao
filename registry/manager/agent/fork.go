package agent

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/yaoapp/yao/registry/manager/common"
)

// ForkOptions configures the Fork operation.
type ForkOptions struct {
	TargetScope string // target scope, defaults to lockfile's default scope (usually "local")
}

// Fork copies an assistant to a new scope for local modification.
// Flow per DESIGN.md Fork:
//  1. If locally installed → copy directory
//  2. If not installed → pull from registry
//  3. Place in target scope directory
//  4. Write registry.yao with managed:false
//  5. Hot-reload
func (m *Manager) Fork(pkgID string, opts ForkOptions) error {
	scope, name, err := common.ParsePackageID(pkgID)
	if err != nil {
		return err
	}

	lf, err := common.LoadLockfile(m.appRoot)
	if err != nil {
		return err
	}

	targetScope := opts.TargetScope
	if targetScope == "" {
		targetScope = lf.DefaultScope()
	}

	targetPkgID := common.FormatPackageID(targetScope, name)

	// Check if target already exists
	targetDir := common.PackageDir(common.TypeAssistant, targetScope, name, m.appRoot)
	if _, err := os.Stat(targetDir); err == nil {
		return fmt.Errorf("target directory %s already exists", targetDir)
	}

	sourceDir := common.PackageDir(common.TypeAssistant, scope, name, m.appRoot)

	if _, ok := lf.GetPackage(pkgID); ok {
		// Local copy
		if err := copyDir(sourceDir, targetDir); err != nil {
			return fmt.Errorf("copy: %w", err)
		}
	} else {
		// Pull from registry
		regType := common.TypeToRegistryType(common.TypeAssistant)
		zipData, _, err := m.client.Pull(regType, "@"+scope, name, "latest")
		if err != nil {
			return fmt.Errorf("pull %s: %w", pkgID, err)
		}
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			return err
		}
		if _, err := common.UnpackTo(zipData, targetDir); err != nil {
			return fmt.Errorf("unpack: %w", err)
		}
	}

	// Compute file hashes
	relDir := common.PackageDirRel(common.TypeAssistant, targetScope, name)
	files, err := common.HashDir(targetDir, relDir)
	if err != nil {
		return fmt.Errorf("hash files: %w", err)
	}

	// Write lockfile entry
	info := common.PackageInfo{
		Type:       common.TypeAssistant,
		Version:    "0.0.0",
		ForkedFrom: pkgID,
		Managed:    common.BoolPtr(false),
		Files:      files,
	}

	// Try to get version from source
	if existing, ok := lf.GetPackage(pkgID); ok {
		info.Version = existing.Version
	}

	lf.SetPackage(targetPkgID, info)
	if err := common.SaveLockfile(m.appRoot, lf); err != nil {
		return err
	}

	yaoID := targetScope + "." + name
	fmt.Printf("✓ Forked %s → %s (ID: %s)\n", pkgID, targetDir, yaoID)
	fmt.Printf("  Internal references (mcp.servers, uses) still point to original scope.\n")
	fmt.Printf("  Edit package.yao if you need to change them.\n")
	return nil
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)

		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}

		return copyFile(path, target)
	})
}

func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
