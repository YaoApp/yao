package manager_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yaoapp/yao/registry"
	"github.com/yaoapp/yao/registry/manager/common"
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

	for _, rel := range []string{
		filepath.Join("..", "..", "..", "yao-dev-app"),
		filepath.Join("..", "..", "..", "..", "app"),
		filepath.Join("..", "yao-dev-app"),
	} {
		abs, _ := filepath.Abs(rel)
		if check(abs) {
			return abs
		}
	}

	t.Skip("yao-dev-app with registry test fixtures not found; set YAO_TEST_APPLICATION")
	return ""
}

// mustMkdir creates a directory tree, failing the test on error.
func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0755); err != nil {
		t.Fatal(err)
	}
}

// mustWriteFile writes content to a file, creating parent directories as needed.
func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

// requireFileExists asserts that a file exists on disk.
func requireFileExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected file to exist: %s", path)
	}
}

// requireFileNotExists asserts that a file does NOT exist on disk.
func requireFileNotExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err == nil {
		t.Fatalf("expected file NOT to exist: %s", path)
	}
}

// requireFileContains asserts that a file exists and its content contains substr.
func requireFileContains(t *testing.T, path, substr string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("cannot read %s: %v", path, err)
	}
	if !strings.Contains(string(data), substr) {
		t.Errorf("expected %s to contain %q, got:\n%s", filepath.Base(path), substr, data)
	}
}

// requireFileNotContains asserts that a file's content does NOT contain substr.
func requireFileNotContains(t *testing.T, path, substr string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("cannot read %s: %v", path, err)
	}
	if strings.Contains(string(data), substr) {
		t.Errorf("expected %s NOT to contain %q, got:\n%s", filepath.Base(path), substr, data)
	}
}

// requireLockfileHas asserts that the lockfile at appRoot has a package with the given ID.
func requireLockfileHas(t *testing.T, appRoot, pkgID string) common.PackageInfo {
	t.Helper()
	lf, err := common.LoadLockfile(appRoot)
	if err != nil {
		t.Fatalf("load lockfile: %v", err)
	}
	pkg, ok := lf.GetPackage(pkgID)
	if !ok {
		t.Fatalf("expected %s in lockfile, packages: %v", pkgID, lockfileKeys(lf))
	}
	return pkg
}

// requireLockfileNotHas asserts that the lockfile does NOT contain the given package.
func requireLockfileNotHas(t *testing.T, appRoot, pkgID string) {
	t.Helper()
	lf, err := common.LoadLockfile(appRoot)
	if err != nil {
		t.Fatalf("load lockfile: %v", err)
	}
	if _, ok := lf.GetPackage(pkgID); ok {
		t.Fatalf("expected %s NOT in lockfile", pkgID)
	}
}

// lockfileKeys returns all package IDs in a lockfile (for debug output).
func lockfileKeys(lf *common.RegistryYao) []string {
	var keys []string
	for k := range lf.Packages {
		keys = append(keys, k)
	}
	return keys
}

// buildV2AgentApp creates a v2 variant of the agent+MCP fixtures in a temp directory.
// Content differs from v1 in yao-dev-app to verify update logic.
func buildV2AgentApp(t *testing.T) string {
	t.Helper()
	root := t.TempDir()

	// Agent v2: updated description, new tools.ts
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
    "servers": [
      { "server_id": "`+testScope+`.registry-mcp" }
    ]
  },
  "tags": ["Test", "Registry", "V2"],
  "sort": 999,
  "readonly": true,
  "automated": false,
  "mentionable": false
}`)
	mustWriteFile(t, filepath.Join(assistDir, "prompts.yml"),
		"- role: system\n  content: |\n    You are the v2 registry test assistant with enhanced capabilities.\n")
	mustWriteFile(t, filepath.Join(assistDir, "tools.ts"),
		`export function newV2Tool(): number { return 42; }`)

	// MCP v2: added "suggest" tool
	mcpDir := filepath.Join(root, "mcps", testScope, "registry-mcp")
	mustMkdir(t, mcpDir)
	mustWriteFile(t, filepath.Join(mcpDir, "server.mcp.yao"), `{
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

// buildV2MCPApp creates a v2 variant of the data-tools MCP for update testing.
func buildV2MCPApp(t *testing.T) string {
	t.Helper()
	root := t.TempDir()

	mcpDir := filepath.Join(root, "mcps", testScope, "data-tools")
	mustMkdir(t, mcpDir)
	mustWriteFile(t, filepath.Join(mcpDir, "server.mcp.yao"), `{
  "label": "Data Tools MCP v2",
  "description": "Enhanced v2 data tools with new merge capability",
  "transport": "process",
  "capabilities": {
    "tools": { "listChanged": false },
    "resources": { "subscribe": false, "listChanged": false }
  },
  "tools": {
    "aggregate": "scripts.`+testScope+`.data_tools.Aggregate",
    "transform": "scripts.`+testScope+`.data_tools.Transform",
    "validate": "scripts.`+testScope+`.data_tools.Validate",
    "merge": "scripts.`+testScope+`.data_tools.Merge",
    "format_csv": "scripts.`+testScope+`.data_utils.FormatCSV",
    "format_json": "scripts.`+testScope+`.data_utils.FormatJSON"
  }
}`)

	scriptDir := filepath.Join(root, "scripts", testScope)
	mustMkdir(t, scriptDir)
	mustWriteFile(t, filepath.Join(scriptDir, "data_tools.ts"), `/**
 * Data Tools MCP v2 - primary script
 */

function Aggregate(data: any[]): Record<string, number> {
  return { count: data.length, version: 2 };
}

function Transform(input: string): string {
  return input.toUpperCase() + "-v2";
}

function Validate(schema: string, data: any): boolean {
  return schema !== "" && data !== null;
}

function Merge(a: any, b: any): any {
  return { ...a, ...b };
}
`)
	mustWriteFile(t, filepath.Join(scriptDir, "data_utils.ts"), `/**
 * Data Tools MCP v2 - utility script
 */

function FormatCSV(rows: string[][]): string {
  return "header\\n" + rows.map((r) => r.join(",")).join("\\n");
}

function FormatJSON(data: any): string {
  return JSON.stringify(data, null, 2);
}
`)
	return root
}

// buildV2AnalyticsApp creates a v2 variant of the analytics agent for update testing.
func buildV2AnalyticsApp(t *testing.T) string {
	t.Helper()
	root := t.TempDir()

	assistDir := filepath.Join(root, "assistants", testScope, "analytics")
	mustMkdir(t, assistDir)
	mustWriteFile(t, filepath.Join(assistDir, "package.yao"), `{
  "name": "Analytics Agent v2",
  "avatar": "/api/__yao/app/icons/app.png",
  "connector": "gpt-4o",
  "description": "Enhanced v2 analytics assistant with charting support",
  "options": { "temperature": 0.2 },
  "public": false,
  "mcp": {
    "servers": [
      { "server_id": "`+testScope+`.registry-mcp" },
      { "server_id": "`+testScope+`.data-tools" }
    ]
  },
  "tags": ["Test", "Registry", "Analytics", "V2"],
  "sort": 998,
  "readonly": true,
  "automated": false,
  "mentionable": false
}`)
	mustWriteFile(t, filepath.Join(assistDir, "prompts.yml"),
		"- role: system\n  content: |\n    You are the v2 analytics assistant with charting capabilities.\n")
	mustWriteFile(t, filepath.Join(assistDir, "chart_helper.ts"),
		`export function renderChart(): string { return "chart-v2"; }`)

	return root
}

// buildRobotZip creates a robot package zip and pushes it to the registry.
func buildAndPushRobotZip(t *testing.T, c *registry.Client, name string, robot interface{}, version string) {
	t.Helper()

	robotBytes, _ := json.Marshal(robot)

	robotZipRoot := t.TempDir()
	robotDir := filepath.Join(robotZipRoot, "package")
	mustMkdir(t, robotDir)
	mustWriteFile(t, filepath.Join(robotDir, "pkg.yao"), `{
  "type": "robot",
  "scope": "@`+testScope+`",
  "name": "`+name+`",
  "version": "`+version+`",
  "description": "E2E test robot"
}`)
	mustWriteFile(t, filepath.Join(robotDir, "robot.json"), string(robotBytes))

	robotZip, err := common.PackDir(robotDir, &common.PkgManifest{
		Type:    common.TypeRobot,
		Scope:   "@" + testScope,
		Name:    name,
		Version: version,
	}, nil)
	if err != nil {
		t.Fatalf("pack robot %s: %v", name, err)
	}
	if _, err := c.Push("robots", "@"+testScope, name, version, robotZip); err != nil {
		t.Fatalf("push robot %s: %v", name, err)
	}
}
