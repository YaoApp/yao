package core

import (
	"regexp"

	"github.com/evanw/esbuild/pkg/api"
	"github.com/yaoapp/gou/runtime/transform"
	"github.com/yaoapp/kun/log"
)

// Compile the page
func (page *Page) Compile(option *BuildOption) (string, error) {

	doc, warnings, err := page.Build(option)
	if err != nil {
		return "", err
	}

	if warnings != nil && len(warnings) > 0 {
		for _, warning := range warnings {
			log.Warn("Compile page %s/%s/%s: %s", page.SuiID, page.TemplateID, page.Route, warning)
		}
	}

	// Page Data
	if page.Codes.DATA.Code != "" {
		doc.Find("body").AppendHtml("\n\n" + `<script name="data" type="json">` + "\n" +
			page.Codes.DATA.Code +
			"\n</script>\n\n",
		)
	}

	// Page Global Data
	if page.GlobalData != nil && len(page.GlobalData) > 0 {
		doc.Find("body").AppendHtml("\n\n" + `<script name="global" type="json">` + "\n" +
			string(page.GlobalData) +
			"\n</script>\n\n",
		)
	}

	// Replace the document
	page.Config = page.GetConfig()
	page.ReplaceDocument(doc)

	html, err := doc.Html()
	if err != nil {
		return "", err
	}

	// @todo: Minify the html
	return html, nil
}

// CompileJS compile the javascript
func (page *Page) CompileJS(source []byte, minify bool) ([]byte, error) {
	jsCode := regexp.MustCompile(`import\s+.*;`).ReplaceAllString(string(source), "")
	if minify {
		minified, err := transform.MinifyJS(jsCode)
		return []byte(minified), err
	}
	return []byte(jsCode), nil
}

// CompileTS compile the typescript
func (page *Page) CompileTS(source []byte, minify bool) ([]byte, error) {
	tsCode := regexp.MustCompile(`import\s+.*;`).ReplaceAllString(string(source), "")
	if minify {
		jsCode, err := transform.TypeScript(string(tsCode), api.TransformOptions{
			Target:            api.ESNext,
			MinifyWhitespace:  true,
			MinifyIdentifiers: true,
			MinifySyntax:      true,
		})

		return []byte(jsCode), err
	}

	jsCode, err := transform.TypeScript(string(tsCode), api.TransformOptions{Target: api.ESNext})
	return []byte(jsCode), err
}

// CompileCSS compile the css
func (page *Page) CompileCSS(source []byte, minify bool) ([]byte, error) {
	if minify {
		cssCode, err := transform.MinifyCSS(string(source))
		return []byte(cssCode), err
	}
	return source, nil
}

// CompileHTML compile the html
func (page *Page) CompileHTML(source []byte, minify bool) ([]byte, error) {
	return source, nil
}
