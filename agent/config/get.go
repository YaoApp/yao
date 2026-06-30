package config

import (
	"github.com/yaoapp/yao/agent/context"
)

// Get reads AssistantID, ChatID and auth from the agent context,
// then calls Resolve to return the fully merged configuration.
func Get(ctx *context.Context) (*Resolved, error) {
	opts := ResolveOptions{
		AssistantID: ctx.AssistantID,
		ChatID:      ctx.ChatID,
	}
	if ctx.Authorized != nil {
		opts.UserID = ctx.Authorized.GetUserID()
		opts.TeamID = ctx.Authorized.GetTeamID()
	}
	return Resolve(opts)
}
