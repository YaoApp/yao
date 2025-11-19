package jsapi

import (
	"fmt"

	v8 "github.com/yaoapp/gou/runtime/v8"
	"github.com/yaoapp/gou/runtime/v8/bridge"
	"rogchap.com/v8go"
)

func init() {
	// Auto-register Trace JavaScript API when package is imported
	v8.RegisterFunction("Trace", ExportFunction)
}

// Usage from JavaScript:
//
//	const trace = new Trace({ driver: "local", path: "/tmp/traces" })
//	const node = trace.Add({ type: "step", content: "Processing..." }, { label: "Step 1" })
//	node.Complete({ result: "Done" })
//
// Objects:
//   - Trace: Main trace manager (constructor)
//   - Node: Individual trace node
//   - Space: Memory space for key-value storage

// ExportFunction exports the Trace constructor function template
// This is used by v8.RegisterFunction
func ExportFunction(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, traceConstructor)
}

// traceConstructor is the JavaScript constructor for Trace
// Usage: new Trace(options)
func traceConstructor(info *v8go.FunctionCallbackInfo) *v8go.Value {
	ctx := info.Context()
	args := info.Args()

	// Parse options
	options := make(map[string]interface{})
	if len(args) > 0 && !args[0].IsNullOrUndefined() {
		optionsJS, err := bridge.GoValue(args[0], ctx)
		if err != nil {
			return bridge.JsException(ctx, fmt.Sprintf("invalid options: %s", err))
		}
		if optionsMap, ok := optionsJS.(map[string]interface{}); ok {
			options = optionsMap
		}
	}

	// Create trace object
	traceObj, err := TraceNew(ctx, options)
	if err != nil {
		return bridge.JsException(ctx, err.Error())
	}

	return traceObj
}

// ExportLoadFunction exports the Trace.Load static method
// This can be used to load an existing trace by ID
func ExportLoadFunction(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, traceLoadFunction)
}

// traceLoadFunction is the JavaScript function for Trace.Load
// Usage: Trace.Load(traceID)
func traceLoadFunction(info *v8go.FunctionCallbackInfo) *v8go.Value {
	ctx := info.Context()
	args := info.Args()

	if len(args) < 1 {
		return bridge.JsException(ctx, "Load requires a trace ID")
	}

	traceID := args[0].String()
	traceObj, err := TraceLoad(ctx, traceID)
	if err != nil {
		return bridge.JsException(ctx, err.Error())
	}

	return traceObj
}
