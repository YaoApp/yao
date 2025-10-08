package template

import (
	"testing"

	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/messenger/types"
	"github.com/yaoapp/yao/test"
)

func TestDebugTemplateLoading(t *testing.T) {
	// Prepare test environment
	test.Prepare(t, config.Conf, "YAO_TEST_APPLICATION")
	defer test.Clean()

	// Test direct template loading
	t.Log("Testing direct template loading...")

	// Try to load a specific template file using application.App
	content, err := application.App.Read("messengers/templates/en/invite_member.mail.html")
	if err != nil {
		t.Logf("Could not read template file: %v", err)
	} else {
		t.Logf("Template file content length: %d", len(content))
		t.Logf("First 200 chars: %s", content[:min(200, len(content))])
	}

	// Test template parsing
	subject, body, html, err := parseTemplateContent(string(content), types.TemplateTypeMail)
	if err != nil {
		t.Logf("Could not parse template: %v", err)
	} else {
		t.Logf("Parsed template content:")
		t.Logf("Subject: %s", subject)
		t.Logf("Body length: %d", len(body))
		t.Logf("HTML length: %d", len(html))
		t.Logf("Body content: %s", body)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
