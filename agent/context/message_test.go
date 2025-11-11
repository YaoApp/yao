package context

import (
	"encoding/json"
	"testing"
)

func TestMessage_UnmarshalJSON_StringContent(t *testing.T) {
	jsonData := `{
		"role": "user",
		"content": "Hello, world!"
	}`

	var msg Message
	err := json.Unmarshal([]byte(jsonData), &msg)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if msg.Role != RoleUser {
		t.Errorf("Expected role %s, got %s", RoleUser, msg.Role)
	}

	content, ok := msg.GetContentAsString()
	if !ok {
		t.Fatal("Expected content to be string")
	}

	if content != "Hello, world!" {
		t.Errorf("Expected content 'Hello, world!', got '%s'", content)
	}
}

func TestMessage_UnmarshalJSON_ArrayContent(t *testing.T) {
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

	var msg Message
	err := json.Unmarshal([]byte(jsonData), &msg)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if msg.Role != RoleUser {
		t.Errorf("Expected role %s, got %s", RoleUser, msg.Role)
	}

	parts, ok := msg.GetContentAsParts()
	if !ok {
		t.Fatal("Expected content to be array of ContentPart")
	}

	if len(parts) != 2 {
		t.Fatalf("Expected 2 content parts, got %d", len(parts))
	}

	// Check first part (text)
	if parts[0].Type != ContentText {
		t.Errorf("Expected type %s, got %s", ContentText, parts[0].Type)
	}
	if parts[0].Text != "What's in this image?" {
		t.Errorf("Expected text 'What's in this image?', got '%s'", parts[0].Text)
	}

	// Check second part (image)
	if parts[1].Type != ContentImageURL {
		t.Errorf("Expected type %s, got %s", ContentImageURL, parts[1].Type)
	}
	if parts[1].ImageURL == nil {
		t.Fatal("Expected ImageURL to be non-nil")
	}
	if parts[1].ImageURL.URL != "https://example.com/image.jpg" {
		t.Errorf("Expected URL 'https://example.com/image.jpg', got '%s'", parts[1].ImageURL.URL)
	}
	if parts[1].ImageURL.Detail != DetailHigh {
		t.Errorf("Expected detail %s, got %s", DetailHigh, parts[1].ImageURL.Detail)
	}
}

func TestMessage_UnmarshalJSON_NullContent(t *testing.T) {
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

	var msg Message
	err := json.Unmarshal([]byte(jsonData), &msg)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if msg.Role != RoleAssistant {
		t.Errorf("Expected role %s, got %s", RoleAssistant, msg.Role)
	}

	if msg.Content != nil {
		t.Errorf("Expected content to be nil, got %v", msg.Content)
	}

	if !msg.HasToolCalls() {
		t.Fatal("Expected message to have tool calls")
	}

	if len(msg.ToolCalls) != 1 {
		t.Fatalf("Expected 1 tool call, got %d", len(msg.ToolCalls))
	}

	if msg.ToolCalls[0].ID != "call_123" {
		t.Errorf("Expected tool call ID 'call_123', got '%s'", msg.ToolCalls[0].ID)
	}
}

func TestMessage_UnmarshalJSON_WithRefusal(t *testing.T) {
	refusalText := "I cannot help with that request."
	jsonData := `{
		"role": "assistant",
		"content": "I'm sorry, but I can't assist with that.",
		"refusal": "I cannot help with that request."
	}`

	var msg Message
	err := json.Unmarshal([]byte(jsonData), &msg)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if !msg.IsRefusal() {
		t.Error("Expected message to be a refusal")
	}

	if msg.Refusal == nil {
		t.Fatal("Expected refusal to be non-nil")
	}

	if *msg.Refusal != refusalText {
		t.Errorf("Expected refusal '%s', got '%s'", refusalText, *msg.Refusal)
	}
}

func TestMessage_UnmarshalJSON_AudioContent(t *testing.T) {
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

	var msg Message
	err := json.Unmarshal([]byte(jsonData), &msg)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	parts, ok := msg.GetContentAsParts()
	if !ok {
		t.Fatal("Expected content to be array of ContentPart")
	}

	if len(parts) != 2 {
		t.Fatalf("Expected 2 content parts, got %d", len(parts))
	}

	// Check audio part
	if parts[1].Type != ContentInputAudio {
		t.Errorf("Expected type %s, got %s", ContentInputAudio, parts[1].Type)
	}
	if parts[1].InputAudio == nil {
		t.Fatal("Expected InputAudio to be non-nil")
	}
	if parts[1].InputAudio.Data != "base64encodedaudiodata" {
		t.Errorf("Expected audio data 'base64encodedaudiodata', got '%s'", parts[1].InputAudio.Data)
	}
	if parts[1].InputAudio.Format != "wav" {
		t.Errorf("Expected format 'wav', got '%s'", parts[1].InputAudio.Format)
	}
}

func TestMessage_MarshalJSON_StringContent(t *testing.T) {
	msg := NewTextMessage(RoleUser, "Hello, AI!")

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	if result["role"] != string(RoleUser) {
		t.Errorf("Expected role %s, got %v", RoleUser, result["role"])
	}

	if result["content"] != "Hello, AI!" {
		t.Errorf("Expected content 'Hello, AI!', got %v", result["content"])
	}
}

func TestMessage_MarshalJSON_ArrayContent(t *testing.T) {
	parts := []ContentPart{
		{
			Type: ContentText,
			Text: "Describe this image",
		},
		{
			Type: ContentImageURL,
			ImageURL: &ImageURL{
				URL:    "https://example.com/test.jpg",
				Detail: DetailLow,
			},
		},
	}

	msg := NewMultipartMessage(RoleUser, parts)

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Unmarshal back to verify
	var result Message
	err = json.Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	resultParts, ok := result.GetContentAsParts()
	if !ok {
		t.Fatal("Expected content to be array of ContentPart")
	}

	if len(resultParts) != 2 {
		t.Fatalf("Expected 2 content parts, got %d", len(resultParts))
	}
}

func TestMessage_MarshalJSON_WithToolCalls(t *testing.T) {
	msg := &Message{
		Role:    RoleAssistant,
		Content: nil,
		ToolCalls: []ToolCall{
			{
				ID:   "call_abc123",
				Type: ToolTypeFunction,
				Function: Function{
					Name:      "get_weather",
					Arguments: `{"location":"San Francisco"}`,
				},
			},
		},
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Unmarshal back to verify
	var result Message
	err = json.Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	if !result.HasToolCalls() {
		t.Error("Expected message to have tool calls")
	}

	if len(result.ToolCalls) != 1 {
		t.Fatalf("Expected 1 tool call, got %d", len(result.ToolCalls))
	}

	if result.ToolCalls[0].Function.Name != "get_weather" {
		t.Errorf("Expected function name 'get_weather', got '%s'", result.ToolCalls[0].Function.Name)
	}
}

func TestMessage_ToolMessage(t *testing.T) {
	toolCallID := "call_abc123"
	jsonData := `{
		"role": "tool",
		"tool_call_id": "call_abc123",
		"content": "The weather in San Francisco is sunny, 72°F"
	}`

	var msg Message
	err := json.Unmarshal([]byte(jsonData), &msg)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if msg.Role != RoleTool {
		t.Errorf("Expected role %s, got %s", RoleTool, msg.Role)
	}

	if msg.ToolCallID == nil {
		t.Fatal("Expected tool_call_id to be non-nil")
	}

	if *msg.ToolCallID != toolCallID {
		t.Errorf("Expected tool_call_id '%s', got '%s'", toolCallID, *msg.ToolCallID)
	}

	content, ok := msg.GetContentAsString()
	if !ok {
		t.Fatal("Expected content to be string")
	}

	if content != "The weather in San Francisco is sunny, 72°F" {
		t.Errorf("Unexpected content: %s", content)
	}
}

func TestNewTextMessage(t *testing.T) {
	msg := NewTextMessage(RoleSystem, "You are a helpful assistant.")

	if msg.Role != RoleSystem {
		t.Errorf("Expected role %s, got %s", RoleSystem, msg.Role)
	}

	content, ok := msg.GetContentAsString()
	if !ok {
		t.Fatal("Expected content to be string")
	}

	if content != "You are a helpful assistant." {
		t.Errorf("Expected content 'You are a helpful assistant.', got '%s'", content)
	}
}

func TestNewMultipartMessage(t *testing.T) {
	parts := []ContentPart{
		{Type: ContentText, Text: "Hello"},
	}

	msg := NewMultipartMessage(RoleUser, parts)

	if msg.Role != RoleUser {
		t.Errorf("Expected role %s, got %s", RoleUser, msg.Role)
	}

	resultParts, ok := msg.GetContentAsParts()
	if !ok {
		t.Fatal("Expected content to be array of ContentPart")
	}

	if len(resultParts) != 1 {
		t.Fatalf("Expected 1 content part, got %d", len(resultParts))
	}
}
