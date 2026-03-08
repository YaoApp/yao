package jsapi

import (
	"rogchap.com/v8go"
)

// sbHost: `sandbox.Host(pool?)` → Host
//
// Go: Manager.Host(ctx, pool) (*Host, error)
//
// Args:
//
//	pool: string (optional) — pool name; empty = default pool
//
// Returns: Host object if the pool has host_exec capability, otherwise throws.
//
// Host executes commands on the Tai host machine (no container). Available only
// when the pool's Tai server exposes HostExec gRPC.
func sbHost(info *v8go.FunctionCallbackInfo) *v8go.Value {
	// TODO: Phase 2
	// 1. pool := ""; if len(info.Args()) > 0 && info.Args()[0].IsString() { pool = info.Args()[0].String() }
	// 2. host, err := sandbox.M().Host(ctx, pool)
	// 3. if err != nil { throw in V8 }
	// 4. Return NewHostObject(v8ctx, host.Pool())
	return v8go.Undefined(info.Context().Isolate())
}

// NewHostObject creates a JS Host object backed by a pool name string.
// All methods delegate to the Go sandbox.M() singleton — no Go *Host passed to V8.
//
// # Properties (read-only)
//
//	host.pool  → string   // pool name  ← Host.Pool()
//
// # Methods — Go mapping
//
// host.Exec(cmd, args, options?) → HostExecResult
//
//	Go: Host.Exec(ctx, cmd string, args []string, opts ...HostExecOption) (*HostExecResult, error)
//
//	JS args:
//	  cmd:     string                    → cmd string
//	  args:    string[]                  → args []string
//	  options: {                         → HostExecOption
//	    workdir:    string,              → WithHostWorkDir(dir)
//	    env:        object,              → WithHostEnv(map[string]string)
//	    stdin:      string,              → WithHostStdin([]byte)
//	    timeout:    number,              → WithHostTimeout(ms int64)
//	    max_output: number              → WithHostMaxOutput(bytes int64)
//	  }
//	JS returns: {
//	  exit_code:   number,               ← HostExecResult.ExitCode
//	  stdout:      string (UTF-8),      ← HostExecResult.Stdout
//	  stderr:      string (UTF-8),      ← HostExecResult.Stderr
//	  duration_ms: number,              ← HostExecResult.DurationMs
//	  error:       string,              ← HostExecResult.Error
//	  truncated:   boolean              ← HostExecResult.Truncated
//	}
//
// host.Stream(cmd, args, callback) / host.Stream(cmd, args, options, callback)
//
//	Go: Host.Stream(ctx, cmd string, args []string, opts ...HostExecOption) (*HostExecStream, error)
//
//	Blocks until the process exits. The last argument must be a JS function.
//	Callback signature: function(type, data)
//	  type = "stdout" → data is string (chunk)
//	  type = "stderr" → data is string (chunk)
//	  type = "exit"   → data is number (exit code)
//
//	JS args:
//	  cmd:      string
//	  args:     string[]
//	  options:  { workdir, env, stdin, timeout, max_output }  (optional, same as host.Exec)
//	  callback: function(type, data)
//
// host.Workspace(sessionID) → WorkspaceFS
//
//	Implemented in workspace/jsapi package. This method calls:
//	  workspace.NewFSObject(v8ctx, sessionID)
//	and returns the resulting WorkspaceFS object directly.
func NewHostObject(v8ctx *v8go.Context, pool string) (*v8go.Value, error) {
	// TODO: Phase 2 implementation
	// 1. Create JS object via v8go.NewObjectTemplate
	// 2. Set read-only property: pool
	// 3. Bind methods: Exec, Stream, Workspace (each resolves Host via sandbox.M().Host(ctx, pool))
	return nil, nil
}
