// Package jsapi registers the sandbox namespace into the Yao V8 runtime.
//
// All methods are static on the sandbox object — no constructor.
// Both sandbox.Create() and sandbox.Host() return a unified Computer object.
//
// # JavaScript API
//
//	const pc   = sandbox.Create({ image: "node:20", owner: "user1" }) // → Computer (kind="box")
//	const pc   = sandbox.Get(id)               // → Computer (kind="box") | null
//	const list = sandbox.List({ owner: "u1" }) // → BoxInfo[]
//	sandbox.Delete(id)                          // → void
//	const host = sandbox.Host("gpu")            // → Computer (kind="host")
//	const node = sandbox.GetNode("tai-abc123") // → NodeInfo | null
//	const all  = sandbox.Nodes()               // → NodeInfo[]
//	const team = sandbox.NodesByTeam("t-001")  // → NodeInfo[]
//
// # Go mapping
//
//	sandbox.Create(opts)  → Manager.Create(ctx, CreateOptions)    → Computer (Box)
//	sandbox.Create(opts)  → Manager.GetOrCreate(ctx, opts)        → Computer (Box)  (when opts.id is set)
//	sandbox.Get(id)       → Manager.Get(ctx, id)                  → Computer (Box)
//	sandbox.List(filter?) → Manager.List(ctx, ListOptions)        → BoxInfo[]
//	sandbox.Delete(id)    → Manager.Remove(ctx, id)               → void
//	sandbox.Host(pool?)   → Manager.Host(ctx, pool)               → Computer (Host)
//	sandbox.GetNode(id)   → registry.Global().Get(id)             → NodeInfo | null
//	sandbox.Nodes()       → registry.Global().List()              → NodeInfo[]
//	sandbox.NodesByTeam(t)→ registry.Global().ListByTeam(t)       → NodeInfo[]
//
// Registration happens via init() — import with:
//
//	_ "github.com/yaoapp/yao/sandbox/v2/jsapi"
package jsapi

import (
	v8 "github.com/yaoapp/gou/runtime/v8"
	"rogchap.com/v8go"
)

func init() {
	v8.RegisterObject("sandbox", ExportObject)
}

// ExportObject exports the sandbox namespace object to V8.
func ExportObject(iso *v8go.Isolate) *v8go.ObjectTemplate {
	obj := v8go.NewObjectTemplate(iso)
	obj.Set("Create", v8go.NewFunctionTemplate(iso, sbCreate))
	obj.Set("Get", v8go.NewFunctionTemplate(iso, sbGet))
	obj.Set("List", v8go.NewFunctionTemplate(iso, sbList))
	obj.Set("Delete", v8go.NewFunctionTemplate(iso, sbDelete))
	obj.Set("Host", v8go.NewFunctionTemplate(iso, sbHost))
	obj.Set("GetNode", v8go.NewFunctionTemplate(iso, sbGetNode))
	obj.Set("Nodes", v8go.NewFunctionTemplate(iso, sbNodes))
	obj.Set("NodesByTeam", v8go.NewFunctionTemplate(iso, sbNodesByTeam))
	return obj
}

// sbCreate: `sandbox.Create(options)` → Box
//
// Go: Manager.Create(ctx, CreateOptions) (*Box, error)
//
//	Manager.GetOrCreate(ctx, CreateOptions) (*Box, error)  — when opts.id is set
//
// JS options → Go CreateOptions mapping:
//
//	{
//	  id:           string   →  CreateOptions.ID           // optional; triggers GetOrCreate
//	  owner:        string   →  CreateOptions.Owner        // required
//	  pool:         string   →  CreateOptions.Pool         // default: first pool
//	  image:        string   →  CreateOptions.Image        // required
//	  workdir:      string   →  CreateOptions.WorkDir
//	  user:         string   →  CreateOptions.User         // e.g. "1000:1000"
//	  env:          object   →  CreateOptions.Env          // map[string]string
//	  memory:       number   →  CreateOptions.Memory       // bytes (int64)
//	  cpus:         number   →  CreateOptions.CPUs         // float64 e.g. 1.5
//	  vnc:          boolean  →  CreateOptions.VNC
//	  ports:        array    →  CreateOptions.Ports        // [{container_port, host_port, host_ip, protocol}] → []PortMapping
//	  policy:       string   →  CreateOptions.Policy       // "oneshot"|"session"|"longrunning"|"persistent"
//	  idle_timeout: number   →  CreateOptions.IdleTimeout  // ms → time.Duration
//	  stop_timeout: number   →  CreateOptions.StopTimeout  // ms → time.Duration
//	  workspace_id: string   →  CreateOptions.WorkspaceID
//	  mount_mode:   string   →  CreateOptions.MountMode    // "rw"|"ro"
//	  mount_path:   string   →  CreateOptions.MountPath
//	  labels:       object   →  CreateOptions.Labels       // map[string]string
//	}
//
// Returns: Computer object (kind="box") — see computer.go
func sbCreate(info *v8go.FunctionCallbackInfo) *v8go.Value {
	// TODO: Phase 2
	// 1. Parse options from info.Args()[0]
	// 2. Validate required fields (image, owner)
	// 3. If opts.id != "" → sandbox.M().GetOrCreate(ctx, opts)
	//    else             → sandbox.M().Create(ctx, opts)
	// 4. Return NewComputerObject(v8ctx, "box", box.ID())
	return v8go.Undefined(info.Context().Isolate())
}

// sbGet: `sandbox.Get(id)` → Box | null
//
// Go: Manager.Get(ctx, id) (*Box, error)
//
// Args:
//
//	id: string  — sandbox ID
//
// Returns: Computer object (kind="box") if found, null if not found
func sbGet(info *v8go.FunctionCallbackInfo) *v8go.Value {
	// TODO: Phase 2
	// 1. id = info.Args()[0].String()
	// 2. box, err := sandbox.M().Get(ctx, id)
	// 3. Return NewComputerObject(v8ctx, "box", id) or null
	return v8go.Undefined(info.Context().Isolate())
}

// sbList: `sandbox.List(filter?)` → BoxInfo[]
//
// Go: Manager.List(ctx, ListOptions) ([]*Box, error)
//
//	then Box.Info(ctx) for each → BoxInfo
//
// JS filter → Go ListOptions mapping:
//
//	{
//	  owner:  string  →  ListOptions.Owner   // filter by owner; empty = all
//	  pool:   string  →  ListOptions.Pool    // filter by pool; empty = all
//	  labels: object  →  ListOptions.Labels  // filter by labels
//	}
//
// Returns: BoxInfo[] — each element:
//
//	{
//	  id:            string   ←  BoxInfo.ID
//	  container_id:  string   ←  BoxInfo.ContainerID
//	  pool:          string   ←  BoxInfo.Pool
//	  owner:         string   ←  BoxInfo.Owner
//	  status:        string   ←  BoxInfo.Status
//	  image:         string   ←  BoxInfo.Image
//	  vnc:           boolean  ←  BoxInfo.VNC
//	  policy:        string   ←  BoxInfo.Policy
//	  labels:        object   ←  BoxInfo.Labels
//	  created_at:    string   ←  BoxInfo.CreatedAt   (ISO 8601)
//	  last_active:   string   ←  BoxInfo.LastActive   (ISO 8601)
//	  process_count: number   ←  BoxInfo.ProcessCount
//	}
func sbList(info *v8go.FunctionCallbackInfo) *v8go.Value {
	// TODO: Phase 2
	// 1. Parse optional filter from info.Args()[0]
	// 2. boxes := sandbox.M().List(ctx, opts)
	// 3. For each box: box.Info(ctx) → BoxInfo → JS object
	// 4. Return JS array of BoxInfo objects
	return v8go.Undefined(info.Context().Isolate())
}

// sbDelete: `sandbox.Delete(id)` → void
//
// Go: Manager.Remove(ctx, id) error
//
// Args:
//
//	id: string  — sandbox ID to remove
func sbDelete(info *v8go.FunctionCallbackInfo) *v8go.Value {
	// TODO: Phase 2
	// 1. id = info.Args()[0].String()
	// 2. sandbox.M().Remove(ctx, id)
	return v8go.Undefined(info.Context().Isolate())
}
