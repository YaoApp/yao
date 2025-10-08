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

	// SendT sends a message using a template (optional - providers may return "not implemented" error)
	SendT(ctx context.Context, templateID string, data TemplateData) error

	// SendTBatch sends multiple messages using the same template with different data (optional - providers may return "not implemented" error)
	SendTBatch(ctx context.Context, templateID string, dataList []TemplateData) error

	// SendTBatchMixed sends multiple messages using different templates with different data (optional - providers may return "not implemented" error)
	SendTBatchMixed(ctx context.Context, templateRequests []TemplateRequest) error

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

	// SendT sends a message using a template
	SendT(ctx context.Context, channel string, templateID string, data TemplateData) error

	// SendTWithProvider sends a message using a template and specific provider
	SendTWithProvider(ctx context.Context, providerName string, templateID string, data TemplateData) error

	// SendTBatch sends multiple messages using the same template with different data
	SendTBatch(ctx context.Context, channel string, templateID string, dataList []TemplateData) error

	// SendTBatchWithProvider sends multiple messages using the same template with different data and specific provider
	SendTBatchWithProvider(ctx context.Context, providerName string, templateID string, dataList []TemplateData) error

	// SendTBatchMixed sends multiple messages using different templates with different data
	SendTBatchMixed(ctx context.Context, channel string, templateRequests []TemplateRequest) error

	// SendTBatchMixedWithProvider sends multiple messages using different templates with different data and specific provider
	SendTBatchMixedWithProvider(ctx context.Context, providerName string, templateRequests []TemplateRequest) error

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
