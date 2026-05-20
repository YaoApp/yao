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
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

func TestTemperatureDeepSeekR1AutoReset(t *testing.T) {
	testprepare.PrepareSandbox(t)

	conn, err := connector.Select("deepseek.mock-think")
	require.NoError(t, err)
	require.NotNil(t, conn)

	invalidTemp := 0.5
	caps := &goullm.Capabilities{Streaming: true, Reasoning: true}
	opts := &context.CompletionOptions{
		Capabilities: caps,
		Temperature:  &invalidTemp,
	}

	provider, err := llm.New(conn, opts)
	require.NoError(t, err)

	ctx := mockTestContext("test-deepseek-r1-temp", "deepseek.mock-think")
	goCtx, cancel := gocontext.WithTimeout(gocontext.Background(), 30*time.Second)
	defer cancel()
	ctx.Context = goCtx

	messages := []context.Message{
		{Role: "user", Content: "Say 'Hello'"},
	}

	maxTokens := 100
	opts.MaxCompletionTokens = &maxTokens

	resp, err := provider.Post(ctx, messages, opts)
	require.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestTemperatureGPT4oPreserved(t *testing.T) {
	testprepare.PrepareSandbox(t)

	conn, err := connector.Select("openai.mock")
	require.NoError(t, err)
	require.NotNil(t, conn)

	customTemp := 0.3
	caps := &goullm.Capabilities{ToolCalls: true}
	opts := &context.CompletionOptions{
		Capabilities: caps,
		Temperature:  &customTemp,
	}

	provider, err := llm.New(conn, opts)
	require.NoError(t, err)

	ctx := mockTestContext("test-gpt4o-temp", "openai.mock")
	goCtx, cancel := gocontext.WithTimeout(gocontext.Background(), 30*time.Second)
	defer cancel()
	ctx.Context = goCtx

	messages := []context.Message{
		{Role: "user", Content: "Say 'OK'"},
	}

	maxTokens := 10
	opts.MaxCompletionTokens = &maxTokens

	resp, err := provider.Post(ctx, messages, opts)
	require.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestTemperatureDeepSeekV3Preserved(t *testing.T) {
	testprepare.PrepareSandbox(t)

	conn, err := connector.Select("deepseek.mock")
	require.NoError(t, err)
	require.NotNil(t, conn)

	customTemp := 0.8
	caps := &goullm.Capabilities{ToolCalls: true}
	opts := &context.CompletionOptions{
		Capabilities: caps,
		Temperature:  &customTemp,
	}

	provider, err := llm.New(conn, opts)
	require.NoError(t, err)

	ctx := mockTestContext("test-deepseek-v3-temp", "deepseek.mock")
	goCtx, cancel := gocontext.WithTimeout(gocontext.Background(), 30*time.Second)
	defer cancel()
	ctx.Context = goCtx

	messages := []context.Message{
		{Role: "user", Content: "Say 'Hello World'"},
	}

	maxTokens := 20
	opts.MaxCompletionTokens = &maxTokens

	resp, err := provider.Post(ctx, messages, opts)
	require.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestTemperatureNoTemperatureProvided(t *testing.T) {
	testprepare.PrepareSandbox(t)

	testCases := []struct {
		name      string
		connector string
		reasoning bool
	}{
		{"OpenAI Mock No Temp", "openai.mock", false},
		{"DeepSeek Mock No Temp", "deepseek.mock", false},
		{"DeepSeek Mock Think No Temp", "deepseek.mock-think", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			conn, err := connector.Select(tc.connector)
			require.NoError(t, err)
			require.NotNil(t, conn)

			caps := &goullm.Capabilities{ToolCalls: true, Reasoning: tc.reasoning}
			opts := &context.CompletionOptions{Capabilities: caps}

			provider, err := llm.New(conn, opts)
			require.NoError(t, err)

			ctx := mockTestContext("test-no-temp-"+tc.connector, tc.connector)
			goCtx, cancel := gocontext.WithTimeout(gocontext.Background(), 30*time.Second)
			defer cancel()
			ctx.Context = goCtx

			messages := []context.Message{
				{Role: "user", Content: "Say 'OK'"},
			}

			maxTokens := 10
			opts.MaxCompletionTokens = &maxTokens

			resp, err := provider.Post(ctx, messages, opts)
			require.NoError(t, err)
			assert.NotNil(t, resp)
		})
	}
}
