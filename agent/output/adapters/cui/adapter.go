package cui

import "github.com/yaoapp/yao/agent/output/message"

// Adapter implements the message.Adapter interface for CUI clients.
// It performs no conversion and outputs messages as-is, as CUI clients
// are designed to directly consume the universal DSL.
type Adapter struct{}

// NewAdapter creates a new CUI adapter.
func NewAdapter() *Adapter {
	return &Adapter{}
}

// Adapt converts a universal Message to one or more client-specific chunks.
// For CUI, it simply returns the original message as a single chunk.
func (a *Adapter) Adapt(msg *message.Message) ([]interface{}, error) {
	// CUI clients consume the universal DSL directly, so no conversion is needed.
	// This includes all message types like text, thinking, loading, events, etc.
	// CUI clients can choose to display or ignore event messages.
	return []interface{}{msg}, nil
}

// SupportsType checks if the adapter explicitly supports a given message type.
// CUI adapter supports all types as it renders them directly.
func (a *Adapter) SupportsType(msgType string) bool {
	return true
}
