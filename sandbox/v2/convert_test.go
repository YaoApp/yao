package sandbox

import (
	"testing"

	sqpb "github.com/yaoapp/yao/tai/systemquery/pb"
)

func TestConvertPorts_Nil(t *testing.T) {
	got := convertPorts(nil)
	if got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}

func TestConvertPorts_Empty(t *testing.T) {
	got := convertPorts([]*sqpb.PortInfo{})
	if got == nil {
		t.Fatal("expected non-nil empty slice")
	}
	if len(got) != 0 {
		t.Fatalf("expected empty, got %d", len(got))
	}
}

func TestConvertPorts_AllFields(t *testing.T) {
	input := []*sqpb.PortInfo{
		{Port: 8080, Protocol: "tcp", Process: "node", Pid: 123, State: "LISTEN", Address: "0.0.0.0", Command: "node /app/server.js"},
		{Port: 53, Protocol: "udp", Process: "dnsmasq", Pid: 456, State: "LISTEN", Address: "::", Command: "/usr/sbin/dnsmasq --keep-in-foreground"},
	}
	got := convertPorts(input)
	if len(got) != 2 {
		t.Fatalf("expected 2, got %d", len(got))
	}

	// Verify pointer slice (design doc requirement)
	if got[0] == nil || got[1] == nil {
		t.Fatal("expected non-nil pointers")
	}

	// Verify int type conversion from proto int32
	p := got[0]
	if p.Port != 8080 {
		t.Errorf("Port: got %d, want 8080", p.Port)
	}
	if p.PID != 123 {
		t.Errorf("PID: got %d, want 123", p.PID)
	}
	if p.Protocol != "tcp" || p.Process != "node" || p.State != "LISTEN" {
		t.Errorf("string fields mismatch: %+v", p)
	}
	if p.Address != "0.0.0.0" {
		t.Errorf("Address: got %q, want %q", p.Address, "0.0.0.0")
	}
	if p.Command != "node /app/server.js" {
		t.Errorf("Command: got %q", p.Command)
	}

	// Second port
	if got[1].Port != 53 || got[1].Protocol != "udp" {
		t.Errorf("port[1] mismatch: %+v", got[1])
	}
}

func TestConvertProcesses_Nil(t *testing.T) {
	got := convertProcesses(nil)
	if got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}

func TestConvertProcesses_AllFields(t *testing.T) {
	input := []*sqpb.ProcessInfo{
		{
			Pid: 1, Ppid: 0, User: "root", Command: "/sbin/init",
			State: "S", CpuPercent: 0.5, MemPercent: 1.2,
			RssBytes: 4096000, VszBytes: 8192000, StartTime: 1700000000,
			CpuTimeMs: 12345, Threads: 4, OpenFiles: 15,
		},
		{
			Pid: 1234, Ppid: 1, User: "app", Command: "python3 main.py",
			State: "R", CpuPercent: 99.9, MemPercent: 45.6,
			RssBytes: 2000000000, VszBytes: 4000000000, StartTime: 1700001000,
			CpuTimeMs: 999999, Threads: 32, OpenFiles: 256,
		},
	}
	got := convertProcesses(input)
	if len(got) != 2 {
		t.Fatalf("expected 2, got %d", len(got))
	}

	// Verify pointer slice
	if got[0] == nil || got[1] == nil {
		t.Fatal("expected non-nil pointers")
	}

	// First process - all fields
	p := got[0]
	if p.PID != 1 || p.PPID != 0 {
		t.Errorf("PID/PPID mismatch: %d/%d", p.PID, p.PPID)
	}
	if p.User != "root" {
		t.Errorf("User: got %q, want %q", p.User, "root")
	}
	if p.Command != "/sbin/init" {
		t.Errorf("Command: got %q", p.Command)
	}
	if p.State != "S" {
		t.Errorf("State: got %q", p.State)
	}
	if p.CPUPercent != 0.5 {
		t.Errorf("CPUPercent: got %f, want 0.5", p.CPUPercent)
	}
	if p.MemPercent != 1.2 {
		t.Errorf("MemPercent: got %f, want 1.2", p.MemPercent)
	}
	if p.RSSBytes != 4096000 {
		t.Errorf("RSSBytes: got %d", p.RSSBytes)
	}
	if p.VSZBytes != 8192000 {
		t.Errorf("VSZBytes: got %d", p.VSZBytes)
	}
	if p.StartTime != 1700000000 {
		t.Errorf("StartTime: got %d", p.StartTime)
	}
	if p.CPUTimeMs != 12345 {
		t.Errorf("CPUTimeMs: got %d", p.CPUTimeMs)
	}
	if p.Threads != 4 {
		t.Errorf("Threads: got %d", p.Threads)
	}
	if p.OpenFiles != 15 {
		t.Errorf("OpenFiles: got %d", p.OpenFiles)
	}

	// Second process - large values (int32→int conversion boundary)
	p2 := got[1]
	if p2.PID != 1234 || p2.Threads != 32 || p2.OpenFiles != 256 {
		t.Errorf("process[1] int fields: PID=%d Threads=%d OpenFiles=%d", p2.PID, p2.Threads, p2.OpenFiles)
	}
}

func TestConvertLoad_Nil(t *testing.T) {
	got := convertLoad(nil)
	if got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}

func TestConvertLoad_AllFields(t *testing.T) {
	input := &sqpb.SystemLoad{
		Load1: 1.5, Load5: 2.0, Load15: 3.0,
		MemTotal: 16000000000, MemUsed: 8000000000, MemAvailable: 9000000000,
		SwapTotal: 4000000000, SwapUsed: 1000000000,
		CpuCount: 8, CpuUsage: 45.5, UptimeSec: 86400,
	}
	got := convertLoad(input)
	if got == nil {
		t.Fatal("expected non-nil")
	}

	if got.Load1 != 1.5 || got.Load5 != 2.0 || got.Load15 != 3.0 {
		t.Errorf("load mismatch: %.2f/%.2f/%.2f", got.Load1, got.Load5, got.Load15)
	}
	if got.MemTotal != 16000000000 {
		t.Errorf("MemTotal: got %d", got.MemTotal)
	}
	if got.MemUsed != 8000000000 {
		t.Errorf("MemUsed: got %d", got.MemUsed)
	}
	if got.MemAvailable != 9000000000 {
		t.Errorf("MemAvailable: got %d", got.MemAvailable)
	}
	if got.SwapTotal != 4000000000 {
		t.Errorf("SwapTotal: got %d", got.SwapTotal)
	}
	if got.SwapUsed != 1000000000 {
		t.Errorf("SwapUsed: got %d", got.SwapUsed)
	}
	// CPUCount: int32→int
	if got.CPUCount != 8 {
		t.Errorf("CPUCount: got %d, want 8", got.CPUCount)
	}
	if got.CPUUsage != 45.5 {
		t.Errorf("CPUUsage: got %f", got.CPUUsage)
	}
	if got.UptimeSec != 86400 {
		t.Errorf("UptimeSec: got %d", got.UptimeSec)
	}
}

func TestConvertLoad_ZeroValues(t *testing.T) {
	input := &sqpb.SystemLoad{}
	got := convertLoad(input)
	if got == nil {
		t.Fatal("expected non-nil for zero-value input")
	}
	if got.CPUCount != 0 || got.MemTotal != 0 || got.UptimeSec != 0 {
		t.Errorf("zero input should produce zero output, got %+v", got)
	}
}
