package core

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRender(t *testing.T) {
	prepare(t)
	defer clean()

	page := testDataPage(t)
	request := &Request{
		Query:  map[string][]string{"show": {"no"}},
		Locale: "zh-cn",
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

	for i, err := range parser.errors {
		fmt.Println(i, err)
	}

	assert.NotEmpty(t, html)
	assert.Contains(t, html, "hello space")
	assert.Equal(t, 0, len(parser.errors))
}
