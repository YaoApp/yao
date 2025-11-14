package hook

import (
	"github.com/yaoapp/yao/agent/context"
)

// Execute execute the script
func (s *Script) Execute(ctx *context.Context, method string, args ...interface{}) (interface{}, error) {
	if s == nil || s.Script == nil {
		return nil, nil
	}

	scriptCtx, err := s.NewContext(ctx.Sid, nil)
	if err != nil {
		return nil, err
	}
	defer scriptCtx.Close()

	// The first argument is the context
	args = append([]interface{}{ctx}, args...)
	return scriptCtx.CallWith(ctx.Context, method, args...)
}
