package openai

import (
	"context"
	"fmt"
)

// Chat the chat struct
type Chat struct {
	ID       string `json:"chat_id"`
	ThreadID string `json:"thread_id"`
}

// NewChat create a new chat
func (ast *OpenAI) NewChat() {}

// Chat the chat
func (ast *OpenAI) Chat(ctx context.Context, messages []map[string]interface{}, option map[string]interface{}, cb func(data []byte) int) error {

	if ast.openai == nil {
		return fmt.Errorf("openai is not initialized")
	}

	_, ext := ast.openai.ChatCompletionsWith(ctx, messages, option, cb)
	if ext != nil {
		return fmt.Errorf("openai chat completions with error: %s", ext.Message)
	}

	return nil
}
