package core

import (
	"bufio"
	"fmt"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html"
)

var slotRe = regexp.MustCompile(`\[\{([^\}]+)\}\]`)
var cssRe = regexp.MustCompile(`([\.a-z0-9A-Z-:# ]+)\{`)
var transStmtReSingle = regexp.MustCompile(`'::([^:']+)'`)
var transStmtReDouble = regexp.MustCompile(`"::([^:"]+)"`)
var transFuncRe = regexp.MustCompile(`__m\s*\(\s*["'](.*?)["']\s*\)`)

// Build build the page
func (page *Page) Build(ctx *BuildContext, option *BuildOption) (*goquery.Document, []string, error) {
	// Create the context if not exists
	if ctx == nil {
		ctx = NewBuildContext(nil)
	}

	// Push the current page onto the stack and increment the visit counter
	ctx.stack = append(ctx.stack, page.Route)
	ctx.visited[page.Route]++
	defer func() {
		ctx.stack = ctx.stack[:len(ctx.stack)-1] // Pop the stack
		ctx.visited[page.Route]--
	}()

	// Check for recursive calls
	if ctx.visited[page.Route] > 1 {
		return nil, ctx.warnings, fmt.Errorf("recursive build detected for page %s", page.Route)
	}

	ctx.sequence++

	namespace := Namespace(page.Route, ctx.sequence, option.ScriptMinify)
	page.namespace = namespace

	source, err := page.BuildHTML(option)
	if err != nil {
		ctx.warnings = append(ctx.warnings, err.Error())
	}

	doc, err := NewDocumentString(source)
	if err != nil {
		return nil, ctx.warnings, err
	}
	doc.Find("body").SetAttr("s:ns", namespace)

	warnings, err := page.buildComponents(doc, ctx, option)
	if err != nil {
		return nil, ctx.warnings, err
	}
	if warnings != nil && len(warnings) > 0 {
		ctx.warnings = append(ctx.warnings, warnings...)
	}

	// Scripts
	scripts, err := page.BuildScripts(ctx, option, "__page", namespace)
	if err != nil {
		return nil, ctx.warnings, err
	}

	// Styles
	styles, err := page.BuildStyles(ctx, option, "__page", namespace)
	if err != nil {
		return nil, ctx.warnings, err
	}

	// Add the translation marks
	sequence := 1
	err = page.TranslateMarks(ctx, doc, &sequence)
	if err != nil {
		return nil, warnings, err
	}

	// Translate the scripts
	if (scripts != nil) && len(scripts) > 0 {
		for i, script := range scripts {
			if script.Source == "" {
				continue
			}
			trans, keys, err := page.translateScript(script.Source, &sequence)
			if err != nil {
				return nil, ctx.warnings, err
			}
			if len(keys) > 0 {
				ctx.translations = append(ctx.translations, trans...)
				scripts[i].Attrs = append(script.Attrs, html.Attribute{Key: "s:trans-script", Val: strings.Join(keys, ",")})
			}
		}
	}

	// Append the scripts and styles
	ctx.scripts = append(ctx.scripts, scripts...)
	ctx.styles = append(ctx.styles, styles...)

	return doc, ctx.warnings, err
}

// BuildAsComponent build the page as component
func (page *Page) BuildAsComponent(sel *goquery.Selection, ctx *BuildContext, option *BuildOption) (string, error) {

	if page.parent == nil {
		return "", fmt.Errorf("The parent page is not set")
	}

	if ctx == nil {
		ctx = NewBuildContext(nil)
	}

	// Push the current page onto the stack and increment the visit counter
	ctx.stack = append(ctx.stack, page.Route)
	ctx.visited[page.Route]++
	defer func() {
		ctx.stack = ctx.stack[:len(ctx.stack)-1] // Pop the stack
		ctx.visited[page.Route]--
	}()

	// Check for recursive calls
	if ctx.visited[page.Route] > 1 {
		return "", fmt.Errorf("recursive build detected for page %s", page.Route)
	}

	name, exists := sel.Attr("is")
	if !exists {
		return "", fmt.Errorf("The component tag must have an is attribute")
	}

	namespace := Namespace(name, ctx.sequence, option.ScriptMinify)
	component := ComponentName(name, option.ScriptMinify)
	page.namespace = namespace
	attrs := []html.Attribute{
		{Key: "s:ns", Val: namespace},
		{Key: "s:cn", Val: component},
		{Key: "s:ready", Val: component + "()"},
		{Key: "s:parent", Val: page.parent.namespace},
	}

	ctx.sequence++
	var opt = *option
	opt.IgnoreDocument = true
	source, err := page.BuildHTML(&opt)
	if err != nil {
		return "", err
	}

	doc, err := NewDocumentStringWithWrapper(source)
	if err != nil {
		return "", err
	}

	body := doc.Selection.Find("body")

	if body.Children().Length() == 0 {
		return "", fmt.Errorf("page %s as component should have one root element", page.Route)
	}

	if body.Children().Length() > 1 {
		return "", fmt.Errorf("page %s as component should have only one root element", page.Route)
	}

	// Scripts
	scripts, err := page.BuildScripts(ctx, &opt, component, namespace)
	if err != nil {
		return "", err
	}

	styles, err := page.BuildStyles(ctx, &opt, component, namespace)
	if err != nil {
		return "", err
	}

	// Pass the component props
	first := body.Children().First()

	page.copyProps(ctx, sel, first, attrs...)
	page.copySlots(sel, first)
	page.copyChildren(sel, first)
	page.buildComponents(doc, ctx, &opt)
	data := Data{"$props": page.Attrs}
	data.ReplaceSelectionUse(slotRe, first)

	// Add the translation marks
	sequence := 1
	err = page.TranslateMarks(ctx, doc, &sequence)
	if err != nil {
		return "", err
	}

	// Translate the scripts
	if (scripts != nil) && len(scripts) > 0 {
		for i, script := range scripts {
			if script.Source == "" {
				continue
			}
			trans, keys, err := page.translateScript(script.Source, &sequence)
			if err != nil {
				return "", err
			}
			if len(keys) > 0 {
				ctx.translations = append(ctx.translations, trans...)
				scripts[i].Attrs = append(script.Attrs, html.Attribute{Key: "s:trans-script", Val: strings.Join(keys, ",")})
			}
		}
	}

	// Append the scripts
	ctx.scripts = append(ctx.scripts, scripts...)
	ctx.styles = append(ctx.styles, styles...)

	source, err = body.Html()
	if err != nil {
		return "", err
	}
	sel.ReplaceWithHtml(source)
	ctx.components[page.Route] = true
	return source, nil
}

func (page *Page) copySlots(from *goquery.Selection, to *goquery.Selection) error {
	slots := from.Find("slot")
	if slots.Length() == 0 {
		return nil
	}

	for i := 0; i < slots.Length(); i++ {
		slot := slots.Eq(i)
		name, has := slot.Attr("name")
		if !has {
			continue
		}

		// Get the slot
		slotSel := to.Find(name)
		if slotSel.Length() == 0 {
			continue
		}

		slotSel.ReplaceWithSelection(slot.Contents())
	}

	return nil
}

func (page *Page) copyChildren(from *goquery.Selection, to *goquery.Selection) error {
	children := from.Contents()
	if children.Length() == 0 {
		return nil
	}
	children.Find("slot").Remove()
	to.Find("children").ReplaceWithSelection(children)
	return nil
}

func (page *Page) copyProps(ctx *BuildContext, from *goquery.Selection, to *goquery.Selection, extra ...html.Attribute) error {
	attrs := from.Get(0).Attr
	prefix := "s:prop"
	if page.Attrs == nil {
		page.Attrs = map[string]string{}
	}
	for _, attr := range attrs {
		if strings.HasPrefix(attr.Key, "s:") || attr.Key == "is" || attr.Key == "parsed" {
			continue
		}

		if strings.HasPrefix(attr.Key, "...$props") {
			data := Data{"$props": page.parent.Attrs}
			val, err := data.Exec(fmt.Sprintf("{{ %s }}", attr.Key[3:]))
			if err != nil {
				ctx.warnings = append(ctx.warnings, err.Error())
				setError(to, err)
				continue
			}
			switch value := val.(type) {
			case map[string]string:
				for key, value := range value {
					page.Attrs[key] = value
					key = fmt.Sprintf("%s:%s", prefix, key)
					to.SetAttr(key, value)
				}
			}
			continue
		}

		val := attr.Val
		if strings.HasPrefix(attr.Key, `...\$props`) {
			val = fmt.Sprintf("{{ $props.%s }}", attr.Key[9:])
		}

		if strings.HasPrefix(attr.Key, "...") {
			val = attr.Key[3:]
		}
		page.Attrs[attr.Key] = val
		key := fmt.Sprintf("%s:%s", prefix, attr.Key)
		to.SetAttr(key, val)
	}

	if len(extra) > 0 {
		for _, attr := range extra {
			to.SetAttr(attr.Key, attr.Val)
		}
	}

	return nil
}

func (page *Page) buildComponents(doc *goquery.Document, ctx *BuildContext, option *BuildOption) ([]string, error) {
	warnings := []string{}
	sui := SUIs[page.SuiID]
	if sui == nil {
		return warnings, fmt.Errorf("SUI %s not found", page.SuiID)
	}

	public := sui.GetPublic()
	tmpl, err := sui.GetTemplate(page.TemplateID)
	if err != nil {
		return warnings, err
	}

	doc.Find("*").Each(func(i int, sel *goquery.Selection) {
		// Get the translation

		name, has := sel.Attr("is")
		if !has {
			return
		}

		// Slot tag
		tagName := sel.Get(0).Data
		if tagName == "slot" {
			return
		}

		// Check if Just-In-Time Component ( "is" has variable )
		if ctx.isJitComponent(name) {
			sel.SetAttr("s:jit", "true")
			sel.SetAttr("s:parent", page.namespace)
			sel.SetAttr("s:root", public.Root)
			ctx.addJitComponent(name)
			return
		}

		sel.SetAttr("parsed", "true")
		ipage, err := tmpl.Page(name)
		if err != nil {
			message := err.Error()
			warnings = append(warnings, message)
			setError(sel, err)
			return
		}

		err = ipage.Load()
		if err != nil {
			message := err.Error()
			warnings = append(warnings, message)
			setError(sel, err)
			return
		}

		component := ipage.Get()
		component.parent = page
		_, err = component.BuildAsComponent(sel, ctx, option)
		if err != nil {
			message := err.Error()
			warnings = append(warnings, message)
			setError(sel, err)
			return
		}
	})

	return warnings, nil
}

// BuildStyles build the styles for the page
func (page *Page) BuildStyles(ctx *BuildContext, option *BuildOption, component string, namespace string) ([]StyleNode, error) {
	styles := []StyleNode{}
	if page.Codes.CSS.Code == "" {
		return styles, nil
	}

	if _, has := ctx.styleUnique[component]; has {
		return styles, nil
	}
	ctx.styleUnique[component] = true

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
			return styles, err
		}
		styles = append(styles, StyleNode{
			Namespace: namespace,
			Component: component,
			Source:    string(res),
			Parent:    "head",
			Attrs: []html.Attribute{
				{Key: "rel", Val: "stylesheet"},
				{Key: "type", Val: "text/css"},
			},
		})
		return styles, nil
	}

	res, err := page.CompileCSS([]byte(code), option.StyleMinify)
	if err != nil {
		return styles, err
	}
	styles = append(styles, StyleNode{
		Namespace: namespace,
		Component: component,
		Parent:    "head",
		Source:    string(res),
		Attrs: []html.Attribute{
			{Key: "rel", Val: "stylesheet"},
			{Key: "type", Val: "text/css"},
		},
	})

	return styles, nil
}

// BuildScripts build the scripts for the page
func (page *Page) BuildScripts(ctx *BuildContext, option *BuildOption, component string, namespace string) ([]ScriptNode, error) {

	ispage := component == "__page"
	if ispage {
		component = ComponentName(page.Route, option.ScriptMinify)
	}

	scripts := []ScriptNode{}
	if page.Codes.JS.Code == "" && page.Codes.TS.Code == "" {
		return scripts, nil
	}
	if _, has := ctx.scriptUnique[component]; has {
		return scripts, nil
	}

	ctx.scriptUnique[component] = true
	var err error = nil
	var imports []string = nil
	var source []byte = nil
	if page.Codes.TS.Code != "" {
		source, imports, err = page.CompileTS([]byte(page.Codes.TS.Code), option.ScriptMinify)
		if err != nil {
			return nil, err
		}

	} else if page.Codes.JS.Code != "" {
		source, imports, err = page.CompileJS([]byte(page.Codes.JS.Code), option.ScriptMinify)
		if err != nil {
			return nil, err
		}

	}

	// Add the script
	if imports != nil {
		for _, src := range imports {
			scripts = append(scripts, ScriptNode{
				Namespace: namespace,
				Component: component,
				Parent:    "head",
				Attrs: []html.Attribute{
					{Key: "src", Val: fmt.Sprintf("%s/%s", option.AssetRoot, src)},
					{Key: "type", Val: "text/javascript"},
				}},
			)
		}
	}

	// Replace the assets
	if !option.IgnoreAssetRoot && source != nil {
		source = AssetsRe.ReplaceAllFunc(source, func(match []byte) []byte {
			return []byte(strings.ReplaceAll(string(match), "@assets", option.AssetRoot))
		})

		code := string(source)
		parent := "body"
		if !ispage {
			parent = "head"
			code = fmt.Sprintf("function %s(){\n%s\n}\n", component, addTabToEachLine(code))
		}

		scripts = append(scripts, ScriptNode{
			Namespace: namespace,
			Component: component,
			Source:    code,
			Parent:    parent,
			Attrs: []html.Attribute{
				{Key: "type", Val: "text/javascript"},
			},
		})
	}

	return scripts, nil
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

func setError(sel *goquery.Selection, err error) {
	html := `<div style="color:red; margin:10px 0px; font-size: 12px; font-family: monospace; padding: 10px; border: 1px solid red; background-color: #f8d7da;">%s</div>`
	sel.SetHtml(fmt.Sprintf(html, err.Error()))
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

// TranslateMarks add the translation marks to the document
func (page *Page) TranslateMarks(ctx *BuildContext, doc *goquery.Document, sequence *int) error {

	if doc.Length() == 0 {
		return nil
	}

	if ctx == nil {
		ctx = NewBuildContext(nil)
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

		sel := goquery.NewDocumentFromNode(node)
		// Script
		if node.Data == "script" {
			if _, has := sel.Attr("s:trans-script"); has {
				break
			}
			code := goquery.NewDocumentFromNode(node).Text()
			trans, keys, err := page.translateScript(code, sequence)
			if err != nil {
				return nil, err
			}
			if len(keys) > 0 {
				raw := strings.Join(keys, ",")
				sel.SetAttr("s:trans-script", raw)
				translations = append(translations, trans...)
			}
			break
		}

		for _, attr := range node.Attr {

			if _, has := sel.Attr("s:trans-attr-" + attr.Key); has {
				continue
			}

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
			if _, has := parentSel.Attr("s:trans-node"); has {
				break
			}

			key := TranslationKey(page.Route, *sequence)
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
			parentSel.SetAttr("s:trans-escape", "true")
		}

		if _, has := parentSel.Attr("s:trans-text"); has {
			break
		}
		trans, keys, err := page.translateText(node.Data, sequence, "text")
		if err != nil {
			return nil, err
		}
		if len(keys) > 0 {
			raw := strings.Join(keys, ",")
			parentSel.SetAttr("s:trans-text", raw)
			translations = append(translations, trans...)
		}
		break
	}

	return translations, nil
}

func (page *Page) translateText(text string, sequence *int, transType string) ([]Translation, []string, error) {
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
			key := TranslationKey(page.Route, *sequence)
			keys = append(keys, key)
			translations = append(translations, Translation{
				Key:     key,
				Message: message,
				Type:    transType,
			})
			*sequence = *sequence + 1
		}
	}
	return translations, keys, nil
}

func (page *Page) translateScript(code string, sequence *int) ([]Translation, []string, error) {

	translations := []Translation{}
	keys := []string{}
	if code == "" {
		return translations, keys, nil
	}
	matches := transFuncRe.FindAllStringSubmatch(code, -1)
	for _, match := range matches {
		key := TranslationKey(page.Route, *sequence)
		translations = append(translations, Translation{
			Key:     key,
			Message: match[1],
			Type:    "script",
		})
		*sequence = *sequence + 1
		keys = append(keys, key)
	}
	return translations, keys, nil
}
