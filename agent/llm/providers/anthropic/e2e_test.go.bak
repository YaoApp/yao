//go:build e2e

package anthropic_test

import (
	gocontext "context"
	"testing"
	"time"

	"github.com/yaoapp/gou/connector"
	goullm "github.com/yaoapp/gou/llm"
	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/llm"
	"github.com/yaoapp/yao/agent/output/message"
	"github.com/yaoapp/yao/agent/testutils"
)

func TestE2EAnthropicHaiku(t *testing.T) {
	testutils.PrepareAgent(t)
	defer testutils.Clean(t)
	testutils.RequireE2EKeys(t)

	conn, err := connector.Select("anthropic.haiku")
	if err != nil {
		t.Fatalf("select connector: %v", err)
	}

	if !conn.Is(connector.ANTHROPIC) {
		t.Fatal("expected ANTHROPIC type connector")
	}

	options := &context.CompletionOptions{
		Capabilities: &goullm.Capabilities{
			Streaming: true,
			ToolCalls: true,
		},
	}

	inst, err := llm.New(conn, options)
	if err != nil {
		t.Fatalf("create LLM: %v", err)
	}

	messages := []context.Message{{Role: context.RoleUser, Content: "Say 'Hi' in one word."}}
	maxTok := 10
	options.MaxTokens = &maxTok
	ctx := mockAnthropicTestContext("e2e-anthropic", "anthropic.haiku")

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
