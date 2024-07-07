package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/PuerkitoBio/goquery"
	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/kun/log"
	"golang.org/x/net/html"
	"gopkg.in/yaml.v3"
)

// Load the jit components
var components = map[string]string{}

// TemplateParser parser for the template
type TemplateParser struct {
	data     Data
	mapping  map[string]Mapping                  // variable mapping
	sequence int                                 // sequence for the rendering
	errors   []error                             // errors
	replace  map[*goquery.Selection][]*html.Node // replace nodes
	option   *ParserOption                       // parser option
	locale   *Locale                             // locale
	context  *ParserContext                      // parser context
}

// ParserContext parser context for the template
type ParserContext struct {
}

// Mapping mapping for the template
type Mapping struct {
	Key   string      `json:"key,omitempty"`
	Type  string      `json:"type,omitempty"`
	Value interface{} `json:"value,omitempty"`
}

// ParserOption parser option
type ParserOption struct {
	Component    bool   `json:"component,omitempty"`
	Editor       bool   `json:"editor,omitempty"`
	Preview      bool   `json:"preview,omitempty"`
	Debug        bool   `json:"debug,omitempty"`
	DisableCache bool   `json:"disableCache,omitempty"`
	Request      bool   `json:"request,omitempty"`
	Route        string `json:"route,omitempty"`
	Theme        any    `json:"theme,omitempty"`
	Locale       any    `json:"locale,omitempty"`
}

var keepWords = map[string]bool{
	"s:if":        true,
	"s:for":       true,
	"s:for-item":  true,
	"s:for-index": true,
	"s:elif":      true,
	"s:else":      true,
	"s:set":       true,
	"s:bind":      true,
}

var keepAttrs = map[string]bool{
	"s:ns":    true,
	"s:cn":    true,
	"s:ready": true,
	"s:click": true,
}

// Locales the locales
var Locales = map[string]map[string]*Locale{}

type localeData struct {
	name   string
	path   string
	locale *Locale
	cmd    uint8
}

var chLocale = make(chan *localeData, 1)

const (
	saveLocale uint8 = iota
	removeLocale
)

func init() {
	go localeWriter()
}

func localeWriter() {
	for {
		select {
		case data := <-chLocale:
			switch data.cmd {
			case saveLocale:
				if _, ok := Locales[data.name]; !ok {
					Locales[data.name] = map[string]*Locale{}
				}
				Locales[data.name][data.path] = data.locale

			case removeLocale:
				if _, ok := Locales[data.name]; ok {
					delete(Locales[data.name], data.path)
				}
			}
		}
	}
}

// Locale get the locale
func (parser *TemplateParser) Locale() *Locale {
	var locales map[string]*Locale = nil
	name, ok := parser.option.Locale.(string)
	if !ok {
		return nil
	}

	route := parser.option.Route
	disableCache := parser.option.Preview || parser.option.Debug || parser.option.Editor || parser.option.DisableCache

	locales, ok = Locales[name]
	if !ok {
		locales = map[string]*Locale{}
	}

	locale, ok := locales[route]
	if ok && !disableCache {
		return locale
	}

	path := filepath.Join("public", ".locales", name, route+".yml")
	if exists, err := application.App.Exists(path); !exists {
		if err != nil {
			log.Error("[parser] %s Locale %s", route, err.Error())
		}
		return nil
	}

	// Load the locale
	locale = &Locale{}
	raw, err := application.App.Read(path)
	if err != nil {
		log.Error("[parser] %s Locale %s", route, err.Error())
		return nil
	}

	err = yaml.Unmarshal(raw, locale)
	if err != nil {
		log.Error("[parser] %s Locale %s", route, err.Error())
		return nil
	}

	chLocale <- &localeData{name, route, locale, saveLocale}
	return locale
}

// NewTemplateParser create a new template parser
func NewTemplateParser(data Data, option *ParserOption) *TemplateParser {
	if option == nil {
		option = &ParserOption{}
	}

	return &TemplateParser{
		data:     data,
		mapping:  map[string]Mapping{},
		sequence: 0,
		errors:   []error{},
		replace:  map[*goquery.Selection][]*html.Node{},
		option:   option,
	}
}

// Render parses and renders the HTML template
func (parser *TemplateParser) Render(html string) (string, error) {

	// Set the locale
	parser.locale = parser.Locale()

	if !strings.Contains(html, "<html") {
		html = fmt.Sprintf(`<!DOCTYPE html><html lang="en-us">%s</html>`, html)
	}

	doc, err := NewDocumentString(html)
	if err != nil {
		return "", err
	}

	root := doc.Selection.Find("html")
	parser.parseNode(root.Nodes[0])

	// Replace the nodes
	for sel, nodes := range parser.replace {
		sel.ReplaceWithNodes(nodes...)
		delete(parser.replace, sel)
	}

	// Print the data
	jsPrintData := ""
	if parser.option != nil && parser.option.Debug {
		jsPrintData = "console.log(__sui_data);\n"
	}

	// Append the data to the body
	body := doc.Find("body")
	if body.Length() > 0 && !parser.option.Component {
		data, err := jsoniter.MarshalToString(parser.data)
		if err != nil {
			data, _ = jsoniter.MarshalToString(map[string]string{"error": err.Error()})
		}
		body.AppendHtml("<script>\n" +
			"try { " +
			`var __sui_data = ` + data + ";\n" +
			"} catch (e) { console.log('init data error:', e); }\n" +

			`document.addEventListener("DOMContentLoaded", function () {` + "\n" +
			`	try {` + "\n" +
			`		__sui_data_ready( __sui_data );` + "\n" +
			`	} catch(e) {}` + "\n" +
			`});` + "\n" + jsPrintData +

			"</script>\n",
		)
	}

	// For editor
	if parser.option != nil && parser.option.Editor {
		return doc.Find("body").Html()
	}

	// For Request
	if parser.option != nil && (parser.option.Request || parser.option.Preview) {
		// Remove the sui-hide attribute
		doc.Find("[sui-hide]").Remove()
		parser.tidy(doc.Selection)
	}

	// fmt.Println(doc.Html())
	// fmt.Println(parser.errors)
	return doc.Html()
}

// Parse  parses and renders the HTML template
func (parser *TemplateParser) parseNode(node *html.Node) {

	skipChildren := false

	switch node.Type {
	case html.ElementNode:
		sel := goquery.NewDocumentFromNode(node).Selection
		if parser.hasParsed(sel) {
			break
		}
		parser.parseElementNode(sel)

		// Skip children if the node is a loop node
		if _, exist := sel.Attr("s:for"); exist {
			skipChildren = true
		}

	case html.TextNode:
		parser.parseTextNode(node)
	}

	// Recursively process child nodes
	if !skipChildren {
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			parser.parseNode(child)
		}
	}
}

func (parser *TemplateParser) parseElementNode(sel *goquery.Selection) {

	node := sel.Get(0)

	if _, exist := sel.Attr("s:for"); exist {
		parser.forStatementNode(sel)
	}

	if _, exist := sel.Attr("s:if"); exist {
		parser.ifStatementNode(sel)
	}

	if _, exist := sel.Attr("s:set"); exist || node.Data == "s:set" {
		parser.setStatementNode(sel)
	}

	// JIT Compile the element
	if _, exist := sel.Attr("s:jit"); exist {
		parser.parseJitElementNode(sel)
	}

	// Parse the attributes
	parser.parseElementAttrs(sel)

	// Translations
	parser.transElementNode(sel)
}

func (parser *TemplateParser) parseJitElementNode(sel *goquery.Selection) {

	is := sel.AttrOr("is", "")
	if is == "" {
		sel.Remove()
		return
	}

	parser.parsed(sel)
	// Render the JIT component Data
	is, _ = parser.data.Replace(is)
	root := sel.AttrOr("s:root", "/")
	file := filepath.Join(string(os.PathSeparator), "public", root, is+".jit")

	// Should be cached to reduce the file read and unnecessary parsing
	content, err := application.App.Read(file)
	if err != nil {
		log.Error("[parser] %s JIT %s", file, err.Error())
		setError(sel, err)
		return
	}
	sel.ReplaceWith(string(content))

	// With Properties

	// // copy options
	// option := *parser.option
	// option.Component = true
	// p := NewTemplateParser(parser.data, &option)
	// html, err := p.Render(string(content))
	// if err != nil {
	// 	log.Error("[parser] %s JIT %s", file, err.Error())
	// 	return
	// }

	// // Replace the node
	// sel.ReplaceWithHtml(html)
}

func (parser *TemplateParser) transElementNode(sel *goquery.Selection) {
	if parser.locale == nil {
		return
	}

	key, exist := sel.Attr("s:trans-node")
	if !exist {
		return
	}

	text := sel.Text()
	message := strings.TrimSpace(text)
	if message == "" {
		return
	}

	if lcMessage, has := parser.locale.Keys[key]; has && lcMessage != message {
		sel.SetText(strings.Replace(text, message, lcMessage, 1))
		return
	}

	if lcMessage, has := parser.locale.Messages[message]; has {
		sel.SetText(strings.Replace(text, message, lcMessage, 1))
		return
	}
}

// Remove the slot tag and replace it with the children
func (parser *TemplateParser) removeSlotWrapper(sel *goquery.Selection) {
	children := sel.Children()
	if children.Length() == 0 {
		sel.Remove()
		return
	}
	sel.ReplaceWithSelection(children)
}

func (parser *TemplateParser) setStatementNode(sel *goquery.Selection) {

	sel.SetAttr("parsed", "true")

	name := sel.AttrOr("name", "")
	if name == "" {
		return
	}

	valueExp := sel.AttrOr("value", "")
	if stmtRe.MatchString(valueExp) {
		val, err := parser.data.Exec(valueExp)
		if err != nil {
			log.Warn("Set %s: %s", valueExp, err)
			parser.data[name] = valueExp
			return
		}
		parser.data[name] = val
		return
	}

	parser.data[name] = valueExp
}

func (parser *TemplateParser) parseElementAttrs(sel *goquery.Selection) {
	if len(sel.Nodes) < 0 {
		return
	}

	if sel.AttrOr("parsed", "false") == "true" {
		return
	}

	attrs := sel.Nodes[0].Attr
	for _, attr := range attrs {

		// Ignore the s: attributes
		if strings.HasPrefix(attr.Key, "s:") {
			continue
		}

		parser.sequence = parser.sequence + 1
		res, hasStmt := parser.data.Replace(attr.Val)
		if hasStmt {
			bindings := strings.TrimSpace(attr.Val)
			key := fmt.Sprintf("%v", parser.sequence)
			parser.mapping[attr.Key] = Mapping{
				Key:   key,
				Type:  "attr",
				Value: bindings,
			}
			sel.SetAttr(attr.Key, res)
			bindname := fmt.Sprintf("s:bind:%s", attr.Key)
			sel.SetAttr(bindname, bindings)
		}
	}
}

// Check if the element attributes have the s:raw command.
// If true, the sub-node will output the raw data instead of the escaped value.
func checkIsRawElement(node *html.Node) bool {
	if node.Parent != nil && len(node.Parent.Attr) > 0 {
		for _, attr := range node.Parent.Attr {
			if attr.Key == "s:raw" && attr.Val == "true" {
				return true
			}
		}
	}
	return false
}
func (parser *TemplateParser) parseTextNode(node *html.Node) {
	parser.sequence = parser.sequence + 1
	res, hasStmt := parser.data.Replace(node.Data)
	// Bind the variable to the parent node
	if node.Parent != nil && hasStmt {
		bindings := strings.TrimSpace(node.Data)
		key := fmt.Sprintf("%v", parser.sequence)
		if bindings != "" {
			if checkIsRawElement(node) {
				node.Type = html.RawNode
			}
			node.Parent.Attr = append(node.Parent.Attr, []html.Attribute{
				{Key: "s:bind", Val: bindings},
				{Key: "s:key-text", Val: key},
			}...)
		}
	}
	node.Data = res
}

func (parser *TemplateParser) forStatementNode(sel *goquery.Selection) {

	parser.sequence = parser.sequence + 1
	parser.setKey("for", sel, parser.sequence)
	parser.parsed(sel)
	parser.hide(sel) // Hide loop node

	forAttr, _ := sel.Attr("s:for")
	forItems, err := parser.data.Exec(forAttr)
	if err != nil {
		parser.errors = append(parser.errors, err)
		return
	}

	items, err := parser.toArray(forItems)
	if err != nil {
		parser.errors = append(parser.errors, err)
		return
	}

	itemVarName := sel.AttrOr("s:for-item", "item")
	indexVarName := sel.AttrOr("s:for-index", "index")
	itemNodes := []*html.Node{}

	// Keep the node if the editor is enabled
	if parser.option.Editor {
		clone := sel.Clone()
		itemNodes = append(itemNodes, clone.Nodes...)
	}

	for idx, item := range items {

		// Create a new node
		new := sel.Clone()
		parser.removeParsed(new)
		parser.data[itemVarName] = item
		parser.data[indexVarName] = idx

		// parser attributes
		// Copy the if Attr from the parent node
		if ifAttr, exists := new.Attr("s:if"); exists {

			res, err := parser.data.Exec(ifAttr)
			if err != nil {
				parser.errors = append(parser.errors, fmt.Errorf("if statement %v error: %v", parser.sequence, err))
				setError(new, err)
				parser.show(new)
				itemNodes = append(itemNodes, new.Nodes...)
				continue
			}

			if res == true {
				parser.hide(new)
				continue
			}
		}

		parser.parseElementAttrs(new)
		parser.parsed(new)

		// Set the key
		parser.sequence = parser.sequence + 1
		parser.setKey("for-item-index", new, idx)
		parser.setKey("for-item-key", new, parser.sequence)

		// Show the node
		parser.show(new)

		if parser.option.Editor {
			parser.setSuiAttr(new, "generate", "true")
		}

		// Process the new node
		for i := range new.Nodes {
			parser.parseNode(new.Nodes[i])
		}
		itemNodes = append(itemNodes, new.Nodes...)
	}

	// Clean up the variables
	delete(parser.data, itemVarName)
	delete(parser.data, indexVarName)

	// Replace the node
	// sel.ReplaceWithNodes(itemNodes...)
	parser.replace[sel] = itemNodes
}

func (parser *TemplateParser) ifStatementNode(sel *goquery.Selection) {

	parser.sequence = parser.sequence + 1
	parser.setKey("if", sel, parser.sequence)
	parser.parsed(sel)
	parser.hide(sel) // Hide all elif and else nodes

	ifAttr, _ := sel.Attr("s:if")
	elifNodes, elseNode := parser.elseStatementNode(sel)

	for _, elifNode := range elifNodes {
		parser.hide(elifNode)
	}

	if elseNode != nil {
		parser.hide(elseNode)
	}

	// show the node if the condition is true
	res, err := parser.data.Exec(ifAttr)
	if err != nil {
		parser.errors = append(parser.errors, fmt.Errorf("if statement %v error: %v", parser.sequence, err))
		return
	}

	if res == true {
		parser.removeParsed(sel)
		parser.parseElementAttrs(sel)
		parser.parsed(sel)
		parser.show(sel)
		return
	}

	// else if
	for _, elifNode := range elifNodes {
		elifAttr := elifNode.AttrOr("s:elif", "")
		res, err := parser.data.Exec(elifAttr)
		if err != nil {
			parser.errors = append(parser.errors, err)
			return
		}

		if res == true {
			parser.removeParsed(elifNode)
			parser.parseElementAttrs(elifNode)
			parser.parsed(elifNode)
			parser.show(elifNode)
			return
		}
	}

	// else
	if elseNode != nil {
		parser.removeParsed(elseNode)
		parser.parseElementAttrs(elseNode)
		parser.parsed(elseNode)
		parser.show(elseNode)
	}
}

func (parser *TemplateParser) elseStatementNode(sel *goquery.Selection) ([]*goquery.Selection, *goquery.Selection) {
	var elseNode *goquery.Selection = nil
	elifNodes := []*goquery.Selection{}
	key := parser.key("if", sel)
	for next := sel.Next(); next != nil; next = next.Next() {
		if _, exist := next.Attr("s:elif"); exist {
			parser.parsed(next)
			parser.setKey("if", next, key)
			elifNodes = append(elifNodes, next)
			continue
		}

		if _, exist := next.Attr("s:else"); exist {
			parser.parsed(next)
			parser.setKey("if", next, key)
			elseNode = next
			continue
		}
		break
	}

	return elifNodes, elseNode
}

func (parser *TemplateParser) setSuiAttr(sel *goquery.Selection, key, value string) *goquery.Selection {
	key = fmt.Sprintf("data-sui-%s", key)
	return sel.SetAttr(key, value)
}

func (parser *TemplateParser) removeSuiAttr(sel *goquery.Selection, key string) *goquery.Selection {
	key = fmt.Sprintf("data-sui-%s", key)
	return sel.RemoveAttr(key)
}

func (parser *TemplateParser) hide(sel *goquery.Selection) {

	if parser.option.Editor {
		parser.setSuiAttr(sel, "hide", "true")
		return
	}

	sel.SetAttr("sui-hide", "true")

	// style := sel.AttrOr("style", "")
	// if strings.Contains(style, "display: none") {
	// 	return
	// }

	// if style != "" {
	// 	style = fmt.Sprintf("%s; display: none", style)
	// } else {
	// 	style = "display: none"
	// }
	// sel.SetAttr("style", style)
}

func (parser *TemplateParser) show(sel *goquery.Selection) {

	if parser.option.Editor {
		parser.removeSuiAttr(sel, "hide")
		return
	}

	sel.RemoveAttr("sui-hide")

	// style := sel.AttrOr("style", "")
	// if !strings.Contains(style, "display: none") {
	// 	return
	// }

	// style = strings.ReplaceAll(style, "display: none", "")
	// if style == "" {
	// 	sel.RemoveAttr("style")
	// 	return
	// }

	// sel.SetAttr("style", style)
}

func (parser *TemplateParser) tidy(s *goquery.Selection) {

	s.Contents().Each(func(i int, child *goquery.Selection) {

		node := child.Get(0)
		if node.Data == "slot" {
			parser.tidy(child)
			parser.removeSlotWrapper(child)
			return
		}

		if node.Type == html.CommentNode {
			child.Remove()
			return
		}

		// Remove the parsed attribute
		attrs := []html.Attribute{}
		for _, attr := range node.Attr {
			if strings.HasPrefix(attr.Key, "s:") && !keepAttrs[attr.Key] {
				continue
			}

			if attr.Key == "parsed" || attr.Key == "is" || strings.HasPrefix(attr.Key, "...") {
				continue
			}
			attrs = append(attrs, attr)
		}

		node.Attr = attrs
		parser.tidy(child)
	})

}

func (parser *TemplateParser) key(prefix string, sel *goquery.Selection) string {
	name := fmt.Sprintf("s:key-%s", prefix)
	return sel.AttrOr(name, "")
}

func (parser *TemplateParser) setKey(prefix string, sel *goquery.Selection, key interface{}) {
	name := fmt.Sprintf("s:key-%s", prefix)
	value := fmt.Sprintf("%v", key)
	sel.SetAttr(name, value)
}

func (parser *TemplateParser) parsed(sel *goquery.Selection) {
	sel.SetAttr("parsed", "true")
}

func (parser *TemplateParser) removeParsed(sel *goquery.Selection) {
	sel.RemoveAttr("parsed")
}

func (parser *TemplateParser) hasParsed(sel *goquery.Selection) bool {
	if parseed, exist := sel.Attr("parsed"); exist && parseed == "true" {
		return true
	}
	return false
}

func (parser *TemplateParser) toArray(value interface{}) ([]interface{}, error) {
	switch values := value.(type) {

	case []interface{}:
		return values, nil

	case []map[string]interface{}:
		res := []interface{}{}
		for _, v := range values {
			res = append(res, v)
		}
		return res, nil

	case nil:
		return []interface{}{}, nil

	case []map[string]string:
		res := []interface{}{}
		for _, v := range values {
			res = append(res, v)
		}
		return res, nil

	case []string:
		res := []interface{}{}
		for _, v := range values {
			res = append(res, v)
		}
		return res, nil

	case []float64:
		res := []interface{}{}
		for _, v := range values {
			res = append(res, v)
		}
		return res, nil

	case []int:
		res := []interface{}{}
		for _, v := range values {
			res = append(res, v)
		}
		return res, nil

	}

	return nil, fmt.Errorf("Cannot convert %v to array", value)
}
