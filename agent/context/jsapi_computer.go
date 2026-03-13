package context

import (
	"context"
	"fmt"
	"strings"

	"github.com/yaoapp/gou/runtime/v8/bridge"
	infraV2 "github.com/yaoapp/yao/sandbox/v2"
	"github.com/yaoapp/yao/tai/workspace"
	"rogchap.com/v8go"
)

// SetComputer sets the V2 computer and its workspace for this context.
// Should be called after Runner.Prepare succeeds in initSandboxV2.
func (ctx *Context) SetComputer(computer infraV2.Computer) {
	ctx.computer = computer
	if computer != nil {
		ctx.workspace = computer.Workplace()
	}
}

// SetWorkspace sets the workspace FS directly without requiring a Computer.
// Use this when the user selected a workspace but no sandbox is configured.
func (ctx *Context) SetWorkspace(ws workspace.FS) {
	ctx.workspace = ws
}

// GetComputer returns the V2 computer if available.
func (ctx *Context) GetComputer() infraV2.Computer {
	return ctx.computer
}

// GetWorkspace returns the V2 workspace FS if available.
func (ctx *Context) GetWorkspace() workspace.FS {
	return ctx.workspace
}

// HasComputer returns true if V2 computer is available.
func (ctx *Context) HasComputer() bool {
	return ctx.computer != nil
}

// HasWorkspace returns true if workspace FS is available.
func (ctx *Context) HasWorkspace() bool {
	return ctx.workspace != nil
}

// createComputerInstance creates the ctx.computer JavaScript object.
func (ctx *Context) createComputerInstance(v8ctx *v8go.Context) *v8go.Value {
	if ctx.computer == nil {
		return nil
	}

	iso := v8ctx.Isolate()
	objTpl := v8go.NewObjectTemplate(iso)

	info := ctx.computer.ComputerInfo()
	id := info.BoxID
	if id == "" {
		id = info.NodeID
	}
	objTpl.Set("id", id)

	objTpl.Set("Exec", ctx.computerExecMethod(iso))
	objTpl.Set("VNC", ctx.computerVNCMethod(iso))
	objTpl.Set("Proxy", ctx.computerProxyMethod(iso))
	objTpl.Set("Info", ctx.computerInfoMethod(iso))

	instance, err := objTpl.NewInstance(v8ctx)
	if err != nil {
		return nil
	}
	return instance.Value
}

// computerExecMethod implements ctx.computer.Exec(cmd)
// cmd can be a string or an array of strings.
// Returns: { stdout, stderr, exit_code }
func (ctx *Context) computerExecMethod(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		v8ctx := info.Context()
		args := info.Args()

		if ctx.computer == nil {
			return bridge.JsException(v8ctx, "computer not available")
		}
		if len(args) < 1 {
			return bridge.JsException(v8ctx, "Exec requires a command argument")
		}

		cmd, err := parseCommandArg(v8ctx, args[0])
		if err != nil {
			return bridge.JsException(v8ctx, err.Error())
		}

		result, err := ctx.computer.Exec(context.Background(), cmd)
		if err != nil {
			return bridge.JsException(v8ctx, "Exec failed: "+err.Error())
		}

		res := map[string]interface{}{
			"stdout":    result.Stdout,
			"stderr":    result.Stderr,
			"exit_code": int32(result.ExitCode),
		}
		jsVal, err := bridge.JsValue(v8ctx, res)
		if err != nil {
			return bridge.JsException(v8ctx, err.Error())
		}
		return jsVal
	})
}

// computerVNCMethod implements ctx.computer.VNC()
// Returns the VNC URL string.
func (ctx *Context) computerVNCMethod(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		v8ctx := info.Context()

		if ctx.computer == nil {
			return bridge.JsException(v8ctx, "computer not available")
		}

		url, err := ctx.computer.VNC(context.Background())
		if err != nil {
			return bridge.JsException(v8ctx, "VNC failed: "+err.Error())
		}

		jsVal, err := v8go.NewValue(iso, url)
		if err != nil {
			return bridge.JsException(v8ctx, err.Error())
		}
		return jsVal
	})
}

// computerProxyMethod implements ctx.computer.Proxy(port, path?)
// Returns the proxy URL string.
func (ctx *Context) computerProxyMethod(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		v8ctx := info.Context()
		args := info.Args()

		if ctx.computer == nil {
			return bridge.JsException(v8ctx, "computer not available")
		}
		if len(args) < 1 || !args[0].IsNumber() {
			return bridge.JsException(v8ctx, "Proxy requires a port number")
		}

		port := int(args[0].Integer())
		path := ""
		if len(args) >= 2 && args[1].IsString() {
			path = args[1].String()
		}

		url, err := ctx.computer.Proxy(context.Background(), port, path)
		if err != nil {
			return bridge.JsException(v8ctx, "Proxy failed: "+err.Error())
		}

		jsVal, err := v8go.NewValue(iso, url)
		if err != nil {
			return bridge.JsException(v8ctx, err.Error())
		}
		return jsVal
	})
}

// computerInfoMethod implements ctx.computer.Info()
// Returns a JS object with computer identity and system information.
func (ctx *Context) computerInfoMethod(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		v8ctx := info.Context()

		if ctx.computer == nil {
			return bridge.JsException(v8ctx, "computer not available")
		}

		ci := ctx.computer.ComputerInfo()
		result := map[string]interface{}{
			"kind":    ci.Kind,
			"node_id": ci.NodeID,
			"tai_id":  ci.TaiID,
			"status":  ci.Status,
			"system": map[string]interface{}{
				"os":       ci.System.OS,
				"arch":     ci.System.Arch,
				"hostname": ci.System.Hostname,
				"num_cpu":  int32(ci.System.NumCPU),
				"shell":    ci.System.Shell,
			},
		}
		if ci.BoxID != "" {
			result["box_id"] = ci.BoxID
			result["container_id"] = ci.ContainerID
			result["image"] = ci.Image
			result["policy"] = string(ci.Policy)
		}

		jsVal, err := bridge.JsValue(v8ctx, result)
		if err != nil {
			return bridge.JsException(v8ctx, err.Error())
		}
		return jsVal
	})
}

// parseCommandArg converts a JS value (string or string array) to []string.
func parseCommandArg(v8ctx *v8go.Context, val *v8go.Value) ([]string, error) {
	if val.IsString() {
		raw := val.String()
		return strings.Fields(raw), nil
	}

	if val.IsArray() {
		obj, err := val.AsObject()
		if err != nil {
			return nil, err
		}
		lengthVal, err := obj.Get("length")
		if err != nil {
			return nil, err
		}
		length := int(lengthVal.Integer())
		cmd := make([]string, length)
		for i := 0; i < length; i++ {
			item, err := obj.GetIdx(uint32(i))
			if err != nil {
				return nil, err
			}
			cmd[i] = item.String()
		}
		return cmd, nil
	}

	return nil, fmt.Errorf("command must be a string or array of strings")
}
