package agent

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/llmprovider"
	"github.com/yaoapp/yao/setting"
	"github.com/yaoapp/yao/test"
	"google.golang.org/grpc/metadata"
)

func TestMain(m *testing.M) {
	test.Prepare(nil, config.Conf)
	defer test.Clean()
	os.Exit(m.Run())
}

// --- Pure function tests (no app environment needed) ---

func TestValidateID_Valid(t *testing.T) {
	valid := []string{
		"yao.slides",
		"smith.weather",
		"ns.agent-name",
		"a.b",
	}
	for _, id := range valid {
		if err := validateID(id); err != nil {
			t.Errorf("validateID(%q) unexpected error: %v", id, err)
		}
	}
}

func TestValidateID_Invalid(t *testing.T) {
	cases := []struct {
		id   string
		want string
	}{
		{"", "invalid id format"},
		{"nodot", "invalid id format"},
		{".leading", "invalid id format"},
		{"trailing.", "invalid id format"},
		{"a..b", "path traversal"},
		{"a/b", "dot notation"},
		{"a\\b", "dot notation"},
		{"ns/name.ext", "dot notation"},
	}
	for _, tc := range cases {
		err := validateID(tc.id)
		if err == nil {
			t.Errorf("validateID(%q) expected error containing %q, got nil", tc.id, tc.want)
			continue
		}
		if !contains(err.Error(), tc.want) {
			t.Errorf("validateID(%q) error = %q, want substring %q", tc.id, err.Error(), tc.want)
		}
	}
}

func TestIdToPath(t *testing.T) {
	cases := []struct {
		id   string
		want string
	}{
		{"yao.slides", "yao/slides"},
		{"smith.weather", "smith/weather"},
		{"ns.agent.extra", "ns/agent.extra"},
	}
	for _, tc := range cases {
		got := idToPath(tc.id)
		if got != tc.want {
			t.Errorf("idToPath(%q) = %q, want %q", tc.id, got, tc.want)
		}
	}
}

func TestSettingStr(t *testing.T) {
	m := map[string]interface{}{
		"key1": "value1",
		"key2": 42,
		"key3": nil,
	}

	if v := settingStr(m, "key1"); v != "value1" {
		t.Errorf("settingStr(key1) = %q, want %q", v, "value1")
	}
	if v := settingStr(m, "key2"); v != "" {
		t.Errorf("settingStr(key2) = %q, want empty (non-string)", v)
	}
	if v := settingStr(m, "key3"); v != "" {
		t.Errorf("settingStr(key3) = %q, want empty (nil value)", v)
	}
	if v := settingStr(m, "missing"); v != "" {
		t.Errorf("settingStr(missing) = %q, want empty", v)
	}
	if v := settingStr(nil, "any"); v != "" {
		t.Errorf("settingStr(nil map) = %q, want empty", v)
	}
}

func TestSanitizeCapabilities(t *testing.T) {
	caps := map[string]interface{}{
		"tool_calls": true,
		"streaming":  true,
		"key":        "sk-secret-123",
		"secret":     "my-secret",
		"token":      "bearer-xyz",
		"reasoning":  false,
	}
	result := sanitizeCapabilities(caps)
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map, got %T", result)
	}
	if _, has := m["key"]; has {
		t.Error("sanitizeCapabilities should remove 'key'")
	}
	if _, has := m["secret"]; has {
		t.Error("sanitizeCapabilities should remove 'secret'")
	}
	if _, has := m["token"]; has {
		t.Error("sanitizeCapabilities should remove 'token'")
	}
	if m["tool_calls"] != true {
		t.Error("sanitizeCapabilities should preserve 'tool_calls'")
	}
	if m["streaming"] != true {
		t.Error("sanitizeCapabilities should preserve 'streaming'")
	}
	if m["reasoning"] != false {
		t.Error("sanitizeCapabilities should preserve 'reasoning'")
	}
}

func TestSanitizeCapabilities_NonMap(t *testing.T) {
	result := sanitizeCapabilities("not-a-map")
	if result != "not-a-map" {
		t.Errorf("non-map input should be returned as-is, got %v", result)
	}
}

func TestSanitizeCapabilities_Nil(t *testing.T) {
	result := sanitizeCapabilities(nil)
	if m, ok := result.(map[string]interface{}); ok && m != nil {
		t.Errorf("nil input should yield nil map, got %v", m)
	}
}

func TestExtractWorkspaceID_WithMetadata(t *testing.T) {
	md := metadata.Pairs("x-workspace-id", "ws-abc-123")
	ctx := metadata.NewIncomingContext(context.Background(), md)
	proc := &process.Process{Context: ctx}

	id := extractWorkspaceID(proc)
	if id != "ws-abc-123" {
		t.Errorf("extractWorkspaceID = %q, want %q", id, "ws-abc-123")
	}
}

func TestExtractWorkspaceID_NoMetadata(t *testing.T) {
	proc := &process.Process{Context: context.Background()}
	id := extractWorkspaceID(proc)
	if id != "" {
		t.Errorf("extractWorkspaceID without metadata = %q, want empty", id)
	}
}

func TestExtractWorkspaceID_NilContext(t *testing.T) {
	proc := &process.Process{}
	id := extractWorkspaceID(proc)
	if id != "" {
		t.Errorf("extractWorkspaceID with nil context = %q, want empty", id)
	}
}

func TestExtractWorkspaceID_EmptyValue(t *testing.T) {
	md := metadata.Pairs("x-workspace-id", "")
	ctx := metadata.NewIncomingContext(context.Background(), md)
	proc := &process.Process{Context: ctx}

	id := extractWorkspaceID(proc)
	if id != "" {
		t.Errorf("extractWorkspaceID with empty value = %q, want empty", id)
	}
}

func TestExtractWorkspaceID_OtherKeys(t *testing.T) {
	md := metadata.Pairs("x-sandbox-id", "sb-123")
	ctx := metadata.NewIncomingContext(context.Background(), md)
	proc := &process.Process{Context: ctx}

	id := extractWorkspaceID(proc)
	if id != "" {
		t.Errorf("extractWorkspaceID with wrong key = %q, want empty", id)
	}
}

func TestSchemaJSON_NonEmpty(t *testing.T) {
	schemas := map[string][]byte{
		"ListSchemaJSON":       ListSchemaJSON,
		"DownloadSchemaJSON":   DownloadSchemaJSON,
		"ReferenceSchemaJSON":  ReferenceSchemaJSON,
		"DeploySchemaJSON":     DeploySchemaJSON,
		"ConnectorsSchemaJSON": ConnectorsSchemaJSON,
	}
	for name, data := range schemas {
		if len(data) == 0 {
			t.Errorf("%s is empty", name)
			continue
		}
		var parsed map[string]interface{}
		if err := json.Unmarshal(data, &parsed); err != nil {
			t.Errorf("%s is not valid JSON: %v", name, err)
			continue
		}
		if parsed["name"] == nil {
			t.Errorf("%s missing 'name' field", name)
		}
		if parsed["process"] == nil {
			t.Errorf("%s missing 'process' field", name)
		}
	}
}

// --- Integration tests (require test.Prepare via TestMain) ---

func TestListHandler_All(t *testing.T) {
	proc := &process.Process{Args: []interface{}{}}
	result := ListHandler(proc)
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map, got %T", result)
	}
	if errMsg, has := m["error"]; has {
		t.Fatalf("ListHandler returned error: %v", errMsg)
	}
	agents, ok := m["agents"]
	if !ok {
		t.Fatal("ListHandler result missing 'agents' key")
	}
	agentList, ok := agents.([]agentInfo)
	if !ok {
		t.Fatalf("agents field is %T, expected []agentInfo", agents)
	}
	if len(agentList) == 0 {
		t.Error("expected at least one agent in yao-dev-app")
	}
	for _, a := range agentList {
		if a.ID == "" {
			t.Error("agent ID should not be empty")
		}
		if !contains(a.ID, ".") {
			t.Errorf("agent ID %q should use dot notation", a.ID)
		}
	}
	t.Logf("ListHandler returned %d agents", len(agentList))
}

func TestListHandler_Namespace(t *testing.T) {
	proc := &process.Process{Args: []interface{}{"yaobots"}}
	result := ListHandler(proc)
	m := result.(map[string]interface{})
	if errMsg, has := m["error"]; has {
		t.Fatalf("ListHandler returned error: %v", errMsg)
	}
	agentList := m["agents"].([]agentInfo)
	for _, a := range agentList {
		if !hasPrefix(a.ID, "yaobots.") {
			t.Errorf("agent %q should be in yaobots namespace", a.ID)
		}
	}
	t.Logf("namespace 'yaobots': %d agents", len(agentList))
}

func TestListHandler_NonexistentNamespace(t *testing.T) {
	proc := &process.Process{Args: []interface{}{"nonexistent_ns_xyz"}}
	result := ListHandler(proc)
	m := result.(map[string]interface{})
	agentList := m["agents"].([]agentInfo)
	if len(agentList) != 0 {
		t.Errorf("expected 0 agents for nonexistent namespace, got %d", len(agentList))
	}
}

func TestListHandler_SkipsYaoInternal(t *testing.T) {
	proc := &process.Process{Args: []interface{}{}}
	result := ListHandler(proc)
	m := result.(map[string]interface{})
	agentList := m["agents"].([]agentInfo)
	for _, a := range agentList {
		if hasPrefix(a.ID, "__yao.") {
			t.Errorf("internal agent %q should be filtered out", a.ID)
		}
	}
}

func TestConnectorsHandler_NoProvider(t *testing.T) {
	saved := llmprovider.Global
	llmprovider.Global = nil
	defer func() { llmprovider.Global = saved }()

	proc := &process.Process{Args: []interface{}{}}
	result := ConnectorsHandler(proc)
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map, got %T", result)
	}
	errMsg, has := m["error"]
	if !has {
		t.Fatal("expected error when llmprovider.Global is nil")
	}
	if !contains(errMsg.(string), "not initialized") {
		t.Errorf("error = %q, want substring 'not initialized'", errMsg)
	}
}

func TestConnectorsHandler_WithProvider(t *testing.T) {
	if err := setting.Init(); err != nil {
		t.Skipf("setting.Init failed: %v", err)
	}
	if err := llmprovider.Init(); err != nil {
		t.Skipf("llmprovider.Init failed (may need full env): %v", err)
	}
	if llmprovider.Global == nil {
		t.Skip("llmprovider.Global is nil after Init")
	}

	proc := &process.Process{Args: []interface{}{}}
	result := ConnectorsHandler(proc)
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map, got %T", result)
	}
	if errMsg, has := m["error"]; has {
		t.Fatalf("ConnectorsHandler returned error: %v", errMsg)
	}
	t.Logf("ConnectorsHandler returned %d roles", len(m))
}

func TestDeployHandler_MissingID(t *testing.T) {
	proc := &process.Process{Args: []interface{}{""}}
	result := DeployHandler(proc)
	m := result.(map[string]interface{})
	if _, has := m["error"]; !has {
		t.Error("expected error for empty id")
	}
}

func TestDeployHandler_WrongNamespace(t *testing.T) {
	proc := &process.Process{Args: []interface{}{"yao.slides"}}
	result := DeployHandler(proc)
	m := result.(map[string]interface{})
	if m["status"] != "error" {
		t.Errorf("expected status 'error' for non-smith namespace, got %v", m["status"])
	}
	msg, _ := m["message"].(string)
	if !contains(msg, "smith") {
		t.Errorf("error message should mention 'smith', got %q", msg)
	}
}

func TestDeployHandler_InvalidID(t *testing.T) {
	cases := []string{"smith/bad", "a..b", "onlyname"}
	for _, id := range cases {
		proc := &process.Process{Args: []interface{}{id}}
		result := DeployHandler(proc)
		m := result.(map[string]interface{})
		if _, has := m["error"]; !has {
			t.Errorf("DeployHandler(%q) expected error", id)
		}
	}
}

func TestDownloadHandler_MissingID(t *testing.T) {
	proc := &process.Process{Args: []interface{}{""}}
	result := DownloadHandler(proc)
	m := result.(map[string]interface{})
	if _, has := m["error"]; !has {
		t.Error("expected error for empty id")
	}
}

func TestDownloadHandler_InvalidID(t *testing.T) {
	cases := []string{"no/slash", "a..b", ""}
	for _, id := range cases {
		proc := &process.Process{Args: []interface{}{id}}
		result := DownloadHandler(proc)
		m := result.(map[string]interface{})
		if _, has := m["error"]; !has {
			t.Errorf("DownloadHandler(%q) expected error", id)
		}
	}
}

func TestDownloadHandler_WrongNamespace(t *testing.T) {
	proc := &process.Process{Args: []interface{}{"yao.slides"}}
	result := DownloadHandler(proc)
	m := result.(map[string]interface{})
	errMsg, has := m["error"]
	if !has {
		t.Fatal("expected error for non-smith namespace")
	}
	if !contains(errMsg.(string), "smith") {
		t.Errorf("error = %q, want substring 'smith'", errMsg)
	}
	if !contains(errMsg.(string), "agent_reference") {
		t.Errorf("error = %q, should suggest agent_reference", errMsg)
	}
}

func TestDownloadHandler_MissingWorkspace(t *testing.T) {
	proc := &process.Process{
		Args:    []interface{}{"smith.test"},
		Context: context.Background(),
	}
	result := DownloadHandler(proc)
	m := result.(map[string]interface{})
	errMsg, has := m["error"]
	if !has {
		t.Fatal("expected error when workspace_id is missing")
	}
	if !contains(errMsg.(string), "workspace_id") {
		t.Errorf("error = %q, want substring 'workspace_id'", errMsg)
	}
}

func TestReferenceHandler_MissingID(t *testing.T) {
	proc := &process.Process{Args: []interface{}{""}}
	result := ReferenceHandler(proc)
	m := result.(map[string]interface{})
	if _, has := m["error"]; !has {
		t.Error("expected error for empty id")
	}
}

func TestReferenceHandler_InvalidID(t *testing.T) {
	cases := []string{"no/slash", "a..b", ""}
	for _, id := range cases {
		proc := &process.Process{Args: []interface{}{id}}
		result := ReferenceHandler(proc)
		m := result.(map[string]interface{})
		if _, has := m["error"]; !has {
			t.Errorf("ReferenceHandler(%q) expected error", id)
		}
	}
}

func TestReferenceHandler_MissingWorkspace(t *testing.T) {
	proc := &process.Process{
		Args:    []interface{}{"yao.slides"},
		Context: context.Background(),
	}
	result := ReferenceHandler(proc)
	m := result.(map[string]interface{})
	errMsg, has := m["error"]
	if !has {
		t.Fatal("expected error when workspace_id is missing")
	}
	if !contains(errMsg.(string), "workspace_id") {
		t.Errorf("error = %q, want substring 'workspace_id'", errMsg)
	}
}

func TestDeployHandler_MissingWorkspace(t *testing.T) {
	proc := &process.Process{
		Args:    []interface{}{"smith.test"},
		Context: context.Background(),
	}
	result := DeployHandler(proc)
	m := result.(map[string]interface{})
	errMsg, has := m["error"]
	if !has {
		t.Fatal("expected error when workspace_id is missing")
	}
	if !contains(errMsg.(string), "workspace_id") {
		t.Errorf("error = %q, want substring 'workspace_id'", errMsg)
	}
}

// --- extractLocale tests ---

func TestExtractLocale_WithMetadata(t *testing.T) {
	md := metadata.Pairs("x-locale", "zh-cn")
	ctx := metadata.NewIncomingContext(context.Background(), md)
	proc := &process.Process{Context: ctx}

	locale := extractLocale(proc)
	if locale != "zh-cn" {
		t.Errorf("extractLocale = %q, want %q", locale, "zh-cn")
	}
}

func TestExtractLocale_UpperCase(t *testing.T) {
	md := metadata.Pairs("x-locale", "ZH-CN")
	ctx := metadata.NewIncomingContext(context.Background(), md)
	proc := &process.Process{Context: ctx}

	locale := extractLocale(proc)
	if locale != "zh-cn" {
		t.Errorf("extractLocale = %q, want %q (should lowercase)", locale, "zh-cn")
	}
}

func TestExtractLocale_NoMetadata(t *testing.T) {
	proc := &process.Process{Context: context.Background()}
	locale := extractLocale(proc)
	if locale != "en-us" {
		t.Errorf("extractLocale without metadata = %q, want default %q", locale, "en-us")
	}
}

func TestExtractLocale_NilContext(t *testing.T) {
	proc := &process.Process{}
	locale := extractLocale(proc)
	if locale != "en-us" {
		t.Errorf("extractLocale with nil context = %q, want default %q", locale, "en-us")
	}
}

func TestExtractLocale_EmptyValue(t *testing.T) {
	md := metadata.Pairs("x-locale", "")
	ctx := metadata.NewIncomingContext(context.Background(), md)
	proc := &process.Process{Context: ctx}

	locale := extractLocale(proc)
	if locale != "en-us" {
		t.Errorf("extractLocale with empty value = %q, want default %q", locale, "en-us")
	}
}

// --- helpers ---

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchSubstring(s, substr)
}

func searchSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}
