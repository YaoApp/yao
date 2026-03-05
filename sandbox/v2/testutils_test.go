package sandbox_test

import (
	"context"
	"os"
	"testing"
	"time"

	sandbox "github.com/yaoapp/yao/sandbox/v2"
)

type poolConfig struct {
	Name string
	Addr string
}

// testPools returns all available pool configurations for dual-mode testing.
// Always includes "local"; includes "remote" when SANDBOX_TEST_REMOTE_ADDR is set.
func testPools() []poolConfig {
	pools := []poolConfig{
		{Name: "local", Addr: testLocalAddr()},
	}
	if addr := os.Getenv("SANDBOX_TEST_REMOTE_ADDR"); addr != "" {
		pools = append(pools, poolConfig{Name: "remote", Addr: addr})
	}
	return pools
}

func skipIfNoDocker(t *testing.T) {
	t.Helper()
	addr := testLocalAddr()
	if addr == "" {
		t.Skip("SANDBOX_TEST_LOCAL_ADDR not set, skipping Docker tests")
	}
}

func skipIfNoTai(t *testing.T) {
	t.Helper()
	if os.Getenv("SANDBOX_TEST_REMOTE_ADDR") == "" {
		t.Skip("SANDBOX_TEST_REMOTE_ADDR not set, skipping Tai proxy tests")
	}
}

func testLocalAddr() string {
	if addr := os.Getenv("SANDBOX_TEST_LOCAL_ADDR"); addr != "" {
		return addr
	}
	return "local"
}

func testImage() string {
	if img := os.Getenv("SANDBOX_TEST_IMAGE"); img != "" {
		return img
	}
	return "alpine:latest"
}

func setupManager(t *testing.T, pools ...sandbox.Pool) *sandbox.Manager {
	t.Helper()
	cfg := sandbox.Config{Pool: pools}
	if err := sandbox.Init(cfg); err != nil {
		t.Fatalf("Init: %v", err)
	}
	m := sandbox.M()
	t.Cleanup(func() {
		m.Close()
	})
	return m
}

func setupManagerForPool(t *testing.T, pc poolConfig, mutators ...func(*sandbox.Pool)) *sandbox.Manager {
	t.Helper()
	pool := sandbox.Pool{Name: pc.Name, Addr: pc.Addr}
	for _, fn := range mutators {
		fn(&pool)
	}
	return setupManager(t, pool)
}

func createTestBox(t *testing.T, m *sandbox.Manager, opts ...func(*sandbox.CreateOptions)) *sandbox.Box {
	t.Helper()
	co := sandbox.CreateOptions{
		Image: testImage(),
		Owner: "test-user",
	}
	for _, fn := range opts {
		fn(&co)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	box, err := m.Create(ctx, co)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	t.Cleanup(func() {
		m.Remove(context.Background(), box.ID())
	})
	return box
}
