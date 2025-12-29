package assistant

import (
	"fmt"

	"github.com/yaoapp/yao/agent/content"
	contentTypes "github.com/yaoapp/yao/agent/content/types"
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

	// Build parse options
	parseOptions := &contentTypes.Options{
		Capabilities:      capabilities,
		CompletionOptions: options,
		Connector:         connector,
		StreamOptions:     options.StreamOptions,
	}

	contentMessages, referenceContext, err := content.ParseUserInput(ctx, messages, parseOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to parse content: %w", err)
	}

	// Inject reference context into messages
	if referenceContext != nil {
		contentMessages = ast.injectSearchContext(contentMessages, referenceContext)
	}

	return contentMessages, nil
}
