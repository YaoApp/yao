package sandboxv2_test

import (
	"reflect"
	"testing"

	sandboxv2 "github.com/yaoapp/yao/agent/sandbox/v2"
)

func TestShellWrap_Sh(t *testing.T) {
	got := sandboxv2.ExportShellWrap(int(sandboxv2.ExportShellSh), "echo hello")
	want := []string{"sh", "-c", "echo hello"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("shellSh: got %v, want %v", got, want)
	}
}

func TestShellWrap_Pwsh(t *testing.T) {
	got := sandboxv2.ExportShellWrap(int(sandboxv2.ExportShellPwsh), "Write-Output hi")
	want := []string{"pwsh", "-NoProfile", "-Command", "Write-Output hi"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("shellPwsh: got %v, want %v", got, want)
	}
}

func TestShellWrap_PS(t *testing.T) {
	got := sandboxv2.ExportShellWrap(int(sandboxv2.ExportShellPS), "Get-Date")
	want := []string{"powershell", "-NoProfile", "-Command", "Get-Date"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("shellPS: got %v, want %v", got, want)
	}
}

func TestShellWrap_Cmd(t *testing.T) {
	got := sandboxv2.ExportShellWrap(int(sandboxv2.ExportShellCmd), "dir")
	want := []string{"cmd.exe", "/C", "dir"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("shellCmd: got %v, want %v", got, want)
	}
}

func TestShellWrap_UnknownKind(t *testing.T) {
	got := sandboxv2.ExportShellWrap(999, "test")
	want := []string{"sh", "-c", "test"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("unknown kind: got %v, want %v (should default to sh)", got, want)
	}
}
