package adapters

import (
	"github.com/yaoapp/yao/agent/context"
)

// ToolCallAdapter handles tool calling capability
// If model doesn't support native tool calls, it injects tool instructions into prompts
type ToolCallAdapter struct {
	*BaseAdapter
	nativeSupport bool
}

// NewToolCallAdapter creates a new tool call adapter
func NewToolCallAdapter(nativeSupport bool) *ToolCallAdapter {
	return &ToolCallAdapter{
		BaseAdapter:   NewBaseAdapter("ToolCallAdapter"),
		nativeSupport: nativeSupport,
	}
}

// PreprocessMessages injects tool calling instructions if not natively supported
func (a *ToolCallAdapter) PreprocessMessages(messages []context.Message) ([]context.Message, error) {
	if a.nativeSupport {
		// Native support, no preprocessing needed
		return messages, nil
	}

	// TODO: Inject tool calling instructions into system prompt
	// - Generate tool description prompt
	// - Add to system message or create new system message
	// - Include tool schemas and usage instructions
	return messages, nil
}

// PreprocessOptions removes tool-related options if not natively supported
func (a *ToolCallAdapter) PreprocessOptions(options *context.CompletionOptions) (*context.CompletionOptions, error) {
	if a.nativeSupport {
		// Native support, keep options as-is
		return options, nil
	}

	if options == nil {
		return options, nil
	}

	// Remove tool parameters for non-native models
	newOptions := *options
	newOptions.Tools = nil
	newOptions.ToolChoice = nil
	return &newOptions, nil
}

// PostprocessResponse extracts tool calls from text if not natively supported
func (a *ToolCallAdapter) PostprocessResponse(response *context.CompletionResponse) (*context.CompletionResponse, error) {
	if a.nativeSupport {
		// Native support, response already has structured tool calls
		return response, nil
	}

	// TODO: Extract tool calls from text response
	// - Look for JSON blocks or specific patterns
	// - Parse tool name and arguments
	// - Add to response.ToolCalls
	return response, nil
}
