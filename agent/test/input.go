package test

import (
	"fmt"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/yao/agent/context"
)

// ParseInput converts various input formats to []context.Message
// Supported formats:
//   - string: converted to single user message
//   - map (Message): single message with role and content
//   - []interface{} ([]Message): array of messages (conversation history)
func ParseInput(input interface{}) ([]context.Message, error) {
	if input == nil {
		return nil, fmt.Errorf("input is nil")
	}

	switch v := input.(type) {
	case string:
		// Simple string input -> single user message
		return []context.Message{
			{
				Role:    context.RoleUser,
				Content: v,
			},
		}, nil

	case map[string]interface{}:
		// Single message object
		msg, err := parseMessageMap(v)
		if err != nil {
			return nil, fmt.Errorf("failed to parse message: %w", err)
		}
		return []context.Message{*msg}, nil

	case []interface{}:
		// Array of messages (conversation history)
		messages := make([]context.Message, 0, len(v))
		for i, item := range v {
			switch m := item.(type) {
			case map[string]interface{}:
				msg, err := parseMessageMap(m)
				if err != nil {
					return nil, fmt.Errorf("failed to parse message at index %d: %w", i, err)
				}
				messages = append(messages, *msg)
			default:
				return nil, fmt.Errorf("invalid message type at index %d: expected object, got %T", i, item)
			}
		}
		return messages, nil

	default:
		return nil, fmt.Errorf("unsupported input type: %T", input)
	}
}

// parseMessageMap converts a map to context.Message
func parseMessageMap(m map[string]interface{}) (*context.Message, error) {
	msg := &context.Message{}

	// Parse role (required)
	if role, ok := m["role"].(string); ok {
		msg.Role = context.MessageRole(role)
	} else {
		// Default to user role if not specified
		msg.Role = context.RoleUser
	}

	// Parse content (required)
	if content, ok := m["content"]; ok {
		msg.Content = content
	} else {
		return nil, fmt.Errorf("message missing 'content' field")
	}

	// Parse optional name
	if name, ok := m["name"].(string); ok {
		msg.Name = &name
	}

	// Parse optional tool_call_id (for tool messages)
	if toolCallID, ok := m["tool_call_id"].(string); ok {
		msg.ToolCallID = &toolCallID
	}

	// Parse optional tool_calls (for assistant messages)
	if toolCalls, ok := m["tool_calls"].([]interface{}); ok {
		msg.ToolCalls = make([]context.ToolCall, 0, len(toolCalls))
		for _, tc := range toolCalls {
			if tcMap, ok := tc.(map[string]interface{}); ok {
				toolCall, err := parseToolCall(tcMap)
				if err != nil {
					return nil, fmt.Errorf("failed to parse tool_call: %w", err)
				}
				msg.ToolCalls = append(msg.ToolCalls, *toolCall)
			}
		}
	}

	// Parse optional refusal (for assistant messages)
	if refusal, ok := m["refusal"].(string); ok {
		msg.Refusal = &refusal
	}

	return msg, nil
}

// parseToolCall converts a map to context.ToolCall
func parseToolCall(m map[string]interface{}) (*context.ToolCall, error) {
	tc := &context.ToolCall{}

	if id, ok := m["id"].(string); ok {
		tc.ID = id
	}

	if typ, ok := m["type"].(string); ok {
		tc.Type = context.ToolCallType(typ)
	} else {
		tc.Type = context.ToolTypeFunction
	}

	if fn, ok := m["function"].(map[string]interface{}); ok {
		if name, ok := fn["name"].(string); ok {
			tc.Function.Name = name
		}
		if args, ok := fn["arguments"].(string); ok {
			tc.Function.Arguments = args
		} else if args, ok := fn["arguments"].(map[string]interface{}); ok {
			// Convert map to JSON string
			argsBytes, err := jsoniter.Marshal(args)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal arguments: %w", err)
			}
			tc.Function.Arguments = string(argsBytes)
		}
	}

	return tc, nil
}

// ExtractTextContent extracts text content from various content formats
// Used for display in reports
func ExtractTextContent(content interface{}) string {
	if content == nil {
		return ""
	}

	switch v := content.(type) {
	case string:
		return v

	case []interface{}:
		// ContentPart array
		var texts []string
		for _, part := range v {
			if partMap, ok := part.(map[string]interface{}); ok {
				if partMap["type"] == "text" {
					if text, ok := partMap["text"].(string); ok {
						texts = append(texts, text)
					}
				}
			}
		}
		if len(texts) > 0 {
			result := texts[0]
			for i := 1; i < len(texts); i++ {
				result += "\n" + texts[i]
			}
			return result
		}
		return fmt.Sprintf("[%d content parts]", len(v))

	case map[string]interface{}:
		// Single ContentPart or Message
		if v["type"] == "text" {
			if text, ok := v["text"].(string); ok {
				return text
			}
		}
		if content, ok := v["content"]; ok {
			return ExtractTextContent(content)
		}
		return fmt.Sprintf("%v", v)

	default:
		return fmt.Sprintf("%v", v)
	}
}

// SummarizeInput creates a short summary of the input for display
func SummarizeInput(input interface{}, maxLen int) string {
	text := ""

	switch v := input.(type) {
	case string:
		text = v

	case map[string]interface{}:
		if content, ok := v["content"]; ok {
			text = ExtractTextContent(content)
		}

	case []interface{}:
		// Get the last user message for summary
		for i := len(v) - 1; i >= 0; i-- {
			if msg, ok := v[i].(map[string]interface{}); ok {
				if msg["role"] == "user" {
					if content, ok := msg["content"]; ok {
						text = ExtractTextContent(content)
						break
					}
				}
			}
		}
		if text == "" && len(v) > 0 {
			text = fmt.Sprintf("[%d messages]", len(v))
		}

	default:
		text = fmt.Sprintf("%v", v)
	}

	if maxLen > 0 && len(text) > maxLen {
		return text[:maxLen-3] + "..."
	}
	return text
}
