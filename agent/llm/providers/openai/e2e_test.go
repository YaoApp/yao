//go:build e2e

package openai_test

import (
	gocontext "context"
	"testing"
	"time"

	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/gou/connector/openai"
	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/llm"
	"github.com/yaoapp/yao/agent/output/message"
	"github.com/yaoapp/yao/agent/testutils"
)

func TestE2EOpenAIGPT4oMini(t *testing.T) {
	testutils.PrepareAgent(t)
	defer testutils.Clean(t)
	testutils.RequireE2EKeys(t)

	conn, err := connector.Select("openai.gpt-4o-mini")
	if err != nil {
		t.Fatalf("select connector: %v", err)
	}

	options := &context.CompletionOptions{
		Capabilities: &openai.Capabilities{
			Streaming: true,
			ToolCalls: true,
		},
	}

	inst, err := llm.New(conn, options)
	if err != nil {
		t.Fatalf("create LLM: %v", err)
	}

	messages := []context.Message{{Role: context.RoleUser, Content: "Say 'Hello' in one word."}}
	maxTok := 10
	options.MaxTokens = &maxTok
	ctx := mockTestContext("e2e-openai", "openai.gpt-4o-mini")

	var chunks []string
	handler := func(chunkType message.StreamChunkType, data []byte) int {
		chunks = append(chunks, string(data))
		return 0
	}

	deadline, cancel := gocontext.WithTimeout(gocontext.Background(), 60*time.Second)
	defer cancel()
	ctx.Context = deadline

	resp, err := inst.Stream(ctx, messages, options, handler)
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	if resp == nil || resp.Content == "" {
		t.Error("expected non-empty E2E response")
	}
}

func TestE2EDeepSeekV4Flash(t *testing.T) {
	testutils.PrepareAgent(t)
	defer testutils.Clean(t)
	testutils.RequireE2EKeys(t)

	conn, err := connector.Select("deepseek.v4-flash")
	if err != nil {
		t.Fatalf("select connector: %v", err)
	}

	options := &context.CompletionOptions{
		Capabilities: &openai.Capabilities{
			Streaming: true,
			ToolCalls: true,
		},
	}

	inst, err := llm.New(conn, options)
	if err != nil {
		t.Fatalf("create LLM: %v", err)
	}

	messages := []context.Message{{Role: context.RoleUser, Content: "Say 'Hello' in one word."}}
	maxTok := 10
	options.MaxTokens = &maxTok
	ctx := mockTestContext("e2e-deepseek", "deepseek.v4-flash")

	var chunks []string
	handler := func(chunkType message.StreamChunkType, data []byte) int {
		chunks = append(chunks, string(data))
		return 0
	}

	deadline, cancel := gocontext.WithTimeout(gocontext.Background(), 60*time.Second)
	defer cancel()
	ctx.Context = deadline

	resp, err := inst.Stream(ctx, messages, options, handler)
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	if resp == nil || resp.Content == "" {
		t.Error("expected non-empty E2E response")
	}
}
