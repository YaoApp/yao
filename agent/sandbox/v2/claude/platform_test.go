package claude

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestPosixBase(os string) posixBase {
	return posixBase{
		os:      os,
		workDir: "/workspace",
		shell:   "bash",
		tempDir: "/tmp",
	}
}

func TestPosixBase_Accessors(t *testing.T) {
	b := newTestPosixBase("linux")
	assert.Equal(t, "linux", b.OS())
	assert.Equal(t, "bash", b.Shell())
	assert.Equal(t, "/", b.RootDir())
	assert.Equal(t, ".config/claude", b.ConfigDir())
}

func TestPosixBase_PathJoin(t *testing.T) {
	b := newTestPosixBase("linux")
	assert.Equal(t, "/workspace/.yao/config", b.PathJoin("/workspace", ".yao", "config"))
	assert.Equal(t, "a/b/c", b.PathJoin("a", "b", "c"))
}

func TestPosixBase_HomeEnv(t *testing.T) {
	b := newTestPosixBase("linux")
	env := b.HomeEnv("/workspace/data")
	assert.Equal(t, "/workspace/data", env["HOME"])
	assert.Len(t, env, 1)
}

func TestPosixBase_ShellCmd(t *testing.T) {
	b := newTestPosixBase("linux")
	cmd := b.ShellCmd("echo hello")
	require.Len(t, cmd, 3)
	assert.Equal(t, "bash", cmd[0])
	assert.Equal(t, "-c", cmd[1])
	assert.Equal(t, "echo hello", cmd[2])
}

func TestPosixBase_KillCmd(t *testing.T) {
	b := newTestPosixBase("linux")
	cmd := b.KillCmd("claude")
	require.Len(t, cmd, 3)
	assert.Equal(t, "sh", cmd[0])
	assert.Contains(t, cmd[2], "pkill")
	assert.Contains(t, cmd[2], "claude")
}

func TestPosixBase_ListDirCmd(t *testing.T) {
	b := newTestPosixBase("linux")
	cmd := b.ListDirCmd("/workspace/.claude")
	require.Len(t, cmd, 2)
	assert.Equal(t, "ls", cmd[0])
	assert.Equal(t, "/workspace/.claude", cmd[1])
}

func TestPosixBase_XauthoritySetup(t *testing.T) {
	b := newTestPosixBase("linux")
	assert.Empty(t, b.XauthoritySetup("/workspace"))
}

func TestPosixBase_BuildBashScript_NoPrompt(t *testing.T) {
	b := newTestPosixBase("linux")
	in := scriptInput{
		args:       []string{"--verbose", "--output-format", "stream-json"},
		inputJSONL: `{"type":"user","message":{"role":"user","content":"hello"}}`,
		workDir:    "/workspace",
		promptFile: "/workspace/.yao/.system-prompt.txt",
	}
	script := b.buildBashScript(in, "")
	assert.Contains(t, script, "cat << 'INPUTEOF' | claude -p")
	assert.Contains(t, script, "--verbose")
	assert.Contains(t, script, "INPUTEOF")
	assert.NotContains(t, script, "PROMPTEOF")
	assert.NotContains(t, script, "set -e", "set -e should not be present when no prompt is written")
}

func TestPosixBase_BuildBashScript_WithPrompt(t *testing.T) {
	b := newTestPosixBase("linux")
	in := scriptInput{
		args:         []string{"--verbose"},
		systemPrompt: "You are a helpful assistant.",
		inputJSONL:   `{"type":"user"}`,
		workDir:      "/workspace",
		promptFile:   "/workspace/.yao/assistants/test-id/system-prompt.txt",
	}
	script := b.buildBashScript(in, "")
	assert.Contains(t, script, "mkdir -p")
	assert.Contains(t, script, "PROMPTEOF")
	assert.Contains(t, script, "You are a helpful assistant.")
	assert.Contains(t, script, "--append-system-prompt-file")

	promptIdx := strings.Index(script, "PROMPTEOF")
	claudeIdx := strings.Index(script, "claude -p")
	assert.True(t, strings.Contains(script, "set -e"), "script should enable set -e before prompt write")
	assert.True(t, strings.Contains(script, "set +e"), "script should disable set -e before claude command")
	setEIdx := strings.Index(script, "set -e")
	setNoEIdx := strings.Index(script, "set +e")
	assert.Less(t, setEIdx, promptIdx, "set -e should come before PROMPTEOF")
	assert.Less(t, promptIdx, setNoEIdx, "set +e should come after PROMPTEOF")
	assert.Less(t, setNoEIdx, claudeIdx, "set +e should come before claude -p")
}

func TestPosixBase_BuildBashScript_WithXauth(t *testing.T) {
	b := newTestPosixBase("linux")
	in := scriptInput{
		args:       []string{"--verbose"},
		inputJSONL: `{"type":"user"}`,
		workDir:    "/workspace",
		promptFile: "/workspace/.yao/.system-prompt.txt",
	}
	xauth := "cp /root/.Xauthority /workspace/.Xauthority\n"
	script := b.buildBashScript(in, xauth)
	assert.True(t, strings.HasPrefix(script, "cp /root/.Xauthority"),
		"script should start with xauth command")
}

// --- Darwin ---

func TestDarwin_EnvPromptNote(t *testing.T) {
	p := &darwinPlatform{posixBase: newTestPosixBase("darwin")}
	note := p.EnvPromptNote()
	assert.Contains(t, note, "macOS desktop")
	assert.Contains(t, note, "GUI applications")
}

func TestDarwin_BuildScript(t *testing.T) {
	p := &darwinPlatform{posixBase: newTestPosixBase("darwin")}
	script, stdin := p.BuildScript(scriptInput{
		args:       []string{"--verbose"},
		inputJSONL: `{"type":"user"}`,
		workDir:    "/workspace",
		promptFile: "/workspace/.yao/.system-prompt.txt",
	})
	assert.Contains(t, script, "claude -p")
	assert.Nil(t, stdin)
}

// --- Linux ---

func TestLinux_XauthoritySetup_Headless(t *testing.T) {
	p := &linuxPlatform{
		posixBase:  newTestPosixBase("linux"),
		hasDisplay: false,
		sysHome:    "/root",
	}
	assert.Empty(t, p.XauthoritySetup("/workspace"))
}

func TestLinux_XauthoritySetup_Desktop(t *testing.T) {
	p := &linuxPlatform{
		posixBase:  newTestPosixBase("linux"),
		hasDisplay: true,
		sysHome:    "/root",
	}
	cmd := p.XauthoritySetup("/workspace")
	assert.Contains(t, cmd, "/root/.Xauthority")
	assert.Contains(t, cmd, "/workspace/.Xauthority")
	assert.Contains(t, cmd, "cp")
}

func TestLinux_XauthoritySetup_NoSysHome(t *testing.T) {
	p := &linuxPlatform{
		posixBase:  newTestPosixBase("linux"),
		hasDisplay: true,
		sysHome:    "",
	}
	assert.Empty(t, p.XauthoritySetup("/workspace"))
}

func TestLinux_EnvPromptNote_Desktop(t *testing.T) {
	p := &linuxPlatform{posixBase: newTestPosixBase("linux"), hasDisplay: true}
	note := p.EnvPromptNote()
	assert.Contains(t, note, "VNC")
}

func TestLinux_EnvPromptNote_Headless(t *testing.T) {
	p := &linuxPlatform{posixBase: newTestPosixBase("linux"), hasDisplay: false}
	assert.Empty(t, p.EnvPromptNote())
}

func TestLinux_BuildScript_Desktop(t *testing.T) {
	p := &linuxPlatform{
		posixBase:  newTestPosixBase("linux"),
		hasDisplay: true,
		sysHome:    "/root",
	}
	script, stdin := p.BuildScript(scriptInput{
		args:       []string{"--verbose"},
		inputJSONL: `{"type":"user"}`,
		workDir:    "/workspace",
		promptFile: "/workspace/.yao/.system-prompt.txt",
	})
	assert.Contains(t, script, ".Xauthority")
	assert.Nil(t, stdin)
}

// --- Windows ---

func TestWindows_NewDefaults(t *testing.T) {
	w := newWindowsPlatform(`C:\workspace`, "", "")
	assert.Equal(t, "pwsh", w.Shell())
	assert.Equal(t, `C:\workspace\.tmp`, w.tempDir)
}

func TestWindows_Accessors(t *testing.T) {
	w := newWindowsPlatform(`C:\workspace`, "pwsh", `C:\temp`)
	assert.Equal(t, "windows", w.OS())
	assert.Equal(t, `C:\`, w.RootDir())
	assert.Equal(t, `.claude`, w.ConfigDir())
	assert.Empty(t, w.XauthoritySetup(`C:\workspace`))
}

func TestWindows_PathJoin(t *testing.T) {
	w := newWindowsPlatform(`C:\workspace`, "pwsh", "")
	assert.Equal(t, `C:\workspace\.yao\config`, w.PathJoin(`C:\workspace`, ".yao", "config"))
}

func TestWindows_HomeEnv(t *testing.T) {
	w := newWindowsPlatform(`C:\workspace`, "pwsh", "")
	env := w.HomeEnv(`C:\workspace`)
	assert.Equal(t, `C:\workspace`, env["HOME"])
	assert.Equal(t, `C:\workspace`, env["USERPROFILE"])
	assert.Equal(t, `C:`, env["HOMEDRIVE"])
	assert.Equal(t, `\workspace`, env["HOMEPATH"])
}

func TestWindows_HomeEnv_ShortPath(t *testing.T) {
	w := newWindowsPlatform("X", "pwsh", "")
	env := w.HomeEnv("X")
	assert.Equal(t, "X", env["HOME"])
	assert.Equal(t, "X", env["USERPROFILE"])
	_, hasDrive := env["HOMEDRIVE"]
	assert.False(t, hasDrive, "should not set HOMEDRIVE for paths without drive letter")
}

func TestWindows_ShellCmd_Pwsh(t *testing.T) {
	w := newWindowsPlatform(`C:\ws`, "pwsh", "")
	cmd := w.ShellCmd("echo hello")
	assert.Equal(t, []string{"pwsh", "-NoProfile", "-Command", "echo hello"}, cmd)
}

func TestWindows_ShellCmd_Powershell(t *testing.T) {
	w := newWindowsPlatform(`C:\ws`, "powershell", "")
	cmd := w.ShellCmd("echo hello")
	assert.Equal(t, "powershell", cmd[0])
}

func TestWindows_ShellCmd_Cmd(t *testing.T) {
	w := newWindowsPlatform(`C:\ws`, "cmd.exe", "")
	cmd := w.ShellCmd("echo hello")
	assert.Equal(t, []string{"cmd.exe", "/C", "echo hello"}, cmd)
}

func TestPosixBase_KillSessionCmd(t *testing.T) {
	b := newTestPosixBase("linux")
	cmd := b.KillSessionCmd("yao-robot_m1_e1")
	require.Len(t, cmd, 3)
	assert.Equal(t, "sh", cmd[0])
	assert.Equal(t, "-c", cmd[1])
	assert.Contains(t, cmd[2], "pkill -9 -f")
	assert.Contains(t, cmd[2], "yao-robot_m1_e1")
	assert.Contains(t, cmd[2], "|| true")
}

func TestWindows_KillSessionCmd(t *testing.T) {
	w := newWindowsPlatform(`C:\ws`, "pwsh", "")
	cmd := w.KillSessionCmd("yao-robot_m1_e1")
	require.Len(t, cmd, 4)
	assert.Equal(t, "pwsh", cmd[0])
	assert.Contains(t, cmd[3], "CommandLine")
	assert.Contains(t, cmd[3], "yao-robot_m1_e1")
	assert.Contains(t, cmd[3], "taskkill")
}

func TestWindows_KillCmd(t *testing.T) {
	w := newWindowsPlatform(`C:\ws`, "pwsh", "")
	cmd := w.KillCmd("claude")
	require.Len(t, cmd, 4)
	assert.Equal(t, "pwsh", cmd[0])
	assert.Contains(t, cmd[3], "claude")
	assert.Contains(t, cmd[3], "taskkill")
}

func TestWindows_ListDirCmd(t *testing.T) {
	w := newWindowsPlatform(`C:\ws`, "pwsh", "")
	cmd := w.ListDirCmd(`C:\ws\.claude`)
	require.Len(t, cmd, 4)
	assert.Contains(t, cmd[3], "Get-ChildItem")
}

func TestWindows_BuildScript_NoPrompt(t *testing.T) {
	w := newWindowsPlatform(`C:\workspace`, "pwsh", `C:\temp`)
	script, stdin := w.BuildScript(scriptInput{
		args:       []string{"--verbose"},
		inputJSONL: `{"type":"user"}`,
		workDir:    `C:\workspace`,
		promptFile: `C:\workspace\.yao\.system-prompt.txt`,
	})
	assert.Contains(t, script, "UTF8")
	assert.Contains(t, script, "claude -p")
	assert.Contains(t, script, "'--verbose'")
	require.NotNil(t, stdin)
	assert.Contains(t, string(stdin), `{"type":"user"}`)
}

func TestWindows_BuildScript_WithPrompt(t *testing.T) {
	w := newWindowsPlatform(`C:\workspace`, "pwsh", `C:\temp`)
	script, stdin := w.BuildScript(scriptInput{
		args:         []string{"--verbose"},
		systemPrompt: "You are helpful",
		inputJSONL:   `{"type":"user"}`,
		workDir:      `C:\workspace`,
		promptFile:   `C:\workspace\.yao\assistants\test\system-prompt.txt`,
	})
	assert.Contains(t, script, "WriteAllText")
	assert.Contains(t, script, "You are helpful")
	assert.Contains(t, script, "--append-system-prompt-file")
	require.NotNil(t, stdin)
}

func TestWindows_EnvPromptNote(t *testing.T) {
	w := newWindowsPlatform(`C:\ws`, "pwsh", "")
	note := w.EnvPromptNote()
	assert.Contains(t, note, "Windows desktop")
}
