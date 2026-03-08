package sandbox_test

import (
	"testing"

	sandbox "github.com/yaoapp/yao/sandbox/v2"
)

func TestBuildGRPCEnvLocal(t *testing.T) {
	pool := &sandbox.Pool{Name: "local", Addr: "local"}
	env := sandbox.BuildGRPCEnv(pool, "sb-001", "access-tok", "refresh-tok", 9099)

	if env["YAO_SANDBOX_ID"] != "sb-001" {
		t.Errorf("YAO_SANDBOX_ID = %q", env["YAO_SANDBOX_ID"])
	}
	if env["YAO_TOKEN"] != "access-tok" {
		t.Errorf("YAO_TOKEN = %q", env["YAO_TOKEN"])
	}
	if env["YAO_GRPC_ADDR"] != "127.0.0.1:9099" {
		t.Errorf("YAO_GRPC_ADDR = %q", env["YAO_GRPC_ADDR"])
	}
}

func TestBuildGRPCEnvRemote(t *testing.T) {
	pool := &sandbox.Pool{Name: "gpu", Addr: "tai://gpu-server"}
	env := sandbox.BuildGRPCEnv(pool, "sb-002", "access", "refresh", 9099)

	if env["YAO_GRPC_ADDR"] != "gpu-server:19100" {
		t.Errorf("YAO_GRPC_ADDR = %q, want gpu-server:19100", env["YAO_GRPC_ADDR"])
	}
}

func TestBuildGRPCEnvTunnel(t *testing.T) {
	pool := &sandbox.Pool{Name: "tunnel", Addr: "tunnel://relay.example.com"}
	env := sandbox.BuildGRPCEnv(pool, "sb-003", "access", "refresh", 9099)

	if env["YAO_GRPC_ADDR"] != "127.0.0.1:9099" {
		t.Errorf("YAO_GRPC_ADDR = %q, want 127.0.0.1:9099", env["YAO_GRPC_ADDR"])
	}
}

func TestCreateContainerTokens(t *testing.T) {
	access, refresh, err := sandbox.CreateContainerTokens("sb-001", "user1", nil)
	if err != nil {
		t.Fatalf("CreateContainerTokens: %v", err)
	}
	if len(access) != 64 {
		t.Errorf("access token len = %d, want 64 hex chars", len(access))
	}
	if len(refresh) != 64 {
		t.Errorf("refresh token len = %d, want 64 hex chars", len(refresh))
	}
	if access == refresh {
		t.Error("access and refresh tokens should be different")
	}
}
