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

func TestStartBox(t *testing.T) {
	skipIfNoDocker(t)

	for _, pc := range testNodes() {
		pc := pc
		t.Run(pc.Name, func(t *testing.T) {
			m := setupManagerForNode(t, &pc)
			box := createTestBox(t, m, pc)
			boxID := box.ID()
			ctx := context.Background()

			if err := box.Stop(ctx); err != nil {
				t.Fatalf("Stop: %v", err)
			}

			time.Sleep(500 * time.Millisecond)

			if err := m.StartBox(ctx, boxID); err != nil {
				t.Fatalf("StartBox: %v", err)
			}

			info, err := box.Info(ctx)
			if err != nil {
				t.Fatalf("Info after StartBox: %v", err)
			}
			if info.Status != "running" {
				t.Errorf("status = %q after StartBox, want running", info.Status)
			}
		})
	}
}

func TestSnapshotReadsStatus(t *testing.T) {
	skipIfNoDocker(t)

	for _, pc := range testNodes() {
		pc := pc
		t.Run(pc.Name, func(t *testing.T) {
			m := setupManagerForNode(t, &pc)
			box := createTestBox(t, m, pc)

			snap := box.Snapshot()
			if snap.Status != "running" {
				t.Errorf("initial snapshot status = %q, want running", snap.Status)
			}
		})
	}
}
