//go:build unit

package text_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/agent/content/text"
)

func TestIsSupportedExtension(t *testing.T) {
	supported := []string{
		"test.md", "test.txt", "test.go", "test.ts", "test.json",
		"test.jsonc", "test.yao", "test.yaml", "test.yml",
		"test.py", "test.js", "test.css", "test.html",
	}
	for _, f := range supported {
		assert.True(t, text.IsSupportedExtension(f), "expected %s to be supported", f)
	}

	unsupported := []string{
		"test.docx", "test.pptx", "test.pdf", "test.png",
		"test.jpg", "test.exe", "test.zip",
	}
	for _, f := range unsupported {
		assert.False(t, text.IsSupportedExtension(f), "expected %s to be unsupported", f)
	}
}
