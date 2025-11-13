package hook

import "github.com/yaoapp/yao/agent/context"

// Create create a new assistant
func (s *Script) Create(ctx *context.Context, messages []context.Message) (*context.ResponseHookCreate, error) {
	res, err := s.Execute(ctx, "Create", messages)
	if err != nil {
		return nil, err
	}

	_ = res
	return &context.ResponseHookCreate{}, nil
}
