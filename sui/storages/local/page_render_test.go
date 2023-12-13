package local

import (
	"testing"

	"github.com/stretchr/testify/assert"
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

	res, err := page.EditorRender()
	if err != nil {
		t.Fatalf("EditorRender error: %v", err)
	}

	assert.NotEmpty(t, res.HTML)
	assert.NotEmpty(t, res.CSS)
	assert.NotEmpty(t, res.Scripts)
	assert.NotEmpty(t, res.Styles)
	assert.Equal(t, 3, len(res.Scripts))
	assert.Equal(t, 4, len(res.Styles))

	assert.Equal(t, "@assets/libs/tiny-slider/min/tiny-slider.js", res.Scripts[0])
	assert.Equal(t, "@assets/libs/feather-icons/feather.min.js", res.Scripts[1])
	assert.Equal(t, "@assets/js/plugins.init.js", res.Scripts[2])

	assert.Equal(t, "@assets/libs/tiny-slider/tiny-slider.css", res.Styles[0])
	assert.Equal(t, "@assets/libs/@iconscout/unicons/css/line.css", res.Styles[1])
	assert.Equal(t, "@assets/libs/@mdi/font/css/materialdesignicons.min.css", res.Styles[2])
	assert.Equal(t, "@assets/css/tailwind.css", res.Styles[3])
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

	html, err := page.PreviewRender("")
	if err != nil {
		t.Fatalf("PreviewRender error: %v", err)
	}

	assert.NotEmpty(t, html)
	assert.Contains(t, html, "function Hello()")
	// assert.Contains(t, html, "color: #2c3e50;")
	assert.Contains(t, html, "/api/__yao/sui/v1/demo/asset/tech-blue/@assets")
}
