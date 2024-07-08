package core

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

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
	comp, props, slots, err := parser.getComponent(sel)
	if err != nil {
		parser.errors = append(parser.errors, err)
		setError(sel, err)
		return
	}

	html, _, err := parser.RenderComponent(comp, props, slots)
	if err != nil {
		parser.errors = append(parser.errors, err)
		setError(sel, err)
		return
	}

	sel.SetHtml(html)
}

// RenderComponent render the component
func (parser *TemplateParser) RenderComponent(comp *JitComponent, props map[string]interface{}, slots *goquery.Selection) (string, string, error) {
	html := comp.html
	randvar := fmt.Sprintf("__%s_$props", time.Now().Format("20060102150405"))
	html = slotRe.ReplaceAllStringFunc(html, func(exp string) string {
		exp = strings.ReplaceAll(exp, "[{", "{{")
		exp = strings.ReplaceAll(exp, "}]", "}}")
		exp = strings.ReplaceAll(exp, "$props", randvar)
		return exp
	})

	data := Data{}
	data[randvar] = props
	if parser.data != nil {
		for key, val := range parser.data {
			data[key] = val
		}
	}

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
		slotSel := root.Find(fmt.Sprintf("slot[name='%s']", name))
		if slotSel.Length() == 0 {
			return
		}

		// Replace the slot
		slotSel.ReplaceWithNodes(s.Contents().Nodes...)
	})

	option := *parser.option
	compParser := NewTemplateParser(data, &option)

	// Parse the node
	ns := Namespace(comp.route, compParser.sequence, comp.buildOption.ScriptMinify)
	cn := ComponentName(comp.route, comp.buildOption.ScriptMinify)
	compParser.parseNode(root.Get(0))
	for sel, nodes := range compParser.replace {
		sel.ReplaceWithNodes(nodes...)
		delete(parser.replace, sel)
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

func (parser *TemplateParser) addScripts(sel *goquery.Selection, scripts []ScriptNode) {
	if scripts == nil {
		return
	}
	for _, script := range scripts {
		query := fmt.Sprintf(`script[s\:cn="%s"]`, script.Component)
		if sel.Find(query).Length() > 0 {
			continue
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

func (parser *TemplateParser) getComponent(sel *goquery.Selection) (*JitComponent, map[string]interface{}, *goquery.Selection, error) {

	slots := sel.Find("slot")

	props, err := parser.componentProps(sel)
	if err != nil {
		return nil, nil, slots, err
	}

	file, route, err := parser.componentFile(sel, props)
	if err != nil {
		return nil, props, slots, err
	}

	comp, err := getComponent(route, file, parser.disableCache())
	if err != nil {
		return nil, props, slots, err
	}

	return comp, props, slots, nil
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
				for k, v := range values.(map[string]interface{}) {
					result[k] = v
				}
			}
			continue
		}

		if strings.HasPrefix(val, "{{") && strings.HasSuffix(val, "}}") {
			value, err := parser.data.Exec(val)
			if err != nil {
				return map[string]interface{}{}, err
			}
			result[key] = value
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
