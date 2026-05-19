//go:build e2e

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

func TestE2EOpenAIGPT4oMini(t *testing.T) {
	testprepare.PrepareE2E(t)

	conn, err := connector.Select("openai.gpt-4o-mini")
	require.NoError(t, err)
	require.NotNil(t, conn)

	caps := &goullm.Capabilities{Streaming: true, ToolCalls: true}
	maxTok := 10
	opts := &context.CompletionOptions{Capabilities: caps, MaxTokens: &maxTok}
	provider, err := llm.New(conn, opts)
	require.NoError(t, err)

	ctx := mockTestContext("test-e2e-gpt4o-mini", "openai.gpt-4o-mini")
	goCtx, cancel := gocontext.WithTimeout(gocontext.Background(), 60*time.Second)
	defer cancel()
	ctx.Context = goCtx

	messages := []context.Message{
		{Role: "user", Content: "Say 'Hello' in one word."},
	}

	handler := func(chunkType message.StreamChunkType, data []byte) int { return 0 }

	resp, err := provider.Stream(ctx, messages, opts, handler)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.NotEmpty(t, resp.Content)
}

func TestE2EDeepSeekV4Flash(t *testing.T) {
	testprepare.PrepareE2E(t)

	conn, err := connector.Select("deepseek.v4-flash")
	require.NoError(t, err)
	require.NotNil(t, conn)

	caps := &goullm.Capabilities{Streaming: true, ToolCalls: true}
	maxTok := 10
	opts := &context.CompletionOptions{Capabilities: caps, MaxTokens: &maxTok}
	provider, err := llm.New(conn, opts)
	require.NoError(t, err)

	ctx := mockTestContext("test-e2e-deepseek-v4", "deepseek.v4-flash")
	goCtx, cancel := gocontext.WithTimeout(gocontext.Background(), 60*time.Second)
	defer cancel()
	ctx.Context = goCtx

	messages := []context.Message{
		{Role: "user", Content: "Say 'Hello' in one word."},
	}

	handler := func(chunkType message.StreamChunkType, data []byte) int { return 0 }

	resp, err := provider.Stream(ctx, messages, opts, handler)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.NotEmpty(t, resp.Content)
}
