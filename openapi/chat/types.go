package chat

import "github.com/yaoapp/yao/agent/context"

// AppendMessagesRequest represents the request body for appending messages to running completion
type AppendMessagesRequest struct {
	Type     context.InterruptType  `json:"type" binding:"required"` // Interrupt type: "graceful" or "force"
	Messages []context.Message      `json:"messages" binding:"required"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}
