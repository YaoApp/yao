package dsh

import (
	"fmt"
	"strings"

	"github.com/yaoapp/yao/agent/sandbox/v2/shared"
)

type winPlatform struct {
	shell string
}

func newWindowsPlatform(shell string) *winPlatform {
	if shell == "" {
		shell = "pwsh"
	}
	return &winPlatform{shell: shell}
}

func (w *winPlatform) OS() string { return "windows" }

func (w *winPlatform) HomeEnv(workDir string) map[string]string {
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

func (w *winPlatform) PathJoin(parts ...string) string {
	return strings.Join(parts, `\`)
}

func (w *winPlatform) ShellCmd(script string) []string {
	shell := strings.ToLower(w.shell)
	switch shell {
	case "pwsh":
		return []string{"pwsh", "-NoProfile", "-Command", script}
	case "powershell":
		return []string{"powershell", "-NoProfile", "-Command", script}
	default:
		return []string{"pwsh", "-NoProfile", "-Command", script}
	}
}

func (w *winPlatform) KillCmd(pattern string) []string {
	script := fmt.Sprintf(
		"Get-Process -ErrorAction SilentlyContinue | Where-Object {$_.ProcessName -like '*%s*'} | ForEach-Object { taskkill /F /T /PID $_.Id 2>$null }; "+
			"Get-Process -ErrorAction SilentlyContinue | Where-Object {$_.ProcessName -like '*%s*'} | Stop-Process -Force -ErrorAction SilentlyContinue",
		pattern, pattern)
	return w.ShellCmd(script)
}

func (w *winPlatform) KillSessionCmd(sessionName string) []string {
	script := fmt.Sprintf(
		"Get-Process -ErrorAction SilentlyContinue | "+
			"Where-Object { $_.CommandLine -like '*%s*' } | "+
			"ForEach-Object { taskkill /F /T /PID $_.Id 2>$null }",
		sessionName)
	return w.ShellCmd(script)
}

func (w *winPlatform) GracefulKillSessionCmd(sessionName string) []string {
	return w.KillSessionCmd(sessionName)
}

// BuildScript generates a PowerShell script that:
// 1. Searches for tai.exe in user .local\bin directories
// 2. Writes cordis.yml to the config file path
// 3. Launches `tai dsh --config <path>` with JSON-RPC input via stdin
func (w *winPlatform) BuildScript(in scriptInput) (string, []byte) {
	var b strings.Builder

	b.WriteString("[Console]::InputEncoding = [System.Text.Encoding]::UTF8\n")
	b.WriteString("[Console]::OutputEncoding = [System.Text.Encoding]::UTF8\n")
	b.WriteString("$OutputEncoding = [System.Text.Encoding]::UTF8\n")

	shared.PsSearchExe(&b, "tai.exe")
	shared.PsWriteFileUTF8(&b, in.configFile, in.cordisYAML)

	b.WriteString(fmt.Sprintf("tai dsh --config %s", shared.PsQuoteArg(in.configFile)))

	return b.String(), []byte(in.inputJSONRPC + "\n")
}
