package hook

import (
	"github.com/yaoapp/yao/agent/context"
)

// Done done hook
func (s *Script) Done(ctx *context.Context, inputMessages []context.Message, completionResponse *context.ResponseCompletion, mcpResponse *context.ResponseHookMCP) (*context.ResponseHookDone, error) {
	return &context.ResponseHookDone{}, nil
}
