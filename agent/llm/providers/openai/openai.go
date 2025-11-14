package openai

import (
	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/llm/providers/base"
)

// Provider OpenAI-compatible provider
// Supports: vision, tool calls, streaming, JSON mode
type Provider struct {
	*base.Provider
}

// New create a new OpenAI provider
func New(conn connector.Connector, capabilities *context.ModelCapabilities) *Provider {
	return &Provider{
		Provider: base.NewProvider(conn, capabilities),
	}
}

// Stream stream completion from OpenAI API
func (p *Provider) Stream(ctx *context.Context, messages []context.Message, options *context.CompletionOptions, handler context.StreamFunc) (*context.CompletionResponse, error) {
	// TODO: Implement OpenAI streaming
	// - Preprocess messages (vision, audio, tools)
	//   - Remove vision content if not supported
	//   - Remove audio content if not supported
	//   - Convert to text where needed
	// - Build request body
	// - Make streaming HTTP request
	// - Parse SSE chunks
	// - Call handler for each chunk
	// - Aggregate final response
	return nil, nil
}

// Post post completion request to OpenAI API
func (p *Provider) Post(ctx *context.Context, messages []context.Message, options *context.CompletionOptions) (*context.CompletionResponse, error) {
	// TODO: Implement OpenAI non-streaming completion
	// - Preprocess messages
	// - Build request body
	// - Make HTTP POST request
	// - Parse response
	return nil, nil
}

// SupportsAudio check if this provider supports audio
func (p *Provider) SupportsAudio() bool {
	return p.Capabilities != nil && p.Capabilities.Audio != nil && *p.Capabilities.Audio
}
