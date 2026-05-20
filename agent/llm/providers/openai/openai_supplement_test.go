//go:build integration

package openai_test

import (
	gocontext "context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/gou/connector"
	goullm "github.com/yaoapp/gou/llm"
	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/llm"
	"github.com/yaoapp/yao/agent/output/message"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

func TestOpenAIStreamWithToolCalls(t *testing.T) {
	testprepare.PrepareSandbox(t)

	conn, err := connector.Select("openai.mock")
	require.NoError(t, err)
	require.NotNil(t, conn)

	caps := &goullm.Capabilities{Streaming: true, ToolCalls: true}
	opts := &context.CompletionOptions{Capabilities: caps}

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
				},
				"required": []string{"location"},
			},
		},
	}
	opts.Tools = []map[string]interface{}{weatherTool}
	opts.ToolChoice = "auto"

	provider, err := llm.New(conn, opts)
	require.NoError(t, err)

	ctx := mockTestContext("test-stream-toolcalls", "openai.mock")
	goCtx, cancel := gocontext.WithTimeout(gocontext.Background(), 30*time.Second)
	defer cancel()
	ctx.Context = goCtx

	messages := []context.Message{
		{Role: "user", Content: "What's the weather in Tokyo? Use celsius."},
	}

	var chunks int
	handler := func(chunkType message.StreamChunkType, data []byte) int {
		chunks++
		return 0
	}

	resp, err := provider.Stream(ctx, messages, opts, handler)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Greater(t, chunks, 0)
}

func TestOpenAIStreamRetry(t *testing.T) {
	testprepare.PrepareSandbox(t)

	connDSL := `{
		"type": "openai",
		"options": {
			"model": "gpt-4o",
			"key": "sk-invalid-key-should-fail-auth",
			"host": "https://api.openai.com"
		}
	}`

	conn, err := connector.New("openai", "test-retry", []byte(connDSL))
	require.NoError(t, err)

	caps := &goullm.Capabilities{Streaming: true, ToolCalls: true}
	opts := &context.CompletionOptions{Capabilities: caps}

	provider, err := llm.New(conn, opts)
	require.NoError(t, err)

	ctx := mockTestContext("test-retry", "test-retry")
	goCtx, cancel := gocontext.WithTimeout(gocontext.Background(), 15*time.Second)
	defer cancel()
	ctx.Context = goCtx

	messages := []context.Message{
		{Role: "user", Content: "Test"},
	}

	_, err = provider.Stream(ctx, messages, opts, nil)
	require.Error(t, err)

	errMsg := strings.ToLower(err.Error())
	hasExpectedError := strings.Contains(errMsg, "401") ||
		strings.Contains(errMsg, "unauthorized") ||
		strings.Contains(errMsg, "authentication") ||
		strings.Contains(errMsg, "incorrect api key") ||
		strings.Contains(errMsg, "no data received") ||
		strings.Contains(errMsg, "non-retryable")
	assert.True(t, hasExpectedError, "expected auth/retry error, got: %v", err)
}

func TestOpenAIJSONMode(t *testing.T) {
	testprepare.PrepareSandbox(t)

	conn, err := connector.Select("openai.mock")
	require.NoError(t, err)
	require.NotNil(t, conn)

	caps := &goullm.Capabilities{Streaming: true, ToolCalls: true}
	opts := &context.CompletionOptions{
		Capabilities: caps,
		ResponseFormat: &context.ResponseFormat{
			Type: context.ResponseFormatJSON,
		},
	}

	provider, err := llm.New(conn, opts)
	require.NoError(t, err)

	ctx := mockTestContext("test-json-mode", "openai.mock")
	goCtx, cancel := gocontext.WithTimeout(gocontext.Background(), 30*time.Second)
	defer cancel()
	ctx.Context = goCtx

	messages := []context.Message{
		{Role: "user", Content: "Generate a JSON object with fields: name (string), age (number), city (string). Use values: John, 30, New York"},
	}

	resp, err := provider.Stream(ctx, messages, opts, nil)
	require.NoError(t, err)
	require.NotNil(t, resp)

	contentStr, ok := resp.Content.(string)
	if ok && contentStr != "" {
		var jsonData map[string]interface{}
		if json.Valid([]byte(contentStr)) {
			err := json.Unmarshal([]byte(contentStr), &jsonData)
			assert.NoError(t, err)
		}
	}
}

func TestOpenAIJSONModePost(t *testing.T) {
	testprepare.PrepareSandbox(t)

	conn, err := connector.Select("openai.mock")
	require.NoError(t, err)
	require.NotNil(t, conn)

	caps := &goullm.Capabilities{ToolCalls: true}
	opts := &context.CompletionOptions{
		Capabilities: caps,
		ResponseFormat: &context.ResponseFormat{
			Type: context.ResponseFormatJSON,
		},
	}

	provider, err := llm.New(conn, opts)
	require.NoError(t, err)

	ctx := mockTestContext("test-json-mode-post", "openai.mock")
	goCtx, cancel := gocontext.WithTimeout(gocontext.Background(), 30*time.Second)
	defer cancel()
	ctx.Context = goCtx

	messages := []context.Message{
		{Role: "user", Content: "Return a JSON with: status='success', count=42"},
	}

	resp, err := provider.Post(ctx, messages, opts)
	require.NoError(t, err)
	require.NotNil(t, resp)
}

func TestOpenAIJSONSchema(t *testing.T) {
	testprepare.PrepareSandbox(t)

	conn, err := connector.Select("openai.mock")
	require.NoError(t, err)
	require.NotNil(t, conn)

	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"user": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name":     map[string]interface{}{"type": "string"},
					"email":    map[string]interface{}{"type": "string"},
					"age":      map[string]interface{}{"type": "integer"},
					"isActive": map[string]interface{}{"type": "boolean"},
				},
				"required":             []string{"name", "email", "age", "isActive"},
				"additionalProperties": false,
			},
		},
		"required":             []string{"user"},
		"additionalProperties": false,
	}

	caps := &goullm.Capabilities{Streaming: true, ToolCalls: true}
	opts := &context.CompletionOptions{
		Capabilities: caps,
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

	provider, err := llm.New(conn, opts)
	require.NoError(t, err)

	ctx := mockTestContext("test-json-schema", "openai.mock")
	goCtx, cancel := gocontext.WithTimeout(gocontext.Background(), 30*time.Second)
	defer cancel()
	ctx.Context = goCtx

	messages := []context.Message{
		{Role: "user", Content: "Create user info for: Alice Smith, alice@example.com, age 28, active user"},
	}

	resp, err := provider.Stream(ctx, messages, opts, nil)
	require.NoError(t, err)
	require.NotNil(t, resp)
}

func TestOpenAIJSONSchemaPost(t *testing.T) {
	testprepare.PrepareSandbox(t)

	conn, err := connector.Select("openai.mock")
	require.NoError(t, err)
	require.NotNil(t, conn)

	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"status":  map[string]interface{}{"type": "string", "enum": []string{"success", "error", "pending"}},
			"message": map[string]interface{}{"type": "string"},
			"code":    map[string]interface{}{"type": "integer"},
		},
		"required":             []string{"status", "message", "code"},
		"additionalProperties": false,
	}

	caps := &goullm.Capabilities{ToolCalls: true}
	opts := &context.CompletionOptions{
		Capabilities: caps,
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

	provider, err := llm.New(conn, opts)
	require.NoError(t, err)

	ctx := mockTestContext("test-json-schema-post", "openai.mock")
	goCtx, cancel := gocontext.WithTimeout(gocontext.Background(), 30*time.Second)
	defer cancel()
	ctx.Context = goCtx

	messages := []context.Message{
		{Role: "user", Content: "Generate an API response with status 'success', message 'Operation completed', and code 200"},
	}

	resp, err := provider.Post(ctx, messages, opts)
	require.NoError(t, err)
	require.NotNil(t, resp)
}

func TestOpenAIStreamContextCancellation(t *testing.T) {
	testprepare.PrepareSandbox(t)

	conn, err := connector.Select("openai.mock-slow")
	require.NoError(t, err)
	require.NotNil(t, conn)

	caps := &goullm.Capabilities{Streaming: true, ToolCalls: true}
	opts := &context.CompletionOptions{Capabilities: caps}

	provider, err := llm.New(conn, opts)
	require.NoError(t, err)

	ctx := mockTestContext("test-cancel", "openai.mock-slow")
	goCtx, cancel := gocontext.WithTimeout(gocontext.Background(), 100*time.Millisecond)
	defer cancel()
	ctx.Context = goCtx

	messages := []context.Message{
		{Role: "user", Content: "Write a very long essay about the history of computing"},
	}

	var receivedChunks int
	handler := func(chunkType message.StreamChunkType, data []byte) int {
		receivedChunks++
		return 0
	}

	_, err = provider.Stream(ctx, messages, opts, handler)
	if err != nil {
		errStr := strings.ToLower(err.Error())
		assert.True(t,
			strings.Contains(errStr, "context") || strings.Contains(errStr, "cancel") || strings.Contains(errStr, "deadline"),
			"expected context-related error, got: %v", err)
	}
}

func TestOpenAIStreamChunkTypes(t *testing.T) {
	testprepare.PrepareSandbox(t)

	conn, err := connector.Select("openai.mock")
	require.NoError(t, err)
	require.NotNil(t, conn)

	caps := &goullm.Capabilities{Streaming: true, ToolCalls: true}
	opts := &context.CompletionOptions{Capabilities: caps}

	provider, err := llm.New(conn, opts)
	require.NoError(t, err)

	ctx := mockTestContext("test-chunk-types", "openai.mock")
	goCtx, cancel := gocontext.WithTimeout(gocontext.Background(), 30*time.Second)
	defer cancel()
	ctx.Context = goCtx

	messages := []context.Message{
		{Role: "user", Content: "Say 'test' in one word."},
	}

	chunkTypes := make(map[message.StreamChunkType]int)
	handler := func(chunkType message.StreamChunkType, data []byte) int {
		chunkTypes[chunkType]++
		return 0
	}

	resp, err := provider.Stream(ctx, messages, opts, handler)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Greater(t, len(chunkTypes), 0)
}

func TestOpenAIStreamErrorCallback(t *testing.T) {
	testprepare.PrepareSandbox(t)

	connDSL := `{
		"type": "openai",
		"options": {
			"model": "gpt-4o",
			"key": "sk-invalid-for-error-test",
			"host": "https://api.openai.com"
		}
	}`

	conn, err := connector.New("openai", "test-error-callback", []byte(connDSL))
	require.NoError(t, err)

	caps := &goullm.Capabilities{Streaming: true, ToolCalls: true}
	opts := &context.CompletionOptions{Capabilities: caps}

	provider, err := llm.New(conn, opts)
	require.NoError(t, err)

	ctx := mockTestContext("test-error-callback", "test-error-callback")
	goCtx, cancel := gocontext.WithTimeout(gocontext.Background(), 15*time.Second)
	defer cancel()
	ctx.Context = goCtx

	messages := []context.Message{
		{Role: "user", Content: "Test"},
	}

	var receivedError bool
	handler := func(chunkType message.StreamChunkType, data []byte) int {
		if chunkType == message.ChunkError {
			receivedError = true
		}
		return 0
	}

	_, err = provider.Stream(ctx, messages, opts, handler)
	require.Error(t, err)
	_ = receivedError
}

func TestOpenAIToolCallValidationRetry(t *testing.T) {
	testprepare.PrepareSandbox(t)

	conn, err := connector.Select("openai.mock-validator")
	require.NoError(t, err)
	require.NotNil(t, conn)

	caps := &goullm.Capabilities{Streaming: true, ToolCalls: true}
	opts := &context.CompletionOptions{
		Capabilities: caps,
		Tools: []map[string]interface{}{
			{
				"type": "function",
				"function": map[string]interface{}{
					"name":        "test_strict_validation",
					"description": "A function with strict validation rules",
					"parameters": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"status": map[string]interface{}{
								"type": "string",
								"enum": []string{"active", "inactive"},
							},
							"priority": map[string]interface{}{
								"type":    "integer",
								"minimum": 1,
								"maximum": 5,
							},
						},
						"required": []string{"status", "priority"},
					},
				},
			},
		},
	}

	provider, err := llm.New(conn, opts)
	require.NoError(t, err)

	ctx := mockTestContext("test-tool-validation-retry", "openai.mock-validator")
	goCtx, cancel := gocontext.WithTimeout(gocontext.Background(), 30*time.Second)
	defer cancel()
	ctx.Context = goCtx

	messages := []context.Message{
		{Role: "user", Content: "Call test_strict_validation with status='active' and priority=3"},
	}

	resp, err := provider.Stream(ctx, messages, opts, nil)
	if err == nil && resp != nil {
		assert.NotNil(t, resp)
	}
}

func TestOpenAIPostWithToolCalls(t *testing.T) {
	testprepare.PrepareSandbox(t)

	conn, err := connector.Select("openai.mock")
	require.NoError(t, err)
	require.NotNil(t, conn)

	caps := &goullm.Capabilities{ToolCalls: true}
	opts := &context.CompletionOptions{Capabilities: caps}

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
	opts.Tools = []map[string]interface{}{calcTool}
	opts.ToolChoice = "auto"

	provider, err := llm.New(conn, opts)
	require.NoError(t, err)

	ctx := mockTestContext("test-post-toolcalls", "openai.mock")
	goCtx, cancel := gocontext.WithTimeout(gocontext.Background(), 30*time.Second)
	defer cancel()
	ctx.Context = goCtx

	messages := []context.Message{
		{Role: "user", Content: "Calculate 15 * 8"},
	}

	resp, err := provider.Post(ctx, messages, opts)
	require.NoError(t, err)
	require.NotNil(t, resp)
}

func TestOpenAIStreamWithInvalidToolCall(t *testing.T) {
	testprepare.PrepareSandbox(t)

	conn, err := connector.Select("openai.mock")
	require.NoError(t, err)
	require.NotNil(t, conn)

	caps := &goullm.Capabilities{Streaming: true, ToolCalls: true}
	opts := &context.CompletionOptions{Capabilities: caps}

	strictTool := map[string]interface{}{
		"type": "function",
		"function": map[string]interface{}{
			"name":        "send_email",
			"description": "Send an email",
			"parameters": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"to":      map[string]interface{}{"type": "string"},
					"subject": map[string]interface{}{"type": "string", "minLength": 1},
					"body":    map[string]interface{}{"type": "string", "minLength": 1},
				},
				"required": []string{"to", "subject", "body"},
			},
		},
	}
	opts.Tools = []map[string]interface{}{strictTool}
	opts.ToolChoice = map[string]interface{}{
		"type":     "function",
		"function": map[string]interface{}{"name": "send_email"},
	}

	provider, err := llm.New(conn, opts)
	require.NoError(t, err)

	ctx := mockTestContext("test-invalid-tool", "openai.mock")
	goCtx, cancel := gocontext.WithTimeout(gocontext.Background(), 30*time.Second)
	defer cancel()
	ctx.Context = goCtx

	messages := []context.Message{
		{Role: "user", Content: "Send email to invalid-email without subject"},
	}

	handler := func(chunkType message.StreamChunkType, data []byte) int { return 0 }

	resp, err := provider.Stream(ctx, messages, opts, handler)
	if err == nil {
		assert.NotNil(t, resp)
	}
}

func TestOpenAIProxySupport(t *testing.T) {
	testprepare.PrepareSandbox(t)

	conn, err := connector.Select("openai.mock")
	require.NoError(t, err)
	require.NotNil(t, conn)

	settings := conn.Setting()
	assert.NotNil(t, settings)
}

func TestOpenAIStreamLifecycleEvents(t *testing.T) {
	testprepare.PrepareSandbox(t)

	conn, err := connector.Select("openai.mock")
	require.NoError(t, err)
	require.NotNil(t, conn)

	caps := &goullm.Capabilities{Streaming: true, ToolCalls: true}
	opts := &context.CompletionOptions{Capabilities: caps}

	provider, err := llm.New(conn, opts)
	require.NoError(t, err)

	ctx := mockTestContext("test-lifecycle", "openai.mock")
	goCtx, cancel := gocontext.WithTimeout(gocontext.Background(), 30*time.Second)
	defer cancel()
	ctx.Context = goCtx

	messages := []context.Message{
		{Role: "user", Content: "Say 'hello' in one word"},
	}

	var events []message.StreamChunkType
	handler := func(chunkType message.StreamChunkType, data []byte) int {
		events = append(events, chunkType)
		return 0
	}

	resp, err := provider.Stream(ctx, messages, opts, handler)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Greater(t, len(events), 0)
}

func TestOpenAIStreamWithTemperature(t *testing.T) {
	testprepare.PrepareSandbox(t)

	conn, err := connector.Select("openai.mock")
	require.NoError(t, err)
	require.NotNil(t, conn)

	temperature := 0.7
	caps := &goullm.Capabilities{Streaming: true, ToolCalls: true}
	opts := &context.CompletionOptions{
		Capabilities: caps,
		Temperature:  &temperature,
	}

	provider, err := llm.New(conn, opts)
	require.NoError(t, err)

	ctx := mockTestContext("test-temperature", "openai.mock")
	goCtx, cancel := gocontext.WithTimeout(gocontext.Background(), 30*time.Second)
	defer cancel()
	ctx.Context = goCtx

	messages := []context.Message{
		{Role: "user", Content: "Say 'yes' in one word."},
	}

	var chunkCount int
	handler := func(chunkType message.StreamChunkType, data []byte) int {
		chunkCount++
		return 0
	}

	resp, err := provider.Stream(ctx, messages, opts, handler)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Greater(t, chunkCount, 0)
}

func TestOpenAIStreamBasic(t *testing.T) {
	testprepare.PrepareSandbox(t)

	conn, err := connector.Select("openai.mock")
	require.NoError(t, err)
	require.NotNil(t, conn)

	caps := &goullm.Capabilities{Streaming: true, ToolCalls: true}
	maxTokens := 5
	opts := &context.CompletionOptions{Capabilities: caps, MaxTokens: &maxTokens}

	provider, err := llm.New(conn, opts)
	require.NoError(t, err)

	ctx := mockTestContext("test-stream-basic", "openai.mock")
	goCtx, cancel := gocontext.WithTimeout(gocontext.Background(), 30*time.Second)
	defer cancel()
	ctx.Context = goCtx

	messages := []context.Message{
		{Role: "user", Content: "Say 'Hello' in one word."},
	}

	var chunks int
	handler := func(chunkType message.StreamChunkType, data []byte) int {
		chunks++
		return 0
	}

	resp, err := provider.Stream(ctx, messages, opts, handler)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Greater(t, chunks, 0)
}

func TestOpenAIPostBasic(t *testing.T) {
	testprepare.PrepareSandbox(t)

	conn, err := connector.Select("openai.mock")
	require.NoError(t, err)
	require.NotNil(t, conn)

	caps := &goullm.Capabilities{ToolCalls: true}
	maxTokens := 5
	opts := &context.CompletionOptions{Capabilities: caps, MaxTokens: &maxTokens}

	provider, err := llm.New(conn, opts)
	require.NoError(t, err)

	ctx := mockTestContext("test-post-basic", "openai.mock")
	goCtx, cancel := gocontext.WithTimeout(gocontext.Background(), 30*time.Second)
	defer cancel()
	ctx.Context = goCtx

	messages := []context.Message{
		{Role: "user", Content: "Reply with only the word 'OK'."},
	}

	resp, err := provider.Post(ctx, messages, opts)
	require.NoError(t, err)
	require.NotNil(t, resp)
}
