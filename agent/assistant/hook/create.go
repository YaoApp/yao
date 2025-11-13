package hook

import "github.com/yaoapp/yao/agent/context"

// Create create a new assistant
func (s *Script) Create(ctx *context.Context, messages []context.Message) (*context.ResponseHookCreate, error) {
	return &context.ResponseHookCreate{}, nil
}
