package context

import (
	"rogchap.com/v8go"
)

// JsValue return the JavaScript value of the context
func (ctx *Context) JsValue(v8ctx *v8go.Context) (*v8go.Value, error) {
	return ctx.NewObject(v8ctx)
}

// NewObject Create a new JavaScript object from the context
func (ctx *Context) NewObject(v8ctx *v8go.Context) (*v8go.Value, error) {
	jsObject := v8go.NewObjectTemplate(v8ctx.Isolate())
	jsObject.Set("ChatID", ctx.ChatID)
	jsObject.Set("AssistantID", ctx.AssistantID)
	jsObject.Set("Sid", ctx.Sid)
	instance, err := jsObject.NewInstance(v8ctx)
	if err != nil {
		return nil, err
	}
	return instance.Value, nil
}
