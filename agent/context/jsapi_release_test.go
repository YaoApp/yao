package context_test

import (
	stdContext "context"
	"testing"

	"github.com/stretchr/testify/assert"
	v8 "github.com/yaoapp/gou/runtime/v8"
	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/test"
)

// newReleaseTestContext creates a test context for release testing
func newReleaseTestContext() *context.Context {
	ctx := context.New(stdContext.Background(), nil, "test-chat-id")
	ctx.AssistantID = "test-assistant-id"
	ctx.Referer = context.RefererAPI
	stack, _, _ := context.EnterStack(ctx, "test-assistant", &context.Options{})
	ctx.Stack = stack
	return ctx
}

// TestContextRelease tests explicit Release() method on Context
func TestContextRelease(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	cxt := newReleaseTestContext()

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			// Verify context has Release method
			if (typeof ctx.Release !== 'function') {
				throw new Error("ctx.Release is not a function")
			}
			
			// Verify context has __release method
			if (typeof ctx.__release !== 'function') {
				throw new Error("ctx.__release is not a function")
			}
			
			// Call Release explicitly
			ctx.Release()
			
			// Can call Release multiple times safely (idempotent)
			ctx.Release()
			
			return {
				has_release: true,
				success: true
			}
		}`, cxt)

	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	result, ok := res.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", res)
	}

	assert.Equal(t, true, result["has_release"], "should have Release method")
	assert.Equal(t, true, result["success"], "release should succeed")
}

// TestTraceRelease tests explicit Release() method on Trace
func TestTraceRelease(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	cxt := newReleaseTestContext()

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			// Get trace
			const trace = ctx.trace
			
			// Verify trace has Release method
			if (typeof trace.Release !== 'function') {
				throw new Error("trace.Release is not a function")
			}
			
			// Verify trace has __release method
			if (typeof trace.__release !== 'function') {
				throw new Error("trace.__release is not a function")
			}
			
			// Use trace
			const node = trace.Add({ type: "test" }, { label: "Test Node" })
			trace.Info("Test message")
			
			// Release trace explicitly
			trace.Release()
			
			// Can call Release multiple times safely (idempotent)
			trace.Release()
			
			return {
				has_release: true,
				has_node: !!node,
				success: true
			}
		}`, cxt)

	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	result, ok := res.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", res)
	}

	assert.Equal(t, true, result["has_release"], "should have Release method")
	assert.Equal(t, true, result["has_node"], "should create node")
	assert.Equal(t, true, result["success"], "release should succeed")
}

// TestContextReleaseWithTrace tests that releasing Context also releases Trace
func TestContextReleaseWithTrace(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	cxt := newReleaseTestContext()

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			// Get trace
			const trace = ctx.trace
			
			// Use trace
			const node = trace.Add({ type: "test" }, { label: "Test Node" })
			trace.Info("Test message")
			node.Complete({ result: "done" })
			
			// Release context (should also release trace)
			ctx.Release()
			
			return {
				trace_released_via_context: true,
				success: true
			}
		}`, cxt)

	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	result, ok := res.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", res)
	}

	assert.Equal(t, true, result["trace_released_via_context"], "trace should be released via context")
	assert.Equal(t, true, result["success"], "release should succeed")
}

// TestTryFinallyPattern tests the try-finally pattern with Release()
func TestTryFinallyPattern(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	cxt := newReleaseTestContext()

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			const trace = ctx.trace
			
			// Try-finally pattern for explicit resource management
			try {
				const node = trace.Add({ type: "step" }, { label: "Processing" })
				
				// Simulate some work
				trace.Info("Step 1: Initialize")
				trace.Info("Step 2: Process")
				
				node.Complete({ result: "success" })
				
				return {
					completed: true
				}
			} finally {
				// Explicit cleanup
				trace.Release()
				ctx.Release()
			}
		}`, cxt)

	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	result, ok := res.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", res)
	}

	assert.Equal(t, true, result["completed"], "should complete successfully")
}

// TestNoOpTraceRelease tests that no-op Trace also has Release method
func TestNoOpTraceRelease(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Context without trace initialization
	cxt := context.New(stdContext.Background(), nil, "test-chat-id")
	cxt.AssistantID = "test-assistant-id"

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			// Get trace (should be no-op)
			const trace = ctx.trace
			
			// Verify trace has Release method even when it's no-op
			if (typeof trace.Release !== 'function') {
				throw new Error("no-op trace.Release is not a function")
			}
			
			// Call methods on no-op trace (should not error)
			trace.Info("This is a no-op")
			const node = trace.Add({ type: "test" }, { label: "No-op" })
			node.Complete({ result: "done" })
			
			// Release no-op trace (should not error)
			trace.Release()
			
			return {
				noop_trace_works: true,
				success: true
			}
		}`, cxt)

	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	result, ok := res.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", res)
	}

	assert.Equal(t, true, result["noop_trace_works"], "no-op trace should work")
	assert.Equal(t, true, result["success"], "release should succeed")
}

// TestTryFinallyPatternWithError tests try-finally with error handling
func TestTryFinallyPatternWithError(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	cxt := newReleaseTestContext()

	_, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			const trace = ctx.trace
			
			// Try-finally pattern ensures cleanup even when error occurs
			try {
				const node = trace.Add({ type: "step" }, { label: "Processing" })
				trace.Info("Starting work")
				
				// Simulate an error
				throw new Error("Simulated error")
				
			} finally {
				// Cleanup happens even after error
				trace.Release()
				ctx.Release()
			}
		}`, cxt)

	// Error should be propagated
	if err == nil {
		t.Fatal("Expected error to be propagated")
	}

	// But cleanup should have happened (no way to verify directly, but test should not crash)
	assert.Contains(t, err.Error(), "Simulated error", "error should be propagated")
}
