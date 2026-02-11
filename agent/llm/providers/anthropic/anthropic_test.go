package anthropic_test

import (
	gocontext "context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/gou/connector/openai"
	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/llm"
	"github.com/yaoapp/yao/agent/output/message"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/test"
)

// testConnectorID uses the cheapest model (Claude Haiku 3) to save tokens
const testConnectorID = "claude.haiku-3_0"

// TestAnthropicStreamBasic tests basic streaming completion with Anthropic API
func TestAnthropicStreamBasic(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	conn, err := connector.Select(testConnectorID)
	if err != nil {
		t.Fatalf("Failed to select connector: %v", err)
	}

	// Verify it's an Anthropic connector
	if !conn.Is(connector.ANTHROPIC) {
		t.Fatal("Connector is not ANTHROPIC type")
	}

	// Use openai.Capabilities — SelectProvider auto-detects Anthropic format from connector type
	options := &context.CompletionOptions{
		Capabilities: &openai.Capabilities{
			Streaming: true,
			ToolCalls: true,
		},
	}

	llmInstance, err := llm.New(conn, options)
	if err != nil {
		t.Fatalf("Failed to create LLM instance: %v", err)
	}

	messages := []context.Message{
		{
			Role:    context.RoleUser,
			Content: "Say 'Hi' in one word.",
		},
	}

	maxTokens := 10
	options.MaxTokens = &maxTokens

	ctx := newTestContext("test-anthropic-stream", testConnectorID)

	var chunks []string
	handler := func(chunkType message.StreamChunkType, data []byte) int {
		chunks = append(chunks, string(data))
		t.Logf("Stream chunk [%s]: %s", chunkType, string(data))
		return 0
	}

	response, err := llmInstance.Stream(ctx, messages, options, handler)
	if err != nil {
		t.Fatalf("Stream failed: %v", err)
	}

	if response == nil {
		t.Fatal("Response is nil")
	}
	if response.ID == "" {
		t.Error("Response ID is empty")
	}
	if response.Model == "" {
		t.Error("Response Model is empty")
	}
	if response.Content == "" {
		t.Error("Response content is empty")
	}
	if response.FinishReason == "" {
		t.Error("FinishReason is empty")
	}
	if response.Usage == nil {
		t.Error("Response Usage is nil")
	} else {
		t.Logf("Usage: prompt=%d, completion=%d, total=%d",
			response.Usage.PromptTokens, response.Usage.CompletionTokens, response.Usage.TotalTokens)
	}
	if len(chunks) == 0 {
		t.Error("No streaming chunks received")
	}

	t.Logf("Final response content: %s", response.Content)
	t.Logf("Total chunks received: %d", len(chunks))
}

// TestAnthropicStreamWithToolCalls tests streaming with tool calls via Anthropic API
func TestAnthropicStreamWithToolCalls(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	conn, err := connector.Select(testConnectorID)
	if err != nil {
		t.Fatalf("Failed to select connector: %v", err)
	}

	options := &context.CompletionOptions{
		Capabilities: &openai.Capabilities{
			Streaming: true,
			ToolCalls: true,
		},
	}

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

	options.Tools = []map[string]interface{}{weatherTool}
	options.ToolChoice = "auto"

	llmInstance, err := llm.New(conn, options)
	if err != nil {
		t.Fatalf("Failed to create LLM instance: %v", err)
	}

	messages := []context.Message{
		{
			Role:    context.RoleUser,
			Content: "What's the weather in Tokyo?",
		},
	}

	ctx := newTestContext("test-anthropic-tool", testConnectorID)

	var toolCallChunks int
	handler := func(chunkType message.StreamChunkType, data []byte) int {
		if chunkType == message.ChunkToolCall {
			toolCallChunks++
		}
		t.Logf("Stream chunk [%s]: %s", chunkType, string(data))
		return 0
	}

	response, err := llmInstance.Stream(ctx, messages, options, handler)
	if err != nil {
		t.Fatalf("Stream with tool calls failed: %v", err)
	}

	if response == nil {
		t.Fatal("Response is nil")
	}

	if len(response.ToolCalls) == 0 {
		t.Error("Expected tool calls but got none")
	} else {
		t.Logf("Received %d tool call(s)", len(response.ToolCalls))
		for i, tc := range response.ToolCalls {
			t.Logf("Tool call %d: %s(%s)", i, tc.Function.Name, tc.Function.Arguments)

			if tc.ID == "" {
				t.Errorf("Tool call %d missing ID", i)
			}
			if tc.Function.Name == "" {
				t.Errorf("Tool call %d missing function name", i)
			}
			if tc.Function.Name != "get_weather" {
				t.Errorf("Tool call %d expected 'get_weather', got '%s'", i, tc.Function.Name)
			}
			if tc.Function.Arguments == "" {
				t.Errorf("Tool call %d missing arguments", i)
			}

			// Verify arguments contain location
			var args map[string]interface{}
			if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err == nil {
				if _, hasLocation := args["location"]; !hasLocation {
					t.Errorf("Tool call %d arguments missing 'location'", i)
				}
			}
		}
	}

	if response.FinishReason != context.FinishReasonToolCalls {
		t.Logf("Warning: Expected finish_reason='tool_calls', got '%s'", response.FinishReason)
	}

	t.Logf("Final response: %+v", response)
}

// TestAnthropicStreamRetry tests error handling with invalid API key
func TestAnthropicStreamRetry(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	connDSL := `{
		"type": "anthropic",
		"options": {
			"model": "claude-3-haiku-20240307",
			"key": "sk-ant-invalid-key-should-fail"
		}
	}`

	conn, err := connector.New("anthropic", "test-anthropic-retry", []byte(connDSL))
	if err != nil {
		t.Fatalf("Failed to create test connector: %v", err)
	}

	options := &context.CompletionOptions{
		Capabilities: &openai.Capabilities{
			Streaming: true,
			ToolCalls: true,
		},
	}

	llmInstance, err := llm.New(conn, options)
	if err != nil {
		t.Fatalf("Failed to create LLM instance: %v", err)
	}

	messages := []context.Message{
		{
			Role:    context.RoleUser,
			Content: "Test",
		},
	}

	ctx := newTestContext("test-anthropic-retry", "test-anthropic-retry")

	_, err = llmInstance.Stream(ctx, messages, options, nil)
	if err == nil {
		t.Fatal("Expected error due to invalid API key, but got success")
	}

	errMsg := strings.ToLower(err.Error())
	hasExpectedError := strings.Contains(errMsg, "401") ||
		strings.Contains(errMsg, "authentication") ||
		strings.Contains(errMsg, "invalid") ||
		strings.Contains(errMsg, "no data received")

	if !hasExpectedError {
		t.Errorf("Expected authentication error, got: %v", err)
	}

	t.Logf("Failed as expected with error: %v", err)
}

// ============================================================================
// Helper Functions
// ============================================================================

func newTestContext(chatID, connectorID string) *context.Context {
	authorized := &types.AuthorizedInfo{
		Subject:   "test-user",
		ClientID:  "test-client",
		UserID:    "test-user-123",
		TeamID:    "test-team-456",
		TenantID:  "test-tenant-789",
		SessionID: "test-session-id",
		Constraints: types.DataConstraints{
			TeamOnly: true,
			Extra: map[string]interface{}{
				"test": "anthropic-provider",
			},
		},
	}

	ctx := context.New(gocontext.Background(), authorized, chatID)
	ctx.AssistantID = "test-assistant"
	ctx.Locale = "en-us"
	ctx.Theme = "light"
	ctx.Client = context.Client{
		Type:      "web",
		UserAgent: "AnthropicProviderTest/1.0",
		IP:        "127.0.0.1",
	}
	ctx.Referer = context.RefererAPI
	ctx.Accept = context.AcceptStandard
	ctx.Route = "/api/test"
	ctx.Metadata = make(map[string]interface{})
	return ctx
}
