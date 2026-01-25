package context

import (
	"time"

	"github.com/yaoapp/gou/runtime/v8/bridge"
	"github.com/yaoapp/yao/agent/memory"
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

	// Set release function (both __release and Release do the same thing)
	// __release: Internal cleanup (called by GC or Use())
	// Release: Public method for manual cleanup (try-finally pattern)
	releaseFunc := ctx.objectRelease(v8ctx.Isolate(), goValueID)
	jsObject.Set("__release", releaseFunc)
	jsObject.Set("Release", releaseFunc)

	// Set primitive fields in template
	jsObject.Set("chat_id", ctx.ChatID)
	jsObject.Set("assistant_id", ctx.AssistantID)
	jsObject.Set("locale", ctx.Locale)
	jsObject.Set("theme", ctx.Theme)
	jsObject.Set("referer", ctx.Referer)
	jsObject.Set("accept", string(ctx.Accept))
	jsObject.Set("route", ctx.Route)

	// Set methods
	jsObject.Set("Send", ctx.sendMethod(v8ctx.Isolate()))
	jsObject.Set("SendStream", ctx.sendStreamMethod(v8ctx.Isolate()))
	jsObject.Set("Replace", ctx.replaceMethod(v8ctx.Isolate()))
	jsObject.Set("Append", ctx.appendMethod(v8ctx.Isolate()))
	jsObject.Set("Merge", ctx.mergeMethod(v8ctx.Isolate()))
	jsObject.Set("Set", ctx.setMethod(v8ctx.Isolate()))
	jsObject.Set("End", ctx.endMethod(v8ctx.Isolate()))

	// Set ID generator methods
	jsObject.Set("MessageID", ctx.messageIDMethod(v8ctx.Isolate()))
	jsObject.Set("BlockID", ctx.blockIDMethod(v8ctx.Isolate()))
	jsObject.Set("ThreadID", ctx.threadIDMethod(v8ctx.Isolate()))

	// Lifecycle methods
	jsObject.Set("EndBlock", ctx.endBlockMethod(v8ctx.Isolate()))

	// Set mcp object
	jsObject.Set("mcp", ctx.newMCPObject(v8ctx.Isolate()))

	// Set search object
	jsObject.Set("search", ctx.newSearchObject(v8ctx.Isolate()))

	// Set agent object for calling other agents
	jsObject.Set("agent", ctx.newAgentObject(v8ctx.Isolate()))

	// Set llm object for direct LLM calls
	jsObject.Set("llm", ctx.newLlmObject(v8ctx.Isolate()))

	// Note: Space object will be set after instance creation (requires v8ctx)

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

	// Set trace object (property, not method)
	// If trace is not initialized, use no-op object
	traceObj := ctx.createTraceObject(v8ctx)
	if traceObj != nil {
		obj.Set("trace", traceObj)
	}

	// Set complex objects (maps, arrays) after instance creation using bridge
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

	// Metadata object - always set to empty map if nil
	metadataData := ctx.Metadata
	if metadataData == nil {
		metadataData = map[string]interface{}{}
	}
	metadataVal, err := bridge.JsValue(v8ctx, metadataData)
	if err == nil {
		obj.Set("metadata", metadataVal)
		metadataVal.Release() // Release Go-side Persistent handle, V8 internal reference remains
	}

	// Authorized object - pass the complete structure
	if ctx.Authorized != nil {
		authorizedVal, err := bridge.JsValue(v8ctx, ctx.Authorized)
		if err == nil {
			obj.Set("authorized", authorizedVal)
			authorizedVal.Release() // Release Go-side Persistent handle, V8 internal reference remains
		}
	} else {
		// Set to empty object when nil
		emptyObj, err := bridge.JsValue(v8ctx, map[string]interface{}{})
		if err == nil {
			obj.Set("authorized", emptyObj)
			emptyObj.Release()
		}
	}

	// Memory object - create a JavaScript object with User/Team/Chat/Context namespaces
	if ctx.Memory != nil {
		memoryObj := ctx.createMemoryObject(v8ctx)
		obj.Set("memory", memoryObj)
		memoryObj.Release()
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
			// NOTE: We do NOT automatically release Trace object here
			//
			// Rationale:
			// 1. Each Hook execution creates a new V8 script context (scriptCtx)
			// 2. The agent Context (ctx) is passed to the Hook as a parameter
			// 3. When scriptCtx.Close() is called (via defer), V8 cleanup triggers ctx.__release()
			// 4. If we release Trace here, it gets released after EVERY Hook execution
			// 5. This causes "context canceled" errors in subsequent operations
			//
			// Trace lifecycle:
			// - Trace is created when agent.Stream() starts (in Context.Trace())
			// - Trace should persist across ALL Hook executions (Create, Next, Done)
			// - Trace is released when agent Context.Release() is called (after agent.Stream() completes)
			//
			// Memory management:
			// - If JS code explicitly calls trace.Release(), it will work (trace/jsapi/trace.go:traceGoRelease)
			// - If not explicitly called, Context.Release() will clean it up (context/context.go:Release)
			// - This is the correct lifecycle: one Context -> one Trace -> multiple Hook executions

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

// sendMethod implements ctx.Send(message, blockId?)
// Usage: const messageId = ctx.Send({ type: "text", props: { content: "Hello" } })
// Usage: const messageId = ctx.Send("Hello") // shorthand for text message
// Usage: const messageId = ctx.Send("Hello", "B1") // specify block ID
// Automatically generates MessageID and BlockID (if not specified), flushes output
// Returns: message_id (string)
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

		// Get optional blockId argument (second argument)
		// Note: message object's block_id has higher priority
		if len(args) >= 2 && args[1].IsString() && msg.BlockID == "" {
			msg.BlockID = args[1].String()
		}

		// Generate unique MessageID if not provided
		if msg.MessageID == "" {
			if ctx.IDGenerator != nil {
				msg.MessageID = ctx.IDGenerator.GenerateMessageID()
			} else {
				msg.MessageID = output.GenerateID()
			}
		}

		// Call ctx.Send (will auto-generate BlockID if still empty)
		if err := ctx.Send(msg); err != nil {
			return bridge.JsException(v8ctx, "Send failed: "+err.Error())
		}

		// Automatically flush after sending
		if err := ctx.Flush(); err != nil {
			return bridge.JsException(v8ctx, "Flush failed: "+err.Error())
		}

		// Return the message ID
		messageID, err := v8go.NewValue(iso, msg.MessageID)
		if err != nil {
			return bridge.JsException(v8ctx, "Failed to create return value: "+err.Error())
		}
		return messageID
	})
}

// sendStreamMethod implements ctx.SendStream(message)
// Usage: const msgId = ctx.SendStream({ type: "text", props: { content: "Initial content" } })
// Starts a streaming message that can be appended to with ctx.Append()
// Must be finalized with ctx.End(msgId) or ctx.End(msgId, "final content")
// Unlike Send(), this does NOT automatically send message_end event
// Returns: message_id (string)
func (ctx *Context) sendStreamMethod(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		v8ctx := info.Context()
		args := info.Args()

		if len(args) < 1 {
			return bridge.JsException(v8ctx, "SendStream requires a message argument")
		}

		// Parse message argument
		msg, err := parseMessage(v8ctx, args[0])
		if err != nil {
			return bridge.JsException(v8ctx, "invalid message: "+err.Error())
		}

		// Get optional blockId argument (second argument)
		if len(args) >= 2 && args[1].IsString() && msg.BlockID == "" {
			msg.BlockID = args[1].String()
		}

		// Call ctx.SendStream
		messageID, err := ctx.SendStream(msg)
		if err != nil {
			return bridge.JsException(v8ctx, "SendStream failed: "+err.Error())
		}

		// Automatically flush after sending
		if err := ctx.Flush(); err != nil {
			return bridge.JsException(v8ctx, "Flush failed: "+err.Error())
		}

		// Return the message ID
		returnID, err := v8go.NewValue(iso, messageID)
		if err != nil {
			return bridge.JsException(v8ctx, "Failed to create return value: "+err.Error())
		}
		return returnID
	})
}

// endMethod implements ctx.End(messageId, finalContent?)
// Usage: ctx.End(msgId) or ctx.End(msgId, "final content to append")
// Finalizes a streaming message started with SendStream()
// Sends message_end event with the complete accumulated content
// Returns: message_id (string)
func (ctx *Context) endMethod(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		v8ctx := info.Context()
		args := info.Args()

		if len(args) < 1 {
			return bridge.JsException(v8ctx, "End requires a messageId argument")
		}

		// Get message ID (first argument)
		if !args[0].IsString() {
			return bridge.JsException(v8ctx, "messageId must be a string")
		}
		messageID := args[0].String()

		// Get optional final content (second argument)
		var finalContent string
		if len(args) >= 2 && args[1].IsString() {
			finalContent = args[1].String()
		}

		// Call ctx.End
		var err error
		if finalContent != "" {
			err = ctx.End(messageID, finalContent)
		} else {
			err = ctx.End(messageID)
		}
		if err != nil {
			return bridge.JsException(v8ctx, "End failed: "+err.Error())
		}

		// Automatically flush after sending
		if err := ctx.Flush(); err != nil {
			return bridge.JsException(v8ctx, "Flush failed: "+err.Error())
		}

		// Return the message ID
		returnID, err := v8go.NewValue(iso, messageID)
		if err != nil {
			return bridge.JsException(v8ctx, "Failed to create return value: "+err.Error())
		}
		return returnID
	})
}

// replaceMethod implements ctx.Replace(messageId, message)
// Usage: ctx.Replace(messageId, { type: "text", props: { content: "Updated content" } })
// Replaces the entire message content with the specified message_id
// Automatically flushes output
// Returns: message_id (string)
func (ctx *Context) replaceMethod(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		v8ctx := info.Context()
		args := info.Args()

		// Validate arguments
		if len(args) < 2 {
			return bridge.JsException(v8ctx, "Replace requires messageId and message arguments")
		}

		// Get message ID (first argument)
		if !args[0].IsString() {
			return bridge.JsException(v8ctx, "messageId must be a string")
		}
		messageID := args[0].String()

		// Parse message argument (second argument)
		msg, err := parseMessage(v8ctx, args[1])
		if err != nil {
			return bridge.JsException(v8ctx, "invalid message: "+err.Error())
		}

		// Set message ID to the provided ID
		msg.MessageID = messageID

		// Set delta mode for replacement
		msg.Delta = true
		msg.DeltaAction = message.DeltaReplace
		msg.DeltaPath = "" // Empty path means replace entire message

		// Call ctx.Send
		if err := ctx.Send(msg); err != nil {
			return bridge.JsException(v8ctx, "Replace failed: "+err.Error())
		}

		// Automatically flush after sending
		if err := ctx.Flush(); err != nil {
			return bridge.JsException(v8ctx, "Flush failed: "+err.Error())
		}

		// Return the message ID
		returnID, err := v8go.NewValue(iso, messageID)
		if err != nil {
			return bridge.JsException(v8ctx, "Failed to create return value: "+err.Error())
		}
		return returnID
	})
}

// appendMethod implements ctx.Append(messageId, content, path?)
// Usage: ctx.Append(messageId, "more text")  // append to default content path
// Usage: ctx.Append(messageId, "more text", "props.content")  // append to specific path
// Usage: ctx.Append(messageId, { type: "text", props: { content: "more text" } })
// Usage: ctx.Append(messageId, { props: { content: "more text" } }, "props.data")  // append to custom path
// Appends content to an existing message (delta append operation)
// Automatically flushes output
// Returns: message_id (string)
func (ctx *Context) appendMethod(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		v8ctx := info.Context()
		args := info.Args()

		// Validate arguments
		if len(args) < 2 {
			return bridge.JsException(v8ctx, "Append requires messageId and content arguments")
		}

		// Get message ID (first argument)
		if !args[0].IsString() {
			return bridge.JsException(v8ctx, "messageId must be a string")
		}
		messageID := args[0].String()

		// Parse content argument (second argument)
		msg, err := parseMessage(v8ctx, args[1])
		if err != nil {
			return bridge.JsException(v8ctx, "invalid content: "+err.Error())
		}

		// Get optional path argument (third argument)
		deltaPath := ""
		if len(args) >= 3 && args[2].IsString() {
			deltaPath = args[2].String()
		}

		// Set message ID to the provided ID
		msg.MessageID = messageID

		// Set delta mode for append
		msg.Delta = true
		msg.DeltaAction = message.DeltaAppend
		msg.DeltaPath = deltaPath // Empty path means append to default content, or specify custom path

		// Call ctx.Send
		if err := ctx.Send(msg); err != nil {
			return bridge.JsException(v8ctx, "Append failed: "+err.Error())
		}

		// Automatically flush after sending
		if err := ctx.Flush(); err != nil {
			return bridge.JsException(v8ctx, "Flush failed: "+err.Error())
		}

		// Return the message ID
		returnID, err := v8go.NewValue(iso, messageID)
		if err != nil {
			return bridge.JsException(v8ctx, "Failed to create return value: "+err.Error())
		}
		return returnID
	})
}

// mergeMethod implements ctx.Merge(messageId, data, path?)
// Usage: ctx.Merge(messageId, { key: "value" })  // merge to default object path
// Usage: ctx.Merge(messageId, { status: "done" }, "props")  // merge to specific path
// Usage: ctx.Merge(messageId, { props: { status: "done", progress: 100 } })
// Merges data into an existing message object (delta merge operation)
// Automatically flushes output
// Returns: message_id (string)
func (ctx *Context) mergeMethod(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		v8ctx := info.Context()
		args := info.Args()

		// Validate arguments
		if len(args) < 2 {
			return bridge.JsException(v8ctx, "Merge requires messageId and data arguments")
		}

		// Get message ID (first argument)
		if !args[0].IsString() {
			return bridge.JsException(v8ctx, "messageId must be a string")
		}
		messageID := args[0].String()

		// Parse data argument (second argument)
		msg, err := parseMessage(v8ctx, args[1])
		if err != nil {
			return bridge.JsException(v8ctx, "invalid data: "+err.Error())
		}

		// Get optional path argument (third argument)
		deltaPath := ""
		if len(args) >= 3 && args[2].IsString() {
			deltaPath = args[2].String()
		}

		// Set message ID to the provided ID
		msg.MessageID = messageID

		// Set delta mode for merge
		msg.Delta = true
		msg.DeltaAction = message.DeltaMerge
		msg.DeltaPath = deltaPath // Empty path means merge to default object, or specify custom path

		// Call ctx.Send
		if err := ctx.Send(msg); err != nil {
			return bridge.JsException(v8ctx, "Merge failed: "+err.Error())
		}

		// Automatically flush after sending
		if err := ctx.Flush(); err != nil {
			return bridge.JsException(v8ctx, "Flush failed: "+err.Error())
		}

		// Return the message ID
		returnID, err := v8go.NewValue(iso, messageID)
		if err != nil {
			return bridge.JsException(v8ctx, "Failed to create return value: "+err.Error())
		}
		return returnID
	})
}

// setMethod implements ctx.Set(messageId, data, path)
// Usage: ctx.Set(messageId, "value", "props.newField")  // set new field at specific path
// Usage: ctx.Set(messageId, { newKey: "value" }, "props")  // set new fields in props
// Sets a new field or value in an existing message (delta set operation)
// Automatically flushes output
// Returns: message_id (string)
func (ctx *Context) setMethod(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		v8ctx := info.Context()
		args := info.Args()

		// Validate arguments (path is required for Set operation)
		if len(args) < 3 {
			return bridge.JsException(v8ctx, "Set requires messageId, data, and path arguments")
		}

		// Get message ID (first argument)
		if !args[0].IsString() {
			return bridge.JsException(v8ctx, "messageId must be a string")
		}
		messageID := args[0].String()

		// Parse data argument (second argument)
		msg, err := parseMessage(v8ctx, args[1])
		if err != nil {
			return bridge.JsException(v8ctx, "invalid data: "+err.Error())
		}

		// Get path argument (third argument - required)
		if !args[2].IsString() {
			return bridge.JsException(v8ctx, "path must be a string")
		}
		deltaPath := args[2].String()

		if deltaPath == "" {
			return bridge.JsException(v8ctx, "path cannot be empty for Set operation")
		}

		// Set message ID to the provided ID
		msg.MessageID = messageID

		// Set delta mode for set
		msg.Delta = true
		msg.DeltaAction = message.DeltaSet
		msg.DeltaPath = deltaPath // Path is required for Set operation

		// Call ctx.Send
		if err := ctx.Send(msg); err != nil {
			return bridge.JsException(v8ctx, "Set failed: "+err.Error())
		}

		// Automatically flush after sending
		if err := ctx.Flush(); err != nil {
			return bridge.JsException(v8ctx, "Flush failed: "+err.Error())
		}

		// Return the message ID
		returnID, err := v8go.NewValue(iso, messageID)
		if err != nil {
			return bridge.JsException(v8ctx, "Failed to create return value: "+err.Error())
		}
		return returnID
	})
}

// messageIDMethod implements ctx.MessageID()
// Usage: const msgId = ctx.MessageID()  // Returns: "M1", "M2", "M3"...
// Generates a unique message ID for manual message management
// Returns: message_id (string)
func (ctx *Context) messageIDMethod(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		v8ctx := info.Context()

		var messageID string
		if ctx.IDGenerator != nil {
			messageID = ctx.IDGenerator.GenerateMessageID()
		} else {
			messageID = output.GenerateID()
		}

		// Return the generated ID
		id, err := v8go.NewValue(iso, messageID)
		if err != nil {
			return bridge.JsException(v8ctx, "Failed to generate message ID: "+err.Error())
		}
		return id
	})
}

// blockIDMethod implements ctx.BlockID()
// Usage: const blockId = ctx.BlockID()  // Returns: "B1", "B2", "B3"...
// Generates a unique block ID for grouping messages
// Returns: block_id (string)
func (ctx *Context) blockIDMethod(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		v8ctx := info.Context()

		var blockID string
		if ctx.IDGenerator != nil {
			blockID = ctx.IDGenerator.GenerateBlockID()
		} else {
			blockID = output.GenerateID()
		}

		// Return the generated ID
		id, err := v8go.NewValue(iso, blockID)
		if err != nil {
			return bridge.JsException(v8ctx, "Failed to generate block ID: "+err.Error())
		}
		return id
	})
}

// threadIDMethod implements ctx.ThreadID()
// Usage: const threadId = ctx.ThreadID()  // Returns: "T1", "T2", "T3"...
// Generates a unique thread ID for concurrent operations
// Returns: thread_id (string)
func (ctx *Context) threadIDMethod(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		v8ctx := info.Context()

		var threadID string
		if ctx.IDGenerator != nil {
			threadID = ctx.IDGenerator.GenerateThreadID()
		} else {
			threadID = output.GenerateID()
		}

		// Return the generated ID
		id, err := v8go.NewValue(iso, threadID)
		if err != nil {
			return bridge.JsException(v8ctx, "Failed to generate thread ID: "+err.Error())
		}
		return id
	})
}

// endBlockMethod implements ctx.EndBlock(block_id)
// Usage: ctx.EndBlock("B1")
// Sends a block_end event for the specified block
// Returns: undefined
func (ctx *Context) endBlockMethod(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		v8ctx := info.Context()
		args := info.Args()

		// Validate arguments
		if len(args) < 1 {
			return bridge.JsException(v8ctx, "EndBlock requires block_id argument")
		}

		if !args[0].IsString() {
			return bridge.JsException(v8ctx, "block_id must be a string")
		}

		blockID := args[0].String()

		// Call ctx.EndBlock
		if err := ctx.EndBlock(blockID); err != nil {
			return bridge.JsException(v8ctx, "EndBlock failed: "+err.Error())
		}

		// Automatically flush after ending block
		if err := ctx.Flush(); err != nil {
			return bridge.JsException(v8ctx, "Flush failed: "+err.Error())
		}

		return v8go.Undefined(iso)
	})
}

// createMemoryObject creates a Memory object for JavaScript access
// Memory provides four namespaces: User, Team, Chat, Context
// Each namespace supports: Get, Set, Del, Has, Keys, Len, Clear, Incr, Decr
func (ctx *Context) createMemoryObject(v8ctx *v8go.Context) *v8go.Value {
	iso := v8ctx.Isolate()
	objTpl := v8go.NewObjectTemplate(iso)
	obj, _ := objTpl.NewInstance(v8ctx)

	// Create namespace accessors
	if ctx.Memory.User != nil {
		userObj := ctx.createNamespaceObject(v8ctx, ctx.Memory.User)
		obj.Set("user", userObj)
		userObj.Release()
	}

	if ctx.Memory.Team != nil {
		teamObj := ctx.createNamespaceObject(v8ctx, ctx.Memory.Team)
		obj.Set("team", teamObj)
		teamObj.Release()
	}

	if ctx.Memory.Chat != nil {
		chatObj := ctx.createNamespaceObject(v8ctx, ctx.Memory.Chat)
		obj.Set("chat", chatObj)
		chatObj.Release()
	}

	if ctx.Memory.Context != nil {
		contextObj := ctx.createNamespaceObject(v8ctx, ctx.Memory.Context)
		obj.Set("context", contextObj)
		contextObj.Release()
	}

	return obj.Value
}

// createNamespaceObject creates a namespace object with KV store methods
func (ctx *Context) createNamespaceObject(v8ctx *v8go.Context, ns *memory.Namespace) *v8go.Value {
	iso := v8ctx.Isolate()
	objTpl := v8go.NewObjectTemplate(iso)
	obj, _ := objTpl.NewInstance(v8ctx)

	// Get method: ns.Get(key)
	getFunc := v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		if len(info.Args()) < 1 {
			return bridge.JsException(info.Context(), "Get requires a key argument")
		}

		key := info.Args()[0].String()
		value, ok := ns.Get(key)
		if !ok {
			return v8go.Null(iso)
		}

		jsValue, err := bridge.JsValue(info.Context(), value)
		if err != nil {
			return v8go.Null(iso)
		}

		return jsValue
	})
	getFuncVal := getFunc.GetFunction(v8ctx)
	obj.Set("Get", getFuncVal.Value)

	// Set method: ns.Set(key, value, ttl?)
	setFunc := v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		if len(info.Args()) < 2 {
			return bridge.JsException(info.Context(), "Set requires key and value arguments")
		}

		key := info.Args()[0].String()
		value, err := bridge.GoValue(info.Args()[1], info.Context())
		if err != nil {
			return bridge.JsException(info.Context(), "Failed to convert value: "+err.Error())
		}

		// Optional TTL in milliseconds (third argument)
		var ttl time.Duration
		if len(info.Args()) >= 3 && info.Args()[2].IsNumber() {
			ttlMs := info.Args()[2].Integer()
			ttl = time.Duration(ttlMs) * time.Millisecond
		}

		if err := ns.Set(key, value, ttl); err != nil {
			return bridge.JsException(info.Context(), "Failed to set value: "+err.Error())
		}

		return v8go.Undefined(iso)
	})
	setFuncVal := setFunc.GetFunction(v8ctx)
	obj.Set("Set", setFuncVal.Value)

	// Del method: ns.Del(key)
	delFunc := v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		if len(info.Args()) < 1 {
			return bridge.JsException(info.Context(), "Del requires a key argument")
		}

		key := info.Args()[0].String()
		if err := ns.Del(key); err != nil {
			return bridge.JsException(info.Context(), "Failed to delete key: "+err.Error())
		}

		return v8go.Undefined(iso)
	})
	delFuncVal := delFunc.GetFunction(v8ctx)
	obj.Set("Del", delFuncVal.Value)

	// Has method: ns.Has(key)
	hasFunc := v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		if len(info.Args()) < 1 {
			return bridge.JsException(info.Context(), "Has requires a key argument")
		}

		key := info.Args()[0].String()
		exists := ns.Has(key)

		jsValue, _ := v8go.NewValue(iso, exists)
		return jsValue
	})
	hasFuncVal := hasFunc.GetFunction(v8ctx)
	obj.Set("Has", hasFuncVal.Value)

	// Keys method: ns.Keys()
	keysFunc := v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		keys := ns.Keys()
		jsValue, err := bridge.JsValue(info.Context(), keys)
		if err != nil {
			return bridge.JsException(info.Context(), "Failed to get keys: "+err.Error())
		}
		return jsValue
	})
	keysFuncVal := keysFunc.GetFunction(v8ctx)
	obj.Set("Keys", keysFuncVal.Value)

	// Len method: ns.Len()
	lenFunc := v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		length := ns.Len()
		// Use int32 for JavaScript Number (int64 becomes BigInt which is incompatible)
		jsValue, _ := v8go.NewValue(iso, int32(length))
		return jsValue
	})
	lenFuncVal := lenFunc.GetFunction(v8ctx)
	obj.Set("Len", lenFuncVal.Value)

	// Clear method: ns.Clear()
	clearFunc := v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		ns.Clear()
		return v8go.Undefined(iso)
	})
	clearFuncVal := clearFunc.GetFunction(v8ctx)
	obj.Set("Clear", clearFuncVal.Value)

	// Incr method: ns.Incr(key, delta?)
	incrFunc := v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		if len(info.Args()) < 1 {
			return bridge.JsException(info.Context(), "Incr requires a key argument")
		}

		key := info.Args()[0].String()
		delta := int64(1)
		if len(info.Args()) >= 2 && info.Args()[1].IsNumber() {
			delta = info.Args()[1].Integer()
		}

		newValue, err := ns.Incr(key, delta)
		if err != nil {
			return bridge.JsException(info.Context(), "Failed to increment: "+err.Error())
		}

		// Use int32 for JavaScript Number (int64 becomes BigInt which is incompatible with ===)
		// For counters, int32 range (-2^31 to 2^31-1) is sufficient
		jsValue, _ := v8go.NewValue(iso, int32(newValue))
		return jsValue
	})
	incrFuncVal := incrFunc.GetFunction(v8ctx)
	obj.Set("Incr", incrFuncVal.Value)

	// Decr method: ns.Decr(key, delta?)
	decrFunc := v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		if len(info.Args()) < 1 {
			return bridge.JsException(info.Context(), "Decr requires a key argument")
		}

		key := info.Args()[0].String()
		delta := int64(1)
		if len(info.Args()) >= 2 && info.Args()[1].IsNumber() {
			delta = info.Args()[1].Integer()
		}

		newValue, err := ns.Decr(key, delta)
		if err != nil {
			return bridge.JsException(info.Context(), "Failed to decrement: "+err.Error())
		}

		// Use int32 for JavaScript Number (int64 becomes BigInt which is incompatible with ===)
		jsValue, _ := v8go.NewValue(iso, int32(newValue))
		return jsValue
	})
	decrFuncVal := decrFunc.GetFunction(v8ctx)
	obj.Set("Decr", decrFuncVal.Value)

	// GetDel method: ns.GetDel(key) - Get value and delete immediately
	getDelFunc := v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		if len(info.Args()) < 1 {
			return bridge.JsException(info.Context(), "GetDel requires a key argument")
		}

		key := info.Args()[0].String()
		value, ok := ns.GetDel(key)
		if !ok {
			return v8go.Null(iso)
		}

		jsValue, err := bridge.JsValue(info.Context(), value)
		if err != nil {
			return v8go.Null(iso)
		}

		return jsValue
	})
	getDelFuncVal := getDelFunc.GetFunction(v8ctx)
	obj.Set("GetDel", getDelFuncVal.Value)

	return obj.Value
}

// sendGroupMethod implements ctx.SendGroup(group)
// Usage: ctx.SendGroup({ id: "group1", messages: [...] })
// Automatically generates IDs, sends group_start/group_end events, and flushes output
