package opencode

import (
	"fmt"
	"path"
	"strings"

	infra "github.com/yaoapp/yao/sandbox/v2"
)

// platform encapsulates OS-dependent behaviors for the target environment.
type platform interface {
	OS() string
	Shell() string
	HomeEnv(workDir string) map[string]string
	PathJoin(parts ...string) string
	ShellCmd(script string) []string
	KillCmd(pattern string) []string
	KillSessionCmd(sessionName string) []string
}

type posixBase struct {
	os      string
	workDir string
	shell   string
}

func (b *posixBase) OS() string                      { return b.os }
func (b *posixBase) Shell() string                   { return b.shell }
func (b *posixBase) PathJoin(parts ...string) string { return path.Join(parts...) }

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

func resolvePlatform(computer infra.Computer) platform {
	sys := computer.ComputerInfo().System
	osName := strings.ToLower(sys.OS)
	workDir := computer.GetWorkDir()
	shell := sys.Shell

	if osName == "windows" {
		return newWindowsPlatform(workDir, shell)
	}

	base := posixBase{os: osName, workDir: workDir, shell: shell}
	if base.shell == "" {
		base.shell = "bash"
	}
	if base.os == "" {
		base.os = "linux"
	}
	return &base
}
