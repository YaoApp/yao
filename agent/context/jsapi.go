package context

import (
	"sync"

	"github.com/google/uuid"
	"github.com/yaoapp/gou/runtime/v8/bridge"
	"rogchap.com/v8go"
)

var objectsMutex = sync.Mutex{}
var objects = map[string]*Context{}

// JsValue return the JavaScript value of the context
func (ctx *Context) JsValue(v8ctx *v8go.Context) (*v8go.Value, error) {
	return ctx.NewObject(v8ctx)
}

// NewObject Create a new JavaScript object from the context
func (ctx *Context) NewObject(v8ctx *v8go.Context) (*v8go.Value, error) {

	jsObject := v8go.NewObjectTemplate(v8ctx.Isolate())

	// Set the id and release function
	id := ctx.objectRegister()
	jsObject.Set("__id", id)
	jsObject.Set("__release", ctx.objectRelease(v8ctx.Isolate(), id))

	// Set primitive fields in template
	jsObject.Set("chat_id", ctx.ChatID)
	jsObject.Set("assistant_id", ctx.AssistantID)
	jsObject.Set("connector", ctx.Connector)
	if ctx.Search != nil {
		jsObject.Set("search", *ctx.Search)
	}

	jsObject.Set("retry", ctx.Retry)
	jsObject.Set("retry_times", uint32(ctx.RetryTimes))
	jsObject.Set("locale", ctx.Locale)
	jsObject.Set("theme", ctx.Theme)
	jsObject.Set("referer", ctx.Referer)
	jsObject.Set("accept", string(ctx.Accept))
	jsObject.Set("route", ctx.Route)

	// Create instance
	instance, err := jsObject.NewInstance(v8ctx)
	if err != nil {
		ctx.objectRelease(v8ctx.Isolate(), id)
		return nil, err
	}

	obj, err := instance.Value.AsObject()
	if err != nil {
		ctx.objectRelease(v8ctx.Isolate(), id)
		return nil, err
	}

	// Set complex objects (maps, arrays) after instance creation using bridge
	// Args array
	if ctx.Args != nil {
		argsVal, err := bridge.JsValue(v8ctx, ctx.Args)
		if err == nil {
			obj.Set("args", argsVal)
			argsVal.Release() // Release Go-side Persistent handle, V8 internal reference remains
		}
	}

	// Client object
	clientData := map[string]interface{}{
		"type":       ctx.Client.Type,
		"user_agent": ctx.Client.UserAgent,
		"ip":         ctx.Client.IP,
	}
	clientVal, err := bridge.JsValue(v8ctx, clientData)
	if err == nil {
		obj.Set("client", clientVal)
		clientVal.Release() // Release Go-side Persistent handle, V8 internal reference remains
	}

	// Metadata object
	if ctx.Metadata != nil {
		metadataVal, err := bridge.JsValue(v8ctx, ctx.Metadata)
		if err == nil {
			obj.Set("metadata", metadataVal)
			metadataVal.Release() // Release Go-side Persistent handle, V8 internal reference remains
		}
	}

	// Authorized object
	if ctx.Authorized != nil {
		authorizedVal, err := bridge.JsValue(v8ctx, ctx.Authorized)
		if err == nil {
			obj.Set("authorized", authorizedVal)
			authorizedVal.Release() // Release Go-side Persistent handle, V8 internal reference remains
		}
	}

	return instance.Value, nil
}

func (ctx *Context) objectRegister() string {
	id := uuid.NewString()
	objectsMutex.Lock()
	defer objectsMutex.Unlock()
	objects[id] = ctx
	return id
}

func (ctx *Context) objectRelease(iso *v8go.Isolate, id string) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		objectsMutex.Lock()
		defer objectsMutex.Unlock()
		delete(objects, id)
		return v8go.Undefined(info.Context().Isolate())
	})
}
