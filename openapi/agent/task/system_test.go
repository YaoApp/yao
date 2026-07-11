package task

import (
	"encoding/json"
	"testing"

	sandbox "github.com/yaoapp/yao/sandbox/v2"
	"github.com/yaoapp/yao/tai/registry"
	taitypes "github.com/yaoapp/yao/tai/types"
)

func TestPortInfoJSONFormat(t *testing.T) {
	ports := []*sandbox.PortInfo{
		{Port: 8080, Protocol: "tcp", Process: "node", PID: 123, State: "LISTEN", Address: "0.0.0.0", Command: "node /app/server.js"},
		{Port: 443, Protocol: "tcp", Process: "nginx", PID: 456, State: "LISTEN", Address: "::", Command: "/usr/sbin/nginx -g daemon off;"},
	}

	data, err := json.Marshal(ports)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var parsed []map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(parsed) != 2 {
		t.Fatalf("expected 2 ports, got %d", len(parsed))
	}

	p := parsed[0]
	checks := map[string]interface{}{
		"port":     float64(8080),
		"protocol": "tcp",
		"process":  "node",
		"pid":      float64(123),
		"state":    "LISTEN",
		"address":  "0.0.0.0",
		"command":  "node /app/server.js",
	}
	for k, want := range checks {
		got, ok := p[k]
		if !ok {
			t.Errorf("missing field %q in JSON", k)
			continue
		}
		if got != want {
			t.Errorf("field %q: got %v, want %v", k, got, want)
		}
	}

	// Verify second port
	if parsed[1]["port"] != float64(443) || parsed[1]["process"] != "nginx" {
		t.Errorf("port[1] mismatch: %v", parsed[1])
	}
}

func TestProcessInfoJSONFormat(t *testing.T) {
	procs := []*sandbox.ProcessInfo{
		{
			PID: 1, PPID: 0, User: "root", Command: "/sbin/init",
			State: "S", CPUPercent: 0.5, MemPercent: 1.2,
			RSSBytes: 4096000, VSZBytes: 8192000, StartTime: 1700000000,
			CPUTimeMs: 12345, Threads: 4, OpenFiles: 15,
		},
	}

	data, err := json.Marshal(procs)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var parsed []map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(parsed) != 1 {
		t.Fatalf("expected 1 process, got %d", len(parsed))
	}

	p := parsed[0]
	expectedFields := []string{
		"pid", "ppid", "user", "command", "state",
		"cpuPercent", "memPercent", "rssBytes", "vszBytes",
		"startTime", "cpuTimeMs", "threads", "openFiles",
	}
	for _, f := range expectedFields {
		if _, ok := p[f]; !ok {
			t.Errorf("missing field %q in JSON output", f)
		}
	}

	// Verify specific values including new fields
	if p["pid"] != float64(1) {
		t.Errorf("pid: got %v, want 1", p["pid"])
	}
	if p["user"] != "root" {
		t.Errorf("user: got %v, want root", p["user"])
	}
	if p["command"] != "/sbin/init" {
		t.Errorf("command: got %v, want /sbin/init", p["command"])
	}
	if p["memPercent"].(float64) < 1.1 || p["memPercent"].(float64) > 1.3 {
		t.Errorf("memPercent: got %v, want ~1.2", p["memPercent"])
	}
	if p["vszBytes"] != float64(8192000) {
		t.Errorf("vszBytes: got %v, want 8192000", p["vszBytes"])
	}
	if p["cpuTimeMs"] != float64(12345) {
		t.Errorf("cpuTimeMs: got %v, want 12345", p["cpuTimeMs"])
	}
	if p["threads"] != float64(4) {
		t.Errorf("threads: got %v, want 4", p["threads"])
	}
	if p["openFiles"] != float64(15) {
		t.Errorf("openFiles: got %v, want 15", p["openFiles"])
	}
}

func TestSystemLoadJSONFormat(t *testing.T) {
	load := &sandbox.SystemLoad{
		Load1: 1.5, Load5: 2.0, Load15: 3.0,
		MemTotal: 16000000000, MemUsed: 8000000000, MemAvailable: 9000000000,
		SwapTotal: 4000000000, SwapUsed: 1000000000,
		CPUCount: 8, CPUUsage: 45.5, UptimeSec: 86400,
	}

	data, err := json.Marshal(load)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	expectedFields := []string{
		"load1", "load5", "load15",
		"memTotal", "memUsed", "memAvailable",
		"swapTotal", "swapUsed",
		"cpuCount", "cpuUsage", "uptimeSec",
	}
	for _, f := range expectedFields {
		if _, ok := parsed[f]; !ok {
			t.Errorf("missing field %q in JSON output", f)
		}
	}

	// Verify specific new fields
	if parsed["cpuCount"] != float64(8) {
		t.Errorf("cpuCount: got %v, want 8", parsed["cpuCount"])
	}
	if parsed["swapTotal"] != float64(4000000000) {
		t.Errorf("swapTotal: got %v, want 4000000000", parsed["swapTotal"])
	}
	if parsed["swapUsed"] != float64(1000000000) {
		t.Errorf("swapUsed: got %v, want 1000000000", parsed["swapUsed"])
	}
	if parsed["memAvailable"] != float64(9000000000) {
		t.Errorf("memAvailable: got %v, want 9000000000", parsed["memAvailable"])
	}
	if parsed["uptimeSec"] != float64(86400) {
		t.Errorf("uptimeSec: got %v, want 86400", parsed["uptimeSec"])
	}
}

func TestListProcessesOption_WithSkipCPU(t *testing.T) {
	opts := []sandbox.ListProcessesOption{sandbox.WithSkipCPU()}
	if len(opts) == 0 {
		t.Fatal("expected at least one option")
	}
}

func TestPortInfoPointerSlice(t *testing.T) {
	// Verify that the design doc's []*PortInfo pattern works correctly
	var ports []*sandbox.PortInfo
	ports = append(ports, &sandbox.PortInfo{Port: 80, Protocol: "tcp"})
	ports = append(ports, &sandbox.PortInfo{Port: 443, Protocol: "tcp"})

	data, err := json.Marshal(map[string]interface{}{"ports": ports})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	portsList, ok := result["ports"].([]interface{})
	if !ok {
		t.Fatal("ports should be an array")
	}
	if len(portsList) != 2 {
		t.Errorf("expected 2 ports, got %d", len(portsList))
	}
}

func TestSandboxNotRunningResponse(t *testing.T) {
	response := map[string]interface{}{
		"status":  "sandbox_not_running",
		"message": "sandbox is not running",
	}

	data, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if parsed["status"] != "sandbox_not_running" {
		t.Errorf("status: got %v", parsed["status"])
	}
}

func setupMockRegistry(t *testing.T) func() {
	t.Helper()
	reg := registry.NewForTest()
	registry.SetGlobalForTest(reg)
	return func() {
		registry.SetGlobalForTest(nil)
	}
}

func TestResolveHostNode(t *testing.T) {
	t.Run("selects online public node with HostExec", func(t *testing.T) {
		teardown := setupMockRegistry(t)
		defer teardown()

		reg := registry.Global()
		reg.Register(&registry.TaiNode{
			TaiID:        "tai-tunnel",
			Mode:         "tunnel",
			Capabilities: taitypes.Capabilities{HostExec: true},
		})
		reg.Register(&registry.TaiNode{
			TaiID:        "tai-cloud",
			Mode:         "cloud",
			Capabilities: taitypes.Capabilities{HostExec: true},
		})

		got := resolveHostNode()
		if got != "tai-cloud" {
			t.Errorf("resolveHostNode() = %q, want tai-cloud", got)
		}
	})

	t.Run("falls back to local when no suitable public node", func(t *testing.T) {
		teardown := setupMockRegistry(t)
		defer teardown()

		reg := registry.Global()
		reg.Register(&registry.TaiNode{
			TaiID:        "tai-tunnel",
			Mode:         "tunnel",
			Capabilities: taitypes.Capabilities{HostExec: true},
		})
		reg.Register(&registry.TaiNode{
			TaiID:        "tai-cloud",
			Mode:         "cloud",
			Capabilities: taitypes.Capabilities{HostExec: false},
		})

		got := resolveHostNode()
		if got != "local" {
			t.Errorf("resolveHostNode() = %q, want local", got)
		}
	})

	t.Run("falls back to local when registry is nil", func(t *testing.T) {
		registry.SetGlobalForTest(nil)
		got := resolveHostNode()
		if got != "local" {
			t.Errorf("resolveHostNode() = %q, want local", got)
		}
	})
}
