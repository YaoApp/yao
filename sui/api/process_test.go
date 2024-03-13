package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
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

func TestBlockGet(t *testing.T) {
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
	assert.Equal(t, 7, len(res.([]core.IBlock)))
	assert.Equal(t, "ColumnsTwo", res.([]core.IBlock)[0].(*local.Block).ID)
	assert.Equal(t, "Hero", res.([]core.IBlock)[1].(*local.Block).ID)
	assert.Equal(t, "Image", res.([]core.IBlock)[2].(*local.Block).ID)
	assert.Equal(t, "Section", res.([]core.IBlock)[3].(*local.Block).ID)
}

func TestBlockFind(t *testing.T) {
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

func TestBlockExport(t *testing.T) {
	load(t)
	defer clean()

	// test demo
	p, err := process.Of("sui.block.export", "demo", "tech-blue")
	if err != nil {
		t.Fatal(err)
	}

	res, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.IsType(t, &core.BlockLayoutItems{}, res)
	assert.Equal(t, 3, len(res.(*core.BlockLayoutItems).Categories))
}

func TestBlockMedia(t *testing.T) {
	load(t)
	defer clean()

	// test demo
	p, err := process.Of("sui.block.media", "demo", "tech-blue", "ColumnsTwo")
	if err != nil {
		t.Fatal(err)
	}

	res, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.IsType(t, map[string]interface{}{}, res)
	assert.Equal(t, "image/png", res.(map[string]interface{})["type"])
	assert.NotEmpty(t, res.(map[string]interface{})["content"])
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
	assert.Equal(t, 6, len(res.([]core.IComponent)))
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
	assert.Equal(t, 6, len(res.([]*core.PageTreeNode)))
	assert.Equal(t, "error", res.([]*core.PageTreeNode)[0].Name)
	assert.Equal(t, "footer", res.([]*core.PageTreeNode)[1].Name)
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
	assert.Equal(t, 9, len(pages))
	for _, page := range pages {
		assert.IsType(t, &local.Page{}, page)
	}
}

func TestPageExist(t *testing.T) {

	load(t)
	defer clean()

	// test demo
	p, err := process.Of("sui.page.exist", "demo", "tech-blue", "/index/[invite]")
	if err != nil {
		t.Fatal(err)
	}

	res, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.IsType(t, true, res)
	assert.Equal(t, true, res.(bool))

	p, err = process.Of("sui.page.exist", "demo", "tech-blue", "/index/[invite]/[invite]")
	if err != nil {
		t.Fatal(err)
	}

	res, err = p.Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.IsType(t, false, res)
	assert.Equal(t, false, res.(bool))
}

func TestPageCreate(t *testing.T) {

	load(t)
	defer clean()
	defer func() {
		_, err := process.New("sui.page.remove", "demo", "tech-blue", "/unit-test").Exec()
		if err != nil {
			t.Fatal(err)
		}
	}()
	// test demo
	p, err := process.Of("sui.page.create", "demo", "tech-blue", "/unit-test")
	if err != nil {
		t.Fatal(err)
	}

	res, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Nil(t, res)
}

func TestPageRename(t *testing.T) {

	load(t)
	defer clean()
	defer func() {
		_, err := process.New("sui.page.remove", "demo", "tech-blue", "/unit-test-2").Exec()
		if err != nil {
			t.Fatal(err)
		}
	}()

	// test demo
	p, err := process.Of("sui.page.create", "demo", "tech-blue", "/unit-test")
	if err != nil {
		t.Fatal(err)
	}

	res, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}

	assert.Nil(t, res)

	// rename
	p, err = process.Of("sui.page.rename", "demo", "tech-blue", "/unit-test", map[string]interface{}{"route": "/unit-test-2"})
	if err != nil {
		t.Fatal(err)
	}

	res, err = p.Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Nil(t, res)
}

func TestPageDuplicate(t *testing.T) {

	load(t)
	defer clean()
	defer func() {
		_, err := process.New("sui.page.remove", "demo", "tech-blue", "/unit-test").Exec()
		if err != nil {
			t.Fatal(err)
		}
	}()

	// test demo
	p, err := process.Of("sui.page.duplicate", "demo", "tech-blue", "/page/[id]", map[string]interface{}{"title": "hello", "route": "/unit-test"})
	if err != nil {
		t.Fatal(err)
	}

	res, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Nil(t, res)
}

func TestPageCreateSaveThenRemove(t *testing.T) {

	load(t)
	defer clean()

	// test demo
	p, err := process.Of("sui.page.create", "demo", "tech-blue", "/unit-test", `{"uid":"unit-test", "needToSave":{"page":true}, "page":{"source":"<div>1</div>", "language":"html"}}`)
	if err != nil {
		t.Fatal(err)
	}

	res, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Nil(t, res)

	// test demo
	p, err = process.Of("sui.page.remove", "demo", "tech-blue", "/unit-test")
	if err != nil {
		t.Fatal(err)
	}

	res, err = p.Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Nil(t, res)
}

func TestPageSaveThenRemove(t *testing.T) {

	load(t)
	defer clean()

	// test demo
	p, err := process.Of("sui.page.create", "demo", "tech-blue", "/unit-test", `{"uid":"unit-test", "needToSave":{"page":true}, "page":{"source":"<div>1</div>", "language":"html"}}`)
	if err != nil {
		t.Fatal(err)
	}

	res, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Nil(t, res)

	// test demo
	p, err = process.Of("sui.page.SaveTemp", "demo", "tech-blue", "/unit-test", `{"uid":"unit-test", "needToSave":{"page":true}, "page":{"source":"<div>1</div>", "language":"html"}}`)
	if err != nil {
		t.Fatal(err)
	}

	res, err = p.Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Nil(t, res)

	// test demo
	p, err = process.Of("sui.page.Save", "demo", "tech-blue", "/unit-test", `{"uid":"unit-test", "needToSave":{"page":true}, "page":{"source":"<div>1</div>", "language":"html"}}`)
	if err != nil {
		t.Fatal(err)
	}

	res, err = p.Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Nil(t, res)

	// test demo
	p, err = process.Of("sui.page.remove", "demo", "tech-blue", "/unit-test")
	if err != nil {
		t.Fatal(err)
	}

	res, err = p.Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Nil(t, res)
}

func TestGetSource(t *testing.T) {

	// *core.RequestSource
	var payload interface{} = &core.RequestSource{UID: "unit-test"}
	args := []interface{}{"demo", "tech-blue", "/index/[invite]", payload}
	p, err := process.Of("sui.page.Save", args...)
	if err != nil {
		t.Fatal(err)
	}

	src, err := getSource(p)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "unit-test", src.UID)

	// String
	args[3] = `{"uid":"unit-test-string"}`
	p, err = process.Of("sui.page.Save", args...)
	if err != nil {
		t.Fatal(err)
	}
	src, err = getSource(p)
	assert.Equal(t, "unit-test-string", src.UID)

	// String & Payload
	args[3] = "unit-test-string2"
	newArgs := append(args, map[string]interface{}{
		"page": map[string]interface{}{
			"source":   "<div>1</div>",
			"language": "html",
		}})

	p, err = process.Of("sui.page.Save", newArgs...)
	if err != nil {
		t.Fatal(err)
	}
	src, err = getSource(p)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "unit-test-string2", src.UID)

	// Default
	args[3] = map[string]interface{}{
		"uid": "unit-test-map",
		"page": map[string]interface{}{
			"source":   "<div>1</div>",
			"language": "html",
		}}
	p, err = process.Of("sui.page.Save", args...)
	if err != nil {
		t.Fatal(err)
	}
	src, err = getSource(p)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "unit-test-map", src.UID)

	// Gin Context
	requestBody := []byte(`{"page": {"source":"gin-context Test", "language":"html"} }`)
	router := gin.Default()
	router.POST("/unit-test", func(ctx *gin.Context) {
		args[3] = ctx
		p, err = process.Of("sui.page.Save", args...)
		if err != nil {
			t.Fatal(err)
		}

		src, err = getSource(p)
		if err != nil {
			t.Fatal(err)
		}

		if src.Page == nil {
			t.Fatalf("Page is nil")
		}

		assert.Equal(t, "unit-test-gin-context", src.UID)
		assert.Equal(t, "html", src.Page.Language)
		assert.Equal(t, "gin-context Test", src.Page.Source)
	})

	req, err := http.NewRequest("POST", "/unit-test", bytes.NewBuffer(requestBody))
	if err != nil {
		t.Fatalf("Couldn't create request: %v\n", err)
		return
	}
	req.Header.Set("Yao-Builder-Uid", "unit-test-gin-context")
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(httptest.NewRecorder(), req)
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

	assert.IsType(t, &core.ResponseEditorRender{}, res)
	assert.NotEmpty(t, res.(*core.ResponseEditorRender).HTML)
	assert.NotEmpty(t, res.(*core.ResponseEditorRender).Config)
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
		assert.IsType(t, core.SourceData{}, res)
		assert.NotEmpty(t, res.(core.SourceData).Source)
		assert.NotEmpty(t, res.(core.SourceData).Language)
	}
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

	assert.IsType(t, &core.ResponseEditorRender{}, res)
	assert.NotEmpty(t, res.(*core.ResponseEditorRender).HTML)
}

func TestPreviewRender(t *testing.T) {
	load(t)
	defer clean()

	// test demo
	p, err := process.Of("sui.preview.render", "demo", "tech-blue", "/index")
	if err != nil {
		t.Fatal(err)
	}

	res, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}

	assert.IsType(t, "", res)
	assert.NotEmpty(t, res)
}

func TestBuildAll(t *testing.T) {
	load(t)
	defer clean()

	// test demo
	p, err := process.Of("sui.build.all", "demo", "tech-blue", map[string]interface{}{"ssr": true})
	if err != nil {
		t.Fatal(err)
	}

	res, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}

	assert.Nil(t, res)
}

func TestBuildPage(t *testing.T) {
	load(t)
	defer clean()

	// test demo
	p, err := process.Of("sui.build.page", "demo", "tech-blue", "/index", map[string]interface{}{"ssr": true})
	if err != nil {
		t.Fatal(err)
	}

	res, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}

	assert.Nil(t, res)
}

func TestSyncAssetFile(t *testing.T) {
	load(t)
	defer clean()

	// test demo
	p, err := process.Of("sui.sync.assetfile", "demo", "tech-blue", "/images/about/ab01.jpg", map[string]interface{}{"ssr": true})
	if err != nil {
		t.Fatal(err)
	}

	res, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}

	assert.Nil(t, res)
}

func load(t *testing.T) {
	prepare(t)
	err := Load(config.Conf)
	if err != nil {
		t.Fatal(err)
	}
}
