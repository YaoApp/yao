package openai

import (
	"context"
)

// Chat the chat struct
type Chat struct {
	ID       string `json:"chat_id"`
	ThreadID string `json:"thread_id"`
}

// NewChat create a new chat
func (ast *OpenAI) NewChat() {}

// Chat the chat
func (ast *OpenAI) Chat(ctx context.Context, messages []map[string]interface{}, option map[string]interface{}, cb func(data []byte) int) (interface{}, error) {
	return nil, nil
}
