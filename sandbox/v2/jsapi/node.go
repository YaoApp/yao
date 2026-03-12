package jsapi

import (
	"encoding/json"
	"time"

	"github.com/yaoapp/yao/tai/registry"
	taitypes "github.com/yaoapp/yao/tai/types"
	"rogchap.com/v8go"
)

// sbGetNode: `sandbox.GetNode(taiID)` → NodeInfo | null
func sbGetNode(info *v8go.FunctionCallbackInfo) *v8go.Value {
	iso := info.Context().Isolate()
	args := info.Args()
	if len(args) < 1 || !args[0].IsString() {
		return throwError(info, "GetNode requires taiID (string)")
	}

	reg := registry.Global()
	if reg == nil {
		return throwError(info, "registry not initialized")
	}

	snap, ok := reg.Get(args[0].String())
	if !ok {
		return v8go.Null(iso)
	}

	val, err := snapshotToJS(info.Context(), snap)
	if err != nil {
		return throwError(info, err.Error())
	}
	return val
}

// sbNodes: `sandbox.Nodes()` → NodeInfo[]
func sbNodes(info *v8go.FunctionCallbackInfo) *v8go.Value {
	v8ctx := info.Context()

	reg := registry.Global()
	if reg == nil {
		return throwError(info, "registry not initialized")
	}

	snaps := reg.List()
	return snapshotsToJSArray(v8ctx, snaps)
}

// sbNodesByTeam: `sandbox.NodesByTeam(teamID)` → NodeInfo[]
func sbNodesByTeam(info *v8go.FunctionCallbackInfo) *v8go.Value {
	v8ctx := info.Context()
	args := info.Args()
	if len(args) < 1 || !args[0].IsString() {
		return throwError(info, "NodesByTeam requires teamID (string)")
	}

	reg := registry.Global()
	if reg == nil {
		return throwError(info, "registry not initialized")
	}

	snaps := reg.ListByTeam(args[0].String())
	return snapshotsToJSArray(v8ctx, snaps)
}

// snapshotToJS converts a NodeMeta to a JS NodeInfo object.
// Auth and YaoBase are excluded for security.
func snapshotToJS(v8ctx *v8go.Context, snap *taitypes.NodeMeta) (*v8go.Value, error) {
	ports := map[string]interface{}{
		"grpc": snap.Ports.GRPC, "http": snap.Ports.HTTP,
		"vnc": snap.Ports.VNC, "docker": snap.Ports.Docker, "k8s": snap.Ports.K8s,
	}

	caps := map[string]interface{}{
		"docker": snap.Capabilities.Docker, "k8s": snap.Capabilities.K8s,
		"host_exec": snap.Capabilities.HostExec,
	}

	data, err := json.Marshal(map[string]interface{}{
		"tai_id":       snap.TaiID,
		"machine_id":   snap.MachineID,
		"version":      snap.Version,
		"mode":         snap.Mode,
		"addr":         snap.Addr,
		"status":       snap.Status,
		"display_name": snap.DisplayName,
		"connected_at": snap.ConnectedAt.Format(time.RFC3339),
		"last_ping":    snap.LastPing.Format(time.RFC3339),
		"ports":        ports,
		"capabilities": caps,
		"system": map[string]interface{}{
			"os":        snap.System.OS,
			"arch":      snap.System.Arch,
			"hostname":  snap.System.Hostname,
			"num_cpu":   snap.System.NumCPU,
			"total_mem": snap.System.TotalMem,
		},
	})
	if err != nil {
		return nil, err
	}

	return v8go.JSONParse(v8ctx, string(data))
}

func snapshotsToJSArray(v8ctx *v8go.Context, snaps []taitypes.NodeMeta) *v8go.Value {
	items := make([]interface{}, 0, len(snaps))
	for i := range snaps {
		snap := &snaps[i]
		ports := map[string]interface{}{
			"grpc": snap.Ports.GRPC, "http": snap.Ports.HTTP,
			"vnc": snap.Ports.VNC, "docker": snap.Ports.Docker, "k8s": snap.Ports.K8s,
		}
		caps := map[string]interface{}{
			"docker": snap.Capabilities.Docker, "k8s": snap.Capabilities.K8s,
			"host_exec": snap.Capabilities.HostExec,
		}
		items = append(items, map[string]interface{}{
			"tai_id":       snap.TaiID,
			"node_id":      snap.TaiID,
			"machine_id":   snap.MachineID,
			"version":      snap.Version,
			"mode":         snap.Mode,
			"addr":         snap.Addr,
			"status":       snap.Status,
			"display_name": snap.DisplayName,
			"connected_at": snap.ConnectedAt.Format(time.RFC3339),
			"last_ping":    snap.LastPing.Format(time.RFC3339),
			"ports":        ports,
			"capabilities": caps,
			"system": map[string]interface{}{
				"os":        snap.System.OS,
				"arch":      snap.System.Arch,
				"hostname":  snap.System.Hostname,
				"num_cpu":   snap.System.NumCPU,
				"total_mem": snap.System.TotalMem,
			},
		})
	}
	data, _ := json.Marshal(items)
	val, _ := v8go.JSONParse(v8ctx, string(data))
	return val
}
