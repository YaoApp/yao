package sandbox_test

import (
	"testing"

	sandbox "github.com/yaoapp/yao/sandbox/v2"
)

func TestDefaultWorkspaceName_English(t *testing.T) {
	name := sandbox.ExportDefaultWorkspaceName("en-US")
	if name != "Default Workspace" {
		t.Errorf("got %q, want %q", name, "Default Workspace")
	}
}

func TestDefaultWorkspaceName_Chinese(t *testing.T) {
	cases := []string{"zh", "zh-CN", "zh-TW", "ZH-cn"}
	for _, locale := range cases {
		name := sandbox.ExportDefaultWorkspaceName(locale)
		if name != "默认工作区" {
			t.Errorf("locale=%q: got %q, want %q", locale, name, "默认工作区")
		}
	}
}

func TestDefaultWorkspaceName_Empty(t *testing.T) {
	name := sandbox.ExportDefaultWorkspaceName("")
	if name != "Default Workspace" {
		t.Errorf("got %q, want %q", name, "Default Workspace")
	}
}

func TestSystemInfoFromLabels_Full(t *testing.T) {
	labels := map[string]string{
		"sandbox-sys-os":       "linux",
		"sandbox-sys-arch":     "amd64",
		"sandbox-sys-hostname": "node-1",
		"sandbox-sys-numcpu":   "8",
		"sandbox-sys-totalmem": "17179869184",
		"sandbox-sys-shell":    "bash",
	}
	sys := sandbox.ExportSystemInfoFromLabels(labels)

	if sys.OS != "linux" {
		t.Errorf("OS: got %q", sys.OS)
	}
	if sys.Arch != "amd64" {
		t.Errorf("Arch: got %q", sys.Arch)
	}
	if sys.Hostname != "node-1" {
		t.Errorf("Hostname: got %q", sys.Hostname)
	}
	if sys.NumCPU != 8 {
		t.Errorf("NumCPU: got %d", sys.NumCPU)
	}
	if sys.TotalMem != 17179869184 {
		t.Errorf("TotalMem: got %d", sys.TotalMem)
	}
	if sys.Shell != "bash" {
		t.Errorf("Shell: got %q", sys.Shell)
	}
}

func TestSystemInfoFromLabels_Empty(t *testing.T) {
	sys := sandbox.ExportSystemInfoFromLabels(map[string]string{})
	if sys.OS != "" || sys.Arch != "" || sys.Hostname != "" {
		t.Errorf("expected zero strings, got OS=%q Arch=%q Host=%q", sys.OS, sys.Arch, sys.Hostname)
	}
	if sys.NumCPU != 0 || sys.TotalMem != 0 {
		t.Errorf("expected zero numerics, got CPU=%d Mem=%d", sys.NumCPU, sys.TotalMem)
	}
}

func TestSystemInfoFromLabels_InvalidNumbers(t *testing.T) {
	labels := map[string]string{
		"sandbox-sys-numcpu":   "not-a-number",
		"sandbox-sys-totalmem": "invalid",
	}
	sys := sandbox.ExportSystemInfoFromLabels(labels)
	if sys.NumCPU != 0 {
		t.Errorf("NumCPU should be 0 on parse error, got %d", sys.NumCPU)
	}
	if sys.TotalMem != 0 {
		t.Errorf("TotalMem should be 0 on parse error, got %d", sys.TotalMem)
	}
}

func TestSystemInfoFromLabels_Partial(t *testing.T) {
	labels := map[string]string{
		"sandbox-sys-os":   "windows",
		"sandbox-sys-arch": "arm64",
	}
	sys := sandbox.ExportSystemInfoFromLabels(labels)
	if sys.OS != "windows" || sys.Arch != "arm64" {
		t.Errorf("got OS=%q Arch=%q", sys.OS, sys.Arch)
	}
	if sys.Hostname != "" || sys.Shell != "" {
		t.Errorf("missing fields should be empty")
	}
}
