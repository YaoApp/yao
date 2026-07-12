package sandbox

import (
	"testing"

	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/tai/registry"
	"github.com/yaoapp/yao/tai/types"
)

func setupTestRegistry(t *testing.T, nodes ...*registry.TaiNode) {
	t.Helper()
	reg := registry.NewForTest()
	for _, n := range nodes {
		reg.Register(n)
	}
	registry.SetGlobalForTest(reg)
	t.Cleanup(func() { registry.SetGlobalForTest(nil) })
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
	want := "host.tai.internal:19100"
	if got != want {
		t.Errorf("resolveGRPCAddr(tunnel, port=0) = %q, want %q", got, want)
	}
}

func TestResolveGRPCAddr_DirectMode(t *testing.T) {
	setupTestRegistry(t, &registry.TaiNode{
		TaiID: "node-direct",
		Mode:  "direct",
		Ports: types.Ports{GRPC: 19300},
	})

	m := newManager()
	got := m.resolveGRPCAddr("node-direct")
	want := "host.tai.internal:19300"
	if got != want {
		t.Errorf("resolveGRPCAddr(direct) = %q, want %q", got, want)
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
	registry.SetGlobalForTest(nil)
	t.Cleanup(func() { registry.SetGlobalForTest(nil) })

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
	registry.SetGlobalForTest(nil)
	t.Cleanup(func() { registry.SetGlobalForTest(nil) })

	m := newManager()
	b := &Box{nodeID: "missing", manager: m}
	cfg := &execConfig{}

	b.injectGRPCAddr(cfg)

	if cfg.Env != nil {
		t.Errorf("cfg.Env should remain nil when addr is unresolvable, got %v", cfg.Env)
	}
}
