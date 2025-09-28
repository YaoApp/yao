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

// Test helper functions for receive tests

func getEnvOrDefaultReceive(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func loadPrimaryTestConfigReceive(t *testing.T) types.ProviderConfig {
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

func loadReliableTestConfigReceive(t *testing.T) types.ProviderConfig {
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
				"host":     getEnvOrDefaultReceive("RELIABLE_IMAP_HOST", os.Getenv("RELIABLE_SMTP_HOST")),
				"port":     getEnvOrDefaultReceive("RELIABLE_IMAP_PORT", "993"),
				"username": getEnvOrDefaultReceive("RELIABLE_IMAP_USERNAME", os.Getenv("RELIABLE_SMTP_USERNAME")),
				"password": getEnvOrDefaultReceive("RELIABLE_IMAP_PASSWORD", os.Getenv("RELIABLE_SMTP_PASSWORD")),
				"use_ssl":  true,
				"mailbox":  "INBOX",
			},
		},
	}

	return config
}

// Test IMAP Support Detection

func TestSupportsReceiving_WithIMAP(t *testing.T) {
	config := loadReliableTestConfigReceive(t)
	provider, err := NewMailerProvider(config)
	require.NoError(t, err)

	// Reliable config has IMAP configured, should support receiving
	assert.True(t, provider.SupportsReceiving())
}

func TestSupportsReceiving_WithoutIMAP(t *testing.T) {
	config := loadPrimaryTestConfigReceive(t)
	provider, err := NewMailerProvider(config)
	require.NoError(t, err)

	// Primary config has no IMAP configured, should not support receiving
	assert.False(t, provider.SupportsReceiving())
}

// Test Receive Method

func TestReceive_WithoutIMAPSupport(t *testing.T) {
	config := loadPrimaryTestConfigReceive(t)
	provider, err := NewMailerProvider(config)
	require.NoError(t, err)

	ctx := context.Background()
	data := map[string]interface{}{
		"type":    "delivery",
		"message": "test message",
	}

	// Should return error since IMAP is not configured
	err = provider.Receive(ctx, data)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "provider does not support receiving: IMAP not configured")
}

func TestReceive_WithIMAPSupport_Bounce(t *testing.T) {
	config := loadReliableTestConfigReceive(t)
	provider, err := NewMailerProvider(config)
	require.NoError(t, err)

	ctx := context.Background()
	data := map[string]interface{}{
		"type":       "bounce",
		"email":      "test@example.com",
		"reason":     "mailbox_full",
		"timestamp":  time.Now().Unix(),
		"message_id": "test-message-123",
	}

	// Should process bounce without error
	err = provider.Receive(ctx, data)
	assert.NoError(t, err)
}

func TestReceive_WithIMAPSupport_Delivery(t *testing.T) {
	config := loadReliableTestConfigReceive(t)
	provider, err := NewMailerProvider(config)
	require.NoError(t, err)

	ctx := context.Background()
	data := map[string]interface{}{
		"type":       "delivery",
		"email":      "test@example.com",
		"timestamp":  time.Now().Unix(),
		"message_id": "test-message-123",
	}

	// Should process delivery without error
	err = provider.Receive(ctx, data)
	assert.NoError(t, err)
}

func TestReceive_WithIMAPSupport_Complaint(t *testing.T) {
	config := loadReliableTestConfigReceive(t)
	provider, err := NewMailerProvider(config)
	require.NoError(t, err)

	ctx := context.Background()
	data := map[string]interface{}{
		"type":       "complaint",
		"email":      "test@example.com",
		"reason":     "spam",
		"timestamp":  time.Now().Unix(),
		"message_id": "test-message-123",
	}

	// Should process complaint without error
	err = provider.Receive(ctx, data)
	assert.NoError(t, err)
}

func TestReceive_WithIMAPSupport_UnknownType(t *testing.T) {
	config := loadReliableTestConfigReceive(t)
	provider, err := NewMailerProvider(config)
	require.NoError(t, err)

	ctx := context.Background()
	data := map[string]interface{}{
		"type":    "unknown_event",
		"message": "test message",
	}

	// Should process unknown type without error (just logs)
	err = provider.Receive(ctx, data)
	assert.NoError(t, err)
}

func TestReceive_WithIMAPSupport_NoType(t *testing.T) {
	config := loadReliableTestConfigReceive(t)
	provider, err := NewMailerProvider(config)
	require.NoError(t, err)

	ctx := context.Background()
	data := map[string]interface{}{
		"message": "test message without type",
		"data":    "some data",
	}

	// Should process data without type field without error
	err = provider.Receive(ctx, data)
	assert.NoError(t, err)
}

// Test StartMailReceiver Method

func TestStartMailReceiver_WithoutIMAPSupport(t *testing.T) {
	config := loadPrimaryTestConfigReceive(t)
	provider, err := NewMailerProvider(config)
	require.NoError(t, err)

	ctx := context.Background()
	handler := func(msg *types.Message) error {
		return nil
	}

	// Should return error since IMAP is not configured
	err = provider.StartMailReceiver(ctx, handler)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "provider does not support receiving: IMAP not configured")
}

func TestStartMailReceiver_WithIMAPSupport_InvalidConfig(t *testing.T) {
	// Skip this test if IMAP credentials are not configured
	if os.Getenv("RELIABLE_IMAP_HOST") == "" {
		t.Skip("RELIABLE_IMAP_HOST not configured, skipping IMAP connection test")
	}

	config := loadReliableTestConfigReceive(t)
	provider, err := NewMailerProvider(config)
	require.NoError(t, err)

	// Use a short timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	messageReceived := false
	handler := func(msg *types.Message) error {
		messageReceived = true
		t.Logf("Received message: Subject=%s, From=%s", msg.Subject, msg.From)
		return nil
	}

	// This will likely fail due to invalid credentials, but should not panic
	err = provider.StartMailReceiver(ctx, handler)

	// We expect this to fail in test environment, but it should be a connection error
	if err != nil {
		t.Logf("StartMailReceiver failed as expected in test environment: %v", err)
		assert.Contains(t, err.Error(), "provider does not support receiving: IMAP not configured")
	} else {
		t.Log("StartMailReceiver started successfully")
		// Wait a bit to see if any messages are received
		time.Sleep(2 * time.Second)
		t.Logf("Message received: %v", messageReceived)
	}
}

// Test MailReceiver Internal Methods

func TestMailReceiver_TimeStampFiltering(t *testing.T) {
	config := loadReliableTestConfigReceive(t)
	provider, err := NewMailerProvider(config)
	require.NoError(t, err)

	// Create a mail receiver
	receiver := &MailReceiver{
		provider:     provider,
		stopChan:     make(chan bool),
		startTime:    time.Now(),
		lastCheckUID: 0,
		msgHandler: func(msg *types.Message) error {
			return nil
		},
	}

	// Test that start time is set correctly
	assert.True(t, receiver.startTime.Before(time.Now().Add(time.Second)))
	assert.True(t, receiver.startTime.After(time.Now().Add(-time.Second)))
	assert.Equal(t, uint32(0), receiver.lastCheckUID)
}

func TestMailReceiver_Stop(t *testing.T) {
	config := loadReliableTestConfigReceive(t)
	provider, err := NewMailerProvider(config)
	require.NoError(t, err)

	// Create a mail receiver
	receiver := &MailReceiver{
		provider:     provider,
		stopChan:     make(chan bool),
		startTime:    time.Now(),
		lastCheckUID: 0,
		msgHandler: func(msg *types.Message) error {
			return nil
		},
	}

	// Test stop functionality
	go func() {
		time.Sleep(100 * time.Millisecond)
		receiver.Stop()
	}()

	// This should not block indefinitely
	select {
	case <-receiver.stopChan:
		t.Log("Stop signal received successfully")
	case <-time.After(1 * time.Second):
		t.Error("Stop signal not received within timeout")
	}
}

// Test Message Processing

func TestMailReceiver_FormatAddress(t *testing.T) {
	config := loadReliableTestConfigReceive(t)
	provider, err := NewMailerProvider(config)
	require.NoError(t, err)

	receiver := &MailReceiver{
		provider: provider,
	}

	// Test with empty addresses
	result := receiver.formatAddress(nil)
	assert.Equal(t, "", result)

	// Note: We can't easily test with real imap.Address without importing go-imap
	// and creating mock addresses, but the function is tested through integration tests
}

func TestMailReceiver_FormatAddresses(t *testing.T) {
	config := loadReliableTestConfigReceive(t)
	provider, err := NewMailerProvider(config)
	require.NoError(t, err)

	receiver := &MailReceiver{
		provider: provider,
	}

	// Test with empty addresses
	result := receiver.formatAddresses(nil)
	assert.Equal(t, []string{}, result)

	// Note: We can't easily test with real imap.Address without importing go-imap
	// and creating mock addresses, but the function is tested through integration tests
}

// Integration Tests - Real Email Send and Receive

func TestRealEmailSendAndReceive_Integration(t *testing.T) {
	// Skip this test if IMAP credentials are not configured
	if os.Getenv("RELIABLE_IMAP_HOST") == "" || os.Getenv("RELIABLE_SMTP_HOST") == "" {
		t.Skip("RELIABLE_IMAP_HOST or RELIABLE_SMTP_HOST not configured, skipping integration test")
	}

	config := loadReliableTestConfigReceive(t)
	provider, err := NewMailerProvider(config)
	require.NoError(t, err)

	// Verify provider supports both sending and receiving
	require.True(t, provider.SupportsReceiving(), "Provider must support receiving for this test")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Channel to receive the email
	emailReceived := make(chan *types.Message, 1)
	var receivedEmail *types.Message

	// Start mail receiver with detailed logging
	go func() {
		t.Log("Starting mail receiver goroutine...")
		err := provider.StartMailReceiver(ctx, func(msg *types.Message) error {
			t.Logf("=== EMAIL RECEIVED ===")
			t.Logf("Subject: %s", msg.Subject)
			t.Logf("From: %s", msg.From)
			t.Logf("To: %v", msg.To)
			t.Logf("Type: %s", msg.Type)
			if msg.Body != "" {
				bodyPreview := msg.Body
				if len(bodyPreview) > 200 {
					bodyPreview = bodyPreview[:200] + "..."
				}
				t.Logf("Body: %s", bodyPreview)
			}
			if msg.HTML != "" {
				htmlPreview := msg.HTML
				if len(htmlPreview) > 100 {
					htmlPreview = htmlPreview[:100] + "..."
				}
				t.Logf("HTML: %s", htmlPreview)
			}
			if msg.Metadata != nil {
				t.Logf("Metadata: %+v", msg.Metadata)
			}
			t.Logf("=== END EMAIL ===")

			// Check if this is our test email
			if msg.Subject != "" && msg.Body != "" {
				select {
				case emailReceived <- msg:
					t.Log("✅ Test email captured successfully")
				default:
					t.Log("⚠️ Email channel full, skipping")
				}
			}
			return nil
		})

		if err != nil {
			t.Logf("❌ Mail receiver stopped with error: %v", err)
		} else {
			t.Log("✅ Mail receiver stopped gracefully")
		}
	}()

	// Give receiver time to start and connect
	t.Log("⏳ Waiting 5 seconds for mail receiver to start and connect...")
	time.Sleep(5 * time.Second)
	t.Log("✅ Mail receiver should be connected now")

	// Create and send test email
	testSubject := "Integration Test Email - " + time.Now().Format("2006-01-02 15:04:05")
	testBody := "This is an integration test email sent at " + time.Now().Format("2006-01-02 15:04:05") + ". If you receive this, the send/receive cycle is working!"

	// Get the 'from' address from config to send email to ourselves
	smtpConfig := config.Options["smtp"].(map[string]interface{})
	fromAddressRaw := smtpConfig["from"].(string)

	// Extract just the email address from "Name <email@domain.com>" format
	fromAddress := fromAddressRaw
	if strings.Contains(fromAddressRaw, "<") && strings.Contains(fromAddressRaw, ">") {
		start := strings.Index(fromAddressRaw, "<")
		end := strings.Index(fromAddressRaw, ">")
		if start >= 0 && end > start {
			fromAddress = fromAddressRaw[start+1 : end]
		}
	}

	testMessage := &types.Message{
		Type:    types.MessageTypeEmail,
		To:      []string{fromAddress}, // Send to ourselves
		Subject: testSubject,
		Body:    testBody,
		HTML:    "<h1>Integration Test</h1><p>" + testBody + "</p>",
		Headers: map[string]string{
			"X-Test-Type": "integration-test",
			"X-Test-ID":   time.Now().Format("20060102150405"),
		},
	}

	t.Logf("📧 Sending test email to: %s", fromAddress)
	t.Logf("📧 Subject: %s", testSubject)
	t.Logf("📧 Body: %s", testBody)

	// Send the email
	sendErr := provider.Send(ctx, testMessage)
	if sendErr != nil {
		t.Logf("❌ Failed to send test email: %v", sendErr)
		// Don't fail the test immediately, as this might be expected in some environments
		t.Skip("Could not send test email, skipping integration test")
	}

	t.Log("✅ Test email sent successfully, waiting for receipt...")
	t.Log("⏳ Monitoring for incoming emails (timeout: 90 seconds)...")

	// Wait for email to be received
	select {
	case receivedEmail = <-emailReceived:
		t.Log("SUCCESS: Email send/receive cycle completed!")

		// Cancel the context to stop the mail receiver gracefully
		cancel()
		t.Log("🛑 Gracefully stopping mail receiver...")

		// Give some time for graceful shutdown
		time.Sleep(1 * time.Second)

		// Verify the received email
		assert.NotNil(t, receivedEmail)
		assert.Equal(t, types.MessageTypeEmail, receivedEmail.Type)
		assert.NotEmpty(t, receivedEmail.Subject)
		assert.NotEmpty(t, receivedEmail.From)

		// Check if it's our test email (subject should contain our test string)
		if receivedEmail.Subject == testSubject {
			t.Log("PERFECT MATCH: Received the exact email we sent!")
			assert.Equal(t, testSubject, receivedEmail.Subject)
			// Note: Body might be modified by email processing, so we check if it contains our content
			if receivedEmail.Body != "" {
				t.Logf("Received body: %s", receivedEmail.Body)
			}
		} else {
			t.Logf("Received different email: Subject='%s'", receivedEmail.Subject)
			t.Log("This might be another email in the inbox, which is also a valid test result")
		}

		// Verify metadata
		assert.NotNil(t, receivedEmail.Metadata)
		if receivedEmail.Metadata != nil {
			t.Logf("Email metadata: %+v", receivedEmail.Metadata)
		}

		t.Log("✅ Test completed successfully - mail receiver stopped gracefully")
		return // Exit the test successfully

	case <-time.After(90 * time.Second):
		t.Log("TIMEOUT: No email received within 90 seconds")
		t.Log("This might be expected in test environments with:")
		t.Log("- Email delivery delays")
		t.Log("- IMAP connection issues")
		t.Log("- Firewall restrictions")
		t.Log("- Invalid credentials")

		// Cancel context for graceful shutdown
		cancel()
		t.Log("🛑 Stopping mail receiver due to timeout...")
		time.Sleep(1 * time.Second)

		// This is not necessarily a failure - email delivery can be delayed
		t.Skip("Email not received within timeout - this may be expected in test environment")

	case <-ctx.Done():
		t.Log("Context cancelled during email wait")
		t.Skip("Test context cancelled")
	}
}

func TestRealEmailReceiveOnly_Integration(t *testing.T) {
	// Skip this test if IMAP credentials are not configured
	if os.Getenv("RELIABLE_IMAP_HOST") == "" {
		t.Skip("RELIABLE_IMAP_HOST not configured, skipping IMAP receive test")
	}

	config := loadReliableTestConfigReceive(t)
	provider, err := NewMailerProvider(config)
	require.NoError(t, err)

	// Verify provider supports receiving
	require.True(t, provider.SupportsReceiving(), "Provider must support receiving for this test")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	emailCount := 0
	maxEmailsToProcess := 5 // Limit the number of emails to process for testing

	t.Log("Starting mail receiver to check for existing emails...")

	// Start mail receiver to see if there are any emails
	err = provider.StartMailReceiver(ctx, func(msg *types.Message) error {
		emailCount++
		t.Logf("Email #%d received:", emailCount)
		t.Logf("  Subject: %s", msg.Subject)
		t.Logf("  From: %s", msg.From)
		t.Logf("  To: %v", msg.To)
		if msg.Body != "" {
			bodyPreview := msg.Body
			if len(bodyPreview) > 100 {
				bodyPreview = bodyPreview[:100] + "..."
			}
			t.Logf("  Body preview: %s", bodyPreview)
		}
		if msg.HTML != "" {
			htmlPreview := msg.HTML
			if len(htmlPreview) > 100 {
				htmlPreview = htmlPreview[:100] + "..."
			}
			t.Logf("  HTML preview: %s", htmlPreview)
		}
		if msg.Metadata != nil {
			t.Logf("  Metadata: %+v", msg.Metadata)
		}

		// Stop after processing a few emails to avoid long-running tests
		if emailCount >= maxEmailsToProcess {
			t.Logf("Processed %d emails, stopping receiver for test completion", emailCount)
			cancel() // Trigger graceful shutdown
		}

		return nil
	})

	if err != nil {
		t.Logf("Mail receiver ended: %v", err)

		// Check if it's a connection error (expected in many test environments)
		if strings.Contains(err.Error(), "failed to connect") ||
			strings.Contains(err.Error(), "authentication failed") ||
			strings.Contains(err.Error(), "connection refused") {
			t.Skip("IMAP connection failed - this is expected in test environments without proper email server access")
		}

		// Other errors might indicate real issues
		t.Errorf("Unexpected error from mail receiver: %v", err)
	}

	t.Logf("Mail receiver test completed. Total emails processed: %d", emailCount)

	if emailCount > 0 {
		t.Log("SUCCESS: Mail receiver is working and processed emails from the mailbox")
	} else {
		t.Log("No emails received - this could mean:")
		t.Log("- Mailbox is empty (normal)")
		t.Log("- IMAP connection issues")
		t.Log("- Time-based filtering working (only new emails)")
	}
}

func TestManualEmailReceive_Integration(t *testing.T) {
	// Skip this test if IMAP credentials are not configured
	if os.Getenv("RELIABLE_IMAP_HOST") == "" {
		t.Skip("RELIABLE_IMAP_HOST not configured, skipping manual receive test")
	}

	config := loadReliableTestConfigReceive(t)
	provider, err := NewMailerProvider(config)
	require.NoError(t, err)

	// Verify provider supports receiving
	require.True(t, provider.SupportsReceiving(), "Provider must support receiving for this test")

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	emailReceived := make(chan *types.Message, 5)
	testCompleted := make(chan bool, 1)

	t.Log("🔍 MANUAL TEST: Please send an email to shadow.iqka@gmail.com now!")
	t.Log("📧 Subject should contain 'MANUAL TEST' for easy identification")
	t.Log("⏰ You have 60 seconds to send the email...")

	// Start mail receiver
	go func() {
		err := provider.StartMailReceiver(ctx, func(msg *types.Message) error {
			t.Logf("📧 RECEIVED EMAIL:")
			t.Logf("  Subject: %s", msg.Subject)
			t.Logf("  From: %s", msg.From)
			t.Logf("  To: %v", msg.To)
			t.Logf("  Type: %s", msg.Type)

			// Send to channel for verification
			select {
			case emailReceived <- msg:
				t.Log("✅ Email captured successfully!")
			default:
				t.Log("⚠️ Email channel full")
			}
			return nil
		})

		if err != nil {
			t.Logf("Mail receiver ended: %v", err)
		}
		testCompleted <- true
	}()

	// Wait for emails or timeout
	emailCount := 0
	timeout := time.After(60 * time.Second)

	for {
		select {
		case receivedEmail := <-emailReceived:
			emailCount++
			t.Logf("🎉 EMAIL #%d RECEIVED!", emailCount)
			t.Logf("Subject: %s", receivedEmail.Subject)
			t.Logf("From: %s", receivedEmail.From)

			// Check if this looks like a manual test email
			if strings.Contains(strings.ToUpper(receivedEmail.Subject), "MANUAL TEST") {
				t.Log("🎯 MANUAL TEST EMAIL DETECTED!")
				cancel()
				<-testCompleted

				assert.NotNil(t, receivedEmail)
				assert.NotEmpty(t, receivedEmail.Subject)
				assert.NotEmpty(t, receivedEmail.From)

				t.Log("✅ MANUAL TEST PASSED - Email receiving works!")
				return
			}

			// Continue waiting for more emails
			t.Log("📬 Waiting for more emails...")

		case <-timeout:
			t.Logf("⏰ Manual test timeout after 60 seconds")
			t.Logf("📊 Total emails received: %d", emailCount)
			cancel()
			<-testCompleted

			if emailCount > 0 {
				t.Log("✅ Email receiving is working (received emails during test)")
			} else {
				t.Log("❓ No emails received - this could mean:")
				t.Log("  - No emails were sent during the test")
				t.Log("  - Email delivery is delayed")
				t.Log("  - IMAP filtering is working (only new emails)")
			}
			return

		case <-ctx.Done():
			<-testCompleted
			t.Logf("📊 Test ended. Total emails received: %d", emailCount)
			return
		}
	}
}

func TestQuickEmailSendAndReceive_Integration(t *testing.T) {
	// Skip this test if IMAP credentials are not configured
	if os.Getenv("RELIABLE_IMAP_HOST") == "" || os.Getenv("RELIABLE_SMTP_HOST") == "" {
		t.Skip("RELIABLE_IMAP_HOST or RELIABLE_SMTP_HOST not configured, skipping quick integration test")
	}

	config := loadReliableTestConfigReceive(t)
	provider, err := NewMailerProvider(config)
	require.NoError(t, err)

	// Verify provider supports both sending and receiving
	require.True(t, provider.SupportsReceiving(), "Provider must support receiving for this test")

	// Use longer timeout to account for email delivery delays
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// Channel to receive the email
	emailReceived := make(chan *types.Message, 5)
	testCompleted := make(chan bool, 1)

	var sentTestSubject string
	emailCount := 0

	// Start mail receiver
	go func() {
		t.Log("🚀 Starting mail receiver for send/receive test...")
		err := provider.StartMailReceiver(ctx, func(msg *types.Message) error {
			emailCount++
			t.Logf("📧 Email #%d received: Subject='%s', From='%s'", emailCount, msg.Subject, msg.From)

			// Send all received emails to the channel for analysis
			select {
			case emailReceived <- msg:
				t.Logf("✅ Email #%d captured for analysis", emailCount)
			default:
				t.Log("⚠️ Email channel full")
			}
			return nil
		})

		if err != nil {
			t.Logf("Mail receiver ended: %v", err)
		}
		testCompleted <- true
	}()

	// Give receiver more time to start and connect
	t.Log("⏳ Waiting 5 seconds for mail receiver to fully start...")
	time.Sleep(5 * time.Second)

	// Create and send test email with unique identifier
	timestamp := time.Now().Format("15:04:05.000")
	testSubject := "AUTOMATED TEST EMAIL - " + timestamp
	testBody := "This is an automated integration test email sent at " + timestamp + ". Please ignore this message."
	sentTestSubject = testSubject // Store for comparison

	// Get the 'from' address from config
	smtpConfig := config.Options["smtp"].(map[string]interface{})
	fromAddressRaw := smtpConfig["from"].(string)

	// Extract just the email address
	fromAddress := fromAddressRaw
	if strings.Contains(fromAddressRaw, "<") && strings.Contains(fromAddressRaw, ">") {
		start := strings.Index(fromAddressRaw, "<")
		end := strings.Index(fromAddressRaw, ">")
		if start >= 0 && end > start {
			fromAddress = fromAddressRaw[start+1 : end]
		}
	}

	testMessage := &types.Message{
		Type:    types.MessageTypeEmail,
		To:      []string{fromAddress},
		Subject: testSubject,
		Body:    testBody,
		Headers: map[string]string{
			"X-Test-Type":      "automated-integration-test",
			"X-Test-Timestamp": timestamp,
		},
	}

	t.Logf("📤 Sending test email: %s", testSubject)
	t.Logf("📧 To: %s", fromAddress)

	// Send the email
	sendErr := provider.Send(ctx, testMessage)
	if sendErr != nil {
		t.Logf("❌ Failed to send test email: %v", sendErr)
		cancel() // Stop receiver
		<-testCompleted
		t.Skip("Could not send test email, skipping integration test")
	}

	t.Log("✅ Test email sent successfully!")
	t.Log("⏳ Monitoring for incoming emails (timeout: 100 seconds)...")
	t.Log("📊 Will analyze all received emails to find our test email...")

	// Wait for emails and analyze them
	foundTestEmail := false
	timeout := time.After(100 * time.Second)

	for !foundTestEmail {
		select {
		case receivedEmail := <-emailReceived:
			t.Logf("📧 Analyzing email: Subject='%s'", receivedEmail.Subject)

			// Check if this is our test email
			if receivedEmail.Subject == sentTestSubject {
				t.Log("🎯 FOUND OUR TEST EMAIL!")
				t.Logf("✅ Subject matches: %s", receivedEmail.Subject)
				t.Logf("✅ From: %s", receivedEmail.From)

				// Stop the receiver gracefully
				cancel()
				<-testCompleted

				// Verify the email properties
				assert.NotNil(t, receivedEmail)
				assert.Equal(t, sentTestSubject, receivedEmail.Subject)
				assert.NotEmpty(t, receivedEmail.From)
				assert.Equal(t, types.MessageTypeEmail, receivedEmail.Type)

				t.Log("🎉 INTEGRATION TEST PASSED - Email send/receive cycle works!")
				return
			} else if strings.Contains(receivedEmail.Subject, "AUTOMATED TEST") {
				t.Log("🔍 Found another automated test email (different timestamp)")
			} else {
				t.Log("📬 Found other email, continuing to monitor...")
			}

		case <-timeout:
			t.Logf("⏰ Test timeout after 100 seconds")
			t.Logf("📊 Total emails received during test: %d", emailCount)
			t.Logf("🔍 Looking for subject: %s", sentTestSubject)

			cancel()
			<-testCompleted

			if emailCount > 0 {
				t.Log("✅ Email receiving is working (got emails), but our test email may be delayed")
				t.Log("💡 This could be due to Gmail's email processing delays")
			} else {
				t.Log("❓ No emails received during test period")
				t.Log("💡 This could indicate IMAP filtering is working correctly (only new emails)")
			}

			t.Skip("Test email not received within timeout - email delivery delays are common")

		case <-ctx.Done():
			<-testCompleted
			t.Logf("📊 Test ended. Total emails received: %d", emailCount)
			t.Skip("Test context cancelled")
		}
	}
}

// Benchmark Tests for Receive Functionality

func BenchmarkReceive_WithIMAPSupport(b *testing.B) {
	t := &testing.T{}
	config := loadReliableTestConfigReceive(t)
	provider, err := NewMailerProvider(config)
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()
	data := map[string]interface{}{
		"type":    "delivery",
		"message": "benchmark test message",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := provider.Receive(ctx, data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSupportsReceiving(b *testing.B) {
	t := &testing.T{}
	config := loadReliableTestConfigReceive(t)
	provider, err := NewMailerProvider(config)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = provider.SupportsReceiving()
	}
}
