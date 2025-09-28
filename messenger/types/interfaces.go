package types

import "context"

// MessageHandler defines a callback function for handling received messages
type MessageHandler func(ctx context.Context, message *Message) error

// Provider defines the interface for message providers
type Provider interface {
	// Send sends a message using the provider
	Send(ctx context.Context, message *Message) error

	// SendBatch sends multiple messages in batch
	SendBatch(ctx context.Context, messages []*Message) error

	// TriggerWebhook processes webhook requests and converts to Message
	TriggerWebhook(c interface{}) (*Message, error)

	// GetType returns the provider type (smtp, twilio, mailgun, etc.)
	GetType() string

	// GetName returns the provider name/identifier
	GetName() string

	// GetPublicInfo returns public information about the provider (name, description, type)
	GetPublicInfo() ProviderPublicInfo

	// Validate validates the provider configuration
	Validate() error

	// Close closes the provider connection if needed
	Close() error
}

// Messenger defines the main messenger interface
type Messenger interface {
	// Send sends a message using the specified channel or default provider
	Send(ctx context.Context, channel string, message *Message) error

	// SendWithProvider sends a message using a specific provider
	SendWithProvider(ctx context.Context, providerName string, message *Message) error

	// SendBatch sends multiple messages in batch
	SendBatch(ctx context.Context, channel string, messages []*Message) error

	// GetProvider returns a provider by name
	GetProvider(name string) (Provider, error)

	// GetProviders returns all providers for a channel type
	GetProviders(channelType string) []Provider

	// GetAllProviders returns all providers
	GetAllProviders() []Provider

	// GetChannels returns all available channels
	GetChannels() []string

	// OnReceive registers a message handler for received messages
	// Multiple handlers can be registered and will be called in order
	OnReceive(handler MessageHandler) error

	// RemoveReceiveHandler removes a previously registered message handler
	RemoveReceiveHandler(handler MessageHandler) error

	// TriggerWebhook processes incoming webhook data and triggers OnReceive handlers
	// This is used by OPENAPI endpoints to handle incoming messages
	TriggerWebhook(providerName string, c interface{}) error

	// Close closes all provider connections
	Close() error
}
