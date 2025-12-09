package chat

import "github.com/yaoapp/yao/agent/context"

// =============================================================================
// Completion Types
// =============================================================================

// AppendMessagesRequest represents the request body for appending messages to running completion
type AppendMessagesRequest struct {
	Type     context.InterruptType  `json:"type" binding:"required"` // Interrupt type: "graceful" or "force"
	Messages []context.Message      `json:"messages" binding:"required"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// =============================================================================
// Chat Session Types
// =============================================================================

// UpdateChatRequest represents the request for updating a chat session
type UpdateChatRequest struct {
	Title    *string                `json:"title,omitempty"`    // Chat title
	Status   *string                `json:"status,omitempty"`   // Status: "active" or "archived"
	Metadata map[string]interface{} `json:"metadata,omitempty"` // Additional metadata
}
