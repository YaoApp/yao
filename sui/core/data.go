package core

import (
	"fmt"
	"hash/fnv"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/ast"
	"github.com/expr-lang/expr/vm"
	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/process"
	"golang.org/x/net/html"
)

// If set the map value, should keep the space at the end of the statement
// var stmtRe = regexp.MustCompile(`\{\{([\s\S]*?)\}\}`)
// var propRe = regexp.MustCompile(`\[\{([\s\S]*?)\}\]`)  // [{ xxx }] will be deprecated
// var propNewRe = regexp.MustCompile(`\{%([\s\S]*)?%\}`) // {% xxx %}
var propVarNameRe = regexp.MustCompile(`(?:\$props\.)?(?:$begin:math:display$'([^']+)'$end:math:display$|(\w+))`)

// Data data for the template
type Data map[string]interface{}

// Identifier the identifier
type Identifier struct {
	Value string
	Type  string
}

// Visitor the visitor
type Visitor struct {
	Identifiers []Identifier
}

// StringValue the string value
type StringValue struct {
	Value       string
	Stmt        string
	Data        interface{}
	JSON        bool
	Identifiers []Identifier
	Error       error
}

var options = []expr.Option{
	expr.Function("P_", _process),
	expr.Function("True", _true),
	expr.Function("False", _false),
	expr.Function("Empty", _empty),
	expr.AllowUndefinedVariables(),
}

// Visit visit the node
func (v *Visitor) Visit(node *ast.Node) {
	if n, ok := (*node).(*ast.IdentifierNode); ok {
		typ := "unknown"
		t := n.Type()
		if t != nil {
			typ = t.Name()
			if typ == "" {
				typ = "json"
			}
		}
		v.Identifiers = append(v.Identifiers, Identifier{
			Value: n.Value,
			Type:  typ,
		})
	}
}

// Hash get the hash of the data
func (data Data) Hash() string {
	h := fnv.New64a()
	h.Write([]byte(fmt.Sprintf("%v", data)))
	return fmt.Sprintf("%x", h.Sum64())
}

// New create a new expression
func (data Data) New(stmt string) (*vm.Program, error) {

	stmt = dataTokens.ReplaceAllStringFunc(stmt, func(stmt string) string {
		matches := dataTokens.FindAllStringSubmatch(stmt, -1)
		if len(matches) > 0 {
			stmt = strings.ReplaceAll(stmt, matches[0][0], matches[0][1])
		}
		return stmt
	})

	stmt = propTokens.ReplaceAllStringFunc(stmt, func(stmt string) string {
		matches := propTokens.FindAllStringSubmatch(stmt, -1)
		if len(matches) > 0 {
			stmt = strings.ReplaceAll(stmt, matches[0][0], matches[0][1])
		}
		return stmt
	})

	stmt = strings.TrimSpace(stmt)
	// &#39; => ' &#34; => "
	stmt = strings.ReplaceAll(stmt, "&#39;", "'")
	stmt = strings.ReplaceAll(stmt, "&#34;", "\"")
	return expr.Compile(stmt, append([]expr.Option{expr.Env(data)}, options...)...)
}

// Exec exec statement for the template
func (data Data) Exec(stmt string) (interface{}, []Identifier, error) {
	program, err := data.New(stmt)
	if err != nil {
		return nil, nil, err
	}

	node := program.Node()
	v := &Visitor{}
	ast.Walk(&node, v)

	res, err := expr.Run(program, data)
	if err != nil {
		return nil, nil, err
	}

	return res, v.Identifiers, nil
}

// Identifiers get the identifiers for the statement
func (data Data) Identifiers(stmt string) ([]Identifier, error) {
	program, err := data.New(stmt)
	if err != nil {
		return nil, err
	}
	node := program.Node()
	v := &Visitor{}
	ast.Walk(&node, v)
	return v.Identifiers, nil
}

// ExecString exec statement for the template
func (data Data) ExecString(stmt string) StringValue {

	str := StringValue{Stmt: stmt, Value: "", JSON: false, Identifiers: []Identifier{}, Error: nil}
	res, identifiers, err := data.Exec(stmt)
	if err != nil {
		str.Error = err
		return str
	}

	if res == nil {
		return str
	}

	str.Data = res
	switch v := res.(type) {
	case string:
		str.Value = v
		break
	case []byte:
		str.Value = string(v)
		break
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64, bool:
		str.Value = fmt.Sprintf("%v", v)
		break
	default:
		res, err := jsoniter.MarshalToString(res)
		if err != nil {
			str.Error = err
			break
		}
		str.Value = res
		str.JSON = true
	}

	str.Identifiers = identifiers
	return str
}

// Replace replace the statement
func (data Data) Replace(value string) (string, []StringValue) {
	return data.ReplaceUse(dataTokens, value)
}

// ReplaceUse replace the statement use the regexp
func (data Data) ReplaceUse(tokens Tokens, value string) (string, []StringValue) {
	values := []StringValue{}
	res := tokens.ReplaceAllStringFunc(value, func(stmt string) string {
		v := data.ExecString(stmt)
		values = append(values, v)
		return v.Value
	})
	return res, values
}

// ReplaceSelection replace the statement in the selection
func (data Data) ReplaceSelection(sel *goquery.Selection) []StringValue {
	return data.ReplaceSelectionUse(dataTokens, sel)
}

// ReplaceSelectionUse replace the statement in the selection use the regexp
func (data Data) ReplaceSelectionUse(tokens Tokens, sel *goquery.Selection) []StringValue {
	res := []StringValue{}
	for _, node := range sel.Nodes {
		values := data.replaceNodeUse(tokens, node)
		if len(values) > 0 {
			res = append(res, values...)
		}
	}
	return res
}

func (data Data) replaceNodeUse(tokens Tokens, node *html.Node) []StringValue {
	res := []StringValue{}
	switch node.Type {
	case html.TextNode:
		v, values := data.ReplaceUse(tokens, node.Data)
		node.Data = v
		if len(values) > 0 {
			res = append(res, values...)
		}
		break

	case html.ElementNode:
		for i := range node.Attr {

			if (strings.HasPrefix(node.Attr[i].Key, "s:") || node.Attr[i].Key == "is") && !allowUsePropAttrs[node.Attr[i].Key] {
				continue
			}

			v, values := data.ReplaceUse(tokens, node.Attr[i].Val)
			node.Attr[i].Val = v
			if len(values) > 0 {
				res = append(res, values...)
			}
		}

		for c := node.FirstChild; c != nil; c = c.NextSibling {
			values := data.replaceNodeUse(tokens, c)
			if len(values) > 0 {
				res = append(res, values...)
			}
		}
		break
	}
	return res

}

func _false(args ...any) (interface{}, error) {
	v, err := _true(args...)
	if err != nil {
		return false, err
	}
	return !v.(bool), nil
}

func _true(args ...any) (interface{}, error) {

	if len(args) < 1 {
		return false, nil
	}

	if v, ok := args[0].(bool); ok {
		return v, nil
	}

	if v, ok := args[0].(string); ok {
		v = strings.ToLower(v)
		return v != "false" && v != "0", nil
	}

	if v, ok := args[0].(int); ok {
		return v != 0, nil
	}

	return false, nil
}

func _empty(args ...any) (interface{}, error) {

	if len(args) < 1 {
		return true, nil
	}

	if args[0] == nil {
		return true, nil
	}

	if v, ok := args[0].(string); ok {
		return v == "", nil
	}

	if v, ok := args[0].(int); ok {
		return v == 0, nil
	}

	if v, ok := args[0].(bool); ok {
		return !v, nil
	}

	if v, ok := args[0].(map[string]interface{}); ok {
		return len(v) == 0, nil
	}

	if v, ok := args[0].(map[string]string); ok {
		return len(v) == 0, nil
	}

	if v, ok := args[0].([]interface{}); ok {
		return len(v) == 0, nil
	}

	if v, ok := args[0].([]string); ok {
		return len(v) == 0, nil
	}

	if v, ok := args[0].([]int); ok {
		return len(v) == 0, nil
	}

	if v, ok := args[0].(Data); ok {
		return len(v) == 0, nil
	}

	return true, nil
}

func _process(args ...any) (interface{}, error) {

	if len(args) < 1 {
		return nil, fmt.Errorf("process should have at least one parameter")
	}

	name, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("process function only accept string")
	}

	args = append([]any{}, args[1:]...)
	process, err := process.Of(name, args...)
	if err != nil {
		return nil, err
	}

	res, err := process.Exec()
	if err != nil {
		return nil, err
	}

	return res, nil
}

// PropFindAllStringSubmatch find all string submatch
func PropFindAllStringSubmatch(value string) [][]string {
	matched := propTokens.FindAllStringSubmatch(value, -1)
	return matched
}

// PropGetVarNames get the variable names
func PropGetVarNames(value string) []string {
	matched := propVarNameRe.FindAllStringSubmatch(value, -1)
	varNames := []string{}
	for _, m := range matched {
		m1 := strings.TrimSpace(m[1])
		m2 := strings.TrimSpace(m[2])
		if m1 != "" {
			varNames = append(varNames, m1)
		} else {
			varNames = append(varNames, m2)
		}
	}
	return varNames
}
