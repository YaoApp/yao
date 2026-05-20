package sandbox_test

import (
	"os"
	"testing"

	sandbox "github.com/yaoapp/yao/sandbox/v2"
)

func TestHostComputerInfo(t *testing.T) {
	h := sandbox.ExportNewHostForTest("node-2", sandbox.SystemInfo{
		OS: "linux", Arch: "arm64", Hostname: "host-1",
		NumCPU: 4, TotalMem: 8589934592, Shell: "bash",
	})
	info := h.ComputerInfo()

	if info.Kind != "host" {
		t.Errorf("Kind: got %q, want %q", info.Kind, "host")
	}
	if info.NodeID != "node-2" {
		t.Errorf("NodeID: got %q", info.NodeID)
	}
	if info.System.OS != "linux" {
		t.Errorf("System.OS: got %q", info.System.OS)
	}
	if info.System.Arch != "arm64" {
		t.Errorf("System.Arch: got %q", info.System.Arch)
	}
	if info.System.Hostname != "host-1" {
		t.Errorf("System.Hostname: got %q", info.System.Hostname)
	}
	if info.System.NumCPU != 4 {
		t.Errorf("System.NumCPU: got %d", info.System.NumCPU)
	}
	if info.Status != "online" {
		t.Errorf("Status: got %q", info.Status)
	}
}

func TestHostComputerInfo_Windows(t *testing.T) {
	h := sandbox.ExportNewHostForTest("win-node", sandbox.SystemInfo{
		OS: "windows", Arch: "amd64", Shell: "pwsh",
	})
	info := h.ComputerInfo()
	if info.System.OS != "windows" || info.System.Shell != "pwsh" {
		t.Errorf("Windows info: OS=%q Shell=%q", info.System.OS, info.System.Shell)
	}
}

func TestHostNodeID(t *testing.T) {
	h := sandbox.ExportNewHostForTest("my-node", sandbox.SystemInfo{})
	if h.NodeID() != "my-node" {
		t.Errorf("NodeID: got %q", h.NodeID())
	}
}

func TestHostGetWorkDir_TempDir(t *testing.T) {
	h := sandbox.ExportNewHostForTest("node-x", sandbox.SystemInfo{
		TempDir: "/var/tmp",
	})
	dir := h.GetWorkDir()
	if dir != "/var/tmp" {
		t.Errorf("GetWorkDir: got %q, want %q", dir, "/var/tmp")
	}
}

func TestHostGetWorkDir_Default(t *testing.T) {
	h := sandbox.ExportNewHostForTest("node-y", sandbox.SystemInfo{})
	dir := h.GetWorkDir()
	expected := os.TempDir()
	if dir != expected {
		t.Errorf("GetWorkDir: got %q, want %q", dir, expected)
	}
}
