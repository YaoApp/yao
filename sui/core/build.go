package core

import (
	"bufio"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/fatih/color"
	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/kun/log"
	"golang.org/x/net/html"
)

var slotRe = regexp.MustCompile(`\[\{([^\}]+)\}\]`)
var cssRe = regexp.MustCompile(`([\.a-z0-9A-Z-:# ]+)\{`)
var langFuncRe = regexp.MustCompile(`L\s*\(\s*["'](.*?)["']\s*\)`)
var langAttrRe = regexp.MustCompile(`'::(.*?)'`)

// Build is the struct for the public
func (page *Page) Build(ctx *BuildContext, option *BuildOption) (*goquery.Document, []string, error) {

	// Create the context if not exists
	if ctx == nil {
		ctx = NewBuildContext(nil)
	}

	warnings := []string{}
	html, err := page.BuildHTML(option)
	if err != nil {
		warnings = append(warnings, err.Error())
	}

	// Add Style & Script & Warning
	doc, err := NewDocumentString(html)
	if err != nil {
		warnings = append(warnings, err.Error())
	}

	// Append the nested html
	err = page.parse(ctx, doc, option, warnings)
	if err != nil {
		warnings = append(warnings, err.Error())
	}

	// Add Style
	style, err := page.BuildStyle(ctx, option)
	if err != nil {
		warnings = append(warnings, err.Error())
	}
	doc.Selection.Find("head").AppendHtml(fmt.Sprintf("\n"+`<style type="text/css">`+"\n%s\n"+`</style>`+"\n%s\n", strings.Join(ctx.styles, "\n"), style))

	// Add Script
	code, scripts, err := page.BuildScript(ctx, option, option.Namespace)
	if err != nil {
		warnings = append(warnings, err.Error())
	}
	if scripts != nil {
		for _, script := range scripts {
			doc.Selection.Find("body").AppendHtml("\n" + `<script src="` + script + `"></script>` + "\n")
		}
	}
	doc.Selection.Find("body").AppendHtml(fmt.Sprintf("\n"+`<script type="text/javascript">`+"\n%s\n"+`</script>`+"\n%s\n", strings.Join(ctx.scripts, "\n"), code))
	return doc, warnings, nil
}

// BuildForImport build the page for import
func (page *Page) BuildForImport(ctx *BuildContext, option *BuildOption, slots map[string]interface{}, attrs map[string]string) (string, string, string, []string, error) {

	defer func() {
		if option.ComponentName != "" {
			ctx.components[option.ComponentName] = true
		}
	}()

	warnings := []string{}
	html, err := page.BuildHTML(option)
	if err != nil {
		warnings = append(warnings, err.Error())
	}

	data := map[string]interface{}{}
	if slots != nil {
		slotvars := map[string]interface{}{}
		for k, v := range slots {
			slotvars[k] = v
		}
		data["$slot"] = slotvars // Will be deprecated use $slots instead
		data["$slots"] = slotvars
	}

	if attrs != nil {
		data["$prop"] = attrs // Will be deprecated use $props instead
		data["$props"] = attrs
		page.Attrs = attrs
	}

	// Add Style & Script & Warning
	doc, err := NewDocumentStringWithWrapper(html)
	if err != nil {
		warnings = append(warnings, err.Error())
	}

	// Append the nested html
	err = page.parse(ctx, doc, option, warnings)
	if err != nil {
		warnings = append(warnings, err.Error())
	}

	// Add Style
	style, err := page.BuildStyle(ctx, option)
	if err != nil {
		warnings = append(warnings, err.Error())
	}

	code, _, err := page.BuildScript(ctx, option, option.Namespace)
	if err != nil {
		warnings = append(warnings, err.Error())
	}

	body := doc.Selection.Find("body")
	if body.Length() > 1 {
		body.SetHtml("<div>" + html + "</div>")
	}

	body.Children().First().SetAttr("s:cn", option.ComponentName)
	body.Children().First().SetAttr("s:ns", option.Namespace)
	body.Children().First().SetAttr("s:ready", option.Namespace+"()")
	html, err = body.Html()
	if err != nil {
		return "", "", "", warnings, err
	}

	// Replace the slots
	html, _ = Data(data).ReplaceUse(slotRe, html)
	return html, style, code, warnings, nil
}

func (page *Page) parse(ctx *BuildContext, doc *goquery.Document, option *BuildOption, warnings []string) error {

	// Find the import pages
	pages := doc.Find("*").FilterFunction(func(i int, sel *goquery.Selection) bool {

		// Get the translation
		if translations := getNodeTranslation(sel, i, option.Namespace); len(translations) > 0 {
			page.Translations = append(page.Translations, translations...)
		}

		tagName := sel.Get(0).Data
		if tagName == "page" {
			return true
		}

		if tagName == "slot" {
			return false
		}

		name, has := sel.Attr("is")
		_, jit := sel.Attr("s:jit") // Just in time rendering
		if has && jit {
			ctx.addJitComponent(name)
		}
		return has && !jit
	})

	sui := SUIs[page.SuiID]
	if sui == nil {
		return fmt.Errorf("SUI %s not found", page.SuiID)
	}

	tmpl, err := sui.GetTemplate(page.TemplateID)
	if err != nil {
		return err
	}

	for _, node := range pages.Nodes {
		sel := goquery.NewDocumentFromNode(node)
		name, has := sel.Attr("is")
		if !has {
			msg := fmt.Sprintf("Page %s/%s/%s: page tag must have an is attribute", page.SuiID, page.TemplateID, page.Route)
			sel.ReplaceWith(fmt.Sprintf("<!-- %s -->", msg))
			log.Warn(msg)
			continue
		}

		sel.SetAttr("parsed", "true")
		ipage, err := tmpl.Page(name)
		if err != nil {
			sel.ReplaceWith(fmt.Sprintf("<!-- %s -->", err.Error()))
			log.Warn("Page %s/%s/%s: %s", page.SuiID, page.TemplateID, page.Route, err.Error())
			continue
		}

		err = ipage.Load()
		if err != nil {
			sel.ReplaceWith(fmt.Sprintf("<!-- %s -->", err.Error()))
			log.Warn("Page %s/%s/%s: %s", page.SuiID, page.TemplateID, page.Route, err.Error())
			continue
		}

		// Set the parent
		slots := map[string]interface{}{}
		for _, slot := range sel.Find("slot").Nodes {
			slotSel := goquery.NewDocumentFromNode(slot)
			slotName, has := slotSel.Attr("is")
			if !has {
				continue
			}
			slotHTML, err := slotSel.Html()
			if err != nil {
				continue
			}
			slots[slotName] = strings.TrimSpace(slotHTML)
		}

		// Set Attrs
		attrs := map[string]string{}
		if sel.Length() > 0 {
			for _, attr := range sel.Nodes[0].Attr {
				if attr.Key == "is" || attr.Key == "parsed" {
					continue
				}
				val := attr.Val
				if page.Attrs != nil {
					parentProps := Data{
						"$prop":  page.Attrs, // Will be deprecated use $props instead
						"$props": page.Attrs,
					}
					val, _ = parentProps.ReplaceUse(slotRe, val)
				}
				attrs[attr.Key] = val
			}
		}

		p := ipage.Get()
		namespace := Namespace(name, ctx.sequence+1)
		componentName := ComponentName(name)
		html, _, _, warns, err := p.BuildForImport(ctx, &BuildOption{
			SSR:             option.SSR,
			AssetRoot:       option.AssetRoot,
			IgnoreAssetRoot: option.IgnoreAssetRoot,
			KeepPageTag:     option.KeepPageTag,
			IgnoreDocument:  true,
			Namespace:       namespace,
			ComponentName:   componentName,
			ScriptMinify:    true,
			StyleMinify:     true,
		}, slots, attrs)

		// append translations
		page.Translations = append(page.Translations, p.Translations...)

		if err != nil {
			sel.ReplaceWith(fmt.Sprintf("<!-- %s -->", err.Error()))
			log.Warn("Page %s/%s/%s: %s", page.SuiID, page.TemplateID, page.Route, err.Error())
			continue
		}

		if warns != nil {
			warnings = append(warnings, warns...)
		}

		sel.SetAttr("s:ns", namespace)
		sel.SetAttr("s:ready", namespace+"()")
		if option.KeepPageTag {
			sel.SetHtml(fmt.Sprintf("\n%s\n", addTabToEachLine(html)))

			// Set Slot HTML
			slotsAttr, err := jsoniter.MarshalToString(slots)
			if err != nil {
				warns = append(warns, err.Error())
				continue
			}

			sel.SetAttr("s:slots", slotsAttr)

			// Set Attrs
			for k, v := range attrs {
				sel.SetAttr(k, v)
			}
			continue
		}
		sel.ReplaceWithHtml(fmt.Sprintf("\n%s\n", addTabToEachLine(html)))
		ctx.sequence++
	}
	return nil
}

// BuildHTML build the html
func (page *Page) BuildHTML(option *BuildOption) (string, error) {

	html := string(page.Codes.HTML.Code)

	if option.WithWrapper {
		html = fmt.Sprintf("<body>%s</body>", html)
	}

	if !option.IgnoreDocument {
		html = string(page.Document)
		if page.Codes.HTML.Code != "" {
			html = strings.Replace(html, "{{ __page }}", page.Codes.HTML.Code, 1)
		}
	}

	if !option.IgnoreAssetRoot {
		html = strings.ReplaceAll(html, "@assets", option.AssetRoot)
	}

	res, err := page.CompileHTML([]byte(html), false)
	if err != nil {
		return "", err
	}

	return string(res), nil
}

// BuildStyle build the style
func (page *Page) BuildStyle(ctx *BuildContext, option *BuildOption) (string, error) {
	if page.Codes.CSS.Code == "" {
		return "", nil
	}

	code := page.Codes.CSS.Code

	// Replace the assets
	if !option.IgnoreAssetRoot {
		code = AssetsRe.ReplaceAllStringFunc(code, func(match string) string {
			return strings.ReplaceAll(match, "@assets", option.AssetRoot)
		})
	}

	if option.ComponentName != "" {
		code = cssRe.ReplaceAllStringFunc(code, func(css string) string {
			return fmt.Sprintf("[s\\:cn=%s] %s", option.ComponentName, css)
		})
		res, err := page.CompileCSS([]byte(code), option.StyleMinify)
		if err != nil {
			return "", err
		}

		ctx.styles = append(ctx.styles, string(res))
		return fmt.Sprintf("<style type=\"text/css\">\n%s\n</style>\n", res), nil
	}

	res, err := page.CompileCSS([]byte(code), option.StyleMinify)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("<style type=\"text/css\">\n%s\n</style>\n", res), nil
}

// BuildScript build the script
func (page *Page) BuildScript(ctx *BuildContext, option *BuildOption, namespace string) (string, []string, error) {

	if page.Codes.JS.Code == "" && page.Codes.TS.Code == "" {
		return "", nil, nil
	}

	instanceCode := fmt.Sprintf("function %s(){ %s(...arguments);}", option.Namespace, option.ComponentName)

	// if the script is a component and not the first import
	if option.ComponentName != "" && ctx.components[option.ComponentName] {
		ctx.scripts = append(ctx.scripts, instanceCode)
		return fmt.Sprintf("<script type=\"text/javascript\">\n%s\n</script>\n", instanceCode), []string{}, nil
	}

	// TypeScript
	if page.Codes.TS.Code != "" {
		code, scripts, err := page.CompileTS([]byte(page.Codes.TS.Code), option.ScriptMinify)
		if err != nil {
			return "", nil, err
		}

		// Get the translation
		if translations := getScriptTranslation(string(code), namespace); len(translations) > 0 {
			page.Translations = append(page.Translations, translations...)
		}

		// Replace the assets
		if !option.IgnoreAssetRoot {
			code = AssetsRe.ReplaceAllFunc(code, func(match []byte) []byte {
				return []byte(strings.ReplaceAll(string(match), "@assets", option.AssetRoot))
			})

			if scripts != nil {
				for i, script := range scripts {
					scripts[i] = filepath.Join(option.AssetRoot, script)
				}
			}
		}

		if option.Namespace == "" {
			return fmt.Sprintf("<script type=\"text/javascript\">\n%s\n</script>\n", code), scripts, nil
		}

		componentCode := fmt.Sprintf("function %s(){\n%s\n}\n%s\n", option.ComponentName, addTabToEachLine(string(code)), instanceCode)
		ctx.scripts = append(ctx.scripts, componentCode)
		return fmt.Sprintf("<script type=\"text/javascript\">%s</script>\n", componentCode), scripts, nil
	}

	// JavaScript
	code, scripts, err := page.CompileJS([]byte(page.Codes.JS.Code), option.ScriptMinify)
	if err != nil {
		return "", nil, err
	}

	// Get the translation
	if translations := getScriptTranslation(string(code), namespace); len(translations) > 0 {
		page.Translations = append(page.Translations, translations...)
	}

	// Replace the assets
	if !option.IgnoreAssetRoot {
		code = AssetsRe.ReplaceAllFunc(code, func(match []byte) []byte {
			return []byte(strings.ReplaceAll(string(match), "@assets", option.AssetRoot))
		})
		if scripts != nil {
			for i, script := range scripts {
				scripts[i] = filepath.Join(option.AssetRoot, script)
			}
		}
	}

	if option.Namespace == "" {
		return fmt.Sprintf("<script type=\"text/javascript\">\n%s\n</script>\n", code), scripts, nil
	}

	componentCode := fmt.Sprintf("function %s(){\n%s\n}\n%s\n", option.ComponentName, addTabToEachLine(string(code)), instanceCode)
	ctx.scripts = append(ctx.scripts, componentCode)
	return fmt.Sprintf("<script type=\"text/javascript\">%s</script>\n", componentCode), scripts, nil
}

func addTabToEachLine(input string, prefix ...string) string {
	var lines []string

	space := "  "
	if len(prefix) > 0 {
		space = prefix[0]
	}

	scanner := bufio.NewScanner(strings.NewReader(input))
	for scanner.Scan() {
		line := scanner.Text()
		lineWithTab := space + line
		lines = append(lines, lineWithTab)
	}

	return strings.Join(lines, "\n")
}

func getScriptTranslation(code string, namespace string) []Translation {
	translations := []Translation{}
	matches := langFuncRe.FindAllStringSubmatch(code, -1)
	for i, match := range matches {
		translations = append(translations, Translation{
			Key:     fmt.Sprintf("%s_script_%d", namespace, i),
			Message: match[1],
			Type:    "script",
		})
	}
	return translations
}

func getNodeTranslation(sel *goquery.Selection, index int, namespace string) []Translation {

	translations := []Translation{}
	nodeType := sel.Get(0).Type
	switch nodeType {
	case html.ElementNode:

		// Get the translation
		if typ, has := sel.Attr("s:trans"); has {
			typ = strings.TrimSpace(typ)
			if typ == "" {
				typ = "html"
			}

			key := fmt.Sprintf("%s_index_%d", namespace, index)
			translations = append(translations, Translation{
				Key:     key,
				Message: strings.TrimSpace(sel.Text()),
				Type:    typ,
			})
			sel.SetAttr("s:trans-node", key)
			sel.RemoveAttr("s:trans")
		}

		// Attributes
		keys := map[string][]string{}
		has := false
		for i, attr := range sel.Get(0).Attr {

			keys[attr.Key] = []string{}

			// value="::attr"
			if strings.HasPrefix(attr.Val, "::") {
				key := fmt.Sprintf("%s_index_attr_%d_%d", namespace, index, i)
				translations = append(translations, Translation{
					Key:     fmt.Sprintf("%s_index_attr_%d_%d", namespace, index, i),
					Message: attr.Val[2:],
					Name:    attr.Key,
					Type:    "attr",
				})
				keys[attr.Key] = append(keys[attr.Key], key)
				has = true
			}

			// value="{{ 'key': '::value' }}"
			matches := langAttrRe.FindAllStringSubmatch(attr.Val, -1)
			if len(matches) > 0 {
				for j, match := range matches {
					key := fmt.Sprintf("%s_index_attr_%d_%d_%d", namespace, index, i, j)
					translations = append(translations, Translation{
						Key:     fmt.Sprintf("%s_index_attr_%d_%d_%d", namespace, index, i, j),
						Message: match[1],
						Name:    attr.Key,
						Type:    "attr",
					})
					keys[attr.Key] = append(keys[attr.Key], key)
					has = true
				}
			}
		}

		if has {
			raw, err := jsoniter.Marshal(keys)
			if err != nil {
				fmt.Println(color.RedString(err.Error()))
				break
			}
			sel.SetAttr("s:trans-attrs", string(raw))
			sel.RemoveAttr("s:trans")
		}

	case html.TextNode:
		if strings.HasPrefix(sel.Text(), "::") {
			key := fmt.Sprintf("%s_index_%d", namespace, index)
			translations = append(translations, Translation{
				Key:     fmt.Sprintf("%s_index_%d", namespace, index),
				Message: strings.TrimSpace(sel.Text()[2:]),
				Type:    "text",
			})
			sel.SetAttr("s:trans-node", key)
			sel.RemoveAttr("s:trans")
		}
	}

	return translations

}
