package assistant

import (
	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/trace/types"
)

// WithHistory merges the input messages with chat history and traces it
// This method can be overridden or extended to implement actual history loading
func (ast *Assistant) WithHistory(
	ctx *context.Context,
	inputMessages []context.Message,
	agentNode types.Node,
) ([]context.Message, error) {

	// TODO: Implement actual history loading logic here
	// For now, just simulate a check and return the input messages as is

	// Simulate error check (this is where actual history loading would happen)
	// if some_condition {
	//     ast.traceAgentFail(agentNode, err)
	//     return nil, err
	// }

	fullMessages := inputMessages

	// Log the chat history
	ast.traceAgentHistory(ctx, agentNode, fullMessages)

	return fullMessages, nil
}
