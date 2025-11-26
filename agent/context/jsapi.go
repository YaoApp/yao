package context

import (
	"time"

	"github.com/yaoapp/gou/runtime/v8/bridge"
	"github.com/yaoapp/yao/agent/output"
	"github.com/yaoapp/yao/agent/output/message"
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

	// Set release function that will be called when JavaScript object is released
	jsObject.Set("__release", ctx.objectRelease(v8ctx.Isolate(), goValueID))

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
	jsObject.Set("Trace", ctx.traceMethod(v8ctx.Isolate()))
	jsObject.Set("Send", ctx.sendMethod(v8ctx.Isolate()))
	jsObject.Set("SendGroup", ctx.sendGroupMethod(v8ctx.Isolate()))
	jsObject.Set("SendGroupStart", ctx.sendGroupStartMethod(v8ctx.Isolate()))
	jsObject.Set("SendGroupEnd", ctx.sendGroupEndMethod(v8ctx.Isolate()))

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
func (ctx *Context) objectRelease(iso *v8go.Isolate, goValueID string) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		// Get the context object (this)
		thisObj, err := info.This().AsObject()
		if err == nil && thisObj.InternalFieldCount() > 0 {
			// Get goValueID from internal field (index 0)
			goValueID := thisObj.GetInternalField(0)
			if goValueID != nil && goValueID.IsString() {
				// Release from global bridge registry
				bridge.ReleaseGoObject(goValueID.String())
			}
		}

		return v8go.Undefined(info.Context().Isolate())
	})
}

func (ctx *Context) traceMethod(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		v8ctx := info.Context()

		// Get trace manager (lazy initialization)
		manager, err := ctx.Trace()
		if err != nil {
			return bridge.JsException(v8ctx, err.Error())
		}

		// Get trace ID
		traceID := ""
		if ctx.Stack != nil {
			traceID = ctx.Stack.TraceID
		}

		// Create JavaScript Trace object directly
		// The Trace object will be used within JavaScript and its __release will be called
		// when the JavaScript value is released via defer bridge.FreeJsValue(jsRes)
		traceObj, err := traceJsapi.NewTraceObject(v8ctx, traceID, manager)
		if err != nil {
			return bridge.JsException(v8ctx, err.Error())
		}

		return traceObj
	})
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
func (ctx *Context) sendGroupMethod(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		v8ctx := info.Context()
		args := info.Args()

		if len(args) < 1 {
			return bridge.JsException(v8ctx, "SendGroup requires a group argument")
		}

		// Parse group argument
		group, err := parseGroup(v8ctx, args[0])
		if err != nil {
			return bridge.JsException(v8ctx, "invalid group: "+err.Error())
		}

		// Generate block ID if not provided
		if group.ID == "" {
			if ctx.IDGenerator != nil {
				group.ID = ctx.IDGenerator.GenerateBlockID()
			} else {
				group.ID = output.GenerateID()
			}
		}

		// Send group_start event
		startTime := time.Now()
		startEvent := output.NewEventMessage(
			message.EventGroupStart,
			"Group started",
			message.EventMessageStartData{
				MessageID: group.ID,
				Type:      "mixed", // Mixed types in group
				Timestamp: startTime.UnixMilli(),
			},
		)
		if err := ctx.Send(startEvent); err != nil {
			return bridge.JsException(v8ctx, "Failed to send group_start event: "+err.Error())
		}
		if err := ctx.Flush(); err != nil {
			return bridge.JsException(v8ctx, "Flush failed after group_start: "+err.Error())
		}

		// Generate MessageIDs for messages and set BlockID
		for _, msg := range group.Messages {
			if msg.MessageID == "" {
				if ctx.IDGenerator != nil {
					msg.MessageID = ctx.IDGenerator.GenerateMessageID()
				} else {
					msg.MessageID = output.GenerateID()
				}
			}
			if msg.BlockID == "" {
				msg.BlockID = group.ID
			}
		}

		// Call ctx.SendGroup
		if err := ctx.SendGroup(group); err != nil {
			return bridge.JsException(v8ctx, "SendGroup failed: "+err.Error())
		}
		if err := ctx.Flush(); err != nil {
			return bridge.JsException(v8ctx, "Flush failed after SendGroup: "+err.Error())
		}

		// Send group_end event
		endEvent := output.NewEventMessage(
			message.EventGroupEnd,
			"Group completed",
			message.EventMessageEndData{
				MessageID:  group.ID,
				Type:       "mixed",
				Timestamp:  time.Now().UnixMilli(),
				DurationMs: time.Since(startTime).Milliseconds(),
				ChunkCount: len(group.Messages),
				Status:     "completed",
			},
		)
		if err := ctx.Send(endEvent); err != nil {
			return bridge.JsException(v8ctx, "Failed to send group_end event: "+err.Error())
		}
		if err := ctx.Flush(); err != nil {
			return bridge.JsException(v8ctx, "Flush failed after group_end: "+err.Error())
		}

		return v8go.Undefined(iso)
	})
}

// sendGroupStartMethod implements ctx.SendGroupStart(type?, id?)
// Usage: const groupId = ctx.SendGroupStart() // type="mixed", auto-generate ID
// Usage: const groupId = ctx.SendGroupStart("text") // type="text", auto-generate ID
// Usage: const groupId = ctx.SendGroupStart("text", "my-group-id") // type="text", use provided ID
// Returns the group ID (generated or provided)
func (ctx *Context) sendGroupStartMethod(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		v8ctx := info.Context()
		args := info.Args()

		// Get type (default: "mixed")
		groupType := "mixed"
		if len(args) > 0 && args[0].IsString() {
			groupType = args[0].String()
		}

		// Get or generate block ID
		var groupID string
		if len(args) > 1 && args[1].IsString() {
			groupID = args[1].String()
		} else {
			if ctx.IDGenerator != nil {
				groupID = ctx.IDGenerator.GenerateBlockID()
			} else {
				groupID = output.GenerateID()
			}
		}

		// Send group_start event
		startEvent := output.NewEventMessage(
			message.EventGroupStart,
			"Group started",
			message.EventMessageStartData{
				MessageID: groupID,
				Type:      groupType,
				Timestamp: time.Now().UnixMilli(),
			},
		)
		if err := ctx.Send(startEvent); err != nil {
			return bridge.JsException(v8ctx, "Failed to send group_start event: "+err.Error())
		}
		if err := ctx.Flush(); err != nil {
			return bridge.JsException(v8ctx, "Flush failed after group_start: "+err.Error())
		}

		// Return the group ID
		groupIDVal, err := v8go.NewValue(iso, groupID)
		if err != nil {
			return bridge.JsException(v8ctx, "Failed to create return value: "+err.Error())
		}
		return groupIDVal
	})
}

// sendGroupEndMethod implements ctx.SendGroupEnd(id, chunkCount?)
// Usage: ctx.SendGroupEnd(groupId)
// Usage: ctx.SendGroupEnd(groupId, 10) // With chunk count
func (ctx *Context) sendGroupEndMethod(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		v8ctx := info.Context()
		args := info.Args()

		// Group ID is required
		if len(args) < 1 || !args[0].IsString() {
			return bridge.JsException(v8ctx, "SendGroupEnd requires a group ID (string) as first argument")
		}
		groupID := args[0].String()

		// Optional chunk count
		chunkCount := 0
		if len(args) > 1 && args[1].IsNumber() {
			chunkCount = int(args[1].Integer())
		}

		// Send group_end event
		endEvent := output.NewEventMessage(
			message.EventGroupEnd,
			"Group completed",
			message.EventMessageEndData{
				MessageID:  groupID,
				Type:       "mixed",
				Timestamp:  time.Now().UnixMilli(),
				DurationMs: 0, // Duration not tracked at this level
				ChunkCount: chunkCount,
				Status:     "completed",
			},
		)
		if err := ctx.Send(endEvent); err != nil {
			return bridge.JsException(v8ctx, "Failed to send group_end event: "+err.Error())
		}
		if err := ctx.Flush(); err != nil {
			return bridge.JsException(v8ctx, "Flush failed after group_end: "+err.Error())
		}

		return v8go.Undefined(iso)
	})
}
