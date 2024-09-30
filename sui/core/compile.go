package core

import (
	"fmt"
	"hash/fnv"
	"regexp"
	"strings"

	"github.com/evanw/esbuild/pkg/api"
	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/runtime/transform"
	"github.com/yaoapp/kun/log"
)

var quoteRe = "'\"`"
var importRe = regexp.MustCompile(`import\s*\t*\n*[^;]*;`)                              // import { foo, bar } from 'hello'; ...
var importAssetsRe = regexp.MustCompile(`import\s*\t*\n*\s*['"]@assets\/([^'"]+)['"];`) // import '@assets/foo.js'; or import "@assets/foo.js";

// AssetsRe is the regexp for assets
var AssetsRe = regexp.MustCompile(`[` + quoteRe + `]@assets\/([^` + quoteRe + `]+)[` + quoteRe + `]`) // '@assets/foo.js' or "@assets/foo.js" or `@assets/foo`

// Compile the page
func (page *Page) Compile(ctx *BuildContext, option *BuildOption) (string, string, []string, error) {

	doc, warnings, err := page.Build(ctx, option)
	if err != nil {
		return "", "", warnings, fmt.Errorf("Page build error: %s", err.Error())
	}

	if warnings != nil && len(warnings) > 0 {
		for _, warning := range warnings {
			log.Warn("Compile page %s/%s/%s: %s", page.SuiID, page.TemplateID, page.Route, warning)
		}
	}

	body := doc.Find("body")
	head := doc.Find("head")

	// Scripts
	if ctx != nil && ctx.scripts != nil {
		for _, script := range ctx.scripts {
			if script.Parent == "head" {
				head.AppendHtml(script.HTML() + "\n")
				continue
			}
			body.AppendHtml(script.HTML() + "\n")
		}
	}

	// Styles
	if ctx != nil && ctx.styles != nil {
		for _, style := range ctx.styles {
			if style.Parent == "head" {
				head.AppendHtml(style.HTML() + "\n")
				continue
			}
			body.AppendHtml(style.HTML() + "\n")
		}

	}

	// Page Config
	page.Config = page.GetConfig()

	// Config Data
	config := ""
	if page.Config != nil {
		config = page.ExportConfig()
		body.AppendHtml("\n\n" + `<script name="config" type="json">` + "\n" +
			config +
			"\n</script>\n\n",
		)
	}

	// Page Data
	if page.Codes.DATA.Code != "" {
		body.AppendHtml("\n\n" + `<script name="data" type="json">` + "\n" +
			page.Codes.DATA.Code +
			"\n</script>\n\n",
		)
	}

	// Page Global Data
	if page.GlobalData != nil && len(page.GlobalData) > 0 {
		body.AppendHtml("\n\n" + `<script name="global" type="json">` + "\n" +
			string(page.GlobalData) +
			"\n</script>\n\n",
		)
	}

	// Page Components
	if ctx != nil && ctx.components != nil && len(ctx.components) > 0 {
		rawComponents, _ := jsoniter.MarshalToString(ctx.components)
		body.AppendHtml("\n\n" + `<script name="imports" type="json">` + "\n" + rawComponents + "\n</script>\n\n")
	}

	page.ReplaceDocument(doc)
	html, err := doc.Html()
	if err != nil {
		return "", "", warnings, fmt.Errorf("Generate html error: %s", err.Error())
	}

	// @todo: Minify the html
	return html, config, warnings, nil
}

// CompileAsComponent compile the page as component
func (page *Page) CompileAsComponent(ctx *BuildContext, option *BuildOption) (string, []string, error) {

	opt := *option
	opt.IgnoreDocument = true
	opt.WithWrapper = true
	opt.JitMode = true
	doc, warnings, err := page.Build(ctx, &opt)
	if err != nil {
		return "", warnings, err
	}

	if warnings != nil && len(warnings) > 0 {
		for _, warning := range warnings {
			log.Warn("Compile page %s/%s/%s: %s", page.SuiID, page.TemplateID, page.Route, warning)
		}
	}

	body := doc.Find("body")
	rawScripts, err := jsoniter.MarshalToString(ctx.scripts)
	if err != nil {
		return "", warnings, err
	}

	rawStyles, err := jsoniter.MarshalToString(ctx.styles)
	if err != nil {
		return "", warnings, err
	}

	rawOption, err := jsoniter.MarshalToString(option)
	if err != nil {
		return "", warnings, err
	}

	rawComponents := "{}"
	if ctx != nil && ctx.components != nil && len(ctx.components) > 0 {
		rawComponents, _ = jsoniter.MarshalToString(ctx.components)
	}

	if body.Children().Length() == 0 {
		return "", warnings, fmt.Errorf("page %s as component should have one root element", page.Route)
	}

	if body.Children().Length() > 1 {
		return "", warnings, fmt.Errorf("page %s as component should have only one root element", page.Route)
	}

	body.Children().First().AppendHtml(fmt.Sprintf(`<script name="scripts" type="json">%s</script>`+"\n", rawScripts))
	body.Children().First().AppendHtml(fmt.Sprintf(`<script name="styles" type="json">%s</script>`+"\n", rawStyles))
	body.Children().First().AppendHtml(fmt.Sprintf(`<script name="option" type="json">%s</script>`+"\n", rawOption))
	body.Children().First().AppendHtml(fmt.Sprintf(`<script name="imports" type="json">%s</script>`+"\n", rawComponents))

	html, err := body.Html()
	return html, warnings, err
}

// CompileJS compile the javascript
func (page *Page) CompileJS(source []byte, minify bool) ([]byte, []string, error) {
	scripts := []string{}
	matches := importAssetsRe.FindAll(source, -1)
	for _, match := range matches {
		assets := AssetsRe.FindStringSubmatch(string(match))
		if len(assets) > 1 {
			scripts = append(scripts, assets[1])
		}
	}
	jsCode := importRe.ReplaceAllString(string(source), "")
	if minify {
		minified, err := transform.MinifyJS(jsCode, api.ES2015)
		return []byte(minified), scripts, err
	}

	jsCode, err := transform.JavaScript(string(jsCode), api.TransformOptions{Target: api.ES2015})
	return []byte(jsCode), scripts, err
}

// CompileTS compile the typescript
func (page *Page) CompileTS(source []byte, minify bool) ([]byte, []string, error) {

	scripts := []string{}
	matches := importAssetsRe.FindAll(source, -1)
	for _, match := range matches {
		assets := AssetsRe.FindStringSubmatch(string(match))
		if len(assets) > 1 {
			scripts = append(scripts, assets[1])
		}
	}

	tsCode := importRe.ReplaceAllString(string(source), "")
	if minify {
		jsCode, err := transform.TypeScript(string(tsCode), api.TransformOptions{
			Target:            api.ES2015,
			MinifyWhitespace:  true,
			MinifyIdentifiers: true,
			MinifySyntax:      true,
		})
		return []byte(jsCode), scripts, err
	}

	jsCode, err := transform.TypeScript(string(tsCode), api.TransformOptions{Target: api.ES2015})
	return []byte(jsCode), scripts, err
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

// Hash return the hash of the script
func (script ScriptNode) Hash() string {
	raw := fmt.Sprintf("%s|%v|%s", script.Component, script.Attrs, script.Parent)
	h := fnv.New64a()
	h.Write([]byte(raw))
	return fmt.Sprintf("script_%x", h.Sum64())
}

// HTML return the html of the script
func (script ScriptNode) HTML() string {

	attrs := []string{
		"s:ns=\"" + script.Namespace + "\"",
		"s:cn=\"" + script.Component + "\"",
		"s:hash=\"" + script.Hash() + "\"",
	}
	if script.Attrs != nil {
		for _, attr := range script.Attrs {
			attrs = append(attrs, attr.Key+"=\""+attr.Val+"\"")
		}
	}
	// Inline Script
	if script.Source == "" {
		return "<script " + strings.Join(attrs, " ") + "></script>"
	}
	return "<script " + strings.Join(attrs, " ") + ">\n" + script.Source + "\n</script>"
}

// ComponentHTML return the html of the script
func (script ScriptNode) ComponentHTML(ns string) string {

	attrs := []string{
		"s:ns=\"" + ns + "\"",
		"s:cn=\"" + script.Component + "\"",
		"s:hash=\"" + script.Hash() + "\"",
	}
	if script.Attrs != nil {
		for _, attr := range script.Attrs {
			attrs = append(attrs, attr.Key+"=\""+attr.Val+"\"")
		}
	}
	// Inline Script
	if script.Source == "" {
		return "<script " + strings.Join(attrs, " ") + "></script>"
	}

	source := script.Source
	if !strings.Contains(script.Source, "function "+script.Component) {
		source = fmt.Sprintf(`function %s( component ){%s};`, script.Component, script.Source)
	}

	if script.Component == "" {
		return "<script " + strings.Join(attrs, " ") + ">\n" + script.Source + "\n</script>"
	}
	return "<script " + strings.Join(attrs, " ") + ">\n" + source + "\n</script>"
}

// AttrOr return the attribute value or the default value
func (script ScriptNode) AttrOr(key string, or string) string {
	for _, attr := range script.Attrs {
		if attr.Key == key {
			return attr.Val
		}
	}
	return or
}

// AttrOr return the attribute value or the default value
func (style StyleNode) AttrOr(key string, or string) string {
	for _, attr := range style.Attrs {
		if attr.Key == key {
			return attr.Val
		}
	}
	return or
}

// HTML return the html of the style node
func (style StyleNode) HTML() string {
	attrs := []string{
		"s:ns=\"" + style.Namespace + "\"",
		"s:cn=\"" + style.Component + "\"",
	}
	if style.Attrs != nil {
		for _, attr := range style.Attrs {
			attrs = append(attrs, attr.Key+"=\""+attr.Val+"\"")
		}
	}
	// Inline Style
	if style.Source == "" {
		return "<link " + strings.Join(attrs, " ") + "></link>"
	}
	return "<style " + strings.Join(attrs, " ") + ">\n" + style.Source + "\n</style>"

}
