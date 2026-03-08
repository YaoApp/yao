package sandbox_test

import (
	"testing"

	"github.com/yaoapp/yao/config"
	sandbox "github.com/yaoapp/yao/sandbox/v2"
)

func TestBuildGRPCEnvLocal(t *testing.T) {
	config.Conf.GRPC.Port = 9099
	env := sandbox.BuildGRPCEnv("local", "", "sb-001")

	if env["YAO_SANDBOX_ID"] != "sb-001" {
		t.Errorf("YAO_SANDBOX_ID = %q", env["YAO_SANDBOX_ID"])
	}
	if _, ok := env["YAO_TOKEN"]; ok {
		t.Error("YAO_TOKEN should not be set by BuildGRPCEnv")
	}
	if env["YAO_GRPC_ADDR"] != "host.docker.internal:9099" {
		t.Errorf("YAO_GRPC_ADDR = %q, want host.docker.internal:9099", env["YAO_GRPC_ADDR"])
	}
}

func TestBuildGRPCEnvDirect(t *testing.T) {
	config.Conf.GRPC.Port = 9099
	env := sandbox.BuildGRPCEnv("direct", "tai://gpu-server", "sb-002")

	if env["YAO_GRPC_ADDR"] != "gpu-server:19100" {
		t.Errorf("YAO_GRPC_ADDR = %q, want gpu-server:19100", env["YAO_GRPC_ADDR"])
	}
}

func TestBuildGRPCEnvTunnel(t *testing.T) {
	config.Conf.GRPC.Port = 9099
	env := sandbox.BuildGRPCEnv("tunnel", "tunnel://relay.example.com", "sb-003")

	if env["YAO_GRPC_ADDR"] != "127.0.0.1:9099" {
		t.Errorf("YAO_GRPC_ADDR = %q, want 127.0.0.1:9099", env["YAO_GRPC_ADDR"])
	}
}
