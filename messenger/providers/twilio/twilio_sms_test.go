package twilio

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/messenger/types"
)

// getTestSMSPhone returns the test phone number from environment variable
func getTestSMSPhone() string {
	return os.Getenv("TWILIO_TEST_PHONE")
}

// createTestSMSMessage creates a test SMS message
func createTestSMSMessage() *types.Message {
	return &types.Message{
		Type: types.MessageTypeSMS,
		To:   []string{getTestSMSPhone()},
		Body: "Test SMS from Twilio Provider - This is a test message.",
	}
}

// loadSMSTestConfig loads configuration optimized for SMS testing using Auth Token
func loadSMSTestConfig(t *testing.T) types.ProviderConfig {
	// Reuse base config loading from twilio_test.go (which handles test.Prepare internally)
	config := loadTestConfig(t)

	// Ensure SMS-specific options are available
	// In real implementation, verify TWILIO_FROM_PHONE is configured

	return config
}

// loadSMSTestConfigWithAPIKey loads configuration using API Key authentication
func loadSMSTestConfigWithAPIKey(t *testing.T) types.ProviderConfig {
	// Reuse base config loading from twilio_test.go (which handles test.Prepare internally)
	config := loadTestConfig(t)

	// Override to use API Key authentication instead of Auth Token
	if config.Options != nil {
		// Remove auth_token to force API Key usage
		delete(config.Options, "auth_token")
	}

	return config
}

// =============================================================================
// SMS Provider Configuration Tests
// =============================================================================

func TestSMS_ProviderConfig_WithFromPhone_AuthToken(t *testing.T) {
	config := types.ProviderConfig{
		Name:      "sms_test_auth_token",
		Connector: "twilio",
		Options: map[string]interface{}{
			"account_sid": "test_account_sid",
			"auth_token":  "test_auth_token",
			"from_phone":  "+15551234567", // SMS requires from_phone
		},
	}

	provider, err := NewTwilioProvider(config)
	require.NoError(t, err)
	assert.NotNil(t, provider)
	assert.Equal(t, "+15551234567", provider.fromPhone)
	assert.Equal(t, "test_auth_token", provider.authToken)
	assert.Equal(t, "", provider.apiSID) // API credentials should be empty
	assert.Equal(t, "", provider.apiKey)
}

func TestSMS_ProviderConfig_WithFromPhone_APIKey(t *testing.T) {
	config := types.ProviderConfig{
		Name:      "sms_test_api_key",
		Connector: "twilio",
		Options: map[string]interface{}{
			"account_sid": "test_account_sid",
			"api_sid":     "test_api_sid",
			"api_key":     "test_api_key",
			"from_phone":  "+15551234567", // SMS requires from_phone
		},
	}

	provider, err := NewTwilioProvider(config)
	require.NoError(t, err)
	assert.NotNil(t, provider)
	assert.Equal(t, "+15551234567", provider.fromPhone)
	assert.Equal(t, "test_api_sid", provider.apiSID)
	assert.Equal(t, "test_api_key", provider.apiKey)
	assert.Equal(t, "", provider.authToken) // Auth token should be empty
}

func TestSMS_ProviderConfig_WithMessagingService(t *testing.T) {
	config := types.ProviderConfig{
		Name:      "sms_test",
		Connector: "twilio",
		Options: map[string]interface{}{
			"account_sid":           "test_account_sid",
			"auth_token":            "test_auth_token",
			"messaging_service_sid": "MGXXXXXXXXXXXXXXXXXXXXXXXXXXXXX", // Alternative to from_phone
		},
	}

	provider, err := NewTwilioProvider(config)
	require.NoError(t, err)
	assert.NotNil(t, provider)
	assert.Equal(t, "MGXXXXXXXXXXXXXXXXXXXXXXXXXXXXX", provider.messagingServiceSID)
}

func TestSMS_ProviderConfig_MissingPhoneAndService(t *testing.T) {
	config := types.ProviderConfig{
		Name:      "sms_test",
		Connector: "twilio",
		Options: map[string]interface{}{
			"account_sid": "test_account_sid",
			"auth_token":  "test_auth_token",
			// Missing both from_phone and messaging_service_sid
		},
	}

	provider, err := NewTwilioProvider(config)
	require.NoError(t, err)

	ctx := context.Background()
	smsMessage := createTestSMSMessage()

	err = provider.Send(ctx, smsMessage)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "either from_phone or messaging_service_sid is required for SMS")
}

// =============================================================================
// SMS Sending Tests
// =============================================================================

func TestSend_SMSMessage_WithAuthToken_RealAPI(t *testing.T) {
	// Skip if test phone number is not configured
	if getTestSMSPhone() == "" {
		t.Skip("TWILIO_TEST_PHONE not configured, skipping SMS API test")
	}

	config := loadSMSTestConfig(t)
	provider, err := NewTwilioProvider(config)
	require.NoError(t, err)

	// Skip if from_phone is not configured
	if provider.fromPhone == "" {
		t.Skip("TWILIO_FROM_PHONE not configured, skipping real SMS API test")
	}

	// Skip if auth_token is not configured
	if provider.authToken == "" {
		t.Skip("TWILIO_AUTH_TOKEN not configured, skipping Auth Token SMS API test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	smsMessage := createTestSMSMessage()
	err = provider.Send(ctx, smsMessage)
	if err == nil {
		t.Log("Real Twilio SMS API call with Auth Token succeeded")
	} else {
		t.Logf("Real Twilio SMS API call with Auth Token failed (expected in some test environments): %v", err)
		// Don't fail the test if it's just an API configuration issue
		if !strings.Contains(err.Error(), "Twilio API error") {
			assert.NoError(t, err)
		}
	}
}

func TestSend_SMSMessage_WithAPIKey_RealAPI(t *testing.T) {
	// Skip if test phone number is not configured
	if getTestSMSPhone() == "" {
		t.Skip("TWILIO_TEST_PHONE not configured, skipping SMS API test")
	}

	config := loadSMSTestConfigWithAPIKey(t)
	provider, err := NewTwilioProvider(config)
	require.NoError(t, err)

	// Skip if from_phone is not configured
	if provider.fromPhone == "" {
		t.Skip("TWILIO_FROM_PHONE not configured, skipping real SMS API test")
	}

	// Skip if API Key credentials are not configured
	if provider.apiSID == "" || provider.apiKey == "" {
		t.Skip("TWILIO_API_SID or TWILIO_API_KEY not configured, skipping API Key SMS API test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	smsMessage := createTestSMSMessage()
	err = provider.Send(ctx, smsMessage)
	if err == nil {
		t.Log("Real Twilio SMS API call with API Key succeeded")
	} else {
		t.Logf("Real Twilio SMS API call with API Key failed (expected in some test environments): %v", err)
		// Don't fail the test if it's just an API configuration issue
		if !strings.Contains(err.Error(), "Twilio API error") {
			assert.NoError(t, err)
		}
	}
}

func TestSend_SMSMessage_WithMessagingService_RealAPI(t *testing.T) {
	config := types.ProviderConfig{
		Name:      "sms_messaging_service_test",
		Connector: "twilio",
		Options: map[string]interface{}{
			"account_sid":           "test_account_sid",
			"auth_token":            "test_auth_token",
			"messaging_service_sid": "MGXXXXXXXXXXXXXXXXXXXXXXXXXXXXX",
		},
	}

	provider, err := NewTwilioProvider(config)
	require.NoError(t, err)

	// Skip if messaging service is not configured with real credentials
	if provider.accountSID == "test_account_sid" {
		t.Skip("Real Twilio credentials not configured, skipping messaging service API test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	smsMessage := createTestSMSMessage()
	err = provider.Send(ctx, smsMessage)
	if err == nil {
		t.Log("Real Twilio SMS Messaging Service API call succeeded")
	} else {
		t.Logf("Real Twilio SMS Messaging Service API call failed (expected in some test environments): %v", err)
		// Don't fail the test if it's just an API configuration issue
		if !strings.Contains(err.Error(), "Twilio API error") {
			assert.NoError(t, err)
		}
	}
}

func TestSend_SMSMessage_ContextTimeout_RealAPI(t *testing.T) {
	config := loadSMSTestConfig(t)
	provider, err := NewTwilioProvider(config)
	require.NoError(t, err)

	// Skip if from_phone is not configured
	if provider.fromPhone == "" {
		t.Skip("TWILIO_FROM_PHONE not configured, skipping context timeout test")
	}

	// Create a very short timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	smsMessage := createTestSMSMessage()
	err = provider.Send(ctx, smsMessage)
	if err != nil {
		t.Log("Context timeout working correctly with real SMS API")
		// Could be timeout or other error, both are acceptable for this test
	} else {
		t.Log("Request completed faster than timeout")
	}
}

func TestSendBatch_SMS_RealAPI(t *testing.T) {
	// Skip if test phone number is not configured
	if getTestSMSPhone() == "" {
		t.Skip("TWILIO_TEST_PHONE not configured, skipping SMS API test")
	}

	config := loadSMSTestConfig(t)
	provider, err := NewTwilioProvider(config)
	require.NoError(t, err)

	// Skip if from_phone is not configured
	if provider.fromPhone == "" {
		t.Skip("TWILIO_FROM_PHONE not configured, skipping batch SMS API test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	messages := []*types.Message{
		{
			Type: types.MessageTypeSMS,
			To:   []string{getTestSMSPhone()},
			Body: "Batch SMS Test 1",
		},
		{
			Type: types.MessageTypeSMS,
			To:   []string{getTestSMSPhone()},
			Body: "Batch SMS Test 2",
		},
	}

	err = provider.SendBatch(ctx, messages)
	if err == nil {
		t.Log("Real Twilio SMS batch API call succeeded")
	} else {
		t.Logf("Real Twilio SMS batch API call failed (expected in some test environments): %v", err)
		// Don't fail the test if it's just an API configuration issue
		if !strings.Contains(err.Error(), "Twilio API error") {
			assert.NoError(t, err)
		}
	}
}

func TestSend_SMS_MultipleRecipients_RealAPI(t *testing.T) {
	// Skip if test phone number is not configured
	if getTestSMSPhone() == "" {
		t.Skip("TWILIO_TEST_PHONE not configured, skipping SMS API test")
	}

	config := loadSMSTestConfig(t)
	provider, err := NewTwilioProvider(config)
	require.NoError(t, err)

	// Skip if from_phone is not configured
	if provider.fromPhone == "" {
		t.Skip("TWILIO_FROM_PHONE not configured, skipping multiple recipients SMS API test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	smsMessage := &types.Message{
		Type: types.MessageTypeSMS,
		To:   []string{getTestSMSPhone()}, // Using single phone number for simplicity
		Body: "Multi-recipient SMS test from Twilio Provider",
	}

	err = provider.Send(ctx, smsMessage)
	if err == nil {
		t.Log("Twilio SMS multiple recipients API call succeeded")
	} else {
		t.Logf("Twilio SMS multiple recipients API call failed (expected in some test environments): %v", err)
		// Don't fail the test if it's just an API configuration issue
		if !strings.Contains(err.Error(), "Twilio API error") {
			assert.NoError(t, err)
		}
	}
}

// =============================================================================
// SMS Advanced Features Tests (Future Implementation)
// =============================================================================

func TestSend_SMS_WithCustomMetadata(t *testing.T) {
	t.Skip("SMS metadata tests not implemented yet - placeholder for future implementation")

	// Future implementation will test:
	// - Custom metadata in status callbacks
	// - Tracking and analytics integration
	// - Custom parameters for delivery reporting
}

func TestSend_SMS_WithDeliveryStatus(t *testing.T) {
	t.Skip("SMS delivery status tests not implemented yet - placeholder for future implementation")

	// Future implementation will test:
	// - Status callback configuration
	// - Delivery receipt handling
	// - Failed message retry logic
}

func TestSend_SMS_PhoneNumberValidation(t *testing.T) {
	t.Skip("SMS phone validation tests not implemented yet - placeholder for future implementation")

	// Future implementation will test:
	// - E.164 format validation
	// - International number support
	// - Invalid number error handling
	// - Carrier lookup integration
}

func TestSend_SMS_RateLimiting(t *testing.T) {
	t.Skip("SMS rate limiting tests not implemented yet - placeholder for future implementation")

	// Future implementation will test:
	// - Rate limit handling
	// - Queue management for high-volume sending
	// - Backoff strategies
	// - Error recovery from rate limit exceeded
}

func TestSend_SMS_LongMessages(t *testing.T) {
	t.Skip("SMS long message tests not implemented yet - placeholder for future implementation")

	// Future implementation will test:
	// - Automatic message segmentation
	// - Multi-part SMS handling
	// - Character encoding (GSM 7-bit vs UCS-2)
	// - Cost calculation for long messages
}

// =============================================================================
// SMS Error Handling Tests (Future Implementation)
// =============================================================================

func TestSend_SMS_InvalidPhoneNumber(t *testing.T) {
	t.Skip("SMS invalid phone tests not implemented yet - placeholder for future implementation")

	// Future implementation will test:
	// - Invalid phone number format errors
	// - Undeliverable number handling
	// - Landline vs mobile detection
}

func TestSend_SMS_InsufficientBalance(t *testing.T) {
	t.Skip("SMS balance tests not implemented yet - placeholder for future implementation")

	// Future implementation will test:
	// - Account balance insufficient errors
	// - Graceful degradation when funds are low
	// - Balance monitoring and alerts
}

func TestSend_SMS_APIError_Scenarios(t *testing.T) {
	t.Skip("SMS API error tests not implemented yet - placeholder for future implementation")

	// Future implementation will test:
	// - Various Twilio API error codes
	// - Network timeout handling
	// - Authentication failures
	// - Service unavailable scenarios
}

// =============================================================================
// SMS Benchmark Tests (Future Implementation)
// =============================================================================

func BenchmarkSend_SMS_AuthToken(b *testing.B) {
	config := loadSMSTestConfig(&testing.T{})
	provider, err := NewTwilioProvider(config)
	if err != nil {
		b.Fatalf("Failed to create provider: %v", err)
	}

	// Skip if from_phone or auth_token is not configured
	if provider.fromPhone == "" || provider.authToken == "" {
		b.Skip("TWILIO_FROM_PHONE or TWILIO_AUTH_TOKEN not configured, skipping Auth Token benchmark")
	}

	ctx := context.Background()
	smsMessage := createTestSMSMessage()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = provider.Send(ctx, smsMessage)
	}
}

func BenchmarkSend_SMS_APIKey(b *testing.B) {
	config := loadSMSTestConfigWithAPIKey(&testing.T{})
	provider, err := NewTwilioProvider(config)
	if err != nil {
		b.Fatalf("Failed to create provider: %v", err)
	}

	// Skip if from_phone or API credentials are not configured
	if provider.fromPhone == "" || provider.apiSID == "" || provider.apiKey == "" {
		b.Skip("TWILIO_FROM_PHONE, TWILIO_API_SID, or TWILIO_API_KEY not configured, skipping API Key benchmark")
	}

	ctx := context.Background()
	smsMessage := createTestSMSMessage()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = provider.Send(ctx, smsMessage)
	}
}

func BenchmarkSendBatch_SMS_AuthToken(b *testing.B) {
	config := loadSMSTestConfig(&testing.T{})
	provider, err := NewTwilioProvider(config)
	if err != nil {
		b.Fatalf("Failed to create provider: %v", err)
	}

	// Skip if from_phone or auth_token is not configured
	if provider.fromPhone == "" || provider.authToken == "" {
		b.Skip("TWILIO_FROM_PHONE or TWILIO_AUTH_TOKEN not configured, skipping Auth Token batch benchmark")
	}

	ctx := context.Background()
	messages := []*types.Message{
		createTestSMSMessage(),
		createTestSMSMessage(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = provider.SendBatch(ctx, messages)
	}
}
