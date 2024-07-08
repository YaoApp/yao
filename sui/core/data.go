package core

import (
	"fmt"
	"hash/fnv"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/log"
	"golang.org/x/net/html"
)

// If set the map value, should keep the space at the end of the statement
var stmtRe = regexp.MustCompile(`\{\{([\s\S]*?)\}\}`)
var propRe = regexp.MustCompile(`\[\{([\s\S]*?)\}\]`)

// Data data for the template
type Data map[string]interface{}

var options = []expr.Option{
	expr.Function("P_", _process),
	expr.AllowUndefinedVariables(),
}

// Hash get the hash of the data
func (data Data) Hash() string {
	h := fnv.New64a()
	h.Write([]byte(fmt.Sprintf("%v", data)))
	return fmt.Sprintf("%x", h.Sum64())
}

// New create a new expression
func (data Data) New(stmt string) (*vm.Program, error) {

	stmt = stmtRe.ReplaceAllStringFunc(stmt, func(stmt string) string {
		matches := stmtRe.FindStringSubmatch(stmt)
		if len(matches) > 0 {
			stmt = strings.ReplaceAll(stmt, matches[0], matches[1])
		}
		return stmt
	})

	stmt = propRe.ReplaceAllStringFunc(stmt, func(stmt string) string {
		matches := propRe.FindStringSubmatch(stmt)
		if len(matches) > 0 {
			stmt = strings.ReplaceAll(stmt, matches[0], matches[1])
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
func (data Data) Exec(stmt string) (interface{}, error) {
	program, err := data.New(stmt)
	if err != nil {
		return nil, err
	}
	return expr.Run(program, data)
}

// ExecString exec statement for the template
func (data Data) ExecString(stmt string) (string, error) {

	res, err := data.Exec(stmt)
	if err != nil {
		return "", err
	}

	if res == nil {
		return "", nil
	}

	if v, ok := res.(string); ok {
		return v, nil
	}
	return fmt.Sprintf("%v", res), nil
}

// Replace replace the statement
func (data Data) Replace(value string) (string, bool) {
	return data.ReplaceUse(stmtRe, value)
}

// ReplaceUse replace the statement use the regexp
func (data Data) ReplaceUse(re *regexp.Regexp, value string) (string, bool) {
	hasStmt := false
	res := re.ReplaceAllStringFunc(value, func(stmt string) string {
		hasStmt = true
		res, err := data.ExecString(stmt)
		if err != nil {
			log.Warn("Replace %s: %s", stmt, err)
		}
		return res
	})
	return res, hasStmt
}

// ReplaceSelection replace the statement in the selection
func (data Data) ReplaceSelection(sel *goquery.Selection) bool {
	return data.ReplaceSelectionUse(stmtRe, sel)
}

// ReplaceSelectionUse replace the statement in the selection use the regexp
func (data Data) ReplaceSelectionUse(re *regexp.Regexp, sel *goquery.Selection) bool {
	hasStmt := false
	for _, node := range sel.Nodes {
		ok := data.replaceNodeUse(re, node)
		if ok {
			hasStmt = true
		}
	}
	return hasStmt
}

func (data Data) replaceNodeUse(re *regexp.Regexp, node *html.Node) bool {
	hasStmt := false
	switch node.Type {
	case html.TextNode:
		v, ok := data.ReplaceUse(re, node.Data)
		node.Data = v
		if ok {
			hasStmt = true
		}
		break

	case html.ElementNode:
		for i := range node.Attr {

			if (strings.HasPrefix(node.Attr[i].Key, "s:") || node.Attr[i].Key == "is") && !allowUsePropAttrs[node.Attr[i].Key] {
				continue
			}

			v, ok := data.ReplaceUse(re, node.Attr[i].Val)
			node.Attr[i].Val = v
			if ok {
				hasStmt = true
			}
		}

		for c := node.FirstChild; c != nil; c = c.NextSibling {
			ok := data.replaceNodeUse(re, c)
			if ok {
				hasStmt = true
			}
		}
		break
	}

	return hasStmt

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
