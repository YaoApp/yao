package jsapi

import (
	"rogchap.com/v8go"
)

// sbHost: `sandbox.Host(pool?)` → Computer (kind="host")
//
// Go: Manager.Host(ctx, pool) (*Host, error)
//
// Args:
//
//	pool: string (optional) — pool name; empty = default pool
//
// Returns: Computer object (kind="host") if the pool has host_exec capability, otherwise throws.
func sbHost(info *v8go.FunctionCallbackInfo) *v8go.Value {
	// TODO: Phase 2
	// 1. pool := ""; if len(info.Args()) > 0 && info.Args()[0].IsString() { pool = info.Args()[0].String() }
	// 2. host, err := sandbox.M().Host(ctx, pool)
	// 3. if err != nil { throw in V8 }
	// 4. Return NewComputerObject(v8ctx, "host", pool)
	return v8go.Undefined(info.Context().Isolate())
}

// NewComputerObject creates a unified JS Computer object backed by either a Box or Host.
// The `kind` field ("box" or "host") determines which methods are available at runtime.
// Box-only methods (Attach, Info, Start, Stop, Remove) throw an error when called on a host.
//
// # Properties (read-only)
//
//	pc.kind  → string  // "box" | "host"  ← ComputerInfo().Kind
//	pc.id    → string  // sandbox ID      ← Box.ID() (empty for host)
//	pc.owner → string  // owner           ← Box.Owner() (empty for host)
//	pc.pool  → string  // pool name       ← ComputerInfo().Pool
//
// # Methods — Computer interface (both box and host)
//
// pc.Exec(cmd, options?) → ExecResult
//
//	Go: Computer.Exec(ctx, cmd []string, opts ...ExecOption) (*ExecResult, error)
//
//	JS args:
//	  cmd:     string[]                  → cmd []string
//	  options: {                         → ExecOption
//	    workdir:    string,              → WithWorkDir(dir)
//	    env:        object,              → WithEnv(map[string]string)
//	    stdin:      string,              → WithStdin([]byte)
//	    timeout:    number,              → WithTimeout(ms → time.Duration)
//	    max_output: number               → WithMaxOutput(bytes int64)
//	  }
//	JS returns: {
//	  exit_code:   number,               ← ExecResult.ExitCode
//	  stdout:      string,               ← ExecResult.Stdout
//	  stderr:      string,               ← ExecResult.Stderr
//	  duration_ms: number,               ← ExecResult.DurationMs
//	  error:       string,               ← ExecResult.Error
//	  truncated:   boolean               ← ExecResult.Truncated
//	}
//
// pc.Stream(cmd, callback) / pc.Stream(cmd, options, callback)
//
//	Go: Computer.Stream(ctx, cmd []string, opts ...ExecOption) (*ExecStream, error)
//
//	Blocks until the process exits. The last argument must be a JS function.
//	Callback signature: function(type, data)
//	  type = "stdout" → data is string (chunk)
//	  type = "stderr" → data is string (chunk)
//	  type = "exit"   → data is number (exit code)
//
// pc.VNC() → string
//
//	Go: Computer.VNC(ctx) (string, error)
//	Box:  returns ws://host:port/vnc/{containerID}/ws
//	Host: returns ws://host:port/vnc/__host__/ws
//
// pc.Proxy(port, path?) → string
//
//	Go: Computer.Proxy(ctx, port int, path string) (string, error)
//	Box:  returns http://host:port/{containerID}:{port}/{path}
//	Host: returns http://host:port/__host__:{port}/{path}
//
// pc.ComputerInfo() → ComputerInfo
//
//	Go: Computer.ComputerInfo() ComputerInfo
//	JS returns: { kind, pool, tai_id, machine_id, version, mode, status, capabilities,
//	              system: { os, arch, hostname, num_cpu, total_mem },
//	              box_id, container_id, owner, image, policy, labels }
//
// pc.BindWorkplace(workspaceID) → void
//
//	Go: Computer.BindWorkplace(workspaceID string)
//
// pc.Workplace() → WorkspaceFS | null
//
//	Go: Computer.Workplace() workspace.FS
//	Returns WorkspaceFS if a workplace is bound, null otherwise.
//
// # Methods — Box-only (throw on host)
//
// pc.Attach(port, options?) → string
//
//	Gets a WebSocket/SSE endpoint URL for a container service.
//	JS args:
//	  port:    number
//	  options: { protocol: "ws"|"sse", path: string }
//	JS returns: string (URL)
//
// pc.Info() → BoxInfo
//
//	Go: Box.Info(ctx) (*BoxInfo, error)
//	JS returns: { id, container_id, pool, owner, status, image, vnc, policy,
//	              labels, created_at, last_active, process_count }
//
// pc.Start() → void
//
//	Go: Box.Start(ctx) error
//
// pc.Stop() → void
//
//	Go: Box.Stop(ctx) error
//
// pc.Remove() → void
//
//	Go: Box.Remove(ctx) error
func NewComputerObject(v8ctx *v8go.Context, kind string, id string) (*v8go.Value, error) {
	// TODO: Phase 2 implementation
	// 1. Create JS object via v8go.NewObjectTemplate
	// 2. Set read-only properties: kind, id, owner, pool
	//    - kind: "box" or "host"
	//    - id/owner: from sandbox.M().Get(id) for box; empty for host
	//    - pool: from ComputerInfo().Pool
	// 3. Bind Computer interface methods:
	//    - Exec, Stream, VNC, Proxy, ComputerInfo, BindWorkplace, Workplace
	// 4. Bind box-only methods with kind guard:
	//    - Attach, Info, Start, Stop, Remove
	//    - If kind == "host", these throw: "not supported: {method}() requires a box computer"
	return nil, nil
}
