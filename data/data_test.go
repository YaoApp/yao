package data

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReplaceXGenIndex(t *testing.T) {
	err := ReplaceXGen("__yao_admin_root", "Admin-Replaced")
	if err != nil {
		t.Fatal(err)
	}

	content, err := bindataRead(_xgenV10IndexHtml, "index.html")
	if err != nil {
		t.Fatal(err)
	}

	content, err = bindataRead(_xgenV10UmiJs, "umi.js")
	if err != nil {
		t.Fatal(err)
	}
	assert.Contains(t, string(content), "Admin-Replaced")
}
