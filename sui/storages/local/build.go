package local

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hashicorp/go-multierror"
	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/sui/core"
)

// Build the template
func (tmpl *Template) Build(option *core.BuildOption) error {
	var err error

	root, err := tmpl.local.DSL.PublicRoot(option.Data)
	if err != nil {
		log.Error("SyncAssets: Get the public root error: %s. use %s", err.Error(), tmpl.local.DSL.Public.Root)
		root = tmpl.local.DSL.Public.Root
	}

	if option.AssetRoot == "" {
		option.AssetRoot = filepath.Join(root, "assets")
	}

	// Sync the assets
	if err = tmpl.SyncAssets(option); err != nil {
		return err
	}

	// Build all pages
	pages, err := tmpl.Pages()
	if err != nil {
		return err
	}

	for _, page := range pages {
		perr := page.Load()
		if err != nil {
			err = multierror.Append(perr)
			continue
		}

		perr = page.Build(option)
		if perr != nil {
			err = multierror.Append(perr)
		}
	}

	return err
}

// SyncAssetFile sync the assets
func (tmpl *Template) SyncAssetFile(file string, option *core.BuildOption) error {

	// get source abs path
	sourceRoot := filepath.Join(tmpl.local.fs.Root(), tmpl.Root, "__assets")
	if exist, _ := os.Stat(sourceRoot); exist == nil {
		return nil
	}

	//get target abs path
	root, err := tmpl.local.DSL.PublicRoot(option.Data)
	if err != nil {
		log.Error("SyncAssets: Get the public root error: %s. use %s", err.Error(), tmpl.local.DSL.Public.Root)
		root = tmpl.local.DSL.Public.Root
	}

	targetRoot := filepath.Join(application.App.Root(), "public", root, "assets")
	sourceFile := filepath.Join(sourceRoot, file)
	targetFile := filepath.Join(targetRoot, file)

	// create the target directory
	if exist, _ := os.Stat(targetFile); exist == nil {
		os.MkdirAll(filepath.Dir(targetFile), os.ModePerm)
	}

	return copy(sourceFile, targetFile)
}

// SyncAssets sync the assets
func (tmpl *Template) SyncAssets(option *core.BuildOption) error {

	// get source abs path
	sourceRoot := filepath.Join(tmpl.local.fs.Root(), tmpl.Root, "__assets")
	if exist, _ := os.Stat(sourceRoot); exist == nil {
		return nil
	}

	//get target abs path
	root, err := tmpl.local.DSL.PublicRoot(option.Data)
	if err != nil {
		log.Error("SyncAssets: Get the public root error: %s. use %s", err.Error(), tmpl.local.DSL.Public.Root)
		root = tmpl.local.DSL.Public.Root
	}
	targetRoot := filepath.Join(application.App.Root(), "public", root, "assets")

	if exist, _ := os.Stat(targetRoot); exist == nil {
		os.MkdirAll(targetRoot, os.ModePerm)
	}
	os.RemoveAll(targetRoot)

	return copyDirectory(sourceRoot, targetRoot)
}

// Build is the struct for the public
func (page *Page) Build(option *core.BuildOption) error {

	if option.AssetRoot == "" {
		root, err := page.tmpl.local.DSL.PublicRoot(option.Data)
		if err != nil {
			log.Error("SyncAssets: Get the public root error: %s. use %s", err.Error(), page.tmpl.local.DSL.Public.Root)
			root = page.tmpl.local.DSL.Public.Root
		}
		option.AssetRoot = filepath.Join(root, "assets")
	}

	log.Trace("Build the page %s AssetRoot: %s", page.Route, option.AssetRoot)
	html, err := page.Page.Compile(option)
	if err != nil {
		return err
	}

	// Save the html
	return page.writeHTML([]byte(html), option.Data)
}

func (page *Page) publicFile(data map[string]interface{}) string {
	root, err := page.tmpl.local.DSL.PublicRoot(data)
	if err != nil {
		log.Error("publicFile: Get the public root error: %s. use %s", err.Error(), page.tmpl.local.DSL.Public.Root)
		root = page.tmpl.local.DSL.Public.Root
	}
	return filepath.Join("/", "public", root, page.Route)
}

// writeHTMLTo write the html to file
func (page *Page) writeHTML(html []byte, data map[string]interface{}) error {
	htmlFile := fmt.Sprintf("%s.sui", page.publicFile(data))
	htmlFileAbs := filepath.Join(application.App.Root(), htmlFile)
	dir := filepath.Dir(htmlFileAbs)
	if exist, _ := os.Stat(dir); exist == nil {
		os.MkdirAll(dir, os.ModePerm)
	}
	err := os.WriteFile(htmlFileAbs, html, 0644)
	if err != nil {
		return err
	}

	core.RemoveCache(htmlFile)
	log.Trace("The page %s is removed", htmlFile)
	return nil
}
