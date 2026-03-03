package mcp

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	goujson "github.com/yaoapp/gou/json"
)

// ProcessRef represents a process reference extracted from a .mcp.yao file.
type ProcessRef struct {
	ToolName    string // e.g. "search"
	ProcessPath string // e.g. "scripts.yao.rag.Search"
	ScriptPath  string // resolved filesystem path: "scripts/yao/rag.ts"
	Scope       string // extracted scope: "yao"
}

// mcpDSL is a minimal representation of .mcp.yao for process reference extraction.
type mcpDSL struct {
	Transport string            `json:"transport"`
	Tools     map[string]string `json:"tools,omitempty"`
}

// ExtractProcessRefs parses a .mcp.yao file and extracts all "scripts.*" process references.
func ExtractProcessRefs(mcpYaoPath string) ([]ProcessRef, error) {
	data, err := os.ReadFile(mcpYaoPath)
	if err != nil {
		return nil, err
	}
	return ExtractProcessRefsFromBytes(data)
}

// ExtractProcessRefsFromBytes extracts process refs from .mcp.yao content bytes.
// Supports JSONC format (// and /* */ comments) used by Yao DSL files.
func ExtractProcessRefsFromBytes(data []byte) ([]ProcessRef, error) {
	var dsl mcpDSL
	if err := goujson.ParseFile(".mcp.yao", data, &dsl); err != nil {
		return nil, fmt.Errorf("parse .mcp.yao: %w", err)
	}

	if dsl.Transport != "process" {
		return nil, nil
	}

	var refs []ProcessRef
	for toolName, processPath := range dsl.Tools {
		if !strings.HasPrefix(processPath, "scripts.") {
			continue
		}
		ref, err := parseProcessRef(toolName, processPath)
		if err != nil {
			continue
		}
		refs = append(refs, ref)
	}
	return refs, nil
}

// parseProcessRef parses "scripts.yao.rag.Search" into a ProcessRef.
// Convention: scripts.{scope}.{path...}.{Function}
// Script file: scripts/{scope}/{path_joined}.ts
func parseProcessRef(toolName, processPath string) (ProcessRef, error) {
	parts := strings.Split(processPath, ".")
	// At minimum: scripts.scope.file.Function = 4 parts
	if len(parts) < 4 {
		return ProcessRef{}, fmt.Errorf("process path %q too short", processPath)
	}

	scope := parts[1]
	// The middle parts (between scope and function name) form the script path
	scriptParts := parts[2 : len(parts)-1]
	scriptFile := strings.Join(scriptParts, "/") + ".ts"
	scriptPath := filepath.Join("scripts", scope, scriptFile)

	return ProcessRef{
		ToolName:    toolName,
		ProcessPath: processPath,
		ScriptPath:  filepath.ToSlash(scriptPath),
		Scope:       scope,
	}, nil
}

// RewriteProcessRefs rewrites all "scripts.{oldScope}." references to
// "scripts.{newScope}." in a .mcp.yao file content.
func RewriteProcessRefs(mcpContent []byte, oldScope, newScope string) []byte {
	old := "scripts." + oldScope + "."
	replacement := "scripts." + newScope + "."
	return []byte(strings.ReplaceAll(string(mcpContent), old, replacement))
}

// ValidateScriptScope checks that all process references in a .mcp.yao point to
// scripts in the expected scope directory. Returns an error if any violate the rule.
func ValidateScriptScope(mcpYaoPath, expectedScope, appRoot string) error {
	refs, err := ExtractProcessRefs(mcpYaoPath)
	if err != nil {
		return err
	}

	for _, ref := range refs {
		if ref.Scope != expectedScope {
			return fmt.Errorf(
				"MCP script scope mismatch: %s references scripts.%s.* but expected scripts.%s.*\n"+
					"  Scripts must be in scripts/%s/ to match MCP scope",
				mcpYaoPath, ref.Scope, expectedScope, expectedScope,
			)
		}

		// Verify script file exists
		scriptPath := filepath.Join(appRoot, ref.ScriptPath)
		if _, err := os.Stat(scriptPath); err != nil {
			return fmt.Errorf("script file %s referenced by %s not found", ref.ScriptPath, ref.ProcessPath)
		}
	}
	return nil
}

// CollectScripts gathers all script files referenced by a .mcp.yao file.
// Returns a map of relative path (under package/) → absolute path on disk.
func CollectScripts(mcpYaoPath, appRoot string) (map[string]string, error) {
	refs, err := ExtractProcessRefs(mcpYaoPath)
	if err != nil {
		return nil, err
	}

	scripts := map[string]string{}
	for _, ref := range refs {
		absPath := filepath.Join(appRoot, ref.ScriptPath)
		if _, err := os.Stat(absPath); err != nil {
			return nil, fmt.Errorf("script %s not found: %w", ref.ScriptPath, err)
		}
		scripts[ref.ScriptPath] = absPath
	}

	return scripts, nil
}

// FindMCPYaoFiles finds all .mcp.yao files in an MCP package directory.
func FindMCPYaoFiles(mcpDir string) ([]string, error) {
	var files []string
	err := filepath.Walk(mcpDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".mcp.yao") {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

// ScriptPathsFromFiles extracts script-related paths from a file hash map.
// Returns only entries starting with "scripts/".
func ScriptPathsFromFiles(files map[string]string) map[string]string {
	result := map[string]string{}
	for path, hash := range files {
		if strings.HasPrefix(path, "scripts/") {
			result[path] = hash
		}
	}
	return result
}

// processRefRegex matches "scripts.{scope}.{rest}" patterns.
var processRefRegex = regexp.MustCompile(`scripts\.([a-zA-Z0-9_-]+)\.`)

// ExtractScopeFromProcessRef extracts the scope from a process reference like "scripts.yao.rag.Search".
func ExtractScopeFromProcessRef(processPath string) string {
	matches := processRefRegex.FindStringSubmatch(processPath)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
}
