package sandbox

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/yaoapp/yao/grpc/pb"
)

func TestHeartbeat_StoresData(t *testing.T) {
	h := NewHandler(nil)

	req := &pb.HeartbeatRequest{
		SandboxId:    "sb-1",
		CpuPercent:   25,
		MemBytes:     1024 * 1024,
		RunningProcs: 3,
	}
	resp, err := h.Heartbeat(context.Background(), req)
	if err != nil {
		t.Fatalf("Heartbeat: %v", err)
	}
	if resp.Action != "ok" {
		t.Errorf("action = %q, want %q", resp.Action, "ok")
	}

	data := h.LastHeartbeat("sb-1")
	if data == nil {
		t.Fatal("LastHeartbeat returned nil")
	}
	if data.SandboxID != "sb-1" {
		t.Errorf("SandboxID = %q", data.SandboxID)
	}
	if data.CPUPercent != 25 {
		t.Errorf("CPUPercent = %d", data.CPUPercent)
	}
	if data.MemBytes != 1024*1024 {
		t.Errorf("MemBytes = %d", data.MemBytes)
	}
	if data.RunningProcs != 3 {
		t.Errorf("RunningProcs = %d", data.RunningProcs)
	}
	if time.Since(data.LastSeen) > time.Second {
		t.Errorf("LastSeen too old: %v", data.LastSeen)
	}
}

func TestHeartbeat_OnBeatCallback(t *testing.T) {
	var received *HeartbeatData
	h := NewHandler(func(d *HeartbeatData) string {
		received = d
		return "shutdown"
	})

	resp, err := h.Heartbeat(context.Background(), &pb.HeartbeatRequest{
		SandboxId:    "sb-2",
		CpuPercent:   90,
		MemBytes:     4096,
		RunningProcs: 10,
	})
	if err != nil {
		t.Fatalf("Heartbeat: %v", err)
	}
	if resp.Action != "shutdown" {
		t.Errorf("action = %q, want %q", resp.Action, "shutdown")
	}
	if received == nil || received.SandboxID != "sb-2" {
		t.Errorf("callback not invoked or wrong data")
	}
}

func TestHeartbeat_OnBeatEmptyReturnDefaultsToOK(t *testing.T) {
	h := NewHandler(func(d *HeartbeatData) string {
		return ""
	})

	resp, err := h.Heartbeat(context.Background(), &pb.HeartbeatRequest{SandboxId: "sb-3"})
	if err != nil {
		t.Fatalf("Heartbeat: %v", err)
	}
	if resp.Action != "ok" {
		t.Errorf("action = %q, want %q", resp.Action, "ok")
	}
}

func TestLastHeartbeat_Unknown(t *testing.T) {
	h := NewHandler(nil)
	if d := h.LastHeartbeat("nonexistent"); d != nil {
		t.Errorf("expected nil for unknown sandbox, got %+v", d)
	}
}

func TestRemoveHeartbeat(t *testing.T) {
	h := NewHandler(nil)

	h.Heartbeat(context.Background(), &pb.HeartbeatRequest{SandboxId: "sb-rm"})
	if h.LastHeartbeat("sb-rm") == nil {
		t.Fatal("expected data after heartbeat")
	}

	h.RemoveHeartbeat("sb-rm")
	if h.LastHeartbeat("sb-rm") != nil {
		t.Error("expected nil after RemoveHeartbeat")
	}
}

func TestRemoveHeartbeat_Idempotent(t *testing.T) {
	h := NewHandler(nil)
	h.RemoveHeartbeat("never-existed")
}

func TestHeartbeat_ConcurrentAccess(t *testing.T) {
	h := NewHandler(nil)
	var wg sync.WaitGroup

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			id := "sb-concurrent"
			h.Heartbeat(context.Background(), &pb.HeartbeatRequest{
				SandboxId:    id,
				CpuPercent:   int32(n),
				RunningProcs: int32(n),
			})
			h.LastHeartbeat(id)
		}(i)
	}
	wg.Wait()

	if d := h.LastHeartbeat("sb-concurrent"); d == nil {
		t.Error("expected data after concurrent heartbeats")
	}
}

func TestHeartbeat_MultiSandbox(t *testing.T) {
	h := NewHandler(nil)

	for _, id := range []string{"a", "b", "c"} {
		h.Heartbeat(context.Background(), &pb.HeartbeatRequest{SandboxId: id, CpuPercent: 10})
	}

	for _, id := range []string{"a", "b", "c"} {
		if d := h.LastHeartbeat(id); d == nil {
			t.Errorf("missing heartbeat for %q", id)
		}
	}

	h.RemoveHeartbeat("b")
	if h.LastHeartbeat("b") != nil {
		t.Error("b should be removed")
	}
	if h.LastHeartbeat("a") == nil || h.LastHeartbeat("c") == nil {
		t.Error("a and c should still exist")
	}
}
