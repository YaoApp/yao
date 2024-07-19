package local

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/sui/core"
	"github.com/yaoapp/yao/test"
)

type TestCase struct {
	Test *Local
	Web  *Local
}

func TestGetTemplates(t *testing.T) {
	tests := prepare(t)
	defer clean()

	testTmpls, err := tests.Test.GetTemplates()
	if err != nil {
		t.Fatalf("GetTemplates error: %v", err)
	}

	if len(testTmpls) != 2 {
		t.Fatalf("The test templates not equal 2 (%v!=2)", len(testTmpls))
	}

	// Advanced Template
	assert.Equal(t, "advanced", testTmpls[0].(*Template).ID)
	assert.Equal(t, "The advanced template", testTmpls[0].(*Template).Name)
	assert.Len(t, testTmpls[0].Themes(), 2)
	assert.Len(t, testTmpls[0].Locales(), 5)
	assert.Len(t, testTmpls[0].(*Template).Template.Themes, 2)
	assert.Len(t, testTmpls[0].(*Template).Template.Locales, 5)

	// Basic Template
	assert.Equal(t, "basic", testTmpls[1].(*Template).ID)
	assert.Equal(t, "The basic template", testTmpls[1].(*Template).Name)
	assert.Len(t, testTmpls[1].Themes(), 0)
	assert.Len(t, testTmpls[1].Locales(), 0)
	assert.Len(t, testTmpls[1].(*Template).Template.Themes, 0)
	assert.Len(t, testTmpls[1].(*Template).Template.Locales, 0)

	// Default Template ( Application )
	webTmpls, err := tests.Web.GetTemplates()
	if err != nil {
		t.Fatalf("GetTemplates error: %v", err)
	}

	if len(webTmpls) != 1 {
		t.Fatalf("The web templates not equal 1 (%v!=1)", len(webTmpls))
	}

	// Default Template
	assert.Equal(t, "default", webTmpls[0].(*Template).ID)
	assert.Equal(t, "Yao Startup Webapp", webTmpls[0].(*Template).Name)
	assert.Len(t, webTmpls[0].Themes(), 2)
	assert.Len(t, webTmpls[0].Locales(), 5)
	assert.Len(t, webTmpls[0].(*Template).Template.Themes, 2)
	assert.Len(t, webTmpls[0].(*Template).Template.Locales, 5)
}

func TestGetTemplate(t *testing.T) {
	tests := prepare(t)
	defer clean()

	basicTmpl, err := tests.Test.GetTemplate("basic")
	if err != nil {
		t.Fatalf("GetTemplate error: %v", err)
	}

	assert.Equal(t, "basic", basicTmpl.(*Template).ID)

	advancedTmpl, err := tests.Test.GetTemplate("advanced")
	if err != nil {
		t.Fatalf("GetTemplate error: %v", err)
	}
	assert.Equal(t, "advanced", advancedTmpl.(*Template).ID)

	defaultTmpl, err := tests.Web.GetTemplate("default")
	if err != nil {
		t.Fatalf("GetTemplate error: %v", err)
	}
	assert.Equal(t, "default", defaultTmpl.(*Template).ID)
}

func prepare(t *testing.T) TestCase {

	test.Prepare(t, config.Conf, "YAO_SUI_TEST_APPLICATION")
	webDSL, err := core.Load("/suis/web.sui.yao", "web")
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	web, err := New(webDSL)
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	core.SUIs["web"] = web

	testDSL, err := core.Load("/suis/test.sui.yao", "test")
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	test, err := New(testDSL)
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	core.SUIs["test"] = test
	return TestCase{
		Test: test,
		Web:  web,
	}
}

func clean() {
	test.Clean()
}
