package manager_test

import (
	"testing"

	agentmgr "github.com/yaoapp/yao/registry/manager/agent"
	"github.com/yaoapp/yao/registry/manager/common"
	mcpmgr "github.com/yaoapp/yao/registry/manager/mcp"
	robotmgr "github.com/yaoapp/yao/registry/manager/robot"
)

// =============================================================================
// Robot Add with agent + MCP dependencies
// =============================================================================

func TestE2ERobot_AddWithDeps(t *testing.T) {
	c := authClient()
	devApp := appRoot(t)

	defer cleanupPkg(c, "mcps", "@"+testScope, "registry-mcp", "1.0.0")
	defer cleanupPkg(c, "assistants", "@"+testScope, "registry-agent", "1.0.0")
	defer cleanupPkg(c, "robots", "@"+testScope, "test-bot", "1.0.0")

	// Push dependencies
	mcpMgr := mcpmgr.New(c, devApp, &common.AutoConfirmPrompter{})
	agentMgr := agentmgr.New(c, devApp, &common.AutoConfirmPrompter{})

	if err := mcpMgr.Push(testScope+".registry-mcp", mcpmgr.PushOptions{Version: "1.0.0"}); err != nil {
		t.Fatalf("Push MCP: %v", err)
	}
	if err := agentMgr.Push(testScope+".registry-agent", agentmgr.PushOptions{Version: "1.0.0"}); err != nil {
		t.Fatalf("Push agent: %v", err)
	}

	// Build and push robot package
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
	buildAndPushRobotZip(t, c, "test-bot", robotJSON, "1.0.0")

	// Install robot to fresh app
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

	// Lockfile: robot entry
	robotPkg := requireLockfileHas(t, installApp, "@"+testScope+"/test-bot")
	if robotPkg.Type != common.TypeRobot {
		t.Errorf("want robot type, got %s", robotPkg.Type)
	}
	if robotPkg.TeamID != "team-e2e" {
		t.Errorf("want team_id team-e2e, got %s", robotPkg.TeamID)
	}

	// Dependencies auto-installed
	requireLockfileHas(t, installApp, "@"+testScope+"/registry-agent")
	requireLockfileHas(t, installApp, "@"+testScope+"/registry-mcp")

	// Files on disk
	requireFileExists(t, installApp+"/assistants/"+testScope+"/registry-agent/package.yao")
	requireFileExists(t, installApp+"/mcps/"+testScope+"/registry-mcp/server.mcp.yao")

	// required_by on agent from robot
	lf, _ := common.LoadLockfile(installApp)
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
// Robot Add with no dependencies
// =============================================================================

func TestE2ERobot_AddNoDeps(t *testing.T) {
	c := authClient()

	defer cleanupPkg(c, "robots", "@"+testScope, "simple-bot", "1.0.0")

	robotJSON := map[string]interface{}{
		"display_name":   "Simple Bot",
		"system_prompt":  "You are a simple bot with no dependencies.",
		"language_model": "gpt-4o-mini",
	}
	buildAndPushRobotZip(t, c, "simple-bot", robotJSON, "1.0.0")

	installApp := t.TempDir()
	rMgr := robotmgr.New(c, installApp, &common.AutoConfirmPrompter{})

	robot, err := rMgr.Add("@"+testScope+"/simple-bot", robotmgr.AddOptions{TeamID: "team-simple"})
	if err != nil {
		t.Fatalf("Add robot: %v", err)
	}

	if robot.DisplayName != "Simple Bot" {
		t.Errorf("want 'Simple Bot', got %q", robot.DisplayName)
	}
	if robot.LanguageModel != "gpt-4o-mini" {
		t.Errorf("want gpt-4o-mini, got %s", robot.LanguageModel)
	}

	robotPkg := requireLockfileHas(t, installApp, "@"+testScope+"/simple-bot")
	if robotPkg.Type != common.TypeRobot {
		t.Errorf("want robot type, got %s", robotPkg.Type)
	}
	if robotPkg.TeamID != "team-simple" {
		t.Errorf("want team_id team-simple, got %s", robotPkg.TeamID)
	}

	// No other packages should be installed
	lf, _ := common.LoadLockfile(installApp)
	for id := range lf.Packages {
		if id != "@"+testScope+"/simple-bot" {
			t.Errorf("unexpected package %s in lockfile (no-dep robot should be alone)", id)
		}
	}
}

// =============================================================================
// Robot Add: team ID is required
// =============================================================================

func TestE2ERobot_AddRequiresTeam(t *testing.T) {
	c := authClient()

	defer cleanupPkg(c, "robots", "@"+testScope, "simple-bot", "1.0.0")

	robotJSON := map[string]interface{}{
		"display_name":  "Simple Bot",
		"system_prompt": "You are a simple bot.",
	}
	buildAndPushRobotZip(t, c, "simple-bot", robotJSON, "1.0.0")

	installApp := t.TempDir()
	rMgr := robotmgr.New(c, installApp, &common.AutoConfirmPrompter{})

	_, err := rMgr.Add("@"+testScope+"/simple-bot", robotmgr.AddOptions{})
	if err == nil {
		t.Fatal("expected error when team is missing")
	}
	if err.Error() != "--team is required for robot add" {
		t.Errorf("unexpected error: %v", err)
	}
}

// =============================================================================
// Robot → Agent → MCP dependency chain: required_by propagation
// =============================================================================

func TestE2ERobot_RequiredByChain(t *testing.T) {
	c := authClient()
	devApp := appRoot(t)

	defer cleanupPkg(c, "mcps", "@"+testScope, "registry-mcp", "1.0.0")
	defer cleanupPkg(c, "mcps", "@"+testScope, "data-tools", "1.0.0")
	defer cleanupPkg(c, "assistants", "@"+testScope, "analytics", "1.0.0")
	defer cleanupPkg(c, "robots", "@"+testScope, "analytics-bot", "1.0.0")

	// Push full dependency tree: 2 MCPs → analytics agent → robot
	mcpMgr := mcpmgr.New(c, devApp, &common.AutoConfirmPrompter{})
	mcpMgr.Push(testScope+".registry-mcp", mcpmgr.PushOptions{Version: "1.0.0"})
	mcpMgr.Push(testScope+".data-tools", mcpmgr.PushOptions{Version: "1.0.0"})

	agentMgr := agentmgr.New(c, devApp, &common.AutoConfirmPrompter{})
	agentMgr.Push(testScope+".analytics", agentmgr.PushOptions{Version: "1.0.0"})

	robotJSON := map[string]interface{}{
		"display_name":   "Analytics Bot",
		"system_prompt":  "You are an analytics bot.",
		"language_model": "gpt-4o",
		"robot_config": map[string]interface{}{
			"resources": map[string]interface{}{
				"phases": map[string]string{
					"host": testScope + ".analytics",
				},
			},
		},
	}
	buildAndPushRobotZip(t, c, "analytics-bot", robotJSON, "1.0.0")

	// Install robot to fresh app
	installApp := t.TempDir()
	rMgr := robotmgr.New(c, installApp, &common.AutoConfirmPrompter{})

	_, err := rMgr.Add("@"+testScope+"/analytics-bot", robotmgr.AddOptions{TeamID: "team-chain"})
	if err != nil {
		t.Fatalf("Add robot: %v", err)
	}

	// Entire dependency chain should be installed
	requireLockfileHas(t, installApp, "@"+testScope+"/analytics-bot")
	requireLockfileHas(t, installApp, "@"+testScope+"/analytics")
	requireLockfileHas(t, installApp, "@"+testScope+"/registry-mcp")
	requireLockfileHas(t, installApp, "@"+testScope+"/data-tools")

	// required_by: robot → analytics
	lf, _ := common.LoadLockfile(installApp)
	analyticsPkg, _ := lf.GetPackage("@" + testScope + "/analytics")
	foundRobot := false
	for _, rb := range analyticsPkg.RequiredBy {
		if rb == "@"+testScope+"/analytics-bot" {
			foundRobot = true
		}
	}
	if !foundRobot {
		t.Errorf("expected analytics-bot in analytics's required_by, got %v", analyticsPkg.RequiredBy)
	}

	// required_by: analytics → MCPs (set by agent Add's dependency installation)
	// The MCP's required_by may include analytics (set by agent add) and/or analytics-bot (set by robot add)
	for _, mcpID := range []string{"@" + testScope + "/registry-mcp", "@" + testScope + "/data-tools"} {
		mcpPkg, ok := lf.GetPackage(mcpID)
		if !ok {
			t.Errorf("MCP %s not found in lockfile", mcpID)
			continue
		}
		if len(mcpPkg.RequiredBy) == 0 {
			t.Errorf("expected required_by on %s, got empty", mcpID)
		}
	}

	// Verify disk completeness
	requireFileExists(t, installApp+"/assistants/"+testScope+"/analytics/package.yao")
	requireFileExists(t, installApp+"/mcps/"+testScope+"/registry-mcp/server.mcp.yao")
	requireFileExists(t, installApp+"/mcps/"+testScope+"/data-tools/server.mcp.yao")
	requireFileExists(t, installApp+"/scripts/"+testScope+"/registry_mcp.ts")
	requireFileExists(t, installApp+"/scripts/"+testScope+"/data_tools.ts")
	requireFileExists(t, installApp+"/scripts/"+testScope+"/data_utils.ts")
}
