//go:build integration

package context_test

import (
	stdContext "context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	agentctx "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

func TestNewStack(t *testing.T) {
	testprepare.PrepareSandbox(t)

	traceID := "12345678"
	assistantID := "test-assistant"
	referer := agentctx.RefererAPI
	opts := &agentctx.Options{}

	stack := agentctx.NewStack(traceID, assistantID, referer, opts)

	require.NotNil(t, stack)
	assert.Equal(t, traceID, stack.TraceID)
	assert.Equal(t, assistantID, stack.AssistantID)
	assert.Equal(t, referer, stack.Referer)
	assert.Equal(t, 0, stack.Depth)
	assert.Empty(t, stack.ParentID)
	assert.True(t, stack.IsRoot())
	assert.Equal(t, agentctx.StackStatusRunning, stack.Status)
	assert.NotEmpty(t, stack.ID)
	assert.Len(t, stack.Path, 1)
	assert.Equal(t, stack.ID, stack.Path[0])
}

func TestNewStack_GenerateTraceID(t *testing.T) {
	testprepare.PrepareSandbox(t)

	stack := agentctx.NewStack("", "test-assistant", agentctx.RefererAPI, &agentctx.Options{})

	require.NotNil(t, stack)
	assert.NotEmpty(t, stack.TraceID)
	assert.GreaterOrEqual(t, len(stack.TraceID), 8)
}

func TestNewChildStack(t *testing.T) {
	testprepare.PrepareSandbox(t)

	parentStack := agentctx.NewStack("12345678", "parent-assistant", agentctx.RefererAPI, &agentctx.Options{})
	require.NotNil(t, parentStack)

	childStack := parentStack.NewChildStack("child-assistant", agentctx.RefererAgent, &agentctx.Options{})

	require.NotNil(t, childStack)
	assert.Equal(t, parentStack.TraceID, childStack.TraceID)
	assert.Equal(t, parentStack.ID, childStack.ParentID)
	assert.Equal(t, parentStack.Depth+1, childStack.Depth)
	assert.False(t, childStack.IsRoot())
	assert.Len(t, childStack.Path, 2)
	assert.Equal(t, parentStack.ID, childStack.Path[0])
	assert.Equal(t, childStack.ID, childStack.Path[1])
	assert.Equal(t, agentctx.StackStatusRunning, childStack.Status)
}

func TestStackComplete(t *testing.T) {
	testprepare.PrepareSandbox(t)

	stack := agentctx.NewStack("12345678", "test-assistant", agentctx.RefererAPI, &agentctx.Options{})
	time.Sleep(10 * time.Millisecond)

	stack.Complete()

	assert.Equal(t, agentctx.StackStatusCompleted, stack.Status)
	require.NotNil(t, stack.CompletedAt)
	require.NotNil(t, stack.DurationMs)
	assert.GreaterOrEqual(t, *stack.DurationMs, int64(10))
	assert.True(t, stack.IsCompleted())
	assert.False(t, stack.IsRunning())
}

func TestStackFail(t *testing.T) {
	testprepare.PrepareSandbox(t)

	stack := agentctx.NewStack("12345678", "test-assistant", agentctx.RefererAPI, &agentctx.Options{})

	stack.Fail(nil)
	stack.Error = "test error message"

	assert.Equal(t, agentctx.StackStatusFailed, stack.Status)
	assert.Equal(t, "test error message", stack.Error)
	assert.True(t, stack.IsCompleted())
	require.NotNil(t, stack.CompletedAt)
	require.NotNil(t, stack.DurationMs)
}

func TestStackTimeout(t *testing.T) {
	testprepare.PrepareSandbox(t)

	stack := agentctx.NewStack("12345678", "test-assistant", agentctx.RefererAPI, &agentctx.Options{})

	stack.Timeout()

	assert.Equal(t, agentctx.StackStatusTimeout, stack.Status)
	assert.True(t, stack.IsCompleted())
	require.NotNil(t, stack.CompletedAt)
	require.NotNil(t, stack.DurationMs)
}

func TestEnterStack_RootCreation(t *testing.T) {
	testprepare.PrepareSandbox(t)

	ctx := agentctx.New(stdContext.Background(), nil, "test-chat-id")
	ctx.Referer = agentctx.RefererAPI

	stack, traceID, done := agentctx.EnterStack(ctx, "test-assistant", &agentctx.Options{})
	defer done()

	require.NotNil(t, stack)
	assert.NotEmpty(t, traceID)
	assert.GreaterOrEqual(t, len(traceID), 8)
	assert.Equal(t, traceID, stack.TraceID)
	assert.Equal(t, stack, ctx.Stack)
	require.NotNil(t, ctx.Stacks)
	assert.Equal(t, stack, ctx.Stacks[stack.ID])
	assert.True(t, stack.IsRoot())
}

func TestEnterStack_ChildCreation(t *testing.T) {
	testprepare.PrepareSandbox(t)

	ctx := agentctx.New(stdContext.Background(), nil, "test-chat-id")
	ctx.Referer = agentctx.RefererAPI

	parentStack, parentTraceID, parentDone := agentctx.EnterStack(ctx, "parent-assistant", &agentctx.Options{})
	defer parentDone()
	require.NotNil(t, parentStack)

	childStack, childTraceID, childDone := agentctx.EnterStack(ctx, "child-assistant", &agentctx.Options{})
	defer childDone()
	require.NotNil(t, childStack)

	assert.Equal(t, parentTraceID, childTraceID)
	assert.Equal(t, parentStack.ID, childStack.ParentID)
	assert.Len(t, ctx.Stacks, 2)
	assert.Equal(t, childStack, ctx.Stack)
}

func TestEnterStack_DoneCallback(t *testing.T) {
	testprepare.PrepareSandbox(t)

	ctx := agentctx.New(stdContext.Background(), nil, "test-chat-id")
	ctx.Referer = agentctx.RefererAPI

	parentStack, _, parentDone := agentctx.EnterStack(ctx, "parent-assistant", &agentctx.Options{})
	childStack, _, childDone := agentctx.EnterStack(ctx, "child-assistant", &agentctx.Options{})

	assert.Equal(t, childStack, ctx.Stack)

	childDone()
	assert.Equal(t, parentStack, ctx.Stack)
	assert.True(t, childStack.IsCompleted())

	parentDone()
	assert.True(t, parentStack.IsCompleted())
}

func TestContextGetAllStacks(t *testing.T) {
	testprepare.PrepareSandbox(t)

	ctx := agentctx.New(stdContext.Background(), nil, "test-chat-id")
	ctx.Referer = agentctx.RefererAPI

	_, _, done1 := agentctx.EnterStack(ctx, "assistant1", &agentctx.Options{})
	defer done1()
	_, _, done2 := agentctx.EnterStack(ctx, "assistant2", &agentctx.Options{})
	defer done2()
	_, _, done3 := agentctx.EnterStack(ctx, "assistant3", &agentctx.Options{})
	defer done3()

	allStacks := ctx.GetAllStacks()
	assert.Len(t, allStacks, 3)
}

func TestContextGetStackByID(t *testing.T) {
	testprepare.PrepareSandbox(t)

	ctx := agentctx.New(stdContext.Background(), nil, "test-chat-id")
	ctx.Referer = agentctx.RefererAPI

	stack, _, done := agentctx.EnterStack(ctx, "test-assistant", &agentctx.Options{})
	defer done()

	found := ctx.GetStackByID(stack.ID)
	require.NotNil(t, found)
	assert.Equal(t, stack.ID, found.ID)

	notFound := ctx.GetStackByID("non-existent-id")
	assert.Nil(t, notFound)
}

func TestContextGetStacksByTraceID(t *testing.T) {
	testprepare.PrepareSandbox(t)

	ctx := agentctx.New(stdContext.Background(), nil, "test-chat-id")
	ctx.Referer = agentctx.RefererAPI

	_, traceID, done1 := agentctx.EnterStack(ctx, "parent-assistant", &agentctx.Options{})
	defer done1()
	_, _, done2 := agentctx.EnterStack(ctx, "child-assistant", &agentctx.Options{})
	defer done2()

	stacks := ctx.GetStacksByTraceID(traceID)
	assert.Len(t, stacks, 2)

	for _, s := range stacks {
		assert.Equal(t, traceID, s.TraceID)
	}
}

func TestContextGetRootStack(t *testing.T) {
	testprepare.PrepareSandbox(t)

	ctx := agentctx.New(stdContext.Background(), nil, "test-chat-id")
	ctx.Referer = agentctx.RefererAPI

	parentStack, _, done1 := agentctx.EnterStack(ctx, "parent-assistant", &agentctx.Options{})
	defer done1()
	_, _, done2 := agentctx.EnterStack(ctx, "child-assistant", &agentctx.Options{})
	defer done2()

	rootStack := ctx.GetRootStack()
	require.NotNil(t, rootStack)
	assert.Equal(t, parentStack.ID, rootStack.ID)
	assert.True(t, rootStack.IsRoot())
}

func TestStackClone(t *testing.T) {
	testprepare.PrepareSandbox(t)

	original := agentctx.NewStack("12345678", "test-assistant", agentctx.RefererAPI, &agentctx.Options{})
	original.Complete()

	clone := original.Clone()

	require.NotNil(t, clone)
	assert.Equal(t, original.ID, clone.ID)
	assert.Equal(t, original.TraceID, clone.TraceID)
	assert.Equal(t, original.AssistantID, clone.AssistantID)
	assert.Equal(t, original.Status, clone.Status)
	assert.Equal(t, original.Referer, clone.Referer)
	assert.Equal(t, original.Depth, clone.Depth)
	assert.Equal(t, original.ParentID, clone.ParentID)
	assert.Len(t, clone.Path, len(original.Path))

	if len(clone.Path) > 0 {
		clone.Path[0] = "modified"
		assert.NotEqual(t, "modified", original.Path[0], "Path should be deep-copied")
	}

	require.NotNil(t, clone.CompletedAt)
	assert.Equal(t, *original.CompletedAt, *clone.CompletedAt)
	require.NotNil(t, clone.DurationMs)
	assert.Equal(t, *original.DurationMs, *clone.DurationMs)
}
