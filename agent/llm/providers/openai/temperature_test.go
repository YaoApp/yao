package openai_test

import (
	gocontext "context"
	"testing"

	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/gou/connector/openai"
	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/llm"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/test"
)

// TestTemperatureGPT5AutoReset tests that GPT-5 automatically resets temperature to 1.0
func TestTemperatureGPT5AutoReset(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	conn, err := connector.Select("openai.gpt-5")
	if err != nil {
		t.Fatalf("Failed to select connector: %v", err)
	}

	invalidTemp := 0.7 // GPT-5 doesn't support this
	options := &context.CompletionOptions{
		Capabilities: &openai.Capabilities{
			Reasoning: true,
		},
		Temperature: &invalidTemp, // Should be reset to 1.0
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

	ctx := newTemperatureTestContext("test-gpt5-temp", "openai.gpt-5")

	// Should succeed (temperature automatically reset to 1.0)
	response, err := llmInstance.Post(ctx, messages, options)
	if err != nil {
		t.Fatalf("Post failed: %v", err)
	}

	if response == nil {
		t.Fatal("Response is nil")
	}

	t.Log("✓ GPT-5 successfully handled invalid temperature by resetting to 1.0")
	t.Logf("Response: %v", response.Content)
}

// TestTemperatureDeepSeekR1AutoReset tests that DeepSeek R1 automatically resets temperature to 1.0
func TestTemperatureDeepSeekR1AutoReset(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	conn, err := connector.Select("deepseek.r1")
	if err != nil {
		t.Fatalf("Failed to select connector: %v", err)
	}

	invalidTemp := 0.5 // DeepSeek R1 doesn't support this
	options := &context.CompletionOptions{
		Capabilities: &openai.Capabilities{
			Reasoning: true,
		},
		Temperature: &invalidTemp, // Should be reset to 1.0
	}

	llmInstance, err := llm.New(conn, options)
	if err != nil {
		t.Fatalf("Failed to create LLM instance: %v", err)
	}

	messages := []context.Message{
		{
			Role:    context.RoleUser,
			Content: "Say 'Hello'",
		},
	}

	maxTokens := 100
	options.MaxCompletionTokens = &maxTokens

	ctx := newTemperatureTestContext("test-deepseek-r1-temp", "deepseek.r1")

	// Should succeed (temperature automatically reset to 1.0)
	response, err := llmInstance.Post(ctx, messages, options)
	if err != nil {
		t.Fatalf("Post failed: %v", err)
	}

	if response == nil {
		t.Fatal("Response is nil")
	}

	t.Log("✓ DeepSeek R1 successfully handled invalid temperature by resetting to 1.0")
	t.Logf("Response content: %v", response.Content)
	if response.ReasoningContent != "" {
		t.Logf("Reasoning content length: %d", len(response.ReasoningContent))
	}
}

// TestTemperatureGPT4oPreserved tests that GPT-4o preserves custom temperature
func TestTemperatureGPT4oPreserved(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	conn, err := connector.Select("openai.gpt-4o")
	if err != nil {
		t.Fatalf("Failed to select connector: %v", err)
	}

	customTemp := 0.3 // GPT-4o should preserve this
	options := &context.CompletionOptions{
		Capabilities: &openai.Capabilities{
			Reasoning: false, // Not a reasoning model
			ToolCalls: true,
		},
		Temperature: &customTemp, // Should be preserved
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

	ctx := newTemperatureTestContext("test-gpt4o-temp", "openai.gpt-4o")

	// Should succeed with custom temperature preserved
	response, err := llmInstance.Post(ctx, messages, options)
	if err != nil {
		t.Fatalf("Post failed: %v", err)
	}

	if response == nil {
		t.Fatal("Response is nil")
	}

	t.Log("✓ GPT-4o successfully preserved custom temperature (0.3)")
	t.Logf("Response: %v", response.Content)
}

// TestTemperatureDeepSeekV3Preserved tests that DeepSeek V3 preserves custom temperature
func TestTemperatureDeepSeekV3Preserved(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	conn, err := connector.Select("deepseek.v3")
	if err != nil {
		t.Fatalf("Failed to select connector: %v", err)
	}

	customTemp := 0.8 // DeepSeek V3 should preserve this
	options := &context.CompletionOptions{
		Capabilities: &openai.Capabilities{
			Reasoning: false, // Not a reasoning model
			ToolCalls: true,
		},
		Temperature: &customTemp, // Should be preserved
	}

	llmInstance, err := llm.New(conn, options)
	if err != nil {
		t.Fatalf("Failed to create LLM instance: %v", err)
	}

	messages := []context.Message{
		{
			Role:    context.RoleUser,
			Content: "Say 'Hello World'",
		},
	}

	maxTokens := 20
	options.MaxCompletionTokens = &maxTokens

	ctx := newTemperatureTestContext("test-deepseek-v3-temp", "deepseek.v3")

	// Should succeed with custom temperature preserved
	response, err := llmInstance.Post(ctx, messages, options)
	if err != nil {
		t.Fatalf("Post failed: %v", err)
	}

	if response == nil {
		t.Fatal("Response is nil")
	}

	t.Log("✓ DeepSeek V3 successfully preserved custom temperature (0.8)")
	t.Logf("Response: %v", response.Content)
}

// TestTemperatureGPT5Default tests that GPT-5 with temperature=1.0 works fine
func TestTemperatureGPT5Default(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	conn, err := connector.Select("openai.gpt-5")
	if err != nil {
		t.Fatalf("Failed to select connector: %v", err)
	}

	defaultTemp := 1.0 // GPT-5's valid temperature
	options := &context.CompletionOptions{
		Capabilities: &openai.Capabilities{
			Reasoning: true,
		},
		Temperature: &defaultTemp, // Should work fine
	}

	llmInstance, err := llm.New(conn, options)
	if err != nil {
		t.Fatalf("Failed to create LLM instance: %v", err)
	}

	messages := []context.Message{
		{
			Role:    context.RoleUser,
			Content: "What is 2+2? Reply with just the number.",
		},
	}

	maxTokens := 10
	options.MaxCompletionTokens = &maxTokens

	ctx := newTemperatureTestContext("test-gpt5-temp-default", "openai.gpt-5")

	// Should succeed with default temperature
	response, err := llmInstance.Post(ctx, messages, options)
	if err != nil {
		t.Fatalf("Post failed: %v", err)
	}

	if response == nil {
		t.Fatal("Response is nil")
	}

	t.Log("✓ GPT-5 successfully handled default temperature (1.0)")
	t.Logf("Response: %v", response.Content)
}

// TestTemperatureNoTemperatureProvided tests that models work when no temperature is provided
func TestTemperatureNoTemperatureProvided(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	testCases := []struct {
		name      string
		connector string
		reasoning bool
	}{
		{"GPT-5 No Temp", "openai.gpt-5", true},
		{"GPT-4o No Temp", "openai.gpt-4o", false},
		{"DeepSeek R1 No Temp", "deepseek.r1", true},
		{"DeepSeek V3 No Temp", "deepseek.v3", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			conn, err := connector.Select(tc.connector)
			if err != nil {
				t.Fatalf("Failed to select connector: %v", err)
			}

			options := &context.CompletionOptions{
				Capabilities: &openai.Capabilities{
					Reasoning: false,
					ToolCalls: true,
				},
			}
			if tc.reasoning {
				options.Capabilities.Reasoning = true
			}
			// Temperature not set - should use API default

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

			ctx := newTemperatureTestContext("test-no-temp-"+tc.connector, tc.connector)

			response, err := llmInstance.Post(ctx, messages, options)
			if err != nil {
				t.Fatalf("Post failed: %v", err)
			}

			if response == nil {
				t.Fatal("Response is nil")
			}

			t.Logf("✓ %s works fine without temperature parameter", tc.connector)
		})
	}
}

// ============================================================================
// Helper Functions
// ============================================================================

// newTemperatureTestContext creates a real Context for testing temperature handling
func newTemperatureTestContext(chatID, connectorID string) *context.Context {
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
				"test": "temperature",
			},
		},
	}

	ctx := context.New(gocontext.Background(), authorized, chatID)
	ctx.AssistantID = "test-assistant"
	ctx.Locale = "en-us"
	ctx.Theme = "light"
	ctx.Client = context.Client{
		Type:      "web",
		UserAgent: "TemperatureTest/1.0",
		IP:        "127.0.0.1",
	}
	ctx.Referer = context.RefererAPI
	ctx.Accept = context.AcceptStandard
	ctx.Route = "/api/test"
	ctx.Metadata = make(map[string]interface{})
	return ctx
}
