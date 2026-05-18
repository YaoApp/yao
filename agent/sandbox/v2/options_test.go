package sandboxv2_test

import (
	"testing"

	sandboxv2 "github.com/yaoapp/yao/agent/sandbox/v2"
	"github.com/yaoapp/yao/agent/sandbox/v2/types"
	infra "github.com/yaoapp/yao/sandbox/v2"
)

func TestParseMemory(t *testing.T) {
	cases := []struct {
		input string
		want  int64
		err   bool
	}{
		{"4GB", 4 * (1 << 30), false},
		{"4G", 4 * (1 << 30), false},
		{"4g", 4 * (1 << 30), false},
		{"512MB", 512 * (1 << 20), false},
		{"512M", 512 * (1 << 20), false},
		{"1024KB", 1024 * (1 << 10), false},
		{"1024K", 1024 * (1 << 10), false},
		{"1T", 1 * (1 << 40), false},
		{"1TB", 1 * (1 << 40), false},
		{"1024", 1024, false},
		{"", 0, false},
		{"invalid", 0, true},
		{"xyzGB", 0, true},
	}
	for _, tc := range cases {
		got, err := sandboxv2.ExportParseMemory(tc.input)
		if tc.err && err == nil {
			t.Errorf("parseMemory(%q): expected error", tc.input)
			continue
		}
		if !tc.err && err != nil {
			t.Errorf("parseMemory(%q): unexpected error: %v", tc.input, err)
			continue
		}
		if got != tc.want {
			t.Errorf("parseMemory(%q) = %d, want %d", tc.input, got, tc.want)
		}
	}
}

func TestBuildCreateOptions_Lifecycle(t *testing.T) {
	cases := []struct {
		lifecycle string
		want      infra.LifecyclePolicy
	}{
		{"oneshot", infra.OneShot},
		{"session", infra.Session},
		{"longrunning", infra.LongRunning},
		{"persistent", infra.Persistent},
		{"", infra.OneShot},
		{"unknown", infra.OneShot},
	}
	for _, tc := range cases {
		cfg := &types.SandboxConfig{Lifecycle: tc.lifecycle}
		opts, err := sandboxv2.BuildCreateOptions(cfg, "id", "owner", "ws")
		if err != nil {
			t.Errorf("lifecycle=%q: %v", tc.lifecycle, err)
			continue
		}
		if opts.Policy != tc.want {
			t.Errorf("lifecycle=%q: got %q, want %q", tc.lifecycle, opts.Policy, tc.want)
		}
	}
}

func TestBuildCreateOptions_Timeouts(t *testing.T) {
	cfg := &types.SandboxConfig{
		IdleTimeout: "5m",
		MaxLifetime: "2h",
		StopTimeout: "10s",
	}
	opts, err := sandboxv2.BuildCreateOptions(cfg, "id", "owner", "")
	if err != nil {
		t.Fatal(err)
	}
	if opts.IdleTimeout.Minutes() != 5 {
		t.Errorf("IdleTimeout: got %v", opts.IdleTimeout)
	}
	if opts.MaxLifetime.Hours() != 2 {
		t.Errorf("MaxLifetime: got %v", opts.MaxLifetime)
	}
	if opts.StopTimeout.Seconds() != 10 {
		t.Errorf("StopTimeout: got %v", opts.StopTimeout)
	}
}

func TestBuildCreateOptions_InvalidTimeout(t *testing.T) {
	cfg := &types.SandboxConfig{IdleTimeout: "not-a-duration"}
	_, err := sandboxv2.BuildCreateOptions(cfg, "id", "owner", "")
	if err == nil {
		t.Error("expected error for invalid idle_timeout")
	}
}

func TestBuildCreateOptions_VNC(t *testing.T) {
	cfg := &types.SandboxConfig{
		Computer: types.ComputerConfig{
			VNC: types.VNCConfig{
				Enabled:    true,
				Password:   "secret",
				Resolution: "1920x1080",
				ViewOnly:   true,
			},
		},
	}
	opts, err := sandboxv2.BuildCreateOptions(cfg, "id", "owner", "")
	if err != nil {
		t.Fatal(err)
	}
	if !opts.VNC {
		t.Error("VNC should be true")
	}
	if opts.Env["VNC_ENABLED"] != "true" {
		t.Error("VNC_ENABLED should be set")
	}
	if opts.Env["VNC_PASSWORD"] != "secret" {
		t.Error("VNC_PASSWORD should be set")
	}
	if opts.Env["VNC_RESOLUTION"] != "1920x1080" {
		t.Error("VNC_RESOLUTION should be set")
	}
	if opts.Env["VNC_VIEW_ONLY"] != "true" {
		t.Error("VNC_VIEW_ONLY should be set")
	}
}

func TestBuildCreateOptions_Ports(t *testing.T) {
	cfg := &types.SandboxConfig{
		Computer: types.ComputerConfig{
			Ports: []types.PortMapping{
				{Port: 3000, HostPort: 13000, Protocol: "tcp"},
				{Port: 8080},
			},
		},
	}
	opts, err := sandboxv2.BuildCreateOptions(cfg, "id", "owner", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(opts.Ports) != 2 {
		t.Fatalf("Ports: got %d, want 2", len(opts.Ports))
	}
	if opts.Ports[0].ContainerPort != 3000 || opts.Ports[0].HostPort != 13000 {
		t.Errorf("Port[0]: %+v", opts.Ports[0])
	}
}

func TestBuildCreateOptions_EnvMerge(t *testing.T) {
	cfg := &types.SandboxConfig{
		Environment: map[string]string{"A": "1", "B": "2"},
		Secrets:     map[string]string{"B": "secret", "C": "3"},
	}
	opts, err := sandboxv2.BuildCreateOptions(cfg, "id", "owner", "")
	if err != nil {
		t.Fatal(err)
	}
	if opts.Env["A"] != "1" {
		t.Error("env A should be 1")
	}
	if opts.Env["B"] != "secret" {
		t.Errorf("env B should be overridden by secret, got %q", opts.Env["B"])
	}
	if opts.Env["C"] != "3" {
		t.Error("env C should be 3")
	}
}

func TestBuildCreateOptions_Labels(t *testing.T) {
	cfg := &types.SandboxConfig{
		Labels: map[string]string{"team": "alpha"},
	}
	opts, err := sandboxv2.BuildCreateOptions(cfg, "id", "owner", "")
	if err != nil {
		t.Fatal(err)
	}
	if opts.Labels["team"] != "alpha" {
		t.Errorf("Labels: got %v", opts.Labels)
	}
}

func TestBuildCreateOptions_Memory(t *testing.T) {
	cfg := &types.SandboxConfig{
		Computer: types.ComputerConfig{Memory: "4GB"},
	}
	opts, err := sandboxv2.BuildCreateOptions(cfg, "id", "owner", "")
	if err != nil {
		t.Fatal(err)
	}
	if opts.Memory != 4*(1<<30) {
		t.Errorf("Memory: got %d, want %d", opts.Memory, 4*(1<<30))
	}
}

func TestBuildCreateOptions_InvalidMemory(t *testing.T) {
	cfg := &types.SandboxConfig{
		Computer: types.ComputerConfig{Memory: "invalid"},
	}
	_, err := sandboxv2.BuildCreateOptions(cfg, "id", "owner", "")
	if err == nil {
		t.Error("expected error for invalid memory")
	}
}

func TestBuildCreateOptions_BasicFields(t *testing.T) {
	cfg := &types.SandboxConfig{
		Computer: types.ComputerConfig{
			Image:   "my-image:v1",
			WorkDir: "/app",
			User:    "sandbox",
		},
		NodeID:      "node-1",
		DisplayName: "My Sandbox",
	}
	opts, err := sandboxv2.BuildCreateOptions(cfg, "sb-123", "owner-1", "ws-456")
	if err != nil {
		t.Fatal(err)
	}
	if opts.ID != "sb-123" {
		t.Errorf("ID: got %q", opts.ID)
	}
	if opts.Owner != "owner-1" {
		t.Errorf("Owner: got %q", opts.Owner)
	}
	if opts.Image != "my-image:v1" {
		t.Errorf("Image: got %q", opts.Image)
	}
	if opts.WorkDir != "/app" {
		t.Errorf("WorkDir: got %q", opts.WorkDir)
	}
	if opts.WorkspaceID != "ws-456" {
		t.Errorf("WorkspaceID: got %q", opts.WorkspaceID)
	}
	if opts.NodeID != "node-1" {
		t.Errorf("NodeID: got %q", opts.NodeID)
	}
	if opts.DisplayName != "My Sandbox" {
		t.Errorf("DisplayName: got %q", opts.DisplayName)
	}
}
