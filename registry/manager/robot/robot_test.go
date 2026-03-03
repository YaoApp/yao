package robot

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yaoapp/yao/registry"
	"github.com/yaoapp/yao/registry/manager/common"
	"github.com/yaoapp/yao/registry/testdata"
)

func buildRobotZip(scope, name, version string, robotJSON *RobotJSON, deps []testdata.ManifestDep) []byte {
	robotBytes, _ := json.Marshal(robotJSON)
	zip, err := testdata.BuildZip(&testdata.Manifest{
		Type:         "robot",
		Scope:        scope,
		Name:         name,
		Version:      version,
		Dependencies: deps,
	}, map[string]string{
		"robot.json": string(robotBytes),
	})
	if err != nil {
		panic(err)
	}
	return zip
}

func buildAgentZip(scope, name, version string) []byte {
	zip, err := testdata.BuildZip(&testdata.Manifest{
		Type:    "assistant",
		Scope:   scope,
		Name:    name,
		Version: version,
	}, map[string]string{
		"package.yao": `{"name":"` + name + `"}`,
	})
	if err != nil {
		panic(err)
	}
	return zip
}

func buildMCPZip(scope, name, version string) []byte {
	zip, err := testdata.BuildZip(&testdata.Manifest{
		Type:    "mcp",
		Scope:   scope,
		Name:    name,
		Version: version,
	}, map[string]string{
		name + ".mcp.yao": `{"transport":"stdio","command":"echo"}`,
	})
	if err != nil {
		panic(err)
	}
	return zip
}

func mockServer(packages map[string][]byte) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/yao-registry" {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"registry": map[string]string{"version": "1.0.0", "api": "/v1"},
				"types":    []string{"assistants", "mcps", "robots"},
			})
			return
		}

		if r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/pull") {
			parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/v1/"), "/")
			if len(parts) >= 4 {
				key := parts[0] + "/" + parts[1] + "/" + parts[2]
				if zipData, ok := packages[key]; ok {
					w.Header().Set("X-Digest", "sha256-test")
					w.Write(zipData)
					return
				}
			}
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "not found"})
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
}

func TestAnalyzeDeps(t *testing.T) {
	robot := &RobotJSON{
		RobotConfig: json.RawMessage(`{
			"resources": {
				"phases": {
					"host": "yao.robot-host",
					"goals": "yao.robot-goals",
					"builtin": "__yao.default-host"
				}
			}
		}`),
		Agents:     []string{"yao.keeper.fetch", "__yao.system"},
		MCPServers: []string{"ark.image.text2img"},
	}

	deps := AnalyzeDeps(robot)

	depMap := map[string]string{}
	for _, d := range deps {
		depMap[d.PackageID] = d.Type
	}

	// phases
	if depMap["@yao/robot-host"] != "assistant" {
		t.Error("expected @yao/robot-host as assistant")
	}
	if depMap["@yao/robot-goals"] != "assistant" {
		t.Error("expected @yao/robot-goals as assistant")
	}

	// agents (first-layer)
	if depMap["@yao/keeper"] != "assistant" {
		t.Error("expected @yao/keeper as assistant")
	}

	// mcp_servers
	if depMap["@ark/image.text2img"] != "mcp" {
		t.Error("expected @ark/image.text2img as mcp")
	}

	// __yao.* should be excluded
	if _, ok := depMap["@__yao/default-host"]; ok {
		t.Error("expected __yao.default-host to be excluded")
	}
	if _, ok := depMap["@__yao/system"]; ok {
		t.Error("expected __yao.system to be excluded")
	}
}

func TestAnalyzeDepsEmpty(t *testing.T) {
	robot := &RobotJSON{}
	deps := AnalyzeDeps(robot)
	if len(deps) != 0 {
		t.Errorf("expected 0 deps, got %d", len(deps))
	}
}

func TestAnalyzeDepsDedupe(t *testing.T) {
	robot := &RobotJSON{
		RobotConfig: json.RawMessage(`{
			"resources": {
				"phases": {
					"host": "yao.robot-host"
				}
			}
		}`),
		Agents: []string{"yao.robot-host.run"},
	}

	deps := AnalyzeDeps(robot)
	count := 0
	for _, d := range deps {
		if d.PackageID == "@yao/robot-host" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected @yao/robot-host once, got %d times", count)
	}
}

func TestAddRobotNoTeam(t *testing.T) {
	appRoot := t.TempDir()
	srv := mockServer(nil)
	defer srv.Close()

	client := registry.New(srv.URL)
	mgr := New(client, appRoot, &common.AutoConfirmPrompter{})

	_, err := mgr.Add("@test/my-robot", AddOptions{})
	if err == nil {
		t.Fatal("expected error for missing team")
	}
	if !strings.Contains(err.Error(), "--team") {
		t.Errorf("expected team error, got: %v", err)
	}
}

func TestAddRobotWithDeps(t *testing.T) {
	appRoot := t.TempDir()

	agentZip := buildAgentZip("@test", "robot-host", "1.0.0")
	mcpZip := buildMCPZip("@test", "image-gen", "1.0.0")

	robotJSON := &RobotJSON{
		DisplayName: "Test Robot",
		RobotConfig: json.RawMessage(`{
			"resources": {
				"phases": {
					"host": "test.robot-host"
				}
			}
		}`),
		MCPServers: []string{"test.image-gen"},
	}
	robotZip := buildRobotZip("@test", "my-robot", "1.0.0", robotJSON, nil)

	srv := mockServer(map[string][]byte{
		"robots/@test/my-robot":       robotZip,
		"assistants/@test/robot-host": agentZip,
		"mcps/@test/image-gen":        mcpZip,
	})
	defer srv.Close()

	client := registry.New(srv.URL)
	mgr := New(client, appRoot, &common.AutoConfirmPrompter{})

	robot, err := mgr.Add("@test/my-robot", AddOptions{TeamID: "team-123"})
	if err != nil {
		t.Fatalf("Add robot failed: %v", err)
	}

	if robot.DisplayName != "Test Robot" {
		t.Errorf("expected display_name 'Test Robot', got %q", robot.DisplayName)
	}

	// Verify lockfile
	lf, _ := common.LoadLockfile(appRoot)
	pkg, ok := lf.GetPackage("@test/my-robot")
	if !ok {
		t.Fatal("expected @test/my-robot in lockfile")
	}
	if pkg.Type != common.TypeRobot {
		t.Errorf("expected type robot, got %s", pkg.Type)
	}
	if pkg.TeamID != "team-123" {
		t.Errorf("expected team_id team-123, got %s", pkg.TeamID)
	}

	// Verify dependencies were installed
	if _, ok := lf.GetPackage("@test/robot-host"); !ok {
		t.Error("expected @test/robot-host dependency installed")
	}
	if _, ok := lf.GetPackage("@test/image-gen"); !ok {
		t.Error("expected @test/image-gen dependency installed")
	}

	// Verify assistant directory was created
	agentDir := filepath.Join(appRoot, "assistants", "test", "robot-host")
	if _, err := os.Stat(agentDir); err != nil {
		t.Error("expected assistant directory created")
	}
}

func TestAddRobotNoDeps(t *testing.T) {
	appRoot := t.TempDir()

	robotJSON := &RobotJSON{
		DisplayName: "Simple Robot",
	}
	robotZip := buildRobotZip("@test", "simple-bot", "1.0.0", robotJSON, nil)

	srv := mockServer(map[string][]byte{
		"robots/@test/simple-bot": robotZip,
	})
	defer srv.Close()

	client := registry.New(srv.URL)
	mgr := New(client, appRoot, &common.AutoConfirmPrompter{})

	robot, err := mgr.Add("@test/simple-bot", AddOptions{TeamID: "team-1"})
	if err != nil {
		t.Fatalf("Add simple robot failed: %v", err)
	}
	if robot.DisplayName != "Simple Robot" {
		t.Errorf("expected 'Simple Robot', got %q", robot.DisplayName)
	}
}

func TestYaoIDToPackageID(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"yao.robot-host", "@yao/robot-host"},
		{"ark.image.text2img", "@ark/image.text2img"},
		{"bad", ""},
		{"", ""},
	}
	for _, tt := range tests {
		got := yaoIDToPackageID(tt.input)
		if got != tt.want {
			t.Errorf("yaoIDToPackageID(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestAgentYaoIDToPackageID(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"yao.keeper.fetch", "@yao/keeper"},
		{"yao.keeper", "@yao/keeper"},
		{"bad", ""},
	}
	for _, tt := range tests {
		got := agentYaoIDToPackageID(tt.input)
		if got != tt.want {
			t.Errorf("agentYaoIDToPackageID(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestIsBuiltIn(t *testing.T) {
	if !isBuiltIn("__yao.default-host") {
		t.Error("expected __yao.default-host to be built-in")
	}
	if isBuiltIn("yao.robot-host") {
		t.Error("expected yao.robot-host NOT to be built-in")
	}
}
