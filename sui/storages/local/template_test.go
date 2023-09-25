package local

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTemplatePages(t *testing.T) {
	tests := prepare(t)
	defer clean()

	tmpl, err := tests.Demo.GetTemplate("website-ai")
	if err != nil {
		t.Fatalf("GetTemplate error: %v", err)
	}

	pages, err := tmpl.Pages()
	if err != nil {
		t.Fatalf("Pages error: %v", err)
	}

	if len(pages) < 1 {
		t.Fatalf("Pages error: %v", len(pages))
	}

	assert.Equal(t, "/index", pages[0].(*Page).Route)
	assert.Equal(t, "/templates/website-ai", pages[0].(*Page).Root)
	assert.Equal(t, "/index.css", pages[0].(*Page).Codes.CSS.File)
	assert.Equal(t, "/index.html", pages[0].(*Page).Codes.HTML.File)
	assert.Equal(t, "/index.js", pages[0].(*Page).Codes.JS.File)
	assert.Equal(t, "/index.less", pages[0].(*Page).Codes.LESS.File)
	assert.Equal(t, "/index.ts", pages[0].(*Page).Codes.TS.File)
	assert.Equal(t, "/index.json", pages[0].(*Page).Codes.DATA.File)
}

func TestTemplatePage(t *testing.T) {

	tests := prepare(t)
	defer clean()

	tmpl, err := tests.Demo.GetTemplate("website-ai")
	if err != nil {
		t.Fatalf("GetTemplate error: %v", err)
	}

	page, err := tmpl.Page("/index")
	if err != nil {
		t.Fatalf("Page error: %v", err)
	}
	assert.Equal(t, "/index", page.(*Page).Route)
	assert.Equal(t, "/templates/website-ai", page.(*Page).Root)
	assert.Equal(t, "/index.css", page.(*Page).Codes.CSS.File)
	assert.Equal(t, "/index.html", page.(*Page).Codes.HTML.File)
	assert.Equal(t, "/index.js", page.(*Page).Codes.JS.File)
	assert.Equal(t, "/index.less", page.(*Page).Codes.LESS.File)
	assert.Equal(t, "/index.ts", page.(*Page).Codes.TS.File)
	assert.Equal(t, "/index.json", page.(*Page).Codes.DATA.File)

	_, err = tmpl.Page("/the/page/could/not/be/found")
	assert.Contains(t, err.Error(), "Page /the/page/could/not/be/found not found")
}
