package context

import (
	"context"

	"github.com/yaoapp/gou/runtime/v8/bridge"
	infraSandbox "github.com/yaoapp/yao/sandbox"
	"rogchap.com/v8go"
)

// SandboxExecutor defines the interface for sandbox operations
// This interface is implemented by agent/sandbox.Executor
// It's defined here to avoid import cycles
type SandboxExecutor interface {
	// Filesystem operations
	ReadFile(ctx context.Context, path string) ([]byte, error)
	WriteFile(ctx context.Context, path string, content []byte) error
	ListDir(ctx context.Context, path string) ([]infraSandbox.FileInfo, error)

	// Command execution
	Exec(ctx context.Context, cmd []string) (string, error)

	// Workspace info
	GetWorkDir() string
}

// SetSandboxExecutor sets the sandbox executor for this context
// This should be called before hooks are executed
func (ctx *Context) SetSandboxExecutor(executor SandboxExecutor) {
	ctx.sandboxExecutor = executor
}

// GetSandboxExecutor returns the sandbox executor if available
func (ctx *Context) GetSandboxExecutor() SandboxExecutor {
	return ctx.sandboxExecutor
}

// HasSandbox returns true if sandbox executor is available
func (ctx *Context) HasSandbox() bool {
	return ctx.sandboxExecutor != nil
}

// newSandboxObject creates the ctx.sandbox JavaScript object
// Returns nil if sandbox executor is not available
func (ctx *Context) newSandboxObject(iso *v8go.Isolate) *v8go.ObjectTemplate {
	if ctx.sandboxExecutor == nil {
		return nil
	}

	sandboxObj := v8go.NewObjectTemplate(iso)

	// Set methods
	sandboxObj.Set("ReadFile", ctx.sandboxReadFileMethod(iso))
	sandboxObj.Set("WriteFile", ctx.sandboxWriteFileMethod(iso))
	sandboxObj.Set("ListDir", ctx.sandboxListDirMethod(iso))
	sandboxObj.Set("Exec", ctx.sandboxExecMethod(iso))

	return sandboxObj
}

// createSandboxInstance creates the sandbox object instance with workdir property
func (ctx *Context) createSandboxInstance(v8ctx *v8go.Context) *v8go.Value {
	if ctx.sandboxExecutor == nil {
		return nil
	}

	sandboxTemplate := ctx.newSandboxObject(v8ctx.Isolate())
	if sandboxTemplate == nil {
		return nil
	}

	// Set workdir as a property
	sandboxTemplate.Set("workdir", ctx.sandboxExecutor.GetWorkDir())

	instance, err := sandboxTemplate.NewInstance(v8ctx)
	if err != nil {
		return nil
	}

	return instance.Value
}

// sandboxReadFileMethod implements ctx.sandbox.ReadFile(path)
func (ctx *Context) sandboxReadFileMethod(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		v8ctx := info.Context()
		args := info.Args()

		if ctx.sandboxExecutor == nil {
			return bridge.JsException(v8ctx, "sandbox executor not available")
		}

		if len(args) < 1 {
			return bridge.JsException(v8ctx, "ReadFile requires path parameter")
		}

		path := args[0].String()

		content, err := ctx.sandboxExecutor.ReadFile(context.Background(), path)
		if err != nil {
			return bridge.JsException(v8ctx, err.Error())
		}

		// Return as string
		jsVal, err := v8go.NewValue(iso, string(content))
		if err != nil {
			return bridge.JsException(v8ctx, err.Error())
		}

		return jsVal
	})
}

// sandboxWriteFileMethod implements ctx.sandbox.WriteFile(path, content)
func (ctx *Context) sandboxWriteFileMethod(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		v8ctx := info.Context()
		args := info.Args()

		if ctx.sandboxExecutor == nil {
			return bridge.JsException(v8ctx, "sandbox executor not available")
		}

		if len(args) < 2 {
			return bridge.JsException(v8ctx, "WriteFile requires path and content parameters")
		}

		path := args[0].String()
		content := args[1].String()

		err := ctx.sandboxExecutor.WriteFile(context.Background(), path, []byte(content))
		if err != nil {
			return bridge.JsException(v8ctx, err.Error())
		}

		// Return undefined on success
		return v8go.Undefined(iso)
	})
}

// sandboxListDirMethod implements ctx.sandbox.ListDir(path)
func (ctx *Context) sandboxListDirMethod(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		v8ctx := info.Context()
		args := info.Args()

		if ctx.sandboxExecutor == nil {
			return bridge.JsException(v8ctx, "sandbox executor not available")
		}

		if len(args) < 1 {
			return bridge.JsException(v8ctx, "ListDir requires path parameter")
		}

		path := args[0].String()

		files, err := ctx.sandboxExecutor.ListDir(context.Background(), path)
		if err != nil {
			return bridge.JsException(v8ctx, err.Error())
		}

		// Convert to JavaScript array of objects
		result := make([]map[string]interface{}, len(files))
		for i, f := range files {
			result[i] = map[string]interface{}{
				"name":   f.Name,
				"size":   f.Size,
				"is_dir": f.IsDir,
			}
		}

		jsVal, err := bridge.JsValue(v8ctx, result)
		if err != nil {
			return bridge.JsException(v8ctx, err.Error())
		}

		return jsVal
	})
}

// sandboxExecMethod implements ctx.sandbox.Exec(cmd)
func (ctx *Context) sandboxExecMethod(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		v8ctx := info.Context()
		args := info.Args()

		if ctx.sandboxExecutor == nil {
			return bridge.JsException(v8ctx, "sandbox executor not available")
		}

		if len(args) < 1 {
			return bridge.JsException(v8ctx, "Exec requires cmd parameter (array of strings)")
		}

		// Parse command array
		cmdArg := args[0]
		if !cmdArg.IsArray() {
			return bridge.JsException(v8ctx, "Exec requires cmd to be an array of strings")
		}

		cmdObj, err := cmdArg.AsObject()
		if err != nil {
			return bridge.JsException(v8ctx, "failed to parse cmd array: "+err.Error())
		}

		// Get array length
		lengthVal, err := cmdObj.Get("length")
		if err != nil {
			return bridge.JsException(v8ctx, "failed to get cmd array length: "+err.Error())
		}
		length := int(lengthVal.Integer())

		// Build command slice
		cmd := make([]string, length)
		for i := 0; i < length; i++ {
			itemVal, err := cmdObj.GetIdx(uint32(i))
			if err != nil {
				return bridge.JsException(v8ctx, "failed to get cmd array element: "+err.Error())
			}
			cmd[i] = itemVal.String()
		}

		output, err := ctx.sandboxExecutor.Exec(context.Background(), cmd)
		if err != nil {
			return bridge.JsException(v8ctx, err.Error())
		}

		jsVal, err := v8go.NewValue(v8ctx.Isolate(), output)
		if err != nil {
			return bridge.JsException(v8ctx, err.Error())
		}

		return jsVal
	})
}
