package claude

import (
	"fmt"
	"strings"
)

// windowsPlatform is a standalone implementation for Windows — no posixBase.
type windowsPlatform struct {
	workDir string
	shell   string
	tempDir string
}

func newWindowsPlatform(workDir, shell, tempDir string) *windowsPlatform {
	if shell == "" {
		shell = "pwsh"
	}
	if tempDir == "" {
		tempDir = workDir + `\.tmp`
	}
	return &windowsPlatform{workDir: workDir, shell: shell, tempDir: tempDir}
}

func (w *windowsPlatform) OS() string                      { return "windows" }
func (w *windowsPlatform) Shell() string                   { return w.shell }
func (w *windowsPlatform) RootDir() string                 { return `C:\` }
func (w *windowsPlatform) ConfigDir() string               { return `.claude` }
func (w *windowsPlatform) XauthoritySetup(_ string) string { return "" }

func (w *windowsPlatform) PathJoin(parts ...string) string {
	return strings.Join(parts, `\`)
}

// HomeEnv sets all four HOME-related variables to prevent Windows path
// resolution issues. The HOME variable is critical: without it, tools like
// Git for Windows inherit the host HOME, causing ~ to expand to the wrong
// directory. See: anthropics/claude-code#13138
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

func (w *windowsPlatform) EnvPromptNote() string {
	return `
- **Desktop Environment**: You have full access to the Windows desktop (GUI applications, browsers, etc.)
- **Important**: When you launch GUI applications (browsers, editors, etc.), do NOT close them unless explicitly asked — the user expects them to remain open`
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
		"Get-Process -ErrorAction SilentlyContinue | Where-Object {$_.ProcessName -like '*%s*'} | ForEach-Object { taskkill /F /T /PID $_.Id 2>$null }; "+
			"Get-Process -ErrorAction SilentlyContinue | Where-Object {$_.ProcessName -like '*%s*'} | Stop-Process -Force -ErrorAction SilentlyContinue",
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

func (w *windowsPlatform) ListDirCmd(dir string) []string {
	return w.ShellCmd(fmt.Sprintf("Get-ChildItem -Name '%s'", dir))
}

func (w *windowsPlatform) BuildScript(in scriptInput) (string, []byte) {
	var b strings.Builder
	noBOM := "(New-Object System.Text.UTF8Encoding $false)"

	b.WriteString("[Console]::InputEncoding = [System.Text.Encoding]::UTF8\n")
	b.WriteString("[Console]::OutputEncoding = [System.Text.Encoding]::UTF8\n")
	b.WriteString("$OutputEncoding = [System.Text.Encoding]::UTF8\n")

	b.WriteString("foreach ($d in (Get-ChildItem 'C:\\Users' -Directory -ErrorAction SilentlyContinue)) {\n")
	b.WriteString("  $p = Join-Path $d.FullName '.local\\bin'\n")
	b.WriteString("  if (Test-Path (Join-Path $p 'claude.exe')) { $env:PATH = \"$p;$env:PATH\"; break }\n")
	b.WriteString("}\n")
	b.WriteString("if ($env:APPDATA) { $env:PATH = \"$env:APPDATA\\npm;$env:PATH\" }\n")

	yaoDir := w.PathJoin(in.workDir, ".yao")
	b.WriteString(fmt.Sprintf("if (!(Test-Path '%s')) { New-Item -ItemType Directory -Path '%s' -Force | Out-Null }\n", yaoDir, yaoDir))

	if in.systemPrompt != "" {
		promptDir := in.promptFile[:strings.LastIndex(in.promptFile, `\`)]
		b.WriteString(fmt.Sprintf("if (!(Test-Path '%s')) { New-Item -ItemType Directory -Path '%s' -Force | Out-Null }\n", promptDir, promptDir))
		escaped := strings.ReplaceAll(in.systemPrompt, "'", "''")
		b.WriteString(fmt.Sprintf("[IO.File]::WriteAllText('%s', @'\n%s\n'@, %s)\n", in.promptFile, escaped, noBOM))
		in.args = append(in.args, "--append-system-prompt-file", in.promptFile)
	}

	b.WriteString("claude -p")
	for _, arg := range in.args {
		b.WriteString(fmt.Sprintf(" '%s'", strings.ReplaceAll(arg, "'", "''")))
	}

	return b.String(), []byte(in.inputJSONL + "\n")
}
