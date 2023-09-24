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

func TestTemplateBlocks(t *testing.T) {
	tests := prepare(t)
	defer clean()

	tmpl, err := tests.Demo.GetTemplate("tech-blue")
	if err != nil {
		t.Fatalf("GetTemplate error: %v", err)
	}

	blocks, err := tmpl.Blocks()
	if err != nil {
		t.Fatalf("Blocks error: %v", err)
	}

	if len(blocks) < 3 {
		t.Fatalf("Blocks error: %v", len(blocks))
	}

	assert.Equal(t, "columns-two", blocks[0].(*Block).ID)
	assert.Equal(t, "/columns-two/main.html", blocks[0].(*Block).Codes.HTML.File)
	assert.Equal(t, "/columns-two/main.js", blocks[0].(*Block).Codes.JS.File)
	assert.Equal(t, "/columns-two/main.ts", blocks[0].(*Block).Codes.TS.File)

	assert.Equal(t, "hero", blocks[1].(*Block).ID)
	assert.Equal(t, "/hero/main.html", blocks[1].(*Block).Codes.HTML.File)
	assert.Equal(t, "/hero/main.js", blocks[1].(*Block).Codes.JS.File)
	assert.Equal(t, "/hero/main.ts", blocks[1].(*Block).Codes.TS.File)

	assert.Equal(t, "section", blocks[2].(*Block).ID)
	assert.Equal(t, "/section/main.html", blocks[2].(*Block).Codes.HTML.File)
	assert.Equal(t, "/section/main.js", blocks[2].(*Block).Codes.JS.File)
	assert.Equal(t, "/section/main.ts", blocks[2].(*Block).Codes.TS.File)
}

func TestTemplateBlockJS(t *testing.T) {
	tests := prepare(t)
	defer clean()

	tmpl, err := tests.Demo.GetTemplate("tech-blue")
	if err != nil {
		t.Fatalf("GetTemplate error: %v", err)
	}

	block, err := tmpl.Block("columns-two")
	if err != nil {
		t.Fatalf("Blocks error: %v", err)
	}

	assert.Equal(t, "columns-two", block.(*Block).ID)
	assert.NotEmpty(t, block.(*Block).Codes.HTML.Code)
	assert.NotEmpty(t, block.(*Block).Codes.JS.Code)
	assert.Contains(t, block.(*Block).Compiled, "window.block__columns_two")
	assert.Contains(t, block.(*Block).Compiled, `<div class="columns-two-left"`)
}

func TestTemplateBlockTS(t *testing.T) {
	tests := prepare(t)
	defer clean()

	tmpl, err := tests.Demo.GetTemplate("tech-blue")
	if err != nil {
		t.Fatalf("GetTemplate error: %v", err)
	}

	block, err := tmpl.Block("hero")
	if err != nil {
		t.Fatalf("Blocks error: %v", err)
	}

	assert.Equal(t, "hero", block.(*Block).ID)
	assert.Empty(t, block.(*Block).Codes.HTML.Code)
	assert.NotEmpty(t, block.(*Block).Codes.TS.Code)
	assert.Contains(t, block.(*Block).Compiled, "window.block__hero")
	assert.Contains(t, block.(*Block).Compiled, `<div data-gjs-type='nav'></div>`)
}
