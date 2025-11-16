package adapters

import (
	"github.com/yaoapp/yao/agent/context"
)

// AudioAdapter handles audio capability
// If model doesn't support audio, it removes or converts audio content
type AudioAdapter struct {
	*BaseAdapter
	nativeSupport bool
}

// NewAudioAdapter creates a new audio adapter
func NewAudioAdapter(nativeSupport bool) *AudioAdapter {
	return &AudioAdapter{
		BaseAdapter:   NewBaseAdapter("AudioAdapter"),
		nativeSupport: nativeSupport,
	}
}

// PreprocessMessages removes or converts audio content if not supported
func (a *AudioAdapter) PreprocessMessages(messages []context.Message) ([]context.Message, error) {
	if a.nativeSupport {
		// Native support, no preprocessing needed
		return messages, nil
	}

	// Process messages to remove audio content
	processed := make([]context.Message, 0, len(messages))
	for _, msg := range messages {
		processedMsg := msg

		// Handle multimodal content (array of ContentPart)
		if contentParts, ok := msg.Content.([]context.ContentPart); ok {
			filteredParts := make([]context.ContentPart, 0)

			for _, part := range contentParts {
				// Skip audio content if not supported
				if part.Type == context.ContentInputAudio {
					// TODO: Optionally convert to transcription text if available
					continue
				}
				filteredParts = append(filteredParts, part)
			}

			// If all parts were filtered out, add placeholder text
			if len(filteredParts) == 0 {
				processedMsg.Content = "[Audio content not supported by this model]"
			} else if len(filteredParts) == 1 && filteredParts[0].Type == context.ContentText {
				// Single text part, convert to string
				processedMsg.Content = filteredParts[0].Text
			} else {
				processedMsg.Content = filteredParts
			}
		}

		processed = append(processed, processedMsg)
	}

	return processed, nil
}
