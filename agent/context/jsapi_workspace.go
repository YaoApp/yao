package context

import (
	"io/fs"
	"os"

	"github.com/yaoapp/gou/runtime/v8/bridge"
	"rogchap.com/v8go"
)

// createWorkspaceInstance creates the ctx.workspace JavaScript object.
func (ctx *Context) createWorkspaceInstance(v8ctx *v8go.Context) *v8go.Value {
	if ctx.workspace == nil {
		return nil
	}

	iso := v8ctx.Isolate()
	objTpl := v8go.NewObjectTemplate(iso)

	objTpl.Set("ReadFile", ctx.wsReadFileMethod(iso))
	objTpl.Set("WriteFile", ctx.wsWriteFileMethod(iso))
	objTpl.Set("ReadDir", ctx.wsReadDirMethod(iso))
	objTpl.Set("MkdirAll", ctx.wsMkdirAllMethod(iso))
	objTpl.Set("Remove", ctx.wsRemoveMethod(iso))
	objTpl.Set("RemoveAll", ctx.wsRemoveAllMethod(iso))
	objTpl.Set("Rename", ctx.wsRenameMethod(iso))
	objTpl.Set("Copy", ctx.wsCopyMethod(iso))
	objTpl.Set("Stat", ctx.wsStatMethod(iso))
	objTpl.Set("Exists", ctx.wsExistsMethod(iso))

	instance, err := objTpl.NewInstance(v8ctx)
	if err != nil {
		return nil
	}
	return instance.Value
}

// wsReadFileMethod implements ctx.workspace.ReadFile(path)
func (ctx *Context) wsReadFileMethod(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		v8ctx := info.Context()
		args := info.Args()

		if ctx.workspace == nil {
			return bridge.JsException(v8ctx, "workspace not available")
		}
		if len(args) < 1 {
			return bridge.JsException(v8ctx, "ReadFile requires a path argument")
		}

		data, err := ctx.workspace.ReadFile(args[0].String())
		if err != nil {
			return bridge.JsException(v8ctx, "ReadFile failed: "+err.Error())
		}

		jsVal, err := v8go.NewValue(iso, string(data))
		if err != nil {
			return bridge.JsException(v8ctx, err.Error())
		}
		return jsVal
	})
}

// wsWriteFileMethod implements ctx.workspace.WriteFile(path, content)
func (ctx *Context) wsWriteFileMethod(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		v8ctx := info.Context()
		args := info.Args()

		if ctx.workspace == nil {
			return bridge.JsException(v8ctx, "workspace not available")
		}
		if len(args) < 2 {
			return bridge.JsException(v8ctx, "WriteFile requires path and content arguments")
		}

		path := args[0].String()
		content := args[1].String()

		if err := ctx.workspace.WriteFile(path, []byte(content), 0o644); err != nil {
			return bridge.JsException(v8ctx, "WriteFile failed: "+err.Error())
		}
		return v8go.Undefined(iso)
	})
}

// wsReadDirMethod implements ctx.workspace.ReadDir(path)
func (ctx *Context) wsReadDirMethod(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		v8ctx := info.Context()
		args := info.Args()

		if ctx.workspace == nil {
			return bridge.JsException(v8ctx, "workspace not available")
		}

		path := "."
		if len(args) >= 1 && args[0].IsString() {
			path = args[0].String()
		}

		entries, err := ctx.workspace.ReadDir(path)
		if err != nil {
			return bridge.JsException(v8ctx, "ReadDir failed: "+err.Error())
		}

		result := make([]map[string]interface{}, 0, len(entries))
		for _, e := range entries {
			fi, _ := e.Info()
			item := map[string]interface{}{
				"name":   e.Name(),
				"is_dir": e.IsDir(),
			}
			if fi != nil {
				item["size"] = int32(fi.Size())
			}
			result = append(result, item)
		}

		jsVal, err := bridge.JsValue(v8ctx, result)
		if err != nil {
			return bridge.JsException(v8ctx, err.Error())
		}
		return jsVal
	})
}

// wsMkdirAllMethod implements ctx.workspace.MkdirAll(path)
func (ctx *Context) wsMkdirAllMethod(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		v8ctx := info.Context()
		args := info.Args()

		if ctx.workspace == nil {
			return bridge.JsException(v8ctx, "workspace not available")
		}
		if len(args) < 1 {
			return bridge.JsException(v8ctx, "MkdirAll requires a path argument")
		}

		if err := ctx.workspace.MkdirAll(args[0].String(), 0o755); err != nil {
			return bridge.JsException(v8ctx, "MkdirAll failed: "+err.Error())
		}
		return v8go.Undefined(iso)
	})
}

// wsRemoveMethod implements ctx.workspace.Remove(path)
func (ctx *Context) wsRemoveMethod(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		v8ctx := info.Context()
		args := info.Args()

		if ctx.workspace == nil {
			return bridge.JsException(v8ctx, "workspace not available")
		}
		if len(args) < 1 {
			return bridge.JsException(v8ctx, "Remove requires a path argument")
		}

		if err := ctx.workspace.Remove(args[0].String()); err != nil {
			return bridge.JsException(v8ctx, "Remove failed: "+err.Error())
		}
		return v8go.Undefined(iso)
	})
}

// wsRemoveAllMethod implements ctx.workspace.RemoveAll(path)
func (ctx *Context) wsRemoveAllMethod(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		v8ctx := info.Context()
		args := info.Args()

		if ctx.workspace == nil {
			return bridge.JsException(v8ctx, "workspace not available")
		}
		if len(args) < 1 {
			return bridge.JsException(v8ctx, "RemoveAll requires a path argument")
		}

		if err := ctx.workspace.RemoveAll(args[0].String()); err != nil {
			return bridge.JsException(v8ctx, "RemoveAll failed: "+err.Error())
		}
		return v8go.Undefined(iso)
	})
}

// wsRenameMethod implements ctx.workspace.Rename(oldName, newName)
func (ctx *Context) wsRenameMethod(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		v8ctx := info.Context()
		args := info.Args()

		if ctx.workspace == nil {
			return bridge.JsException(v8ctx, "workspace not available")
		}
		if len(args) < 2 {
			return bridge.JsException(v8ctx, "Rename requires oldName and newName arguments")
		}

		if err := ctx.workspace.Rename(args[0].String(), args[1].String()); err != nil {
			return bridge.JsException(v8ctx, "Rename failed: "+err.Error())
		}
		return v8go.Undefined(iso)
	})
}

// wsCopyMethod implements ctx.workspace.Copy(src, dst)
func (ctx *Context) wsCopyMethod(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		v8ctx := info.Context()
		args := info.Args()

		if ctx.workspace == nil {
			return bridge.JsException(v8ctx, "workspace not available")
		}
		if len(args) < 2 {
			return bridge.JsException(v8ctx, "Copy requires src and dst arguments")
		}

		if _, err := ctx.workspace.Copy(args[0].String(), args[1].String()); err != nil {
			return bridge.JsException(v8ctx, "Copy failed: "+err.Error())
		}
		return v8go.Undefined(iso)
	})
}

// wsStatMethod implements ctx.workspace.Stat(path)
func (ctx *Context) wsStatMethod(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		v8ctx := info.Context()
		args := info.Args()

		if ctx.workspace == nil {
			return bridge.JsException(v8ctx, "workspace not available")
		}
		if len(args) < 1 {
			return bridge.JsException(v8ctx, "Stat requires a path argument")
		}

		fi, err := ctx.workspace.Stat(args[0].String())
		if err != nil {
			return bridge.JsException(v8ctx, "Stat failed: "+err.Error())
		}

		result := map[string]interface{}{
			"name":   fi.Name(),
			"size":   int32(fi.Size()),
			"is_dir": fi.IsDir(),
			"mode":   int32(fi.Mode()),
			"mtime":  fi.ModTime().UnixMilli(),
		}
		jsVal, err := bridge.JsValue(v8ctx, result)
		if err != nil {
			return bridge.JsException(v8ctx, err.Error())
		}
		return jsVal
	})
}

// wsExistsMethod implements ctx.workspace.Exists(path)
func (ctx *Context) wsExistsMethod(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		v8ctx := info.Context()
		args := info.Args()

		if ctx.workspace == nil {
			return bridge.JsException(v8ctx, "workspace not available")
		}
		if len(args) < 1 {
			return bridge.JsException(v8ctx, "Exists requires a path argument")
		}

		_, err := ctx.workspace.Stat(args[0].String())
		exists := err == nil || !isNotExist(err)

		jsVal, _ := v8go.NewValue(iso, exists)
		return jsVal
	})
}

func isNotExist(err error) bool {
	if os.IsNotExist(err) {
		return true
	}
	pathErr, ok := err.(*fs.PathError)
	if ok && os.IsNotExist(pathErr.Err) {
		return true
	}
	return false
}
