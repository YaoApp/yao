package core

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/log"
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
	hasStmt := false
	res := stmtRe.ReplaceAllStringFunc(value, func(stmt string) string {
		hasStmt = true
		res, err := data.ExecString(stmt)
		if err != nil {
			log.Warn("Replace %s: %s", stmt, err)
		}
		return res
	})
	return res, hasStmt
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
