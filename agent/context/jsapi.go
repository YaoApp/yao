package context

import (
	"github.com/yaoapp/gou/runtime/v8/bridge"
	"github.com/yaoapp/yao/agent/output"
	traceJsapi "github.com/yaoapp/yao/trace/jsapi"
	"rogchap.com/v8go"
)

// JsValue return the JavaScript value of the context
func (ctx *Context) JsValue(v8ctx *v8go.Context) (*v8go.Value, error) {
	return ctx.NewObject(v8ctx)
}

// NewObject Create a new JavaScript object from the context
func (ctx *Context) NewObject(v8ctx *v8go.Context) (*v8go.Value, error) {

	jsObject := v8go.NewObjectTemplate(v8ctx.Isolate())

	// Set internal field count to 1 to store the goValueID
	// Internal fields are not accessible from JavaScript, providing better security
	jsObject.SetInternalFieldCount(1)

	// Register context in global bridge registry for efficient Go object retrieval
	// The goValueID will be stored in internal field (index 0) after instance creation
	goValueID := bridge.RegisterGoObject(ctx)

	// Set release function (both __release and Release do the same thing)
	// __release: Internal cleanup (called by GC or Use())
	// Release: Public method for manual cleanup (try-finally pattern)
	releaseFunc := ctx.objectRelease(v8ctx.Isolate(), goValueID)
	jsObject.Set("__release", releaseFunc)
	jsObject.Set("Release", releaseFunc)

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

	// Set methods
	jsObject.Set("Send", ctx.sendMethod(v8ctx.Isolate()))

	// Set MCP object
	jsObject.Set("MCP", ctx.newMCPObject(v8ctx.Isolate()))

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

	// Set Trace object (property, not method)
	// If trace is not initialized, use no-op object
	traceObj := ctx.createTraceObject(v8ctx)
	if traceObj != nil {
		obj.Set("Trace", traceObj)
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

// objectRelease releases the Go object from the global bridge registry
// It retrieves the goValueID from internal field (index 0) and releases the Go object
// Also releases associated Trace object if present
func (ctx *Context) objectRelease(iso *v8go.Isolate, goValueID string) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		// Get the context object (this)
		thisObj, err := info.This().AsObject()
		if err == nil {
			// Release Trace object if it has __release method
			if traceVal, err := thisObj.Get("Trace"); err == nil && !traceVal.IsNullOrUndefined() {
				if traceObj, err := traceVal.AsObject(); err == nil {
					if releaseFunc, err := traceObj.Get("__release"); err == nil && releaseFunc.IsFunction() {
						// Call Trace.__release() to cleanup trace resources
						if releaseFn, err := releaseFunc.AsFunction(); err == nil {
							releaseFn.Call(traceObj.Value) // Ignore errors in cleanup
						}
					}
				}
			}

			// Release Context Go object from bridge registry
			if thisObj.InternalFieldCount() > 0 {
				// Get goValueID from internal field (index 0)
				goValueID := thisObj.GetInternalField(0)
				if goValueID != nil && goValueID.IsString() {
					// Release from global bridge registry
					bridge.ReleaseGoObject(goValueID.String())
				}
			}
		}

		return v8go.Undefined(info.Context().Isolate())
	})
}

// createTraceObject creates a Trace object instance
// Returns a no-op Trace object if trace is not initialized
func (ctx *Context) createTraceObject(v8ctx *v8go.Context) *v8go.Value {
	// Try to get trace manager
	manager, err := ctx.Trace()
	if err != nil || manager == nil {
		// Return no-op trace object if initialization fails
		noOpTrace, _ := traceJsapi.NewNoOpTraceObject(v8ctx)
		return noOpTrace
	}

	// Get trace ID
	traceID := ""
	if ctx.Stack != nil {
		traceID = ctx.Stack.TraceID
	}

	// Create JavaScript Trace object
	traceObj, err := traceJsapi.NewTraceObject(v8ctx, traceID, manager)
	if err != nil {
		// Return no-op trace object if creation fails
		noOpTrace, _ := traceJsapi.NewNoOpTraceObject(v8ctx)
		return noOpTrace
	}

	return traceObj
}

// sendMethod implements ctx.Send(message)
// Usage: ctx.Send({ type: "text", props: { content: "Hello" } })
// Usage: ctx.Send("Hello") // shorthand for text message
// Automatically generates ID and flushes output
func (ctx *Context) sendMethod(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		v8ctx := info.Context()
		args := info.Args()

		if len(args) < 1 {
			return bridge.JsException(v8ctx, "Send requires a message argument")
		}

		// Parse message argument
		msg, err := parseMessage(v8ctx, args[0])
		if err != nil {
			return bridge.JsException(v8ctx, "invalid message: "+err.Error())
		}

		// Generate unique MessageID if not provided
		if msg.MessageID == "" {
			if ctx.IDGenerator != nil {
				msg.MessageID = ctx.IDGenerator.GenerateMessageID()
			} else {
				msg.MessageID = output.GenerateID()
			}
		}

		// Call ctx.Send
		if err := ctx.Send(msg); err != nil {
			return bridge.JsException(v8ctx, "Send failed: "+err.Error())
		}

		// Automatically flush after sending
		if err := ctx.Flush(); err != nil {
			return bridge.JsException(v8ctx, "Flush failed: "+err.Error())
		}

		return v8go.Undefined(iso)
	})
}

// sendGroupMethod implements ctx.SendGroup(group)
// Usage: ctx.SendGroup({ id: "group1", messages: [...] })
// Automatically generates IDs, sends group_start/group_end events, and flushes output
