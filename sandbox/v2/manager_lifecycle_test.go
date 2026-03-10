package sandbox_test

import (
	"context"
	"testing"
	"time"

	sandbox "github.com/yaoapp/yao/sandbox/v2"
)

func TestHeartbeatUpdates(t *testing.T) {
	skipIfNoDocker(t)

	for _, pc := range testNodes() {
		pc := pc
		t.Run(pc.Name, func(t *testing.T) {
			m := setupManagerForNode(t, &pc)
			box := createTestBox(t, m, pc)

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
	for _, pc := range testNodes() {
		pc := pc
		t.Run(pc.Name, func(t *testing.T) {
			m := setupManagerForNode(t, &pc)
			err := m.Heartbeat("nonexistent", true, 1)
			if err != sandbox.ErrNotFound {
				t.Errorf("err = %v, want ErrNotFound", err)
			}
		})
	}
}

func TestIdleCleanup(t *testing.T) {
	skipIfNoDocker(t)

	for _, pc := range testNodes() {
		pc := pc
		t.Run(pc.Name, func(t *testing.T) {
			m := setupManagerForNode(t, &pc)
			ensureTestImage(t, m, pc.TaiID)

			ctx := context.Background()
			box, err := m.Create(ctx, sandbox.CreateOptions{
				Image:       testImage(),
				Owner:       "test-user",
				NodeID:      pc.TaiID,
				Policy:      sandbox.Session,
				IdleTimeout: 1 * time.Second,
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

	for _, pc := range testNodes() {
		pc := pc
		t.Run(pc.Name, func(t *testing.T) {
			m1 := setupManagerForNode(t, &pc)
			box := createTestBox(t, m1, pc)
			boxID := box.ID()

			sandbox.Init()
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

	for _, pc := range testNodes() {
		pc := pc
		t.Run(pc.Name, func(t *testing.T) {
			m := setupManagerForNode(t, &pc)

			box := createTestBox(t, m, pc, func(co *sandbox.CreateOptions) {
				co.Policy = sandbox.Persistent
				co.IdleTimeout = 1 * time.Second
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
