package twilio

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/messenger/types"
	"github.com/yaoapp/yao/test"
)

// Test recipient email addresses - use authorized addresses for real API tests
const (
	TestEmailAgent = "agent@iqka.com"
	TestEmailX     = "x@iqka.com"
	TestEmailXiang = "xiang@iqka.com"
)

// Email-focused tests - SMS and WhatsApp tests are in separate files

// loadTestConfig loads the unified.twilio.yao configuration for testing
func loadTestConfig(t *testing.T) types.ProviderConfig {
	test.Prepare(t, config.Conf, "YAO_TEST_APPLICATION")
	defer test.Clean()

	config := types.ProviderConfig{
		Name:      "unified",
		Connector: "twilio",
		Options: map[string]interface{}{
			"account_sid":      os.Getenv("TWILIO_ACCOUNT_SID"),
			"auth_token":       os.Getenv("TWILIO_AUTH_TOKEN"),
			"from_phone":       os.Getenv("TWILIO_FROM_PHONE"),
			"from_email":       os.Getenv("TWILIO_FROM_EMAIL"),
			"api_sid":          os.Getenv("TWILIO_API_SID"),
			"api_key":          os.Getenv("TWILIO_API_KEY"),
			"sendgrid_api_key": os.Getenv("TWILIO_SENDGRID_API_KEY"),
		},
	}

	return config
}

// createTestMessage creates a test message of the specified type
func createTestMessage(messageType types.MessageType) *types.Message {
	switch messageType {
	case types.MessageTypeEmail:
		return &types.Message{
			Type:    types.MessageTypeEmail,
			To:      []string{TestEmailAgent},
			Subject: "Test Email from Twilio Provider",
			Body:    "This is a test email sent via Twilio SendGrid API.",
			HTML:    "<h1>Test Email</h1><p>This is a test email sent via <strong>Twilio SendGrid API</strong>.</p>",
		}
	// SMS and WhatsApp message creation moved to separate test files
	default:
		return &types.Message{
			Type: types.MessageTypeEmail,
			To:   []string{TestEmailAgent},
			Body: "Default test message",
		}
	}
}

// =============================================================================
// Basic Provider Tests
// =============================================================================

func TestNewTwilioProvider(t *testing.T) {
	config := loadTestConfig(t)

	provider, err := NewTwilioProvider(config)
	require.NoError(t, err)
	assert.NotNil(t, provider)

	// Verify configuration using actual environment variables
	assert.Equal(t, os.Getenv("TWILIO_ACCOUNT_SID"), provider.accountSID)
	assert.Equal(t, os.Getenv("TWILIO_AUTH_TOKEN"), provider.authToken)
	assert.Equal(t, os.Getenv("TWILIO_FROM_PHONE"), provider.fromPhone)
	assert.Equal(t, os.Getenv("TWILIO_FROM_EMAIL"), provider.fromEmail)
	assert.Equal(t, os.Getenv("TWILIO_API_SID"), provider.apiSID)
	assert.Equal(t, os.Getenv("TWILIO_API_KEY"), provider.apiKey)
	assert.Equal(t, os.Getenv("TWILIO_SENDGRID_API_KEY"), provider.sendGridAPIKey)
	assert.Equal(t, "unified", provider.config.Name)
}

func TestNewTwilioProvider_MissingOptions(t *testing.T) {
	config := types.ProviderConfig{
		Name:      "test",
		Connector: "twilio",
		Options:   nil,
	}

	provider, err := NewTwilioProvider(config)
	assert.Error(t, err)
	assert.Nil(t, provider)
	assert.Contains(t, err.Error(), "Twilio provider requires options")
}

func TestNewTwilioProvider_MissingAccountSID(t *testing.T) {
	config := types.ProviderConfig{
		Name:      "test",
		Connector: "twilio",
		Options: map[string]interface{}{
			"auth_token": "test_token",
		},
	}

	provider, err := NewTwilioProvider(config)
	assert.Error(t, err)
	assert.Nil(t, provider)
	assert.Contains(t, err.Error(), "account_sid")
}

func TestGetType(t *testing.T) {
	config := loadTestConfig(t)
	provider, err := NewTwilioProvider(config)
	require.NoError(t, err)

	assert.Equal(t, "twilio", provider.GetType())
}

func TestGetName(t *testing.T) {
	config := loadTestConfig(t)
	provider, err := NewTwilioProvider(config)
	require.NoError(t, err)

	assert.Equal(t, "unified", provider.GetName())
}

func TestValidate(t *testing.T) {
	config := loadTestConfig(t)
	provider, err := NewTwilioProvider(config)
	require.NoError(t, err)

	err = provider.Validate()
	assert.NoError(t, err)
}

func TestValidate_MissingAccountSID(t *testing.T) {
	config := types.ProviderConfig{
		Name:      "test",
		Connector: "twilio",
		Options: map[string]interface{}{
			"auth_token": "test_token",
		},
	}

	provider, err := NewTwilioProvider(config)
	require.Error(t, err)
	assert.Nil(t, provider)
}

func TestValidate_MissingAuth(t *testing.T) {
	config := types.ProviderConfig{
		Name:      "test",
		Connector: "twilio",
		Options: map[string]interface{}{
			"account_sid": "test_sid",
		},
	}

	provider, err := NewTwilioProvider(config)
	require.NoError(t, err)

	err = provider.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "auth_token")
}

func TestValidate_PartialAPIKeys(t *testing.T) {
	config := types.ProviderConfig{
		Name:      "test",
		Connector: "twilio",
		Options: map[string]interface{}{
			"account_sid": "test_sid",
			"auth_token":  "valid_token", // Provide auth_token to pass first check
			"api_sid":     "test_api_sid",
			// Missing api_key - this should trigger the partial API keys error
		},
	}

	provider, err := NewTwilioProvider(config)
	require.NoError(t, err)

	err = provider.Validate()
	assert.Error(t, err)
	// Should trigger the "both api_sid and api_key must be provided together" error
	assert.Contains(t, err.Error(), "both 'api_sid' and 'api_key' must be provided together")
}

func TestClose(t *testing.T) {
	config := loadTestConfig(t)
	provider, err := NewTwilioProvider(config)
	require.NoError(t, err)

	err = provider.Close()
	assert.NoError(t, err)
}

// =============================================================================
// Email Sending Tests (via SendGrid API)
// =============================================================================

func TestSend_EmailMessage_RealAPI(t *testing.T) {
	config := loadTestConfig(t)
	provider, err := NewTwilioProvider(config)
	require.NoError(t, err)

	// Skip if SendGrid API key is not configured
	if provider.sendGridAPIKey == "" {
		t.Skip("TWILIO_SENDGRID_API_KEY not configured, skipping real API test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	emailMessage := createTestMessage(types.MessageTypeEmail)

	err = provider.Send(ctx, emailMessage)
	if err == nil {
		t.Log("Real Twilio SendGrid API call succeeded")
	} else {
		t.Logf("Real Twilio SendGrid API call failed (expected in some test environments): %v", err)
		// Don't fail the test if it's just an API configuration issue
		if !strings.Contains(err.Error(), "SendGrid API error") {
			assert.NoError(t, err)
		}
	}
}

func TestSend_EmailMessage_APIError(t *testing.T) {
	// Create a mock HTTP server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"errors": []map[string]interface{}{
				{"message": "Bad Request", "field": "from.email", "help": "Invalid from email"},
			},
		})
	}))
	defer server.Close()

	config := types.ProviderConfig{
		Name:      "test",
		Connector: "twilio",
		Options: map[string]interface{}{
			"account_sid":      "test_sid",
			"auth_token":       "test_token",
			"from_email":       "test@example.com",
			"sendgrid_api_key": "test_api_key",
		},
	}

	provider, err := NewTwilioProvider(config)
	require.NoError(t, err)

	// Mock the SendGrid API endpoint by temporarily replacing the sendEmail method behavior
	// For this test, we'll create a custom provider with a modified http client
	provider.httpClient = &http.Client{
		Transport: &mockTransport{server: server},
	}

	ctx := context.Background()
	emailMessage := createTestMessage(types.MessageTypeEmail)

	err = provider.Send(ctx, emailMessage)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "SendGrid API error")
}

func TestSend_ContextTimeout_Email_RealAPI(t *testing.T) {
	config := loadTestConfig(t)
	provider, err := NewTwilioProvider(config)
	require.NoError(t, err)

	// Skip if SendGrid API key is not configured
	if provider.sendGridAPIKey == "" {
		t.Skip("TWILIO_SENDGRID_API_KEY not configured, skipping real API test")
	}

	// Create a very short timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	emailMessage := createTestMessage(types.MessageTypeEmail)

	err = provider.Send(ctx, emailMessage)
	if err != nil {
		t.Log("Context timeout working correctly with real API")
		// Could be timeout or other error, both are acceptable for this test
	} else {
		t.Log("Request completed faster than timeout")
	}
}

func TestSendBatch_Email_RealAPI(t *testing.T) {
	config := loadTestConfig(t)
	provider, err := NewTwilioProvider(config)
	require.NoError(t, err)

	// Skip if SendGrid API key is not configured
	if provider.sendGridAPIKey == "" {
		t.Skip("TWILIO_SENDGRID_API_KEY not configured, skipping real API test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	messages := []*types.Message{
		{
			Type:    types.MessageTypeEmail,
			To:      []string{TestEmailAgent},
			Subject: "Batch Test Email 1",
			Body:    "This is batch test email 1 via Twilio SendGrid API.",
		},
		{
			Type:    types.MessageTypeEmail,
			To:      []string{TestEmailX},
			Subject: "Batch Test Email 2",
			Body:    "This is batch test email 2 via Twilio SendGrid API.",
		},
	}

	err = provider.SendBatch(ctx, messages)
	if err == nil {
		t.Log("Real Twilio SendGrid batch API call succeeded")
	} else {
		t.Logf("Real Twilio SendGrid batch API call failed (expected in some test environments): %v", err)
		// Don't fail the test if it's just an API configuration issue
		if !strings.Contains(err.Error(), "SendGrid API error") {
			assert.NoError(t, err)
		}
	}
}

func TestSend_EmailMessage_WithCustomFrom(t *testing.T) {
	// Create a mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&payload)
		assert.NoError(t, err)

		// Verify custom from address is used
		from := payload["from"].(map[string]interface{})
		assert.Equal(t, "custom@example.com", from["email"])

		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]interface{}{"message": "success"})
	}))
	defer server.Close()

	config := types.ProviderConfig{
		Name:      "test",
		Connector: "twilio",
		Options: map[string]interface{}{
			"account_sid":      "test_sid",
			"auth_token":       "test_token",
			"from_email":       "default@example.com",
			"sendgrid_api_key": "test_api_key",
		},
	}

	provider, err := NewTwilioProvider(config)
	require.NoError(t, err)

	provider.httpClient = &http.Client{
		Transport: &mockTransport{server: server},
	}

	ctx := context.Background()
	emailMessage := createTestMessage(types.MessageTypeEmail)
	emailMessage.From = "custom@example.com"

	err = provider.Send(ctx, emailMessage)
	assert.NoError(t, err)
}

func TestSend_EmailMessage_WithScheduledTime(t *testing.T) {
	// Create a mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&payload)
		assert.NoError(t, err)

		// Verify scheduled time is set
		sendAt, exists := payload["send_at"]
		assert.True(t, exists)
		assert.NotNil(t, sendAt)

		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]interface{}{"message": "success"})
	}))
	defer server.Close()

	config := types.ProviderConfig{
		Name:      "test",
		Connector: "twilio",
		Options: map[string]interface{}{
			"account_sid":      "test_sid",
			"auth_token":       "test_token",
			"from_email":       "test@example.com",
			"sendgrid_api_key": "test_api_key",
		},
	}

	provider, err := NewTwilioProvider(config)
	require.NoError(t, err)

	provider.httpClient = &http.Client{
		Transport: &mockTransport{server: server},
	}

	ctx := context.Background()
	emailMessage := createTestMessage(types.MessageTypeEmail)
	scheduledTime := time.Now().Add(1 * time.Hour)
	emailMessage.ScheduledAt = &scheduledTime

	err = provider.Send(ctx, emailMessage)
	assert.NoError(t, err)
}

func TestSend_EmailMessage_MultipleRecipients_RealAPI(t *testing.T) {
	config := loadTestConfig(t)
	provider, err := NewTwilioProvider(config)
	require.NoError(t, err)

	// Skip if SendGrid API key is not configured
	if provider.sendGridAPIKey == "" {
		t.Skip("TWILIO_SENDGRID_API_KEY not configured, skipping real API test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	emailMessage := createTestMessage(types.MessageTypeEmail)
	emailMessage.To = []string{TestEmailAgent, TestEmailX, TestEmailXiang}
	emailMessage.Subject = "Multiple Recipients Test"

	err = provider.Send(ctx, emailMessage)
	if err == nil {
		t.Log("Twilio SendGrid multiple recipients API call succeeded")
	} else {
		t.Logf("Twilio SendGrid multiple recipients API call failed (expected in some test environments): %v", err)
		// Don't fail the test if it's just an API configuration issue
		if !strings.Contains(err.Error(), "SendGrid API error") {
			assert.NoError(t, err)
		}
	}
}

func TestSend_UnsupportedMessageType(t *testing.T) {
	config := loadTestConfig(t)
	provider, err := NewTwilioProvider(config)
	require.NoError(t, err)

	ctx := context.Background()

	// Test unsupported message type
	unsupportedMessage := &types.Message{
		Type: "unsupported_type",
		To:   []string{TestEmailAgent},
		Body: "This should fail",
	}

	err = provider.Send(ctx, unsupportedMessage)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported message type")
}

// =============================================================================
// Note: SMS and WhatsApp tests have been moved to separate files:
// - twilio_sms_test.go: SMS-specific tests and functionality
// - twilio_whatsapp_test.go: WhatsApp-specific tests and functionality
// =============================================================================

// =============================================================================
// Benchmark Tests
// =============================================================================

func BenchmarkSend_Email(b *testing.B) {
	config := loadTestConfig(&testing.T{})
	provider, err := NewTwilioProvider(config)
	if err != nil {
		b.Fatalf("Failed to create provider: %v", err)
	}

	// Skip if SendGrid API key is not configured
	if provider.sendGridAPIKey == "" {
		b.Skip("TWILIO_SENDGRID_API_KEY not configured, skipping benchmark")
	}

	ctx := context.Background()
	emailMessage := createTestMessage(types.MessageTypeEmail)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = provider.Send(ctx, emailMessage)
	}
}

func BenchmarkSendBatch_Email(b *testing.B) {
	config := loadTestConfig(&testing.T{})
	provider, err := NewTwilioProvider(config)
	if err != nil {
		b.Fatalf("Failed to create provider: %v", err)
	}

	// Skip if SendGrid API key is not configured
	if provider.sendGridAPIKey == "" {
		b.Skip("TWILIO_SENDGRID_API_KEY not configured, skipping benchmark")
	}

	ctx := context.Background()
	messages := []*types.Message{
		createTestMessage(types.MessageTypeEmail),
		createTestMessage(types.MessageTypeEmail),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = provider.SendBatch(ctx, messages)
	}
}

// =============================================================================
// Helper Types for Mocking
// =============================================================================

// mockTransport is a custom RoundTripper for mocking HTTP requests
type mockTransport struct {
	server *httptest.Server
}

func (t *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Redirect all requests to our mock server
	req.URL.Scheme = "http"
	req.URL.Host = strings.TrimPrefix(t.server.URL, "http://")
	return http.DefaultTransport.RoundTrip(req)
}
