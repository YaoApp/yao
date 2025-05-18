package data

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReplaceCUIIndex(t *testing.T) {
	err := ReplaceCUI("__yao_admin_root", "Admin-Replaced")
	if err != nil {
		t.Fatal(err)
	}

	content, err := bindataRead(_cuiV10IndexHtml, "index.html")
	if err != nil {
		t.Fatal(err)
	}

	content, err = bindataRead(_cuiV10UmiJs, "umi.js")
	if err != nil {
		t.Fatal(err)
	}
	assert.Contains(t, string(content), "Admin-Replaced")
}
