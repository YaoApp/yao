package messenger

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/messenger/providers/mailgun"
	"github.com/yaoapp/yao/messenger/providers/smtp"
	"github.com/yaoapp/yao/messenger/providers/twilio"
	"github.com/yaoapp/yao/messenger/types"
	"github.com/yaoapp/yao/share"
)

// Instance is the global messenger instance
var Instance types.Messenger = nil

// Pools holds all loaded providers
var Pools = map[string]types.Provider{}
var rwlock sync.RWMutex

// Service implements the Messenger interface
type Service struct {
	config          *types.Config
	providers       map[string]types.Provider              // All providers by name
	providersByType map[types.MessageType][]types.Provider // Providers grouped by message type
	channels        map[string]types.Channel
	defaults        map[string]string
	mutex           sync.RWMutex
}

// Load loads the messenger configuration and providers
func Load(cfg config.Config) error {
	// Check if messengers directory exists
	exists, err := application.App.Exists("messengers")
	if err != nil {
		return err
	}
	if !exists {
		log.Warn("[Messenger] messengers directory not found, skip loading messenger")
		return nil
	}

	// Load channels configuration
	channelsPath := filepath.Join("messengers", "channels.yao")
	var channelsConfig map[string]interface{}
	if exists, _ := application.App.Exists(channelsPath); exists {
		raw, err := application.App.Read(channelsPath)
		if err != nil {
			return err
		}
		err = application.Parse("channels.yao", raw, &channelsConfig)
		if err != nil {
			return err
		}
	}

	// Load provider configurations
	providers, err := loadProviders()
	if err != nil {
		return err
	}

	// Create messenger configuration
	config := &types.Config{
		Providers: []types.ProviderConfig{},
		Channels:  make(map[string]types.Channel),
		Defaults:  make(map[string]string),
		Global: types.GlobalConfig{
			RetryAttempts: 3,
			RetryDelay:    time.Second * 2,
			Timeout:       time.Second * 30,
			LogLevel:      "info",
		},
	}

	// Parse channels configuration and convert to defaults map
	if channelsConfig != nil {
		parseChannelsConfig(channelsConfig, config.Defaults)
	}

	// Group providers by message type
	providersByType := make(map[types.MessageType][]types.Provider)
	for _, provider := range providers {
		// Determine which message types this provider supports
		supportedTypes := getSupportedMessageTypes(provider)
		for _, msgType := range supportedTypes {
			providersByType[msgType] = append(providersByType[msgType], provider)
		}
	}

	// Create messenger service
	service := &Service{
		config:          config,
		providers:       providers,
		providersByType: providersByType,
		channels:        make(map[string]types.Channel),
		defaults:        config.Defaults,
	}

	// Set global instance
	Instance = service
	return nil
}

// loadProviders loads all provider configurations from the providers directory
func loadProviders() (map[string]types.Provider, error) {
	providers := make(map[string]types.Provider)

	// Check if providers directory exists
	providersPath := "messengers/providers"
	exists, err := application.App.Exists(providersPath)
	if err != nil {
		return providers, err
	}
	if !exists {
		return providers, nil
	}

	// Walk through provider files
	messages := []string{}
	exts := []string{"*.yao", "*.json", "*.jsonc"}
	err = application.App.Walk(providersPath, func(root, file string, isdir bool) error {
		if isdir {
			return nil
		}

		provider, err := loadProvider(file, share.ID(root, file))
		if err != nil {
			messages = append(messages, err.Error())
			return nil // Continue loading other providers
		}

		if provider != nil {
			providers[provider.GetName()] = provider
		}
		return nil
	}, exts...)

	if len(messages) > 0 {
		log.Warn("[Messenger] Some providers failed to load: %s", strings.Join(messages, "; "))
	}

	return providers, err
}

// loadProvider loads a single provider configuration
func loadProvider(file string, name string) (types.Provider, error) {
	raw, err := application.App.Read(file)
	if err != nil {
		return nil, err
	}

	var config types.ProviderConfig
	err = application.Parse(file, raw, &config)
	if err != nil {
		return nil, err
	}

	// Set name if not provided
	if config.Name == "" {
		config.Name = name
	}

	// Create provider based on type
	return createProvider(config)
}

// createProvider creates a provider instance based on configuration
func createProvider(config types.ProviderConfig) (types.Provider, error) {
	// Default to enabled if not specified
	if !config.Enabled && config.Enabled != false {
		config.Enabled = true
	}

	if !config.Enabled {
		return nil, nil
	}

	// Use connector field to determine provider type
	connector := strings.ToLower(config.Connector)

	// Create provider based on connector
	switch connector {
	case "smtp":
		return smtp.NewSMTPProvider(config)
	case "twilio":
		return createTwilioProvider(config)
	case "mailgun":
		return mailgun.NewMailgunProvider(config)
	default:
		return nil, fmt.Errorf("unsupported connector: %s", connector)
	}
}

// createTwilioProvider creates a unified Twilio provider that handles all message types
func createTwilioProvider(config types.ProviderConfig) (types.Provider, error) {
	return twilio.NewTwilioProvider(config)
}

// Send sends a message using the specified channel or default provider
func (m *Service) Send(channel string, message *types.Message) error {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	// Get provider for channel
	providerName := m.getProviderForChannel(channel, string(message.Type))
	if providerName == "" {
		return fmt.Errorf("no provider configured for channel: %s, type: %s", channel, message.Type)
	}

	return m.SendWithProvider(providerName, message)
}

// SendWithProvider sends a message using a specific provider
func (m *Service) SendWithProvider(providerName string, message *types.Message) error {
	provider, exists := m.providers[providerName]
	if !exists {
		return fmt.Errorf("provider not found: %s", providerName)
	}

	// Validate message
	if err := m.validateMessage(message); err != nil {
		return fmt.Errorf("message validation failed: %w", err)
	}

	// Send message with retry logic
	var lastErr error
	maxAttempts := m.config.Global.RetryAttempts
	if maxAttempts <= 0 {
		maxAttempts = 1
	}

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		err := provider.Send(message)
		if err == nil {
			log.Info("[Messenger] Message sent successfully via %s (attempt %d/%d)", providerName, attempt, maxAttempts)
			return nil
		}

		lastErr = err
		if attempt < maxAttempts {
			log.Warn("[Messenger] Send attempt %d/%d failed for provider %s: %v", attempt, maxAttempts, providerName, err)
			time.Sleep(m.config.Global.RetryDelay)
		}
	}

	return fmt.Errorf("failed to send message after %d attempts: %w", maxAttempts, lastErr)
}

// SendBatch sends multiple messages in batch
func (m *Service) SendBatch(channel string, messages []*types.Message) error {
	if len(messages) == 0 {
		return nil
	}

	// Group messages by provider
	providerMessages := make(map[string][]*types.Message)
	for _, message := range messages {
		providerName := m.getProviderForChannel(channel, string(message.Type))
		if providerName == "" {
			return fmt.Errorf("no provider configured for channel: %s, type: %s", channel, message.Type)
		}
		providerMessages[providerName] = append(providerMessages[providerName], message)
	}

	// Send messages by provider
	var errors []string
	for providerName, msgs := range providerMessages {
		provider, exists := m.providers[providerName]
		if !exists {
			errors = append(errors, fmt.Sprintf("provider not found: %s", providerName))
			continue
		}

		err := provider.SendBatch(msgs)
		if err != nil {
			errors = append(errors, fmt.Sprintf("provider %s: %v", providerName, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("batch send errors: %s", strings.Join(errors, "; "))
	}
	return nil
}

// GetProvider returns a provider by name
func (m *Service) GetProvider(name string) (types.Provider, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	provider, exists := m.providers[name]
	if !exists {
		return nil, fmt.Errorf("provider not found: %s", name)
	}
	return provider, nil
}

// GetProviders returns all providers for a message type
func (m *Service) GetProviders(messageType string) []types.Provider {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	msgType := types.MessageType(strings.ToLower(messageType))
	if providers, exists := m.providersByType[msgType]; exists {
		return providers
	}
	return []types.Provider{}
}

// GetProvidersByMessageType returns all providers grouped by message type
func (m *Service) GetProvidersByMessageType() map[types.MessageType][]types.Provider {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	// Create a copy to avoid external modifications
	result := make(map[types.MessageType][]types.Provider)
	for msgType, providers := range m.providersByType {
		result[msgType] = make([]types.Provider, len(providers))
		copy(result[msgType], providers)
	}
	return result
}

// GetChannels returns all available channels
func (m *Service) GetChannels() []string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	channels := make([]string, 0, len(m.channels))
	for channel := range m.channels {
		channels = append(channels, channel)
	}

	// Add default channels
	for channel := range m.defaults {
		found := false
		for _, existing := range channels {
			if existing == channel {
				found = true
				break
			}
		}
		if !found {
			channels = append(channels, channel)
		}
	}

	return channels
}

// Close closes all provider connections
func (m *Service) Close() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	var errors []string
	for name, provider := range m.providers {
		if err := provider.Close(); err != nil {
			errors = append(errors, fmt.Sprintf("provider %s: %v", name, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("close errors: %s", strings.Join(errors, "; "))
	}
	return nil
}

// Helper methods

// getProviderForChannel returns the provider name for a given channel and message type
func (m *Service) getProviderForChannel(channel, messageType string) string {
	// Check channel-specific configuration first
	if ch, exists := m.channels[channel]; exists {
		if ch.Provider != "" {
			return ch.Provider
		}
	}

	// Check defaults for channel.messageType
	key := channel + "." + messageType
	if provider, exists := m.defaults[key]; exists {
		return provider
	}

	// Check defaults for messageType only
	if provider, exists := m.defaults[messageType]; exists {
		return provider
	}

	// Check defaults for channel only
	if provider, exists := m.defaults[channel]; exists {
		return provider
	}

	// If no specific provider configured, try to find any available provider for this message type
	msgType := types.MessageType(strings.ToLower(messageType))
	if providers, exists := m.providersByType[msgType]; exists && len(providers) > 0 {
		// Return the first available provider (could implement load balancing here)
		return providers[0].GetName()
	}

	return ""
}

// parseChannelsConfig parses the channels configuration and converts it to a defaults map
func parseChannelsConfig(channelsConfig map[string]interface{}, defaults map[string]string) {
	for channelName, channelData := range channelsConfig {
		if channelMap, ok := channelData.(map[string]interface{}); ok {
			// Iterate through each message type in the channel
			for key, value := range channelMap {
				if key == "description" {
					// Skip description field
					continue
				}

				if valueMap, ok := value.(map[string]interface{}); ok {
					// This is a message type configuration (email, sms, whatsapp)
					if provider, exists := valueMap["provider"]; exists {
						if providerStr, ok := provider.(string); ok {
							// Set channel.messageType -> provider mapping
							defaults[channelName+"."+key] = providerStr
						}
					}
				} else if valueStr, ok := value.(string); ok {
					// Direct provider assignment (legacy support)
					defaults[channelName+"."+key] = valueStr
				}
			}
		}
	}
}

// validateMessage validates a message before sending
func (m *Service) validateMessage(message *types.Message) error {
	if message == nil {
		return fmt.Errorf("message is nil")
	}
	if len(message.To) == 0 {
		return fmt.Errorf("message has no recipients")
	}
	if message.Body == "" && message.HTML == "" {
		return fmt.Errorf("message has no content")
	}
	if message.Type == types.MessageTypeEmail && message.Subject == "" {
		return fmt.Errorf("email message requires a subject")
	}
	return nil
}

// supportsChannelType checks if a provider supports a given channel type
func (m *Service) supportsChannelType(provider types.Provider, channelType string) bool {
	providerType := strings.ToLower(provider.GetType())
	channelType = strings.ToLower(channelType)

	switch channelType {
	case "email":
		return providerType == "smtp" || providerType == "mailgun" || providerType == "twilio"
	case "sms":
		return providerType == "twilio"
	case "whatsapp":
		return providerType == "twilio"
	default:
		return false
	}
}

// getSupportedMessageTypes returns the message types that a provider supports
func getSupportedMessageTypes(provider types.Provider) []types.MessageType {
	providerType := strings.ToLower(provider.GetType())

	switch providerType {
	case "smtp":
		return []types.MessageType{types.MessageTypeEmail}
	case "mailgun":
		return []types.MessageType{types.MessageTypeEmail}
	case "twilio":
		// Twilio provider supports all message types
		return []types.MessageType{types.MessageTypeSMS, types.MessageTypeWhatsApp, types.MessageTypeEmail}
	default:
		return []types.MessageType{}
	}
}
