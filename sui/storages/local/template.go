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

func (tmpl *Template) getPageRoute(path string) string {
	return strings.TrimSuffix(path[len(tmpl.Root):], filepath.Ext(path))
}

// Blocks get the blocks
func (tmpl *Template) Blocks() ([]core.IBlock, error) {
	path := filepath.Join(tmpl.Root, "__blocks")

	blocks := []core.IBlock{}
	if exist, _ := tmpl.local.fs.Exists(path); !exist {
		return blocks, nil
	}

	dirs, err := tmpl.local.fs.ReadDir(path, false)
	if err != nil {
		return nil, err
	}

	for _, dir := range dirs {
		if !tmpl.local.fs.IsDir(dir) {
			continue
		}

		block, err := tmpl.getBlockFrom(dir)
		if err != nil {
			log.Error("Get block error: %v", err)
			continue
		}

		blocks = append(blocks, block)
	}

	return blocks, nil
}

// Block get the block
func (tmpl *Template) Block(id string) (core.IBlock, error) {
	path := filepath.Join(tmpl.Root, "__blocks", id)
	if exist, _ := tmpl.local.fs.Exists(path); !exist {
		return nil, fmt.Errorf("Block %s not found", id)
	}

	block, err := tmpl.getBlockFrom(path)
	if err != nil {
		return nil, err
	}

	err = block.Load()
	if err != nil {
		return nil, err
	}

	_, err = block.Compile()
	if err != nil {
		return nil, err
	}

	return block, nil
}

func (tmpl *Template) getBlockFrom(path string) (core.IBlock, error) {
	id := tmpl.getBlockID(path)
	return tmpl.getBlock(id)
}

func (tmpl *Template) getBlock(id string) (core.IBlock, error) {

	path := filepath.Join(tmpl.Root, "__blocks", id)
	if !tmpl.local.fs.IsDir(path) {
		return nil, fmt.Errorf("Block %s not found", id)
	}

	jsFile := filepath.Join("/", id, "main.js")
	tsFile := filepath.Join("/", id, "main.ts")
	htmlFile := filepath.Join("/", id, "main.html")
	block := &Block{
		tmpl: tmpl,
		Block: &core.Block{
			ID: id,
			Codes: core.SourceCodes{
				HTML: core.Source{File: htmlFile},
				JS:   core.Source{File: jsFile},
				TS:   core.Source{File: tsFile},
			},
		},
	}

	return block, nil
}

func (tmpl *Template) getBlockID(path string) string {
	return filepath.Base(path)
}

// Components get the components
func (tmpl *Template) Components() ([]core.IComponent, error) {
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
