package manager_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	agentmgr "github.com/yaoapp/yao/registry/manager/agent"
	"github.com/yaoapp/yao/registry/manager/common"
	mcpmgr "github.com/yaoapp/yao/registry/manager/mcp"
)

// =============================================================================
// Agent with 1 MCP dep: Full lifecycle — Push → Add (auto dep) → Update → Fork
// =============================================================================

func TestE2EAgent_SingleDepLifecycle(t *testing.T) {
	c := authClient()
	devApp := appRoot(t)

	defer cleanupPkg(c, "mcps", "@"+testScope, "registry-mcp", "1.0.0")
	defer cleanupPkg(c, "assistants", "@"+testScope, "registry-agent", "1.0.0")
	defer cleanupPkg(c, "assistants", "@"+testScope, "registry-agent", "2.0.0")

	// Push MCP dependency first
	mcpMgr := mcpmgr.New(c, devApp, &common.AutoConfirmPrompter{})
	if err := mcpMgr.Push(testScope+".registry-mcp", mcpmgr.PushOptions{Version: "1.0.0"}); err != nil {
		t.Fatalf("Push MCP: %v", err)
	}

	// Verify .yaoignore fixtures exist in source before push
	srcAgent := filepath.Join(devApp, "assistants", testScope, "registry-agent")
	requireFileExists(t, filepath.Join(srcAgent, ".yaoignore"))
	requireFileExists(t, filepath.Join(srcAgent, "dev-notes.md"))
	requireFileExists(t, filepath.Join(srcAgent, "wireframe.sketch"))
	requireFileExists(t, filepath.Join(srcAgent, "debug", "trace.log"))

	// Push agent
	agentMgr := agentmgr.New(c, devApp, &common.AutoConfirmPrompter{})
	if err := agentMgr.Push(testScope+".registry-agent", agentmgr.PushOptions{Version: "1.0.0"}); err != nil {
		t.Fatalf("Push agent: %v", err)
	}

	packument, err := c.GetPackument("assistants", "@"+testScope, "registry-agent")
	if err != nil {
		t.Fatalf("GetPackument: %v", err)
	}
	if packument.DistTags["latest"] != "1.0.0" {
		t.Errorf("expected latest=1.0.0, got %s", packument.DistTags["latest"])
	}

	// Add to fresh app — MCP should auto-install
	installApp := t.TempDir()
	installAgent := agentmgr.New(c, installApp, &common.AutoConfirmPrompter{})

	if err := installAgent.Add("@"+testScope+"/registry-agent", agentmgr.AddOptions{}); err != nil {
		t.Fatalf("Add agent: %v", err)
	}

	// Agent on disk
	agentDir := filepath.Join(installApp, "assistants", testScope, "registry-agent")
	requireFileExists(t, filepath.Join(agentDir, "package.yao"))
	requireFileContains(t, filepath.Join(agentDir, "prompts.yml"), "registry E2E testing")

	// .yaoignore: excluded files must NOT appear in the installed package
	requireFileNotExists(t, filepath.Join(agentDir, ".yaoignore"))
	requireFileNotExists(t, filepath.Join(agentDir, "dev-notes.md"))
	requireFileNotExists(t, filepath.Join(agentDir, "wireframe.sketch"))
	requireFileNotExists(t, filepath.Join(agentDir, "debug", "trace.log"))

	// Lockfile: agent entry
	agentPkg := requireLockfileHas(t, installApp, "@"+testScope+"/registry-agent")
	if agentPkg.Version != "1.0.0" {
		t.Errorf("want version 1.0.0, got %s", agentPkg.Version)
	}
	if agentPkg.Type != common.TypeAssistant {
		t.Errorf("want type assistant, got %s", agentPkg.Type)
	}
	if len(agentPkg.Files) == 0 {
		t.Error("expected file hashes in lockfile")
	}

	// MCP dependency auto-installed
	depPkg := requireLockfileHas(t, installApp, "@"+testScope+"/registry-mcp")
	if depPkg.Version != "1.0.0" {
		t.Errorf("dependency version: want 1.0.0, got %s", depPkg.Version)
	}

	found := false
	for _, rb := range depPkg.RequiredBy {
		if rb == "@"+testScope+"/registry-agent" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected agent in MCP's required_by, got %v", depPkg.RequiredBy)
	}

	// Update: push v2, locally modify, then update
	v2App := buildV2AgentApp(t)
	pushAgentV2 := agentmgr.New(c, v2App, &common.AutoConfirmPrompter{})
	if err := pushAgentV2.Push(testScope+".registry-agent", agentmgr.PushOptions{Version: "2.0.0"}); err != nil {
		t.Fatalf("Push agent v2: %v", err)
	}

	customPrompt := "My custom prompt - DO NOT OVERWRITE."
	os.WriteFile(filepath.Join(agentDir, "prompts.yml"), []byte(customPrompt), 0644)

	if err := installAgent.Update("@"+testScope+"/registry-agent", agentmgr.UpdateOptions{Version: "2.0.0"}); err != nil {
		t.Fatalf("Update agent: %v", err)
	}

	agentPkg = requireLockfileHas(t, installApp, "@"+testScope+"/registry-agent")
	if agentPkg.Version != "2.0.0" {
		t.Errorf("expected v2.0.0, got %s", agentPkg.Version)
	}

	// Local modification preserved
	preservedData, _ := os.ReadFile(filepath.Join(agentDir, "prompts.yml"))
	if string(preservedData) != customPrompt {
		t.Errorf("local modification should be preserved, got: %s", preservedData)
	}
	requireFileExists(t, filepath.Join(agentDir, "prompts.yml.new"))
	requireFileContains(t, filepath.Join(agentDir, "prompts.yml.new"), "v2 registry test assistant")

	// New file added by v2
	requireFileExists(t, filepath.Join(agentDir, "tools.ts"))

	// Fork
	if err := installAgent.Fork("@"+testScope+"/registry-agent", agentmgr.ForkOptions{TargetScope: "local"}); err != nil {
		t.Fatalf("Fork agent: %v", err)
	}

	forkDir := filepath.Join(installApp, "assistants", "local", "registry-agent")
	requireFileExists(t, filepath.Join(forkDir, "package.yao"))

	forkedPkg := requireLockfileHas(t, installApp, "@local/registry-agent")
	if forkedPkg.ForkedFrom != "@"+testScope+"/registry-agent" {
		t.Errorf("expected forked_from=@%s/registry-agent, got %s", testScope, forkedPkg.ForkedFrom)
	}
	if forkedPkg.IsManaged() {
		t.Error("forked package should not be managed")
	}
}

// =============================================================================
// Agent with 2 MCP deps: both auto-installed
// =============================================================================

func TestE2EAgent_MultiDepLifecycle(t *testing.T) {
	c := authClient()
	devApp := appRoot(t)

	defer cleanupPkg(c, "mcps", "@"+testScope, "registry-mcp", "1.0.0")
	defer cleanupPkg(c, "mcps", "@"+testScope, "data-tools", "1.0.0")
	defer cleanupPkg(c, "assistants", "@"+testScope, "analytics", "1.0.0")

	// Push both MCP dependencies
	mcpMgr := mcpmgr.New(c, devApp, &common.AutoConfirmPrompter{})
	if err := mcpMgr.Push(testScope+".registry-mcp", mcpmgr.PushOptions{Version: "1.0.0"}); err != nil {
		t.Fatalf("Push registry-mcp: %v", err)
	}
	if err := mcpMgr.Push(testScope+".data-tools", mcpmgr.PushOptions{Version: "1.0.0"}); err != nil {
		t.Fatalf("Push data-tools: %v", err)
	}

	// Push analytics agent
	agentMgr := agentmgr.New(c, devApp, &common.AutoConfirmPrompter{})
	if err := agentMgr.Push(testScope+".analytics", agentmgr.PushOptions{Version: "1.0.0"}); err != nil {
		t.Fatalf("Push analytics: %v", err)
	}

	// Install to fresh app
	installApp := t.TempDir()
	installAgent := agentmgr.New(c, installApp, &common.AutoConfirmPrompter{})

	if err := installAgent.Add("@"+testScope+"/analytics", agentmgr.AddOptions{}); err != nil {
		t.Fatalf("Add analytics: %v", err)
	}

	// Agent on disk
	requireFileExists(t, filepath.Join(installApp, "assistants", testScope, "analytics", "package.yao"))
	requireFileContains(t, filepath.Join(installApp, "assistants", testScope, "analytics", "prompts.yml"),
		"analytics assistant")

	// Both MCP deps auto-installed
	requireLockfileHas(t, installApp, "@"+testScope+"/registry-mcp")
	requireLockfileHas(t, installApp, "@"+testScope+"/data-tools")

	// MCP files on disk
	requireFileExists(t, filepath.Join(installApp, "mcps", testScope, "registry-mcp", "server.mcp.yao"))
	requireFileExists(t, filepath.Join(installApp, "mcps", testScope, "data-tools", "server.mcp.yao"))

	// Scripts from both MCPs
	requireFileExists(t, filepath.Join(installApp, "scripts", testScope, "registry_mcp.ts"))
	requireFileExists(t, filepath.Join(installApp, "scripts", testScope, "data_tools.ts"))
	requireFileExists(t, filepath.Join(installApp, "scripts", testScope, "data_utils.ts"))

	// required_by on both MCPs should reference analytics
	lf, _ := common.LoadLockfile(installApp)
	for _, mcpID := range []string{"@" + testScope + "/registry-mcp", "@" + testScope + "/data-tools"} {
		mcpPkg, _ := lf.GetPackage(mcpID)
		found := false
		for _, rb := range mcpPkg.RequiredBy {
			if rb == "@"+testScope+"/analytics" {
				found = true
			}
		}
		if !found {
			t.Errorf("expected analytics in %s's required_by, got %v", mcpID, mcpPkg.RequiredBy)
		}
	}

	// Analytics lockfile entry
	agentPkg := requireLockfileHas(t, installApp, "@"+testScope+"/analytics")
	if agentPkg.Version != "1.0.0" {
		t.Errorf("want version 1.0.0, got %s", agentPkg.Version)
	}
}

// =============================================================================
// Agent with zero deps: standalone push/add/fork
// =============================================================================

func TestE2EAgent_NoDep(t *testing.T) {
	c := authClient()
	devApp := appRoot(t)

	defer cleanupPkg(c, "assistants", "@"+testScope, "simple-greeter", "1.0.0")

	agentMgr := agentmgr.New(c, devApp, &common.AutoConfirmPrompter{})
	if err := agentMgr.Push(testScope+".simple-greeter", agentmgr.PushOptions{Version: "1.0.0"}); err != nil {
		t.Fatalf("Push simple-greeter: %v", err)
	}

	installApp := t.TempDir()
	installAgent := agentmgr.New(c, installApp, &common.AutoConfirmPrompter{})

	if err := installAgent.Add("@"+testScope+"/simple-greeter", agentmgr.AddOptions{}); err != nil {
		t.Fatalf("Add simple-greeter: %v", err)
	}

	requireFileExists(t, filepath.Join(installApp, "assistants", testScope, "simple-greeter", "package.yao"))
	requireFileContains(t, filepath.Join(installApp, "assistants", testScope, "simple-greeter", "prompts.yml"),
		"friendly greeter")

	pkg := requireLockfileHas(t, installApp, "@"+testScope+"/simple-greeter")
	if pkg.Version != "1.0.0" {
		t.Errorf("want version 1.0.0, got %s", pkg.Version)
	}
	if pkg.Type != common.TypeAssistant {
		t.Errorf("want type assistant, got %s", pkg.Type)
	}

	// No MCP deps should be installed
	lf, _ := common.LoadLockfile(installApp)
	for id := range lf.Packages {
		if id != "@"+testScope+"/simple-greeter" {
			t.Errorf("unexpected package in lockfile: %s (standalone agent should have no deps)", id)
		}
	}

	// Fork
	if err := installAgent.Fork("@"+testScope+"/simple-greeter", agentmgr.ForkOptions{TargetScope: "local"}); err != nil {
		t.Fatalf("Fork simple-greeter: %v", err)
	}

	requireFileExists(t, filepath.Join(installApp, "assistants", "local", "simple-greeter", "package.yao"))

	forkedPkg := requireLockfileHas(t, installApp, "@local/simple-greeter")
	if forkedPkg.ForkedFrom != "@"+testScope+"/simple-greeter" {
		t.Errorf("expected forked_from, got %s", forkedPkg.ForkedFrom)
	}
}

// =============================================================================
// Add already installed — reject without --force
// =============================================================================

func TestE2EAgent_AddAlreadyInstalled(t *testing.T) {
	c := authClient()
	devApp := appRoot(t)

	defer cleanupPkg(c, "assistants", "@"+testScope, "simple-greeter", "1.0.0")

	agentMgr := agentmgr.New(c, devApp, &common.AutoConfirmPrompter{})
	agentMgr.Push(testScope+".simple-greeter", agentmgr.PushOptions{Version: "1.0.0"})

	installApp := t.TempDir()
	installAgent := agentmgr.New(c, installApp, &common.AutoConfirmPrompter{})
	installAgent.Add("@"+testScope+"/simple-greeter", agentmgr.AddOptions{})

	err := installAgent.Add("@"+testScope+"/simple-greeter", agentmgr.AddOptions{})
	if err == nil {
		t.Fatal("expected already-installed error")
	}
	if !strings.Contains(err.Error(), "already installed") {
		t.Errorf("expected 'already installed' error, got: %v", err)
	}
}

// =============================================================================
// Fork to custom scope
// =============================================================================

func TestE2EAgent_ForkToCustomScope(t *testing.T) {
	c := authClient()
	devApp := appRoot(t)

	defer cleanupPkg(c, "assistants", "@"+testScope, "simple-greeter", "1.0.0")

	agentMgr := agentmgr.New(c, devApp, &common.AutoConfirmPrompter{})
	agentMgr.Push(testScope+".simple-greeter", agentmgr.PushOptions{Version: "1.0.0"})

	installApp := t.TempDir()
	installAgent := agentmgr.New(c, installApp, &common.AutoConfirmPrompter{})
	installAgent.Add("@"+testScope+"/simple-greeter", agentmgr.AddOptions{})

	if err := installAgent.Fork("@"+testScope+"/simple-greeter", agentmgr.ForkOptions{TargetScope: "acme"}); err != nil {
		t.Fatalf("Fork to custom scope: %v", err)
	}

	requireFileExists(t, filepath.Join(installApp, "assistants", "acme", "simple-greeter", "package.yao"))

	pkg := requireLockfileHas(t, installApp, "@acme/simple-greeter")
	if pkg.ForkedFrom != "@"+testScope+"/simple-greeter" {
		t.Errorf("expected forked_from, got %s", pkg.ForkedFrom)
	}
}

// =============================================================================
// Fork target already exists
// =============================================================================

func TestE2EAgent_ForkTargetExists(t *testing.T) {
	c := authClient()
	devApp := appRoot(t)

	defer cleanupPkg(c, "assistants", "@"+testScope, "simple-greeter", "1.0.0")

	agentMgr := agentmgr.New(c, devApp, &common.AutoConfirmPrompter{})
	agentMgr.Push(testScope+".simple-greeter", agentmgr.PushOptions{Version: "1.0.0"})

	installApp := t.TempDir()
	installAgent := agentmgr.New(c, installApp, &common.AutoConfirmPrompter{})
	installAgent.Add("@"+testScope+"/simple-greeter", agentmgr.AddOptions{})

	// Pre-create target
	targetDir := filepath.Join(installApp, "assistants", "local", "simple-greeter")
	mustMkdir(t, targetDir)

	err := installAgent.Fork("@"+testScope+"/simple-greeter", agentmgr.ForkOptions{TargetScope: "local"})
	if err == nil {
		t.Fatal("expected fork to fail when target exists")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("expected 'already exists' error, got: %v", err)
	}
}

// =============================================================================
// Push @local is rejected
// =============================================================================

func TestE2EAgent_PushLocalRejected(t *testing.T) {
	c := authClient()
	devApp := t.TempDir()

	localDir := filepath.Join(devApp, "assistants", "local", "my-thing")
	mustMkdir(t, localDir)
	mustWriteFile(t, filepath.Join(localDir, "package.yao"), `{"name":"my-thing"}`)

	mgr := agentmgr.New(c, devApp, &common.AutoConfirmPrompter{})
	err := mgr.Push("local.my-thing", agentmgr.PushOptions{Version: "1.0.0"})
	if err == nil {
		t.Fatal("expected push of @local to be rejected")
	}
	if !strings.Contains(err.Error(), "@local") {
		t.Errorf("expected @local rejection error, got: %v", err)
	}
}

// =============================================================================
// Shared MCP dep: two agents share same MCP, verify required_by
// =============================================================================

func TestE2EAgent_SharedMCPDep(t *testing.T) {
	c := authClient()
	devApp := appRoot(t)

	defer cleanupPkg(c, "mcps", "@"+testScope, "registry-mcp", "1.0.0")
	defer cleanupPkg(c, "assistants", "@"+testScope, "registry-agent", "1.0.0")
	defer cleanupPkg(c, "assistants", "@"+testScope, "analytics", "1.0.0")
	defer cleanupPkg(c, "mcps", "@"+testScope, "data-tools", "1.0.0")

	// Push all dependencies
	mcpMgr := mcpmgr.New(c, devApp, &common.AutoConfirmPrompter{})
	mcpMgr.Push(testScope+".registry-mcp", mcpmgr.PushOptions{Version: "1.0.0"})
	mcpMgr.Push(testScope+".data-tools", mcpmgr.PushOptions{Version: "1.0.0"})

	agentPushMgr := agentmgr.New(c, devApp, &common.AutoConfirmPrompter{})
	agentPushMgr.Push(testScope+".registry-agent", agentmgr.PushOptions{Version: "1.0.0"})
	agentPushMgr.Push(testScope+".analytics", agentmgr.PushOptions{Version: "1.0.0"})

	// Install agent A (depends on registry-mcp)
	installApp := t.TempDir()
	installAgent := agentmgr.New(c, installApp, &common.AutoConfirmPrompter{})

	if err := installAgent.Add("@"+testScope+"/registry-agent", agentmgr.AddOptions{}); err != nil {
		t.Fatalf("Add registry-agent: %v", err)
	}

	// registry-mcp should be installed with required_by=[registry-agent]
	mcpPkg := requireLockfileHas(t, installApp, "@"+testScope+"/registry-mcp")
	if mcpPkg.Version != "1.0.0" {
		t.Errorf("want MCP version 1.0.0, got %s", mcpPkg.Version)
	}

	// Install agent B (depends on registry-mcp AND data-tools)
	if err := installAgent.Add("@"+testScope+"/analytics", agentmgr.AddOptions{}); err != nil {
		t.Fatalf("Add analytics: %v", err)
	}

	// registry-mcp should NOT be reinstalled, but required_by should include both agents
	lf, _ := common.LoadLockfile(installApp)
	mcpPkg2, _ := lf.GetPackage("@" + testScope + "/registry-mcp")

	requiredBySet := map[string]bool{}
	for _, rb := range mcpPkg2.RequiredBy {
		requiredBySet[rb] = true
	}
	if !requiredBySet["@"+testScope+"/registry-agent"] {
		t.Error("expected registry-agent in registry-mcp's required_by")
	}
	if !requiredBySet["@"+testScope+"/analytics"] {
		t.Error("expected analytics in registry-mcp's required_by")
	}

	// data-tools should also be installed
	requireLockfileHas(t, installApp, "@"+testScope+"/data-tools")

	// Verify disk files are not duplicated — only one copy of each MCP
	requireFileExists(t, filepath.Join(installApp, "mcps", testScope, "registry-mcp", "server.mcp.yao"))
	requireFileExists(t, filepath.Join(installApp, "mcps", testScope, "data-tools", "server.mcp.yao"))
}

// =============================================================================
// Agent Fork from registry: not installed locally, pull then fork
// =============================================================================

func TestE2EAgent_ForkFromRegistry(t *testing.T) {
	c := authClient()
	devApp := appRoot(t)

	defer cleanupPkg(c, "assistants", "@"+testScope, "simple-greeter", "1.0.0")

	agentMgr := agentmgr.New(c, devApp, &common.AutoConfirmPrompter{})
	agentMgr.Push(testScope+".simple-greeter", agentmgr.PushOptions{Version: "1.0.0"})

	// Fresh app — nothing installed
	forkApp := t.TempDir()
	forkAgent := agentmgr.New(c, forkApp, &common.AutoConfirmPrompter{})

	if err := forkAgent.Fork("@"+testScope+"/simple-greeter", agentmgr.ForkOptions{TargetScope: "local"}); err != nil {
		t.Fatalf("Fork from registry: %v", err)
	}

	requireFileExists(t, filepath.Join(forkApp, "assistants", "local", "simple-greeter", "package.yao"))
	requireFileContains(t, filepath.Join(forkApp, "assistants", "local", "simple-greeter", "prompts.yml"),
		"friendly greeter")

	pkg := requireLockfileHas(t, forkApp, "@local/simple-greeter")
	if pkg.ForkedFrom != "@"+testScope+"/simple-greeter" {
		t.Errorf("expected forked_from, got %s", pkg.ForkedFrom)
	}
}
