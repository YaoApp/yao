//go:build integration

package anthropic_test

import (
	gocontext "context"
	"encoding/json"
	"strings"
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

func TestAnthropicStreamBasic(t *testing.T) {
	testprepare.PrepareSandbox(t)

	conn, err := connector.Select("anthropic.mock")
	require.NoError(t, err)
	require.NotNil(t, conn)
	assert.True(t, conn.Is(connector.ANTHROPIC))

	caps := &goullm.Capabilities{Streaming: true, ToolCalls: true}
	maxTokens := 10
	opts := &context.CompletionOptions{Capabilities: caps, MaxTokens: &maxTokens}

	provider, err := llm.New(conn, opts)
	require.NoError(t, err)

	ctx := mockAnthropicTestContext("test-anthropic-stream", "anthropic.mock")
	goCtx, cancel := gocontext.WithTimeout(gocontext.Background(), 30*time.Second)
	defer cancel()
	ctx.Context = goCtx

	messages := []context.Message{
		{Role: "user", Content: "Say 'Hi' in one word."},
	}

	var chunks int
	handler := func(chunkType message.StreamChunkType, data []byte) int {
		chunks++
		return 0
	}

	resp, err := provider.Stream(ctx, messages, opts, handler)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Greater(t, chunks, 0)
}

func TestAnthropicStreamWithToolCalls(t *testing.T) {
	testprepare.PrepareSandbox(t)

	conn, err := connector.Select("anthropic.mock")
	require.NoError(t, err)
	require.NotNil(t, conn)

	caps := &goullm.Capabilities{Streaming: true, ToolCalls: true}
	opts := &context.CompletionOptions{Capabilities: caps}

	weatherTool := map[string]interface{}{
		"type": "function",
		"function": map[string]interface{}{
			"name":        "get_weather",
			"description": "Get the current weather for a location",
			"parameters": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"location": map[string]interface{}{
						"type":        "string",
						"description": "The city name, e.g. Tokyo",
					},
				},
				"required": []string{"location"},
			},
		},
	}
	opts.Tools = []map[string]interface{}{weatherTool}
	opts.ToolChoice = "auto"

	provider, err := llm.New(conn, opts)
	require.NoError(t, err)

	ctx := mockAnthropicTestContext("test-anthropic-tool", "anthropic.mock")
	goCtx, cancel := gocontext.WithTimeout(gocontext.Background(), 30*time.Second)
	defer cancel()
	ctx.Context = goCtx

	messages := []context.Message{
		{Role: "user", Content: "What's the weather in Tokyo?"},
	}

	var chunks int
	handler := func(chunkType message.StreamChunkType, data []byte) int {
		chunks++
		return 0
	}

	resp, err := provider.Stream(ctx, messages, opts, handler)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Greater(t, chunks, 0)

	if len(resp.ToolCalls) > 0 {
		tc := resp.ToolCalls[0]
		assert.NotEmpty(t, tc.ID)
		assert.NotEmpty(t, tc.Function.Name)
		if tc.Function.Arguments != "" {
			var args map[string]interface{}
			err := json.Unmarshal([]byte(tc.Function.Arguments), &args)
			assert.NoError(t, err)
		}
	}
}

func TestAnthropicStreamRetry(t *testing.T) {
	testprepare.PrepareSandbox(t)

	connDSL := `{
		"type": "anthropic",
		"options": {
			"model": "claude-haiku-4-5-20251001",
			"key": "sk-ant-invalid-key-should-fail"
		}
	}`

	conn, err := connector.New("anthropic", "test-anthropic-retry", []byte(connDSL))
	require.NoError(t, err)

	caps := &goullm.Capabilities{Streaming: true, ToolCalls: true}
	opts := &context.CompletionOptions{Capabilities: caps}

	provider, err := llm.New(conn, opts)
	require.NoError(t, err)

	ctx := mockAnthropicTestContext("test-anthropic-retry", "test-anthropic-retry")
	goCtx, cancel := gocontext.WithTimeout(gocontext.Background(), 15*time.Second)
	defer cancel()
	ctx.Context = goCtx

	messages := []context.Message{
		{Role: "user", Content: "Test"},
	}

	_, err = provider.Stream(ctx, messages, opts, nil)
	require.Error(t, err)

	errMsg := strings.ToLower(err.Error())
	hasExpectedError := strings.Contains(errMsg, "401") ||
		strings.Contains(errMsg, "authentication") ||
		strings.Contains(errMsg, "invalid") ||
		strings.Contains(errMsg, "no data received") ||
		strings.Contains(errMsg, "non-retryable")
	assert.True(t, hasExpectedError, "expected authentication error, got: %v", err)
}
