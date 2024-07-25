package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/sui/core"
)

func TestMakeCache(t *testing.T) {
	prepare(t)
	defer clean()
	r := makeRequest("/unit-test/index.sui", t)
	c, status, err := r.MakeCache()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, http.StatusOK, status)
	assert.Contains(t, c.HTML, "The advanced test cases")
}

func TestRender(t *testing.T) {
	prepare(t)
	defer clean()

	parser, html, data := makeParser("/unit-test/index.sui", t)
	result, err := parser.Render(html)
	if err != nil {
		t.Fatal(err)
	}

	assert.Contains(t, result, "The advanced test cases")
	assert.NotNil(t, data["$global"])
	assert.NotNil(t, data["foo"])
	assert.NotNil(t, data["items"])

	// fmt.Println(result)
}

func makeParser(route string, t *testing.T) (*core.TemplateParser, string, core.Data) {
	r := makeRequest(route, t)
	c, _, err := r.MakeCache()
	if err != nil {
		t.Fatal(err)
	}

	data := r.Request.NewData()
	if c.Data != "" {
		err = r.Request.ExecStringMerge(data, c.Data)
		if err != nil {
			t.Fatal(err)
		}
	}

	if c.Global != "" {
		global, err := r.Request.ExecString(c.Global)
		if err != nil {
			t.Fatal(err)
		}
		data["$global"] = global
	}

	// Set the page request data
	option := core.ParserOption{
		Theme:        r.Request.Theme,
		Locale:       r.Request.Locale,
		Debug:        r.Request.DebugMode(),
		DisableCache: r.Request.DisableCache(),
		Route:        r.Request.URL.Path,
		Request:      r.Request,
	}

	// Parse the template
	return core.NewTemplateParser(data, &option), c.HTML, data
}

func makeRequest(path string, t *testing.T) *Request {
	req, err := http.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	r, status, err := NewRequestContext(c)
	if err != nil {
		t.Fatal(err)
	}

	if status != http.StatusOK {
		t.Fatalf("Status: %d", status)
	}
	return r
}
