package claude_test

import (
	"os"
	"strings"
	"testing"

	"github.com/yaoapp/yao/agent/sandbox/v2/claude"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

func TestMain(m *testing.M) {
	testprepare.MustLoadEnv()
	os.Exit(m.Run())
}

// ---------------------------------------------------------------------------
// POSIX / Linux
// ---------------------------------------------------------------------------

func TestPosix_Accessors(t *testing.T) {
	p := claude.ExportNewPosixPlatform("linux", "/workspace", "bash", "/tmp")
	if p.OS() != "linux" {
		t.Errorf("OS = %q, want linux", p.OS())
	}
	if p.Shell() != "bash" {
		t.Errorf("Shell = %q, want bash", p.Shell())
	}
	if p.RootDir() != "/" {
		t.Errorf("RootDir = %q, want /", p.RootDir())
	}
	if p.ConfigDir() != ".config/claude" {
		t.Errorf("ConfigDir = %q", p.ConfigDir())
	}
}

func TestPosix_PathJoin(t *testing.T) {
	p := claude.ExportNewPosixPlatform("linux", "/workspace", "bash", "/tmp")
	got := p.PathJoin("/workspace", ".yao", "config")
	if got != "/workspace/.yao/config" {
		t.Errorf("PathJoin = %q, want /workspace/.yao/config", got)
	}
}

func TestPosix_HomeEnv(t *testing.T) {
	p := claude.ExportNewPosixPlatform("linux", "/workspace", "bash", "/tmp")
	env := p.HomeEnv("/workspace/data")
	if env["HOME"] != "/workspace/data" {
		t.Errorf("HOME = %q", env["HOME"])
	}
	if len(env) != 1 {
		t.Errorf("expected 1 key, got %d", len(env))
	}
}

func TestPosix_ShellCmd(t *testing.T) {
	p := claude.ExportNewPosixPlatform("linux", "/workspace", "bash", "/tmp")
	cmd := p.ShellCmd("echo hello")
	if len(cmd) != 3 || cmd[0] != "bash" || cmd[1] != "-c" || cmd[2] != "echo hello" {
		t.Errorf("ShellCmd = %v", cmd)
	}
}

func TestPosix_KillCmd(t *testing.T) {
	p := claude.ExportNewPosixPlatform("linux", "/workspace", "bash", "/tmp")
	cmd := p.KillCmd("claude")
	if len(cmd) != 3 || cmd[0] != "sh" {
		t.Errorf("KillCmd = %v", cmd)
	}
	if !strings.Contains(cmd[2], "pkill") || !strings.Contains(cmd[2], "claude") {
		t.Errorf("KillCmd body = %q", cmd[2])
	}
}

func TestPosix_KillSessionCmd(t *testing.T) {
	p := claude.ExportNewPosixPlatform("linux", "/workspace", "bash", "/tmp")
	cmd := p.KillSessionCmd("yao-robot_m1_e1")
	if len(cmd) != 3 || cmd[0] != "sh" {
		t.Errorf("KillSessionCmd = %v", cmd)
	}
	if !strings.Contains(cmd[2], "pkill -9 -f") || !strings.Contains(cmd[2], "yao-robot_m1_e1") {
		t.Errorf("KillSessionCmd body = %q", cmd[2])
	}
}

func TestPosix_ListDirCmd(t *testing.T) {
	p := claude.ExportNewPosixPlatform("linux", "/workspace", "bash", "/tmp")
	cmd := p.ListDirCmd("/workspace/.claude")
	if len(cmd) != 2 || cmd[0] != "ls" || cmd[1] != "/workspace/.claude" {
		t.Errorf("ListDirCmd = %v", cmd)
	}
}

func TestPosix_XauthoritySetup(t *testing.T) {
	p := claude.ExportNewPosixPlatform("linux", "/workspace", "bash", "/tmp")
	if got := p.XauthoritySetup("/workspace"); got != "" {
		t.Errorf("XauthoritySetup should be empty for basic posix, got %q", got)
	}
}

// ---------------------------------------------------------------------------
// Linux specifics
// ---------------------------------------------------------------------------

func TestLinux_XauthoritySetup_Headless(t *testing.T) {
	p := claude.ExportNewLinuxPlatform("/workspace", "bash", "/tmp", false, "/root")
	if got := p.XauthoritySetup("/workspace"); got != "" {
		t.Errorf("should be empty for headless, got %q", got)
	}
}

func TestLinux_XauthoritySetup_Desktop(t *testing.T) {
	p := claude.ExportNewLinuxPlatform("/workspace", "bash", "/tmp", true, "/root")
	cmd := p.XauthoritySetup("/workspace")
	if !strings.Contains(cmd, "/root/.Xauthority") || !strings.Contains(cmd, "cp") {
		t.Errorf("XauthoritySetup = %q", cmd)
	}
}

func TestLinux_XauthoritySetup_NoSysHome(t *testing.T) {
	p := claude.ExportNewLinuxPlatform("/workspace", "bash", "/tmp", true, "")
	if got := p.XauthoritySetup("/workspace"); got != "" {
		t.Errorf("should be empty without sysHome, got %q", got)
	}
}

func TestLinux_EnvPromptNote_Desktop(t *testing.T) {
	p := claude.ExportNewLinuxPlatform("/workspace", "bash", "/tmp", true, "/root")
	note := p.EnvPromptNote()
	if !strings.Contains(note, "VNC") {
		t.Errorf("EnvPromptNote = %q, expected VNC", note)
	}
}

func TestLinux_EnvPromptNote_Headless(t *testing.T) {
	p := claude.ExportNewLinuxPlatform("/workspace", "bash", "/tmp", false, "")
	if note := p.EnvPromptNote(); note != "" {
		t.Errorf("EnvPromptNote should be empty for headless, got %q", note)
	}
}

func TestLinux_BuildScript_Desktop(t *testing.T) {
	p := claude.ExportNewLinuxPlatform("/workspace", "bash", "/tmp", true, "/root")
	script, stdin := p.BuildScript(claude.ExportScriptInput{
		Args:       []string{"--verbose"},
		InputJSONL: `{"type":"user"}`,
		WorkDir:    "/workspace",
		PromptFile: "/workspace/.yao/.system-prompt.txt",
	})
	if !strings.Contains(script, ".Xauthority") {
		t.Errorf("script should contain Xauthority copy")
	}
	if stdin != nil {
		t.Errorf("stdin should be nil for Linux")
	}
}

func TestLinux_BuildScript_NoPrompt(t *testing.T) {
	p := claude.ExportNewLinuxPlatform("/workspace", "bash", "/tmp", false, "")
	script, _ := p.BuildScript(claude.ExportScriptInput{
		Args:       []string{"--verbose", "--output-format", "stream-json"},
		InputJSONL: `{"type":"user","message":{"role":"user","content":"hello"}}`,
		WorkDir:    "/workspace",
		PromptFile: "/workspace/.yao/.system-prompt.txt",
	})
	if !strings.Contains(script, "cat << 'INPUTEOF' | claude -p") {
		t.Error("script should contain heredoc + claude -p")
	}
	if !strings.Contains(script, "--verbose") {
		t.Error("script should contain args")
	}
	if strings.Contains(script, "PROMPTEOF") {
		t.Error("script should not contain PROMPTEOF without system prompt")
	}
}

func TestLinux_BuildScript_WithPrompt(t *testing.T) {
	p := claude.ExportNewLinuxPlatform("/workspace", "bash", "/tmp", false, "")
	script, _ := p.BuildScript(claude.ExportScriptInput{
		Args:         []string{"--verbose"},
		SystemPrompt: "You are a helpful assistant.",
		InputJSONL:   `{"type":"user"}`,
		WorkDir:      "/workspace",
		PromptFile:   "/workspace/.yao/assistants/test-id/system-prompt.txt",
	})
	if !strings.Contains(script, "mkdir -p") {
		t.Error("script should create prompt dir")
	}
	if !strings.Contains(script, "PROMPTEOF") {
		t.Error("script should contain PROMPTEOF")
	}
	if !strings.Contains(script, "You are a helpful assistant.") {
		t.Error("script should contain system prompt")
	}
	if !strings.Contains(script, "--append-system-prompt-file") {
		t.Error("script should append system prompt file arg")
	}
	if !strings.Contains(script, "set -e") || !strings.Contains(script, "set +e") {
		t.Error("script should use set -e/+e around prompt write")
	}
	setEIdx := strings.Index(script, "set -e")
	promptIdx := strings.Index(script, "PROMPTEOF")
	setNoEIdx := strings.Index(script, "set +e")
	claudeIdx := strings.Index(script, "claude -p")
	if setEIdx >= promptIdx || promptIdx >= setNoEIdx || setNoEIdx >= claudeIdx {
		t.Errorf("ordering: set -e(%d) < PROMPTEOF(%d) < set +e(%d) < claude -p(%d)",
			setEIdx, promptIdx, setNoEIdx, claudeIdx)
	}
}

// ---------------------------------------------------------------------------
// Darwin
// ---------------------------------------------------------------------------

func TestDarwin_EnvPromptNote(t *testing.T) {
	p := claude.ExportNewDarwinPlatform("/workspace", "bash", "/tmp")
	note := p.EnvPromptNote()
	if !strings.Contains(note, "macOS desktop") {
		t.Errorf("EnvPromptNote = %q", note)
	}
}

func TestDarwin_BuildScript(t *testing.T) {
	p := claude.ExportNewDarwinPlatform("/workspace", "bash", "/tmp")
	script, stdin := p.BuildScript(claude.ExportScriptInput{
		Args:       []string{"--verbose"},
		InputJSONL: `{"type":"user"}`,
		WorkDir:    "/workspace",
		PromptFile: "/workspace/.yao/.system-prompt.txt",
	})
	if !strings.Contains(script, "claude -p") {
		t.Error("script should contain claude -p")
	}
	if stdin != nil {
		t.Error("stdin should be nil for Darwin")
	}
}

// ---------------------------------------------------------------------------
// Windows
// ---------------------------------------------------------------------------

func TestWindows_Defaults(t *testing.T) {
	p := claude.ExportNewWindowsPlatform(`C:\workspace`, "", "")
	if p.Shell() != "pwsh" {
		t.Errorf("Shell = %q, want pwsh", p.Shell())
	}
	if p.OS() != "windows" {
		t.Errorf("OS = %q", p.OS())
	}
	if p.RootDir() != `C:\` {
		t.Errorf("RootDir = %q", p.RootDir())
	}
	if p.ConfigDir() != `.claude` {
		t.Errorf("ConfigDir = %q", p.ConfigDir())
	}
}

func TestWindows_PathJoin(t *testing.T) {
	p := claude.ExportNewWindowsPlatform(`C:\workspace`, "pwsh", "")
	got := p.PathJoin(`C:\workspace`, ".yao", "config")
	if got != `C:\workspace\.yao\config` {
		t.Errorf("PathJoin = %q", got)
	}
}

func TestWindows_HomeEnv(t *testing.T) {
	p := claude.ExportNewWindowsPlatform(`C:\workspace`, "pwsh", "")
	env := p.HomeEnv(`C:\workspace`)
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
}

func TestWindows_HomeEnv_ShortPath(t *testing.T) {
	p := claude.ExportNewWindowsPlatform("X", "pwsh", "")
	env := p.HomeEnv("X")
	if env["HOME"] != "X" {
		t.Errorf("HOME = %q", env["HOME"])
	}
	if _, ok := env["HOMEDRIVE"]; ok {
		t.Error("should not set HOMEDRIVE for short path")
	}
}

func TestWindows_ShellCmd_Pwsh(t *testing.T) {
	p := claude.ExportNewWindowsPlatform(`C:\ws`, "pwsh", "")
	cmd := p.ShellCmd("echo hello")
	want := []string{"pwsh", "-NoProfile", "-Command", "echo hello"}
	if len(cmd) != 4 || cmd[0] != want[0] || cmd[1] != want[1] || cmd[2] != want[2] || cmd[3] != want[3] {
		t.Errorf("ShellCmd = %v, want %v", cmd, want)
	}
}

func TestWindows_ShellCmd_Powershell(t *testing.T) {
	p := claude.ExportNewWindowsPlatform(`C:\ws`, "powershell", "")
	cmd := p.ShellCmd("echo hello")
	if cmd[0] != "powershell" {
		t.Errorf("cmd[0] = %q", cmd[0])
	}
}

func TestWindows_ShellCmd_Cmd(t *testing.T) {
	p := claude.ExportNewWindowsPlatform(`C:\ws`, "cmd.exe", "")
	cmd := p.ShellCmd("echo hello")
	want := []string{"cmd.exe", "/C", "echo hello"}
	if len(cmd) != 3 || cmd[0] != want[0] || cmd[1] != want[1] || cmd[2] != want[2] {
		t.Errorf("ShellCmd = %v, want %v", cmd, want)
	}
}

func TestWindows_ShellCmd_DefaultFallback(t *testing.T) {
	p := claude.ExportNewWindowsPlatform(`C:\ws`, "unknown-shell", "")
	cmd := p.ShellCmd("echo hello")
	if cmd[0] != "pwsh" {
		t.Errorf("default should be pwsh, got %q", cmd[0])
	}
}

func TestWindows_KillCmd(t *testing.T) {
	p := claude.ExportNewWindowsPlatform(`C:\ws`, "pwsh", "")
	cmd := p.KillCmd("claude")
	if len(cmd) != 4 || cmd[0] != "pwsh" {
		t.Errorf("KillCmd = %v", cmd)
	}
	if !strings.Contains(cmd[3], "claude") || !strings.Contains(cmd[3], "taskkill") {
		t.Errorf("KillCmd body = %q", cmd[3])
	}
}

func TestWindows_KillSessionCmd(t *testing.T) {
	p := claude.ExportNewWindowsPlatform(`C:\ws`, "pwsh", "")
	cmd := p.KillSessionCmd("yao-robot_m1_e1")
	if len(cmd) != 4 || cmd[0] != "pwsh" {
		t.Errorf("KillSessionCmd = %v", cmd)
	}
	if !strings.Contains(cmd[3], "CommandLine") || !strings.Contains(cmd[3], "yao-robot_m1_e1") {
		t.Errorf("KillSessionCmd body = %q", cmd[3])
	}
}

func TestWindows_ListDirCmd(t *testing.T) {
	p := claude.ExportNewWindowsPlatform(`C:\ws`, "pwsh", "")
	cmd := p.ListDirCmd(`C:\ws\.claude`)
	if len(cmd) != 4 {
		t.Fatalf("ListDirCmd got %d args", len(cmd))
	}
	if !strings.Contains(cmd[3], "Get-ChildItem") {
		t.Errorf("ListDirCmd body = %q", cmd[3])
	}
}

func TestWindows_EnvPromptNote(t *testing.T) {
	p := claude.ExportNewWindowsPlatform(`C:\ws`, "pwsh", "")
	note := p.EnvPromptNote()
	if !strings.Contains(note, "Windows desktop") {
		t.Errorf("EnvPromptNote = %q", note)
	}
}

func TestWindows_BuildScript_NoPrompt(t *testing.T) {
	p := claude.ExportNewWindowsPlatform(`C:\workspace`, "pwsh", `C:\temp`)
	script, stdin := p.BuildScript(claude.ExportScriptInput{
		Args:       []string{"--verbose"},
		InputJSONL: `{"type":"user"}`,
		WorkDir:    `C:\workspace`,
		PromptFile: `C:\workspace\.yao\.system-prompt.txt`,
	})
	if !strings.Contains(script, "UTF8") {
		t.Error("script should set UTF8 encoding")
	}
	if !strings.Contains(script, "claude -p") {
		t.Error("script should contain claude -p")
	}
	if !strings.Contains(script, "'--verbose'") {
		t.Error("script should contain args")
	}
	if stdin == nil {
		t.Fatal("stdin should not be nil for Windows")
	}
	if !strings.Contains(string(stdin), `{"type":"user"}`) {
		t.Errorf("stdin = %q", stdin)
	}
}

func TestWindows_BuildScript_WithPrompt(t *testing.T) {
	p := claude.ExportNewWindowsPlatform(`C:\workspace`, "pwsh", `C:\temp`)
	script, stdin := p.BuildScript(claude.ExportScriptInput{
		Args:         []string{"--verbose"},
		SystemPrompt: "You are helpful",
		InputJSONL:   `{"type":"user"}`,
		WorkDir:      `C:\workspace`,
		PromptFile:   `C:\workspace\.yao\assistants\test\system-prompt.txt`,
	})
	if !strings.Contains(script, "WriteAllText") {
		t.Error("script should write prompt file")
	}
	if !strings.Contains(script, "You are helpful") {
		t.Error("script should contain system prompt")
	}
	if !strings.Contains(script, "--append-system-prompt-file") {
		t.Error("script should append prompt file arg")
	}
	if stdin == nil {
		t.Fatal("stdin should not be nil")
	}
}

func TestWindows_XauthoritySetup(t *testing.T) {
	p := claude.ExportNewWindowsPlatform(`C:\ws`, "pwsh", "")
	if got := p.XauthoritySetup(`C:\ws`); got != "" {
		t.Errorf("XauthoritySetup should be empty on Windows, got %q", got)
	}
}
