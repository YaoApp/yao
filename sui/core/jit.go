package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/yaoapp/gou/application"
)

// JitComponent component
type JitComponent struct {
	html    string
	scripts []ScriptNode
	styles  []StyleNode
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

func init() {
	go componentWriter()
}

// parseComponent parse the component
func (parser *TemplateParser) parseComponent(sel *goquery.Selection) {
	comp, props, slots, err := parser.getComponent(sel)
	if err != nil {
		parser.errors = append(parser.errors, err)
		setError(sel, err)
		return
	}

	html, err := parser.RenderComponent(comp, props, slots)
	if err != nil {
		parser.errors = append(parser.errors, err)
		setError(sel, err)
		return
	}
	sel.SetHtml(html)
}

// RenderComponent render the component
func (parser *TemplateParser) RenderComponent(comp *JitComponent, props map[string]interface{}, slots *goquery.Selection) (string, error) {
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
		return "", err
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
	compParser.parseNode(root.Get(0))
	for sel, nodes := range compParser.replace {
		sel.ReplaceWithNodes(nodes...)
		delete(parser.replace, sel)
	}

	// Update sequence
	parser.sequence = compParser.sequence + parser.sequence
	return root.Html()
}

func (parser *TemplateParser) getComponent(sel *goquery.Selection) (*JitComponent, map[string]interface{}, *goquery.Selection, error) {

	slots := sel.Find("slot")

	props, err := parser.componentProps(sel)
	if err != nil {
		return nil, nil, slots, err
	}

	file, err := parser.componentFile(sel, props)
	if err != nil {
		return nil, props, slots, err
	}

	comp, err := getComponent(file, parser.disableCache())
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
	props := map[string]string{}
	parentSel := sel.Parent()
	if parentSel == nil {
		return map[string]interface{}{}, nil
	}

	for _, attr := range parentSel.Nodes[0].Attr {
		if !strings.HasPrefix(attr.Key, "s:prop") {
			continue
		}
		key := strings.TrimPrefix(attr.Key, "s:prop:")
		props[key] = attr.Val
	}

	return parser.parseComponentProps(props)
}

func (parser *TemplateParser) parseComponentProps(props map[string]string) (map[string]interface{}, error) {
	result := map[string]interface{}{}
	for key, val := range props {
		if strings.HasPrefix(key, "...") {
			values, err := parser.data.Exec(fmt.Sprintf("{{ %s }}", val))
			if err != nil {
				return nil, err
			}

			if values == nil {
				continue
			}

			for k, v := range values.(map[string]interface{}) {
				result[k] = v
			}

			continue
		}
		value, err := parser.data.Exec(val)
		if err != nil {
			return nil, err
		}
		result[key] = value
	}
	return result, nil
}

func (parser *TemplateParser) componentFile(sel *goquery.Selection, props map[string]interface{}) (string, error) {
	route := sel.AttrOr("is", "")
	if route == "" {
		return "", fmt.Errorf("Component route is required")
	}

	data := Data{"$props": props}
	route, _ = data.ReplaceUse(slotRe, route)
	route, _ = parser.data.Replace(route)
	root := sel.AttrOr("s:root", "/")
	file := filepath.Join(string(os.PathSeparator), "public", root, route+".jit")
	return file, nil
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

func getComponent(file string, disableCache ...bool) (*JitComponent, error) {

	noCache := false
	if len(disableCache) > 0 && disableCache[0] {
		noCache = true
	}

	if comp, ok := Components[file]; ok && !noCache {
		return comp, nil
	}

	comp, err := readComponent(file)
	if err != nil {
		return nil, err
	}

	// Save the component to the cache
	chComp <- &componentData{file, comp, saveComponent}
	return comp, nil
}

// readComponent read the JIT component
func readComponent(file string) (*JitComponent, error) {
	content, err := application.App.Read(file)
	if err != nil {
		return nil, err
	}
	return &JitComponent{html: string(content)}, nil
}