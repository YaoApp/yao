package output

import (
	"github.com/yaoapp/yao/agent/output/message"
)

// Helper functions for creating built-in message types

// NewUserInputMessage creates a user input message (for frontend display)
// content can be string or []ContentPart for multimodal content
func NewUserInputMessage(content interface{}, role, name string) *message.Message {
	props := map[string]interface{}{
		"content": content,
	}
	if role != "" {
		props["role"] = role
	}
	if name != "" {
		props["name"] = name
	}
	return &message.Message{
		Type:  message.TypeUserInput,
		Props: props,
	}
}

// NewTextMessage creates a text message
func NewTextMessage(content string) *message.Message {
	return &message.Message{
		Type: message.TypeText,
		Props: map[string]interface{}{
			"content": content,
		},
	}
}

// NewThinkingMessage creates a thinking message
func NewThinkingMessage(content string) *message.Message {
	return &message.Message{
		Type: message.TypeThinking,
		Props: map[string]interface{}{
			"content": content,
		},
	}
}

// NewLoadingMessage creates a loading message
func NewLoadingMessage(msg string) *message.Message {
	return &message.Message{
		Type: message.TypeLoading,
		Props: map[string]interface{}{
			"message": msg,
		},
	}
}

// NewToolCallMessage creates a tool call message
func NewToolCallMessage(id, name, arguments string) *message.Message {
	return &message.Message{
		Type: message.TypeToolCall,
		Props: map[string]interface{}{
			"id":        id,
			"name":      name,
			"arguments": arguments,
		},
	}
}

// NewErrorMessage creates an error message
func NewErrorMessage(msg, code string) *message.Message {
	return &message.Message{
		Type: message.TypeError,
		Props: map[string]interface{}{
			"message": msg,
			"code":    code,
		},
	}
}

// NewActionMessage creates an action message
func NewActionMessage(name string, payload map[string]interface{}) *message.Message {
	return &message.Message{
		Type: message.TypeAction,
		Props: map[string]interface{}{
			"name":    name,
			"payload": payload,
		},
	}
}

// NewEventMessage creates an event message
func NewEventMessage(event string, msg string, data interface{}) *message.Message {
	return &message.Message{
		Type: message.TypeEvent,
		Props: map[string]interface{}{
			"event":   event,
			"message": msg,
			"data":    data,
		},
	}
}

// NewImageMessage creates an image message
func NewImageMessage(url string, alt string) *message.Message {
	return &message.Message{
		Type: message.TypeImage,
		Props: map[string]interface{}{
			"url": url,
			"alt": alt,
		},
	}
}

// NewAudioMessage creates an audio message
func NewAudioMessage(url string, format string) *message.Message {
	return &message.Message{
		Type: message.TypeAudio,
		Props: map[string]interface{}{
			"url":    url,
			"format": format,
		},
	}
}

// NewVideoMessage creates a video message
func NewVideoMessage(url string) *message.Message {
	return &message.Message{
		Type: message.TypeVideo,
		Props: map[string]interface{}{
			"url": url,
		},
	}
}

// IsBuiltinType checks if a message type is a built-in type
func IsBuiltinType(msgType string) bool {
	switch msgType {
	case message.TypeUserInput, message.TypeText, message.TypeThinking, message.TypeLoading, message.TypeToolCall, message.TypeError, message.TypeImage, message.TypeAudio, message.TypeVideo, message.TypeAction, message.TypeEvent:
		return true
	default:
		return false
	}
}

// GenerateID generates a unique message ID using nanoid
// Deprecated: Use message.GenerateMessageID(), message.GenerateChunkID(),
// message.GenerateBlockID(), or message.GenerateThreadID() instead
func GenerateID() string {
	return message.GenerateNanoID()
}
