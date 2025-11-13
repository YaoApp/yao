package context

import (
	"sync"

	"github.com/google/uuid"
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

	id := uuid.NewString()
	ctx.objectRegister(id)

	// Set the id and release function
	jsObject.Set("__id", id)
	jsObject.Set("__release", ctx.objectRelease(v8ctx.Isolate(), id))

	jsObject.Set("ChatID", ctx.ChatID)
	jsObject.Set("AssistantID", ctx.AssistantID)
	jsObject.Set("Sid", ctx.Sid)
	instance, err := jsObject.NewInstance(v8ctx)
	if err != nil {
		return nil, err
	}
	return instance.Value, nil
}

func (ctx *Context) objectRegister(id string) {
	objectsMutex.Lock()
	defer objectsMutex.Unlock()
	objects[id] = ctx
}

func (ctx *Context) objectRelease(iso *v8go.Isolate, id string) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		objectsMutex.Lock()
		defer objectsMutex.Unlock()
		delete(objects, id)
		return v8go.Undefined(info.Context().Isolate())
	})
}
