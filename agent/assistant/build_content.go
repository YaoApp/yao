package assistant

import (
	"fmt"

	"github.com/yaoapp/yao/agent/content"
	"github.com/yaoapp/yao/agent/content/text"
	contentTypes "github.com/yaoapp/yao/agent/content/types"
	"github.com/yaoapp/yao/agent/context"
)

// BuildContent processes messages through Vision function to convert extended content types
// (file, data) to standard LLM-compatible types (text, image_url, input_audio)
//
// This should be called after BuildRequest and before executing LLM call
func (ast *Assistant) BuildContent(ctx *context.Context, messages []context.Message, options *context.CompletionOptions, opts *context.Options) ([]context.Message, error) {
	// Skip complex content parsing if requested (for internal calls like needsearch)
	// Still convert file attachments to raw text
	if opts != nil && opts.Skip != nil && opts.Skip.ContentParsing {
		return convertFilesToText(ctx, messages), nil
	}

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

// convertFilesToText converts file attachments in messages to raw text
// Used when SkipContentParsing is enabled - simple text extraction without vision/PDF processing
func convertFilesToText(ctx *context.Context, messages []context.Message) []context.Message {
	result := make([]context.Message, 0, len(messages))
	textHandler := text.New(nil)

	for _, msg := range messages {
		// Only process user messages
		if msg.Role != context.RoleUser {
			result = append(result, msg)
			continue
		}

		// Handle content parts
		parts, ok := msg.Content.([]context.ContentPart)
		if !ok {
			// Try []interface{} (from history/JSON)
			if iparts, ok := msg.Content.([]interface{}); ok {
				parts = convertInterfaceToParts(iparts)
			}
		}

		if len(parts) == 0 {
			result = append(result, msg)
			continue
		}

		// Convert file parts to text
		newParts := make([]context.ContentPart, 0, len(parts))
		for _, part := range parts {
			switch part.Type {
			case context.ContentFile:
				// Convert file to raw text
				if part.File != nil && part.File.URL != "" {
					textPart, _, err := textHandler.ParseRaw(ctx, part)
					if err == nil {
						newParts = append(newParts, textPart)
						continue
					}
				}
				newParts = append(newParts, part)

			case context.ContentImageURL:
				// Skip images - cannot convert to text without vision
				continue

			default:
				newParts = append(newParts, part)
			}
		}

		newMsg := msg
		newMsg.Content = newParts
		result = append(result, newMsg)
	}

	return result
}

// convertInterfaceToParts converts []interface{} to []ContentPart for file extraction
func convertInterfaceToParts(items []interface{}) []context.ContentPart {
	parts := make([]context.ContentPart, 0, len(items))
	for _, item := range items {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		typeStr, _ := m["type"].(string)
		part := context.ContentPart{
			Type: context.ContentPartType(typeStr),
		}

		switch typeStr {
		case "text":
			if t, ok := m["text"].(string); ok {
				part.Text = t
			}
		case "file":
			if fileData, ok := m["file"].(map[string]interface{}); ok {
				part.File = &context.FileAttachment{}
				if url, ok := fileData["url"].(string); ok {
					part.File.URL = url
				}
				if filename, ok := fileData["filename"].(string); ok {
					part.File.Filename = filename
				}
			}
		case "image_url":
			part.Type = context.ContentImageURL
		default:
			continue
		}

		parts = append(parts, part)
	}
	return parts
}
