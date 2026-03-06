package jsapi

import (
	"rogchap.com/v8go"
)

// NewFSObject creates a JS WorkspaceFS object backed by a workspace ID string.
// All methods delegate to workspace.M() → FS — no Go object passed to V8.
//
// # Properties (read-only)
//
//	ws.id   → string  // workspace ID  ← workspaceID arg
//	ws.name → string  // workspace name ← Workspace.Name
//	ws.node → string  // tai node name  ← Workspace.Node
//
// # Methods — Go mapping
//
// Each method internally does: fs, _ := workspace.M().FS(ctx, workspaceID)
// then calls the corresponding method on taiworkspace.FS.
//
// ws.ReadFile(path) → string
//
//	Go: FS.ReadFile(name string) ([]byte, error)
//	    — also available via Manager.ReadFile(ctx, id, path)
//	JS args:  path string
//	JS returns: string (UTF-8 content of the file)
//
// ws.WriteFile(path, data, perm?) → void
//
//	Go: FS.WriteFile(name string, data []byte, perm os.FileMode) error
//	    — also available via Manager.WriteFile(ctx, id, path, data, perm)
//	JS args:  path string, data string|Uint8Array, perm? number (default 0644)
//
// ws.ReadDir(path?) → DirEntry[]
//
//	Go: FS.ReadDir(name string) ([]fs.DirEntry, error)
//	    — also available via Manager.ListDir(ctx, id, path)
//	JS args:  path string (default ".")
//	JS returns: [{
//	  name:   string,   ← DirEntry.Name()
//	  is_dir: boolean,  ← DirEntry.IsDir()
//	  size:   number    ← DirEntry.Info().Size()
//	}]
//
// ws.Stat(path) → FileInfo
//
//	Go: FS.Stat(name string) (fs.FileInfo, error)
//	JS args:  path string
//	JS returns: {
//	  name:     string,   ← FileInfo.Name()
//	  size:     number,   ← FileInfo.Size()
//	  is_dir:   boolean,  ← FileInfo.IsDir()
//	  mod_time: string    ← FileInfo.ModTime() (ISO 8601)
//	}
//
// ws.MkdirAll(path, perm?) → void
//
//	Go: FS.MkdirAll(name string, perm os.FileMode) error
//	JS args:  path string, perm? number (default 0755)
//
// ws.Remove(path) → void
//
//	Go: FS.Remove(name string) error
//	    — also available via Manager.Remove(ctx, id, path)
//	JS args:  path string (single file or empty directory)
//
// ws.RemoveAll(path) → void
//
//	Go: FS.RemoveAll(name string) error
//	JS args:  path string (recursive removal)
//
// ws.Rename(from, to) → void
//
//	Go: FS.Rename(oldname, newname string) error
//	JS args:  from string, to string
//
// # Base64 variants (PLANNED — not yet implemented)
//
// Avoids V8↔Go binary bridge overhead for images, archives, etc.
//
// ws.ReadFileBase64(path) → string
//
//	Go: FS.ReadFile(name) → base64.StdEncoding.EncodeToString(data)
//	JS args:    path string
//	JS returns: string (base64-encoded content)
//
// ws.WriteFileBase64(path, b64, perm?) → void
//
//	Go: base64.StdEncoding.DecodeString(b64) → FS.WriteFile(name, data, perm)
//	JS args:  path string, b64 string, perm? number (default 0644)
//
// # Host copy (PLANNED — not yet implemented)
//
// Copy files/dirs from Yao host filesystem into the workspace volume.
// Useful for seeding workspaces with templates, config files, assets, etc.
//
// ws.CopyFromHost(hostPath, destPath?) → void
//
//	Copies a single file or directory tree from the Yao host into the workspace.
//	Go: read host file(s) → FS.WriteFile / FS.MkdirAll for each entry
//	JS args:  hostPath string (absolute path on Yao host),
//	          destPath? string (target path inside workspace, default basename of hostPath)
//
// ws.CopyFromHostArchive(hostPath, destPath?) → void
//
//	For large directory trees: zip on host → transfer → unzip on Tai node.
//	Requires Tai server-side unarchive support.
//	Go: zip hostPath → tai Volume upload → tai unarchive at destPath
//	JS args:  hostPath string, destPath? string (default ".")
func NewFSObject(v8ctx *v8go.Context, workspaceID string) (*v8go.Value, error) {
	// TODO: Phase 2 implementation
	// 1. Create JS object via v8go.NewObjectTemplate
	// 2. Set read-only properties: id, name, node (from workspace.M().Get(workspaceID))
	// 3. Bind each method as FunctionTemplate:
	//    - ReadFile  → workspace.M().FS(ctx, id).ReadFile(path)
	//    - WriteFile → workspace.M().FS(ctx, id).WriteFile(path, data, perm)
	//    - ReadDir   → workspace.M().FS(ctx, id).ReadDir(path)
	//    - Stat      → workspace.M().FS(ctx, id).Stat(path)
	//    - MkdirAll  → workspace.M().FS(ctx, id).MkdirAll(path, perm)
	//    - Remove    → workspace.M().FS(ctx, id).Remove(path)
	//    - RemoveAll → workspace.M().FS(ctx, id).RemoveAll(path)
	//    - Rename    → workspace.M().FS(ctx, id).Rename(old, new)
	//
	// PLANNED (not yet implemented):
	//    - ReadFileBase64  → ReadFile + base64 encode in Go
	//    - WriteFileBase64 → base64 decode in Go + WriteFile
	//    - CopyFromHost    → host fs.Read → FS.Write (file-by-file)
	//    - CopyFromHostArchive → zip on host → tai transfer → unzip (needs Tai support)
	return nil, nil
}
