package local

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/application"
)

func TestTemplateThemes(t *testing.T) {
	tests := prepare(t)
	defer clean()

	tmpl, err := tests.Test.GetTemplate("advanced")
	if err != nil {
		t.Fatalf("GetTemplate error: %v", err)
	}

	themes := tmpl.Themes()
	if len(themes) != 2 {
		t.Fatalf("Themes error: %v", len(themes))
	}

	assert.Equal(t, "light", themes[0].Value)
	assert.Equal(t, "Light", themes[0].Label)
	assert.Equal(t, "dark", themes[1].Value)
	assert.Equal(t, "Dark", themes[1].Label)
}

func TestTemplateLocales(t *testing.T) {
	tests := prepare(t)
	defer clean()

	tmpl, err := tests.Test.GetTemplate("advanced")
	if err != nil {
		t.Fatalf("GetTemplate error: %v", err)
	}

	locales := tmpl.Locales()
	if len(locales) < 3 {
		t.Fatalf("Locales error: %v", len(locales))
	}

	assert.Equal(t, "English", locales[0].Label)
	assert.Equal(t, "en-us", locales[0].Value)

	assert.Equal(t, "简体中文", locales[1].Label)
	assert.Equal(t, "zh-cn", locales[1].Value)

	assert.Equal(t, "繁體中文", locales[2].Label)
	assert.Equal(t, "zh-hk", locales[2].Value)

	assert.Equal(t, "日本語", locales[3].Label)
	assert.Equal(t, "ja-jp", locales[3].Value)
}

func TestTemplateAsset(t *testing.T) {
	tests := prepare(t)
	defer clean()

	tmpl, err := tests.Test.GetTemplate("advanced")
	if err != nil {
		t.Fatalf("GetTemplate error: %v", err)
	}

	// JavaScript
	asset, err := tmpl.Asset("/js/yao.js", 0, 0)
	if err != nil {
		t.Fatalf("Asset error: %v", err)
	}

	assert.Equal(t, "application/javascript; charset=utf-8", asset.Type)
	assert.NotEmpty(t, asset.Content)

	// CSS
	asset, err = tmpl.Asset("/css/app.css", 0, 0)
	if err != nil {
		t.Fatalf("Asset error: %v", err)
	}
	assert.Equal(t, "text/css; charset=utf-8", asset.Type)
	assert.NotEmpty(t, asset.Content)

	// IMAGE
	asset, err = tmpl.Asset("/images/icons/app.png", 100, 100)
	if err != nil {
		t.Fatalf("Asset error: %v", err)
	}
	assert.Equal(t, "image/png", asset.Type)
	assert.NotEmpty(t, asset.Content)
	exists, err := application.App.Exists("/data/test-cases/advanced/__assets/.cache/100x100/test-cases/advanced/__assets/images/icons/app.png")
	if err != nil {
		t.Fatalf("Asset error: %v", err)
	}
	assert.True(t, exists)

	// IMAGE SVG
	asset, err = tmpl.Asset("/images/logos/logo_color.svg", 100, 100)
	if err != nil {
		t.Fatalf("Asset error: %v", err)
	}
	assert.Equal(t, "image/svg+xml", asset.Type)

}
