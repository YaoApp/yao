package sandbox

import (
	"testing"

	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/tai/registry"
	"github.com/yaoapp/yao/tai/types"
)

func setupTestRegistry(t *testing.T, nodes ...*registry.TaiNode) {
	t.Helper()
	prev := registry.Global()
	reg := registry.NewForTest()
	for _, n := range nodes {
		reg.Register(n)
	}
	registry.SetGlobalForTest(reg)
	t.Cleanup(func() { registry.SetGlobalForTest(prev) })
}

func TestResolveGRPCAddr_LocalMode(t *testing.T) {
	setupTestRegistry(t, &registry.TaiNode{
		TaiID: "node-local",
		Mode:  "local",
		Ports: types.Ports{GRPC: 19200},
	})
	config.Conf.GRPC.Port = 9099

	m := newManager()
	got := m.resolveGRPCAddr("node-local")
	want := "host.tai.internal:9099"
	if got != want {
		t.Errorf("resolveGRPCAddr(local) = %q, want %q", got, want)
	}
}

func TestResolveGRPCAddr_LocalMode_ZeroPort(t *testing.T) {
	setupTestRegistry(t, &registry.TaiNode{
		TaiID: "node-local-zero",
		Mode:  "local",
	})
	saved := config.Conf.GRPC.Port
	config.Conf.GRPC.Port = 0
	defer func() { config.Conf.GRPC.Port = saved }()

	m := newManager()
	got := m.resolveGRPCAddr("node-local-zero")
	want := "host.tai.internal:9099"
	if got != want {
		t.Errorf("resolveGRPCAddr(local, port=0) = %q, want %q", got, want)
	}
}

func TestResolveGRPCAddr_TunnelMode(t *testing.T) {
	setupTestRegistry(t, &registry.TaiNode{
		TaiID: "node-tunnel",
		Mode:  "tunnel",
		Ports: types.Ports{GRPC: 19200},
	})

	m := newManager()
	got := m.resolveGRPCAddr("node-tunnel")
	want := "host.tai.internal:19200"
	if got != want {
		t.Errorf("resolveGRPCAddr(tunnel) = %q, want %q", got, want)
	}
}

func TestResolveGRPCAddr_TunnelMode_ZeroPort(t *testing.T) {
	setupTestRegistry(t, &registry.TaiNode{
		TaiID: "node-tunnel-zero",
		Mode:  "tunnel",
		Ports: types.Ports{GRPC: 0},
	})

	m := newManager()
	got := m.resolveGRPCAddr("node-tunnel-zero")
	if got != "" {
		t.Errorf("resolveGRPCAddr(tunnel, port=0) = %q, want empty (no valid gRPC port)", got)
	}
}

func TestResolveGRPCAddr_CloudMode(t *testing.T) {
	setupTestRegistry(t, &registry.TaiNode{
		TaiID: "node-cloud",
		Mode:  "cloud",
		Ports: types.Ports{GRPC: 19400},
	})

	m := newManager()
	got := m.resolveGRPCAddr("node-cloud")
	want := "host.tai.internal:19400"
	if got != want {
		t.Errorf("resolveGRPCAddr(cloud) = %q, want %q", got, want)
	}
}

func TestResolveGRPCAddr_UnknownMode(t *testing.T) {
	setupTestRegistry(t, &registry.TaiNode{
		TaiID: "node-unknown",
		Mode:  "banana",
	})

	m := newManager()
	got := m.resolveGRPCAddr("node-unknown")
	if got != "" {
		t.Errorf("resolveGRPCAddr(unknown mode) = %q, want empty", got)
	}
}

func TestResolveGRPCAddr_NodeNotFound(t *testing.T) {
	setupTestRegistry(t)

	m := newManager()
	got := m.resolveGRPCAddr("nonexistent")
	if got != "" {
		t.Errorf("resolveGRPCAddr(missing node) = %q, want empty", got)
	}
}

func TestResolveGRPCAddr_NilRegistry(t *testing.T) {
	prev := registry.Global()
	registry.SetGlobalForTest(nil)
	t.Cleanup(func() { registry.SetGlobalForTest(prev) })

	m := newManager()
	got := m.resolveGRPCAddr("any")
	if got != "" {
		t.Errorf("resolveGRPCAddr(nil registry) = %q, want empty", got)
	}
}

func TestInjectGRPCAddr_SetsEnv(t *testing.T) {
	setupTestRegistry(t, &registry.TaiNode{
		TaiID: "node-inject",
		Mode:  "tunnel",
		Ports: types.Ports{GRPC: 19500},
	})

	m := newManager()
	b := &Box{nodeID: "node-inject", manager: m}
	cfg := &execConfig{}

	b.injectGRPCAddr(cfg)

	want := "host.tai.internal:19500"
	if cfg.Env == nil {
		t.Fatal("cfg.Env should be initialized")
	}
	if cfg.Env["YAO_GRPC_ADDR"] != want {
		t.Errorf("injected YAO_GRPC_ADDR = %q, want %q", cfg.Env["YAO_GRPC_ADDR"], want)
	}
}

func TestInjectGRPCAddr_DoesNotOverrideExplicit(t *testing.T) {
	setupTestRegistry(t, &registry.TaiNode{
		TaiID: "node-explicit",
		Mode:  "tunnel",
		Ports: types.Ports{GRPC: 19500},
	})

	m := newManager()
	b := &Box{nodeID: "node-explicit", manager: m}
	cfg := &execConfig{
		Env: map[string]string{"YAO_GRPC_ADDR": "custom:1234"},
	}

	b.injectGRPCAddr(cfg)

	if cfg.Env["YAO_GRPC_ADDR"] != "custom:1234" {
		t.Errorf("explicit value overridden: got %q, want %q", cfg.Env["YAO_GRPC_ADDR"], "custom:1234")
	}
}

func TestInjectGRPCAddr_NoopWhenUnresolvable(t *testing.T) {
	prev := registry.Global()
	registry.SetGlobalForTest(nil)
	t.Cleanup(func() { registry.SetGlobalForTest(prev) })

	m := newManager()
	b := &Box{nodeID: "missing", manager: m}
	cfg := &execConfig{}

	b.injectGRPCAddr(cfg)

	if cfg.Env != nil {
		t.Errorf("cfg.Env should remain nil when addr is unresolvable, got %v", cfg.Env)
	}
}

// --- resolveGRPCTLS tests ---

func TestResolveGRPCTLS_LocalWithTLS(t *testing.T) {
	setupTestRegistry(t, &registry.TaiNode{TaiID: "n1", Mode: "local"})
	saved := config.Conf.GRPC
	config.Conf.GRPC.Cert = "grpc-cert.pem"
	config.Conf.GRPC.Key = "grpc-key.pem"
	defer func() { config.Conf.GRPC = saved }()

	m := newManager()
	if !m.resolveGRPCTLS("n1") {
		t.Error("resolveGRPCTLS(local + TLS) should return true")
	}
}

func TestResolveGRPCTLS_LocalNoTLS(t *testing.T) {
	setupTestRegistry(t, &registry.TaiNode{TaiID: "n2", Mode: "local"})
	saved := config.Conf.GRPC
	config.Conf.GRPC.Cert = ""
	config.Conf.GRPC.Key = ""
	defer func() { config.Conf.GRPC = saved }()

	m := newManager()
	if m.resolveGRPCTLS("n2") {
		t.Error("resolveGRPCTLS(local, no TLS) should return false")
	}
}

func TestResolveGRPCTLS_TunnelWithTLS(t *testing.T) {
	setupTestRegistry(t, &registry.TaiNode{TaiID: "n3", Mode: "tunnel"})
	saved := config.Conf.GRPC
	config.Conf.GRPC.Cert = "grpc-cert.pem"
	config.Conf.GRPC.Key = "grpc-key.pem"
	defer func() { config.Conf.GRPC = saved }()

	m := newManager()
	if m.resolveGRPCTLS("n3") {
		t.Error("resolveGRPCTLS(tunnel) should return false (goes through Gateway)")
	}
}

func TestResolveGRPCTLS_CloudWithTLS(t *testing.T) {
	setupTestRegistry(t, &registry.TaiNode{TaiID: "n4", Mode: "cloud"})
	saved := config.Conf.GRPC
	config.Conf.GRPC.Cert = "grpc-cert.pem"
	config.Conf.GRPC.Key = "grpc-key.pem"
	defer func() { config.Conf.GRPC = saved }()

	m := newManager()
	if m.resolveGRPCTLS("n4") {
		t.Error("resolveGRPCTLS(cloud) should return false (goes through Gateway)")
	}
}

func TestResolveGRPCTLS_UnknownModeWithTLS(t *testing.T) {
	setupTestRegistry(t, &registry.TaiNode{TaiID: "n5", Mode: "custom"})
	saved := config.Conf.GRPC
	config.Conf.GRPC.Cert = "grpc-cert.pem"
	config.Conf.GRPC.Key = "grpc-key.pem"
	defer func() { config.Conf.GRPC = saved }()

	m := newManager()
	if !m.resolveGRPCTLS("n5") {
		t.Error("resolveGRPCTLS(unknown mode + TLS) should return true (direct to Yao)")
	}
}

func TestResolveGRPCTLS_NilRegistry(t *testing.T) {
	prev := registry.Global()
	registry.SetGlobalForTest(nil)
	t.Cleanup(func() { registry.SetGlobalForTest(prev) })

	m := newManager()
	if m.resolveGRPCTLS("any") {
		t.Error("resolveGRPCTLS(nil registry) should return false")
	}
}

func TestResolveGRPCTLS_NodeNotFound(t *testing.T) {
	setupTestRegistry(t)
	m := newManager()
	if m.resolveGRPCTLS("missing") {
		t.Error("resolveGRPCTLS(missing node) should return false")
	}
}

func TestInjectGRPCAddr_InjectsTLS(t *testing.T) {
	setupTestRegistry(t, &registry.TaiNode{
		TaiID: "node-tls",
		Mode:  "local",
	})
	saved := config.Conf.GRPC
	config.Conf.GRPC.Cert = "grpc-cert.pem"
	config.Conf.GRPC.Key = "grpc-key.pem"
	config.Conf.GRPC.Port = 9099
	defer func() { config.Conf.GRPC = saved }()

	m := newManager()
	b := &Box{nodeID: "node-tls", manager: m}
	cfg := &execConfig{}

	b.injectGRPCAddr(cfg)

	if cfg.Env["YAO_GRPC_TLS"] != "true" {
		t.Errorf("YAO_GRPC_TLS = %q, want %q", cfg.Env["YAO_GRPC_TLS"], "true")
	}
	if cfg.Env["YAO_GRPC_ADDR"] != "host.tai.internal:9099" {
		t.Errorf("YAO_GRPC_ADDR = %q, want %q", cfg.Env["YAO_GRPC_ADDR"], "host.tai.internal:9099")
	}
}

func TestInjectGRPCAddr_NoTLSForTunnel(t *testing.T) {
	setupTestRegistry(t, &registry.TaiNode{
		TaiID: "node-tunnel-tls",
		Mode:  "tunnel",
		Ports: types.Ports{GRPC: 19100},
	})
	saved := config.Conf.GRPC
	config.Conf.GRPC.Cert = "grpc-cert.pem"
	config.Conf.GRPC.Key = "grpc-key.pem"
	defer func() { config.Conf.GRPC = saved }()

	m := newManager()
	b := &Box{nodeID: "node-tunnel-tls", manager: m}
	cfg := &execConfig{}

	b.injectGRPCAddr(cfg)

	if _, ok := cfg.Env["YAO_GRPC_TLS"]; ok {
		t.Error("YAO_GRPC_TLS should not be set for tunnel mode")
	}
}

func TestInjectGRPCAddr_TLSNotOverrideExplicit(t *testing.T) {
	setupTestRegistry(t, &registry.TaiNode{
		TaiID: "node-tls-explicit",
		Mode:  "local",
	})
	saved := config.Conf.GRPC
	config.Conf.GRPC.Cert = "grpc-cert.pem"
	config.Conf.GRPC.Key = "grpc-key.pem"
	defer func() { config.Conf.GRPC = saved }()

	m := newManager()
	b := &Box{nodeID: "node-tls-explicit", manager: m}
	cfg := &execConfig{
		Env: map[string]string{"YAO_GRPC_TLS": "false"},
	}

	b.injectGRPCAddr(cfg)

	if cfg.Env["YAO_GRPC_TLS"] != "false" {
		t.Errorf("explicit YAO_GRPC_TLS overridden: got %q, want %q", cfg.Env["YAO_GRPC_TLS"], "false")
	}
}

// --- BuildGRPCEnv TLS tests ---

func TestBuildGRPCEnv_LocalTLS(t *testing.T) {
	saved := config.Conf.GRPC
	config.Conf.GRPC.Port = 9099
	config.Conf.GRPC.Cert = "grpc-cert.pem"
	config.Conf.GRPC.Key = "grpc-key.pem"
	defer func() { config.Conf.GRPC = saved }()

	env := BuildGRPCEnv("local", 0, "sb-1", "", "")
	if env["YAO_GRPC_TLS"] != "true" {
		t.Errorf("YAO_GRPC_TLS = %q, want %q", env["YAO_GRPC_TLS"], "true")
	}
}

func TestBuildGRPCEnv_LocalNoTLS(t *testing.T) {
	saved := config.Conf.GRPC
	config.Conf.GRPC.Port = 9099
	config.Conf.GRPC.Cert = ""
	config.Conf.GRPC.Key = ""
	defer func() { config.Conf.GRPC = saved }()

	env := BuildGRPCEnv("local", 0, "sb-2", "", "")
	if _, ok := env["YAO_GRPC_TLS"]; ok {
		t.Error("YAO_GRPC_TLS should not be set when TLS is not configured")
	}
}

func TestBuildGRPCEnv_TunnelNoTLS(t *testing.T) {
	saved := config.Conf.GRPC
	config.Conf.GRPC.Cert = "grpc-cert.pem"
	config.Conf.GRPC.Key = "grpc-key.pem"
	defer func() { config.Conf.GRPC = saved }()

	env := BuildGRPCEnv("tunnel", 19100, "sb-3", "", "")
	if _, ok := env["YAO_GRPC_TLS"]; ok {
		t.Error("YAO_GRPC_TLS should not be set for tunnel mode")
	}
}

func TestBuildGRPCEnv_DefaultTLS(t *testing.T) {
	saved := config.Conf.GRPC
	config.Conf.GRPC.Port = 9099
	config.Conf.GRPC.Cert = "grpc-cert.pem"
	config.Conf.GRPC.Key = "grpc-key.pem"
	defer func() { config.Conf.GRPC = saved }()

	env := BuildGRPCEnv("unknown-mode", 0, "sb-4", "", "")
	if env["YAO_GRPC_TLS"] != "true" {
		t.Errorf("YAO_GRPC_TLS = %q, want %q (default mode direct to Yao)", env["YAO_GRPC_TLS"], "true")
	}
}
