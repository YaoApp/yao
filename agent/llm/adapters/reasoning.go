package adapters

import (
	"github.com/yaoapp/yao/agent/context"
)

// ReasoningFormat represents the reasoning content format
type ReasoningFormat string

const (
	ReasoningFormatNone     ReasoningFormat = "none"        // No reasoning support
	ReasoningFormatOpenAI   ReasoningFormat = "openai-o1"   // OpenAI o1 format
	ReasoningFormatDeepSeek ReasoningFormat = "deepseek-r1" // DeepSeek R1 format
	ReasoningFormatGPTThink ReasoningFormat = "gpt-think"   // Future GPT with thinking
)

// ReasoningAdapter handles reasoning content capability
// Parses reasoning_content from different model formats
type ReasoningAdapter struct {
	*BaseAdapter
	format ReasoningFormat
}

// NewReasoningAdapter creates a new reasoning adapter
func NewReasoningAdapter(format ReasoningFormat) *ReasoningAdapter {
	return &ReasoningAdapter{
		BaseAdapter: NewBaseAdapter("ReasoningAdapter"),
		format:      format,
	}
}

// ProcessStreamChunk processes streaming chunks with reasoning content
func (a *ReasoningAdapter) ProcessStreamChunk(chunkType context.StreamChunkType, data []byte) (context.StreamChunkType, []byte, error) {
	if a.format == ReasoningFormatNone {
		// No reasoning support, pass through
		return chunkType, data, nil
	}

	// TODO: Parse reasoning_content based on format
	// - OpenAI o1: reasoning_content field in delta
	// - DeepSeek R1: may have different format
	// - Extract and emit as ChunkThinking

	return chunkType, data, nil
}

// PostprocessResponse extracts reasoning content from the final response
func (a *ReasoningAdapter) PostprocessResponse(response *context.CompletionResponse) (*context.CompletionResponse, error) {
	if a.format == ReasoningFormatNone {
		// No reasoning support
		return response, nil
	}

	// TODO: Extract reasoning content from response
	// - Set response.ReasoningContent if present
	// - Separate thinking from final answer

	return response, nil
}
