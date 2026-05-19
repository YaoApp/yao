//go:build integration

package openai_test

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

func TestLLMMockOpenAIStreamEcho(t *testing.T) {
	testprepare.PrepareSandbox(t)

	conn, err := connector.Select("openai.mock")
	require.NoError(t, err)
	require.NotNil(t, conn)

	caps := &goullm.Capabilities{Streaming: true, ToolCalls: true}
	opts := &context.CompletionOptions{Capabilities: caps}
	provider, err := llm.New(conn, opts)
	require.NoError(t, err)

	ctx := mockTestContext("test-chat-stream", "openai.mock")
	goCtx, cancel := gocontext.WithTimeout(gocontext.Background(), 30*time.Second)
	defer cancel()
	ctx.Context = goCtx

	messages := []context.Message{
		{Role: "user", Content: "hello from mock test"},
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

func TestLLMMockOpenAIPost(t *testing.T) {
	testprepare.PrepareSandbox(t)

	conn, err := connector.Select("openai.mock")
	require.NoError(t, err)
	require.NotNil(t, conn)

	caps := &goullm.Capabilities{ToolCalls: true}
	opts := &context.CompletionOptions{Capabilities: caps}
	provider, err := llm.New(conn, opts)
	require.NoError(t, err)

	ctx := mockTestContext("test-chat-post", "openai.mock")
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

func TestLLMMockDeepSeekThinking(t *testing.T) {
	testprepare.PrepareSandbox(t)

	conn, err := connector.Select("deepseek.mock-think")
	require.NoError(t, err)
	require.NotNil(t, conn)

	caps := &goullm.Capabilities{Streaming: true, ToolCalls: true}
	opts := &context.CompletionOptions{Capabilities: caps}
	provider, err := llm.New(conn, opts)
	require.NoError(t, err)

	ctx := mockTestContext("test-chat-think", "deepseek.mock-think")
	goCtx, cancel := gocontext.WithTimeout(gocontext.Background(), 30*time.Second)
	defer cancel()
	ctx.Context = goCtx

	messages := []context.Message{
		{Role: "user", Content: "think about something"},
	}

	handler := func(chunkType message.StreamChunkType, data []byte) int { return 0 }

	resp, err := provider.Stream(ctx, messages, opts, handler)
	require.NoError(t, err)
	assert.NotNil(t, resp)
}
