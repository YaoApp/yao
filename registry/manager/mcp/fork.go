package mcp

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/yaoapp/yao/registry/manager/common"
)

// ForkOptions configures the Fork operation.
type ForkOptions struct {
	TargetScope string
}

// Fork copies an MCP to a new scope with process reference rewriting.
// Per DESIGN-MCP.md Fork:
//  1. Copy mcps/{scope}/{name}/ → mcps/{target}/{name}/
//  2. Copy scripts precisely based on registry.yao files record
//  3. Rewrite process references in .mcp.yao (scripts.old. → scripts.new.)
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

	// Check target directory
	targetDir := common.PackageDir(common.TypeMCP, targetScope, name, m.appRoot)
	if _, err := os.Stat(targetDir); err == nil {
		return fmt.Errorf("target directory %s already exists", targetDir)
	}

	sourceDir := common.PackageDir(common.TypeMCP, scope, name, m.appRoot)
	var existing common.PackageInfo
	var isLocal bool

	if pkg, ok := lf.GetPackage(pkgID); ok {
		existing = pkg
		isLocal = true
	}

	if isLocal {
		// Copy MCP directory
		if err := copyDir(sourceDir, targetDir); err != nil {
			return fmt.Errorf("copy MCP dir: %w", err)
		}

		// Copy scripts precisely based on registry.yao files record
		scriptFiles := ScriptPathsFromFiles(existing.Files)
		for scriptPath := range scriptFiles {
			// Rewrite script path: scripts/{oldScope}/ → scripts/{targetScope}/
			newScriptPath := rewriteScriptPath(scriptPath, scope, targetScope)
			srcAbs := filepath.Join(m.appRoot, scriptPath)
			dstAbs := filepath.Join(m.appRoot, newScriptPath)
			if err := copyFileTo(srcAbs, dstAbs); err != nil {
				return fmt.Errorf("copy script %s: %w", scriptPath, err)
			}
		}
	} else {
		// Pull from registry
		regType := common.TypeToRegistryType(common.TypeMCP)
		zipData, _, err := m.client.Pull(regType, "@"+scope, name, "latest")
		if err != nil {
			return fmt.Errorf("pull %s: %w", pkgID, err)
		}

		// Unpack to temp, then sort
		tempDir, err := os.MkdirTemp("", "yao-mcp-fork-*")
		if err != nil {
			return err
		}
		defer os.RemoveAll(tempDir)

		files, err := common.UnpackTo(zipData, tempDir)
		if err != nil {
			return err
		}

		for _, f := range files {
			srcPath := filepath.Join(tempDir, f)
			if strings.HasPrefix(f, "scripts/") {
				newPath := rewriteScriptPath(f, scope, targetScope)
				dstPath := filepath.Join(m.appRoot, newPath)
				if err := copyFileTo(srcPath, dstPath); err != nil {
					return err
				}
			} else {
				dstPath := filepath.Join(targetDir, f)
				if err := copyFileTo(srcPath, dstPath); err != nil {
					return err
				}
			}
		}
	}

	// Rewrite process references in all .mcp.yao files in the target directory
	mcpFiles, err := FindMCPYaoFiles(targetDir)
	if err != nil {
		return err
	}
	for _, mcpFile := range mcpFiles {
		data, err := os.ReadFile(mcpFile)
		if err != nil {
			return err
		}
		rewritten := RewriteProcessRefs(data, scope, targetScope)
		if err := os.WriteFile(mcpFile, rewritten, 0644); err != nil {
			return err
		}
	}

	// Compute file hashes for the forked package
	mcpRelDir := common.PackageDirRel(common.TypeMCP, targetScope, name)
	fileHashes, err := common.HashDir(targetDir, mcpRelDir)
	if err != nil {
		return err
	}

	// Also hash the forked scripts
	scriptsDir := filepath.Join(m.appRoot, "scripts", targetScope)
	if _, err := os.Stat(scriptsDir); err == nil {
		scriptHashes, err := common.HashDir(scriptsDir, "scripts/"+targetScope)
		if err != nil {
			return err
		}
		for k, v := range scriptHashes {
			fileHashes[k] = v
		}
	}

	info := common.PackageInfo{
		Type:       common.TypeMCP,
		Version:    "0.0.0",
		ForkedFrom: pkgID,
		Managed:    common.BoolPtr(false),
		Files:      fileHashes,
	}
	if isLocal {
		info.Version = existing.Version
	}

	lf.SetPackage(targetPkgID, info)
	if err := common.SaveLockfile(m.appRoot, lf); err != nil {
		return err
	}

	yaoID := targetScope + "." + name
	fmt.Printf("✓ Forked %s → %s (ID: %s)\n", pkgID, targetDir, yaoID)
	fmt.Printf("  Process references rewritten: scripts.%s.* → scripts.%s.*\n", scope, targetScope)
	return nil
}

// rewriteScriptPath changes "scripts/{oldScope}/..." to "scripts/{newScope}/..."
func rewriteScriptPath(path, oldScope, newScope string) string {
	old := "scripts/" + oldScope + "/"
	replacement := "scripts/" + newScope + "/"
	return strings.Replace(path, old, replacement, 1)
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
		return copyFileTo(path, target)
	})
}

func copyFileTo(src, dst string) error {
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
