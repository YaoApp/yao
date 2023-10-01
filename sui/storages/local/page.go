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

// PageTree gets the page tree.
func (tmpl *Template) PageTree(route string) ([]*core.PageTreeNode, error) {

	exts := []string{"*.sui", "*.html", "*.htm", "*.page"}
	rootNode := &core.PageTreeNode{
		Name:     tmpl.Name,
		IsDir:    true,
		Expand:   true,
		Children: []*core.PageTreeNode{}, // 初始为空的切片
	}

	tmpl.local.fs.Walk(tmpl.Root, func(root, file string, isdir bool) error {
		name := filepath.Base(file)
		relPath := file

		if isdir {
			if strings.HasPrefix(name, "__") {
				return filepath.SkipDir
			}

			// Create directory nodes in the tree structure.
			currentDir := rootNode
			dirs := strings.Split(relPath, string(filepath.Separator))

			for _, dir := range dirs {
				if dir == "" {
					continue
				}

				// Check if the directory node already exists.
				var found bool
				for _, child := range currentDir.Children {
					if child.Name == dir {
						currentDir = child
						found = true
						break
					}
				}
				// If not found, create a new directory node.
				if !found {
					newDir := &core.PageTreeNode{
						Name:     dir,
						IsDir:    true,
						Children: []*core.PageTreeNode{},
						Expand:   true,
					}
					currentDir.Children = append(currentDir.Children, newDir)
					currentDir = newDir
				}
			}
			return nil
		}

		if strings.HasPrefix(name, "__") {
			return nil
		}

		page, err := tmpl.getPageFrom(file)
		if err != nil {
			log.Error("Get page error: %v", err)
			return err
		}

		pageInfo := page.Get()
		active := route == pageInfo.Route

		// Attach the page to the appropriate directory node.
		dirs := strings.Split(relPath, string(filepath.Separator))
		currentDir := rootNode
		for _, dir := range dirs {
			for _, child := range currentDir.Children {
				if child.Name == dir {
					currentDir = child
					break
				}
			}
		}

		currentDir.Expand = active
		currentDir.Children = append(currentDir.Children, &core.PageTreeNode{
			Name:   tmpl.getPageBase(currentDir.Name),
			IsDir:  false,
			IPage:  page,
			Active: active,
		})

		return nil
	}, exts...)

	return rootNode.Children, nil
}

// Page get the page
func (tmpl *Template) Page(route string) (core.IPage, error) {
	path := tmpl.getPagePath(route)
	exts := []string{".sui", ".html", ".htm", ".page"}
	for _, ext := range exts {
		file := fmt.Sprintf("%s%s", path, ext)
		if exist, _ := tmpl.local.fs.Exists(file); exist {
			page, err := tmpl.getPageFrom(file)
			if err != nil {
				return nil, err
			}

			// Load the page source code
			err = page.Load()
			if err != nil {
				return nil, err
			}

			return page, nil
		}
	}
	return nil, fmt.Errorf("Page %s not found", route)
}

func (tmpl *Template) getPageFrom(file string) (core.IPage, error) {
	route := tmpl.getPageRoute(file)
	return tmpl.getPage(route, file)
}

func (tmpl *Template) getPage(route, file string) (core.IPage, error) {
	path := filepath.Dir(file)
	name := tmpl.getPageBase(route)
	return &Page{
		tmpl: tmpl,
		Page: &core.Page{
			Route: route,
			Path:  path,
			Name:  name,
			Codes: core.SourceCodes{
				HTML: core.Source{File: fmt.Sprintf("%s%s", name, filepath.Ext(file))},
				CSS:  core.Source{File: fmt.Sprintf("%s.css", name)},
				JS:   core.Source{File: fmt.Sprintf("%s.js", name)},
				DATA: core.Source{File: fmt.Sprintf("%s.json", name)},
				TS:   core.Source{File: fmt.Sprintf("%s.ts", name)},
				LESS: core.Source{File: fmt.Sprintf("%s.less", name)},
			},
		},
	}, nil
}

func (tmpl *Template) getPageRoute(file string) string {
	return filepath.Dir(file[len(tmpl.Root):])
}

func (tmpl *Template) getPagePath(route string) string {
	name := tmpl.getPageBase(route)
	return filepath.Join(tmpl.Root, route, name)
}

func (tmpl *Template) getPageBase(route string) string {
	return filepath.Base(route)
}

// Load get the page from the storage
func (page *Page) Load() error {

	// Read the Script code
	// Type script is the default language
	tsFile := filepath.Join(page.Path, page.Codes.TS.File)
	if exist, _ := page.tmpl.local.fs.Exists(tsFile); exist {
		tsCode, err := page.tmpl.local.fs.ReadFile(tsFile)
		if err != nil {
			return err
		}
		page.Codes.TS.Code = string(tsCode)

	} else {
		jsFile := filepath.Join(page.Path, page.Codes.JS.File)
		if exist, _ := page.tmpl.local.fs.Exists(jsFile); exist {
			jsCode, err := page.tmpl.local.fs.ReadFile(jsFile)
			if err != nil {
				return err
			}
			page.Codes.JS.Code = string(jsCode)
		}
	}

	// Read the HTML code
	htmlFile := filepath.Join(page.Path, page.Codes.HTML.File)
	if exist, _ := page.tmpl.local.fs.Exists(htmlFile); exist {
		htmlCode, err := page.tmpl.local.fs.ReadFile(htmlFile)
		if err != nil {
			return err
		}
		page.Codes.HTML.Code = string(htmlCode)
	}

	// Read the CSS code
	// @todo: Less support
	cssFile := filepath.Join(page.Path, page.Codes.CSS.File)
	if exist, _ := page.tmpl.local.fs.Exists(cssFile); exist {
		cssCode, err := page.tmpl.local.fs.ReadFile(cssFile)
		if err != nil {
			return err
		}
		page.Codes.CSS.Code = string(cssCode)
	}

	// Read the JSON code
	dataFile := filepath.Join(page.Path, page.Codes.DATA.File)
	if exist, _ := page.tmpl.local.fs.Exists(dataFile); exist {
		dataCode, err := page.tmpl.local.fs.ReadFile(dataFile)
		if err != nil {
			return err
		}
		page.Codes.DATA.Code = string(dataCode)
	}

	// Set the page document
	page.Document = page.tmpl.Document
	return nil
}

// GetPageFromAsset get the page from the asset
func (tmpl *Template) GetPageFromAsset(file string) (core.IPage, error) {
	route := filepath.Dir(file)
	name := tmpl.getPageBase(route)
	return &Page{
		tmpl: tmpl,
		Page: &core.Page{
			Route: route,
			Path:  filepath.Join(tmpl.Root, route),
			Name:  name,
			Codes: core.SourceCodes{
				CSS:  core.Source{File: fmt.Sprintf("%s.css", name)},
				JS:   core.Source{File: fmt.Sprintf("%s.js", name)},
				TS:   core.Source{File: fmt.Sprintf("%s.ts", name)},
				LESS: core.Source{File: fmt.Sprintf("%s.less", name)},
			},
		},
	}, nil
}

// AssetScript get the script
func (page *Page) AssetScript() (*core.Asset, error) {

	// Read the Script code
	// Type script is the default language
	tsFile := filepath.Join(page.Path, page.Codes.TS.File)
	if exist, _ := page.tmpl.local.fs.Exists(tsFile); exist {
		tsCode, err := page.tmpl.local.fs.ReadFile(tsFile)
		if err != nil {
			return nil, err
		}

		jsCode, err := page.CompileTS(tsCode, false)
		if err != nil {
			return nil, err
		}

		return &core.Asset{
			Type:    "text/javascript; charset=utf-8",
			Content: []byte(jsCode),
		}, nil
	}

	jsFile := filepath.Join(page.Path, page.Codes.JS.File)
	if exist, _ := page.tmpl.local.fs.Exists(jsFile); exist {
		jsCode, err := page.tmpl.local.fs.ReadFile(jsFile)
		if err != nil {
			return nil, err
		}

		jsCode, err = page.CompileJS(jsCode, false)
		if err != nil {
			return nil, err
		}

		return &core.Asset{
			Type:    "text/javascript; charset=utf-8",
			Content: jsCode,
		}, nil
	}

	return nil, fmt.Errorf("Page %s script not found", page.Route)
}

// AssetStyle get the style
func (page *Page) AssetStyle() (*core.Asset, error) {
	cssFile := filepath.Join(page.Path, page.Codes.CSS.File)
	if exist, _ := page.tmpl.local.fs.Exists(cssFile); exist {
		cssCode, err := page.tmpl.local.fs.ReadFile(cssFile)
		if err != nil {
			return nil, err
		}

		cssCode, err = page.CompileCSS(cssCode, false)
		if err != nil {
			return nil, err
		}

		return &core.Asset{
			Type:    "text/css; charset=utf-8",
			Content: cssCode,
		}, nil
	}
	return nil, fmt.Errorf("Page %s style not found", page.Route)
}
