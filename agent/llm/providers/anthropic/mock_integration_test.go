//go:build integration

package anthropic_test

import (
	gocontext "context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/gou/connector"
	goullm "github.com/yaoapp/gou/llm"
	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/llm"
	"github.com/yaoapp/yao/agent/output/message"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

func TestLLMMockAnthropicStreamEcho(t *testing.T) {
	testprepare.PrepareSandbox(t)

	conn, err := connector.Select("anthropic.mock")
	require.NoError(t, err)
	require.NotNil(t, conn)
	assert.True(t, conn.Is(connector.ANTHROPIC))

	caps := &goullm.Capabilities{Streaming: true, ToolCalls: true}
	opts := &context.CompletionOptions{Capabilities: caps}
	provider, err := llm.New(conn, opts)
	require.NoError(t, err)

	ctx := mockAnthropicTestContext("test-chat-anthropic-stream", "anthropic.mock")
	goCtx, cancel := gocontext.WithTimeout(gocontext.Background(), 30*time.Second)
	defer cancel()
	ctx.Context = goCtx

	messages := []context.Message{
		{Role: "user", Content: "hello from mock anthropic"},
	}

	var chunks int
	handler := func(chunkType message.StreamChunkType, data []byte) int {
		chunks++
		return 0
	}

	resp, err := provider.Stream(ctx, messages, opts, handler)
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Greater(t, chunks, 0)
}

func TestLLMMockAnthropicPost(t *testing.T) {
	testprepare.PrepareSandbox(t)

	conn, err := connector.Select("anthropic.mock")
	require.NoError(t, err)
	require.NotNil(t, conn)

	caps := &goullm.Capabilities{ToolCalls: true}
	opts := &context.CompletionOptions{Capabilities: caps}
	provider, err := llm.New(conn, opts)
	require.NoError(t, err)

	ctx := mockAnthropicTestContext("test-chat-anthropic-post", "anthropic.mock")
	goCtx, cancel := gocontext.WithTimeout(gocontext.Background(), 30*time.Second)
	defer cancel()
	ctx.Context = goCtx

	messages := []context.Message{
		{Role: "user", Content: "hello non-stream"},
	}

	resp, err := provider.Post(ctx, messages, opts)
	require.NoError(t, err)
	assert.NotNil(t, resp)
}
