package assistant

import "github.com/yaoapp/yao/agent/context"

// Stream stream the agent
func (ast *Assistant) Stream(ctx context.Context, messages []context.Message, handler context.StreamFunc) error {
	return nil
}

// Run run the agent
func (ast *Assistant) Run(ctx context.Context, messages []context.Message) (*context.Response, error) {
	return &context.Response{}, nil
}
