package openai_test

import (
	gocontext "context"
	"strings"
	"testing"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/gou/connector/openai"
	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/llm"
	"github.com/yaoapp/yao/agent/output/message"
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
	options := &context.CompletionOptions{
		Capabilities: &openai.Capabilities{
			Streaming:  true,
			Reasoning:  true,  // DeepSeek R1 supports reasoning
			ToolCalls:  false, // R1 doesn't support native tool calls
			Vision:     false,
			Audio:      false,
			Multimodal: false,
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

	// Track streaming chunks and group events
	var reasoningChunks []string
	var contentChunks []string
	var thinkingGroupEnded bool
	var textGroupEnded bool

	handler := func(chunkType message.StreamChunkType, data []byte) int {
		dataStr := string(data)
		t.Logf("Stream chunk [%s]: %s", chunkType, dataStr)

		// Track different chunk types
		switch chunkType {
		case message.ChunkThinking:
			reasoningChunks = append(reasoningChunks, dataStr)
		case message.ChunkText:
			contentChunks = append(contentChunks, dataStr)
		}

		// Track group_end events to verify type field
		if chunkType == message.ChunkMessageEnd {
			// Parse the group_end data to check the type field
			var groupEndData struct {
				GroupID    string `json:"group_id"`
				Type       string `json:"type"`
				Timestamp  int64  `json:"timestamp"`
				DurationMs int64  `json:"duration_ms"`
				ChunkCount int    `json:"chunk_count"`
				Status     string `json:"status"`
			}

			if err := jsoniter.Unmarshal(data, &groupEndData); err == nil {
				t.Logf("✓ group_end received: type=%s, chunks=%d, duration=%dms",
					groupEndData.Type, groupEndData.ChunkCount, groupEndData.DurationMs)

				// Verify the type field matches expected group types
				switch groupEndData.Type {
				case "thinking":
					thinkingGroupEnded = true
					if groupEndData.ChunkCount == 0 {
						t.Error("thinking group_end should have chunk_count > 0")
					}
				case "text":
					textGroupEnded = true
					if groupEndData.ChunkCount == 0 {
						t.Error("text group_end should have chunk_count > 0")
					}
				}
			} else {
				t.Errorf("Failed to parse group_end data: %v", err)
			}
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

	// Verify group_end events were received with correct types
	if !thinkingGroupEnded {
		t.Error("❌ Expected thinking group_end event but didn't receive it")
	} else {
		t.Log("✅ Thinking group_end event received with type='thinking'")
	}

	if !textGroupEnded {
		t.Error("❌ Expected text group_end event but didn't receive it")
	} else {
		t.Log("✅ Text group_end event received with type='text'")
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
	options := &context.CompletionOptions{
		Capabilities: &openai.Capabilities{
			Reasoning:  true,
			ToolCalls:  false,
			Vision:     false,
			Audio:      false,
			Multimodal: false,
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

	options := &context.CompletionOptions{
		Capabilities: &openai.Capabilities{
			Streaming:  true,
			Reasoning:  true,
			ToolCalls:  false,
			Vision:     false,
			Audio:      false,
			Multimodal: false,
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
	handler := func(chunkType message.StreamChunkType, data []byte) int {
		if chunkType == message.ChunkThinking && len(data) > 0 {
			hasReasoning = true
		} else if chunkType == message.ChunkText && len(data) > 0 {
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
				"test": "deepseek-provider",
			},
		},
	}

	ctx := context.New(gocontext.Background(), authorized, chatID)
	ctx.AssistantID = "test-assistant"
	ctx.Locale = "en-us"
	ctx.Theme = "light"
	ctx.Client = context.Client{
		Type:      "web",
		UserAgent: "DeepSeekProviderTest/1.0",
		IP:        "127.0.0.1",
	}
	ctx.Referer = context.RefererAPI
	ctx.Accept = context.AcceptStandard
	ctx.Route = "/api/test"
	ctx.Metadata = make(map[string]interface{})
	return ctx
}
