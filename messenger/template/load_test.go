package template

import (
	"testing"

	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/share"
	"github.com/yaoapp/yao/test"
)

func TestLoadTemplate(t *testing.T) {
	// Prepare test environment
	test.Prepare(t, config.Conf, "YAO_TEST_APPLICATION")
	defer test.Clean()

	// Test loading a specific template
	file := "messengers/templates/en/invite_member.mail.html"
	templateID := share.ID("messengers/templates", file)

	t.Logf("Testing loadTemplate with file: %s, templateID: %s", file, templateID)

	template, err := loadTemplate(file, templateID)
	if err != nil {
		t.Fatalf("Failed to load template: %v", err)
	}

	if template == nil {
		t.Fatal("Template is nil")
	}

	t.Logf("Loaded template: ID=%s, Type=%s, Language=%s", template.ID, template.Type, template.Language)
	t.Logf("Subject: %s", template.Subject)
	t.Logf("Body length: %d", len(template.Body))
	t.Logf("HTML length: %d", len(template.HTML))
	t.Logf("Body content: %s", template.Body)
}
