package systemquery

import (
	"context"
	"testing"
)

func TestLocalCollectorListPorts(t *testing.T) {
	c := NewLocalCollector()
	ports, err := c.ListPorts(context.Background())
	if err != nil {
		t.Fatalf("ListPorts error: %v", err)
	}
	if ports == nil {
		t.Log("no listening ports (expected in minimal env)")
	}
	for _, p := range ports {
		if p.Port <= 0 {
			t.Errorf("invalid port number: %d", p.Port)
		}
		if p.Protocol == "" {
			t.Error("empty protocol")
		}
	}
}

func TestLocalCollectorListProcesses(t *testing.T) {
	c := NewLocalCollector()

	t.Run("with_cpu_sample", func(t *testing.T) {
		procs, load, err := c.ListProcesses(context.Background(), false)
		if err != nil {
			t.Fatalf("ListProcesses error: %v", err)
		}
		if len(procs) == 0 {
			t.Fatal("expected at least one process")
		}
		if load == nil {
			t.Fatal("expected system load")
		}

		if load.CpuCount <= 0 {
			t.Errorf("expected positive cpu_count, got %d", load.CpuCount)
		}
		if load.MemTotal <= 0 {
			t.Errorf("expected positive mem_total, got %d", load.MemTotal)
		}
		if load.MemAvailable <= 0 {
			t.Errorf("expected positive mem_available, got %d", load.MemAvailable)
		}
	})

	t.Run("skip_cpu", func(t *testing.T) {
		procs, load, err := c.ListProcesses(context.Background(), true)
		if err != nil {
			t.Fatalf("ListProcesses (skipCPU) error: %v", err)
		}
		if len(procs) == 0 {
			t.Fatal("expected at least one process")
		}
		if load == nil {
			t.Fatal("expected system load even with skip_cpu")
		}
		if load.CpuUsage != 0 {
			t.Errorf("expected 0 cpu_usage with skip_cpu, got %f", load.CpuUsage)
		}
	})
}

func TestLocalCollectorProcessInfoFields(t *testing.T) {
	c := NewLocalCollector()
	procs, _, err := c.ListProcesses(context.Background(), true)
	if err != nil {
		t.Fatalf("ListProcesses error: %v", err)
	}

	for _, p := range procs {
		if p.Pid <= 0 {
			continue
		}
		if p.User != "" && p.Command != "" && p.RssBytes > 0 {
			return
		}
	}
	t.Log("no process found with all fields populated (may be permission issue)")
}
