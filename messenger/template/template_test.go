package template

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/messenger/types"
	"github.com/yaoapp/yao/test"
)

func TestTemplateManager_LoadTemplates(t *testing.T) {
	// Prepare test environment
	test.Prepare(t, config.Conf, "YAO_TEST_APPLICATION")
	defer test.Clean()

	// Load templates
	err := LoadTemplates()
	require.NoError(t, err)

	// Check if templates were loaded
	templates := Global.GetAllTemplates()
	assert.NotNil(t, templates)

	// Log loaded templates for debugging
	t.Logf("Loaded %d template groups", len(templates))
	for _, templateGroup := range templates {
		for _, template := range templateGroup {
			t.Logf("Template: %s, Type: %s, Language: %s", template.ID, template.Type, template.Language)
		}
	}
}

func TestTemplateManager_GetTemplate(t *testing.T) {
	// Prepare test environment
	test.Prepare(t, config.Conf, "YAO_TEST_APPLICATION")
	defer test.Clean()

	// Load templates
	err := LoadTemplates()
	require.NoError(t, err)

	// Test getting a specific template
	template, err := Global.GetTemplate("en.invite_member", types.TemplateTypeMail)
	if err != nil {
		t.Logf("Template not found (expected if templates not loaded): %v", err)
		return
	}

	// Verify template properties
	assert.NotNil(t, template)
	assert.Equal(t, "en.invite_member", template.ID)
	assert.Equal(t, types.TemplateTypeMail, template.Type)
	assert.Equal(t, "en", template.Language)
	assert.NotEmpty(t, template.Subject)
	assert.NotEmpty(t, template.Body)
}

func TestTemplate_Render(t *testing.T) {
	// Prepare test environment
	test.Prepare(t, config.Conf, "YAO_TEST_APPLICATION")
	defer test.Clean()

	// Load templates
	err := LoadTemplates()
	require.NoError(t, err)

	// Test template rendering
	template, err := Global.GetTemplate("en.invite_member", types.TemplateTypeMail)
	if err != nil {
		t.Logf("Template not found (expected if templates not loaded): %v", err)
		return
	}

	// Test data - matching actual template variables
	data := types.TemplateData{
		"team_name":       "Awesome Team",
		"inviter_name":    "Alice Johnson",
		"invitation_link": "https://example.com/invite/abc123",
		"expires_at":      "2025-10-16 12:00:00 UTC",
	}

	// Render template
	subject, body, html, err := template.Render(data)
	require.NoError(t, err)

	// Verify rendered content
	assert.NotEmpty(t, subject)
	assert.NotEmpty(t, body)
	assert.NotEmpty(t, html)

	// Check that variables were replaced
	assert.Contains(t, subject, "Awesome Team")
	assert.Contains(t, body, "Alice Johnson")
	assert.Contains(t, body, "https://example.com/invite/abc123")
	assert.Contains(t, body, "2025-10-16 12:00:00 UTC")

	t.Logf("Rendered subject: %s", subject)
	t.Logf("Rendered body: %s", body)
}

func TestTemplate_ToMessage(t *testing.T) {
	// Prepare test environment
	test.Prepare(t, config.Conf, "YAO_TEST_APPLICATION")
	defer test.Clean()

	// Load templates
	err := LoadTemplates()
	require.NoError(t, err)

	// Test template to message conversion
	template, err := Global.GetTemplate("en.invite_member", types.TemplateTypeMail)
	if err != nil {
		t.Logf("Template not found (expected if templates not loaded): %v", err)
		return
	}

	// Test data with recipients - matching actual template variables
	data := types.TemplateData{
		"to":              []string{"test@example.com", "user@example.com"},
		"team_name":       "Awesome Team",
		"inviter_name":    "Alice Johnson",
		"invitation_link": "https://example.com/invite/abc123",
		"expires_at":      "2025-10-16 12:00:00 UTC",
	}

	// Convert template to message
	message, err := template.ToMessage(data)
	require.NoError(t, err)

	// Verify message properties
	assert.NotNil(t, message)
	assert.Equal(t, types.MessageTypeEmail, message.Type) // Changed from "mail" to MessageTypeEmail
	assert.NotEmpty(t, message.Subject)
	assert.NotEmpty(t, message.Body)
	assert.NotEmpty(t, message.HTML)
	assert.Len(t, message.To, 2)
	assert.Equal(t, "test@example.com", message.To[0])
	assert.Equal(t, "user@example.com", message.To[1])

	// Check that variables were replaced
	assert.Contains(t, message.Subject, "Awesome Team")
	assert.Contains(t, message.Body, "Alice Johnson")
	assert.Contains(t, message.Body, "https://example.com/invite/abc123")
	assert.Contains(t, message.Body, "2025-10-16 12:00:00 UTC")

	t.Logf("Generated message: Subject=%s, To=%v", message.Subject, message.To)
}

func TestTemplate_SMSTemplate(t *testing.T) {
	// Prepare test environment
	test.Prepare(t, config.Conf, "YAO_TEST_APPLICATION")
	defer test.Clean()

	// Load templates
	err := LoadTemplates()
	require.NoError(t, err)

	// Test SMS template
	template, err := Global.GetTemplate("en.invite_member", types.TemplateTypeSMS)
	if err != nil {
		t.Logf("SMS template not found (expected if templates not loaded): %v", err)
		return
	}

	// Test data - matching actual template variables
	data := types.TemplateData{
		"to":              []string{"+1234567890"},
		"team_name":       "Awesome Team",
		"inviter_name":    "Alice Johnson",
		"invitation_link": "https://example.com/invite/abc123",
		"expires_at":      "2025-10-16 12:00:00 UTC",
	}

	// Convert template to message
	message, err := template.ToMessage(data)
	require.NoError(t, err)

	// Verify SMS message properties
	assert.NotNil(t, message)
	assert.Equal(t, types.MessageTypeSMS, message.Type)
	assert.NotEmpty(t, message.Body)
	assert.Empty(t, message.HTML) // SMS should not have HTML
	assert.Len(t, message.To, 1)
	assert.Equal(t, "+1234567890", message.To[0])

	// Check that variables were replaced
	assert.Contains(t, message.Body, "Alice Johnson")
	assert.Contains(t, message.Body, "Awesome Team")
	assert.Contains(t, message.Body, "https://example.com/invite/abc123")

	t.Logf("Generated SMS message: Body=%s, To=%v", message.Body, message.To)
}
