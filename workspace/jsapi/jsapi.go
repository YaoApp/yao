// Package jsapi registers the workspace namespace into the Yao V8 runtime.
//
// All methods are static on the workspace object — no constructor.
//
// # JavaScript API
//
//	const ws = workspace.Create({ name: "proj", owner: "user1", node: "default" })
//	const ws = workspace.Get(id)
//	ws.ReadFile("main.go")          → string
//	ws.WriteFile("out.txt", data)   → void
//	ws.ReadDir("src/")              → [{ name, is_dir, size }]
//	workspace.Delete(id)            → void
//
// # Go mapping
//
//	workspace.Create(opts)  → Manager.Create(ctx, CreateOptions)   → *Workspace → WorkspaceFS
//	workspace.Get(id)       → Manager.Get(ctx, id)                 → *Workspace → WorkspaceFS
//	workspace.List(filter?) → Manager.List(ctx, ListOptions)       → []*Workspace → WorkspaceInfo[]
//	workspace.Delete(id)    → Manager.Delete(ctx, id, false)       → void
//
// Registration happens via init() — import with:
//
//	_ "github.com/yaoapp/yao/workspace/jsapi"
package jsapi

import (
	v8 "github.com/yaoapp/gou/runtime/v8"
	"rogchap.com/v8go"
)

func init() {
	v8.RegisterObject("workspace", ExportObject)
}

// ExportObject exports the workspace namespace object to V8.
func ExportObject(iso *v8go.Isolate) *v8go.ObjectTemplate {
	obj := v8go.NewObjectTemplate(iso)
	obj.Set("Create", v8go.NewFunctionTemplate(iso, wsCreate))
	obj.Set("Get", v8go.NewFunctionTemplate(iso, wsGet))
	obj.Set("List", v8go.NewFunctionTemplate(iso, wsList))
	obj.Set("Delete", v8go.NewFunctionTemplate(iso, wsDelete))
	return obj
}

// wsCreate: `workspace.Create(options)` → WorkspaceFS
//
// Go: Manager.Create(ctx, CreateOptions) (*Workspace, error)
//
// JS options → Go CreateOptions mapping:
//
//	{
//	  id:     string  →  CreateOptions.ID      // optional; auto-generated if empty
//	  name:   string  →  CreateOptions.Name    // required, human-readable name
//	  owner:  string  →  CreateOptions.Owner   // required, user ID
//	  node:   string  →  CreateOptions.Node    // required, target Tai node
//	  labels: object  →  CreateOptions.Labels  // optional, map[string]string
//	}
//
// Returns: WorkspaceFS object (see fs.go)
func wsCreate(info *v8go.FunctionCallbackInfo) *v8go.Value {
	// TODO: Phase 2
	// 1. Parse options from info.Args()[0]
	// 2. Validate required fields (name, owner, node)
	// 3. ws := workspace.M().Create(ctx, opts)
	// 4. Return NewFSObject(v8ctx, ws.ID)
	return v8go.Undefined(info.Context().Isolate())
}

// wsGet: `workspace.Get(id)` → WorkspaceFS | null
//
// Go: Manager.Get(ctx, id) (*Workspace, error)
//
// Args:
//
//	id: string  — workspace ID
//
// Returns: WorkspaceFS object if found, null if not found
func wsGet(info *v8go.FunctionCallbackInfo) *v8go.Value {
	// TODO: Phase 2
	// 1. id = info.Args()[0].String()
	// 2. ws, err := workspace.M().Get(ctx, id)
	// 3. Return NewFSObject(v8ctx, id) or null
	return v8go.Undefined(info.Context().Isolate())
}

// wsList: `workspace.List(filter?)` → WorkspaceInfo[]
//
// Go: Manager.List(ctx, ListOptions) ([]*Workspace, error)
//
// JS filter → Go ListOptions mapping:
//
//	{
//	  owner: string  →  ListOptions.Owner  // filter by owner; empty = all
//	  node:  string  →  ListOptions.Node   // filter by node; empty = all
//	}
//
// Returns: WorkspaceInfo[] — each element:
//
//	{
//	  id:         string  ←  Workspace.ID
//	  name:       string  ←  Workspace.Name
//	  owner:      string  ←  Workspace.Owner
//	  node:       string  ←  Workspace.Node
//	  labels:     object  ←  Workspace.Labels
//	  created_at: string  ←  Workspace.CreatedAt (ISO 8601)
//	  updated_at: string  ←  Workspace.UpdatedAt (ISO 8601)
//	}
func wsList(info *v8go.FunctionCallbackInfo) *v8go.Value {
	// TODO: Phase 2
	// 1. Parse optional filter from info.Args()[0]
	// 2. list := workspace.M().List(ctx, opts)
	// 3. Convert each *Workspace → JS object
	// 4. Return JS array
	return v8go.Undefined(info.Context().Isolate())
}

// wsDelete: `workspace.Delete(id)` → void
//
// Go: Manager.Delete(ctx, id string, force bool) error
//
// Args:
//
//	id: string  — workspace ID to remove (force = false)
func wsDelete(info *v8go.FunctionCallbackInfo) *v8go.Value {
	// TODO: Phase 2
	// 1. id = info.Args()[0].String()
	// 2. workspace.M().Delete(ctx, id, false)
	return v8go.Undefined(info.Context().Isolate())
}
