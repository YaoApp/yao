//go:build unit

package context_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/context"
)

// =============================================================================
// Message UnmarshalJSON Tests
// =============================================================================

func TestMessageUnmarshalJSON_StringContent(t *testing.T) {
	jsonData := `{
		"role": "user",
		"content": "Hello, world!"
	}`

	var msg context.Message
	err := json.Unmarshal([]byte(jsonData), &msg)
	require.NoError(t, err)

	assert.Equal(t, context.RoleUser, msg.Role)

	content, ok := msg.GetContentAsString()
	require.True(t, ok)
	assert.Equal(t, "Hello, world!", content)
}

func TestMessageUnmarshalJSON_ArrayContent(t *testing.T) {
	jsonData := `{
		"role": "user",
		"content": [
			{
				"type": "text",
				"text": "What's in this image?"
			},
			{
				"type": "image_url",
				"image_url": {
					"url": "https://example.com/image.jpg",
					"detail": "high"
				}
			}
		]
	}`

	var msg context.Message
	err := json.Unmarshal([]byte(jsonData), &msg)
	require.NoError(t, err)

	assert.Equal(t, context.RoleUser, msg.Role)

	parts, ok := msg.GetContentAsParts()
	require.True(t, ok)
	require.Len(t, parts, 2)

	assert.Equal(t, context.ContentText, parts[0].Type)
	assert.Equal(t, "What's in this image?", parts[0].Text)

	assert.Equal(t, context.ContentImageURL, parts[1].Type)
	require.NotNil(t, parts[1].ImageURL)
	assert.Equal(t, "https://example.com/image.jpg", parts[1].ImageURL.URL)
	assert.Equal(t, context.DetailHigh, parts[1].ImageURL.Detail)
}

func TestMessageUnmarshalJSON_NullContent(t *testing.T) {
	jsonData := `{
		"role": "assistant",
		"content": null,
		"tool_calls": [
			{
				"id": "call_123",
				"type": "function",
				"function": {
					"name": "get_weather",
					"arguments": "{\"location\":\"Tokyo\"}"
				}
			}
		]
	}`

	var msg context.Message
	err := json.Unmarshal([]byte(jsonData), &msg)
	require.NoError(t, err)

	assert.Equal(t, context.RoleAssistant, msg.Role)
	assert.Nil(t, msg.Content)

	assert.True(t, msg.HasToolCalls())
	require.Len(t, msg.ToolCalls, 1)
	assert.Equal(t, "call_123", msg.ToolCalls[0].ID)
}

func TestMessageUnmarshalJSON_WithRefusal(t *testing.T) {
	refusalText := "I cannot help with that request."
	jsonData := `{
		"role": "assistant",
		"content": "I'm sorry, but I can't assist with that.",
		"refusal": "I cannot help with that request."
	}`

	var msg context.Message
	err := json.Unmarshal([]byte(jsonData), &msg)
	require.NoError(t, err)

	assert.True(t, msg.IsRefusal())
	require.NotNil(t, msg.Refusal)
	assert.Equal(t, refusalText, *msg.Refusal)
}

func TestMessageUnmarshalJSON_AudioContent(t *testing.T) {
	jsonData := `{
		"role": "user",
		"content": [
			{
				"type": "text",
				"text": "Transcribe this audio"
			},
			{
				"type": "input_audio",
				"input_audio": {
					"data": "base64encodedaudiodata",
					"format": "wav"
				}
			}
		]
	}`

	var msg context.Message
	err := json.Unmarshal([]byte(jsonData), &msg)
	require.NoError(t, err)

	parts, ok := msg.GetContentAsParts()
	require.True(t, ok)
	require.Len(t, parts, 2)

	assert.Equal(t, context.ContentInputAudio, parts[1].Type)
	require.NotNil(t, parts[1].InputAudio)
	assert.Equal(t, "base64encodedaudiodata", parts[1].InputAudio.Data)
	assert.Equal(t, "wav", parts[1].InputAudio.Format)
}

func TestMessageUnmarshalJSON_EmptyContent(t *testing.T) {
	jsonData := `{
		"role": "assistant"
	}`

	var msg context.Message
	err := json.Unmarshal([]byte(jsonData), &msg)
	require.NoError(t, err)

	assert.Equal(t, context.RoleAssistant, msg.Role)
	assert.Nil(t, msg.Content)
}

func TestMessageUnmarshalJSON_WithName(t *testing.T) {
	name := "example_user"
	jsonData := `{
		"role": "user",
		"name": "example_user",
		"content": "Hello"
	}`

	var msg context.Message
	err := json.Unmarshal([]byte(jsonData), &msg)
	require.NoError(t, err)

	require.NotNil(t, msg.Name)
	assert.Equal(t, name, *msg.Name)
}

func TestMessageUnmarshalJSON_WithReasoningContent(t *testing.T) {
	jsonData := `{
		"role": "assistant",
		"reasoning_content": "Let me think about this...",
		"content": "The answer is 42."
	}`

	var msg context.Message
	err := json.Unmarshal([]byte(jsonData), &msg)
	require.NoError(t, err)

	assert.Equal(t, "Let me think about this...", msg.ReasoningContent)
	content, ok := msg.GetContentAsString()
	require.True(t, ok)
	assert.Equal(t, "The answer is 42.", content)
}

// =============================================================================
// Message MarshalJSON Tests
// =============================================================================

func TestMessageMarshalJSON_StringContent(t *testing.T) {
	msg := context.NewTextMessage(context.RoleUser, "Hello, AI!")

	data, err := json.Marshal(msg)
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)

	assert.Equal(t, string(context.RoleUser), result["role"])
	assert.Equal(t, "Hello, AI!", result["content"])
}

func TestMessageMarshalJSON_ArrayContent(t *testing.T) {
	parts := []context.ContentPart{
		{
			Type: context.ContentText,
			Text: "Describe this image",
		},
		{
			Type: context.ContentImageURL,
			ImageURL: &context.ImageURL{
				URL:    "https://example.com/test.jpg",
				Detail: context.DetailLow,
			},
		},
	}

	msg := context.NewMultipartMessage(context.RoleUser, parts)

	data, err := json.Marshal(msg)
	require.NoError(t, err)

	var result context.Message
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)

	resultParts, ok := result.GetContentAsParts()
	require.True(t, ok)
	require.Len(t, resultParts, 2)
}

func TestMessageMarshalJSON_WithToolCalls(t *testing.T) {
	msg := &context.Message{
		Role:    context.RoleAssistant,
		Content: nil,
		ToolCalls: []context.ToolCall{
			{
				ID:   "call_abc123",
				Type: context.ToolTypeFunction,
				Function: context.Function{
					Name:      "get_weather",
					Arguments: `{"location":"San Francisco"}`,
				},
			},
		},
	}

	data, err := json.Marshal(msg)
	require.NoError(t, err)

	var result context.Message
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)

	assert.True(t, result.HasToolCalls())
	require.Len(t, result.ToolCalls, 1)
	assert.Equal(t, "get_weather", result.ToolCalls[0].Function.Name)
}

func TestMessageMarshalJSON_RoundTrip(t *testing.T) {
	refusal := "Cannot help with that."
	original := &context.Message{
		Role:    context.RoleAssistant,
		Content: "Hello",
		Refusal: &refusal,
		ToolCalls: []context.ToolCall{
			{
				ID:   "call_1",
				Type: context.ToolTypeFunction,
				Function: context.Function{
					Name:      "test_func",
					Arguments: `{"key":"value"}`,
				},
			},
		},
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded context.Message
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, original.Role, decoded.Role)
	content, ok := decoded.GetContentAsString()
	require.True(t, ok)
	assert.Equal(t, "Hello", content)
	assert.True(t, decoded.IsRefusal())
	assert.Equal(t, refusal, *decoded.Refusal)
	assert.True(t, decoded.HasToolCalls())
	assert.Equal(t, "test_func", decoded.ToolCalls[0].Function.Name)
}

// =============================================================================
// NewTextMessage / NewMultipartMessage Tests
// =============================================================================

func TestNewTextMessage(t *testing.T) {
	msg := context.NewTextMessage(context.RoleSystem, "You are a helpful assistant.")

	assert.Equal(t, context.RoleSystem, msg.Role)

	content, ok := msg.GetContentAsString()
	require.True(t, ok)
	assert.Equal(t, "You are a helpful assistant.", content)
}

func TestNewTextMessageAllRoles(t *testing.T) {
	roles := []context.MessageRole{
		context.RoleDeveloper,
		context.RoleSystem,
		context.RoleUser,
		context.RoleAssistant,
		context.RoleTool,
	}

	for _, role := range roles {
		msg := context.NewTextMessage(role, "test content")
		assert.Equal(t, role, msg.Role)
	}
}

func TestNewMultipartMessage(t *testing.T) {
	parts := []context.ContentPart{
		{Type: context.ContentText, Text: "Hello"},
	}

	msg := context.NewMultipartMessage(context.RoleUser, parts)

	assert.Equal(t, context.RoleUser, msg.Role)

	resultParts, ok := msg.GetContentAsParts()
	require.True(t, ok)
	require.Len(t, resultParts, 1)
}

func TestNewMultipartMessageWithMultipleParts(t *testing.T) {
	parts := []context.ContentPart{
		{Type: context.ContentText, Text: "Look at this"},
		{
			Type:     context.ContentImageURL,
			ImageURL: &context.ImageURL{URL: "https://example.com/img.jpg", Detail: context.DetailAuto},
		},
		{
			Type:       context.ContentInputAudio,
			InputAudio: &context.InputAudio{Data: "base64data", Format: "mp3"},
		},
	}

	msg := context.NewMultipartMessage(context.RoleUser, parts)

	resultParts, ok := msg.GetContentAsParts()
	require.True(t, ok)
	require.Len(t, resultParts, 3)

	assert.Equal(t, context.ContentText, resultParts[0].Type)
	assert.Equal(t, context.ContentImageURL, resultParts[1].Type)
	assert.Equal(t, context.ContentInputAudio, resultParts[2].Type)
}

// =============================================================================
// GetContentAsString / GetContentAsParts Tests
// =============================================================================

func TestGetContentAsString(t *testing.T) {
	t.Run("StringContent", func(t *testing.T) {
		msg := context.NewTextMessage(context.RoleUser, "Hello")
		content, ok := msg.GetContentAsString()
		assert.True(t, ok)
		assert.Equal(t, "Hello", content)
	})

	t.Run("NonStringContent", func(t *testing.T) {
		msg := context.NewMultipartMessage(context.RoleUser, []context.ContentPart{
			{Type: context.ContentText, Text: "Hello"},
		})
		_, ok := msg.GetContentAsString()
		assert.False(t, ok)
	})

	t.Run("NilContent", func(t *testing.T) {
		msg := &context.Message{Role: context.RoleAssistant, Content: nil}
		_, ok := msg.GetContentAsString()
		assert.False(t, ok)
	})
}

func TestGetContentAsParts(t *testing.T) {
	t.Run("PartsContent", func(t *testing.T) {
		parts := []context.ContentPart{
			{Type: context.ContentText, Text: "Hello"},
			{Type: context.ContentText, Text: "World"},
		}
		msg := context.NewMultipartMessage(context.RoleUser, parts)

		resultParts, ok := msg.GetContentAsParts()
		assert.True(t, ok)
		assert.Len(t, resultParts, 2)
	})

	t.Run("NonPartsContent", func(t *testing.T) {
		msg := context.NewTextMessage(context.RoleUser, "Hello")
		_, ok := msg.GetContentAsParts()
		assert.False(t, ok)
	})

	t.Run("NilContent", func(t *testing.T) {
		msg := &context.Message{Role: context.RoleAssistant, Content: nil}
		_, ok := msg.GetContentAsParts()
		assert.False(t, ok)
	})
}

// =============================================================================
// HasToolCalls / IsRefusal Tests
// =============================================================================

func TestHasToolCalls(t *testing.T) {
	t.Run("WithToolCalls", func(t *testing.T) {
		msg := &context.Message{
			Role: context.RoleAssistant,
			ToolCalls: []context.ToolCall{
				{
					ID:   "call_1",
					Type: context.ToolTypeFunction,
					Function: context.Function{
						Name:      "test",
						Arguments: "{}",
					},
				},
			},
		}
		assert.True(t, msg.HasToolCalls())
	})

	t.Run("WithoutToolCalls", func(t *testing.T) {
		msg := context.NewTextMessage(context.RoleAssistant, "Hello")
		assert.False(t, msg.HasToolCalls())
	})

	t.Run("EmptyToolCalls", func(t *testing.T) {
		msg := &context.Message{
			Role:      context.RoleAssistant,
			ToolCalls: []context.ToolCall{},
		}
		assert.False(t, msg.HasToolCalls())
	})

	t.Run("NilToolCalls", func(t *testing.T) {
		msg := &context.Message{
			Role:      context.RoleAssistant,
			ToolCalls: nil,
		}
		assert.False(t, msg.HasToolCalls())
	})
}

func TestIsRefusal(t *testing.T) {
	t.Run("WithRefusal", func(t *testing.T) {
		refusal := "I cannot help with that."
		msg := &context.Message{
			Role:    context.RoleAssistant,
			Refusal: &refusal,
		}
		assert.True(t, msg.IsRefusal())
	})

	t.Run("WithEmptyRefusal", func(t *testing.T) {
		empty := ""
		msg := &context.Message{
			Role:    context.RoleAssistant,
			Refusal: &empty,
		}
		assert.False(t, msg.IsRefusal())
	})

	t.Run("WithNilRefusal", func(t *testing.T) {
		msg := &context.Message{
			Role:    context.RoleAssistant,
			Refusal: nil,
		}
		assert.False(t, msg.IsRefusal())
	})

	t.Run("NonRefusalMessage", func(t *testing.T) {
		msg := context.NewTextMessage(context.RoleAssistant, "Sure, I can help!")
		assert.False(t, msg.IsRefusal())
	})
}

// =============================================================================
// Tool Message Serialization Tests
// =============================================================================

func TestToolMessageSerialization(t *testing.T) {
	toolCallID := "call_abc123"
	jsonData := `{
		"role": "tool",
		"tool_call_id": "call_abc123",
		"content": "The weather in San Francisco is sunny, 72°F"
	}`

	var msg context.Message
	err := json.Unmarshal([]byte(jsonData), &msg)
	require.NoError(t, err)

	assert.Equal(t, context.RoleTool, msg.Role)
	require.NotNil(t, msg.ToolCallID)
	assert.Equal(t, toolCallID, *msg.ToolCallID)

	content, ok := msg.GetContentAsString()
	require.True(t, ok)
	assert.Equal(t, "The weather in San Francisco is sunny, 72°F", content)
}

func TestToolMessageRoundTrip(t *testing.T) {
	toolCallID := "call_xyz789"
	original := &context.Message{
		Role:       context.RoleTool,
		ToolCallID: &toolCallID,
		Content:    "Result: success",
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded context.Message
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, context.RoleTool, decoded.Role)
	require.NotNil(t, decoded.ToolCallID)
	assert.Equal(t, toolCallID, *decoded.ToolCallID)

	content, ok := decoded.GetContentAsString()
	require.True(t, ok)
	assert.Equal(t, "Result: success", content)
}

// =============================================================================
// Message Role Constants Tests
// =============================================================================

func TestMessageRoleConstants(t *testing.T) {
	assert.Equal(t, context.MessageRole("developer"), context.RoleDeveloper)
	assert.Equal(t, context.MessageRole("system"), context.RoleSystem)
	assert.Equal(t, context.MessageRole("user"), context.RoleUser)
	assert.Equal(t, context.MessageRole("assistant"), context.RoleAssistant)
	assert.Equal(t, context.MessageRole("tool"), context.RoleTool)
}

// =============================================================================
// Content Part Type Constants Tests
// =============================================================================

func TestContentPartTypeConstants(t *testing.T) {
	assert.Equal(t, context.ContentPartType("text"), context.ContentText)
	assert.Equal(t, context.ContentPartType("image_url"), context.ContentImageURL)
	assert.Equal(t, context.ContentPartType("input_audio"), context.ContentInputAudio)
	assert.Equal(t, context.ContentPartType("file"), context.ContentFile)
	assert.Equal(t, context.ContentPartType("data"), context.ContentData)
}

// =============================================================================
// Multiple ToolCalls Tests
// =============================================================================

func TestMultipleToolCalls(t *testing.T) {
	jsonData := `{
		"role": "assistant",
		"content": null,
		"tool_calls": [
			{
				"id": "call_1",
				"type": "function",
				"function": {
					"name": "get_weather",
					"arguments": "{\"location\":\"NYC\"}"
				}
			},
			{
				"id": "call_2",
				"type": "function",
				"function": {
					"name": "get_time",
					"arguments": "{\"timezone\":\"EST\"}"
				}
			}
		]
	}`

	var msg context.Message
	err := json.Unmarshal([]byte(jsonData), &msg)
	require.NoError(t, err)

	assert.True(t, msg.HasToolCalls())
	require.Len(t, msg.ToolCalls, 2)

	assert.Equal(t, "call_1", msg.ToolCalls[0].ID)
	assert.Equal(t, context.ToolTypeFunction, msg.ToolCalls[0].Type)
	assert.Equal(t, "get_weather", msg.ToolCalls[0].Function.Name)

	assert.Equal(t, "call_2", msg.ToolCalls[1].ID)
	assert.Equal(t, "get_time", msg.ToolCalls[1].Function.Name)
}

// =============================================================================
// Image Detail Level Constants Tests
// =============================================================================

func TestImageDetailLevelConstants(t *testing.T) {
	assert.Equal(t, context.ImageDetailLevel("auto"), context.DetailAuto)
	assert.Equal(t, context.ImageDetailLevel("low"), context.DetailLow)
	assert.Equal(t, context.ImageDetailLevel("high"), context.DetailHigh)
}

// =============================================================================
// ToolCall Type Constants Tests
// =============================================================================

func TestToolCallTypeConstants(t *testing.T) {
	assert.Equal(t, context.ToolCallType("function"), context.ToolTypeFunction)
}
