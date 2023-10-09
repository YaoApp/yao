package core

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/evanw/esbuild/pkg/api"
	"github.com/yaoapp/gou/runtime/transform"
)

// Compile the page
func (page *Page) Compile() {}

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

// BuildHTML build the html
func (page *Page) BuildHTML(assetRoot string) (string, error) {

	html := string(page.Document)
	if page.Codes.HTML.Code != "" {
		html = strings.Replace(html, "{{ __page }}", page.Codes.HTML.Code, 1)
	}

	code := strings.ReplaceAll(html, "@assets", assetRoot)
	res, err := page.CompileHTML([]byte(code), false)
	if err != nil {
		return "", err
	}

	return string(res), nil
}

// BuildStyle build the style
func (page *Page) BuildStyle(assetRoot string) (string, error) {
	if page.Codes.CSS.Code == "" {
		return "", nil
	}

	code := strings.ReplaceAll(page.Codes.CSS.Code, "@assets", assetRoot)
	res, err := page.CompileCSS([]byte(code), false)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("<style>\n%s\n</style>\n", res), nil
}

// BuildScript build the script
func (page *Page) BuildScript(assetRoot string) (string, error) {

	if page.Codes.JS.Code == "" && page.Codes.TS.Code == "" {
		return "", nil
	}

	if page.Codes.TS.Code != "" {
		res, err := page.CompileTS([]byte(page.Codes.TS.Code), false)
		if err != nil {
			return "", err
		}

		return fmt.Sprintf("<script>\n%s\n</script>\n", res), nil
	}

	code := strings.ReplaceAll(page.Codes.JS.Code, "@assets", assetRoot)
	res, err := page.CompileJS([]byte(code), false)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("<script>\n%s\n</script>\n", res), nil
}
