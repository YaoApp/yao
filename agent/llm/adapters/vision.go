package adapters

import (
	"github.com/yaoapp/yao/agent/context"
)

// VisionAdapter handles vision (image) capability
// If model doesn't support vision, it removes or converts image content
type VisionAdapter struct {
	*BaseAdapter
	nativeSupport bool
}

// NewVisionAdapter creates a new vision adapter
func NewVisionAdapter(nativeSupport bool) *VisionAdapter {
	return &VisionAdapter{
		BaseAdapter:   NewBaseAdapter("VisionAdapter"),
		nativeSupport: nativeSupport,
	}
}

// PreprocessMessages removes or converts image content if not supported
func (a *VisionAdapter) PreprocessMessages(messages []context.Message) ([]context.Message, error) {
	if a.nativeSupport {
		// Native support, no preprocessing needed
		return messages, nil
	}

	// Process messages to remove image content
	processed := make([]context.Message, 0, len(messages))
	for _, msg := range messages {
		processedMsg := msg

		// Handle multimodal content (array of ContentPart)
		if contentParts, ok := msg.Content.([]context.ContentPart); ok {
			filteredParts := make([]context.ContentPart, 0)

			for _, part := range contentParts {
				// Skip image content if not supported
				if part.Type == context.ContentImageURL {
					// TODO: Optionally convert to text description
					continue
				}
				filteredParts = append(filteredParts, part)
			}

			// If all parts were filtered out, add placeholder text
			if len(filteredParts) == 0 {
				processedMsg.Content = "[Image content not supported by this model]"
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
