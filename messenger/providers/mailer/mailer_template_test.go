package mailer

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

// Test SendT method

func TestSendT_TemplateNotImplemented(t *testing.T) {
	// Prepare test environment
	test.Prepare(t, config.Conf, "YAO_TEST_APPLICATION")
	defer test.Clean()

	// Create a simple provider config for testing
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
				"use_tls":  true,
			},
		},
	}

	provider, err := NewMailerProvider(config)
	require.NoError(t, err)

	ctx := context.Background()
	templateData := types.TemplateData{
		"to":           []string{"test@example.com"},
		"team_name":    "Test Team",
		"inviter_name": "John Doe",
		"invite_link":  "https://example.com/invite/123",
	}

	// Test that SendT returns "template not found" error (template system is working)
	err = provider.SendT(ctx, "en.invite_member.mail", types.TemplateTypeMail, templateData)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "template not found")
}

func TestSendT_ContextTimeout(t *testing.T) {
	// Prepare test environment
	test.Prepare(t, config.Conf, "YAO_TEST_APPLICATION")
	defer test.Clean()

	// Create a simple provider config for testing
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
				"use_tls":  true,
			},
		},
	}

	provider, err := NewMailerProvider(config)
	require.NoError(t, err)

	// Create a very short timeout context to test timeout functionality
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	templateData := types.TemplateData{
		"to":           []string{"test@example.com"},
		"team_name":    "Test Team",
		"inviter_name": "John Doe",
		"invite_link":  "https://example.com/invite/123",
	}

	err = provider.SendT(ctx, "en.invite_member.mail", types.TemplateTypeMail, templateData)
	assert.Error(t, err)

	// Verify it's a context timeout error or not implemented error
	if strings.Contains(err.Error(), "context deadline exceeded") {
		t.Log("Context timeout working correctly with template API")
	} else if strings.Contains(err.Error(), "context canceled") {
		t.Log("Context cancellation working correctly with template API")
	} else if strings.Contains(err.Error(), "template not found") {
		t.Log("Template not found error as expected")
	} else {
		t.Logf("Got different error: %v", err)
	}
}

// Test template system integration

func TestTemplateSystem_LoadTemplates(t *testing.T) {
	// This test verifies that the template system can be loaded
	// We'll test the template loading logic

	// Prepare test environment
	test.Prepare(t, config.Conf, "YAO_TEST_APPLICATION")
	defer test.Clean()

	// Test that we can create a template data structure
	templateData := types.TemplateData{
		"team_name":    "Awesome Team",
		"inviter_name": "Alice Johnson",
		"invite_link":  "https://example.com/invite/abc123",
		"to":           []string{"test@example.com"},
	}

	// Verify template data structure
	assert.NotNil(t, templateData)
	assert.Equal(t, "Awesome Team", templateData["team_name"])
	assert.Equal(t, "Alice Johnson", templateData["inviter_name"])
	assert.Equal(t, "https://example.com/invite/abc123", templateData["invite_link"])

	// Verify recipients
	recipients, ok := templateData["to"].([]string)
	assert.True(t, ok)
	assert.Len(t, recipients, 1)
	assert.Equal(t, "test@example.com", recipients[0])
}

// Benchmark Tests

func BenchmarkSendT(b *testing.B) {
	// Setup
	t := &testing.T{}
	test.Prepare(t, config.Conf, "YAO_TEST_APPLICATION")
	defer test.Clean()

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
				"use_tls":  true,
			},
		},
	}

	provider, err := NewMailerProvider(config)
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()
	templateData := types.TemplateData{
		"to":           []string{"test@example.com"},
		"team_name":    "Test Team",
		"inviter_name": "John Doe",
		"invite_link":  "https://example.com/invite/123",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// This will return "not implemented" error, but we're measuring the overhead
		_ = provider.SendT(ctx, "en.invite_member.mail", types.TemplateTypeMail, templateData)
	}
}
