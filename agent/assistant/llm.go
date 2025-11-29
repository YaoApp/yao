package assistant

import (
	"fmt"

	"github.com/yaoapp/kun/log"
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
) (*context.CompletionResponse, error) {

	// === Debug LLM Stream Start ===
	fmt.Println(">>> executeLLMStream: STARTING")
	fmt.Printf(">>> Messages count: %d\n", len(completionMessages))
	fmt.Printf(">>> Tools count: %d\n", len(completionOptions.Tools))
	// === End Debug ===

	// Get connector object (capabilities were already set above, before stream_start)
	conn, capabilities, err := ast.GetConnector(ctx)
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

	// Trace Add LLM request
	ast.traceLLMRequest(ctx, conn.ID(), completionMessages, completionOptions)

	// Create LLM instance with connector and options
	llmInstance, err := llm.New(conn, completionOptions)
	if err != nil {
		return nil, err
	}

	// Call the LLM Completion Stream (streamHandler was set earlier)
	log.Trace("[AGENT] Calling LLM Stream: assistant=%s", ast.ID)

	// === Debug LLM Stream Call ===
	fmt.Println(">>> executeLLMStream: CALLING llmInstance.Stream()")
	// === End Debug ===

	completionResponse, err := llmInstance.Stream(ctx, completionMessages, completionOptions, streamHandler)

	// === Debug LLM Stream Return ===
	fmt.Println(">>> executeLLMStream: llmInstance.Stream() RETURNED")
	fmt.Printf(">>> err: %v\n", err)
	if completionResponse != nil {
		fmt.Printf(">>> ToolCalls: %d\n", len(completionResponse.ToolCalls))
	}
	// === End Debug ===

	log.Trace("[AGENT] LLM Stream returned: assistant=%s, err=%v", ast.ID, err)
	if err != nil {
		log.Trace("[AGENT] Calling sendStreamEndOnError")
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
) (*context.CompletionResponse, error) {

	// Get connector object
	conn, capabilities, err := ast.GetConnector(ctx)
	if err != nil {
		ast.traceAgentFail(agentNode, err)
		return nil, err
	}

	// Set capabilities in options if not already set
	if completionOptions.Capabilities == nil && capabilities != nil {
		completionOptions.Capabilities = capabilities
	}

	// Trace Add LLM retry request
	ast.traceLLMRetryRequest(ctx, conn.ID(), completionMessages, completionOptions)

	// Create LLM instance with connector and options
	llmInstance, err := llm.New(conn, completionOptions)
	if err != nil {
		return nil, err
	}

	// Call the LLM Completion Stream (still streaming for tool retry)
	log.Trace("[AGENT] Calling LLM Stream for tool retry: assistant=%s", ast.ID)
	completionResponse, err := llmInstance.Stream(ctx, completionMessages, completionOptions, streamHandler)
	log.Trace("[AGENT] LLM tool retry stream returned: assistant=%s, err=%v", ast.ID, err)
	if err != nil {
		return nil, err
	}

	// Mark LLM Request Complete
	ast.traceLLMComplete(ctx, completionResponse)

	return completionResponse, nil
}
