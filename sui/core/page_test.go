package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/kun/any"
)

func TestPageExec(t *testing.T) {
	prepare(t)
	defer clean()

	page := testDataPage(t)
	request := &Request{
		URL:    ReqeustURL{Path: "/test/path"},
		Query:  map[string][]string{"show": {"yes"}},
		Locale: "zh-cn",
		Theme:  "dark",
	}

	data, err := page.Exec(request)
	if err != nil {
		t.Fatalf("Exec error: %v", err)
	}

	assert.NotEmpty(t, data)

	res := any.Of(data).Map().Dot()
	assert.Equal(t, "yes", res.Get("array[3][0].query"))
	assert.Equal(t, "文章搜索 1", res.Get("articles.data[0].description"))
	assert.Equal(t, "/test/path", res.Get("url.path"))
}
