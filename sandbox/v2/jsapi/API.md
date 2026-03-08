# Sandbox JavaScript API

All methods are available on the global `sandbox` object. No constructor needed.

## Quick Start

```javascript
// Create a sandbox container
const box = sandbox.Create({ image: "node:20", owner: "user-123" })

// Execute a command
const result = box.Exec(["node", "-e", "console.log('hello')"])
console.log(result.stdout) // "hello\n"

// Clean up
box.Remove()
```

---

## Static Methods

### sandbox.Create(options) → Box

Create a new sandbox container. If `options.id` is set and a sandbox with that ID already exists, returns the existing one (GetOrCreate semantics).

```javascript
const box = sandbox.Create({
  image:        "node:20",          // required — container image
  owner:        "user-123",         // required — owner identifier
  pool:         "gpu",              // optional — pool name (default: first pool)
  id:           "my-sandbox",       // optional — if set, uses GetOrCreate
  workdir:      "/app",             // optional — working directory
  user:         "1000:1000",        // optional — UID:GID
  env:          { NODE_ENV: "dev" },// optional — environment variables
  memory:       536870912,          // optional — memory limit in bytes (512MB)
  cpus:         1.5,                // optional — CPU limit
  vnc:          true,               // optional — enable VNC desktop
  ports:        [                   // optional — port mappings
    { container_port: 3000, host_port: 3000, host_ip: "", protocol: "tcp" }
  ],
  policy:       "session",          // optional — "oneshot"|"session"|"longrunning"|"persistent"
  idle_timeout: 600000,             // optional — idle timeout in ms (10min)
  stop_timeout: 30000,              // optional — stop timeout in ms
  workspace_id: "ws-abc",           // optional — bind a workspace
  mount_mode:   "rw",               // optional — "rw"|"ro"
  mount_path:   "/workspace",       // optional — mount path in container
  labels:       { team: "backend" } // optional — custom labels
})
```

### sandbox.Get(id) → Box | null

Get an existing sandbox by ID. Returns `null` if not found.

```javascript
const box = sandbox.Get("my-sandbox")
if (box) {
  console.log(box.id, box.owner, box.pool)
}
```

### sandbox.List(filter?) → BoxInfo[]

List all sandboxes, optionally filtered.

```javascript
// All sandboxes
const all = sandbox.List()

// Filter by owner
const mine = sandbox.List({ owner: "user-123" })

// Filter by pool and labels
const gpu = sandbox.List({ pool: "gpu", labels: { team: "ml" } })
```

Each element in the returned array:

```javascript
{
  id:            "sb-xxx",
  container_id:  "abc123...",
  pool:          "default",
  owner:         "user-123",
  status:        "running",       // "running"|"stopped"|"creating"|...
  image:         "node:20",
  vnc:           false,
  policy:        "session",
  labels:        { team: "backend" },
  created_at:    "2026-03-07T10:00:00Z",
  last_active:   "2026-03-07T10:05:00Z",
  process_count: 2
}
```

### sandbox.Delete(id) → void

Remove a sandbox and its container.

```javascript
sandbox.Delete("my-sandbox")
```

### sandbox.Host(pool?) → Host

Get a Host object for executing commands directly on the Tai host machine (no container). Only available when the pool's Tai server has `host_exec` capability.

```javascript
const host = sandbox.Host()        // default pool
const gpu  = sandbox.Host("gpu")   // specific pool
```

### sandbox.GetNode(taiID) → NodeInfo | null

Get information about a registered node by its Tai ID.

```javascript
const node = sandbox.GetNode("tai-abc123")
if (node) {
  console.log(node.status, node.system.hostname)
}
```

### sandbox.Nodes() → NodeInfo[]

List all registered nodes.

```javascript
const nodes = sandbox.Nodes()
nodes.forEach(function(n) {
  console.log(n.tai_id, n.status, n.pool, n.system.os)
})
```

### sandbox.NodesByTeam(teamID) → NodeInfo[]

List nodes belonging to a specific team.

```javascript
const nodes = sandbox.NodesByTeam("team-001")
```

---

## Box Object

Returned by `sandbox.Create()` and `sandbox.Get()`. Holds a sandbox ID internally; all operations delegate to the backend.

### Properties (read-only)

| Property | Type | Description |
|----------|------|-------------|
| `box.id` | string | Sandbox ID |
| `box.owner` | string | Owner identifier |
| `box.pool` | string | Pool name |

### box.Exec(cmd, options?) → ExecResult

Execute a command in the container and wait for it to finish.

```javascript
const result = box.Exec(["ls", "-la", "/app"])
console.log(result.exit_code) // 0
console.log(result.stdout)    // file listing
console.log(result.stderr)    // empty string
```

Options:

```javascript
box.Exec(["npm", "test"], {
  workdir: "/app",
  env:     { CI: "true" },
  timeout: 60000              // ms
})
```

Return value:

```javascript
{
  exit_code: 0,       // process exit code
  stdout:    "...",   // captured stdout (string)
  stderr:    "..."    // captured stderr (string)
}
```

### box.Stream(cmd, callback) / box.Stream(cmd, options, callback)

Execute a command with streaming output via callback. The call blocks until the process exits.

Callback signature: `function(type, data)`
- `type = "stdout"` → `data` is a string chunk from stdout
- `type = "stderr"` → `data` is a string chunk from stderr
- `type = "exit"` → `data` is the exit code (number)

```javascript
// Basic
box.Stream(["npm", "run", "dev"], function(type, data) {
  if (type === "stdout") console.log(data)
  if (type === "stderr") console.log("[ERR]", data)
  if (type === "exit")   console.log("exited:", data)
})

// With options
box.Stream(["npm", "test"], {
  workdir: "/app",
  env:     { CI: "true" },
  timeout: 60000
}, function(type, data) {
  console.log(type, data)
})
```

### box.Attach(port, options?) → string

Get a WebSocket or SSE endpoint URL for a service running inside the container. Use this for persistent connections (WS/SSE). For plain HTTP requests, use `box.Proxy()` instead.

```javascript
// Get WebSocket URL
const wsURL = box.Attach(3000, { protocol: "ws", path: "/ws" })
// "ws://host:8099/container-id:3000/ws"

// Get SSE URL
const sseURL = box.Attach(8080, { protocol: "sse", path: "/events" })
// "http://host:8099/container-id:8080/events"
```

Options:

```javascript
{
  protocol: "ws" | "sse",   // default "ws"; affects URL scheme (ws:// vs http://)
  path:     "/ws"           // optional URL path suffix
}
```

### box.VNC() → string

Get the VNC WebSocket URL for a VNC-enabled sandbox.

```javascript
const url = box.VNC()
// "ws://host:16080/vnc/sb-xxx"
```

### box.Proxy(port, path?) → string

Get an HTTP proxy URL for a port inside the container. Use this for plain HTTP requests. For WebSocket/SSE connections, use `box.Attach()` instead.

```javascript
const url = box.Proxy(3000)
// "http://host:8099/proxy/sb-xxx/3000/"

const url = box.Proxy(8080, "/api/v1")
// "http://host:8099/proxy/sb-xxx/8080/api/v1"
```

### box.Workspace() → WorkspaceFS

Access the workspace filesystem bound to this sandbox. The WorkspaceFS object is implemented in the `workspace/jsapi` package; this method returns it directly by calling `workspace.NewFSObject(v8ctx, box.WorkspaceID())`.

```javascript
const ws = box.Workspace()
const content = ws.ReadFile("src/main.go")
ws.WriteFile("src/main.go", "package main\n...")
ws.MkdirAll("src/utils")
const entries = ws.ReadDir("src/")
```

See [WorkspaceFS Object](#workspacefs-object) for the full method list.

### box.Info() → BoxInfo

Get current status information.

```javascript
const info = box.Info()
console.log(info.status, info.process_count, info.last_active)
```

Returns the same structure as elements in `sandbox.List()`.

### box.Start() → void

Start a stopped sandbox.

```javascript
box.Start()
```

### box.Stop() → void

Stop a running sandbox.

```javascript
box.Stop()
```

### box.Remove() → void

Remove the sandbox and its container.

```javascript
box.Remove()
```

---

## Host Object

Returned by `sandbox.Host()`. Executes commands directly on the Tai host machine without a container. Requires the pool to have `host_exec` capability.

### Properties (read-only)

| Property | Type | Description |
|----------|------|-------------|
| `host.pool` | string | Pool name |

### host.Exec(cmd, args, options?) → HostExecResult

Execute a command on the host and wait for it to finish.

```javascript
const result = host.Exec("ls", ["-la", "/workspace"])
console.log(result.exit_code)    // 0
console.log(result.stdout)       // file listing
console.log(result.duration_ms)  // execution time
```

Options:

```javascript
host.Exec("python3", ["train.py"], {
  workdir:    "/workspace/ml",
  env:        { CUDA_VISIBLE_DEVICES: "0" },
  stdin:      "input data",
  timeout:    300000,            // ms
  max_output: 10485760           // bytes (10MB)
})
```

Return value:

```javascript
{
  exit_code:   0,
  stdout:      "...",           // UTF-8 string
  stderr:      "...",           // UTF-8 string
  duration_ms: 1234,            // execution time in ms
  error:       "",              // error message (empty on success)
  truncated:   false            // true if output was truncated by max_output
}
```

### host.Stream(cmd, args, callback) / host.Stream(cmd, args, options, callback)

Execute a command on the host with streaming output via callback. The call blocks until the process exits.

Callback signature: same as `box.Stream` — `function(type, data)`.

```javascript
// Basic
host.Stream("tail", ["-f", "/var/log/app.log"], function(type, data) {
  if (type === "stdout") console.log(data)
})

// With options
host.Stream("python3", ["train.py"], {
  workdir: "/workspace/ml",
  timeout: 3600000
}, function(type, data) {
  if (type === "stderr") console.log("[WARN]", data)
  if (type === "exit")   console.log("done, code:", data)
})
```

### host.Workspace(sessionID) → WorkspaceFS

Access a workspace on the host by session ID. Same as `box.Workspace()`, the WorkspaceFS object is implemented in the `workspace/jsapi` package; this method calls `workspace.NewFSObject(v8ctx, sessionID)`.

```javascript
const ws = host.Workspace("my-session")
ws.ReadFile("config.yml")
ws.WriteFile("output.json", JSON.stringify(data))
ws.ReadDir("results/")
```

See [WorkspaceFS Object](#workspacefs-object) for the full method list.

---

## NodeInfo Object

Returned by `sandbox.GetNode()`, `sandbox.Nodes()`, `sandbox.NodesByTeam()`. Read-only view of a registered Tai node.

```javascript
{
  tai_id:       "tai-abc123",
  machine_id:   "m-xyz",
  version:      "1.2.0",
  mode:         "direct",          // "direct" | "tunnel"
  addr:         "192.168.1.100",
  status:       "online",          // "online" | "offline" | "connecting"
  pool:         "gpu",
  connected_at: "2026-03-07T08:00:00Z",
  last_ping:    "2026-03-07T10:05:00Z",
  ports: {
    grpc:   19100,
    http:   8099,
    vnc:    16080,
    docker: 12375,
    k8s:    16443
  },
  capabilities: {
    docker:    true,
    k8s:       false,
    host_exec: true
  },
  system: {
    os:        "linux",
    arch:      "amd64",
    hostname:  "gpu-server-01",
    num_cpu:   16,
    total_mem: 68719476736         // bytes (64GB)
  }
}
```

---

## WorkspaceFS Object

Returned by `box.Workspace()`, `host.Workspace()`, `workspace.Get()`, and `workspace.Create()`.

### Properties (read-only)

| Property | Type | Description |
|----------|------|-------------|
| `ws.id` | string | Workspace ID |
| `ws.name` | string | Workspace name |
| `ws.node` | string | Node name |

### Methods

| Method | Returns | Description |
|--------|---------|-------------|
| `ws.ReadFile(path)` | `string` | Read file content as UTF-8 string |
| `ws.WriteFile(path, data, perm?)` | `void` | Write string data to file. `perm` defaults to `0644` |
| `ws.ReadDir(path?)` | `DirEntry[]` | List directory contents. Defaults to root |
| `ws.Stat(path)` | `FileInfo` | Get file/directory metadata |
| `ws.MkdirAll(path, perm?)` | `void` | Create directory tree. `perm` defaults to `0755` |
| `ws.Remove(path)` | `void` | Remove a file |
| `ws.RemoveAll(path)` | `void` | Remove a file or directory recursively |
| `ws.Rename(from, to)` | `void` | Rename/move a file or directory |

Return types:

```javascript
// DirEntry
{ name: "main.go", is_dir: false, size: 1234 }

// FileInfo
{ name: "main.go", size: 1234, is_dir: false, mod_time: "2026-03-07T10:00:00Z" }
```

---

## Examples

### Run a build and check output

```javascript
const box = sandbox.Create({
  image: "golang:1.23",
  owner: "ci-bot",
  workspace_id: "ws-project-abc"
})

const build = box.Exec(["go", "build", "./..."], {
  workdir: "/workspace",
  timeout: 120000
})

if (build.exit_code !== 0) {
  console.log("Build failed:", build.stderr)
  box.Remove()
  throw new Error("build failed")
}

const test = box.Exec(["go", "test", "./..."], {
  workdir: "/workspace",
  env: { CGO_ENABLED: "0" }
})

console.log("Tests:", test.exit_code === 0 ? "PASS" : "FAIL")
box.Remove()
```

### Stream a long-running process

```javascript
const box = sandbox.Create({
  image: "node:20",
  owner: "user-123",
  policy: "session"
})

box.Exec(["npm", "install"], { workdir: "/app" })

box.Stream(["npm", "run", "dev"], { workdir: "/app" }, function(type, data) {
  if (type === "stdout") console.log(data)
  if (type === "stderr") console.log("[ERR]", data)
  if (type === "exit")   console.log("dev server exited:", data)
})
```

### Host execution for GPU workloads

```javascript
const host = sandbox.Host("gpu")

const result = host.Exec("nvidia-smi", [])
console.log(result.stdout)

host.Exec("python3", ["train.py", "--epochs=10"], {
  workdir: "/workspace/ml",
  env: { CUDA_VISIBLE_DEVICES: "0,1" },
  timeout: 3600000
})
```

### Query cluster nodes

```javascript
const nodes = sandbox.Nodes()

// Find online GPU nodes
const gpuNodes = nodes.filter(function(n) {
  return n.status === "online" && n.pool === "gpu"
})

console.log("Available GPU nodes:", gpuNodes.length)
gpuNodes.forEach(function(n) {
  console.log(
    n.tai_id,
    n.system.hostname,
    n.system.num_cpu + " CPUs",
    Math.round(n.system.total_mem / 1073741824) + "GB RAM"
  )
})
```

### Workspace file operations

```javascript
const ws = workspace.Create({
  name:  "my-project",
  owner: "user-123",
  node:  "default"
})

ws.MkdirAll("src/utils")
ws.WriteFile("src/main.go", 'package main\n\nfunc main() {\n\tprintln("hello")\n}\n')
ws.WriteFile("go.mod", "module myproject\n\ngo 1.23\n")

const entries = ws.ReadDir("src/")
entries.forEach(function(e) {
  console.log(e.name, e.is_dir ? "(dir)" : e.size + " bytes")
})

const content = ws.ReadFile("src/main.go")
console.log(content)
```

### Permission check pattern

```javascript
const auth = Authorized()
if (!auth) throw new Error("not authenticated")

const box = sandbox.Get(id)
if (!box) throw new Error("sandbox not found")
if (box.owner !== auth.user_id) throw new Error("permission denied")

box.Exec(["ls", "-la"])
```
