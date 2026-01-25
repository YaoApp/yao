package context

import (
	"github.com/yaoapp/gou/runtime/v8/bridge"
	"github.com/yaoapp/yao/agent/output/message"
	"rogchap.com/v8go"
)

// LlmAPI defines the LLM JSAPI interface for ctx.llm.*
// This interface is defined here to avoid circular dependency between context and llm packages.
// The actual implementation is in agent/llm/jsapi.go
type LlmAPI interface {
	// Stream calls LLM with streaming output to ctx.Writer
	// Returns *llm.Result or error information
	Stream(connector string, messages []interface{}, opts map[string]interface{}) interface{}

	// Parallel LLM call methods - inspired by JavaScript Promise
	// All waits for all LLM calls to complete (like Promise.all)
	All(requests []interface{}) []interface{}
	// Any returns when any LLM call succeeds (like Promise.any)
	Any(requests []interface{}) []interface{}
	// Race returns when any LLM call completes (like Promise.race)
	Race(requests []interface{}) []interface{}
}

// LlmAPIWithCallback extends LlmAPI with callback support
// This interface provides methods that accept OnMessage handlers for real-time message processing
type LlmAPIWithCallback interface {
	LlmAPI

	// StreamWithHandler calls LLM with an OnMessage handler
	// handler receives SSE messages: func(msg *message.Message) int
	StreamWithHandler(connector string, messages []interface{}, opts map[string]interface{}, handler OnMessageFunc) interface{}

	// AllWithHandler executes all LLM calls with handlers
	// globalHandler receives messages with connectorID and index: func(connectorID, index, msg) int
	AllWithHandler(requests []interface{}, globalHandler LlmBatchOnMessageFunc) []interface{}

	// AnyWithHandler executes LLM calls and returns on first success, with handlers
	AnyWithHandler(requests []interface{}, globalHandler LlmBatchOnMessageFunc) []interface{}

	// RaceWithHandler executes LLM calls and returns on first completion, with handlers
	RaceWithHandler(requests []interface{}, globalHandler LlmBatchOnMessageFunc) []interface{}
}

// LlmBatchOnMessageFunc is the OnMessage function for batch LLM calls
// It includes connectorID and index to identify the source of each message
type LlmBatchOnMessageFunc func(connectorID string, index int, msg *message.Message) int

// LlmAPIFactory is a function type that creates a LlmAPI for a context
// This is set by the llm package during initialization
var LlmAPIFactory func(ctx *Context) LlmAPI

// Llm returns the LLM API for this context
// Returns nil if LlmAPIFactory is not set
func (ctx *Context) Llm() LlmAPI {
	if LlmAPIFactory == nil {
		return nil
	}
	return LlmAPIFactory(ctx)
}

// newLlmObject creates a new llm object with all llm methods
// This is called from jsapi.go NewObject() to mount ctx.llm
func (ctx *Context) newLlmObject(iso *v8go.Isolate) *v8go.ObjectTemplate {
	llmObj := v8go.NewObjectTemplate(iso)

	// Single LLM call method
	llmObj.Set("Stream", ctx.llmStreamMethod(iso))

	// Parallel LLM call methods - inspired by JavaScript Promise
	llmObj.Set("All", ctx.llmAllMethod(iso))
	llmObj.Set("Any", ctx.llmAnyMethod(iso))
	llmObj.Set("Race", ctx.llmRaceMethod(iso))

	return llmObj
}

// llmStreamMethod implements ctx.llm.Stream(connector, messages, options?)
// Usage: const result = ctx.llm.Stream("gpt-4o", [{ role: "user", content: "Hello" }], { temperature: 0.7, onChunk: (msg) => 0 })
// Returns: { connector, response, content, error }
func (ctx *Context) llmStreamMethod(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		v8ctx := info.Context()
		args := info.Args()

		// Validate arguments
		if len(args) < 2 {
			return bridge.JsException(v8ctx, "Stream requires connector and messages parameters")
		}

		// Get connector ID (first argument)
		if !args[0].IsString() {
			return bridge.JsException(v8ctx, "connector must be a string")
		}
		connector := args[0].String()

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

		// Get LLM API
		llmAPI := ctx.Llm()
		if llmAPI == nil {
			return bridge.JsException(v8ctx, "LLM API not available")
		}

		var result interface{}

		// If onChunk callback is provided and API supports it, use StreamWithHandler
		if onChunkFn != nil {
			if apiWithCb, ok := llmAPI.(LlmAPIWithCallback); ok {
				// Create Go OnMessageFunc that calls JS callback
				handler := createJSStreamHandler(v8ctx, onChunkFn)
				result = apiWithCb.StreamWithHandler(connector, messages, opts, handler)
			} else {
				// Fallback: ignore callback if API doesn't support it
				result = llmAPI.Stream(connector, messages, opts)
			}
		} else {
			// No callback, use regular Stream
			result = llmAPI.Stream(connector, messages, opts)
		}

		// Convert result to JS value
		jsVal, err := bridge.JsValue(v8ctx, result)
		if err != nil {
			return bridge.JsException(v8ctx, "failed to convert result: "+err.Error())
		}

		return jsVal
	})
}

// llmAllMethod implements ctx.llm.All(requests, options?)
// Usage: const results = ctx.llm.All([
//
//	{ connector: "gpt-4o", messages: [...], options: {...} },
//	{ connector: "claude-3", messages: [...] }
//
// ], { onChunk: (connectorID, index, msg) => 0 })
// Returns: [{ connector, response, content, error }, ...]
func (ctx *Context) llmAllMethod(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		return ctx.executeLlmBatchMethod(info, LlmBatchMethodAll)
	})
}

// llmAnyMethod implements ctx.llm.Any(requests, options?)
// Returns first successful result
func (ctx *Context) llmAnyMethod(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		return ctx.executeLlmBatchMethod(info, LlmBatchMethodAny)
	})
}

// llmRaceMethod implements ctx.llm.Race(requests, options?)
// Returns first completed result (success or failure)
func (ctx *Context) llmRaceMethod(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		return ctx.executeLlmBatchMethod(info, LlmBatchMethodRace)
	})
}

// LlmBatchMethod represents the type of batch LLM operation
type LlmBatchMethod int

const (
	LlmBatchMethodAll LlmBatchMethod = iota
	LlmBatchMethodAny
	LlmBatchMethodRace
)

// executeLlmBatchMethod handles All/Any/Race batch LLM calls
func (ctx *Context) executeLlmBatchMethod(info *v8go.FunctionCallbackInfo, method LlmBatchMethod) *v8go.Value {
	v8ctx := info.Context()
	args := info.Args()

	// Validate arguments
	if len(args) < 1 {
		return bridge.JsException(v8ctx, "requires requests array parameter")
	}

	// Parse requests array (first argument)
	requestsVal, err := bridge.GoValue(args[0], v8ctx)
	if err != nil {
		return bridge.JsException(v8ctx, "invalid requests: "+err.Error())
	}
	requests, ok := requestsVal.([]interface{})
	if !ok {
		return bridge.JsException(v8ctx, "requests must be an array")
	}

	// Get LLM API
	llmAPI := ctx.Llm()
	if llmAPI == nil {
		return bridge.JsException(v8ctx, "LLM API not available")
	}

	// Parse optional global callback from second argument (options object)
	var globalCallback *v8go.Function
	if len(args) >= 2 && !args[1].IsUndefined() && !args[1].IsNull() {
		optsObj, err := args[1].AsObject()
		if err == nil && optsObj != nil {
			onChunkVal, _ := optsObj.Get("onChunk")
			if onChunkVal != nil && onChunkVal.IsFunction() {
				globalCallback, _ = onChunkVal.AsFunction()
			}
		}
	}

	var results []interface{}

	// If callback is provided and API supports it, use channel-based execution
	if globalCallback != nil {
		if apiWithCb, ok := llmAPI.(LlmAPIWithCallback); ok {
			results = ctx.executeLlmBatchWithCallback(method, requests, globalCallback, v8ctx, apiWithCb)
		} else {
			// Fallback: execute without callback
			results = ctx.executeLlmBatchWithoutCallback(method, requests, llmAPI)
		}
	} else {
		// No callback, use regular batch methods
		results = ctx.executeLlmBatchWithoutCallback(method, requests, llmAPI)
	}

	// Convert results to JS value
	jsVal, err := bridge.JsValue(v8ctx, results)
	if err != nil {
		return bridge.JsException(v8ctx, "failed to convert results: "+err.Error())
	}

	return jsVal
}

// executeLlmBatchWithoutCallback executes batch LLM calls without callback
func (ctx *Context) executeLlmBatchWithoutCallback(method LlmBatchMethod, requests []interface{}, llmAPI LlmAPI) []interface{} {
	switch method {
	case LlmBatchMethodAll:
		return llmAPI.All(requests)
	case LlmBatchMethodAny:
		return llmAPI.Any(requests)
	case LlmBatchMethodRace:
		return llmAPI.Race(requests)
	default:
		return llmAPI.All(requests)
	}
}

// llmBatchMessage is used for channel communication in batch LLM calls
type llmBatchMessage struct {
	ConnectorID string
	Index       int
	Message     *message.Message
}

// executeLlmBatchWithCallback executes batch LLM calls with callback using channel
// This ensures V8 thread safety by serializing callback invocations
func (ctx *Context) executeLlmBatchWithCallback(method LlmBatchMethod, requests []interface{}, callback *v8go.Function, v8ctx *v8go.Context, apiWithCb LlmAPIWithCallback) []interface{} {
	// Create a buffered channel for messages
	// Using blocking send to ensure all messages are delivered
	msgChan := make(chan llmBatchMessage, 1000)
	doneChan := make(chan []interface{}, 1)

	// Create Go handler that sends to channel
	goHandler := func(connectorID string, index int, msg *message.Message) int {
		msgChan <- llmBatchMessage{
			ConnectorID: connectorID,
			Index:       index,
			Message:     msg,
		}
		return 0
	}

	// Execute batch calls in background goroutine
	go func() {
		defer close(msgChan)

		var results []interface{}
		switch method {
		case LlmBatchMethodAll:
			results = apiWithCb.AllWithHandler(requests, goHandler)
		case LlmBatchMethodAny:
			results = apiWithCb.AnyWithHandler(requests, goHandler)
		case LlmBatchMethodRace:
			results = apiWithCb.RaceWithHandler(requests, goHandler)
		default:
			results = apiWithCb.AllWithHandler(requests, goHandler)
		}
		doneChan <- results
	}()

	// Process messages in main goroutine (V8 thread)
	for msg := range msgChan {
		callJSLlmBatchCallback(v8ctx, callback, msg.ConnectorID, msg.Index, msg.Message)
	}

	// Wait for results
	return <-doneChan
}

// callJSLlmBatchCallback calls the JS callback function for batch LLM calls
func callJSLlmBatchCallback(v8ctx *v8go.Context, callback *v8go.Function, connectorID string, index int, msg *message.Message) {
	if callback == nil || v8ctx == nil {
		return
	}

	iso := v8ctx.Isolate()

	// Create arguments: connectorID, index, message
	connectorVal, err := v8go.NewValue(iso, connectorID)
	if err != nil {
		return
	}

	indexVal, err := v8go.NewValue(iso, int32(index))
	if err != nil {
		return
	}

	// Convert message to JS object
	msgVal, err := bridge.JsValue(v8ctx, msg)
	if err != nil {
		return
	}

	// Call the callback
	_, _ = callback.Call(v8go.Undefined(iso), connectorVal, indexVal, msgVal)
}
