package jsapi

import (
	"rogchap.com/v8go"
)

// NewBoxObject creates a JS Box object backed by a sandbox ID string.
// All methods delegate to the Go sandbox.M() singleton — no Go object is
// passed to V8, no bridge registration, no Release() needed.
//
// # Properties (read-only)
//
//	box.id    → string   // sandbox ID       ← Box.ID()
//	box.owner → string   // owner user ID    ← Box.Owner()
//	box.pool  → string   // pool name        ← Box.Pool()
//
// # Methods — Go mapping
//
// box.Exec(cmd, options?) → ExecResult
//
//	Go: Box.Exec(ctx, cmd []string, opts ...ExecOption) (*ExecResult, error)
//
//	JS args:
//	  cmd:     string[]                         → cmd []string
//	  options: {                                → ExecOption functional options
//	    workdir: string,                        → WithWorkDir(dir)
//	    env:     object,                        → WithEnv(map[string]string)
//	    timeout: number                         → WithTimeout(ms → time.Duration)
//	  }
//	JS returns: {
//	  exit_code: number,                        ← ExecResult.ExitCode
//	  stdout:    string,                        ← ExecResult.Stdout
//	  stderr:    string                         ← ExecResult.Stderr
//	}
//
// box.Stream(cmd, callback) / box.Stream(cmd, options, callback)
//
//	Go: Box.Stream(ctx, cmd []string, opts ...ExecOption) (*ExecStream, error)
//
//	Blocks until the process exits. The last argument must be a JS function.
//	Callback signature: function(type, data)
//	  type = "stdout" → data is string (chunk)
//	  type = "stderr" → data is string (chunk)
//	  type = "exit"   → data is number (exit code)
//
//	JS args:
//	  cmd:      string[]
//	  options:  { workdir, env, timeout }   (optional, same as Exec)
//	  callback: function(type, data)
//
// box.Attach(port, options?) → string
//
//	Go: Proxy.URL(ctx, containerID, port, path) (string, error)
//
//	Returns the service URL string. Caller (frontend/Agent) establishes WS/SSE.
//	JS args:
//	  port:    number                           → port int
//	  options: {                                → AttachOption
//	    protocol: "ws"|"sse",                   → affects URL scheme
//	    path:     string,                       → URL path suffix
//	  }
//	JS returns: string (URL)
//
// box.VNC() → string
//
//	Go: Box.VNC(ctx) (string, error)
//	Returns: VNC WebSocket URL
//
// box.Proxy(port, path?) → string
//
//	Go: Box.Proxy(ctx, port int, path string) (string, error)
//	Returns: HTTP proxy URL
//
// box.Workspace() → WorkspaceFS
//
//	Implemented in workspace/jsapi package. This method calls:
//	  workspace.NewFSObject(v8ctx, box.WorkspaceID())
//	and returns the resulting WorkspaceFS object directly.
//
// box.Info() → BoxInfo
//
//	Go: Box.Info(ctx) (*BoxInfo, error)
//	JS returns: {
//	  id:            string,                    ← BoxInfo.ID
//	  container_id:  string,                    ← BoxInfo.ContainerID
//	  pool:          string,                    ← BoxInfo.Pool
//	  owner:         string,                    ← BoxInfo.Owner
//	  status:        string,                    ← BoxInfo.Status
//	  image:         string,                    ← BoxInfo.Image
//	  vnc:           boolean,                   ← BoxInfo.VNC
//	  policy:        string,                    ← BoxInfo.Policy (LifecyclePolicy)
//	  labels:        object,                    ← BoxInfo.Labels (map[string]string)
//	  created_at:    string,                    ← BoxInfo.CreatedAt (ISO 8601)
//	  last_active:   string,                    ← BoxInfo.LastActive (ISO 8601)
//	  process_count: number                     ← BoxInfo.ProcessCount
//	}
//
// box.Start() → void
//
//	Go: Box.Start(ctx) error
//
// box.Stop() → void
//
//	Go: Box.Stop(ctx) error
//
// box.Remove() → void
//
//	Go: Box.Remove(ctx) error
func NewBoxObject(v8ctx *v8go.Context, boxID string) (*v8go.Value, error) {
	// TODO: Phase 2 implementation
	// 1. Create JS object via v8go.NewObjectTemplate
	// 2. Set read-only properties: id, owner, pool (from sandbox.M().Get(boxID))
	// 3. Bind each method as FunctionTemplate:
	//    - Exec    → sandbox.M().Get(id).Exec(ctx, cmd, opts...)
	//    - Stream  → sandbox.M().Get(id).Stream(ctx, cmd, opts...)
	//    - Attach  → client.Proxy().URL(ctx, containerID, port, path) → string
	//    - VNC     → sandbox.M().Get(id).VNC(ctx)
	//    - Proxy   → sandbox.M().Get(id).Proxy(ctx, port, path)
	//    - Workspace → NewFSObject(v8ctx, sandbox.M().Get(id).WorkspaceID())
	//    - Info    → sandbox.M().Get(id).Info(ctx) → JS object
	//    - Start   → sandbox.M().Get(id).Start(ctx)
	//    - Stop    → sandbox.M().Get(id).Stop(ctx)
	//    - Remove  → sandbox.M().Get(id).Remove(ctx)
	return nil, nil
}
