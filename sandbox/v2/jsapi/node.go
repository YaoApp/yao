package jsapi

import (
	"rogchap.com/v8go"
)

// sbGetNode: `sandbox.GetNode(taiID)` → NodeInfo | null
//
// Go: registry.Global().Get(taiID) (*NodeSnapshot, bool)
//
// Args:
//
//	taiID: string  — Tai node ID
//
// Returns: NodeInfo object if found, null if not found.
// Auth and YaoBase fields are excluded for security.
func sbGetNode(info *v8go.FunctionCallbackInfo) *v8go.Value {
	// TODO: Phase 2
	// 1. taiID = info.Args()[0].String()
	// 2. snap, ok := registry.Global().Get(taiID)
	// 3. if !ok { return v8go.Null }
	// 4. Return snapshotToJS(v8ctx, snap)
	return v8go.Undefined(info.Context().Isolate())
}

// sbNodes: `sandbox.Nodes()` → NodeInfo[]
//
// Go: registry.Global().List() []NodeSnapshot
//
// Returns: array of NodeInfo objects for all registered Tai nodes.
func sbNodes(info *v8go.FunctionCallbackInfo) *v8go.Value {
	// TODO: Phase 2
	// 1. snaps := registry.Global().List()
	// 2. Build JS array, for each: snapshotToJS(v8ctx, snap)
	// 3. Return JS array
	return v8go.Undefined(info.Context().Isolate())
}

// sbNodesByTeam: `sandbox.NodesByTeam(teamID)` → NodeInfo[]
//
// Go: registry.Global().ListByTeam(teamID) []NodeSnapshot
//
// Args:
//
//	teamID: string  — team ID to filter by
//
// Returns: array of NodeInfo objects belonging to the given team.
func sbNodesByTeam(info *v8go.FunctionCallbackInfo) *v8go.Value {
	// TODO: Phase 2
	// 1. teamID = info.Args()[0].String()
	// 2. snaps := registry.Global().ListByTeam(teamID)
	// 3. Build JS array, for each: snapshotToJS(v8ctx, snap)
	// 4. Return JS array
	return v8go.Undefined(info.Context().Isolate())
}

// snapshotToJS converts a NodeSnapshot to a JS NodeInfo object.
//
// Excluded: Auth (sensitive), YaoBase (internal URL).
//
// NodeInfo JS object shape:
//
//	{
//	  tai_id:       string,           ← NodeSnapshot.TaiID
//	  machine_id:   string,           ← NodeSnapshot.MachineID
//	  version:      string,           ← NodeSnapshot.Version
//	  mode:         string,           ← NodeSnapshot.Mode  ("direct"|"tunnel")
//	  addr:         string,           ← NodeSnapshot.Addr
//	  status:       string,           ← NodeSnapshot.Status ("online"|"offline"|"connecting")
//	  pool:         string,           ← NodeSnapshot.PoolName
//	  connected_at: string,           ← NodeSnapshot.ConnectedAt (ISO 8601)
//	  last_ping:    string,           ← NodeSnapshot.LastPing    (ISO 8601)
//	  ports: {                        ← NodeSnapshot.Ports
//	    grpc:   number,
//	    http:   number,
//	    vnc:    number,
//	    docker: number,
//	    k8s:    number,
//	  },
//	  capabilities: {                 ← NodeSnapshot.Capabilities
//	    docker:    boolean,
//	    k8s:       boolean,
//	    host_exec: boolean,
//	  },
//	  system: {                       ← NodeSnapshot.System (SystemInfo)
//	    os:        string,
//	    arch:      string,
//	    hostname:  string,
//	    num_cpu:   number,
//	    total_mem: number,
//	  }
//	}
//
//nolint:unused // placeholder for Phase 2
func snapshotToJS(v8ctx *v8go.Context, snap interface{}) (*v8go.Value, error) {
	// TODO: Phase 2 implementation
	// 1. Create JS object via v8go.NewObjectTemplate
	// 2. Set scalar fields: tai_id, machine_id, version, mode, addr, status, pool
	// 3. Set time fields: connected_at, last_ping → snap.ConnectedAt.Format(time.RFC3339)
	// 4. Build ports sub-object from snap.Ports map
	// 5. Build capabilities sub-object from snap.Capabilities map
	// 6. Build system sub-object from snap.System (OS, Arch, Hostname, NumCPU, TotalMem)
	// 7. Return the JS object
	return nil, nil
}
