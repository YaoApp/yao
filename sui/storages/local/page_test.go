package local

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/sui/core"
)

func TestTemplatePages(t *testing.T) {
	tests := prepare(t)
	defer clean()

	tmpl, err := tests.Demo.GetTemplate("tech-blue")
	if err != nil {
		t.Fatalf("GetTemplate error: %v", err)
	}

	pages, err := tmpl.Pages()
	if err != nil {
		t.Fatalf("Pages error: %v", err)
	}

	if len(pages) < 8 {
		t.Fatalf("Pages error: %v", len(pages))
	}

	for _, page := range pages {

		page := page.(*Page)
		name := filepath.Base(page.Path)
		dir := page.Path[len(tmpl.(*Template).Root):]
		path := filepath.Join(tmpl.(*Template).Root, dir)

		assert.Equal(t, dir, page.Route)
		assert.Equal(t, path, page.Path)
		assert.Equal(t, name+".css", page.Codes.CSS.File)
		assert.Equal(t, name+".html", page.Codes.HTML.File)
		assert.Equal(t, name+".js", page.Codes.JS.File)
		assert.Equal(t, name+".less", page.Codes.LESS.File)
		assert.Equal(t, name+".ts", page.Codes.TS.File)
		assert.Equal(t, name+".json", page.Codes.DATA.File)
	}
}

func TestTemplatePageTS(t *testing.T) {

	tests := prepare(t)
	defer clean()

	tmpl, err := tests.Demo.GetTemplate("tech-blue")
	if err != nil {
		t.Fatalf("GetTemplate error: %v", err)
	}

	ipage, err := tmpl.Page("/page/[id]")
	if err != nil {
		t.Fatalf("Page error: %v", err)
	}

	page := ipage.(*Page)

	assert.Equal(t, "/page/[id]", page.Route)
	assert.Equal(t, "/templates/tech-blue/page/[id]", page.Path)
	assert.Equal(t, "[id].css", page.Codes.CSS.File)
	assert.Equal(t, "[id].html", page.Codes.HTML.File)
	assert.Equal(t, "[id].js", page.Codes.JS.File)
	assert.Equal(t, "[id].less", page.Codes.LESS.File)
	assert.Equal(t, "[id].ts", page.Codes.TS.File)
	assert.Equal(t, "[id].json", page.Codes.DATA.File)

	assert.NotEmpty(t, page.Codes.TS.Code)
	assert.Empty(t, page.Codes.JS.Code)
	assert.NotEmpty(t, page.Codes.HTML.Code)
	assert.NotEmpty(t, page.Codes.CSS.Code)
	assert.NotEmpty(t, page.Codes.DATA.Code)

	_, err = tmpl.Page("/the/page/could/not/be/found")
	assert.Contains(t, err.Error(), "Page /the/page/could/not/be/found not found")
}

func TestTemplatePageJS(t *testing.T) {

	tests := prepare(t)
	defer clean()

	tmpl, err := tests.Demo.GetTemplate("tech-blue")
	if err != nil {
		t.Fatalf("GetTemplate error: %v", err)
	}

	ipage, err := tmpl.Page("/page/404")
	if err != nil {
		t.Fatalf("Page error: %v", err)
	}

	page := ipage.(*Page)
	assert.Equal(t, "/page/404", page.Route)
	assert.Equal(t, "/templates/tech-blue/page/404", page.Path)
	assert.Equal(t, "404.css", page.Codes.CSS.File)
	assert.Equal(t, "404.html", page.Codes.HTML.File)
	assert.Equal(t, "404.js", page.Codes.JS.File)
	assert.Equal(t, "404.less", page.Codes.LESS.File)
	assert.Equal(t, "404.ts", page.Codes.TS.File)
	assert.Equal(t, "404.json", page.Codes.DATA.File)

	assert.NotEmpty(t, page.Codes.JS.Code)
	assert.Empty(t, page.Codes.TS.Code)
	assert.NotEmpty(t, page.Codes.HTML.Code)
	assert.Empty(t, page.Codes.CSS.Code)
	assert.NotEmpty(t, page.Codes.DATA.Code)

	_, err = tmpl.Page("/the/page/could/not/be/found")
	assert.Contains(t, err.Error(), "Page /the/page/could/not/be/found not found")
}

func TestPageRenderEditor(t *testing.T) {

	tests := prepare(t)
	defer clean()

	tmpl, err := tests.Demo.GetTemplate("tech-blue")
	if err != nil {
		t.Fatalf("GetTemplate error: %v", err)
	}

	page, err := tmpl.Page("/index")
	if err != nil {
		t.Fatalf("Page error: %v", err)
	}

	r := &core.Request{Method: "GET"}
	res, err := page.RenderEditor(r)
	if err != nil {
		t.Fatalf("RenderEditor error: %v", err)
	}

	assert.NotEmpty(t, res.HTML)
	assert.NotEmpty(t, res.CSS)
	assert.NotEmpty(t, res.Scripts)
	assert.NotEmpty(t, res.Styles)
	assert.Equal(t, 4, len(res.Scripts))
	assert.Equal(t, 5, len(res.Styles))

	assert.Equal(t, "@assets/libs/tiny-slider/min/tiny-slider.js", res.Scripts[0])
	assert.Equal(t, "@assets/libs/feather-icons/feather.min.js", res.Scripts[1])
	assert.Equal(t, "@assets/js/plugins.init.js", res.Scripts[2])
	assert.Equal(t, "@pages/index/index.js", res.Scripts[3])

	assert.Equal(t, "@assets/libs/tiny-slider/tiny-slider.css", res.Styles[0])
	assert.Equal(t, "@assets/libs/@iconscout/unicons/css/line.css", res.Styles[1])
	assert.Equal(t, "@assets/libs/@mdi/font/css/materialdesignicons.min.css", res.Styles[2])
	assert.Equal(t, "@assets/css/tailwind.css", res.Styles[3])
	assert.Equal(t, "@pages/index/index.css", res.Styles[4])
}
