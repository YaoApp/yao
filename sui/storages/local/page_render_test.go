package local

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPageEditorRender(t *testing.T) {

	tests := prepare(t)
	defer clean()

	tmpl, err := tests.Test.GetTemplate("advanced")
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
	// assert.NotEmpty(t, res.Scripts)
	assert.NotEmpty(t, res.Styles)
	assert.GreaterOrEqual(t, len(res.Styles), 1)
	// assert.GreaterOrEqual(t, len(res.Scripts), 1)

}

func TestPagePreviewRender(t *testing.T) {

	tests := prepare(t)
	defer clean()

	tmpl, err := tests.Test.GetTemplate("advanced")
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
	assert.Contains(t, html, "var __sui_data")
	assert.Contains(t, html, "/api/__yao/sui/v1/test/asset/advanced/@assets")
}
