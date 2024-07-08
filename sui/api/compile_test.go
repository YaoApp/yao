package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/sui/core"
)

func TestCompile(t *testing.T) {
	prepare(t)
	defer clean()

	page := testPage(t)
	html, warnings, err := page.Compile(nil, &core.BuildOption{KeepPageTag: false})
	if err != nil {
		t.Fatalf("Compile error: %v", err)
	}
	assert.Contains(t, html, `The basic test cases`)
	assert.Len(t, warnings, 0)
}

func testPage(t *testing.T) *core.Page {

	sui := core.SUIs["test"]
	if sui == nil {
		t.Fatal("SUI test not found")
	}

	tmpl, err := sui.GetTemplate("basic")
	if err != nil {
		t.Fatal(err)
	}

	page, err := tmpl.Page("/index")
	if err != nil {
		t.Fatal(err)
	}

	return page.Get()
}
