package reasoning

import (
	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/llm/providers/base"
)

// Provider reasoning model provider (o1, DeepSeek R1, etc.)
// Handles special response format with reasoning_content
// Note: Some reasoning models (e.g. DeepSeek R1) don't support native tool calls
type Provider struct {
	*base.Provider
	supportsNativeTools bool // Whether this reasoning model supports native tool calling
}

// New create a new reasoning provider
func New(conn connector.Connector, capabilities *context.ModelCapabilities) *Provider {
	// Check if this reasoning model supports native tool calls
	supportsTools := false
	if capabilities != nil && capabilities.ToolCalls != nil && *capabilities.ToolCalls {
		supportsTools = true
	}

	return &Provider{
		Provider:            base.NewProvider(conn, capabilities),
		supportsNativeTools: supportsTools,
	}
}

// Stream stream completion from reasoning model
func (p *Provider) Stream(ctx *context.Context, messages []context.Message, options *context.CompletionOptions, handler context.StreamFunc) (*context.CompletionResponse, error) {
	// TODO: Implement reasoning model streaming
	// - Preprocess messages (reasoning models have restrictions)

	// Handle tool calls based on model support
	if !p.supportsNativeTools && options != nil && len(options.Tools) > 0 {
		// Model doesn't support native tool calls (e.g. DeepSeek R1)
		// Inject tool instructions into messages
		messages = p.injectToolInstructions(messages, options.Tools)
		// Remove tools from options to avoid API error
		options = p.removeToolsFromOptions(options)
	}

	// - Build request body (special parameters for reasoning)
	// - Make streaming HTTP request
	// - Parse SSE chunks with reasoning_content
	// - Handle both thinking and answer phases
	// - Call handler for each chunk

	// If tools were injected, extract tool calls from text response
	// if !p.supportsNativeTools && hasTools {
	//     toolCalls = p.extractToolCallsFromText(response.Content)
	// }

	// - Aggregate final response with reasoning content
	return nil, nil
}

// Post post completion request to reasoning model
func (p *Provider) Post(ctx *context.Context, messages []context.Message, options *context.CompletionOptions) (*context.CompletionResponse, error) {
	// TODO: Implement reasoning model non-streaming completion
	// - Preprocess messages
	// - Build request body
	// - Make HTTP POST request
	// - Parse response with reasoning_content field
	// - Separate thinking from final answer
	return nil, nil
}

// ParseReasoningResponse parse response with reasoning content
// Handles both OpenAI o1 format and DeepSeek R1 format
func (p *Provider) ParseReasoningResponse(data []byte) (*context.CompletionResponse, error) {
	// TODO: Implement reasoning response parsing
	// - Detect format (OpenAI vs DeepSeek)
	// - Extract reasoning_content
	// - Extract final content
	// - Set ContentTypes correctly (text + reasoning)
	return nil, nil
}

// injectToolInstructions inject tool calling instructions into messages
// Used for reasoning models that don't support native tool calls (e.g. DeepSeek R1)
func (p *Provider) injectToolInstructions(messages []context.Message, tools []map[string]interface{}) []context.Message {
	// TODO: Implement tool instruction injection for reasoning models
	// - Generate tool description prompt (optimized for reasoning models)
	// - Add to system message or create new system message
	// - Include tool schemas and usage instructions
	// - Format should encourage reasoning about tool usage
	return messages
}

// extractToolCallsFromText extract tool calls from reasoning model's text response
// Used when model doesn't support native tool calls
func (p *Provider) extractToolCallsFromText(text string) []context.ToolCallResult {
	// TODO: Implement tool call extraction from text
	// - Look for JSON blocks or specific patterns
	// - Parse tool name and arguments
	// - Return structured tool calls
	// - Handle reasoning model's specific output format
	return nil
}

// removeToolsFromOptions remove tool-related parameters from options
// Used when sending request to models that don't support native tool calls
func (p *Provider) removeToolsFromOptions(options *context.CompletionOptions) *context.CompletionOptions {
	// TODO: Create a copy of options without tool parameters
	// - Remove Tools field
	// - Remove ToolChoice field
	// - Keep other options intact
	newOptions := *options
	newOptions.Tools = nil
	newOptions.ToolChoice = nil
	return &newOptions
}
