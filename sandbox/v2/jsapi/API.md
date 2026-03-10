# Sandbox JavaScript API

All methods are available on the global `sandbox` object. No constructor needed.

## Quick Start

```javascript
// Create a container computer
const pc = sandbox.Create({ image: "node:20", owner: "user-123" })
const result = pc.Exec(["node", "-e", "console.log('hello')"])
console.log(result.stdout) // "hello\n"
pc.Remove()

// Or use the host directly (no container)
const host = sandbox.Host()
const info = host.Exec(["uname", "-a"])
console.log(info.stdout) // same ExecResult as box
```

Both `sandbox.Create()` and `sandbox.Host()` return a **Computer** object with the same interface. The `kind` property tells you which type it is.

---

## Static Methods

### sandbox.Create(options) → Computer

Create a new sandbox container. Returns a Computer (`kind = "box"`). If `options.id` is set and a sandbox with that ID already exists, returns the existing one (GetOrCreate semantics).

```javascript
const pc = sandbox.Create({
  image:        "node:20",          // required — container image
  owner:        "user-123",         // required — owner identifier
  node_id:      "192.168.1.10-19100", // optional — TaiID from registry (required unless workspace_id routes to a node)
  id:           "my-sandbox",       // optional — if set, uses GetOrCreate
  workdir:      "/app",             // optional — working directory
  user:         "1000:1000",        // optional — UID:GID
  env:          { NODE_ENV: "dev" },// optional — environment variables
  memory:       536870912,          // optional — memory limit in bytes (512MB)
  cpus:         1.5,               // optional — CPU limit
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

### sandbox.Get(id) → Computer | null

Get an existing sandbox by ID. Returns a Computer (`kind = "box"`) or `null` if not found.

```javascript
const pc = sandbox.Get("my-sandbox")
if (pc) {
  console.log(pc.kind, pc.id, pc.owner, pc.node_id)
}
```

### sandbox.List(filter?) → BoxInfo[]

List all sandboxes, optionally filtered.

```javascript
// All sandboxes
const all = sandbox.List()

// Filter by owner
const mine = sandbox.List({ owner: "user-123" })

// Filter by node_id (TaiID) and labels
const gpu = sandbox.List({ node_id: "10.0.0.5-19100", labels: { team: "ml" } })
```

Each element in the returned array:

```javascript
{
  id:            "sb-xxx",
  container_id:  "abc123...",
  node_id:       "192.168.1.10-19100",
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

### sandbox.Host(nodeID?) → Computer

Get a Computer (`kind = "host"`) for executing commands directly on the Tai host machine (no container). Only available when the node's Tai server has `host_exec` capability. The `nodeID` argument is the TaiID (e.g. `"192.168.1.10-19100"`).

```javascript
const host = sandbox.Host("192.168.1.10-19100")
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
  console.log(n.tai_id, n.status, n.display_name, n.system.os)
})
```

### sandbox.NodesByTeam(teamID) → NodeInfo[]

List nodes belonging to a specific team.

```javascript
const nodes = sandbox.NodesByTeam("team-001")
```

---

## Computer Object

Returned by `sandbox.Create()`, `sandbox.Get()`, and `sandbox.Host()`. This is the unified interface for all execution environments — containers and bare-metal hosts.

Use the `kind` property to check the type. Methods marked **box-only** throw an error when called on a host computer. `Proxy()` covers HTTP, WebSocket, and SSE — use it for all protocol access to container/host services.

### Properties (read-only)

| Property | Type | Description |
|----------|------|-------------|
| `pc.kind` | string | `"box"` or `"host"` |
| `pc.id` | string | Sandbox ID (box-only; empty for host) |
| `pc.owner` | string | Owner identifier (box-only; empty for host) |
| `pc.node_id` | string | TaiID (e.g. `"192.168.1.10-19100"`, `"local"`) |

### pc.Exec(cmd, options?) → ExecResult

Execute a command and wait for it to finish.

```javascript
const result = pc.Exec(["ls", "-la", "/app"])
console.log(result.exit_code) // 0
console.log(result.stdout)    // file listing
```

Options:

```javascript
pc.Exec(["python3", "train.py"], {
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

### pc.Stream(cmd, callback) / pc.Stream(cmd, options, callback)

Execute a command with streaming output via callback. The call blocks until the process exits.

Callback signature: `function(type, data)`
- `type = "stdout"` → `data` is a string chunk from stdout
- `type = "stderr"` → `data` is a string chunk from stderr
- `type = "exit"` → `data` is the exit code (number)

```javascript
pc.Stream(["npm", "run", "dev"], function(type, data) {
  if (type === "stdout") console.log(data)
  if (type === "stderr") console.log("[ERR]", data)
  if (type === "exit")   console.log("exited:", data)
})

// With options
pc.Stream(["npm", "test"], {
  workdir: "/app",
  env:     { CI: "true" },
  timeout: 60000
}, function(type, data) {
  console.log(type, data)
})
```

### pc.VNC() → string

Get the VNC WebSocket URL.

- **Box**: routes to the container's VNC server (`:5900`)
- **Host**: routes to the Tai host via `__host__` identifier (configurable via `host_vnc_port`)

```javascript
const url = pc.VNC()
// Box:  "ws://tai-host:16080/vnc/container-id/ws"
// Host: "ws://tai-host:16080/vnc/__host__/ws"
```

If no VNC server is running, the WebSocket connection will fail — handle this in the caller.

### pc.Proxy(port, path?) → string

Get a proxy URL for a service port. Supports HTTP, WebSocket (`ws://`), and SSE — the Tai proxy handles protocol upgrades automatically.

- **Box**: routes to `container-ip:{port}`
- **Host**: routes to `127.0.0.1:{port}` on the Tai machine via `__host__`

```javascript
const url = pc.Proxy(3000)
// Box:  "http://tai-host:8099/container-id:3000/"
// Host: "http://tai-host:8099/__host__:3000/"

const url = pc.Proxy(8080, "/api/v1")
// Box:  "http://tai-host:8099/container-id:8080/api/v1"
// Host: "http://tai-host:8099/__host__:8080/api/v1"
```

### pc.ComputerInfo() → ComputerInfo

Get identity and registry information.

```javascript
const info = pc.ComputerInfo()
console.log(info.kind)        // "box" or "host"
console.log(info.node_id)    // TaiID
console.log(info.system.os)   // "linux" | "windows" | "darwin"
console.log(info.status)      // "running" | "stopped" | ...
```

Returns a [ComputerInfo](#computerinfo-object) object.

### pc.BindWorkplace(workspaceID) → void

Bind a workspace to this computer for the current session. For box computers created with a `workspace_id` option, the workspace is already bound at creation time — calling `BindWorkplace` overrides it.

```javascript
pc.BindWorkplace("ws-project-abc")
```

### pc.Workplace() → WorkspaceFS | null

Access the workspace filesystem bound via `BindWorkplace()`. Returns `null` if no workspace is bound. ("Workplace" is the binding on a Computer; "Workspace" is the filesystem it points to.)

```javascript
pc.BindWorkplace("ws-project-abc")
const ws = pc.Workplace()
ws.ReadFile("config.yml")
ws.WriteFile("output.json", JSON.stringify(data))
```

See [WorkspaceFS Object](#workspacefs-object) for the full method list.

### pc.Info() → BoxInfo — box-only

Get current container runtime status (process count, last active time, etc.). For node-level identity info (OS, CPU, capabilities), use `ComputerInfo()` instead. Throws on host computers.

```javascript
const info = pc.Info()
console.log(info.status, info.process_count, info.last_active)
```

Returns the same structure as elements in `sandbox.List()`.

### pc.Start() → void — box-only

Start a stopped container. Throws on host computers.

```javascript
pc.Start()
```

### pc.Stop() → void — box-only

Stop a running container. Throws on host computers.

```javascript
pc.Stop()
```

### pc.Remove() → void — box-only

Remove the container. Throws on host computers.

```javascript
pc.Remove()
```

---

## ComputerInfo Object

Returned by `pc.ComputerInfo()`. Read-only snapshot of a Computer's identity and state.

```javascript
{
  kind:          "box",              // "box" | "host"
  node_id:       "192.168.1.10-19100", // TaiID
  tai_id:        "tai-abc123",
  machine_id:    "m-xyz",
  version:       "1.2.3",
  mode:          "direct",           // "direct" | "tunnel"
  status:        "running",
  capabilities:  { docker: true, k8s: false, host_exec: true },
  system: {
    os:        "linux",
    arch:      "amd64",
    hostname:  "gpu-server-01",
    num_cpu:   16,
    total_mem: 68719476736
  },

  // Box-only fields (empty/zero for host)
  box_id:        "sb-xxx",
  container_id:  "abc123...",
  owner:         "user-123",
  image:         "node:20",
  policy:        "session",
  labels:        { team: "backend" }
}
```

---

## NodeInfo Object

Returned by `sandbox.GetNode()`, `sandbox.Nodes()`, `sandbox.NodesByTeam()`. Read-only view of a registered Tai node.

```javascript
{
  tai_id:       "tai-abc123",
  machine_id:   "m-xyz",
  version:      "1.2.3",
  mode:         "direct",          // "direct" | "tunnel"
  addr:         "tai://192.168.1.100:19100",
  status:       "online",          // "online" | "offline" | "connecting"
  display_name: "GPU Node",        // optional human-readable name for UI
  node_id:      "gpu",
  connected_at: "2026-03-07T08:00:00Z",
  last_ping:    "2026-03-07T10:05:00Z",
  ports: {
    grpc:     19100,
    http:     8099,
    vnc:      16080,
    docker:   12375,
    k8s:      16443,
    host_vnc: 5900               // VNC port on host for __host__ routing
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
    total_mem: 68719476736       // bytes (64GB)
  }
}
```

---

## WorkspaceFS Object

Returned by `pc.Workplace()`, `workspace.Get()`, and `workspace.Create()`.

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
const pc = sandbox.Create({
  image: "golang:1.23",
  owner: "ci-bot",
  workspace_id: "ws-project-abc"
})

const build = pc.Exec(["go", "build", "./..."], {
  workdir: "/workspace",
  timeout: 120000
})

if (build.exit_code !== 0) {
  console.log("Build failed:", build.stderr)
  pc.Remove()
  throw new Error("build failed")
}

const test = pc.Exec(["go", "test", "./..."], {
  workdir: "/workspace",
  env: { CGO_ENABLED: "0" }
})

console.log("Tests:", test.exit_code === 0 ? "PASS" : "FAIL")
pc.Remove()
```

### Stream a long-running process

```javascript
const pc = sandbox.Create({
  image: "node:20",
  owner: "user-123",
  policy: "session"
})

pc.Exec(["npm", "install"], { workdir: "/app" })

pc.Stream(["npm", "run", "dev"], { workdir: "/app" }, function(type, data) {
  if (type === "stdout") console.log(data)
  if (type === "stderr") console.log("[ERR]", data)
  if (type === "exit")   console.log("dev server exited:", data)
})
```

### Host execution for GPU workloads

```javascript
const host = sandbox.Host("10.0.0.5-19100")

const result = host.Exec(["nvidia-smi"])
console.log(result.stdout)

const train = host.Exec(["python3", "train.py", "--epochs=10"], {
  workdir: "/workspace/ml",
  env: { CUDA_VISIBLE_DEVICES: "0,1" },
  timeout: 3600000
})
if (train.exit_code !== 0) throw new Error("training failed: " + train.stderr)
```

### Uniform interface — same code for box and host

```javascript
function runTask(pc, cmd, opts) {
  const result = pc.Exec(cmd, opts)
  if (result.exit_code !== 0) {
    throw new Error(pc.kind + " exec failed: " + result.stderr)
  }
  return result.stdout
}

// Works the same for both
const box  = sandbox.Create({ image: "node:20", owner: "u1" })
const host = sandbox.Host("10.0.0.5-19100")

runTask(box,  ["node", "-e", "console.log('hi')"])
runTask(host, ["echo", "hello"])
```

### VNC and HTTP proxy

```javascript
const pc = sandbox.Create({
  image: "kasmweb/chrome:latest",
  owner: "user-123",
  vnc:   true
})

// Get VNC desktop URL
const vncURL = pc.VNC()
// "ws://tai-host:16080/vnc/container-id/ws"

// Get HTTP proxy to a web service inside the container
const appURL = pc.Proxy(3000)
// "http://tai-host:8099/container-id:3000/"

// Same methods work on host
const host = sandbox.Host("192.168.1.10-19100")
const hostVNC = host.VNC()
// "ws://tai-host:16080/vnc/__host__/ws"
```

### Query cluster nodes

```javascript
const nodes = sandbox.Nodes()

// Find online GPU nodes
const gpuNodes = nodes.filter(function(n) {
  return n.status === "online" && n.display_name === "gpu"  // n.display_name is optional label for UI
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
const pc = sandbox.Create({
  image: "node:20",
  owner: "user-123"
})

pc.BindWorkplace("ws-my-project")
const ws = pc.Workplace()

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

const pc = sandbox.Get(id)
if (!pc) throw new Error("sandbox not found")
if (pc.owner !== auth.user_id) throw new Error("permission denied")

pc.Exec(["ls", "-la"])
```
