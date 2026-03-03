package mcp

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/yaoapp/yao/registry/manager/common"
)

// PushOptions configures the Push operation.
type PushOptions struct {
	Version string
	Force   bool // delete existing version before push
}

// Push packages and uploads an MCP to the registry.
// Per DESIGN-MCP.md:
//  1. ID → path
//  2. Validate .mcp.yao exists
//  3. Derive scope/name, reject @local
//  4. Validate scripts are in scripts/{scope}/
//  5. Pack MCP dir + collect scripts from project root
//  6. Generate pkg.yao
//  7. Push to registry
func (m *Manager) Push(yaoID string, opts PushOptions) error {
	if opts.Version == "" {
		return fmt.Errorf("--version is required for push")
	}

	scope, name, err := common.IDFromYaoID(yaoID)
	if err != nil {
		return fmt.Errorf("invalid MCP ID %q: %w", yaoID, err)
	}

	if common.IsLocalScope(scope) {
		return fmt.Errorf("cannot push @local packages. Fork to your own scope first")
	}

	mcpDir := common.PackageDir(common.TypeMCP, scope, name, m.appRoot)

	// Find .mcp.yao files
	mcpYaoFiles, err := FindMCPYaoFiles(mcpDir)
	if err != nil || len(mcpYaoFiles) == 0 {
		return fmt.Errorf(".mcp.yao not found in %s", mcpDir)
	}

	// Validate script scope and collect scripts
	allScripts := map[string]string{}
	for _, mcpFile := range mcpYaoFiles {
		if err := ValidateScriptScope(mcpFile, scope, m.appRoot); err != nil {
			return err
		}
		scripts, err := CollectScripts(mcpFile, m.appRoot)
		if err != nil {
			return err
		}
		for k, v := range scripts {
			allScripts[k] = v
		}
	}

	manifest := &common.PkgManifest{
		Type:    common.TypeMCP,
		Scope:   "@" + scope,
		Name:    name,
		Version: opts.Version,
	}

	zipData, err := common.PackDir(mcpDir, manifest, allScripts)
	if err != nil {
		return fmt.Errorf("pack: %w", err)
	}

	regType := common.TypeToRegistryType(common.TypeMCP)

	if opts.Force {
		m.client.DeleteVersion(regType, "@"+scope, name, opts.Version)
	}

	result, err := m.client.Push(regType, "@"+scope, name, opts.Version, zipData)
	if err != nil {
		return fmt.Errorf("push: %w", err)
	}

	fmt.Printf("✓ Pushed %s@%s (digest: %s)\n", common.FormatPackageID(scope, name), result.Version, result.Digest)

	// Report packed scripts
	if len(allScripts) > 0 {
		fmt.Printf("  Scripts packed:\n")
		for scriptPath := range allScripts {
			fmt.Printf("    %s\n", scriptPath)
		}
	}

	return nil
}

// pushValidateMCPDir checks that the MCP directory structure is valid for push.
func pushValidateMCPDir(mcpDir string) error {
	if _, err := os.Stat(mcpDir); err != nil {
		return fmt.Errorf("MCP directory %s not found", mcpDir)
	}

	mcpFiles, err := FindMCPYaoFiles(mcpDir)
	if err != nil || len(mcpFiles) == 0 {
		return fmt.Errorf("no .mcp.yao files found in %s", mcpDir)
	}

	return nil
}

// listDirFiles lists all files under a directory relative to root.
func listDirFiles(dir string) ([]string, error) {
	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		files = append(files, filepath.ToSlash(rel))
		return nil
	})
	return files, err
}
