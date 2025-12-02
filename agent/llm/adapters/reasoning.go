package adapters

import (
	"github.com/yaoapp/gou/connector/openai"
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
// - Manages temperature parameter constraints (reasoning models typically require temperature=1)
// - Extracts reasoning_tokens from usage
// - Parses visible reasoning content (DeepSeek R1)
type ReasoningAdapter struct {
	*BaseAdapter
	format              ReasoningFormat
	supportsEffort      bool // Whether the model supports reasoning_effort parameter
	supportsTemperature bool // Whether the model supports temperature adjustment
}

// NewReasoningAdapter creates a new reasoning adapter
// If cap.TemperatureAdjustable is provided, it overrides the default behavior
func NewReasoningAdapter(format ReasoningFormat, cap *openai.Capabilities) *ReasoningAdapter {
	supportsEffort := false
	supportsTemperature := true

	// Set defaults based on reasoning format
	switch format {
	case ReasoningFormatOpenAI, ReasoningFormatGPT5:
		// OpenAI o1 and GPT-5: support reasoning_effort, but NOT temperature adjustment
		supportsEffort = true
		supportsTemperature = false
	case ReasoningFormatDeepSeek:
		// DeepSeek R1: no reasoning_effort, no temperature adjustment
		supportsEffort = false
		supportsTemperature = false
	case ReasoningFormatNone:
		// Non-reasoning models: no reasoning_effort, but support temperature
		supportsEffort = false
		supportsTemperature = true
	}

	// Override with explicit capability if provided
	if cap != nil {
		supportsTemperature = cap.TemperatureAdjustable
	}

	return &ReasoningAdapter{
		BaseAdapter:         NewBaseAdapter("ReasoningAdapter"),
		format:              format,
		supportsEffort:      supportsEffort,
		supportsTemperature: supportsTemperature,
	}
}

// PreprocessOptions handles reasoning_effort and temperature parameters
func (a *ReasoningAdapter) PreprocessOptions(options *context.CompletionOptions) (*context.CompletionOptions, error) {
	if options == nil {
		return options, nil
	}

	newOptions := *options
	modified := false

	// 1. Handle reasoning_effort parameter
	if !a.supportsEffort && newOptions.ReasoningEffort != nil {
		// Model doesn't support reasoning_effort, remove the parameter
		newOptions.ReasoningEffort = nil
		modified = true
	}

	// 2. Handle temperature parameter
	if !a.supportsTemperature && newOptions.Temperature != nil {
		currentTemp := *newOptions.Temperature
		if currentTemp != 1.0 {
			// Model doesn't support temperature adjustment, reset to default (1.0)
			defaultTemp := 1.0
			newOptions.Temperature = &defaultTemp
			modified = true
		}
	}

	if modified {
		return &newOptions, nil
	}

	// No modifications needed
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
