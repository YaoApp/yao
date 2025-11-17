package adapters

import (
	"github.com/yaoapp/yao/agent/context"
)

// ReasoningFormat represents the reasoning content format
type ReasoningFormat string

const (
	ReasoningFormatNone     ReasoningFormat = "none"        // No reasoning support
	ReasoningFormatOpenAI   ReasoningFormat = "openai-o1"   // OpenAI o1 format (hidden reasoning)
	ReasoningFormatGPT5     ReasoningFormat = "gpt-5"       // GPT-5 format (hidden reasoning)
	ReasoningFormatDeepSeek ReasoningFormat = "deepseek-r1" // DeepSeek R1 format (visible reasoning)
)

// ReasoningAdapter handles reasoning content capability
// - Manages reasoning_effort parameter (o1, GPT-5)
// - Extracts reasoning_tokens from usage
// - Parses visible reasoning content (DeepSeek R1)
type ReasoningAdapter struct {
	*BaseAdapter
	format         ReasoningFormat
	supportsEffort bool // Whether the model supports reasoning_effort parameter
}

// NewReasoningAdapter creates a new reasoning adapter
func NewReasoningAdapter(format ReasoningFormat) *ReasoningAdapter {
	supportsEffort := false

	// Only OpenAI o1 and GPT-5 support reasoning_effort parameter
	if format == ReasoningFormatOpenAI || format == ReasoningFormatGPT5 {
		supportsEffort = true
	}

	return &ReasoningAdapter{
		BaseAdapter:    NewBaseAdapter("ReasoningAdapter"),
		format:         format,
		supportsEffort: supportsEffort,
	}
}

// PreprocessOptions handles reasoning_effort parameter
func (a *ReasoningAdapter) PreprocessOptions(options *context.CompletionOptions) (*context.CompletionOptions, error) {
	if options == nil {
		return options, nil
	}

	// If model doesn't support reasoning_effort, remove it
	if !a.supportsEffort && options.ReasoningEffort != nil {
		// Model doesn't support reasoning_effort, remove the parameter
		newOptions := *options
		newOptions.ReasoningEffort = nil
		return &newOptions, nil
	}

	// If model supports reasoning_effort, keep it as-is (user can set "low", "medium", or "high")
	return options, nil
}

// ProcessStreamChunk processes streaming chunks with reasoning content
func (a *ReasoningAdapter) ProcessStreamChunk(chunkType context.StreamChunkType, data []byte) (context.StreamChunkType, []byte, error) {
	if a.format == ReasoningFormatNone {
		// No reasoning support, pass through
		return chunkType, data, nil
	}

	// TODO: Parse reasoning_content based on format
	// - OpenAI o1: No visible reasoning in stream (reasoning happens internally)
	// - GPT-5: No visible reasoning in stream (reasoning happens internally)
	// - DeepSeek R1: May have <think>...</think> tags or reasoning_content field

	return chunkType, data, nil
}

// PostprocessResponse extracts reasoning content and tokens from the final response
func (a *ReasoningAdapter) PostprocessResponse(response *context.CompletionResponse) (*context.CompletionResponse, error) {
	if a.format == ReasoningFormatNone {
		// No reasoning support
		return response, nil
	}

	// Reasoning tokens are already extracted in Usage.CompletionTokensDetails.ReasoningTokens
	// by the OpenAI response parser, no additional processing needed for o1/GPT-5

	// TODO: For DeepSeek R1, extract visible reasoning content
	// - Parse <think>...</think> tags from content
	// - Set response.ReasoningContent
	// - Remove <think> tags from response.Content (keep only final answer)

	return response, nil
}
