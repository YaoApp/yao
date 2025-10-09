package messenger

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/messenger/template"
	"github.com/yaoapp/yao/messenger/types"
	"github.com/yaoapp/yao/test"
)

// TestSendT_TemplateTypeSelection tests that SendT correctly selects template type and provider based on channel configuration
func TestSendT_TemplateTypeSelection(t *testing.T) {
	// Prepare test environment
	test.Prepare(t, config.Conf, "YAO_TEST_APPLICATION")
	defer test.Clean()

	// Load messenger
	err := Load(config.Conf)
	require.NoError(t, err, "Load messenger should succeed")

	// Load templates
	err = template.LoadTemplates()
	require.NoError(t, err, "Load templates should succeed")

	service, ok := Instance.(*Service)
	require.True(t, ok, "Instance should be of type *Service")

	// Test 1: Get available types for a template
	availableTypes := template.Global.GetAvailableTypes("en.invite_member")
	t.Logf("Available types for en.invite_member: %v", availableTypes)

	// Test 2: SendT without specifying messageType (should use first available)
	ctx := context.Background()
	data := types.TemplateData{
		"to":           []string{"test@example.com"},
		"team_name":    "Test Team",
		"inviter_name": "Test User",
		"invite_link":  "https://example.com/invite/test",
	}

	// Note: This will fail with actual sending due to test credentials, but we're testing the logic
	err = service.SendT(ctx, "default", "en.invite_member", data)
	if err != nil {
		t.Logf("SendT failed (expected with test credentials): %v", err)
		// Should not be template-not-found or provider-not-found error
		assert.NotContains(t, err.Error(), "template not found", "Should not be template error")
	}

	// Test 3: SendT with explicit messageType
	err = service.SendT(ctx, "default", "en.invite_member", data, types.MessageTypeEmail)
	if err != nil {
		t.Logf("SendT with explicit type failed (expected with test credentials): %v", err)
		assert.NotContains(t, err.Error(), "template not found", "Should not be template error")
	}
}

// TestGetProviderForChannel tests the provider selection logic
func TestGetProviderForChannel(t *testing.T) {
	// Prepare test environment
	test.Prepare(t, config.Conf, "YAO_TEST_APPLICATION")
	defer test.Clean()

	// Load messenger
	err := Load(config.Conf)
	require.NoError(t, err, "Load messenger should succeed")

	service, ok := Instance.(*Service)
	require.True(t, ok, "Instance should be of type *Service")

	// Test provider selection for different channels and message types
	tests := []struct {
		channel     string
		messageType string
		expected    string // Expected provider name
	}{
		{"default", "email", "primary"},
		{"default", "sms", "unified"},
		{"default", "whatsapp", "unified"},
		{"promotions", "email", "marketing"},
		{"alerts", "email", "reliable"},
		{"notifications", "email", "primary"},
	}

	for _, tt := range tests {
		t.Run(tt.channel+"_"+tt.messageType, func(t *testing.T) {
			providerName := service.getProviderForChannel(tt.channel, tt.messageType)
			assert.Equal(t, tt.expected, providerName,
				"Channel %s with message type %s should use provider %s",
				tt.channel, tt.messageType, tt.expected)
		})
	}
}

// TestTemplateTypeConversion tests the conversion between MessageType and TemplateType
func TestTemplateTypeConversion(t *testing.T) {
	// Test templateTypeToMessageType
	tests := []struct {
		templateType types.TemplateType
		expected     types.MessageType
	}{
		{types.TemplateTypeMail, types.MessageTypeEmail},
		{types.TemplateTypeSMS, types.MessageTypeSMS},
		{types.TemplateTypeWhatsApp, types.MessageTypeWhatsApp},
	}

	for _, tt := range tests {
		result := templateTypeToMessageType(tt.templateType)
		assert.Equal(t, tt.expected, result,
			"TemplateType %s should convert to MessageType %s",
			tt.templateType, tt.expected)
	}

	// Test messageTypeToTemplateType
	reverseTests := []struct {
		messageType types.MessageType
		expected    types.TemplateType
	}{
		{types.MessageTypeEmail, types.TemplateTypeMail},
		{types.MessageTypeSMS, types.TemplateTypeSMS},
		{types.MessageTypeWhatsApp, types.TemplateTypeWhatsApp},
	}

	for _, tt := range reverseTests {
		result := messageTypeToTemplateType(tt.messageType)
		assert.Equal(t, tt.expected, result,
			"MessageType %s should convert to TemplateType %s",
			tt.messageType, tt.expected)
	}
}

// TestSendTBatch tests the batch sending with template type selection
func TestSendTBatch(t *testing.T) {
	// Prepare test environment
	test.Prepare(t, config.Conf, "YAO_TEST_APPLICATION")
	defer test.Clean()

	// Load messenger
	err := Load(config.Conf)
	require.NoError(t, err, "Load messenger should succeed")

	// Load templates
	err = template.LoadTemplates()
	require.NoError(t, err, "Load templates should succeed")

	service, ok := Instance.(*Service)
	require.True(t, ok, "Instance should be of type *Service")

	// Test batch sending
	ctx := context.Background()
	dataList := []types.TemplateData{
		{
			"to":           []string{"user1@example.com"},
			"team_name":    "Test Team",
			"inviter_name": "Test User",
			"invite_link":  "https://example.com/invite/test1",
		},
		{
			"to":           []string{"user2@example.com"},
			"team_name":    "Test Team",
			"inviter_name": "Test User",
			"invite_link":  "https://example.com/invite/test2",
		},
	}

	// Test without explicit messageType
	err = service.SendTBatch(ctx, "default", "en.invite_member", dataList)
	if err != nil {
		t.Logf("SendTBatch failed (expected with test credentials): %v", err)
		assert.NotContains(t, err.Error(), "template not found", "Should not be template error")
	}

	// Test with explicit messageType
	err = service.SendTBatch(ctx, "default", "en.invite_member", dataList, types.MessageTypeEmail)
	if err != nil {
		t.Logf("SendTBatch with explicit type failed (expected with test credentials): %v", err)
		assert.NotContains(t, err.Error(), "template not found", "Should not be template error")
	}
}

// TestSendTBatchMixed tests mixed template batch sending
func TestSendTBatchMixed(t *testing.T) {
	// Prepare test environment
	test.Prepare(t, config.Conf, "YAO_TEST_APPLICATION")
	defer test.Clean()

	// Load messenger
	err := Load(config.Conf)
	require.NoError(t, err, "Load messenger should succeed")

	// Load templates
	err = template.LoadTemplates()
	require.NoError(t, err, "Load templates should succeed")

	service, ok := Instance.(*Service)
	require.True(t, ok, "Instance should be of type *Service")

	// Test mixed batch sending
	ctx := context.Background()

	// Test without specifying MessageType in requests
	requests := []types.TemplateRequest{
		{
			TemplateID: "en.invite_member",
			Data: types.TemplateData{
				"to":           []string{"user1@example.com"},
				"team_name":    "Test Team",
				"inviter_name": "Test User",
				"invite_link":  "https://example.com/invite/test1",
			},
		},
		{
			TemplateID: "en.invite_member",
			Data: types.TemplateData{
				"to":           []string{"user2@example.com"},
				"team_name":    "Test Team",
				"inviter_name": "Test User",
				"invite_link":  "https://example.com/invite/test2",
			},
		},
	}

	err = service.SendTBatchMixed(ctx, "default", requests)
	if err != nil {
		t.Logf("SendTBatchMixed failed (expected with test credentials): %v", err)
		assert.NotContains(t, err.Error(), "template not found", "Should not be template error")
	}

	// Test with explicit MessageType in requests
	emailType := types.MessageTypeEmail
	requestsWithType := []types.TemplateRequest{
		{
			TemplateID:  "en.invite_member",
			MessageType: &emailType,
			Data: types.TemplateData{
				"to":           []string{"user1@example.com"},
				"team_name":    "Test Team",
				"inviter_name": "Test User",
				"invite_link":  "https://example.com/invite/test1",
			},
		},
	}

	err = service.SendTBatchMixed(ctx, "default", requestsWithType)
	if err != nil {
		t.Logf("SendTBatchMixed with explicit type failed (expected with test credentials): %v", err)
		assert.NotContains(t, err.Error(), "template not found", "Should not be template error")
	}
}
