package dsh

import (
	"fmt"
	"path"
	"strings"
)

type posixPlatform struct{}

func newPosixPlatform() *posixPlatform { return &posixPlatform{} }

func (p *posixPlatform) OS() string { return "posix" }

func (p *posixPlatform) HomeEnv(workDir string) map[string]string {
	return map[string]string{"HOME": workDir}
}

func (p *posixPlatform) PathJoin(parts ...string) string {
	return path.Join(parts...)
}

func (p *posixPlatform) ShellCmd(script string) []string {
	return []string{"bash", "-c", script}
}

func (p *posixPlatform) KillCmd(pattern string) []string {
	return []string{"sh", "-c", fmt.Sprintf("pkill -f '%s' || true", pattern)}
}

func (p *posixPlatform) KillSessionCmd(sessionName string) []string {
	return []string{"sh", "-c", fmt.Sprintf("pkill -9 -f '%s' || true", sessionName)}
}

func (p *posixPlatform) GracefulKillSessionCmd(sessionName string) []string {
	return []string{"sh", "-c", fmt.Sprintf("pkill -TERM -f '%s' || true", sessionName)}
}

// BuildScript generates a bash script that:
// 1. Writes cordis.yml to the config file path
// 2. Uses `exec tai dsh` to replace the shell process, inheriting stdin directly
func (p *posixPlatform) BuildScript(in scriptInput) (string, []byte) {
	var s strings.Builder

	s.WriteString("set -e\n")
	s.WriteString(fmt.Sprintf("mkdir -p \"$(dirname %q)\"\n", in.configFile))
	s.WriteString(fmt.Sprintf("cat << 'CORDISEOF' > %q\n", in.configFile))
	s.WriteString(in.cordisYAML)
	s.WriteString("\nCORDISEOF\n")
	s.WriteString("set +e\n")

	s.WriteString(fmt.Sprintf("exec tai dsh --config %q\n", in.configFile))

	return s.String(), []byte(in.inputJSONRPC + "\n")
}
