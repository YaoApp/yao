package local

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/sui/core"
)

func TestPageEditorRender(t *testing.T) {

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
	res, err := page.EditorRender(r)
	if err != nil {
		t.Fatalf("EditorRender error: %v", err)
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

func TestPagePreviewRender(t *testing.T) {

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

	r := &core.Request{
		Method:    "GET",
		AssetRoot: fmt.Sprintf("/api/__yao/sui/v1/%s/asset/%s/@assets", page.Get().SuiID, page.Get().TemplateID),
	}

	html, err := page.PreviewRender(r)
	if err != nil {
		t.Fatalf("PreviewRender error: %v", err)
	}

	assert.NotEmpty(t, html)
	assert.Contains(t, html, "function Hello()")
	assert.Contains(t, html, "color: #2c3e50;")
	assert.Contains(t, html, "/api/__yao/sui/v1/demo/asset/tech-blue/@assets")
}
