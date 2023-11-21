package core

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/antonmedv/expr"
	"github.com/antonmedv/expr/ast"
	"github.com/antonmedv/expr/vm"
	"github.com/yaoapp/gou/process"
)

var stmtRe = regexp.MustCompile(`\{\{([^}]+)\}\}`)

// Data data for the template
type Data map[string]interface{}

var functions = map[string]*ast.Function{}

var options = []expr.Option{
	expr.Function("P_", _process),
	expr.AllowUndefinedVariables(),
}

// New create a new expression
func (data Data) New(stmt string) (*vm.Program, error) {
	stmt = strings.TrimSpace(strings.TrimRight(strings.TrimLeft(stmt, "{{ "), "}}"))
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
