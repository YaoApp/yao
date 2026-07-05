package mobile

import (
	_ "embed"

	"github.com/yaoapp/gou/process"
)

//go:embed list_schema.json
var ListSchemaJSON []byte

// ListHandler is the tools.mobile_list process handler.
// Returns all online Android devices.
func ListHandler(proc *process.Process) interface{} {
	nodes := listAndroidNodes()
	devices := make([]map[string]any, 0, len(nodes))
	for _, n := range nodes {
		devices = append(devices, map[string]any{
			"device_id":    n.TaiID,
			"display_name": n.DisplayName,
			"model":        n.System.Hostname,
			"os":           n.System.OS,
			"arch":         n.System.Arch,
			"status":       n.Status,
		})
	}
	return map[string]any{"devices": devices}
}
