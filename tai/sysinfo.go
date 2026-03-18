package tai

import (
	"os"
	"os/exec"
	goruntime "runtime"

	"github.com/yaoapp/yao/tai/types"
)

// CollectSystemInfo gathers system information for the local host.
// The result is identical in structure to what a remote Tai node reports
// via the ServerInfo gRPC service, keeping local and remote nodes symmetric.
func CollectSystemInfo() types.SystemInfo {
	hostname, _ := os.Hostname()
	return types.SystemInfo{
		OS:       goruntime.GOOS,
		Arch:     goruntime.GOARCH,
		Hostname: hostname,
		NumCPU:   goruntime.NumCPU(),
		Shell:    detectShell(),
		TempDir:  os.TempDir(),
	}
}

func detectShell() string {
	if goruntime.GOOS != "windows" {
		return "sh"
	}
	if _, err := exec.LookPath("pwsh"); err == nil {
		return "pwsh"
	}
	if _, err := exec.LookPath("powershell"); err == nil {
		return "powershell"
	}
	return "cmd.exe"
}
