package openai_test

import (
	gocontext "context"
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

// newClaudeTestContext creates a real Context for testing Claude provider
func newClaudeTestContext(chatID, connectorID string) *context.Context {
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
				"test": "claude-provider",
			},
		},
	}

	ctx := context.New(gocontext.Background(), authorized, chatID)
	ctx.AssistantID = "test-assistant"
	ctx.Locale = "en-us"
	ctx.Theme = "light"
	ctx.Client = context.Client{
		Type:      "web",
		UserAgent: "ClaudeProviderTest/1.0",
		IP:        "127.0.0.1",
	}
	ctx.Referer = context.RefererAPI
	ctx.Accept = context.AcceptStandard
	ctx.Route = "/api/test"
	ctx.Metadata = make(map[string]interface{})
	return ctx
}

// TestClaudeSonnet4StreamBasic tests basic streaming completion with Claude Sonnet 4
func TestClaudeSonnet4StreamBasic(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	conn, err := connector.Select("claude.sonnet-4_0")
	if err != nil {
		t.Fatalf("Failed to select connector: %v", err)
	}

	options := &context.CompletionOptions{
		Capabilities: &openai.Capabilities{
			Streaming:  true,
			Reasoning:  false, // Claude Sonnet 4 (non-thinking) doesn't expose reasoning
			ToolCalls:  true,
			Vision:     "claude", // Claude requires base64 format
			Multimodal: true,
		},
	}

	llmInstance, err := llm.New(conn, options)
	if err != nil {
		t.Fatalf("Failed to create LLM instance: %v", err)
	}

	messages := []context.Message{
		{
			Role:    context.RoleUser,
			Content: "What is 3+3? Reply with just the number.",
		},
	}

	maxTokens := 100
	options.MaxTokens = &maxTokens

	ctx := newClaudeTestContext("test-claude-sonnet4-basic", "claude.sonnet-4_0")

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

	// Basic validation
	if response.ID == "" {
		t.Error("Response ID is empty")
	}
	if response.Model == "" {
		t.Error("Response Model is empty")
	}

	// Validate content
	contentStr, ok := response.Content.(string)
	if !ok {
		t.Errorf("Content is not a string: %T", response.Content)
	}
	if len(contentStr) == 0 {
		t.Error("Content is empty")
	}

	t.Logf("Response content: %v", response.Content)
	t.Logf("Usage: prompt=%d, completion=%d, total=%d",
		response.Usage.PromptTokens, response.Usage.CompletionTokens, response.Usage.TotalTokens)

	t.Logf("Final response: %+v", response)
	t.Logf("Total chunks received: %d", len(chunks))
}

// TestClaudeSonnet4PostBasic tests non-streaming completion
func TestClaudeSonnet4PostBasic(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	conn, err := connector.Select("claude.sonnet-4_0")
	if err != nil {
		t.Fatalf("Failed to select connector: %v", err)
	}

	options := &context.CompletionOptions{
		Capabilities: &openai.Capabilities{
			Streaming:  false,
			Reasoning:  false,
			ToolCalls:  true,
			Vision:     "claude", // Claude requires base64 format
			Multimodal: true,
		},
	}

	llmInstance, err := llm.New(conn, options)
	if err != nil {
		t.Fatalf("Failed to create LLM instance: %v", err)
	}

	messages := []context.Message{
		{
			Role:    context.RoleUser,
			Content: "What is 4+4? Reply with just the number.",
		},
	}

	maxTokens := 100
	options.MaxTokens = &maxTokens

	ctx := newClaudeTestContext("test-claude-sonnet4-post", "claude.sonnet-4_0")

	response, err := llmInstance.Post(ctx, messages, options)
	if err != nil {
		t.Fatalf("Post failed: %v", err)
	}

	if response == nil {
		t.Fatal("Response is nil")
	}

	// Validate content
	contentStr, ok := response.Content.(string)
	if !ok {
		t.Fatalf("Content is not a string: %T", response.Content)
	}

	t.Logf("Response content: %s", contentStr)
	t.Logf("Usage: prompt=%d, completion=%d, total=%d",
		response.Usage.PromptTokens, response.Usage.CompletionTokens, response.Usage.TotalTokens)

	// Basic content validation
	if len(contentStr) == 0 {
		t.Error("Content is empty")
	}

	t.Logf("Response: %+v", response)
}

// TestClaudeSonnet4WithToolCalls tests tool calling capability
func TestClaudeSonnet4WithToolCalls(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	conn, err := connector.Select("claude.sonnet-4_0")
	if err != nil {
		t.Fatalf("Failed to select connector: %v", err)
	}

	options := &context.CompletionOptions{
		Capabilities: &openai.Capabilities{
			Streaming:  false,
			Reasoning:  false,
			ToolCalls:  true,
			Vision:     "claude", // Claude requires base64 format
			Multimodal: true,
		},
	}

	// Define a simple tool with minimal parameters
	simpleTool := map[string]interface{}{
		"type": "function",
		"function": map[string]interface{}{
			"name":        "get_info",
			"description": "Get information",
			"parameters": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "Query string (single letter)",
					},
					"count": map[string]interface{}{
						"type":        "number",
						"description": "Count (single digit)",
					},
				},
				"required": []string{"query", "count"},
			},
		},
	}

	options.Tools = []map[string]interface{}{simpleTool}
	options.ToolChoice = "auto"

	// Set enough tokens for tool call response
	maxTokens := 150
	options.MaxTokens = &maxTokens

	llmInstance, err := llm.New(conn, options)
	if err != nil {
		t.Fatalf("Failed to create LLM instance: %v", err)
	}

	messages := []context.Message{
		{
			Role:    context.RoleUser,
			Content: "Please use the get_info function to retrieve information. Pass 'A' as the query parameter and 1 as the count parameter.",
		},
	}

	ctx := newClaudeTestContext("test-claude-sonnet4-tools", "claude.sonnet-4_0")

	response, err := llmInstance.Post(ctx, messages, options)
	if err != nil {
		t.Fatalf("Post failed: %v", err)
	}

	if response == nil {
		t.Fatal("Response is nil")
	}

	// Validate tool calls
	if len(response.ToolCalls) == 0 {
		t.Error("No tool calls in response")
	} else {
		tc := response.ToolCalls[0]
		t.Logf("✓ Tool call: %s(%s)", tc.Function.Name, tc.Function.Arguments)

		if tc.Function.Name != "get_info" {
			t.Errorf("Expected tool name 'get_info', got '%s'", tc.Function.Name)
		}
	}

	t.Logf("Usage: prompt=%d, completion=%d, total=%d",
		response.Usage.PromptTokens, response.Usage.CompletionTokens, response.Usage.TotalTokens)

	t.Logf("Response: %+v", response)
}

// TestClaudeSonnet4Vision tests vision capability with image input
func TestClaudeSonnet4Vision(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	conn, err := connector.Select("claude.sonnet-4_0")
	if err != nil {
		t.Fatalf("Failed to select connector: %v", err)
	}

	options := &context.CompletionOptions{
		Capabilities: &openai.Capabilities{
			Streaming:  false,
			Reasoning:  false,
			ToolCalls:  true,
			Vision:     "claude", // Claude requires base64 format
			Multimodal: true,
		},
	}

	llmInstance, err := llm.New(conn, options)
	if err != nil {
		t.Fatalf("Failed to create LLM instance: %v", err)
	}

	// Use a test image URL
	imageURL := "https://upload.wikimedia.org/wikipedia/commons/thumb/d/dd/Gfp-wisconsin-madison-the-nature-boardwalk.jpg/2560px-Gfp-wisconsin-madison-the-nature-boardwalk.jpg"

	messages := []context.Message{
		{
			Role: context.RoleUser,
			Content: []map[string]interface{}{
				{
					"type": "text",
					"text": "Describe this image in one sentence.",
				},
				{
					"type": "image_url",
					"image_url": map[string]string{
						"url": imageURL,
					},
				},
			},
		},
	}

	maxTokens := 150
	options.MaxTokens = &maxTokens

	ctx := newClaudeTestContext("test-claude-sonnet4-vision", "claude.sonnet-4_0")

	response, err := llmInstance.Post(ctx, messages, options)
	if err != nil {
		t.Fatalf("Post failed: %v", err)
	}

	if response == nil {
		t.Fatal("Response is nil")
	}

	// Validate content
	contentStr, ok := response.Content.(string)
	if !ok {
		t.Fatalf("Content is not a string: %T", response.Content)
	}

	if len(contentStr) == 0 {
		t.Error("Image description is empty")
	}

	t.Logf("Image description: %s", contentStr)
	t.Logf("Usage: prompt=%d, completion=%d, total=%d",
		response.Usage.PromptTokens, response.Usage.CompletionTokens, response.Usage.TotalTokens)
}

// TestClaudeSonnet4ThinkingStream tests Claude Sonnet 4 Thinking with streaming
func TestClaudeSonnet4ThinkingStream(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	conn, err := connector.Select("claude.sonnet-4_0-thinking")
	if err != nil {
		t.Fatalf("Failed to select connector: %v", err)
	}

	options := &context.CompletionOptions{
		Capabilities: &openai.Capabilities{
			Streaming:  true,
			Reasoning:  true, // Claude Thinking mode exposes reasoning
			ToolCalls:  false,
			Vision:     "claude", // Claude requires base64 format
			Multimodal: true,
		},
	}

	llmInstance, err := llm.New(conn, options)
	if err != nil {
		t.Fatalf("Failed to create LLM instance: %v", err)
	}

	messages := []context.Message{
		{
			Role:    context.RoleUser,
			Content: "If Sally has 3 apples and gives 2 to John, how many does she have left? Think through this step by step.",
		},
	}

	maxTokens := 500
	options.MaxTokens = &maxTokens

	ctx := newClaudeTestContext("test-claude-thinking-stream", "claude.sonnet-4_0-thinking")

	var thinkingChunks []string
	var textChunks []string
	handler := func(chunkType message.StreamChunkType, data []byte) int {
		t.Logf("Stream chunk [%s]: %s", chunkType, string(data))
		if chunkType == message.ChunkThinking {
			thinkingChunks = append(thinkingChunks, string(data))
		} else if chunkType == message.ChunkText {
			textChunks = append(textChunks, string(data))
		}
		return 0
	}

	response, err := llmInstance.Stream(ctx, messages, options, handler)
	if err != nil {
		t.Fatalf("Stream failed: %v", err)
	}

	if response == nil {
		t.Fatal("Response is nil")
	}

	// Validate response
	contentStr, ok := response.Content.(string)
	if !ok {
		t.Errorf("Content is not a string: %T", response.Content)
	}

	t.Logf("Reasoning/Thinking content length: %d characters", len(response.ReasoningContent))
	t.Logf("Response content: %v", contentStr)
	t.Logf("Usage: prompt=%d, completion=%d, total=%d",
		response.Usage.PromptTokens, response.Usage.CompletionTokens, response.Usage.TotalTokens)

	if response.Usage != nil && response.Usage.CompletionTokensDetails != nil {
		t.Logf("Reasoning tokens: %d", response.Usage.CompletionTokensDetails.ReasoningTokens)
	}

	t.Logf("Received %d thinking chunks", len(thinkingChunks))
	t.Logf("Received %d text chunks", len(textChunks))
	t.Logf("Final response: %+v", response)
}

// TestClaudeSonnet4ThinkingPost tests Claude Sonnet 4 Thinking in non-streaming mode
func TestClaudeSonnet4ThinkingPost(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	conn, err := connector.Select("claude.sonnet-4_0-thinking")
	if err != nil {
		t.Fatalf("Failed to select connector: %v", err)
	}

	options := &context.CompletionOptions{
		Capabilities: &openai.Capabilities{
			Streaming:  false,
			Reasoning:  true,
			ToolCalls:  false,
			Vision:     "claude", // Claude requires base64 format
			Multimodal: true,
		},
	}

	llmInstance, err := llm.New(conn, options)
	if err != nil {
		t.Fatalf("Failed to create LLM instance: %v", err)
	}

	messages := []context.Message{
		{
			Role:    context.RoleUser,
			Content: "Is 7 greater than 5? Explain your reasoning.",
		},
	}

	maxTokens := 500
	options.MaxTokens = &maxTokens

	ctx := newClaudeTestContext("test-claude-thinking-post", "claude.sonnet-4_0-thinking")

	response, err := llmInstance.Post(ctx, messages, options)
	if err != nil {
		t.Fatalf("Post failed: %v", err)
	}

	if response == nil {
		t.Fatal("Response is nil")
	}

	// Validate content
	contentStr, ok := response.Content.(string)
	if !ok {
		t.Fatalf("Content is not a string: %T", response.Content)
	}

	t.Logf("Reasoning content: %s", response.ReasoningContent)
	t.Logf("Final answer: %s", contentStr)

	// Check for reasoning content
	if len(response.ReasoningContent) > 0 {
		t.Logf("✓ Reasoning content present: %d characters", len(response.ReasoningContent))
	}

	if response.Usage != nil && response.Usage.CompletionTokensDetails != nil && response.Usage.CompletionTokensDetails.ReasoningTokens > 0 {
		t.Logf("✓ Reasoning tokens: %d", response.Usage.CompletionTokensDetails.ReasoningTokens)
	}

	t.Logf("Usage: prompt=%d, completion=%d, total=%d",
		response.Usage.PromptTokens, response.Usage.CompletionTokens, response.Usage.TotalTokens)

	t.Logf("Response: %+v", response)
}

// TestClaudeTemperatureHandling tests that Claude models handle temperature parameter correctly
func TestClaudeTemperatureHandling(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	tests := []struct {
		name        string
		connector   string
		temperature float64
		reasoning   bool
	}{
		{
			name:        "Sonnet 4 with temperature 0.7",
			connector:   "claude.sonnet-4_0",
			temperature: 0.7,
			reasoning:   false,
		},
		{
			name:        "Sonnet 4 Thinking with temperature 0.5",
			connector:   "claude.sonnet-4_0-thinking",
			temperature: 0.5,
			reasoning:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn, err := connector.Select(tt.connector)
			if err != nil {
				t.Fatalf("Failed to select connector: %v", err)
			}

			options := &context.CompletionOptions{
				Capabilities: &openai.Capabilities{
					Streaming: false,
					Reasoning: tt.reasoning,
					ToolCalls: true,
					Vision:    true,
				},
			}

			llmInstance, err := llm.New(conn, options)
			if err != nil {
				t.Fatalf("Failed to create LLM instance: %v", err)
			}

			messages := []context.Message{
				{
					Role:    context.RoleUser,
					Content: "Say 'hello'.",
				},
			}

			maxTokens := 50
			options.MaxTokens = &maxTokens
			options.Temperature = &tt.temperature

			ctx := newClaudeTestContext("test-claude-temp-"+tt.connector, tt.connector)

			response, err := llmInstance.Post(ctx, messages, options)
			if err != nil {
				t.Fatalf("Post failed: %v", err)
			}

			if response == nil {
				t.Fatal("Response is nil")
			}

			t.Logf("✓ %s completed successfully with temperature=%.1f", tt.name, tt.temperature)
		})
	}
}
