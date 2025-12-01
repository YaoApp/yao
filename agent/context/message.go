package context

import (
	"encoding/json"
	"fmt"
	"sync"
)

// messageMetadataStore provides thread-safe storage for message and block metadata
type messageMetadataStore struct {
	messages map[string]*MessageMetadata // Message metadata by MessageID
	blocks   map[string]*BlockMetadata   // Block metadata by BlockID
	mu       sync.RWMutex
}

// newMessageMetadataStore creates a new message metadata store
func newMessageMetadataStore() *messageMetadataStore {
	return &messageMetadataStore{
		messages: make(map[string]*MessageMetadata),
		blocks:   make(map[string]*BlockMetadata),
	}
}

// setMessage stores metadata for a message (thread-safe)
func (s *messageMetadataStore) setMessage(messageID string, metadata *MessageMetadata) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messages[messageID] = metadata
}

// getMessage retrieves metadata for a message (thread-safe)
func (s *messageMetadataStore) getMessage(messageID string) *MessageMetadata {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.messages[messageID]
}

// setBlock stores metadata for a block (thread-safe)
func (s *messageMetadataStore) setBlock(blockID string, metadata *BlockMetadata) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.blocks[blockID] = metadata
}

// getBlock retrieves metadata for a block (thread-safe)
func (s *messageMetadataStore) getBlock(blockID string) *BlockMetadata {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.blocks[blockID]
}

// updateBlock updates block metadata (thread-safe)
func (s *messageMetadataStore) updateBlock(blockID string, update func(*BlockMetadata)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if block, exists := s.blocks[blockID]; exists {
		update(block)
	}
}

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
