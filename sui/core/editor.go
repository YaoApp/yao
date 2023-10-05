package core

import (
	"path/filepath"

	"github.com/PuerkitoBio/goquery"
)

// EditorRender render HTML for the editor
func (page *Page) EditorRender(request *Request) (*ResponseEditor, error) {

	html := page.Codes.HTML.Code
	res := &ResponseEditor{
		HTML:     "",
		CSS:      page.Codes.CSS.Code,
		Scripts:  []string{},
		Styles:   []string{},
		Warnings: []string{},
		Setting:  map[string]interface{}{},
	}

	// Get The scripts and styles
	// Global scripts
	scripts, err := page.GlobalScripts()
	if err != nil {
		res.Warnings = append(res.Warnings, err.Error())
	}
	res.Scripts = append(res.Scripts, scripts...)

	// Global styles
	styles, err := page.GlobalStyles()
	if err != nil {
		res.Warnings = append(res.Warnings, err.Error())
	}
	res.Styles = append(res.Styles, styles...)

	// Page Styles
	if page.Codes.CSS.Code != "" {
		res.Styles = append(res.Styles, filepath.Join("@pages", page.Route, page.Name+".css"))
	}

	// Render the HTML with the data
	// Page Scripts
	if page.Codes.JS.Code != "" {
		res.Scripts = append(res.Scripts, filepath.Join("@pages", page.Route, page.Name+".js"))
	}
	if page.Codes.TS.Code != "" {
		res.Scripts = append(res.Scripts, filepath.Join("@pages", page.Route, page.Name+".ts"))
	}

	data, setting, err := page.Data(request)
	if err != nil {
		res.Warnings = append(res.Warnings, err.Error())
	}

	res.Setting = setting
	if data == nil {
		res.HTML = html
		return res, nil
	}

	if html != "" {
		html, err := page.renderData(html, data, res.Warnings)
		if err != nil {
			res.Warnings = append(res.Warnings, err.Error())
		}
		res.HTML = html
	}

	return res, nil
}

// EditorPageSource get the editor page source code
func (page *Page) EditorPageSource() SourceData {
	return SourceData{
		Source:   page.Codes.HTML.Code,
		Language: "html",
	}
}

// EditorScriptSource get the editor script source code
func (page *Page) EditorScriptSource() SourceData {
	if page.Codes.TS.Code != "" {
		return SourceData{
			Source:   page.Codes.TS.Code,
			Language: "typescript",
		}
	}

	return SourceData{
		Source:   page.Codes.JS.Code,
		Language: "javascript",
	}
}

// EditorStyleSource get the editor style source code
func (page *Page) EditorStyleSource() SourceData {
	return SourceData{
		Source:   page.Codes.CSS.Code,
		Language: "css",
	}
}

// EditorDataSource get the editor data source code
func (page *Page) EditorDataSource() SourceData {
	return SourceData{
		Source:   page.Codes.DATA.Code,
		Language: "json",
	}
}

// GlobalScripts get the global scripts
func (page *Page) GlobalScripts() ([]string, error) {
	if page.Document == nil {
		return []string{}, nil
	}

	doc, err := NewDocument(page.Document)
	if err != nil {
		return []string{}, err
	}

	// Global scripts
	scripts := []string{}
	doc.Find("script").Each(func(i int, s *goquery.Selection) {
		src, _ := s.Attr("src")
		if src != "" {
			scripts = append(scripts, src)
		}
	})

	return scripts, nil
}

// GlobalStyles get the global styles
func (page *Page) GlobalStyles() ([]string, error) {

	if page.Document == nil {
		return []string{}, nil
	}

	doc, err := NewDocument(page.Document)
	if err != nil {
		return []string{}, err
	}

	// Global styles
	styles := []string{}
	doc.Find("link[rel=stylesheet]").Each(func(i int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		if href != "" {
			styles = append(styles, href)
		}
	})

	return styles, nil
}

func (page *Page) document() []byte {
	if page.Document != nil {
		return page.Document
	}
	return DocumentDefault
}
