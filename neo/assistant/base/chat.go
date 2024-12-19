package base

import (
	"context"
	"fmt"
)

// Chat the chat
func (ast *Base) Chat(ctx context.Context, messages []map[string]interface{}, option map[string]interface{}, cb func(data []byte) int) error {

	if ast.openai == nil {
		return fmt.Errorf("api is not initialized")
	}

	_, ext := ast.openai.ChatCompletionsWith(ctx, messages, option, cb)
	if ext != nil {
		return fmt.Errorf("openai chat completions with error: %s", ext.Message)
	}

	return nil
}
