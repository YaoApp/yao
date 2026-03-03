package registry_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/yaoapp/yao/registry"
	"github.com/yaoapp/yao/registry/testdata"
)

const (
	testScope = "@yaoagents"
)

func serverURL() string {
	if u := os.Getenv("YAO_REGISTRY_URL"); u != "" {
		return u
	}
	return "http://localhost:8080"
}

func newClient() *registry.Client {
	user := os.Getenv("YAO_REGISTRY_USER")
	pass := os.Getenv("YAO_REGISTRY_PASS")
	if user == "" {
		user = "yaoagents"
	}
	if pass == "" {
		pass = "yaoagents"
	}
	return registry.New(serverURL(),
		registry.WithAuth(user, pass),
	)
}

func newPublicClient() *registry.Client {
	return registry.New(serverURL())
}

// cleanup removes a version and ignores 404 errors.
func cleanup(c *registry.Client, pkgType, scope, name, version string) {
	c.DeleteVersion(pkgType, scope, name, version)
}

// --- Discovery ---

func TestDiscover(t *testing.T) {
	c := newPublicClient()
	info, err := c.Discover()
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}
	if info.Registry.Version == "" {
		t.Error("expected non-empty registry version")
	}
	if info.Registry.API == "" {
		t.Error("expected non-empty API path")
	}
	if len(info.Types) == 0 {
		t.Error("expected at least one supported type")
	}
}

func TestInfo(t *testing.T) {
	c := newPublicClient()
	info, err := c.Info()
	if err != nil {
		t.Fatalf("Info failed: %v", err)
	}
	if info.Name == "" {
		t.Error("expected non-empty name")
	}
	if info.Version == "" {
		t.Error("expected non-empty version")
	}
}

// --- Assistant CRUD ---

func TestAssistantCRUD(t *testing.T) {
	c := newClient()
	pkgType := "assistants"
	name := "test-assistant"

	zip10, err := testdata.BuildZip(&testdata.Manifest{
		Type:        "assistant",
		Scope:       testScope,
		Name:        name,
		Version:     "1.0.0",
		Description: "Test assistant for unit tests",
		Keywords:    []string{"test", "assistant"},
		License:     "MIT",
		Author:      &testdata.ManifestAuthor{Name: "Test", Email: "test@test.com"},
	}, map[string]string{
		"prompts/main.md": "You are a test assistant.",
	})
	if err != nil {
		t.Fatalf("BuildZip: %v", err)
	}

	defer cleanup(c, pkgType, testScope, name, "1.0.0")
	defer cleanup(c, pkgType, testScope, name, "1.1.0")

	// Push v1.0.0
	result, err := c.Push(pkgType, testScope, name, "1.0.0", zip10)
	if err != nil {
		t.Fatalf("Push 1.0.0 failed: %v", err)
	}
	if result.Version != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %s", result.Version)
	}
	if result.Digest == "" {
		t.Error("expected non-empty digest")
	}
	if result.Type != pkgType {
		t.Errorf("expected type %s, got %s", pkgType, result.Type)
	}

	// Push v1.1.0
	zip11, _ := testdata.BuildZip(&testdata.Manifest{
		Type:        "assistant",
		Scope:       testScope,
		Name:        name,
		Version:     "1.1.0",
		Description: "Updated test assistant",
	}, nil)
	result, err = c.Push(pkgType, testScope, name, "1.1.0", zip11)
	if err != nil {
		t.Fatalf("Push 1.1.0 failed: %v", err)
	}
	if result.Version != "1.1.0" {
		t.Errorf("expected version 1.1.0, got %s", result.Version)
	}

	// Get packument
	pack, err := c.GetPackument(pkgType, testScope, name)
	if err != nil {
		t.Fatalf("GetPackument failed: %v", err)
	}
	if pack.Type != pkgType {
		t.Errorf("expected type %s, got %s", pkgType, pack.Type)
	}
	if pack.Scope != testScope {
		t.Errorf("expected scope %s, got %s", testScope, pack.Scope)
	}
	if pack.Name != name {
		t.Errorf("expected name %s, got %s", name, pack.Name)
	}
	if len(pack.Versions) < 2 {
		t.Errorf("expected at least 2 versions, got %d", len(pack.Versions))
	}
	if pack.DistTags["latest"] != "1.1.0" {
		t.Errorf("expected latest=1.1.0, got %s", pack.DistTags["latest"])
	}

	// Get single version
	ver, err := c.GetVersion(pkgType, testScope, name, "1.0.0")
	if err != nil {
		t.Fatalf("GetVersion failed: %v", err)
	}
	if ver.Version != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %s", ver.Version)
	}
	if ver.Digest == "" {
		t.Error("expected non-empty digest in version detail")
	}

	// Pull
	data, digest, err := c.Pull(pkgType, testScope, name, "1.0.0")
	if err != nil {
		t.Fatalf("Pull failed: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty pull data")
	}
	if digest == "" {
		t.Error("expected non-empty digest header from pull")
	}

	// Pull by latest tag
	dataLatest, _, err := c.Pull(pkgType, testScope, name, "latest")
	if err != nil {
		t.Fatalf("Pull latest failed: %v", err)
	}
	if len(dataLatest) == 0 {
		t.Error("expected non-empty pull data for latest")
	}

	// Delete v1.0.0
	del, err := c.DeleteVersion(pkgType, testScope, name, "1.0.0")
	if err != nil {
		t.Fatalf("DeleteVersion 1.0.0 failed: %v", err)
	}
	if del.Deleted != "1.0.0" {
		t.Errorf("expected deleted=1.0.0, got %s", del.Deleted)
	}

	// Verify v1.0.0 is gone
	_, err = c.GetVersion(pkgType, testScope, name, "1.0.0")
	if err == nil {
		t.Error("expected error after deleting version 1.0.0")
	}

	// Delete v1.1.0
	_, err = c.DeleteVersion(pkgType, testScope, name, "1.1.0")
	if err != nil {
		t.Fatalf("DeleteVersion 1.1.0 failed: %v", err)
	}
}

// --- MCP Tool CRUD ---

func TestMCPToolCRUD(t *testing.T) {
	c := newClient()
	pkgType := "mcps"
	name := "test-mcp-tool"

	zipData, err := testdata.BuildZip(&testdata.Manifest{
		Type:        "mcp",
		Scope:       testScope,
		Name:        name,
		Version:     "2.0.0",
		Description: "Test MCP tool for SDK tests",
		Keywords:    []string{"test", "mcp"},
		Engines:     map[string]string{"yao": ">=0.10.0"},
	}, map[string]string{
		"tools.json":     `[{"name":"echo","description":"Echo tool"}]`,
		"scripts/run.js": "function main(args) { return args; }",
	})
	if err != nil {
		t.Fatalf("BuildZip: %v", err)
	}

	defer cleanup(c, pkgType, testScope, name, "2.0.0")

	result, err := c.Push(pkgType, testScope, name, "2.0.0", zipData)
	if err != nil {
		t.Fatalf("Push failed: %v", err)
	}
	if result.Scope != testScope {
		t.Errorf("expected scope %s, got %s", testScope, result.Scope)
	}

	// Pull the MCP tool
	data, _, err := c.Pull(pkgType, testScope, name, "2.0.0")
	if err != nil {
		t.Fatalf("Pull failed: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty data")
	}

	// Get version detail
	ver, err := c.GetVersion(pkgType, testScope, name, "2.0.0")
	if err != nil {
		t.Fatalf("GetVersion failed: %v", err)
	}
	if ver.Size <= 0 {
		t.Error("expected positive size")
	}

	// Delete
	_, err = c.DeleteVersion(pkgType, testScope, name, "2.0.0")
	if err != nil {
		t.Fatalf("DeleteVersion failed: %v", err)
	}
}

// --- Robot CRUD ---

func TestRobotCRUD(t *testing.T) {
	c := newClient()
	pkgType := "robots"
	name := "test-robot"

	zipData, err := testdata.BuildZip(&testdata.Manifest{
		Type:        "robot",
		Scope:       testScope,
		Name:        name,
		Version:     "0.5.0",
		Description: "Test robot for SDK tests",
	}, map[string]string{
		"robot.json": `{"name":"test-robot","model":"gpt-4o"}`,
	})
	if err != nil {
		t.Fatalf("BuildZip: %v", err)
	}

	defer cleanup(c, pkgType, testScope, name, "0.5.0")

	result, err := c.Push(pkgType, testScope, name, "0.5.0", zipData)
	if err != nil {
		t.Fatalf("Push failed: %v", err)
	}
	if result.Name != name {
		t.Errorf("expected name %s, got %s", name, result.Name)
	}

	// Packument
	pack, err := c.GetPackument(pkgType, testScope, name)
	if err != nil {
		t.Fatalf("GetPackument failed: %v", err)
	}
	if pack.DistTags["latest"] != "0.5.0" {
		t.Errorf("expected latest=0.5.0, got %s", pack.DistTags["latest"])
	}

	// Delete
	_, err = c.DeleteVersion(pkgType, testScope, name, "0.5.0")
	if err != nil {
		t.Fatalf("DeleteVersion failed: %v", err)
	}
}

// --- Dist-Tags ---

func TestDistTags(t *testing.T) {
	c := newClient()
	pkgType := "assistants"
	name := "test-tags"

	zip10, _ := testdata.BuildZip(&testdata.Manifest{
		Type: "assistant", Scope: testScope, Name: name, Version: "1.0.0",
	}, nil)
	zip20, _ := testdata.BuildZip(&testdata.Manifest{
		Type: "assistant", Scope: testScope, Name: name, Version: "2.0.0",
	}, nil)

	defer cleanup(c, pkgType, testScope, name, "1.0.0")
	defer cleanup(c, pkgType, testScope, name, "2.0.0")

	c.Push(pkgType, testScope, name, "1.0.0", zip10)
	c.Push(pkgType, testScope, name, "2.0.0", zip20)

	// Set a custom tag
	tagResult, err := c.SetTag(pkgType, testScope, name, "stable", "1.0.0")
	if err != nil {
		t.Fatalf("SetTag failed: %v", err)
	}
	if tagResult.Tag != "stable" {
		t.Errorf("expected tag=stable, got %s", tagResult.Tag)
	}
	if tagResult.Version != "1.0.0" {
		t.Errorf("expected version=1.0.0, got %s", tagResult.Version)
	}

	// Verify tag in packument
	pack, err := c.GetPackument(pkgType, testScope, name)
	if err != nil {
		t.Fatalf("GetPackument failed: %v", err)
	}
	if pack.DistTags["stable"] != "1.0.0" {
		t.Errorf("expected stable=1.0.0, got %s", pack.DistTags["stable"])
	}
	if pack.DistTags["latest"] != "2.0.0" {
		t.Errorf("expected latest=2.0.0, got %s", pack.DistTags["latest"])
	}

	// Pull by custom tag
	data, _, err := c.Pull(pkgType, testScope, name, "stable")
	if err != nil {
		t.Fatalf("Pull by stable tag failed: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected data from pull by tag")
	}

	// Delete the custom tag
	delTag, err := c.DeleteTag(pkgType, testScope, name, "stable")
	if err != nil {
		t.Fatalf("DeleteTag failed: %v", err)
	}
	if delTag.Deleted != "stable" {
		t.Errorf("expected deleted=stable, got %s", delTag.Deleted)
	}

	// Verify tag removed
	pack, _ = c.GetPackument(pkgType, testScope, name)
	if _, ok := pack.DistTags["stable"]; ok {
		t.Error("expected stable tag to be removed")
	}

	// Cleanup
	c.DeleteVersion(pkgType, testScope, name, "2.0.0")
	c.DeleteVersion(pkgType, testScope, name, "1.0.0")
}

// --- Dependencies ---

func TestDependencies(t *testing.T) {
	c := newClient()

	// Create a base MCP tool (no deps)
	mcpZip, _ := testdata.BuildZip(&testdata.Manifest{
		Type: "mcp", Scope: testScope, Name: "dep-base", Version: "1.0.0",
		Description: "Base MCP dependency",
	}, map[string]string{"tools.json": `[]`})

	// Create an assistant that depends on the MCP tool
	astZip, _ := testdata.BuildZip(&testdata.Manifest{
		Type: "assistant", Scope: testScope, Name: "dep-consumer", Version: "1.0.0",
		Description: "Assistant that depends on MCP tool",
		Dependencies: []testdata.ManifestDep{
			{Type: "mcp", Scope: testScope, Name: "dep-base", Version: "^1.0.0"},
		},
	}, map[string]string{"prompts/main.md": "Hello"})

	defer cleanup(c, "mcps", testScope, "dep-base", "1.0.0")
	defer cleanup(c, "assistants", testScope, "dep-consumer", "1.0.0")

	_, err := c.Push("mcps", testScope, "dep-base", "1.0.0", mcpZip)
	if err != nil {
		t.Fatalf("Push MCP dep-base failed: %v", err)
	}
	_, err = c.Push("assistants", testScope, "dep-consumer", "1.0.0", astZip)
	if err != nil {
		t.Fatalf("Push assistant dep-consumer failed: %v", err)
	}

	// Query dependencies
	deps, err := c.GetDependencies("assistants", testScope, "dep-consumer", "1.0.0", false)
	if err != nil {
		t.Fatalf("GetDependencies failed: %v", err)
	}
	if len(deps.Dependencies) == 0 {
		t.Error("expected at least 1 dependency")
	}

	// Query dependents of the MCP tool
	dependents, err := c.GetDependents("mcps", testScope, "dep-base")
	if err != nil {
		t.Fatalf("GetDependents failed: %v", err)
	}
	if len(dependents.Dependents) == 0 {
		t.Error("expected at least 1 dependent")
	}

	// Cleanup
	c.DeleteVersion("assistants", testScope, "dep-consumer", "1.0.0")
	c.DeleteVersion("mcps", testScope, "dep-base", "1.0.0")
}

// --- List & Search ---

func TestListAndSearch(t *testing.T) {
	c := newClient()
	pkgType := "assistants"
	name := "test-searchable"

	zipData, _ := testdata.BuildZip(&testdata.Manifest{
		Type:        "assistant",
		Scope:       testScope,
		Name:        name,
		Version:     "1.0.0",
		Description: "A searchable test assistant",
		Keywords:    []string{"searchable", "e2e"},
	}, nil)

	defer cleanup(c, pkgType, testScope, name, "1.0.0")

	_, err := c.Push(pkgType, testScope, name, "1.0.0", zipData)
	if err != nil {
		t.Fatalf("Push failed: %v", err)
	}

	// List assistants
	list, err := c.List(pkgType, "", "", 1, 20)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if list.Total < 1 {
		t.Errorf("expected at least 1 package, got %d", list.Total)
	}
	if list.Page != 1 {
		t.Errorf("expected page=1, got %d", list.Page)
	}

	// List with scope filter
	listScoped, err := c.List(pkgType, testScope, "", 1, 20)
	if err != nil {
		t.Fatalf("List with scope failed: %v", err)
	}
	if listScoped.Total < 1 {
		t.Errorf("expected at least 1 package in scope %s, got %d", testScope, listScoped.Total)
	}

	// Search
	search, err := c.Search("searchable", "", 1, 20)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if search.Total < 1 {
		t.Errorf("expected at least 1 search result, got %d", search.Total)
	}

	// Search with type filter
	searchTyped, err := c.Search("searchable", pkgType, 1, 20)
	if err != nil {
		t.Fatalf("Search with type failed: %v", err)
	}
	if searchTyped.Total < 1 {
		t.Errorf("expected at least 1 typed search result, got %d", searchTyped.Total)
	}

	// Cleanup
	c.DeleteVersion(pkgType, testScope, name, "1.0.0")
}

// --- Options coverage ---

func TestClientOptions(t *testing.T) {
	hc := &http.Client{Timeout: 10 * time.Second}
	c := registry.New(serverURL(),
		registry.WithAuth("u", "p"),
		registry.WithHTTPClient(hc),
		registry.WithTimeout(30*time.Second),
	)
	// Verify the client works (at least doesn't panic)
	_, err := c.Discover()
	if err != nil {
		t.Fatalf("Discover with custom options failed: %v", err)
	}
}

func TestAPIErrorString(t *testing.T) {
	err := &registry.APIError{StatusCode: 404, Message: "not found"}
	s := err.Error()
	if s != "registry: HTTP 404: not found" {
		t.Errorf("unexpected error string: %s", s)
	}
}

func TestNetworkError(t *testing.T) {
	c := registry.New("http://127.0.0.1:19999")

	_, err := c.Discover()
	if err == nil {
		t.Error("expected network error for Discover")
	}
	_, err = c.Info()
	if err == nil {
		t.Error("expected network error for Info")
	}
	_, err = c.List("assistants", "", "", 1, 20)
	if err == nil {
		t.Error("expected network error for List")
	}
	_, err = c.Search("q", "", 1, 20)
	if err == nil {
		t.Error("expected network error for Search")
	}
	_, err = c.GetPackument("assistants", "@x", "y")
	if err == nil {
		t.Error("expected network error for GetPackument")
	}
	_, err = c.GetVersion("assistants", "@x", "y", "1.0.0")
	if err == nil {
		t.Error("expected network error for GetVersion")
	}
	_, err = c.GetDependencies("assistants", "@x", "y", "1.0.0", true)
	if err == nil {
		t.Error("expected network error for GetDependencies")
	}
	_, err = c.GetDependents("assistants", "@x", "y")
	if err == nil {
		t.Error("expected network error for GetDependents")
	}
	_, _, err = c.Pull("assistants", "@x", "y", "1.0.0")
	if err == nil {
		t.Error("expected network error for Pull")
	}
	_, err = c.Push("assistants", "@x", "y", "1.0.0", []byte("data"))
	if err == nil {
		t.Error("expected network error for Push")
	}
	_, err = c.SetTag("assistants", "@x", "y", "t", "1.0.0")
	if err == nil {
		t.Error("expected network error for SetTag")
	}
	_, err = c.DeleteTag("assistants", "@x", "y", "t")
	if err == nil {
		t.Error("expected network error for DeleteTag")
	}
	_, err = c.DeleteVersion("assistants", "@x", "y", "1.0.0")
	if err == nil {
		t.Error("expected network error for DeleteVersion")
	}
}

func TestRecursiveDependencies(t *testing.T) {
	c := newClient()

	base, _ := testdata.BuildZip(&testdata.Manifest{
		Type: "mcp", Scope: testScope, Name: "recurse-base", Version: "1.0.0",
	}, nil)
	mid, _ := testdata.BuildZip(&testdata.Manifest{
		Type: "assistant", Scope: testScope, Name: "recurse-mid", Version: "1.0.0",
		Dependencies: []testdata.ManifestDep{
			{Type: "mcp", Scope: testScope, Name: "recurse-base", Version: "^1.0.0"},
		},
	}, nil)
	top, _ := testdata.BuildZip(&testdata.Manifest{
		Type: "robot", Scope: testScope, Name: "recurse-top", Version: "1.0.0",
		Dependencies: []testdata.ManifestDep{
			{Type: "assistant", Scope: testScope, Name: "recurse-mid", Version: "^1.0.0"},
		},
	}, nil)

	defer cleanup(c, "mcps", testScope, "recurse-base", "1.0.0")
	defer cleanup(c, "assistants", testScope, "recurse-mid", "1.0.0")
	defer cleanup(c, "robots", testScope, "recurse-top", "1.0.0")

	c.Push("mcps", testScope, "recurse-base", "1.0.0", base)
	c.Push("assistants", testScope, "recurse-mid", "1.0.0", mid)
	c.Push("robots", testScope, "recurse-top", "1.0.0", top)

	deps, err := c.GetDependencies("robots", testScope, "recurse-top", "1.0.0", true)
	if err != nil {
		t.Fatalf("GetDependencies recursive failed: %v", err)
	}
	if len(deps.Dependencies) == 0 {
		t.Error("expected recursive dependencies")
	}

	c.DeleteVersion("robots", testScope, "recurse-top", "1.0.0")
	c.DeleteVersion("assistants", testScope, "recurse-mid", "1.0.0")
	c.DeleteVersion("mcps", testScope, "recurse-base", "1.0.0")
}

// --- Error handling ---

func TestPushWithoutAuth(t *testing.T) {
	c := newPublicClient()
	zipData, _ := testdata.BuildZip(&testdata.Manifest{
		Type: "assistant", Scope: testScope, Name: "noauth", Version: "1.0.0",
	}, nil)

	_, err := c.Push("assistants", testScope, "noauth", "1.0.0", zipData)
	if err == nil {
		t.Fatal("expected error when pushing without auth")
	}
	apiErr, ok := err.(*registry.APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T: %v", err, err)
	}
	if apiErr.StatusCode != 401 {
		t.Errorf("expected 401, got %d", apiErr.StatusCode)
	}
}

func TestGetNonExistentPackage(t *testing.T) {
	c := newPublicClient()
	_, err := c.GetPackument("assistants", testScope, "does-not-exist")
	if err == nil {
		t.Fatal("expected error for non-existent package")
	}
	apiErr, ok := err.(*registry.APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}
	if apiErr.StatusCode != 404 {
		t.Errorf("expected 404, got %d", apiErr.StatusCode)
	}
}

func TestPullNonExistentVersion(t *testing.T) {
	c := newPublicClient()
	_, _, err := c.Pull("assistants", testScope, "does-not-exist", "9.9.9")
	if err == nil {
		t.Fatal("expected error for non-existent version pull")
	}
}

func TestInvalidType(t *testing.T) {
	c := newPublicClient()
	_, err := c.List("invalidtype", "", "", 1, 20)
	if err == nil {
		t.Fatal("expected error for invalid type")
	}
	apiErr, ok := err.(*registry.APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}
	if apiErr.StatusCode != 400 {
		t.Errorf("expected 400, got %d", apiErr.StatusCode)
	}
}

func TestDeleteNonExistentTag(t *testing.T) {
	c := newClient()
	name := "tag-noexist"
	zipData, _ := testdata.BuildZip(&testdata.Manifest{
		Type: "assistant", Scope: testScope, Name: name, Version: "1.0.0",
	}, nil)

	defer cleanup(c, "assistants", testScope, name, "1.0.0")

	c.Push("assistants", testScope, name, "1.0.0", zipData)

	_, err := c.DeleteTag("assistants", testScope, name, "nonexistent")
	if err == nil {
		t.Fatal("expected error deleting non-existent tag")
	}

	// Cannot delete latest tag
	_, err = c.DeleteTag("assistants", testScope, name, "latest")
	if err == nil {
		t.Fatal("expected error deleting latest tag")
	}

	c.DeleteVersion("assistants", testScope, name, "1.0.0")
}

// --- Push / Delete error responses ---

func TestPushDuplicateVersion(t *testing.T) {
	c := newClient()
	zipData, _ := testdata.BuildZip(&testdata.Manifest{
		Type: "assistant", Scope: testScope, Name: "dup-push", Version: "1.0.0",
	}, nil)

	defer cleanup(c, "assistants", testScope, "dup-push", "1.0.0")

	_, err := c.Push("assistants", testScope, "dup-push", "1.0.0", zipData)
	if err != nil {
		t.Fatalf("first push failed: %v", err)
	}

	_, err = c.Push("assistants", testScope, "dup-push", "1.0.0", zipData)
	if err == nil {
		t.Fatal("expected error on duplicate push")
	}
	apiErr, ok := err.(*registry.APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}
	if apiErr.StatusCode != 409 {
		t.Errorf("expected 409, got %d", apiErr.StatusCode)
	}

	c.DeleteVersion("assistants", testScope, "dup-push", "1.0.0")
}

func TestDeleteNonExistentVersion(t *testing.T) {
	c := newClient()
	_, err := c.DeleteVersion("assistants", testScope, "never-existed", "9.9.9")
	if err == nil {
		t.Fatal("expected error deleting non-existent version")
	}
	apiErr, ok := err.(*registry.APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}
	if apiErr.StatusCode != 404 {
		t.Errorf("expected 404, got %d", apiErr.StatusCode)
	}
}

func TestSetTagNonExistentPackage(t *testing.T) {
	c := newClient()
	_, err := c.SetTag("assistants", testScope, "no-such-pkg", "beta", "1.0.0")
	if err == nil {
		t.Fatal("expected error setting tag on non-existent package")
	}
	apiErr, ok := err.(*registry.APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}
	if apiErr.StatusCode != 404 {
		t.Errorf("expected 404, got %d", apiErr.StatusCode)
	}
}

func TestListWithQueryFilter(t *testing.T) {
	c := newClient()
	zipData, _ := testdata.BuildZip(&testdata.Manifest{
		Type: "mcp", Scope: testScope, Name: "list-query-mcp", Version: "1.0.0",
		Description: "filterable mcp tool",
	}, nil)

	defer cleanup(c, "mcps", testScope, "list-query-mcp", "1.0.0")

	c.Push("mcps", testScope, "list-query-mcp", "1.0.0", zipData)

	list, err := c.List("mcps", testScope, "filterable", 1, 10)
	if err != nil {
		t.Fatalf("List with scope+query failed: %v", err)
	}
	if list.Total < 1 {
		t.Errorf("expected at least 1 result, got %d", list.Total)
	}

	c.DeleteVersion("mcps", testScope, "list-query-mcp", "1.0.0")
}

// --- Release type CRUD ---

func TestReleaseCRUD(t *testing.T) {
	c := newClient()
	pkgType := "releases"
	name := "test-release"

	zipData, err := testdata.BuildZip(&testdata.Manifest{
		Type:        "release",
		Scope:       testScope,
		Name:        name,
		Version:     "1.0.0",
		Description: "Test release binary placeholder",
	}, map[string]string{
		"bin/yao": "#!/bin/sh\necho hello",
	})
	if err != nil {
		t.Fatalf("BuildZip: %v", err)
	}

	defer cleanup(c, pkgType, testScope, name, "1.0.0")

	result, err := c.Push(pkgType, testScope, name, "1.0.0", zipData)
	if err != nil {
		t.Fatalf("Push release failed: %v", err)
	}
	if result.Type != pkgType {
		t.Errorf("expected type %s, got %s", pkgType, result.Type)
	}

	// Get version detail
	ver, err := c.GetVersion(pkgType, testScope, name, "1.0.0")
	if err != nil {
		t.Fatalf("GetVersion failed: %v", err)
	}
	if ver.Version != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %s", ver.Version)
	}

	// Pull
	data, _, err := c.Pull(pkgType, testScope, name, "1.0.0")
	if err != nil {
		t.Fatalf("Pull failed: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty pull data")
	}

	// Delete
	_, err = c.DeleteVersion(pkgType, testScope, name, "1.0.0")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
}

// --- httptest-based edge-case coverage ---

func TestParseErrorNonJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("plain text error"))
	}))
	defer srv.Close()

	c := registry.New(srv.URL, registry.WithAuth("u", "p"))
	_, err := c.Push("assistants", "@t", "x", "1.0.0", []byte("data"))
	if err == nil {
		t.Fatal("expected error")
	}
	apiErr, ok := err.(*registry.APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}
	if apiErr.Message != "plain text error" {
		t.Errorf("expected plain text body in message, got %q", apiErr.Message)
	}
}

func TestInvalidBaseURL(t *testing.T) {
	c := registry.New("http://invalid\x7f:8080", registry.WithAuth("u", "p"))

	_, err := c.Push("assistants", "@t", "x", "1.0.0", []byte("zip"))
	if err == nil {
		t.Error("expected error from Push with invalid URL")
	}

	_, err = c.SetTag("assistants", "@t", "x", "beta", "1.0.0")
	if err == nil {
		t.Error("expected error from SetTag with invalid URL")
	}

	_, err = c.DeleteTag("assistants", "@t", "x", "beta")
	if err == nil {
		t.Error("expected error from DeleteTag with invalid URL")
	}

	_, err = c.DeleteVersion("assistants", "@t", "x", "1.0.0")
	if err == nil {
		t.Error("expected error from DeleteVersion with invalid URL")
	}
}

func TestPullNonOKStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"error":"forbidden"}`))
	}))
	defer srv.Close()

	c := registry.New(srv.URL)
	_, _, err := c.Pull("assistants", "@t", "x", "1.0.0")
	if err == nil {
		t.Fatal("expected error for forbidden pull")
	}
	apiErr, ok := err.(*registry.APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}
	if apiErr.StatusCode != 403 {
		t.Errorf("expected 403, got %d", apiErr.StatusCode)
	}
}

func TestSetTagDeleteTagError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"bad request"}`))
	}))
	defer srv.Close()

	c := registry.New(srv.URL, registry.WithAuth("u", "p"))

	_, err := c.SetTag("assistants", "@t", "x", "beta", "1.0.0")
	if err == nil {
		t.Error("expected error from SetTag")
	}

	_, err = c.DeleteTag("assistants", "@t", "x", "beta")
	if err == nil {
		t.Error("expected error from DeleteTag")
	}
}

func TestDeleteVersionError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		w.Write([]byte(`{"error":"conflict"}`))
	}))
	defer srv.Close()

	c := registry.New(srv.URL, registry.WithAuth("u", "p"))
	_, err := c.DeleteVersion("assistants", "@t", "x", "1.0.0")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestMalformedResponseBody(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		switch {
		case r.Method == http.MethodPut && r.URL.Path != "/tags/" && callCount <= 2:
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte("not-json"))
		case r.Method == http.MethodPut:
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("not-json"))
		case r.Method == http.MethodDelete:
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("not-json"))
		default:
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("not-json"))
		}
	}))
	defer srv.Close()

	c := registry.New(srv.URL, registry.WithAuth("u", "p"))

	_, err := c.Push("assistants", "@t", "x", "1.0.0", []byte("zip"))
	if err == nil {
		t.Error("expected decode error from Push")
	}

	_, err = c.SetTag("assistants", "@t", "x", "beta", "1.0.0")
	if err == nil {
		t.Error("expected decode error from SetTag")
	}

	_, err = c.DeleteTag("assistants", "@t", "x", "beta")
	if err == nil {
		t.Error("expected decode error from DeleteTag")
	}

	_, err = c.DeleteVersion("assistants", "@t", "x", "1.0.0")
	if err == nil {
		t.Error("expected decode error from DeleteVersion")
	}
}

func TestPullReadBodyError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "99999")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("short"))
	}))
	defer srv.Close()

	c := registry.New(srv.URL)
	data, _, err := c.Pull("assistants", "@t", "x", "1.0.0")
	if err != nil {
		t.Logf("got expected error: %v", err)
		return
	}
	if len(data) == 99999 {
		t.Error("expected incomplete read")
	}
}

func TestDoTransportError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	closedURL := srv.URL
	srv.Close()

	c := registry.New(closedURL, registry.WithAuth("u", "p"))

	_, err := c.Push("assistants", "@t", "x", "1.0.0", []byte("zip"))
	if err == nil {
		t.Error("expected transport error from Push")
	}

	_, _, err = c.Pull("assistants", "@t", "x", "1.0.0")
	if err == nil {
		t.Error("expected transport error from Pull")
	}

	_, err = c.SetTag("assistants", "@t", "x", "beta", "1.0.0")
	if err == nil {
		t.Error("expected transport error from SetTag")
	}

	_, err = c.DeleteTag("assistants", "@t", "x", "beta")
	if err == nil {
		t.Error("expected transport error from DeleteTag")
	}

	_, err = c.DeleteVersion("assistants", "@t", "x", "1.0.0")
	if err == nil {
		t.Error("expected transport error from DeleteVersion")
	}
}
