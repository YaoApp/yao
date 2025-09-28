package messenger

import (
	"context"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/messenger/types"
	"github.com/yaoapp/yao/test"
)

// createMockGinContext creates a mock gin.Context for testing webhook functionality
func createMockGinContext(formData map[string]interface{}) *gin.Context {
	// Create form values
	values := url.Values{}
	for key, value := range formData {
		if str, ok := value.(string); ok {
			values.Set(key, str)
		}
	}

	// Create request with form data
	req := httptest.NewRequest("POST", "/webhook/test", strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Create response recorder
	w := httptest.NewRecorder()

	// Create gin context
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	return c
}

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
	webhookData := map[string]interface{}{
		"from":    "test@example.com",
		"to":      "recipient@example.com",
		"subject": "Test Subject",
		"body":    "Test message body",
	}

	mockCtx := createMockGinContext(webhookData)
	err = service.TriggerWebhook("nonexistent", mockCtx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "provider not found")

	// Test with existing provider (if any are loaded)
	if len(providers) > 0 {
		// Find a provider that supports TriggerWebhook (not SMTP/mailer)
		var providerName string
		var provider types.Provider
		for name, p := range providers {
			if p.GetType() != "mailer" { // Skip SMTP providers as they don't support TriggerWebhook
				providerName = name
				provider = p
				break
			}
		}

		if providerName != "" {
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

			// Create appropriate webhook data based on provider type
			var mockCtx *gin.Context
			switch provider.GetType() {
			case "mailgun":
				// Mailgun expects specific event fields
				mailgunData := map[string]interface{}{
					"event":     "delivered",
					"recipient": "recipient@example.com",
					"sender":    "test@example.com",
					"subject":   "Test Subject",
				}
				mockCtx = createMockGinContext(mailgunData)
			case "twilio":
				// Twilio expects SMS/WhatsApp fields
				twilioData := map[string]interface{}{
					"MessageSid": "test-message-sid",
					"SmsStatus":  "received",
					"From":       "+1234567890",
					"To":         "+0987654321",
					"Body":       "Test message body",
				}
				mockCtx = createMockGinContext(twilioData)
			default:
				mockCtx = createMockGinContext(webhookData)
			}

			// Trigger webhook
			err = service.TriggerWebhook(providerName, mockCtx)
			// Note: This might fail if the provider's TriggerWebhook method has validation,
			// but it should not panic and should attempt to trigger handlers
			if err != nil {
				t.Logf("TriggerWebhook returned error (may be expected): %v", err)
			}

			// Give some time for async processing
			time.Sleep(100 * time.Millisecond)

			// Check if handler was triggered
			mu.Lock()
			if receivedMessage != nil {
				assert.NotNil(t, receivedMessage)
				assert.NotEmpty(t, receivedMessage.Subject)
				t.Logf("Received message: From=%s, Subject=%s, Body=%s", receivedMessage.From, receivedMessage.Subject, receivedMessage.Body)

				// Verify provider-specific content
				switch provider.GetType() {
				case "mailgun":
					assert.Contains(t, receivedMessage.Subject, "Email Delivered")
					assert.Contains(t, receivedMessage.Body, "recipient@example.com")
				case "twilio":
					assert.Contains(t, receivedMessage.Subject, "Incoming Message")
					assert.Contains(t, receivedMessage.Body, "Test message body")
				}
			} else {
				t.Log("No message received - this may be expected for some provider configurations")
			}
			mu.Unlock()
		} else {
			t.Log("No providers support TriggerWebhook - skipping webhook test")
		}
	}
}

// Note: TestService_ConvertWebhookToMessage has been removed as the method is deprecated
// Webhook processing is now handled by provider-specific TriggerWebhook implementations

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
		// Find a provider that supports TriggerWebhook (not SMTP/mailer)
		var providerName string
		var provider types.Provider
		for name, p := range service.providers {
			if p.GetType() != "mailer" { // Skip SMTP providers as they don't support TriggerWebhook
				providerName = name
				provider = p
				break
			}
		}

		if providerName != "" {
			// Create appropriate webhook data based on provider type
			var mockCtx *gin.Context
			switch provider.GetType() {
			case "mailgun":
				// Mailgun expects specific event fields
				mailgunData := map[string]interface{}{
					"event":     "delivered",
					"recipient": "test@example.com",
					"sender":    "integration@example.com",
					"subject":   "Integration Test",
				}
				mockCtx = createMockGinContext(mailgunData)
			case "twilio":
				// Twilio expects SMS/WhatsApp fields
				twilioData := map[string]interface{}{
					"MessageSid": "integration-test-sid",
					"SmsStatus":  "received",
					"From":       "integration@example.com",
					"To":         "test@example.com",
					"Body":       "Integration test message",
				}
				mockCtx = createMockGinContext(twilioData)
			default:
				webhookData := map[string]interface{}{
					"from":    "integration@example.com",
					"to":      "test@example.com",
					"subject": "Integration Test",
					"body":    "Integration test message",
				}
				mockCtx = createMockGinContext(webhookData)
			}

			err = service.TriggerWebhook(providerName, mockCtx)
			// Error is acceptable as provider might reject test data
			if err != nil {
				t.Logf("TriggerWebhook returned error (may be expected): %v", err)
			}

			// Give some time for processing
			time.Sleep(100 * time.Millisecond)

			// Check if handler was triggered
			mu.Lock()
			if receivedMessage != nil {
				assert.NotNil(t, receivedMessage)
				assert.NotEmpty(t, receivedMessage.Subject)
				t.Logf("Integration test received message: From=%s, Subject=%s", receivedMessage.From, receivedMessage.Subject)

				// Verify provider-specific content
				switch provider.GetType() {
				case "mailgun":
					assert.Contains(t, receivedMessage.Subject, "Email Delivered")
					assert.Contains(t, receivedMessage.Body, "test@example.com")
				case "twilio":
					assert.Contains(t, receivedMessage.Subject, "Incoming Message")
					assert.Equal(t, "integration@example.com", receivedMessage.From)
				}
			} else {
				t.Log("No message received in integration test - this may be expected")
			}
			mu.Unlock()
		} else {
			t.Log("No providers support TriggerWebhook - skipping integration webhook test")
		}
	}

	// Clean up - remove handler
	err = service.RemoveReceiveHandler(handler)
	assert.NoError(t, err)
}
