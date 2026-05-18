package claude

import (
	"fmt"
	"path"
	"strings"

	infra "github.com/yaoapp/yao/sandbox/v2"
)

// platform encapsulates all OS-dependent behaviors for the target environment.
// Upper-layer business code uses this interface exclusively, with zero platform
// branching (no if isWindows() checks).
type platform interface {
	OS() string
	Shell() string
	HomeEnv(workDir string) map[string]string
	EnvPromptNote() string
	PathJoin(parts ...string) string
	RootDir() string
	ShellCmd(script string) []string
	KillCmd(pattern string) []string
	KillSessionCmd(sessionName string) []string
	ListDirCmd(dir string) []string
	ConfigDir() string
	XauthoritySetup(workDir string) string
	BuildScript(input scriptInput) (script string, stdin []byte)
}

type scriptInput struct {
	args         []string
	systemPrompt string
	inputJSONL   string
	workDir      string
	promptFile   string
}

// posixBase provides shared POSIX-compatible implementation (~80% of methods)
// for macOS and Linux. darwinPlatform and linuxPlatform embed this.
type posixBase struct {
	os       string
	workDir  string
	shell    string
	tempDir  string
	userHome string
}

func (b *posixBase) OS() string                      { return b.os }
func (b *posixBase) Shell() string                   { return b.shell }
func (b *posixBase) PathJoin(parts ...string) string { return path.Join(parts...) }
func (b *posixBase) RootDir() string                 { return "/" }
func (b *posixBase) ConfigDir() string               { return ".config/claude" }
func (b *posixBase) XauthoritySetup(_ string) string { return "" }

func (b *posixBase) HomeEnv(workDir string) map[string]string {
	return map[string]string{"HOME": workDir}
}

func (b *posixBase) ShellCmd(script string) []string {
	return []string{"bash", "-c", script}
}

func (b *posixBase) KillCmd(pattern string) []string {
	return []string{"sh", "-c", fmt.Sprintf("pkill -f '%s' || true", pattern)}
}

func (b *posixBase) KillSessionCmd(sessionName string) []string {
	return []string{"sh", "-c", fmt.Sprintf("pkill -9 -f '%s' || true", sessionName)}
}

func (b *posixBase) ListDirCmd(dir string) []string {
	return []string{"ls", dir}
}

// buildBashScript is the shared bash script builder for macOS/Linux.
//
// When a system prompt is present, the script uses `set -e` to ensure that
// any failure in directory creation or prompt file writing aborts the entire
// script before Claude CLI is launched. This prevents silent fallback to
// running without a system prompt.
func (b *posixBase) buildBashScript(in scriptInput, xauthCmd string) string {
	var s strings.Builder

	if xauthCmd != "" {
		s.WriteString(xauthCmd)
	}

	if in.systemPrompt != "" {
		s.WriteString("set -e\n")
		s.WriteString(fmt.Sprintf("mkdir -p \"$(dirname %q)\"\n", in.promptFile))
		s.WriteString(fmt.Sprintf("cat << 'PROMPTEOF' > %s\n", in.promptFile))
		s.WriteString(in.systemPrompt)
		s.WriteString("\nPROMPTEOF\n")
		s.WriteString("set +e\n")
		in.args = append(in.args, "--append-system-prompt-file", in.promptFile)
	}

	s.WriteString("cat << 'INPUTEOF' | claude -p")
	for _, arg := range in.args {
		s.WriteString(fmt.Sprintf(" %q", arg))
	}
	s.WriteString("\n")
	s.WriteString(in.inputJSONL)
	s.WriteString("\nINPUTEOF")

	return s.String()
}

// resolvePlatform creates the appropriate platform implementation based on
// the Computer's reported OS.
func resolvePlatform(computer infra.Computer) platform {
	sys := computer.ComputerInfo().System
	osName := strings.ToLower(sys.OS)
	workDir := computer.GetWorkDir()
	shell := sys.Shell
	tempDir := sys.TempDir

	base := posixBase{
		os: osName, workDir: workDir,
		shell: shell, tempDir: tempDir,
	}
	if base.shell == "" {
		base.shell = "bash"
	}
	if base.tempDir == "" {
		base.tempDir = path.Join(workDir, ".tmp")
	}

	// DISPLAY and system HOME are not yet available via SystemInfo.
	// For Desktop Linux containers (VNC/noVNC), these will be populated
	// once infra reports environment variables. Until then, the Xauthority
	// copy will be a no-op — same as the old code's behavior.
	hasDisplay := false
	sysHome := ""

	switch osName {
	case "darwin":
		return &darwinPlatform{posixBase: base}
	case "windows":
		return newWindowsPlatform(workDir, shell, tempDir)
	default:
		return &linuxPlatform{
			posixBase:  base,
			hasDisplay: hasDisplay,
			sysHome:    sysHome,
		}
	}
}
