package types

import (
	"encoding/json"
	"testing"
)

func TestRoleConnector_UnmarshalJSON_String(t *testing.T) {
	var rc RoleConnector
	if err := json.Unmarshal([]byte(`"thinking"`), &rc); err != nil {
		t.Fatal(err)
	}
	if rc.Connector != "thinking" {
		t.Errorf("expected connector=thinking, got %s", rc.Connector)
	}
	if rc.Override != "force" {
		t.Errorf("expected override=force, got %s", rc.Override)
	}
}

func TestRoleConnector_UnmarshalJSON_Object(t *testing.T) {
	var rc RoleConnector
	if err := json.Unmarshal([]byte(`{"connector":"thinking","override":"user"}`), &rc); err != nil {
		t.Fatal(err)
	}
	if rc.Connector != "thinking" {
		t.Errorf("expected connector=thinking, got %s", rc.Connector)
	}
	if rc.Override != "user" {
		t.Errorf("expected override=user, got %s", rc.Override)
	}
}

func TestRoleConnector_UnmarshalJSON_ObjectDefaultForce(t *testing.T) {
	var rc RoleConnector
	if err := json.Unmarshal([]byte(`{"connector":"thinking"}`), &rc); err != nil {
		t.Fatal(err)
	}
	if rc.Connector != "thinking" {
		t.Errorf("expected connector=thinking, got %s", rc.Connector)
	}
	if rc.Override != "force" {
		t.Errorf("expected default override=force, got %s", rc.Override)
	}
}

func TestRoleConnector_UnmarshalJSON_InvalidJSON(t *testing.T) {
	var rc RoleConnector
	if err := json.Unmarshal([]byte(`123`), &rc); err == nil {
		t.Fatal("expected error for invalid JSON type")
	}
}

func TestRunnerConfig_UnmarshalJSON_WithConnectors(t *testing.T) {
	raw := `{
		"name": "claude",
		"mode": "interactive",
		"connectors": {
			"heavy": "thinking",
			"light": {"connector": "fast", "override": "user"},
			"vision": "multimodal"
		}
	}`

	var rc RunnerConfig
	if err := json.Unmarshal([]byte(raw), &rc); err != nil {
		t.Fatal(err)
	}

	if rc.Name != "claude" {
		t.Errorf("expected name=claude, got %s", rc.Name)
	}
	if rc.Mode != "interactive" {
		t.Errorf("expected mode=interactive, got %s", rc.Mode)
	}
	if len(rc.Connectors) != 3 {
		t.Fatalf("expected 3 connectors, got %d", len(rc.Connectors))
	}

	heavy := rc.Connectors["heavy"]
	if heavy.Connector != "thinking" || heavy.Override != "force" {
		t.Errorf("heavy: expected {thinking, force}, got {%s, %s}", heavy.Connector, heavy.Override)
	}

	light := rc.Connectors["light"]
	if light.Connector != "fast" || light.Override != "user" {
		t.Errorf("light: expected {fast, user}, got {%s, %s}", light.Connector, light.Override)
	}

	vision := rc.Connectors["vision"]
	if vision.Connector != "multimodal" || vision.Override != "force" {
		t.Errorf("vision: expected {multimodal, force}, got {%s, %s}", vision.Connector, vision.Override)
	}
}
