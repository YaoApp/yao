package jsapi

import (
	"github.com/yaoapp/gou/runtime/v8/bridge"
	"github.com/yaoapp/yao/trace/types"
	"rogchap.com/v8go"
)

// NewSpaceObject creates a JavaScript Space object (pure JS object, no Go mapping)
// Since TraceSpace is a struct and not an interface, we use manager methods to operate on it
func NewSpaceObject(v8ctx *v8go.Context, manager types.Manager, space *types.TraceSpace) (*v8go.Value, error) {
	jsObject := v8go.NewObjectTemplate(v8ctx.Isolate())

	// Set primitive fields
	jsObject.Set("id", space.ID)

	// Set methods - they operate through manager
	jsObject.Set("Set", spaceSetMethod(v8ctx.Isolate(), manager, space.ID))
	jsObject.Set("Get", spaceGetMethod(v8ctx.Isolate(), manager, space.ID))
	jsObject.Set("Has", spaceHasMethod(v8ctx.Isolate(), manager, space.ID))
	jsObject.Set("Delete", spaceDeleteMethod(v8ctx.Isolate(), manager, space.ID))
	jsObject.Set("Clear", spaceClearMethod(v8ctx.Isolate(), manager, space.ID))
	jsObject.Set("Keys", spaceKeysMethod(v8ctx.Isolate(), manager, space.ID))

	// Create instance
	instance, err := jsObject.NewInstance(v8ctx)
	if err != nil {
		return nil, err
	}

	return instance.Value, nil
}

// Space method templates

func spaceSetMethod(iso *v8go.Isolate, manager types.Manager, spaceID string) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		ctx := info.Context()
		args := info.Args()

		if len(args) < 2 {
			return bridge.JsException(ctx, "Set requires 2 arguments: (key, value)")
		}

		key := args[0].String()
		value, err := bridge.GoValue(args[1], ctx)
		if err != nil {
			return bridge.JsException(ctx, err.Error())
		}

		if err := manager.SetSpaceValue(spaceID, key, value); err != nil {
			return bridge.JsException(ctx, err.Error())
		}

		return info.This().Value
	})
}

func spaceGetMethod(iso *v8go.Isolate, manager types.Manager, spaceID string) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		ctx := info.Context()
		args := info.Args()

		if len(args) < 1 {
			return bridge.JsException(ctx, "Get requires a key argument")
		}

		key := args[0].String()
		value, err := manager.GetSpaceValue(spaceID, key)
		if err != nil {
			return bridge.JsException(ctx, err.Error())
		}

		jsVal, err := bridge.JsValue(ctx, value)
		if err != nil {
			return bridge.JsException(ctx, err.Error())
		}

		return jsVal
	})
}

func spaceHasMethod(iso *v8go.Isolate, manager types.Manager, spaceID string) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		ctx := info.Context()
		args := info.Args()

		if len(args) < 1 {
			return bridge.JsException(ctx, "Has requires a key argument")
		}

		key := args[0].String()
		has := manager.HasSpaceValue(spaceID, key)

		jsVal, _ := v8go.NewValue(iso, has)
		return jsVal
	})
}

func spaceDeleteMethod(iso *v8go.Isolate, manager types.Manager, spaceID string) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		ctx := info.Context()
		args := info.Args()

		if len(args) < 1 {
			return bridge.JsException(ctx, "Delete requires a key argument")
		}

		key := args[0].String()
		if err := manager.DeleteSpaceValue(spaceID, key); err != nil {
			return bridge.JsException(ctx, err.Error())
		}

		return info.This().Value
	})
}

func spaceClearMethod(iso *v8go.Isolate, manager types.Manager, spaceID string) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		ctx := info.Context()

		if err := manager.ClearSpaceValues(spaceID); err != nil {
			return bridge.JsException(ctx, err.Error())
		}

		return info.This().Value
	})
}

func spaceKeysMethod(iso *v8go.Isolate, manager types.Manager, spaceID string) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		ctx := info.Context()

		keys := manager.ListSpaceKeys(spaceID)
		jsVal, err := bridge.JsValue(ctx, keys)
		if err != nil {
			return bridge.JsException(ctx, err.Error())
		}

		return jsVal
	})
}
