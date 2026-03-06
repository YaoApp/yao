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
	if _, ok := env["YAO_GRPC_TAI"]; ok {
		t.Error("local mode should not set YAO_GRPC_TAI")
	}
}

func TestBuildGRPCEnvRemote(t *testing.T) {
	pool := &sandbox.Pool{Name: "gpu", Addr: "tai://gpu-server"}
	env := sandbox.BuildGRPCEnv(pool, "sb-002", "access", "refresh", 9099)

	if env["YAO_GRPC_TAI"] != "enable" {
		t.Errorf("YAO_GRPC_TAI = %q, want enable", env["YAO_GRPC_TAI"])
	}
	if env["YAO_GRPC_ADDR"] != "gpu-server:9100" {
		t.Errorf("YAO_GRPC_ADDR = %q", env["YAO_GRPC_ADDR"])
	}
	if env["YAO_GRPC_UPSTREAM"] != "127.0.0.1:9099" {
		t.Errorf("YAO_GRPC_UPSTREAM = %q", env["YAO_GRPC_UPSTREAM"])
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
