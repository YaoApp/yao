package base

import (
	"context"
)

// Chat the chat
func (ast *Base) Chat(ctx context.Context, messages []map[string]interface{}, option map[string]interface{}, cb func(data []byte) int) (interface{}, error) {
	return nil, nil
}
