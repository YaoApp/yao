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
	stack, _ := agentctx.EnterStack(ctx, "test-assistant", &agentctx.Options{})
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
