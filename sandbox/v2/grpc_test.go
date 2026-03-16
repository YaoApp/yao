package sandbox_test

import (
	"testing"

	"github.com/yaoapp/yao/config"
	sandbox "github.com/yaoapp/yao/sandbox/v2"
)

func TestBuildGRPCEnvLocal(t *testing.T) {
	config.Conf.GRPC.Port = 9099
	env := sandbox.BuildGRPCEnv("local", 19100, "sb-001")

	if env["YAO_SANDBOX_ID"] != "sb-001" {
		t.Errorf("YAO_SANDBOX_ID = %q", env["YAO_SANDBOX_ID"])
	}
	if _, ok := env["YAO_TOKEN"]; ok {
		t.Error("YAO_TOKEN should not be set by BuildGRPCEnv")
	}
	want := "host.tai.internal:9099"
	if env["YAO_GRPC_ADDR"] != want {
		t.Errorf("YAO_GRPC_ADDR = %q, want %q", env["YAO_GRPC_ADDR"], want)
	}
}

func TestBuildGRPCEnvDirect(t *testing.T) {
	config.Conf.GRPC.Port = 9099
	env := sandbox.BuildGRPCEnv("direct", 19100, "sb-002")

	want := "host.tai.internal:19100"
	if env["YAO_GRPC_ADDR"] != want {
		t.Errorf("YAO_GRPC_ADDR = %q, want %q", env["YAO_GRPC_ADDR"], want)
	}
}

func TestBuildGRPCEnvDirectDefaultPort(t *testing.T) {
	config.Conf.GRPC.Port = 9099
	env := sandbox.BuildGRPCEnv("direct", 0, "sb-002")

	want := "host.tai.internal:19100"
	if env["YAO_GRPC_ADDR"] != want {
		t.Errorf("YAO_GRPC_ADDR = %q, want %q (default tai port)", env["YAO_GRPC_ADDR"], want)
	}
}

func TestBuildGRPCEnvTunnel(t *testing.T) {
	config.Conf.GRPC.Port = 9099
	env := sandbox.BuildGRPCEnv("tunnel", 19200, "sb-003")

	want := "host.tai.internal:19200"
	if env["YAO_GRPC_ADDR"] != want {
		t.Errorf("YAO_GRPC_ADDR = %q, want %q", env["YAO_GRPC_ADDR"], want)
	}
}

func TestBuildGRPCEnvUnknownMode(t *testing.T) {
	config.Conf.GRPC.Port = 8888
	env := sandbox.BuildGRPCEnv("unknown", 19100, "sb-004")

	want := "host.tai.internal:8888"
	if env["YAO_GRPC_ADDR"] != want {
		t.Errorf("YAO_GRPC_ADDR = %q, want %q (fallback to yao port)", env["YAO_GRPC_ADDR"], want)
	}
}
