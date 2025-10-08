package template

import (
	"testing"

	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/test"
)

func TestWalkTemplates(t *testing.T) {
	// Prepare test environment
	test.Prepare(t, config.Conf, "YAO_TEST_APPLICATION")
	defer test.Clean()

	// Test Walk function directly
	t.Log("Testing Walk function directly...")

	// Check if templates directory exists
	exists, err := application.App.Exists("messengers/templates")
	if err != nil {
		t.Fatalf("Error checking templates directory: %v", err)
	}
	if !exists {
		t.Log("Templates directory not found")
		return
	}

	t.Log("Templates directory exists")

	// Test Walk with different extensions
	exts := []string{"*.mail.html", "*.sms.txt", "*.whatsapp.html"}
	t.Logf("Testing Walk with extensions: %v", exts)

	fileCount := 0
	err = application.App.Walk("messengers/templates", func(root, file string, isdir bool) error {
		t.Logf("Walk callback: root=%s, file=%s, isdir=%v", root, file, isdir)
		if !isdir {
			fileCount++
		}
		return nil
	}, exts...)

	if err != nil {
		t.Fatalf("Walk failed: %v", err)
	}

	t.Logf("Walk completed, found %d files", fileCount)
}
