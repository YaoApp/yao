package jsapi

import (
	"fmt"

	"github.com/yaoapp/gou/runtime/v8/bridge"
	"github.com/yaoapp/yao/trace/types"
	"rogchap.com/v8go"
)

// Helper functions

func parseTraceInput(obj *v8go.Object, ctx *v8go.Context) (types.TraceInput, error) {
	goVal, err := bridge.GoValue(obj.Value, ctx)
	if err != nil {
		return nil, err
	}
	return goVal, nil
}

func parseTraceNodeOption(obj *v8go.Object) types.TraceNodeOption {
	option := types.TraceNodeOption{}
	if labelVal, err := obj.Get("label"); err == nil && !labelVal.IsNullOrUndefined() {
		option.Label = labelVal.String()
	}
	if typeVal, err := obj.Get("type"); err == nil && !typeVal.IsNullOrUndefined() {
		option.Type = typeVal.String()
	}
	if iconVal, err := obj.Get("icon"); err == nil && !iconVal.IsNullOrUndefined() {
		option.Icon = iconVal.String()
	}
	if descVal, err := obj.Get("description"); err == nil && !descVal.IsNullOrUndefined() {
		option.Description = descVal.String()
	}
	if autoCompleteVal, err := obj.Get("autoCompleteParent"); err == nil && !autoCompleteVal.IsNullOrUndefined() {
		boolVal := autoCompleteVal.Boolean()
		option.AutoCompleteParent = &boolVal
	}
	return option
}

// NewNodeObject creates a JavaScript Node object (pure JS object, no Go mapping)
func NewNodeObject(v8ctx *v8go.Context, node types.Node) (*v8go.Value, error) {
	jsObject := v8go.NewObjectTemplate(v8ctx.Isolate())

	// Set primitive fields
	jsObject.Set("id", node.ID())

	// Set methods
	jsObject.Set("Info", nodeInfoMethod(v8ctx.Isolate(), node))
	jsObject.Set("Debug", nodeDebugMethod(v8ctx.Isolate(), node))
	jsObject.Set("Error", nodeErrorMethod(v8ctx.Isolate(), node))
	jsObject.Set("Warn", nodeWarnMethod(v8ctx.Isolate(), node))
	jsObject.Set("Add", nodeAddMethod(v8ctx.Isolate(), node))
	jsObject.Set("Parallel", nodeParallelMethod(v8ctx.Isolate(), node))
	jsObject.Set("SetOutput", nodeSetOutputMethod(v8ctx.Isolate(), node))
	jsObject.Set("SetMetadata", nodeSetMetadataMethod(v8ctx.Isolate(), node))
	jsObject.Set("Complete", nodeCompleteMethod(v8ctx.Isolate(), node))
	jsObject.Set("Fail", nodeFailMethod(v8ctx.Isolate(), node))

	// Create instance
	instance, err := jsObject.NewInstance(v8ctx)
	if err != nil {
		return nil, err
	}

	return instance.Value, nil
}

// Node method templates

func nodeInfoMethod(iso *v8go.Isolate, node types.Node) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) > 0 {
			node.Info(args[0].String())
		}
		return info.This().Value
	})
}

func nodeDebugMethod(iso *v8go.Isolate, node types.Node) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) > 0 {
			node.Debug(args[0].String())
		}
		return info.This().Value
	})
}

func nodeErrorMethod(iso *v8go.Isolate, node types.Node) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) > 0 {
			node.Error(args[0].String())
		}
		return info.This().Value
	})
}

func nodeWarnMethod(iso *v8go.Isolate, node types.Node) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) > 0 {
			node.Warn(args[0].String())
		}
		return info.This().Value
	})
}

func nodeAddMethod(iso *v8go.Isolate, node types.Node) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		ctx := info.Context()
		args := info.Args()

		if len(args) < 2 {
			return bridge.JsException(ctx, "Add requires 2 arguments: (input, option)")
		}

		// Parse input
		inputObj, err := args[0].AsObject()
		if err != nil {
			return bridge.JsException(ctx, "first argument must be an object")
		}
		input, _ := parseTraceInput(inputObj, ctx)

		// Parse option
		optionObj, err := args[1].AsObject()
		if err != nil {
			return bridge.JsException(ctx, "second argument must be an object")
		}
		option := parseTraceNodeOption(optionObj)

		// Call node.Add
		childNode, err := node.Add(input, option)
		if err != nil {
			return bridge.JsException(ctx, err.Error())
		}

		// Create child node JS object
		childNodeObj, err := NewNodeObject(ctx, childNode)
		if err != nil {
			return bridge.JsException(ctx, err.Error())
		}

		return childNodeObj
	})
}

func nodeParallelMethod(iso *v8go.Isolate, node types.Node) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		ctx := info.Context()
		args := info.Args()

		if len(args) < 1 {
			return bridge.JsException(ctx, "Parallel requires an array argument")
		}

		// Parse array
		arrayObj, err := args[0].AsObject()
		if err != nil {
			return bridge.JsException(ctx, "argument must be an array")
		}

		lengthVal, err := arrayObj.Get("length")
		if err != nil {
			return bridge.JsException(ctx, "invalid array")
		}

		length := int(lengthVal.Int32())
		parallelInputs := make([]types.TraceParallelInput, 0, length)

		for i := 0; i < length; i++ {
			itemVal, err := arrayObj.GetIdx(uint32(i))
			if err != nil {
				continue
			}

			itemObj, err := itemVal.AsObject()
			if err != nil {
				continue
			}

			// Parse input
			var input types.TraceInput
			if inputVal, err := itemObj.Get("input"); err == nil && inputVal.IsObject() {
				inputObjInner, _ := inputVal.AsObject()
				input, _ = parseTraceInput(inputObjInner, ctx)
			}

			// Parse option
			var option types.TraceNodeOption
			if optionVal, err := itemObj.Get("option"); err == nil && optionVal.IsObject() {
				optionObjInner, _ := optionVal.AsObject()
				option = parseTraceNodeOption(optionObjInner)
			}

			parallelInputs = append(parallelInputs, types.TraceParallelInput{
				Input:  input,
				Option: option,
			})
		}

		// Call node.Parallel
		childNodes, err := node.Parallel(parallelInputs)
		if err != nil {
			return bridge.JsException(ctx, err.Error())
		}

		// Create array of node objects
		result := make([]interface{}, len(childNodes))
		for i, childNode := range childNodes {
			childNodeObj, err := NewNodeObject(ctx, childNode)
			if err != nil {
				continue
			}
			result[i] = childNodeObj
		}

		jsVal, err := bridge.JsValue(ctx, result)
		if err != nil {
			return bridge.JsException(ctx, err.Error())
		}

		return jsVal
	})
}

func nodeSetOutputMethod(iso *v8go.Isolate, node types.Node) *v8go.FunctionTemplate {
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

		if err := node.SetOutput(output); err != nil {
			return bridge.JsException(ctx, err.Error())
		}

		return info.This().Value
	})
}

func nodeSetMetadataMethod(iso *v8go.Isolate, node types.Node) *v8go.FunctionTemplate {
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

		if err := node.SetMetadata(key, value); err != nil {
			return bridge.JsException(ctx, err.Error())
		}

		return info.This().Value
	})
}

func nodeCompleteMethod(iso *v8go.Isolate, node types.Node) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		ctx := info.Context()
		args := info.Args()

		// Optional output parameter
		if len(args) > 0 {
			output, err := bridge.GoValue(args[0], ctx)
			if err != nil {
				return bridge.JsException(ctx, err.Error())
			}
			if err := node.Complete(output); err != nil {
				return bridge.JsException(ctx, err.Error())
			}
		} else {
			if err := node.Complete(); err != nil {
				return bridge.JsException(ctx, err.Error())
			}
		}

		return info.This().Value
	})
}

func nodeFailMethod(iso *v8go.Isolate, node types.Node) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		ctx := info.Context()
		args := info.Args()

		if len(args) < 1 {
			return bridge.JsException(ctx, "Fail requires an error message")
		}

		errMsg := args[0].String()
		if err := node.Fail(fmt.Errorf("%s", errMsg)); err != nil {
			return bridge.JsException(ctx, err.Error())
		}

		return info.This().Value
	})
}

// NewNoOpNodeObject creates a no-op Node object for when trace is not initialized
// All methods return the node itself (for chaining) and do nothing
func NewNoOpNodeObject(v8ctx *v8go.Context) (*v8go.Value, error) {
	jsObject := v8go.NewObjectTemplate(v8ctx.Isolate())
	iso := v8ctx.Isolate()

	// Set id to empty string
	jsObject.Set("id", "")

	// No-op method that returns this (for chaining)
	noOpChainMethod := func() *v8go.FunctionTemplate {
		return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
			return info.This().Value
		})
	}

	// No-op node factory for Add and Parallel methods (returns new no-op node)
	noOpNodeMethod := func() *v8go.FunctionTemplate {
		return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
			nodeObj, _ := NewNoOpNodeObject(v8ctx)
			return nodeObj
		})
	}

	// Set all methods
	jsObject.Set("Info", noOpChainMethod())
	jsObject.Set("Debug", noOpChainMethod())
	jsObject.Set("Error", noOpChainMethod())
	jsObject.Set("Warn", noOpChainMethod())
	jsObject.Set("Add", noOpNodeMethod())
	jsObject.Set("Parallel", noOpNodeMethod())
	jsObject.Set("SetOutput", noOpChainMethod())
	jsObject.Set("SetMetadata", noOpChainMethod())
	jsObject.Set("Complete", noOpChainMethod())
	jsObject.Set("Fail", noOpChainMethod())

	// Create instance
	instance, err := jsObject.NewInstance(v8ctx)
	if err != nil {
		return nil, err
	}

	return instance.Value, nil
}
