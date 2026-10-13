package assistant

import (
	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/llm"
	"github.com/yaoapp/yao/agent/output/message"
)

// executeLLMStream executes the LLM streaming call with pre-built request
// Returns completionResponse and error
func (ast *Assistant) executeLLMStream(
	ctx *context.Context,
	completionMessages []context.Message,
	completionOptions *context.CompletionOptions,
	streamHandler message.StreamFunc,
	opts *context.Options,
) (*context.CompletionResponse, error) {

	// Get connector object (capabilities were already set above, before stream_start)
	conn, capabilities, err := ast.GetConnector(ctx, opts)
	if err != nil {
		return nil, err
	}

	// Set capabilities in options if not already set
	if completionOptions.Capabilities == nil && capabilities != nil {
		completionOptions.Capabilities = capabilities
	}

	// Build content - convert extended types (file, data, __yao.attachment://) to standard LLM types
	// This is done here (right before LLM call) to ensure:
	// 1. autoSearch receives original messages (not converted)
	// 2. delegate receives original messages (not converted)
	// 3. Only the actual LLM call sees converted messages
	llmMessages, err := ast.BuildContent(ctx, completionMessages, completionOptions, opts)
	if err != nil {
		return nil, err
	}

	// Log LLM call start
	ctx.Logger.LLMStart(conn.ID(), "", len(llmMessages))

	// Create LLM instance with connector and options
	llmInstance, err := llm.New(conn, completionOptions)
	if err != nil {
		return nil, err
	}

	// Call the LLM Completion Stream (streamHandler was set earlier)
	// Use llmMessages (converted) instead of completionMessages (original)
	completionResponse, err := llmInstance.Stream(ctx, llmMessages, completionOptions, streamHandler)
	if err != nil {
		return nil, err
	}

	return completionResponse, nil
}

// executeLLMForToolRetry executes LLM call for tool retry with streaming output
// This is used when retrying tool calls - we still want to show LLM's response to users
// Returns completionResponse and error
func (ast *Assistant) executeLLMForToolRetry(
	ctx *context.Context,
	completionMessages []context.Message,
	completionOptions *context.CompletionOptions,
	streamHandler message.StreamFunc,
	opts *context.Options,
) (*context.CompletionResponse, error) {

	// Get connector object
	conn, capabilities, err := ast.GetConnector(ctx, opts)
	if err != nil {
		return nil, err
	}

	// Set capabilities in options if not already set
	if completionOptions.Capabilities == nil && capabilities != nil {
		completionOptions.Capabilities = capabilities
	}

	// Build content - convert extended types for LLM call
	llmMessages, err := ast.BuildContent(ctx, completionMessages, completionOptions, opts)
	if err != nil {
		return nil, err
	}

	// Log LLM call start (retry)
	ctx.Logger.LLMStart(conn.ID(), "", len(llmMessages))

	// Create LLM instance with connector and options
	llmInstance, err := llm.New(conn, completionOptions)
	if err != nil {
		return nil, err
	}

	// Call the LLM Completion Stream (still streaming for tool retry)
	// Use llmMessages (converted) instead of completionMessages (original)
	completionResponse, err := llmInstance.Stream(ctx, llmMessages, completionOptions, streamHandler)
	if err != nil {
		return nil, err
	}

	return completionResponse, nil
}
