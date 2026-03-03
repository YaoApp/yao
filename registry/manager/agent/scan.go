package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	goujson "github.com/yaoapp/gou/json"
	"github.com/yaoapp/yao/registry/manager/common"
)

// packageYao represents the assistant's package.yao DSL (subset of fields we care about).
type packageYao struct {
	MCP    *mcpConfig      `json:"mcp,omitempty"`
	Agents []string        `json:"agents,omitempty"`
	Uses   json.RawMessage `json:"uses,omitempty"`
}

type mcpConfig struct {
	Servers []mcpServerEntry `json:"servers,omitempty"`
}

type mcpServerEntry struct {
	ServerID string `json:"server_id,omitempty"`
}

// ScanDependencies scans an assistant directory's package.yao for external dependencies.
// It finds MCP dependencies from mcp.servers and returns them as "@scope/name" → "*" entries.
// Only MCPs with a scope directory (mcps/{scope}/) are included; top-level mcps/ are skipped.
func ScanDependencies(assistantDir, appRoot string) (map[string]string, error) {
	pkgPath := filepath.Join(assistantDir, "package.yao")
	data, err := os.ReadFile(pkgPath)
	if err != nil {
		return nil, fmt.Errorf("read package.yao: %w", err)
	}

	var pkg packageYao
	if err := goujson.ParseFile("package.yao", data, &pkg); err != nil {
		return nil, fmt.Errorf("parse package.yao: %w", err)
	}

	deps := map[string]string{}

	// Scan MCP servers
	if pkg.MCP != nil {
		for _, entry := range pkg.MCP.Servers {
			serverID := entry.ServerID
			if serverID == "" {
				continue
			}
			pkgID, err := resolveMCPDep(serverID, appRoot)
			if err != nil {
				continue
			}
			if pkgID != "" {
				deps[pkgID] = "*"
			}
		}
	}

	return deps, nil
}

// resolveMCPDep resolves an MCP server_id to a package ID if it lives under a scoped directory.
// Returns empty string for non-scoped (local) MCPs.
func resolveMCPDep(serverID, appRoot string) (string, error) {
	// server_id like "yao.rag-tools" → scope=yao, name=rag-tools
	// Check if mcps/yao/rag-tools/ exists (has scope directory)
	scope, name, err := common.IDFromYaoID(serverID)
	if err != nil {
		return "", nil
	}

	mcpDir := filepath.Join(appRoot, "mcps", scope, strings.ReplaceAll(name, ".", "/"))
	if _, err := os.Stat(mcpDir); err == nil {
		return common.FormatPackageID(scope, name), nil
	}

	// Also check for single-file MCP: mcps/{scope}/{name}.mcp.yao
	// This is less common but possible
	mcpFile := filepath.Join(appRoot, "mcps", scope, name+".mcp.yao")
	if _, err := os.Stat(mcpFile); err == nil {
		return common.FormatPackageID(scope, name), nil
	}

	return "", nil
}
