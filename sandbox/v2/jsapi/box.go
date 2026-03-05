package jsapi

import (
	"rogchap.com/v8go"
)

// NewBoxObject creates a JS Box object with the following methods:
//
//	box.ID()                        → string      // sandbox ID
//	box.Owner()                     → string      // owner
//	box.ContainerID()               → string      // underlying container/pod ID
//	box.Pool()                      → string      // pool name
//	box.WorkspaceID()               → string      // mounted workspace ID (empty if none)
//
//	box.Exec(cmd, options?)         → ExecResult  // run command, wait for completion
//	  cmd:     string[]                            // command + args
//	  options: { workdir, env, timeout }
//	  returns: { exit_code: number, stdout: string, stderr: string }
//
//	box.Stream(cmd, options?)       → ExecStream  // streaming I/O
//	  returns: { stdout: ReadableStream, stderr: ReadableStream,
//	             stdin: WritableStream, wait: ()=>number, cancel: ()=>void }
//
//	box.Attach(port, options?)      → ServiceConn // WebSocket/SSE attach
//	  port:    number                              // container port
//	  options: { protocol, path, headers }
//	  returns: { url: string, close: ()=>void }
//
//	box.VNC()                       → string      // VNC WebSocket URL
//	box.Proxy(port, path?)          → string      // HTTP proxy URL
//
//	box.Workspace()                 → WorkspaceFS // workspace file system
//	  returns WorkspaceFS object (see workspace/jsapi)
//
//	box.Info()                      → BoxInfo     // container status
//	  returns: { id, container_id, pool, owner, status, policy,
//	             labels, image, created_at, last_active, process_count, vnc }
//
//	box.Start()                     → void        // start stopped box
//	box.Stop()                      → void        // stop running box
//	box.Remove()                    → void        // remove box permanently
//	box.Release()                   → void        // release JS bridge ref
func NewBoxObject(v8ctx *v8go.Context /* , box *sandbox.Box */) (*v8go.Value, error) {
	// TODO: Phase 2 implementation
	// 1. Create ObjectTemplate with InternalFieldCount(1)
	// 2. Register box in bridge
	// 3. Bind property accessors: ID, Owner, ContainerID, Pool, WorkspaceID
	// 4. Bind methods: Exec, Stream, Attach, VNC, Proxy, Workspace,
	//    Info, Start, Stop, Remove, Release
	// 5. Create instance, set internal field
	return nil, nil
}
