package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/sui/core"
	"github.com/yaoapp/yao/sui/storages/local"
)

func TestTemplateGet(t *testing.T) {
	load(t)
	defer clean()

	// test demo
	p, err := process.Of("sui.template.get", "demo")
	if err != nil {
		t.Fatal(err)
	}

	res, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}

	assert.IsType(t, []core.ITemplate{}, res)
	assert.Equal(t, 3, len(res.([]core.ITemplate)))
}

func TestTemplateFind(t *testing.T) {
	load(t)
	defer clean()

	// test demo
	p, err := process.Of("sui.template.find", "demo", "tech-blue")
	if err != nil {
		t.Fatal(err)
	}

	res, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}

	assert.IsType(t, &local.Template{}, res)
	assert.Equal(t, "tech-blue", res.(*local.Template).ID)
}

func TestTemplateLocaleGet(t *testing.T) {
	load(t)
	defer clean()

	// test demo
	p, err := process.Of("sui.template.locale.get", "demo", "tech-blue")
	if err != nil {
		t.Fatal(err)
	}

	res, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}

	assert.IsType(t, []core.SelectOption{}, res)
	assert.Equal(t, 3, len(res.([]core.SelectOption)))
	assert.Equal(t, "ar", res.([]core.SelectOption)[0].Value)
	assert.Equal(t, "zh-cn", res.([]core.SelectOption)[1].Value)
	assert.Equal(t, "zh-tw", res.([]core.SelectOption)[2].Value)
}

func TestTemplateThemeGet(t *testing.T) {
	load(t)
	defer clean()

	// test demo
	p, err := process.Of("sui.template.theme.get", "demo", "tech-blue")
	if err != nil {
		t.Fatal(err)
	}

	res, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}

	assert.IsType(t, []core.SelectOption{}, res)
	assert.Equal(t, 2, len(res.([]core.SelectOption)))
	assert.Equal(t, "dark", res.([]core.SelectOption)[0].Value)
	assert.Equal(t, "light", res.([]core.SelectOption)[1].Value)
}

func TestEditorRender(t *testing.T) {
	load(t)
	defer clean()

	// test demo
	p, err := process.Of("sui.editor.render", "demo", "tech-blue", "/index")
	if err != nil {
		t.Fatal(err)
	}

	res, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}

	assert.IsType(t, &core.ResponseEditor{}, res)
	assert.NotEmpty(t, res.(*core.ResponseEditor).HTML)
}

func TestEditorRenderWithQuery(t *testing.T) {
	load(t)
	defer clean()

	// test demo
	p, err := process.Of("sui.editor.render", "demo", "tech-blue", "/index", map[string]interface{}{
		"method": "POST",
	})
	if err != nil {
		t.Fatal(err)
	}

	res, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}

	assert.IsType(t, &core.ResponseEditor{}, res)
	assert.NotEmpty(t, res.(*core.ResponseEditor).HTML)
}

func TestEditorPageSource(t *testing.T) {
	load(t)
	defer clean()

	sources := []string{"page", "script", "style", "data"}
	for _, source := range sources {
		p, err := process.Of("sui.editor.source", "demo", "tech-blue", "/index", source)
		if err != nil {
			t.Fatal(err)
		}

		res, err := p.Exec()
		if err != nil {
			t.Fatal(err)
		}
		assert.IsType(t, core.ResponseSource{}, res)
		assert.NotEmpty(t, res.(core.ResponseSource).Source)
		assert.NotEmpty(t, res.(core.ResponseSource).Language)
	}
}

func load(t *testing.T) {
	prepare(t)
	err := Load(config.Conf)
	if err != nil {
		t.Fatal(err)
	}
}
