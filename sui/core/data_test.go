package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/kun/any"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/test"
)

func TestPageExec(t *testing.T) {
	prepare(t)
	defer clean()

	page := testPage(t)
	request := &Request{
		Query:  map[string][]string{"show": {"yes"}},
		Locale: "zh-CN",
		Theme:  "dark",
	}

	data, err := page.Exec(request)
	if err != nil {
		t.Fatalf("Exec error: %v", err)
	}

	assert.NotEmpty(t, data)

	res := any.Of(data).Map().Dot()
	assert.Equal(t, "yes", res.Get("array[3][0].query"))
	assert.Equal(t, "Article Search", res.Get("articles.data[0].description"))
}

func prepare(t *testing.T) {
	test.Prepare(t, config.Conf, "YAO_TEST_BUILDER_APPLICATION")
}

func clean() {
	test.Clean()
}

func testPage(t *testing.T) *Page {

	page := &Page{
		Name:  "test",
		Route: "test",
		Codes: SourceCodes{
			HTML: Source{
				File: "test.html",
				Code: `<div class="p-10">
				<div>For</div>
				<div s:for="articles" s:for-item="article" s:for-index="idx">
				  <div>{{ idx }} {{ article.title }}</div>
				  <div>{{ article.desc }}</div>
				  <div>{{ article.type == "article" ? "article" : "others"}}</div>
				  <div s:if="article.type == 'article'">article</div>
				  <div s:elif="article.type == 'image'">image</div>
				  <div s:else="">others</div>
				</div>
				<div class="mt-10">IF</div>
				<div s:if="articles.length > 0">
				  <div>{{ articles.length > 0 }} articles.length 大于 0</div>
				</div>
				<div s:if="showImage == 'yes'">showImage</div>
				<div s:elif="showImage == 'no'">noImage</div>
				<div s:else="">others</div>
			  
				<div class="mt-10">Bind</div>
				<div>
				  <div class="w-200">{{ input.data }}</div>
				  <div class="mt-5">
					<input
					  class="w-200 p-2 bg-purple-900 text-white"
					  type="text"
					  s:bind="input.data"
					  placeholder="数据双向绑定"
					/>
				  </div>
				  <div class="mt-5">
					<input
					  type="button"
					  value="修改"
					  class="text-blue-600 p-2"
					  s:click="changeInput"
					/>
				  </div>
				</div>
			  </div>`,
			},
			DATA: Source{
				File: "test.json",
				Code: `{
					"$articles": "scripts.article.Search",
					"$showImage": {
					  "process": "scripts.article.ShowImage",
					  "args": ["$query.show"]
					},
					"array": [ 
						"item-1", 
						"$scripts.article.Setting", 
						{"$images": "scripts.article.Images"},
						{"process": "scripts.article.Thumbs", "args": ["$query.show"], "__exec": true }
					],
					"input": { "data": "hello world" }
				  }
				`,
			},
		},
	}

	return page
}
