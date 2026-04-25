package opencode

import (
	"fmt"
	"strings"
)

// windowsPlatform implements the platform interface for Windows containers.
// Aligned with the Claude runner's plat_win.go: uses PowerShell for shell
// commands, backslash path joins, and full HOME-related env vars.
type windowsPlatform struct {
	workDir string
	shell   string
}

func newWindowsPlatform(workDir, shell string) *windowsPlatform {
	if shell == "" {
		shell = "pwsh"
	}
	return &windowsPlatform{workDir: workDir, shell: shell}
}

func (w *windowsPlatform) OS() string    { return "windows" }
func (w *windowsPlatform) Shell() string { return w.shell }

func (w *windowsPlatform) PathJoin(parts ...string) string {
	return strings.Join(parts, `\`)
}

// HomeEnv sets HOME, USERPROFILE, HOMEDRIVE, and HOMEPATH so that Git for
// Windows, npm, and other tools resolve ~ correctly inside the container.
// See Claude runner plat_win.go and anthropics/claude-code#13138.
func (w *windowsPlatform) HomeEnv(workDir string) map[string]string {
	env := map[string]string{
		"HOME":        workDir,
		"USERPROFILE": workDir,
	}
	if len(workDir) >= 2 && workDir[1] == ':' {
		env["HOMEDRIVE"] = workDir[:2]
		env["HOMEPATH"] = workDir[2:]
	}
	return env
}

func (w *windowsPlatform) ShellCmd(script string) []string {
	shell := strings.ToLower(w.shell)
	switch shell {
	case "pwsh":
		return []string{"pwsh", "-NoProfile", "-Command", script}
	case "powershell":
		return []string{"powershell", "-NoProfile", "-Command", script}
	case "cmd.exe", "cmd":
		return []string{"cmd.exe", "/C", script}
	default:
		return []string{"pwsh", "-NoProfile", "-Command", script}
	}
}

func (w *windowsPlatform) KillCmd(pattern string) []string {
	script := fmt.Sprintf(
		"Get-Process -ErrorAction SilentlyContinue | Where-Object {$_.ProcessName -like '*%s*'} | "+
			"ForEach-Object { taskkill /F /T /PID $_.Id 2>$null }; "+
			"Get-Process -ErrorAction SilentlyContinue | Where-Object {$_.ProcessName -like '*%s*'} | "+
			"Stop-Process -Force -ErrorAction SilentlyContinue",
		pattern, pattern)
	return w.ShellCmd(script)
}

func (w *windowsPlatform) KillSessionCmd(sessionName string) []string {
	script := fmt.Sprintf(
		"Get-Process -ErrorAction SilentlyContinue | "+
			"Where-Object { $_.CommandLine -like '*%s*' } | "+
			"ForEach-Object { taskkill /F /T /PID $_.Id 2>$null }",
		sessionName)
	return w.ShellCmd(script)
}
