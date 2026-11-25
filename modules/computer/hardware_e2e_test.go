//go:build e2e

package computer_test

import (
	"context"
	"testing"
	"time"

	"github.com/yaoapp/yao/modules/computer"
	"github.com/yaoapp/yao/modules/computer/types"
	"github.com/yaoapp/yao/openapi/oauth"
)

func TestE2EComputerSnapshotPersistence(t *testing.T) {
	_ = testEnvironment(t)
	ctx := context.Background()
	svc := computer.New().WithTokens(oauth.OAuth)
	created, err := svc.Create(ctx, types.CreateComputerRequest{
		NodeID: "node-snapshot-" + time.Now().Format("150405.000000000"),
		Name:   "Snapshot Persistence E2E", Status: types.StatusPending,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	t.Cleanup(func() { _ = svc.Delete(context.Background(), created.ID) })

	snapshot := &types.ComputerSnapshot{
		OS: "darwin", Arch: "arm64", CPUCores: 12, MemoryMB: 32768, DiskFreeMB: 512000,
		Capabilities:      []string{"session"},
		ContainerRuntimes: []types.ContainerRuntime{{Name: "docker", Version: "27", Socket: "/var/run/docker.sock"}},
		TaiVersion:        "2.0.0",
		IP:                "192.168.3.110",
		Networks:          []types.Network{{ID: "en0", Name: "en0", Kind: "lan", Family: "ipv4", Address: "192.168.3.110", PrefixLength: 24, Primary: true}},
		LastSeenAt:        time.Now().UTC(),
	}
	if _, err := svc.CreateOrUpdate(ctx, types.CreateOrUpdateComputerRequest{
		ID: created.ID, NodeID: created.NodeID, Status: types.StatusOnline, Snapshot: snapshot,
	}); err != nil {
		t.Fatalf("snapshot update: %v", err)
	}
	got, err := svc.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Hardware == nil || got.Hardware.CPUCores != 12 || got.Hardware.TaiVersion != "2.0.0" {
		t.Fatalf("hardware=%+v", got.Hardware)
	}
	if got.IP != "192.168.3.110" || len(got.Networks) != 1 || !got.Networks[0].Primary {
		t.Fatalf("networks ip=%s list=%+v", got.IP, got.Networks)
	}

	// A reconnect snapshot with no capabilities/runtimes must clear the old ones.
	if _, err := svc.CreateOrUpdate(ctx, types.CreateOrUpdateComputerRequest{
		ID: created.ID, NodeID: created.NodeID, Status: types.StatusOnline,
		Snapshot: &types.ComputerSnapshot{OS: "darwin", Arch: "arm64", CPUCores: 16, MemoryMB: 65536, DiskFreeMB: 1024, TaiVersion: "2.0.1", LastSeenAt: time.Now().UTC()},
	}); err != nil {
		t.Fatalf("snapshot replacement: %v", err)
	}
	got, err = svc.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("get after replacement: %v", err)
	}
	if got.Hardware == nil || got.Hardware.CPUCores != 16 || got.Hardware.TaiVersion != "2.0.1" {
		t.Fatalf("hardware after replacement=%+v", got.Hardware)
	}
	if len(got.Hardware.Capabilities) != 0 || len(got.Hardware.ContainerRuntimes) != 0 {
		t.Fatalf("stale arrays after replacement: %+v", got.Hardware)
	}
}
