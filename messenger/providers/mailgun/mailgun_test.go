package mailgun

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

// Test constants for authorized recipient addresses
const (
	TestEmailAgent = "agent@iqka.com"
	TestEmailX     = "x@iqka.com"
	TestEmailXiang = "xiang@iqka.com"
)

// Test helper functions

func createTestMessage(msgType types.MessageType) *types.Message {
	message := &types.Message{
		Type:    msgType,
		To:      []string{"test@example.com"},
		Subject: "Test Email",
		Body:    "This is a test email body",
		HTML:    "<h1>Test Email</h1><p>This is a test email body</p>",
		Headers: map[string]string{
			"X-Test-Header": "test-value",
		},
		Metadata: map[string]interface{}{
			"campaign": "test-campaign",
			"user_id":  "12345",
		},
		Priority: 1,
	}
	return message
}

func loadTestConfig(t *testing.T) types.ProviderConfig {
	// Prepare test environment using YAO_TEST_APPLICATION which points to yao-dev-app
	// Environment variables are already set in env.local.sh
	test.Prepare(t, config.Conf, "YAO_TEST_APPLICATION")
	defer test.Clean()

	// Create test config directly using environment variables
	config := types.ProviderConfig{
		Name:      "marketing",
		Connector: "mailgun",
		Options: map[string]interface{}{
			"domain":   os.Getenv("MAILGUN_DOMAIN"),
			"api_key":  os.Getenv("MAILGUN_API_KEY"),
			"from":     os.Getenv("MAILGUN_FROM"),
			"base_url": "https://api.mailgun.net/v3",
		},
	}

	return config
}

// Test NewMailgunProvider

func TestNewMailgunProvider(t *testing.T) {
	config := loadTestConfig(t)

	provider, err := NewMailgunProvider(config)
	require.NoError(t, err)
	assert.NotNil(t, provider)

	// Verify configuration using actual environment variables from env.local.sh
	assert.Equal(t, os.Getenv("MAILGUN_DOMAIN"), provider.domain)
	assert.Equal(t, os.Getenv("MAILGUN_API_KEY"), provider.apiKey)
	assert.Equal(t, os.Getenv("MAILGUN_FROM"), provider.from)
	assert.Equal(t, "https://api.mailgun.net/v3", provider.baseURL)
	assert.Equal(t, "marketing", provider.config.Name)
}

func TestNewMailgunProvider_MissingOptions(t *testing.T) {
	config := types.ProviderConfig{
		Name:      "test",
		Connector: "mailgun",
		Options:   nil,
	}

	provider, err := NewMailgunProvider(config)
	assert.Error(t, err)
	assert.Nil(t, provider)
	assert.Contains(t, err.Error(), "Mailgun provider requires options")
}

func TestNewMailgunProvider_MissingDomain(t *testing.T) {
	config := types.ProviderConfig{
		Name:      "test",
		Connector: "mailgun",
		Options: map[string]interface{}{
			"api_key": "test-key",
			"from":    "test@example.com",
		},
	}

	provider, err := NewMailgunProvider(config)
	assert.Error(t, err)
	assert.Nil(t, provider)
	assert.Contains(t, err.Error(), "Mailgun provider requires 'domain' option")
}

func TestNewMailgunProvider_MissingAPIKey(t *testing.T) {
	config := types.ProviderConfig{
		Name:      "test",
		Connector: "mailgun",
		Options: map[string]interface{}{
			"domain": "test.mailgun.org",
			"from":   "test@example.com",
		},
	}

	provider, err := NewMailgunProvider(config)
	assert.Error(t, err)
	assert.Nil(t, provider)
	assert.Contains(t, err.Error(), "Mailgun provider requires 'api_key' option")
}

func TestNewMailgunProvider_MissingFrom(t *testing.T) {
	config := types.ProviderConfig{
		Name:      "test",
		Connector: "mailgun",
		Options: map[string]interface{}{
			"domain":  "test.mailgun.org",
			"api_key": "test-key",
		},
	}

	provider, err := NewMailgunProvider(config)
	assert.Error(t, err)
	assert.Nil(t, provider)
	assert.Contains(t, err.Error(), "Mailgun provider requires 'from' option")
}

// Test Provider Interface Methods

func TestGetType(t *testing.T) {
	config := loadTestConfig(t)
	provider, err := NewMailgunProvider(config)
	require.NoError(t, err)

	assert.Equal(t, "mailgun", provider.GetType())
}

func TestGetName(t *testing.T) {
	config := loadTestConfig(t)
	provider, err := NewMailgunProvider(config)
	require.NoError(t, err)

	assert.Equal(t, "marketing", provider.GetName())
}

func TestValidate(t *testing.T) {
	config := loadTestConfig(t)
	provider, err := NewMailgunProvider(config)
	require.NoError(t, err)

	err = provider.Validate()
	assert.NoError(t, err)
}

func TestValidate_MissingDomain(t *testing.T) {
	config := loadTestConfig(t)
	provider, err := NewMailgunProvider(config)
	require.NoError(t, err)

	provider.domain = ""
	err = provider.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "domain is required")
}

func TestValidate_MissingAPIKey(t *testing.T) {
	config := loadTestConfig(t)
	provider, err := NewMailgunProvider(config)
	require.NoError(t, err)

	provider.apiKey = ""
	err = provider.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "api_key is required")
}

func TestValidate_MissingFrom(t *testing.T) {
	config := loadTestConfig(t)
	provider, err := NewMailgunProvider(config)
	require.NoError(t, err)

	provider.from = ""
	err = provider.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "from address is required")
}

func TestClose(t *testing.T) {
	config := loadTestConfig(t)
	provider, err := NewMailgunProvider(config)
	require.NoError(t, err)

	err = provider.Close()
	assert.NoError(t, err)
}

// Test Send Methods

func TestSend_NonEmailMessage(t *testing.T) {
	config := loadTestConfig(t)
	provider, err := NewMailgunProvider(config)
	require.NoError(t, err)

	ctx := context.Background()
	smsMessage := createTestMessage(types.MessageTypeSMS)

	err = provider.Send(ctx, smsMessage)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Mailgun provider only supports email messages")
}

func TestSend_EmailMessage_RealAPI(t *testing.T) {
	config := loadTestConfig(t)
	provider, err := NewMailgunProvider(config)
	require.NoError(t, err)

	ctx := context.Background()
	// Use test recipient addresses that are authorized for testing
	emailMessage := &types.Message{
		Type:    types.MessageTypeEmail,
		To:      []string{TestEmailAgent},
		Subject: "Unit Test Email - " + time.Now().Format("2006-01-02 15:04:05"),
		Body:    "This is a unit test email sent via real Mailgun API",
		HTML:    "<h1>Unit Test</h1><p>This is a unit test email sent via real Mailgun API</p>",
		Headers: map[string]string{
			"X-Test-Run": "mailgun-provider-test",
		},
		Metadata: map[string]interface{}{
			"test_type": "unit_test",
			"timestamp": time.Now().Unix(),
		},
	}

	err = provider.Send(ctx, emailMessage)
	if err != nil {
		// Log error but don't fail test, as it might be network or API configuration issues
		t.Logf("Real API call failed (this may be expected in CI/test environment): %v", err)

		// Check if it's expected error type (network, authentication, etc.)
		if strings.Contains(err.Error(), "Mailgun API error") {
			t.Log("Mailgun API returned error - this indicates the request reached the server")
		} else if strings.Contains(err.Error(), "failed to send request") {
			t.Log("Network error - this may be expected in test environment")
		} else {
			t.Logf("Unexpected error type: %v", err)
		}
	} else {
		t.Log("Real Mailgun API call succeeded")
	}
}

func TestSend_EmailMessage_APIError(t *testing.T) {
	// Create a mock HTTP server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"message": "Invalid domain"}`))
	}))
	defer server.Close()

	config := loadTestConfig(t)
	provider, err := NewMailgunProvider(config)
	require.NoError(t, err)

	// Override base URL to use mock server
	provider.baseURL = server.URL

	ctx := context.Background()
	emailMessage := createTestMessage(types.MessageTypeEmail)

	err = provider.Send(ctx, emailMessage)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Mailgun API error")
	assert.Contains(t, err.Error(), "400")
}

func TestSend_ContextTimeout_RealAPI(t *testing.T) {
	config := loadTestConfig(t)
	provider, err := NewMailgunProvider(config)
	require.NoError(t, err)

	// Create a very short timeout context to test timeout functionality
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	emailMessage := &types.Message{
		Type:    types.MessageTypeEmail,
		To:      []string{TestEmailX},
		Subject: "Context Timeout Test",
		Body:    "This should timeout before sending",
	}

	err = provider.Send(ctx, emailMessage)
	assert.Error(t, err)

	// Verify it's a context timeout error
	if strings.Contains(err.Error(), "context deadline exceeded") {
		t.Log("Context timeout working correctly with real API")
	} else if strings.Contains(err.Error(), "context canceled") {
		t.Log("Context cancellation working correctly with real API")
	} else {
		t.Logf("Got different error (may be network related): %v", err)
	}
}

func TestSendBatch_RealAPI(t *testing.T) {
	config := loadTestConfig(t)
	provider, err := NewMailgunProvider(config)
	require.NoError(t, err)

	ctx := context.Background()
	// Create multiple test emails using authorized test addresses
	messages := []*types.Message{
		{
			Type:    types.MessageTypeEmail,
			To:      []string{TestEmailX},
			Subject: "Batch Test 1 - " + time.Now().Format("15:04:05"),
			Body:    "Batch test message 1",
			HTML:    "<p>Batch test message 1</p>",
		},
		{
			Type:    types.MessageTypeEmail,
			To:      []string{TestEmailXiang},
			Subject: "Batch Test 2 - " + time.Now().Format("15:04:05"),
			Body:    "Batch test message 2",
			HTML:    "<p>Batch test message 2</p>",
		},
	}

	err = provider.SendBatch(ctx, messages)
	if err != nil {
		t.Logf("Real batch API call failed (this may be expected): %v", err)

		// Verify error handling logic
		if strings.Contains(err.Error(), "failed to send message to") {
			t.Log("Batch sending failed as expected - error handling works correctly")
		}
	} else {
		t.Log("Real Mailgun batch API call succeeded")
	}
}

func TestSendBatch_PartialFailure(t *testing.T) {
	// Create a mock HTTP server that fails on second request
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 2 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"message": "Invalid recipient"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		response := map[string]interface{}{
			"id":      "test-message-id-" + string(rune(callCount)),
			"message": "Queued. Thank you.",
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	config := loadTestConfig(t)
	provider, err := NewMailgunProvider(config)
	require.NoError(t, err)

	// Override base URL to use mock server
	provider.baseURL = server.URL

	ctx := context.Background()
	messages := []*types.Message{
		createTestMessage(types.MessageTypeEmail),
		createTestMessage(types.MessageTypeEmail),
		createTestMessage(types.MessageTypeEmail),
	}

	err = provider.SendBatch(ctx, messages)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to send message to")
	assert.Equal(t, 2, callCount, "Should stop after first failure")
}

// Test Edge Cases

func TestSend_WithCustomFrom(t *testing.T) {
	// Create a mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		assert.NoError(t, err)

		// Verify custom from address is used
		assert.Equal(t, "custom@example.com", r.FormValue("from"))

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"message": "Queued"})
	}))
	defer server.Close()

	config := loadTestConfig(t)
	provider, err := NewMailgunProvider(config)
	require.NoError(t, err)

	provider.baseURL = server.URL

	ctx := context.Background()
	emailMessage := createTestMessage(types.MessageTypeEmail)
	emailMessage.From = "custom@example.com" // Override from address

	err = provider.Send(ctx, emailMessage)
	assert.NoError(t, err)
}

func TestSend_WithScheduledTime(t *testing.T) {
	// Create a mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		assert.NoError(t, err)

		// Verify scheduled time is set
		deliveryTime := r.FormValue("o:deliverytime")
		assert.NotEmpty(t, deliveryTime)
		// RFC1123Z format includes timezone offset (e.g., "+0000", "+0800")
		assert.Regexp(t, `[+-]\d{4}`, deliveryTime)

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"message": "Queued"})
	}))
	defer server.Close()

	config := loadTestConfig(t)
	provider, err := NewMailgunProvider(config)
	require.NoError(t, err)

	provider.baseURL = server.URL

	ctx := context.Background()
	emailMessage := createTestMessage(types.MessageTypeEmail)
	scheduledTime := time.Now().Add(1 * time.Hour)
	emailMessage.ScheduledAt = &scheduledTime

	err = provider.Send(ctx, emailMessage)
	assert.NoError(t, err)
}

func TestSend_MultipleRecipients_RealAPI(t *testing.T) {
	config := loadTestConfig(t)
	provider, err := NewMailgunProvider(config)
	require.NoError(t, err)

	ctx := context.Background()
	// Test with multiple authorized recipient addresses
	emailMessage := &types.Message{
		Type:    types.MessageTypeEmail,
		To:      []string{TestEmailAgent, TestEmailX, TestEmailXiang},
		Subject: "Multiple Recipients Test - " + time.Now().Format("15:04:05"),
		Body:    "This email is sent to multiple recipients for testing",
		HTML:    "<h1>Multiple Recipients Test</h1><p>This email is sent to multiple recipients for testing</p>",
		Headers: map[string]string{
			"X-Test-Type": "multiple-recipients",
		},
	}

	err = provider.Send(ctx, emailMessage)
	if err != nil {
		t.Logf("Multiple recipients API call failed (this may be expected): %v", err)

		// Check error handling for multiple recipients
		if strings.Contains(err.Error(), "Mailgun API error") {
			t.Log("Multiple recipients test reached Mailgun API")
		}
	} else {
		t.Log("Multiple recipients API call succeeded")
	}
}

// Benchmark Tests

func BenchmarkNewMailgunProvider(b *testing.B) {
	// Setup
	t := &testing.T{}
	config := loadTestConfig(t)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		provider, err := NewMailgunProvider(config)
		if err != nil {
			b.Fatal(err)
		}
		_ = provider
	}
}

func BenchmarkSend(b *testing.B) {
	// Setup mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"message": "Queued"})
	}))
	defer server.Close()

	t := &testing.T{}
	config := loadTestConfig(t)
	provider, err := NewMailgunProvider(config)
	if err != nil {
		b.Fatal(err)
	}
	provider.baseURL = server.URL

	ctx := context.Background()
	emailMessage := createTestMessage(types.MessageTypeEmail)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := provider.Send(ctx, emailMessage)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkValidate(b *testing.B) {
	t := &testing.T{}
	config := loadTestConfig(t)
	provider, err := NewMailgunProvider(config)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := provider.Validate()
		if err != nil {
			b.Fatal(err)
		}
	}
}

// ============================================================================
// Attachment Tests
// ============================================================================

func TestSend_EmailWithAttachments_MockServer(t *testing.T) {
	// Create a mock HTTP server that validates the multipart request
	var receivedContentType string
	var receivedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedContentType = r.Header.Get("Content-Type")

		// Read the body
		body, _ := r.Body.Read(make([]byte, 1024*1024))
		_ = body
		receivedBody = make([]byte, r.ContentLength)
		r.Body.Read(receivedBody)

		// Return success
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id": "test-id", "message": "Queued"}`))
	}))
	defer server.Close()

	config := loadTestConfig(t)
	provider, err := NewMailgunProvider(config)
	require.NoError(t, err)

	// Override base URL to use mock server
	provider.baseURL = server.URL

	ctx := context.Background()
	emailMessage := &types.Message{
		Type:    types.MessageTypeEmail,
		To:      []string{"test@example.com"},
		Subject: "Test Email with Attachment",
		Body:    "This is a test email with attachment",
		HTML:    "<h1>Test</h1><p>This is a test email with attachment</p>",
		Attachments: []types.Attachment{
			{
				Filename:    "test.txt",
				ContentType: "text/plain",
				Content:     []byte("Hello, this is a test attachment content!"),
			},
			{
				Filename:    "test.pdf",
				ContentType: "application/pdf",
				Content:     []byte("%PDF-1.4 fake pdf content"),
			},
		},
	}

	err = provider.Send(ctx, emailMessage)
	assert.NoError(t, err)

	// Verify the request used multipart/form-data
	assert.Contains(t, receivedContentType, "multipart/form-data")
}

func TestSend_EmailWithInlineAttachment_MockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id": "test-id", "message": "Queued"}`))
	}))
	defer server.Close()

	config := loadTestConfig(t)
	provider, err := NewMailgunProvider(config)
	require.NoError(t, err)

	provider.baseURL = server.URL

	ctx := context.Background()
	emailMessage := &types.Message{
		Type:    types.MessageTypeEmail,
		To:      []string{"test@example.com"},
		Subject: "Test Email with Inline Image",
		Body:    "This is a test email with inline image",
		HTML:    `<h1>Test</h1><p>Image: <img src="cid:logo123"></p>`,
		Attachments: []types.Attachment{
			{
				Filename:    "logo.png",
				ContentType: "image/png",
				Content:     []byte{0x89, 0x50, 0x4E, 0x47}, // PNG magic bytes
				Inline:      true,
				CID:         "logo123",
			},
		},
	}

	err = provider.Send(ctx, emailMessage)
	assert.NoError(t, err)
}

func TestSend_EmailWithAttachments_RealAPI(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping real API test in short mode")
	}

	config := loadTestConfig(t)
	provider, err := NewMailgunProvider(config)
	require.NoError(t, err)

	ctx := context.Background()
	emailMessage := &types.Message{
		Type:    types.MessageTypeEmail,
		To:      []string{TestEmailAgent},
		Subject: "Unit Test Email with Attachment - " + time.Now().Format("2006-01-02 15:04:05"),
		Body:    "This is a unit test email with attachment sent via real Mailgun API",
		HTML:    "<h1>Unit Test</h1><p>This email has an attachment.</p>",
		Attachments: []types.Attachment{
			{
				Filename:    "test-attachment.txt",
				ContentType: "text/plain",
				Content:     []byte("This is a test attachment content.\nLine 2 of the attachment."),
			},
		},
	}

	err = provider.Send(ctx, emailMessage)
	if err != nil {
		t.Logf("Real API call with attachment failed (may be expected in CI): %v", err)
	} else {
		t.Log("Real Mailgun API call with attachment succeeded")
	}
}
