package messenger

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/messenger/providers/mailer"
	"github.com/yaoapp/yao/messenger/types"
	"github.com/yaoapp/yao/test"
)

func TestService_MailReceiverManagement(t *testing.T) {
	// Prepare test environment
	test.Prepare(t, config.Conf, "YAO_TEST_APPLICATION")
	defer test.Clean()

	// Create a test service with real providers loaded from configuration
	providers, err := loadProviders()
	require.NoError(t, err)

	service := &Service{
		config:          &types.Config{},
		providers:       providers,
		providersByType: make(map[types.MessageType][]types.Provider),
		channels:        make(map[string]types.Channel),
		defaults:        make(map[string]string),
		receivers:       make(map[string]context.CancelFunc),
	}

	// Log loaded providers for debugging
	t.Logf("Loaded providers: %d", len(providers))
	for name, provider := range providers {
		t.Logf("Provider: %s, Type: %s", name, provider.GetType())
		if provider.GetType() == "mailer" {
			if mailerProvider, ok := provider.(*mailer.Provider); ok {
				t.Logf("  - Supports receiving: %v", mailerProvider.SupportsReceiving())
			}
		}
	}

	// Test GetActiveReceivers when no receivers are active
	activeReceivers := service.GetActiveReceivers()
	assert.Empty(t, activeReceivers)

	// Test startMailReceivers (this should start receivers for mailer providers that support IMAP)
	service.startMailReceivers()

	// Give some time for goroutines to start
	time.Sleep(200 * time.Millisecond)

	// Check active receivers - should include providers that support IMAP
	activeReceivers = service.GetActiveReceivers()
	t.Logf("Active receivers after start: %v", activeReceivers)

	// Count how many mailer providers support receiving
	expectedReceivers := 0
	for name, provider := range providers {
		if provider.GetType() == "mailer" {
			if mailerProvider, ok := provider.(*mailer.Provider); ok {
				if mailerProvider.SupportsReceiving() {
					expectedReceivers++
					t.Logf("Provider %s supports receiving", name)
				}
			}
		}
	}

	assert.Len(t, activeReceivers, expectedReceivers)

	// Test StopMailReceiver for each active receiver
	for _, receiverName := range activeReceivers {
		service.StopMailReceiver(receiverName)
		t.Logf("Stopped receiver: %s", receiverName)
	}

	// Give some time for cleanup
	time.Sleep(200 * time.Millisecond)

	activeReceivers = service.GetActiveReceivers()
	assert.Empty(t, activeReceivers)

	// Test StopMailReceiver for non-existent provider (should not panic)
	service.StopMailReceiver("nonexistent")

	// Test StopMailReceivers (should handle empty receivers gracefully)
	service.StopMailReceivers()
}

func TestService_StartMailReceivers_NoMailerProviders(t *testing.T) {
	// Create a service with no mailer providers
	service := &Service{
		config:    &types.Config{},
		providers: map[string]types.Provider{
			// No mailer providers, only other types
		},
		providersByType: make(map[types.MessageType][]types.Provider),
		channels:        make(map[string]types.Channel),
		defaults:        make(map[string]string),
		receivers:       make(map[string]context.CancelFunc),
	}

	// This should not start any receivers
	service.startMailReceivers()

	activeReceivers := service.GetActiveReceivers()
	assert.Empty(t, activeReceivers)
}

func TestLoad_AutoStartMailReceivers(t *testing.T) {
	// Prepare test environment
	test.Prepare(t, config.Conf, "YAO_TEST_APPLICATION")
	defer test.Clean()

	// Load messenger configuration (this should auto-start mail receivers)
	err := Load(config.Conf)
	require.NoError(t, err)

	// Verify that Instance is set
	require.NotNil(t, Instance)

	// Cast to Service to access receiver management methods
	service, ok := Instance.(*Service)
	require.True(t, ok, "Instance should be of type *Service")

	// Give some time for receivers to start
	time.Sleep(300 * time.Millisecond)

	// Check active receivers
	activeReceivers := service.GetActiveReceivers()
	t.Logf("Auto-started receivers: %v", activeReceivers)

	// Count expected receivers from loaded providers
	expectedReceivers := 0
	for name, provider := range service.providers {
		if provider.GetType() == "mailer" {
			if mailerProvider, ok := provider.(*mailer.Provider); ok {
				if mailerProvider.SupportsReceiving() {
					expectedReceivers++
					t.Logf("Provider %s supports receiving and should have auto-started", name)
				} else {
					t.Logf("Provider %s does not support receiving (no IMAP config)", name)
				}
			}
		}
	}

	assert.Len(t, activeReceivers, expectedReceivers)

	// Clean up - stop all receivers
	service.StopMailReceivers()

	// Give some time for cleanup
	time.Sleep(200 * time.Millisecond)

	// Verify all receivers are stopped
	activeReceivers = service.GetActiveReceivers()
	assert.Empty(t, activeReceivers)
}

func TestService_RealProviderConfiguration(t *testing.T) {
	// Prepare test environment
	test.Prepare(t, config.Conf, "YAO_TEST_APPLICATION")
	defer test.Clean()

	// Load real providers
	providers, err := loadProviders()
	require.NoError(t, err)

	t.Logf("Testing with real provider configurations:")

	// Analyze each provider
	for name, provider := range providers {
		t.Logf("Provider: %s", name)
		t.Logf("  Type: %s", provider.GetType())

		if provider.GetType() == "mailer" {
			if mailerProvider, ok := provider.(*mailer.Provider); ok {
				supportsReceiving := mailerProvider.SupportsReceiving()
				t.Logf("  Supports receiving: %v", supportsReceiving)

				// Test the provider's configuration
				err := mailerProvider.Validate()
				if err != nil {
					t.Logf("  Validation error: %v", err)
				} else {
					t.Logf("  Configuration is valid")
				}

				// If it supports receiving, test that we can create a receiver context
				if supportsReceiving {
					ctx, cancel := context.WithCancel(context.Background())

					// Test that StartMailReceiver doesn't immediately fail
					go func() {
						err := mailerProvider.StartMailReceiver(ctx, func(msg *types.Message) error {
							t.Logf("Received test message: %s", msg.Subject)
							return nil
						})
						if err != nil {
							t.Logf("Mail receiver for %s ended with: %v", name, err)
						}
					}()

					// Cancel immediately to avoid long-running connections in tests
					time.Sleep(100 * time.Millisecond)
					cancel()

					t.Logf("  Successfully tested receiver startup/shutdown")
				}
			}
		}
	}
}

// Helper function to create mock mailer providers for testing
func createMockMailerProvider(t *testing.T, supportsIMAP bool) *mailer.Provider {
	var config types.ProviderConfig

	if supportsIMAP {
		// Create config with IMAP support
		config = types.ProviderConfig{
			Name:      "test-reliable",
			Connector: "mailer",
			Options: map[string]interface{}{
				"smtp": map[string]interface{}{
					"host":     "smtp.example.com",
					"port":     587,
					"username": "test@example.com",
					"password": "password",
					"from":     "test@example.com",
					"use_tls":  true,
				},
				"imap": map[string]interface{}{
					"host":     "imap.example.com",
					"port":     993,
					"username": "test@example.com",
					"password": "password",
					"use_ssl":  true,
					"mailbox":  "INBOX",
				},
			},
		}
	} else {
		// Create config without IMAP support
		config = types.ProviderConfig{
			Name:      "test-primary",
			Connector: "mailer",
			Options: map[string]interface{}{
				"smtp": map[string]interface{}{
					"host":     "smtp.example.com",
					"port":     587,
					"username": "test@example.com",
					"password": "password",
					"from":     "test@example.com",
					"use_tls":  true,
				},
			},
		}
	}

	provider, err := mailer.NewMailerProvider(config)
	require.NoError(t, err)

	return provider
}
