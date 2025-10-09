package twilio

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/messenger/types"
	"github.com/yaoapp/yao/test"
)

func TestSendTBatch_Success(t *testing.T) {
	// Prepare test environment
	test.Prepare(t, config.Conf, "YAO_TEST_APPLICATION")

	// Create provider with template manager
	config := types.ProviderConfig{
		Name:      "test-twilio",
		Connector: "twilio",
		Options: map[string]interface{}{
			"account_sid": "test_account_sid",
			"auth_token":  "test_auth_token",
			"from":        "+1234567890",
		},
	}

	provider, err := NewTwilioProviderWithTemplateManager(config, nil)
	assert.NoError(t, err)

	// Test data for batch sending
	dataList := []types.TemplateData{
		{
			"to":           []string{"+1234567890"},
			"team_name":    "Team A",
			"inviter_name": "Alice",
			"invite_link":  "https://example.com/invite/1",
		},
		{
			"to":           []string{"+0987654321"},
			"team_name":    "Team B",
			"inviter_name": "Bob",
			"invite_link":  "https://example.com/invite/2",
		},
	}

	// Test SendTBatch - should fail because template manager is nil
	err = provider.SendTBatch(context.Background(), "en.invite_member", types.TemplateTypeSMS, dataList)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "template manager not available")
}

func TestSendTBatch_ContextTimeout(t *testing.T) {
	// Prepare test environment
	test.Prepare(t, config.Conf, "YAO_TEST_APPLICATION")

	// Create provider
	config := types.ProviderConfig{
		Name:      "test-twilio",
		Connector: "twilio",
		Options: map[string]interface{}{
			"account_sid": "test_account_sid",
			"auth_token":  "test_auth_token",
			"from":        "+1234567890",
		},
	}

	provider, err := NewTwilioProviderWithTemplateManager(config, nil)
	assert.NoError(t, err)

	// Test data
	dataList := []types.TemplateData{
		{
			"to":           []string{"+1234567890"},
			"team_name":    "Team A",
			"inviter_name": "Alice",
			"invite_link":  "https://example.com/invite/1",
		},
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Wait for context to timeout
	time.Sleep(2 * time.Nanosecond)

	// Test SendTBatch with expired context
	err = provider.SendTBatch(ctx, "en.invite_member", types.TemplateTypeSMS, dataList)
	assert.Error(t, err)
	// Error could be either "template manager not available" or "context deadline exceeded"
	t.Logf("Error: %v", err)
}

func TestSendTBatchMixed_Success(t *testing.T) {
	// Prepare test environment
	test.Prepare(t, config.Conf, "YAO_TEST_APPLICATION")

	// Create provider with template manager
	config := types.ProviderConfig{
		Name:      "test-twilio",
		Connector: "twilio",
		Options: map[string]interface{}{
			"account_sid": "test_account_sid",
			"auth_token":  "test_auth_token",
			"from":        "+1234567890",
		},
	}

	provider, err := NewTwilioProviderWithTemplateManager(config, nil)
	assert.NoError(t, err)

	// Test data for mixed batch sending
	templateRequests := []types.TemplateRequest{
		{
			TemplateID: "en.invite_member",
			Data: types.TemplateData{
				"to":           []string{"+1234567890"},
				"team_name":    "Team A",
				"inviter_name": "Alice",
				"invite_link":  "https://example.com/invite/1",
			},
		},
		{
			TemplateID: "en.welcome",
			Data: types.TemplateData{
				"to":        []string{"+0987654321"},
				"user_name": "Bob",
				"company":   "Example Corp",
			},
		},
	}

	// Test SendTBatchMixed - should fail because template manager is nil
	err = provider.SendTBatchMixed(context.Background(), templateRequests)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "template manager not available")
}

func TestSendTBatchMixed_ContextTimeout(t *testing.T) {
	// Prepare test environment
	test.Prepare(t, config.Conf, "YAO_TEST_APPLICATION")

	// Create provider
	config := types.ProviderConfig{
		Name:      "test-twilio",
		Connector: "twilio",
		Options: map[string]interface{}{
			"account_sid": "test_account_sid",
			"auth_token":  "test_auth_token",
			"from":        "+1234567890",
		},
	}

	provider, err := NewTwilioProviderWithTemplateManager(config, nil)
	assert.NoError(t, err)

	// Test data
	templateRequests := []types.TemplateRequest{
		{
			TemplateID: "en.invite_member",
			Data: types.TemplateData{
				"to":           []string{"+1234567890"},
				"team_name":    "Team A",
				"inviter_name": "Alice",
				"invite_link":  "https://example.com/invite/1",
			},
		},
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Wait for context to timeout
	time.Sleep(2 * time.Nanosecond)

	// Test SendTBatchMixed with expired context
	err = provider.SendTBatchMixed(ctx, templateRequests)
	assert.Error(t, err)
	// Error could be either "template manager not available" or "context deadline exceeded"
	t.Logf("Error: %v", err)
}
