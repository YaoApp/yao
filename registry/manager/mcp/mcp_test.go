package mcp

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

func buildMCPZip(scope, name, version string, files map[string]string) []byte {
	zip, err := testdata.BuildZip(&testdata.Manifest{
		Type:    "mcp",
		Scope:   scope,
		Name:    name,
		Version: version,
	}, files)
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

		if r.Method == http.MethodPut {
			w.WriteHeader(http.StatusCreated)
			parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/v1/"), "/")
			json.NewEncoder(w).Encode(map[string]string{
				"type": parts[0], "scope": parts[1], "name": parts[2],
				"version": parts[3], "digest": "sha256-pushed",
			})
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
}

func TestAddMCP(t *testing.T) {
	appRoot := t.TempDir()

	zip := buildMCPZip("@test", "echo-mcp", "1.0.0", map[string]string{
		"echo.mcp.yao":         `{"transport":"process","tools":{"echo":"scripts.test.echo.Echo"}}`,
		"scripts/test/echo.ts": "export function Echo() {}",
	})

	srv := mockServer(map[string][]byte{"mcps/@test/echo-mcp": zip})
	defer srv.Close()

	client := registry.New(srv.URL)
	mgr := New(client, appRoot, &common.AutoConfirmPrompter{})

	err := mgr.Add("@test/echo-mcp", AddOptions{})
	if err != nil {
		t.Fatalf("Add MCP failed: %v", err)
	}

	// Verify MCP directory
	mcpDir := filepath.Join(appRoot, "mcps", "test", "echo-mcp")
	if _, err := os.Stat(filepath.Join(mcpDir, "echo.mcp.yao")); err != nil {
		t.Error("expected echo.mcp.yao in MCP dir")
	}

	// Verify scripts extracted to project root
	scriptPath := filepath.Join(appRoot, "scripts", "test", "echo.ts")
	if _, err := os.Stat(scriptPath); err != nil {
		t.Error("expected scripts/test/echo.ts in project root")
	}

	// Verify lockfile
	lf, _ := common.LoadLockfile(appRoot)
	pkg, ok := lf.GetPackage("@test/echo-mcp")
	if !ok {
		t.Fatal("expected @test/echo-mcp in lockfile")
	}
	if pkg.Type != common.TypeMCP {
		t.Errorf("expected type mcp, got %s", pkg.Type)
	}

	// Verify files include both MCP dir and scripts
	hasScript := false
	hasMCPFile := false
	for path := range pkg.Files {
		if strings.HasPrefix(path, "scripts/") {
			hasScript = true
		}
		if strings.HasPrefix(path, "mcps/") {
			hasMCPFile = true
		}
	}
	if !hasScript {
		t.Error("expected script path in files")
	}
	if !hasMCPFile {
		t.Error("expected MCP file path in files")
	}
}

func TestUpdateMCP(t *testing.T) {
	appRoot := t.TempDir()

	zipV1 := buildMCPZip("@test", "upd-mcp", "1.0.0", map[string]string{
		"upd.mcp.yao":         `{"transport":"process","tools":{"run":"scripts.test.upd.Run"}}`,
		"scripts/test/upd.ts": "export function Run() { return 'v1'; }",
	})
	zipV2 := buildMCPZip("@test", "upd-mcp", "2.0.0", map[string]string{
		"upd.mcp.yao":         `{"transport":"process","tools":{"run":"scripts.test.upd.Run"}}`,
		"scripts/test/upd.ts": "export function Run() { return 'v2'; }",
	})

	srvV1 := mockServer(map[string][]byte{"mcps/@test/upd-mcp": zipV1})
	clientV1 := registry.New(srvV1.URL)
	mgrV1 := New(clientV1, appRoot, &common.AutoConfirmPrompter{})
	if err := mgrV1.Add("@test/upd-mcp", AddOptions{}); err != nil {
		t.Fatal(err)
	}
	srvV1.Close()

	srvV2 := mockServer(map[string][]byte{"mcps/@test/upd-mcp": zipV2})
	defer srvV2.Close()
	clientV2 := registry.New(srvV2.URL)
	mgrV2 := New(clientV2, appRoot, &common.AutoConfirmPrompter{})

	err := mgrV2.Update("@test/upd-mcp", UpdateOptions{})
	if err != nil {
		t.Fatalf("Update MCP failed: %v", err)
	}

	lf, _ := common.LoadLockfile(appRoot)
	pkg, _ := lf.GetPackage("@test/upd-mcp")
	if pkg.Version != "2.0.0" {
		t.Errorf("expected version 2.0.0, got %s", pkg.Version)
	}

	// Verify updated script content
	data, _ := os.ReadFile(filepath.Join(appRoot, "scripts", "test", "upd.ts"))
	if !strings.Contains(string(data), "v2") {
		t.Errorf("expected updated script content, got: %s", data)
	}
}

func TestPushMCP(t *testing.T) {
	appRoot := t.TempDir()

	// Create MCP directory structure
	mcpDir := filepath.Join(appRoot, "mcps", "max", "search")
	os.MkdirAll(mcpDir, 0755)
	os.WriteFile(filepath.Join(mcpDir, "search.mcp.yao"), []byte(`{
		"transport": "process",
		"tools": {"search": "scripts.max.search.Search"}
	}`), 0644)

	// Create scripts in the proper scope directory
	scriptDir := filepath.Join(appRoot, "scripts", "max")
	os.MkdirAll(scriptDir, 0755)
	os.WriteFile(filepath.Join(scriptDir, "search.ts"), []byte("export function Search() {}"), 0644)

	srv := mockServer(nil)
	defer srv.Close()

	client := registry.New(srv.URL, registry.WithAuth("u", "p"))
	mgr := New(client, appRoot, &common.AutoConfirmPrompter{})

	err := mgr.Push("max.search", PushOptions{Version: "1.0.0"})
	if err != nil {
		t.Fatalf("Push MCP failed: %v", err)
	}
}

func TestPushMCPWrongScope(t *testing.T) {
	appRoot := t.TempDir()

	mcpDir := filepath.Join(appRoot, "mcps", "max", "bad-scope")
	os.MkdirAll(mcpDir, 0755)
	os.WriteFile(filepath.Join(mcpDir, "bad.mcp.yao"), []byte(`{
		"transport": "process",
		"tools": {"run": "scripts.other.bad.Run"}
	}`), 0644)

	// Scripts in wrong scope
	os.MkdirAll(filepath.Join(appRoot, "scripts", "other"), 0755)
	os.WriteFile(filepath.Join(appRoot, "scripts", "other", "bad.ts"), []byte("nope"), 0644)

	srv := mockServer(nil)
	defer srv.Close()
	client := registry.New(srv.URL, registry.WithAuth("u", "p"))
	mgr := New(client, appRoot, &common.AutoConfirmPrompter{})

	err := mgr.Push("max.bad-scope", PushOptions{Version: "1.0.0"})
	if err == nil {
		t.Fatal("expected error for wrong script scope")
	}
	if !strings.Contains(err.Error(), "scope mismatch") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestForkMCPLocal(t *testing.T) {
	appRoot := t.TempDir()

	// Create installed MCP
	mcpDir := filepath.Join(appRoot, "mcps", "yao", "rag-tools")
	os.MkdirAll(mcpDir, 0755)
	mcpContent := `{"transport":"process","tools":{"search":"scripts.yao.rag.Search"}}`
	os.WriteFile(filepath.Join(mcpDir, "rag-tools.mcp.yao"), []byte(mcpContent), 0644)

	// Create scripts
	os.MkdirAll(filepath.Join(appRoot, "scripts", "yao"), 0755)
	os.WriteFile(filepath.Join(appRoot, "scripts", "yao", "rag.ts"), []byte("export function Search() {}"), 0644)

	lf := &common.RegistryYao{
		Scope: "@local",
		Packages: map[string]common.PackageInfo{
			"@yao/rag-tools": {
				Type:    common.TypeMCP,
				Version: "1.0.0",
				Files: map[string]string{
					"mcps/yao/rag-tools/rag-tools.mcp.yao": "sha256-aaa",
					"scripts/yao/rag.ts":                   "sha256-bbb",
				},
			},
		},
	}
	common.SaveLockfile(appRoot, lf)

	srv := mockServer(nil)
	defer srv.Close()
	client := registry.New(srv.URL)
	mgr := New(client, appRoot, &common.AutoConfirmPrompter{})

	err := mgr.Fork("@yao/rag-tools", ForkOptions{})
	if err != nil {
		t.Fatalf("Fork MCP failed: %v", err)
	}

	// Verify forked MCP directory
	forkedDir := filepath.Join(appRoot, "mcps", "local", "rag-tools")
	if _, err := os.Stat(forkedDir); err != nil {
		t.Fatal("expected forked MCP directory")
	}

	// Verify process references rewritten
	data, err := os.ReadFile(filepath.Join(forkedDir, "rag-tools.mcp.yao"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "scripts.local.rag.Search") {
		t.Errorf("expected rewritten process ref, got: %s", data)
	}
	if strings.Contains(string(data), "scripts.yao.rag.Search") {
		t.Error("expected old process ref to be removed")
	}

	// Verify scripts copied to new scope
	forkedScript := filepath.Join(appRoot, "scripts", "local", "rag.ts")
	if _, err := os.Stat(forkedScript); err != nil {
		t.Error("expected forked script in scripts/local/")
	}

	// Verify lockfile
	lf, _ = common.LoadLockfile(appRoot)
	pkg, ok := lf.GetPackage("@local/rag-tools")
	if !ok {
		t.Fatal("expected @local/rag-tools in lockfile")
	}
	if pkg.ForkedFrom != "@yao/rag-tools" {
		t.Errorf("expected forked_from @yao/rag-tools, got %s", pkg.ForkedFrom)
	}
	if pkg.IsManaged() {
		t.Error("expected managed=false")
	}
}

func TestScriptExtraction(t *testing.T) {
	refs, err := ExtractProcessRefsFromBytes([]byte(`{
		"transport": "process",
		"tools": {
			"search": "scripts.yao.rag.Search",
			"index": "scripts.yao.rag.Index",
			"status": "agents.robot.host.tools.Status"
		}
	}`))
	if err != nil {
		t.Fatal(err)
	}

	if len(refs) != 2 {
		t.Fatalf("expected 2 process refs, got %d", len(refs))
	}

	for _, ref := range refs {
		if ref.Scope != "yao" {
			t.Errorf("expected scope yao, got %s", ref.Scope)
		}
		if !strings.HasPrefix(ref.ScriptPath, "scripts/yao/") {
			t.Errorf("expected scripts/yao/ prefix, got %s", ref.ScriptPath)
		}
	}
}

func TestScriptExtractionNonProcess(t *testing.T) {
	refs, err := ExtractProcessRefsFromBytes([]byte(`{
		"transport": "stdio",
		"command": "echo"
	}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(refs) != 0 {
		t.Error("expected no refs for non-process transport")
	}
}

func TestRewriteProcessRefs(t *testing.T) {
	original := []byte(`{"tools":{"search":"scripts.yao.rag.Search","index":"scripts.yao.rag.Index"}}`)
	rewritten := RewriteProcessRefs(original, "yao", "local")

	if !strings.Contains(string(rewritten), "scripts.local.rag.Search") {
		t.Error("expected rewritten search ref")
	}
	if !strings.Contains(string(rewritten), "scripts.local.rag.Index") {
		t.Error("expected rewritten index ref")
	}
	if strings.Contains(string(rewritten), "scripts.yao.") {
		t.Error("expected no remaining yao refs")
	}
}

func TestExtractScopeFromProcessRef(t *testing.T) {
	if s := ExtractScopeFromProcessRef("scripts.yao.rag.Search"); s != "yao" {
		t.Errorf("expected yao, got %s", s)
	}
	if s := ExtractScopeFromProcessRef("scripts.max.search.Do"); s != "max" {
		t.Errorf("expected max, got %s", s)
	}
	if s := ExtractScopeFromProcessRef("agents.robot.host"); s != "" {
		t.Errorf("expected empty for non-scripts ref, got %s", s)
	}
}

func TestScriptPathsFromFiles(t *testing.T) {
	files := map[string]string{
		"mcps/yao/rag-tools/rag.mcp.yao": "sha256-aaa",
		"scripts/yao/rag.ts":             "sha256-bbb",
		"scripts/yao/index.ts":           "sha256-ccc",
	}

	scripts := ScriptPathsFromFiles(files)
	if len(scripts) != 2 {
		t.Fatalf("expected 2 scripts, got %d", len(scripts))
	}
	if _, ok := scripts["scripts/yao/rag.ts"]; !ok {
		t.Error("expected scripts/yao/rag.ts")
	}
}
