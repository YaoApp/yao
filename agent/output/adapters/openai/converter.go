package openai

import (
	"fmt"
	"time"

	"github.com/yaoapp/yao/agent/i18n"
	"github.com/yaoapp/yao/agent/output/message"
)

// ConverterRegistry manages message type converters
type ConverterRegistry struct {
	converters map[string]ConverterFunc
}

// NewConverterRegistry creates a new converter registry with default converters
func NewConverterRegistry() *ConverterRegistry {
	return &ConverterRegistry{
		converters: map[string]ConverterFunc{
			message.TypeText:         convertText,
			message.TypeThinking:     convertThinking,
			message.TypeLoading:      convertLoading,
			message.TypeToolCall:     convertToolCall,
			message.TypeError:        convertError,
			message.TypeImage:        convertImage,
			message.TypeAudio:        convertToLink,
			message.TypeVideo:        convertToLink,
			message.TypeAction:       convertAction,
			message.EventStreamStart: convertStreamStart, // Handle stream_start events
		},
	}
}

// Register registers a custom converter for a message type
func (r *ConverterRegistry) Register(msgType string, converter ConverterFunc) {
	r.converters[msgType] = converter
}

// GetConverter retrieves a converter for a given message type.
func (r *ConverterRegistry) GetConverter(msgType string) (ConverterFunc, bool) {
	converter, exists := r.converters[msgType]
	return converter, exists
}

// Convert converts a message using registered converters
// If no converter is found, converts to link format
func (r *ConverterRegistry) Convert(msg *message.Message, config *AdapterConfig) ([]interface{}, error) {
	// Check for registered converter
	if converter, exists := r.converters[msg.Type]; exists {
		return converter(msg, config)
	}

	// Fallback: convert to link format
	return convertToLink(msg, config)
}

// convertText converts text messages to OpenAI format
func convertText(msg *message.Message, config *AdapterConfig) ([]interface{}, error) {
	content := getStringProp(msg.Props, "content", "")

	return []interface{}{
		createOpenAIChunk(msg.MessageID, config.Model, map[string]interface{}{
			"content": content,
		}),
	}, nil
}

// convertThinking converts thinking messages to OpenAI reasoning format (o1 series)
func convertThinking(msg *message.Message, config *AdapterConfig) ([]interface{}, error) {
	content := getStringProp(msg.Props, "content", "")

	return []interface{}{
		createOpenAIChunk(msg.MessageID, config.Model, map[string]interface{}{
			"reasoning_content": content,
		}),
	}, nil
}

// convertLoading converts loading messages to OpenAI reasoning format
// This makes loading messages visible in standard OpenAI clients as thinking process
func convertLoading(msg *message.Message, config *AdapterConfig) ([]interface{}, error) {
	message := getStringProp(msg.Props, "message", "Processing...")

	// Convert loading to reasoning_content so it shows in OpenAI clients
	return []interface{}{
		createOpenAIChunk(msg.MessageID, config.Model, map[string]interface{}{
			"reasoning_content": message,
		}),
	}, nil
}

// convertToolCall converts tool_call messages to OpenAI format
func convertToolCall(msg *message.Message, config *AdapterConfig) ([]interface{}, error) {
	// Tool call format varies, pass through the props
	toolCalls := []map[string]interface{}{}

	// If props contain tool call data, use it
	if id, ok := msg.Props["id"].(string); ok {
		toolCall := map[string]interface{}{
			"id":   id,
			"type": "function",
		}

		if function, ok := msg.Props["function"].(map[string]interface{}); ok {
			toolCall["function"] = function
		}

		toolCalls = append(toolCalls, toolCall)
	}

	return []interface{}{
		createOpenAIChunk(msg.MessageID, config.Model, map[string]interface{}{
			"tool_calls": toolCalls,
		}),
	}, nil
}

// convertError converts error messages to OpenAI error format
func convertError(msg *message.Message, config *AdapterConfig) ([]interface{}, error) {
	message := getStringProp(msg.Props, "message", "An error occurred")
	code := getStringProp(msg.Props, "code", "server_error")

	return []interface{}{
		map[string]interface{}{
			"error": map[string]interface{}{
				"message": message,
				"type":    code,
				"code":    code,
			},
		},
	}, nil
}

// convertAction converts action messages to nothing (silent in OpenAI clients)
// Action messages are system-level commands (open panel, navigate, etc.)
// and should not be sent to standard chat clients
func convertAction(msg *message.Message, config *AdapterConfig) ([]interface{}, error) {
	// Return empty slice - no output for action messages in OpenAI format
	return []interface{}{}, nil
}

// convertStreamStart converts stream_start event to OpenAI format
// If model supports reasoning: converts to reasoning_content (thinking)
// Otherwise: converts to regular Markdown text with trace link
func convertStreamStart(msg *message.Message, config *AdapterConfig) ([]interface{}, error) {
	// Extract stream_start data from props
	data, ok := msg.Props["data"]
	if !ok {
		// No data, skip this message
		return []interface{}{}, nil
	}

	// Try to convert to EventStreamStartData
	var startData message.EventStreamStartData
	switch v := data.(type) {
	case message.EventStreamStartData:
		startData = v
	case map[string]interface{}:
		// If it's a map, try to extract traceID
		if traceID, ok := v["trace_id"].(string); ok {
			startData.TraceID = traceID
		}
		if requestID, ok := v["request_id"].(string); ok {
			startData.RequestID = requestID
		}
	default:
		// Unknown data type, skip
		return []interface{}{}, nil
	}

	// Check if we have a trace ID to link to
	if startData.TraceID == "" {
		// No trace ID, skip this message
		return []interface{}{}, nil
	}

	// Generate trace link
	traceLink := generateTraceLink(startData.TraceID, config)

	// Check if model supports reasoning
	supportsReasoning := false
	if config.Capabilities != nil && config.Capabilities.Reasoning != nil {
		supportsReasoning = *config.Capabilities.Reasoning
	}

	// Get localized text using i18n
	streamStartText := i18n.T(config.Locale, "output.stream_start")
	viewTraceText := i18n.T(config.Locale, "output.view_trace")

	// Convert based on reasoning support
	if supportsReasoning {
		// Convert to thinking format (reasoning_content)
		// Reasoning models display this as part of the thinking process
		content := fmt.Sprintf("🔍 %s - [%s](%s)\n", streamStartText, viewTraceText, traceLink)
		chunk := createOpenAIChunk(msg.MessageID, config.Model, map[string]interface{}{
			"reasoning_content": content,
		})
		return []interface{}{chunk}, nil
	}

	// Convert to regular Markdown text
	content := fmt.Sprintf("🚀 %s - [%s](%s)\n", streamStartText, viewTraceText, traceLink)
	chunk := createOpenAIChunk(msg.MessageID, config.Model, map[string]interface{}{
		"content": content,
	})
	return []interface{}{chunk}, nil
}

// generateTraceLink generates a trace link URL
// Uses 'view' mode (clean page without sidebar) for better viewing experience in chat
func generateTraceLink(traceID string, config *AdapterConfig) string {
	baseURL := config.BaseURL
	if baseURL == "" {
		// If no base URL, return a relative link
		return fmt.Sprintf("/trace/%s/view", traceID)
	}
	return fmt.Sprintf("%s/trace/%s/view", baseURL, traceID)
}

// convertImage converts image messages to Markdown image format
// Uses ![alt](url) which displays inline in Markdown-supporting clients
func convertImage(msg *message.Message, config *AdapterConfig) ([]interface{}, error) {
	// Get URL
	url, ok := msg.Props["url"].(string)
	if !ok || url == "" {
		return nil, fmt.Errorf("image message missing url")
	}

	// Transform URL if transformer is provided
	if config.LinkTransformer != nil {
		transformedURL, err := config.LinkTransformer(url, msg.Type, msg.MessageID)
		if err != nil {
			return nil, err
		}
		url = transformedURL
	}

	// Get alt text (default to "Image")
	alt := getStringProp(msg.Props, "alt", "Image")

	// Format as Markdown image: ![alt](url)
	template := getLinkTemplate(msg.Type, config)
	text := fmt.Sprintf(template, alt, url)

	return []interface{}{
		createOpenAIChunk(msg.MessageID, config.Model, map[string]interface{}{
			"content": text,
		}),
	}, nil
}

// convertToLink converts any message type to a Markdown link format
func convertToLink(msg *message.Message, config *AdapterConfig) ([]interface{}, error) {
	// Generate link
	link, err := generateViewLink(msg, config)
	if err != nil {
		return nil, err
	}

	// Get template
	template := getLinkTemplate(msg.Type, config)

	// Format text
	var text string
	if msg.Type == "button" {
		// Button is special: needs button text
		buttonText := getStringProp(msg.Props, "text", "Button")
		text = fmt.Sprintf(template, buttonText, link)
	} else {
		text = fmt.Sprintf(template, link)
	}

	return []interface{}{
		createOpenAIChunk(msg.MessageID, config.Model, map[string]interface{}{
			"content": text,
		}),
	}, nil
}

// generateViewLink generates a view link for a message
func generateViewLink(msg *message.Message, config *AdapterConfig) (string, error) {
	// If Props contains a URL, use it
	if url, ok := msg.Props["url"].(string); ok {
		// Transform URL if transformer is provided
		if config.LinkTransformer != nil {
			return config.LinkTransformer(url, msg.Type, msg.MessageID)
		}
		return url, nil
	}

	// Generate view link: {baseURL}/agent/view/{type}/{id}
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "" // TODO: Get from environment or context
	}

	viewURL := fmt.Sprintf("%s/agent/view/%s/%s", baseURL, msg.Type, msg.MessageID)

	// Transform URL if transformer is provided
	if config.LinkTransformer != nil {
		return config.LinkTransformer(viewURL, msg.Type, msg.MessageID)
	}

	return viewURL, nil
}

// getLinkTemplate gets the link template for a message type
func getLinkTemplate(msgType string, config *AdapterConfig) string {
	if template, exists := config.LinkTemplates[msgType]; exists {
		return template
	}

	// Default fallback template
	return "📎 [View %s](" + msgType + ")"
}

// createOpenAIChunk creates an OpenAI chat completion chunk
func createOpenAIChunk(id string, model string, delta map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"id":      id,
		"object":  "chat.completion.chunk",
		"created": time.Now().Unix(),
		"model":   model,
		"choices": []map[string]interface{}{
			{
				"index":         0,
				"delta":         delta,
				"finish_reason": nil,
			},
		},
	}
}

// getStringProp safely gets a string property from props
func getStringProp(props map[string]interface{}, key string, defaultValue string) string {
	if val, ok := props[key].(string); ok {
		return val
	}
	return defaultValue
}
