package hook

import "github.com/yaoapp/yao/agent/context"

// MCP MCP hook
func (s *Script) MCP(ctx *context.Context, messages []context.Message) (*context.ResponseHookMCP, error) {
	return &context.ResponseHookMCP{}, nil
}
