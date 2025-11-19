package jsapi

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	v8 "github.com/yaoapp/gou/runtime/v8"
	"github.com/yaoapp/gou/runtime/v8/bridge"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/test"
	"github.com/yaoapp/yao/trace/types"
	"rogchap.com/v8go"
)

// TestTraceNew test creating a new trace from JavaScript
func TestTraceNew(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	v8.RegisterFunction("testTraceNew", testTraceNewEmbed)
	res, err := v8.Call(v8.CallOptions{}, `
		function test() {
			const trace = new Trace({ driver: "local", path: "/tmp/test-traces" })
			return testTraceNew(trace)
		}`)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	// With Internal Field + goMaps, res should now be the manager directly
	manager, ok := res.(types.Manager)
	if !ok {
		t.Fatalf("Expected types.Manager, got %T", res)
	}

	assert.NotNil(t, manager, "manager should not be nil")

	// After v8.Call returns, the jsRes should have been released via defer bridge.FreeJsValue(jsRes)
	// This should have triggered __release() on the trace object, cleaning up the goMaps
	// Note: We can't directly check goMaps as it's in the bridge package
}

func testTraceNewEmbed(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, testTraceNewFunction)
}

func testTraceNewFunction(info *v8go.FunctionCallbackInfo) *v8go.Value {
	args := info.Args()
	if len(args) < 1 {
		return bridge.JsException(info.Context(), "Missing parameters")
	}

	trace, err := args[0].AsObject()
	if err != nil {
		return bridge.JsException(info.Context(), err)
	}

	// Verify the trace has id field
	traceID, err := trace.Get("id")
	if err != nil {
		return bridge.JsException(info.Context(), err)
	}

	if !traceID.IsString() {
		return bridge.JsException(info.Context(), fmt.Errorf("id should be a string"))
	}

	// Verify the trace has __release function
	release, err := trace.Get("__release")
	if err != nil {
		return bridge.JsException(info.Context(), err)
	}

	if !release.IsFunction() {
		return bridge.JsException(info.Context(), fmt.Errorf("__release should be a function"))
	}

	// Return the trace object itself so its __release will be called when v8.Call finishes
	return args[0]
}

// TestTraceAddNode test adding nodes to trace
func TestTraceAddNode(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	v8.RegisterFunction("testTraceAddNode", testTraceAddNodeEmbed)
	res, err := v8.Call(v8.CallOptions{}, `
		function test() {
			const trace = new Trace({ driver: "local", path: "/tmp/test-traces" })
			const node = trace.Add({ type: "step", content: "Test step" }, { label: "Step 1" })
			// testTraceAddNode returns the trace object itself
			return testTraceAddNode(trace, node)
		}`)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	// With Internal Field + goMaps, res should now be the manager directly
	manager, ok := res.(types.Manager)
	if !ok {
		t.Fatalf("Expected types.Manager, got %T", res)
	}

	assert.NotNil(t, manager, "manager should not be nil")
	// Note: We can't directly check goMaps cleanup as it's in the bridge package
}

func testTraceAddNodeEmbed(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, testTraceAddNodeFunction)
}

func testTraceAddNodeFunction(info *v8go.FunctionCallbackInfo) *v8go.Value {
	args := info.Args()
	if len(args) < 2 {
		return bridge.JsException(info.Context(), "Missing parameters")
	}

	trace, err := args[0].AsObject()
	if err != nil {
		return bridge.JsException(info.Context(), err)
	}

	node, err := args[1].AsObject()
	if err != nil {
		return bridge.JsException(info.Context(), err)
	}

	// Get trace ID
	traceID, err := trace.Get("id")
	if err != nil {
		return bridge.JsException(info.Context(), err)
	}

	// Get node ID
	nodeID, err := node.Get("id")
	if err != nil {
		return bridge.JsException(info.Context(), err)
	}

	// Verify IDs exist
	if traceID.IsUndefined() || nodeID.IsUndefined() {
		return bridge.JsException(info.Context(), fmt.Errorf("missing trace or node ID"))
	}

	// Return the trace object itself to trigger __release
	return args[0]
}

// TestTraceNodeComplete test completing a node
func TestTraceNodeComplete(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	res, err := v8.Call(v8.CallOptions{}, `
		function test() {
			const trace = new Trace({ driver: "local", path: "/tmp/test-traces" })
			const node = trace.Add({ type: "step", content: "Test step" }, { label: "Step 1" })
			
			// Log some messages
			node.Info("Processing...")
			node.Debug("Debug info")
			
			// Complete the node
			node.Complete({ result: "success" })
			
			return trace
		}`)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	// With Internal Field + goMaps, res should now be the manager directly
	manager, ok := res.(types.Manager)
	if !ok {
		t.Fatalf("Expected types.Manager, got %T", res)
	}

	assert.NotNil(t, manager, "manager should not be nil")
	// Note: We can't directly check goMaps cleanup as it's in the bridge package
}

// TestTraceSpace test creating and using spaces
func TestTraceSpace(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	v8.RegisterFunction("testTraceSpace", testTraceSpaceEmbed)
	res, err := v8.Call(v8.CallOptions{}, `
		function test() {
			const trace = new Trace({ driver: "local", path: "/tmp/test-traces" })
			const space = trace.CreateSpace({ label: "Test Space" })
			
			// Set some values
			space.Set("key1", "value1")
			space.Set("key2", 123)
			space.Set("key3", { nested: "object" })
			
			// Call test function for verification
			testTraceSpace(space)
			
			// Return trace object to trigger __release
			return trace
		}`)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	// With Internal Field + goMaps, res should now be the manager directly
	manager, ok := res.(types.Manager)
	if !ok {
		t.Fatalf("Expected types.Manager, got %T", res)
	}

	assert.NotNil(t, manager, "manager should not be nil")
	// Note: We can't directly check goMaps cleanup as it's in the bridge package
}

func testTraceSpaceEmbed(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, testTraceSpaceFunction)
}

func testTraceSpaceFunction(info *v8go.FunctionCallbackInfo) *v8go.Value {
	args := info.Args()
	if len(args) < 1 {
		return bridge.JsException(info.Context(), "Missing parameters")
	}

	space, err := args[0].AsObject()
	if err != nil {
		return bridge.JsException(info.Context(), err)
	}

	// Get space ID
	spaceID, err := space.Get("id")
	if err != nil {
		return bridge.JsException(info.Context(), err)
	}

	// Get stored values to verify
	getVal := func(obj *v8go.Object, method string, key string) (interface{}, error) {
		getMethod, err := obj.Get(method)
		if err != nil {
			return nil, err
		}
		defer getMethod.Release()
		getFn, err := getMethod.AsFunction()
		if err != nil {
			return nil, err
		}
		keyVal, err := v8go.NewValue(info.Context().Isolate(), key)
		if err != nil {
			return nil, err
		}
		defer keyVal.Release()
		result, err := getFn.Call(obj.Value, keyVal)
		if err != nil {
			return nil, err
		}
		return bridge.GoValue(result, info.Context())
	}

	key1, _ := getVal(space, "Get", "key1")
	key2, _ := getVal(space, "Get", "key2")

	// Verify values exist
	if spaceID.IsUndefined() || key1 == nil || key2 == nil {
		return bridge.JsException(info.Context(), fmt.Errorf("missing space data"))
	}

	// This function is just for verification, we don't need to return a new object
	// Return undefined, the outer JavaScript will return the trace object
	return v8go.Undefined(info.Context().Isolate())
}

// TestTraceConcurrent test concurrent trace operations
func TestTraceConcurrent(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Number of concurrent goroutines
	concurrency := 10
	iterationsPerGoroutine := 5

	var wg sync.WaitGroup
	errors := make(chan error, concurrency*iterationsPerGoroutine)
	results := make(chan string, concurrency*iterationsPerGoroutine)

	// Launch concurrent goroutines
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(routineID int) {
			defer wg.Done()

			for j := 0; j < iterationsPerGoroutine; j++ {
				script := fmt.Sprintf(`
					function test() {
						const trace = new Trace({ driver: "local", path: "/tmp/test-traces-%d-%d" })
						const node = trace.Add({ type: "step", content: "Test %d-%d" }, { label: "Step" })
						node.Info("Processing")
						node.Complete({ result: "done" })
						return trace
					}`, routineID, j, routineID, j)

				res, err := v8.Call(v8.CallOptions{}, script)
				if err != nil {
					errors <- fmt.Errorf("routine %d iteration %d failed: %v", routineID, j, err)
					return
				}

				// With Internal Field + goMaps, res should now be the manager directly
				if _, ok := res.(types.Manager); ok {
					// Successfully got manager, add a result
					results <- fmt.Sprintf("routine-%d-iteration-%d", routineID, j)
				} else {
					errors <- fmt.Errorf("routine %d iteration %d: expected types.Manager, got %T", routineID, j, res)
				}
			}
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()
	close(errors)
	close(results)

	// Check for errors
	for err := range errors {
		t.Error(err)
	}

	// Verify all results
	resultCount := 0
	for range results {
		resultCount++
	}

	// Verify the correct number of results
	expectedResults := concurrency * iterationsPerGoroutine
	assert.Equal(t, expectedResults, resultCount, "Should have %d results", expectedResults)

	// Verify all traces are cleaned up
	// Note: We can't directly check goMaps cleanup as it's in the bridge package
}

// TestTraceParallel test parallel node execution
func TestTraceParallel(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	v8.RegisterFunction("testTraceParallel", testTraceParallelEmbed)
	res, err := v8.Call(v8.CallOptions{}, `
		function test() {
			const trace = new Trace({ driver: "local", path: "/tmp/test-traces" })
			
			// Must call Add() first to create root node
			trace.Add({ type: "root", content: "Root" }, { label: "Root" })
			
			// Create parallel nodes
			const nodes = trace.Parallel([
				{ input: { type: "task", content: "Task 1" }, option: { label: "Parallel 1" } },
				{ input: { type: "task", content: "Task 2" }, option: { label: "Parallel 2" } },
				{ input: { type: "task", content: "Task 3" }, option: { label: "Parallel 3" } }
			])
			
			// Call test function for verification
			testTraceParallel(nodes)
			
			// Return trace object to trigger __release
			return trace
		}`)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	// With Internal Field + goMaps, res should now be the manager directly
	manager, ok := res.(types.Manager)
	if !ok {
		t.Fatalf("Expected types.Manager, got %T", res)
	}

	assert.NotNil(t, manager, "manager should not be nil")
	// Note: We can't directly check goMaps cleanup as it's in the bridge package
}

func testTraceParallelEmbed(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, testTraceParallelFunction)
}

func testTraceParallelFunction(info *v8go.FunctionCallbackInfo) *v8go.Value {
	args := info.Args()
	if len(args) < 1 {
		return bridge.JsException(info.Context(), "Missing parameters")
	}

	nodes, err := args[0].AsObject()
	if err != nil {
		return bridge.JsException(info.Context(), err)
	}

	// Get array length
	lengthVal, err := nodes.Get("length")
	if err != nil {
		return bridge.JsException(info.Context(), err)
	}

	// Verify we got 3 nodes
	if lengthVal.Int32() != 3 {
		return bridge.JsException(info.Context(), fmt.Errorf("expected 3 nodes, got %d", lengthVal.Int32()))
	}

	// Return undefined, the outer JavaScript will return the trace object
	return v8go.Undefined(info.Context().Isolate())
}

// TestTraceMarkComplete test marking trace as complete
func TestTraceMarkComplete(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	res, err := v8.Call(v8.CallOptions{}, `
		function test() {
			const trace = new Trace({ driver: "local", path: "/tmp/test-traces" })
			const node = trace.Add({ type: "step", content: "Test step" }, { label: "Step 1" })
			node.Complete({ result: "success" })
			
			// Mark entire trace as complete
			trace.MarkComplete()
			
			// Check if complete
			const isComplete = trace.IsComplete()
			
			return trace
		}`)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	// With Internal Field + goMaps, res should now be the manager directly
	manager, ok := res.(types.Manager)
	if !ok {
		t.Fatalf("Expected types.Manager, got %T", res)
	}

	assert.NotNil(t, manager, "manager should not be nil")
	// Note: We can't directly check goMaps cleanup as it's in the bridge package
}

// TestTracePassAsParameter test passing trace object as parameter to a function
func TestTracePassAsParameter(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	v8.RegisterFunction("processTrace", processTraceEmbed)
	res, err := v8.Call(v8.CallOptions{}, `
		function test() {
			// Define a function that accepts trace as parameter
			const handleTrace = function(trace) {
				// Add a node using the passed trace
				const node = trace.Add({ type: "step", content: "Step from handler" }, { label: "Handler Step" })
				node.Info("Processing in handler function")
				node.Complete({ result: "handler completed" })
				
				// Return some info
				return {
					traceId: trace.id,
					nodeId: node.id
				}
			}
			
			// Create trace and pass it to the handler
			const trace = new Trace({ driver: "local", path: "/tmp/test-traces" })
			const result = handleTrace(trace)
			
			// Also test passing to a Go function
			processTrace(trace)
			
			// Return trace to trigger __release
			return trace
		}`)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	// With __govalue function, res should now be the manager directly
	manager, ok := res.(types.Manager)
	if !ok {
		t.Fatalf("Expected types.Manager, got %T", res)
	}

	assert.NotNil(t, manager, "manager should not be nil")
	// Note: We can't directly check goMaps cleanup as it's in the bridge package
}

func processTraceEmbed(iso *v8go.Isolate) *v8go.FunctionTemplate {
	return v8go.NewFunctionTemplate(iso, processTraceFunction)
}

func processTraceFunction(info *v8go.FunctionCallbackInfo) *v8go.Value {
	args := info.Args()
	if len(args) < 1 {
		return bridge.JsException(info.Context(), "Missing trace parameter")
	}

	// Try to get the trace object
	traceValue := args[0]

	// Convert to Go value - with __govalue function, this should return the manager directly
	goValue, err := bridge.GoValue(traceValue, info.Context())
	if err != nil {
		return bridge.JsException(info.Context(), fmt.Errorf("failed to convert trace: %v", err))
	}

	fmt.Printf("\n=== Go function received trace ===\n")
	fmt.Printf("Type: %T\n", goValue)

	// Check if we got the manager directly (new __govalue behavior)
	if manager, ok := goValue.(types.Manager); ok {
		fmt.Printf("✅ SUCCESS: Got manager directly via __govalue function!\n")
		fmt.Printf("   Manager type: %T\n", manager)

		// Now we can use the manager directly!
		manager.Info("Message from Go function via __govalue")
		return v8go.Undefined(info.Context().Isolate())
	}

	// Fallback: if we got a map (shouldn't happen with __govalue)
	if traceMap, ok := goValue.(map[string]interface{}); ok {
		fmt.Printf("⚠️  Got map (fallback): %v\n", getMapKeys(traceMap))
		return bridge.JsException(info.Context(), fmt.Errorf("unexpected: got map instead of manager"))
	}

	return bridge.JsException(info.Context(), fmt.Errorf("unexpected type: %T", goValue))
}

func getMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
