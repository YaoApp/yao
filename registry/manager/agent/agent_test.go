package agent

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

// buildTestZip builds a simple assistant .yao.zip for testing.
func buildTestZip(scope, name, version string, deps []testdata.ManifestDep, files map[string]string) []byte {
	zip, err := testdata.BuildZip(&testdata.Manifest{
		Type:         "assistant",
		Scope:        scope,
		Name:         name,
		Version:      version,
		Dependencies: deps,
	}, files)
	if err != nil {
		panic(err)
	}
	return zip
}

// buildMCPTestZip builds a simple MCP .yao.zip for testing.
func buildMCPTestZip(scope, name, version string) []byte {
	zip, err := testdata.BuildZip(&testdata.Manifest{
		Type:    "mcp",
		Scope:   scope,
		Name:    name,
		Version: version,
	}, map[string]string{
		"test.mcp.yao": `{"transport":"process"}`,
	})
	if err != nil {
		panic(err)
	}
	return zip
}

// mockRegistryServer creates a test HTTP server that serves pre-built zip packages.
func mockRegistryServer(packages map[string][]byte) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// well-known discovery
		if r.URL.Path == "/.well-known/yao-registry" {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"registry": map[string]string{"version": "1.0.0", "api": "/v1"},
				"types":    []string{"assistants", "mcps", "robots"},
			})
			return
		}

		// Pull: GET /v1/{type}/{scope}/{name}/{version}/pull
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

		// Delete: DELETE /v1/{type}/{scope}/{name}/{version}
		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
			return
		}

		// Push: PUT /v1/{type}/{scope}/{name}/{version}
		if r.Method == http.MethodPut {
			w.WriteHeader(http.StatusCreated)
			parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/v1/"), "/")
			result := map[string]string{
				"type":    parts[0],
				"scope":   parts[1],
				"name":    parts[2],
				"version": parts[3],
				"digest":  "sha256-pushed",
			}
			json.NewEncoder(w).Encode(result)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
}

func TestAddBasic(t *testing.T) {
	appRoot := t.TempDir()

	zip := buildTestZip("@test", "demo-agent", "1.0.0", nil, map[string]string{
		"package.yao": `{"name":"demo"}`,
		"prompts.yml": "You are a demo.",
	})

	srv := mockRegistryServer(map[string][]byte{
		"assistants/@test/demo-agent": zip,
	})
	defer srv.Close()

	client := registry.New(srv.URL, registry.WithAuth("u", "p"))
	mgr := New(client, appRoot, &common.AutoConfirmPrompter{})

	err := mgr.Add("@test/demo-agent", AddOptions{})
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	// Verify directory created
	destDir := filepath.Join(appRoot, "assistants", "test", "demo-agent")
	if _, err := os.Stat(destDir); err != nil {
		t.Fatalf("expected directory %s to exist", destDir)
	}

	// Verify files
	if _, err := os.Stat(filepath.Join(destDir, "package.yao")); err != nil {
		t.Error("expected package.yao")
	}
	if _, err := os.Stat(filepath.Join(destDir, "prompts.yml")); err != nil {
		t.Error("expected prompts.yml")
	}

	// Verify lockfile
	lf, err := common.LoadLockfile(appRoot)
	if err != nil {
		t.Fatal(err)
	}
	pkg, ok := lf.GetPackage("@test/demo-agent")
	if !ok {
		t.Fatal("expected @test/demo-agent in lockfile")
	}
	if pkg.Version != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %s", pkg.Version)
	}
	if pkg.Type != common.TypeAssistant {
		t.Errorf("expected type assistant, got %s", pkg.Type)
	}
	if len(pkg.Files) == 0 {
		t.Error("expected non-empty files hash")
	}
}

func TestAddAlreadyInstalled(t *testing.T) {
	appRoot := t.TempDir()

	zip := buildTestZip("@test", "dup", "1.0.0", nil, nil)
	srv := mockRegistryServer(map[string][]byte{
		"assistants/@test/dup": zip,
	})
	defer srv.Close()

	client := registry.New(srv.URL)
	mgr := New(client, appRoot, &common.AutoConfirmPrompter{})

	if err := mgr.Add("@test/dup", AddOptions{}); err != nil {
		t.Fatal(err)
	}

	// Second add should fail
	err := mgr.Add("@test/dup", AddOptions{})
	if err == nil {
		t.Fatal("expected error for duplicate install")
	}
	if !strings.Contains(err.Error(), "already installed") {
		t.Errorf("expected 'already installed' error, got: %v", err)
	}
}

func TestAddDirectoryConflict(t *testing.T) {
	appRoot := t.TempDir()

	// Create conflicting directory manually (not managed)
	os.MkdirAll(filepath.Join(appRoot, "assistants", "test", "conflict"), 0755)

	zip := buildTestZip("@test", "conflict", "1.0.0", nil, nil)
	srv := mockRegistryServer(map[string][]byte{
		"assistants/@test/conflict": zip,
	})
	defer srv.Close()

	client := registry.New(srv.URL)
	mgr := New(client, appRoot, &common.AutoConfirmPrompter{})

	err := mgr.Add("@test/conflict", AddOptions{})
	if err == nil {
		t.Fatal("expected conflict error")
	}
	if !strings.Contains(err.Error(), "not managed by registry") {
		t.Errorf("expected conflict error, got: %v", err)
	}
}

func TestAddWithDependencies(t *testing.T) {
	appRoot := t.TempDir()

	mcpZip := buildMCPTestZip("@test", "dep-mcp", "1.0.0")
	agentZip := buildTestZip("@test", "dep-agent", "1.0.0",
		[]testdata.ManifestDep{
			{Type: "mcp", Scope: "@test", Name: "dep-mcp", Version: "^1.0.0"},
		},
		map[string]string{"package.yao": `{"name":"dep-agent"}`},
	)

	srv := mockRegistryServer(map[string][]byte{
		"assistants/@test/dep-agent": agentZip,
		"mcps/@test/dep-mcp":         mcpZip,
	})
	defer srv.Close()

	client := registry.New(srv.URL)
	mgr := New(client, appRoot, &common.AutoConfirmPrompter{})

	err := mgr.Add("@test/dep-agent", AddOptions{})
	if err != nil {
		t.Fatalf("Add with deps failed: %v", err)
	}

	// Verify dependency was installed
	lf, _ := common.LoadLockfile(appRoot)
	if _, ok := lf.GetPackage("@test/dep-mcp"); !ok {
		t.Error("expected dependency @test/dep-mcp to be installed")
	}
}

func TestUpdateBasic(t *testing.T) {
	appRoot := t.TempDir()

	zipV1 := buildTestZip("@test", "updatable", "1.0.0", nil, map[string]string{
		"package.yao": `{"name":"updatable"}`,
		"prompts.yml": "Original prompt.",
	})
	zipV2 := buildTestZip("@test", "updatable", "2.0.0", nil, map[string]string{
		"package.yao": `{"name":"updatable","version":"2.0.0"}`,
		"prompts.yml": "Updated prompt.",
		"new-file.md": "New in v2.",
	})

	srv := mockRegistryServer(map[string][]byte{
		"assistants/@test/updatable": zipV2,
	})
	defer srv.Close()

	// First install v1 using the real zip
	srvV1 := mockRegistryServer(map[string][]byte{
		"assistants/@test/updatable": zipV1,
	})
	clientV1 := registry.New(srvV1.URL)
	mgrV1 := New(clientV1, appRoot, &common.AutoConfirmPrompter{})
	if err := mgrV1.Add("@test/updatable", AddOptions{}); err != nil {
		t.Fatal(err)
	}
	srvV1.Close()

	// Now update to v2
	client := registry.New(srv.URL)
	mgr := New(client, appRoot, &common.AutoConfirmPrompter{})

	err := mgr.Update("@test/updatable", UpdateOptions{})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// Verify version updated in lockfile
	lf, _ := common.LoadLockfile(appRoot)
	pkg, _ := lf.GetPackage("@test/updatable")
	if pkg.Version != "2.0.0" {
		t.Errorf("expected version 2.0.0, got %s", pkg.Version)
	}

	// Verify new file exists
	newFilePath := filepath.Join(appRoot, "assistants", "test", "updatable", "new-file.md")
	if _, err := os.Stat(newFilePath); err != nil {
		t.Error("expected new-file.md to be added")
	}
}

func TestUpdateLocallyModified(t *testing.T) {
	appRoot := t.TempDir()

	zipV1 := buildTestZip("@test", "modified", "1.0.0", nil, map[string]string{
		"package.yao": `{"name":"modified"}`,
		"prompts.yml": "Original.",
	})
	zipV2 := buildTestZip("@test", "modified", "2.0.0", nil, map[string]string{
		"package.yao": `{"name":"modified"}`,
		"prompts.yml": "Updated.",
	})

	// Install v1
	srvV1 := mockRegistryServer(map[string][]byte{
		"assistants/@test/modified": zipV1,
	})
	clientV1 := registry.New(srvV1.URL)
	mgrV1 := New(clientV1, appRoot, &common.AutoConfirmPrompter{})
	if err := mgrV1.Add("@test/modified", AddOptions{}); err != nil {
		t.Fatal(err)
	}
	srvV1.Close()

	// Modify prompts.yml locally
	promptsPath := filepath.Join(appRoot, "assistants", "test", "modified", "prompts.yml")
	os.WriteFile(promptsPath, []byte("My custom prompt."), 0644)

	// Update to v2
	srvV2 := mockRegistryServer(map[string][]byte{
		"assistants/@test/modified": zipV2,
	})
	defer srvV2.Close()
	clientV2 := registry.New(srvV2.URL)
	mgrV2 := New(clientV2, appRoot, &common.AutoConfirmPrompter{})

	err := mgrV2.Update("@test/modified", UpdateOptions{})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// prompts.yml should be preserved (locally modified)
	data, _ := os.ReadFile(promptsPath)
	if string(data) != "My custom prompt." {
		t.Errorf("expected locally modified prompts.yml preserved, got: %s", data)
	}

	// New version should be saved as .new
	newPath := promptsPath + ".new"
	if _, err := os.Stat(newPath); err != nil {
		t.Error("expected prompts.yml.new to exist")
	}
	newData, _ := os.ReadFile(newPath)
	if string(newData) != "Updated." {
		t.Errorf("expected .new file to contain new version, got: %s", newData)
	}
}

func TestUpdateNotInstalled(t *testing.T) {
	appRoot := t.TempDir()
	srv := mockRegistryServer(nil)
	defer srv.Close()

	client := registry.New(srv.URL)
	mgr := New(client, appRoot, &common.AutoConfirmPrompter{})

	err := mgr.Update("@test/nonexistent", UpdateOptions{})
	if err == nil {
		t.Fatal("expected error for not-installed package")
	}
	if !strings.Contains(err.Error(), "not installed") {
		t.Errorf("expected 'not installed' error, got: %v", err)
	}
}

func TestUpdateForkedPackage(t *testing.T) {
	appRoot := t.TempDir()

	// Set up a forked package in lockfile
	lf := &common.RegistryYao{
		Scope: "@local",
		Packages: map[string]common.PackageInfo{
			"@local/keeper": {
				Type:       common.TypeAssistant,
				Version:    "1.0.0",
				ForkedFrom: "@yao/keeper",
				Managed:    common.BoolPtr(false),
			},
		},
	}
	common.SaveLockfile(appRoot, lf)

	srv := mockRegistryServer(nil)
	defer srv.Close()
	client := registry.New(srv.URL)
	mgr := New(client, appRoot, &common.AutoConfirmPrompter{})

	err := mgr.Update("@local/keeper", UpdateOptions{})
	if err == nil {
		t.Fatal("expected error for forked package")
	}
	if !strings.Contains(err.Error(), "forked") {
		t.Errorf("expected 'forked' error, got: %v", err)
	}
}

func TestPushBasic(t *testing.T) {
	appRoot := t.TempDir()

	// Create assistant directory
	assistantDir := filepath.Join(appRoot, "assistants", "max", "my-agent")
	os.MkdirAll(assistantDir, 0755)
	os.WriteFile(filepath.Join(assistantDir, "package.yao"), []byte(`{"name":"my-agent"}`), 0644)
	os.WriteFile(filepath.Join(assistantDir, "prompts.yml"), []byte("test prompt"), 0644)

	srv := mockRegistryServer(nil)
	defer srv.Close()

	client := registry.New(srv.URL, registry.WithAuth("u", "p"))
	mgr := New(client, appRoot, &common.AutoConfirmPrompter{})

	err := mgr.Push("max.my-agent", PushOptions{Version: "1.0.0"})
	if err != nil {
		t.Fatalf("Push failed: %v", err)
	}
}

func TestPushLocalScope(t *testing.T) {
	appRoot := t.TempDir()

	srv := mockRegistryServer(nil)
	defer srv.Close()

	client := registry.New(srv.URL, registry.WithAuth("u", "p"))
	mgr := New(client, appRoot, &common.AutoConfirmPrompter{})

	err := mgr.Push("local.my-agent", PushOptions{Version: "1.0.0"})
	if err == nil {
		t.Fatal("expected error for @local push")
	}
	if !strings.Contains(err.Error(), "@local") {
		t.Errorf("expected @local rejection, got: %v", err)
	}
}

func TestPushForce(t *testing.T) {
	appRoot := t.TempDir()

	assistantDir := filepath.Join(appRoot, "assistants", "max", "my-agent")
	os.MkdirAll(assistantDir, 0755)
	os.WriteFile(filepath.Join(assistantDir, "package.yao"), []byte(`{"name":"my-agent"}`), 0644)

	var deleteCalled bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/yao-registry" {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"registry": map[string]string{"version": "1.0.0", "api": "/v1"},
				"types":    []string{"assistants"},
			})
			return
		}
		if r.Method == http.MethodDelete {
			deleteCalled = true
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
			return
		}
		if r.Method == http.MethodPut {
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{
				"type": "assistants", "scope": "@max",
				"name": "my-agent", "version": "1.0.0", "digest": "sha256-forced",
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := registry.New(srv.URL, registry.WithAuth("u", "p"))
	mgr := New(client, appRoot, &common.AutoConfirmPrompter{})

	err := mgr.Push("max.my-agent", PushOptions{Version: "1.0.0", Force: true})
	if err != nil {
		t.Fatalf("Force push failed: %v", err)
	}
	if !deleteCalled {
		t.Error("expected DELETE to be called before PUT when Force=true")
	}
}

func TestPushNoVersion(t *testing.T) {
	appRoot := t.TempDir()
	srv := mockRegistryServer(nil)
	defer srv.Close()

	client := registry.New(srv.URL)
	mgr := New(client, appRoot, &common.AutoConfirmPrompter{})

	err := mgr.Push("max.my-agent", PushOptions{})
	if err == nil {
		t.Fatal("expected error for missing version")
	}
}

func TestForkFromInstalled(t *testing.T) {
	appRoot := t.TempDir()

	// Set up an installed package
	assistantDir := filepath.Join(appRoot, "assistants", "yao", "keeper")
	os.MkdirAll(assistantDir, 0755)
	os.WriteFile(filepath.Join(assistantDir, "package.yao"), []byte(`{"name":"keeper"}`), 0644)
	os.WriteFile(filepath.Join(assistantDir, "prompts.yml"), []byte("keeper prompt"), 0644)

	lf := &common.RegistryYao{
		Scope: "@local",
		Packages: map[string]common.PackageInfo{
			"@yao/keeper": {
				Type:    common.TypeAssistant,
				Version: "2.0.0",
			},
		},
	}
	common.SaveLockfile(appRoot, lf)

	srv := mockRegistryServer(nil)
	defer srv.Close()
	client := registry.New(srv.URL)
	mgr := New(client, appRoot, &common.AutoConfirmPrompter{})

	err := mgr.Fork("@yao/keeper", ForkOptions{})
	if err != nil {
		t.Fatalf("Fork failed: %v", err)
	}

	// Verify forked directory
	forkDir := filepath.Join(appRoot, "assistants", "local", "keeper")
	if _, err := os.Stat(forkDir); err != nil {
		t.Fatal("expected forked directory")
	}

	data, _ := os.ReadFile(filepath.Join(forkDir, "package.yao"))
	if string(data) != `{"name":"keeper"}` {
		t.Errorf("expected copied content, got: %s", data)
	}

	// Verify lockfile
	lf, _ = common.LoadLockfile(appRoot)
	pkg, ok := lf.GetPackage("@local/keeper")
	if !ok {
		t.Fatal("expected @local/keeper in lockfile")
	}
	if pkg.ForkedFrom != "@yao/keeper" {
		t.Errorf("expected forked_from @yao/keeper, got %s", pkg.ForkedFrom)
	}
	if pkg.IsManaged() {
		t.Error("expected managed=false")
	}
	if pkg.Version != "2.0.0" {
		t.Errorf("expected version 2.0.0, got %s", pkg.Version)
	}
}

func TestForkFromRegistry(t *testing.T) {
	appRoot := t.TempDir()

	zip := buildTestZip("@test", "remote-agent", "3.0.0", nil, map[string]string{
		"package.yao": `{"name":"remote-agent"}`,
	})

	srv := mockRegistryServer(map[string][]byte{
		"assistants/@test/remote-agent": zip,
	})
	defer srv.Close()

	client := registry.New(srv.URL)
	mgr := New(client, appRoot, &common.AutoConfirmPrompter{})

	err := mgr.Fork("@test/remote-agent", ForkOptions{})
	if err != nil {
		t.Fatalf("Fork from registry failed: %v", err)
	}

	forkDir := filepath.Join(appRoot, "assistants", "local", "remote-agent")
	if _, err := os.Stat(forkDir); err != nil {
		t.Fatal("expected forked directory")
	}
}

func TestForkTargetExists(t *testing.T) {
	appRoot := t.TempDir()

	// Create target directory
	os.MkdirAll(filepath.Join(appRoot, "assistants", "local", "existing"), 0755)

	srv := mockRegistryServer(nil)
	defer srv.Close()
	client := registry.New(srv.URL)
	mgr := New(client, appRoot, &common.AutoConfirmPrompter{})

	err := mgr.Fork("@yao/existing", ForkOptions{})
	if err == nil {
		t.Fatal("expected error when target exists")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestForkCustomScope(t *testing.T) {
	appRoot := t.TempDir()

	zip := buildTestZip("@yao", "keeper", "1.0.0", nil, map[string]string{
		"package.yao": `{"name":"keeper"}`,
	})

	srv := mockRegistryServer(map[string][]byte{
		"assistants/@yao/keeper": zip,
	})
	defer srv.Close()

	client := registry.New(srv.URL)
	mgr := New(client, appRoot, &common.AutoConfirmPrompter{})

	err := mgr.Fork("@yao/keeper", ForkOptions{TargetScope: "max"})
	if err != nil {
		t.Fatalf("Fork to custom scope failed: %v", err)
	}

	forkDir := filepath.Join(appRoot, "assistants", "max", "keeper")
	if _, err := os.Stat(forkDir); err != nil {
		t.Fatal("expected directory in max scope")
	}

	lf, _ := common.LoadLockfile(appRoot)
	if _, ok := lf.GetPackage("@max/keeper"); !ok {
		t.Error("expected @max/keeper in lockfile")
	}
}

func TestScanDependencies(t *testing.T) {
	appRoot := t.TempDir()

	// Create assistant with MCP dependency
	assistantDir := filepath.Join(appRoot, "assistants", "max", "test-scan")
	os.MkdirAll(assistantDir, 0755)
	os.WriteFile(filepath.Join(assistantDir, "package.yao"), []byte(`{
		"name":"test-scan",
		"mcp": {
			"servers": [
				{"server_id": "yao.rag-tools"}
			]
		}
	}`), 0644)

	// Create scoped MCP directory so it gets picked up
	os.MkdirAll(filepath.Join(appRoot, "mcps", "yao", "rag-tools"), 0755)

	deps, err := ScanDependencies(assistantDir, appRoot)
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := deps["@yao/rag-tools"]; !ok {
		t.Error("expected @yao/rag-tools in scanned dependencies")
	}
}

func TestScanDependenciesSkipUnscoped(t *testing.T) {
	appRoot := t.TempDir()

	assistantDir := filepath.Join(appRoot, "assistants", "max", "test-local")
	os.MkdirAll(assistantDir, 0755)
	os.WriteFile(filepath.Join(assistantDir, "package.yao"), []byte(`{
		"name":"test-local",
		"mcp": {
			"servers": [
				{"server_id": "echo"}
			]
		}
	}`), 0644)

	deps, err := ScanDependencies(assistantDir, appRoot)
	if err != nil {
		t.Fatal(err)
	}

	// "echo" has no dot, so IDFromYaoID should fail and it should be skipped
	if len(deps) != 0 {
		t.Errorf("expected no dependencies for unscoped MCP, got %v", deps)
	}
}
