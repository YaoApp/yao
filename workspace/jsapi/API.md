# Workspace JavaScript API

All methods are available on the global `workspace` object. No constructor needed.

## Quick Start

```javascript
// Create a workspace
const ws = workspace.Create({ name: "my-project", owner: "user-123", node: "default" })

// File I/O
ws.WriteFile("src/main.go", 'package main\n\nfunc main() {}\n')
const content = ws.ReadFile("src/main.go")

// Binary file (Base64)
const b64 = ws.ReadFileBase64("image.png")
ws.WriteFileBase64("copy.png", b64)

// Clean up
workspace.Delete(ws.id)
```

---

## Static Methods

### workspace.Create(options) → WorkspaceFS

Create a new workspace on a Tai node.

```javascript
const ws = workspace.Create({
  name:   "my-project",       // required — human-readable name
  owner:  "user-123",         // required — user ID
  node:   "default",          // required — target Tai node
  id:     "ws-custom-id",     // optional — auto-generated if empty
  labels: { team: "backend" } // optional — custom labels
})
```

### workspace.Get(id) → WorkspaceFS | null

Get an existing workspace by ID. Returns `null` if not found.

```javascript
const ws = workspace.Get("ws-abc123")
if (ws) {
  console.log(ws.id, ws.name, ws.node)
}
```

### workspace.List(filter?) → WorkspaceInfo[]

List all workspaces, optionally filtered.

```javascript
const all = workspace.List()
const mine = workspace.List({ owner: "user-123" })
const onNode = workspace.List({ node: "gpu-01" })
```

Each element:

```javascript
{
  id:         "ws-abc123",
  name:       "my-project",
  owner:      "user-123",
  node:       "default",
  labels:     { team: "backend" },
  created_at: "2026-03-07T10:00:00Z",
  updated_at: "2026-03-07T10:05:00Z"
}
```

### workspace.Delete(id) → void

Delete a workspace and its storage.

```javascript
workspace.Delete("ws-abc123")
```

---

## WorkspaceFS Object

Returned by `workspace.Create()`, `workspace.Get()`, `box.Workspace()`, and `host.Workspace()`.

### Properties (read-only)

| Property | Type | Description |
|----------|------|-------------|
| `ws.id` | string | Workspace ID |
| `ws.name` | string | Workspace name |
| `ws.node` | string | Tai node name |

---

### File Reading

#### ws.ReadFile(path) → string

Read file content as UTF-8 string.

```javascript
const content = ws.ReadFile("src/main.go")
```

Go: `workspace.M().ReadFile(id, name)` → `Volume.ReadFile` → `string(data)`

#### ws.ReadFileBase64(path) → string

Read file content as Base64-encoded string. Use for binary files (images, archives, etc.).

```javascript
const b64 = ws.ReadFileBase64("assets/logo.png")
```

Go: `M().ReadFile` → `base64.StdEncoding.EncodeToString(data)`

#### ws.ReadFileBuffer(path) → string

Read file content as Base64 string. Alias for `ReadFileBase64` (temporary — Uint8Array support pending v8go upgrade).

```javascript
const b64 = ws.ReadFileBuffer("data.bin")
```

Go: `M().ReadFile` → `base64.StdEncoding.EncodeToString(data)`

---

### File Writing

#### ws.WriteFile(path, data, perm?) → void

Write string data to a file. Creates parent directories if needed.

```javascript
ws.WriteFile("src/main.go", "package main\n...")
ws.WriteFile("config.yml", yamlContent, 0644)
```

Go: `workspace.M().WriteFile(id, name, []byte(data), perm)` → `Volume.WriteFile` — perm defaults to `0644`

#### ws.WriteFileBase64(path, b64, perm?) → void

Write Base64-encoded data to a file. Use for binary files.

```javascript
ws.WriteFileBase64("assets/logo.png", b64Data)
```

Go: `base64.Decode` → `M().WriteFile` → `Volume.WriteFile`

#### ws.WriteFileBuffer(path, b64, perm?) → void

Write Base64-encoded data to a file. Alias for `WriteFileBase64` (temporary — Uint8Array support pending v8go upgrade).

```javascript
ws.WriteFileBuffer("data.bin", b64Data)
```

Go: `base64.Decode` → `M().WriteFile` → `Volume.WriteFile`

---

### Directory Operations

#### ws.ReadDir(path?, recursive?) → DirEntry[]

List directory contents. Defaults to root (`"."`), non-recursive.

```javascript
// One level (default)
const entries = ws.ReadDir("src/")
// → [{ name: "main.go", ... }, { name: "utils", is_dir: true, ... }]

// Recursive — name becomes relative path
const all = ws.ReadDir("src/", true)
// → [{ name: "main.go", ... }, { name: "utils/helper.go", ... }]

entries.forEach(function(e) {
  console.log(e.name, e.is_dir ? "(dir)" : e.size + " bytes")
})
```

Return type:

```javascript
{ name: "main.go", is_dir: false, size: 1234 }
// recursive mode: name is relative path, e.g. "utils/helper.go"
```

Go: non-recursive → `workspace.M().ListDir` → `Volume.ListDir`; recursive → `M().FS()` → `fs.WalkDir`

#### ws.MkdirAll(path) → void

Create a directory tree recursively. Permission is always `0755`.

```javascript
ws.MkdirAll("src/utils/helpers")
```

Go: `workspace.M().MkdirAll(id, name)` → `Volume.MkdirAll` (0755)

---

### File Operations

#### ws.Remove(path) → void

Remove a single file or empty directory.

```javascript
ws.Remove("tmp.txt")
```

Go: `FS.Remove(name)`

#### ws.RemoveAll(path) → void

Remove a file or directory recursively.

```javascript
ws.RemoveAll("build/")
```

Go: `FS.RemoveAll(name)`

#### ws.Rename(from, to) → void

Rename or move a file/directory within the workspace.

```javascript
ws.Rename("old.txt", "new.txt")
ws.Rename("src/foo.go", "src/bar.go")
```

Go: `workspace.M().Rename(id, oldname, newname)` → `Volume.Rename`

#### ws.Copy(src, dst, options?) → SyncResult | void

Unified copy method. Supports workspace-internal copy and host ↔ workspace copy via `local://` URI prefix.

**Path resolution:**

- No prefix → workspace-internal path (relative to workspace root)
- `local://` prefix → relative to App Root (`config.Conf.AppSource`), e.g. `local:///data/templates` → `{AppRoot}/data/templates`
- `tmp://` prefix → relative to `os.TempDir()`, e.g. `tmp:///workspace-staging` → `/tmp/workspace-staging`

**Security:** `..` traversal is rejected for both `local://` and `tmp://`. `local://` paths escaping App Root are rejected.

**Examples:**

```javascript
// workspace → workspace
ws.Copy("src/main.go", "src/main_backup.go")
ws.Copy("templates/", "projects/new/")

// host → workspace
ws.Copy("local:///app/templates/nextjs", "prompts/")
ws.Copy("local:///data/assistants/bot-a/config", "config/")

// workspace → host
ws.Copy("build/dist/", "local:///app/output/dist")

// host → host (no extra fs needed)
ws.Copy("local:///templates/nextjs", "local:///backup/nextjs-backup")

// tmp dir → workspace
ws.Copy("tmp:///workspace-staging/data", "imported/")

// workspace → tmp dir
ws.Copy("build/dist/", "tmp:///export-staging")

// with options (excludes, force)
ws.Copy("local:///app/templates/nextjs", "project/", {
  excludes: ["node_modules", ".git", "*.log"],
  force:    true
})
```

Options:

```javascript
{
  excludes: ["node_modules"],     // optional — glob patterns to exclude
  force:    false                 // optional — skip incremental diff, sync everything
}
```

Return value:

- **Workspace internal**: void
- **Host ↔ workspace** (`local://`/`tmp://` on one side): `SyncResult`
- **Host → host** (both `local://`/`tmp://`): void

```javascript
{
  files_synced:       42,         // number of files transferred
  bytes_transferred:  1048576,    // total bytes
  duration_ms:        1234        // time taken in ms
}
```

**Dispatch rules (Go layer):**

| src | dst | Implementation |
|-----|-----|----------------|
| workspace | workspace | `FS.ReadFile` + `FS.WriteFile` (recursive for dirs) |
| host URI | workspace | `Volume.SyncPush(hostPath, wsPath, opts)` |
| workspace | host URI | `Volume.SyncPull(wsPath, hostPath, opts)` |
| host URI | host URI | `os` package recursive copy |

Where "host URI" = `local://` (relative to App Root) or `tmp://` (relative to `os.TempDir()`).

---

### Archive & Compression

All archive methods operate on paths within the workspace. Pack operations support an `excludes` option.

#### ws.Zip(src, dst, options?) → ArchiveResult

Create a ZIP archive from `src` directory to `dst` file.

```javascript
const result = ws.Zip("src/", "dist.zip")
const filtered = ws.Zip("src/", "dist.zip", { excludes: ["*.log", "node_modules"] })
```

#### ws.Unzip(src, dst) → ArchiveResult

Extract a ZIP archive from `src` file to `dst` directory.

```javascript
const result = ws.Unzip("dist.zip", "extracted/")
```

#### ws.Gzip(src, dst) → ArchiveResult

Compress a single file with gzip.

```javascript
ws.Gzip("data.json", "data.json.gz")
```

#### ws.Gunzip(src, dst) → ArchiveResult

Decompress a gzip file.

```javascript
ws.Gunzip("data.json.gz", "data.json")
```

#### ws.Tar(src, dst, options?) → ArchiveResult

Create a tar archive from `src` directory.

```javascript
ws.Tar("src/", "archive.tar")
ws.Tar("src/", "archive.tar", { excludes: [".git"] })
```

#### ws.Untar(src, dst) → ArchiveResult

Extract a tar archive.

```javascript
ws.Untar("archive.tar", "extracted/")
```

#### ws.Tgz(src, dst, options?) → ArchiveResult

Create a gzip-compressed tar archive (.tar.gz / .tgz).

```javascript
ws.Tgz("src/", "archive.tgz")
```

#### ws.Untgz(src, dst) → ArchiveResult

Extract a gzip-compressed tar archive.

```javascript
ws.Untgz("archive.tgz", "extracted/")
```

**ArchiveResult:**

```javascript
{
  size_bytes:  102400,  // output file size (pack) or total extracted size (unpack)
  files_count: 15       // number of files processed
}
```

Go: delegates to `Volume.Zip`, `Volume.Unzip`, `Volume.Gzip`, `Volume.Gunzip`, `Volume.Tar`, `Volume.Untar`, `Volume.Tgz`, `Volume.Untgz`.

---

### File Information

#### ws.Stat(path) → FileInfo

Get file or directory metadata.

```javascript
const info = ws.Stat("src/main.go")
console.log(info.name, info.size, info.is_dir, info.mod_time, info.mode)
```

Return type:

```javascript
{
  name:     "main.go",
  size:     1234,
  is_dir:   false,
  mod_time: "2026-03-07T10:00:00Z",
  mode:     0644
}
```

Go: `FS.Stat(name) → fs.FileInfo`

#### ws.Exists(path) → boolean

Check if a file or directory exists.

```javascript
if (ws.Exists("config.yml")) {
  // ...
}
```

Go: `FS.Stat(name)` — returns `true` if err == nil

#### ws.IsDir(path) → boolean

Check if a path is a directory. Returns `false` if not found.

```javascript
if (ws.IsDir("src/")) {
  // ...
}
```

Go: `FS.Stat(name) → info.IsDir()`

#### ws.IsFile(path) → boolean

Check if a path is a regular file. Returns `false` if not found.

```javascript
if (ws.IsFile("main.go")) {
  // ...
}
```

Go: `FS.Stat(name) → !info.IsDir()`

---

## Go Interface Reference

WorkspaceFS methods map to `taiworkspace.FS` and `volume.Volume`. Some methods (Stat, Remove, RemoveAll) use `workspace.M().FS()` directly; others (ReadFile, WriteFile, Rename, MkdirAll) go through `workspace.M()` → `Volume`:

FS interface (defined in `tai/workspace/workspace.go`):

```go
type FS interface {
    fs.FS         // Open(name) (fs.File, error)
    fs.StatFS     // Stat(name) (fs.FileInfo, error)
    fs.ReadFileFS // ReadFile(name) ([]byte, error)
    fs.ReadDirFS  // ReadDir(name) ([]fs.DirEntry, error)
    io.Closer

    WriteFile(name string, data []byte, perm os.FileMode) error
    Remove(name string) error
    RemoveAll(name string) error
    Rename(oldname, newname string) error
    MkdirAll(name string, perm os.FileMode) error
}
```

Archive methods delegate to `volume.Volume` (defined in `tai/volume/volume.go`):

```go
type Volume interface {
    // ... FS methods ...
    Zip(ctx, sessionID, src, dst string, excludes []string) (*ArchiveResult, error)
    Unzip(ctx, sessionID, src, dst string) (*ArchiveResult, error)
    Gzip(ctx, sessionID, src, dst string) (*ArchiveResult, error)
    Gunzip(ctx, sessionID, src, dst string) (*ArchiveResult, error)
    Tar(ctx, sessionID, src, dst string, excludes []string) (*ArchiveResult, error)
    Untar(ctx, sessionID, src, dst string) (*ArchiveResult, error)
    Tgz(ctx, sessionID, src, dst string, excludes []string) (*ArchiveResult, error)
    Untgz(ctx, sessionID, src, dst string) (*ArchiveResult, error)
}
```

`Exists`, `IsDir`, `IsFile` are thin JSAPI wrappers over `FS.Stat`.

`ReadDir` adds a `recursive` parameter in Go — non-recursive calls `Volume.ListDir`, recursive uses `M().FS()` + `fs.WalkDir`.

`Copy` is implemented in Go with host URI dispatch: workspace paths use `FS.ReadFile`/`FS.WriteFile`; `local://` (App Root) and `tmp://` (`os.TempDir()`) paths use `Volume.SyncPush`/`SyncPull` with `WithRemotePath`; host-to-host uses `os` package.

Base64/Buffer variants (`ReadFileBase64`, `ReadFileBuffer`, `WriteFileBase64`, `WriteFileBuffer`) are Go-side encoding wrappers around `M().ReadFile` / `M().WriteFile` (→ `Volume`). Buffer variants currently use Base64 encoding (Uint8Array support pending v8go upgrade).

---

## Method Summary

### Standard FS — core file operations

| Method | Returns | Go mapping |
|--------|---------|------------|
| `ws.ReadFile(path)` | string | `M().ReadFile` → `Volume.ReadFile` |
| `ws.WriteFile(path, data, perm?)` | void | `M().WriteFile` → `Volume.WriteFile` |
| `ws.Stat(path)` | FileInfo | `M().FS()` → `FS.Stat` |
| `ws.MkdirAll(path)` | void | `M().MkdirAll` → `Volume.MkdirAll` (0755) |
| `ws.Remove(path)` | void | `M().FS()` → `FS.Remove` |
| `ws.RemoveAll(path)` | void | `M().FS()` → `FS.RemoveAll` |
| `ws.Rename(from, to)` | void | `M().Rename` → `Volume.Rename` |

### Go wrapper — implemented in Go, called directly from JSAPI

| Method | Returns | Go implementation |
|--------|---------|-------------------|
| `ws.ReadDir(path?, recursive?)` | DirEntry[] | non-recursive: `Volume.ListDir`; recursive: `FS` + `fs.WalkDir` |
| `ws.ReadFileBase64(path)` | string | `M().ReadFile` → `base64.Encode` |
| `ws.ReadFileBuffer(path)` | string | `M().ReadFile` → `base64.Encode` (temp, Uint8Array pending) |
| `ws.WriteFileBase64(path, b64, perm?)` | void | `base64.Decode` → `M().WriteFile` |
| `ws.WriteFileBuffer(path, b64, perm?)` | void | `base64.Decode` → `M().WriteFile` (temp, Uint8Array pending) |
| `ws.Copy(src, dst, opts?)` | void / SyncResult | dispatch by host URI prefix (see below) |

`ws.Copy` dispatch (all handled in Go):

| src | dst | Returns | Go implementation |
|-----|-----|---------|-------------------|
| workspace | workspace | void | `FS.ReadFile` + `FS.WriteFile` (recursive for dirs) |
| host URI | workspace | SyncResult | `Volume.SyncPush` with `WithRemotePath` |
| workspace | host URI | SyncResult | `Volume.SyncPull` with `WithRemotePath` |
| host URI | host URI | void | `os` package recursive copy |

Where "host URI" = `local://` (App Root) or `tmp://` (`os.TempDir()`).

### Archive — delegates to `Volume` interface

| Method | Returns | Go implementation |
|--------|---------|-------------------|
| `ws.Zip(src, dst, opts?)` | ArchiveResult | `Volume.Zip` |
| `ws.Unzip(src, dst)` | ArchiveResult | `Volume.Unzip` |
| `ws.Gzip(src, dst)` | ArchiveResult | `Volume.Gzip` |
| `ws.Gunzip(src, dst)` | ArchiveResult | `Volume.Gunzip` |
| `ws.Tar(src, dst, opts?)` | ArchiveResult | `Volume.Tar` |
| `ws.Untar(src, dst)` | ArchiveResult | `Volume.Untar` |
| `ws.Tgz(src, dst, opts?)` | ArchiveResult | `Volume.Tgz` |
| `ws.Untgz(src, dst)` | ArchiveResult | `Volume.Untgz` |

### JSAPI composition — thin JS wrappers over standard FS

| Method | Returns | Composed from |
|--------|---------|---------------|
| `ws.Exists(path)` | boolean | `Stat` → err == nil |
| `ws.IsDir(path)` | boolean | `Stat` → `info.IsDir()` |
| `ws.IsFile(path)` | boolean | `Stat` → `!info.IsDir()` |

**Total: 24 methods + 3 read-only properties** (7 standard FS + 6 Go wrapper + 8 archive + 3 JSAPI composition)
