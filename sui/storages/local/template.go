package local

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/sui/core"
)

// Get get the template
func (tmpl *Template) Get() error {
	return nil
}

// Save save the template
func (tmpl *Template) Save() error {
	return nil
}

// Pages get the pages
func (tmpl *Template) Pages() ([]core.IPage, error) {

	exts := []string{"*.sui", "*.html", "*.htm", "*.page"}
	pages := []core.IPage{}
	tmpl.local.fs.Walk(tmpl.Root, func(root, file string, isdir bool) error {
		name := filepath.Base(file)
		if isdir {
			if strings.HasPrefix(name, "__") {
				return filepath.SkipDir
			}
			return nil
		}

		if strings.HasPrefix(name, "__") {
			return nil
		}

		page, err := tmpl.getPageFrom(file)
		if err != nil {
			log.Error("Get page error: %v", err)
			return nil
		}

		pages = append(pages, page)
		return nil
	}, exts...)

	return pages, nil
}

// Page get the page
func (tmpl *Template) Page(route string) (core.IPage, error) {
	path := filepath.Join(tmpl.Root, route)
	exts := []string{".sui", ".html", ".htm", ".page"}
	for _, ext := range exts {
		file := fmt.Sprintf("%s%s", path, ext)
		if exist, _ := tmpl.local.fs.Exists(file); exist {
			return tmpl.getPageFrom(file)
		}
	}
	return nil, fmt.Errorf("Page %s not found", route)
}

func (tmpl *Template) getPageFrom(path string) (core.IPage, error) {
	route := tmpl.getPageRoute(path)
	return tmpl.getPage(route, path)
}

func (tmpl *Template) getPage(route, file string) (core.IPage, error) {
	root := filepath.Dir(file)
	return &Page{
		tmpl: tmpl,
		Page: &core.Page{
			Route: route,
			Root:  root,
			Files: core.PageFiles{
				HTML: fmt.Sprintf("%s%s", route, filepath.Ext(file)),
				CSS:  fmt.Sprintf("%s.css", route),
				JS:   fmt.Sprintf("%s.js", route),
				DATA: fmt.Sprintf("%s.json", route),
				TS:   fmt.Sprintf("%s.ts", route),
				LESS: fmt.Sprintf("%s.less", route),
			},
		},
	}, nil
}

func (tmpl *Template) getPageRoute(path string) string {
	return strings.TrimSuffix(path[len(tmpl.Root):], filepath.Ext(path))
}

// Blocks get the blocks
func (tmpl *Template) Blocks() ([]core.IBlock, error) {
	return nil, nil
}

// Components get the components
func (tmpl *Template) Components() ([]core.IComponent, error) {
	return nil, nil
}

// Block get the block
func (tmpl *Template) Block(name string) (core.IBlock, error) {
	return nil, nil
}

// Component get the component
func (tmpl *Template) Component(name string) (core.IComponent, error) {
	return nil, nil
}

// Styles get the global styles
func (tmpl *Template) Styles() []string {
	return nil
}

// Locales get the global locales
func (tmpl *Template) Locales() []string {
	return nil
}

// Themes get the global themes
func (tmpl *Template) Themes() []string {
	return nil
}
