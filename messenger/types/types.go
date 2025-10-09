package types

import (
	"time"

	"github.com/yaoapp/gou/types"
)

// MessageType defines the type of message
type MessageType string

// Message type constants for different messaging channels
const (
	// MessageTypeEmail represents email messaging
	MessageTypeEmail MessageType = "email"
	// MessageTypeSMS represents SMS messaging
	MessageTypeSMS MessageType = "sms"
	// MessageTypeWhatsApp represents WhatsApp messaging
	MessageTypeWhatsApp MessageType = "whatsapp"
)

// Message represents a message to be sent
type Message struct {
	Type        MessageType            `json:"type"`
	To          []string               `json:"to"`
	From        string                 `json:"from,omitempty"`
	Subject     string                 `json:"subject,omitempty"` // For email
	Body        string                 `json:"body"`
	HTML        string                 `json:"html,omitempty"`         // For email HTML content
	Attachments []Attachment           `json:"attachments,omitempty"`  // For email attachments
	Headers     map[string]string      `json:"headers,omitempty"`      // Custom headers
	Metadata    map[string]interface{} `json:"metadata,omitempty"`     // Additional metadata
	Priority    int                    `json:"priority,omitempty"`     // Message priority
	ScheduledAt *time.Time             `json:"scheduled_at,omitempty"` // For scheduled sending
}

// Attachment represents an email attachment
type Attachment struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Content     []byte `json:"content"`
	Inline      bool   `json:"inline,omitempty"` // For inline attachments
	CID         string `json:"cid,omitempty"`    // Content-ID for inline attachments
}

// ProviderConfig represents the configuration for a message provider
type ProviderConfig struct {
	types.MetaInfo
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Connector   string                 `json:"connector"`         // Provider type: mailer, twilio, mailgun
	Options     map[string]interface{} `json:"options,omitempty"` // Provider-specific options
	Enabled     bool                   `json:"enabled,omitempty"` // Whether the provider is enabled (default: true)
}

// Config represents the messenger configuration
type Config struct {
	Defaults  map[string]string  `json:"defaults,omitempty"`  // Default providers for each channel
	Channels  map[string]Channel `json:"channels,omitempty"`  // Channel-specific configurations
	Providers []ProviderConfig   `json:"providers,omitempty"` // Provider configurations
	Global    GlobalConfig       `json:"global,omitempty"`    // Global settings
}

// Channel represents a message channel configuration
type Channel struct {
	Provider    string                 `json:"provider,omitempty"`    // Default provider for this channel
	Description string                 `json:"description,omitempty"` // Channel description
	Fallbacks   []string               `json:"fallbacks,omitempty"`   // Fallback providers
	RateLimit   *RateLimit             `json:"rate_limit,omitempty"`  // Rate limiting settings
	Settings    map[string]interface{} `json:"settings,omitempty"`    // Channel-specific settings
	Templates   map[string]Template    `json:"templates,omitempty"`   // Message templates
	Types       map[string]*Channel    `json:"types,omitempty"`       // Type-specific configurations (email, sms, whatsapp)
}

// RateLimit represents rate limiting configuration
type RateLimit struct {
	Enabled    bool          `json:"enabled"`
	MaxPerHour int           `json:"max_per_hour,omitempty"`
	MaxPerDay  int           `json:"max_per_day,omitempty"`
	Window     time.Duration `json:"window,omitempty"`
}

// GlobalConfig represents global messenger settings
type GlobalConfig struct {
	RetryAttempts int           `json:"retry_attempts,omitempty"`
	RetryDelay    time.Duration `json:"retry_delay,omitempty"`
	Timeout       time.Duration `json:"timeout,omitempty"`
	LogLevel      string        `json:"log_level,omitempty"`
}

// SendOptions represents options for sending messages
type SendOptions struct {
	Provider    string                 `json:"provider,omitempty"`
	Template    string                 `json:"template,omitempty"`
	Variables   map[string]interface{} `json:"variables,omitempty"`
	Priority    int                    `json:"priority,omitempty"`
	ScheduledAt *time.Time             `json:"scheduled_at,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// SendResult represents the result of a send operation
type SendResult struct {
	Success   bool                   `json:"success"`
	MessageID string                 `json:"message_id,omitempty"`
	Provider  string                 `json:"provider"`
	Error     error                  `json:"error,omitempty"`
	Attempts  int                    `json:"attempts"`
	SentAt    time.Time              `json:"sent_at"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// ProviderPublicInfo defines the public information structure for providers
type ProviderPublicInfo struct {
	Name         string   `json:"name"`
	Type         string   `json:"type"`
	Description  string   `json:"description"`
	Capabilities []string `json:"capabilities"`
	Features     Features `json:"features"`
}

// Features defines the features supported by a provider
type Features struct {
	SupportsWebhooks   bool `json:"supports_webhooks"`
	SupportsReceiving  bool `json:"supports_receiving"`
	SupportsTracking   bool `json:"supports_tracking"`
	SupportsScheduling bool `json:"supports_scheduling"`
}

// TemplateRequest represents a request to send a message using a specific template
type TemplateRequest struct {
	TemplateID  string       `json:"template_id"`
	Data        TemplateData `json:"data"`
	MessageType *MessageType `json:"message_type,omitempty"` // Optional: if not specified, will use first available template type
}
