package openai

import "github.com/yaoapp/yao/agent/output/message"

// Adapter is the OpenAI adapter that converts messages to OpenAI format
type Adapter struct {
	config   *AdapterConfig
	registry *ConverterRegistry
}

// NewAdapter creates a new OpenAI adapter with default configuration
func NewAdapter(options ...Option) *Adapter {
	adapter := &Adapter{
		config:   DefaultAdapterConfig(),
		registry: NewConverterRegistry(),
	}

	// Apply options
	for _, opt := range options {
		opt(adapter)
	}

	return adapter
}

// Option is a function that configures the adapter
type Option func(*Adapter)

// WithBaseURL sets the base URL for generating view links
func WithBaseURL(baseURL string) Option {
	return func(a *Adapter) {
		a.config.BaseURL = baseURL
	}
}

// WithLinkTemplate sets a custom link template for a message type
func WithLinkTemplate(msgType string, template string) Option {
	return func(a *Adapter) {
		a.config.LinkTemplates[msgType] = template
	}
}

// WithLinkTransformer sets the link transformer function
func WithLinkTransformer(transformer LinkTransformer) Option {
	return func(a *Adapter) {
		a.config.LinkTransformer = transformer
	}
}

// WithModel sets the model name for OpenAI responses
func WithModel(model string) Option {
	return func(a *Adapter) {
		a.config.Model = model
	}
}

// WithCapabilities sets the model capabilities
func WithCapabilities(capabilities *ModelCapabilities) Option {
	return func(a *Adapter) {
		a.config.Capabilities = capabilities
	}
}

// WithLocale sets the locale for internationalization
func WithLocale(locale string) Option {
	return func(a *Adapter) {
		a.config.Locale = locale
	}
}

// WithConverter registers a custom converter for a message type
func WithConverter(msgType string, converter ConverterFunc) Option {
	return func(a *Adapter) {
		a.registry.Register(msgType, converter)
	}
}

// Adapt converts a universal Message to OpenAI-compatible format
func (a *Adapter) Adapt(msg *message.Message) ([]interface{}, error) {
	// Handle event messages specially
	if msg.Type == message.TypeEvent {
		// Check if this is a stream_start event
		if event, ok := msg.Props["event"].(string); ok && event == message.EventStreamStart {
			// Use the stream_start converter
			if converter, exists := a.registry.GetConverter(message.EventStreamStart); exists {
				return converter(msg, a.config)
			}
		}
		// Other event messages are CUI-only, skip them
		return []interface{}{}, nil // Return empty array, nothing to send
	}

	// Get converter for this message type
	converter, exists := a.registry.GetConverter(msg.Type)
	if !exists {
		// Use default converter for unknown types (convert to link)
		converter = convertToLink
	}

	// Convert the message
	return converter(msg, a.config)
}

// SupportsType checks if the adapter explicitly supports a given message type
func (a *Adapter) SupportsType(msgType string) bool {
	_, exists := a.registry.GetConverter(msgType)
	return exists
}

// GetConfig returns the adapter configuration
func (a *Adapter) GetConfig() *AdapterConfig {
	return a.config
}

// GetRegistry returns the converter registry
func (a *Adapter) GetRegistry() *ConverterRegistry {
	return a.registry
}
