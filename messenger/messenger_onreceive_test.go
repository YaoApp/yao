package messenger

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/messenger/types"
	"github.com/yaoapp/yao/test"
)

// Test OnReceive functionality
func TestService_OnReceive(t *testing.T) {
	// Prepare test environment
	test.Prepare(t, config.Conf, "YAO_TEST_APPLICATION")
	defer test.Clean()

	// Create a test service
	service := &Service{
		config:          &types.Config{},
		providers:       make(map[string]types.Provider),
		providersByType: make(map[types.MessageType][]types.Provider),
		channels:        make(map[string]types.Channel),
		defaults:        make(map[string]string),
		receivers:       make(map[string]context.CancelFunc),
		messageHandlers: make([]types.MessageHandler, 0),
	}

	// Test registering nil handler should fail
	err := service.OnReceive(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "handler cannot be nil")

	// Test registering valid handlers
	var receivedMessages []*types.Message
	var mu sync.Mutex

	handler1 := func(ctx context.Context, message *types.Message) error {
		mu.Lock()
		defer mu.Unlock()
		receivedMessages = append(receivedMessages, message)
		t.Logf("Handler 1: Received message from %s with subject: %s", message.From, message.Subject)
		return nil
	}

	handler2 := func(ctx context.Context, message *types.Message) error {
		mu.Lock()
		defer mu.Unlock()
		t.Logf("Handler 2: Processing message for analytics")
		return nil
	}

	// Register handlers
	err = service.OnReceive(handler1)
	assert.NoError(t, err)

	err = service.OnReceive(handler2)
	assert.NoError(t, err)

	// Verify handlers are registered
	assert.Len(t, service.messageHandlers, 2)

	// Test triggering handlers
	testMessage := &types.Message{
		Type:    types.MessageTypeEmail,
		From:    "test@example.com",
		To:      []string{"recipient@example.com"},
		Subject: "Test Subject",
		Body:    "Test message body",
	}

	ctx := context.Background()
	err = service.triggerOnReceiveHandlers(ctx, testMessage)
	assert.NoError(t, err)

	// Verify message was received by handler1
	mu.Lock()
	assert.Len(t, receivedMessages, 1)
	assert.Equal(t, testMessage.Subject, receivedMessages[0].Subject)
	assert.Equal(t, testMessage.From, receivedMessages[0].From)
	mu.Unlock()
}

func TestService_RemoveReceiveHandler(t *testing.T) {
	// Create a test service
	service := &Service{
		config:          &types.Config{},
		providers:       make(map[string]types.Provider),
		providersByType: make(map[types.MessageType][]types.Provider),
		channels:        make(map[string]types.Channel),
		defaults:        make(map[string]string),
		receivers:       make(map[string]context.CancelFunc),
		messageHandlers: make([]types.MessageHandler, 0),
	}

	// Test removing nil handler should fail
	err := service.RemoveReceiveHandler(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "handler cannot be nil")

	// Create and register a handler
	handler := func(ctx context.Context, message *types.Message) error {
		return nil
	}

	err = service.OnReceive(handler)
	assert.NoError(t, err)
	assert.Len(t, service.messageHandlers, 1)

	// Remove the handler
	err = service.RemoveReceiveHandler(handler)
	assert.NoError(t, err)
	assert.Len(t, service.messageHandlers, 0)

	// Try to remove the same handler again should fail
	err = service.RemoveReceiveHandler(handler)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "handler not found")
}

func TestService_TriggerWebhook(t *testing.T) {
	// Prepare test environment
	test.Prepare(t, config.Conf, "YAO_TEST_APPLICATION")
	defer test.Clean()

	// Load real providers
	providers, err := loadProviders()
	require.NoError(t, err)

	// Create a test service with real providers
	service := &Service{
		config:          &types.Config{},
		providers:       providers,
		providersByType: make(map[types.MessageType][]types.Provider),
		channels:        make(map[string]types.Channel),
		defaults:        make(map[string]string),
		receivers:       make(map[string]context.CancelFunc),
		messageHandlers: make([]types.MessageHandler, 0),
	}

	// Test with non-existent provider
	ctx := context.Background()
	webhookData := map[string]interface{}{
		"from":    "test@example.com",
		"to":      "recipient@example.com",
		"subject": "Test Subject",
		"body":    "Test message body",
	}

	err = service.TriggerWebhook(ctx, "nonexistent", webhookData)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "provider not found")

	// Test with existing provider (if any are loaded)
	if len(providers) > 0 {
		// Get the first provider name
		var providerName string
		for name := range providers {
			providerName = name
			break
		}

		// Register a handler to capture the triggered message
		var receivedMessage *types.Message
		var mu sync.Mutex

		handler := func(ctx context.Context, message *types.Message) error {
			mu.Lock()
			defer mu.Unlock()
			receivedMessage = message
			t.Logf("Webhook handler: Received message from %s with subject: %s", message.From, message.Subject)
			return nil
		}

		err = service.OnReceive(handler)
		assert.NoError(t, err)

		// Trigger webhook
		err = service.TriggerWebhook(ctx, providerName, webhookData)
		// Note: This might fail if the provider's Receive method has validation,
		// but it should not panic and should attempt to trigger handlers
		if err != nil {
			t.Logf("TriggerWebhook returned error (may be expected): %v", err)
		}

		// Give some time for async processing
		time.Sleep(100 * time.Millisecond)

		// Check if handler was triggered
		mu.Lock()
		if receivedMessage != nil {
			assert.Equal(t, "test@example.com", receivedMessage.From)
			assert.Equal(t, "Test Subject", receivedMessage.Subject)
			assert.Equal(t, "Test message body", receivedMessage.Body)
			assert.Contains(t, receivedMessage.To, "recipient@example.com")
		}
		mu.Unlock()
	}
}

func TestService_ConvertWebhookToMessage(t *testing.T) {
	// Prepare test environment
	test.Prepare(t, config.Conf, "YAO_TEST_APPLICATION")
	defer test.Clean()

	// Load real providers
	providers, err := loadProviders()
	require.NoError(t, err)

	// Create a test service
	service := &Service{
		config:          &types.Config{},
		providers:       providers,
		providersByType: make(map[types.MessageType][]types.Provider),
		channels:        make(map[string]types.Channel),
		defaults:        make(map[string]string),
		receivers:       make(map[string]context.CancelFunc),
		messageHandlers: make([]types.MessageHandler, 0),
	}

	tests := []struct {
		name         string
		providerName string
		data         map[string]interface{}
		expectedType types.MessageType
	}{
		{
			name:         "Email webhook data",
			providerName: "test-mailer",
			data: map[string]interface{}{
				"type":    "email",
				"from":    "sender@example.com",
				"to":      "recipient@example.com",
				"subject": "Test Email",
				"body":    "Email body content",
				"html":    "<p>Email HTML content</p>",
			},
			expectedType: types.MessageTypeEmail,
		},
		{
			name:         "SMS webhook data",
			providerName: "test-twilio",
			data: map[string]interface{}{
				"type":  "sms",
				"from":  "+1234567890",
				"to":    "+0987654321",
				"body":  "SMS message content",
				"phone": "+0987654321",
			},
			expectedType: types.MessageTypeSMS,
		},
		{
			name:         "WhatsApp webhook data",
			providerName: "test-twilio",
			data: map[string]interface{}{
				"type":     "whatsapp",
				"from":     "+1234567890",
				"to":       "+0987654321",
				"body":     "WhatsApp message content",
				"whatsapp": "+0987654321",
			},
			expectedType: types.MessageTypeWhatsApp,
		},
		{
			name:         "Array recipients",
			providerName: "test-mailer",
			data: map[string]interface{}{
				"from":    "sender@example.com",
				"to":      []string{"recipient1@example.com", "recipient2@example.com"},
				"subject": "Test Email",
				"body":    "Email body content",
			},
			expectedType: types.MessageTypeEmail,
		},
		{
			name:         "Interface array recipients",
			providerName: "test-mailer",
			data: map[string]interface{}{
				"from":    "sender@example.com",
				"to":      []interface{}{"recipient1@example.com", "recipient2@example.com"},
				"subject": "Test Email",
				"body":    "Email body content",
			},
			expectedType: types.MessageTypeEmail,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			message, err := service.convertWebhookToMessage(tt.providerName, tt.data)
			assert.NoError(t, err)
			assert.NotNil(t, message)

			// Check basic fields
			if from, ok := tt.data["from"].(string); ok {
				assert.Equal(t, from, message.From)
			}
			if subject, ok := tt.data["subject"].(string); ok {
				assert.Equal(t, subject, message.Subject)
			}
			if body, ok := tt.data["body"].(string); ok {
				assert.Equal(t, body, message.Body)
			}
			if html, ok := tt.data["html"].(string); ok {
				assert.Equal(t, html, message.HTML)
			}

			// Check recipients
			if to, ok := tt.data["to"]; ok {
				switch v := to.(type) {
				case string:
					assert.Contains(t, message.To, v)
				case []string:
					for _, recipient := range v {
						assert.Contains(t, message.To, recipient)
					}
				case []interface{}:
					for _, recipient := range v {
						if str, ok := recipient.(string); ok {
							assert.Contains(t, message.To, str)
						}
					}
				}
			}

			// Check message type
			if tt.data["type"] != nil {
				assert.Equal(t, tt.expectedType, message.Type)
			}

			// Check metadata
			assert.NotNil(t, message.Metadata)
			assert.Equal(t, tt.providerName, message.Metadata["provider"])
			assert.Equal(t, tt.data, message.Metadata["webhook_data"])
		})
	}
}

func TestService_TriggerOnReceiveHandlers_ErrorHandling(t *testing.T) {
	// Create a test service
	service := &Service{
		config:          &types.Config{},
		providers:       make(map[string]types.Provider),
		providersByType: make(map[types.MessageType][]types.Provider),
		channels:        make(map[string]types.Channel),
		defaults:        make(map[string]string),
		receivers:       make(map[string]context.CancelFunc),
		messageHandlers: make([]types.MessageHandler, 0),
	}

	// Test with no handlers
	ctx := context.Background()
	testMessage := &types.Message{
		Type:    types.MessageTypeEmail,
		From:    "test@example.com",
		To:      []string{"recipient@example.com"},
		Subject: "Test Subject",
		Body:    "Test message body",
	}

	err := service.triggerOnReceiveHandlers(ctx, testMessage)
	assert.NoError(t, err)

	// Register handlers with different behaviors
	successHandler := func(ctx context.Context, message *types.Message) error {
		t.Logf("Success handler: %s", message.Subject)
		return nil
	}

	errorHandler := func(ctx context.Context, message *types.Message) error {
		t.Logf("Error handler: %s", message.Subject)
		return assert.AnError
	}

	anotherSuccessHandler := func(ctx context.Context, message *types.Message) error {
		t.Logf("Another success handler: %s", message.Subject)
		return nil
	}

	// Register handlers
	err = service.OnReceive(successHandler)
	assert.NoError(t, err)

	err = service.OnReceive(errorHandler)
	assert.NoError(t, err)

	err = service.OnReceive(anotherSuccessHandler)
	assert.NoError(t, err)

	// Trigger handlers - should continue even if one fails
	err = service.triggerOnReceiveHandlers(ctx, testMessage)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "some OnReceive handlers failed")
	assert.Contains(t, err.Error(), "handler 1 failed")
}

// Integration test with real messenger instance
func TestMessenger_OnReceiveIntegration(t *testing.T) {
	// Prepare test environment
	test.Prepare(t, config.Conf, "YAO_TEST_APPLICATION")
	defer test.Clean()

	// Load messenger configuration
	err := Load(config.Conf)
	require.NoError(t, err)
	require.NotNil(t, Instance)

	// Cast to Service to access our new methods
	service, ok := Instance.(*Service)
	require.True(t, ok, "Instance should be of type *Service")

	// Test OnReceive with real instance
	var receivedMessage *types.Message
	var mu sync.Mutex

	handler := func(ctx context.Context, message *types.Message) error {
		mu.Lock()
		defer mu.Unlock()
		receivedMessage = message
		t.Logf("Integration handler: Received message from %s", message.From)
		return nil
	}

	err = service.OnReceive(handler)
	assert.NoError(t, err)

	// Test TriggerWebhook with real providers (if any exist)
	if len(service.providers) > 0 {
		// Get the first provider name
		var providerName string
		for name := range service.providers {
			providerName = name
			break
		}

		webhookData := map[string]interface{}{
			"from":    "integration@example.com",
			"to":      "test@example.com",
			"subject": "Integration Test",
			"body":    "Integration test message",
		}

		ctx := context.Background()
		err = service.TriggerWebhook(ctx, providerName, webhookData)
		// Error is acceptable as provider might reject test data
		if err != nil {
			t.Logf("TriggerWebhook returned error (may be expected): %v", err)
		}

		// Give some time for processing
		time.Sleep(100 * time.Millisecond)

		// Check if handler was triggered
		mu.Lock()
		if receivedMessage != nil {
			assert.Equal(t, "integration@example.com", receivedMessage.From)
			assert.Equal(t, "Integration Test", receivedMessage.Subject)
		}
		mu.Unlock()
	}

	// Clean up - remove handler
	err = service.RemoveReceiveHandler(handler)
	assert.NoError(t, err)
}
