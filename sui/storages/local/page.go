package local

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	jsoniter "github.com/json-iterator/go"
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
			if strings.HasPrefix(name, "__") || name == ".tmp" {
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
		log.Debug("[PageTree] Walk | file: %s isdir: %v name: %v", relPath, isdir, name)

		if isdir {
			if strings.HasPrefix(name, "__") || name == ".tmp" {
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

				log.Debug("[PageTree] Walk | dirs: %s found: %v", dir, found)
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

		log.Debug("[PageTree] getPageFrom | file: %s", file)
		page, err := tmpl.getPageFrom(file)
		if err != nil {
			log.Error("Get page error: %v", err)
			return err
		}

		pageInfo := page.Get()
		active := route == pageInfo.Route
		log.Debug("[PageTree] getPageFrom |\t pageInfo.Name: %s", pageInfo.Name)

		// Attach the page to the appropriate directory node.
		dirs := strings.Split(relPath, string(filepath.Separator))
		currentDir := rootNode
		log.Debug("[PageTree] currentDir | name: %s Children: %d", currentDir.Name, len(currentDir.Children))

		for _, dir := range dirs {
			for _, child := range currentDir.Children {
				log.Debug("[PageTree] currentDir.Children | child.Name: %s dir:%s", child.Name, dir)
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
	return nil, fmt.Errorf("%s not found", route)
}

// PageExist check if the page exist
func (tmpl *Template) PageExist(route string) bool {
	path := tmpl.getPagePath(route)
	exts := []string{".sui", ".html", ".htm", ".page"}
	for _, ext := range exts {
		file := fmt.Sprintf("%s%s", path, ext)
		if exist, _ := tmpl.local.fs.Exists(file); exist {
			return true
		}
	}
	return false
}

// RemovePage remove the page
func (tmpl *Template) RemovePage(route string) error {
	if !tmpl.PageExist(route) {
		return nil
	}

	path := filepath.Join(tmpl.Root, route)
	name := filepath.Base(path) + ".*"
	name = strings.ReplaceAll(name, "[", "\\[")
	name = strings.ReplaceAll(name, "]", "\\]")
	err := tmpl.local.fs.Walk(path, func(root, file string, isdir bool) error {
		if isdir {
			return nil
		}
		return tmpl.local.fs.Remove(file)
	}, name)

	if err != nil {
		return err
	}

	// Remove .tmp directory
	tmpPath := filepath.Join(tmpl.Root, route, ".tmp")
	if exist, _ := tmpl.local.fs.Exists(tmpPath); exist {
		err = tmpl.local.fs.RemoveAll(tmpPath)
		if err != nil {
			return err
		}
	}

	return tmpl.removeEmptyPath(path)
}

func (tmpl *Template) removeEmptyPath(path string) error {
	dirs, err := tmpl.local.fs.ReadDir(path, false)
	if err != nil {
		return err
	}

	if len(dirs) == 0 {
		err = tmpl.local.fs.RemoveAll(path)
		if err != nil {
			return err
		}
		parent := filepath.Dir(path)
		if parent == tmpl.Root {
			return nil
		}
		return tmpl.removeEmptyPath(parent)
	}
	return nil
}

// SaveAs save the page as
func (page *Page) SaveAs(route string, setting *core.PageSetting) (core.IPage, error) {

	if page.tmpl.PageExist(route) {
		return nil, fmt.Errorf("Page %s already exist", route)
	}

	root := page.tmpl.Root
	target := filepath.Join(root, route)
	targetBaseName := filepath.Base(target)
	baseName := filepath.Base(page.Path)
	patterns := []string{"*.js", "*.ts", "*.html", "*.css", "*.config", "*.json"}
	err := page.tmpl.local.fs.Walk(page.Path, func(root, file string, isdir bool) error {
		if isdir {
			return nil
		}

		if filepath.Base(filepath.Dir(file)) != baseName {
			return nil
		}

		fileName := filepath.Base(file)
		targetFileName := strings.Replace(fileName, baseName, targetBaseName, 1)
		targetFile := filepath.Join(target, targetFileName)

		// Copy the file
		return page.tmpl.local.fs.Copy(file, targetFile)

	}, patterns...)

	if err != nil {
		return nil, err
	}

	return page.tmpl.Page(route)
}

// CreatePage create a new page by the source
func (tmpl *Template) CreatePage(source string) core.IPage {
	name := uuid.New().String()
	route := "/" + uuid.New().String()
	return &Page{
		tmpl: tmpl,
		Page: &core.Page{
			Route:      route,
			TemplateID: tmpl.ID,
			SuiID:      tmpl.local.ID,
			Path:       filepath.Join(tmpl.Root, route),
			Name:       name,
			Codes: core.SourceCodes{
				HTML: core.Source{File: fmt.Sprintf("%s.html", name), Code: source},
				CSS:  core.Source{File: fmt.Sprintf("%s.css", name)},
				JS:   core.Source{File: fmt.Sprintf("%s.js", name)},
				TS:   core.Source{File: fmt.Sprintf("%s.ts", name)},
				LESS: core.Source{File: fmt.Sprintf("%s.less", name)},
				CONF: core.Source{File: fmt.Sprintf("%s.config", name)},
			},
		},
	}
}

// CreateEmptyPage create a new empty
func (tmpl *Template) CreateEmptyPage(route string, setting *core.PageSetting) (core.IPage, error) {
	if tmpl.PageExist(route) {
		return nil, fmt.Errorf("Page %s already exist", route)
	}

	// Create the page directory
	name := tmpl.getPageBase(route)
	page := &Page{
		tmpl: tmpl,
		Page: &core.Page{
			Route:      route,
			TemplateID: tmpl.ID,
			SuiID:      tmpl.local.ID,
			Path:       filepath.Join(tmpl.Root, route),
			Name:       name,
			Codes: core.SourceCodes{
				HTML: core.Source{File: fmt.Sprintf("%s.html", name)},
				CSS:  core.Source{File: fmt.Sprintf("%s.css", name)},
				JS:   core.Source{File: fmt.Sprintf("%s.js", name)},
				TS:   core.Source{File: fmt.Sprintf("%s.ts", name)},
				LESS: core.Source{File: fmt.Sprintf("%s.less", name)},
				CONF: core.Source{File: fmt.Sprintf("%s.config", name)},
			},
		},
	}

	title := route
	if setting != nil {
		title = setting.Title
	}

	err := page.Save(&core.RequestSource{
		Page:       &core.SourceData{Source: fmt.Sprintf("<div>%s</div>", title), Language: "html"},
		Setting:    setting,
		NeedToSave: core.ReqeustSourceNeedToSave{Page: true, Setting: true},
	})
	if err != nil {
		return nil, err
	}
	return page, nil
}

// Remove remove the page
func (page *Page) Remove() error {
	return page.tmpl.RemovePage(page.Route)
}

// GetPageFromAsset get the page from the asset
func (tmpl *Template) GetPageFromAsset(file string) (core.IPage, error) {
	route := filepath.Dir(file)
	name := tmpl.getPageBase(route)
	return &Page{
		tmpl: tmpl,
		Page: &core.Page{
			Route:      route,
			TemplateID: tmpl.ID,
			SuiID:      tmpl.local.ID,
			Path:       filepath.Join(tmpl.Root, route),
			Name:       name,
			Codes: core.SourceCodes{
				CSS:  core.Source{File: fmt.Sprintf("%s.css", name)},
				JS:   core.Source{File: fmt.Sprintf("%s.js", name)},
				TS:   core.Source{File: fmt.Sprintf("%s.ts", name)},
				LESS: core.Source{File: fmt.Sprintf("%s.less", name)},
			},
		},
	}, nil
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
			Route:      route,
			Path:       path,
			Name:       name,
			TemplateID: tmpl.ID,
			SuiID:      tmpl.local.ID,
			Codes: core.SourceCodes{
				HTML: core.Source{File: fmt.Sprintf("%s%s", name, filepath.Ext(file))},
				CSS:  core.Source{File: fmt.Sprintf("%s.css", name)},
				JS:   core.Source{File: fmt.Sprintf("%s.js", name)},
				DATA: core.Source{File: fmt.Sprintf("%s.json", name)},
				TS:   core.Source{File: fmt.Sprintf("%s.ts", name)},
				LESS: core.Source{File: fmt.Sprintf("%s.less", name)},
				CONF: core.Source{File: fmt.Sprintf("%s.config", name)},
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

	// Read the config code
	confFile := filepath.Join(page.Path, page.Codes.CONF.File)
	if exist, _ := page.tmpl.local.fs.Exists(confFile); exist {
		confCode, err := page.tmpl.local.fs.ReadFile(confFile)
		if err != nil {
			return err
		}
		page.Codes.CONF.Code = string(confCode)
	}

	// Set the page CacheStore
	page.CacheStore = page.tmpl.local.DSL.CacheStore

	// Set the page document
	page.Document = page.tmpl.Document

	// Set the page global data
	page.GlobalData = page.tmpl.GlobalData

	// Load the backend script
	err := page.loadBackendScript()
	if err != nil {
		return err
	}

	return nil
}

// SaveTemp save page to the temp file
func (page *Page) SaveTemp(request *core.RequestSource) error {
	tempPath := filepath.Join(page.Path, ".tmp", request.UID)
	return page.save(tempPath, request)

}

// Save save page to the storage, if the page not exist, create it
func (page *Page) Save(request *core.RequestSource) error {
	path := page.Path
	err := page.save(path, request)
	if err != nil {
		return err
	}

	// Remove the temp file
	tempPath := filepath.Join(page.Path, ".tmp", request.UID)
	if exist, _ := page.tmpl.local.fs.Exists(tempPath); exist {
		err = page.tmpl.local.fs.RemoveAll(tempPath)
		if err != nil {
			return err
		}

		dirs, err := page.tmpl.local.fs.ReadDir(filepath.Join(page.Path, ".tmp"), false)
		if err != nil {
			return err
		}

		if len(dirs) == 0 {
			return page.tmpl.local.fs.Remove(filepath.Join(page.Path, ".tmp"))
		}
	}
	return nil
}

// save page to the temp file
func (page *Page) save(path string, request *core.RequestSource) error {
	if request.NeedToSave.Board {
		err := page.saveBoard(path, request.Board)
		if err != nil {
			return err
		}
	}

	if request.NeedToSave.Page {
		err := page.savePage(path, request.Page)
		if err != nil {
			return err
		}
	}

	if request.NeedToSave.Style {
		err := page.saveStyle(path, request.Style)
		if err != nil {
			return err
		}
	}

	if request.NeedToSave.Script {
		err := page.saveScript(path, request.Script)
		if err != nil {
			return err
		}
	}

	if request.NeedToSave.Data {
		err := page.saveData(path, request.Data)
		if err != nil {
			return err
		}
	}

	if request.NeedToSave.Setting || request.NeedToSave.Mock {
		err := page.saveSetting(path, request.Setting, request.Mock)
		if err != nil {
			return err
		}
	}

	return nil
}

// saveBoard save the board to the storage
func (page *Page) saveBoard(path string, board *core.BoardSourceData) error {

	htmlFile := filepath.Join(path, page.Codes.HTML.File)
	_, err := page.tmpl.local.fs.WriteFile(htmlFile, []byte(board.HTML), 0644)
	if err != nil {
		return err
	}

	cssFile := filepath.Join(path, page.Codes.CSS.File)
	_, err = page.tmpl.local.fs.WriteFile(cssFile, []byte(board.Style), 0644)
	return err
}

func (page *Page) savePage(path string, src *core.SourceData) error {
	if src.Language != "html" {
		return fmt.Errorf("Page %s language not support", page.Route)
	}

	htmlFile := filepath.Join(path, page.Codes.HTML.File)
	_, err := page.tmpl.local.fs.WriteFile(htmlFile, []byte(src.Source), 0644)
	return err
}

func (page *Page) saveStyle(path string, src *core.SourceData) error {
	if src.Language != "css" {
		return fmt.Errorf("Page %s language not support", page.Route)
	}

	cssFile := filepath.Join(path, page.Codes.CSS.File)
	_, err := page.tmpl.local.fs.WriteFile(cssFile, []byte(src.Source), 0644)
	return err
}

func (page *Page) saveScript(path string, src *core.SourceData) error {

	switch src.Language {
	case "typescript":
		tsFile := filepath.Join(path, page.Codes.TS.File)
		_, err := page.tmpl.local.fs.WriteFile(tsFile, []byte(src.Source), 0644)
		return err
	case "javascript":
		jsFile := filepath.Join(path, page.Codes.JS.File)
		_, err := page.tmpl.local.fs.WriteFile(jsFile, []byte(src.Source), 0644)
		return err

	default:
		return fmt.Errorf("Page %s language not support", page.Route)
	}
}

func (page *Page) saveData(path string, src *core.SourceData) error {
	if src.Language != "json" {
		return fmt.Errorf("Page %s language not support", page.Route)
	}

	dataFile := filepath.Join(path, page.Codes.DATA.File)
	_, err := page.tmpl.local.fs.WriteFile(dataFile, []byte(src.Source), 0644)
	return err
}

func (page *Page) saveSetting(path string, setting *core.PageSetting, mock *core.PageMock) error {

	config := core.PageConfig{Mock: mock}
	if setting != nil {
		config.PageSetting = *setting
	}

	configBytes, err := jsoniter.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	if setting != nil || mock != nil {
		dataFile := filepath.Join(path, page.Codes.CONF.File)
		_, err = page.tmpl.local.fs.WriteFile(dataFile, configBytes, 0644)
		return err
	}

	return nil
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

		jsCode, _, err := page.CompileTS(tsCode, false)
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

		jsCode, _, err = page.CompileJS(jsCode, false)
		if err != nil {
			return nil, err
		}

		return &core.Asset{
			Type:    "text/javascript; charset=utf-8",
			Content: jsCode,
		}, nil
	}

	return nil, fmt.Errorf("%s script not found", page.Route)
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
	return nil, fmt.Errorf("%s style not found", page.Route)
}
