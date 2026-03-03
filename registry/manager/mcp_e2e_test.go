package manager_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yaoapp/yao/registry/manager/common"
	mcpmgr "github.com/yaoapp/yao/registry/manager/mcp"
)

// =============================================================================
// Process MCP: Full lifecycle — Push → Add → Update → Fork
// =============================================================================

func TestE2EMCP_ProcessLifecycle(t *testing.T) {
	c := authClient()
	devApp := appRoot(t)

	defer cleanupPkg(c, "mcps", "@"+testScope, "registry-mcp", "1.0.0")
	defer cleanupPkg(c, "mcps", "@"+testScope, "registry-mcp", "2.0.0")

	// Phase 1: Push v1 from yao-dev-app
	pushMgr := mcpmgr.New(c, devApp, &common.AutoConfirmPrompter{})
	if err := pushMgr.Push(testScope+".registry-mcp", mcpmgr.PushOptions{Version: "1.0.0"}); err != nil {
		t.Fatalf("Push MCP v1: %v", err)
	}

	packument, err := c.GetPackument("mcps", "@"+testScope, "registry-mcp")
	if err != nil {
		t.Fatalf("GetPackument: %v", err)
	}
	if packument.DistTags["latest"] != "1.0.0" {
		t.Errorf("expected latest=1.0.0, got %s", packument.DistTags["latest"])
	}

	// Phase 2: Add to a fresh app
	installApp := t.TempDir()
	installMgr := mcpmgr.New(c, installApp, &common.AutoConfirmPrompter{})

	if err := installMgr.Add("@"+testScope+"/registry-mcp", mcpmgr.AddOptions{}); err != nil {
		t.Fatalf("Add MCP: %v", err)
	}

	// Verify .mcp.yao on disk
	installedMCP := filepath.Join(installApp, "mcps", testScope, "registry-mcp", "server.mcp.yao")
	requireFileExists(t, installedMCP)
	requireFileContains(t, installedMCP, "scripts."+testScope+".registry_mcp.Ping")

	// Verify scripts extracted to project root
	installedScript := filepath.Join(installApp, "scripts", testScope, "registry_mcp.ts")
	requireFileExists(t, installedScript)
	requireFileContains(t, installedScript, "pong")

	// Verify lockfile
	pkg := requireLockfileHas(t, installApp, "@"+testScope+"/registry-mcp")
	if pkg.Version != "1.0.0" {
		t.Errorf("lockfile version: want 1.0.0, got %s", pkg.Version)
	}
	if pkg.Type != common.TypeMCP {
		t.Errorf("lockfile type: want mcp, got %s", pkg.Type)
	}
	if pkg.Integrity == "" {
		t.Error("expected integrity digest in lockfile")
	}

	hasScript, hasMCP := false, false
	for path := range pkg.Files {
		if strings.HasPrefix(path, "scripts/") {
			hasScript = true
		}
		if strings.HasPrefix(path, "mcps/") {
			hasMCP = true
		}
	}
	if !hasScript {
		t.Error("lockfile missing script file entries")
	}
	if !hasMCP {
		t.Error("lockfile missing MCP file entries")
	}

	// Phase 3: Push v2, then Update
	v2App := buildV2AgentApp(t)
	pushMgrV2 := mcpmgr.New(c, v2App, &common.AutoConfirmPrompter{})
	if err := pushMgrV2.Push(testScope+".registry-mcp", mcpmgr.PushOptions{Version: "2.0.0"}); err != nil {
		t.Fatalf("Push MCP v2: %v", err)
	}

	if err := installMgr.Update("@"+testScope+"/registry-mcp", mcpmgr.UpdateOptions{Version: "2.0.0"}); err != nil {
		t.Fatalf("Update MCP to v2: %v", err)
	}

	pkg = requireLockfileHas(t, installApp, "@"+testScope+"/registry-mcp")
	if pkg.Version != "2.0.0" {
		t.Errorf("expected v2.0.0 after update, got %s", pkg.Version)
	}
	requireFileContains(t, installedScript, "pong-v2")

	// Phase 4: Fork to @local
	if err := installMgr.Fork("@"+testScope+"/registry-mcp", mcpmgr.ForkOptions{TargetScope: "local"}); err != nil {
		t.Fatalf("Fork MCP: %v", err)
	}

	forkedDir := filepath.Join(installApp, "mcps", "local", "registry-mcp")
	requireFileExists(t, forkedDir)

	forkedMCPPath := filepath.Join(forkedDir, "server.mcp.yao")
	requireFileContains(t, forkedMCPPath, "scripts.local.registry_mcp.Ping")
	requireFileNotContains(t, forkedMCPPath, "scripts."+testScope+".")

	forkedScript := filepath.Join(installApp, "scripts", "local", "registry_mcp.ts")
	requireFileExists(t, forkedScript)

	forkedPkg := requireLockfileHas(t, installApp, "@local/registry-mcp")
	if forkedPkg.ForkedFrom != "@"+testScope+"/registry-mcp" {
		t.Errorf("expected forked_from=@%s/registry-mcp, got %s", testScope, forkedPkg.ForkedFrom)
	}
	if forkedPkg.IsManaged() {
		t.Error("forked package should not be managed")
	}
}

// =============================================================================
// SSE MCP: Push → Add → Fork (no scripts, no process refs)
// =============================================================================

func TestE2EMCP_SSELifecycle(t *testing.T) {
	c := authClient()
	devApp := appRoot(t)

	defer cleanupPkg(c, "mcps", "@"+testScope, "sse-proxy", "1.0.0")

	pushMgr := mcpmgr.New(c, devApp, &common.AutoConfirmPrompter{})
	if err := pushMgr.Push(testScope+".sse-proxy", mcpmgr.PushOptions{Version: "1.0.0"}); err != nil {
		t.Fatalf("Push SSE MCP: %v", err)
	}

	// Add to fresh app
	installApp := t.TempDir()
	installMgr := mcpmgr.New(c, installApp, &common.AutoConfirmPrompter{})

	if err := installMgr.Add("@"+testScope+"/sse-proxy", mcpmgr.AddOptions{}); err != nil {
		t.Fatalf("Add SSE MCP: %v", err)
	}

	// Verify .mcp.yao on disk
	installedMCP := filepath.Join(installApp, "mcps", testScope, "sse-proxy", "server.mcp.yao")
	requireFileExists(t, installedMCP)
	requireFileContains(t, installedMCP, `"transport": "sse"`)
	requireFileContains(t, installedMCP, "mcp.example.com")

	// No scripts should be extracted for SSE
	scriptsDir := filepath.Join(installApp, "scripts")
	if _, err := os.Stat(scriptsDir); err == nil {
		entries, _ := os.ReadDir(scriptsDir)
		if len(entries) > 0 {
			t.Errorf("SSE MCP should not extract any scripts, found entries under scripts/")
		}
	}

	// Lockfile
	pkg := requireLockfileHas(t, installApp, "@"+testScope+"/sse-proxy")
	if pkg.Version != "1.0.0" {
		t.Errorf("want version 1.0.0, got %s", pkg.Version)
	}
	if pkg.Type != common.TypeMCP {
		t.Errorf("want type mcp, got %s", pkg.Type)
	}

	// No script files tracked in lockfile
	for path := range pkg.Files {
		if strings.HasPrefix(path, "scripts/") {
			t.Errorf("SSE MCP lockfile should not track scripts, found: %s", path)
		}
	}

	// Fork to @local — no process ref rewriting needed
	if err := installMgr.Fork("@"+testScope+"/sse-proxy", mcpmgr.ForkOptions{TargetScope: "local"}); err != nil {
		t.Fatalf("Fork SSE MCP: %v", err)
	}

	forkedMCP := filepath.Join(installApp, "mcps", "local", "sse-proxy", "server.mcp.yao")
	requireFileExists(t, forkedMCP)
	requireFileContains(t, forkedMCP, `"transport": "sse"`)

	forkedPkg := requireLockfileHas(t, installApp, "@local/sse-proxy")
	if forkedPkg.ForkedFrom != "@"+testScope+"/sse-proxy" {
		t.Errorf("expected forked_from=@%s/sse-proxy, got %s", testScope, forkedPkg.ForkedFrom)
	}
}

// =============================================================================
// Multi-script MCP: Push → Add → verify all scripts unpacked
// =============================================================================

func TestE2EMCP_MultiScriptPack(t *testing.T) {
	c := authClient()
	devApp := appRoot(t)

	defer cleanupPkg(c, "mcps", "@"+testScope, "data-tools", "1.0.0")

	pushMgr := mcpmgr.New(c, devApp, &common.AutoConfirmPrompter{})
	if err := pushMgr.Push(testScope+".data-tools", mcpmgr.PushOptions{Version: "1.0.0"}); err != nil {
		t.Fatalf("Push data-tools: %v", err)
	}

	installApp := t.TempDir()
	installMgr := mcpmgr.New(c, installApp, &common.AutoConfirmPrompter{})
	if err := installMgr.Add("@"+testScope+"/data-tools", mcpmgr.AddOptions{}); err != nil {
		t.Fatalf("Add data-tools: %v", err)
	}

	// Both script files should be extracted
	requireFileExists(t, filepath.Join(installApp, "scripts", testScope, "data_tools.ts"))
	requireFileExists(t, filepath.Join(installApp, "scripts", testScope, "data_utils.ts"))
	requireFileContains(t, filepath.Join(installApp, "scripts", testScope, "data_tools.ts"), "Aggregate")
	requireFileContains(t, filepath.Join(installApp, "scripts", testScope, "data_utils.ts"), "FormatCSV")

	// MCP definition on disk
	requireFileExists(t, filepath.Join(installApp, "mcps", testScope, "data-tools", "server.mcp.yao"))
	requireFileContains(t, filepath.Join(installApp, "mcps", testScope, "data-tools", "server.mcp.yao"),
		"scripts."+testScope+".data_utils.FormatCSV")

	// Lockfile tracks both script files
	pkg := requireLockfileHas(t, installApp, "@"+testScope+"/data-tools")
	scriptCount := 0
	for path := range pkg.Files {
		if strings.HasPrefix(path, "scripts/") {
			scriptCount++
		}
	}
	if scriptCount < 2 {
		t.Errorf("expected at least 2 script files tracked in lockfile, got %d", scriptCount)
	}
}

// =============================================================================
// Multi-script MCP: Update with local modification
// =============================================================================

func TestE2EMCP_MultiScriptUpdate(t *testing.T) {
	c := authClient()
	devApp := appRoot(t)

	defer cleanupPkg(c, "mcps", "@"+testScope, "data-tools", "1.0.0")
	defer cleanupPkg(c, "mcps", "@"+testScope, "data-tools", "2.0.0")

	// Push v1
	pushMgr := mcpmgr.New(c, devApp, &common.AutoConfirmPrompter{})
	if err := pushMgr.Push(testScope+".data-tools", mcpmgr.PushOptions{Version: "1.0.0"}); err != nil {
		t.Fatalf("Push v1: %v", err)
	}

	// Install v1
	installApp := t.TempDir()
	installMgr := mcpmgr.New(c, installApp, &common.AutoConfirmPrompter{})
	if err := installMgr.Add("@"+testScope+"/data-tools", mcpmgr.AddOptions{}); err != nil {
		t.Fatalf("Add v1: %v", err)
	}

	// Locally modify one of the script files
	modifiedScript := filepath.Join(installApp, "scripts", testScope, "data_tools.ts")
	customContent := "// MY CUSTOM DATA TOOLS\nfunction Aggregate() { return 'custom'; }\n"
	os.WriteFile(modifiedScript, []byte(customContent), 0644)

	// Push v2
	v2App := buildV2MCPApp(t)
	pushMgrV2 := mcpmgr.New(c, v2App, &common.AutoConfirmPrompter{})
	if err := pushMgrV2.Push(testScope+".data-tools", mcpmgr.PushOptions{Version: "2.0.0"}); err != nil {
		t.Fatalf("Push v2: %v", err)
	}

	// Update to v2
	if err := installMgr.Update("@"+testScope+"/data-tools", mcpmgr.UpdateOptions{Version: "2.0.0"}); err != nil {
		t.Fatalf("Update to v2: %v", err)
	}

	// Modified script should be preserved, .new file created
	content, _ := os.ReadFile(modifiedScript)
	if !strings.Contains(string(content), "MY CUSTOM DATA TOOLS") {
		t.Error("local modification to data_tools.ts should be preserved")
	}
	requireFileExists(t, modifiedScript+".new")
	requireFileContains(t, modifiedScript+".new", "version: 2")

	// Unmodified script should be updated in-place
	utilsScript := filepath.Join(installApp, "scripts", testScope, "data_utils.ts")
	requireFileContains(t, utilsScript, "header")

	// Lockfile version should be 2.0.0
	pkg := requireLockfileHas(t, installApp, "@"+testScope+"/data-tools")
	if pkg.Version != "2.0.0" {
		t.Errorf("expected v2.0.0 after update, got %s", pkg.Version)
	}
}

// =============================================================================
// Push → Pull roundtrip (byte-for-byte content verification)
// =============================================================================

func TestE2EMCP_PushPullRoundtrip(t *testing.T) {
	c := authClient()
	devApp := appRoot(t)

	defer cleanupPkg(c, "mcps", "@"+testScope, "registry-mcp", "1.0.0")

	pushMgr := mcpmgr.New(c, devApp, &common.AutoConfirmPrompter{})
	if err := pushMgr.Push(testScope+".registry-mcp", mcpmgr.PushOptions{Version: "1.0.0"}); err != nil {
		t.Fatalf("Push: %v", err)
	}

	pullApp := t.TempDir()
	pullMgr := mcpmgr.New(c, pullApp, &common.AutoConfirmPrompter{})
	if err := pullMgr.Add("@"+testScope+"/registry-mcp", mcpmgr.AddOptions{}); err != nil {
		t.Fatalf("Add: %v", err)
	}

	origScript, _ := os.ReadFile(filepath.Join(devApp, "scripts", testScope, "registry_mcp.ts"))
	pulledScript, _ := os.ReadFile(filepath.Join(pullApp, "scripts", testScope, "registry_mcp.ts"))
	if string(origScript) != string(pulledScript) {
		t.Errorf("script mismatch.\nOriginal:\n%s\nPulled:\n%s", origScript, pulledScript)
	}

	origMCP, _ := os.ReadFile(filepath.Join(devApp, "mcps", testScope, "registry-mcp", "server.mcp.yao"))
	pulledMCP, _ := os.ReadFile(filepath.Join(pullApp, "mcps", testScope, "registry-mcp", "server.mcp.yao"))
	if string(origMCP) != string(pulledMCP) {
		t.Errorf("MCP mismatch.\nOriginal:\n%s\nPulled:\n%s", origMCP, pulledMCP)
	}
}

// =============================================================================
// Push rejects wrong script scope
// =============================================================================

func TestE2EMCP_PushWrongScriptScope(t *testing.T) {
	c := authClient()
	devApp := t.TempDir()

	mcpDir := filepath.Join(devApp, "mcps", testScope, "bad-mcp")
	mustMkdir(t, mcpDir)
	mustWriteFile(t, filepath.Join(mcpDir, "bad.mcp.yao"), `{
  "transport": "process",
  "tools": {
    "run": "scripts.other.bad.Run"
  }
}`)
	mustWriteFile(t, filepath.Join(devApp, "scripts", "other", "bad.ts"), "export function Run() {}")

	mgr := mcpmgr.New(c, devApp, &common.AutoConfirmPrompter{})
	err := mgr.Push(testScope+".bad-mcp", mcpmgr.PushOptions{Version: "1.0.0"})
	if err == nil {
		t.Fatal("expected push to be rejected due to script scope mismatch")
	}
	if !strings.Contains(err.Error(), "scope mismatch") {
		t.Errorf("expected scope mismatch error, got: %v", err)
	}
}

// =============================================================================
// Push rejects when referenced script file is missing
// =============================================================================

func TestE2EMCP_PushMissingScript(t *testing.T) {
	c := authClient()
	devApp := t.TempDir()

	mcpDir := filepath.Join(devApp, "mcps", testScope, "missing-script-mcp")
	mustMkdir(t, mcpDir)
	mustWriteFile(t, filepath.Join(mcpDir, "server.mcp.yao"), `{
  "transport": "process",
  "tools": {
    "run": "scripts.`+testScope+`.nonexistent.Run"
  }
}`)

	mgr := mcpmgr.New(c, devApp, &common.AutoConfirmPrompter{})
	err := mgr.Push(testScope+".missing-script-mcp", mcpmgr.PushOptions{Version: "1.0.0"})
	if err == nil {
		t.Fatal("expected push to fail when script file is missing")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

// =============================================================================
// Push requires --version
// =============================================================================

func TestE2EMCP_PushNoVersion(t *testing.T) {
	c := authClient()
	devApp := appRoot(t)

	mgr := mcpmgr.New(c, devApp, &common.AutoConfirmPrompter{})
	err := mgr.Push(testScope+".registry-mcp", mcpmgr.PushOptions{})
	if err == nil {
		t.Fatal("expected push to fail without --version")
	}
	if !strings.Contains(err.Error(), "version") {
		t.Errorf("expected version error, got: %v", err)
	}
}

// =============================================================================
// Add: directory conflict (exists but not managed)
// =============================================================================

func TestE2EMCP_AddDirectoryConflict(t *testing.T) {
	c := authClient()
	devApp := appRoot(t)

	defer cleanupPkg(c, "mcps", "@"+testScope, "registry-mcp", "1.0.0")

	pushMgr := mcpmgr.New(c, devApp, &common.AutoConfirmPrompter{})
	pushMgr.Push(testScope+".registry-mcp", mcpmgr.PushOptions{Version: "1.0.0"})

	installApp := t.TempDir()
	conflictDir := filepath.Join(installApp, "mcps", testScope, "registry-mcp")
	mustMkdir(t, conflictDir)
	mustWriteFile(t, filepath.Join(conflictDir, "my-custom.mcp.yao"), `{"transport":"sse"}`)

	installMgr := mcpmgr.New(c, installApp, &common.AutoConfirmPrompter{})
	err := installMgr.Add("@"+testScope+"/registry-mcp", mcpmgr.AddOptions{})
	if err == nil {
		t.Fatal("expected directory conflict error")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("expected 'already exists' error, got: %v", err)
	}
}

// =============================================================================
// Add: script conflict (scripts/ file exists but not tracked)
// =============================================================================

func TestE2EMCP_AddScriptConflict(t *testing.T) {
	c := authClient()
	devApp := appRoot(t)

	defer cleanupPkg(c, "mcps", "@"+testScope, "registry-mcp", "1.0.0")

	pushMgr := mcpmgr.New(c, devApp, &common.AutoConfirmPrompter{})
	pushMgr.Push(testScope+".registry-mcp", mcpmgr.PushOptions{Version: "1.0.0"})

	installApp := t.TempDir()
	mustWriteFile(t, filepath.Join(installApp, "scripts", testScope, "registry_mcp.ts"),
		"// my existing custom script — should block install\n")

	installMgr := mcpmgr.New(c, installApp, &common.AutoConfirmPrompter{})
	err := installMgr.Add("@"+testScope+"/registry-mcp", mcpmgr.AddOptions{})
	if err == nil {
		t.Fatal("expected script conflict error")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("expected 'already exists' error about script, got: %v", err)
	}
}

// =============================================================================
// Add: already installed — reject without --force
// =============================================================================

func TestE2EMCP_AddAlreadyInstalled(t *testing.T) {
	c := authClient()
	devApp := appRoot(t)

	defer cleanupPkg(c, "mcps", "@"+testScope, "registry-mcp", "1.0.0")

	pushMgr := mcpmgr.New(c, devApp, &common.AutoConfirmPrompter{})
	pushMgr.Push(testScope+".registry-mcp", mcpmgr.PushOptions{Version: "1.0.0"})

	installApp := t.TempDir()
	installMgr := mcpmgr.New(c, installApp, &common.AutoConfirmPrompter{})
	installMgr.Add("@"+testScope+"/registry-mcp", mcpmgr.AddOptions{})

	// Second add should fail
	err := installMgr.Add("@"+testScope+"/registry-mcp", mcpmgr.AddOptions{})
	if err == nil {
		t.Fatal("expected already-installed error on second add")
	}
	if !strings.Contains(err.Error(), "already installed") {
		t.Errorf("expected 'already installed' error, got: %v", err)
	}
}

// =============================================================================
// Add: --force reinstall
// =============================================================================

func TestE2EMCP_AddForceReinstall(t *testing.T) {
	c := authClient()
	devApp := appRoot(t)

	defer cleanupPkg(c, "mcps", "@"+testScope, "registry-mcp", "1.0.0")

	pushMgr := mcpmgr.New(c, devApp, &common.AutoConfirmPrompter{})
	pushMgr.Push(testScope+".registry-mcp", mcpmgr.PushOptions{Version: "1.0.0"})

	installApp := t.TempDir()
	installMgr := mcpmgr.New(c, installApp, &common.AutoConfirmPrompter{})
	installMgr.Add("@"+testScope+"/registry-mcp", mcpmgr.AddOptions{})

	// Force reinstall should succeed
	err := installMgr.Add("@"+testScope+"/registry-mcp", mcpmgr.AddOptions{Force: true})
	if err != nil {
		t.Fatalf("force reinstall should succeed: %v", err)
	}

	pkg := requireLockfileHas(t, installApp, "@"+testScope+"/registry-mcp")
	if pkg.Version != "1.0.0" {
		t.Errorf("expected 1.0.0 after force reinstall, got %s", pkg.Version)
	}
}

// =============================================================================
// Update of forked package is rejected
// =============================================================================

func TestE2EMCP_UpdateForkedRejected(t *testing.T) {
	c := authClient()
	devApp := appRoot(t)

	defer cleanupPkg(c, "mcps", "@"+testScope, "registry-mcp", "1.0.0")

	pushMgr := mcpmgr.New(c, devApp, &common.AutoConfirmPrompter{})
	pushMgr.Push(testScope+".registry-mcp", mcpmgr.PushOptions{Version: "1.0.0"})

	installApp := t.TempDir()
	installMgr := mcpmgr.New(c, installApp, &common.AutoConfirmPrompter{})
	installMgr.Add("@"+testScope+"/registry-mcp", mcpmgr.AddOptions{})
	installMgr.Fork("@"+testScope+"/registry-mcp", mcpmgr.ForkOptions{TargetScope: "local"})

	err := installMgr.Update("@local/registry-mcp", mcpmgr.UpdateOptions{Version: "2.0.0"})
	if err == nil {
		t.Fatal("expected update of forked package to be rejected")
	}
	if !strings.Contains(err.Error(), "forked") {
		t.Errorf("expected 'forked' in error, got: %v", err)
	}
}

// =============================================================================
// Fork: target already exists
// =============================================================================

func TestE2EMCP_ForkTargetExists(t *testing.T) {
	c := authClient()
	devApp := appRoot(t)

	defer cleanupPkg(c, "mcps", "@"+testScope, "registry-mcp", "1.0.0")

	pushMgr := mcpmgr.New(c, devApp, &common.AutoConfirmPrompter{})
	pushMgr.Push(testScope+".registry-mcp", mcpmgr.PushOptions{Version: "1.0.0"})

	installApp := t.TempDir()
	installMgr := mcpmgr.New(c, installApp, &common.AutoConfirmPrompter{})
	installMgr.Add("@"+testScope+"/registry-mcp", mcpmgr.AddOptions{})

	// Pre-create target directory
	targetDir := filepath.Join(installApp, "mcps", "local", "registry-mcp")
	mustMkdir(t, targetDir)

	err := installMgr.Fork("@"+testScope+"/registry-mcp", mcpmgr.ForkOptions{TargetScope: "local"})
	if err == nil {
		t.Fatal("expected fork to fail when target directory exists")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("expected 'already exists' error, got: %v", err)
	}
}

// =============================================================================
// Fork from registry: package not installed locally, pull then fork
// =============================================================================

func TestE2EMCP_ForkFromRegistry(t *testing.T) {
	c := authClient()
	devApp := appRoot(t)

	defer cleanupPkg(c, "mcps", "@"+testScope, "registry-mcp", "1.0.0")

	pushMgr := mcpmgr.New(c, devApp, &common.AutoConfirmPrompter{})
	pushMgr.Push(testScope+".registry-mcp", mcpmgr.PushOptions{Version: "1.0.0"})

	// Fresh app — nothing installed
	forkApp := t.TempDir()
	forkMgr := mcpmgr.New(c, forkApp, &common.AutoConfirmPrompter{})

	if err := forkMgr.Fork("@"+testScope+"/registry-mcp", mcpmgr.ForkOptions{TargetScope: "local"}); err != nil {
		t.Fatalf("Fork from registry: %v", err)
	}

	// Forked MCP on disk
	forkedMCP := filepath.Join(forkApp, "mcps", "local", "registry-mcp", "server.mcp.yao")
	requireFileExists(t, forkedMCP)
	requireFileContains(t, forkedMCP, "scripts.local.registry_mcp.Ping")

	// Forked scripts on disk
	requireFileExists(t, filepath.Join(forkApp, "scripts", "local", "registry_mcp.ts"))

	// Lockfile entry
	pkg := requireLockfileHas(t, forkApp, "@local/registry-mcp")
	if pkg.ForkedFrom != "@"+testScope+"/registry-mcp" {
		t.Errorf("expected forked_from, got %s", pkg.ForkedFrom)
	}
	if pkg.IsManaged() {
		t.Error("forked package should not be managed")
	}
}

// =============================================================================
// Fork to custom scope (not @local)
// =============================================================================

func TestE2EMCP_ForkToCustomScope(t *testing.T) {
	c := authClient()
	devApp := appRoot(t)

	defer cleanupPkg(c, "mcps", "@"+testScope, "registry-mcp", "1.0.0")

	pushMgr := mcpmgr.New(c, devApp, &common.AutoConfirmPrompter{})
	pushMgr.Push(testScope+".registry-mcp", mcpmgr.PushOptions{Version: "1.0.0"})

	installApp := t.TempDir()
	installMgr := mcpmgr.New(c, installApp, &common.AutoConfirmPrompter{})
	installMgr.Add("@"+testScope+"/registry-mcp", mcpmgr.AddOptions{})

	if err := installMgr.Fork("@"+testScope+"/registry-mcp", mcpmgr.ForkOptions{TargetScope: "mycompany"}); err != nil {
		t.Fatalf("Fork to custom scope: %v", err)
	}

	forkedMCP := filepath.Join(installApp, "mcps", "mycompany", "registry-mcp", "server.mcp.yao")
	requireFileExists(t, forkedMCP)
	requireFileContains(t, forkedMCP, "scripts.mycompany.registry_mcp.Ping")

	forkedScript := filepath.Join(installApp, "scripts", "mycompany", "registry_mcp.ts")
	requireFileExists(t, forkedScript)

	pkg := requireLockfileHas(t, installApp, "@mycompany/registry-mcp")
	if pkg.ForkedFrom != "@"+testScope+"/registry-mcp" {
		t.Errorf("unexpected forked_from: %s", pkg.ForkedFrom)
	}
}
