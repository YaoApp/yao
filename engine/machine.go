package engine

import (
	"crypto/sha256"
	"fmt"
	"net"
	"os"
	"runtime"
	"strings"
	"sync"

	"github.com/yaoapp/gou/process"
)

// MachineInfo contains deterministic machine identification.
type MachineInfo struct {
	ID       string `json:"id"`       // "yao-cli-{hash32}" deterministic client ID
	Hostname string `json:"hostname"` // OS hostname
	Platform string `json:"platform"` // runtime.GOOS: "darwin", "linux", "windows"
}

var (
	cachedMachineInfo *MachineInfo
	machineOnce       sync.Once
	machineErr        error
)

func init() {
	process.Register("utils.app.MachineID", processMachineID)
}

// GetMachineID returns a deterministic machine fingerprint.
// The result is cached after the first call.
func GetMachineID() (*MachineInfo, error) {
	machineOnce.Do(func() {
		cachedMachineInfo, machineErr = computeMachineID()
	})
	return cachedMachineInfo, machineErr
}

func computeMachineID() (*MachineInfo, error) {
	hostname, _ := os.Hostname()

	raw, err := platformMachineID()
	if err != nil || strings.TrimSpace(raw) == "" {
		raw = fallbackMachineID(hostname)
	}

	hash := sha256.Sum256([]byte(raw))
	id := fmt.Sprintf("yao-cli-%x", hash[:16]) // 32 hex chars

	return &MachineInfo{
		ID:       id,
		Hostname: hostname,
		Platform: runtime.GOOS,
	}, nil
}

func fallbackMachineID(hostname string) string {
	mac := firstHardwareAddr()
	return hostname + ":" + mac
}

func firstHardwareAddr() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "unknown"
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback != 0 || len(iface.HardwareAddr) == 0 {
			continue
		}
		return iface.HardwareAddr.String()
	}
	return "unknown"
}

func processMachineID(p *process.Process) interface{} {
	info, err := GetMachineID()
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	return map[string]interface{}{
		"id":       info.ID,
		"hostname": info.Hostname,
		"platform": info.Platform,
	}
}
