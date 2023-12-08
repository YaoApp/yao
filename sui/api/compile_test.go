package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/sui/core"
)

func TestCompile(t *testing.T) {
	prepare(t)
	defer clean()
	loadTestSui(t)

	page := testPage(t)
	html, err := page.Compile(&core.BuildOption{KeepPageTag: false})
	if err != nil {
		t.Fatalf("Compile error: %v", err)
	}
	assert.Contains(t, html, `<a href="Link2">Link2</a>`)
	assert.Contains(t, html, `<a href="Link">Link</a>`)
	assert.Contains(t, html, "input.data")
}

func testPage(t *testing.T) *core.Page {

	sui := core.SUIs["demo"]
	if sui == nil {
		t.Fatal("SUI demo not found")
	}

	tmpl, err := sui.GetTemplate("tech-blue")
	if err != nil {
		t.Fatal(err)
	}

	page, err := tmpl.Page("/index")
	if err != nil {
		t.Fatal(err)
	}

	return page.Get()
}
