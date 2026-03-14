package claude

import (
	"fmt"
	"path"
	"strings"

	"github.com/yaoapp/yao/agent/sandbox/v2/types"
	infra "github.com/yaoapp/yao/sandbox/v2"
)

// osEnv captures OS-dependent paths and shell settings derived from the
// Computer's SystemInfo. All runner code should use osEnv instead of
// hardcoded Linux constants.
type osEnv struct {
	OS       string // "windows", "linux", "darwin", ...
	Shell    string // preferred shell binary: "bash", "pwsh", "cmd.exe", ...
	WorkDir  string // working directory on the target machine
	UserHome string // user home directory (empty if irrelevant)
	TempDir  string // system temp directory
}

func (e *osEnv) isWindows() bool {
	return strings.EqualFold(e.OS, "windows")
}

// resolveOSEnv builds an osEnv from the Computer's reported SystemInfo,
// falling back to SandboxConfig values where available, then to per-OS defaults.
func resolveOSEnv(computer infra.Computer, _ *types.SandboxConfig) *osEnv {
	sys := computer.ComputerInfo().System

	env := &osEnv{
		OS:      strings.ToLower(sys.OS),
		Shell:   sys.Shell,
		TempDir: sys.TempDir,
		WorkDir: computer.GetWorkDir(),
	}

	if env.TempDir == "" {
		env.TempDir = env.pathJoin(env.WorkDir, ".tmp")
	}

	return env
}

// shellCmd returns the command slice to run a script through the appropriate shell.
func (e *osEnv) shellCmd(script string) []string {
	shell := strings.ToLower(e.Shell)
	switch shell {
	case "pwsh":
		return []string{"pwsh", "-NoProfile", "-Command", script}
	case "powershell":
		return []string{"powershell", "-NoProfile", "-Command", script}
	case "cmd.exe", "cmd":
		return []string{"cmd.exe", "/C", script}
	default:
		return []string{"bash", "-c", script}
	}
}

// mkdirCmd returns a shell command string to create a directory (with parents).
func (e *osEnv) mkdirCmd(dir string) string {
	if e.isWindows() {
		return fmt.Sprintf(`if (!(Test-Path '%s')) { New-Item -ItemType Directory -Path '%s' -Force | Out-Null }`, dir, dir)
	}
	return fmt.Sprintf("mkdir -p %s", dir)
}

// listDirCmd returns a command slice to list directory contents.
func (e *osEnv) listDirCmd(dir string) []string {
	if e.isWindows() {
		return e.shellCmd(fmt.Sprintf("Get-ChildItem -Name '%s'", dir))
	}
	return []string{"ls", dir}
}

// killProcessCmd returns a command slice to kill processes matching a pattern.
func (e *osEnv) killProcessCmd(pattern string) []string {
	if e.isWindows() {
		script := fmt.Sprintf("Get-Process | Where-Object {$_.ProcessName -like '*%s*'} | Stop-Process -Force -ErrorAction SilentlyContinue", pattern)
		return e.shellCmd(script)
	}
	return []string{"sh", "-c", fmt.Sprintf("pkill -f '%s' || true", pattern)}
}

// rootDir returns the filesystem root for the target OS.
func (e *osEnv) rootDir() string {
	if e.isWindows() {
		return `C:\`
	}
	return "/"
}

// pathJoin joins path segments using the appropriate separator.
func (e *osEnv) pathJoin(parts ...string) string {
	if e.isWindows() {
		return strings.Join(parts, `\`)
	}
	return path.Join(parts...)
}

// buildCLIScript builds the complete CLI invocation script for the target OS.
// Returns (script, stdin) — on Linux stdin is nil (heredoc handles it),
// on Windows stdin contains inputJSONL bytes to pass via gRPC Stdin.
func (e *osEnv) buildCLIScript(args []string, systemPrompt, inputJSONL string) (string, []byte) {
	workDir := e.WorkDir
	promptFile := e.pathJoin(workDir, ".yao", ".system-prompt.txt")

	if e.isWindows() {
		return e.buildPowerShellScript(args, systemPrompt, inputJSONL, workDir, promptFile)
	}
	return e.buildBashScript(args, systemPrompt, inputJSONL, workDir, promptFile), nil
}

func (e *osEnv) buildBashScript(args []string, systemPrompt, inputJSONL, workDir, promptFile string) string {
	var b strings.Builder

	if e.UserHome != "" {
		b.WriteString(fmt.Sprintf("touch %s/.Xauthority 2>/dev/null; ", e.UserHome))
	}
	b.WriteString("touch \"$HOME/.Xauthority\" 2>/dev/null\n")

	if systemPrompt != "" {
		b.WriteString(fmt.Sprintf("mkdir -p %s/.yao\n", workDir))
		b.WriteString(fmt.Sprintf("cat << 'PROMPTEOF' > %s\n", promptFile))
		b.WriteString(systemPrompt)
		b.WriteString("\nPROMPTEOF\n")
		args = append(args, "--append-system-prompt-file", promptFile)
	}

	b.WriteString("cat << 'INPUTEOF' | claude -p")
	for _, arg := range args {
		b.WriteString(fmt.Sprintf(" %q", arg))
	}
	b.WriteString(" 2>&1\n")
	b.WriteString(inputJSONL)
	b.WriteString("\nINPUTEOF")

	return b.String()
}

// buildPowerShellScript builds a script that writes the system prompt file,
// then launches claude -p. inputJSONL is returned as stdin bytes to be passed
// directly via gRPC, bypassing PowerShell's encoding entirely.
func (e *osEnv) buildPowerShellScript(args []string, systemPrompt, inputJSONL, workDir, promptFile string) (string, []byte) {
	var b strings.Builder
	noBOM := "(New-Object System.Text.UTF8Encoding $false)"

	yaoDir := e.pathJoin(workDir, ".yao")
	b.WriteString(fmt.Sprintf("if (!(Test-Path '%s')) { New-Item -ItemType Directory -Path '%s' -Force | Out-Null }\n", yaoDir, yaoDir))

	if systemPrompt != "" {
		escaped := strings.ReplaceAll(systemPrompt, "'", "''")
		b.WriteString(fmt.Sprintf("[IO.File]::WriteAllText('%s', @'\n%s\n'@, %s)\n", promptFile, escaped, noBOM))
		args = append(args, "--append-system-prompt-file", promptFile)
	}

	b.WriteString("claude -p")
	for _, arg := range args {
		b.WriteString(fmt.Sprintf(" '%s'", strings.ReplaceAll(arg, "'", "''")))
	}

	return b.String(), []byte(inputJSONL + "\n")
}
