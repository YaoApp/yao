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

func TestDeepSeekR1StreamBasic(t *testing.T) {
	testprepare.PrepareSandbox(t)

	conn, err := connector.Select("deepseek.mock-think")
	require.NoError(t, err)
	require.NotNil(t, conn)

	caps := &goullm.Capabilities{Streaming: true, Reasoning: true}
	maxTokens := 500
	opts := &context.CompletionOptions{Capabilities: caps, MaxTokens: &maxTokens}

	provider, err := llm.New(conn, opts)
	require.NoError(t, err)

	ctx := mockTestContext("test-deepseek-r1-basic", "deepseek.mock-think")
	goCtx, cancel := gocontext.WithTimeout(gocontext.Background(), 30*time.Second)
	defer cancel()
	ctx.Context = goCtx

	messages := []context.Message{
		{Role: "user", Content: "What is 2 + 2?"},
	}

	var reasoningChunks, contentChunks int
	handler := func(chunkType message.StreamChunkType, data []byte) int {
		switch chunkType {
		case message.ChunkThinking:
			reasoningChunks++
		case message.ChunkText:
			contentChunks++
		}
		return 0
	}

	resp, err := provider.Stream(ctx, messages, opts, handler)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Greater(t, reasoningChunks+contentChunks, 0)
}

func TestDeepSeekR1PostBasic(t *testing.T) {
	testprepare.PrepareSandbox(t)

	conn, err := connector.Select("deepseek.mock-think")
	require.NoError(t, err)
	require.NotNil(t, conn)

	caps := &goullm.Capabilities{Reasoning: true}
	maxTokens := 500
	opts := &context.CompletionOptions{Capabilities: caps, MaxTokens: &maxTokens}

	provider, err := llm.New(conn, opts)
	require.NoError(t, err)

	ctx := mockTestContext("test-deepseek-r1-post", "deepseek.mock-think")
	goCtx, cancel := gocontext.WithTimeout(gocontext.Background(), 30*time.Second)
	defer cancel()
	ctx.Context = goCtx

	messages := []context.Message{
		{Role: "user", Content: "What is 1+1?"},
	}

	resp, err := provider.Post(ctx, messages, opts)
	require.NoError(t, err)
	require.NotNil(t, resp)
}

func TestDeepSeekR1LogicPuzzle(t *testing.T) {
	testprepare.PrepareSandbox(t)

	conn, err := connector.Select("deepseek.mock-think")
	require.NoError(t, err)
	require.NotNil(t, conn)

	caps := &goullm.Capabilities{Streaming: true, Reasoning: true}
	maxTokens := 800
	opts := &context.CompletionOptions{Capabilities: caps, MaxTokens: &maxTokens}

	provider, err := llm.New(conn, opts)
	require.NoError(t, err)

	ctx := mockTestContext("test-deepseek-r1-logic", "deepseek.mock-think")
	goCtx, cancel := gocontext.WithTimeout(gocontext.Background(), 30*time.Second)
	defer cancel()
	ctx.Context = goCtx

	messages := []context.Message{
		{Role: "user", Content: "Is 5 greater than 3? Explain your reasoning."},
	}

	var hasReasoning, hasContent bool
	handler := func(chunkType message.StreamChunkType, data []byte) int {
		if chunkType == message.ChunkThinking && len(data) > 0 {
			hasReasoning = true
		} else if chunkType == message.ChunkText && len(data) > 0 {
			hasContent = true
		}
		return 0
	}

	resp, err := provider.Stream(ctx, messages, opts, handler)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, hasReasoning || hasContent, "expected at least reasoning or content chunks")
}
