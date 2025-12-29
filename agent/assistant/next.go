package assistant

import (
	"fmt"

	agentContext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/output/message"
)

// processNextResponse processes the Next hook's response and handles agent delegation or custom data
func (ast *Assistant) processNextResponse(npc *NextProcessContext) (*agentContext.Response, error) {
	// If no Next hook response, return standard response
	if npc.NextResponse == nil {
		return ast.buildStandardResponse(npc), nil
	}

	// Handle Delegate: call another agent
	// Note: User input is already buffered by root agent, delegated agent will skip buffering
	if npc.NextResponse.Delegate != nil {
		return ast.handleDelegation(npc.Context, npc.NextResponse.Delegate, npc.StreamHandler)
	}

	// Handle custom Data: return as-is wrapped in standard Response
	if npc.NextResponse.Data != nil {
		return &agentContext.Response{
			ContextID:   npc.Context.ID,
			RequestID:   npc.Context.RequestID(),
			TraceID:     npc.Context.TraceID(),
			ChatID:      npc.Context.ChatID,
			AssistantID: ast.ID,
			Create:      npc.CreateResponse,
			Next:        npc.NextResponse.Data, // Put custom data in Next field
			Completion:  npc.CompletionResponse,
			Tools:       npc.ToolCallResponses,
		}, nil
	}

	// No delegate or data, return standard response
	return ast.buildStandardResponse(npc), nil
}

// handleDelegation handles calling another agent based on DelegateConfig
func (ast *Assistant) handleDelegation(
	ctx *agentContext.Context,
	delegate *agentContext.DelegateConfig,
	streamHandler func(message.StreamChunkType, []byte) int,
) (*agentContext.Response, error) {
	// Load the target assistant
	targetAssistant, err := Get(delegate.AgentID)
	if err != nil {
		return nil, fmt.Errorf("failed to load delegated assistant '%s': %w", delegate.AgentID, err)
	}

	// Call the delegated assistant with the same context
	// The delegated assistant's Stream method will:
	// 1. Call EnterStack() to push itself onto the Stack (creating parent-child relationship)
	// 2. Execute with the same Context (preserving ID, Space, Writer, etc.)
	// 3. Call done() to pop from Stack when finished
	// This ensures proper Stack tracing: parent assistant -> delegated assistant

	// Convert options map from delegate config to Options struct
	delegateOpts := agentContext.OptionsFromMap(delegate.Options)
	return targetAssistant.Stream(ctx, delegate.Messages, delegateOpts)
}

// buildStandardResponse builds the standard agent response when no custom Next hook processing is needed
func (ast *Assistant) buildStandardResponse(npc *NextProcessContext) *agentContext.Response {

	var next interface{} = nil
	if npc.NextResponse != nil {
		next = npc.NextResponse
	}

	return &agentContext.Response{
		ContextID:   npc.Context.ID,
		RequestID:   npc.Context.RequestID(),
		TraceID:     npc.Context.TraceID(),
		ChatID:      npc.Context.ChatID,
		AssistantID: ast.ID,
		Create:      npc.CreateResponse,
		Next:        next,
		Completion:  npc.CompletionResponse,
		Tools:       npc.ToolCallResponses,
	}
}
