//go:build linux

package commercial

import (
	"os"
	"strings"
)

func platformMachineID() string {
	data, err := os.ReadFile("/etc/machine-id")
	if err != nil {
		data, err = os.ReadFile("/var/lib/dbus/machine-id")
		if err != nil {
			return ""
		}
	}
	id := strings.TrimSpace(strings.ToLower(string(data)))
	if id == "" {
		return ""
	}
	return id
}
