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

	// SendT sends a message using a template with specified type
	// templateType specifies which template variant to use (mail, sms, whatsapp)
	SendT(ctx context.Context, templateID string, templateType TemplateType, data TemplateData) error

	// SendTBatch sends multiple messages using the same template with different data
	// templateType specifies which template variant to use (mail, sms, whatsapp)
	SendTBatch(ctx context.Context, templateID string, templateType TemplateType, dataList []TemplateData) error

	// SendTBatchMixed sends multiple messages using different templates with different data
	// Each TemplateRequest can optionally specify its own MessageType
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
	// messageType is optional - if not specified, the first available template type will be used
	SendT(ctx context.Context, channel string, templateID string, data TemplateData, messageType ...MessageType) error

	// SendTWithProvider sends a message using a template and specific provider
	// messageType is optional - if not specified, the first available template type will be used
	SendTWithProvider(ctx context.Context, providerName string, templateID string, data TemplateData, messageType ...MessageType) error

	// SendTBatch sends multiple messages using the same template with different data
	// messageType is optional - if not specified, the first available template type will be used
	SendTBatch(ctx context.Context, channel string, templateID string, dataList []TemplateData, messageType ...MessageType) error

	// SendTBatchWithProvider sends multiple messages using the same template with different data and specific provider
	// messageType is optional - if not specified, the first available template type will be used
	SendTBatchWithProvider(ctx context.Context, providerName string, templateID string, dataList []TemplateData, messageType ...MessageType) error

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
