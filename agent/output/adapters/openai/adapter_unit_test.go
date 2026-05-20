//go:build unit

package openai_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/output/adapters/openai"
	"github.com/yaoapp/yao/agent/output/message"
)

func TestNewAdapter_Default(t *testing.T) {
	adapter := openai.NewAdapter()
	require.NotNil(t, adapter)

	config := adapter.GetConfig()
	require.NotNil(t, config)
	assert.Equal(t, "yao-agent", config.Model)
	assert.NotNil(t, config.LinkTemplates)
	assert.Nil(t, config.LinkTransformer)
}

func TestNewAdapter_WithOptions(t *testing.T) {
	adapter := openai.NewAdapter(
		openai.WithBaseURL("https://api.example.com"),
		openai.WithModel("gpt-4"),
		openai.WithLocale("zh-CN"),
	)
	require.NotNil(t, adapter)

	config := adapter.GetConfig()
	assert.Equal(t, "https://api.example.com", config.BaseURL)
	assert.Equal(t, "gpt-4", config.Model)
	assert.Equal(t, "zh-CN", config.Locale)
}

func TestAdapt_TextMessage(t *testing.T) {
	adapter := openai.NewAdapter()
	msg := &message.Message{
		Type: message.TypeText,
		Props: map[string]interface{}{
			"content": "Hello, world!",
		},
	}

	chunks, err := adapter.Adapt(msg)
	require.NoError(t, err)
	require.NotEmpty(t, chunks)

	chunk, ok := chunks[0].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "chat.completion.chunk", chunk["object"])

	choices, ok := chunk["choices"].([]map[string]interface{})
	require.True(t, ok)
	require.Len(t, choices, 1)

	delta, ok := choices[0]["delta"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "Hello, world!", delta["content"])
}

func TestAdapt_ThinkingMessage(t *testing.T) {
	adapter := openai.NewAdapter()
	msg := &message.Message{
		Type: message.TypeThinking,
		Props: map[string]interface{}{
			"content": "Let me think about this...",
		},
	}

	chunks, err := adapter.Adapt(msg)
	require.NoError(t, err)
	require.NotEmpty(t, chunks)

	chunk, ok := chunks[0].(map[string]interface{})
	require.True(t, ok)

	choices, ok := chunk["choices"].([]map[string]interface{})
	require.True(t, ok)
	require.Len(t, choices, 1)

	delta, ok := choices[0]["delta"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "Let me think about this...", delta["reasoning_content"])
}

func TestAdapt_ErrorMessage(t *testing.T) {
	adapter := openai.NewAdapter()
	msg := &message.Message{
		Type: message.TypeError,
		Props: map[string]interface{}{
			"message": "something failed",
			"code":    "server_error",
		},
	}

	chunks, err := adapter.Adapt(msg)
	require.NoError(t, err)
	require.NotEmpty(t, chunks)

	chunk, ok := chunks[0].(map[string]interface{})
	require.True(t, ok)

	errObj, ok := chunk["error"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "something failed", errObj["message"])
	assert.Equal(t, "server_error", errObj["code"])
}

func TestAdapt_UnknownType(t *testing.T) {
	adapter := openai.NewAdapter()
	msg := &message.Message{
		Type: "custom_widget",
		Props: map[string]interface{}{
			"url": "https://example.com/widget/123",
		},
	}

	chunks, err := adapter.Adapt(msg)
	require.NoError(t, err)
	require.NotEmpty(t, chunks)

	chunk, ok := chunks[0].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "chat.completion.chunk", chunk["object"])
}

func TestAdapt_ActionMessage(t *testing.T) {
	adapter := openai.NewAdapter()
	msg := &message.Message{
		Type: message.TypeAction,
		Props: map[string]interface{}{
			"name":    "open_panel",
			"payload": map[string]interface{}{"panel": "settings"},
		},
	}

	chunks, err := adapter.Adapt(msg)
	require.NoError(t, err)
	assert.Empty(t, chunks)
}

func TestAdapt_EventMessage(t *testing.T) {
	adapter := openai.NewAdapter()
	msg := &message.Message{
		Type: message.TypeEvent,
		Props: map[string]interface{}{
			"event":   "stream_end",
			"message": "Stream ended",
		},
	}

	chunks, err := adapter.Adapt(msg)
	require.NoError(t, err)
	assert.Empty(t, chunks)
}

func TestSupportsType(t *testing.T) {
	adapter := openai.NewAdapter()

	assert.True(t, adapter.SupportsType(message.TypeText))
	assert.True(t, adapter.SupportsType(message.TypeThinking))
	assert.True(t, adapter.SupportsType(message.TypeError))
	assert.True(t, adapter.SupportsType(message.TypeToolCall))
	assert.True(t, adapter.SupportsType(message.TypeImage))
	assert.True(t, adapter.SupportsType(message.TypeAction))

	assert.False(t, adapter.SupportsType("custom_widget"))
	assert.False(t, adapter.SupportsType("unknown"))
}

func TestAdapt_LoadingMessage(t *testing.T) {
	adapter := openai.NewAdapter()
	msg := &message.Message{
		Type: message.TypeLoading,
		Props: map[string]interface{}{
			"message": "Searching knowledge base...",
		},
	}

	chunks, err := adapter.Adapt(msg)
	require.NoError(t, err)
	require.NotEmpty(t, chunks)

	chunk, ok := chunks[0].(map[string]interface{})
	require.True(t, ok)

	choices, ok := chunk["choices"].([]map[string]interface{})
	require.True(t, ok)
	require.Len(t, choices, 1)

	delta, ok := choices[0]["delta"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "Searching knowledge base...", delta["reasoning_content"])
}

func TestAdapt_ToolCallMessage(t *testing.T) {
	adapter := openai.NewAdapter()
	msg := &message.Message{
		Type: message.TypeToolCall,
		Props: map[string]interface{}{
			"id":   "call_abc123",
			"name": "get_weather",
			"function": map[string]interface{}{
				"name":      "get_weather",
				"arguments": `{"city":"Beijing"}`,
			},
		},
	}

	chunks, err := adapter.Adapt(msg)
	require.NoError(t, err)
	require.NotEmpty(t, chunks)

	chunk, ok := chunks[0].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "chat.completion.chunk", chunk["object"])
}

func TestAdapt_ImageMessage(t *testing.T) {
	adapter := openai.NewAdapter()
	msg := &message.Message{
		Type: message.TypeImage,
		Props: map[string]interface{}{
			"url": "https://example.com/image.png",
			"alt": "A beautiful sunset",
		},
	}

	chunks, err := adapter.Adapt(msg)
	require.NoError(t, err)
	require.NotEmpty(t, chunks)

	chunk, ok := chunks[0].(map[string]interface{})
	require.True(t, ok)

	choices, ok := chunk["choices"].([]map[string]interface{})
	require.True(t, ok)
	require.Len(t, choices, 1)

	delta, ok := choices[0]["delta"].(map[string]interface{})
	require.True(t, ok)
	content, ok := delta["content"].(string)
	require.True(t, ok)
	assert.Contains(t, content, "https://example.com/image.png")
	assert.Contains(t, content, "A beautiful sunset")
}
