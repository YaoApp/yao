package openai_test

import (
	gocontext "context"
	"strings"
	"testing"

	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/gou/plan"
	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/llm"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/test"
)

// TestDeepSeekR1StreamBasic tests basic streaming completion with DeepSeek R1
func TestDeepSeekR1StreamBasic(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Create connector from real configuration
	conn, err := connector.Select("deepseek.r1")
	if err != nil {
		t.Fatalf("Failed to select connector: %v", err)
	}

	// Create LLM instance with capabilities
	trueVal := true
	falseVal := false
	options := &context.CompletionOptions{
		Capabilities: &context.ModelCapabilities{
			Streaming:  &trueVal,
			Reasoning:  &trueVal,  // DeepSeek R1 supports reasoning
			ToolCalls:  &falseVal, // R1 doesn't support native tool calls
			Vision:     &falseVal,
			Audio:      &falseVal,
			Multimodal: &falseVal,
		},
	}

	llmInstance, err := llm.New(conn, options)
	if err != nil {
		t.Fatalf("Failed to create LLM instance: %v", err)
	}

	// Prepare messages with reasoning prompt (simple question for faster reasoning)
	messages := []context.Message{
		{
			Role:    context.RoleUser,
			Content: "What is 2 + 2?",
		},
	}

	// Set max tokens (higher for reasoning models to allow full reasoning + answer)
	maxTokens := 500
	options.MaxTokens = &maxTokens

	// Create context
	ctx := newDeepSeekTestContext("test-deepseek-r1-basic", "deepseek.r1")

	// Track streaming chunks
	var reasoningChunks []string
	var contentChunks []string
	handler := func(chunkType context.StreamChunkType, data []byte) int {
		dataStr := string(data)
		t.Logf("Stream chunk [%s]: %s", chunkType, dataStr)

		// Track different chunk types
		if chunkType == context.ChunkThinking {
			reasoningChunks = append(reasoningChunks, dataStr)
		} else if chunkType == context.ChunkText {
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
	if response.Content == "" {
		t.Error("Response content is empty")
	}
	if response.FinishReason == "" {
		t.Error("FinishReason is empty")
	}

	// DeepSeek R1 should have reasoning content
	if response.ReasoningContent == "" {
		t.Error("Expected reasoning_content but got empty")
	} else {
		t.Logf("Reasoning content length: %d characters", len(response.ReasoningContent))
	}

	// Check reasoning tokens in usage
	if response.Usage == nil {
		t.Error("Response Usage is nil")
	} else {
		if response.Usage.TotalTokens == 0 {
			t.Error("Response Usage.TotalTokens is 0")
		}
		if response.Usage.CompletionTokensDetails != nil {
			if response.Usage.CompletionTokensDetails.ReasoningTokens == 0 {
				t.Error("Expected reasoning_tokens > 0 for DeepSeek R1")
			} else {
				t.Logf("Reasoning tokens: %d", response.Usage.CompletionTokensDetails.ReasoningTokens)
			}
		}
		t.Logf("Usage: prompt=%d, completion=%d, total=%d",
			response.Usage.PromptTokens, response.Usage.CompletionTokens, response.Usage.TotalTokens)
	}

	// Should have received reasoning chunks
	if len(reasoningChunks) == 0 {
		t.Error("Expected reasoning chunks (ChunkThinking) but got none")
	} else {
		t.Logf("Received %d reasoning chunks", len(reasoningChunks))
	}

	// Should have received content chunks
	if len(contentChunks) == 0 {
		t.Error("Expected content chunks (ChunkText) but got none")
	} else {
		t.Logf("Received %d content chunks", len(contentChunks))
	}

	t.Logf("Final response: %+v", response)
}

// TestDeepSeekR1PostBasic tests basic non-streaming completion
func TestDeepSeekR1PostBasic(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Create connector
	conn, err := connector.Select("deepseek.r1")
	if err != nil {
		t.Fatalf("Failed to select connector: %v", err)
	}

	// Create LLM instance
	trueVal := true
	falseVal := false
	options := &context.CompletionOptions{
		Capabilities: &context.ModelCapabilities{
			Reasoning:  &trueVal,
			ToolCalls:  &falseVal,
			Vision:     &falseVal,
			Audio:      &falseVal,
			Multimodal: &falseVal,
		},
	}

	llmInstance, err := llm.New(conn, options)
	if err != nil {
		t.Fatalf("Failed to create LLM instance: %v", err)
	}

	// Prepare messages (very simple question)
	messages := []context.Message{
		{
			Role:    context.RoleUser,
			Content: "What is 1+1?",
		},
	}

	// Set max tokens (enough for reasoning + answer)
	maxTokens := 500
	options.MaxTokens = &maxTokens

	// Create context
	ctx := newDeepSeekTestContext("test-deepseek-r1-post", "deepseek.r1")

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
	if response.Content == "" {
		t.Error("Response content is empty")
	}

	// DeepSeek R1 should have reasoning content
	if response.ReasoningContent == "" {
		t.Error("Expected reasoning_content but got empty")
	} else {
		t.Logf("Reasoning content: %s", response.ReasoningContent)
		t.Logf("Final answer: %s", response.Content)
	}

	// Check usage
	if response.Usage == nil {
		t.Error("Response Usage is nil")
	} else {
		if response.Usage.TotalTokens == 0 {
			t.Error("Response Usage.TotalTokens is 0")
		}
		if response.Usage.CompletionTokensDetails != nil && response.Usage.CompletionTokensDetails.ReasoningTokens > 0 {
			t.Logf("Reasoning tokens: %d", response.Usage.CompletionTokensDetails.ReasoningTokens)
		}
		t.Logf("Usage: prompt=%d, completion=%d, total=%d",
			response.Usage.PromptTokens, response.Usage.CompletionTokens, response.Usage.TotalTokens)
	}

	t.Logf("Response: %+v", response)
}

// TestDeepSeekR1LogicPuzzle tests DeepSeek R1's reasoning with a logic puzzle
func TestDeepSeekR1LogicPuzzle(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	conn, err := connector.Select("deepseek.r1")
	if err != nil {
		t.Fatalf("Failed to select connector: %v", err)
	}

	trueVal := true
	falseVal := false
	options := &context.CompletionOptions{
		Capabilities: &context.ModelCapabilities{
			Streaming:  &trueVal,
			Reasoning:  &trueVal,
			ToolCalls:  &falseVal,
			Vision:     &falseVal,
			Audio:      &falseVal,
			Multimodal: &falseVal,
		},
	}

	maxTokens := 800
	options.MaxTokens = &maxTokens

	llmInstance, err := llm.New(conn, options)
	if err != nil {
		t.Fatalf("Failed to create LLM instance: %v", err)
	}

	// Use a simpler logic question
	messages := []context.Message{
		{
			Role:    context.RoleUser,
			Content: "Is 5 greater than 3? Explain your reasoning.",
		},
	}

	ctx := newDeepSeekTestContext("test-deepseek-r1-logic", "deepseek.r1")

	// Track reasoning and content separately
	var hasReasoning, hasContent bool
	handler := func(chunkType context.StreamChunkType, data []byte) int {
		if chunkType == context.ChunkThinking && len(data) > 0 {
			hasReasoning = true
		} else if chunkType == context.ChunkText && len(data) > 0 {
			hasContent = true
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

	// Should have both reasoning and content
	if !hasReasoning {
		t.Error("Expected to receive reasoning chunks but didn't")
	}
	if !hasContent {
		t.Error("Expected to receive content chunks but didn't")
	}

	// Validate reasoning content exists and is substantial
	if response.ReasoningContent == "" {
		t.Error("Expected reasoning_content but got empty")
	} else if len(response.ReasoningContent) < 50 {
		t.Errorf("Reasoning content too short (%d chars), expected detailed thinking", len(response.ReasoningContent))
	} else {
		t.Logf("✓ Reasoning content length: %d characters", len(response.ReasoningContent))
	}

	// Validate final answer
	contentStr := ""
	if response.Content != nil {
		if str, ok := response.Content.(string); ok {
			contentStr = str
		}
	}

	if len(contentStr) == 0 {
		t.Error("Content is empty")
	} else {
		// Should mention "Yes" or affirm that 5 > 3
		if !strings.Contains(strings.ToLower(contentStr), "yes") && !strings.Contains(strings.ToLower(contentStr), "greater") {
			t.Logf("Warning: Content might not contain expected answer. Content: %s", contentStr)
		} else {
			t.Logf("✓ Final answer: %s", contentStr)
		}
	}

	// Check reasoning tokens
	if response.Usage != nil && response.Usage.CompletionTokensDetails != nil {
		reasoningTokens := response.Usage.CompletionTokensDetails.ReasoningTokens
		if reasoningTokens == 0 {
			t.Error("Expected reasoning_tokens > 0")
		} else {
			t.Logf("✓ Reasoning tokens: %d", reasoningTokens)
		}
	}

	t.Log("Logic puzzle test completed successfully")
}

// ============================================================================
// Helper Functions
// ============================================================================

// newDeepSeekTestContext creates a real Context for testing DeepSeek provider
func newDeepSeekTestContext(chatID, connectorID string) *context.Context {
	return &context.Context{
		Context:     gocontext.Background(),
		Space:       plan.NewMemorySharedSpace(),
		ChatID:      chatID,
		AssistantID: "test-assistant",
		Connector:   connectorID,
		Locale:      "en-us",
		Theme:       "light",
		Client: context.Client{
			Type:      "web",
			UserAgent: "DeepSeekProviderTest/1.0",
			IP:        "127.0.0.1",
		},
		Referer:  context.RefererAPI,
		Accept:   context.AcceptStandard,
		Route:    "/api/test",
		Metadata: make(map[string]interface{}),
		Authorized: &types.AuthorizedInfo{
			Subject:   "test-user",
			ClientID:  "test-client",
			UserID:    "test-user-123",
			TeamID:    "test-team-456",
			TenantID:  "test-tenant-789",
			SessionID: "test-session-id",
			Constraints: types.DataConstraints{
				TeamOnly: true,
				Extra: map[string]interface{}{
					"test": "deepseek-provider",
				},
			},
		},
	}
}
