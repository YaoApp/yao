package hook

import (
	"github.com/yaoapp/yao/agent/context"
)

// Failback failback hook
func (s *Script) Failback(ctx *context.Context, inputMessages []context.Message, completionResponse *context.CompletionResponse) (*context.ResponseHookFailback, error) {
	return &context.ResponseHookFailback{}, nil
}
