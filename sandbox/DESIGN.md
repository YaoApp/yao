# Sandbox Design

## 1. Overview

Sandbox provides **persistent Docker containers** as isolated execution environments for external CLI agents like Claude Code.

### Why Docker?

- **Persistence**: Claude installs dependencies (npm, pip, apt), which must persist across sessions
- **Cross-platform**: Works on Linux, macOS, and Windows
- **Strong isolation**: Process, filesystem, and network isolation
- **Mature ecosystem**: Well-documented, easy to maintain

### Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                           Yao Server                                 │
│                                                                      │
│  ┌────────────────────────────────────────────────────────────────┐  │
│  │                      Sandbox Manager                            │  │
│  │                                                                 │  │
│  │   containers: map[containerName]*Container                       │  │
│  │                                                                 │  │
│  │   - GetOrCreate(userID, chatID) → get or create container       │  │
│  │   - Exec(containerName, cmd) → execute command in container     │  │
│  │   - Stop(containerName) → stop container (preserve data)        │  │
│  │   - Remove(containerName) → delete container                    │  │
│  │                                                                 │  │
│  └──────────────────────────┬─────────────────────────────────────┘  │
│                             │                                        │
│              ┌──────────────┼──────────────┐                         │
│              │              │              │                         │
│              ▼              ▼              ▼                         │
│     ┌──────────────┐ ┌──────────────┐ ┌──────────────┐               │
│     │ Container-A  │ │ Container-B  │ │ Container-C  │               │
│     │ (user1-chat1)│ │ (user1-chat2)│ │ (user2-chat1)│               │
│     │              │ │              │ │              │               │
│     │ - Claude CLI │ │ - Claude CLI │ │ - Claude CLI │               │
│     │ - Node.js    │ │ - Python     │ │ - Go         │               │
│     │ - User code  │ │ - User code  │ │ - User code  │               │
│     └──────┬───────┘ └──────┬───────┘ └──────┬───────┘               │
│            │                │                │                       │
│     ───────┴────────────────┴────────────────┴───────                │
│                      Unix Socket IPC                                 │
│               (one socket per container)                             │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 2. Container Lifecycle

```
Create ──────────► Running ──────────► Stopped ──────────► Removed
(docker create)   (docker start)      (docker stop)       (docker rm)
     │                 │                   │                   │
     │                 │                   │                   │
     ▼                 ▼                   ▼                   ▼
First request      Execute tasks       Idle timeout        Cleanup policy
                   (persistent)        (data preserved)    (manual/scheduled)
```

### Container States

| State     | Description                                    |
| --------- | ---------------------------------------------- |
| `created` | Container created, not started                 |
| `running` | Container running, can execute commands        |
| `stopped` | Container stopped, data preserved, can restart |
| `removed` | Container deleted                              |

### Naming Convention

```
yao-sandbox-{userID}-{chatID}

Example: yao-sandbox-u123-c456
```

---

## 3. IPC Communication

### Problem

Claude CLI runs inside the sandbox but needs to call Yao's MCP Tools (Yao Processes) which run outside.

### Solution: Unix Socket + MCP JSON-RPC

```
┌────────────────────────────────────────────────────────────────────┐
│                        Docker Container                            │
│                                                                    │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │  Claude CLI                                                   │  │
│  │     │                                                         │  │
│  │     │ .mcp.json: "yao" → stdio                               │  │
│  │     ▼                                                         │  │
│  │  ┌──────────────────────────────────────────────────────────┐ │  │
│  │  │  yao-bridge (lightweight binary)                         │ │  │
│  │  │  stdin/stdout ↔ /tmp/yao.sock                            │ │  │
│  │  └──────────────────────────────────────────────────────────┘ │  │
│  └──────────────────────────────────────────────────────────────┘  │
│                          │                                         │
│                   /tmp/yao.sock                                    │
│                          │                                         │
└──────────────────────────┼─────────────────────────────────────────┘
                           │
              Unix Socket (bind mount)
                           │
┌──────────────────────────┼─────────────────────────────────────────┐
│                          │                                         │
│              {YAO_DATA_ROOT}/sandbox/ipc/{sessionID}.sock          │
│                          │                                         │
│  ┌───────────────────────▼──────────────────────────────────────┐  │
│  │                    IPC Server (goroutine)                     │  │
│  │                                                               │  │
│  │   MCP JSON-RPC Methods:                                       │  │
│  │   - initialize      → handshake                               │  │
│  │   - tools/list      → return authorized Yao MCP tools         │  │
│  │   - tools/call      → execute process.New(name, args...)      │  │
│  │   - resources/list  → list Yao resources                      │  │
│  │   - resources/read  → read Yao resource                       │  │
│  │                                                               │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                    │
│                           Yao Server                               │
└────────────────────────────────────────────────────────────────────┘
```

### Protocol

- **Format**: MCP standard JSON-RPC 2.0 over NDJSON (newline-delimited JSON)
- **Transport**: Unix Socket
- **Bridge**: `yao-bridge` binary converts stdio ↔ socket

---

## 4. Core Interfaces

### 4.1 Sandbox Manager

```go
// sandbox/manager.go
package sandbox

type Manager struct {
    mu           sync.Mutex         // Protects creation
    containers   sync.Map           // containerName → *Container
    running      int32              // Running container count
    ipcManager   *ipc.Manager
    dockerClient *docker.Client
    config       *Config
}

var ErrTooManyContainers = errors.New("sandbox: too many running containers, please try again later")

type Config struct {
    Image         string        // Docker image, default: yao/sandbox:latest
    WorkspaceRoot string        // Host workspace root directory
    IPCDir        string        // IPC socket directory
    MaxContainers int           // Maximum concurrent containers
    IdleTimeout   time.Duration // Idle timeout before stopping container
    MaxMemory     string        // Memory limit, e.g., "2g"
    MaxCPU        float64       // CPU limit, e.g., 1.0
}

type Container struct {
    ID          string
    Name        string    // yao-sandbox-{userID}-{chatID}
    UserID      string
    ChatID      string
    Status      string    // created, running, stopped
    CreatedAt   time.Time
    LastUsedAt  time.Time
    IPCSession  *ipc.Session
}
```

### 4.2 Manager Methods

```go
// GetOrCreate returns existing container or creates new one
// Returns ErrTooManyContainers if limit exceeded
func (m *Manager) GetOrCreate(ctx context.Context, userID, chatID string) (*Container, error)

// Stream executes command and returns stdout reader
func (m *Manager) Stream(ctx context.Context, containerName string, cmd []string, opts *ExecOptions) (io.ReadCloser, error)

// Exec executes command and waits for completion
func (m *Manager) Exec(ctx context.Context, containerName string, cmd []string, opts *ExecOptions) (*ExecResult, error)

// Stop stops container but preserves data
func (m *Manager) Stop(ctx context.Context, containerName string) error

// Start starts a stopped container
func (m *Manager) Start(ctx context.Context, containerName string) error

// Remove deletes container and its data
func (m *Manager) Remove(ctx context.Context, containerName string) error

// List returns all containers for a user
func (m *Manager) List(ctx context.Context, userID string) ([]*Container, error)

// Cleanup stops idle containers
func (m *Manager) Cleanup(ctx context.Context) error
```

### 4.3 Filesystem Methods

```go
// WriteFile writes content to a file in container
func (m *Manager) WriteFile(ctx context.Context, containerName, path string, content []byte) error

// ReadFile reads content from a file in container
func (m *Manager) ReadFile(ctx context.Context, containerName, path string) ([]byte, error)

// ListDir lists directory contents in container
func (m *Manager) ListDir(ctx context.Context, containerName, path string) ([]FileInfo, error)

// Stat returns file info
func (m *Manager) Stat(ctx context.Context, containerName, path string) (*FileInfo, error)

// MkDir creates directory in container
func (m *Manager) MkDir(ctx context.Context, containerName, path string) error

// Remove removes file or directory in container
func (m *Manager) RemoveFile(ctx context.Context, containerName, path string) error

// CopyToContainer copies file/directory from host to container
func (m *Manager) CopyToContainer(ctx context.Context, containerName, hostPath, containerPath string) error

// CopyFromContainer copies file/directory from container to host
func (m *Manager) CopyFromContainer(ctx context.Context, containerName, containerPath, hostPath string) error

// FileInfo represents file metadata
type FileInfo struct {
    Name    string
    Path    string
    Size    int64
    Mode    os.FileMode
    ModTime time.Time
    IsDir   bool
}
```

### 4.4 ExecOptions

```go
type ExecOptions struct {
    WorkDir string            // Working directory inside container
    Env     map[string]string // Environment variables
    Stdin   io.Reader         // Standard input
    Timeout time.Duration     // Execution timeout
}

type ExecResult struct {
    ExitCode int
    Stdout   string
    Stderr   string
}
```

---

## 5. IPC System

### 5.1 IPC Session

```go
// ipc/session.go
package ipc

type Session struct {
    ID         string           // Usually equals chatID
    SocketPath string           // {IPCDir}/{id}.sock
    Listener   net.Listener
    Conn       net.Conn
    Context    *AgentContext
    MCPTools   map[string]*MCPTool
    cancel     context.CancelFunc
}

type AgentContext struct {
    UserID   string
    ChatID   string
    Locale   string
}

type MCPTool struct {
    Name        string
    Description string
    Process     string          // Yao process name
    InputSchema json.RawMessage // JSON Schema
}
```

### 5.2 IPC Manager

```go
// ipc/manager.go
type Manager struct {
    sessions sync.Map  // sessionID → *Session
    sockDir  string    // {YAO_DATA_ROOT}/sandbox/ipc/
}

// Create creates new IPC session
func (m *Manager) Create(ctx context.Context, sessionID string, agentCtx *AgentContext, mcpTools map[string]*MCPTool) (*Session, error)

// Close closes IPC session and cleans up
func (m *Manager) Close(sessionID string) error

// Get returns existing session
func (m *Manager) Get(sessionID string) (*Session, bool)
```

### 5.3 JSON-RPC Message Handling

```go
// JSON-RPC request structure
type JSONRPCRequest struct {
    JSONRPC string          `json:"jsonrpc"`
    ID      interface{}     `json:"id,omitempty"`
    Method  string          `json:"method"`
    Params  json.RawMessage `json:"params,omitempty"`
}

// JSON-RPC response structure
type JSONRPCResponse struct {
    JSONRPC string          `json:"jsonrpc"`
    ID      interface{}     `json:"id,omitempty"`
    Result  interface{}     `json:"result,omitempty"`
    Error   *JSONRPCError   `json:"error,omitempty"`
}

type JSONRPCError struct {
    Code    int         `json:"code"`
    Message string      `json:"message"`
    Data    interface{} `json:"data,omitempty"`
}
```

### 5.4 Session Message Loop

```go
func (s *Session) serve(ctx context.Context) {
    defer s.cleanup()

    for {
        select {
        case <-ctx.Done():
            return
        default:
        }

        conn, err := s.Listener.Accept()
        if err != nil {
            continue
        }
        s.Conn = conn
        s.handleConnection(ctx, conn)
    }
}

func (s *Session) handleConnection(ctx context.Context, conn net.Conn) {
    defer conn.Close()

    scanner := bufio.NewScanner(conn)
    for scanner.Scan() {
        select {
        case <-ctx.Done():
            return
        default:
        }

        line := scanner.Text()
        response := s.handleMessage(line)
        if response != "" {
            conn.Write([]byte(response + "\n"))
        }
    }
}

func (s *Session) handleMessage(line string) string {
    var req JSONRPCRequest
    if err := json.Unmarshal([]byte(line), &req); err != nil {
        return s.errorResponse(nil, -32700, "Parse error")
    }

    switch req.Method {
    case "initialize":
        return s.handleInitialize(req)
    case "initialized":
        return "" // notification, no response
    case "tools/list":
        return s.handleListTools(req)
    case "tools/call":
        return s.handleCallTool(req)
    case "resources/list":
        return s.handleListResources(req)
    case "resources/read":
        return s.handleReadResource(req)
    default:
        return s.errorResponse(req.ID, -32601, "Method not found")
    }
}
```

### 5.5 Tool Call Handler

```go
func (s *Session) handleCallTool(req JSONRPCRequest) string {
    var params struct {
        Name      string                 `json:"name"`
        Arguments map[string]interface{} `json:"arguments"`
    }
    json.Unmarshal(req.Params, &params)

    // Check authorization
    tool, ok := s.MCPTools[params.Name]
    if !ok {
        return s.errorResponse(req.ID, -32602, "Tool not found or not authorized")
    }

    // Execute Yao Process
    proc := process.New(tool.Process, params.Arguments)
    proc.WithContext(s.Context)

    if err := proc.Execute(); err != nil {
        return s.toolErrorResponse(req.ID, params.Name, err)
    }
    defer proc.Release()

    result := proc.Value()
    return s.toolSuccessResponse(req.ID, result)
}
```

---

## 6. Docker Container Management

### 6.1 NewManager Constructor

```go
func NewManager(config *Config) (*Manager, error) {
    // Initialize Docker client
    cli, err := docker.NewClientWithOpts(docker.FromEnv, docker.WithAPIVersionNegotiation())
    if err != nil {
        return nil, fmt.Errorf("failed to create Docker client: %w", err)
    }

    // Ping Docker to verify connection
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    if _, err := cli.Ping(ctx); err != nil {
        return nil, fmt.Errorf("Docker not available: %w", err)
    }

    // Ensure directories exist
    os.MkdirAll(config.WorkspaceRoot, 0755)
    os.MkdirAll(config.IPCDir, 0755)

    m := &Manager{
        dockerClient: cli,
        config:       config,
        ipcManager:   ipc.NewManager(config.IPCDir),
    }

    // Start cleanup loop
    go m.startCleanupLoop(context.Background())

    return m, nil
}
```

### 6.2 GetOrCreate with Limit Check

```go
func (m *Manager) GetOrCreate(ctx context.Context, userID, chatID string) (*Container, error) {
    containerName := fmt.Sprintf("yao-sandbox-%s-%s", userID, chatID)

    // Check if container already exists (fast path)
    if c, ok := m.containers.Load(containerName); ok {
        container := c.(*Container)
        container.LastUsedAt = time.Now()
        return container, nil
    }

    // Use mutex for creation to avoid race condition
    m.mu.Lock()
    defer m.mu.Unlock()

    // Double-check after acquiring lock
    if c, ok := m.containers.Load(containerName); ok {
        container := c.(*Container)
        container.LastUsedAt = time.Now()
        return container, nil
    }

    // Check running container limit
    if m.running >= int32(m.config.MaxContainers) {
        return nil, ErrTooManyContainers
    }

    // Create new container
    container, err := m.createContainer(ctx, userID, chatID)
    if err != nil {
        return nil, err
    }

    // Store and increment counter
    m.containers.Store(containerName, container)
    m.running++

    return container, nil
}
```

### 6.3 Create Container (internal)

```go
func (m *Manager) createContainer(ctx context.Context, userID, chatID string) (*Container, error) {
    containerName := fmt.Sprintf("yao-sandbox-%s-%s", userID, chatID)

    // Ensure image exists, auto-pull if not
    if err := m.ensureImage(ctx, m.config.Image); err != nil {
        return nil, err
    }

    // Workspace directory
    workspaceHost := filepath.Join(m.config.WorkspaceRoot, userID, chatID)
    os.MkdirAll(workspaceHost, 0755)

    // IPC socket path
    sessionID := chatID
    ipcSocketHost := filepath.Join(m.config.IPCDir, sessionID+".sock")

    // Create container
    resp, err := m.dockerClient.ContainerCreate(ctx,
        &container.Config{
            Image:      m.config.Image,
            Cmd:        []string{"sleep", "infinity"}, // Keep running
            WorkingDir: "/workspace",
            Env: []string{
                "YAO_IPC_SOCKET=/tmp/yao.sock",
            },
        },
        &container.HostConfig{
            Binds: []string{
                workspaceHost + ":/workspace",
                ipcSocketHost + ":/tmp/yao.sock",
            },
            Resources: container.Resources{
                Memory:   parseMemory(m.config.MaxMemory),
                NanoCPUs: int64(m.config.MaxCPU * 1e9),
            },
            SecurityOpt: []string{"no-new-privileges"},
            CapDrop:     []string{"ALL"},
        },
        nil, nil, containerName,
    )

    if err != nil {
        return nil, err
    }

    return &Container{
        ID:        resp.ID,
        Name:      containerName,
        UserID:    userID,
        ChatID:    chatID,
        Status:    "created",
        CreatedAt: time.Now(),
    }, nil
}

// ensureImage ensures the image exists locally, pulls if not
func (m *Manager) ensureImage(ctx context.Context, imageName string) error {
    // Check if image exists locally
    _, _, err := m.dockerClient.ImageInspectWithRaw(ctx, imageName)
    if err == nil {
        return nil // Image exists
    }

    // Image not found, pull it
    reader, err := m.dockerClient.ImagePull(ctx, imageName, image.PullOptions{})
    if err != nil {
        return fmt.Errorf("failed to pull image %s: %w", imageName, err)
    }
    defer reader.Close()

    // Wait for pull to complete
    io.Copy(io.Discard, reader)
    return nil
}
```

### 6.4 Ensure Running

```go
func (m *Manager) ensureRunning(ctx context.Context, containerName string) error {
    c, ok := m.containers.Load(containerName)
    if !ok {
        return fmt.Errorf("container not found: %s", containerName)
    }
    cont := c.(*Container)

    if cont.Status == "running" {
        return nil
    }

    // Start the container
    if err := m.dockerClient.ContainerStart(ctx, cont.ID, container.StartOptions{}); err != nil {
        return err
    }

    m.mu.Lock()
    cont.Status = "running"
    cont.LastUsedAt = time.Now()
    m.mu.Unlock()

    return nil
}
```

### 6.5 Execute Command (Streaming)

```go
func (m *Manager) Stream(ctx context.Context, containerName string, cmd []string, opts *ExecOptions) (io.ReadCloser, error) {
    // Ensure container is running
    if err := m.ensureRunning(ctx, containerName); err != nil {
        return nil, err
    }

    // Get container
    c, _ := m.containers.Load(containerName)
    cont := c.(*Container)

    // Create exec instance
    execConfig := container.ExecOptions{
        Cmd:          cmd,
        WorkingDir:   opts.WorkDir,
        Env:          mapToSlice(opts.Env),
        AttachStdout: true,
        AttachStderr: true,
    }

    execResp, err := m.dockerClient.ContainerExecCreate(ctx, cont.ID, execConfig)
    if err != nil {
        return nil, err
    }

    // Attach to exec
    attachResp, err := m.dockerClient.ContainerExecAttach(ctx, execResp.ID, container.ExecAttachOptions{})
    if err != nil {
        return nil, err
    }

    return attachResp.Reader, nil
}

// Exec executes command and waits for completion
func (m *Manager) Exec(ctx context.Context, containerName string, cmd []string, opts *ExecOptions) (*ExecResult, error) {
    if opts == nil {
        opts = &ExecOptions{}
    }

    reader, err := m.Stream(ctx, containerName, cmd, opts)
    if err != nil {
        return nil, err
    }
    defer reader.Close()

    // Read all output
    output, err := io.ReadAll(reader)
    if err != nil {
        return nil, err
    }

    // TODO: Parse stdout/stderr from Docker multiplexed stream
    // TODO: Get exit code from ContainerExecInspect

    return &ExecResult{
        ExitCode: 0,
        Stdout:   string(output),
        Stderr:   "",
    }, nil
}
```

### 6.6 Filesystem Operations

```go
// WriteFile writes content to a file in container using docker cp
func (m *Manager) WriteFile(ctx context.Context, containerName, path string, content []byte) error {
    c, ok := m.containers.Load(containerName)
    if !ok {
        return fmt.Errorf("container not found: %s", containerName)
    }
    cont := c.(*Container)
    // Create a tar archive with the file
    var buf bytes.Buffer
    tw := tar.NewWriter(&buf)

    hdr := &tar.Header{
        Name: filepath.Base(path),
        Mode: 0644,
        Size: int64(len(content)),
    }
    tw.WriteHeader(hdr)
    tw.Write(content)
    tw.Close()

    // Copy to container
    return m.dockerClient.CopyToContainer(ctx, cont.ID, filepath.Dir(path), &buf, container.CopyToContainerOptions{})
}

// ReadFile reads content from a file in container
func (m *Manager) ReadFile(ctx context.Context, containerName, path string) ([]byte, error) {
    c, ok := m.containers.Load(containerName)
    if !ok {
        return nil, fmt.Errorf("container not found: %s", containerName)
    }
    cont := c.(*Container)

    reader, _, err := m.dockerClient.CopyFromContainer(ctx, cont.ID, path)
    if err != nil {
        return nil, err
    }
    defer reader.Close()

    // Extract from tar
    tr := tar.NewReader(reader)
    _, err = tr.Next()
    if err != nil {
        return nil, err
    }

    return io.ReadAll(tr)
}

// ListDir lists directory contents
func (m *Manager) ListDir(ctx context.Context, containerName, path string) ([]FileInfo, error) {
    result, err := m.Exec(ctx, containerName, []string{"ls", "-la", "--time-style=+%s", path}, nil)
    if err != nil {
        return nil, err
    }

    return parseLS(result.Stdout), nil
}

// Stat returns file info
func (m *Manager) Stat(ctx context.Context, containerName, path string) (*FileInfo, error) {
    result, err := m.Exec(ctx, containerName, []string{"stat", "--format=%n|%s|%f|%Y|%F", path}, nil)
    if err != nil {
        return nil, err
    }
    return parseStat(result.Stdout), nil
}

// MkDir creates directory in container
func (m *Manager) MkDir(ctx context.Context, containerName, path string) error {
    _, err := m.Exec(ctx, containerName, []string{"mkdir", "-p", path}, nil)
    return err
}

// RemoveFile removes file or directory in container
func (m *Manager) RemoveFile(ctx context.Context, containerName, path string) error {
    _, err := m.Exec(ctx, containerName, []string{"rm", "-rf", path}, nil)
    return err
}

// CopyToContainer copies from host to container
func (m *Manager) CopyToContainer(ctx context.Context, containerName, hostPath, containerPath string) error {
    c, ok := m.containers.Load(containerName)
    if !ok {
        return fmt.Errorf("container not found: %s", containerName)
    }
    cont := c.(*Container)

    // Create tar archive from host path
    archive, err := createTarFromPath(hostPath)
    if err != nil {
        return err
    }
    defer archive.Close()

    return m.dockerClient.CopyToContainer(ctx, cont.ID, containerPath, archive, container.CopyToContainerOptions{})
}

// CopyFromContainer copies from container to host
func (m *Manager) CopyFromContainer(ctx context.Context, containerName, containerPath, hostPath string) error {
    c, ok := m.containers.Load(containerName)
    if !ok {
        return fmt.Errorf("container not found: %s", containerName)
    }
    cont := c.(*Container)

    reader, _, err := m.dockerClient.CopyFromContainer(ctx, cont.ID, containerPath)
    if err != nil {
        return err
    }
    defer reader.Close()

    return extractTarToPath(reader, hostPath)
}
```

### 6.7 Cleanup Strategy

```go
func (m *Manager) startCleanupLoop(ctx context.Context) {
    ticker := time.NewTicker(5 * time.Minute)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            m.Cleanup(ctx)
        }
    }
}

func (m *Manager) Cleanup(ctx context.Context) error {
    now := time.Now()

    m.containers.Range(func(key, value interface{}) bool {
        containerName := key.(string)
        c := value.(*Container)

        // Stop idle containers
        if c.Status == "running" && now.Sub(c.LastUsedAt) > m.config.IdleTimeout {
            m.Stop(ctx, containerName)
        }

        return true
    })

    return nil
}

func (m *Manager) Start(ctx context.Context, containerName string) error {
    return m.ensureRunning(ctx, containerName)
}

func (m *Manager) Stop(ctx context.Context, containerName string) error {
    c, ok := m.containers.Load(containerName)
    if !ok {
        return nil
    }
    cont := c.(*Container)

    if err := m.dockerClient.ContainerStop(ctx, cont.ID, container.StopOptions{}); err != nil {
        return err
    }

    // Update status, decrement running count
    m.mu.Lock()
    if cont.Status == "running" {
        cont.Status = "stopped"
        m.running--
    }
    m.mu.Unlock()

    return nil
}

func (m *Manager) Remove(ctx context.Context, containerName string) error {
    // Stop first if running
    m.Stop(ctx, containerName)

    c, ok := m.containers.Load(containerName)
    if !ok {
        return nil
    }
    cont := c.(*Container)

    if err := m.dockerClient.ContainerRemove(ctx, cont.ID, container.RemoveOptions{}); err != nil {
        return err
    }

    // Remove from map
    m.containers.Delete(containerName)

    return nil
}

func (m *Manager) List(ctx context.Context, userID string) ([]*Container, error) {
    var result []*Container
    prefix := fmt.Sprintf("yao-sandbox-%s-", userID)

    m.containers.Range(func(key, value interface{}) bool {
        containerName := key.(string)
        if strings.HasPrefix(containerName, prefix) {
            result = append(result, value.(*Container))
        }
        return true
    })

    return result, nil
}
```

### 6.8 Helper Functions

```go
// mapToSlice converts map to []string for env vars
func mapToSlice(m map[string]string) []string {
    if m == nil {
        return nil
    }
    result := make([]string, 0, len(m))
    for k, v := range m {
        result = append(result, k+"="+v)
    }
    return result
}

// parseMemory converts string like "2g" to bytes
func parseMemory(s string) int64 {
    // Implementation: parse "2g" → 2*1024*1024*1024
    // Use Docker's units package or manual parsing
    return 0 // placeholder
}

// parseLS parses ls -la output to []FileInfo
func parseLS(output string) []FileInfo {
    // Implementation: parse ls output lines
    return nil // placeholder
}

// parseStat parses stat output to *FileInfo
func parseStat(output string) *FileInfo {
    // Implementation: parse stat --format output
    return nil // placeholder
}

// createTarFromPath creates a tar archive from a host path
func createTarFromPath(hostPath string) (io.ReadCloser, error) {
    // Implementation: walk directory, create tar entries
    return nil, nil // placeholder
}

// extractTarToPath extracts a tar archive to a host path
func extractTarToPath(reader io.Reader, hostPath string) error {
    // Implementation: read tar entries, write to disk
    return nil // placeholder
}
```

---

## 7. yao-bridge

Lightweight binary inside container that bridges stdio to Unix socket.

```go
// cmd/yao-bridge/main.go
package main

import (
    "io"
    "net"
    "os"
)

func main() {
    if len(os.Args) < 2 {
        os.Exit(1)
    }

    sockPath := os.Args[1]

    // Connect to Unix socket
    conn, err := net.Dial("unix", sockPath)
    if err != nil {
        os.Exit(1)
    }
    defer conn.Close()

    // stdin → socket
    go func() {
        io.Copy(conn, os.Stdin)
        conn.(*net.UnixConn).CloseWrite()
    }()

    // socket → stdout
    io.Copy(os.Stdout, conn)
}
```

Build as static binary and include in Docker image.

---

## 8. Docker Image

### 8.1 Image Naming Convention

```
yao/sandbox-{tool}:{variant}

Examples:
  yao/sandbox-claude:latest     # Claude CLI + Node.js + Python (default)
  yao/sandbox-claude:full       # + Go
  yao/sandbox-cursor:latest     # Cursor CLI + Node.js + Python (future)
```

### 8.2 Source Directory Structure

```
sandbox/
├── docker/
│   ├── base/
│   │   └── Dockerfile.base         # Common base image
│   ├── claude/
│   │   ├── Dockerfile              # Default: Claude + Node + Python
│   │   └── Dockerfile.full         # + Go
│   ├── cursor/                     # Future
│   │   └── Dockerfile
│   ├── build.sh
│   └── scripts/
│       └── entrypoint.sh
├── bridge/
│   └── main.go                     # yao-bridge source
├── ipc/
│   ├── manager.go
│   └── session.go
├── manager.go
├── config.go
└── types.go
```

### 8.3 Base Image

```dockerfile
# sandbox/docker/base/Dockerfile.base
FROM ubuntu:22.04

# Base tools
RUN apt-get update && apt-get install -y \
    curl \
    git \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# yao-bridge (common to all tools)
COPY yao-bridge /usr/local/bin/yao-bridge
RUN chmod +x /usr/local/bin/yao-bridge

# Working directory
WORKDIR /workspace

# Non-root user
RUN useradd -m -s /bin/bash sandbox
USER sandbox

CMD ["sleep", "infinity"]
```

### 8.4 Claude Tool Images

```dockerfile
# sandbox/docker/claude/Dockerfile
# Default image: Claude CLI + Node.js + Python
FROM yao/sandbox-base:latest

USER root

# Claude CLI
RUN curl -fsSL https://claude.ai/install.sh | sh

# Node.js 20
RUN curl -fsSL https://deb.nodesource.com/setup_20.x | bash - \
    && apt-get install -y nodejs \
    && rm -rf /var/lib/apt/lists/*

# Python 3.11
RUN apt-get update && apt-get install -y \
    python3.11 \
    python3-pip \
    && rm -rf /var/lib/apt/lists/*

USER sandbox
```

```dockerfile
# sandbox/docker/claude/Dockerfile.full
# Full image: + Go
FROM yao/sandbox-claude:latest

USER root

# Go 1.23
RUN curl -fsSL https://go.dev/dl/go1.23.linux-amd64.tar.gz | tar -C /usr/local -xzf - \
    && ln -s /usr/local/go/bin/go /usr/local/bin/go

USER sandbox
```

### 8.5 Build Script

```bash
#!/bin/bash
# sandbox/docker/build.sh

set -e

TOOL=${1:-claude}

# Build yao-bridge
cd ../bridge
CGO_ENABLED=0 go build -o ../docker/yao-bridge .
cd ../docker

# Build base image
docker build -t yao/sandbox-base:latest -f base/Dockerfile.base .

# Build tool-specific images
case $TOOL in
  claude)
    docker build -t yao/sandbox-claude:latest -f claude/Dockerfile .
    docker build -t yao/sandbox-claude:full -f claude/Dockerfile.full .
    ;;
  cursor)
    docker build -t yao/sandbox-cursor:latest -f cursor/Dockerfile .
    ;;
  all)
    $0 claude
    $0 cursor
    ;;
esac

echo "Images built for tool: $TOOL"
```

### 8.6 Image Variants

| Image                       | Tool   | Size   | Pre-installed                       |
| --------------------------- | ------ | ------ | ----------------------------------- |
| `yao/sandbox-base:latest`   | -      | ~200MB | git, curl, yao-bridge               |
| `yao/sandbox-claude:latest` | Claude | ~700MB | Claude CLI, Node.js 20, Python 3.11 |
| `yao/sandbox-claude:full`   | Claude | ~1.3GB | + Go 1.23                           |
| `yao/sandbox-cursor:latest` | Cursor | ~700MB | Cursor CLI, Node.js 20, Python 3.11 |

Default: `yao/sandbox-claude:latest` (includes Node + Python)

---

## 9. Configuration

### 9.1 Environment Variables

| Env Variable               | Default                             | Description               |
| -------------------------- | ----------------------------------- | ------------------------- |
| `YAO_SANDBOX_IMAGE`        | `yao/sandbox-claude:latest`         | Default Docker image      |
| `YAO_SANDBOX_WORKSPACE`    | `{YAO_DATA_ROOT}/sandbox/workspace` | Workspace root directory  |
| `YAO_SANDBOX_IPC`          | `{YAO_DATA_ROOT}/sandbox/ipc`       | IPC socket directory      |
| `YAO_SANDBOX_MAX`          | `100`                               | Max concurrent containers |
| `YAO_SANDBOX_IDLE_TIMEOUT` | `30m`                               | Idle timeout              |
| `YAO_SANDBOX_MEMORY`       | `2g`                                | Default memory limit      |
| `YAO_SANDBOX_CPU`          | `1.0`                               | Default CPU limit         |

### 9.2 Go Config Struct

```go
// sandbox/config.go
type Config struct {
    Image         string        `json:"image,omitempty" env:"YAO_SANDBOX_IMAGE" envDefault:"yao/sandbox-claude:latest"`
    WorkspaceRoot string        `json:"workspace_root,omitempty" env:"YAO_SANDBOX_WORKSPACE"`
    IPCDir        string        `json:"ipc_dir,omitempty" env:"YAO_SANDBOX_IPC"`
    MaxContainers int           `json:"max_containers,omitempty" env:"YAO_SANDBOX_MAX" envDefault:"100"`
    IdleTimeout   time.Duration `json:"idle_timeout,omitempty" env:"YAO_SANDBOX_IDLE_TIMEOUT" envDefault:"30m"`
    MaxMemory     string        `json:"max_memory,omitempty" env:"YAO_SANDBOX_MEMORY" envDefault:"2g"`
    MaxCPU        float64       `json:"max_cpu,omitempty" env:"YAO_SANDBOX_CPU" envDefault:"1.0"`
}

// Init sets defaults based on Yao config
func (c *Config) Init(dataRoot string) {
    if c.WorkspaceRoot == "" {
        c.WorkspaceRoot = filepath.Join(dataRoot, "sandbox", "workspace")
    }
    if c.IPCDir == "" {
        c.IPCDir = filepath.Join(dataRoot, "sandbox", "ipc")
    }
}
```

### 9.3 app.yao (optional override)

```yaml
sandbox:
  image: "yao/sandbox-claude:full"
  max_memory: "4g"
```

### 9.4 Assistant-level Configuration (package.yao)

```yaml
name: "My Coder"
type: claude

sandbox:
  image: "yao/sandbox-claude:full" # Override image
  max_memory: "4g" # Override memory limit
```

### 9.5 Image Resolution

```
1. If package.yao sandbox.image is set → use it
2. Else if type is set → use yao/sandbox-{type}:latest
3. Else → use YAO_SANDBOX_IMAGE (or app.yao sandbox.image)
```

---

## 10. Data Persistence

### Directory Structure

```
{YAO_DATA_ROOT}/sandbox/
├── workspace/
│   └── {userID}/
│       ├── {chatID-1}/              # Mounted as /workspace in container
│       │   ├── .mcp.json            # MCP configuration
│       │   ├── .claude/             # Claude configuration
│       │   │   └── skills/          # Skills symlink
│       │   ├── project/             # User project code
│       │   └── node_modules/        # Installed dependencies
│       │
│       └── {chatID-2}/
│           └── ...
│
└── ipc/
    ├── {chatID-1}.sock              # IPC socket
    └── {chatID-2}.sock
```

### What Persists

| Item               | Location              | Persists                       |
| ------------------ | --------------------- | ------------------------------ |
| User code          | `/workspace/`         | ✅ Yes (host mount)            |
| Installed packages | Container filesystem  | ✅ Yes (container persists)    |
| Claude config      | `/workspace/.claude/` | ✅ Yes                         |
| IPC socket         | `/tmp/yao.sock`       | ❌ No (recreated each session) |

---

## 11. Stability Guarantees

| Concern                 | Solution                                                          |
| ----------------------- | ----------------------------------------------------------------- |
| **Container isolation** | One container per user+chat                                       |
| **IPC isolation**       | One socket per session                                            |
| **Resource limits**     | Docker memory/CPU limits                                          |
| **Idle cleanup**        | Auto-stop after timeout (preserve data)                           |
| **Data persistence**    | Workspace directory mount, container preserves installed packages |
| **Connection handling** | Goroutine detects EOF, auto-cleanup                               |
| **Concurrency safety**  | sync.Map + dedicated goroutines                                   |

---

## 12. Claude Executor Integration

### Execution Flow

```go
func (e *ClaudeExecutor) Stream(ctx *context.Context, messages []context.Message, opts ...*context.Options) (*context.Response, error) {

    // 1. Get or create sandbox container
    container, err := e.SandboxManager.GetOrCreate(ctx, ctx.User.ID, ctx.ChatID)
    if err != nil {
        return nil, fmt.Errorf("failed to get sandbox: %w", err)
    }

    // 2. Create IPC session
    mcpTools := e.getMCPTools()
    ipcSession, err := e.IPCManager.Create(ctx, ctx.ChatID, &AgentContext{
        UserID: ctx.User.ID,
        ChatID: ctx.ChatID,
        Locale: ctx.Locale,
    }, mcpTools)
    if err != nil {
        return nil, fmt.Errorf("failed to create IPC session: %w", err)
    }
    defer e.IPCManager.Close(ctx.ChatID)

    // 3. Generate .mcp.json
    if err := e.writeMCPConfig(ctx, container); err != nil {
        return nil, err
    }

    // 4. Setup skills
    if err := e.setupSkills(container); err != nil {
        return nil, err
    }

    // 5. Build Claude CLI arguments
    args := e.buildArgs(ctx, messages)

    // 6. Execute Claude CLI in container
    stdout, err := e.SandboxManager.Stream(ctx, container.Name,
        append([]string{"claude"}, args...),
        &ExecOptions{
            WorkDir: "/workspace",
            Env:     e.buildEnvMap(ctx),
            Timeout: e.getTimeout(),
        },
    )
    if err != nil {
        return nil, err
    }
    defer stdout.Close()

    // 7. Parse stream-json output
    return e.parseClaudeOutput(ctx, stdout)
}
```

### MCP Configuration Generation

```go
func (e *ClaudeExecutor) writeMCPConfig(ctx context.Context, container *Container) error {
    config := map[string]interface{}{
        "mcpServers": map[string]interface{}{
            "yao": map[string]interface{}{
                "command": "yao-bridge",
                "args":    []string{"/tmp/yao.sock"},
            },
        },
    }

    // Add other MCP servers (external stdio/sse)
    for _, server := range e.Assistant.MCP.Servers {
        if server.Transport != "process" {
            config["mcpServers"].(map[string]interface{})[server.Name] = server.ToClaudeConfig()
        }
    }

    data, _ := json.MarshalIndent(config, "", "  ")
    return e.SandboxManager.WriteFile(ctx, container.Name, "/workspace/.mcp.json", data)
}
```

---

## 13. Security Considerations

| Layer          | Measures                                               |
| -------------- | ------------------------------------------------------ |
| **Filesystem** | Only workspace mounted, host filesystem not accessible |
| **Network**    | Can be restricted with `--network none` if needed      |
| **Privileges** | `--cap-drop ALL`, `no-new-privileges`                  |
| **Resources**  | Memory and CPU limits                                  |
| **User**       | Non-root user inside container                         |
| **IPC**        | Per-session socket, authorized tools only              |

---

## 14. Summary

| Aspect                     | Description                               |
| -------------------------- | ----------------------------------------- |
| **Core approach**          | Persistent Docker containers              |
| **Communication**          | Unix Socket + MCP JSON-RPC                |
| **Container granularity**  | One container per user+chat               |
| **Data persistence**       | Workspace mount + container filesystem    |
| **Dependency persistence** | npm/pip packages persist in container     |
| **Security isolation**     | Full isolation between users and sessions |
| **Resource control**       | Memory, CPU, idle timeout                 |
| **Estimated code**         | ~1200 lines                               |
