//go:build integration

package context_test

import (
	stdContext "context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	agentctx "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

func TestContextNew_PreservesAuthorizedInfo(t *testing.T) {
	testprepare.PrepareSandbox(t)

	authInfo := &types.AuthorizedInfo{
		UserID:   "716942074991",
		TeamID:   "565955042879",
		TenantID: "tenant-001",
	}

	ctx := agentctx.New(stdContext.Background(), authInfo, "test-chat-123")
	defer ctx.Release()

	require.NotNil(t, ctx)
	require.NotNil(t, ctx.Authorized)
	assert.Equal(t, "716942074991", ctx.Authorized.UserID)
	assert.Equal(t, "565955042879", ctx.Authorized.TeamID)
	assert.Equal(t, "tenant-001", ctx.Authorized.TenantID)
	assert.Equal(t, "test-chat-123", ctx.ChatID)
}

func TestContextNew_NilAuthorized(t *testing.T) {
	testprepare.PrepareSandbox(t)

	ctx := agentctx.New(stdContext.Background(), nil, "test-chat-789")
	defer ctx.Release()

	require.NotNil(t, ctx)
	assert.Nil(t, ctx.Authorized)
	assert.Equal(t, "test-chat-789", ctx.ChatID)
	assert.NotNil(t, ctx.Memory)
	assert.NotNil(t, ctx.IDGenerator)
}
