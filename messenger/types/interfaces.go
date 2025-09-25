package types

// Provider defines the interface for message providers
type Provider interface {
	// Send sends a message using the provider
	Send(message *Message) error

	// SendBatch sends multiple messages in batch
	SendBatch(messages []*Message) error

	// GetType returns the provider type (smtp, twilio, mailgun, etc.)
	GetType() string

	// GetName returns the provider name/identifier
	GetName() string

	// Validate validates the provider configuration
	Validate() error

	// Close closes the provider connection if needed
	Close() error
}

// Messenger defines the main messenger interface
type Messenger interface {
	// Send sends a message using the specified channel or default provider
	Send(channel string, message *Message) error

	// SendWithProvider sends a message using a specific provider
	SendWithProvider(providerName string, message *Message) error

	// SendBatch sends multiple messages in batch
	SendBatch(channel string, messages []*Message) error

	// GetProvider returns a provider by name
	GetProvider(name string) (Provider, error)

	// GetProviders returns all providers for a channel type
	GetProviders(channelType string) []Provider

	// GetChannels returns all available channels
	GetChannels() []string

	// Close closes all provider connections
	Close() error
}
