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
	// Get connector and capabilities
	_, capabilities, err := ast.GetConnector(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get connector: %w", err)
	}

	// Get Uses configuration from options (already merged in BuildRequest)
	uses := options.Uses

	// Process content through Vision function
	processedMessages, err := content.Vision(ctx, capabilities, messages, uses)
	if err != nil {
		return nil, fmt.Errorf("failed to process content: %w", err)
	}

	return processedMessages, nil
}
