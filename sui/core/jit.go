package core

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/hashicorp/go-multierror"
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
	imports     map[string]string
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
var reImports = regexp.MustCompile(`<script[^>]*name="imports"[^>]*>(.*?)</script>`)

func init() {
	go componentWriter()
}

// parseComponent parse the component
func (parser *TemplateParser) parseJitComponent(sel *goquery.Selection) {
	parser.parsed(sel)
	comp, err := parser.getJitComponent(sel)
	if err != nil {
		parser.errors = append(parser.errors, err)
		setError(sel, err)
		return
	}

	comsel, err := parser.newJitComponentSel(sel, comp)
	if err != nil {
		parser.errors = append(parser.errors, err)
		setError(sel, err)
		return
	}
	parser.parseElementComponent(comsel)
	sel.ReplaceWithSelection(comsel)

	if len(comp.scripts) == 0 && len(comp.styles) == 0 {
		return
	}

	if parser.context == nil {
		parser.context = &ParserContext{
			scripts:    []ScriptNode{},
			styles:     []StyleNode{},
			scriptMaps: map[string]bool{},
			styleMaps:  map[string]bool{},
		}
	}

	// Add the scripts
	if comp.scripts != nil {
		for _, script := range comp.scripts {
			hash := script.Hash()
			if parser.context.scriptMaps[hash] {
				continue
			}
			script.Parent = "head"
			parser.context.scriptMaps[hash] = true
			parser.context.scripts = append(parser.context.scripts, script)
		}
	}

	// Add the styles
	if comp.styles != nil {
		for _, style := range comp.styles {
			if parser.context.styleMaps[style.Component] {
				continue
			}
			parser.context.styles = append(parser.context.styles, style)
			parser.context.styleMaps[style.Component] = true
		}
	}
}

func (parser *TemplateParser) newJitComponentSel(sel *goquery.Selection, comp *JitComponent) (*goquery.Selection, error) {

	ns := Namespace(comp.route, parser.sequence+1, comp.buildOption.ScriptMinify)
	cn := ComponentName(comp.route, comp.buildOption.ScriptMinify)
	props := map[string]string{
		"s:ns":    ns,
		"s:cn":    cn,
		"s:ready": cn + "()",
	}

	doc, err := NewDocumentString(comp.html)
	if err != nil {
		return nil, fmt.Errorf("Component %s failed to load, please recompile the component. %s", comp.route, err.Error())
	}

	root := doc.Find("body").First()
	compSel := doc.Find("body").Children().First()
	data := Data{}
	for _, attr := range sel.Nodes[0].Attr {
		if attr.Key == "is" || attr.Key == "s:jit" {
			continue
		}

		// ...variable
		if strings.HasPrefix(attr.Key, "...") {
			key := attr.Key[3:]
			if parser.data != nil {
				if values, ok := parser.data[key].(map[string]any); ok {
					for name, value := range values {
						switch v := value.(type) {
						case string:
							props[name] = v
						case bool, int, float64:
							props[name] = fmt.Sprintf("%v", v)

						case nil:
							props[name] = ""

						default:
							str, err := jsoniter.MarshalToString(value)
							if err != nil {
								continue
							}
							props[name] = str
							props[fmt.Sprintf("json-attr-%s", name)] = "true"
						}
					}
				}
			}
			continue
		}

		val, values := parser.data.Replace(attr.Val)
		if HasJSON(values) {
			props[fmt.Sprintf("json-attr-%s", attr.Key)] = "true"
		}
		props[attr.Key] = val
		data[attr.Key] = val
	}

	data.replaceNodeUse(propTokens, compSel.Nodes[0])
	for key, val := range props {
		if strings.HasPrefix(key, "s:") || key == "parsed" {
			compSel.SetAttr(key, val)
			continue
		}
		// copy the json-attr- to the prop:
		if strings.HasPrefix(key, "json-attr-") {
			compSel.SetAttr(fmt.Sprintf("json-attr-prop:%s", key[10:]), val)
			continue
		}
		compSel.SetAttr(fmt.Sprintf("prop:%s", key), val)
	}

	// Mark as jit component
	compSel.SetAttr("s:route", comp.route)

	// Replace the slots
	slots := sel.Find("slot")
	if slots.Length() > 0 {
		for i := 0; i < slots.Length(); i++ {
			name := slots.Eq(i).AttrOr("name", "")
			if name == "" {
				continue
			}

			slots.Eq(i).Remove()
			slotSel := compSel.Find(name)
			if slotSel.Length() == 0 {
				continue
			}
			slotSel.ReplaceWithSelection(slots.Eq(i).Contents())
		}
	}

	// Replace the children
	children := sel.Contents()
	compSel.Find("children").ReplaceWithSelection(children)
	parser.BindEvent(root, ns, cn) // bind the events
	return compSel, nil
}

func (parser *TemplateParser) getJitComponent(sel *goquery.Selection) (*JitComponent, error) {
	is := sel.AttrOr("is", "")
	if is == "" {
		return nil, fmt.Errorf("Component route is required")

	}

	is, _ = parser.data.Replace(is)
	if parser.option == nil {
		parser.option = &ParserOption{Debug: true, DisableCache: false}
	}

	// Load the component
	if comp, has := Components[is]; has && parser.option.Debug == false && parser.option.DisableCache == false {
		return comp, nil
	}

	file := filepath.Join(string(os.PathSeparator), "public", parser.option.Root, is+".jit")
	if exist, _ := application.App.Exists(file); !exist {
		return nil, fmt.Errorf("Component %s file not found, please recompile the component", is)
	}

	source, err := application.App.Read(file)
	if err != nil {
		return nil, fmt.Errorf("Component %s failed to load, please recompile the component", is)
	}

	// Get the scripts
	var scriptnodes []ScriptNode = []ScriptNode{}
	var stylenodes []StyleNode = []StyleNode{}
	var imports map[string]string = map[string]string{}
	var buildOption *BuildOption = &BuildOption{}

	source, scriptnodes, err = parser.getScriptNodes(source)
	if err != nil {
		return nil, fmt.Errorf("Component %s failed to load, please recompile the component. %s", is, err.Error())
	}

	source, stylenodes, err = parser.getStyleNodes(source)
	if err != nil {
		return nil, fmt.Errorf("Component %s failed to load, please recompile the component. %s", is, err.Error())
	}

	source, imports, err = parser.getImports(source)
	if err != nil {
		return nil, fmt.Errorf("Component %s failed to load, please recompile the component. %s", is, err.Error())
	}

	source, buildOption, err = parser.getBuildOption(source)
	if err != nil {
		return nil, fmt.Errorf("Component %s failed to load, please recompile the component. %s", is, err.Error())
	}

	comp := &JitComponent{
		file:        file,
		route:       is,
		html:        string(source),
		scripts:     scriptnodes,
		styles:      stylenodes,
		imports:     imports,
		buildOption: buildOption,
	}

	// Save the component to the cache
	chComp <- &componentData{is, comp, saveComponent}
	return comp, nil
}

func (parser *TemplateParser) getImports(source []byte) ([]byte, map[string]string, error) {
	imports := map[string]string{}
	source = reImports.ReplaceAllFunc(source, func(raw []byte) []byte {
		raw = reImports.ReplaceAll(raw, []byte("$1"))
		err := jsoniter.Unmarshal(raw, &imports)
		if err != nil {
			return nil
		}
		return []byte{}
	})
	return source, imports, nil
}

func (parser *TemplateParser) getBuildOption(source []byte) ([]byte, *BuildOption, error) {
	var buildOption BuildOption
	rawOption := []byte{}
	source = reOption.ReplaceAllFunc(source, func(raw []byte) []byte {
		rawOption = reOption.ReplaceAll(raw, []byte("$1"))
		return []byte{}
	})

	if rawOption != nil {
		err := jsoniter.Unmarshal(rawOption, &buildOption)
		if err != nil {
			return source, nil, err
		}
	}
	return source, &buildOption, nil
}

func (parser *TemplateParser) getStyleNodes(source []byte) ([]byte, []StyleNode, error) {
	var errs error
	nodes := []StyleNode{}
	source = reStyles.ReplaceAllFunc(source, func(raw []byte) []byte {
		raw = reStyles.ReplaceAll(raw, []byte("$1"))
		var stylenodes []StyleNode
		err := jsoniter.Unmarshal(raw, &stylenodes)
		if err != nil {
			errs = multierror.Append(errs, err)
			return nil
		}
		nodes = append(nodes, stylenodes...)
		return []byte{}
	})
	return source, nodes, errs
}

func (parser *TemplateParser) getScriptNodes(source []byte) ([]byte, []ScriptNode, error) {
	var errs error
	nodes := []ScriptNode{}

	// Get the scripts
	source = reScripts.ReplaceAllFunc(source, func(raw []byte) []byte {
		raw = reScripts.ReplaceAll(raw, []byte("$1"))
		var scripts []ScriptNode
		err := jsoniter.Unmarshal(raw, &scripts)
		if err != nil {
			errs = multierror.Append(errs, err)
			return nil
		}
		nodes = append(nodes, scripts...)
		return []byte{}
	})
	return source, nodes, errs
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
			query := fmt.Sprintf(`script[s\:hash="%s"]`, script.Hash())
			if sel.Find(query).Length() > 0 {
				continue
			}
		}

		src := script.AttrOr("src", "")
		if src != "" {
			query := fmt.Sprintf(`script[src="%s"]`, src)
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

// isJitComponent check if the selection is a component
func (parser *TemplateParser) isJitComponent(sel *goquery.Selection) bool {
	_, exist := sel.Attr("s:jit")
	is := sel.AttrOr("is", "")
	return exist && is != ""
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
