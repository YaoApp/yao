package assistant

import (
	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/llm"
	"github.com/yaoapp/yao/agent/output/message"
	"github.com/yaoapp/yao/trace/types"
)

// executeLLMStream executes the LLM streaming call with pre-built request
// Returns completionResponse and error
func (ast *Assistant) executeLLMStream(
	ctx *context.Context,
	completionMessages []context.Message,
	completionOptions *context.CompletionOptions,
	agentNode types.Node,
	streamHandler message.StreamFunc,
	opts *context.Options,
) (*context.CompletionResponse, error) {

	// Get connector object (capabilities were already set above, before stream_start)
	conn, capabilities, err := ast.GetConnector(ctx, opts)
	if err != nil {
		ast.traceAgentFail(agentNode, err)
		return nil, err
	}

	// Set capabilities in options if not already set
	if completionOptions.Capabilities == nil && capabilities != nil {
		completionOptions.Capabilities = capabilities
	}

	// Log the capabilities
	ast.traceConnectorCapabilities(agentNode, capabilities)

	// Build content - convert extended types (file, data, __yao.attachment://) to standard LLM types
	// This is done here (right before LLM call) to ensure:
	// 1. autoSearch receives original messages (not converted)
	// 2. delegate receives original messages (not converted)
	// 3. Only the actual LLM call sees converted messages
	llmMessages, err := ast.BuildContent(ctx, completionMessages, completionOptions, opts)
	if err != nil {
		ast.traceAgentFail(agentNode, err)
		return nil, err
	}

	// Trace Add LLM request (use converted messages for trace)
	ast.traceLLMRequest(ctx, conn.ID(), llmMessages, completionOptions)

	// Log LLM call start
	ctx.Logger.LLMStart(conn.ID(), "", len(llmMessages))

	// Create LLM instance with connector and options
	llmInstance, err := llm.New(conn, completionOptions)
	if err != nil {
		// Mark LLM Request as failed in trace
		ast.traceLLMFail(ctx, err)
		return nil, err
	}

	// Call the LLM Completion Stream (streamHandler was set earlier)
	// Use llmMessages (converted) instead of completionMessages (original)
	completionResponse, err := llmInstance.Stream(ctx, llmMessages, completionOptions, streamHandler)

	if err != nil {
		// Mark LLM Request as failed in trace
		ast.traceLLMFail(ctx, err)
		return nil, err
	}

	// Mark LLM Request Complete
	ast.traceLLMComplete(ctx, completionResponse)

	return completionResponse, nil
}

// executeLLMForToolRetry executes LLM call for tool retry with streaming output
// This is used when retrying tool calls - we still want to show LLM's response to users
// Returns completionResponse and error
func (ast *Assistant) executeLLMForToolRetry(
	ctx *context.Context,
	completionMessages []context.Message,
	completionOptions *context.CompletionOptions,
	agentNode types.Node,
	streamHandler message.StreamFunc,
	opts *context.Options,
) (*context.CompletionResponse, error) {

	// Get connector object
	conn, capabilities, err := ast.GetConnector(ctx, opts)
	if err != nil {
		ast.traceAgentFail(agentNode, err)
		return nil, err
	}

	// Set capabilities in options if not already set
	if completionOptions.Capabilities == nil && capabilities != nil {
		completionOptions.Capabilities = capabilities
	}

	// Build content - convert extended types for LLM call
	llmMessages, err := ast.BuildContent(ctx, completionMessages, completionOptions, opts)
	if err != nil {
		ast.traceAgentFail(agentNode, err)
		return nil, err
	}

	// Trace Add LLM retry request
	ast.traceLLMRetryRequest(ctx, conn.ID(), llmMessages, completionOptions)

	// Log LLM call start (retry)
	ctx.Logger.LLMStart(conn.ID(), "", len(llmMessages))

	// Create LLM instance with connector and options
	llmInstance, err := llm.New(conn, completionOptions)
	if err != nil {
		// Mark LLM Retry Request as failed in trace
		ast.traceLLMFail(ctx, err)
		return nil, err
	}

	// Call the LLM Completion Stream (still streaming for tool retry)
	// Use llmMessages (converted) instead of completionMessages (original)
	completionResponse, err := llmInstance.Stream(ctx, llmMessages, completionOptions, streamHandler)
	if err != nil {
		// Mark LLM Retry Request as failed in trace
		ast.traceLLMFail(ctx, err)
		return nil, err
	}

	// Mark LLM Request Complete
	ast.traceLLMComplete(ctx, completionResponse)

	return completionResponse, nil
}
