package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/yao/sui/core"
	"github.com/yaoapp/yao/sui/storages/local"
)

func TestTemplateGet(t *testing.T) {
	prepare(t)
	defer clean()

	// test demo
	p, err := process.Of("sui.template.get", "test")
	if err != nil {
		t.Fatal(err)
	}

	res, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}

	assert.IsType(t, []core.ITemplate{}, res)
	assert.Equal(t, 2, len(res.([]core.ITemplate)))
}

func TestTemplateFind(t *testing.T) {
	prepare(t)
	defer clean()

	// test demo
	p, err := process.Of("sui.template.find", "test", "advanced")
	if err != nil {
		t.Fatal(err)
	}

	res, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}

	assert.IsType(t, &local.Template{}, res)
	assert.Equal(t, "advanced", res.(*local.Template).ID)
}

func TestTemplateAsset(t *testing.T) {
	prepare(t)
	defer clean()

	// test demo
	p, err := process.Of("sui.template.asset", "test", "advanced", "/css/app.css")
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
	prepare(t)
	defer clean()

	// test demo
	p, err := process.Of("sui.locale.get", "test", "advanced")
	if err != nil {
		t.Fatal(err)
	}

	res, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}

	assert.IsType(t, []core.SelectOption{}, res)
	assert.Equal(t, 5, len(res.([]core.SelectOption)))
	assert.Equal(t, "en-us", res.([]core.SelectOption)[0].Value)
	assert.True(t, res.([]core.SelectOption)[0].Default)

	assert.Equal(t, "zh-cn", res.([]core.SelectOption)[1].Value)
	assert.Equal(t, "zh-hk", res.([]core.SelectOption)[2].Value)
	assert.Equal(t, "ja-jp", res.([]core.SelectOption)[3].Value)
}

func TestTemplateThemeGet(t *testing.T) {
	prepare(t)
	defer clean()

	// test demo
	p, err := process.Of("sui.theme.get", "test", "advanced")
	if err != nil {
		t.Fatal(err)
	}

	res, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}

	assert.IsType(t, []core.SelectOption{}, res)
	assert.Equal(t, 2, len(res.([]core.SelectOption)))
	assert.Equal(t, "light", res.([]core.SelectOption)[0].Value)
	assert.Equal(t, "dark", res.([]core.SelectOption)[1].Value)
}

func TestBlockGet(t *testing.T) {
	prepare(t)
	defer clean()

	// test demo
	p, err := process.Of("sui.block.get", "test", "advanced")
	if err != nil {
		t.Fatal(err)
	}

	res, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}

	assert.IsType(t, []core.IBlock{}, res)
	assert.Equal(t, 0, len(res.([]core.IBlock)))
}

func TestBlockFind(t *testing.T) {
	prepare(t)
	defer clean()

	// test demo
	p, err := process.Of("sui.block.find", "test", "advanced", "not-found")
	if err != nil {
		t.Fatal(err)
	}

	_, err = p.Exec()
	assert.NotNil(t, err)
}

func TestBlockExport(t *testing.T) {
	prepare(t)
	defer clean()

	// test demo
	p, err := process.Of("sui.block.export", "test", "advanced")
	if err != nil {
		t.Fatal(err)
	}

	_, err = p.Exec()
	assert.Nil(t, err)
}

func TestBlockMedia(t *testing.T) {
	prepare(t)
	defer clean()

	// test demo
	p, err := process.Of("sui.block.media", "test", "advanced", "not-found")
	if err != nil {
		t.Fatal(err)
	}

	_, err = p.Exec()
	assert.NotNil(t, err)

	// assert.IsType(t, map[string]interface{}{}, res)
	// assert.Equal(t, "image/png", res.(map[string]interface{})["type"])
	// assert.NotEmpty(t, res.(map[string]interface{})["content"])
}

func TestTemplateComponentGet(t *testing.T) {
	prepare(t)
	defer clean()

	// test demo
	p, err := process.Of("sui.component.get", "test", "advanced")
	if err != nil {
		t.Fatal(err)
	}

	res, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}

	assert.IsType(t, []core.IComponent{}, res)
	assert.Equal(t, 0, len(res.([]core.IComponent)))
}

func TestTemplateComponentFind(t *testing.T) {
	prepare(t)
	defer clean()

	// test demo
	p, err := process.Of("sui.component.find", "test", "advanced", "not-found")
	if err != nil {
		t.Fatal(err)
	}

	_, err = p.Exec()
	assert.NotNil(t, err)
}

func TestPageTree(t *testing.T) {
	prepare(t)
	defer clean()

	// test demo
	p, err := process.Of("sui.page.tree", "test", "advanced")
	if err != nil {
		t.Fatal(err)
	}

	res, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}

	assert.IsType(t, []*core.PageTreeNode{}, res)
	assert.GreaterOrEqual(t, len(res.([]*core.PageTreeNode)), 2)
}

func TestPageGet(t *testing.T) {
	prepare(t)
	defer clean()

	// test demo
	p, err := process.Of("sui.page.get", "test", "advanced", "/page/[id]")
	if err != nil {
		t.Fatal(err)
	}

	res, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}

	pages := res.([]core.IPage)
	assert.IsType(t, []core.IPage{}, pages)
	assert.GreaterOrEqual(t, len(pages), 2)
	for _, page := range pages {
		assert.IsType(t, &local.Page{}, page)
	}
}

func TestPageExist(t *testing.T) {

	prepare(t)
	defer clean()

	// test demo
	p, err := process.Of("sui.page.exist", "test", "advanced", "/page/[id]")
	if err != nil {
		t.Fatal(err)
	}

	res, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.IsType(t, true, res)
	assert.Equal(t, true, res.(bool))

	p, err = process.Of("sui.page.exist", "test", "advanced", "/page/[id]/[id]")
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

	prepare(t)
	defer clean()
	defer func() {
		_, err := process.New("sui.page.remove", "test", "advanced", "/unit-test").Exec()
		if err != nil {
			t.Fatal(err)
		}
	}()
	// test demo
	p, err := process.Of("sui.page.create", "test", "advanced", "/unit-test")
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

	prepare(t)
	defer clean()
	defer func() {
		_, err := process.New("sui.page.remove", "test", "advanced", "/unit-test-2").Exec()
		if err != nil {
			t.Fatal(err)
		}
	}()

	// test demo
	p, err := process.Of("sui.page.create", "test", "advanced", "/unit-test")
	if err != nil {
		t.Fatal(err)
	}

	res, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}

	assert.Nil(t, res)

	// rename
	p, err = process.Of("sui.page.rename", "test", "advanced", "/unit-test", map[string]interface{}{"route": "/unit-test-2"})
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

	prepare(t)
	defer clean()
	defer func() {
		_, err := process.New("sui.page.remove", "test", "advanced", "/unit-test").Exec()
		if err != nil {
			t.Fatal(err)
		}
	}()

	// test demo
	p, err := process.Of("sui.page.duplicate", "test", "advanced", "/page/[id]", map[string]interface{}{"title": "hello", "route": "/unit-test"})
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

	prepare(t)
	defer clean()

	// test demo
	p, err := process.Of("sui.page.create", "test", "advanced", "/unit-test", `{"uid":"unit-test", "needToSave":{"page":true}, "page":{"source":"<div>1</div>", "language":"html"}}`)
	if err != nil {
		t.Fatal(err)
	}

	res, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Nil(t, res)

	// test demo
	p, err = process.Of("sui.page.remove", "test", "advanced", "/unit-test")
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

	prepare(t)
	defer clean()

	// test demo
	p, err := process.Of("sui.page.create", "test", "advanced", "/unit-test", `{"uid":"unit-test", "needToSave":{"page":true}, "page":{"source":"<div>1</div>", "language":"html"}}`)
	if err != nil {
		t.Fatal(err)
	}

	res, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Nil(t, res)

	// test demo
	p, err = process.Of("sui.page.SaveTemp", "test", "advanced", "/unit-test", `{"uid":"unit-test", "needToSave":{"page":true}, "page":{"source":"<div>1</div>", "language":"html"}}`)
	if err != nil {
		t.Fatal(err)
	}

	res, err = p.Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Nil(t, res)

	// test demo
	p, err = process.Of("sui.page.Save", "test", "advanced", "/unit-test", `{"uid":"unit-test", "needToSave":{"page":true}, "page":{"source":"<div>1</div>", "language":"html"}}`)
	if err != nil {
		t.Fatal(err)
	}

	res, err = p.Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Nil(t, res)

	// test demo
	p, err = process.Of("sui.page.remove", "test", "advanced", "/unit-test")
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
	args := []interface{}{"test", "advanced", "/index/[invite]", payload}
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

	prepare(t)
	defer clean()

	// test demo
	p, err := process.Of("sui.page.asset", "test", "advanced", "/page/[id]/404/404.js")
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
	prepare(t)
	defer clean()

	// test demo
	p, err := process.Of("sui.page.asset", "test", "advanced", "/page/[id]/404/404.ts")
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
	prepare(t)
	defer clean()

	// test demo
	p, err := process.Of("sui.page.asset", "test", "advanced", "/page/[id]/[id].css")
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
	prepare(t)
	defer clean()

	// test demo
	p, err := process.Of("sui.editor.render", "test", "advanced", "/index")
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
	prepare(t)
	defer clean()

	sources := []string{"page", "script", "style", "data"}
	for _, source := range sources {
		p, err := process.Of("sui.editor.source", "test", "advanced", "/index", source)
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
	prepare(t)
	defer clean()

	// test demo
	p, err := process.Of("sui.editor.render", "test", "advanced", "/index", map[string]interface{}{
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
	prepare(t)
	defer clean()

	// test demo
	p, err := process.Of("sui.preview.render", "test", "advanced", "/index")
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
	prepare(t)
	defer clean()

	// test demo
	p, err := process.Of("sui.build.all", "test", "advanced", map[string]interface{}{"ssr": true})
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
	prepare(t)
	defer clean()

	// test demo
	p, err := process.Of("sui.build.page", "test", "advanced", "/index", map[string]interface{}{"ssr": true})
	if err != nil {
		t.Fatal(err)
	}

	res, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}

	assert.Nil(t, res)
}

func TestTransAll(t *testing.T) {
	prepare(t)
	defer clean()

	// test demo
	p, err := process.Of("sui.trans.all", "test", "advanced", map[string]interface{}{"ssr": true})
	if err != nil {
		t.Fatal(err)
	}

	res, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}

	assert.Nil(t, res)
}

func TestTransPage(t *testing.T) {
	prepare(t)
	defer clean()

	// test demo
	p, err := process.Of("sui.trans.page", "test", "advanced", "/i18n", map[string]interface{}{"ssr": true})
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
	prepare(t)
	defer clean()

	// test demo
	p, err := process.Of("sui.sync.assetfile", "test", "advanced", "/images/logos/wordmark.svg", map[string]interface{}{"ssr": true})
	if err != nil {
		t.Fatal(err)
	}

	res, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}

	assert.Nil(t, res)
}
