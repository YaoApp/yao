package sandbox_test

import (
	"context"
	"testing"
	"time"

	sandbox "github.com/yaoapp/yao/sandbox/v2"
)

func TestHeartbeatUpdates(t *testing.T) {
	skipIfNoDocker(t)

	for _, pc := range testPools() {
		t.Run(pc.Name, func(t *testing.T) {
			m := setupManagerForPool(t, pc)
			box := createTestBox(t, m)

			err := m.Heartbeat(box.ID(), true, 5)
			if err != nil {
				t.Fatalf("Heartbeat: %v", err)
			}

			info, err := box.Info(context.Background())
			if err != nil {
				t.Fatalf("Info: %v", err)
			}
			if info.ProcessCount != 5 {
				t.Errorf("ProcessCount = %d, want 5", info.ProcessCount)
			}
		})
	}
}

func TestHeartbeatUnknownBox(t *testing.T) {
	for _, pc := range testPools() {
		t.Run(pc.Name, func(t *testing.T) {
			m := setupManagerForPool(t, pc)
			err := m.Heartbeat("nonexistent", true, 1)
			if err != sandbox.ErrNotFound {
				t.Errorf("err = %v, want ErrNotFound", err)
			}
		})
	}
}

func TestIdleCleanup(t *testing.T) {
	skipIfNoDocker(t)

	for _, pc := range testPools() {
		t.Run(pc.Name, func(t *testing.T) {
			m := setupManagerForPool(t, pc, func(p *sandbox.Pool) {
				p.IdleTimeout = 1 * time.Second
			})

			ctx := context.Background()
			box, err := m.Create(ctx, sandbox.CreateOptions{
				Image:  testImage(),
				Owner:  "test-user",
				Policy: sandbox.Session,
			})
			if err != nil {
				t.Fatalf("Create: %v", err)
			}
			boxID := box.ID()

			time.Sleep(2 * time.Second)

			if err := m.Cleanup(ctx); err != nil {
				t.Fatalf("Cleanup: %v", err)
			}

			_, err = m.Get(ctx, boxID)
			if err != sandbox.ErrNotFound {
				t.Errorf("after idle cleanup, Get err = %v, want ErrNotFound", err)
			}
		})
	}
}

func TestStartRecovery(t *testing.T) {
	skipIfNoDocker(t)

	for _, pc := range testPools() {
		t.Run(pc.Name, func(t *testing.T) {
			pool := sandbox.Pool{Name: pc.Name, Addr: pc.Addr}

			m1 := setupManager(t, pool)
			box := createTestBox(t, m1)
			boxID := box.ID()

			cfg := sandbox.Config{Pool: []sandbox.Pool{pool}}
			if err := sandbox.Init(cfg); err != nil {
				t.Fatalf("Init2: %v", err)
			}
			m2 := sandbox.M()
			defer m2.Close()

			ctx := context.Background()
			if err := m2.Start(ctx); err != nil {
				t.Fatalf("Start: %v", err)
			}

			recovered, err := m2.Get(ctx, boxID)
			if err != nil {
				t.Fatalf("Get recovered box: %v", err)
			}
			if recovered.Owner() != "test-user" {
				t.Errorf("owner = %q, want %q", recovered.Owner(), "test-user")
			}

			m2.Remove(ctx, boxID)
		})
	}
}

func TestPersistentNotCleaned(t *testing.T) {
	skipIfNoDocker(t)

	for _, pc := range testPools() {
		t.Run(pc.Name, func(t *testing.T) {
			m := setupManagerForPool(t, pc, func(p *sandbox.Pool) {
				p.IdleTimeout = 1 * time.Second
			})

			box := createTestBox(t, m, func(co *sandbox.CreateOptions) {
				co.Policy = sandbox.Persistent
			})

			time.Sleep(2 * time.Second)

			ctx := context.Background()
			m.Cleanup(ctx)

			_, err := m.Get(ctx, box.ID())
			if err != nil {
				t.Errorf("persistent box should not be cleaned: %v", err)
			}
		})
	}
}
