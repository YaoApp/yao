package core

import (
	"github.com/PuerkitoBio/goquery"
)

// EditorRender render HTML for the editor
func (page *Page) EditorRender() (*ResponseEditorRender, error) {

	res := &ResponseEditorRender{
		HTML:     "",
		CSS:      page.Codes.CSS.Code,
		Scripts:  []string{},
		Styles:   []string{},
		Warnings: []string{},
		Config:   page.GetConfig(),
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

	// Render the page
	request := NewRequestMock(page.Config.Mock)

	// Set Default Sid
	if request.Sid == "" {
		request.Sid, _ = page.Sid()
	}

	link := page.Link(request)
	if request.URL.Path == "" {
		request.URL.Path = link
	}

	// Render tools
	// res.Scripts = append(res.Scripts, filepath.Join("@assets", "__render.js"))
	// res.Styles = append(res.Styles, filepath.Join("@assets", "__render.css"))
	ctx := NewBuildContext(nil)
	doc, warnings, err := page.Build(ctx, &BuildOption{
		SSR:             true,
		IgnoreAssetRoot: true,
		IgnoreDocument:  true,
		WithWrapper:     true,
		KeepPageTag:     true,
	})

	if err != nil {
		res.Warnings = append(res.Warnings, err.Error())
	}

	if warnings != nil {
		res.Warnings = append(res.Warnings, warnings...)
	}

	// Block save event
	jsCode := `
	document.addEventListener('keydown', function (event) {
		const isCtrlOrCmdPressed = event.ctrlKey || event.metaKey;
		const isSPressed = event.key === 's';
		if (isCtrlOrCmdPressed && isSPressed) {
		event.preventDefault();
		console.log('Control/Command + S pressed in iframe! Default save behavior prevented.');
		}
	});
	`
	doc.Find("body").AppendHtml(`<script type="text/javascript">` + jsCode + `</script>`)
	res.HTML, err = doc.Html()
	if err != nil {
		return nil, err
	}

	var data Data = nil
	if page.Codes.DATA.Code != "" {
		data, err = page.Exec(request)
		if err != nil {
			res.Warnings = append(res.Warnings, err.Error())
		}
	}
	res.Render(data)

	// Set the title
	res.Config.Rendered = &PageConfigRendered{
		Title: page.RenderTitle(data),
		Link:  link,
	}

	return res, nil
}

// Render render for the html
func (res *ResponseEditorRender) Render(data map[string]interface{}) error {
	if res.HTML == "" {
		return nil
	}

	if data == nil || len(data) == 0 {
		return nil
	}

	var err error
	parser := NewTemplateParser(data, &ParserOption{Editor: true, Debug: true})

	res.HTML, err = parser.Render(res.HTML)
	if err != nil {
		return err
	}

	if len(parser.errors) > 0 {
		for _, err := range parser.errors {
			res.Warnings = append(res.Warnings, err.Error())
		}
	}

	return nil
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
