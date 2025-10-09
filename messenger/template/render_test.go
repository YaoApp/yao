package template

import (
	"testing"

	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/messenger/types"
	"github.com/yaoapp/yao/test"
)

func TestTemplateRender(t *testing.T) {
	// Prepare test environment
	test.Prepare(t, config.Conf, "YAO_TEST_APPLICATION")
	defer test.Clean()

	// Load templates
	err := LoadTemplates()
	if err != nil {
		t.Fatalf("Failed to load templates: %v", err)
	}

	// Get template
	template, err := Global.GetTemplate("en.invite_member", types.TemplateTypeMail)
	if err != nil {
		t.Fatalf("Failed to get template: %v", err)
	}

	// Test data - matching actual template variables
	data := types.TemplateData{
		"to":              []string{"test@example.com"},
		"team_name":       "Awesome Team",
		"inviter_name":    "Alice Johnson",
		"invitation_link": "https://example.com/invite/abc123",
		"expires_at":      "2025-10-16 12:00:00 UTC",
	}

	// Test rendering
	subject, body, html, err := template.Render(data)
	if err != nil {
		t.Fatalf("Failed to render template: %v", err)
	}

	// Verify rendered content
	t.Logf("Rendered subject: %s", subject)
	t.Logf("Rendered body length: %d", len(body))
	t.Logf("Rendered HTML length: %d", len(html))

	// Check that variables were replaced
	if !contains(subject, "Awesome Team") {
		t.Errorf("Subject should contain 'Awesome Team', got: %s", subject)
	}
	if !contains(body, "Alice Johnson") {
		t.Errorf("Body should contain 'Alice Johnson', got: %s", body)
	}
	if !contains(body, "https://example.com/invite/abc123") {
		t.Errorf("Body should contain invite link, got: %s", body)
	}
}

func TestTemplateToMessage(t *testing.T) {
	// Prepare test environment
	test.Prepare(t, config.Conf, "YAO_TEST_APPLICATION")
	defer test.Clean()

	// Load templates
	err := LoadTemplates()
	if err != nil {
		t.Fatalf("Failed to load templates: %v", err)
	}

	// Get template
	template, err := Global.GetTemplate("en.invite_member", types.TemplateTypeMail)
	if err != nil {
		t.Fatalf("Failed to get template: %v", err)
	}

	// Test data - matching actual template variables
	data := types.TemplateData{
		"to":              []string{"test@example.com", "user@example.com"},
		"team_name":       "Awesome Team",
		"inviter_name":    "Alice Johnson",
		"invitation_link": "https://example.com/invite/abc123",
		"expires_at":      "2025-10-16 12:00:00 UTC",
	}

	// Convert template to message
	message, err := template.ToMessage(data)
	if err != nil {
		t.Fatalf("Failed to convert template to message: %v", err)
	}

	// Verify message properties - email type, not "mail"
	if message.Type != types.MessageTypeEmail {
		t.Errorf("Expected message type 'email', got %s", message.Type)
	}
	if len(message.To) != 2 {
		t.Errorf("Expected 2 recipients, got %d", len(message.To))
	}
	if !contains(message.Subject, "Awesome Team") {
		t.Errorf("Subject should contain 'Awesome Team', got: %s", message.Subject)
	}
	if !contains(message.Body, "Alice Johnson") {
		t.Errorf("Body should contain 'Alice Johnson', got: %s", message.Body)
	}

	t.Logf("Generated message: Subject=%s, To=%v", message.Subject, message.To)
}

func TestNestedTemplateRender(t *testing.T) {
	// Test nested object access
	template := &types.Template{
		Subject: "Hello {{ user.name }}, welcome to {{ team.name }}!",
		Body:    "Your role is {{ user.role }} in {{ team.department.name }}.",
	}

	data := types.TemplateData{
		"user": map[string]interface{}{
			"name": "John Doe",
			"role": "Developer",
		},
		"team": map[string]interface{}{
			"name": "Awesome Team",
			"department": map[string]interface{}{
				"name": "Engineering",
			},
		},
	}

	subject, body, _, err := template.Render(data)
	if err != nil {
		t.Fatalf("Failed to render template: %v", err)
	}

	// Verify nested access works
	if !contains(subject, "John Doe") {
		t.Errorf("Subject should contain 'John Doe', got: %s", subject)
	}
	if !contains(subject, "Awesome Team") {
		t.Errorf("Subject should contain 'Awesome Team', got: %s", subject)
	}
	if !contains(body, "Developer") {
		t.Errorf("Body should contain 'Developer', got: %s", body)
	}
	if !contains(body, "Engineering") {
		t.Errorf("Body should contain 'Engineering', got: %s", body)
	}

	t.Logf("Nested render - Subject: %s", subject)
	t.Logf("Nested render - Body: %s", body)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			contains(s[1:], substr))))
}
