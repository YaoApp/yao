//go:build unit

package opencode_test

import (
	"strings"
	"testing"

	opencode "github.com/yaoapp/yao/agent/sandbox/v2/opencode"
)

// ---------------------------------------------------------------------------
// POSIX posixBase tests
// ---------------------------------------------------------------------------

func TestPosixBase_Accessors(t *testing.T) {
	b := opencode.NewPosixBase("linux", "/workspace", "bash")
	if b.OS() != "linux" {
		t.Errorf("OS() = %q, want linux", b.OS())
	}
	if b.Shell() != "bash" {
		t.Errorf("Shell() = %q, want bash", b.Shell())
	}
	got := b.PathJoin("/workspace", ".yao", "config")
	if got != "/workspace/.yao/config" {
		t.Errorf("PathJoin() = %q, want /workspace/.yao/config", got)
	}
}

func TestPosixBase_HomeEnv(t *testing.T) {
	b := opencode.NewPosixBase("", "", "")
	env := b.HomeEnv("/workspace")
	if env["HOME"] != "/workspace" {
		t.Errorf("HOME = %q, want /workspace", env["HOME"])
	}
	if len(env) != 1 {
		t.Errorf("len(env) = %d, want 1", len(env))
	}
}

func TestPosixBase_ShellCmd(t *testing.T) {
	b := opencode.NewPosixBase("", "", "")
	cmd := b.ShellCmd("echo hello")
	want := []string{"bash", "-c", "echo hello"}
	if len(cmd) != len(want) {
		t.Fatalf("ShellCmd len = %d, want %d", len(cmd), len(want))
	}
	for i := range want {
		if cmd[i] != want[i] {
			t.Errorf("ShellCmd[%d] = %q, want %q", i, cmd[i], want[i])
		}
	}
}

func TestPosixBase_KillCmd(t *testing.T) {
	b := opencode.NewPosixBase("", "", "")
	cmd := b.KillCmd("opencode")
	if len(cmd) != 3 {
		t.Fatalf("KillCmd len = %d, want 3", len(cmd))
	}
	if cmd[0] != "sh" {
		t.Errorf("KillCmd[0] = %q, want sh", cmd[0])
	}
	if !strings.Contains(cmd[2], "pkill") {
		t.Error("KillCmd should contain pkill")
	}
	if !strings.Contains(cmd[2], "opencode") {
		t.Error("KillCmd should contain opencode")
	}
}

func TestPosixBase_KillSessionCmd(t *testing.T) {
	b := opencode.NewPosixBase("", "", "")
	cmd := b.KillSessionCmd("yao-oc-session123")
	if len(cmd) != 3 {
		t.Fatalf("KillSessionCmd len = %d, want 3", len(cmd))
	}
	if cmd[0] != "sh" {
		t.Errorf("KillSessionCmd[0] = %q, want sh", cmd[0])
	}
	if !strings.Contains(cmd[2], "pkill -9 -f") {
		t.Error("KillSessionCmd should contain 'pkill -9 -f'")
	}
	if !strings.Contains(cmd[2], "yao-oc-session123") {
		t.Error("KillSessionCmd should contain session name")
	}
}

// ---------------------------------------------------------------------------
// Windows windowsPlatform tests
// ---------------------------------------------------------------------------

func TestWindows_NewDefaults(t *testing.T) {
	w := opencode.NewWindowsPlatformForTest(`C:\workspace`, "")
	if w.Shell() != "pwsh" {
		t.Errorf("Shell() = %q, want pwsh", w.Shell())
	}
}

func TestWindows_Accessors(t *testing.T) {
	w := opencode.NewWindowsPlatformForTest(`C:\workspace`, "pwsh")
	if w.OS() != "windows" {
		t.Errorf("OS() = %q, want windows", w.OS())
	}
	if w.Shell() != "pwsh" {
		t.Errorf("Shell() = %q, want pwsh", w.Shell())
	}
}

func TestWindows_PathJoin(t *testing.T) {
	w := opencode.NewWindowsPlatformForTest(`C:\workspace`, "pwsh")
	got := w.PathJoin(`C:\workspace`, ".yao", "config")
	if got != `C:\workspace\.yao\config` {
		t.Errorf("PathJoin = %q, want C:\\workspace\\.yao\\config", got)
	}
	got = w.PathJoin("a", "b", "c")
	if got != `a\b\c` {
		t.Errorf("PathJoin = %q, want a\\b\\c", got)
	}
}

func TestWindows_HomeEnv(t *testing.T) {
	w := opencode.NewWindowsPlatformForTest(`C:\workspace`, "pwsh")
	env := w.HomeEnv(`C:\workspace`)
	if env["HOME"] != `C:\workspace` {
		t.Errorf("HOME = %q", env["HOME"])
	}
	if env["USERPROFILE"] != `C:\workspace` {
		t.Errorf("USERPROFILE = %q", env["USERPROFILE"])
	}
	if env["HOMEDRIVE"] != `C:` {
		t.Errorf("HOMEDRIVE = %q", env["HOMEDRIVE"])
	}
	if env["HOMEPATH"] != `\workspace` {
		t.Errorf("HOMEPATH = %q", env["HOMEPATH"])
	}
	if len(env) != 4 {
		t.Errorf("len(env) = %d, want 4", len(env))
	}
}

func TestWindows_HomeEnv_NoDrive(t *testing.T) {
	w := opencode.NewWindowsPlatformForTest("X", "pwsh")
	env := w.HomeEnv("X")
	if env["HOME"] != "X" {
		t.Errorf("HOME = %q", env["HOME"])
	}
	if env["USERPROFILE"] != "X" {
		t.Errorf("USERPROFILE = %q", env["USERPROFILE"])
	}
	if _, ok := env["HOMEDRIVE"]; ok {
		t.Error("should not set HOMEDRIVE for path without drive letter")
	}
	if len(env) != 2 {
		t.Errorf("len(env) = %d, want 2", len(env))
	}
}

func TestWindows_ShellCmd_Pwsh(t *testing.T) {
	w := opencode.NewWindowsPlatformForTest(`C:\ws`, "pwsh")
	cmd := w.ShellCmd("echo hello")
	want := []string{"pwsh", "-NoProfile", "-Command", "echo hello"}
	if len(cmd) != len(want) {
		t.Fatalf("len = %d, want %d", len(cmd), len(want))
	}
	for i := range want {
		if cmd[i] != want[i] {
			t.Errorf("cmd[%d] = %q, want %q", i, cmd[i], want[i])
		}
	}
}

func TestWindows_ShellCmd_Powershell(t *testing.T) {
	w := opencode.NewWindowsPlatformForTest(`C:\ws`, "powershell")
	cmd := w.ShellCmd("echo hello")
	if cmd[0] != "powershell" {
		t.Errorf("cmd[0] = %q, want powershell", cmd[0])
	}
	if cmd[1] != "-NoProfile" {
		t.Errorf("cmd[1] = %q, want -NoProfile", cmd[1])
	}
}

func TestWindows_ShellCmd_Cmd(t *testing.T) {
	w := opencode.NewWindowsPlatformForTest(`C:\ws`, "cmd.exe")
	cmd := w.ShellCmd("echo hello")
	want := []string{"cmd.exe", "/C", "echo hello"}
	if len(cmd) != len(want) {
		t.Fatalf("len = %d, want %d", len(cmd), len(want))
	}
	for i := range want {
		if cmd[i] != want[i] {
			t.Errorf("cmd[%d] = %q, want %q", i, cmd[i], want[i])
		}
	}
}

func TestWindows_ShellCmd_Default(t *testing.T) {
	w := opencode.NewWindowsPlatformForTest(`C:\ws`, "unknown-shell")
	cmd := w.ShellCmd("echo")
	if cmd[0] != "pwsh" {
		t.Errorf("cmd[0] = %q, want pwsh (unknown shell should fall back)", cmd[0])
	}
}

func TestWindows_KillCmd(t *testing.T) {
	w := opencode.NewWindowsPlatformForTest(`C:\ws`, "pwsh")
	cmd := w.KillCmd("opencode")
	if len(cmd) != 4 {
		t.Fatalf("KillCmd len = %d, want 4", len(cmd))
	}
	if cmd[0] != "pwsh" {
		t.Errorf("cmd[0] = %q, want pwsh", cmd[0])
	}
	if !strings.Contains(cmd[3], "opencode") {
		t.Error("should contain opencode")
	}
	if !strings.Contains(cmd[3], "taskkill") {
		t.Error("should contain taskkill")
	}
	if !strings.Contains(cmd[3], "Stop-Process") {
		t.Error("should contain Stop-Process")
	}
}

func TestWindows_KillSessionCmd(t *testing.T) {
	w := opencode.NewWindowsPlatformForTest(`C:\ws`, "pwsh")
	cmd := w.KillSessionCmd("yao-oc-session123")
	if len(cmd) != 4 {
		t.Fatalf("KillSessionCmd len = %d, want 4", len(cmd))
	}
	if cmd[0] != "pwsh" {
		t.Errorf("cmd[0] = %q, want pwsh", cmd[0])
	}
	if !strings.Contains(cmd[3], "CommandLine") {
		t.Error("should contain CommandLine")
	}
	if !strings.Contains(cmd[3], "yao-oc-session123") {
		t.Error("should contain session name")
	}
	if !strings.Contains(cmd[3], "taskkill") {
		t.Error("should contain taskkill")
	}
}

// ---------------------------------------------------------------------------
// shellQuote / shellQuoteForPlatform tests
// ---------------------------------------------------------------------------

func TestShellQuote_POSIX(t *testing.T) {
	result := opencode.ShellQuote("opencode", "run", "--format", "json")
	if result != "opencode run --format json" {
		t.Errorf("got %q, want 'opencode run --format json'", result)
	}
}

func TestShellQuote_POSIX_SpecialChars(t *testing.T) {
	result := opencode.ShellQuote("opencode", "run", "hello world", "it's")
	if !strings.Contains(result, "'hello world'") {
		t.Errorf("should quote 'hello world', got %q", result)
	}
	if !strings.Contains(result, `'\''`) {
		t.Errorf("should escape single quote with '\\''")
	}
}

func TestShellQuotePowerShell(t *testing.T) {
	result := opencode.ShellQuotePowerShell("opencode", "run", "--format", "json")
	if result != "opencode run --format json" {
		t.Errorf("got %q, want 'opencode run --format json'", result)
	}
}

func TestShellQuotePowerShell_SpecialChars(t *testing.T) {
	result := opencode.ShellQuotePowerShell("opencode", "run", "hello world", "it's")
	if !strings.Contains(result, "'hello world'") {
		t.Errorf("should quote 'hello world', got %q", result)
	}
	if !strings.Contains(result, "'it''s'") {
		t.Errorf("should double single quote in PowerShell, got %q", result)
	}
	if strings.Contains(result, `'\''`) {
		t.Error("PowerShell should use '' not '\\''")
	}
}

func TestShellQuoteForPlatform_POSIX(t *testing.T) {
	p := opencode.NewPosixBase("linux", "", "")
	result := opencode.ShellQuoteForPlatformExport(p, "opencode", "it's")
	if !strings.Contains(result, `'\''`) {
		t.Errorf("POSIX should use '\\'' escaping, got %q", result)
	}
}

func TestShellQuoteForPlatform_Windows(t *testing.T) {
	p := opencode.NewWindowsPlatformForTest(`C:\ws`, "pwsh")
	result := opencode.ShellQuoteForPlatformExport(p, "opencode", "it's")
	if !strings.Contains(result, "''s'") {
		t.Errorf("Windows should use '' escaping, got %q", result)
	}
	if strings.Contains(result, `'\''`) {
		t.Error("Windows should not use POSIX escaping")
	}
}

// ---------------------------------------------------------------------------
// buildSandboxEnvPrompt tests
// ---------------------------------------------------------------------------

func TestBuildSandboxEnvPrompt_Linux(t *testing.T) {
	p := opencode.NewPosixBase("linux", "", "bash")
	prompt := opencode.BuildSandboxEnvPrompt(p, "/workspace")
	if !strings.Contains(prompt, "linux") {
		t.Error("should contain linux")
	}
	if !strings.Contains(prompt, "bash") {
		t.Error("should contain bash")
	}
	if !strings.Contains(prompt, "/workspace") {
		t.Error("should contain /workspace")
	}
}

func TestBuildSandboxEnvPrompt_Windows(t *testing.T) {
	p := opencode.NewWindowsPlatformForTest(`C:\workspace`, "pwsh")
	prompt := opencode.BuildSandboxEnvPrompt(p, `C:\workspace`)
	if !strings.Contains(prompt, "windows") {
		t.Error("should contain windows")
	}
	if !strings.Contains(prompt, "pwsh") {
		t.Error("should contain pwsh")
	}
	if !strings.Contains(prompt, `C:\workspace`) {
		t.Error("should contain workspace path")
	}
}

// ---------------------------------------------------------------------------
// Vision read.ts copy step generation logic
// ---------------------------------------------------------------------------

func TestVisionCopyStep_Linux(t *testing.T) {
	p := opencode.NewPosixBase("linux", "/workspace", "bash")
	cmd := opencode.VisionCopyCmd(p)
	if !strings.Contains(cmd, "mkdir -p") {
		t.Error("should contain 'mkdir -p'")
	}
	if !strings.Contains(cmd, "opencode-tools") {
		t.Error("should contain 'opencode-tools'")
	}
	if strings.Contains(cmd, "PowerShell") {
		t.Error("should not contain PowerShell for linux")
	}
}

func TestVisionCopyStep_Windows(t *testing.T) {
	p := opencode.NewWindowsPlatformForTest(`C:\workspace`, "pwsh")
	cmd := opencode.VisionCopyCmd(p)
	if !strings.Contains(cmd, "Test-Path") {
		t.Error("should contain Test-Path")
	}
	if !strings.Contains(cmd, "Copy-Item") {
		t.Error("should contain Copy-Item")
	}
	if !strings.Contains(cmd, "opencode-tools") {
		t.Error("should contain opencode-tools")
	}
	if strings.Contains(cmd, "mkdir -p") {
		t.Error("should not contain 'mkdir -p' for Windows")
	}
}
