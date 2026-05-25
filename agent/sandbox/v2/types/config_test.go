//go:build unit

package types_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/yaoapp/yao/agent/sandbox/v2/types"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

func TestMain(m *testing.M) {
	testprepare.MustLoadEnv()
	os.Exit(m.Run())
}

func TestRoleConnector_UnmarshalJSON_String(t *testing.T) {
	var rc types.RoleConnector
	if err := json.Unmarshal([]byte(`"thinking"`), &rc); err != nil {
		t.Fatal(err)
	}
	if rc.Connector != "thinking" {
		t.Errorf("Connector: got %q, want %q", rc.Connector, "thinking")
	}
	if rc.Override != "force" {
		t.Errorf("Override: got %q, want %q", rc.Override, "force")
	}
}

func TestRoleConnector_UnmarshalJSON_Object(t *testing.T) {
	var rc types.RoleConnector
	if err := json.Unmarshal([]byte(`{"connector":"thinking","override":"user"}`), &rc); err != nil {
		t.Fatal(err)
	}
	if rc.Connector != "thinking" || rc.Override != "user" {
		t.Errorf("got {%q, %q}", rc.Connector, rc.Override)
	}
}

func TestRoleConnector_UnmarshalJSON_ObjectDefaultForce(t *testing.T) {
	var rc types.RoleConnector
	if err := json.Unmarshal([]byte(`{"connector":"thinking"}`), &rc); err != nil {
		t.Fatal(err)
	}
	if rc.Override != "force" {
		t.Errorf("default override: got %q, want %q", rc.Override, "force")
	}
}

func TestRoleConnector_UnmarshalJSON_Invalid(t *testing.T) {
	var rc types.RoleConnector
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
	var rc types.RunnerConfig
	if err := json.Unmarshal([]byte(raw), &rc); err != nil {
		t.Fatal(err)
	}
	if rc.Name != "claude" {
		t.Errorf("Name: got %q", rc.Name)
	}
	if len(rc.Connectors) != 3 {
		t.Fatalf("Connectors: got %d, want 3", len(rc.Connectors))
	}
	if rc.Connectors["heavy"].Connector != "thinking" || rc.Connectors["heavy"].Override != "force" {
		t.Errorf("heavy: %+v", rc.Connectors["heavy"])
	}
	if rc.Connectors["light"].Connector != "fast" || rc.Connectors["light"].Override != "user" {
		t.Errorf("light: %+v", rc.Connectors["light"])
	}
}

func TestStringOrArray_String(t *testing.T) {
	var sa types.StringOrArray
	if err := json.Unmarshal([]byte(`"single"`), &sa); err != nil {
		t.Fatal(err)
	}
	if len(sa) != 1 || sa[0] != "single" {
		t.Errorf("got %v, want [single]", sa)
	}
}

func TestStringOrArray_Array(t *testing.T) {
	var sa types.StringOrArray
	if err := json.Unmarshal([]byte(`["a","b","c"]`), &sa); err != nil {
		t.Fatal(err)
	}
	if len(sa) != 3 {
		t.Fatalf("got %v, want 3 elements", sa)
	}
}

func TestStringOrArray_Invalid(t *testing.T) {
	var sa types.StringOrArray
	if err := json.Unmarshal([]byte(`123`), &sa); err == nil {
		t.Fatal("expected error for invalid type")
	}
}

func TestVNCConfig_Bool(t *testing.T) {
	var vc types.VNCConfig
	if err := json.Unmarshal([]byte(`true`), &vc); err != nil {
		t.Fatal(err)
	}
	if !vc.Enabled {
		t.Error("expected Enabled=true")
	}
}

func TestVNCConfig_Object(t *testing.T) {
	var vc types.VNCConfig
	if err := json.Unmarshal([]byte(`{"enabled":true,"password":"pw","resolution":"1080p","view_only":true}`), &vc); err != nil {
		t.Fatal(err)
	}
	if !vc.Enabled || vc.Password != "pw" || vc.Resolution != "1080p" || !vc.ViewOnly {
		t.Errorf("got %+v", vc)
	}
}

func TestVNCConfig_BoolFalse(t *testing.T) {
	var vc types.VNCConfig
	if err := json.Unmarshal([]byte(`false`), &vc); err != nil {
		t.Fatal(err)
	}
	if vc.Enabled {
		t.Error("expected Enabled=false")
	}
}

func TestPortList_IntArray(t *testing.T) {
	var pl types.PortList
	if err := json.Unmarshal([]byte(`[3000, 8080]`), &pl); err != nil {
		t.Fatal(err)
	}
	if len(pl) != 2 {
		t.Fatalf("got %d ports", len(pl))
	}
	if pl[0].Port != 3000 || pl[1].Port != 8080 {
		t.Errorf("ports: %+v", pl)
	}
}

func TestPortList_ObjectArray(t *testing.T) {
	var pl types.PortList
	if err := json.Unmarshal([]byte(`[{"port":3000,"host_port":13000,"protocol":"tcp"}]`), &pl); err != nil {
		t.Fatal(err)
	}
	if len(pl) != 1 || pl[0].Port != 3000 || pl[0].HostPort != 13000 || pl[0].Protocol != "tcp" {
		t.Errorf("got %+v", pl)
	}
}

func TestPortList_MixedArray_Error(t *testing.T) {
	var pl types.PortList
	err := json.Unmarshal([]byte(`[3000, {"port":8080,"host_port":18080}]`), &pl)
	if err == nil {
		t.Fatal("expected error for mixed int/object array, but got nil")
	}
}

func TestSandboxConfig_FullParse(t *testing.T) {
	raw := `{
		"runner": {"name": "claude"},
		"lifecycle": "session",
		"idle_timeout": "30m",
		"display_name": "Test Sandbox",
		"computer": {
			"image": "alpine:latest",
			"work_dir": "/workspace",
			"memory": "2GB",
			"cpus": 2.0,
			"vnc": true,
			"ports": [3000]
		},
		"environment": {"APP_ENV": "test"},
		"secrets": {"API_KEY": "secret123"}
	}`
	var cfg types.SandboxConfig
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Runner.Name != "claude" {
		t.Errorf("Runner.Name: got %q", cfg.Runner.Name)
	}
	if cfg.Lifecycle != "session" {
		t.Errorf("Lifecycle: got %q", cfg.Lifecycle)
	}
	if cfg.Computer.Image != "alpine:latest" {
		t.Errorf("Image: got %q", cfg.Computer.Image)
	}
	if !cfg.Computer.VNC.Enabled {
		t.Error("VNC should be enabled")
	}
	if len(cfg.Computer.Ports) != 1 {
		t.Errorf("Ports: got %d", len(cfg.Computer.Ports))
	}
	if cfg.Secrets["API_KEY"] == nil || cfg.Secrets["API_KEY"].Value != "secret123" {
		t.Errorf("Secrets: got %v", cfg.Secrets)
	}
}

func TestPrepareStep_Exec(t *testing.T) {
	raw := `{"action":"exec","cmd":"npm install","background":true}`
	var step types.PrepareStep
	if err := json.Unmarshal([]byte(raw), &step); err != nil {
		t.Fatal(err)
	}
	if step.Action != "exec" {
		t.Errorf("Action: got %q", step.Action)
	}
	if step.Cmd != "npm install" {
		t.Errorf("Cmd: got %q", step.Cmd)
	}
	if !step.Background {
		t.Error("Background should be true")
	}
}

func TestPrepareStep_Copy(t *testing.T) {
	raw := `{"action":"copy","src":"/local/path","dst":"/remote/path","once":true}`
	var step types.PrepareStep
	if err := json.Unmarshal([]byte(raw), &step); err != nil {
		t.Fatal(err)
	}
	if step.Action != "copy" {
		t.Errorf("Action: got %q", step.Action)
	}
	if step.Src != "/local/path" || step.Dst != "/remote/path" {
		t.Errorf("Src/Dst: got %q/%q", step.Src, step.Dst)
	}
	if !step.Once {
		t.Error("Once should be true")
	}
}
