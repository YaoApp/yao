package context

import (
	"github.com/yaoapp/gou/runtime/v8/bridge"
	"github.com/yaoapp/yao/agent/output/message"
	"rogchap.com/v8go"
)

// AgentAPI defines the agent JSAPI interface for ctx.agent.*
// This interface is defined here to avoid circular dependency between context and caller packages.
// The actual implementation is in agent/caller/jsapi.go
type AgentAPI interface {
	// Call executes a single agent call
	// Returns *caller.Result or error information
	Call(agentID string, messages []interface{}, opts map[string]interface{}) interface{}

	// Parallel agent call methods - inspired by JavaScript Promise
	// All waits for all agent calls to complete (like Promise.all)
	All(requests []interface{}) []interface{}
	// Any returns when any agent call succeeds (like Promise.any)
	Any(requests []interface{}) []interface{}
	// Race returns when any agent call completes (like Promise.race)
	Race(requests []interface{}) []interface{}
}

// AgentAPIWithCallback extends AgentAPI with callback support
// This interface provides methods that accept OnMessage handlers for real-time message processing
type AgentAPIWithCallback interface {
	AgentAPI

	// CallWithHandler executes a single agent call with an OnMessage handler
	// handler receives SSE messages: func(msg *message.Message) int
	CallWithHandler(agentID string, messages []interface{}, opts map[string]interface{}, handler OnMessageFunc) interface{}

	// AllWithHandler executes all agent calls with handlers
	// globalHandler receives messages with agentID and index: func(agentID, index, msg) int
	// Individual request handlers (if set) take precedence over globalHandler
	AllWithHandler(requests []interface{}, globalHandler BatchOnMessageFunc) []interface{}

	// AnyWithHandler executes agent calls and returns on first success, with handlers
	AnyWithHandler(requests []interface{}, globalHandler BatchOnMessageFunc) []interface{}

	// RaceWithHandler executes agent calls and returns on first completion, with handlers
	RaceWithHandler(requests []interface{}, globalHandler BatchOnMessageFunc) []interface{}
}

// BatchOnMessageFunc is the OnMessage function for batch calls
// It includes agentID and index to identify the source of each message
type BatchOnMessageFunc func(agentID string, index int, msg *message.Message) int

// AgentAPIFactory is a function type that creates an AgentAPI for a context
// This is set by the caller package during initialization
var AgentAPIFactory func(ctx *Context) AgentAPI

// Agent returns the agent API for this context
// Returns nil if AgentAPIFactory is not set
func (ctx *Context) Agent() AgentAPI {
	if AgentAPIFactory == nil {
		return nil
	}
	return AgentAPIFactory(ctx)
}

// newAgentObject creates a new agent object with all agent methods
// This is called from jsapi.go NewObject() to mount ctx.agent
func (ctx *Context) newAgentObject(iso *v8go.Isolate) *v8go.ObjectTemplate {
	agentObj := v8go.NewObjectTemplate(iso)

	// Single agent call method
	agentObj.Set("Call", ctx.agentCallMethod(iso))

	// Parallel agent call methods - inspired by JavaScript Promise
	agentObj.Set("All", ctx.agentAllMethod(iso))
	agentObj.Set("Any", ctx.agentAnyMethod(iso))
	agentObj.Set("Race", ctx.agentRaceMethod(iso))

	return agentObj
}

// agentCallMethod implements ctx.agent.Call(agentID, messages, options?)
// Usage: const result = ctx.agent.Call("assistant-id", [{ role: "user", content: "Hello" }], { connector: "gpt4", onChunk: (type, data) => 0 })
// Returns: { agent_id, response, content, error }
func (ctx *Context) agentCallMethod(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		v8ctx := info.Context()
		args := info.Args()

		// Validate arguments
		if len(args) < 2 {
			return bridge.JsException(v8ctx, "Call requires agentID and messages parameters")
		}

		// Get agent ID (first argument)
		if !args[0].IsString() {
			return bridge.JsException(v8ctx, "agentID must be a string")
		}
		agentID := args[0].String()

		// Parse messages (second argument)
		messagesVal, err := bridge.GoValue(args[1], v8ctx)
		if err != nil {
			return bridge.JsException(v8ctx, "invalid messages: "+err.Error())
		}
		messages, ok := messagesVal.([]interface{})
		if !ok {
			return bridge.JsException(v8ctx, "messages must be an array")
		}

		// Parse options (optional third argument) - extract onChunk separately
		var opts map[string]interface{}
		var onChunkFn *v8go.Function

		if len(args) >= 3 && !args[2].IsUndefined() && !args[2].IsNull() {
			optsObj, err := args[2].AsObject()
			if err == nil && optsObj != nil {
				// Extract onChunk callback before converting to Go value
				onChunkVal, _ := optsObj.Get("onChunk")
				if onChunkVal != nil && onChunkVal.IsFunction() {
					onChunkFn, _ = onChunkVal.AsFunction()
				}

				// Convert the rest of options to Go map
				goVal, err := bridge.GoValue(args[2], v8ctx)
				if err == nil {
					if optsMap, ok := goVal.(map[string]interface{}); ok {
						// Remove onChunk from the map (it's handled separately)
						delete(optsMap, "onChunk")
						opts = optsMap
					}
				}
			}
		}

		// Get agent API
		agentAPI := ctx.Agent()
		if agentAPI == nil {
			return bridge.JsException(v8ctx, "agent API not available")
		}

		var result interface{}

		// If onChunk callback is provided and API supports it, use CallWithHandler
		if onChunkFn != nil {
			if apiWithCb, ok := agentAPI.(AgentAPIWithCallback); ok {
				// Create Go StreamFunc that calls JS callback
				handler := createJSStreamHandler(v8ctx, onChunkFn)
				result = apiWithCb.CallWithHandler(agentID, messages, opts, handler)
			} else {
				// Fallback: ignore callback if API doesn't support it
				result = agentAPI.Call(agentID, messages, opts)
			}
		} else {
			// No callback, use regular Call
			result = agentAPI.Call(agentID, messages, opts)
		}

		// Convert result to JS value
		jsVal, err := bridge.JsValue(v8ctx, result)
		if err != nil {
			return bridge.JsException(v8ctx, "failed to convert result: "+err.Error())
		}

		return jsVal
	})
}

// createJSOnMessageHandler creates a Go OnMessageFunc that calls a JS callback
// JS callback signature: (msg: object) => number
// msg contains: type, props, delta, message_id, chunk_id, etc.
func createJSStreamHandler(v8ctx *v8go.Context, callback *v8go.Function) OnMessageFunc {
	return func(msg *message.Message) int {
		if callback == nil || v8ctx == nil || msg == nil {
			return 0 // Continue if no callback
		}

		// Convert message to JS value
		jsMsg, err := bridge.JsValue(v8ctx, msg)
		if err != nil {
			return 1 // Stop on error
		}

		// Call the JS callback with the message object
		result, err := callback.Call(v8ctx.Global(), jsMsg)
		if err != nil {
			return 1 // Stop on error
		}

		// Check return value (0 = continue, non-zero = stop)
		if result != nil && result.IsNumber() {
			ret := result.Integer()
			if ret != 0 {
				return int(ret)
			}
		}

		return 0 // Continue
	}
}

// agentAllMethod implements ctx.agent.All(requests, options?)
// Waits for all agent calls to complete (like Promise.all)
// Each request should have:
//   - agent: string - target agent ID
//   - messages: array - messages to send
//   - options?: object - call options
//
// Global options (second argument):
//   - onChunk?: (agentID, index, msg) => number - callback for all messages (uses channel for V8 safety)
func (ctx *Context) agentAllMethod(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		v8ctx := info.Context()
		args := info.Args()

		// Validate arguments
		if len(args) < 1 {
			return bridge.JsException(v8ctx, "All requires requests parameter")
		}

		// Parse requests and extract global callback
		requests, globalCallback := ctx.parseRequestsForBatch(args, v8ctx)

		// Get agent API
		agentAPI := ctx.Agent()
		if agentAPI == nil {
			return bridge.JsException(v8ctx, "agent API not available")
		}

		// Execute with channel-based callback handling
		results := ctx.executeBatchWithCallback(BatchMethodAll, requests, globalCallback, v8ctx)

		// Convert results to JS value
		jsVal, err := bridge.JsValue(v8ctx, results)
		if err != nil {
			return bridge.JsException(v8ctx, "failed to convert results: "+err.Error())
		}

		return jsVal
	})
}

// agentAnyMethod implements ctx.agent.Any(requests, options?)
// Returns when any agent call succeeds (like Promise.any)
// Each request should have:
//   - agent: string - target agent ID
//   - messages: array - messages to send
//   - options?: object - call options
//
// Global options (second argument):
//   - onChunk?: (agentID, index, msg) => number - callback for all messages (uses channel for V8 safety)
func (ctx *Context) agentAnyMethod(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		v8ctx := info.Context()
		args := info.Args()

		// Validate arguments
		if len(args) < 1 {
			return bridge.JsException(v8ctx, "Any requires requests parameter")
		}

		// Parse requests and extract global callback
		requests, globalCallback := ctx.parseRequestsForBatch(args, v8ctx)

		// Get agent API
		agentAPI := ctx.Agent()
		if agentAPI == nil {
			return bridge.JsException(v8ctx, "agent API not available")
		}

		// Execute with channel-based callback handling
		results := ctx.executeBatchWithCallback(BatchMethodAny, requests, globalCallback, v8ctx)

		// Convert results to JS value
		jsVal, err := bridge.JsValue(v8ctx, results)
		if err != nil {
			return bridge.JsException(v8ctx, "failed to convert results: "+err.Error())
		}

		return jsVal
	})
}

// agentRaceMethod implements ctx.agent.Race(requests, options?)
// Returns when any agent call completes (like Promise.race)
// Each request should have:
//   - agent: string - target agent ID
//   - messages: array - messages to send
//   - options?: object - call options
//
// Global options (second argument):
//   - onChunk?: (agentID, index, msg) => number - callback for all messages (uses channel for V8 safety)
func (ctx *Context) agentRaceMethod(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		v8ctx := info.Context()
		args := info.Args()

		// Validate arguments
		if len(args) < 1 {
			return bridge.JsException(v8ctx, "Race requires requests parameter")
		}

		// Parse requests and extract global callback
		requests, globalCallback := ctx.parseRequestsForBatch(args, v8ctx)

		// Get agent API
		agentAPI := ctx.Agent()
		if agentAPI == nil {
			return bridge.JsException(v8ctx, "agent API not available")
		}

		// Execute with channel-based callback handling
		results := ctx.executeBatchWithCallback(BatchMethodRace, requests, globalCallback, v8ctx)

		// Convert results to JS value
		jsVal, err := bridge.JsValue(v8ctx, results)
		if err != nil {
			return bridge.JsException(v8ctx, "failed to convert results: "+err.Error())
		}

		return jsVal
	})
}

// batchMessage represents a message from a batch call for channel-based callback handling
type batchMessage struct {
	AgentID string           // Agent ID that generated this message
	Index   int              // Index of the request in the batch
	Message *message.Message // The message object
}

// parseRequestsForBatch parses the requests array and extracts global callback for batch calls
// Returns the requests array and the global JS callback function (if any)
func (ctx *Context) parseRequestsForBatch(args []*v8go.Value, v8ctx *v8go.Context) ([]interface{}, *v8go.Function) {
	var globalCallback *v8go.Function

	// Parse global options (second argument) for global onChunk
	if len(args) >= 2 && !args[1].IsUndefined() && !args[1].IsNull() {
		globalOptsObj, err := args[1].AsObject()
		if err == nil && globalOptsObj != nil {
			onChunkVal, _ := globalOptsObj.Get("onChunk")
			if onChunkVal != nil && onChunkVal.IsFunction() {
				globalCallback, _ = onChunkVal.AsFunction()
			}
		}
	}

	// Parse requests array
	if len(args) < 1 || args[0].IsUndefined() || args[0].IsNull() {
		return []interface{}{}, globalCallback
	}

	requestsObj, err := args[0].AsObject()
	if err != nil {
		return []interface{}{}, globalCallback
	}

	// Get array length
	lengthVal, err := requestsObj.Get("length")
	if err != nil {
		return []interface{}{}, globalCallback
	}

	length := int(lengthVal.Integer())
	requests := make([]interface{}, 0, length)

	for i := 0; i < length; i++ {
		itemVal, err := requestsObj.GetIdx(uint32(i))
		if err != nil || itemVal.IsUndefined() || itemVal.IsNull() {
			continue
		}

		// Convert to Go map
		goVal, err := bridge.GoValue(itemVal, v8ctx)
		if err != nil {
			continue
		}

		reqMap, ok := goVal.(map[string]interface{})
		if !ok {
			continue
		}

		// Remove onChunk from per-request options (only global callback is supported)
		if opts, ok := reqMap["options"].(map[string]interface{}); ok {
			delete(opts, "onChunk")
		}

		requests = append(requests, reqMap)
	}

	return requests, globalCallback
}

// BatchMethod represents the type of batch operation
type BatchMethod int

const (
	BatchMethodAll BatchMethod = iota
	BatchMethodAny
	BatchMethodRace
)

// executeBatchWithCallback executes a batch operation with channel-based callback handling
// This ensures V8 thread safety by processing all callbacks in the main goroutine
func (ctx *Context) executeBatchWithCallback(
	method BatchMethod,
	requests []interface{},
	callback *v8go.Function,
	v8ctx *v8go.Context,
) []interface{} {
	// Get agent API
	agentAPI := ctx.Agent()
	if agentAPI == nil {
		return []interface{}{}
	}

	// If no callback, just execute directly
	if callback == nil {
		switch method {
		case BatchMethodAll:
			return agentAPI.All(requests)
		case BatchMethodAny:
			return agentAPI.Any(requests)
		case BatchMethodRace:
			return agentAPI.Race(requests)
		}
		return []interface{}{}
	}

	// Check if API supports callbacks
	apiWithCb, ok := agentAPI.(AgentAPIWithCallback)
	if !ok {
		switch method {
		case BatchMethodAll:
			return agentAPI.All(requests)
		case BatchMethodAny:
			return agentAPI.Any(requests)
		case BatchMethodRace:
			return agentAPI.Race(requests)
		}
		return []interface{}{}
	}

	// Create message channel for callback handling
	// Use a large buffer (1000) to reduce blocking, with blocking send to guarantee no message loss
	msgChan := make(chan batchMessage, 1000)
	doneChan := make(chan []interface{}, 1)

	// Create Go handler that sends messages to channel
	// Blocking send ensures no message is lost (natural backpressure)
	goHandler := func(agentID string, index int, msg *message.Message) int {
		msgChan <- batchMessage{AgentID: agentID, Index: index, Message: msg}
		return 0
	}

	// Start batch execution in background goroutine
	go func() {
		defer close(msgChan)
		var results []interface{}

		switch method {
		case BatchMethodAll:
			results = apiWithCb.AllWithHandler(requests, goHandler)
		case BatchMethodAny:
			results = apiWithCb.AnyWithHandler(requests, goHandler)
		case BatchMethodRace:
			results = apiWithCb.RaceWithHandler(requests, goHandler)
		}

		doneChan <- results
	}()

	// Process messages in main goroutine (V8 thread-safe)
	for msg := range msgChan {
		callJSBatchCallback(v8ctx, callback, msg.AgentID, msg.Index, msg.Message)
	}

	// Wait for results
	return <-doneChan
}

// callJSBatchCallback calls the JS callback with batch message parameters
// Must be called from the main V8 goroutine
func callJSBatchCallback(v8ctx *v8go.Context, callback *v8go.Function, agentID string, index int, msg *message.Message) {
	if callback == nil || v8ctx == nil || msg == nil {
		return
	}

	iso := v8ctx.Isolate()

	agentIDVal, err := v8go.NewValue(iso, agentID)
	if err != nil {
		return
	}

	indexVal, err := v8go.NewValue(iso, int32(index))
	if err != nil {
		return
	}

	// Convert message to JS value
	jsMsg, err := bridge.JsValue(v8ctx, msg)
	if err != nil {
		return
	}

	callback.Call(v8ctx.Global(), agentIDVal, indexVal, jsMsg)
}
