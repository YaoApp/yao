package assistant

import "github.com/yaoapp/yao/agent/context"

// Stream stream the agent
func (ast *Assistant) Stream(ctx *context.Context, messages []context.Message, handler context.StreamFunc) error {

	// Initialize stack and auto-handle completion/failure/restore
	_, traceID, done := context.EnterStack(ctx, ast.ID, ctx.Referer)
	defer done()

	_ = traceID // traceID is available for trace logging

	// Request Create hook ( Optional )

	// LLM Call Stream ( Optional )

	// Request Done hook ( Optional )

	return nil
}

// Run run the agent
func (ast *Assistant) Run(ctx *context.Context, messages []context.Message) (*context.Response, error) {

	// Initialize stack and auto-handle completion/failure/restore
	_, traceID, done := context.EnterStack(ctx, ast.ID, ctx.Referer)
	defer done()

	_ = traceID // traceID is available for trace logging

	return &context.Response{}, nil
}
