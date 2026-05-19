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
	"github.com/yaoapp/yao/openapi/oauth/types"
)

func TestLLMMockOpenAIStreamEcho(t *testing.T) {
	testutils.PrepareAgent(t)
	defer testutils.Clean(t)
	testutils.SkipWithoutMockLLM(t)

	conn, err := connector.Select("openai.mock")
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

	messages := []context.Message{{Role: context.RoleUser, Content: "hello from mock test"}}
	maxTok := 100
	options.MaxTokens = &maxTok
	ctx := mockTestContext("mock-openai-echo", "openai.mock")

	var chunks []string
	handler := func(chunkType message.StreamChunkType, data []byte) int {
		chunks = append(chunks, string(data))
		return 0
	}

	deadline, cancel := gocontext.WithTimeout(gocontext.Background(), 30*time.Second)
	defer cancel()
	ctx.Context = deadline

	resp, err := inst.Stream(ctx, messages, options, handler)
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}

	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if len(chunks) == 0 {
		t.Error("no streaming chunks received")
	}
}

func TestLLMMockOpenAIPost(t *testing.T) {
	testutils.PrepareAgent(t)
	defer testutils.Clean(t)
	testutils.SkipWithoutMockLLM(t)

	conn, err := connector.Select("openai.mock")
	if err != nil {
		t.Fatalf("select connector: %v", err)
	}

	options := &context.CompletionOptions{
		Capabilities: &openai.Capabilities{
			ToolCalls: true,
		},
	}

	inst, err := llm.New(conn, options)
	if err != nil {
		t.Fatalf("create LLM: %v", err)
	}

	messages := []context.Message{{Role: context.RoleUser, Content: "hello non-stream"}}
	maxTok := 100
	options.MaxTokens = &maxTok
	ctx := mockTestContext("mock-openai-post", "openai.mock")

	deadline, cancel := gocontext.WithTimeout(gocontext.Background(), 30*time.Second)
	defer cancel()
	ctx.Context = deadline

	resp, err := inst.Post(ctx, messages, options)
	if err != nil {
		t.Fatalf("Post: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
}

func TestLLMMockDeepSeekThinking(t *testing.T) {
	testutils.PrepareAgent(t)
	defer testutils.Clean(t)
	testutils.SkipWithoutMockLLM(t)

	conn, err := connector.Select("deepseek.mock-think")
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

	messages := []context.Message{{Role: context.RoleUser, Content: "think about something"}}
	ctx := mockTestContext("mock-deepseek-think", "deepseek.mock-think")

	var chunks []string
	handler := func(chunkType message.StreamChunkType, data []byte) int {
		chunks = append(chunks, string(data))
		return 0
	}

	deadline, cancel := gocontext.WithTimeout(gocontext.Background(), 30*time.Second)
	defer cancel()
	ctx.Context = deadline

	resp, err := inst.Stream(ctx, messages, options, handler)
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
}

func mockTestContext(chatID, connectorID string) *context.Context {
	authorized := &types.AuthorizedInfo{
		Subject:   "test-user",
		ClientID:  "test-client",
		UserID:    "test-user-123",
		TeamID:    "test-team-456",
		TenantID:  "test-tenant-789",
		SessionID: "test-session-id",
		Constraints: types.DataConstraints{
			TeamOnly: true,
			Extra:    map[string]interface{}{"test": "mock-provider"},
		},
	}

	ctx := context.New(gocontext.Background(), authorized, chatID)
	ctx.AssistantID = "test-assistant"
	ctx.Locale = "en-us"
	ctx.Theme = "light"
	ctx.Client = context.Client{
		Type:      "web",
		UserAgent: "MockProviderTest/1.0",
		IP:        "127.0.0.1",
	}
	ctx.Referer = context.RefererAPI
	ctx.Accept = context.AcceptStandard
	ctx.Route = "/api/test"
	ctx.Metadata = make(map[string]interface{})
	return ctx
}
