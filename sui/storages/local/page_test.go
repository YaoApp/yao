package local

import (
	"path/filepath"
	"testing"

	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/sui/core"
)

func TestTemplatePages(t *testing.T) {
	tests := prepare(t)
	defer clean()

	tmpl, err := tests.Test.GetTemplate("advanced")
	if err != nil {
		t.Fatalf("GetTemplate error: %v", err)
	}

	pages, err := tmpl.Pages()
	if err != nil {
		t.Fatalf("Pages error: %v", err)
	}

	if len(pages) < 1 {
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
		assert.Equal(t, name+".config", page.Codes.CONF.File)
	}
}

func TestTemplatePageTree(t *testing.T) {
	tests := prepare(t)
	defer clean()

	tmpl, err := tests.Test.GetTemplate("advanced")
	if err != nil {
		t.Fatalf("GetTemplate error: %v", err)
	}

	pages, err := tmpl.PageTree("/")
	if err != nil {
		t.Fatalf("Pages error: %v", err)
	}

	if len(pages) < 4 {
		t.Fatalf("Pages error: %v", len(pages))
	}

	assert.NotEmpty(t, pages)
	assert.NotEmpty(t, pages[1].Children)
	if len(pages[1].Children) < 3 {
		t.Fatalf("Pages error: %v", len(pages[1].Children))
	}

	assert.NotEmpty(t, pages[3].Children[0].Children)
	if len(pages[3].Children[0].Children) < 2 {
		t.Fatalf("Pages error: %v", len(pages[2].Children[0].Children))
	}
}

func TestTemplatePageTS(t *testing.T) {

	tests := prepare(t)
	defer clean()

	tmpl, err := tests.Test.GetTemplate("advanced")
	if err != nil {
		t.Fatalf("GetTemplate error: %v", err)
	}

	ipage, err := tmpl.Page("/page/[id]")
	if err != nil {
		t.Fatalf("Page error: %v", err)
	}

	page := ipage.(*Page)

	assert.Equal(t, "/page/[id]", page.Route)
	assert.Equal(t, "/test-cases/advanced/page/[id]", page.Path)
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
	assert.Contains(t, err.Error(), "/the/page/could/not/be/found not found")
}

func TestTemplatePageJS(t *testing.T) {

	tests := prepare(t)
	defer clean()

	tmpl, err := tests.Test.GetTemplate("advanced")
	if err != nil {
		t.Fatalf("GetTemplate error: %v", err)
	}

	ipage, err := tmpl.Page("/page/[id]/404")
	if err != nil {
		t.Fatalf("Page error: %v", err)
	}

	page := ipage.(*Page)
	assert.Equal(t, "/page/[id]/404", page.Route)
	assert.Equal(t, "/test-cases/advanced/page/[id]/404", page.Path)
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
	assert.Contains(t, err.Error(), "/the/page/could/not/be/found not found")
}

func TestPageSaveTempBoard(t *testing.T) {

	tests := prepare(t)
	defer clean()

	tmpl, err := tests.Test.GetTemplate("advanced")
	if err != nil {
		t.Fatalf("GetTemplate error: %v", err)
	}

	const payload = `{
		"page": null,
		"style": null,
		"script": null,
		"data": null,
		"board": {
		  "html": "<div class=\"bg-purple-700 p-4 text-base\"><span class=\"text-white mr-2\">Home</span><a href=\"/index/{{user.id}}\" class=\"text-white\">Invite</a></div>\n<div id=\"i2j7\" cui:type=\"Card\">\n    <h1>Card Instance</h1>\n    <p>Card xx</p>\n    <div> Table </div>\n</div>",
		  "style": "#i2j7 {\n    color: #2c3e50;\n    width: 100%;\n    height: 300px;\n    background: #d1c2d3;\n    padding: .5em;\n}"
		},
		"needToSave": {
		  "page": false,
		  "style": false,
		  "script": false,
		  "data": false,
		  "board": true,
		  "validate": true
		}
	  }`

	req := &core.RequestSource{UID: "19e09e7e-9e19-44c1-bbab-2a55c51c9df3"}
	jsoniter.Unmarshal([]byte(payload), &req)

	page, err := tmpl.Page("/index")
	if err != nil {
		t.Fatalf("Page error: %v", err)
	}

	err = page.SaveTemp(req)
	assert.Nil(t, err)
}

func TestPageSaveTempPage(t *testing.T) {
	tests := prepare(t)
	defer clean()

	tmpl, err := tests.Test.GetTemplate("advanced")
	if err != nil {
		t.Fatalf("GetTemplate error: %v", err)
	}

	const payload = `{
		"page": {
		  "source": "<div class=\"bg-purple-700 p-4 text-base\">\n  <span class=\"text-white mr-2\">Home</span>\n  <a href=\"/index/{{user.id}}\" class=\"text-white\">Invite</a>\n</div>\n\n<div cui:type=\"Card\">\n  <h1>Card Instance</h1>\n  <p>Card XYZ</p>\n</div>\n",
		  "language": "html"
		},
		"style": null,
		"script": null,
		"data": null,
		"board": {
		  "html": "<body service=\"Index\" data-gjs-type=\"wrapper\" data-gjs-stylable=\"[&quot;background&quot;,&quot;background-color&quot;,&quot;background-image&quot;,&quot;background-repeat&quot;,&quot;background-attachment&quot;,&quot;background-position&quot;,&quot;background-size&quot;]\"><div class=\"bg-purple-700 p-4 text-base\"><span class=\"text-white mr-2\" data-gjs-tagName=\"span\" data-gjs-type=\"text\">Home</span><a href=\"/index/{{user.id}}\" class=\"text-white\" data-gjs-type=\"link\">Invite</a></div><div id=\"izaw\" data-gjs-type=\"Card\" data-gjs-style=\"\"><h1 data-gjs-tagName=\"h1\" data-gjs-type=\"text\">Card Instance</h1><p data-gjs-tagName=\"p\" data-gjs-type=\"text\">Card xx</p></div></body>",
		  "style": "#izaw{color:#2c3e50;width:100%;height:300px;background:#d1c2d3;padding:.5em;}"
		},
		"needToSave": {
		  "page": true,
		  "style": false,
		  "script": false,
		  "data": false,
		  "board": false,
		  "validate": true
		}
	  }`

	req := &core.RequestSource{UID: "19e09e7e-9e19-44c1-bbab-2a55c51c9df3"}
	jsoniter.Unmarshal([]byte(payload), &req)

	page, err := tmpl.Page("/index")
	if err != nil {
		t.Fatalf("Page error: %v", err)
	}

	err = page.SaveTemp(req)
	assert.Nil(t, err)
}

func TestPageSaveTempStyle(t *testing.T) {
	tests := prepare(t)
	defer clean()

	tmpl, err := tests.Test.GetTemplate("advanced")
	if err != nil {
		t.Fatalf("GetTemplate error: %v", err)
	}

	const payload = `{
		"page": null,
		"style": {
		  "source": "* { box-sizing: border-box; }\nbody { margin: 0; }\n#ihjf { color:#ffffff;width:100%;height:100px;background:#1c0d1a;padding:.5em;display:flex; }\n\n",
		  "language": "css"
		},
		"script": null,
		"data": null,
		"board": {
		  "html": "<body service=\"Index\" data-gjs-type=\"wrapper\" data-gjs-stylable=\"[&quot;background&quot;,&quot;background-color&quot;,&quot;background-image&quot;,&quot;background-repeat&quot;,&quot;background-attachment&quot;,&quot;background-position&quot;,&quot;background-size&quot;]\"><div class=\"bg-purple-700 p-4 text-base\"><span class=\"text-white mr-2\" data-gjs-tagName=\"span\" data-gjs-type=\"text\">Home</span><a href=\"/index/{{user.id}}\" class=\"text-white\" data-gjs-type=\"link\">Invite</a></div><div id=\"inhy\" data-gjs-type=\"Card\" data-gjs-style=\"\"><h1 data-gjs-tagName=\"h1\" data-gjs-type=\"text\">Card Instance</h1><p data-gjs-tagName=\"p\" data-gjs-type=\"text\">Card xx</p></div></body>",
		  "style": "#inhy{color:#2c3e50;width:100%;height:300px;background:#d1c2d3;padding:.5em;}"
		},
		"needToSave": {
		  "page": false,
		  "style": true,
		  "script": false,
		  "data": false,
		  "board": false,
		  "validate": true
		}
	  }`

	req := &core.RequestSource{UID: "19e09e7e-9e19-44c1-bbab-2a55c51c9df3"}
	jsoniter.Unmarshal([]byte(payload), &req)

	page, err := tmpl.Page("/index")
	if err != nil {
		t.Fatalf("Page error: %v", err)
	}

	err = page.SaveTemp(req)
	assert.Nil(t, err)
}

func TestPageSaveTempScriptJS(t *testing.T) {
	tests := prepare(t)
	defer clean()

	tmpl, err := tests.Test.GetTemplate("advanced")
	if err != nil {
		t.Fatalf("GetTemplate error: %v", err)
	}

	const payload = `{
		"page": null,
		"style": null,
		"script": {
		  "source": "function Hello() {\n  console.log(\"Hello World!\");\n}\n\nfunction Index() {\n  return {\n    title: \"Customers\",\n    hello: \"world\",\n    rows: [\n      { name: \"John\", age: 30, city: \"New York\" },\n      { name: \"Mary\", age: 20, city: \"Paris\" },\n      { name: \"Peter\", age: 40, city: \"London\" },\n    ],\n  };\n}\n",
		  "language": "javascript"
		},
		"data": null,
		"board": {
		  "html": "<body service=\"Index\" data-gjs-type=\"wrapper\" data-gjs-stylable=\"[&quot;background&quot;,&quot;background-color&quot;,&quot;background-image&quot;,&quot;background-repeat&quot;,&quot;background-attachment&quot;,&quot;background-position&quot;,&quot;background-size&quot;]\"><div class=\"bg-purple-700 p-4 text-base\"><span class=\"text-white mr-2\" data-gjs-tagName=\"span\" data-gjs-type=\"text\">Home</span><a href=\"/index/{{user.id}}\" class=\"text-white\" data-gjs-type=\"link\">Invite</a></div><div id=\"icqh\" data-gjs-type=\"Card\" data-gjs-style=\"\"><h1 data-gjs-tagName=\"h1\" data-gjs-type=\"text\">Card Instance</h1><p data-gjs-tagName=\"p\" data-gjs-type=\"text\">Card xx</p></div></body>",
		  "style": "#icqh{color:#2c3e50;width:100%;height:300px;background:#d1c2d3;padding:.5em;}"
		},
		"needToSave": {
		  "page": false,
		  "style": false,
		  "script": true,
		  "data": false,
		  "board": false,
		  "validate": true
		}
	  }`

	req := &core.RequestSource{UID: "19e09e7e-9e19-44c1-bbab-2a55c51c9df3"}
	jsoniter.Unmarshal([]byte(payload), &req)

	page, err := tmpl.Page("/index")
	if err != nil {
		t.Fatalf("Page error: %v", err)
	}

	err = page.SaveTemp(req)
	assert.Nil(t, err)
}

func TestPageSaveTempScriptTS(t *testing.T) {
	tests := prepare(t)
	defer clean()

	tmpl, err := tests.Test.GetTemplate("advanced")
	if err != nil {
		t.Fatalf("GetTemplate error: %v", err)
	}

	const payload = `{
		"page": null,
		"style": null,
		"script": {
			"source": "import \"@assets/dark/dark.css\";\nimport \"@assets/light/light.css\";\n\nimport { Hello } from \"@assets/main.js\";\n\nconst onPageLoad = (event: Event) => {\n  console.log(\"Page Loaded 103\");\n};\n\nconst onPageReady = (event: Event) => {\n  Hello(\"world\");\n  Foo(\"Bar\");\n  console.log(\"Page Ready\");\n  return;\n};\n\nconst onData = (\n  params: { [key: string]: string[] },\n  query: { [key: string]: string[] }\n) => {\n  console.log(\"Page Send Data Request\");\n};\n\nconst onDataSuccess = (data: { [key: string]: any }) => {\n  console.log(\"Page Data Ready\");\n};\n\nconst onDataError = (data: { code: number; message: string }) => {\n  console.log(\"Page Data Ready\");\n};\n\nconst onResize = (event: Event) => {\n  console.log(\"Page Resize\");\n};\n\nconst onPageScroll = (event: Event) => {\n  console.log(\"Page Scroll\");\n};\n\nfunction Foo(bar: string) {\n  console.log(` + "`Foo ${bar}`" + `);\n}\n",
			"language": "typescript"
		},
		"data": null,
		"board": {
		  "html": "<body service=\"Index\" data-gjs-type=\"wrapper\" data-gjs-stylable=\"[&quot;background&quot;,&quot;background-color&quot;,&quot;background-image&quot;,&quot;background-repeat&quot;,&quot;background-attachment&quot;,&quot;background-position&quot;,&quot;background-size&quot;]\"><div class=\"bg-purple-700 p-4 text-base\"><span class=\"text-white mr-2\" data-gjs-tagName=\"span\" data-gjs-type=\"text\">Home</span><a href=\"/index/{{user.id}}\" class=\"text-white\" data-gjs-type=\"link\">Invite</a></div><div id=\"icqh\" data-gjs-type=\"Card\" data-gjs-style=\"\"><h1 data-gjs-tagName=\"h1\" data-gjs-type=\"text\">Card Instance</h1><p data-gjs-tagName=\"p\" data-gjs-type=\"text\">Card xx</p></div></body>",
		  "style": "#icqh{color:#2c3e50;width:100%;height:300px;background:#d1c2d3;padding:.5em;}"
		},
		"needToSave": {
		  "page": false,
		  "style": false,
		  "script": true,
		  "data": false,
		  "board": false,
		  "validate": true
		}
	  }`

	req := &core.RequestSource{UID: "19e09e7e-9e19-44c1-bbab-2a55c51c9df3"}
	jsoniter.Unmarshal([]byte(payload), &req)

	page, err := tmpl.Page("/index")
	if err != nil {
		t.Fatalf("Page error: %v", err)
	}

	err = page.SaveTemp(req)
	assert.Nil(t, err)
}

func TestPageSaveTempData(t *testing.T) {
	tests := prepare(t)
	defer clean()

	tmpl, err := tests.Test.GetTemplate("advanced")
	if err != nil {
		t.Fatalf("GetTemplate error: %v", err)
	}

	const payload = `{
		"page": null,
		"style": null,
		"script": null,
		"data": {
		  "source": "{\n  \"title\": \"Home Page\",\n  \"data\": { \"service\": \"Index\" },\n  \"foo\": \"bar\",\n  \"preview\": { \"params\": { \"id\": \"1\" } }\n}\n",
		  "language": "json"
		},
		"board": {
		  "html": "<body service=\"Index\" data-gjs-type=\"wrapper\" data-gjs-stylable=\"[&quot;background&quot;,&quot;background-color&quot;,&quot;background-image&quot;,&quot;background-repeat&quot;,&quot;background-attachment&quot;,&quot;background-position&quot;,&quot;background-size&quot;]\"><div class=\"bg-purple-700 p-4 text-base\"><span class=\"text-white mr-2\" data-gjs-tagName=\"span\" data-gjs-type=\"text\">Home</span><a href=\"/index/{{user.id}}\" class=\"text-white\" data-gjs-type=\"link\">Invite</a></div><div id=\"ipxs\" data-gjs-type=\"Card\" data-gjs-style=\"\"><h1 data-gjs-tagName=\"h1\" data-gjs-type=\"text\">Card Instance</h1><p data-gjs-tagName=\"p\" data-gjs-type=\"text\">Card xx</p></div></body>",
		  "style": "#ipxs{color:#2c3e50;width:100%;height:300px;background:#d1c2d3;padding:.5em;}"
		},
		"needToSave": {
		  "page": false,
		  "style": false,
		  "script": false,
		  "data": true,
		  "board": false,
		  "validate": true
		}
	}`

	req := &core.RequestSource{UID: "19e09e7e-9e19-44c1-bbab-2a55c51c9df3"}
	jsoniter.Unmarshal([]byte(payload), &req)

	page, err := tmpl.Page("/index")
	if err != nil {
		t.Fatalf("Page error: %v", err)
	}

	err = page.SaveTemp(req)
	assert.Nil(t, err)
}

func TestPageSaveTempSetting(t *testing.T) {
	tests := prepare(t)
	defer clean()

	tmpl, err := tests.Test.GetTemplate("advanced")
	if err != nil {
		t.Fatalf("GetTemplate error: %v", err)
	}

	const payload = `{
		"page": null,
		"style": null,
		"script": null,
		"setting": { "title": "Home Page | {{ $global.title }}" },
		"mock": { "params": { "id": "1" } },
		"needToSave": {
		  "page": false,
		  "style": false,
		  "script": false,
		  "mock": true,
		  "setting": true,
		  "board": false,
		  "validate": true
		}
	}`

	req := &core.RequestSource{UID: "19e09e7e-9e19-44c1-bbab-2a55c51c9df3"}
	jsoniter.Unmarshal([]byte(payload), &req)

	page, err := tmpl.Page("/index")
	if err != nil {
		t.Fatalf("Page error: %v", err)
	}

	err = page.SaveTemp(req)
	assert.Nil(t, err)
}

func TestPageSave(t *testing.T) {
	tests := prepare(t)
	defer clean()

	tmpl, err := tests.Test.GetTemplate("advanced")
	if err != nil {
		t.Fatalf("GetTemplate error: %v", err)
	}

	const payload = `{
		"page": null,
		"style": null,
		"script": null,
		"data": null,
		"board": {
		  "html": "<div id=\"io71\">404 {{ $query.message || $post.message }}<br>Add IT</div>",
		  "style": ""
		},
		"needToSave": {
		  "page": false,
		  "style": false,
		  "script": false,
		  "data": false,
		  "board": true,
		  "validate": true
		}
	}`

	req := &core.RequestSource{UID: "19e09e7e-9e19-44c1-bbab-2a55c51c9df3"}
	jsoniter.Unmarshal([]byte(payload), &req)

	page, err := tmpl.CreateEmptyPage("/unit-test", nil)
	if err != nil {
		t.Fatalf("Page error: %v", err)
	}
	defer page.Remove()

	err = page.SaveTemp(req)
	if err != nil {
		t.Fatalf("SaveTemp error: %v", err)
	}

	err = page.Save(req)
	assert.Nil(t, err)

}

func TestPageGetPageFromAsset(t *testing.T) {

	tests := prepare(t)
	defer clean()

	tmpl, err := tests.Test.GetTemplate("advanced")
	if err != nil {
		t.Fatalf("GetTemplate error: %v", err)
	}

	file := "/index/index.css"
	page, err := tmpl.GetPageFromAsset(file)
	if err != nil {
		t.Fatalf("GetPageFromAsset error: %v", err)
	}

	assert.Equal(t, "/index", page.Get().Route)
	assert.Equal(t, "/test-cases/advanced/index", page.Get().Path)
	assert.Equal(t, "index", page.Get().Name)

	file = "/page/404/404.js"
	page, err = tmpl.GetPageFromAsset(file)
	if err != nil {
		t.Fatalf("GetPageFromAsset error: %v", err)
	}

	assert.Equal(t, "/page/404", page.Get().Route)
	assert.Equal(t, "/test-cases/advanced/page/404", page.Get().Path)
	assert.Equal(t, "404", page.Get().Name)
}

func TestPageAssetScriptJS(t *testing.T) {
	tests := prepare(t)
	defer clean()

	tmpl, err := tests.Test.GetTemplate("advanced")
	if err != nil {
		t.Fatalf("GetTemplate error: %v", err)
	}

	file := "/page/[id]/404/404.js"
	page, err := tmpl.GetPageFromAsset(file)
	if err != nil {
		t.Fatalf("GetPageFromAsset error: %v", err)
	}

	asset, err := page.AssetScript()
	if err != nil {
		t.Fatalf("AssetScript error: %v", err)
	}

	assert.NotEmpty(t, asset.Content)
	assert.Equal(t, "text/javascript; charset=utf-8", asset.Type)
}

func TestPageAssetScriptTS(t *testing.T) {
	tests := prepare(t)
	defer clean()

	tmpl, err := tests.Test.GetTemplate("advanced")
	if err != nil {
		t.Fatalf("GetTemplate error: %v", err)
	}

	file := "/page/[id]/[id].ts"
	page, err := tmpl.GetPageFromAsset(file)
	if err != nil {
		t.Fatalf("GetPageFromAsset error: %v", err)
	}

	asset, err := page.AssetScript()
	if err != nil {
		t.Fatalf("AssetScript error: %v", err)
	}

	assert.NotEmpty(t, asset.Content)
	assert.Equal(t, "text/javascript; charset=utf-8", asset.Type)
}

func TestPageAssetStyle(t *testing.T) {
	tests := prepare(t)
	defer clean()

	tmpl, err := tests.Test.GetTemplate("advanced")
	if err != nil {
		t.Fatalf("GetTemplate error: %v", err)
	}

	file := "/page/[id]/[id].css"
	page, err := tmpl.GetPageFromAsset(file)
	if err != nil {
		t.Fatalf("GetPageFromAsset error: %v", err)
	}

	asset, err := page.AssetStyle()
	if err != nil {
		t.Fatalf("AssetStyle error: %v", err)
	}

	assert.NotEmpty(t, asset.Content)
	assert.Equal(t, "text/css; charset=utf-8", asset.Type)
}

func TestCreatePage(t *testing.T) {
	tests := prepare(t)
	defer clean()

	tmpl, err := tests.Test.GetTemplate("advanced")
	if err != nil {
		t.Fatalf("GetTemplate error: %v", err)
	}

	page := tmpl.CreatePage("<div>Test</div>")
	if page == nil {
		t.Fatalf("CreatePage error")
	}

	doc, _, err := page.Get().Build(core.NewBuildContext(nil), &core.BuildOption{
		PublicRoot:     tmpl.GetRoot(),
		IgnoreDocument: true,
		JitMode:        true,
	})
	if err != nil {
		t.Fatalf("Compile error: %v", err)
	}

	sel := doc.Find("body")
	html, err := sel.Html()
	if err != nil {
		t.Fatalf("Html error: %v", err)
	}

	assert.Equal(t, "<div>Test</div>", html)
}
