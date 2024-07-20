package core

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/application"
)

// JitComponent component
type JitComponent struct {
	file        string
	route       string
	html        string
	scripts     []ScriptNode
	styles      []StyleNode
	buildOption *BuildOption
}

const (
	saveComponent uint8 = iota
	removeComponent
)

type componentData struct {
	file string
	comp *JitComponent
	cmd  uint8
}

// Components loaded JIT components
var Components = map[string]*JitComponent{}
var chComp = make(chan *componentData, 1)
var reScripts = regexp.MustCompile(`<script[^>]*name="scripts"[^>]*>(.*?)</script>`)
var reStyles = regexp.MustCompile(`<script[^>]*name="styles"[^>]*>(.*?)</script>`)
var reOption = regexp.MustCompile(`<script[^>]*name="option"[^>]*>(.*?)</script>`)

func init() {
	go componentWriter()
}

// parseComponent parse the component
func (parser *TemplateParser) parseComponent(sel *goquery.Selection) {
	parser.parsed(sel)
	comp, props, slots, children, err := parser.getComponent(sel)
	if err != nil {
		parser.errors = append(parser.errors, err)
		setError(sel, err)
		return
	}

	html, _, err := parser.RenderComponent(comp, props, slots, children)
	if err != nil {
		parser.errors = append(parser.errors, err)
		setError(sel, err)
		return
	}

	// fmt.Println(sel.Nodes[0].Attr)
	sel.SetHtml(html)
}

// RenderComponent render the component
func (parser *TemplateParser) RenderComponent(comp *JitComponent, props map[string]interface{}, slots *goquery.Selection, children *goquery.Selection) (string, string, error) {
	html := comp.html

	html = replaceRandVar(html, Data(props))
	option := *parser.option
	option.Route = comp.route
	compParser := NewTemplateParser(parser.data, &option)
	compParser.sequence = parser.sequence + 1
	locale := compParser.Locale()

	// Parse the node
	ns := Namespace(comp.route, compParser.sequence, comp.buildOption.ScriptMinify)
	cn := ComponentName(comp.route, comp.buildOption.ScriptMinify)

	sel, err := NewDocumentString(`<body>` + html + `</body>`)
	if err != nil {
		return "", "", err
	}

	root := sel.Find("body")

	// Replace the slots
	slots.Each(func(i int, s *goquery.Selection) {
		name := s.AttrOr("name", "")
		if name == "" {
			return
		}

		// Find the slot
		slotSel := root.Find(name)
		if slotSel.Length() == 0 {
			return
		}

		// Replace the slot
		slotSel.ReplaceWithSelection(s.Contents())
	})

	// Replace the children
	children.Find("slot").Remove()
	root.Find("children").ReplaceWithSelection(children)

	// Replace the props
	if locale != nil {
		locale.replaceVars(Data(props))
	}

	compParser.locale = locale
	compParser.BindEvent(root, ns, cn)
	compParser.parseNode(root.Get(0))
	for sel, nodes := range compParser.replace {
		sel.ReplaceWithNodes(nodes...)
		delete(parser.replace, sel)
	}

	if compParser.scripts != nil {
		for _, script := range compParser.scripts {
			script.Namespace = ns
			parser.scripts = append(parser.scripts, script)
		}
	}

	if comp.scripts != nil {
		for _, script := range comp.scripts {
			script.Namespace = ns
			parser.scripts = append(parser.scripts, script)
		}
	}

	if comp.styles != nil {
		for _, style := range comp.styles {
			style.Namespace = ns
			parser.styles = append(parser.styles, style)
		}
	}

	// Update sequence
	parser.sequence = compParser.sequence + parser.sequence
	first := root.Children().First()
	first.SetAttr("s:ns", ns)
	first.SetAttr("s:cn", cn)
	first.SetAttr("s:ready", cn+"()")

	html, err = root.Html()
	if err != nil {
		return "", "", err
	}
	return html, ns, nil
}

func (parser *TemplateParser) filterScripts(parent string, scripts []ScriptNode) []ScriptNode {
	if scripts == nil {
		return []ScriptNode{}
	}
	filtered := []ScriptNode{}
	for _, script := range scripts {
		if script.Parent != parent {
			continue
		}
		filtered = append(filtered, script)
	}
	return filtered
}

func (parser *TemplateParser) addScripts(sel *goquery.Selection, scripts []ScriptNode) {
	for _, script := range scripts {
		if script.Component != "" {
			query := fmt.Sprintf(`script[s\:cn="%s"]`, script.Component)
			if sel.Find(query).Length() > 0 {
				continue
			}
		}
		sel.AppendHtml(script.ComponentHTML(script.Namespace))
	}
}

func (parser *TemplateParser) addStyles(sel *goquery.Selection, styles []StyleNode) {
	if styles == nil {
		return
	}
	for _, style := range styles {
		query := fmt.Sprintf(`style[s\:cn="%s"]`, style.Component)
		if sel.Find(query).Length() > 0 {
			continue
		}
		sel.Append(style.HTML())
	}
}

func (parser *TemplateParser) getComponent(sel *goquery.Selection) (*JitComponent, map[string]interface{}, *goquery.Selection, *goquery.Selection, error) {

	slots := sel.Find("slot")
	children := sel.Contents()

	props, err := parser.componentProps(sel)
	if err != nil {
		return nil, nil, slots, children, err
	}

	file, route, err := parser.componentFile(sel, props)
	if err != nil {
		return nil, props, slots, children, err
	}

	comp, err := getComponent(route, file, parser.disableCache())
	if err != nil {
		return nil, props, slots, children, err
	}

	return comp, props, slots, children, nil
}

// isComponent check if the selection is a component
func (parser *TemplateParser) isComponent(sel *goquery.Selection) bool {
	_, exist := sel.Attr("s:jit")
	is := sel.AttrOr("is", "")
	return exist && is != ""
}

func (parser *TemplateParser) componentProps(sel *goquery.Selection) (map[string]interface{}, error) {

	parentProps := map[string]string{}
	props := map[string]string{}
	parent := sel.AttrOr("s:parent", "")
	parentSel := sel.Parents().Find(fmt.Sprintf(`[s\:ns="%s"]`, parent))

	if parentSel != nil && parentSel.Length() > 0 {
		for _, attr := range parentSel.Nodes[0].Attr {

			if !strings.HasPrefix(attr.Key, "s:prop") {
				continue
			}
			key := strings.TrimPrefix(attr.Key, "s:prop:")
			parentProps[key] = attr.Val
		}
	}

	for _, attr := range sel.Nodes[0].Attr {

		// s:on , s:data , s:json
		if strings.HasPrefix(attr.Key, "s:on") || strings.HasPrefix(attr.Key, "s:data") || strings.HasPrefix(attr.Key, "s:json") {
			props[attr.Key] = attr.Val
			continue
		}

		if strings.HasPrefix(attr.Key, "s:") || attr.Key == "is" {
			continue
		}

		if strings.HasPrefix(attr.Key, "...$props") {
			data := Data{"$props": parentProps}
			values, err := data.Exec(fmt.Sprintf("{{ %s }}", strings.TrimPrefix(attr.Key, "...")))
			if err != nil {
				return map[string]interface{}{}, err
			}
			if values == nil {
				continue
			}

			if _, ok := values.(map[string]string); ok {
				for key, val := range values.(map[string]string) {
					props[key] = val
				}
			}
			continue
		}

		props[attr.Key] = attr.Val
	}

	return parser.parseComponentProps(props)
}

func (parser *TemplateParser) parseComponentProps(props map[string]string) (map[string]interface{}, error) {
	result := map[string]interface{}{}
	for key, val := range props {
		if strings.HasPrefix(key, "...") {
			values, err := parser.data.Exec(fmt.Sprintf("{{ %s }}", strings.TrimPrefix(key, "...")))
			if err != nil {
				return nil, err
			}

			if values == nil {
				continue
			}

			if _, ok := values.(map[string]interface{}); ok {
				for k := range values.(map[string]interface{}) {
					result[k] = k
				}
			}
			continue
		}

		result[key] = val
	}
	return result, nil
}

func (parser *TemplateParser) componentFile(sel *goquery.Selection, props map[string]interface{}) (string, string, error) {
	route := sel.AttrOr("is", "")
	if route == "" {
		return "", "", fmt.Errorf("Component route is required")
	}

	data := Data{"$props": props}
	route, _ = data.ReplaceUse(slotRe, route)
	route, _ = parser.data.Replace(route)
	root := sel.AttrOr("s:root", "/")
	file := filepath.Join(string(os.PathSeparator), "public", root, route+".jit")
	return file, route, nil
}

func (locale *Locale) replaceVars(data Data) {
	if locale.Keys != nil {
		for key, val := range locale.Keys {
			locale.Keys[key] = replaceRandVar(val, data)
		}
	}

	if locale.Messages != nil {
		for key, val := range locale.Messages {
			locale.Messages[key] = replaceRandVar(val, data)
		}
	}
}

func componentWriter() {
	for {
		select {
		case data := <-chComp:
			switch data.cmd {
			case saveComponent:
				Components[data.file] = data.comp
			case removeCache:
				delete(Components, data.file)
			}
		}
	}
}

func getComponent(route string, file string, disableCache ...bool) (*JitComponent, error) {

	noCache := false
	if len(disableCache) > 0 && disableCache[0] {
		noCache = true
	}

	if comp, ok := Components[file]; ok && !noCache {
		return comp, nil
	}

	comp, err := readComponent(route, file)
	if err != nil {
		return nil, err
	}

	// Save the component to the cache
	chComp <- &componentData{file, comp, saveComponent}
	return comp, nil
}

// readComponent read the JIT component
func readComponent(route string, file string) (*JitComponent, error) {
	content, err := application.App.Read(file)
	if err != nil {
		return nil, err
	}

	// Get the scripts
	html := string(content)
	rawScripts := ""
	rawStyles := ""
	rawOption := ""
	html = reScripts.ReplaceAllStringFunc(html, func(exp string) string {
		rawScripts = reScripts.ReplaceAllString(exp, "$1")
		return ""
	})

	html = reStyles.ReplaceAllStringFunc(html, func(exp string) string {
		rawStyles = reStyles.ReplaceAllString(exp, "$1")
		return ""
	})

	html = reOption.ReplaceAllStringFunc(html, func(exp string) string {
		rawOption = reOption.ReplaceAllString(exp, "$1")
		return ""
	})

	scripts := []ScriptNode{}
	styles := []StyleNode{}
	buildOption := BuildOption{}

	if rawScripts != "" {
		err := jsoniter.UnmarshalFromString(rawScripts, &scripts)
		if err != nil {
			return nil, err
		}
	}

	if rawStyles != "" {
		err := jsoniter.UnmarshalFromString(rawStyles, &styles)
		if err != nil {
			return nil, err
		}
	}

	if rawOption != "" {
		err := jsoniter.UnmarshalFromString(rawOption, &buildOption)
		if err != nil {
			return nil, err
		}
	}

	return &JitComponent{
		file:        file,
		route:       route,
		html:        html,
		scripts:     scripts,
		styles:      styles,
		buildOption: &buildOption,
	}, nil
}

func replaceRandVar(value string, data Data) string {

	value = propNewRe.ReplaceAllStringFunc(value, func(exp string) string {
		exp = strings.TrimPrefix(exp, "{%")
		exp = strings.TrimSuffix(exp, "%}")
		res, _ := data.ExecString(fmt.Sprintf("{{ %s }}", exp))
		return res
	})

	data = Data{"$props": data}
	return slotRe.ReplaceAllStringFunc(value, func(exp string) string {
		res, _ := data.ExecString(exp)
		return res
	})
}
