package content

import (
	"fmt"
	"strings"

	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/agent/content/docx"
	"github.com/yaoapp/yao/agent/content/image"
	"github.com/yaoapp/yao/agent/content/pdf"
	"github.com/yaoapp/yao/agent/content/pptx"
	"github.com/yaoapp/yao/agent/content/text"
	"github.com/yaoapp/yao/agent/content/types"
	agentContext "github.com/yaoapp/yao/agent/context"
	searchTypes "github.com/yaoapp/yao/agent/search/types"
)

// ParseUserInput  ParseUserInput
func ParseUserInput(ctx *agentContext.Context, messages []agentContext.Message, options *types.Options) ([]agentContext.Message, *searchTypes.ReferenceContext, error) {
	var referenceContext *searchTypes.ReferenceContext = nil
	var parsedMessages []agentContext.Message = make([]agentContext.Message, 0)
	for _, message := range messages {
		// Only process user messages (current or from history)
		if message.Role != agentContext.RoleUser {
			parsedMessages = append(parsedMessages, message)
			continue
		}

		// Parse user input message (Ignore errors)
		parsedMessage, refs, err := parseUserInputMessage(ctx, message, options)
		if err != nil {
			parsedMessages = append(parsedMessages, message)
			log.Error("Failed to parse user input message: %v, %v", message.Content, err)
			continue
		}
		parsedMessages = append(parsedMessages, parsedMessage)

		// Add reference to reference context
		if refs != nil {
			if referenceContext == nil {
				referenceContext = &searchTypes.ReferenceContext{}
			}
			referenceContext.References = append(referenceContext.References, refs...)
		}
	}

	return parsedMessages, referenceContext, nil
}

// parseUserInputMessage parse a user input message
func parseUserInputMessage(ctx *agentContext.Context, message agentContext.Message, options *types.Options) (agentContext.Message, []*searchTypes.Reference, error) {

	// Context content type
	switch content := message.Content.(type) {
	case string:
		return message, nil, nil

	case []agentContext.ContentPart:
		return parseContentParts(ctx, message, content, options)

	case []interface{}:
		// Handle content loaded from history/JSON ([]interface{} instead of []ContentPart)
		parts, ok := convertToContentParts(content)
		if !ok {
			return message, nil, nil
		}
		return parseContentParts(ctx, message, parts, options)
	}

	return message, nil, fmt.Errorf("unsupported content type: %T", message.Content)
}

// parseContentParts parses content parts and returns the parsed message
func parseContentParts(ctx *agentContext.Context, message agentContext.Message, content []agentContext.ContentPart, options *types.Options) (agentContext.Message, []*searchTypes.Reference, error) {
	allRefs := []*searchTypes.Reference{}
	parts := make([]agentContext.ContentPart, 0, len(content))
	for _, part := range content {
		parsedPart, refs, err := parseContentPart(ctx, part, options)
		if err != nil {
			parts = append(parts, part)
			continue
		}
		parts = append(parts, parsedPart)
		if refs != nil {
			allRefs = append(allRefs, refs...)
		}
	}

	parsedMessage := message
	parsedMessage.Content = parts
	return parsedMessage, allRefs, nil
}

// parseContentPart parse a content part
func parseContentPart(ctx *agentContext.Context, content agentContext.ContentPart, options *types.Options) (agentContext.ContentPart, []*searchTypes.Reference, error) {
	switch content.Type {
	case agentContext.ContentText:
		return content, nil, nil

	case agentContext.ContentImageURL:
		return image.New(options).Parse(ctx, content)

	case agentContext.ContentInputAudio:
		return content, nil, nil

	case agentContext.ContentFile:
		return parseFileContent(ctx, content, options)

	case agentContext.ContentData:
		return content, nil, nil

	default:
		return content, nil, fmt.Errorf("unsupported content part type: %s", content.Type)
	}
}

// parseFileContent parses file content based on file type
func parseFileContent(ctx *agentContext.Context, content agentContext.ContentPart, options *types.Options) (agentContext.ContentPart, []*searchTypes.Reference, error) {
	if content.File == nil || content.File.URL == "" {
		return content, nil, nil
	}

	// Determine file type from filename
	filename := strings.ToLower(content.File.Filename)

	// Check file type and route to appropriate handler
	switch {
	case strings.HasSuffix(filename, ".pdf"):
		return pdf.New(options).Parse(ctx, content)

	case strings.HasSuffix(filename, ".docx"):
		return docx.New(options).Parse(ctx, content)

	case strings.HasSuffix(filename, ".pptx"):
		return pptx.New(options).Parse(ctx, content)

	case text.IsSupportedExtension(filename):
		return text.New(options).Parse(ctx, content)
	}

	// For unsupported file types, try to read as text
	// This allows any file to be converted to text content
	return text.New(options).ParseRaw(ctx, content)
}

// convertToContentParts converts []interface{} to []ContentPart
// This is needed when content is loaded from JSON/history and is []interface{} instead of []ContentPart
func convertToContentParts(content []interface{}) ([]agentContext.ContentPart, bool) {
	parts := make([]agentContext.ContentPart, 0, len(content))
	for _, item := range content {
		// Each item should be a map
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		// Get type field
		typeStr, _ := m["type"].(string)
		if typeStr == "" {
			continue
		}

		part := agentContext.ContentPart{
			Type: agentContext.ContentPartType(typeStr),
		}

		switch typeStr {
		case "text":
			if text, ok := m["text"].(string); ok {
				part.Text = text
			}

		case "image_url":
			if imgData, ok := m["image_url"].(map[string]interface{}); ok {
				part.ImageURL = &agentContext.ImageURL{}
				if url, ok := imgData["url"].(string); ok {
					part.ImageURL.URL = url
				}
				if detail, ok := imgData["detail"].(string); ok {
					part.ImageURL.Detail = agentContext.ImageDetailLevel(detail)
				}
			}

		case "file":
			if fileData, ok := m["file"].(map[string]interface{}); ok {
				part.File = &agentContext.FileAttachment{}
				if url, ok := fileData["url"].(string); ok {
					part.File.URL = url
				}
				if filename, ok := fileData["filename"].(string); ok {
					part.File.Filename = filename
				}
			}

		case "input_audio":
			if audioData, ok := m["input_audio"].(map[string]interface{}); ok {
				part.InputAudio = &agentContext.InputAudio{}
				if data, ok := audioData["data"].(string); ok {
					part.InputAudio.Data = data
				}
				if format, ok := audioData["format"].(string); ok {
					part.InputAudio.Format = format
				}
			}

		case "data":
			if dataContent, ok := m["data"].(map[string]interface{}); ok {
				part.Data = &agentContext.DataContent{}
				if sources, ok := dataContent["sources"].([]interface{}); ok {
					part.Data.Sources = make([]agentContext.DataSource, 0, len(sources))
					for _, src := range sources {
						if srcMap, ok := src.(map[string]interface{}); ok {
							source := agentContext.DataSource{}
							if t, ok := srcMap["type"].(string); ok {
								source.Type = agentContext.DataSourceType(t)
							}
							if name, ok := srcMap["name"].(string); ok {
								source.Name = name
							}
							if id, ok := srcMap["id"].(string); ok {
								source.ID = id
							}
							if filters, ok := srcMap["filters"].(map[string]interface{}); ok {
								source.Filters = filters
							}
							if metadata, ok := srcMap["metadata"].(map[string]interface{}); ok {
								source.Metadata = metadata
							}
							part.Data.Sources = append(part.Data.Sources, source)
						}
					}
				}
			}
		}

		parts = append(parts, part)
	}

	if len(parts) == 0 {
		return nil, false
	}

	return parts, true
}
