//go:build integration

package context_test

import (
	stdContext "context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v8 "github.com/yaoapp/gou/runtime/v8"
	agentctx "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

func newReleaseTestContext() *agentctx.Context {
	ctx := agentctx.New(stdContext.Background(), nil, "test-chat-id")
	ctx.AssistantID = "test-assistant-id"
	ctx.Referer = agentctx.RefererAPI
	stack, _, _ := agentctx.EnterStack(ctx, "test-assistant", &agentctx.Options{})
	ctx.Stack = stack
	return ctx
}

func TestContextRelease(t *testing.T) {
	testprepare.PrepareSandbox(t)

	cxt := newReleaseTestContext()

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			if (typeof ctx.Release !== 'function') {
				throw new Error("ctx.Release is not a function")
			}
			
			if (typeof ctx.__release !== 'function') {
				throw new Error("ctx.__release is not a function")
			}
			
			ctx.Release()
			
			ctx.Release()
			
			return {
				has_release: true,
				success: true
			}
		}`, cxt)
	require.NoError(t, err, "Call failed")

	result, ok := res.(map[string]interface{})
	require.True(t, ok, "Expected map result, got %T", res)

	assert.Equal(t, true, result["has_release"], "should have Release method")
	assert.Equal(t, true, result["success"], "release should succeed")
}

func TestTraceRelease(t *testing.T) {
	testprepare.PrepareSandbox(t)

	cxt := newReleaseTestContext()

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			const trace = ctx.trace
			
			if (typeof trace.Release !== 'function') {
				throw new Error("trace.Release is not a function")
			}
			
			if (typeof trace.__release !== 'function') {
				throw new Error("trace.__release is not a function")
			}
			
			const node = trace.Add({ type: "test" }, { label: "Test Node" })
			trace.Info("Test message")
			
			trace.Release()
			
			trace.Release()
			
			return {
				has_release: true,
				has_node: !!node,
				success: true
			}
		}`, cxt)
	require.NoError(t, err, "Call failed")

	result, ok := res.(map[string]interface{})
	require.True(t, ok, "Expected map result, got %T", res)

	assert.Equal(t, true, result["has_release"], "should have Release method")
	assert.Equal(t, true, result["has_node"], "should create node")
	assert.Equal(t, true, result["success"], "release should succeed")
}

func TestContextReleaseWithTrace(t *testing.T) {
	testprepare.PrepareSandbox(t)

	cxt := newReleaseTestContext()

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			const trace = ctx.trace
			
			const node = trace.Add({ type: "test" }, { label: "Test Node" })
			trace.Info("Test message")
			node.Complete({ result: "done" })
			
			ctx.Release()
			
			return {
				trace_released_via_context: true,
				success: true
			}
		}`, cxt)
	require.NoError(t, err, "Call failed")

	result, ok := res.(map[string]interface{})
	require.True(t, ok, "Expected map result, got %T", res)

	assert.Equal(t, true, result["trace_released_via_context"], "trace should be released via context")
	assert.Equal(t, true, result["success"], "release should succeed")
}

func TestTryFinallyPattern(t *testing.T) {
	testprepare.PrepareSandbox(t)

	cxt := newReleaseTestContext()

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			const trace = ctx.trace
			
			try {
				const node = trace.Add({ type: "step" }, { label: "Processing" })
				
				trace.Info("Step 1: Initialize")
				trace.Info("Step 2: Process")
				
				node.Complete({ result: "success" })
				
				return {
					completed: true
				}
			} finally {
				trace.Release()
				ctx.Release()
			}
		}`, cxt)
	require.NoError(t, err, "Call failed")

	result, ok := res.(map[string]interface{})
	require.True(t, ok, "Expected map result, got %T", res)

	assert.Equal(t, true, result["completed"], "should complete successfully")
}

func TestNoOpTraceRelease(t *testing.T) {
	testprepare.PrepareSandbox(t)

	cxt := agentctx.New(stdContext.Background(), nil, "test-chat-id")
	cxt.AssistantID = "test-assistant-id"

	res, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			const trace = ctx.trace
			
			if (typeof trace.Release !== 'function') {
				throw new Error("no-op trace.Release is not a function")
			}
			
			trace.Info("This is a no-op")
			const node = trace.Add({ type: "test" }, { label: "No-op" })
			node.Complete({ result: "done" })
			
			trace.Release()
			
			return {
				noop_trace_works: true,
				success: true
			}
		}`, cxt)
	require.NoError(t, err, "Call failed")

	result, ok := res.(map[string]interface{})
	require.True(t, ok, "Expected map result, got %T", res)

	assert.Equal(t, true, result["noop_trace_works"], "no-op trace should work")
	assert.Equal(t, true, result["success"], "release should succeed")
}

func TestTryFinallyPatternWithError(t *testing.T) {
	testprepare.PrepareSandbox(t)

	cxt := newReleaseTestContext()

	_, err := v8.Call(v8.CallOptions{}, `
		function test(ctx) {
			const trace = ctx.trace
			
			try {
				const node = trace.Add({ type: "step" }, { label: "Processing" })
				trace.Info("Starting work")
				
				throw new Error("Simulated error")
				
			} finally {
				trace.Release()
				ctx.Release()
			}
		}`, cxt)

	require.Error(t, err, "Expected error to be propagated")
	assert.Contains(t, err.Error(), "Simulated error", "error should be propagated")
}
