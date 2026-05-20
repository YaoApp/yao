//go:build integration

package context_test

import (
	stdContext "context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	agentctx "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/trace"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

func TestContextTrace_SavesAuthorizedInfo(t *testing.T) {
	testprepare.PrepareSandbox(t)

	authInfo := &types.AuthorizedInfo{
		UserID:   "716942074991",
		TeamID:   "565955042879",
		TenantID: "tenant-001",
	}

	ctx := agentctx.New(stdContext.Background(), authInfo, "test-chat-456")
	ctx.AssistantID = "test-assistant"
	ctx.Referer = agentctx.RefererAPI

	stack, _, done := agentctx.EnterStack(ctx, "test-assistant", &agentctx.Options{})
	ctx.Stack = stack
	defer done()

	manager, err := ctx.Trace()
	require.NoError(t, err)
	require.NotNil(t, manager)

	info, err := manager.GetTraceInfo()
	require.NoError(t, err)
	require.NotNil(t, info)

	assert.Equal(t, "716942074991", info.CreatedBy)
	assert.Equal(t, "565955042879", info.TeamID)
	assert.Equal(t, "tenant-001", info.TenantID)

	if ctx.Stack != nil && ctx.Stack.TraceID != "" {
		trace.Release(ctx.Stack.TraceID)
		trace.Remove(stdContext.Background(), trace.Local, ctx.Stack.TraceID)
	}
}
