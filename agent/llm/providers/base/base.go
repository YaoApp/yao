package base

import (
	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/yao/agent/context"
)

// Provider base provider implementation
// Provides common functionality for all LLM providers
type Provider struct {
	Connector    connector.Connector
	Capabilities *context.ModelCapabilities
}

// NewProvider create a new base provider
func NewProvider(conn connector.Connector, capabilities *context.ModelCapabilities) *Provider {
	return &Provider{
		Connector:    conn,
		Capabilities: capabilities,
	}
}

// PreprocessMessages preprocess messages before sending to LLM
// Handles vision messages, audio messages, tool messages, etc.
func (p *Provider) PreprocessMessages(messages []context.Message) ([]context.Message, error) {
	// TODO: Implement message preprocessing
	// - Remove vision content if not supported
	// - Remove audio content if not supported
	// - Convert tool messages if needed
	// - Validate message format
	return messages, nil
}

// SupportsVision check if this provider supports vision
func (p *Provider) SupportsVision() bool {
	return p.Capabilities != nil && p.Capabilities.Vision != nil && *p.Capabilities.Vision
}

// SupportsAudio check if this provider supports audio
func (p *Provider) SupportsAudio() bool {
	return p.Capabilities != nil && p.Capabilities.Audio != nil && *p.Capabilities.Audio
}

// SupportsTools check if this provider supports tool calls
func (p *Provider) SupportsTools() bool {
	return p.Capabilities != nil && p.Capabilities.ToolCalls != nil && *p.Capabilities.ToolCalls
}

// BuildRequestBody build the request body for the LLM API
func (p *Provider) BuildRequestBody(messages []context.Message, options *context.CompletionOptions) (map[string]interface{}, error) {
	// TODO: Implement request body building
	// - Convert messages to API format
	// - Apply options (temperature, max_tokens, etc.)
	// - Add model-specific parameters
	return nil, nil
}

// ParseResponse parse the response from LLM API
func (p *Provider) ParseResponse(data []byte, isStreaming bool) (*context.CompletionResponse, error) {
	// TODO: Implement response parsing
	// - Parse JSON response
	// - Extract content, tool calls, reasoning, etc.
	// - Handle streaming chunks
	return nil, nil
}
