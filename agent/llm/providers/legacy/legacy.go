package legacy

import (
	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/llm/providers/base"
)

// Provider legacy LLM provider (no native tool calling support)
// Implements tool calling via prompt engineering
type Provider struct {
	*base.Provider
}

// New create a new legacy provider
func New(conn connector.Connector, capabilities *context.ModelCapabilities) *Provider {
	return &Provider{
		Provider: base.NewProvider(conn, capabilities),
	}
}

// Stream stream completion from legacy model
func (p *Provider) Stream(ctx *context.Context, messages []context.Message, options *context.CompletionOptions, handler context.StreamFunc) (*context.CompletionResponse, error) {
	// TODO: Implement legacy model streaming
	// - Preprocess messages (remove tool-specific fields, vision, audio)
	//   - Remove vision content (convert to text description)
	//   - Remove audio content (convert to text transcription)
	//   - Remove tool messages
	// - Add tool calling instructions to system prompt if tools provided
	// - Build request body without native tool parameters
	// - Make streaming HTTP request
	// - Parse response and detect tool calls from text
	// - Extract tool calls using regex/JSON parsing
	// - Call handler for each chunk
	return nil, nil
}

// Post post completion request to legacy model
func (p *Provider) Post(ctx *context.Context, messages []context.Message, options *context.CompletionOptions) (*context.CompletionResponse, error) {
	// TODO: Implement legacy model non-streaming completion
	// - Preprocess messages
	// - Add tool instructions to prompt
	// - Make HTTP POST request
	// - Parse response and extract tool calls from text
	return nil, nil
}

// InjectToolInstructions inject tool calling instructions into system prompt
func (p *Provider) InjectToolInstructions(messages []context.Message, tools []map[string]interface{}) []context.Message {
	// TODO: Implement tool instruction injection
	// - Generate tool description prompt
	// - Add to system message or create new system message
	// - Include tool schemas and usage instructions
	return messages
}

// ExtractToolCallsFromText extract tool calls from model's text response
func (p *Provider) ExtractToolCallsFromText(text string) []context.ToolCall {
	// TODO: Implement tool call extraction
	// - Look for JSON blocks or specific patterns
	// - Parse tool name and arguments
	// - Return structured tool calls
	return nil
}
