// Package jsapi registers the Sandbox() constructor into the Yao V8 runtime.
//
// # JavaScript API
//
//	const sb = new Sandbox({ pool: "default", image: "node:20", owner: "user1" })
//	const box = sb.Create({ workdir: "/app", env: { NODE_ENV: "dev" } })
//	const result = box.Exec(["node", "-e", "console.log('hi')"])
//	box.Remove()
//
// The constructor returns a SandboxManager object; Create/GetOrCreate returns
// a Box object with Exec/Stream/Attach/VNC/Proxy/Workspace/Info/Stop/Start/Remove.
//
// Registration happens via init() — import with:
//
//	_ "github.com/yaoapp/yao/sandbox/v2/jsapi"
package jsapi

import (
	v8 "github.com/yaoapp/gou/runtime/v8"
	"rogchap.com/v8go"
)

func init() {
	v8.RegisterFunction("Sandbox", ExportFunction)
}

// ExportFunction exports the Sandbox constructor to V8.
func ExportFunction(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, sandboxConstructor)
}

// sandboxConstructor is called when JS executes `new Sandbox(options)`.
//
// Options:
//
//	{
//	  pool:    string  // pool name (required)
//	  image:   string  // container image (required)
//	  owner:   string  // owner ID (required)
//	}
//
// Returns a SandboxManager JS object.
func sandboxConstructor(info *v8go.FunctionCallbackInfo) *v8go.Value {
	// TODO: Phase 2 implementation
	// 1. Parse options from args[0]
	// 2. Validate required fields (pool, image, owner)
	// 3. Get sandbox.M() singleton
	// 4. Return NewManagerObject(v8ctx, manager, options)
	return v8go.Undefined(info.Context().Isolate())
}
