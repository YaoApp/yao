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
	assert.Equal(t, "/index.css", pages[0].(*Page).Files.CSS)
	assert.Equal(t, "/index.html", pages[0].(*Page).Files.HTML)
	assert.Equal(t, "/index.js", pages[0].(*Page).Files.JS)
	assert.Equal(t, "/index.less", pages[0].(*Page).Files.LESS)
	assert.Equal(t, "/index.ts", pages[0].(*Page).Files.TS)
	assert.Equal(t, "/index.json", pages[0].(*Page).Files.DATA)
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
	assert.Equal(t, "/index.css", page.(*Page).Files.CSS)
	assert.Equal(t, "/index.html", page.(*Page).Files.HTML)
	assert.Equal(t, "/index.js", page.(*Page).Files.JS)
	assert.Equal(t, "/index.less", page.(*Page).Files.LESS)
	assert.Equal(t, "/index.ts", page.(*Page).Files.TS)
	assert.Equal(t, "/index.json", page.(*Page).Files.DATA)

	_, err = tmpl.Page("/the/page/could/not/be/found")
	assert.Contains(t, err.Error(), "Page /the/page/could/not/be/found not found")
}
