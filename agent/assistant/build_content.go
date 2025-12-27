package assistant

import (
	"fmt"

	"github.com/yaoapp/yao/agent/content"
	"github.com/yaoapp/yao/agent/context"
)

// BuildContent processes messages through Vision function to convert extended content types
// (file, data) to standard LLM-compatible types (text, image_url, input_audio)
//
// This should be called after BuildRequest and before executing LLM call
func (ast *Assistant) BuildContent(ctx *context.Context, messages []context.Message, options *context.CompletionOptions, opts *context.Options) ([]context.Message, error) {
	// Set AssistantID in context for file info tracking in Space
	// This ensures hooks can access file information using the correct namespace
	if ctx.AssistantID == "" {
		ctx.AssistantID = ast.ID
	}

	// Get connector and capabilities
	connector, capabilities, err := ast.GetConnector(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get connector: %w", err)
	}
	_ = connector // unused but needed for GetConnector call

	// Get Uses configuration from options (already merged in BuildRequest)
	uses := options.Uses

	// Get ForceUses configuration from options
	forceUses := options.ForceUses

	// Process content through Vision function
	processedMessages, err := content.Vision(ctx, capabilities, messages, uses, forceUses)
	if err != nil {
		return nil, fmt.Errorf("failed to process content: %w", err)
	}

	return processedMessages, nil
}
