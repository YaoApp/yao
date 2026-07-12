package sandbox_test

import (
	"testing"

	"github.com/yaoapp/yao/config"
	sandbox "github.com/yaoapp/yao/sandbox/v2"
)

func TestBuildGRPCEnv_Local(t *testing.T) {
	config.Conf.GRPC.Port = 9099
	env := sandbox.BuildGRPCEnv("local", 19100, "sb-001", "", "")

	want := "host.tai.internal:9099"
	if env["YAO_GRPC_ADDR"] != want {
		t.Errorf("YAO_GRPC_ADDR = %q, want %q", env["YAO_GRPC_ADDR"], want)
	}
}

func TestBuildGRPCEnv_Local_ZeroPort(t *testing.T) {
	config.Conf.GRPC.Port = 0
	env := sandbox.BuildGRPCEnv("local", 19100, "sb-zero", "", "")

	want := "host.tai.internal:9099"
	if env["YAO_GRPC_ADDR"] != want {
		t.Errorf("YAO_GRPC_ADDR = %q, want %q (default yao port)", env["YAO_GRPC_ADDR"], want)
	}
}

func TestBuildGRPCEnv_Tunnel(t *testing.T) {
	config.Conf.GRPC.Port = 9099
	env := sandbox.BuildGRPCEnv("tunnel", 19200, "sb-003", "", "")

	want := "host.tai.internal:19200"
	if env["YAO_GRPC_ADDR"] != want {
		t.Errorf("YAO_GRPC_ADDR = %q, want %q", env["YAO_GRPC_ADDR"], want)
	}
}

func TestBuildGRPCEnv_Tunnel_ZeroPort(t *testing.T) {
	config.Conf.GRPC.Port = 9099
	env := sandbox.BuildGRPCEnv("tunnel", 0, "sb-tzero", "", "")

	if _, ok := env["YAO_GRPC_ADDR"]; ok {
		t.Errorf("YAO_GRPC_ADDR should not be set when taiGRPCPort=0, got %q", env["YAO_GRPC_ADDR"])
	}
}

func TestBuildGRPCEnv_Cloud(t *testing.T) {
	config.Conf.GRPC.Port = 9099
	env := sandbox.BuildGRPCEnv("cloud", 19100, "sb-1", "chat-1", "ws-1")

	want := "host.tai.internal:19100"
	if env["YAO_GRPC_ADDR"] != want {
		t.Errorf("YAO_GRPC_ADDR = %q, want %q", env["YAO_GRPC_ADDR"], want)
	}
}

func TestBuildGRPCEnv_DefaultMode(t *testing.T) {
	config.Conf.GRPC.Port = 8888
	env := sandbox.BuildGRPCEnv("unknown", 19100, "sb-004", "", "")

	want := "host.tai.internal:8888"
	if env["YAO_GRPC_ADDR"] != want {
		t.Errorf("YAO_GRPC_ADDR = %q, want %q (fallback to yao port)", env["YAO_GRPC_ADDR"], want)
	}
}

func TestBuildGRPCEnv_DefaultMode_ZeroPort(t *testing.T) {
	config.Conf.GRPC.Port = 0
	env := sandbox.BuildGRPCEnv("", 0, "sb-allzero", "", "")

	want := "host.tai.internal:9099"
	if env["YAO_GRPC_ADDR"] != want {
		t.Errorf("YAO_GRPC_ADDR = %q, want %q (default fallback)", env["YAO_GRPC_ADDR"], want)
	}
}

func TestBuildGRPCEnv_SandboxID(t *testing.T) {
	cases := []struct {
		id   string
		want string
	}{
		{"sb-001", "sb-001"},
		{"", ""},
		{"sb-long-id-with-dashes-123", "sb-long-id-with-dashes-123"},
	}
	for _, tc := range cases {
		env := sandbox.BuildGRPCEnv("local", 0, tc.id, "", "")
		if env["YAO_SANDBOX_ID"] != tc.want {
			t.Errorf("SandboxID(%q): got %q, want %q", tc.id, env["YAO_SANDBOX_ID"], tc.want)
		}
	}
}

func TestBuildGRPCEnv_NoExtraKeys(t *testing.T) {
	config.Conf.GRPC.Port = 9099
	env := sandbox.BuildGRPCEnv("local", 0, "sb-check", "", "")

	allowed := map[string]bool{"YAO_GRPC_ADDR": true, "YAO_SANDBOX_ID": true}
	for k := range env {
		if !allowed[k] {
			t.Errorf("unexpected key %q in env", k)
		}
	}
	if len(env) != 2 {
		t.Errorf("expected 2 keys, got %d", len(env))
	}
}

func TestBuildGRPCEnv_ChatID_WorkspaceID(t *testing.T) {
	config.Conf.GRPC.Port = 9099
	env := sandbox.BuildGRPCEnv("local", 0, "sb-ctx", "chat-abc", "ws-xyz")

	if env["CTX_CHAT_ID"] != "chat-abc" {
		t.Errorf("CTX_CHAT_ID = %q, want %q", env["CTX_CHAT_ID"], "chat-abc")
	}
	if env["CTX_WORKSPACE_ID"] != "ws-xyz" {
		t.Errorf("CTX_WORKSPACE_ID = %q, want %q", env["CTX_WORKSPACE_ID"], "ws-xyz")
	}
	if env["YAO_SANDBOX_ID"] != "sb-ctx" {
		t.Errorf("YAO_SANDBOX_ID = %q, want %q", env["YAO_SANDBOX_ID"], "sb-ctx")
	}
}

func TestBuildGRPCEnv_EmptyChatID_NoKey(t *testing.T) {
	config.Conf.GRPC.Port = 9099
	env := sandbox.BuildGRPCEnv("local", 0, "sb-empty", "", "ws-only")

	if _, ok := env["CTX_CHAT_ID"]; ok {
		t.Error("CTX_CHAT_ID should not be present when chatID is empty")
	}
	if env["CTX_WORKSPACE_ID"] != "ws-only" {
		t.Errorf("CTX_WORKSPACE_ID = %q, want %q", env["CTX_WORKSPACE_ID"], "ws-only")
	}
}
