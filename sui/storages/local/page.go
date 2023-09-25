package local

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/sui/core"
)

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
			Codes: core.SourceCodes{
				HTML: core.Source{File: fmt.Sprintf("%s%s", route, filepath.Ext(file))},
				CSS:  core.Source{File: fmt.Sprintf("%s.css", route)},
				JS:   core.Source{File: fmt.Sprintf("%s.js", route)},
				DATA: core.Source{File: fmt.Sprintf("%s.json", route)},
				TS:   core.Source{File: fmt.Sprintf("%s.ts", route)},
				LESS: core.Source{File: fmt.Sprintf("%s.less", route)},
			},
		},
	}, nil
}

// Get get the page
func (page *Page) Get() error {
	return nil
}

// Save save the page
func (page *Page) Save() error {
	return nil
}
