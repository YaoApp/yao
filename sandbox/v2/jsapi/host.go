package jsapi

import (
	"rogchap.com/v8go"
)

// sbHost: `sandbox.Host(pool?)` → Computer (Host)
//
// Go: Manager.Host(ctx, pool) (*Host, error)
//
// Args:
//
//	pool: string (optional) — pool name; empty = default pool
//
// Returns: Computer object (Host) if the pool has host_exec capability, otherwise throws.
//
// Host executes commands on the Tai host machine (no container). Available only
// when the pool's Tai server exposes HostExec gRPC.
func sbHost(info *v8go.FunctionCallbackInfo) *v8go.Value {
	// TODO: Phase 2
	// 1. pool := ""; if len(info.Args()) > 0 && info.Args()[0].IsString() { pool = info.Args()[0].String() }
	// 2. host, err := sandbox.M().Host(ctx, pool)
	// 3. if err != nil { throw in V8 }
	// 4. Return NewComputerObject(v8ctx, host)
	return v8go.Undefined(info.Context().Isolate())
}

// NewHostObject creates a JS Computer object backed by a Host.
// All methods delegate to the Go sandbox.M() singleton — no Go *Host passed to V8.
//
// Host implements the unified Computer interface, so the JS object exposes the
// same methods as a Box Computer object:
//
// # Properties (read-only)
//
//	host.pool  → string   // pool name
//
// # Methods — Go mapping (unified Computer interface)
//
// host.Exec(cmd, options?) → ExecResult
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
// host.Stream(cmd, callback) / host.Stream(cmd, options, callback)
//
//	Go: Computer.Stream(ctx, cmd []string, opts ...ExecOption) (*ExecStream, error)
//
//	Blocks until the process exits. The last argument must be a JS function.
//	Callback signature: function(type, data)
//	  type = "stdout" → data is string (chunk)
//	  type = "stderr" → data is string (chunk)
//	  type = "exit"   → data is number (exit code)
//
// host.VNC() → string
//
//	Go: Computer.VNC(ctx) (string, error)
//	Returns: VNC WebSocket URL (routes to Tai host via __host__ identifier)
//
// host.Proxy(port, path?) → string
//
//	Go: Computer.Proxy(ctx, port int, path string) (string, error)
//	Returns: HTTP proxy URL (routes to Tai host via __host__ identifier)
//
// host.ComputerInfo() → ComputerInfo
//
//	Go: Computer.ComputerInfo() ComputerInfo
//	JS returns: { kind: "host", pool: string, status: string, ... }
//
// host.BindWorkplace(workspaceID) → void
//
//	Go: Computer.BindWorkplace(workspaceID string)
//
// host.Workplace() → WorkspaceFS | null
//
//	Go: Computer.Workplace() workspace.FS
//	Returns WorkspaceFS if a workplace is bound, null otherwise.
func NewHostObject(v8ctx *v8go.Context, pool string) (*v8go.Value, error) {
	// TODO: Phase 2 implementation
	// 1. Create JS object via v8go.NewObjectTemplate
	// 2. Set read-only property: pool
	// 3. Bind methods via unified Computer interface:
	//    - Exec, Stream, VNC, Proxy, ComputerInfo, BindWorkplace, Workplace
	return nil, nil
}
