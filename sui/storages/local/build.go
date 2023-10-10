package local

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hashicorp/go-multierror"
	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/yao/sui/core"
)

// Build the template
func (tmpl *Template) Build(option *core.BuildOption) error {
	var err error

	if option.AssetRoot == "" {
		option.AssetRoot = filepath.Join(tmpl.local.DSL.Public.Root, "assets")
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

// SyncAssets sync the assets
func (tmpl *Template) SyncAssets(option *core.BuildOption) error {

	// get source abs path
	sourceRoot := filepath.Join(tmpl.local.fs.Root(), tmpl.Root, "__assets")
	if exist, _ := os.Stat(sourceRoot); exist == nil {
		return nil
	}

	//get target abs path
	root := tmpl.local.DSL.Public.Root
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
		option.AssetRoot = filepath.Join(page.tmpl.local.DSL.Public.Root, "assets")
	}

	doc, _, err := page.Page.Build(option)
	if err != nil {
		return err
	}

	html, err := doc.Html()
	if err != nil {
		return err
	}

	// Save the html
	err = page.writeHTML([]byte(html))
	if err != nil {
		return err
	}

	// Save the data
	return page.writeData()
}

func (page *Page) publicFile() string {
	root := page.tmpl.local.DSL.Public.Root
	return filepath.Join(application.App.Root(), "public", root, page.Route)
}

// writeHTMLTo write the html to file
func (page *Page) writeHTML(html []byte) error {
	htmlFile := fmt.Sprintf("%s.html", page.publicFile())
	dir := filepath.Dir(htmlFile)
	if exist, _ := os.Stat(dir); exist == nil {
		os.MkdirAll(dir, os.ModePerm)
	}
	return os.WriteFile(htmlFile, html, 0644)
}

// writeHTMLTo write the html to file
func (page *Page) writeData() error {
	if page.Codes.DATA.Code == "" {
		return nil
	}
	dataFile := fmt.Sprintf("%s.json", page.publicFile())
	dir := filepath.Dir(dataFile)
	if exist, _ := os.Stat(dir); exist == nil {
		os.MkdirAll(dir, os.ModePerm)
	}
	return os.WriteFile(dataFile, []byte(page.Codes.DATA.Code), 0644)
}
