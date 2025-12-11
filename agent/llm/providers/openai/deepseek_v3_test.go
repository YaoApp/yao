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

// TestDeepSeekV3StreamBasic tests basic streaming completion with DeepSeek V3
func TestDeepSeekV3StreamBasic(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	conn, err := connector.Select("deepseek.v3")
	if err != nil {
		t.Fatalf("Failed to select connector: %v", err)
	}

	options := &context.CompletionOptions{
		Capabilities: &openai.Capabilities{
			Streaming:  true,
			Reasoning:  false, // V3 doesn't support reasoning
			ToolCalls:  true,  // V3 supports tool calls
			Vision:     false,
			Audio:      false,
			Multimodal: false,
		},
	}

	llmInstance, err := llm.New(conn, options)
	if err != nil {
		t.Fatalf("Failed to create LLM instance: %v", err)
	}

	// Simple math question
	messages := []context.Message{
		{
			Role:    context.RoleUser,
			Content: "What is 5 + 3?",
		},
	}

	// Set max tokens
	maxTokens := 100
	options.MaxTokens = &maxTokens

	ctx := newDeepSeekV3TestContext("test-deepseek-v3-basic", "deepseek.v3")

	// Track streaming chunks
	var contentChunks []string
	handler := func(chunkType message.StreamChunkType, data []byte) int {
		dataStr := string(data)
		t.Logf("Stream chunk [%s]: %s", chunkType, dataStr)

		if chunkType == message.ChunkText {
			contentChunks = append(contentChunks, dataStr)
		}

		return 0 // Continue
	}

	// Call Stream
	response, err := llmInstance.Stream(ctx, messages, options, handler)
	if err != nil {
		t.Fatalf("Stream failed: %v", err)
	}

	// Validate response
	if response == nil {
		t.Fatal("Response is nil")
	}

	if response.ID == "" {
		t.Error("Response ID is empty")
	}
	if response.Model == "" {
		t.Error("Response Model is empty")
	}

	// Should have content (V3 is not a reasoning model)
	contentStr, ok := response.Content.(string)
	if !ok || contentStr == "" {
		t.Error("Expected content but got empty")
	} else {
		t.Logf("Response content: %s", contentStr)
	}

	// Should NOT have reasoning content (V3 doesn't support reasoning)
	if response.ReasoningContent != "" {
		t.Errorf("Expected no reasoning_content for V3, but got: %s", response.ReasoningContent)
	}

	// Check usage
	if response.Usage == nil {
		t.Error("Response Usage is nil")
	} else {
		t.Logf("Usage: prompt=%d, completion=%d, total=%d",
			response.Usage.PromptTokens, response.Usage.CompletionTokens, response.Usage.TotalTokens)

		// Should have 0 reasoning tokens
		if response.Usage.CompletionTokensDetails != nil {
			reasoningTokens := response.Usage.CompletionTokensDetails.ReasoningTokens
			if reasoningTokens != 0 {
				t.Errorf("Expected reasoning_tokens=0 for V3, got %d", reasoningTokens)
			}
		}
	}

	if len(contentChunks) == 0 {
		t.Error("Expected content chunks but got none")
	} else {
		t.Logf("Received %d content chunks", len(contentChunks))
	}

	t.Logf("Final response: %+v", response)
}

// TestDeepSeekV3PostBasic tests basic non-streaming completion
func TestDeepSeekV3PostBasic(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	conn, err := connector.Select("deepseek.v3")
	if err != nil {
		t.Fatalf("Failed to select connector: %v", err)
	}

	options := &context.CompletionOptions{
		Capabilities: &openai.Capabilities{
			Reasoning:  false,
			ToolCalls:  true,
			Vision:     false,
			Audio:      false,
			Multimodal: false,
		},
	}

	llmInstance, err := llm.New(conn, options)
	if err != nil {
		t.Fatalf("Failed to create LLM instance: %v", err)
	}

	// Simple question
	messages := []context.Message{
		{
			Role:    context.RoleUser,
			Content: "What is 2 * 4?",
		},
	}

	// Set max tokens
	maxTokens := 100
	options.MaxTokens = &maxTokens

	ctx := newDeepSeekV3TestContext("test-deepseek-v3-post", "deepseek.v3")

	// Call Post
	response, err := llmInstance.Post(ctx, messages, options)
	if err != nil {
		t.Fatalf("Post failed: %v", err)
	}

	// Validate response
	if response == nil {
		t.Fatal("Response is nil")
	}

	if response.ID == "" {
		t.Error("Response ID is empty")
	}
	if response.Model == "" {
		t.Error("Response Model is empty")
	}

	// Should have content
	contentStr, ok := response.Content.(string)
	if !ok || contentStr == "" {
		t.Error("Expected content but got empty")
	} else {
		t.Logf("Response content: %s", contentStr)
	}

	// Should NOT have reasoning content
	if response.ReasoningContent != "" {
		t.Errorf("V3 should not have reasoning_content, but got: %s", response.ReasoningContent)
	}

	// Check usage
	if response.Usage == nil {
		t.Error("Response Usage is nil")
	} else {
		t.Logf("Usage: prompt=%d, completion=%d, total=%d",
			response.Usage.PromptTokens, response.Usage.CompletionTokens, response.Usage.TotalTokens)

		// Should have 0 reasoning tokens
		if response.Usage.CompletionTokensDetails != nil {
			reasoningTokens := response.Usage.CompletionTokensDetails.ReasoningTokens
			if reasoningTokens != 0 {
				t.Errorf("Expected reasoning_tokens=0 for V3, got %d", reasoningTokens)
			}
		}
	}

	t.Logf("Response: %+v", response)
}

// TestDeepSeekV3WithToolCalls tests V3 with tool calls
func TestDeepSeekV3WithToolCalls(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	conn, err := connector.Select("deepseek.v3")
	if err != nil {
		t.Fatalf("Failed to select connector: %v", err)
	}

	options := &context.CompletionOptions{
		Capabilities: &openai.Capabilities{
			Reasoning: false,
			ToolCalls: true,
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

	// Set lower max_tokens for faster response
	maxTokens := 50
	options.MaxTokens = &maxTokens

	llmInstance, err := llm.New(conn, options)
	if err != nil {
		t.Fatalf("Failed to create LLM instance: %v", err)
	}

	messages := []context.Message{
		{
			Role:    context.RoleUser,
			Content: "Call get_info with query='A' and count=1",
		},
	}

	ctx := newDeepSeekV3TestContext("test-deepseek-v3-tools", "deepseek.v3")

	response, err := llmInstance.Post(ctx, messages, options)
	if err != nil {
		t.Fatalf("Post with tool calls failed: %v", err)
	}

	if response == nil {
		t.Fatal("Response is nil")
	}

	// Should have tool calls
	if len(response.ToolCalls) == 0 {
		t.Error("Expected tool calls but got none")
	} else {
		tc := response.ToolCalls[0]
		t.Logf("✓ Tool call: %s(%s)", tc.Function.Name, tc.Function.Arguments)

		if tc.Function.Name != "get_info" {
			t.Errorf("Expected tool name 'get_info', got '%s'", tc.Function.Name)
		}
	}

	if response.Usage != nil {
		t.Logf("Usage: prompt=%d, completion=%d, total=%d",
			response.Usage.PromptTokens, response.Usage.CompletionTokens, response.Usage.TotalTokens)
	}

	t.Logf("Response: %+v", response)
}

// TestDeepSeekV3NoReasoningEffort tests that V3 ignores reasoning_effort parameter
func TestDeepSeekV3NoReasoningEffort(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	conn, err := connector.Select("deepseek.v3")
	if err != nil {
		t.Fatalf("Failed to select connector: %v", err)
	}

	effort := "high"
	options := &context.CompletionOptions{
		Capabilities: &openai.Capabilities{
			Reasoning: false, // V3 doesn't support reasoning
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
			Content: "Reply with just: OK",
		},
	}

	maxTokens := 20
	options.MaxTokens = &maxTokens

	ctx := newDeepSeekV3TestContext("test-deepseek-v3-no-reasoning", "deepseek.v3")

	// Should succeed (adapter removes reasoning_effort parameter)
	response, err := llmInstance.Post(ctx, messages, options)
	if err != nil {
		t.Fatalf("Post failed: %v", err)
	}

	if response == nil {
		t.Fatal("Response is nil")
	}

	// Should have 0 reasoning tokens
	if response.Usage != nil && response.Usage.CompletionTokensDetails != nil {
		reasoningTokens := response.Usage.CompletionTokensDetails.ReasoningTokens
		if reasoningTokens != 0 {
			t.Errorf("Expected reasoning_tokens=0 for V3, got %d", reasoningTokens)
		} else {
			t.Log("✓ V3 correctly shows reasoning_tokens=0")
		}
	}

	t.Log("✓ ReasoningAdapter correctly removed reasoning_effort parameter for V3")
}

// ============================================================================
// Helper Functions
// ============================================================================

// newDeepSeekV3TestContext creates a real Context for testing DeepSeek V3 provider
func newDeepSeekV3TestContext(chatID, connectorID string) *context.Context {
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
				"test": "deepseek-v3-provider",
			},
		},
	}

	ctx := context.New(gocontext.Background(), authorized, chatID)
	ctx.AssistantID = "test-assistant"
	ctx.Locale = "en-us"
	ctx.Theme = "light"
	ctx.Client = context.Client{
		Type:      "web",
		UserAgent: "DeepSeekV3ProviderTest/1.0",
		IP:        "127.0.0.1",
	}
	ctx.Referer = context.RefererAPI
	ctx.Accept = context.AcceptStandard
	ctx.Route = "/api/test"
	ctx.Metadata = make(map[string]interface{})
	return ctx
}
