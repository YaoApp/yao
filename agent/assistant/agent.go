package assistant

import (
	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/llm"
)

// Stream stream the agent
func (ast *Assistant) Stream(ctx *context.Context, inputMessages []context.Message, handler context.StreamFunc) (*context.Response, error) {

	var err error

	// Initialize stack and auto-handle completion/failure/restore
	_, traceID, done := context.EnterStack(ctx, ast.ID, ctx.Referer)
	defer done()

	_ = traceID // traceID is available for trace logging

	// Full input messages with chat history
	fullMessages, err := ast.WithHistory(ctx, inputMessages)
	if err != nil {
		return nil, err
	}

	// Request Create hook ( Optional )
	var createResponse *context.ResponseHookCreate
	if ast.Script != nil {
		var err error
		createResponse, err = ast.Script.Create(ctx, fullMessages)
		if err != nil {
			return nil, err
		}
	}
	_ = createResponse // createResponse is available for further processing

	var completionOptions *llm.CompletionOptions // default is nil

	// LLM Call Stream ( Optional )
	var completionMessages []context.Message
	var completionResponse *context.ResponseCompletion
	if ast.Prompts != nil || ast.MCP != nil {
		llm, err := llm.New(ast.GetConnector(ctx))
		if err != nil {
			return nil, err
		}

		// Build the LLM request
		completionMessages, completionOptions, err = ast.BuildLLMRequest(ctx, inputMessages, createResponse)
		if err != nil {
			return nil, err
		}

		// Call the LLM Completion Stream
		completionResponse, err = llm.Stream(ctx, completionMessages, completionOptions, handler)
		if err != nil {
			return nil, err
		}

	}

	// Request MCP hook ( Optional )
	var mcpResponse *context.ResponseHookMCP
	if ast.MCP != nil {
		_ = mcpResponse // mcpResponse is available for further processing

		// MCP Execution Loop
	}

	// Request Done hook ( Optional )
	var doneResponse *context.ResponseHookDone
	if ast.Script != nil {
		var err error
		doneResponse, err = ast.Script.Done(ctx, fullMessages, completionResponse, mcpResponse)
		if err != nil {
			return nil, err
		}
	}

	_ = doneResponse // doneResponse is available for further processing

	return &context.Response{Create: createResponse, Done: doneResponse, Completion: completionResponse}, nil
}

// GetConnector get the connector from the context
func (ast *Assistant) GetConnector(ctx *context.Context) string {
	if ctx.Connector != "" {
		return ctx.Connector
	}
	return ast.Connector
}

// BuildLLMRequest build the LLM request
func (ast *Assistant) BuildLLMRequest(ctx *context.Context, messages []context.Message, createResponse *context.ResponseHookCreate) ([]context.Message, *llm.CompletionOptions, error) {
	return messages, nil, nil
}

// WithHistory with the history messages
func (ast *Assistant) WithHistory(ctx *context.Context, messages []context.Message) ([]context.Message, error) {
	return messages, nil
}
