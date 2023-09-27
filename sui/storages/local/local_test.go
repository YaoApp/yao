package local

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/sui/core"
	"github.com/yaoapp/yao/test"
)

func TestGetTemplates(t *testing.T) {
	tests := prepare(t)
	defer clean()

	dempTmpls, err := tests.Demo.GetTemplates()
	if err != nil {
		t.Fatalf("GetTemplates error: %v", err)
	}

	if len(dempTmpls) < 3 {
		t.Fatalf("The demo templates less than 3 (%v<3)", len(dempTmpls))
	}

	assert.Equal(t, "tech-blue", dempTmpls[0].(*Template).ID)
	assert.Equal(t, "Tech Blue DEMO", dempTmpls[0].(*Template).Name)
	assert.Equal(t, true, len(dempTmpls[1].(*Template).Screenshots) > 0)
	assert.Equal(t, 1, dempTmpls[0].(*Template).Version)
	assert.Equal(t, "Tech Blue DEMO", dempTmpls[0].(*Template).Descrption)

	assert.Equal(t, "website-ai", dempTmpls[1].(*Template).ID)
	assert.Equal(t, "Website DEMO", dempTmpls[1].(*Template).Name)
	assert.Equal(t, true, len(dempTmpls[1].(*Template).Screenshots) > 0)
	assert.Equal(t, 2, dempTmpls[1].(*Template).Version)
	assert.Equal(t, "AI Website DEMO", dempTmpls[1].(*Template).Descrption)

	assert.Equal(t, "wechat-web", dempTmpls[2].(*Template).ID)
	assert.Equal(t, "WECHAT-WEB", dempTmpls[2].(*Template).Name)
	assert.Equal(t, []string{}, dempTmpls[2].(*Template).Screenshots)
	assert.Equal(t, 1, dempTmpls[2].(*Template).Version)
	assert.Equal(t, "", dempTmpls[2].(*Template).Descrption)

}

func TestGetTemplate(t *testing.T) {
	tests := prepare(t)
	defer clean()

	websiteAI, err := tests.Demo.GetTemplate("website-ai")
	if err != nil {
		t.Fatalf("GetTemplate error: %v", err)
	}

	assert.Equal(t, "website-ai", websiteAI.(*Template).ID)
	assert.Equal(t, "Website DEMO", websiteAI.(*Template).Name)
	assert.Equal(t, true, len(websiteAI.(*Template).Screenshots) > 0)
	assert.Equal(t, 2, websiteAI.(*Template).Version)
	assert.Equal(t, "AI Website DEMO", websiteAI.(*Template).Descrption)
}

func prepare(t *testing.T) struct {
	Demo   *Local
	Screen *Local
} {

	test.Prepare(t, config.Conf, "YAO_TEST_BUILDER_APPLICATION")
	demoDSL, err := core.Load("/suis/demo.sui.yao", "demo")
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	demo, err := New(demoDSL)
	if err != nil {
		t.Fatalf("New error: %v", err)
	}

	screenDSL, err := core.Load("/suis/screen.sui.yao", "screen")
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	screen, err := New(screenDSL)
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	return struct {
		Demo   *Local
		Screen *Local
	}{
		Demo:   demo,
		Screen: screen,
	}
}

func clean() {
	test.Clean()
}
