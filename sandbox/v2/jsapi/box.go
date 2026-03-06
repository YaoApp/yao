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
// box.Stream(cmd, options?) → ExecStream
//
//	Go: Box.Stream(ctx, cmd []string, opts ...ExecOption) (*ExecStream, error)
//
//	JS returns: {
//	  stdout: ReadableStream,                   ← ExecStream.Stdout
//	  stderr: ReadableStream,                   ← ExecStream.Stderr
//	  stdin:  WritableStream,                   ← ExecStream.Stdin
//	  wait:   function() → number,              ← ExecStream.Wait() (int, error)
//	  cancel: function() → void                 ← ExecStream.Cancel()
//	}
//
// box.Attach(port, options?) → ServiceConn
//
//	Go: Box.Attach(ctx, port int, opts ...AttachOption) (*ServiceConn, error)
//
//	JS args:
//	  port:    number                           → port int
//	  options: {                                → AttachOption functional options
//	    protocol: "ws"|"sse",                   → WithProtocol(protocol)
//	    path:     string,                       → WithPath(path)
//	    headers:  object                        → WithHeaders(map[string]string)
//	  }
//	JS returns: {
//	  url:    string,                           ← ServiceConn.URL
//	  read:   function() → Uint8Array,          ← ServiceConn.Read() ([]byte, error)
//	  write:  function(data) → void,            ← ServiceConn.Write(data) error
//	  events: AsyncIterable<Uint8Array>,        ← ServiceConn.Events <-chan []byte
//	  close:  function() → void                 ← ServiceConn.Close() error
//	}
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
//	Go: Box.Workspace() workspace.FS
//	      Box.WorkspaceID() string
//	Returns: WorkspaceFS object (see workspace/jsapi/fs.go)
//	         Uses box.WorkspaceID() to create NewFSObject
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
	//    - Attach  → sandbox.M().Get(id).Attach(ctx, port, opts...)
	//    - VNC     → sandbox.M().Get(id).VNC(ctx)
	//    - Proxy   → sandbox.M().Get(id).Proxy(ctx, port, path)
	//    - Workspace → NewFSObject(v8ctx, sandbox.M().Get(id).WorkspaceID())
	//    - Info    → sandbox.M().Get(id).Info(ctx) → JS object
	//    - Start   → sandbox.M().Get(id).Start(ctx)
	//    - Stop    → sandbox.M().Get(id).Stop(ctx)
	//    - Remove  → sandbox.M().Get(id).Remove(ctx)
	return nil, nil
}
