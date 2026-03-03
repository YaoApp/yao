package manager_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yaoapp/yao/registry"
	agentmgr "github.com/yaoapp/yao/registry/manager/agent"
	"github.com/yaoapp/yao/registry/manager/common"
	mcpmgr "github.com/yaoapp/yao/registry/manager/mcp"
	robotmgr "github.com/yaoapp/yao/registry/manager/robot"
)

// testScope is the scope aligned with registry CI credentials (yaoagents:yaoagents).
const testScope = "yaoagents"

func registryURL() string {
	if u := os.Getenv("YAO_REGISTRY_URL"); u != "" {
		return u
	}
	return "http://localhost:8080"
}

func authClient() *registry.Client {
	return registry.New(registryURL(), registry.WithAuth(testScope, testScope))
}

func cleanupPkg(c *registry.Client, pkgType, scope, name, version string) {
	c.DeleteVersion(pkgType, scope, name, version)
}

// appRoot returns the path to yao-dev-app, which contains the real test fixtures
// under assistants/yaoagents/, mcps/yaoagents/, scripts/yaoagents/.
//
// Resolution order:
//  1. YAO_TEST_APPLICATION env var (set by CI and local env.local.sh)
//  2. ../yao-dev-app (local development layout)
//  3. ../app (CI layout after "Move Dependencies" step)
func appRoot(t *testing.T) string {
	t.Helper()

	check := func(root string) bool {
		_, err := os.Stat(filepath.Join(root, "assistants", testScope, "registry-agent", "package.yao"))
		return err == nil
	}

	if root := os.Getenv("YAO_TEST_APPLICATION"); root != "" {
		abs, _ := filepath.Abs(root)
		if check(abs) {
			return abs
		}
		t.Logf("YAO_TEST_APPLICATION=%s exists but missing registry test fixtures", root)
	}

	// Try standard sibling layouts
	for _, rel := range []string{
		filepath.Join("..", "..", "..", "yao-dev-app"), // from registry/manager/ → yao-dev-app
		filepath.Join("..", "..", "..", "..", "app"),   // CI: from yao/registry/manager/ → ../app
		filepath.Join("..", "yao-dev-app"),             // from yao/ → yao-dev-app
	} {
		abs, _ := filepath.Abs(rel)
		if check(abs) {
			return abs
		}
	}

	t.Skip("yao-dev-app with registry test fixtures not found; set YAO_TEST_APPLICATION")
	return ""
}

// buildV2App creates a v2 variant of the test fixtures in a temp directory
// for update testing. The content is intentionally different from v1 in yao-dev-app.
func buildV2App(t *testing.T) string {
	t.Helper()
	root := t.TempDir()

	// Agent v2: updated description, new prompts, added tools.ts
	assistDir := filepath.Join(root, "assistants", testScope, "registry-agent")
	mustMkdir(t, assistDir)
	mustWriteFile(t, filepath.Join(assistDir, "package.yao"), `{
  "name": "Registry Test Agent v2",
  "avatar": "/api/__yao/app/icons/app.png",
  "connector": "gpt-4o",
  "description": "Enhanced v2 test assistant for registry E2E verification",
  "options": { "temperature": 0.5 },
  "public": false,
  "mcp": {
    "servers": {
      "registry-mcp": { "server_id": "`+testScope+`.registry-mcp" }
    }
  },
  "tags": ["Test", "Registry", "V2"],
  "sort": 999,
  "readonly": true,
  "automated": false,
  "mentionable": false
}`)
	mustWriteFile(t, filepath.Join(assistDir, "prompts.yml"),
		"system: |\n  You are the v2 registry test assistant with enhanced capabilities.\n")
	mustWriteFile(t, filepath.Join(assistDir, "tools.ts"),
		`export function newV2Tool(): number { return 42; }`)

	// MCP v2: added "suggest" tool
	mcpDir := filepath.Join(root, "mcps", testScope, "registry-mcp")
	mustMkdir(t, mcpDir)
	mustWriteFile(t, filepath.Join(mcpDir, "registry-mcp.mcp.yao"), `{
  "label": "Registry Test MCP v2",
  "description": "Enhanced v2 MCP for registry E2E testing",
  "transport": "process",
  "capabilities": {
    "tools": { "listChanged": false },
    "resources": { "subscribe": false, "listChanged": false }
  },
  "tools": {
    "ping": "scripts.`+testScope+`.registry_mcp.Ping",
    "echo": "scripts.`+testScope+`.registry_mcp.Echo",
    "suggest": "scripts.`+testScope+`.registry_mcp.Suggest"
  }
}`)

	// Script v2: added Suggest, changed Ping return
	scriptDir := filepath.Join(root, "scripts", testScope)
	mustMkdir(t, scriptDir)
	mustWriteFile(t, filepath.Join(scriptDir, "registry_mcp.ts"), `/**
 * Registry MCP test script v2
 */

function Ping(): string {
  return "pong-v2";
}

function Echo(input: string): string {
  return input;
}

function Suggest(prefix: string): string[] {
  return ["v2-suggestion1", "v2-suggestion2"];
}
`)

	return root
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0755); err != nil {
		t.Fatal(err)
	}
}

func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

// =============================================================================
// E2E: MCP full lifecycle — Push from yao-dev-app → Add → Update → Fork
// =============================================================================

func TestE2EMCPRealLifecycle(t *testing.T) {
	c := authClient()
	devApp := appRoot(t)

	defer cleanupPkg(c, "mcps", "@"+testScope, "registry-mcp", "1.0.0")
	defer cleanupPkg(c, "mcps", "@"+testScope, "registry-mcp", "2.0.0")

	// ---- Phase 1: Push MCP v1 from yao-dev-app (real developer push) ----

	pushMgr := mcpmgr.New(c, devApp, &common.AutoConfirmPrompter{})

	err := pushMgr.Push(testScope+".registry-mcp", mcpmgr.PushOptions{Version: "1.0.0"})
	if err != nil {
		t.Fatalf("Push MCP v1 from yao-dev-app: %v", err)
	}

	// Verify in registry
	packument, err := c.GetPackument("mcps", "@"+testScope, "registry-mcp")
	if err != nil {
		t.Fatalf("GetPackument after push: %v", err)
	}
	if packument.DistTags["latest"] != "1.0.0" {
		t.Errorf("expected latest=1.0.0, got %s", packument.DistTags["latest"])
	}

	// ---- Phase 2: Add to a fresh app (simulates another developer installing) ----

	installApp := t.TempDir()
	installMgr := mcpmgr.New(c, installApp, &common.AutoConfirmPrompter{})

	err = installMgr.Add("@"+testScope+"/registry-mcp", mcpmgr.AddOptions{})
	if err != nil {
		t.Fatalf("Add MCP: %v", err)
	}

	// Verify: .mcp.yao on disk
	installedMCP := filepath.Join(installApp, "mcps", testScope, "registry-mcp", "registry-mcp.mcp.yao")
	if _, err := os.Stat(installedMCP); err != nil {
		t.Fatal("expected registry-mcp.mcp.yao in installed dir")
	}
	mcpContent, _ := os.ReadFile(installedMCP)
	if !strings.Contains(string(mcpContent), "scripts."+testScope+".registry_mcp.Ping") {
		t.Errorf("expected process refs preserved, got: %s", mcpContent)
	}

	// Verify: scripts extracted to project root
	installedScript := filepath.Join(installApp, "scripts", testScope, "registry_mcp.ts")
	if _, err := os.Stat(installedScript); err != nil {
		t.Fatalf("expected scripts/%s/registry_mcp.ts extracted", testScope)
	}
	scriptContent, _ := os.ReadFile(installedScript)
	if !strings.Contains(string(scriptContent), "pong") {
		t.Errorf("expected v1 script with 'pong', got: %s", scriptContent)
	}

	// Verify: lockfile (registry.yao)
	lf, err := common.LoadLockfile(installApp)
	if err != nil {
		t.Fatal(err)
	}
	pkg, ok := lf.GetPackage("@" + testScope + "/registry-mcp")
	if !ok {
		t.Fatal("expected package in lockfile")
	}
	if pkg.Version != "1.0.0" {
		t.Errorf("lockfile version: want 1.0.0, got %s", pkg.Version)
	}
	if pkg.Type != common.TypeMCP {
		t.Errorf("lockfile type: want mcp, got %s", pkg.Type)
	}
	if pkg.Integrity == "" {
		t.Error("expected integrity digest in lockfile")
	}

	// lockfile.Files must track both MCP dir files and script files
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

	// ---- Phase 3: Push v2 from temp dir, then Update ----

	v2App := buildV2App(t)
	pushMgrV2 := mcpmgr.New(c, v2App, &common.AutoConfirmPrompter{})

	err = pushMgrV2.Push(testScope+".registry-mcp", mcpmgr.PushOptions{Version: "2.0.0"})
	if err != nil {
		t.Fatalf("Push MCP v2: %v", err)
	}

	err = installMgr.Update("@"+testScope+"/registry-mcp", mcpmgr.UpdateOptions{Version: "2.0.0"})
	if err != nil {
		t.Fatalf("Update MCP to v2: %v", err)
	}

	// lockfile version should be 2.0.0
	lf, _ = common.LoadLockfile(installApp)
	pkg, _ = lf.GetPackage("@" + testScope + "/registry-mcp")
	if pkg.Version != "2.0.0" {
		t.Errorf("expected v2.0.0 after update, got %s", pkg.Version)
	}

	// Script should contain v2 content
	scriptContent, _ = os.ReadFile(installedScript)
	if !strings.Contains(string(scriptContent), "pong-v2") {
		t.Errorf("expected v2 script content after update, got: %s", scriptContent)
	}

	// ---- Phase 4: Fork to @local ----

	err = installMgr.Fork("@"+testScope+"/registry-mcp", mcpmgr.ForkOptions{TargetScope: "local"})
	if err != nil {
		t.Fatalf("Fork MCP: %v", err)
	}

	// Forked MCP directory
	forkedDir := filepath.Join(installApp, "mcps", "local", "registry-mcp")
	if _, err := os.Stat(forkedDir); err != nil {
		t.Fatal("expected forked MCP directory at mcps/local/registry-mcp/")
	}

	// Process refs rewritten: scripts.yaoagents.* → scripts.local.*
	forkedMCPContent, _ := os.ReadFile(filepath.Join(forkedDir, "registry-mcp.mcp.yao"))
	if !strings.Contains(string(forkedMCPContent), "scripts.local.registry_mcp.Ping") {
		t.Errorf("expected rewritten process ref scripts.local.*, got: %s", forkedMCPContent)
	}
	if strings.Contains(string(forkedMCPContent), "scripts."+testScope+".") {
		t.Errorf("forked MCP still references original scope: %s", forkedMCPContent)
	}

	// Forked scripts copied
	forkedScript := filepath.Join(installApp, "scripts", "local", "registry_mcp.ts")
	if _, err := os.Stat(forkedScript); err != nil {
		t.Fatal("expected scripts/local/registry_mcp.ts after fork")
	}

	// Lockfile: forked entry is unmanaged
	lf, _ = common.LoadLockfile(installApp)
	forkedPkg, ok := lf.GetPackage("@local/registry-mcp")
	if !ok {
		t.Fatal("expected @local/registry-mcp in lockfile")
	}
	if forkedPkg.ForkedFrom != "@"+testScope+"/registry-mcp" {
		t.Errorf("expected forked_from=@%s/registry-mcp, got %s", testScope, forkedPkg.ForkedFrom)
	}
	if forkedPkg.IsManaged() {
		t.Error("forked package should not be managed")
	}
}

// =============================================================================
// E2E: Agent full lifecycle — Push → Add (auto-installs MCP dep) → Update → Fork
// =============================================================================

func TestE2EAgentRealLifecycle(t *testing.T) {
	c := authClient()
	devApp := appRoot(t)

	defer cleanupPkg(c, "mcps", "@"+testScope, "registry-mcp", "1.0.0")
	defer cleanupPkg(c, "assistants", "@"+testScope, "registry-agent", "1.0.0")
	defer cleanupPkg(c, "assistants", "@"+testScope, "registry-agent", "2.0.0")

	// ---- Phase 1: Push MCP dependency first (agent's package.yao references it) ----

	mcpMgr := mcpmgr.New(c, devApp, &common.AutoConfirmPrompter{})
	err := mcpMgr.Push(testScope+".registry-mcp", mcpmgr.PushOptions{Version: "1.0.0"})
	if err != nil {
		t.Fatalf("Push MCP dependency: %v", err)
	}

	// ---- Phase 2: Push assistant from yao-dev-app ----

	agentMgr := agentmgr.New(c, devApp, &common.AutoConfirmPrompter{})
	err = agentMgr.Push(testScope+".registry-agent", agentmgr.PushOptions{Version: "1.0.0"})
	if err != nil {
		t.Fatalf("Push agent: %v", err)
	}

	packument, err := c.GetPackument("assistants", "@"+testScope, "registry-agent")
	if err != nil {
		t.Fatalf("GetPackument agent: %v", err)
	}
	if packument.DistTags["latest"] != "1.0.0" {
		t.Errorf("expected latest=1.0.0, got %s", packument.DistTags["latest"])
	}

	// ---- Phase 3: Add agent to fresh app (MCP dependency should auto-install) ----

	installApp := t.TempDir()
	installAgent := agentmgr.New(c, installApp, &common.AutoConfirmPrompter{})

	err = installAgent.Add("@"+testScope+"/registry-agent", agentmgr.AddOptions{})
	if err != nil {
		t.Fatalf("Add agent: %v", err)
	}

	// Agent directory on disk
	agentDir := filepath.Join(installApp, "assistants", testScope, "registry-agent")
	if _, err := os.Stat(agentDir); err != nil {
		t.Fatal("expected assistants/" + testScope + "/registry-agent/ directory")
	}
	if _, err := os.Stat(filepath.Join(agentDir, "package.yao")); err != nil {
		t.Error("expected package.yao in installed agent")
	}
	promptsContent, _ := os.ReadFile(filepath.Join(agentDir, "prompts.yml"))
	if !strings.Contains(string(promptsContent), "registry E2E testing") {
		t.Errorf("expected original prompts content, got: %s", promptsContent)
	}

	// Lockfile: agent entry
	lf, err := common.LoadLockfile(installApp)
	if err != nil {
		t.Fatal(err)
	}
	agentPkg, ok := lf.GetPackage("@" + testScope + "/registry-agent")
	if !ok {
		t.Fatal("expected agent in lockfile")
	}
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
	depPkg, ok := lf.GetPackage("@" + testScope + "/registry-mcp")
	if !ok {
		t.Fatal("expected MCP dependency @" + testScope + "/registry-mcp auto-installed")
	}
	if depPkg.Version != "1.0.0" {
		t.Errorf("dependency version: want 1.0.0, got %s", depPkg.Version)
	}

	// required_by set correctly
	found := false
	for _, rb := range depPkg.RequiredBy {
		if rb == "@"+testScope+"/registry-agent" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected @%s/registry-agent in dependency's required_by, got %v", testScope, depPkg.RequiredBy)
	}

	// ---- Phase 4: Push agent v2 and Update (local modification preserved) ----

	v2App := buildV2App(t)
	pushAgentV2 := agentmgr.New(c, v2App, &common.AutoConfirmPrompter{})

	err = pushAgentV2.Push(testScope+".registry-agent", agentmgr.PushOptions{Version: "2.0.0"})
	if err != nil {
		t.Fatalf("Push agent v2: %v", err)
	}

	// Locally modify prompts.yml before update (simulates developer customization)
	customPrompt := "My custom prompt - DO NOT OVERWRITE."
	os.WriteFile(filepath.Join(agentDir, "prompts.yml"), []byte(customPrompt), 0644)

	err = installAgent.Update("@"+testScope+"/registry-agent", agentmgr.UpdateOptions{Version: "2.0.0"})
	if err != nil {
		t.Fatalf("Update agent to v2: %v", err)
	}

	// Lockfile updated to v2
	lf, _ = common.LoadLockfile(installApp)
	agentPkg, _ = lf.GetPackage("@" + testScope + "/registry-agent")
	if agentPkg.Version != "2.0.0" {
		t.Errorf("expected v2.0.0 after update, got %s", agentPkg.Version)
	}

	// Locally modified file PRESERVED (not overwritten)
	preservedData, _ := os.ReadFile(filepath.Join(agentDir, "prompts.yml"))
	if string(preservedData) != customPrompt {
		t.Errorf("local modification should be preserved, got: %s", preservedData)
	}

	// .new file created with upstream v2 content
	newFile := filepath.Join(agentDir, "prompts.yml.new")
	if _, err := os.Stat(newFile); err != nil {
		t.Error("expected prompts.yml.new with upstream content")
	}
	newContent, _ := os.ReadFile(newFile)
	if !strings.Contains(string(newContent), "v2 registry test assistant") {
		t.Errorf("expected v2 content in .new file, got: %s", newContent)
	}

	// New file tools.ts added by v2
	if _, err := os.Stat(filepath.Join(agentDir, "tools.ts")); err != nil {
		t.Error("expected new file tools.ts added during update")
	}

	// ---- Phase 5: Fork to @local ----

	err = installAgent.Fork("@"+testScope+"/registry-agent", agentmgr.ForkOptions{TargetScope: "local"})
	if err != nil {
		t.Fatalf("Fork agent: %v", err)
	}

	forkDir := filepath.Join(installApp, "assistants", "local", "registry-agent")
	if _, err := os.Stat(forkDir); err != nil {
		t.Fatal("expected assistants/local/registry-agent/ after fork")
	}
	if _, err := os.Stat(filepath.Join(forkDir, "package.yao")); err != nil {
		t.Error("expected package.yao in forked dir")
	}

	lf, _ = common.LoadLockfile(installApp)
	forkedPkg, ok := lf.GetPackage("@local/registry-agent")
	if !ok {
		t.Fatal("expected @local/registry-agent in lockfile")
	}
	if forkedPkg.ForkedFrom != "@"+testScope+"/registry-agent" {
		t.Errorf("expected forked_from=@%s/registry-agent, got %s", testScope, forkedPkg.ForkedFrom)
	}
	if forkedPkg.IsManaged() {
		t.Error("forked package should not be managed")
	}
}

// =============================================================================
// E2E: Push→Pull roundtrip (byte-for-byte content verification)
// =============================================================================

func TestE2EPushPullRoundtrip(t *testing.T) {
	c := authClient()
	devApp := appRoot(t)

	defer cleanupPkg(c, "mcps", "@"+testScope, "registry-mcp", "1.0.0")

	pushMgr := mcpmgr.New(c, devApp, &common.AutoConfirmPrompter{})
	err := pushMgr.Push(testScope+".registry-mcp", mcpmgr.PushOptions{Version: "1.0.0"})
	if err != nil {
		t.Fatalf("Push: %v", err)
	}

	pullApp := t.TempDir()
	pullMgr := mcpmgr.New(c, pullApp, &common.AutoConfirmPrompter{})
	err = pullMgr.Add("@"+testScope+"/registry-mcp", mcpmgr.AddOptions{})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	// Script byte-for-byte comparison
	origScript, _ := os.ReadFile(filepath.Join(devApp, "scripts", testScope, "registry_mcp.ts"))
	pulledScript, _ := os.ReadFile(filepath.Join(pullApp, "scripts", testScope, "registry_mcp.ts"))
	if string(origScript) != string(pulledScript) {
		t.Errorf("script mismatch.\nOriginal:\n%s\nPulled:\n%s", origScript, pulledScript)
	}

	// MCP definition byte-for-byte comparison
	origMCP, _ := os.ReadFile(filepath.Join(devApp, "mcps", testScope, "registry-mcp", "registry-mcp.mcp.yao"))
	pulledMCP, _ := os.ReadFile(filepath.Join(pullApp, "mcps", testScope, "registry-mcp", "registry-mcp.mcp.yao"))
	if string(origMCP) != string(pulledMCP) {
		t.Errorf("MCP mismatch.\nOriginal:\n%s\nPulled:\n%s", origMCP, pulledMCP)
	}
}

// =============================================================================
// E2E: Robot Add with dependency resolution
// =============================================================================

func TestE2ERobotRealLifecycle(t *testing.T) {
	c := authClient()
	devApp := appRoot(t)

	defer cleanupPkg(c, "mcps", "@"+testScope, "registry-mcp", "1.0.0")
	defer cleanupPkg(c, "assistants", "@"+testScope, "registry-agent", "1.0.0")
	defer cleanupPkg(c, "robots", "@"+testScope, "test-bot", "1.0.0")

	// Push MCP and Agent that the robot depends on
	mcpMgr := mcpmgr.New(c, devApp, &common.AutoConfirmPrompter{})
	agentMgr := agentmgr.New(c, devApp, &common.AutoConfirmPrompter{})

	if err := mcpMgr.Push(testScope+".registry-mcp", mcpmgr.PushOptions{Version: "1.0.0"}); err != nil {
		t.Fatalf("Push MCP: %v", err)
	}
	if err := agentMgr.Push(testScope+".registry-agent", agentmgr.PushOptions{Version: "1.0.0"}); err != nil {
		t.Fatalf("Push agent: %v", err)
	}

	// Build and push robot package (robots are DB records, so we build zip manually)
	robotJSON := map[string]interface{}{
		"display_name":   "E2E Test Bot",
		"system_prompt":  "You are an E2E test robot.",
		"language_model": "gpt-4o",
		"robot_config": map[string]interface{}{
			"resources": map[string]interface{}{
				"phases": map[string]string{
					"host": testScope + ".registry-agent",
				},
			},
		},
		"mcp_servers": []string{testScope + ".registry-mcp"},
	}
	robotBytes, _ := json.Marshal(robotJSON)

	robotZipRoot := t.TempDir()
	robotDir := filepath.Join(robotZipRoot, "package")
	mustMkdir(t, robotDir)
	mustWriteFile(t, filepath.Join(robotDir, "pkg.yao"), `{
  "type": "robot",
  "scope": "@`+testScope+`",
  "name": "test-bot",
  "version": "1.0.0",
  "description": "E2E test robot"
}`)
	mustWriteFile(t, filepath.Join(robotDir, "robot.json"), string(robotBytes))

	robotZip, err := common.PackDir(robotDir, &common.PkgManifest{
		Type:    common.TypeRobot,
		Scope:   "@" + testScope,
		Name:    "test-bot",
		Version: "1.0.0",
	}, nil)
	if err != nil {
		t.Fatalf("Pack robot: %v", err)
	}
	if _, err := c.Push("robots", "@"+testScope, "test-bot", "1.0.0", robotZip); err != nil {
		t.Fatalf("Push robot: %v", err)
	}

	// ---- Add robot to fresh app (dependencies should auto-install) ----

	installApp := t.TempDir()
	rMgr := robotmgr.New(c, installApp, &common.AutoConfirmPrompter{})

	robot, err := rMgr.Add("@"+testScope+"/test-bot", robotmgr.AddOptions{TeamID: "team-e2e"})
	if err != nil {
		t.Fatalf("Add robot: %v", err)
	}

	if robot.DisplayName != "E2E Test Bot" {
		t.Errorf("want display_name 'E2E Test Bot', got %q", robot.DisplayName)
	}
	if robot.SystemPrompt != "You are an E2E test robot." {
		t.Errorf("unexpected system_prompt: %s", robot.SystemPrompt)
	}

	lf, _ := common.LoadLockfile(installApp)

	robotPkg, ok := lf.GetPackage("@" + testScope + "/test-bot")
	if !ok {
		t.Fatal("expected robot in lockfile")
	}
	if robotPkg.Type != common.TypeRobot {
		t.Errorf("want robot type, got %s", robotPkg.Type)
	}
	if robotPkg.TeamID != "team-e2e" {
		t.Errorf("want team_id team-e2e, got %s", robotPkg.TeamID)
	}

	// Dependencies auto-installed
	if _, ok := lf.GetPackage("@" + testScope + "/registry-agent"); !ok {
		t.Error("expected agent dependency auto-installed")
	}
	if _, ok := lf.GetPackage("@" + testScope + "/registry-mcp"); !ok {
		t.Error("expected MCP dependency auto-installed")
	}

	// Files on disk
	if _, err := os.Stat(filepath.Join(installApp, "assistants", testScope, "registry-agent", "package.yao")); err != nil {
		t.Error("expected agent package.yao on disk after robot add")
	}
	if _, err := os.Stat(filepath.Join(installApp, "mcps", testScope, "registry-mcp", "registry-mcp.mcp.yao")); err != nil {
		t.Error("expected MCP .mcp.yao on disk after robot add")
	}

	// required_by
	agentPkg, _ := lf.GetPackage("@" + testScope + "/registry-agent")
	foundRB := false
	for _, rb := range agentPkg.RequiredBy {
		if rb == "@"+testScope+"/test-bot" {
			foundRB = true
		}
	}
	if !foundRB {
		t.Errorf("expected robot in agent's required_by, got %v", agentPkg.RequiredBy)
	}
}

// =============================================================================
// E2E: @local push is rejected
// =============================================================================

func TestE2EPushLocalRejected(t *testing.T) {
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
// E2E: MCP Push rejects scripts in wrong scope
// =============================================================================

func TestE2EMCPPushWrongScriptScope(t *testing.T) {
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
// E2E: Update of forked package is rejected
// =============================================================================

func TestE2EUpdateForkedRejected(t *testing.T) {
	c := authClient()
	devApp := appRoot(t)

	defer cleanupPkg(c, "mcps", "@"+testScope, "registry-mcp", "1.0.0")

	pushMgr := mcpmgr.New(c, devApp, &common.AutoConfirmPrompter{})
	pushMgr.Push(testScope+".registry-mcp", mcpmgr.PushOptions{Version: "1.0.0"})

	installApp := t.TempDir()
	installMgr := mcpmgr.New(c, installApp, &common.AutoConfirmPrompter{})
	installMgr.Add("@"+testScope+"/registry-mcp", mcpmgr.AddOptions{})

	// Fork it
	installMgr.Fork("@"+testScope+"/registry-mcp", mcpmgr.ForkOptions{TargetScope: "local"})

	// Update forked should fail
	err := installMgr.Update("@local/registry-mcp", mcpmgr.UpdateOptions{Version: "2.0.0"})
	if err == nil {
		t.Fatal("expected update of forked package to be rejected")
	}
	if !strings.Contains(err.Error(), "forked") {
		t.Errorf("expected 'forked' in error, got: %v", err)
	}
}

// =============================================================================
// E2E: Directory conflict detection
// =============================================================================

func TestE2EDirectoryConflict(t *testing.T) {
	c := authClient()
	devApp := appRoot(t)

	defer cleanupPkg(c, "mcps", "@"+testScope, "registry-mcp", "1.0.0")

	pushMgr := mcpmgr.New(c, devApp, &common.AutoConfirmPrompter{})
	pushMgr.Push(testScope+".registry-mcp", mcpmgr.PushOptions{Version: "1.0.0"})

	// Pre-create an unmanaged directory at the install path
	installApp := t.TempDir()
	conflictDir := filepath.Join(installApp, "mcps", testScope, "registry-mcp")
	mustMkdir(t, conflictDir)
	mustWriteFile(t, filepath.Join(conflictDir, "my-custom.mcp.yao"), `{"transport":"stdio"}`)

	installMgr := mcpmgr.New(c, installApp, &common.AutoConfirmPrompter{})
	err := installMgr.Add("@"+testScope+"/registry-mcp", mcpmgr.AddOptions{})
	if err == nil {
		t.Fatal("expected directory conflict error")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("expected 'already exists' error, got: %v", err)
	}
}
