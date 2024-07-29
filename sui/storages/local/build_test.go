package local

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/yao/sui/core"
)

func TestTemplateBuild(t *testing.T) {
	tests := prepare(t)
	defer clean()

	tmpl, err := tests.Test.GetTemplate("advanced")
	if err != nil {
		t.Fatalf("GetTemplate error: %v", err)
	}

	root := application.App.Root()
	public := tmpl.(*Template).local.GetPublic()
	path := filepath.Join(root, "public", public.Root)

	// Remove files and directories in Public directory if exists
	err = os.RemoveAll(path)
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("RemoveAll error: %v", err)
	}

	warnings, err := tmpl.Build(&core.BuildOption{SSR: true, ExecScripts: true})
	if err != nil {
		t.Fatalf("Components error: %v", err)
	}

	index := "/index.sui"

	// Check SUI
	assert.FileExists(t, filepath.Join(path, index))
	content, err := os.ReadFile(filepath.Join(path, index))
	if err != nil {
		t.Fatalf("ReadFile error: %v", err)
	}

	assert.Contains(t, string(content), "body")
	assert.Contains(t, string(content), `src="/unit-test/assets/js/import.js"`)
	assert.Contains(t, string(content), `<script name="config" type="json">`)
	assert.Contains(t, string(content), `<script name="data" type="json">`)
	assert.Contains(t, string(content), `<script name="global" type="json">`)
	assert.Len(t, warnings, 0)

}

func TestTemplateBuildAsComponent(t *testing.T) {
	tests := prepare(t)
	defer clean()

	tmpl, err := tests.Test.GetTemplate("advanced")
	if err != nil {
		t.Fatalf("GetTemplate error: %v", err)
	}

	root := application.App.Root()
	public := tmpl.(*Template).local.GetPublic()
	path := filepath.Join(root, "public", public.Root)

	// Remove files and directories in Public directory if exists
	err = os.RemoveAll(path)
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("RemoveAll error: %v", err)
	}

	warnings, err := tmpl.Build(&core.BuildOption{SSR: true})
	if err != nil {
		t.Fatalf("Components error: %v", err)
	}

	block := "/i18n/block.jit"
	bar := "/backend/bar.jit"

	// Check JIT
	assert.FileExists(t, filepath.Join(path, block))
	assert.FileExists(t, filepath.Join(path, bar))
	assert.Len(t, warnings, 0)

	content, err := os.ReadFile(filepath.Join(path, bar))
	if err != nil {
		t.Fatalf("ReadFile error: %v", err)
	}

	assert.NotContains(t, string(content), `<body`)
	assert.NotContains(t, string(content), `<script name="config" type="json">`)
	assert.NotContains(t, string(content), `<script name="data" type="json">`)
	assert.NotContains(t, string(content), `<script name="global" type="json">`)
	assert.Contains(t, string(content), `<script name="scripts" type="json">`)
	assert.Contains(t, string(content), `<script name="styles" type="json">`)
	assert.Contains(t, string(content), `<script name="option" type="json">`)
	assert.Contains(t, string(content), "this.Constants")
	assert.Contains(t, string(content), `type="hook-bar"`)
}

func TestPageBuild(t *testing.T) {
	tests := prepare(t)
	defer clean()

	tmpl, err := tests.Test.GetTemplate("advanced")
	if err != nil {
		t.Fatalf("GetTemplate error: %v", err)
	}

	root := application.App.Root()
	public := tmpl.(*Template).local.GetPublic()
	path := filepath.Join(root, "public", public.Root)

	// Remove files and directories in Public directory if exists
	err = os.RemoveAll(path)
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("RemoveAll error: %v", err)
	}

	page, err := tmpl.Page("/index")
	if err != nil {
		t.Fatalf("Page error: %v", err)
	}

	warnings, err := page.Build(nil, &core.BuildOption{SSR: true, AssetRoot: "/unit-test/assets"})
	if err != nil {
		t.Fatalf("Page Build error: %v", err)
	}
	index := "/index.sui"

	// Check SUI
	assert.FileExists(t, filepath.Join(path, index))

	content, err := os.ReadFile(filepath.Join(path, index))
	if err != nil {
		t.Fatalf("ReadFile error: %v", err)
	}

	assert.Contains(t, string(content), "body")
	assert.Contains(t, string(content), `src="/unit-test/assets/js/import.js"`)
	assert.Contains(t, string(content), `<script name="config" type="json">`)
	assert.Contains(t, string(content), `<script name="data" type="json">`)
	assert.Contains(t, string(content), `<script name="global" type="json">`)
	assert.Len(t, warnings, 0)
}

func TestPageBuildAsComponent(t *testing.T) {
	tests := prepare(t)
	defer clean()

	tmpl, err := tests.Test.GetTemplate("advanced")
	if err != nil {
		t.Fatalf("GetTemplate error: %v", err)
	}

	root := application.App.Root()
	public := tmpl.(*Template).local.GetPublic()
	path := filepath.Join(root, "public", public.Root)

	// Remove files and directories in Public directory if exists
	err = os.RemoveAll(path)
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("RemoveAll error: %v", err)
	}

	page, err := tmpl.Page("/backend")
	if err != nil {
		t.Fatalf("Page error: %v", err)
	}

	warnings, err := page.Build(nil, &core.BuildOption{SSR: true})
	if err != nil {
		t.Fatalf("Components error: %v", err)
	}
	assert.Len(t, warnings, 0)

	foo := "/backend/foo.jit"
	bar := "/backend/bar.jit"

	// Check JIT
	assert.FileExists(t, filepath.Join(path, foo))
	assert.FileExists(t, filepath.Join(path, bar))

	content, err := os.ReadFile(filepath.Join(path, bar))
	if err != nil {
		t.Fatalf("ReadFile error: %v", err)
	}

	assert.NotContains(t, string(content), `<body`)
	assert.NotContains(t, string(content), `<script name="config" type="json">`)
	assert.NotContains(t, string(content), `<script name="data" type="json">`)
	assert.NotContains(t, string(content), `<script name="global" type="json">`)
	assert.Contains(t, string(content), `<script name="scripts" type="json">`)
	assert.Contains(t, string(content), `<script name="styles" type="json">`)
	assert.Contains(t, string(content), `<script name="option" type="json">`)
	assert.Contains(t, string(content), "this.Constants")
	assert.Contains(t, string(content), `type="hook-bar"`)
}

func TestPageTrans(t *testing.T) {
	tests := prepare(t)
	defer clean()

	tmpl, err := tests.Test.GetTemplate("advanced")
	if err != nil {
		t.Fatalf("GetTemplate error: %v", err)
	}

	root := application.App.Root()
	path := filepath.Join(root, "data", tmpl.GetRoot(), "__locales")

	// Remove files and directories in Public directory if exists
	err = os.RemoveAll(path)
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("RemoveAll error: %v", err)
	}

	page, err := tmpl.Page("/i18n")
	if err != nil {
		t.Fatalf("Page error: %v", err)
	}

	warnings, err := page.Trans(nil, &core.BuildOption{SSR: true, AssetRoot: "/unit-test/assets"})
	if err != nil {
		t.Fatalf("Page Build error: %v", err)
	}

	assert.DirExists(t, path)
	assert.DirExists(t, filepath.Join(path, "zh-cn"))
	assert.DirExists(t, filepath.Join(path, "zh-hk"))
	assert.DirExists(t, filepath.Join(path, "ja-jp"))
	assert.FileExists(t, filepath.Join(path, "zh-cn", "i18n.yml"))
	assert.FileExists(t, filepath.Join(path, "zh-hk", "i18n.yml"))
	assert.FileExists(t, filepath.Join(path, "ja-jp", "i18n.yml"))
	assert.Len(t, warnings, 0)
}

func TestTemplateTrans(t *testing.T) {
	tests := prepare(t)
	defer clean()

	tmpl, err := tests.Test.GetTemplate("advanced")
	if err != nil {
		t.Fatalf("GetTemplate error: %v", err)
	}

	root := application.App.Root()
	path := filepath.Join(root, "data", tmpl.GetRoot(), "__locales")

	// Remove files and directories in Public directory if exists
	err = os.RemoveAll(path)
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("RemoveAll error: %v", err)
	}

	warnings, err := tmpl.Trans(&core.BuildOption{SSR: true})
	if err != nil {
		t.Fatalf("Components error: %v", err)
	}

	assert.DirExists(t, path)
	assert.DirExists(t, filepath.Join(path, "zh-cn"))
	assert.DirExists(t, filepath.Join(path, "zh-hk"))
	assert.DirExists(t, filepath.Join(path, "ja-jp"))
	assert.FileExists(t, filepath.Join(path, "zh-cn", "i18n.yml"))
	assert.FileExists(t, filepath.Join(path, "zh-hk", "i18n.yml"))
	assert.FileExists(t, filepath.Join(path, "ja-jp", "i18n.yml"))
	assert.Len(t, warnings, 0)
}
