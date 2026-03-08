package jsapi

import (
	"rogchap.com/v8go"
)

// NewBoxObject creates a JS Box object backed by a sandbox ID string.
// All methods delegate to the Go sandbox.M() singleton — no Go object is
// passed to V8, no bridge registration, no Release() needed.
//
// Box implements the Computer interface, so it shares the unified Exec/Stream/
// VNC/Proxy/ComputerInfo/BindWorkplace/Workplace methods with Host. It also
// has Box-specific methods (Attach, Info, Start, Stop, Remove).
//
// # Properties (read-only)
//
//	box.id    → string   // sandbox ID       ← Box.ID()
//	box.owner → string   // owner user ID    ← Box.Owner()
//	box.pool  → string   // pool name        ← Box.Pool()
//
// # Methods — Computer interface (unified with Host)
//
// box.Exec(cmd, options?) → ExecResult
//
//	Go: Computer.Exec(ctx, cmd []string, opts ...ExecOption) (*ExecResult, error)
//
//	JS args:
//	  cmd:     string[]                         → cmd []string
//	  options: {                                → ExecOption
//	    workdir:    string,                     → WithWorkDir(dir)
//	    env:        object,                     → WithEnv(map[string]string)
//	    stdin:      string,                     → WithStdin([]byte)
//	    timeout:    number,                     → WithTimeout(ms → time.Duration)
//	    max_output: number                      → WithMaxOutput(bytes int64)
//	  }
//	JS returns: {
//	  exit_code:   number,                      ← ExecResult.ExitCode
//	  stdout:      string,                      ← ExecResult.Stdout
//	  stderr:      string,                      ← ExecResult.Stderr
//	  duration_ms: number,                      ← ExecResult.DurationMs (Host fills; Box = 0)
//	  error:       string,                      ← ExecResult.Error (Host fills; Box = "")
//	  truncated:   boolean                      ← ExecResult.Truncated (Host fills; Box = false)
//	}
//
// box.Stream(cmd, callback) / box.Stream(cmd, options, callback)
//
//	Go: Computer.Stream(ctx, cmd []string, opts ...ExecOption) (*ExecStream, error)
//
//	Blocks until the process exits. The last argument must be a JS function.
//	Callback signature: function(type, data)
//	  type = "stdout" → data is string (chunk)
//	  type = "stderr" → data is string (chunk)
//	  type = "exit"   → data is number (exit code)
//
// box.VNC() → string
//
//	Go: Computer.VNC(ctx) (string, error)
//	Returns: VNC WebSocket URL
//
// box.Proxy(port, path?) → string
//
//	Go: Computer.Proxy(ctx, port int, path string) (string, error)
//	Returns: HTTP proxy URL
//
// box.ComputerInfo() → ComputerInfo
//
//	Go: Computer.ComputerInfo() ComputerInfo
//	JS returns: {
//	  kind: "box", pool, status,
//	  box_id, container_id, owner, image, policy, labels, ...
//	}
//
// box.BindWorkplace(workspaceID) → void
//
//	Go: Computer.BindWorkplace(workspaceID string)
//
// box.Workplace() → WorkspaceFS | null
//
//	Go: Computer.Workplace() workspace.FS
//
// # Methods — Box-specific
//
// box.Attach(port, options?) → string
//
//	Go: Proxy.URL(ctx, containerID, port, path) (string, error)
//
//	Returns the service URL string.
//	JS args:
//	  port:    number                           → port int
//	  options: {                                → AttachOption
//	    protocol: "ws"|"sse",                   → affects URL scheme
//	    path:     string,                       → URL path suffix
//	  }
//	JS returns: string (URL)
//
// box.Workspace() → WorkspaceFS
//
//	Implemented in workspace/jsapi package. Calls:
//	  workspace.NewFSObject(v8ctx, box.WorkspaceID())
//
// box.Info() → BoxInfo
//
//	Go: Box.Info(ctx) (*BoxInfo, error)
//	JS returns: {
//	  id, container_id, pool, owner, status, image, vnc, policy,
//	  labels, created_at, last_active, process_count
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
	// 3. Bind Computer interface methods:
	//    - Exec, Stream, VNC, Proxy, ComputerInfo, BindWorkplace, Workplace
	// 4. Bind Box-specific methods:
	//    - Attach  → client.Proxy().URL(ctx, containerID, port, path) → string
	//    - Workspace → NewFSObject(v8ctx, sandbox.M().Get(id).WorkspaceID())
	//    - Info    → sandbox.M().Get(id).Info(ctx) → JS object
	//    - Start   → sandbox.M().Get(id).Start(ctx)
	//    - Stop    → sandbox.M().Get(id).Stop(ctx)
	//    - Remove  → sandbox.M().Get(id).Remove(ctx)
	return nil, nil
}
