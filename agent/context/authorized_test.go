package context

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/test"
	"github.com/yaoapp/yao/trace"
)

func TestContextNew_PreservesAuthorizedInfo(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Create authorized info
	authInfo := &types.AuthorizedInfo{
		UserID:   "716942074991",
		TeamID:   "565955042879",
		TenantID: "tenant-001",
	}

	// Create context using New()
	ctx := New(context.Background(), authInfo, "test-chat-123")

	// Verify authorized info is preserved
	assert.NotNil(t, ctx)
	assert.NotNil(t, ctx.Authorized)
	assert.Equal(t, "716942074991", ctx.Authorized.UserID)
	assert.Equal(t, "565955042879", ctx.Authorized.TeamID)
	assert.Equal(t, "tenant-001", ctx.Authorized.TenantID)
	assert.Equal(t, "test-chat-123", ctx.ChatID)
}

func TestContextTrace_SavesAuthorizedInfo(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Create authorized info
	authInfo := &types.AuthorizedInfo{
		UserID:   "716942074991",
		TeamID:   "565955042879",
		TenantID: "tenant-001",
	}

	// Create context
	ctx := New(context.Background(), authInfo, "test-chat-456")
	ctx.AssistantID = "test-assistant"

	// Initialize trace
	manager, err := ctx.Trace()
	assert.NoError(t, err)
	assert.NotNil(t, manager)

	// Get trace info
	info, err := manager.GetTraceInfo()
	assert.NoError(t, err)
	assert.NotNil(t, info)

	// Verify auth info is saved in trace
	assert.Equal(t, "716942074991", info.CreatedBy)
	assert.Equal(t, "565955042879", info.TeamID)
	assert.Equal(t, "tenant-001", info.TenantID)

	// Clean up
	if ctx.Stack != nil && ctx.Stack.TraceID != "" {
		trace.Release(ctx.Stack.TraceID)
		trace.Remove(context.Background(), trace.Local, ctx.Stack.TraceID)
	}
}

func TestContextNew_NilAuthorized(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Create context with nil authorized info (should not panic)
	ctx := New(context.Background(), nil, "test-chat-789")

	assert.NotNil(t, ctx)
	assert.Nil(t, ctx.Authorized)
	assert.Equal(t, "test-chat-789", ctx.ChatID)
}
