// Package jsapi registers the Workspace() constructor into the Yao V8 runtime.
//
// # JavaScript API
//
//	const ws = new Workspace({ node: "tai-1" })
//	const info = ws.Create({ name: "my-project", owner: "user1" })
//	const file = ws.ReadFile(info.id, "/README.md")
//	ws.WriteFile(info.id, "/app.ts", content)
//
// The constructor returns a WorkspaceManager object; individual workspace
// files are accessed through ReadFile/WriteFile/ListDir or the FS() handle.
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
	v8.RegisterFunction("Workspace", ExportFunction)
}

// ExportFunction exports the Workspace constructor to V8.
func ExportFunction(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, workspaceConstructor)
}

// workspaceConstructor is called when JS executes `new Workspace(options?)`.
//
// Options (all optional — uses global workspace.Manager if omitted):
//
//	{
//	  node: string  // default target node for Create (optional)
//	}
//
// Returns a WorkspaceManager JS object.
func workspaceConstructor(info *v8go.FunctionCallbackInfo) *v8go.Value {
	// TODO: Phase 2 implementation
	// 1. Parse optional options from args[0]
	// 2. Get workspace manager instance
	// 3. Return NewManagerObject(v8ctx, manager, defaults)
	return v8go.Undefined(info.Context().Isolate())
}
