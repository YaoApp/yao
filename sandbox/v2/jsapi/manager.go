package jsapi

import (
	"rogchap.com/v8go"
)

// NewManagerObject creates a JS SandboxManager object with the following methods:
//
//	manager.Create(options?)    → Box        // create a new sandbox box
//	manager.GetOrCreate(opts)   → Box        // get existing or create
//	manager.Get(id)             → Box|null   // get by sandbox ID
//	manager.List(options?)      → Box[]      // list boxes
//	manager.Remove(id)          → void       // remove a box
//	manager.EnsureImage(ref)    → void       // pull image if missing
//	manager.ImageExists(ref)    → boolean    // check image presence
//	manager.Pools()             → PoolInfo[] // list pool info
//	manager.Release()           → void       // release JS bridge ref
//
// Create options (merged with constructor defaults):
//
//	{
//	  id:           string   // explicit sandbox ID (optional)
//	  workdir:      string   // container working directory
//	  user:         string   // container user (e.g. "1000:1000")
//	  env:          object   // environment variables
//	  memory:       number   // memory limit in bytes
//	  cpus:         number   // CPU limit (e.g. 1.5)
//	  vnc:          boolean  // enable VNC
//	  ports:        array    // port mappings [{container: 8080, host: 0}]
//	  policy:       string   // "oneshot"|"session"|"longrunning"|"persistent"
//	  idle_timeout: number   // idle timeout in ms
//	  stop_timeout: number   // stop timeout in ms
//	  workspace_id: string   // workspace to mount
//	  mount_mode:   string   // "rw"|"ro"
//	  mount_path:   string   // mount target in container
//	}
func NewManagerObject(v8ctx *v8go.Context /* manager *sandbox.Manager, defaults CreateDefaults */) (*v8go.Value, error) {
	// TODO: Phase 2 implementation
	// 1. Create ObjectTemplate with InternalFieldCount(1)
	// 2. Register manager in bridge
	// 3. Bind methods: Create, GetOrCreate, Get, List, Remove,
	//    EnsureImage, ImageExists, Pools, Release
	// 4. Create instance, set internal field
	return nil, nil
}
