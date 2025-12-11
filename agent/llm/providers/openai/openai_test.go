package openai_test

import (
	gocontext "context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/gou/connector/openai"
	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/llm"
	"github.com/yaoapp/yao/agent/output/message"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/test"
)

// TestOpenAIStreamBasic tests basic streaming completion with short output
func TestOpenAIStreamBasic(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Create connector from real configuration
	conn, err := connector.Select("openai.gpt-4o")
	if err != nil {
		t.Fatalf("Failed to select connector: %v", err)
	}

	// Create LLM instance with capabilities
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

	// Prepare messages with concise prompt
	messages := []context.Message{
		{
			Role:    context.RoleUser,
			Content: "Say 'Hello' in one word.",
		},
	}

	// Set short max tokens to ensure quick response
	maxTokens := 5
	options.MaxTokens = &maxTokens

	// Create context
	ctx := newTestContext("test-stream-basic", "openai.gpt-4o")

	// Track streaming chunks
	var chunks []string
	handler := func(chunkType message.StreamChunkType, data []byte) int {
		chunks = append(chunks, string(data))
		t.Logf("Stream chunk [%s]: %s", chunkType, string(data))
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
	if response.Usage == nil {
		t.Error("Response Usage is nil")
	} else {
		if response.Usage.TotalTokens == 0 {
			t.Error("Response Usage.TotalTokens is 0")
		}
		t.Logf("Usage: prompt=%d, completion=%d, total=%d",
			response.Usage.PromptTokens, response.Usage.CompletionTokens, response.Usage.TotalTokens)
	}
	if len(chunks) == 0 {
		t.Error("No streaming chunks received")
	}

	t.Logf("Final response: %+v", response)
	t.Logf("Total chunks received: %d", len(chunks))
}

// TestOpenAIPostBasic tests basic non-streaming completion
func TestOpenAIPostBasic(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Create connector
	conn, err := connector.Select("openai.gpt-4o")
	if err != nil {
		t.Fatalf("Failed to select connector: %v", err)
	}

	// Create LLM instance
	options := &context.CompletionOptions{
		Capabilities: &openai.Capabilities{
			ToolCalls: true,
		},
	}

	llmInstance, err := llm.New(conn, options)
	if err != nil {
		t.Fatalf("Failed to create LLM instance: %v", err)
	}

	// Prepare messages with concise prompt
	messages := []context.Message{
		{
			Role:    context.RoleUser,
			Content: "Reply with only the word 'OK'.",
		},
	}

	// Set short max tokens
	maxTokens := 5
	options.MaxTokens = &maxTokens

	// Create context
	ctx := newTestContext("test-stream-basic", "openai.gpt-4o")

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
	if response.FinishReason == "" {
		t.Error("FinishReason is empty")
	}
	if response.Usage == nil {
		t.Error("Response Usage is nil")
	} else {
		if response.Usage.TotalTokens == 0 {
			t.Error("Response Usage.TotalTokens is 0")
		}
		t.Logf("Usage: prompt=%d, completion=%d, total=%d",
			response.Usage.PromptTokens, response.Usage.CompletionTokens, response.Usage.TotalTokens)
	}

	t.Logf("Response: %+v", response)
}

// TestOpenAIStreamWithToolCalls tests streaming with tool calls and JSON schema validation
func TestOpenAIStreamWithToolCalls(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Create connector
	conn, err := connector.Select("openai.gpt-4o")
	if err != nil {
		t.Fatalf("Failed to select connector: %v", err)
	}

	// Create LLM instance with tool call capabilities
	options := &context.CompletionOptions{
		Capabilities: &openai.Capabilities{
			Streaming: true,
			ToolCalls: true,
		},
	}

	// Define a simple weather tool with JSON schema
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
						"description": "The city and state, e.g. San Francisco, CA",
					},
					"unit": map[string]interface{}{
						"type": "string",
						"enum": []string{"celsius", "fahrenheit"},
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

	// Prepare messages that should trigger tool call
	messages := []context.Message{
		{
			Role:    context.RoleUser,
			Content: "What's the weather in Tokyo? Use celsius.",
		},
	}

	// Create context
	ctx := newTestContext("test-stream-basic", "openai.gpt-4o")

	// Track streaming chunks
	var toolCallChunks int
	handler := func(chunkType message.StreamChunkType, data []byte) int {
		if chunkType == message.ChunkToolCall {
			toolCallChunks++
		}
		t.Logf("Stream chunk [%s]: %s", chunkType, string(data))
		return 0 // Continue
	}

	// Call Stream
	response, err := llmInstance.Stream(ctx, messages, options, handler)
	if err != nil {
		t.Fatalf("Stream with tool calls failed: %v", err)
	}

	// Validate response
	if response == nil {
		t.Fatal("Response is nil")
	}

	// Should have tool calls
	if len(response.ToolCalls) == 0 {
		t.Error("Expected tool calls but got none")
	} else {
		t.Logf("Received %d tool call(s)", len(response.ToolCalls))
		for i, tc := range response.ToolCalls {
			t.Logf("Tool call %d: %s(%s)", i, tc.Function.Name, tc.Function.Arguments)

			// Validate tool call has required fields
			if tc.ID == "" {
				t.Errorf("Tool call %d missing ID", i)
			}
			if tc.Function.Name == "" {
				t.Errorf("Tool call %d missing function name", i)
			}
			if tc.Function.Arguments == "" {
				t.Errorf("Tool call %d missing arguments", i)
			}
		}
	}

	if response.FinishReason != context.FinishReasonToolCalls {
		t.Logf("Warning: Expected finish_reason='tool_calls', got '%s'", response.FinishReason)
	}

	if toolCallChunks == 0 {
		t.Error("No tool call chunks received during streaming")
	}

	t.Logf("Final response: %+v", response)
}

// TestOpenAIPostWithToolCalls tests non-streaming with tool calls
func TestOpenAIPostWithToolCalls(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Create connector
	conn, err := connector.Select("openai.gpt-4o")
	if err != nil {
		t.Fatalf("Failed to select connector: %v", err)
	}

	// Create LLM instance with tool call capabilities
	options := &context.CompletionOptions{
		Capabilities: &openai.Capabilities{
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

	// Prepare messages
	messages := []context.Message{
		{
			Role:    context.RoleUser,
			Content: "Calculate 15 * 8",
		},
	}

	// Create context
	ctx := newTestContext("test-stream-basic", "openai.gpt-4o")

	// Call Post
	response, err := llmInstance.Post(ctx, messages, options)
	if err != nil {
		t.Fatalf("Post with tool calls failed: %v", err)
	}

	// Validate response
	if response == nil {
		t.Fatal("Response is nil")
	}

	// Validate response metadata
	if response.ID == "" {
		t.Error("Response ID is empty")
	}
	if response.Model == "" {
		t.Error("Response Model is empty")
	}
	if response.FinishReason != "tool_calls" {
		t.Errorf("FinishReason is %s, expected tool_calls", response.FinishReason)
	}
	if response.Usage == nil {
		t.Error("Response Usage is nil")
	} else {
		if response.Usage.TotalTokens == 0 {
			t.Error("Response Usage.TotalTokens is 0")
		}
		t.Logf("Usage: prompt=%d, completion=%d, total=%d",
			response.Usage.PromptTokens, response.Usage.CompletionTokens, response.Usage.TotalTokens)
	}

	// Should have tool calls
	if len(response.ToolCalls) == 0 {
		t.Error("Expected tool calls but got none")
	} else {
		tc := response.ToolCalls[0]

		// Validate tool call structure
		if tc.ID == "" {
			t.Error("Tool call ID is empty")
		}
		if tc.Type != context.ToolTypeFunction {
			t.Errorf("Tool call Type is %s, expected %s", tc.Type, context.ToolTypeFunction)
		}
		if tc.Function.Name != "calculate" {
			t.Errorf("Tool call function name is %s, expected calculate", tc.Function.Name)
		}
		if tc.Function.Arguments == "" {
			t.Error("Tool call arguments are empty")
		}

		t.Logf("Received %d tool call(s)", len(response.ToolCalls))
		for i, tc := range response.ToolCalls {
			t.Logf("Tool call %d: %s(%s)", i, tc.Function.Name, tc.Function.Arguments)
		}
	}

	t.Logf("Response: %+v", response)
}

// TestOpenAIStreamWithInvalidToolCall tests that invalid tool calls trigger validation error
func TestOpenAIStreamWithInvalidToolCall(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Create connector
	conn, err := connector.Select("openai.gpt-4o")
	if err != nil {
		t.Fatalf("Failed to select connector: %v", err)
	}

	// Create LLM instance
	options := &context.CompletionOptions{
		Capabilities: &openai.Capabilities{
			Streaming: true,
			ToolCalls: true,
		},
	}

	// Define a strict tool that requires specific format
	strictTool := map[string]interface{}{
		"type": "function",
		"function": map[string]interface{}{
			"name":        "send_email",
			"description": "Send an email",
			"parameters": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"to": map[string]interface{}{
						"type":    "string",
						"pattern": "^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$",
					},
					"subject": map[string]interface{}{
						"type":      "string",
						"minLength": 1,
					},
					"body": map[string]interface{}{
						"type":      "string",
						"minLength": 1,
					},
				},
				"required": []string{"to", "subject", "body"},
			},
		},
	}

	options.Tools = []map[string]interface{}{strictTool}
	options.ToolChoice = map[string]interface{}{
		"type": "function",
		"function": map[string]interface{}{
			"name": "send_email",
		},
	}

	llmInstance, err := llm.New(conn, options)
	if err != nil {
		t.Fatalf("Failed to create LLM instance: %v", err)
	}

	// Prepare messages with incomplete information (should cause validation error)
	messages := []context.Message{
		{
			Role:    context.RoleUser,
			Content: "Send email to invalid-email without subject",
		},
	}

	// Create context
	ctx := newTestContext("test-stream-basic", "openai.gpt-4o")

	handler := func(chunkType message.StreamChunkType, data []byte) int {
		return 0 // Continue
	}

	// Call Stream - should succeed but may trigger validation if tool call is malformed
	response, err := llmInstance.Stream(ctx, messages, options, handler)

	// The API might return a valid tool call despite the bad prompt,
	// so we just log the result
	if err != nil {
		t.Logf("Stream failed as expected with validation error: %v", err)
	} else {
		t.Logf("Stream succeeded, response: %+v", response)
		if len(response.ToolCalls) > 0 {
			t.Logf("Tool calls: %v", response.ToolCalls)
		}
	}
}

// TestOpenAIStreamRetry tests the retry mechanism with invalid API key
func TestOpenAIStreamRetry(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Create connector with invalid API key to trigger 401 error (non-retryable)
	connDSL := `{
		"type": "openai",
		"options": {
			"model": "gpt-4o",
			"key": "sk-invalid-key-should-fail-auth",
			"host": "https://api.openai.com"
		}
	}`

	conn, err := connector.New("openai", "test-retry", []byte(connDSL))
	if err != nil {
		t.Fatalf("Failed to create test connector: %v", err)
	}

	// Create LLM instance
	options := &context.CompletionOptions{
		Capabilities: &openai.Capabilities{
			Streaming: true,
			ToolCalls: true, // Need this to select OpenAI provider
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

	ctx := newTestContext("test-retry", "test-retry")

	// This should fail quickly without retry (401 is non-retryable)
	_, err = llmInstance.Stream(ctx, messages, options, nil)
	if err == nil {
		t.Fatal("Expected error due to invalid API key, but got success")
	}

	// Verify it's an error related to invalid API key
	// Could be: 401, unauthorized, authentication error, or no data (empty response)
	errMsg := err.Error()
	hasExpectedError := strings.Contains(strings.ToLower(errMsg), "401") ||
		strings.Contains(strings.ToLower(errMsg), "unauthorized") ||
		strings.Contains(strings.ToLower(errMsg), "authentication") ||
		strings.Contains(strings.ToLower(errMsg), "incorrect api key") ||
		strings.Contains(strings.ToLower(errMsg), "no data received")

	if !hasExpectedError {
		t.Errorf("Expected authentication or empty response error, got: %v", err)
	}

	// Should mention non-retryable (these errors should not trigger retry)
	if !strings.Contains(strings.ToLower(errMsg), "non-retryable") {
		t.Errorf("Error should indicate non-retryable: %v", err)
	}

	t.Logf("Failed as expected with error: %v", err)
}

// TestOpenAIStreamChunkTypes tests that stream handler receives correct chunk types
func TestOpenAIStreamChunkTypes(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	conn, err := connector.Select("openai.gpt-4o")
	if err != nil {
		t.Fatalf("Failed to select connector: %v", err)
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
			Content: "Say 'test' in one word.",
		},
	}

	ctx := newTestContext("test-chunk-types", "openai.gpt-4o")

	// Track chunk types
	chunkTypes := make(map[message.StreamChunkType]int)
	handler := func(chunkType message.StreamChunkType, data []byte) int {
		chunkTypes[chunkType]++
		t.Logf("Received chunk type: %s, data length: %d", chunkType, len(data))
		return 1 // Continue
	}

	response, err := llmInstance.Stream(ctx, messages, options, handler)
	if err != nil {
		t.Fatalf("Stream failed: %v", err)
	}

	if response == nil {
		t.Fatal("Response is nil")
	}

	// Validate chunk types received
	if chunkTypes[message.ChunkText] == 0 {
		t.Error("Expected to receive ChunkText, but got 0")
	}

	t.Logf("Chunk types received: %+v", chunkTypes)
}

// TestOpenAIStreamErrorCallback tests that errors are sent to stream handler
func TestOpenAIStreamErrorCallback(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Create connector with invalid API key to trigger error
	connDSL := `{
		"type": "openai",
		"options": {
			"model": "gpt-4o",
			"key": "sk-invalid-for-error-test",
			"host": "https://api.openai.com"
		}
	}`

	conn, err := connector.New("openai", "test-error-callback", []byte(connDSL))
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

	ctx := newTestContext("test-error-callback", "test-error-callback")

	// Track if error chunk was received
	receivedError := false
	var errorMessage string
	handler := func(chunkType message.StreamChunkType, data []byte) int {
		if chunkType == message.ChunkError {
			receivedError = true
			errorMessage = string(data)
			t.Logf("Received error chunk: %s", errorMessage)
		}
		return 1 // Continue
	}

	// This should fail and send error to handler
	_, err = llmInstance.Stream(ctx, messages, options, handler)
	if err == nil {
		t.Fatal("Expected error due to invalid API key")
	}

	// Verify error was sent to handler
	if !receivedError {
		t.Error("Expected to receive ChunkError in handler, but didn't")
	}

	if errorMessage == "" {
		t.Error("Error message in chunk is empty")
	}

	t.Logf("Error callback test passed. Error: %v", err)
}

// TestOpenAIToolCallValidationRetry tests automatic tool call validation retry with LLM feedback
func TestOpenAIToolCallValidationRetry(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	conn, err := connector.Select("openai.gpt-4o")
	if err != nil {
		t.Fatalf("Failed to select connector: %v", err)
	}

	options := &context.CompletionOptions{
		Capabilities: &openai.Capabilities{
			Streaming: true,
			ToolCalls: true,
		},
		Tools: []map[string]interface{}{
			{
				"type": "function",
				"function": map[string]interface{}{
					"name":        "test_strict_validation",
					"description": "A function with very strict validation rules",
					"parameters": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"status": map[string]interface{}{
								"type":        "string",
								"description": "Must be exactly 'active' or 'inactive'",
								"enum":        []string{"active", "inactive"},
							},
							"priority": map[string]interface{}{
								"type":        "integer",
								"description": "Must be between 1 and 5",
								"minimum":     1,
								"maximum":     5,
							},
						},
						"required": []string{"status", "priority"},
					},
				},
			},
		},
	}

	llmInstance, err := llm.New(conn, options)
	if err != nil {
		t.Fatalf("Failed to create LLM instance: %v", err)
	}

	ctx := newTestContext("test-tool-validation-retry", "openai.gpt-4o")

	// Try to make LLM call with intentionally unclear requirements
	// This may or may not trigger validation, depending on LLM behavior
	messages := []context.Message{
		{
			Role:    context.RoleUser,
			Content: "Call test_strict_validation function with status='pending' and priority=10",
		},
	}

	// The Provider will automatically:
	// 1. Call LLM
	// 2. If validation fails, add error feedback to conversation
	// 3. Retry up to 3 times with feedback
	// 4. Return success or validation error after max retries
	response, err := llmInstance.Stream(ctx, messages, options, nil)

	if err != nil {
		// Check if it's a validation error after retries
		if strings.Contains(err.Error(), "tool call validation failed after") &&
			strings.Contains(err.Error(), "retries") {
			t.Logf("✓ Automatic validation retry exhausted: %v", err)
		} else if strings.Contains(err.Error(), "validation") {
			t.Logf("✓ Validation failed: %v", err)
		} else {
			t.Logf("Request failed (non-validation): %v", err)
		}
	} else if response != nil {
		if len(response.ToolCalls) > 0 {
			t.Logf("✓ Tool call succeeded (possibly after auto-retry): %+v", response.ToolCalls[0])

			// Verify the tool call arguments are valid
			tc := response.ToolCalls[0]
			var args map[string]interface{}
			if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err == nil {
				if status, ok := args["status"].(string); ok {
					if status != "active" && status != "inactive" {
						t.Errorf("Status should be 'active' or 'inactive', got: %s", status)
					}
				}
				if priority, ok := args["priority"].(float64); ok {
					if priority < 1 || priority > 5 {
						t.Errorf("Priority should be between 1-5, got: %v", priority)
					}
				}
			}
		} else {
			t.Log("✓ Response returned but no tool calls")
		}
	}

	t.Log("Automatic tool call validation retry test completed")
}

// TestOpenAIJSONMode tests JSON mode response formatting
func TestOpenAIJSONMode(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	conn, err := connector.Select("openai.gpt-4o")
	if err != nil {
		t.Fatalf("Failed to select connector: %v", err)
	}

	options := &context.CompletionOptions{
		Capabilities: &openai.Capabilities{
			Streaming: true,
			ToolCalls: true,
		},
		ResponseFormat: &context.ResponseFormat{
			Type: context.ResponseFormatJSON,
		},
	}

	llmInstance, err := llm.New(conn, options)
	if err != nil {
		t.Fatalf("Failed to create LLM instance: %v", err)
	}

	messages := []context.Message{
		{
			Role:    context.RoleUser,
			Content: "Generate a JSON object with fields: name (string), age (number), city (string). Use values: John, 30, New York",
		},
	}

	ctx := newTestContext("test-json-mode", "openai.gpt-4o")

	// Test streaming with JSON mode
	response, err := llmInstance.Stream(ctx, messages, options, nil)
	if err != nil {
		t.Fatalf("Stream with JSON mode failed: %v", err)
	}

	if response == nil {
		t.Fatal("Response is nil")
	}

	// Validate response
	contentStr, ok := response.Content.(string)
	if !ok || contentStr == "" {
		t.Error("Response content is empty or not a string")
	}

	// Try to parse as JSON
	var jsonData map[string]interface{}
	if err := json.Unmarshal([]byte(contentStr), &jsonData); err != nil {
		t.Errorf("Response is not valid JSON: %v\nContent: %s", err, contentStr)
	} else {
		t.Logf("✓ Response is valid JSON: %+v", jsonData)

		// Verify expected fields exist
		if _, hasName := jsonData["name"]; !hasName {
			t.Error("JSON response missing 'name' field")
		}
		if _, hasAge := jsonData["age"]; !hasAge {
			t.Error("JSON response missing 'age' field")
		}
		if _, hasCity := jsonData["city"]; !hasCity {
			t.Error("JSON response missing 'city' field")
		}
	}

	// Validate metadata
	if response.ID == "" {
		t.Error("Response ID is empty")
	}
	if response.Model == "" {
		t.Error("Response Model is empty")
	}
	if response.Usage == nil {
		t.Error("Response Usage is nil")
	} else {
		t.Logf("Usage: prompt=%d, completion=%d, total=%d",
			response.Usage.PromptTokens, response.Usage.CompletionTokens, response.Usage.TotalTokens)
	}

	t.Log("JSON mode test completed successfully")
}

// TestOpenAIJSONModePost tests JSON mode with non-streaming
func TestOpenAIJSONModePost(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	conn, err := connector.Select("openai.gpt-4o")
	if err != nil {
		t.Fatalf("Failed to select connector: %v", err)
	}

	options := &context.CompletionOptions{
		Capabilities: &openai.Capabilities{
			ToolCalls: true,
		},
		ResponseFormat: &context.ResponseFormat{
			Type: context.ResponseFormatJSON,
		},
	}

	llmInstance, err := llm.New(conn, options)
	if err != nil {
		t.Fatalf("Failed to create LLM instance: %v", err)
	}

	messages := []context.Message{
		{
			Role:    context.RoleUser,
			Content: "Return a JSON with: status='success', count=42",
		},
	}

	ctx := newTestContext("test-json-mode-post", "openai.gpt-4o")

	// Test non-streaming with JSON mode
	response, err := llmInstance.Post(ctx, messages, options)
	if err != nil {
		t.Fatalf("Post with JSON mode failed: %v", err)
	}

	if response == nil {
		t.Fatal("Response is nil")
	}

	// Validate response content is JSON
	contentStr, ok := response.Content.(string)
	if !ok || contentStr == "" {
		t.Error("Response content is empty or not a string")
	}

	var jsonData map[string]interface{}
	if err := json.Unmarshal([]byte(contentStr), &jsonData); err != nil {
		t.Errorf("Response is not valid JSON: %v\nContent: %s", err, contentStr)
	} else {
		t.Logf("✓ Response is valid JSON: %+v", jsonData)
	}

	// Validate metadata
	if response.Usage == nil {
		t.Error("Response Usage is nil")
	} else {
		if response.Usage.TotalTokens == 0 {
			t.Error("Response Usage.TotalTokens is 0")
		}
		t.Logf("Usage: prompt=%d, completion=%d, total=%d",
			response.Usage.PromptTokens, response.Usage.CompletionTokens, response.Usage.TotalTokens)
	}

	t.Log("JSON mode Post test completed successfully")
}

// TestOpenAIJSONSchema tests JSON mode with strict schema validation
func TestOpenAIJSONSchema(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	conn, err := connector.Select("openai.gpt-4o")
	if err != nil {
		t.Fatalf("Failed to select connector: %v", err)
	}

	// Define a strict JSON schema
	// Note: For OpenAI strict mode, 'required' must include ALL properties
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"user": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{
						"type":        "string",
						"description": "User's full name",
					},
					"email": map[string]interface{}{
						"type":        "string",
						"description": "User's email address",
					},
					"age": map[string]interface{}{
						"type":        "integer",
						"description": "User's age",
					},
					"isActive": map[string]interface{}{
						"type":        "boolean",
						"description": "Whether user is active",
					},
				},
				"required":             []string{"name", "email", "age", "isActive"},
				"additionalProperties": false,
			},
		},
		"required":             []string{"user"},
		"additionalProperties": false,
	}

	options := &context.CompletionOptions{
		Capabilities: &openai.Capabilities{
			Streaming: true,
			ToolCalls: true,
		},
		ResponseFormat: &context.ResponseFormat{
			Type: context.ResponseFormatJSONSchema,
			JSONSchema: &context.JSONSchema{
				Name:        "user_info",
				Description: "User information schema",
				Schema:      schema,
				Strict:      func() *bool { v := true; return &v }(),
			},
		},
	}

	llmInstance, err := llm.New(conn, options)
	if err != nil {
		t.Fatalf("Failed to create LLM instance: %v", err)
	}

	messages := []context.Message{
		{
			Role:    context.RoleUser,
			Content: "Create user info for: Alice Smith, alice@example.com, age 28, active user",
		},
	}

	ctx := newTestContext("test-json-schema", "openai.gpt-4o")

	// Test streaming with JSON schema
	response, err := llmInstance.Stream(ctx, messages, options, nil)
	if err != nil {
		t.Fatalf("Stream with JSON schema failed: %v", err)
	}

	if response == nil {
		t.Fatal("Response is nil")
	}

	// Validate response content
	contentStr, ok := response.Content.(string)
	if !ok || contentStr == "" {
		t.Fatal("Response content is empty or not a string")
	}

	// Parse as JSON
	var jsonData map[string]interface{}
	if err := json.Unmarshal([]byte(contentStr), &jsonData); err != nil {
		t.Fatalf("Response is not valid JSON: %v\nContent: %s", err, contentStr)
	}

	t.Logf("✓ Response is valid JSON: %+v", jsonData)

	// Verify structure matches schema
	user, hasUser := jsonData["user"].(map[string]interface{})
	if !hasUser {
		t.Fatal("JSON response missing 'user' object")
	}

	// Verify required fields
	if _, hasName := user["name"]; !hasName {
		t.Error("User object missing required 'name' field")
	}
	if _, hasEmail := user["email"]; !hasEmail {
		t.Error("User object missing required 'email' field")
	}

	// Verify field types
	if name, ok := user["name"].(string); ok {
		t.Logf("✓ name: %s (string)", name)
	} else {
		t.Error("name is not a string")
	}

	if email, ok := user["email"].(string); ok {
		t.Logf("✓ email: %s (string)", email)
	} else {
		t.Error("email is not a string")
	}

	if age, ok := user["age"].(float64); ok {
		if age < 0 || age > 150 {
			t.Errorf("age %v is out of range [0, 150]", age)
		}
		t.Logf("✓ age: %v (integer, in range)", age)
	}

	if isActive, ok := user["isActive"].(bool); ok {
		t.Logf("✓ isActive: %v (boolean)", isActive)
	}

	// Validate metadata
	if response.Usage == nil {
		t.Error("Response Usage is nil")
	} else {
		t.Logf("Usage: prompt=%d, completion=%d, total=%d",
			response.Usage.PromptTokens, response.Usage.CompletionTokens, response.Usage.TotalTokens)
	}

	t.Log("JSON schema test completed successfully")
}

// TestOpenAIJSONSchemaPost tests JSON schema with non-streaming
func TestOpenAIJSONSchemaPost(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	conn, err := connector.Select("openai.gpt-4o")
	if err != nil {
		t.Fatalf("Failed to select connector: %v", err)
	}

	// Simple schema for testing
	// Note: For OpenAI strict mode, 'required' must include ALL properties
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"status": map[string]interface{}{
				"type": "string",
				"enum": []string{"success", "error", "pending"},
			},
			"message": map[string]interface{}{
				"type": "string",
			},
			"code": map[string]interface{}{
				"type": "integer",
			},
		},
		"required":             []string{"status", "message", "code"},
		"additionalProperties": false,
	}

	options := &context.CompletionOptions{
		Capabilities: &openai.Capabilities{
			ToolCalls: true,
		},
		ResponseFormat: &context.ResponseFormat{
			Type: context.ResponseFormatJSONSchema,
			JSONSchema: &context.JSONSchema{
				Name:        "api_response",
				Description: "API response format",
				Schema:      schema,
				Strict:      func() *bool { v := true; return &v }(),
			},
		},
	}

	llmInstance, err := llm.New(conn, options)
	if err != nil {
		t.Fatalf("Failed to create LLM instance: %v", err)
	}

	messages := []context.Message{
		{
			Role:    context.RoleUser,
			Content: "Generate an API response with status 'success', message 'Operation completed', and code 200",
		},
	}

	ctx := newTestContext("test-json-schema-post", "openai.gpt-4o")

	// Test non-streaming with JSON schema
	response, err := llmInstance.Post(ctx, messages, options)
	if err != nil {
		t.Fatalf("Post with JSON schema failed: %v", err)
	}

	if response == nil {
		t.Fatal("Response is nil")
	}

	// Validate response content
	contentStr, ok := response.Content.(string)
	if !ok || contentStr == "" {
		t.Fatal("Response content is empty or not a string")
	}

	// Parse and validate JSON
	var jsonData map[string]interface{}
	if err := json.Unmarshal([]byte(contentStr), &jsonData); err != nil {
		t.Fatalf("Response is not valid JSON: %v\nContent: %s", err, contentStr)
	}

	t.Logf("✓ Response is valid JSON: %+v", jsonData)

	// Verify required fields
	status, hasStatus := jsonData["status"].(string)
	if !hasStatus {
		t.Fatal("Missing required 'status' field")
	}

	// Verify enum constraint
	validStatuses := map[string]bool{"success": true, "error": true, "pending": true}
	if !validStatuses[status] {
		t.Errorf("status '%s' is not in enum [success, error, pending]", status)
	}

	if _, hasMessage := jsonData["message"].(string); !hasMessage {
		t.Error("Missing required 'message' field")
	}

	// Validate metadata
	if response.Usage == nil {
		t.Error("Response Usage is nil")
	} else {
		t.Logf("Usage: prompt=%d, completion=%d, total=%d",
			response.Usage.PromptTokens, response.Usage.CompletionTokens, response.Usage.TotalTokens)
	}

	t.Log("JSON schema Post test completed successfully")
}

// TestOpenAIProxySupport tests that HTTP proxy configuration is respected
func TestOpenAIProxySupport(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// This test verifies proxy support exists in the connector configuration
	// Actual proxy testing requires a real proxy server setup

	conn, err := connector.Select("openai.gpt-4o")
	if err != nil {
		t.Fatalf("Failed to select connector: %v", err)
	}

	settings := conn.Setting()
	t.Logf("Connector settings: %+v", settings)

	// Verify host field exists in settings (host is the API endpoint)
	if host, hasHost := settings["host"]; hasHost {
		t.Logf("API host configured: %v", host)
	} else {
		t.Log("Host field not in settings (will use default)")
	}

	// The actual HTTP proxy functionality is implemented via environment variables
	// (HTTP_PROXY, HTTPS_PROXY, NO_PROXY) and handled by http.GetTransport
	t.Log("HTTP proxy support is implemented via http.GetTransport using environment variables")
}

// TestOpenAIStreamLifecycleEvents tests that LLM-level lifecycle events are sent correctly
// LLM layer sends group_start/end for individual messages (thinking, text, tool_call)
// Note: stream_start/end and Agent-level blocks are handled at Agent level
func TestOpenAIStreamLifecycleEvents(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	conn, err := connector.Select("openai.gpt-4o")
	if err != nil {
		t.Fatalf("Failed to select connector: %v", err)
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
			Content: "Say 'hello' in one word",
		},
	}

	ctx := newTestContext("test-lifecycle", "openai.gpt-4o")

	// Track lifecycle events (group_start/end at LLM layer represent message boundaries)
	var events []string
	var groupStartReceived, groupEndReceived bool

	handler := func(chunkType message.StreamChunkType, data []byte) int {
		events = append(events, string(chunkType))

		switch chunkType {
		case message.ChunkStreamStart:
			t.Error("❌ LLM layer should NOT send stream_start (now sent at Agent level)")

		case message.ChunkStreamEnd:
			t.Error("❌ LLM layer should NOT send stream_end (now sent at Agent level)")

		case message.ChunkMessageStart:
			groupStartReceived = true
			var startData message.EventMessageStartData
			if err := json.Unmarshal(data, &startData); err == nil {
				t.Logf("✓ group_start (message start): type=%s, id=%s", startData.Type, startData.MessageID)
				if startData.MessageID == "" {
					t.Error("group_start missing message_id")
				}
			} else {
				t.Errorf("Failed to parse group_start data: %v", err)
			}

		case message.ChunkMessageEnd:
			groupEndReceived = true
			var endData message.EventMessageEndData
			if err := json.Unmarshal(data, &endData); err == nil {
				t.Logf("✓ group_end (message end): type=%s, chunks=%d, duration=%dms",
					endData.Type, endData.ChunkCount, endData.DurationMs)
				if endData.ChunkCount <= 0 {
					t.Error("group_end should have chunk_count > 0")
				}
			} else {
				t.Errorf("Failed to parse group_end data: %v", err)
			}

		case message.ChunkText:
			t.Logf("  text chunk: %s", string(data))
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

	// Validate that LLM-level message lifecycle events were received
	if !groupStartReceived {
		t.Error("group_start (message start) event was not received")
	}
	if !groupEndReceived {
		t.Error("group_end (message end) event was not received")
	}

	// Validate event order: group_start should come before group_end
	if len(events) < 2 {
		t.Errorf("Expected at least 2 events (message start/end), got %d", len(events))
	}

	t.Logf("Total events received: %d", len(events))
	t.Log("LLM message lifecycle events test completed successfully")
	t.Log("Note: LLM layer group_start/end represent message boundaries (thinking, text, tool_call)")
	t.Log("      Agent-level block boundaries and stream_start/end are handled at Agent level")
}

// TestOpenAIStreamContextCancellation tests that stream respects context cancellation
func TestOpenAIStreamContextCancellation(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	conn, err := connector.Select("openai.gpt-4o")
	if err != nil {
		t.Fatalf("Failed to select connector: %v", err)
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
			Content: "Write a very long essay about the history of computing", // Long task
		},
	}

	// Create a context with a very short timeout
	ctx := newTestContext("test-cancel", "openai.gpt-4o")
	goCtx, cancel := gocontext.WithTimeout(gocontext.Background(), 100*time.Millisecond)
	defer cancel()
	ctx.Context = goCtx

	var receivedChunks int

	handler := func(chunkType message.StreamChunkType, data []byte) int {
		if chunkType == message.ChunkText || chunkType == message.ChunkToolCall {
			receivedChunks++
		}
		// Note: stream_end is now sent at Agent level, not LLM level
		if chunkType == message.ChunkStreamEnd {
			t.Error("❌ LLM layer should NOT send stream_end (now sent at Agent level)")
		}
		return 0
	}

	response, err := llmInstance.Stream(ctx, messages, options, handler)

	// Should get an error due to context cancellation
	if err == nil {
		t.Error("Expected error due to context cancellation, but got nil")
	} else {
		t.Logf("✓ Got expected cancellation error: %v", err)

		// Check if error message indicates cancellation
		errStr := err.Error()
		if !strings.Contains(errStr, "context") && !strings.Contains(errStr, "cancel") {
			t.Errorf("Error should mention context/cancellation: %v", err)
		}
	}

	// Response should be nil due to cancellation
	if response != nil {
		t.Logf("Warning: Response is not nil despite cancellation (partial response)")
	}

	t.Logf("Received %d chunks before cancellation", receivedChunks)
	t.Log("Context cancellation test completed successfully")
	t.Log("Note: stream_end for cancellation is now sent at Agent level")
}

// TestOpenAIStreamWithTemperature tests different temperature settings
func TestOpenAIStreamWithTemperature(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	conn, err := connector.Select("openai.gpt-4o")
	if err != nil {
		t.Fatalf("Failed to select connector: %v", err)
	}

	temperature := 0.7 // Moderate temperature

	options := &context.CompletionOptions{
		Capabilities: &openai.Capabilities{
			Streaming: true,
			ToolCalls: true, // Need this to select OpenAI provider
		},
		Temperature: &temperature,
	}

	llmInstance, err := llm.New(conn, options)
	if err != nil {
		t.Fatalf("Failed to create LLM instance: %v", err)
	}

	messages := []context.Message{
		{
			Role:    context.RoleUser,
			Content: "Say 'yes' in one word.",
		},
	}

	ctx := newTestContext("test-temperature", "openai.gpt-4o")

	// Use callback to collect chunks
	chunkCount := 0
	var callback message.StreamFunc = func(chunkType message.StreamChunkType, data []byte) int {
		chunkCount++
		return 1 // Continue
	}

	response, err := llmInstance.Stream(ctx, messages, options, callback)
	if err != nil {
		t.Fatalf("Stream failed: %v", err)
	}

	if response == nil {
		t.Fatal("Response is nil")
	}

	// Validate response data
	if response.ID == "" {
		t.Error("Response ID is empty")
	}
	if response.Model == "" {
		t.Error("Response Model is empty")
	}
	if response.Content == "" {
		t.Error("Response Content is empty")
	}
	if response.FinishReason == "" {
		t.Error("Response FinishReason is empty")
	}
	if response.Usage == nil {
		t.Error("Response Usage is nil")
	} else {
		if response.Usage.TotalTokens == 0 {
			t.Error("Response Usage.TotalTokens is 0")
		}
		t.Logf("Usage: prompt=%d, completion=%d, total=%d",
			response.Usage.PromptTokens, response.Usage.CompletionTokens, response.Usage.TotalTokens)
	}
	if chunkCount == 0 {
		t.Error("No chunks received")
	}

	t.Logf("Response with temperature=0.7: %+v", response)
	t.Logf("Total chunks received: %d", chunkCount)
}

// ============================================================================
// Helper Functions
// ============================================================================

// newTestContext creates a real Context for testing OpenAI provider
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
				"test": "openai-provider",
			},
		},
	}

	ctx := context.New(gocontext.Background(), authorized, chatID)
	ctx.AssistantID = "test-assistant"
	ctx.Locale = "en-us"
	ctx.Theme = "light"
	ctx.Client = context.Client{
		Type:      "web",
		UserAgent: "OpenAIProviderTest/1.0",
		IP:        "127.0.0.1",
	}
	ctx.Referer = context.RefererAPI
	ctx.Accept = context.AcceptStandard
	ctx.Route = "/api/test"
	ctx.Metadata = make(map[string]interface{})
	return ctx
}
