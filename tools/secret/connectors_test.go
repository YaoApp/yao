package secret

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
	if err := setting.Init(); err != nil {
		t.Skipf("setting.Init failed: %v", err)
	}
	if err := llmprovider.Init(); err != nil {
		t.Skipf("llmprovider.Init failed (may need full env): %v", err)
	}
	if llmprovider.Global == nil {
		t.Skip("llmprovider.Global is nil after Init")
	}

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
	t.Logf("ConnectorsHandler returned %d roles", len(rolesMap))

	for name, v := range rolesMap {
		entry, ok := v.(map[string]any)
		if !ok {
			t.Errorf("role %q value is %T, expected map", name, v)
			continue
		}
		if _, has := entry["model"]; !has {
			t.Errorf("role %q missing 'model' field", name)
		}
		if _, has := entry["type"]; !has {
			t.Errorf("role %q missing 'type' field", name)
		}
		if _, has := entry["key"]; !has {
			t.Errorf("role %q missing 'key' field", name)
		}
	}
}

func TestConnectorsHandler_ReturnFormat(t *testing.T) {
	if err := setting.Init(); err != nil {
		t.Skipf("setting.Init failed: %v", err)
	}
	if err := llmprovider.Init(); err != nil {
		t.Skipf("llmprovider.Init failed: %v", err)
	}
	if llmprovider.Global == nil {
		t.Skip("llmprovider.Global is nil after Init")
	}

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
		if _, hasErr := parsed["error"]; hasErr {
			t.Skipf("ConnectorsHandler returned error (no roles configured): %s", parsed["error"])
		}
		t.Fatal("top-level key 'roles' missing from return format")
	}

	var roles map[string]map[string]interface{}
	if err := json.Unmarshal(rolesRaw, &roles); err != nil {
		t.Fatalf("'roles' is not map[string]object: %v", err)
	}

	requiredFields := []string{"model", "type", "key"}
	for name, entry := range roles {
		for _, f := range requiredFields {
			if _, ok := entry[f]; !ok {
				t.Errorf("role %q missing required field %q (jq compat)", name, f)
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
