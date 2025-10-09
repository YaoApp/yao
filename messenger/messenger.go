package messenger

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/messenger/providers/mailer"
	"github.com/yaoapp/yao/messenger/providers/mailgun"
	"github.com/yaoapp/yao/messenger/providers/twilio"
	"github.com/yaoapp/yao/messenger/template"
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
	receivers       map[string]context.CancelFunc // Active mail receivers by provider name
	messageHandlers []types.MessageHandler        // Registered message handlers for OnReceive
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

	// Load templates
	err = template.LoadTemplates()
	if err != nil {
		log.Warn("[Messenger] Failed to load templates: %v", err)
		// Don't fail messenger loading if templates fail
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
		receivers:       make(map[string]context.CancelFunc),
		messageHandlers: make([]types.MessageHandler, 0),
	}

	// Set global instance
	Instance = service

	// Auto-start mail receivers for mailer providers that support receiving
	service.startMailReceivers()

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

	// Resolve environment variables in the configuration
	if err := resolveProviderEnvVars(&config); err != nil {
		return nil, fmt.Errorf("failed to resolve environment variables: %w", err)
	}

	// Always use the file-based ID as the provider name for consistency
	// This ensures the name matches what's used in channels.yao configuration
	config.Name = name

	// Create provider based on type
	return createProvider(config)
}

// createProvider creates a provider instance based on configuration
func createProvider(config types.ProviderConfig) (types.Provider, error) {
	// Since bool zero value is false, and our config files don't specify "enabled",
	// we need to default to enabled=true. We'll assume providers are enabled unless
	// explicitly disabled in the configuration.
	// This is a simple fix: just assume enabled=true for all providers that don't explicitly set it
	config.Enabled = true

	if !config.Enabled {
		return nil, nil
	}

	// Use connector field to determine provider type
	connector := strings.ToLower(config.Connector)

	// Create provider based on connector
	switch connector {
	case "mailer":
		return mailer.NewMailerProvider(config)
	case "smtp": // Keep backward compatibility
		return mailer.NewMailerProvider(config)
	case "twilio":
		return createTwilioProvider(config)
	case "mailgun":
		return createMailgunProvider(config)
	default:
		return nil, fmt.Errorf("unsupported connector: %s", connector)
	}
}

// createTwilioProvider creates a unified Twilio provider that handles all message types
func createTwilioProvider(config types.ProviderConfig) (types.Provider, error) {
	return twilio.NewTwilioProviderWithTemplateManager(config, template.Global)
}

// createMailgunProvider creates a Mailgun provider with template manager
func createMailgunProvider(config types.ProviderConfig) (types.Provider, error) {
	return mailgun.NewMailgunProviderWithTemplateManager(config, template.Global)
}

// Send sends a message using the specified channel or default provider
func (m *Service) Send(ctx context.Context, channel string, message *types.Message) error {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	// Get provider for channel
	providerName := m.getProviderForChannel(channel, string(message.Type))
	if providerName == "" {
		return fmt.Errorf("no provider configured for channel: %s, type: %s", channel, message.Type)
	}

	return m.SendWithProvider(ctx, providerName, message)
}

// SendWithProvider sends a message using a specific provider
func (m *Service) SendWithProvider(ctx context.Context, providerName string, message *types.Message) error {
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
		// Check if context is cancelled before each attempt
		select {
		case <-ctx.Done():
			return fmt.Errorf("send cancelled: %w", ctx.Err())
		default:
		}

		err := provider.Send(ctx, message)
		if err == nil {
			log.Info("[Messenger] Message sent successfully via %s (attempt %d/%d)", providerName, attempt, maxAttempts)
			return nil
		}

		lastErr = err
		if attempt < maxAttempts {
			log.Warn("[Messenger] Send attempt %d/%d failed for provider %s: %v", attempt, maxAttempts, providerName, err)

			// Use context-aware sleep for retry delay
			select {
			case <-ctx.Done():
				return fmt.Errorf("send cancelled during retry: %w", ctx.Err())
			case <-time.After(m.config.Global.RetryDelay):
			}
		}
	}

	return fmt.Errorf("failed to send message after %d attempts: %w", maxAttempts, lastErr)
}

// SendT sends a message using a template
// messageType is optional - if not specified, the first available template type will be used
func (m *Service) SendT(ctx context.Context, channel string, templateID string, data types.TemplateData, messageType ...types.MessageType) error {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	// Determine which message type to use
	var msgType types.MessageType
	if len(messageType) > 0 {
		// Use specified message type
		msgType = messageType[0]
	} else {
		// Get available template types and use the first one
		availableTypes := template.Global.GetAvailableTypes(templateID)
		if len(availableTypes) == 0 {
			return fmt.Errorf("template not found: %s", templateID)
		}
		// Convert TemplateType to MessageType
		msgType = templateTypeToMessageType(availableTypes[0])
	}

	// Get provider for this channel and message type
	providerName := m.getProviderForChannel(channel, string(msgType))
	if providerName == "" {
		return fmt.Errorf("no provider configured for channel %s with message type %s", channel, msgType)
	}

	// Get the provider
	provider, exists := m.providers[providerName]
	if !exists {
		return fmt.Errorf("provider not found: %s", providerName)
	}

	// Convert MessageType back to TemplateType
	templateType := messageTypeToTemplateType(msgType)

	// Call provider's SendT method
	return provider.SendT(ctx, templateID, templateType, data)
}

// templateTypeToMessageType converts TemplateType to MessageType
func templateTypeToMessageType(templateType types.TemplateType) types.MessageType {
	switch templateType {
	case types.TemplateTypeMail:
		return types.MessageTypeEmail
	case types.TemplateTypeSMS:
		return types.MessageTypeSMS
	case types.TemplateTypeWhatsApp:
		return types.MessageTypeWhatsApp
	default:
		return ""
	}
}

// messageTypeToTemplateType converts MessageType to TemplateType
func messageTypeToTemplateType(messageType types.MessageType) types.TemplateType {
	switch messageType {
	case types.MessageTypeEmail:
		return types.TemplateTypeMail
	case types.MessageTypeSMS:
		return types.TemplateTypeSMS
	case types.MessageTypeWhatsApp:
		return types.TemplateTypeWhatsApp
	default:
		return ""
	}
}

// SendTWithProvider sends a message using a template and specific provider
// messageType is optional - if not specified, the first available template type will be used
func (m *Service) SendTWithProvider(ctx context.Context, providerName string, templateID string, data types.TemplateData, messageType ...types.MessageType) error {
	// Get provider
	provider, exists := m.providers[providerName]
	if !exists {
		return fmt.Errorf("provider not found: %s", providerName)
	}

	// Determine which message type to use
	var msgType types.MessageType
	if len(messageType) > 0 {
		// Use specified message type
		msgType = messageType[0]
	} else {
		// Get available template types and use the first one
		availableTypes := template.Global.GetAvailableTypes(templateID)
		if len(availableTypes) == 0 {
			return fmt.Errorf("template not found: %s", templateID)
		}
		// Convert TemplateType to MessageType
		msgType = templateTypeToMessageType(availableTypes[0])
	}

	// Convert MessageType to TemplateType
	templateType := messageTypeToTemplateType(msgType)

	// Call provider's SendT method
	return provider.SendT(ctx, templateID, templateType, data)
}

// SendTBatch sends multiple messages using templates in batch
// messageType is optional - if not specified, the first available template type will be used
func (m *Service) SendTBatch(ctx context.Context, channel string, templateID string, dataList []types.TemplateData, messageType ...types.MessageType) error {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if len(dataList) == 0 {
		return nil
	}

	// Determine which message type to use
	var msgType types.MessageType
	if len(messageType) > 0 {
		// Use specified message type
		msgType = messageType[0]
	} else {
		// Get available template types and use the first one
		availableTypes := template.Global.GetAvailableTypes(templateID)
		if len(availableTypes) == 0 {
			return fmt.Errorf("template not found: %s", templateID)
		}
		// Convert TemplateType to MessageType
		msgType = templateTypeToMessageType(availableTypes[0])
	}

	// Get provider for this channel and message type
	providerName := m.getProviderForChannel(channel, string(msgType))
	if providerName == "" {
		return fmt.Errorf("no provider configured for channel %s with message type %s", channel, msgType)
	}

	// Get the provider
	provider, exists := m.providers[providerName]
	if !exists {
		return fmt.Errorf("provider not found: %s", providerName)
	}

	// Convert MessageType to TemplateType
	templateType := messageTypeToTemplateType(msgType)

	// Get the template
	tmpl, err := template.Global.GetTemplate(templateID, templateType)
	if err != nil {
		return fmt.Errorf("template not found: %w", err)
	}

	// Convert all template data to messages
	messages := make([]*types.Message, 0, len(dataList))
	for _, data := range dataList {
		message, err := tmpl.ToMessage(data)
		if err != nil {
			return fmt.Errorf("failed to convert template to message: %w", err)
		}
		messages = append(messages, message)
	}

	// Send batch using the provider
	return provider.SendBatch(ctx, messages)
}

// SendTBatchWithProvider sends multiple messages using templates and specific provider in batch
// messageType is optional - if not specified, the first available template type will be used
func (m *Service) SendTBatchWithProvider(ctx context.Context, providerName string, templateID string, dataList []types.TemplateData, messageType ...types.MessageType) error {
	// Get provider
	provider, exists := m.providers[providerName]
	if !exists {
		return fmt.Errorf("provider not found: %s", providerName)
	}

	if len(dataList) == 0 {
		return nil
	}

	// Determine which message type to use
	var msgType types.MessageType
	if len(messageType) > 0 {
		// Use specified message type
		msgType = messageType[0]
	} else {
		// Get available template types and use the first one
		availableTypes := template.Global.GetAvailableTypes(templateID)
		if len(availableTypes) == 0 {
			return fmt.Errorf("template not found: %s", templateID)
		}
		// Convert TemplateType to MessageType
		msgType = templateTypeToMessageType(availableTypes[0])
	}

	// Convert MessageType to TemplateType
	templateType := messageTypeToTemplateType(msgType)

	// Get the template
	tmpl, err := template.Global.GetTemplate(templateID, templateType)
	if err != nil {
		return fmt.Errorf("template not found: %w", err)
	}

	// Convert all template data to messages
	messages := make([]*types.Message, 0, len(dataList))
	for _, data := range dataList {
		message, err := tmpl.ToMessage(data)
		if err != nil {
			return fmt.Errorf("failed to convert template to message: %w", err)
		}
		messages = append(messages, message)
	}

	// Send batch using the provider
	return provider.SendBatch(ctx, messages)
}

// SendTBatchMixed sends multiple messages using different templates with different data
// Each TemplateRequest can optionally specify its MessageType
func (m *Service) SendTBatchMixed(ctx context.Context, channel string, templateRequests []types.TemplateRequest) error {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if len(templateRequests) == 0 {
		return nil
	}

	// Group messages by provider
	providerMessages := make(map[string][]*types.Message)

	// Process each template request
	for _, request := range templateRequests {
		// Determine message type
		var msgType types.MessageType
		if request.MessageType != nil {
			// Use specified message type
			msgType = *request.MessageType
		} else {
			// Get available template types and use the first one
			availableTypes := template.Global.GetAvailableTypes(request.TemplateID)
			if len(availableTypes) == 0 {
				return fmt.Errorf("template not found: %s", request.TemplateID)
			}
			// Convert TemplateType to MessageType
			msgType = templateTypeToMessageType(availableTypes[0])
		}

		// Get provider for this channel and message type
		providerName := m.getProviderForChannel(channel, string(msgType))
		if providerName == "" {
			return fmt.Errorf("no provider configured for channel %s with message type %s", channel, msgType)
		}

		// Verify provider exists
		if _, exists := m.providers[providerName]; !exists {
			return fmt.Errorf("provider not found: %s", providerName)
		}

		// Convert MessageType to TemplateType
		templateType := messageTypeToTemplateType(msgType)

		// Get the template
		tmpl, err := template.Global.GetTemplate(request.TemplateID, templateType)
		if err != nil {
			return fmt.Errorf("template %s not found: %w", request.TemplateID, err)
		}

		// Convert template to message
		message, err := tmpl.ToMessage(request.Data)
		if err != nil {
			return fmt.Errorf("failed to convert template %s to message: %w", request.TemplateID, err)
		}

		// Add to provider's message list
		providerMessages[providerName] = append(providerMessages[providerName], message)
	}

	// Send batches to each provider
	var errors []string
	for providerName, messages := range providerMessages {
		// Check if context is cancelled
		select {
		case <-ctx.Done():
			return fmt.Errorf("batch send cancelled: %w", ctx.Err())
		default:
		}

		provider, exists := m.providers[providerName]
		if !exists {
			errors = append(errors, fmt.Sprintf("provider not found: %s", providerName))
			continue
		}

		err := provider.SendBatch(ctx, messages)
		if err != nil {
			errors = append(errors, fmt.Sprintf("provider %s: %v", providerName, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("batch send errors: %s", strings.Join(errors, "; "))
	}
	return nil
}

// SendTBatchMixedWithProvider sends multiple messages using different templates with different data and specific provider
// Each TemplateRequest can optionally specify its MessageType
func (m *Service) SendTBatchMixedWithProvider(ctx context.Context, providerName string, templateRequests []types.TemplateRequest) error {
	// Get provider
	provider, exists := m.providers[providerName]
	if !exists {
		return fmt.Errorf("provider not found: %s", providerName)
	}

	if len(templateRequests) == 0 {
		return nil
	}

	// Convert all template requests to messages
	messages := make([]*types.Message, 0, len(templateRequests))

	for _, request := range templateRequests {
		// Determine message type
		var msgType types.MessageType
		if request.MessageType != nil {
			// Use specified message type
			msgType = *request.MessageType
		} else {
			// Get available template types and use the first one
			availableTypes := template.Global.GetAvailableTypes(request.TemplateID)
			if len(availableTypes) == 0 {
				return fmt.Errorf("template not found: %s", request.TemplateID)
			}
			// Convert TemplateType to MessageType
			msgType = templateTypeToMessageType(availableTypes[0])
		}

		// Convert MessageType to TemplateType
		templateType := messageTypeToTemplateType(msgType)

		// Get the template
		tmpl, err := template.Global.GetTemplate(request.TemplateID, templateType)
		if err != nil {
			return fmt.Errorf("template %s not found: %w", request.TemplateID, err)
		}

		// Convert template to message
		message, err := tmpl.ToMessage(request.Data)
		if err != nil {
			return fmt.Errorf("failed to convert template %s to message: %w", request.TemplateID, err)
		}

		messages = append(messages, message)
	}

	// Send batch using the provider
	return provider.SendBatch(ctx, messages)
}

// SendBatch sends multiple messages in batch
func (m *Service) SendBatch(ctx context.Context, channel string, messages []*types.Message) error {
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
		// Check if context is cancelled before each provider
		select {
		case <-ctx.Done():
			return fmt.Errorf("batch send cancelled: %w", ctx.Err())
		default:
		}

		provider, exists := m.providers[providerName]
		if !exists {
			errors = append(errors, fmt.Sprintf("provider not found: %s", providerName))
			continue
		}

		err := provider.SendBatch(ctx, msgs)
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

// GetAllProviders returns all providers
func (m *Service) GetAllProviders() []types.Provider {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	providers := make([]types.Provider, 0, len(m.providers))
	for _, provider := range m.providers {
		providers = append(providers, provider)
	}
	return providers
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

// resolveProviderEnvVars resolves environment variables in provider configuration
func resolveProviderEnvVars(config *types.ProviderConfig) error {
	if config.Options != nil {
		resolved, err := resolveEnvVars(config.Options)
		if err != nil {
			return err
		}
		config.Options = resolved
	}
	return nil
}

// resolveEnvVars resolves environment variables in configuration values
func resolveEnvVars(config map[string]interface{}) (map[string]interface{}, error) {
	resolved := make(map[string]interface{})

	for key, value := range config {
		switch v := value.(type) {
		case string:
			resolved[key] = parseEnvVar(v)
		case map[string]interface{}:
			// Recursively resolve nested maps
			nestedResolved, err := resolveEnvVars(v)
			if err != nil {
				return nil, err
			}
			resolved[key] = nestedResolved
		default:
			resolved[key] = value
		}
	}

	return resolved, nil
}

// parseEnvVar parses environment variable pattern $ENV.VAR_NAME
func parseEnvVar(value string) string {
	// Pattern to match $ENV.VAR_NAME (same as kb package)
	envPattern := regexp.MustCompile(`\$ENV\.([A-Za-z_][A-Za-z0-9_]*)`)

	return envPattern.ReplaceAllStringFunc(value, func(match string) string {
		// Extract variable name (remove $ENV. prefix)
		varName := strings.TrimPrefix(match, "$ENV.")

		// Get environment variable value
		if envValue := os.Getenv(varName); envValue != "" {
			return envValue
		}

		// Return original if environment variable is not set
		return match
	})
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
		return providerType == "mailer" || providerType == "smtp" || providerType == "mailgun" || providerType == "twilio"
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
	case "mailer":
		return []types.MessageType{types.MessageTypeEmail}
	case "smtp": // Keep backward compatibility
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

// startMailReceivers automatically starts mail receivers for mailer providers that support receiving
func (m *Service) startMailReceivers() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	for name, provider := range m.providers {
		// Only handle mailer providers
		if provider.GetType() != "mailer" {
			continue
		}

		// Check if this mailer provider supports receiving
		if mailerProvider, ok := provider.(*mailer.Provider); ok {
			if mailerProvider.SupportsReceiving() {
				log.Info("[Messenger] Starting mail receiver for provider: %s", name)

				// Create context for this receiver
				ctx, cancel := context.WithCancel(context.Background())

				// Start the mail receiver in a goroutine
				go func(providerName string, mp *mailer.Provider) {
					err := mp.StartMailReceiver(ctx, func(msg *types.Message) error {
						log.Info("[Messenger] Received email via %s: Subject=%s, From=%s", providerName, msg.Subject, msg.From)

						// Trigger OnReceive handlers for the received message
						if err := m.triggerOnReceiveHandlers(ctx, msg); err != nil {
							log.Error("[Messenger] Failed to trigger OnReceive handlers: %v", err)
							return err
						}

						return nil
					})

					if err != nil {
						log.Error("[Messenger] Mail receiver for %s stopped with error: %v", providerName, err)
					} else {
						log.Info("[Messenger] Mail receiver for %s stopped gracefully", providerName)
					}
				}(name, mailerProvider)

				// Store the cancel function for later cleanup
				m.receivers[name] = cancel

				log.Info("[Messenger] Mail receiver started for provider: %s", name)
			} else {
				log.Debug("[Messenger] Provider %s does not support receiving (IMAP not configured)", name)
			}
		}
	}
}

// StopMailReceivers stops all active mail receivers
func (m *Service) StopMailReceivers() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	for name, cancel := range m.receivers {
		log.Info("[Messenger] Stopping mail receiver for provider: %s", name)
		cancel()
	}

	// Clear the receivers map
	m.receivers = make(map[string]context.CancelFunc)
	log.Info("[Messenger] All mail receivers stopped")
}

// StopMailReceiver stops a specific mail receiver by provider name
func (m *Service) StopMailReceiver(providerName string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if cancel, exists := m.receivers[providerName]; exists {
		log.Info("[Messenger] Stopping mail receiver for provider: %s", providerName)
		cancel()
		delete(m.receivers, providerName)
	} else {
		log.Warn("[Messenger] No active mail receiver found for provider: %s", providerName)
	}
}

// GetActiveReceivers returns the names of all active mail receivers
func (m *Service) GetActiveReceivers() []string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var receivers []string
	for name := range m.receivers {
		receivers = append(receivers, name)
	}
	return receivers
}

// OnReceive registers a message handler for received messages
// Multiple handlers can be registered and will be called in order
func (m *Service) OnReceive(handler types.MessageHandler) error {
	if handler == nil {
		return fmt.Errorf("handler cannot be nil")
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.messageHandlers = append(m.messageHandlers, handler)
	log.Info("[Messenger] Registered new message handler (total: %d)", len(m.messageHandlers))
	return nil
}

// RemoveReceiveHandler removes a previously registered message handler
func (m *Service) RemoveReceiveHandler(handler types.MessageHandler) error {
	if handler == nil {
		return fmt.Errorf("handler cannot be nil")
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Find and remove the handler by comparing function pointers
	handlerPtr := reflect.ValueOf(handler).Pointer()
	for i, existingHandler := range m.messageHandlers {
		if reflect.ValueOf(existingHandler).Pointer() == handlerPtr {
			// Remove handler at index i
			m.messageHandlers = append(m.messageHandlers[:i], m.messageHandlers[i+1:]...)
			log.Info("[Messenger] Removed message handler (remaining: %d)", len(m.messageHandlers))
			return nil
		}
	}

	return fmt.Errorf("handler not found")
}

// TriggerWebhook processes incoming webhook data and triggers OnReceive handlers
// This is used by OPENAPI endpoints to handle incoming messages
func (m *Service) TriggerWebhook(providerName string, c interface{}) error {
	// Get the provider to process the webhook data
	provider, exists := m.providers[providerName]
	if !exists {
		return fmt.Errorf("provider not found: %s", providerName)
	}

	// Let the provider process the webhook data and convert to Message
	message, err := provider.TriggerWebhook(c)
	if err != nil {
		log.Warn("[Messenger] Provider %s failed to process webhook: %v", providerName, err)
		return err
	}

	// Create context from gin.Context if available, otherwise use background
	var ctx context.Context
	if ginCtx, ok := c.(*gin.Context); ok {
		ctx = ginCtx.Request.Context()
	} else {
		ctx = context.Background()
	}

	// Trigger all registered OnReceive handlers
	return m.triggerOnReceiveHandlers(ctx, message)
}

// Note: convertWebhookToMessage has been removed as it's replaced by provider-specific TriggerWebhook implementations

// triggerOnReceiveHandlers calls all registered OnReceive handlers
func (m *Service) triggerOnReceiveHandlers(ctx context.Context, message *types.Message) error {
	m.mutex.RLock()
	handlers := make([]types.MessageHandler, len(m.messageHandlers))
	copy(handlers, m.messageHandlers)
	m.mutex.RUnlock()

	if len(handlers) == 0 {
		log.Debug("[Messenger] No OnReceive handlers registered")
		return nil
	}

	log.Info("[Messenger] Triggering %d OnReceive handlers for message: %s", len(handlers), message.Subject)

	var errors []string
	for i, handler := range handlers {
		err := handler(ctx, message)
		if err != nil {
			errMsg := fmt.Sprintf("handler %d failed: %v", i, err)
			errors = append(errors, errMsg)
			log.Error("[Messenger] %s", errMsg)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("some OnReceive handlers failed: %s", strings.Join(errors, "; "))
	}

	return nil
}
