package core

import (
	"regexp"

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
