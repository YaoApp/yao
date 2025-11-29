package assistant

import (
	"fmt"

	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/i18n"
	"github.com/yaoapp/yao/trace/types"
)

// initAgentTraceNode creates and returns the agent trace node
func (ast *Assistant) initAgentTraceNode(ctx *context.Context, inputMessages []context.Message) types.Node {
	trace, _ := ctx.Trace()
	if trace == nil {
		return nil
	}

	agentNode, _ := trace.Add(inputMessages, types.TraceNodeOption{
		Label:       i18n.Tr(ast.ID, ctx.Locale, "assistant.agent.stream.label"), // "Assistant {{name}}"
		Type:        "agent",
		Icon:        "assistant",
		Description: i18n.Tr(ast.ID, ctx.Locale, "assistant.agent.stream.description"), // "Assistant {{name}} is processing the request"
	})

	return agentNode
}

// traceAgentHistory logs the chat history to the agent trace node
func (ast *Assistant) traceAgentHistory(ctx *context.Context, agentNode types.Node, fullMessages []context.Message) {
	if agentNode == nil {
		return
	}

	agentNode.Info(
		i18n.Tr(ast.ID, ctx.Locale, "assistant.agent.stream.history"), // "Get Chat History"
		map[string]any{"messages": fullMessages},
	)
}

// traceCreateHook logs the create hook response to the agent trace node
func (ast *Assistant) traceCreateHook(agentNode types.Node, createResponse *context.HookCreateResponse) {
	if agentNode == nil {
		return
	}

	agentNode.Debug("Call Create Hook", map[string]any{"response": createResponse})
}

// traceConnectorCapabilities logs the connector capabilities to the agent trace node
func (ast *Assistant) traceConnectorCapabilities(agentNode types.Node, capabilities *context.ModelCapabilities) {
	if agentNode == nil {
		return
	}

	agentNode.Debug("Get Connector Capabilities", map[string]any{"capabilities": capabilities})
}

// traceLLMRequest adds a LLM trace node to the trace
func (ast *Assistant) traceLLMRequest(ctx *context.Context, connID string, completionMessages []context.Message, completionOptions *context.CompletionOptions) {
	trace, _ := ctx.Trace()
	if trace == nil {
		return
	}

	trace.Add(
		map[string]any{"messages": completionMessages, "options": completionOptions},
		types.TraceNodeOption{
			Label:       fmt.Sprintf(i18n.Tr(ast.ID, ctx.Locale, "llm.openai.stream.label"), connID), // "LLM %s"
			Type:        "llm",
			Icon:        "psychology",
			Description: fmt.Sprintf(i18n.Tr(ast.ID, ctx.Locale, "llm.openai.stream.description"), connID), // "LLM %s is processing the request"
		},
	)
}

// traceLLMComplete marks the LLM request as complete in the trace
func (ast *Assistant) traceLLMComplete(ctx *context.Context, completionResponse *context.CompletionResponse) {
	trace, _ := ctx.Trace()
	if trace == nil {
		return
	}

	trace.Complete(completionResponse)
}

// traceAgentOutput sets the output of the agent trace node
func (ast *Assistant) traceAgentOutput(agentNode types.Node, createResponse *context.HookCreateResponse, doneResponse *context.ResponseHookDone, completionResponse *context.CompletionResponse) {
	if agentNode == nil {
		return
	}

	agentNode.SetOutput(context.Response{
		Create:     createResponse,
		Done:       doneResponse,
		Completion: completionResponse,
	})
}

// traceAgentFail marks the agent trace node as failed
func (ast *Assistant) traceAgentFail(agentNode types.Node, err error) {
	if agentNode == nil {
		return
	}

	agentNode.Fail(err)
}
