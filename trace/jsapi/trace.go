package jsapi

import (
	"context"
	"fmt"

	v8 "github.com/yaoapp/gou/runtime/v8"
	"github.com/yaoapp/gou/runtime/v8/bridge"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/trace"
	"github.com/yaoapp/yao/trace/types"
	"rogchap.com/v8go"
)

func init() {
	// Auto-register Trace JavaScript API when package is imported
	v8.RegisterFunction("Trace", ExportFunction)
}

// NewTraceObject creates a JavaScript Trace object
func NewTraceObject(v8ctx *v8go.Context, traceID string, manager types.Manager) (*v8go.Value, error) {
	jsObject := v8go.NewObjectTemplate(v8ctx.Isolate())

	// Set internal field count to 1 to store the __go_id
	// Internal fields are not accessible from JavaScript, providing better security
	jsObject.SetInternalFieldCount(1)

	// Register manager in global bridge registry for efficient Go object retrieval
	// The goValueID will be stored in internal field (index 0) after instance creation
	// Internal fields are not accessible from JavaScript, providing better security
	goValueID := bridge.RegisterGoObject(manager)

	// Set primitive fields
	jsObject.Set("id", traceID)

	// Set release functions (both __release and Release do the same thing)
	// __release: Internal cleanup (called by GC or Use())
	// Release: Public method for manual cleanup (try-finally pattern)
	releaseFunc := traceGoRelease(v8ctx.Isolate(), traceID)
	jsObject.Set("__release", releaseFunc)
	jsObject.Set("Release", releaseFunc)

	// Set methods
	jsObject.Set("Add", traceAddMethod(v8ctx.Isolate(), manager))
	jsObject.Set("Parallel", traceParallelMethod(v8ctx.Isolate(), manager))
	jsObject.Set("Info", traceInfoMethod(v8ctx.Isolate(), manager))
	jsObject.Set("Debug", traceDebugMethod(v8ctx.Isolate(), manager))
	jsObject.Set("Error", traceErrorMethod(v8ctx.Isolate(), manager))
	jsObject.Set("Warn", traceWarnMethod(v8ctx.Isolate(), manager))
	jsObject.Set("SetOutput", traceSetOutputMethod(v8ctx.Isolate(), manager))
	jsObject.Set("SetMetadata", traceSetMetadataMethod(v8ctx.Isolate(), manager))
	jsObject.Set("Complete", traceCompleteMethod(v8ctx.Isolate(), manager))
	jsObject.Set("Fail", traceFailMethod(v8ctx.Isolate(), manager))
	jsObject.Set("MarkComplete", traceMarkCompleteMethod(v8ctx.Isolate(), manager))
	jsObject.Set("CreateSpace", traceCreateSpaceMethod(v8ctx.Isolate(), manager))
	jsObject.Set("GetSpace", traceGetSpaceMethod(v8ctx.Isolate(), manager))
	jsObject.Set("IsComplete", traceIsCompleteMethod(v8ctx.Isolate(), manager))

	// Create instance
	instance, err := jsObject.NewInstance(v8ctx)
	if err != nil {
		// Clean up: release from global registry if instance creation failed
		bridge.ReleaseGoObject(goValueID)
		return nil, err
	}

	// Store the goValueID in internal field (index 0)
	// This is not accessible from JavaScript, providing better security
	obj, err := instance.Value.AsObject()
	if err != nil {
		bridge.ReleaseGoObject(goValueID)
		return nil, err
	}

	err = obj.SetInternalField(0, goValueID)
	if err != nil {
		bridge.ReleaseGoObject(goValueID)
		return nil, err
	}

	return instance.Value, nil
}

// TraceNew creates a new Trace instance from JavaScript
// Usage: new Trace(options)
func TraceNew(v8ctx *v8go.Context, options map[string]interface{}) (*v8go.Value, error) {
	cfg := config.Conf

	// Prepare driver options
	var driverOptions []any
	var driverType string

	// Allow override from options
	driver, _ := options["driver"].(string)
	if driver == "" {
		driver = cfg.Trace.Driver
	}

	switch driver {
	case "store":
		driverType = trace.Store
		storeID := cfg.Trace.Store
		if sid, ok := options["store"].(string); ok && sid != "" {
			storeID = sid
		}
		prefix := cfg.Trace.Prefix
		if pfx, ok := options["prefix"].(string); ok {
			prefix = pfx
		}
		driverOptions = []any{storeID, prefix}

	case "local", "":
		driverType = trace.Local
		path := cfg.Trace.Path
		if p, ok := options["path"].(string); ok && p != "" {
			path = p
		}
		driverOptions = []any{path}

	default:
		return nil, fmt.Errorf("unsupported trace driver: %s", driver)
	}

	// Parse trace options
	var traceOpt types.TraceOption
	if id, ok := options["id"].(string); ok {
		traceOpt.ID = id
	}

	// Create trace
	goCtx := context.Background()
	traceID, manager, err := trace.New(goCtx, driverType, &traceOpt, driverOptions...)
	if err != nil {
		return nil, err
	}

	return NewTraceObject(v8ctx, traceID, manager)
}

// TraceLoad loads an existing trace from JavaScript
// Usage: Trace.Load(traceID)
func TraceLoad(v8ctx *v8go.Context, traceID string) (*v8go.Value, error) {
	manager, err := trace.Load(traceID)
	if err != nil {
		return nil, err
	}

	return NewTraceObject(v8ctx, traceID, manager)
}

// traceGoRelease releases the Go object from the global bridge registry
// It retrieves the goValueID from internal field (index 0) and releases the Go object
// It also calls trace.Release to cleanup the trace globally
func traceGoRelease(iso *v8go.Isolate, traceID string) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		// Get the trace object (this)
		thisObj, err := info.This().AsObject()
		if err == nil && thisObj.InternalFieldCount() > 0 {
			// Get goValueID from internal field (index 0)
			goValueIDValue := thisObj.GetInternalField(0)
			if goValueIDValue != nil && goValueIDValue.IsString() {
				goValueID := goValueIDValue.String()
				// Release from global bridge registry
				bridge.ReleaseGoObject(goValueID)
			}
		}

		// Call global trace.Release to remove from registry and stop background goroutines
		trace.Release(traceID)

		return v8go.Undefined(info.Context().Isolate())
	})
}

// Method templates

func traceAddMethod(iso *v8go.Isolate, manager types.Manager) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		ctx := info.Context()
		args := info.Args()

		if len(args) < 1 {
			return bridge.JsException(ctx, "Add requires at least 1 argument: (input, option?)")
		}

		// Parse input
		input, err := bridge.GoValue(args[0], ctx)
		if err != nil {
			return bridge.JsException(ctx, fmt.Sprintf("invalid input: %s", err))
		}

		// Parse option
		var option types.TraceNodeOption
		if len(args) > 1 && !args[1].IsNullOrUndefined() {
			optionObj, err := args[1].AsObject()
			if err == nil {
				if labelVal, err := optionObj.Get("label"); err == nil && !labelVal.IsNullOrUndefined() {
					option.Label = labelVal.String()
				}
				if iconVal, err := optionObj.Get("icon"); err == nil && !iconVal.IsNullOrUndefined() {
					option.Icon = iconVal.String()
				}
				if descVal, err := optionObj.Get("description"); err == nil && !descVal.IsNullOrUndefined() {
					option.Description = descVal.String()
				}
			}
		}

		// Call manager.Add
		node, err := manager.Add(input, option)
		if err != nil {
			return bridge.JsException(ctx, err.Error())
		}

		// Create node JS object
		nodeObj, err := NewNodeObject(ctx, node)
		if err != nil {
			return bridge.JsException(ctx, err.Error())
		}

		return nodeObj
	})
}

func traceParallelMethod(iso *v8go.Isolate, manager types.Manager) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		ctx := info.Context()
		args := info.Args()

		if len(args) < 1 {
			return bridge.JsException(ctx, "Parallel requires an array argument")
		}

		// Parse array
		parallelInputsJS, err := bridge.GoValue(args[0], ctx)
		if err != nil {
			return bridge.JsException(ctx, fmt.Sprintf("invalid parallel inputs: %s", err))
		}

		// Convert to []types.TraceParallelInput
		parallelInputsArray, ok := parallelInputsJS.([]interface{})
		if !ok {
			return bridge.JsException(ctx, "Parallel argument must be an array")
		}

		parallelInputs := make([]types.TraceParallelInput, 0, len(parallelInputsArray))
		for _, item := range parallelInputsArray {
			itemMap, ok := item.(map[string]interface{})
			if !ok {
				continue
			}

			parallelInput := types.TraceParallelInput{}
			if input, ok := itemMap["input"]; ok {
				parallelInput.Input = input
			}
			if option, ok := itemMap["option"].(map[string]interface{}); ok {
				if label, ok := option["label"].(string); ok {
					parallelInput.Option.Label = label
				}
				if icon, ok := option["icon"].(string); ok {
					parallelInput.Option.Icon = icon
				}
				if desc, ok := option["description"].(string); ok {
					parallelInput.Option.Description = desc
				}
			}

			parallelInputs = append(parallelInputs, parallelInput)
		}

		// Call manager.Parallel
		nodes, err := manager.Parallel(parallelInputs)
		if err != nil {
			return bridge.JsException(ctx, err.Error())
		}

		// Create array of node objects
		result := make([]interface{}, len(nodes))
		for i, node := range nodes {
			nodeObj, err := NewNodeObject(ctx, node)
			if err != nil {
				continue
			}
			result[i] = nodeObj
		}

		jsVal, err := bridge.JsValue(ctx, result)
		if err != nil {
			return bridge.JsException(ctx, err.Error())
		}

		return jsVal
	})
}

func traceInfoMethod(iso *v8go.Isolate, manager types.Manager) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) > 0 {
			manager.Info(args[0].String())
		}
		return info.This().Value
	})
}

func traceDebugMethod(iso *v8go.Isolate, manager types.Manager) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) > 0 {
			manager.Debug(args[0].String())
		}
		return info.This().Value
	})
}

func traceErrorMethod(iso *v8go.Isolate, manager types.Manager) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) > 0 {
			manager.Error(args[0].String())
		}
		return info.This().Value
	})
}

func traceWarnMethod(iso *v8go.Isolate, manager types.Manager) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) > 0 {
			manager.Warn(args[0].String())
		}
		return info.This().Value
	})
}

func traceSetOutputMethod(iso *v8go.Isolate, manager types.Manager) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		ctx := info.Context()
		args := info.Args()

		if len(args) < 1 {
			return bridge.JsException(ctx, "SetOutput requires an output argument")
		}

		output, err := bridge.GoValue(args[0], ctx)
		if err != nil {
			return bridge.JsException(ctx, err.Error())
		}

		if err := manager.SetOutput(output); err != nil {
			return bridge.JsException(ctx, err.Error())
		}

		return info.This().Value
	})
}

func traceSetMetadataMethod(iso *v8go.Isolate, manager types.Manager) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		ctx := info.Context()
		args := info.Args()

		if len(args) < 2 {
			return bridge.JsException(ctx, "SetMetadata requires 2 arguments: (key, value)")
		}

		key := args[0].String()
		value, err := bridge.GoValue(args[1], ctx)
		if err != nil {
			return bridge.JsException(ctx, err.Error())
		}

		if err := manager.SetMetadata(key, value); err != nil {
			return bridge.JsException(ctx, err.Error())
		}

		return info.This().Value
	})
}

func traceCompleteMethod(iso *v8go.Isolate, manager types.Manager) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		ctx := info.Context()
		args := info.Args()

		// Optional output parameter
		if len(args) > 0 && !args[0].IsNullOrUndefined() {
			output, err := bridge.GoValue(args[0], ctx)
			if err != nil {
				return bridge.JsException(ctx, err.Error())
			}
			if err := manager.Complete(output); err != nil {
				return bridge.JsException(ctx, err.Error())
			}
		} else {
			if err := manager.Complete(); err != nil {
				return bridge.JsException(ctx, err.Error())
			}
		}

		return info.This().Value
	})
}

func traceFailMethod(iso *v8go.Isolate, manager types.Manager) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		ctx := info.Context()
		args := info.Args()

		if len(args) < 1 {
			return bridge.JsException(ctx, "Fail requires an error message")
		}

		errMsg := args[0].String()
		if err := manager.Fail(fmt.Errorf("%s", errMsg)); err != nil {
			return bridge.JsException(ctx, err.Error())
		}

		return info.This().Value
	})
}

func traceMarkCompleteMethod(iso *v8go.Isolate, manager types.Manager) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		ctx := info.Context()

		if err := manager.MarkComplete(); err != nil {
			return bridge.JsException(ctx, err.Error())
		}

		return info.This().Value
	})
}

func traceCreateSpaceMethod(iso *v8go.Isolate, manager types.Manager) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		ctx := info.Context()
		args := info.Args()

		// Parse option
		var option types.TraceSpaceOption
		if len(args) > 0 && !args[0].IsNullOrUndefined() {
			optionObj, err := args[0].AsObject()
			if err == nil {
				if labelVal, err := optionObj.Get("label"); err == nil && !labelVal.IsNullOrUndefined() {
					option.Label = labelVal.String()
				}
				if typeVal, err := optionObj.Get("type"); err == nil && !typeVal.IsNullOrUndefined() {
					option.Type = typeVal.String()
				}
				if iconVal, err := optionObj.Get("icon"); err == nil && !iconVal.IsNullOrUndefined() {
					option.Icon = iconVal.String()
				}
				if descVal, err := optionObj.Get("description"); err == nil && !descVal.IsNullOrUndefined() {
					option.Description = descVal.String()
				}
			}
		}

		// Call manager.CreateSpace
		space, err := manager.CreateSpace(option)
		if err != nil {
			return bridge.JsException(ctx, err.Error())
		}

		// Create space JS object
		spaceObj, err := NewSpaceObject(ctx, manager, space)
		if err != nil {
			return bridge.JsException(ctx, err.Error())
		}

		return spaceObj
	})
}

func traceGetSpaceMethod(iso *v8go.Isolate, manager types.Manager) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		ctx := info.Context()
		args := info.Args()

		if len(args) < 1 {
			return bridge.JsException(ctx, "GetSpace requires a space ID")
		}

		spaceID := args[0].String()
		space, err := manager.GetSpace(spaceID)
		if err != nil {
			return bridge.JsException(ctx, err.Error())
		}

		if space == nil {
			return v8go.Null(iso)
		}

		// Create space JS object
		spaceObj, err := NewSpaceObject(ctx, manager, space)
		if err != nil {
			return bridge.JsException(ctx, err.Error())
		}

		return spaceObj
	})
}

func traceIsCompleteMethod(iso *v8go.Isolate, manager types.Manager) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		isComplete := manager.IsComplete()
		jsVal, _ := v8go.NewValue(iso, isComplete)
		return jsVal
	})
}

// NewNoOpTraceObject creates a no-op Trace object for when trace is not initialized
// All methods return undefined and do nothing
func NewNoOpTraceObject(v8ctx *v8go.Context) (*v8go.Value, error) {
	jsObject := v8go.NewObjectTemplate(v8ctx.Isolate())
	iso := v8ctx.Isolate()

	// Set id to empty string
	jsObject.Set("id", "")

	// No-op method factory
	noOpMethod := func() *v8go.FunctionTemplate {
		return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
			return v8go.Undefined(iso)
		})
	}

	// No-op node factory for Add and Parallel methods
	noOpNodeMethod := func() *v8go.FunctionTemplate {
		return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
			// Return a no-op node object
			nodeObj, _ := NewNoOpNodeObject(v8ctx)
			return nodeObj
		})
	}

	// Set all methods to no-op
	jsObject.Set("Add", noOpNodeMethod())
	jsObject.Set("Parallel", noOpNodeMethod())
	jsObject.Set("Info", noOpMethod())
	jsObject.Set("Debug", noOpMethod())
	jsObject.Set("Error", noOpMethod())
	jsObject.Set("Warn", noOpMethod())
	jsObject.Set("SetOutput", noOpMethod())
	jsObject.Set("SetMetadata", noOpMethod())
	jsObject.Set("Complete", noOpMethod())
	jsObject.Set("Fail", noOpMethod())
	jsObject.Set("MarkComplete", noOpMethod())
	jsObject.Set("CreateSpace", noOpMethod())
	jsObject.Set("GetSpace", noOpMethod())
	jsObject.Set("IsComplete", v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		jsVal, _ := v8go.NewValue(iso, false)
		return jsVal
	}))

	// Set release methods (no-op, but must be present for consistency)
	jsObject.Set("__release", noOpMethod())
	jsObject.Set("Release", noOpMethod())

	// Create instance
	instance, err := jsObject.NewInstance(v8ctx)
	if err != nil {
		return nil, err
	}

	return instance.Value, nil
}
