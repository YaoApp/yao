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

func TestDeepSeekV3StreamBasic(t *testing.T) {
	testprepare.PrepareSandbox(t)

	conn, err := connector.Select("deepseek.mock")
	require.NoError(t, err)
	require.NotNil(t, conn)

	caps := &goullm.Capabilities{Streaming: true, ToolCalls: true}
	maxTokens := 100
	opts := &context.CompletionOptions{Capabilities: caps, MaxTokens: &maxTokens}

	provider, err := llm.New(conn, opts)
	require.NoError(t, err)

	ctx := mockTestContext("test-deepseek-v3-basic", "deepseek.mock")
	goCtx, cancel := gocontext.WithTimeout(gocontext.Background(), 30*time.Second)
	defer cancel()
	ctx.Context = goCtx

	messages := []context.Message{
		{Role: "user", Content: "What is 5 + 3?"},
	}

	var contentChunks int
	handler := func(chunkType message.StreamChunkType, data []byte) int {
		if chunkType == message.ChunkText {
			contentChunks++
		}
		return 0
	}

	resp, err := provider.Stream(ctx, messages, opts, handler)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Greater(t, contentChunks, 0)
}

func TestDeepSeekV3PostBasic(t *testing.T) {
	testprepare.PrepareSandbox(t)

	conn, err := connector.Select("deepseek.mock")
	require.NoError(t, err)
	require.NotNil(t, conn)

	caps := &goullm.Capabilities{ToolCalls: true}
	maxTokens := 100
	opts := &context.CompletionOptions{Capabilities: caps, MaxTokens: &maxTokens}

	provider, err := llm.New(conn, opts)
	require.NoError(t, err)

	ctx := mockTestContext("test-deepseek-v3-post", "deepseek.mock")
	goCtx, cancel := gocontext.WithTimeout(gocontext.Background(), 30*time.Second)
	defer cancel()
	ctx.Context = goCtx

	messages := []context.Message{
		{Role: "user", Content: "What is 2 * 4?"},
	}

	resp, err := provider.Post(ctx, messages, opts)
	require.NoError(t, err)
	require.NotNil(t, resp)
}

func TestDeepSeekV3WithToolCalls(t *testing.T) {
	testprepare.PrepareSandbox(t)

	conn, err := connector.Select("deepseek.mock")
	require.NoError(t, err)
	require.NotNil(t, conn)

	caps := &goullm.Capabilities{ToolCalls: true}
	opts := &context.CompletionOptions{Capabilities: caps}

	simpleTool := map[string]interface{}{
		"type": "function",
		"function": map[string]interface{}{
			"name":        "get_info",
			"description": "Get information",
			"parameters": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{"type": "string"},
					"count": map[string]interface{}{"type": "number"},
				},
				"required": []string{"query", "count"},
			},
		},
	}
	opts.Tools = []map[string]interface{}{simpleTool}
	opts.ToolChoice = "auto"

	maxTokens := 50
	opts.MaxTokens = &maxTokens

	provider, err := llm.New(conn, opts)
	require.NoError(t, err)

	ctx := mockTestContext("test-deepseek-v3-tools", "deepseek.mock")
	goCtx, cancel := gocontext.WithTimeout(gocontext.Background(), 30*time.Second)
	defer cancel()
	ctx.Context = goCtx

	messages := []context.Message{
		{Role: "user", Content: "Call get_info with query='A' and count=1"},
	}

	resp, err := provider.Post(ctx, messages, opts)
	require.NoError(t, err)
	require.NotNil(t, resp)
}

func TestDeepSeekV3NoReasoningEffort(t *testing.T) {
	testprepare.PrepareSandbox(t)

	conn, err := connector.Select("deepseek.mock")
	require.NoError(t, err)
	require.NotNil(t, conn)

	effort := "high"
	caps := &goullm.Capabilities{ToolCalls: true}
	opts := &context.CompletionOptions{
		Capabilities:    caps,
		ReasoningEffort: &effort,
	}

	provider, err := llm.New(conn, opts)
	require.NoError(t, err)

	ctx := mockTestContext("test-deepseek-v3-no-reasoning", "deepseek.mock")
	goCtx, cancel := gocontext.WithTimeout(gocontext.Background(), 30*time.Second)
	defer cancel()
	ctx.Context = goCtx

	messages := []context.Message{
		{Role: "user", Content: "Reply with just: OK"},
	}

	maxTokens := 20
	opts.MaxTokens = &maxTokens

	resp, err := provider.Post(ctx, messages, opts)
	require.NoError(t, err)
	assert.NotNil(t, resp)
}
