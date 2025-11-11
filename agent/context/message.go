package context

import (
	"encoding/json"
	"fmt"
)

// UnmarshalJSON custom unmarshaler for Message to handle Content field
func (m *Message) UnmarshalJSON(data []byte) error {
	// Define a temporary struct to avoid infinite recursion
	type Alias Message
	aux := &struct {
		Content json.RawMessage `json:"content,omitempty"`
		*Alias
	}{
		Alias: (*Alias)(m),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// If content is empty, return early
	if len(aux.Content) == 0 || string(aux.Content) == "null" {
		m.Content = nil
		return nil
	}

	// Try to unmarshal as string first
	var contentStr string
	if err := json.Unmarshal(aux.Content, &contentStr); err == nil {
		m.Content = contentStr
		return nil
	}

	// Try to unmarshal as array of ContentPart
	var contentParts []ContentPart
	if err := json.Unmarshal(aux.Content, &contentParts); err == nil {
		m.Content = contentParts
		return nil
	}

	return fmt.Errorf("content must be either a string or an array of ContentPart")
}

// MarshalJSON custom marshaler for Message
func (m *Message) MarshalJSON() ([]byte, error) {
	type Alias Message
	return json.Marshal(&struct {
		*Alias
	}{
		Alias: (*Alias)(m),
	})
}

// NewTextMessage creates a new message with text content
func NewTextMessage(role MessageRole, text string) *Message {
	return &Message{
		Role:    role,
		Content: text,
	}
}

// NewMultipartMessage creates a new message with multipart content
func NewMultipartMessage(role MessageRole, parts []ContentPart) *Message {
	return &Message{
		Role:    role,
		Content: parts,
	}
}

// GetContentAsString returns content as string if possible
func (m *Message) GetContentAsString() (string, bool) {
	if str, ok := m.Content.(string); ok {
		return str, true
	}
	return "", false
}

// GetContentAsParts returns content as ContentPart array if possible
func (m *Message) GetContentAsParts() ([]ContentPart, bool) {
	if parts, ok := m.Content.([]ContentPart); ok {
		return parts, true
	}
	return nil, false
}

// HasToolCalls checks if the message has tool calls
func (m *Message) HasToolCalls() bool {
	return len(m.ToolCalls) > 0
}

// IsRefusal checks if the message is a refusal
func (m *Message) IsRefusal() bool {
	return m.Refusal != nil && *m.Refusal != ""
}
