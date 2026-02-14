package caller

import (
	"context"

	agentContext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/openapi/oauth/types"
)

// NewHeadlessContext creates a headless agent context from a ProcessCallRequest.
// This is the Process equivalent of openapi.GetCompletionRequest — constructs
// a Context + Options without HTTP dependencies (no Writer, no Interrupt).
//
// Key behaviors:
//   - parent context controls timeout/cancellation (caller is responsible)
//   - skip.output = true (forced): no Writer available, must skip output
//   - skip.history = true (forced): Process calls don't save chat history
//   - authorized info is passed in (from authorized.ProcessAuthInfo by caller)
//   - chatID is auto-generated if not provided
func NewHeadlessContext(parent context.Context, authInfo *types.AuthorizedInfo, req *ProcessCallRequest) (*agentContext.Context, *agentContext.Options) {
	chatID := req.ChatID
	if chatID == "" {
		chatID = agentContext.GenChatID()
	}

	ctx := agentContext.New(parent, authInfo, chatID)
	ctx.AssistantID = req.AssistantID
	ctx.Referer = agentContext.RefererProcess
	ctx.Locale = req.Locale
	ctx.Route = req.Route
	ctx.Metadata = req.Metadata

	// Force skip for headless context — no Writer, no chat history
	skip := req.Skip
	if skip == nil {
		skip = &agentContext.Skip{}
	}
	skip.Output = true  // no Writer available
	skip.History = true // Process calls don't save chat history

	opts := &agentContext.Options{Skip: skip}
	if req.Model != "" {
		opts.Connector = req.Model
	}

	return ctx, opts
}
