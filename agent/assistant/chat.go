package assistant

import (
	"fmt"

	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/trace/types"
)

// WithHistory merges the input messages with chat history and traces it
// This method can be overridden or extended to implement actual history loading
func (ast *Assistant) WithHistory(ctx *context.Context, input []context.Message, agentNode types.Node, options ...*context.Options) ([]context.Message, error) {

	// TODO: Implement actual history loading logic here
	// For now, just simulate a check and return the input messages as is

	// Simulate error check (this is where actual history loading would happen)
	// if some_condition {
	//     ast.traceAgentFail(agentNode, err)
	//     return nil, err
	// }

	fullMessages := input

	// Log the chat history
	ast.traceAgentHistory(ctx, agentNode, fullMessages)

	return fullMessages, nil
}

// initializeConversation  initialize the conversation
func (ast *Assistant) initializeConversation(ctx *context.Context, input []context.Message, options ...*context.Options) error {

	var opts *context.Options
	if len(options) > 0 && options[0] != nil {
		opts = options[0]
	} else {
		opts = &context.Options{}
	}

	// SKIP: History (for internal calls like title/prompt etc.)
	if opts.Skip != nil && opts.Skip.History {
		return nil
	}

	chatid := ctx.ChatID
	teamid := ctx.Authorized.TeamID
	userid := ctx.Authorized.UserID
	fmt.Printf(">>> initializeChat: chatid=%s, teamid=%s, userid=%s\n", chatid, teamid, userid)

	// Prepare kb collection (optional)
	err := ast.prepareKBCollection(ctx, input, opts)
	if err != nil {
		return err
	}

	// Save chat
	err = ast.saveChat(ctx, input, opts)
	if err != nil {
		return err
	}

	return nil
}

// Prepare kb collection (optional)
func (ast *Assistant) prepareKBCollection(ctx *context.Context, input []context.Message, opts *context.Options) error {
	_ = ctx
	_ = opts
	_ = input
	return nil
}

func (ast *Assistant) saveChat(ctx *context.Context, input []context.Message, opts *context.Options) error {
	_ = ctx
	_ = input
	_ = opts
	return nil
}
