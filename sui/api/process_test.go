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

func TestTemplateAsset(t *testing.T) {
	load(t)
	defer clean()

	// test demo
	p, err := process.Of("sui.template.asset", "demo", "tech-blue", "/css/tailwind.css")
	if err != nil {
		t.Fatal(err)
	}

	res, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}

	assert.NotEmpty(t, res)
	assert.Equal(t, "text/css; charset=utf-8", res.(map[string]interface{})["type"])
	assert.NotEmpty(t, res.(map[string]interface{})["content"])
}

func TestTemplateLocaleGet(t *testing.T) {
	load(t)
	defer clean()

	// test demo
	p, err := process.Of("sui.locale.get", "demo", "tech-blue")
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
	p, err := process.Of("sui.theme.get", "demo", "tech-blue")
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

func TestTemplateBlockGet(t *testing.T) {
	load(t)
	defer clean()

	// test demo
	p, err := process.Of("sui.block.get", "demo", "tech-blue")
	if err != nil {
		t.Fatal(err)
	}

	res, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}

	assert.IsType(t, []core.IBlock{}, res)
	assert.Equal(t, 4, len(res.([]core.IBlock)))
	assert.Equal(t, "ColumnsTwo", res.([]core.IBlock)[0].(*local.Block).ID)
	assert.Equal(t, "Hero", res.([]core.IBlock)[1].(*local.Block).ID)
	assert.Equal(t, "Section", res.([]core.IBlock)[2].(*local.Block).ID)
	assert.Equal(t, "Table", res.([]core.IBlock)[3].(*local.Block).ID)
}

func TestTemplateBlockFind(t *testing.T) {
	load(t)
	defer clean()

	// test demo
	p, err := process.Of("sui.block.find", "demo", "tech-blue", "ColumnsTwo")
	if err != nil {
		t.Fatal(err)
	}

	res, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}

	assert.IsType(t, "", res)
	assert.Contains(t, res.(string), "window.block__ColumnsTwo=")
}

func TestTemplateComponentGet(t *testing.T) {
	load(t)
	defer clean()

	// test demo
	p, err := process.Of("sui.component.get", "demo", "tech-blue")
	if err != nil {
		t.Fatal(err)
	}

	res, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}

	assert.IsType(t, []core.IComponent{}, res)
	assert.Equal(t, 3, len(res.([]core.IComponent)))
	assert.Equal(t, "Box", res.([]core.IComponent)[0].(*local.Component).ID)
	assert.Equal(t, "Card", res.([]core.IComponent)[1].(*local.Component).ID)
	assert.Equal(t, "Nav", res.([]core.IComponent)[2].(*local.Component).ID)
}

func TestTemplateComponentFind(t *testing.T) {
	load(t)
	defer clean()

	// test demo
	p, err := process.Of("sui.component.find", "demo", "tech-blue", "Box")
	if err != nil {
		t.Fatal(err)
	}

	res, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}

	assert.IsType(t, "", res)
	assert.Contains(t, res.(string), "window.component__Box=")
}

func TestPageTree(t *testing.T) {
	load(t)
	defer clean()

	// test demo
	p, err := process.Of("sui.page.tree", "demo", "tech-blue")
	if err != nil {
		t.Fatal(err)
	}

	res, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}

	assert.IsType(t, []*core.PageTreeNode{}, res)
	assert.Equal(t, 5, len(res.([]*core.PageTreeNode)))
	assert.Equal(t, "error", res.([]*core.PageTreeNode)[0].Name)
	assert.Equal(t, "index", res.([]*core.PageTreeNode)[1].Name)
}

func TestPageGet(t *testing.T) {
	load(t)
	defer clean()

	// test demo
	p, err := process.Of("sui.page.get", "demo", "tech-blue", "/index/[invite]")
	if err != nil {
		t.Fatal(err)
	}

	res, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}

	pages := res.([]core.IPage)
	assert.IsType(t, []core.IPage{}, pages)
	assert.Equal(t, 8, len(pages))
	for _, page := range pages {
		assert.IsType(t, &local.Page{}, page)
	}
}

func TestPageAssetJS(t *testing.T) {

	load(t)
	defer clean()

	// test demo
	p, err := process.Of("sui.page.asset", "demo", "tech-blue", "/page/404/404.js")
	if err != nil {
		t.Fatal(err)
	}

	res, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.IsType(t, map[string]interface{}{}, res)
	assert.Equal(t, "text/javascript; charset=utf-8", res.(map[string]interface{})["type"])
	assert.NotEmpty(t, res.(map[string]interface{})["content"])
}

func TestPageAssetTS(t *testing.T) {
	load(t)
	defer clean()

	// test demo
	p, err := process.Of("sui.page.asset", "demo", "tech-blue", "/page/404/404.ts")
	if err != nil {
		t.Fatal(err)
	}

	res, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.IsType(t, map[string]interface{}{}, res)
	assert.Equal(t, "text/javascript; charset=utf-8", res.(map[string]interface{})["type"])
	assert.NotEmpty(t, res.(map[string]interface{})["content"])
}

func TestPageAssetCSS(t *testing.T) {
	load(t)
	defer clean()

	// test demo
	p, err := process.Of("sui.page.asset", "demo", "tech-blue", "/page/[id]/[id].css")
	if err != nil {
		t.Fatal(err)
	}

	res, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.IsType(t, map[string]interface{}{}, res)
	assert.Equal(t, "text/css; charset=utf-8", res.(map[string]interface{})["type"])
	assert.NotEmpty(t, res.(map[string]interface{})["content"])
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
