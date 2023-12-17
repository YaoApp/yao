package core

import (
	"bufio"
	"fmt"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/kun/log"
)

var slotRe = regexp.MustCompile(`\[\{([^\}]+)\}\]`)
var cssRe = regexp.MustCompile(`([\.a-z0-9A-Z# ]+)\{`)

// Build is the struct for the public
func (page *Page) Build(option *BuildOption) (*goquery.Document, []string, error) {

	warnings := []string{}
	html, err := page.BuildHTML(option)
	if err != nil {
		warnings = append(warnings, err.Error())
	}

	// Add Style & Script & Warning
	doc, err := NewDocument([]byte(html))
	if err != nil {
		warnings = append(warnings, err.Error())
	}

	// Append the nested html
	err = page.parse(doc, option, warnings)
	if err != nil {
		warnings = append(warnings, err.Error())
	}

	// Add Style
	style, err := page.BuildStyle(option)
	if err != nil {
		warnings = append(warnings, err.Error())
	}
	doc.Selection.Find("head").AppendHtml(style)

	// Add Script
	script, err := page.BuildScript(option)
	if err != nil {
		warnings = append(warnings, err.Error())
	}
	doc.Selection.Find("body").AppendHtml(script)
	return doc, warnings, nil
}

// BuildForImport build the page for import
func (page *Page) BuildForImport(option *BuildOption, slots map[string]interface{}, attrs map[string]string) (string, string, string, []string, error) {
	warnings := []string{}
	html, err := page.BuildHTML(option)
	if err != nil {
		warnings = append(warnings, err.Error())
	}

	data := map[string]interface{}{}
	if slots != nil {
		for k, v := range slots {
			data[k] = v
		}
	}

	if attrs != nil {
		data["$prop"] = attrs
		page.Attrs = attrs
	}

	// Add Style & Script & Warning
	doc, err := NewDocument([]byte(html))
	if err != nil {
		warnings = append(warnings, err.Error())
	}

	// Append the nested html
	err = page.parse(doc, option, warnings)
	if err != nil {
		warnings = append(warnings, err.Error())
	}

	// Add Style
	style, err := page.BuildStyle(option)
	if err != nil {
		warnings = append(warnings, err.Error())
	}

	script, err := page.BuildScript(option)
	if err != nil {
		warnings = append(warnings, err.Error())
	}

	body := doc.Selection.Find("body")
	if body.Length() > 1 {
		body.SetHtml("<div>" + html + "</div>")
	}

	body.Children().First().SetAttr("s:ns", option.Namespace)
	body.Children().First().SetAttr("s:ready", option.Namespace+"()")
	html, err = body.Html()
	if err != nil {
		return "", "", "", warnings, err
	}

	// Replace the slots
	html, _ = Data(data).ReplaceUse(slotRe, html)
	return html, style, script, warnings, nil
}

func (page *Page) parse(doc *goquery.Document, option *BuildOption, warnings []string) error {
	pages := doc.Find("page")
	sui := SUIs[page.SuiID]
	if sui == nil {
		return fmt.Errorf("SUI %s not found", page.SuiID)
	}

	tmpl, err := sui.GetTemplate(page.TemplateID)
	if err != nil {
		return err
	}

	for idx, node := range pages.Nodes {
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
			if page.Attrs != nil {
				parentProps := Data{"$prop": page.Attrs}
				for k, v := range page.Attrs {
					if k == "is" {
						continue
					}
					attrs[k], _ = parentProps.ReplaceUse(slotRe, v)
				}
			} else {
				for _, attr := range sel.Nodes[0].Attr {
					if attr.Key == "is" {
						continue
					}
					attrs[attr.Key] = attr.Val
				}
			}
		}

		p := ipage.Get()
		namespace := fmt.Sprintf("__page_%s_%d", strings.ReplaceAll(name, "/", "_"), idx)
		html, style, script, warns, err := p.BuildForImport(&BuildOption{
			SSR:             option.SSR,
			AssetRoot:       option.AssetRoot,
			IgnoreAssetRoot: option.IgnoreAssetRoot,
			KeepPageTag:     option.KeepPageTag,
			IgnoreDocument:  true,
			Namespace:       namespace,
		}, slots, attrs)

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
			sel.SetHtml(fmt.Sprintf("\n%s\n%s\n%s\n", style, addTabToEachLine(html), script))

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
		sel.ReplaceWithHtml(fmt.Sprintf("\n%s\n%s\n%s\n", style, html, script))

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
func (page *Page) BuildStyle(option *BuildOption) (string, error) {
	if page.Codes.CSS.Code == "" {
		return "", nil
	}

	code := page.Codes.CSS.Code
	if !option.IgnoreAssetRoot {
		code = strings.ReplaceAll(page.Codes.CSS.Code, "@assets", option.AssetRoot)
	}

	if option.Namespace != "" {
		code = cssRe.ReplaceAllStringFunc(code, func(css string) string {
			return fmt.Sprintf("[s\\:ns=%s] %s", option.Namespace, css)
		})
	}

	res, err := page.CompileCSS([]byte(code), false)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("<style>\n%s\n</style>\n", res), nil
}

// BuildScript build the script
func (page *Page) BuildScript(option *BuildOption) (string, error) {

	if page.Codes.JS.Code == "" && page.Codes.TS.Code == "" {
		return "", nil
	}

	if page.Codes.TS.Code != "" {
		res, err := page.CompileTS([]byte(page.Codes.TS.Code), false)
		if err != nil {
			return "", err
		}

		if option.Namespace == "" {
			return fmt.Sprintf("<script>\n%s\n</script>\n", res), nil
		}

		return fmt.Sprintf("<script>\nfunction %s(){\n%s\n}\n</script>\n", option.Namespace, addTabToEachLine(string(res))), nil
	}

	code := page.Codes.JS.Code
	if !option.IgnoreAssetRoot {
		code = strings.ReplaceAll(page.Codes.JS.Code, "@assets", option.AssetRoot)
	}

	res, err := page.CompileJS([]byte(code), false)
	if err != nil {
		return "", err
	}

	if option.Namespace == "" {
		return fmt.Sprintf("<script>\n%s\n</script>\n", res), nil
	}
	return fmt.Sprintf("<script>\nfunction %s(){\n%s\n}\n</script>\n", option.Namespace, addTabToEachLine(string(res))), nil
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
