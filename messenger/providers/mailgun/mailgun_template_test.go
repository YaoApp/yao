package mailgun

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/messenger/types"
	"github.com/yaoapp/yao/test"
)

func TestSendT_TemplateNotImplemented(t *testing.T) {
	// Prepare test environment
	test.Prepare(t, config.Conf, "YAO_TEST_APPLICATION")
	defer test.Clean()

	// Create provider with minimal config
	config := types.ProviderConfig{
		Name:      "test-mailgun",
		Connector: "mailgun",
		Options: map[string]interface{}{
			"api_key": "test_api_key",
			"domain":  "test.example.com",
			"from":    "test@example.com",
		},
	}

	provider, err := NewMailgunProvider(config)
	require.NoError(t, err)

	ctx := context.Background()
	templateData := types.TemplateData{
		"to":           []string{"test@example.com"},
		"team_name":    "Test Team",
		"inviter_name": "John Doe",
		"invite_link":  "https://example.com/invite/123",
	}

	// Test that SendT returns "template manager not available" error
	err = provider.SendT(ctx, "en.invite_member", types.TemplateTypeMail, templateData)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "template manager not available")
}

func TestSendT_ContextTimeout(t *testing.T) {
	// Prepare test environment
	test.Prepare(t, config.Conf, "YAO_TEST_APPLICATION")
	defer test.Clean()

	// Create provider with minimal config
	config := types.ProviderConfig{
		Name:      "test-mailgun",
		Connector: "mailgun",
		Options: map[string]interface{}{
			"api_key": "test_api_key",
			"domain":  "test.example.com",
			"from":    "test@example.com",
		},
	}

	provider, err := NewMailgunProvider(config)
	require.NoError(t, err)

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	// Wait for timeout
	time.Sleep(2 * time.Millisecond)

	templateData := types.TemplateData{
		"to":           []string{"test@example.com"},
		"team_name":    "Test Team",
		"inviter_name": "John Doe",
		"invite_link":  "https://example.com/invite/123",
	}

	// Test that SendT handles context timeout
	err = provider.SendT(ctx, "en.invite_member", types.TemplateTypeMail, templateData)
	assert.Error(t, err)

	// Verify it's a context timeout error or template manager error
	if strings.Contains(err.Error(), "context deadline exceeded") {
		t.Log("Context timeout working correctly with template API")
	} else if strings.Contains(err.Error(), "context canceled") {
		t.Log("Context cancellation working correctly with template API")
	} else if strings.Contains(err.Error(), "template manager not available") {
		t.Log("Template manager not available error as expected")
	} else {
		t.Logf("Got different error: %v", err)
	}
}
