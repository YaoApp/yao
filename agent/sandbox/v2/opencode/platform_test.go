package opencode

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// POSIX posixBase tests
// ---------------------------------------------------------------------------

func TestPosixBase_Accessors(t *testing.T) {
	b := &posixBase{os: "linux", workDir: "/workspace", shell: "bash"}
	assert.Equal(t, "linux", b.OS())
	assert.Equal(t, "bash", b.Shell())
	assert.Equal(t, "/workspace/.yao/config", b.PathJoin("/workspace", ".yao", "config"))
}

func TestPosixBase_HomeEnv(t *testing.T) {
	b := &posixBase{}
	env := b.HomeEnv("/workspace")
	assert.Equal(t, "/workspace", env["HOME"])
	assert.Len(t, env, 1)
}

func TestPosixBase_ShellCmd(t *testing.T) {
	b := &posixBase{}
	cmd := b.ShellCmd("echo hello")
	assert.Equal(t, []string{"bash", "-c", "echo hello"}, cmd)
}

func TestPosixBase_KillCmd(t *testing.T) {
	b := &posixBase{}
	cmd := b.KillCmd("opencode")
	require.Len(t, cmd, 3)
	assert.Equal(t, "sh", cmd[0])
	assert.Contains(t, cmd[2], "pkill")
	assert.Contains(t, cmd[2], "opencode")
}

func TestPosixBase_KillSessionCmd(t *testing.T) {
	b := &posixBase{}
	cmd := b.KillSessionCmd("yao-oc-session123")
	require.Len(t, cmd, 3)
	assert.Equal(t, "sh", cmd[0])
	assert.Contains(t, cmd[2], "pkill -9 -f")
	assert.Contains(t, cmd[2], "yao-oc-session123")
}

// ---------------------------------------------------------------------------
// Windows windowsPlatform tests
// ---------------------------------------------------------------------------

func TestWindows_NewDefaults(t *testing.T) {
	w := newWindowsPlatform(`C:\workspace`, "")
	assert.Equal(t, "pwsh", w.Shell())
}

func TestWindows_Accessors(t *testing.T) {
	w := newWindowsPlatform(`C:\workspace`, "pwsh")
	assert.Equal(t, "windows", w.OS())
	assert.Equal(t, "pwsh", w.Shell())
}

func TestWindows_PathJoin(t *testing.T) {
	w := newWindowsPlatform(`C:\workspace`, "pwsh")
	assert.Equal(t, `C:\workspace\.yao\config`, w.PathJoin(`C:\workspace`, ".yao", "config"))
	assert.Equal(t, `a\b\c`, w.PathJoin("a", "b", "c"))
}

func TestWindows_HomeEnv(t *testing.T) {
	w := newWindowsPlatform(`C:\workspace`, "pwsh")
	env := w.HomeEnv(`C:\workspace`)
	assert.Equal(t, `C:\workspace`, env["HOME"])
	assert.Equal(t, `C:\workspace`, env["USERPROFILE"])
	assert.Equal(t, `C:`, env["HOMEDRIVE"])
	assert.Equal(t, `\workspace`, env["HOMEPATH"])
	assert.Len(t, env, 4)
}

func TestWindows_HomeEnv_NoDrive(t *testing.T) {
	w := newWindowsPlatform("X", "pwsh")
	env := w.HomeEnv("X")
	assert.Equal(t, "X", env["HOME"])
	assert.Equal(t, "X", env["USERPROFILE"])
	_, hasDrive := env["HOMEDRIVE"]
	assert.False(t, hasDrive, "should not set HOMEDRIVE for path without drive letter")
	assert.Len(t, env, 2)
}

func TestWindows_ShellCmd_Pwsh(t *testing.T) {
	w := newWindowsPlatform(`C:\ws`, "pwsh")
	cmd := w.ShellCmd("echo hello")
	assert.Equal(t, []string{"pwsh", "-NoProfile", "-Command", "echo hello"}, cmd)
}

func TestWindows_ShellCmd_Powershell(t *testing.T) {
	w := newWindowsPlatform(`C:\ws`, "powershell")
	cmd := w.ShellCmd("echo hello")
	assert.Equal(t, "powershell", cmd[0])
	assert.Equal(t, "-NoProfile", cmd[1])
}

func TestWindows_ShellCmd_Cmd(t *testing.T) {
	w := newWindowsPlatform(`C:\ws`, "cmd.exe")
	cmd := w.ShellCmd("echo hello")
	assert.Equal(t, []string{"cmd.exe", "/C", "echo hello"}, cmd)
}

func TestWindows_ShellCmd_Default(t *testing.T) {
	w := newWindowsPlatform(`C:\ws`, "unknown-shell")
	cmd := w.ShellCmd("echo")
	assert.Equal(t, "pwsh", cmd[0], "unknown shell should fall back to pwsh")
}

func TestWindows_KillCmd(t *testing.T) {
	w := newWindowsPlatform(`C:\ws`, "pwsh")
	cmd := w.KillCmd("opencode")
	require.Len(t, cmd, 4)
	assert.Equal(t, "pwsh", cmd[0])
	assert.Contains(t, cmd[3], "opencode")
	assert.Contains(t, cmd[3], "taskkill")
	assert.Contains(t, cmd[3], "Stop-Process")
}

func TestWindows_KillSessionCmd(t *testing.T) {
	w := newWindowsPlatform(`C:\ws`, "pwsh")
	cmd := w.KillSessionCmd("yao-oc-session123")
	require.Len(t, cmd, 4)
	assert.Equal(t, "pwsh", cmd[0])
	assert.Contains(t, cmd[3], "CommandLine")
	assert.Contains(t, cmd[3], "yao-oc-session123")
	assert.Contains(t, cmd[3], "taskkill")
}

// ---------------------------------------------------------------------------
// shellQuote / shellQuoteForPlatform tests
// ---------------------------------------------------------------------------

func TestShellQuote_POSIX(t *testing.T) {
	result := shellQuote("opencode", "run", "--format", "json")
	assert.Equal(t, "opencode run --format json", result)
}

func TestShellQuote_POSIX_SpecialChars(t *testing.T) {
	result := shellQuote("opencode", "run", "hello world", "it's")
	assert.Contains(t, result, "'hello world'")
	assert.Contains(t, result, `'\''`)
}

func TestShellQuotePowerShell(t *testing.T) {
	result := shellQuotePowerShell("opencode", "run", "--format", "json")
	assert.Equal(t, "opencode run --format json", result)
}

func TestShellQuotePowerShell_SpecialChars(t *testing.T) {
	result := shellQuotePowerShell("opencode", "run", "hello world", "it's")
	assert.Contains(t, result, "'hello world'")
	assert.Contains(t, result, "'it''s'")
	assert.NotContains(t, result, `'\''`, "PowerShell should use '' not '\\''")
}

func TestShellQuoteForPlatform_POSIX(t *testing.T) {
	p := &posixBase{os: "linux"}
	result := shellQuoteForPlatform(p, "opencode", "it's")
	assert.Contains(t, result, `'\''`)
}

func TestShellQuoteForPlatform_Windows(t *testing.T) {
	p := newWindowsPlatform(`C:\ws`, "pwsh")
	result := shellQuoteForPlatform(p, "opencode", "it's")
	assert.Contains(t, result, "''s'")
	assert.NotContains(t, result, `'\''`)
}

// ---------------------------------------------------------------------------
// buildSandboxEnvPrompt tests
// ---------------------------------------------------------------------------

func TestBuildSandboxEnvPrompt_Linux(t *testing.T) {
	p := &posixBase{os: "linux", shell: "bash"}
	prompt := buildSandboxEnvPrompt(p, "/workspace")
	assert.Contains(t, prompt, "linux")
	assert.Contains(t, prompt, "bash")
	assert.Contains(t, prompt, "/workspace")
	assert.Contains(t, prompt, "$VAR_NAME")
	assert.NotContains(t, prompt, "$env:")
}

func TestBuildSandboxEnvPrompt_Windows(t *testing.T) {
	p := newWindowsPlatform(`C:\workspace`, "pwsh")
	prompt := buildSandboxEnvPrompt(p, `C:\workspace`)
	assert.Contains(t, prompt, "windows")
	assert.Contains(t, prompt, "pwsh")
	assert.Contains(t, prompt, `C:\workspace`)
	assert.Contains(t, prompt, "$env:VAR_NAME")
}

// ---------------------------------------------------------------------------
// Vision read.ts copy step generation logic
// ---------------------------------------------------------------------------

func TestVisionCopyStep_Linux(t *testing.T) {
	p := &posixBase{os: "linux", workDir: "/workspace", shell: "bash"}
	cmd := visionCopyCmd(p)
	assert.Contains(t, cmd, "mkdir -p")
	assert.Contains(t, cmd, "opencode-tools")
	assert.NotContains(t, cmd, "PowerShell")
}

func TestVisionCopyStep_Windows(t *testing.T) {
	p := newWindowsPlatform(`C:\workspace`, "pwsh")
	cmd := visionCopyCmd(p)
	assert.Contains(t, cmd, "Test-Path")
	assert.Contains(t, cmd, "Copy-Item")
	assert.Contains(t, cmd, `opencode-tools`)
	assert.NotContains(t, cmd, "mkdir -p")
}
