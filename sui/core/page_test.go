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
		URL:    ReqeustURL{Path: "/test/path"},
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
	assert.Equal(t, "/test/path", res.Get("url.path"))
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
		Document: []byte(`
		<!DOCTYPE html>
		<html lang="en">
		<head>
		  <meta charset="UTF-8" />
		  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
		  <meta http-equiv="X-UA-Compatible" content="ie=edge" />
		  <title>Document</title>
		  <link rel="stylesheet" href="/assets/css/tailwind.css" />
		  <link rel="stylesheet" href="/assets/css/app.css" />
		  </head>
		  <body>
		  	<div id="app">
				{{ __page }}
			</div>
		</body>
		</html>
		`),
		Codes: SourceCodes{
			HTML: Source{
				File: "test.html",
				Code: `<div class="p-10">
				<div>For</div>
				<div s:for="articles.data" s:for-item="article" s:for-index="idx">
				  <div>{{ idx }} {{ article.title }}</div>
				  <div>{{ article.desc }}</div>
				  <div>{{ article.type == "article" ? "article" : "others"}}</div>

				  <div class="mt-10">IF</div>
				  <div class="p-5">
					<div s:if="article.type == 'article'">article</div>
					<div s:elif="article.type == 'image'">image</div>
					<div s:else>others</div>
				  </div>
			  
				  <div class="mt-10">Nested</div>
				  <div
					s:for="article.images"
					s:for-item="image"
					s:for-index="imgIndex"
					class="p-5"
				  >
					<div s:if="imgIndex == 1 || imgIndex == 2" class="p-5">
					  {{ imgIndex }} {{ image }}
					</div>
				  </div>
				</div>
				
				</div>
				<div class="mt-10">IF</div>
				<div s:if="len(articles) > 0" :name="input.data">
				  <div>{{ len(articles) > 0 }} articles.length &gt; 0</div>
				</div>
				<div s:if="length > 0" :name="input.data">
				  <div>{{ length > 0 }} len &gt; 0</div>
				</div>
				<div s:if="P_('scripts.article.Space', 'hello') == 'hello space'" >
				   hello space  {{ P_('scripts.article.Space', 'hello') }}
				</div>
				<div s:if="showImage == 'yes'">showImage</div>
				<div s:elif="showImage == 'no'">noImage</div>
				<div s:elif="showImage == 'auto'">autoImage</div>
				<div s:else>otherImage</div>
				<div class="mt-10">Bind</div>
				<div>
				  <div class="w-200">{{ input.data }}</div>
				  <div class="mt-5">
					<input type="text" s:bind="input.data" placeholder="Bind Input Data"
					  class="w-200 p-2 bg-purple-900 text-white" />
				  </div>
				  <div class="mt-5">
					<input type="button" value="Change" s:click="changeInput"
					  class="text-blue-600 p-2" />
				  </div>
				</div>

				<div class="mt-10">Import Page</div>
				<page is="/footer">
				  <slot is="link"> Link </slot>
				  <slot is="item"> Item </slot>
				</page>
			  
				<page is="/footer" no-style>
				  <slot is="link"> Link2 </slot>
				  <slot is="item"> Item2 </slot>
				</page>
				
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
					"length": 20,
					"array": [ 
						"item-1", 
						"$scripts.article.Setting", 
						{"$images": "scripts.article.Images"},
						{"process": "scripts.article.Thumbs", "args": ["$query.show"], "__exec": true }
					],
					"input": { "data": "hello world" },
					"url": {"path":"$url.path"}
				  }
				`,
			},
		},
	}

	return page
}
