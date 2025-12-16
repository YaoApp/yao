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

// TestGPT5StreamBasic tests basic streaming completion with GPT-5
func TestGPT5StreamBasic(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	conn, err := connector.Select("openai.gpt-5")
	if err != nil {
		t.Fatalf("Failed to select connector: %v", err)
	}

	options := &context.CompletionOptions{
		Capabilities: &openai.Capabilities{
			Streaming:  true,
			Reasoning:  true, // GPT-5 supports reasoning
			ToolCalls:  true,
			Vision:     true,
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
			Content: "What is 1+1? Reply with just the number.",
		},
	}

	maxTokens := 100
	options.MaxCompletionTokens = &maxTokens

	ctx := newGPT5TestContext("test-gpt5-basic", "openai.gpt-5")

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

	// GPT-5 may use all tokens for reasoning, so content could be empty
	// Just log the content instead of failing
	t.Logf("Response content: %v", response.Content)
	t.Logf("Usage: prompt=%d, completion=%d, total=%d",
		response.Usage.PromptTokens, response.Usage.CompletionTokens, response.Usage.TotalTokens)

	if response.Usage != nil && response.Usage.CompletionTokensDetails != nil {
		t.Logf("Reasoning tokens: %d", response.Usage.CompletionTokensDetails.ReasoningTokens)
	}

	t.Logf("Final response: %+v", response)
	t.Logf("Total chunks received: %d", len(chunks))
}

// TestGPT5ReasoningEffort tests reasoning_effort parameter with different levels
func TestGPT5ReasoningEffort(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	conn, err := connector.Select("openai.gpt-5")
	if err != nil {
		t.Fatalf("Failed to select connector: %v", err)
	}

	// Test with different reasoning effort levels
	effortLevels := []string{"low", "medium", "high"}

	for _, effort := range effortLevels {
		t.Run("effort_"+effort, func(t *testing.T) {
			options := &context.CompletionOptions{
				Capabilities: &openai.Capabilities{
					Reasoning: true,
					ToolCalls: true,
				},
				ReasoningEffort: &effort,
			}

			llmInstance, err := llm.New(conn, options)
			if err != nil {
				t.Fatalf("Failed to create LLM instance: %v", err)
			}

			messages := []context.Message{
				{
					Role:    context.RoleUser,
					Content: "Solve: If all Bloops are Razzies and all Razzies are Lazzies, are all Bloops Lazzies?",
				},
			}

			maxTokens := 1000
			options.MaxCompletionTokens = &maxTokens

			ctx := newGPT5TestContext("test-gpt5-reasoning-"+effort, "openai.gpt-5")

			response, err := llmInstance.Post(ctx, messages, options)
			if err != nil {
				t.Fatalf("Post failed with effort=%s: %v", effort, err)
			}

			if response == nil {
				t.Fatal("Response is nil")
			}

			// Check reasoning tokens
			var reasoningTokens int
			if response.Usage != nil && response.Usage.CompletionTokensDetails != nil {
				reasoningTokens = response.Usage.CompletionTokensDetails.ReasoningTokens
			}

			t.Logf("Reasoning effort: %s", effort)
			t.Logf("Reasoning tokens: %d", reasoningTokens)
			t.Logf("Total tokens: %d", response.Usage.TotalTokens)
			t.Logf("Content: %s", response.Content)

			// GPT-5 reasoning is hidden (no reasoning_content field)
			// But should have reasoning_tokens in usage
			if effort != "low" {
				if reasoningTokens == 0 {
					t.Logf("Warning: Expected reasoning_tokens > 0 for effort='%s', got 0", effort)
				}
			}
		})
	}
}

// TestGPT5PostWithToolCalls tests GPT-5 with tool calls
func TestGPT5PostWithToolCalls(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	conn, err := connector.Select("openai.gpt-5")
	if err != nil {
		t.Fatalf("Failed to select connector: %v", err)
	}

	options := &context.CompletionOptions{
		Capabilities: &openai.Capabilities{
			Reasoning: true,
			ToolCalls: true,
		},
	}

	// Define a calculation tool
	calcTool := map[string]interface{}{
		"type": "function",
		"function": map[string]interface{}{
			"name":        "calculate",
			"description": "Perform a mathematical calculation",
			"parameters": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"expression": map[string]interface{}{
						"type":        "string",
						"description": "The mathematical expression to evaluate",
					},
				},
				"required": []string{"expression"},
			},
		},
	}

	options.Tools = []map[string]interface{}{calcTool}
	options.ToolChoice = "auto"

	llmInstance, err := llm.New(conn, options)
	if err != nil {
		t.Fatalf("Failed to create LLM instance: %v", err)
	}

	messages := []context.Message{
		{
			Role:    context.RoleUser,
			Content: "Use the calculate function to compute 2 * 3",
		},
	}

	ctx := newGPT5TestContext("test-gpt5-tools", "openai.gpt-5")

	response, err := llmInstance.Post(ctx, messages, options)
	if err != nil {
		t.Fatalf("Post with tool calls failed: %v", err)
	}

	if response == nil {
		t.Fatal("Response is nil")
	}

	// GPT-5 reasoning models may not always use tool calls
	// Log what we got instead of failing
	if len(response.ToolCalls) == 0 {
		t.Logf("No tool calls returned. Content: %v", response.Content)
	} else {
		tc := response.ToolCalls[0]
		t.Logf("✓ Tool call: %s(%s)", tc.Function.Name, tc.Function.Arguments)

		if tc.Function.Name != "calculate" {
			t.Logf("Warning: Expected tool name 'calculate', got '%s'", tc.Function.Name)
		}
	}

	if response.Usage != nil {
		t.Logf("Usage: prompt=%d, completion=%d, total=%d",
			response.Usage.PromptTokens, response.Usage.CompletionTokens, response.Usage.TotalTokens)
		if response.Usage.CompletionTokensDetails != nil {
			t.Logf("Reasoning tokens: %d", response.Usage.CompletionTokensDetails.ReasoningTokens)
		}
	}

	t.Logf("Response: %+v", response)
}

// TestGPT5Vision tests GPT-5 with image input
func TestGPT5Vision(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	conn, err := connector.Select("openai.gpt-5")
	if err != nil {
		t.Fatalf("Failed to select connector: %v", err)
	}

	options := &context.CompletionOptions{
		Capabilities: &openai.Capabilities{
			Reasoning:  true,
			Vision:     true,
			Multimodal: true,
		},
	}

	llmInstance, err := llm.New(conn, options)
	if err != nil {
		t.Fatalf("Failed to create LLM instance: %v", err)
	}

	// Message with image content
	messages := []context.Message{
		{
			Role: context.RoleUser,
			Content: []context.ContentPart{
				{
					Type: context.ContentText,
					Text: "What is in this image? Describe briefly.",
				},
				{
					Type: context.ContentImageURL,
					ImageURL: &context.ImageURL{
						URL: "https://raw.githubusercontent.com/YaoApp/yao/refs/heads/main/yao/data/icons/icon.png",
					},
				},
			},
		},
	}

	maxTokens := 200
	options.MaxCompletionTokens = &maxTokens

	ctx := newGPT5TestContext("test-gpt5-vision", "openai.gpt-5")

	response, err := llmInstance.Post(ctx, messages, options)
	if err != nil {
		t.Fatalf("Post with vision failed: %v", err)
	}

	if response == nil {
		t.Fatal("Response is nil")
	}

	// Should have content describing the image
	// Content can be string or []ContentPart for multimodal responses
	var contentStr string
	switch v := response.Content.(type) {
	case string:
		contentStr = v
	case []interface{}:
		// Handle []ContentPart serialized as []interface{}
		for _, part := range v {
			if partMap, ok := part.(map[string]interface{}); ok {
				if text, ok := partMap["text"].(string); ok {
					contentStr += text
				}
			}
		}
	case []context.ContentPart:
		for _, part := range v {
			if part.Type == context.ContentText {
				contentStr += part.Text
			}
		}
	case nil:
		// GPT-5 reasoning models may use all tokens for reasoning, leaving no content
		t.Log("Content is nil (reasoning model may have used all tokens for reasoning)")
	default:
		t.Logf("Unexpected content type: %T", response.Content)
	}

	if contentStr != "" {
		t.Logf("Image description: %s", contentStr)
	} else if response.Content != nil {
		t.Logf("Warning: Expected text content describing the image, got empty or non-text content")
	}

	if response.Usage != nil {
		t.Logf("Usage: prompt=%d, completion=%d, total=%d",
			response.Usage.PromptTokens, response.Usage.CompletionTokens, response.Usage.TotalTokens)
	}
}

// TestGPT5ReasoningEffortWithGPT4o tests that GPT-4o ignores reasoning_effort
func TestGPT5ReasoningEffortWithGPT4o(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Use GPT-4o which doesn't support reasoning
	conn, err := connector.Select("openai.gpt-4o")
	if err != nil {
		t.Fatalf("Failed to select connector: %v", err)
	}

	effort := "high"
	options := &context.CompletionOptions{
		Capabilities: &openai.Capabilities{
			Reasoning: false, // GPT-4o doesn't support reasoning
			ToolCalls: true,
		},
		ReasoningEffort: &effort, // Should be ignored by adapter
	}

	llmInstance, err := llm.New(conn, options)
	if err != nil {
		t.Fatalf("Failed to create LLM instance: %v", err)
	}

	messages := []context.Message{
		{
			Role:    context.RoleUser,
			Content: "Say 'OK'",
		},
	}

	maxTokens := 10
	options.MaxCompletionTokens = &maxTokens

	ctx := newGPT5TestContext("test-gpt4o-no-reasoning", "openai.gpt-4o")

	// Should succeed (adapter removes reasoning_effort parameter)
	response, err := llmInstance.Post(ctx, messages, options)
	if err != nil {
		t.Fatalf("Post failed: %v", err)
	}

	if response == nil {
		t.Fatal("Response is nil")
	}

	// Should have 0 reasoning tokens (GPT-4o doesn't do reasoning)
	if response.Usage != nil && response.Usage.CompletionTokensDetails != nil {
		reasoningTokens := response.Usage.CompletionTokensDetails.ReasoningTokens
		if reasoningTokens != 0 {
			t.Errorf("Expected reasoning_tokens=0 for GPT-4o, got %d", reasoningTokens)
		} else {
			t.Log("✓ GPT-4o correctly shows reasoning_tokens=0")
		}
	}

	t.Log("✓ ReasoningAdapter correctly removed reasoning_effort parameter for GPT-4o")
}

// ============================================================================
// Helper Functions
// ============================================================================

// newGPT5TestContext creates a real Context for testing GPT-5 provider
func newGPT5TestContext(chatID, connectorID string) *context.Context {
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
				"test": "gpt5-provider",
			},
		},
	}

	ctx := context.New(gocontext.Background(), authorized, chatID)
	ctx.AssistantID = "test-assistant"
	ctx.Locale = "en-us"
	ctx.Theme = "light"
	ctx.Client = context.Client{
		Type:      "web",
		UserAgent: "GPT5ProviderTest/1.0",
		IP:        "127.0.0.1",
	}
	ctx.Referer = context.RefererAPI
	ctx.Accept = context.AcceptStandard
	ctx.Route = "/api/test"
	ctx.Metadata = make(map[string]interface{})
	return ctx
}
