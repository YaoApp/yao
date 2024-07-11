package core

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/evanw/esbuild/pkg/api"
	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/runtime/transform"
	"github.com/yaoapp/kun/log"
	"golang.org/x/net/html"
)

var quoteRe = "'\"`"
var importRe = regexp.MustCompile(`import\s*\t*\n*[^;]*;`)                              // import { foo, bar } from 'hello'; ...
var importAssetsRe = regexp.MustCompile(`import\s*\t*\n*\s*['"]@assets\/([^'"]+)['"];`) // import '@assets/foo.js'; or import "@assets/foo.js";
var transStmtReSingle = regexp.MustCompile(`'::([^:']+)'`)
var transStmtReDouble = regexp.MustCompile(`"::([^:"]+)"`)
var transFuncRe = regexp.MustCompile(`__m\s*\(\s*["'](.*?)["']\s*\)`)

// AssetsRe is the regexp for assets
var AssetsRe = regexp.MustCompile(`[` + quoteRe + `]@assets\/([^` + quoteRe + `]+)[` + quoteRe + `]`) // '@assets/foo.js' or "@assets/foo.js" or `@assets/foo`

// Compile the page
func (page *Page) Compile(ctx *BuildContext, option *BuildOption) (string, []string, error) {

	doc, warnings, err := page.Build(ctx, option)
	if err != nil {
		return "", warnings, err
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
	if page.Config != nil {
		body.AppendHtml("\n\n" + `<script name="config" type="json">` + "\n" +
			page.ExportConfig() +
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

	// Add the translation marks
	sequence := 0
	err = page.TranslateMarks(ctx, option, doc, &sequence)
	if err != nil {
		return "", warnings, err
	}

	page.ReplaceDocument(doc)
	html, err := doc.Html()
	if err != nil {
		return "", warnings, err
	}

	// @todo: Minify the html
	return html, warnings, nil
}

// CompileAsComponent compile the page as component
func (page *Page) CompileAsComponent(ctx *BuildContext, option *BuildOption) (string, []string, error) {

	opt := *option
	opt.IgnoreDocument = true
	opt.WithWrapper = true
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

	if body.Children().Length() == 0 {
		return "", warnings, fmt.Errorf("page %s as component should have one root element", page.Route)
	}

	if body.Children().Length() > 1 {
		return "", warnings, fmt.Errorf("page %s as component should have only one root element", page.Route)
	}

	body.Children().First().AppendHtml(fmt.Sprintf(`<script name="scripts" type="json">%s</script>`+"\n", rawScripts))
	body.Children().First().AppendHtml(fmt.Sprintf(`<script name="styles" type="json">%s</script>`+"\n", rawStyles))
	body.Children().First().AppendHtml(fmt.Sprintf(`<script name="option" type="json">%s</script>`+"\n", rawOption))

	// Add the translation marks
	sequence := 0
	err = page.TranslateMarks(ctx, option, doc, &sequence)
	if err != nil {
		return "", warnings, err
	}

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

// HTML return the html of the script
func (script ScriptNode) HTML() string {

	attrs := []string{
		"s:ns=\"" + script.Namespace + "\"",
		"s:cn=\"" + script.Component + "\"",
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

	source := fmt.Sprintf(`function %s(){%s};`, script.Component, script.Source)
	return "<script " + strings.Join(attrs, " ") + ">\n" + source + "\n</script>"
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

// TranslateMarks add the translation marks to the document
func (page *Page) TranslateMarks(ctx *BuildContext, option *BuildOption, doc *goquery.Document, sequence *int) error {

	if doc.Length() == 0 {
		return nil
	}

	if ctx.translations == nil {
		ctx.translations = []Translation{}
	}

	root := doc.First()
	translations, err := page.translateNode(root.Nodes[0], sequence)
	if err != nil {
		return err
	}

	if translations != nil {
		ctx.translations = append(ctx.translations, translations...)
	}
	return nil
}

func (page *Page) translateNode(node *html.Node, sequence *int) ([]Translation, error) {

	translations := []Translation{}
	*sequence = *sequence + 1

	switch node.Type {
	case html.DocumentNode:
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			trans, err := page.translateNode(child, sequence)
			if err != nil {
				return nil, err
			}
			translations = append(translations, trans...)
		}
		break

	case html.ElementNode:

		// Script
		if node.Data == "script" {
			code := goquery.NewDocumentFromNode(node).Text()
			if code != "" {
				translations := []Translation{}
				matches := transFuncRe.FindAllStringSubmatch(code, -1)
				for _, match := range matches {
					key := Namespace(page.Route, *sequence)
					translations = append(translations, Translation{
						Key:     key,
						Message: match[1],
						Type:    "script",
					})
					*sequence = *sequence + 1
				}
			}
			break
		}

		sel := goquery.NewDocumentFromNode(node)
		for _, attr := range node.Attr {

			trans, keys, err := page.translateText(attr.Val, sequence, "attr")
			if err != nil {
				return nil, err
			}
			if len(keys) > 0 {
				raw := strings.Join(keys, ",")
				sel.SetAttr("s:trans-attr-"+attr.Key, raw)
				translations = append(translations, trans...)
			}

		}

		// Node Attributes
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			trans, err := page.translateNode(child, sequence)
			if err != nil {
				return nil, err
			}
			translations = append(translations, trans...)
		}
		break

	case html.TextNode:
		parentSel := goquery.NewDocumentFromNode(node.Parent)
		if _, has := parentSel.Attr("s:trans"); has {
			key := Namespace(page.Route, *sequence)
			message := strings.TrimSpace(node.Data)
			if message != "" {
				translations = append(translations, Translation{
					Key:     key,
					Message: message,
					Type:    "text",
				})
				parentSel.SetAttr("s:trans-node", key)
				*sequence = *sequence + 1
			}
			parentSel.RemoveAttr("s:trans")
		}

		trans, keys, err := page.translateText(node.Data, sequence, "text")
		if err != nil {
			return nil, err
		}
		if len(keys) > 0 {
			raw := strings.Join(keys, ",")
			parentSel.SetAttr("s:trans-text", raw)
			parentSel.RemoveAttr("s:trans")
			translations = append(translations, trans...)
		}
		break
	}

	return translations, nil
}

func (page *Page) translateText(text string, sequence *int, typ string) ([]Translation, []string, error) {
	translations := []Translation{}
	matches := stmtRe.FindAllStringSubmatch(text, -1)
	keys := []string{}
	for _, match := range matches {
		text := strings.TrimSpace(match[1])
		transMatches := transStmtReSingle.FindAllStringSubmatch(text, -1)
		if len(transMatches) == 0 {
			transMatches = transStmtReDouble.FindAllStringSubmatch(text, -1)
		}
		for _, transMatch := range transMatches {
			message := strings.TrimSpace(transMatch[1])
			key := Namespace(page.Route, *sequence)
			keys = append(keys, key)
			translations = append(translations, Translation{
				Key:     key,
				Message: message,
				Type:    typ,
			})
			*sequence = *sequence + 1
		}
	}
	return translations, keys, nil
}
