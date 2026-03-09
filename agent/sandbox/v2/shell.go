package sandboxv2

import (
	"strings"

	infra "github.com/yaoapp/yao/sandbox/v2"
)

// shellKind identifies which shell to use for command execution.
type shellKind int

const (
	shellSh   shellKind = iota // Unix: sh -c
	shellPwsh                  // Windows: pwsh -NoProfile -Command
	shellPS                    // Windows: powershell -NoProfile -Command
	shellCmd                   // Windows: cmd.exe /C (last-resort fallback)
)

// shellWrap returns the Exec command slice to run a script string.
func shellWrap(kind shellKind, script string) []string {
	switch kind {
	case shellPwsh:
		return []string{"pwsh", "-NoProfile", "-Command", script}
	case shellPS:
		return []string{"powershell", "-NoProfile", "-Command", script}
	case shellCmd:
		return []string{"cmd.exe", "/C", script}
	default:
		return []string{"sh", "-c", script}
	}
}

// shellFromSystem resolves shellKind from ComputerInfo().System.Shell
// reported by the Tai node at registration time.
func shellFromSystem(computer infra.Computer) shellKind {
	shell := strings.ToLower(computer.ComputerInfo().System.Shell)
	switch shell {
	case "pwsh":
		return shellPwsh
	case "powershell":
		return shellPS
	case "cmd.exe", "cmd":
		return shellCmd
	default:
		return shellSh
	}
}
