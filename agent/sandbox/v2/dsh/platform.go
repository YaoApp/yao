package dsh

import (
	"strings"

	infra "github.com/yaoapp/yao/sandbox/v2"
)

// platform encapsulates OS-dependent behaviors for the DSH runner.
type platform interface {
	OS() string
	HomeEnv(workDir string) map[string]string
	PathJoin(parts ...string) string
	ShellCmd(script string) []string
	KillCmd(pattern string) []string
	KillSessionCmd(sessionName string) []string
	GracefulKillSessionCmd(sessionName string) []string
	BuildScript(input scriptInput) (script string, stdin []byte)
}

type scriptInput struct {
	cordisYAML   string
	configFile   string
	inputJSONRPC string
}

// resolvePlatform creates the appropriate platform implementation based on Computer info.
func resolvePlatform(computer infra.Computer) platform {
	info := computer.ComputerInfo()
	osName := strings.ToLower(info.System.OS)
	switch osName {
	case "windows":
		return newWindowsPlatform(info.System.Shell)
	default:
		return newPosixPlatform()
	}
}
