package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/kun/utils"
)

func TestRender(t *testing.T) {
	prepare(t)
	defer clean()

	page := testPage(t)
	request := &Request{
		Query:  map[string][]string{"show": {"no"}},
		Locale: "zh-CN",
		Theme:  "dark",
	}

	data, err := page.Exec(request)
	if err != nil {
		t.Fatalf("Exec error: %v", err)
	}

	assert.NotEmpty(t, data)
	parser := NewTemplateParser(data, nil)
	html, err := parser.Render(page.Codes.HTML.Code)
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}

	utils.Dump(html)

	assert.NotEmpty(t, html)
	assert.Contains(t, html, "hello space")
	assert.Equal(t, 0, len(parser.errors))
}
