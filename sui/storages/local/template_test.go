package local

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTemplateThemes(t *testing.T) {
	tests := prepare(t)
	defer clean()

	tmpl, err := tests.Demo.GetTemplate("tech-blue")
	if err != nil {
		t.Fatalf("GetTemplate error: %v", err)
	}

	themes := tmpl.Themes()
	if len(themes) < 2 {
		t.Fatalf("Themes error: %v", len(themes))
	}

	assert.Equal(t, "dark", themes[0].Value)
	assert.Equal(t, "暗色主题", themes[0].Label)
	assert.Equal(t, "light", themes[1].Value)
	assert.Equal(t, "明亮主题", themes[1].Label)
}

func TestTemplateLocales(t *testing.T) {
	tests := prepare(t)
	defer clean()

	tmpl, err := tests.Demo.GetTemplate("tech-blue")
	if err != nil {
		t.Fatalf("GetTemplate error: %v", err)
	}

	locales := tmpl.Locales()
	if len(locales) < 3 {
		t.Fatalf("Locales error: %v", len(locales))
	}

	assert.Equal(t, "ar", locales[0].Label)
	assert.Equal(t, "ar", locales[0].Value)

	assert.Equal(t, "zh-CN", locales[1].Label)
	assert.Equal(t, "zh-cn", locales[1].Value)

	assert.Equal(t, "zh-TW", locales[2].Label)
	assert.Equal(t, "zh-tw", locales[2].Value)
}

func TestTemplateAsset(t *testing.T) {
	tests := prepare(t)
	defer clean()

	tmpl, err := tests.Demo.GetTemplate("tech-blue")
	if err != nil {
		t.Fatalf("GetTemplate error: %v", err)
	}

	asset, err := tmpl.Asset("/css/tailwind.css", 0, 0)
	if err != nil {
		t.Fatalf("Asset error: %v", err)
	}

	assert.Equal(t, "text/css; charset=utf-8", asset.Type)
	assert.NotEmpty(t, asset.Content)

}
