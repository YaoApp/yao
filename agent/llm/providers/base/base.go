package base

import (
	"fmt"

	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/gou/connector/openai"
	"github.com/yaoapp/yao/agent/context"
)

// Provider base provider implementation
// Provides common functionality for all LLM providers
type Provider struct {
	Connector    connector.Connector
	Capabilities *openai.Capabilities
}

// NewProvider create a new base provider
func NewProvider(conn connector.Connector, capabilities *openai.Capabilities) *Provider {
	return &Provider{
		Connector:    conn,
		Capabilities: capabilities,
	}
}

// PreprocessMessages preprocess messages before sending to LLM
// Handles vision messages, audio messages, tool messages, etc.
// Filters out unsupported content types based on model capabilities
func (p *Provider) PreprocessMessages(messages []context.Message) ([]context.Message, error) {
	processed := make([]context.Message, 0, len(messages))

	for _, msg := range messages {
		processedMsg := msg

		// Handle multimodal content (array of ContentPart)
		if contentParts, ok := msg.Content.([]context.ContentPart); ok {
			filteredParts := make([]context.ContentPart, 0, len(contentParts))

			for _, part := range contentParts {
				// Filter vision content if not supported
				if part.Type == context.ContentImageURL {
					if !p.SupportsVision() {
						// Skip image content if vision not supported
						continue
					}
				}

				// Filter audio content if not supported
				if part.Type == context.ContentInputAudio {
					if !p.SupportsAudio() {
						// Skip audio content if audio not supported
						continue
					}
				}

				filteredParts = append(filteredParts, part)
			}

			// If all parts were filtered out, convert to text message
			if len(filteredParts) == 0 {
				processedMsg.Content = "[Content not supported by this model]"
			} else {
				processedMsg.Content = filteredParts
			}
		}

		processed = append(processed, processedMsg)
	}

	return processed, nil
}

// SupportsVision check if this provider supports vision
func (p *Provider) SupportsVision() bool {
	if p.Capabilities == nil {
		return false
	}
	supported, _ := context.GetVisionSupport(p.Capabilities)
	return supported
}

// SupportsAudio check if this provider supports audio
func (p *Provider) SupportsAudio() bool {
	return p.Capabilities != nil && p.Capabilities.Audio
}

// SupportsTools check if this provider supports tool calls
func (p *Provider) SupportsTools() bool {
	return p.Capabilities != nil && p.Capabilities.ToolCalls
}

// SupportsStreaming check if this provider supports streaming
func (p *Provider) SupportsStreaming() bool {
	return p.Capabilities != nil && p.Capabilities.Streaming
}

// SupportsJSON check if this provider supports JSON mode
func (p *Provider) SupportsJSON() bool {
	return p.Capabilities != nil && p.Capabilities.JSON
}

// SupportsReasoning check if this provider supports reasoning mode
func (p *Provider) SupportsReasoning() bool {
	return p.Capabilities != nil && p.Capabilities.Reasoning
}

// GetConnectorSetting gets a setting value from the connector
func (p *Provider) GetConnectorSetting(key string) (interface{}, error) {
	if p.Connector == nil {
		return nil, fmt.Errorf("connector is nil")
	}

	settings := p.Connector.Setting()
	if settings == nil {
		return nil, fmt.Errorf("connector settings are nil")
	}

	value, exists := settings[key]
	if !exists {
		return nil, fmt.Errorf("setting '%s' not found", key)
	}

	return value, nil
}

// GetConnectorStringSetting gets a string setting value from the connector
func (p *Provider) GetConnectorStringSetting(key string) (string, error) {
	value, err := p.GetConnectorSetting(key)
	if err != nil {
		return "", err
	}

	strValue, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("setting '%s' is not a string", key)
	}

	return strValue, nil
}

// GetModel gets the model name from connector settings
func (p *Provider) GetModel() (string, error) {
	return p.GetConnectorStringSetting("model")
}

// GetAPIKey gets the API key from connector settings
func (p *Provider) GetAPIKey() (string, error) {
	return p.GetConnectorStringSetting("key")
}

// GetHost gets the host URL from connector settings
func (p *Provider) GetHost() (string, error) {
	return p.GetConnectorStringSetting("host")
}
