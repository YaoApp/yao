//go:build darwin

package commercial

import (
	"os/exec"
	"strings"
)

func platformMachineID() string {
	out, err := exec.Command("ioreg", "-rd1", "-c", "IOPlatformExpertDevice").Output()
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(out), "\n") {
		if strings.Contains(line, "IOPlatformUUID") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				uuid := strings.Trim(strings.TrimSpace(parts[1]), "\"")
				if uuid != "" {
					return strings.TrimSpace(strings.ToLower(uuid))
				}
			}
		}
	}
	return ""
}
