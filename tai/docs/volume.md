# Package `volume`

File IO and directory synchronization. Provides a `Volume` interface with two implementations:

| Implementation | Constructor | Backend | Mode |
|----------------|-------------|---------|------|
| **Local** | `NewLocal(root)` | Direct filesystem | Local |
| **Remote** | `NewRemote(conn)` | gRPC to Tai :19100 | Remote |

## Interface

```go
type Volume interface {
    ReadFile(ctx context.Context, sessionID, path string) (data []byte, perm os.FileMode, err error)
    WriteFile(ctx context.Context, sessionID, path string, data []byte, perm os.FileMode) error
    Stat(ctx context.Context, sessionID, path string) (*FileInfo, error)
    ListDir(ctx context.Context, sessionID, path string) ([]FileInfo, error)
    Remove(ctx context.Context, sessionID, path string, recursive bool) error
    Rename(ctx context.Context, sessionID, oldPath, newPath string) error
    MkdirAll(ctx context.Context, sessionID, path string) error

    SyncPush(ctx context.Context, sessionID, localDir string, opts ...SyncOption) (*SyncResult, error)
    SyncPull(ctx context.Context, sessionID, localDir string, opts ...SyncOption) (*SyncResult, error)

    Close() error
}
```

All paths are **relative** to the session's workspace root. The `sessionID` identifies the workspace partition — in Local mode this maps to `<root>/<sessionID>/`, in Remote mode the Tai server manages the path.

## Constructors

### NewLocal

```go
func NewLocal(dataDir string) Volume
```

Creates a Volume backed by the local filesystem. Files are stored under `<dataDir>/<sessionID>/`.

### NewRemote

```go
func NewRemote(conn *grpc.ClientConn) Volume
```

Creates a Volume backed by Tai's gRPC Volume service. The connection should target Tai's gRPC port (default 19100). Uses lz4 compression for `SyncPush`/`SyncPull` bulk transfers.

## Types

### FileInfo

```go
type FileInfo struct {
    Path  string
    Size  int64
    Mtime time.Time
    Mode  fs.FileMode
    IsDir bool
}
```

### SyncResult

```go
type SyncResult struct {
    FilesSynced      int
    BytesTransferred int64
    Duration         time.Duration
}
```

## Sync Options

```go
volume.WithForceFull()              // skip snapshot cache, diff against actual disk
volume.WithExcludes("*.log", ".DS_Store") // glob patterns to exclude
```

## File Operations

| Method | Description |
|--------|-------------|
| `ReadFile` | Read file contents and permissions |
| `WriteFile` | Write file with specified permissions (creates parent dirs) |
| `Stat` | Get file/directory metadata |
| `ListDir` | List directory contents (one level) |
| `Remove` | Delete file or directory (`recursive=true` for tree) |
| `Rename` | Move/rename a file or directory |
| `MkdirAll` | Create directory tree |

## Sync Operations

| Method | Direction | Description |
|--------|-----------|-------------|
| `SyncPush` | local → remote | Upload a local directory to the session workspace |
| `SyncPull` | remote → local | Download the session workspace to a local directory |

Both sync methods use snapshot-based diffing to transfer only changed files. Use `WithForceFull()` to bypass the cache and force a full transfer.

Remote sync uses **lz4 compression** on the wire, streaming files via gRPC bidirectional streaming.

## Example

```go
vol := volume.NewLocal("/data/volumes")
defer vol.Close()

// Write a file
vol.WriteFile(ctx, "session-1", "main.py", []byte("print('hi')"), 0644)

// Read it back
data, perm, _ := vol.ReadFile(ctx, "session-1", "main.py")

// Sync a local directory to the session
result, _ := vol.SyncPush(ctx, "session-1", "/tmp/project",
    volume.WithExcludes("node_modules", ".git"),
)
fmt.Printf("synced %d files (%d bytes)\n", result.FilesSynced, result.BytesTransferred)
```
