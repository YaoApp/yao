# Package `workspace`

Provides an `fs.FS`-compatible filesystem abstraction over `volume.Volume`. This allows session workspaces to be used with any Go standard library function that accepts `fs.FS`, such as `fs.WalkDir`, `template.ParseFS`, or `http.FS`.

## Interface

```go
type FS interface {
    fs.FS         // Open(name) (fs.File, error)
    fs.StatFS     // Stat(name) (fs.FileInfo, error)
    fs.ReadFileFS // ReadFile(name) ([]byte, error)
    fs.ReadDirFS  // ReadDir(name) ([]fs.DirEntry, error)
    io.Closer     // Close() error

    WriteFile(name string, data []byte, perm os.FileMode) error
    Remove(name string) error
    RemoveAll(name string) error
    Rename(oldname, newname string) error
    MkdirAll(name string, perm os.FileMode) error
}
```

## Constructor

```go
func New(vol volume.Volume, sessionID string) FS
```

Creates an FS backed by the given Volume for the specified session. The returned FS works transparently whether `vol` is Local or Remote.

Typically accessed through the top-level client:

```go
c, _ := tai.New("tai://host")
ws := c.Workspace("session-123")
```

## Read Operations (fs.FS compatible)

All read operations comply with the `fs.FS` contract. Paths must be valid according to `fs.ValidPath` — forward slashes, no leading slash, no `..` segments.

| Method | Standard Interface | Description |
|--------|--------------------|-------------|
| `Open(name)` | `fs.FS` | Opens a file or directory |
| `Stat(name)` | `fs.StatFS` | Returns file metadata |
| `ReadFile(name)` | `fs.ReadFileFS` | Reads entire file contents |
| `ReadDir(name)` | `fs.ReadDirFS` | Lists directory entries |

`Open` returns an in-memory `fs.File` for regular files (entire content loaded on open) and a directory handle for directories.

## Write Operations

| Method | Description |
|--------|-------------|
| `WriteFile(name, data, perm)` | Write file contents with permissions |
| `Remove(name)` | Delete a single file or empty directory |
| `RemoveAll(name)` | Delete a file or directory tree recursively |
| `Rename(old, new)` | Move/rename a file or directory |
| `MkdirAll(name, perm)` | Create directory tree (perm currently unused) |

## Example

```go
c, _ := tai.New("tai://192.168.1.100")
defer c.Close()

ws := c.Workspace("project-abc")

// Write files
ws.WriteFile("src/main.go", []byte("package main"), 0644)
ws.MkdirAll("src/utils", 0755)

// Read with standard fs.FS
data, _ := fs.ReadFile(ws, "src/main.go")

// Walk the tree
fs.WalkDir(ws, ".", func(path string, d fs.DirEntry, err error) error {
    fmt.Println(path)
    return nil
})

// Use with Go templates
tmpl, _ := template.ParseFS(ws, "templates/*.html")

// Clean up
ws.RemoveAll("src")
```

## Implementation Notes

- `Open` on a regular file reads the entire content into memory. For large files, prefer `ReadFile` or Volume's `ReadFile` directly.
- `Close()` is a no-op — the underlying Volume's lifecycle is managed by the `tai.Client`.
- Path validation follows `fs.ValidPath` rules. Invalid paths return `fs.ErrInvalid`.
