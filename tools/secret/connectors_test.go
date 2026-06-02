package secret

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/gou/store"
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

func setupTestRoles(t *testing.T) {
	t.Helper()

	if err := setting.Init(); err != nil {
		t.Fatalf("setting.Init failed: %v", err)
	}
	if err := llmprovider.Init(); err != nil {
		t.Fatalf("llmprovider.Init failed: %v", err)
	}
	if llmprovider.Global == nil {
		t.Fatal("llmprovider.Global is nil after Init")
	}

	p := &llmprovider.Provider{
		Key:     "test-secret-connector",
		Name:    "Test Secret Connector",
		Type:    "openai",
		APIURL:  "https://api.openai.com",
		APIKey:  "sk-test-secret-key",
		Enabled: true,
		Models: []llmprovider.ModelInfo{
			{ID: "gpt-4o", Name: "GPT-4o", Capabilities: []string{"vision", "tool_calls", "streaming"}, Enabled: true},
		},
		Owner: llmprovider.ProviderOwner{Type: "system"},
	}
	if _, err := llmprovider.Global.Create(p); err != nil {
		t.Fatalf("failed to create test provider: %v", err)
	}

	if err := llmprovider.Global.SetDefaults(map[string]string{"default": p.Key}); err != nil {
		t.Fatalf("failed to set default roles: %v", err)
	}

	t.Cleanup(func() {
		s, _ := store.Get("__yao.store")
		if s != nil {
			s.Del("setting:*")
			s.Del("llmprovider:*")
		}
		c, _ := store.Get("__yao.cache")
		if c != nil {
			c.Del("setting:*")
			c.Del("llmprovider:*")
		}
	})
}

func TestConnectorsHandler_Unauthorized(t *testing.T) {
	md := metadata.Pairs("x-assistant-id", "yao.agent-smith")
	ctx := metadata.NewIncomingContext(context.Background(), md)

	proc := &process.Process{
		Context:    ctx,
		Authorized: nil,
	}
	result := ConnectorsHandler(proc)
	m, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T", result)
	}
	errMsg, has := m["error"]
	if !has {
		t.Fatal("expected error when Authorized is nil")
	}
	if errMsg != "unauthorized" {
		t.Errorf("error = %q, want %q", errMsg, "unauthorized")
	}
}

func TestConnectorsHandler_NoProvider(t *testing.T) {
	saved := llmprovider.Global
	llmprovider.Global = nil
	defer func() { llmprovider.Global = saved }()

	md := metadata.Pairs("x-assistant-id", "yao.agent-smith")
	ctx := metadata.NewIncomingContext(context.Background(), md)

	proc := &process.Process{
		Context: ctx,
		Authorized: &process.AuthorizedInfo{
			UserID: "test-user",
			TeamID: "test-team",
		},
	}
	result := ConnectorsHandler(proc)
	m, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T", result)
	}
	errMsg, has := m["error"]
	if !has {
		t.Fatal("expected error when llmprovider.Global is nil")
	}
	if s, ok := errMsg.(string); !ok || !containsStr(s, "not initialized") {
		t.Errorf("error = %v, want substring 'not initialized'", errMsg)
	}
}

func TestConnectorsHandler_WithProvider(t *testing.T) {
	setupTestRoles(t)

	md := metadata.Pairs("x-assistant-id", "yao.agent-smith")
	ctx := metadata.NewIncomingContext(context.Background(), md)

	proc := &process.Process{
		Context: ctx,
		Authorized: &process.AuthorizedInfo{
			UserID: "test-user",
			TeamID: "test-team",
		},
	}
	result := ConnectorsHandler(proc)
	m, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T", result)
	}
	if errMsg, has := m["error"]; has {
		t.Fatalf("ConnectorsHandler returned error: %v", errMsg)
	}

	roles, has := m["roles"]
	if !has {
		t.Fatal("result missing 'roles' key")
	}

	rolesMap, ok := roles.(map[string]any)
	if !ok {
		t.Fatalf("roles is %T, expected map[string]any", roles)
	}
	if len(rolesMap) == 0 {
		t.Fatal("expected at least 1 role, got 0")
	}

	defaultRole, has := rolesMap["default"]
	if !has {
		t.Fatal("expected 'default' role in roles map")
	}
	entry, ok := defaultRole.(map[string]any)
	if !ok {
		t.Fatalf("default role is %T, expected map[string]any", defaultRole)
	}
	if _, has := entry["model"]; !has {
		t.Error("role 'default' missing 'model' field")
	}
	if _, has := entry["type"]; !has {
		t.Error("role 'default' missing 'type' field")
	}
	if _, has := entry["key"]; !has {
		t.Error("role 'default' missing 'key' field")
	}
}

func TestConnectorsHandler_ReturnFormat(t *testing.T) {
	setupTestRoles(t)

	md := metadata.Pairs("x-assistant-id", "yao.agent-smith")
	ctx := metadata.NewIncomingContext(context.Background(), md)

	proc := &process.Process{
		Context: ctx,
		Authorized: &process.AuthorizedInfo{
			UserID: "test-user",
			TeamID: "test-team",
		},
	}
	result := ConnectorsHandler(proc)

	raw, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var parsed map[string]json.RawMessage
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("returned value is not valid JSON object: %v", err)
	}

	rolesRaw, has := parsed["roles"]
	if !has {
		if errRaw, hasErr := parsed["error"]; hasErr {
			t.Fatalf("ConnectorsHandler returned error: %s", errRaw)
		}
		t.Fatal("top-level key 'roles' missing from return format")
	}

	var roles map[string]map[string]interface{}
	if err := json.Unmarshal(rolesRaw, &roles); err != nil {
		t.Fatalf("'roles' is not map[string]object: %v", err)
	}

	if len(roles) == 0 {
		t.Fatal("expected at least 1 role in JSON output")
	}

	requiredFields := []string{"model", "type", "key"}
	for name, entry := range roles {
		for _, f := range requiredFields {
			if _, ok := entry[f]; !ok {
				t.Errorf("role %q missing required field %q", name, f)
			}
		}
	}
}

func TestConnectorsHandler_SchemaJSON(t *testing.T) {
	if len(ConnectorsSchemaJSON) == 0 {
		t.Fatal("ConnectorsSchemaJSON is empty")
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal(ConnectorsSchemaJSON, &parsed); err != nil {
		t.Fatalf("ConnectorsSchemaJSON is not valid JSON: %v", err)
	}
	if parsed["name"] != "secret_connectors" {
		t.Errorf("schema name = %v, want %q", parsed["name"], "secret_connectors")
	}
	if parsed["process"] != "tools.secret_connectors" {
		t.Errorf("schema process = %v, want %q", parsed["process"], "tools.secret_connectors")
	}
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
