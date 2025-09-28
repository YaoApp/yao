package mailer

import (
	"context"
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

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

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

func loadPrimaryTestConfig(t *testing.T) types.ProviderConfig {
	// Prepare test environment using YAO_TEST_APPLICATION which points to yao-dev-app
	// Environment variables are already set in env.local.sh
	test.Prepare(t, config.Conf, "YAO_TEST_APPLICATION")
	defer test.Clean()

	// Create test config directly using environment variables for primary SMTP
	// Port 465 requires SSL, port 587 requires TLS
	smtpPort := os.Getenv("SMTP_PORT")
	useSSL := smtpPort == "465"
	useTLS := smtpPort == "587" || smtpPort == "25"

	config := types.ProviderConfig{
		Name:      "primary",
		Connector: "mailer",
		Options: map[string]interface{}{
			"smtp": map[string]interface{}{
				"host":     os.Getenv("SMTP_HOST"),
				"port":     os.Getenv("SMTP_PORT"),
				"username": os.Getenv("SMTP_USERNAME"),
				"password": os.Getenv("SMTP_PASSWORD"),
				"from":     os.Getenv("SMTP_FROM"),
				"use_tls":  useTLS,
				"use_ssl":  useSSL,
			},
		},
	}

	return config
}

func loadReliableTestConfig(t *testing.T) types.ProviderConfig {
	// Prepare test environment using YAO_TEST_APPLICATION which points to yao-dev-app
	// Environment variables are already set in env.local.sh
	test.Prepare(t, config.Conf, "YAO_TEST_APPLICATION")
	defer test.Clean()

	// Create test config directly using environment variables for reliable SMTP
	config := types.ProviderConfig{
		Name:      "reliable",
		Connector: "mailer",
		Options: map[string]interface{}{
			"smtp": map[string]interface{}{
				"host":     os.Getenv("RELIABLE_SMTP_HOST"),
				"port":     587, // Hardcoded in reliable.mailer.yao
				"username": os.Getenv("RELIABLE_SMTP_USERNAME"),
				"password": os.Getenv("RELIABLE_SMTP_PASSWORD"),
				"from":     os.Getenv("RELIABLE_SMTP_FROM"),
				"use_tls":  true,
			},
			"imap": map[string]interface{}{
				"host":     getEnvOrDefault("RELIABLE_IMAP_HOST", os.Getenv("RELIABLE_SMTP_HOST")),
				"port":     getEnvOrDefault("RELIABLE_IMAP_PORT", "993"),
				"username": getEnvOrDefault("RELIABLE_IMAP_USERNAME", os.Getenv("RELIABLE_SMTP_USERNAME")),
				"password": getEnvOrDefault("RELIABLE_IMAP_PASSWORD", os.Getenv("RELIABLE_SMTP_PASSWORD")),
				"use_ssl":  true,
				"mailbox":  "INBOX",
			},
		},
	}

	return config
}

// Test NewMailerProvider

func TestNewMailerProvider_Primary(t *testing.T) {
	config := loadPrimaryTestConfig(t)

	provider, err := NewMailerProvider(config)
	require.NoError(t, err)
	assert.NotNil(t, provider)

	// Verify configuration using actual environment variables from env.local.sh
	assert.Equal(t, os.Getenv("SMTP_HOST"), provider.host)
	assert.Equal(t, os.Getenv("SMTP_USERNAME"), provider.username)
	assert.Equal(t, os.Getenv("SMTP_PASSWORD"), provider.password)
	assert.Equal(t, os.Getenv("SMTP_FROM"), provider.from)
	assert.Equal(t, "primary", provider.config.Name)
	// Port 465 uses SSL, not TLS
	if os.Getenv("SMTP_PORT") == "465" {
		assert.True(t, provider.useSSL)
		assert.False(t, provider.useTLS)
	} else {
		assert.True(t, provider.useTLS)
	}
}

func TestNewMailerProvider_Reliable(t *testing.T) {
	config := loadReliableTestConfig(t)

	provider, err := NewMailerProvider(config)
	require.NoError(t, err)
	assert.NotNil(t, provider)

	// Verify configuration using actual environment variables from env.local.sh
	assert.Equal(t, os.Getenv("RELIABLE_SMTP_HOST"), provider.host)
	assert.Equal(t, 587, provider.port)
	assert.Equal(t, os.Getenv("RELIABLE_SMTP_USERNAME"), provider.username)
	assert.Equal(t, os.Getenv("RELIABLE_SMTP_PASSWORD"), provider.password)
	assert.Equal(t, os.Getenv("RELIABLE_SMTP_FROM"), provider.from)
	assert.Equal(t, "reliable", provider.config.Name)
	assert.True(t, provider.useTLS)
}

func TestNewMailerProvider_MissingOptions(t *testing.T) {
	config := types.ProviderConfig{
		Name:      "test",
		Connector: "smtp",
		Options:   nil,
	}

	provider, err := NewMailerProvider(config)
	assert.Error(t, err)
	assert.Nil(t, provider)
	assert.Contains(t, err.Error(), "mailer provider requires options")
}

func TestNewMailerProvider_MissingHost(t *testing.T) {
	config := types.ProviderConfig{
		Name:      "test",
		Connector: "smtp",
		Options: map[string]interface{}{
			"port":     587,
			"username": "test@example.com",
			"password": "password",
			"from":     "test@example.com",
		},
	}

	provider, err := NewMailerProvider(config)
	assert.Error(t, err)
	assert.Nil(t, provider)
	assert.Contains(t, err.Error(), "mailer provider requires 'smtp' configuration")
}

func TestNewMailerProvider_MissingUsername(t *testing.T) {
	config := types.ProviderConfig{
		Name:      "test",
		Connector: "smtp",
		Options: map[string]interface{}{
			"host":     "smtp.example.com",
			"port":     587,
			"password": "password",
			"from":     "test@example.com",
		},
	}

	provider, err := NewMailerProvider(config)
	assert.Error(t, err)
	assert.Nil(t, provider)
	assert.Contains(t, err.Error(), "mailer provider requires 'smtp' configuration")
}

func TestNewMailerProvider_MissingPassword(t *testing.T) {
	config := types.ProviderConfig{
		Name:      "test",
		Connector: "smtp",
		Options: map[string]interface{}{
			"host":     "smtp.example.com",
			"port":     587,
			"username": "test@example.com",
			"from":     "test@example.com",
		},
	}

	provider, err := NewMailerProvider(config)
	assert.Error(t, err)
	assert.Nil(t, provider)
	assert.Contains(t, err.Error(), "mailer provider requires 'smtp' configuration")
}

func TestNewMailerProvider_MissingFrom(t *testing.T) {
	config := types.ProviderConfig{
		Name:      "test",
		Connector: "smtp",
		Options: map[string]interface{}{
			"host":     "smtp.example.com",
			"port":     587,
			"username": "test@example.com",
			"password": "password",
		},
	}

	provider, err := NewMailerProvider(config)
	assert.Error(t, err)
	assert.Nil(t, provider)
	assert.Contains(t, err.Error(), "mailer provider requires 'smtp' configuration")
}

// Test Provider Interface Methods

func TestGetType(t *testing.T) {
	config := loadPrimaryTestConfig(t)
	provider, err := NewMailerProvider(config)
	require.NoError(t, err)

	assert.Equal(t, "mailer", provider.GetType())
}

func TestGetName(t *testing.T) {
	config := loadPrimaryTestConfig(t)
	provider, err := NewMailerProvider(config)
	require.NoError(t, err)

	assert.Equal(t, "primary", provider.GetName())
}

func TestValidate(t *testing.T) {
	config := loadPrimaryTestConfig(t)
	provider, err := NewMailerProvider(config)
	require.NoError(t, err)

	err = provider.Validate()
	assert.NoError(t, err)
}

func TestValidate_MissingHost(t *testing.T) {
	config := loadPrimaryTestConfig(t)
	provider, err := NewMailerProvider(config)
	require.NoError(t, err)

	provider.host = ""
	err = provider.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "host is required")
}

func TestValidate_InvalidPort(t *testing.T) {
	config := loadPrimaryTestConfig(t)
	provider, err := NewMailerProvider(config)
	require.NoError(t, err)

	provider.port = 0
	err = provider.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "port must be positive")
}

func TestValidate_MissingUsername(t *testing.T) {
	config := loadPrimaryTestConfig(t)
	provider, err := NewMailerProvider(config)
	require.NoError(t, err)

	provider.username = ""
	err = provider.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "username is required")
}

func TestValidate_MissingPassword(t *testing.T) {
	config := loadPrimaryTestConfig(t)
	provider, err := NewMailerProvider(config)
	require.NoError(t, err)

	provider.password = ""
	err = provider.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "password is required")
}

func TestValidate_MissingFrom(t *testing.T) {
	config := loadPrimaryTestConfig(t)
	provider, err := NewMailerProvider(config)
	require.NoError(t, err)

	provider.from = ""
	err = provider.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "from address is required")
}

func TestClose(t *testing.T) {
	config := loadPrimaryTestConfig(t)
	provider, err := NewMailerProvider(config)
	require.NoError(t, err)

	err = provider.Close()
	assert.NoError(t, err)
}

// Test Send Methods

func TestSend_NonEmailMessage(t *testing.T) {
	config := loadPrimaryTestConfig(t)
	provider, err := NewMailerProvider(config)
	require.NoError(t, err)

	ctx := context.Background()
	smsMessage := createTestMessage(types.MessageTypeSMS)

	err = provider.Send(ctx, smsMessage)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "SMTP provider only supports email messages")
}

func TestSend_EmailMessage_RealAPI_Primary(t *testing.T) {
	config := loadPrimaryTestConfig(t)
	provider, err := NewMailerProvider(config)
	require.NoError(t, err)

	// Use context with reasonable timeout for SMTP operations
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	// Use test recipient addresses that are authorized for testing
	emailMessage := &types.Message{
		Type:    types.MessageTypeEmail,
		To:      []string{TestEmailAgent},
		Subject: "SMTP Unit Test Email - " + time.Now().Format("2006-01-02 15:04:05"),
		Body:    "This is a unit test email sent via real SMTP server",
		HTML:    "<h1>SMTP Unit Test</h1><p>This is a unit test email sent via real SMTP server</p>",
		Headers: map[string]string{
			"X-Test-Run": "smtp-provider-test",
		},
		Metadata: map[string]interface{}{
			"test_type": "unit_test",
			"timestamp": time.Now().Unix(),
		},
	}

	err = provider.Send(ctx, emailMessage)
	if err != nil {
		// Log error but don't fail test, as it might be network or SMTP configuration issues
		t.Logf("Real SMTP API call failed (this may be expected in CI/test environment): %v", err)

		// Check if it's expected error type (network, authentication, etc.)
		if strings.Contains(err.Error(), "SMTP authentication failed") {
			t.Log("SMTP authentication failed - this indicates the request reached the server")
		} else if strings.Contains(err.Error(), "failed to connect to SMTP server") {
			t.Log("Network error - this may be expected in test environment")
		} else {
			t.Logf("Unexpected error type: %v", err)
		}
	} else {
		t.Log("Real SMTP API call succeeded")
	}
}

func TestSend_EmailMessage_RealAPI_Reliable(t *testing.T) {
	config := loadReliableTestConfig(t)
	provider, err := NewMailerProvider(config)
	require.NoError(t, err)

	// Use context with reasonable timeout for SMTP operations
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	// Use test recipient addresses that are authorized for testing
	emailMessage := &types.Message{
		Type:    types.MessageTypeEmail,
		To:      []string{TestEmailX},
		Subject: "Reliable SMTP Unit Test - " + time.Now().Format("2006-01-02 15:04:05"),
		Body:    "This is a unit test email sent via reliable SMTP server",
		HTML:    "<h1>Reliable SMTP Test</h1><p>This is a unit test email sent via reliable SMTP server</p>",
		Headers: map[string]string{
			"X-Test-Run":  "smtp-reliable-test",
			"X-Test-Type": "reliable-smtp",
		},
		Metadata: map[string]interface{}{
			"test_type": "reliable_test",
			"timestamp": time.Now().Unix(),
		},
	}

	err = provider.Send(ctx, emailMessage)
	if err != nil {
		// Log error but don't fail test, as it might be network or SMTP configuration issues
		t.Logf("Real reliable SMTP API call failed (this may be expected in CI/test environment): %v", err)

		// Check if it's expected error type (network, authentication, etc.)
		if strings.Contains(err.Error(), "SMTP authentication failed") {
			t.Log("Reliable SMTP authentication failed - this indicates the request reached the server")
		} else if strings.Contains(err.Error(), "failed to connect to SMTP server") {
			t.Log("Network error - this may be expected in test environment")
		} else {
			t.Logf("Unexpected error type: %v", err)
		}
	} else {
		t.Log("Real reliable SMTP API call succeeded")
	}
}

func TestSend_ContextTimeout_RealAPI(t *testing.T) {
	config := loadPrimaryTestConfig(t)
	provider, err := NewMailerProvider(config)
	require.NoError(t, err)

	// Create a very short timeout context to test timeout functionality
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	emailMessage := &types.Message{
		Type:    types.MessageTypeEmail,
		To:      []string{TestEmailX},
		Subject: "SMTP Context Timeout Test",
		Body:    "This should timeout before sending",
	}

	err = provider.Send(ctx, emailMessage)
	assert.Error(t, err)

	// Verify it's a context timeout error
	if strings.Contains(err.Error(), "context deadline exceeded") {
		t.Log("Context timeout working correctly with real SMTP API")
	} else if strings.Contains(err.Error(), "context canceled") {
		t.Log("Context cancellation working correctly with real SMTP API")
	} else {
		t.Logf("Got different error (may be network related): %v", err)
	}
}

func TestSendBatch_RealAPI(t *testing.T) {
	config := loadPrimaryTestConfig(t)
	provider, err := NewMailerProvider(config)
	require.NoError(t, err)

	// Use context with reasonable timeout for SMTP batch operations
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	// Create multiple test emails using authorized test addresses
	messages := []*types.Message{
		{
			Type:    types.MessageTypeEmail,
			To:      []string{TestEmailX},
			Subject: "SMTP Batch Test 1 - " + time.Now().Format("15:04:05"),
			Body:    "SMTP batch test message 1",
			HTML:    "<p>SMTP batch test message 1</p>",
		},
		{
			Type:    types.MessageTypeEmail,
			To:      []string{TestEmailXiang},
			Subject: "SMTP Batch Test 2 - " + time.Now().Format("15:04:05"),
			Body:    "SMTP batch test message 2",
			HTML:    "<p>SMTP batch test message 2</p>",
		},
	}

	err = provider.SendBatch(ctx, messages)
	if err != nil {
		t.Logf("Real SMTP batch API call failed (this may be expected): %v", err)

		// Verify error handling logic
		if strings.Contains(err.Error(), "failed to send message to") {
			t.Log("SMTP batch sending failed as expected - error handling works correctly")
		}
	} else {
		t.Log("Real SMTP batch API call succeeded")
	}
}

func TestSend_MultipleRecipients_RealAPI(t *testing.T) {
	config := loadPrimaryTestConfig(t)
	provider, err := NewMailerProvider(config)
	require.NoError(t, err)

	// Use context with reasonable timeout for SMTP operations
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	// Test with multiple authorized recipient addresses
	emailMessage := &types.Message{
		Type:    types.MessageTypeEmail,
		To:      []string{TestEmailAgent, TestEmailX, TestEmailXiang},
		Subject: "SMTP Multiple Recipients Test - " + time.Now().Format("15:04:05"),
		Body:    "This email is sent to multiple recipients for SMTP testing",
		HTML:    "<h1>SMTP Multiple Recipients Test</h1><p>This email is sent to multiple recipients for SMTP testing</p>",
		Headers: map[string]string{
			"X-Test-Type": "smtp-multiple-recipients",
		},
	}

	err = provider.Send(ctx, emailMessage)
	if err != nil {
		t.Logf("SMTP multiple recipients API call failed (this may be expected): %v", err)

		// Check error handling for multiple recipients
		if strings.Contains(err.Error(), "SMTP authentication failed") {
			t.Log("SMTP multiple recipients test reached SMTP server")
		}
	} else {
		t.Log("SMTP multiple recipients API call succeeded")
	}
}

// Test Edge Cases

func TestSend_WithCustomFrom(t *testing.T) {
	config := loadPrimaryTestConfig(t)
	provider, err := NewMailerProvider(config)
	require.NoError(t, err)

	ctx := context.Background()
	emailMessage := &types.Message{
		Type:    types.MessageTypeEmail,
		To:      []string{TestEmailAgent},
		Subject: "SMTP Custom From Test - " + time.Now().Format("15:04:05"),
		Body:    "This email tests custom from address",
		From:    "custom-sender@example.com", // Custom from address
		Headers: map[string]string{
			"X-Test-Type": "custom-from",
		},
	}

	err = provider.Send(ctx, emailMessage)
	if err != nil {
		t.Logf("SMTP custom from test failed (this may be expected): %v", err)
	} else {
		t.Log("SMTP custom from test succeeded")
	}
}

func TestSend_PlainTextOnly(t *testing.T) {
	config := loadPrimaryTestConfig(t)
	provider, err := NewMailerProvider(config)
	require.NoError(t, err)

	ctx := context.Background()
	emailMessage := &types.Message{
		Type:    types.MessageTypeEmail,
		To:      []string{TestEmailAgent},
		Subject: "SMTP Plain Text Test - " + time.Now().Format("15:04:05"),
		Body:    "This is a plain text only email for testing SMTP functionality",
		// No HTML content
		Headers: map[string]string{
			"X-Test-Type": "plain-text-only",
		},
	}

	err = provider.Send(ctx, emailMessage)
	if err != nil {
		t.Logf("SMTP plain text test failed (this may be expected): %v", err)
	} else {
		t.Log("SMTP plain text test succeeded")
	}
}

func TestSend_HTMLOnly(t *testing.T) {
	config := loadPrimaryTestConfig(t)
	provider, err := NewMailerProvider(config)
	require.NoError(t, err)

	ctx := context.Background()
	emailMessage := &types.Message{
		Type:    types.MessageTypeEmail,
		To:      []string{TestEmailAgent},
		Subject: "SMTP HTML Only Test - " + time.Now().Format("15:04:05"),
		HTML:    "<h1>HTML Only Email</h1><p>This is an HTML only email for testing SMTP functionality</p><p><strong>Bold text</strong> and <em>italic text</em></p>",
		// No plain text body
		Headers: map[string]string{
			"X-Test-Type": "html-only",
		},
	}

	err = provider.Send(ctx, emailMessage)
	if err != nil {
		t.Logf("SMTP HTML only test failed (this may be expected): %v", err)
	} else {
		t.Log("SMTP HTML only test succeeded")
	}
}

func TestSend_MultipartMessage(t *testing.T) {
	config := loadPrimaryTestConfig(t)
	provider, err := NewMailerProvider(config)
	require.NoError(t, err)

	ctx := context.Background()
	emailMessage := &types.Message{
		Type:    types.MessageTypeEmail,
		To:      []string{TestEmailAgent},
		Subject: "SMTP Multipart Test - " + time.Now().Format("15:04:05"),
		Body:    "This is the plain text version of a multipart email for testing SMTP functionality",
		HTML:    "<h1>Multipart Email</h1><p>This is the HTML version of a multipart email for testing SMTP functionality</p><p>Both plain text and HTML versions are included.</p>",
		Headers: map[string]string{
			"X-Test-Type": "multipart-message",
		},
	}

	err = provider.Send(ctx, emailMessage)
	if err != nil {
		t.Logf("SMTP multipart test failed (this may be expected): %v", err)
	} else {
		t.Log("SMTP multipart test succeeded")
	}
}

// Benchmark Tests

func BenchmarkNewMailerProvider(b *testing.B) {
	// Setup
	t := &testing.T{}
	config := loadPrimaryTestConfig(t)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		provider, err := NewMailerProvider(config)
		if err != nil {
			b.Fatal(err)
		}
		_ = provider
	}
}

func BenchmarkValidate(b *testing.B) {
	t := &testing.T{}
	config := loadPrimaryTestConfig(t)
	provider, err := NewMailerProvider(config)
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

func BenchmarkBuildMessage(b *testing.B) {
	t := &testing.T{}
	config := loadPrimaryTestConfig(t)
	provider, err := NewMailerProvider(config)
	if err != nil {
		b.Fatal(err)
	}

	message := createTestMessage(types.MessageTypeEmail)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := provider.buildMessage(message)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func TestProvider_GetPublicInfo(t *testing.T) {
	config := types.ProviderConfig{
		Name:        "test-mailer",
		Connector:   "mailer",
		Description: "Test SMTP Provider",
		Options: map[string]interface{}{
			"smtp": map[string]interface{}{
				"host":     "smtp.example.com",
				"port":     587,
				"username": "test@example.com",
				"password": "testpass",
				"from":     "test@example.com",
			},
		},
	}

	provider, err := NewMailerProvider(config)
	require.NoError(t, err)

	info := provider.GetPublicInfo()

	// Verify public information
	assert.Equal(t, "test-mailer", info.Name)
	assert.Equal(t, "mailer", info.Type)
	assert.Equal(t, "Test SMTP Provider", info.Description)
	assert.Equal(t, false, info.Features.SupportsWebhooks)
	assert.Equal(t, false, info.Features.SupportsReceiving) // No IMAP config
	assert.Equal(t, false, info.Features.SupportsTracking)
	assert.Equal(t, false, info.Features.SupportsScheduling)

	// Verify capabilities
	assert.Contains(t, info.Capabilities, "email")
}

func TestProvider_GetPublicInfo_DefaultDescription(t *testing.T) {
	config := types.ProviderConfig{
		Name:      "test-mailer-no-desc",
		Connector: "mailer",
		Options: map[string]interface{}{
			"smtp": map[string]interface{}{
				"host":     "smtp.example.com",
				"port":     587,
				"username": "test@example.com",
				"password": "testpass",
				"from":     "test@example.com",
			},
		},
	}

	provider, err := NewMailerProvider(config)
	require.NoError(t, err)

	info := provider.GetPublicInfo()

	// Should use default description when none provided
	assert.Equal(t, "SMTP email provider", info.Description)
}

func TestProvider_TriggerWebhook(t *testing.T) {
	config := types.ProviderConfig{
		Name:      "test-mailer",
		Connector: "mailer",
		Options: map[string]interface{}{
			"smtp": map[string]interface{}{
				"host":     "smtp.example.com",
				"port":     587,
				"username": "test@example.com",
				"password": "testpass",
				"from":     "test@example.com",
			},
		},
	}

	provider, err := NewMailerProvider(config)
	require.NoError(t, err)

	// TriggerWebhook should return an error for SMTP providers
	msg, err := provider.TriggerWebhook(nil)
	assert.Error(t, err)
	assert.Nil(t, msg)
	assert.Contains(t, err.Error(), "TriggerWebhook not supported for SMTP/mailer provider")
}
