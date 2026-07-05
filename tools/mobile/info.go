package mobile

import (
	_ "embed"

	"github.com/yaoapp/gou/process"
)

//go:embed info_schema.json
var InfoSchemaJSON []byte

// InfoHandler is the tools.mobile_info process handler.
// Returns detailed information about a connected Android device.
//
// Args[0]: device_id (string)
func InfoHandler(proc *process.Process) interface{} {
	deviceID, err := extractDeviceID(proc, 0)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}

	meta, err := getAndroidNode(deviceID)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}

	caps := map[string]bool{}
	if meta.Capabilities.HostExec {
		caps["host_exec"] = true
	}
	if meta.Capabilities.Docker {
		caps["docker"] = true
	}
	if meta.Capabilities.K8s {
		caps["k8s"] = true
	}
	if meta.Capabilities.VNC {
		caps["vnc"] = true
	}

	return map[string]any{
		"device_id":    meta.TaiID,
		"machine_id":   meta.MachineID,
		"display_name": meta.DisplayName,
		"hostname":     meta.System.Hostname,
		"os":           meta.System.OS,
		"arch":         meta.System.Arch,
		"status":       meta.Status,
		"capabilities": caps,
	}
}
