//go:build unit

package dsh_test

import (
	"strings"
	"testing"

	"github.com/yaoapp/yao/agent/sandbox/v2/dsh"
)

// ---------------------------------------------------------------------------
// POSIX
// ---------------------------------------------------------------------------

func TestPosix_OS(t *testing.T) {
	p := dsh.ExportNewPosixPlatform()
	if p.OS() != "posix" {
		t.Errorf("OS = %q, want posix", p.OS())
	}
}

func TestPosix_PathJoin(t *testing.T) {
	p := dsh.ExportNewPosixPlatform()
	got := p.PathJoin("/workspace", ".yao", "dsh", "cordis.yml")
	if got != "/workspace/.yao/dsh/cordis.yml" {
		t.Errorf("PathJoin = %q", got)
	}
}

func TestPosix_HomeEnv(t *testing.T) {
	p := dsh.ExportNewPosixPlatform()
	env := p.HomeEnv("/workspace")
	if env["HOME"] != "/workspace" {
		t.Errorf("HOME = %q", env["HOME"])
	}
	if len(env) != 1 {
		t.Errorf("expected 1 key, got %d", len(env))
	}
}

func TestPosix_ShellCmd(t *testing.T) {
	p := dsh.ExportNewPosixPlatform()
	cmd := p.ShellCmd("echo hello")
	if len(cmd) != 3 || cmd[0] != "bash" || cmd[1] != "-c" || cmd[2] != "echo hello" {
		t.Errorf("ShellCmd = %v", cmd)
	}
}

func TestPosix_KillCmd(t *testing.T) {
	p := dsh.ExportNewPosixPlatform()
	cmd := p.KillCmd("dsh-jsonrpc-agent")
	if len(cmd) != 3 || cmd[0] != "sh" {
		t.Errorf("KillCmd = %v", cmd)
	}
	if !strings.Contains(cmd[2], "pkill") || !strings.Contains(cmd[2], "dsh-jsonrpc-agent") {
		t.Errorf("KillCmd body = %q", cmd[2])
	}
}

func TestPosix_KillSessionCmd(t *testing.T) {
	p := dsh.ExportNewPosixPlatform()
	cmd := p.KillSessionCmd("dsh-chat123")
	if len(cmd) != 3 || cmd[0] != "sh" {
		t.Errorf("KillSessionCmd = %v", cmd)
	}
	if !strings.Contains(cmd[2], "pkill -9 -f") || !strings.Contains(cmd[2], "dsh-chat123") {
		t.Errorf("KillSessionCmd body = %q", cmd[2])
	}
}

func TestPosix_BuildScript_ContainsHeredoc(t *testing.T) {
	p := dsh.ExportNewPosixPlatform()
	script, stdin := p.BuildScript(dsh.ExportScriptInput{
		CordisYAML:   "- id: test\n  name: test-plugin\n",
		ConfigFile:   "/workspace/.yao/dsh/cordis.yml",
		InputJSONRPC: `{"jsonrpc":"2.0","id":1,"method":"initialize"}`,
	})

	if stdin == nil {
		t.Error("POSIX stdin should carry JSON-RPC input")
	}
	if !strings.Contains(string(stdin), "initialize") {
		t.Error("stdin should contain JSON-RPC input")
	}
	if !strings.Contains(script, "CORDISEOF") {
		t.Error("script should contain CORDISEOF heredoc")
	}
	if !strings.Contains(script, "exec tai dsh --config") {
		t.Error("script should use exec for stdin passthrough")
	}
	if !strings.Contains(script, "test-plugin") {
		t.Error("script should contain cordis YAML content")
	}
}

func TestPosix_BuildScript_QuotesConfigPath(t *testing.T) {
	p := dsh.ExportNewPosixPlatform()
	script, _ := p.BuildScript(dsh.ExportScriptInput{
		CordisYAML:   "test",
		ConfigFile:   "/workspace/path with spaces/.yao/dsh/cordis.yml",
		InputJSONRPC: "{}",
	})
	if !strings.Contains(script, `"/workspace/path with spaces/.yao/dsh/cordis.yml"`) {
		t.Errorf("script should quote config path: %s", script)
	}
}

func TestPosix_BuildScript_SetE(t *testing.T) {
	p := dsh.ExportNewPosixPlatform()
	script, _ := p.BuildScript(dsh.ExportScriptInput{
		CordisYAML:   "test",
		ConfigFile:   "/workspace/.yao/dsh/cordis.yml",
		InputJSONRPC: "{}",
	})
	if !strings.Contains(script, "set -e") || !strings.Contains(script, "set +e") {
		t.Error("script should use set -e/+e around config write")
	}
	setEIdx := strings.Index(script, "set -e")
	setNoEIdx := strings.Index(script, "set +e")
	taiIdx := strings.Index(script, "tai dsh")
	if setEIdx >= setNoEIdx || setNoEIdx >= taiIdx {
		t.Errorf("ordering: set -e(%d) < set +e(%d) < tai dsh(%d)", setEIdx, setNoEIdx, taiIdx)
	}
}

// ---------------------------------------------------------------------------
// Windows
// ---------------------------------------------------------------------------

func TestWindows_OS(t *testing.T) {
	p := dsh.ExportNewWindowsPlatform("pwsh")
	if p.OS() != "windows" {
		t.Errorf("OS = %q, want windows", p.OS())
	}
}

func TestWindows_PathJoin(t *testing.T) {
	p := dsh.ExportNewWindowsPlatform("pwsh")
	got := p.PathJoin(`C:\workspace`, ".yao", "dsh", "cordis.yml")
	if got != `C:\workspace\.yao\dsh\cordis.yml` {
		t.Errorf("PathJoin = %q", got)
	}
}

func TestWindows_HomeEnv(t *testing.T) {
	p := dsh.ExportNewWindowsPlatform("pwsh")
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
	p := dsh.ExportNewWindowsPlatform("pwsh")
	env := p.HomeEnv("X")
	if env["HOME"] != "X" {
		t.Errorf("HOME = %q", env["HOME"])
	}
	if _, ok := env["HOMEDRIVE"]; ok {
		t.Error("should not set HOMEDRIVE for short path")
	}
}

func TestWindows_ShellCmd_Pwsh(t *testing.T) {
	p := dsh.ExportNewWindowsPlatform("pwsh")
	cmd := p.ShellCmd("echo hello")
	if len(cmd) != 4 || cmd[0] != "pwsh" || cmd[1] != "-NoProfile" || cmd[2] != "-Command" {
		t.Errorf("ShellCmd = %v", cmd)
	}
}

func TestWindows_ShellCmd_Powershell(t *testing.T) {
	p := dsh.ExportNewWindowsPlatform("powershell")
	cmd := p.ShellCmd("echo hello")
	if cmd[0] != "powershell" {
		t.Errorf("cmd[0] = %q", cmd[0])
	}
}

func TestWindows_ShellCmd_Default(t *testing.T) {
	p := dsh.ExportNewWindowsPlatform("")
	cmd := p.ShellCmd("echo hello")
	if cmd[0] != "pwsh" {
		t.Errorf("default should be pwsh, got %q", cmd[0])
	}
}

func TestWindows_KillCmd(t *testing.T) {
	p := dsh.ExportNewWindowsPlatform("pwsh")
	cmd := p.KillCmd("dsh-jsonrpc-agent")
	if len(cmd) != 4 || cmd[0] != "pwsh" {
		t.Errorf("KillCmd = %v", cmd)
	}
	if !strings.Contains(cmd[3], "dsh-jsonrpc-agent") || !strings.Contains(cmd[3], "taskkill") {
		t.Errorf("KillCmd body = %q", cmd[3])
	}
}

func TestWindows_KillSessionCmd(t *testing.T) {
	p := dsh.ExportNewWindowsPlatform("pwsh")
	cmd := p.KillSessionCmd("dsh-chat123")
	if len(cmd) != 4 || cmd[0] != "pwsh" {
		t.Errorf("KillSessionCmd = %v", cmd)
	}
	if !strings.Contains(cmd[3], "CommandLine") || !strings.Contains(cmd[3], "dsh-chat123") {
		t.Errorf("KillSessionCmd body = %q", cmd[3])
	}
}

func TestWindows_BuildScript_UsesStdin(t *testing.T) {
	p := dsh.ExportNewWindowsPlatform("pwsh")
	script, stdin := p.BuildScript(dsh.ExportScriptInput{
		CordisYAML:   "- id: test\n  name: test-plugin\n",
		ConfigFile:   `C:\workspace\.yao\dsh\cordis.yml`,
		InputJSONRPC: `{"jsonrpc":"2.0","id":1,"method":"initialize"}`,
	})

	if stdin == nil {
		t.Fatal("Windows stdin should not be nil")
	}
	if !strings.Contains(string(stdin), "initialize") {
		t.Errorf("stdin = %q", stdin)
	}
	if !strings.Contains(script, "UTF8") {
		t.Error("script should set UTF8 encoding")
	}
	if !strings.Contains(script, "tai dsh --config") {
		t.Error("script should contain tai dsh command")
	}
	if !strings.Contains(script, "WriteAllText") {
		t.Error("script should write cordis config via WriteAllText")
	}
	if !strings.Contains(script, "test-plugin") {
		t.Error("script should contain cordis YAML content")
	}
}

func TestWindows_BuildScript_NoPipeInput(t *testing.T) {
	p := dsh.ExportNewWindowsPlatform("pwsh")
	script, _ := p.BuildScript(dsh.ExportScriptInput{
		CordisYAML:   "test",
		ConfigFile:   `C:\ws\cordis.yml`,
		InputJSONRPC: "{}",
	})
	if strings.Contains(script, "$input |") {
		t.Error("script should NOT use $input pipe (stdin via WithStdin)")
	}
}

func TestWindows_BuildScript_PreservesSingleQuotes(t *testing.T) {
	p := dsh.ExportNewWindowsPlatform("pwsh")
	script, _ := p.BuildScript(dsh.ExportScriptInput{
		CordisYAML:   "  name: '@deepseek-ai/dsh-yaoapp-jsonrpc-stream'",
		ConfigFile:   `C:\ws\cordis.yml`,
		InputJSONRPC: "{}",
	})
	if !strings.Contains(script, "'@deepseek-ai/dsh-yaoapp-jsonrpc-stream'") {
		t.Error("here-string content must preserve single quotes verbatim")
	}
	if strings.Contains(script, "''@deepseek-ai/dsh-yaoapp-jsonrpc-stream''") {
		t.Error("single quotes must not be doubled inside a PowerShell here-string")
	}
}

func TestWindows_BuildScript_SearchesTaiExe(t *testing.T) {
	p := dsh.ExportNewWindowsPlatform("pwsh")
	script, _ := p.BuildScript(dsh.ExportScriptInput{
		CordisYAML:   "test",
		ConfigFile:   `C:\ws\cordis.yml`,
		InputJSONRPC: "{}",
	})
	if !strings.Contains(script, "tai.exe") {
		t.Error("script should search for tai.exe in PATH")
	}
	if !strings.Contains(script, "Get-ChildItem") {
		t.Error("script should search C:\\Users directories")
	}
	if !strings.Contains(script, ".local\\bin") {
		t.Error("script should search .local\\bin")
	}
}

// ---------------------------------------------------------------------------
// GracefulKillSessionCmd tests
// ---------------------------------------------------------------------------

func TestPosix_GracefulKillSessionCmd(t *testing.T) {
	p := dsh.ExportNewPosixPlatform()
	cmd := p.GracefulKillSessionCmd("dsh-chat123")
	if len(cmd) != 3 || cmd[0] != "sh" {
		t.Errorf("GracefulKillSessionCmd = %v", cmd)
	}
	if !strings.Contains(cmd[2], "pkill -TERM -f") || !strings.Contains(cmd[2], "dsh-chat123") {
		t.Errorf("GracefulKillSessionCmd body = %q, want SIGTERM", cmd[2])
	}
}

func TestWindows_GracefulKillSessionCmd(t *testing.T) {
	p := dsh.ExportNewWindowsPlatform("pwsh")
	cmd := p.GracefulKillSessionCmd("dsh-chat123")
	if len(cmd) == 0 {
		t.Fatal("GracefulKillSessionCmd returned empty")
	}
	if !strings.Contains(strings.Join(cmd, " "), "dsh-chat123") {
		t.Error("GracefulKillSessionCmd should reference session name")
	}
}
