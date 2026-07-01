package systemquery

import (
	"context"
	"testing"

	"github.com/yaoapp/yao/tai/systemquery/pb"
)

func TestLocalClient_ImplementsInterface(t *testing.T) {
	var _ pb.SystemQueryClient = (*LocalClient)(nil)
}

func TestLocalClient_ListPorts(t *testing.T) {
	client := NewLocalClient()
	resp, err := client.ListPorts(context.Background(), &pb.ListPortsRequest{})
	if err != nil {
		t.Fatalf("ListPorts error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	// Verify structural integrity of returned ports
	for _, p := range resp.Ports {
		if p.Port <= 0 || p.Port > 65535 {
			t.Errorf("invalid port: %d", p.Port)
		}
		if p.Protocol != "tcp" && p.Protocol != "udp" {
			if p.Protocol == "" {
				t.Error("empty protocol")
			}
		}
		if p.State == "" {
			t.Error("empty state")
		}
	}
}

func TestLocalClient_ListProcesses_WithCPU(t *testing.T) {
	client := NewLocalClient()
	resp, err := client.ListProcesses(context.Background(), &pb.ListProcessesRequest{SkipCpuSample: false})
	if err != nil {
		t.Fatalf("ListProcesses error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if len(resp.Processes) == 0 {
		t.Fatal("expected at least one process")
	}
	if resp.Load == nil {
		t.Fatal("expected non-nil load")
	}

	// Verify load structure
	if resp.Load.CpuCount <= 0 {
		t.Errorf("expected positive cpu_count, got %d", resp.Load.CpuCount)
	}
	if resp.Load.MemTotal <= 0 {
		t.Errorf("expected positive mem_total, got %d", resp.Load.MemTotal)
	}
	if resp.Load.MemAvailable <= 0 {
		t.Errorf("expected positive mem_available, got %d", resp.Load.MemAvailable)
	}
	if resp.Load.UptimeSec <= 0 {
		t.Errorf("expected positive uptime, got %d", resp.Load.UptimeSec)
	}

	// Verify at least one process has valid data
	var foundValid bool
	for _, p := range resp.Processes {
		if p.Pid > 0 && p.Command != "" {
			foundValid = true
			break
		}
	}
	if !foundValid {
		t.Error("no process with both pid>0 and non-empty command")
	}
}

func TestLocalClient_ListProcesses_SkipCPU(t *testing.T) {
	client := NewLocalClient()
	resp, err := client.ListProcesses(context.Background(), &pb.ListProcessesRequest{SkipCpuSample: true})
	if err != nil {
		t.Fatalf("ListProcesses (skip_cpu) error: %v", err)
	}
	if resp == nil || resp.Load == nil {
		t.Fatal("expected non-nil response and load")
	}
	// With skip_cpu, CpuUsage should be 0
	if resp.Load.CpuUsage != 0 {
		t.Errorf("expected 0 cpu_usage with skip_cpu, got %f", resp.Load.CpuUsage)
	}
	// All processes should have 0 cpu_percent
	for _, p := range resp.Processes {
		if p.CpuPercent != 0 {
			t.Errorf("pid %d: expected 0 cpu_percent with skip_cpu, got %f", p.Pid, p.CpuPercent)
			break
		}
	}
}

func TestLocalClient_ListProcesses_FieldCompleteness(t *testing.T) {
	client := NewLocalClient()
	resp, err := client.ListProcesses(context.Background(), &pb.ListProcessesRequest{SkipCpuSample: true})
	if err != nil {
		t.Fatalf("ListProcesses error: %v", err)
	}

	// Check that new fields (user, mem_percent, vsz_bytes, etc.) are actually populated
	var hasUser, hasMemPercent, hasVSZ, hasCPUTime bool
	for _, p := range resp.Processes {
		if p.User != "" {
			hasUser = true
		}
		if p.MemPercent > 0 {
			hasMemPercent = true
		}
		if p.VszBytes > 0 {
			hasVSZ = true
		}
		if p.CpuTimeMs > 0 {
			hasCPUTime = true
		}
	}

	if !hasUser {
		t.Log("warning: no process had user field populated (permission issue?)")
	}
	if !hasMemPercent {
		t.Log("warning: no process had mem_percent > 0")
	}
	if !hasVSZ {
		t.Log("warning: no process had vsz_bytes > 0")
	}
	if !hasCPUTime {
		t.Log("warning: no process had cpu_time_ms > 0")
	}

	// Verify SystemLoad new fields
	load := resp.Load
	if load.SwapTotal < 0 {
		t.Errorf("swap_total should be >= 0, got %d", load.SwapTotal)
	}
	if load.CpuCount <= 0 {
		t.Errorf("cpu_count should be > 0, got %d", load.CpuCount)
	}
}
